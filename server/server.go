package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mazzegi/bongo/cms"
	"github.com/pkg/errors"
)

func New(bind string, dir string) (*Server, error) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, errors.Wrapf(err, "listen-tcp on %q", bind)
	}

	dirEnv, err := NewDirEnv(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "new-dir-env %q", dir)
	}

	cms, err := cms.New(dirEnv.CMS())
	if err != nil {
		return nil, errors.Wrap(err, "cms-new")
	}

	tpls, err := NewTemplates(dirEnv.Templates(), cms)
	if err != nil {
		return nil, errors.Wrap(err, "new-templates")
	}

	s := &Server{
		listener:   l,
		httpServer: &http.Server{},
		dirEnv:     dirEnv,
		cms:        cms,
		templates:  tpls,
	}

	return s, nil
}

type Server struct {
	listener   net.Listener
	httpServer *http.Server
	dirEnv     *DirEnv
	cms        *cms.CMS
	templates  *Templates
}

func (s *Server) RunCtx(ctx context.Context) error {
	router := mux.NewRouter()

	// serve static files
	staticSiteServer := http.StripPrefix("/static", http.FileServer(http.Dir(s.dirEnv.Static())))
	router.PathPrefix("/static/").Handler(staticSiteServer).Methods("GET")

	// servce site
	router.PathPrefix("/site/").HandlerFunc(s.handleGETTemplate).Methods("GET")

	s.httpServer.Handler = router
	go s.httpServer.Serve(s.listener)
	go s.templates.WatchCtx(ctx, s.templates.Reload)

	<-ctx.Done()
	s.shutdown(1 * time.Second)
	return nil
}

func (s *Server) shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	s.httpServer.Shutdown(ctx)
}

//
type TemplateData struct {
	Content *cms.CMS
}

func (s *Server) handleGETTemplate(w http.ResponseWriter, r *http.Request) {
	tplName := strings.TrimPrefix(r.URL.Path, "/site/")
	data := TemplateData{
		Content: s.cms,
	}
	err := s.templates.Render(w, tplName, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

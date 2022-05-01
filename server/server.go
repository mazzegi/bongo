package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mazzegi/bongo/cms"
	"github.com/mazzegi/log"
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

func (s *Server) RunCtx(ctx context.Context) {
	router := mux.NewRouter()

	// serve static files
	staticSiteServer := http.StripPrefix("/static", http.FileServer(http.Dir(s.dirEnv.Static())))
	router.PathPrefix("/static/").Handler(staticSiteServer).Methods("GET")

	// serve site
	//router.PathPrefix("/site/").HandlerFunc(s.handleSite).Methods("GET", "POST")
	router.PathPrefix("/site/").HandlerFunc(s.handleSite)

	// serve content (under responisbility of the cms)
	router.PathPrefix("/content/").HandlerFunc(s.handleGETPublicContent).Methods("GET")

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/site/index", http.StatusMovedPermanently)
	})

	s.httpServer.Handler = router
	go s.httpServer.Serve(s.listener)
	go s.templates.WatchCtx(ctx, s.templates.Reload)

	<-ctx.Done()
	s.shutdown(1 * time.Second)
}

func (s *Server) shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	s.httpServer.Shutdown(ctx)
}

//

func (s *Server) handleSite(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGETTemplate(w, r)
	case http.MethodPost:
		s.handlePOSTSite(w, r)
	default:
		http.Error(w, "method-not-allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGETTemplate(w http.ResponseWriter, r *http.Request) {
	tplName := strings.TrimPrefix(r.URL.Path, "/site/")
	err := s.templates.Render(w, tplName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGETPublicContent(w http.ResponseWriter, r *http.Request) {
	cname := strings.TrimPrefix(r.URL.Path, "/content/")
	e, err := s.cms.PublicEntry(cname)
	if err != nil {
		http.Error(w, "not-found", http.StatusNotFound)
		return
	}
	w.Header().Add("Content-Type", string(e.ContentType))
	w.WriteHeader(http.StatusOK)
	w.Write(e.Payload)
}

func (s *Server) handlePOSTSite(w http.ResponseWriter, r *http.Request) {
	cname := strings.TrimPrefix(r.URL.Path, "/site/")
	log.Infof("POST to %q (ref by %q)", cname, r.Referer())
	http.Redirect(w, r, r.Referer(), http.StatusFound)
	//TODO: think about calling a LUA script here filepath=<route>.lua with some context (from request and global)
}

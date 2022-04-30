package server

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dietsche/rfsnotify"
	"github.com/mazzegi/bongo/cms"
	"github.com/mazzegi/log"
	"github.com/pkg/errors"
)

const (
	templateSuffix = ".go.html"
)

func NewTemplates(root string, cms *cms.CMS) (*Templates, error) {
	aroot, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Wrapf(err, "abs-root %q", root)
	}
	tpls := &Templates{
		root: aroot,
		cms:  cms,
	}
	err = tpls.Reload()
	if err != nil {
		return nil, err
	}
	return tpls, nil
}

type Templates struct {
	sync.RWMutex
	root string
	cms  *cms.CMS
	tpls map[string]*template.Template
}

func (ts *Templates) WatchCtx(ctx context.Context, onEvt func() error) error {
	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "new-watcher")
	}
	defer watcher.Close()

	err = watcher.AddRecursive(ts.root)
	if err != nil {
		return errors.Wrapf(err, "watcher-add-recursive %q", ts.root)
	}

	log.Infof("watch for changes in %q", ts.root)
	defer log.Infof("stop watching")
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watcher.Events:
			err := onEvt()
			if err != nil {
				log.Errorf("reload-templates: %v", err)
			}
		}
	}
}

func (ts *Templates) Reload() error {
	ts.Lock()
	defer ts.Unlock()

	log.Infof("parse templates ...")
	ts.tpls = make(map[string]*template.Template)
	err := filepath.Walk(ts.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, templateSuffix) {
			return nil
		}

		tpath := strings.TrimPrefix(path, ts.root)
		tpath = strings.TrimPrefix(tpath, "/")
		name := strings.TrimSuffix(tpath, templateSuffix)

		bs, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read-file %q", path)
		}
		tpl := template.New(name)
		tpl.Funcs(ts.funcs())
		_, err = tpl.Parse(string(bs))
		if err != nil {
			return errors.Wrapf(err, "parse template %q", path)
		}
		ts.tpls[name] = tpl
		log.Debugf("add template %q", name)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walk-and-parse")
	}
	log.Infof("parse templates ... done")

	return nil
}

func (ts *Templates) Render(w io.Writer, name string, data any) error {
	ts.RLock()
	defer ts.RUnlock()

	tpl, ok := ts.tpls[name]
	if !ok {
		return errors.Errorf("no such template %q", name)
	}
	return tpl.Execute(w, data)
}

func (ts *Templates) funcs() template.FuncMap {
	fm := template.FuncMap{
		"RenderTemplate": func(name string, data interface{}) (ret template.HTML, err error) {
			ts.RLock()
			defer ts.RUnlock()
			tpl, ok := ts.tpls[name]
			if !ok {
				err = errors.Errorf("no such template %q", name)
				return
			}
			buf := bytes.NewBuffer([]byte{})
			err = tpl.Execute(buf, data)
			ret = template.HTML(buf.String())
			return
		},
		"ContentEntries": func(path string) ([]cms.EntryInfo, error) {
			return ts.cms.Entries(path)
		},
		"ContentHTML": func(path string) (template.HTML, error) {
			s, err := ts.cms.HTML(path)
			return template.HTML(s), err
		},
		"ContentData": func(path string) (map[string]any, error) {
			v, err := ts.cms.Data(path)
			return v, err
		},
	}
	return fm
}

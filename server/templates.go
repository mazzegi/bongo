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
	layoutTemplate = "__layout.go.html"
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

type TemplateExt struct {
	Name           string
	LayoutTemplate *template.Template
	Template       *template.Template
}

type Templates struct {
	sync.RWMutex
	root string
	cms  *cms.CMS
	tpls map[string]*TemplateExt
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

func (ts *Templates) loadDir(dir string) error {
	fis, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrapf(err, "read-dir %q", dir)
	}
	var tpls []*TemplateExt
	var layout *template.Template
	for _, fi := range fis {
		path := filepath.Join(dir, fi.Name())
		if fi.IsDir() {
			err := ts.loadDir(path)
			if err != nil {
				return errors.Wrapf(err, "load-dir %q", path)
			}
			continue
		}
		if !strings.HasSuffix(path, templateSuffix) {
			continue
		}

		//
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
		if fi.Name() == layoutTemplate {
			layout = tpl
		} else {
			tpls = append(tpls, &TemplateExt{
				Name:     name,
				Template: tpl,
			})
		}
	}

	for _, tple := range tpls {
		tple.LayoutTemplate = layout
		ts.tpls[tple.Name] = tple
		log.Debugf("add template %q", tple.Name)
	}

	return nil
}

func (ts *Templates) Reload() error {
	ts.Lock()
	defer ts.Unlock()
	log.Infof("load templates ...")
	ts.tpls = make(map[string]*TemplateExt)
	err := ts.loadDir(ts.root)
	if err != nil {
		return errors.Wrapf(err, "load-dir %q", ts.root)
	}
	log.Infof("load templates ... done")
	return err
}

//

type TemplateContext struct {
	Slot    *template.Template
	Current string
	Args    []any
}

func (tctx TemplateContext) Arg(idx int) (any, error) {
	if idx < 0 || idx >= len(tctx.Args) {
		return nil, errors.Errorf("index out of bounds")
	}
	return tctx.Args[idx], nil
}

func (ts *Templates) Render(w io.Writer, name string) error {
	ts.RLock()
	defer ts.RUnlock()

	tplex, ok := ts.tpls[name]
	if !ok {
		return errors.Errorf("no such template %q", name)
	}

	if tplex.LayoutTemplate != nil {
		return tplex.LayoutTemplate.Execute(w, TemplateContext{
			Slot:    tplex.Template,
			Current: name,
		})
	} else {
		return tplex.Template.Execute(w, TemplateContext{
			Current: name,
		})
	}
}

func (ts *Templates) funcs() template.FuncMap {
	fm := template.FuncMap{
		"RenderTemplate": func(name string, data interface{}) (ret template.HTML, err error) {
			tpl, ok := ts.tpls[name]
			if !ok {
				err = errors.Errorf("no such template %q", name)
				return
			}
			buf := bytes.NewBuffer([]byte{})
			err = tpl.Template.Execute(buf, data)
			ret = template.HTML(buf.String())
			return
		},
		"Slot": func(tctx TemplateContext) (ret template.HTML, err error) {
			buf := bytes.NewBuffer([]byte{})
			err = tctx.Slot.Execute(buf, tctx)
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
		"Current": func(tctx TemplateContext) string {
			return tctx.Current
		},
		"AttrIfCurrent": func(tctx TemplateContext, name string, value string) template.HTMLAttr {
			if name != tctx.Current {
				return ""
			}
			return template.HTMLAttr(value)
		},
		"ClassIfCurrent": func(tctx TemplateContext, name string, value string) template.HTMLAttr {
			if name != tctx.Current {
				return ""
			}
			return template.HTMLAttr("class=" + value)
		},
		"Component": func(tctx TemplateContext, name string, args ...any) (ret template.HTML, err error) {
			tpl, ok := ts.tpls[name]
			if !ok {
				err = errors.Errorf("no such template %q", name)
				return
			}
			tctx.Args = args
			buf := bytes.NewBuffer([]byte{})
			err = tpl.Template.Execute(buf, tctx)
			ret = template.HTML(buf.String())
			return
		},
		"Log": func(v any) error {
			log.Infof("%#v", v)
			return nil
		},
	}
	return fm
}

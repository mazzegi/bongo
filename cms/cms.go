package cms

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gomarkdown/markdown"
	"github.com/pkg/errors"
)

type EntryInfo struct {
	Name  string
	Path  string
	Type  ContentType
	IsDir bool
}

func New(root string) (*CMS, error) {
	aroot, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Wrapf(err, "abs-root %q", root)
	}
	err = os.MkdirAll(aroot, os.ModePerm)
	if err != nil {
		return nil, errors.Wrapf(err, "mkdirall-root %q", root)
	}

	return &CMS{
		root: aroot,
	}, nil
}

type CMS struct {
	root string
}

func (c *CMS) resolve(path string) string {
	return filepath.Join(c.root, path)
}

func (c *CMS) Entries(path string) ([]EntryInfo, error) {
	apath := filepath.Join(c.root, path)
	fis, err := os.ReadDir(apath)
	if err != nil {
		return nil, errors.Wrapf(err, "read-dir %q", apath)
	}
	var es []EntryInfo
	for _, fi := range fis {
		es = append(es, EntryInfo{
			Name:  fi.Name(),
			Path:  filepath.Join(path, fi.Name()),
			IsDir: fi.IsDir(),
			Type:  ContentTypeFromPath(fi.Name()),
		})
	}
	return es, nil
}

func (c *CMS) Markdown(path string) (string, error) {
	if ContentTypeFromPath(path) != ContentTypeMarkdown {
		return "", errors.Errorf("not a markdown file")
	}
	bs, err := os.ReadFile(c.resolve(path))
	if err != nil {
		return "", errors.Wrapf(err, "read-file %q", path)
	}
	return string(bs), nil
}

func (c *CMS) HTML(path string) (string, error) {
	s, err := c.Markdown(path)
	if err != nil {
		return "", err
	}
	bs := markdown.ToHTML([]byte(s), nil, nil)
	return string(bs), nil
}

func (c *CMS) Data(path string) (map[string]any, error) {
	if ContentTypeFromPath(path) != ContentTypeJSON {
		return nil, errors.Errorf("not a JSON file")
	}
	bs, err := os.ReadFile(c.resolve(path))
	if err != nil {
		return nil, errors.Wrapf(err, "read-file %q", path)
	}
	var data map[string]any
	err = json.Unmarshal(bs, &data)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal-json")
	}
	return data, nil
}

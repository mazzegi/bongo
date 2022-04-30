package server

import (
	"path/filepath"

	"github.com/pkg/errors"
)

func NewDirEnv(dir string) (*DirEnv, error) {
	adir, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "abs-dir %q", dir)
	}

	return &DirEnv{
		dir: adir,
	}, nil
}

type DirEnv struct {
	dir string
}

func (e *DirEnv) CMS() string {
	return filepath.Join(e.dir, "content")
}

func (e *DirEnv) Templates() string {
	return filepath.Join(e.dir, "templates")
}

func (e *DirEnv) Static() string {
	return filepath.Join(e.dir, "static")
}

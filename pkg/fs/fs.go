package fs

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5"
	billyUtils "github.com/go-git/go-billy/v5/util"
)

//go:generate mockery -name FS -filename fs.go
//go:generate mockery -name File -filename file.go
type FS interface {
	billy.Filesystem

	CheckExistsOrWrite(filename string, data []byte) (bool, error)

	// Exists checks if the provided path exists in the provided filesystem.
	Exists(path string) (bool, error)

	// ExistsOrDie checks if the provided path exists in the provided filesystem, or panics on any error other then ErrNotExist
	ExistsOrDie(path string) bool

	// ReadFile returns the entire file as []byte
	ReadFile(filename string) ([]byte, error)

	// ReadYamls reads the file data as yaml into o
	ReadYamls(filename string, o ...interface{}) error

	// WriteYamls write the data as yaml into the file
	WriteYamls(filename string, o ...interface{}) error
}

type fsimpl struct {
	billy.Filesystem
}

type File interface {
	billy.File
}

func Create(bfs billy.Filesystem) FS {
	return &fsimpl{bfs}
}

func (fs *fsimpl) CheckExistsOrWrite(filename string, data []byte) (bool, error) {
	exists, err := fs.Exists(filename)
	if err != nil {
		return false, fmt.Errorf("failed to check if file exists on repo at '%s': %w", filename, err)
	}

	if exists {
		return true, nil
	}

	if err = billyUtils.WriteFile(fs, filename, data, 0666); err != nil {
		return false, fmt.Errorf("failed to create file at '%s': %w", filename, err)
	}

	return false, nil
}

func (fs *fsimpl) Exists(path string) (bool, error) {
	if _, err := fs.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func (fs *fsimpl) ExistsOrDie(path string) bool {
	exists, err := fs.Exists(path)
	util.Die(err)
	return exists
}

func (fs *fsimpl) ReadFile(filename string) ([]byte, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}

func (fs *fsimpl) ReadYamls(filename string, o ...interface{}) error {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return err
	}

	yamls := util.SplitManifests(data)
	if len(yamls) < len(o) {
		return fmt.Errorf("expected at least %d manifests when reading '%s'", len(o), filename)
	}

	for i, e := range o {
		if e == nil {
			continue
		}

		err = yaml.Unmarshal(yamls[i], e)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *fsimpl) WriteYamls(filename string, o ...interface{}) error {
	var err error
	yamls := make([][]byte, len(o))
	for i, e := range o {
		yamls[i], err = yaml.Marshal(e)
		if err != nil {
			return err
		}
	}

	data := util.JoinManifests(yamls...)
	return billyUtils.WriteFile(fs, filename, data, 0666)
}

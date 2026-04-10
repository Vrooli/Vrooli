package userconfig

import (
	"errors"
	"io/fs"
	"os"
)

// configIO is the filesystem/environment testing seam for config manager behavior.
type configIO interface {
	Stat(name string) (os.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(name string, data []byte, perm os.FileMode) error
	Rename(oldpath, newpath string) error
	Remove(name string) error
	UserHomeDir() (string, error)
}

type realConfigIO struct{}

func (realConfigIO) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (realConfigIO) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (realConfigIO) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (realConfigIO) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (realConfigIO) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (realConfigIO) Remove(name string) error {
	return os.Remove(name)
}

func (realConfigIO) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func isNotExistErr(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

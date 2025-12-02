package infra

import (
	"io"
	"io/fs"
	"os"
)

// FileSystem abstracts file system operations for testing.
type FileSystem interface {
	// ReadFile reads the entire contents of a file.
	ReadFile(path string) ([]byte, error)
	// WriteFile writes data to a file with the given permissions.
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// MkdirAll creates a directory and all parent directories.
	MkdirAll(path string, perm fs.FileMode) error
	// Stat returns file info for the given path.
	Stat(path string) (fs.FileInfo, error)
	// OpenFile opens a file with the given flags and permissions.
	OpenFile(path string, flag int, perm fs.FileMode) (File, error)
	// Remove removes a file or empty directory.
	Remove(path string) error
}

// File abstracts os.File for testing.
type File interface {
	io.Writer
	io.Closer
	// Sync commits the current contents of the file to stable storage.
	Sync() error
}

// RealFileSystem implements FileSystem using the os package.
type RealFileSystem struct{}

func (RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (RealFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (RealFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (RealFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (RealFileSystem) OpenFile(path string, flag int, perm fs.FileMode) (File, error) {
	return os.OpenFile(path, flag, perm)
}

func (RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// Ensure RealFileSystem implements FileSystem.
var _ FileSystem = RealFileSystem{}

// Package filesystem owns the production adapter for filesystem operations.
// Domain packages keep their narrower injectable interfaces and share this
// adapter for their default constructors.
package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileReader is the smallest file-read capability.
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// DirectoryReader supports the common file and directory listing operations.
type DirectoryReader interface {
	FileReader
	ReadDir(path string) ([]fs.DirEntry, error)
	Exists(path string) bool
}

// Reader adds metadata lookup to DirectoryReader.
type Reader interface {
	DirectoryReader
	Stat(path string) (fs.FileInfo, error)
}

// Writer supports writing files and creating parent directories.
type Writer interface {
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// RemovableWriter extends Writer for domains that also delete files.
type RemovableWriter interface {
	Writer
	Remove(path string) error
}

// FileSystem combines full reader and removable-writer capabilities.
type FileSystem interface {
	Reader
	RemovableWriter
}

// Walker provides the filepath and io/fs directory traversal callbacks.
type Walker interface {
	Walk(root string, fn filepath.WalkFunc) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// OS implements filesystem operations with the Go standard library.
type OS struct{}

var (
	_ FileSystem = OS{}
	_ Walker     = OS{}
)

func (OS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (OS) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }

func (OS) Stat(path string) (fs.FileInfo, error) { return os.Stat(path) }

func (OS) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (OS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (OS) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }

func (OS) Remove(path string) error { return os.Remove(path) }

func (OS) Walk(root string, fn filepath.WalkFunc) error { return filepath.Walk(root, fn) }

func (OS) WalkDir(root string, fn fs.WalkDirFunc) error { return filepath.WalkDir(root, fn) }

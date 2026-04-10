// Package persistence provides data access for session profile management.
package persistence

import (
	"io/fs"
	"os"
)

// FileSystem abstracts file operations for testability.
// It provides a subset of os package functions needed by FileRepository.
type FileSystem interface {
	// ReadFile reads the contents of a file.
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to a file, creating it if necessary.
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// Remove removes a file.
	Remove(name string) error

	// Rename renames (moves) a file.
	Rename(oldpath, newpath string) error

	// ReadDir reads a directory and returns its entries.
	ReadDir(name string) ([]fs.DirEntry, error)

	// MkdirAll creates a directory along with any necessary parents.
	MkdirAll(path string, perm fs.FileMode) error

	// Stat returns file info for a path.
	Stat(name string) (fs.FileInfo, error)
}

// OSFileSystem implements FileSystem using the real os package.
type OSFileSystem struct{}

// ReadFile reads the contents of a file.
func (OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile writes data to a file, creating it if necessary.
func (OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Remove removes a file.
func (OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

// Rename renames (moves) a file.
func (OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// ReadDir reads a directory and returns its entries.
func (OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// MkdirAll creates a directory along with any necessary parents.
func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Stat returns file info for a path.
func (OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// NewOSFileSystem returns a FileSystem that uses the real os package.
func NewOSFileSystem() FileSystem {
	return OSFileSystem{}
}

// Ensure OSFileSystem implements FileSystem at compile time.
var _ FileSystem = OSFileSystem{}

// Package persistence provides data access for session profile management.
package persistence

import (
	"context"
	"io/fs"
	"os"
	"time"

	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
)

// FileSystem abstracts file operations for testability.
// It provides a subset of os package functions needed by FileRepository.
type FileSystem interface {
	// Lock admits one writer, with a bounded wait, across repository instances.
	Lock(path string) (release func(), err error)
	// ReadFile reads the contents of a file.
	ReadFile(name string) ([]byte, error)

	// WriteFileAtomic syncs a complete temporary file before replacing the target.
	WriteFileAtomic(name string, data []byte, perm fs.FileMode) error

	// Remove removes a file.
	Remove(name string) error

	// ReadDir reads a directory and returns its entries.
	ReadDir(name string) ([]fs.DirEntry, error)

	// MkdirAll creates a directory along with any necessary parents.
	MkdirAll(path string, perm fs.FileMode) error

	// Stat returns file info for a path.
	Stat(name string) (fs.FileInfo, error)
}

// OSFileSystem implements FileSystem using the real os package.
type OSFileSystem struct{}

func (OSFileSystem) Lock(path string) (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return platform.AcquireFileLockContext(ctx, path)
}

// ReadFile reads the contents of a file.
func (OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFileAtomic uses the shared storage owner's publication primitive.
func (OSFileSystem) WriteFileAtomic(name string, data []byte, perm fs.FileMode) error {
	return storage.WriteFileAtomic(name, data, perm)
}

// Remove removes a file.
func (OSFileSystem) Remove(name string) error {
	return os.Remove(name)
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

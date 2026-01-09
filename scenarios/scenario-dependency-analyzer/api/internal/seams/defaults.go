package seams

import (
	"io/fs"
	"os"
	"time"

	"github.com/google/uuid"
)

// RealClock implements Clock using the standard library.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// UUIDGenerator implements IDGenerator using google/uuid.
type UUIDGenerator struct{}

// NewID returns a new UUID string.
func (UUIDGenerator) NewID() string {
	return uuid.New().String()
}

// OSFileSystem implements FileSystem using the os package.
type OSFileSystem struct{}

// ReadDir reads a directory.
func (OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// Stat returns file info.
func (OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile reads a file.
func (OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile writes a file.
func (OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// MkdirAll creates a directory tree.
func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// NewDefaults returns Dependencies with real implementations.
func NewDefaults() *Dependencies {
	return &Dependencies{
		Clock: RealClock{},
		IDs:   UUIDGenerator{},
		FS:    OSFileSystem{},
		// Deployment is nil by default; consumers should set it if needed
	}
}

// Default is a package-level default instance for convenience.
// Tests should not use this; instead inject a custom Dependencies.
var Default = NewDefaults()

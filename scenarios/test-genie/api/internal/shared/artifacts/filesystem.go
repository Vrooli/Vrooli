package artifacts

import "os"

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// OSFileSystem is the default filesystem implementation using os package.
type OSFileSystem struct{}

// WriteFile writes data to a file.
func (OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// MkdirAll creates a directory and all parents.
func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

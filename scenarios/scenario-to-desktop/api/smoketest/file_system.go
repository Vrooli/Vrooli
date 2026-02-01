package smoketest

import "os"

// DefaultFileSystem implements FileSystem using the real filesystem.
type DefaultFileSystem struct{}

// NewFileSystem creates a new filesystem wrapper.
func NewFileSystem() *DefaultFileSystem {
	return &DefaultFileSystem{}
}

// Stat returns file info for the given path.
func (fs *DefaultFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// ReadDir reads the contents of a directory.
func (fs *DefaultFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

// Open opens a file for reading.
func (fs *DefaultFileSystem) Open(path string) (*os.File, error) {
	return os.Open(path)
}

// Chmod changes the mode of the named file.
func (fs *DefaultFileSystem) Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

// ReadFile reads a file and returns its contents.
func (fs *DefaultFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

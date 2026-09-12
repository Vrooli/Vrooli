package main

import "os"

// FileIO abstracts filesystem operations to enable testing without real files.
// Production code uses OSFileIO. Tests use FakeFileIO.
type FileIO interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	// Lstat reports on the link itself rather than its target. Callers that
	// classify a path by its content (rather than the content it points at)
	// must use this: git stores a symlink's target path, not the target's
	// bytes, so following one misattributes the target's size and type.
	Lstat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
}

// OSFileIO delegates to the os package.
type OSFileIO struct{}

func (OSFileIO) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (OSFileIO) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (OSFileIO) Stat(path string) (os.FileInfo, error)        { return os.Stat(path) }
func (OSFileIO) Lstat(path string) (os.FileInfo, error)       { return os.Lstat(path) }
func (OSFileIO) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

package main

import "os"

// FileIO abstracts filesystem operations to enable testing without real files.
// Production code uses OSFileIO. Tests use FakeFileIO.
type FileIO interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
}

// OSFileIO delegates to the os package.
type OSFileIO struct{}

func (OSFileIO) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (OSFileIO) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (OSFileIO) Stat(path string) (os.FileInfo, error)        { return os.Stat(path) }
func (OSFileIO) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

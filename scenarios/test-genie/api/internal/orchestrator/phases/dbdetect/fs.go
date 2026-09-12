package dbdetect

import (
	"io/fs"
	"os"
	"path/filepath"
)

// seam: Filesystem — boundary between dbdetect's read-only file inspection
// and the real OS, so collectors are testable without a temp directory.
type Filesystem interface {
	ReadFile(path string) ([]byte, error)
	Walk(root string, fn fs.WalkDirFunc) error
}

// OSFilesystem is the production Filesystem.
type OSFilesystem struct{}

func (OSFilesystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (OSFilesystem) Walk(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

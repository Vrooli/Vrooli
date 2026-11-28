package dbdetect

import (
	"io/fs"

	sharedfs "test-genie/internal/shared/filesystem"
)

// seam: Filesystem — boundary between dbdetect's read-only file inspection
// and the real OS, so collectors are testable without a temp directory.
type Filesystem interface {
	ReadFile(path string) ([]byte, error)
	Walk(root string, fn fs.WalkDirFunc) error
}

// OSFilesystem preserves the domain name while delegating to the shared OS adapter.
type OSFilesystem struct{}

func (OSFilesystem) ReadFile(path string) ([]byte, error) { return sharedfs.OS{}.ReadFile(path) }

func (OSFilesystem) Walk(root string, fn fs.WalkDirFunc) error {
	return sharedfs.OS{}.WalkDir(root, fn)
}

package pagediscovery

import "os"

// FakeFileReader is the test double for FileReader. Files maps an absolute path
// to its bytes; a missing path returns os.ErrNotExist so discovery exercises its
// fallback path.
//
// seam: FakeFileReader is the test wiring for the FileReader seam
// (pagediscovery.go).
type FakeFileReader struct {
	Files map[string][]byte
}

// ReadFile returns the canned bytes for path or os.ErrNotExist.
func (f FakeFileReader) ReadFile(path string) ([]byte, error) {
	if data, ok := f.Files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

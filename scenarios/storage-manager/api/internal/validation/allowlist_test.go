package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemAllowlistRequiresBoundedReviewMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filesystemAllowlistPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const contents = `{"version":1,"entries":[
{"kind":"scenario","id":"reviewed","code":"FILESYSTEM_DIRECT_WRITER","owner":"team-storage","reason":"legacy database lock","scope":"api/database.go:20","review_trigger":"remove after migration v2"},
{"kind":"scenario","id":"unbounded","code":"FILESYSTEM_DIRECT_WRITER"}
]}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadFilesystemAllowlist(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("allowlist entries = %d, want one fully bounded entry", len(got))
	}
}

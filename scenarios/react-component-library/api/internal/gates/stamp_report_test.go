package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileInRootRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "report.json")); err != nil {
		t.Fatal(err)
	}

	if _, err := readFileInRoot(root, "report.json"); err == nil {
		t.Fatal("readFileInRoot followed a symlink outside its root")
	}
}

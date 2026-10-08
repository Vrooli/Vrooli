package testutil

import (
	"os"
	"testing"
)

func TestWriteFileCreatesPrivateFixture(t *testing.T) {
	root := t.TempDir()
	if err := WriteFile(root, "nested/fixture.json", `{"ok":true}`); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(root + "/nested/fixture.json")
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("fixture = %q/%v", data, err)
	}
	invalidRoot := root + "/not-a-directory"
	if err := os.WriteFile(invalidRoot, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(invalidRoot, "child.json", "{}"); err == nil {
		t.Fatal("writing below a file should fail")
	}
}

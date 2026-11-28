package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyRejectsCorruptModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ocr-model.json")
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := (Resolver{DataDir: dir}).Verify()
	if err == nil {
		t.Fatal("Verify() accepted corrupt model")
	}
}

package authcrypto

import (
	"os"
	"testing"
)

func TestLoadExistingAbsentDoesNotGenerateKeys(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadExisting(dir); err == nil {
		t.Fatal("absent enrolled key accepted")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("read-only loading created key files")
	}
}

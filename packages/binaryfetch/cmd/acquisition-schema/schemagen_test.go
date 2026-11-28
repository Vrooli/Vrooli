package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedSchemaMatchesCommitted(t *testing.T) {
	if err := ValidateAcquisitionSchema(filepath.Join("..", "..", "..", "..")); err != nil {
		t.Fatal(err)
	}
	data, err := RenderAcquisitionSchema()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"tar.xz"`) {
		t.Fatal("generated schema omits tar.xz")
	}
}

func TestGeneratedSchemaDetectsTampering(t *testing.T) {
	root := t.TempDir()
	if err := SyncAcquisitionSchema(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(AcquisitionSchemaPath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"title": "Declared artifact acquisition"`, `"title": "tampered"`, 1))
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAcquisitionSchema(root); err == nil || !strings.Contains(err.Error(), "acquisition schema is stale") {
		t.Fatalf("tampered schema accepted: %v", err)
	}
}

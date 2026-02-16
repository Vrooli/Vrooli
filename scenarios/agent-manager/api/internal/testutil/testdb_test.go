package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSchemaDir_ContainsSchemaFile(t *testing.T) {
	schemaPath := filepath.Join(getSchemaDir(), "schema.sql")
	info, err := os.Stat(schemaPath)
	if err != nil {
		t.Fatalf("expected schema file at %s: %v", schemaPath, err)
	}
	if info.IsDir() {
		t.Fatalf("expected schema path to be file, got directory: %s", schemaPath)
	}
}

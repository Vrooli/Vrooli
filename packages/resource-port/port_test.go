package resourceport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesNamedHostPort(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "redis")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "resource.json"), []byte(`{"ports":[{"name":"api","host":6380,"container":6379}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := Resolve(root, "redis", "api"); err != nil || got != "6380" {
		t.Fatalf("Resolve = %q, %v", got, err)
	}
}

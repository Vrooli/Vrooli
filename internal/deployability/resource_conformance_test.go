package deployability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckResourceDeclarationsRejectsUnsupportedDriver(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "resource.json"), []byte(`{"driver":"retired-driver"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckResourceDeclarations(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].Message, "retired-driver") {
		t.Fatalf("expected unsupported-driver finding, got %#v", findings)
	}
}

func TestCheckResourceDeclarationsAcceptsSupportedDriver(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"driver":"managed-service"}`
	if err := os.WriteFile(filepath.Join(path, "resource.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckResourceDeclarations(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected supported driver to pass, got %#v", findings)
	}
}

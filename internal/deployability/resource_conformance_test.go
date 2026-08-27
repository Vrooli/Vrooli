package deployability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckResourceDeclarationsRejectsComposeServiceWithoutDatedOverride(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "resource.json"), []byte(`{"driver":"compose-service"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckResourceDeclarations(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].Message, "compose-service is closed") {
		t.Fatalf("expected closed-driver finding, got %#v", findings)
	}
}

func TestCheckResourceDeclarationsAcceptsCompleteComposeServiceOverride(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"driver":"compose-service","compose_service_override":{"reason":"legacy integration pending migration","review_by":"2027-08-26"}}`
	if err := os.WriteFile(filepath.Join(path, "resource.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckResourceDeclarations(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected dated override to pass, got %#v", findings)
	}
}

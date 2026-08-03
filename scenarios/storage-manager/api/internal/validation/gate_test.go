package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNonScenarioGateFailsOnDeclarationError(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "fixture")
	dataDir := filepath.Join(resourceDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "payload"), []byte("too-large"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(`{"name":"fixture","storage":{"entries":{"data":{"rung":"owned","path":"data","kind":"dir","regenerable":true,"budget":{"max_bytes":"1B"}}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result := New(Deps{RepoRoot: root}).validateNonScenarioGate(context.Background(), "linux")
	if result.OwnerCount != 1 || result.ErrorCount == 0 {
		t.Fatalf("gate result = %+v, want one owner and an error", result)
	}
}

func TestNonScenarioGatePassesCurrentShape(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(`{"name":"fixture","storage":{"entries":{"data":{"rung":"owned","path":"data","kind":"dir","regenerable":true,"budget":{"max_bytes":"1MiB"}}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result := New(Deps{RepoRoot: root}).validateNonScenarioGate(context.Background(), "linux")
	if result.OwnerCount != 1 || result.ErrorCount != 0 {
		t.Fatalf("gate result = %+v, want one owner and no errors", result)
	}
}

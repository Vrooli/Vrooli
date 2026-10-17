package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnrichDeclarationIdentityUsesResourceBeforeScenarioFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "resources", "postgres"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "demo", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	resource := `{"credentials":{"descriptors":[{"logical_id":"vrooli/postgres","field":"password","env":"POSTGRES_PASSWORD"}]}}`
	scenario := `{"credentials":{"descriptors":[{"logical_id":"vrooli/demo","field":"token","env":"POSTGRES_PASSWORD"}]}}`
	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "resource.json"), []byte(resource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := DeploymentSecretEntry{ResourceName: "postgres", SecretKey: "POSTGRES_PASSWORD"}
	enrichDeclarationIdentity(root, &entry, "demo")
	if entry.LogicalID != "vrooli/postgres" || entry.Field != "password" {
		t.Fatalf("identity = %s:%s, want resource declaration", entry.LogicalID, entry.Field)
	}
}

package setup

import (
	"encoding/json"
	repocontracttest "github.com/vrooli/repo-contract-go/repocontracttest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCohortRejectsInputChange(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "internal", "main.go")
	if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := withStableBuildInputs(root, func() error { return os.WriteFile(source, []byte("package main\nvar changed = true\n"), 0600) })
	if err == nil || !strings.Contains(err.Error(), "inputs changed") {
		t.Fatalf("expected input drift refusal, got %v", err)
	}
	if err := withStableBuildInputs(root, func() error { return nil }); err != nil {
		t.Fatalf("stable build: %v", err)
	}
}

func TestBuildCohortAllowsUnrelatedTestEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "packages", "unrelated", "example_test.go")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package unrelated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := withStableBuildInputs(root, func() error {
		return os.WriteFile(path, []byte("package unrelated\nvar changed = true\n"), 0600)
	}); err != nil {
		t.Fatalf("test-only edit interrupted build: %v", err)
	}
}

func TestBuildCohortRejectsRuntimeHomeChange(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	path := filepath.Join(root, ".vrooli", "repo-contract.json")
	err := withStableBuildInputs(root, func() error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc map[string]any
		if err = json.Unmarshal(data, &doc); err != nil {
			return err
		}
		doc["runtime_home"].(map[string]any)["entries"].(map[string]any)["runtime_db"].(map[string]any)["path"] = "state/registry-v2.db"
		data, err = json.Marshal(doc)
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0600)
	})
	if err == nil || !strings.Contains(err.Error(), "inputs changed") {
		t.Fatalf("runtime-home drift: %v", err)
	}
}

package experience

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReaderJoinsContractAndEvidence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "DrawerShell", "versions", "1.0.0")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{"component":{"id":"drawer-shell","title":"DrawerShell","purpose":"Named dialog surface."},"states":[{"id":"full-open","example":"full-open","description":"Open."}],"claims":[{"id":"dialog-present","type":"element-present","statement":"A dialog is present.","tier":"machine","states":["full-open"]}]}`
	if err := os.WriteFile(filepath.Join(path, "experience-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &reader{repoRoot: root, listEvidence: func(_ context.Context, componentID string) ([]Evidence, error) {
		if componentID != "drawer-shell" {
			t.Fatalf("component id = %q", componentID)
		}
		return []Evidence{{ClaimID: "dialog-present", Verdict: "passed", ExampleName: "full-open"}}, nil
	}}
	got, err := r.Get(context.Background(), Component{ID: "cmp-1", LibraryID: "react-component-library:DrawerShell", Slug: "DrawerShell", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractID != "drawer-shell" || got.Title != "DrawerShell" || len(got.States) != 1 || len(got.Claims) != 1 || len(got.Evidence) != 1 || got.EvidenceStatus != "available" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestReaderUsesCanonicalScenarioLevelContract(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "scenarios", "react-component-library", "experience", "components")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "drawer-shell.json"), []byte(`{"component":{"id":"drawer-shell"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &reader{repoRoot: root, listEvidence: func(context.Context, string) ([]Evidence, error) { return nil, nil }}
	got, err := r.Get(context.Background(), Component{ID: "cmp-1", Slug: "DrawerShell", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ContractID != "drawer-shell" || got.Title != "" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestReaderReportsNoContractWithoutCallingEvidence(t *testing.T) {
	called := false
	r := &reader{repoRoot: t.TempDir(), listEvidence: func(context.Context, string) ([]Evidence, error) { called = true; return nil, nil }}
	got, err := r.Get(context.Background(), Component{ID: "cmp-1", LibraryID: "react-component-library:Unknown", Slug: "Unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if got.EvidenceStatus != "not-configured" || called {
		t.Fatalf("snapshot = %+v called=%t", got, called)
	}
}

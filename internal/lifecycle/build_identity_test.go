package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestScenarioBuildIdentityTracksAuthoredInputsAndIgnoresRuntimeOutputs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "PRD.md"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dist", "bundle.js"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := scenario.Scenario{
		Path: root,
		Slug: "alpha",
		Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{
			"api": {
				Build: scenario.ComponentBuild{Kind: "go_module", Dir: ".", Output: "api/mock-api"},
			},
		}},
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "mock-api"), []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	first, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() error = %v", err)
	}
	if first == "" {
		t.Fatal("scenarioBuildIdentity() returned empty identity")
	}
	if err := os.WriteFile(filepath.Join(root, "dist", "bundle.js"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after output change: %v", err)
	}
	if second != first {
		t.Fatalf("runtime output changed identity: first=%q second=%q", first, second)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "mock-api"), []byte("new binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	third, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after component output change: %v", err)
	}
	if third != second {
		t.Fatalf("declared component output changed identity: first=%q third=%q", second, third)
	}
	if err := os.WriteFile(filepath.Join(root, "PRD.md"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}
	fourth, err := scenarioBuildIdentity(item)
	if err != nil {
		t.Fatalf("scenarioBuildIdentity() after source change: %v", err)
	}
	if fourth == third {
		t.Fatal("authored source change did not change build identity")
	}
}

func TestRegistryRuntimeHealthRejectsStaleBuildIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := scenario.Scenario{Slug: "alpha", Path: root}
	runner := &Runner{}

	if runner.isRegistryRuntimeHealthy(item, registryRuntimeView{
		Authoritative: true,
		Instance:      scenarioruntime.Instance{BuildIdentity: "sha256:older"},
	}) {
		t.Fatal("stale build identity must prevent healthy runtime reuse")
	}
}

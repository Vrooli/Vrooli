package gomodreconcile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeModule writes a go.mod (and optional .go files) into dir.
func writeModule(t *testing.T, dir, gomod string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadTopologyReadsModulePaths(t *testing.T) {
	root := t.TempDir()
	writeModule(t, filepath.Join(root, "packages", "leaf"), "module example.com/leaf\n\ngo 1.25.0\n", nil)
	writeModule(t, filepath.Join(root, "scenarios", "demo", "cli"), "module demo/cli\n\ngo 1.25.0\n", nil)
	// node_modules must be skipped.
	writeModule(t, filepath.Join(root, "node_modules", "junk"), "module junk\n\ngo 1.25.0\n", nil)

	topo, err := LoadTopology(root)
	if err != nil {
		t.Fatalf("LoadTopology: %v", err)
	}
	if got := topo["example.com/leaf"]; got != filepath.Join(root, "packages", "leaf") {
		t.Fatalf("leaf dir = %q", got)
	}
	if got := topo["demo/cli"]; got != filepath.Join(root, "scenarios", "demo", "cli") {
		t.Fatalf("demo/cli dir = %q", got)
	}
	if _, ok := topo["junk"]; ok {
		t.Fatalf("node_modules module should be skipped")
	}
}

func TestPlanFlagsMissingInRepoReplaceWithRelPath(t *testing.T) {
	root := t.TempDir()
	leafDir := filepath.Join(root, "packages", "leaf")
	consumerDir := filepath.Join(root, "scenarios", "demo", "cli")
	writeModule(t, leafDir, "module example.com/leaf\n\ngo 1.25.0\n", nil)
	writeModule(t, consumerDir, "module demo/cli\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n", nil)
	topo := Topology{"example.com/leaf": leafDir}

	missing, err := Plan(context.Background(), filepath.Join(consumerDir, "go.mod"), topo)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("missing = %#v", missing)
	}
	if missing[0].Module != "example.com/leaf" || missing[0].RelPath != "../../../packages/leaf" {
		t.Fatalf("missing[0] = %#v", missing[0])
	}
}

func TestPlanIgnoresAlreadyReplacedAndThirdParty(t *testing.T) {
	root := t.TempDir()
	leafDir := filepath.Join(root, "packages", "leaf")
	consumerDir := filepath.Join(root, "scenarios", "demo", "cli")
	writeModule(t, leafDir, "module example.com/leaf\n\ngo 1.25.0\n", nil)
	// requires leaf WITH a replace, plus a third-party module that is not in-repo.
	writeModule(t, consumerDir, "module demo/cli\n\ngo 1.25.0\n\nrequire (\n\texample.com/leaf v0.0.0\n\tgithub.com/third/party v1.2.3\n)\n\nreplace example.com/leaf => ../../../packages/leaf\n", nil)
	topo := Topology{"example.com/leaf": leafDir}

	missing, err := Plan(context.Background(), filepath.Join(consumerDir, "go.mod"), topo)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing replaces (leaf replaced, third-party out of scope), got %#v", missing)
	}
}

func TestPlanFlagsTransitiveInRepoRequireMissingReplace(t *testing.T) {
	root := t.TempDir()
	parentDir := filepath.Join(root, "packages", "parent")
	leafDir := filepath.Join(root, "packages", "leaf")
	consumerDir := filepath.Join(root, "scenarios", "demo", "api")
	writeModule(t, leafDir, "module example.com/leaf\n\ngo 1.25.0\n", nil)
	writeModule(t, parentDir, "module example.com/parent\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n\nreplace example.com/leaf => ../leaf\n", nil)
	writeModule(t, consumerDir, "module demo/api\n\ngo 1.25.0\n\nrequire example.com/parent v0.0.0\n\nreplace example.com/parent => ../../../packages/parent\n", nil)
	topo := Topology{
		"example.com/parent": parentDir,
		"example.com/leaf":   leafDir,
	}

	missing, err := Plan(context.Background(), filepath.Join(consumerDir, "go.mod"), topo)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("missing = %#v", missing)
	}
	if missing[0].Module != "example.com/leaf" || missing[0].RelPath != "../../../packages/leaf" {
		t.Fatalf("missing[0] = %#v", missing[0])
	}
}

func TestPreviewSurfaceProducesDeterministicAfter(t *testing.T) {
	root := t.TempDir()
	leafDir := filepath.Join(root, "packages", "leaf")
	consumerDir := filepath.Join(root, "scenarios", "demo", "cli")
	writeModule(t, leafDir, "module example.com/leaf\n\ngo 1.25.0\n", nil)
	goModPath := filepath.Join(consumerDir, "go.mod")
	writeModule(t, consumerDir, "module demo/cli\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n", nil)
	topo := Topology{"example.com/leaf": leafDir}

	cand, err := PreviewSurface(context.Background(), goModPath, topo)
	if err != nil {
		t.Fatalf("PreviewSurface: %v", err)
	}
	if cand == nil {
		t.Fatal("expected a candidate, got nil")
	}
	if !strings.Contains(cand.After, "replace example.com/leaf => ../../../packages/leaf") {
		t.Fatalf("after missing replace:\n%s", cand.After)
	}
	// Preview must not mutate the original file.
	onDisk, _ := os.ReadFile(goModPath)
	if string(onDisk) != cand.Before {
		t.Fatal("PreviewSurface mutated the original go.mod")
	}
}

// TestApplySurfaceAddsReplaceAndIsIdempotent exercises the real go toolchain:
// it adds the missing replace, tidies, and a second apply is a no-op.
func TestApplySurfaceAddsReplaceAndIsIdempotent(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	root := t.TempDir()
	leafDir := filepath.Join(root, "packages", "leaf")
	consumerDir := filepath.Join(root, "scenarios", "demo", "cli")
	writeModule(t, leafDir, "module example.com/leaf\n\ngo 1.25.0\n", map[string]string{
		"leaf.go": "package leaf\n\n// F is a no-op leaf symbol.\nfunc F() {}\n",
	})
	goModPath := filepath.Join(consumerDir, "go.mod")
	writeModule(t, consumerDir,
		"module demo/cli\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n",
		map[string]string{
			"main.go": "package main\n\nimport \"example.com/leaf\"\n\nfunc main() { leaf.F() }\n",
		})
	topo := Topology{"example.com/leaf": leafDir}

	cand, err := ApplySurface(context.Background(), goModPath, topo)
	if err != nil {
		t.Fatalf("ApplySurface: %v", err)
	}
	if cand == nil || !cand.Applied {
		t.Fatalf("expected applied candidate, got %#v", cand)
	}
	after, _ := os.ReadFile(goModPath)
	if !strings.Contains(string(after), "replace example.com/leaf => ../../../packages/leaf") {
		t.Fatalf("go.mod missing replace after apply:\n%s", after)
	}

	// Idempotent: a second apply makes no change.
	again, err := ApplySurface(context.Background(), goModPath, topo)
	if err != nil {
		t.Fatalf("second ApplySurface: %v", err)
	}
	if again != nil {
		t.Fatalf("expected no-op on converged surface, got %#v", again)
	}
}

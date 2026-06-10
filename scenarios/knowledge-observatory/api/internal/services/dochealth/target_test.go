package dochealth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTarget(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	scenarioDir := filepath.Join(scenarios, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	t.Run("scenario name", func(t *testing.T) {
		tgt, err := svc.resolveTarget("demo", DocHealthOptions{})
		if err != nil || !tgt.isScenario || tgt.scenarioName != "demo" {
			t.Fatalf("scenario target: %#v err=%v", tgt, err)
		}
	})

	t.Run("path inside scenario promotes to scenario", func(t *testing.T) {
		tgt, err := svc.resolveTarget("", DocHealthOptions{Scope: "path", Path: filepath.Join(scenarioDir, "docs")})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !tgt.isScenario || tgt.scenarioName != "demo" {
			t.Fatalf("expected promotion to scenario demo, got %#v", tgt)
		}
		if tgt.root != scenarioDir {
			t.Fatalf("expected scenario root %q, got %q", scenarioDir, tgt.root)
		}
	})

	t.Run("project path is generic", func(t *testing.T) {
		tgt, err := svc.resolveTarget("", DocHealthOptions{Scope: "path", Path: filepath.Join(root, "docs")})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if tgt.isScenario {
			t.Fatalf("expected generic target, got scenario: %#v", tgt)
		}
		if tgt.label != "docs" {
			t.Fatalf("expected label 'docs', got %q", tgt.label)
		}
	})

	t.Run("relative project path anchored at repo root", func(t *testing.T) {
		tgt, err := svc.resolveTarget("", DocHealthOptions{Scope: "path", Path: "docs"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if tgt.isScenario || tgt.root != filepath.Join(root, "docs") {
			t.Fatalf("expected generic repo-root docs, got %#v", tgt)
		}
	})

	t.Run("missing path errors", func(t *testing.T) {
		if _, err := svc.resolveTarget("", DocHealthOptions{Scope: "path", Path: filepath.Join(root, "nope")}); err == nil {
			t.Fatal("expected error for missing path")
		}
	})

	t.Run("empty path errors", func(t *testing.T) {
		if _, err := svc.resolveTarget("", DocHealthOptions{Scope: "path"}); err == nil {
			t.Fatal("expected error for empty path")
		}
	})
}

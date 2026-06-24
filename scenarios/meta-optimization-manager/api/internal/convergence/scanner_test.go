package convergence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a tiny test helper.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFitnessScannerCountsLenses(t *testing.T) {
	root := t.TempDir()
	tmpl := filepath.Join(root, "templates", "scenarios", "demo")
	// registry.go + main.go are central wiring files (coordinated-edit surface).
	writeFile(t, filepath.Join(tmpl, "api", "registry.go"), "package x\n// callers must leave ID zero\nfunc A() {}\n")
	writeFile(t, filepath.Join(tmpl, "api", "main.go"), "package main\nfunc main() {}\n")
	// a hand-rolled fake is a drift surface; also contains a drift marker comment.
	writeFile(t, filepath.Join(tmpl, "internal", "fake_clock.go"), "package x\n// keep in sync with System\nvar Z = 1\n")
	// node_modules is skipped.
	writeFile(t, filepath.Join(tmpl, "ui", "node_modules", "junk.ts"), "export const x = 1\n")

	sc := NewFitnessScannerWithRoot(root)
	out, err := sc.Scan(context.Background(), "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 template, got %d", len(out))
	}
	tf := out[0]
	if tf.Template != "demo" {
		t.Fatalf("template = %q", tf.Template)
	}
	if tf.CoordinatedEditCount != 2 {
		t.Errorf("coordinated edits: want 2 (registry.go + main.go), got %d", tf.CoordinatedEditCount)
	}
	if tf.CommentOnlyContractCount != 1 {
		t.Errorf("comment-only contracts: want 1, got %d", tf.CommentOnlyContractCount)
	}
	// drift: 1 fake file (by name) + 1 in-comment "keep in sync" marker = 2.
	if tf.DriftSurfaceCount != 2 {
		t.Errorf("drift surfaces: want 2, got %d", tf.DriftSurfaceCount)
	}
	if tf.PerReplicaCost == 0 {
		t.Errorf("per-replica cost should count non-blank source LOC, got 0")
	}
	// node_modules must be skipped (no contribution from junk.ts).
	if tf.Tier == TierUnspecified {
		t.Errorf("tier should be derived")
	}
}

func TestFitnessScannerTemplateFilter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "templates", "scenarios", "a", "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "templates", "scenarios", "b", "main.go"), "package main\n")
	sc := NewFitnessScannerWithRoot(root)
	out, err := sc.Scan(context.Background(), "b")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(out) != 1 || out[0].Template != "b" {
		t.Fatalf("template filter failed: %+v", out)
	}
}

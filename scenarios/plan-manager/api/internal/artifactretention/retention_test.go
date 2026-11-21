package artifactretention

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	// Symlinks keep their creation mtime (Chtimes would follow them), so the
	// cycle clock sits well after the fixture is built.
	created = time.Now().Add(-60 * 24 * time.Hour)
	later   = time.Now().Add(30 * 24 * time.Hour)
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

// age sets every entry under root (not following symlinks) to the given time.
func age(t *testing.T, root string, at time.Time) {
	t.Helper()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return os.Chtimes(path, at, at)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// fixture builds an effort workspace whose isolated tree holds source,
// evidence-like files, build outputs and a symlink into a "live" repository.
func fixture(t *testing.T) (root, tree, repo string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "plan-artifacts")
	repo = filepath.Join(base, "repo")
	write(t, filepath.Join(repo, "scenarios", "agent-manager", "api", "node_modules", "keep.js"), 10)
	effort := filepath.Join(root, "efforts", "film-2026-10-01")
	write(t, filepath.Join(effort, "WORK.md"), 5)
	write(t, filepath.Join(effort, "final.mp4"), 50)
	tree = filepath.Join(effort, TreeDirName)
	wc := filepath.Join(tree, "scenarios", "web-console")
	write(t, filepath.Join(wc, "api", "main.go"), 7)
	write(t, filepath.Join(wc, "api", "web-console-api"), 100)
	write(t, filepath.Join(wc, "api", "web-console-api.freshness.json"), 3)
	write(t, filepath.Join(wc, "ui", "src", "App.tsx"), 9)
	write(t, filepath.Join(wc, "ui", "node_modules", "react", "index.js"), 200)
	write(t, filepath.Join(wc, "ui", "dist", "index.html"), 20)
	write(t, filepath.Join(wc, "platforms", "electron", "dist-electron", "app.AppImage"), 300)
	write(t, filepath.Join(wc, "platforms", "electron", "src", "main.ts"), 4)
	if err := os.Symlink(filepath.Join(repo, "scenarios", "agent-manager"), filepath.Join(tree, "scenarios", "agent-manager")); err != nil {
		t.Fatal(err)
	}
	// A plan workspace without a tree and a plan workspace whose "tree" is a
	// symlink must never be discovered.
	write(t, filepath.Join(root, "some-plan", "approach.md"), 3)
	if err := os.Symlink(filepath.Join(repo), filepath.Join(root, "some-plan", TreeDirName)); err != nil {
		t.Fatal(err)
	}
	age(t, root, created)
	return root, tree, repo
}

func run(t *testing.T, cfg Config) Receipt {
	t.Helper()
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return later }
	}
	if cfg.LivePaths == nil {
		cfg.LivePaths = func() ([]string, error) { return nil, nil }
	}
	receipt, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return receipt
}

func exists(path string) bool { _, err := os.Lstat(path); return err == nil }

func TestIdleUnusedTreeLosesOnlyRegenerableOutputs(t *testing.T) {
	root, tree, repo := fixture(t)
	receipt := run(t, Config{Root: root})

	if len(receipt.Trees) != 1 {
		t.Fatalf("want exactly the real effort tree, got %+v", receipt.Trees)
	}
	report := receipt.Trees[0]
	if report.Status != StatusPruned || report.Error != "" {
		t.Fatalf("want pruned, got %+v", report)
	}
	wc := filepath.Join(tree, "scenarios", "web-console")
	for _, gone := range []string{
		filepath.Join(wc, "api", "web-console-api"),
		filepath.Join(wc, "api", "web-console-api.freshness.json"),
		filepath.Join(wc, "ui", "node_modules"),
		filepath.Join(wc, "ui", "dist"),
		filepath.Join(wc, "platforms"),
	} {
		if exists(gone) {
			t.Errorf("regenerable output survived: %s", gone)
		}
	}
	for _, kept := range []string{
		filepath.Join(wc, "api", "main.go"),
		filepath.Join(wc, "ui", "src", "App.tsx"),
		filepath.Join(filepath.Dir(tree), "WORK.md"),
		filepath.Join(filepath.Dir(tree), "final.mp4"),
		filepath.Join(root, "some-plan", "approach.md"),
		filepath.Join(repo, "scenarios", "agent-manager", "api", "node_modules", "keep.js"),
		filepath.Join(tree, "scenarios", "agent-manager"),
	} {
		if !exists(kept) {
			t.Errorf("non-regenerable or out-of-tree path removed: %s", kept)
		}
	}
	if want := int64(100 + 3 + 200 + 20 + 300 + 4); report.FreedBytes != want {
		t.Errorf("freed %d, want %d", report.FreedBytes, want)
	}
	if receipt.FreedBytes != report.FreedBytes || len(receipt.OperatorReview) != 1 || receipt.OperatorReview[0] != tree {
		t.Errorf("receipt totals/operator review wrong: %+v", receipt)
	}
}

func TestLiveProcessProtectsTree(t *testing.T) {
	root, tree, _ := fixture(t)
	cwd := filepath.Join(tree, "scenarios", "web-console", "api")
	receipt := run(t, Config{Root: root, LivePaths: func() ([]string, error) { return []string{"/usr/bin/bash", cwd}, nil }})
	if got := receipt.Trees[0]; got.Status != StatusProtected || got.FreedBytes != 0 {
		t.Fatalf("want protected, got %+v", got)
	}
	if !exists(filepath.Join(tree, "scenarios", "web-console", "ui", "node_modules")) {
		t.Fatal("live tree was pruned")
	}
	if len(receipt.OperatorReview) != 0 {
		t.Fatalf("a live tree must not be offered for removal: %+v", receipt.OperatorReview)
	}
}

func TestLiveProbeFailureFailsClosed(t *testing.T) {
	root, tree, _ := fixture(t)
	receipt := run(t, Config{Root: root, LivePaths: func() ([]string, error) { return nil, os.ErrPermission }})
	if receipt.Trees[0].Status != StatusProtected || !exists(filepath.Join(tree, "scenarios", "web-console", "ui", "dist")) {
		t.Fatalf("probe failure must protect, got %+v", receipt.Trees[0])
	}
}

func TestRecentActivityWithinGraceProtectsTree(t *testing.T) {
	root, tree, _ := fixture(t)
	write(t, filepath.Join(tree, "scenarios", "web-console", "ui", "src", "Edit.tsx"), 1)
	recent := later.Add(-2 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(tree, "scenarios", "web-console", "ui", "src", "Edit.tsx"), recent, recent); err != nil {
		t.Fatal(err)
	}
	receipt := run(t, Config{Root: root, Grace: 7 * 24 * time.Hour})
	if receipt.Trees[0].Status != StatusProtected {
		t.Fatalf("recently edited tree must be protected, got %+v", receipt.Trees[0])
	}
}

func TestReportModeDeletesNothing(t *testing.T) {
	root, tree, _ := fixture(t)
	receipt := run(t, Config{Root: root, Mode: ModeReport})
	got := receipt.Trees[0]
	if got.Status != StatusPrunable || got.FreedBytes != 0 || receipt.ReclaimableBytes != got.RegenerableBytes || got.RegenerableBytes == 0 {
		t.Fatalf("report mode wrong: %+v", receipt)
	}
	if !exists(filepath.Join(tree, "scenarios", "web-console", "platforms")) {
		t.Fatal("report mode deleted data")
	}
}

func TestSecondCycleIsClean(t *testing.T) {
	root, _, _ := fixture(t)
	run(t, Config{Root: root})
	receipt := run(t, Config{Root: root})
	if receipt.Trees[0].Status != StatusClean {
		t.Fatalf("want clean, got %+v", receipt.Trees[0])
	}
}

func TestRegenerableClassification(t *testing.T) {
	dirs := map[string]bool{
		"scenarios/web-console/ui/node_modules":          true,
		"scenarios/web-console/platforms":                true,
		"scenarios/secrets-manager/platforms":            false,
		"scenarios/web-console/docs/platforms":           false,
		"scenarios/web-console/api/bin":                  true,
		"scenarios/web-console/.vrooli-artifact-stage-x": true,
		"scenarios/web-console/ui/src":                   false,
		"scenarios/device-studio/artifacts":              false,
	}
	for rel, want := range dirs {
		if got := IsRegenerableDir(rel); got != want {
			t.Errorf("IsRegenerableDir(%q)=%v want %v", rel, got, want)
		}
	}
	files := map[string]bool{
		"scenarios/web-console/api/web-console-api":    true,
		"scenarios/web-console/api/main":               true,
		"scenarios/web-console/api/main.go":            false,
		"scenarios/web-console/ui/web-console-ui":      true,
		"scenarios/web-console/api/x.freshness.json":   true,
		"scenarios/web-console/docs/web-console-api":   false,
		"scenarios/web-console/api/handlers/route-api": false,
	}
	for rel, want := range files {
		if got := IsRegenerableFile(rel); got != want {
			t.Errorf("IsRegenerableFile(%q)=%v want %v", rel, got, want)
		}
	}
}

func TestRemoveOutputRefusesSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	tree := filepath.Join(base, "tree")
	outside := filepath.Join(base, "outside", "node_modules")
	write(t, filepath.Join(outside, "x.js"), 1)
	if err := os.MkdirAll(filepath.Join(tree, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tree, "scenarios", "node_modules")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := removeOutput(tree, link, os.RemoveAll); err == nil {
		t.Fatal("symlink output must be refused")
	}
	if err := removeOutput(tree, outside, os.RemoveAll); err == nil {
		t.Fatal("path outside the tree must be refused")
	}
	if !exists(filepath.Join(outside, "x.js")) {
		t.Fatal("out-of-tree data removed")
	}
}

func TestWriteReceiptRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retention", "isolated-tree-receipt.json")
	in := Receipt{Owner: "plan-manager", Mode: ModeEnforce, FreedBytes: 42, Trees: []TreeReport{{Path: "/x", Status: StatusPruned}}}
	if err := WriteReceipt(path, in); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out Receipt
	if err := json.Unmarshal(raw, &out); err != nil || out.FreedBytes != 42 || out.Trees[0].Status != StatusPruned {
		t.Fatalf("round trip failed: %+v %v", out, err)
	}
}

func TestSettingsFromEnv(t *testing.T) {
	env := map[string]string{"PLAN_ARTIFACTS_RETENTION_MODE": "report", "PLAN_ARTIFACTS_RETENTION_GRACE": "48h"}
	s, err := SettingsFromEnv(func(k string) string { return env[k] })
	if err != nil || s.Mode != ModeReport || s.Grace != 48*time.Hour || s.Interval != 6*time.Hour {
		t.Fatalf("settings: %+v %v", s, err)
	}
	if _, err := SettingsFromEnv(func(k string) string { return map[string]string{"PLAN_ARTIFACTS_RETENTION_MODE": "yolo"}[k] }); err == nil {
		t.Fatal("invalid mode must fail")
	}
	s, _ = SettingsFromEnv(func(string) string { return "" })
	if s.Mode != ModeEnforce || s.Grace != DefaultGrace || s.Disabled {
		t.Fatalf("defaults wrong: %+v", s)
	}
}

// A hard-linked package-store file inside node_modules can be refreshed by a
// build in another tree; that is not activity in this tree.
func TestActivityInsideOutputsDoesNotProtectTree(t *testing.T) {
	root, tree, _ := fixture(t)
	inside := filepath.Join(tree, "scenarios", "web-console", "ui", "node_modules", "react", "index.js")
	recent := later.Add(-time.Hour)
	if err := os.Chtimes(inside, recent, recent); err != nil {
		t.Fatal(err)
	}
	receipt := run(t, Config{Root: root})
	if receipt.Trees[0].Status != StatusPruned {
		t.Fatalf("output-internal mtime must not count as activity, got %+v", receipt.Trees[0])
	}
}

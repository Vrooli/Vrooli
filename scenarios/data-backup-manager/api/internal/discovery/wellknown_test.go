package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/sources"
)

// buildRuntimeRoot fakes a ~/.vrooli tree: the well-known dirs/files plus some
// ephemeral dirs that must NOT be suggested.
func buildRuntimeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirWithFile := func(rel, file, contents string) {
		dir := filepath.Join(root, rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if file != "" {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(contents), 0o644); err != nil {
				t.Fatalf("write %s: %v", file, err)
			}
		}
	}
	// Well-known (should appear).
	mkdirWithFile("plans", "p.md", "plan body")
	mkdirWithFile("state", "s.json", "{}")
	mkdirWithFile("config", "c.yaml", "k: v")
	if err := os.WriteFile(filepath.Join(root, "secrets.json"), []byte(`{"k":"v"}`), 0o600); err != nil {
		t.Fatalf("write secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime.db"), []byte("SQLitefakebytes"), 0o644); err != nil {
		t.Fatalf("write runtime.db: %v", err)
	}
	// Ephemeral (should NOT appear).
	mkdirWithFile("logs", "a.log", "noise")
	mkdirWithFile("cache", "x", "noise")
	mkdirWithFile("metrics", "m", "noise")
	mkdirWithFile("bin", "tool", "noise")
	mkdirWithFile("processes", "p", "noise")
	return root
}

func TestWellKnownScannerCoversRuntimeStateOnly(t *testing.T) {
	root := buildRuntimeRoot(t)
	scanner := discovery.NewWellKnownScannerWithRoot(root)

	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	byName := map[string]discovery.TargetCandidate{}
	for _, c := range got {
		byName[c.Name] = c
	}

	wantNames := []string{"plans", "state", "config", "secrets", "runtime-db"}
	if len(got) != len(wantNames) {
		t.Fatalf("expected %d candidates, got %d: %+v", len(wantNames), len(got), got)
	}
	for _, n := range wantNames {
		c, ok := byName[n]
		if !ok {
			t.Fatalf("missing well-known candidate %q", n)
		}
		if c.Owner != "vrooli" {
			t.Errorf("%q owner = %q, want vrooli", n, c.Owner)
		}
		if !filepath.IsAbs(c.Locator) {
			t.Errorf("%q locator %q is not absolute", n, c.Locator)
		}
	}

	// runtime.db is suggested as a SQLite source (Contract Decision D5).
	if k := byName["runtime-db"].SourceKind; k != sources.KindSQLite {
		t.Errorf("runtime-db source kind = %q, want sqlite", k)
	}
	if k := byName["plans"].SourceKind; k != sources.KindFilesystem {
		t.Errorf("plans source kind = %q, want filesystem", k)
	}

	// No ephemeral dirs leaked in.
	for _, c := range got {
		base := filepath.Base(c.Locator)
		for _, banned := range []string{"logs", "cache", "metrics", "bin", "processes"} {
			if base == banned {
				t.Errorf("ephemeral dir %q must not be suggested", banned)
			}
		}
	}
}

func TestWellKnownScannerSkipsMissingAndEmpty(t *testing.T) {
	root := t.TempDir()
	// Only an empty plans dir and an empty secrets file → both skipped.
	if err := os.MkdirAll(filepath.Join(root, "plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secrets.json"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	scanner := discovery.NewWellKnownScannerWithRoot(root)
	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no candidates for empty dir + empty file, got %+v", got)
	}
}

func TestWellKnownScannerEmptyRootYieldsNothing(t *testing.T) {
	scanner := discovery.NewWellKnownScannerWithRoot("")
	got, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no candidates for empty root, got %+v", got)
	}
}

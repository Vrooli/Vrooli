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
	// Durable (regenerable=false in the contract → should appear).
	mkdirWithFile("plans", "p.md", "plan body")
	mkdirWithFile("state", "s.json", "{}")
	mkdirWithFile("config", "c.yaml", "k: v")
	mkdirWithFile("data", "d.bin", "durable")
	// runtime.db lives under state/ (contract runtime_db path = state/runtime.db).
	if err := os.WriteFile(filepath.Join(root, "state", "runtime.db"), []byte("SQLitefakebytes"), 0o644); err != nil {
		t.Fatalf("write runtime.db: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "secrets.json"), []byte(`{"k":"v"}`), 0o600); err != nil {
		t.Fatalf("write secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "secrets.enc.json"), []byte(`{"enc":"v"}`), 0o600); err != nil {
		t.Fatalf("write secrets.enc: %v", err)
	}
	// Regenerable (should NOT appear).
	mkdirWithFile("logs", "a.log", "noise")
	mkdirWithFile("cache", "x", "noise")
	mkdirWithFile("metrics", "m", "noise")
	mkdirWithFile("bin", "tool", "noise")
	mkdirWithFile("processes", "p", "noise")
	mkdirWithFile("build", "b", "noise")
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

	// The durable inventory is contract-driven: plans/state/config/data plus the
	// secrets pair and the runtime DB. `data` is now first-class (it was silently
	// omitted by the old hard-coded list).
	wantNames := []string{"plans", "state", "config", "data", "secrets", "secrets-enc", "runtime-db"}
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

	// data/ must be suggested (regression guard for the old omission).
	if _, ok := byName["data"]; !ok {
		t.Error("data must be suggested as a durable target")
	}

	// runtime.db is suggested as a SQLite source (Contract Decision D5) and lives
	// under state/ per the contract runtime_db path.
	if k := byName["runtime-db"].SourceKind; k != sources.KindSQLite {
		t.Errorf("runtime-db source kind = %q, want sqlite", k)
	}
	if base := filepath.Base(byName["runtime-db"].Locator); base != "runtime.db" {
		t.Errorf("runtime-db locator base = %q, want runtime.db", base)
	}
	if k := byName["plans"].SourceKind; k != sources.KindFilesystem {
		t.Errorf("plans source kind = %q, want filesystem", k)
	}

	// No regenerable dirs leaked in.
	for _, c := range got {
		base := filepath.Base(c.Locator)
		for _, banned := range []string{"logs", "cache", "metrics", "bin", "processes", "build"} {
			if base == banned {
				t.Errorf("regenerable dir %q must not be suggested", banned)
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

package scenariospec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeManifest writes a scenarios/<scenario>/.vrooli/service.json under root.
func writeManifest(t *testing.T, root, scenario, body string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

// fixture builds an Inspector rooted at temp repo + temp home dirs.
func fixture(t *testing.T) (*Inspector, string, string) {
	t.Helper()
	repo := t.TempDir()
	home := t.TempDir()
	insp := &Inspector{
		repoRoot: func() (string, error) { return repo, nil },
		homeDir:  func() (string, error) { return home, nil },
	}
	return insp, repo, home
}

// seedDataDir creates the conventional data dir for a scenario and, when
// nonEmpty, drops a file in it.
func seedDataDir(t *testing.T, home, scenario string, nonEmpty bool) string {
	t.Helper()
	dir := filepath.Join(home, ".vrooli", "data", storageAppID, scenario)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if nonEmpty {
		if err := os.WriteFile(filepath.Join(dir, "state.db"), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed data file: %v", err)
		}
	}
	return dir
}

func TestInspect_PostgresEnabled(t *testing.T) {
	insp, repo, _ := fixture(t)
	writeManifest(t, repo, "alpha", `{
		"dependencies": {"resources": {
			"postgres": {"enabled": true, "type": "postgres"},
			"qdrant":   {"enabled": true, "type": "qdrant"}
		}}
	}`)

	facts, err := insp.Inspect(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !facts.UsesPostgres {
		t.Fatalf("UsesPostgres = false, want true (enabled postgres resource)")
	}
}

func TestInspect_PostgresDisabledOrAbsent(t *testing.T) {
	insp, repo, _ := fixture(t)
	// data-backup-manager's own shape: postgres declared but disabled.
	writeManifest(t, repo, "beta", `{
		"dependencies": {"resources": {
			"postgres": {"enabled": false, "type": "postgres"},
			"kopia":    {"enabled": true, "type": "kopia"}
		}}
	}`)

	facts, err := insp.Inspect(context.Background(), "beta")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if facts.UsesPostgres {
		t.Fatalf("UsesPostgres = true, want false (postgres disabled)")
	}
}

func TestInspect_PostgresTypeFallsBackToResourceKey(t *testing.T) {
	insp, repo, _ := fixture(t)
	// No "type" field — detection falls back to the resource key name.
	writeManifest(t, repo, "gamma", `{
		"dependencies": {"resources": {"postgres": {"enabled": true}}}
	}`)

	facts, err := insp.Inspect(context.Background(), "gamma")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !facts.UsesPostgres {
		t.Fatalf("UsesPostgres = false, want true (key-name fallback)")
	}
}

func TestInspect_MalformedManifestIsBestEffort(t *testing.T) {
	insp, repo, _ := fixture(t)
	writeManifest(t, repo, "delta", `{not valid json`)

	facts, err := insp.Inspect(context.Background(), "delta")
	if err != nil {
		t.Fatalf("inspect: %v (malformed manifest must not error)", err)
	}
	if facts.UsesPostgres {
		t.Fatalf("UsesPostgres = true, want false (malformed manifest yields no facts)")
	}
}

func TestInspect_DataDirPresentVsEmpty(t *testing.T) {
	insp, repo, home := fixture(t)
	writeManifest(t, repo, "eps", `{"dependencies":{"resources":{}}}`)

	// No data dir at all → not present.
	facts, err := insp.Inspect(context.Background(), "eps")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if facts.DataDirPresent {
		t.Fatalf("DataDirPresent = true, want false (no data dir)")
	}
	if want := filepath.Join(home, ".vrooli", "data", storageAppID, "eps"); facts.DataDir != want {
		t.Fatalf("DataDir = %q, want %q", facts.DataDir, want)
	}

	// Empty data dir → still not present (nothing to back up).
	seedDataDir(t, home, "eps", false)
	facts, _ = insp.Inspect(context.Background(), "eps")
	if facts.DataDirPresent {
		t.Fatalf("DataDirPresent = true, want false (empty data dir)")
	}

	// Non-empty data dir → present.
	seedDataDir(t, home, "eps", true)
	facts, _ = insp.Inspect(context.Background(), "eps")
	if !facts.DataDirPresent {
		t.Fatalf("DataDirPresent = false, want true (non-empty data dir)")
	}
}

func TestInspect_BlankScenario(t *testing.T) {
	insp, _, _ := fixture(t)
	if _, err := insp.Inspect(context.Background(), "  "); err == nil {
		t.Fatalf("want error for blank scenario")
	}
}

func TestInspect_UnresolvableRepoRoot(t *testing.T) {
	insp := &Inspector{
		repoRoot: func() (string, error) { return "", errors.New("no repo") },
		homeDir:  os.UserHomeDir,
	}
	if _, err := insp.Inspect(context.Background(), "alpha"); err == nil {
		t.Fatalf("want error when repo root is unresolvable")
	}
}

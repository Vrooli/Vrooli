package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRequiresConfigDir(t *testing.T) {
	if _, err := Resolve(""); err == nil {
		t.Fatal("Resolve(\"\") returned nil error; expected required-arg failure")
	}
}

func TestResolveReturnsDistinctNonEmptyRoots(t *testing.T) {
	// configDir resolves via filepath.Abs; the test only asserts that the
	// four storage roots are wired and distinct. We pass the test's own
	// temp dir so the result is deterministic.
	roots, err := Resolve(t.TempDir())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for name, val := range map[string]string{
		"Config":       roots.Config,
		"RuntimeData":  roots.RuntimeData,
		"RuntimeCache": roots.RuntimeCache,
		"RepoRoot":     roots.RepoRoot,
		"ScenariosDir": roots.ScenariosDir,
	} {
		if strings.TrimSpace(val) == "" {
			t.Errorf("Resolve: %s is empty", name)
		}
	}
	// RuntimeData and RuntimeCache must point to different filesystem
	// locations — they are different storage classes.
	if roots.RuntimeData == roots.RuntimeCache {
		t.Errorf("RuntimeData == RuntimeCache (%q); classes must be distinct", roots.RuntimeData)
	}
	if filepath.Join(roots.RepoRoot, "scenarios") != roots.ScenariosDir {
		t.Errorf("ScenariosDir %q != RepoRoot/scenarios %q", roots.ScenariosDir, filepath.Join(roots.RepoRoot, "scenarios"))
	}
}

func TestRootsForTestUsesTempDir(t *testing.T) {
	roots := RootsForTest(t)
	if roots.Config == "" || roots.RuntimeData == "" || roots.RuntimeCache == "" || roots.RepoRoot == "" || roots.ScenariosDir == "" {
		t.Fatalf("RootsForTest returned empty root: %+v", roots)
	}
	base := filepath.Dir(roots.Config)
	for name, val := range map[string]string{
		"RuntimeData":  roots.RuntimeData,
		"RuntimeCache": roots.RuntimeCache,
	} {
		if filepath.Dir(val) != base {
			t.Errorf("%s %q not rooted under shared temp base %q", name, val, base)
		}
	}
	if filepath.Join(roots.RepoRoot, "scenarios") != roots.ScenariosDir {
		t.Errorf("ScenariosDir derivation broken: %q vs %q", roots.ScenariosDir, filepath.Join(roots.RepoRoot, "scenarios"))
	}
}

func TestBackupForPlacesUnderRuntimeDataBackups(t *testing.T) {
	roots := RootsForTest(t)
	got := roots.BackupFor("teams/director-swarm/shared/knowledge.jsonl", "20260528T120000Z")
	want := filepath.Join(roots.RuntimeData, "backups", "teams/director-swarm/shared/knowledge.jsonl.backup-20260528T120000Z")
	if got != want {
		t.Errorf("BackupFor with suffix:\n  got  %q\n  want %q", got, want)
	}
	// Empty suffix means a plain .backup sibling under the backups root.
	got = roots.BackupFor("teams/x/team.json", "")
	want = filepath.Join(roots.RuntimeData, "backups", "teams/x/team.json.backup")
	if got != want {
		t.Errorf("BackupFor without suffix:\n  got  %q\n  want %q", got, want)
	}
	// Critically: backup never lands under Config.
	if strings.HasPrefix(got, roots.Config) {
		t.Errorf("BackupFor leaked into Config root: %q", got)
	}
}

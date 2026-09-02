package autohealwatchdog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

// writeLoopTree lays out the minimal source tree loopBinaryStale walks.
func writeLoopTree(t *testing.T) (root string, loop string) {
	t.Helper()
	root = t.TempDir()
	scenario := filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal")
	loopDir := filepath.Join(scenario, "cli", "loop")
	recoverDir := filepath.Join(scenario, "langrecover")
	for _, dir := range []string{loopDir, recoverDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(loopDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recoverDir, "signatures.go"), []byte("package langrecover\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, filepath.Join(scenario, "cli", "vrooli-autoheal-loop")
}

func touch(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

func TestLoopBinaryStaleWhenMissing(t *testing.T) {
	root, loop := writeLoopTree(t)
	stale, reason := loopBinaryStale(root, loop)
	if !stale {
		t.Fatal("a missing binary must be stale")
	}
	if reason != "binary missing" {
		t.Errorf("reason = %q", reason)
	}
}

func TestLoopBinaryFreshWhenNewerThanSources(t *testing.T) {
	root, loop := writeLoopTree(t)
	if err := os.WriteFile(loop, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, loop, time.Now().Add(time.Hour))

	if stale, reason := loopBinaryStale(root, loop); stale {
		t.Fatalf("binary newer than sources must be fresh, got %q", reason)
	}
}

// The 2026-09-01 case: the binary was months older than its source, the
// safeguard reported "Already present", and the fix never shipped.
func TestLoopBinaryStaleWhenSourceIsNewer(t *testing.T) {
	root, loop := writeLoopTree(t)
	if err := os.WriteFile(loop, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, loop, time.Now().Add(-48*time.Hour))

	stale, reason := loopBinaryStale(root, loop)
	if !stale {
		t.Fatal("source newer than binary must be stale")
	}
	if reason == "" {
		t.Error("stale result must explain which file is newer")
	}
}

// The loop depends on langrecover, so a change there must also force a
// rebuild: otherwise the recovery floor runs superseded detection logic.
func TestLoopBinaryStaleWhenLangrecoverIsNewer(t *testing.T) {
	root, loop := writeLoopTree(t)
	if err := os.WriteFile(loop, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-48 * time.Hour)
	touch(t, loop, past)
	touch(t, filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal", "cli", "loop", "main.go"), past.Add(-time.Hour))
	touch(t, filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal", "langrecover", "signatures.go"), time.Now())

	stale, reason := loopBinaryStale(root, loop)
	if !stale {
		t.Fatal("a newer langrecover source must make the loop binary stale")
	}
	if reason == "" {
		t.Error("expected a reason naming the newer file")
	}
}

// Test files do not change the built binary, so they must not force rebuilds.
func TestLoopBinaryIgnoresTestSources(t *testing.T) {
	root, loop := writeLoopTree(t)
	if err := os.WriteFile(loop, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-48 * time.Hour)
	loopDir := filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal", "cli", "loop")
	touch(t, filepath.Join(loopDir, "main.go"), past.Add(-time.Hour))
	touch(t, filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal", "langrecover", "signatures.go"), past.Add(-time.Hour))
	touch(t, loop, past)

	testFile := filepath.Join(loopDir, "main_test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	touch(t, testFile, time.Now())

	if stale, reason := loopBinaryStale(root, loop); stale {
		t.Fatalf("a newer test file must not force a rebuild, got %q", reason)
	}
}

// A source tree that cannot be walked must not trigger endless rebuilds.
func TestLoopBinaryNotStaleWhenSourcesUnreadable(t *testing.T) {
	root := t.TempDir()
	loop := filepath.Join(root, "vrooli-autoheal-loop")
	if err := os.WriteFile(loop, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No scenario tree exists under root, so the walk fails.
	if stale, _ := loopBinaryStale(root, loop); stale {
		t.Fatal("unreadable sources must not report stale")
	}
}

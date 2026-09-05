package autohealwatchdog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// writeLoopComponent lays out the loop component the way the lifecycle engine
// leaves it after `vrooli scenario setup`: a module directory, a built binary,
// and the freshness manifest stamped next to the binary. It returns the repo
// root and the binary path.
func writeLoopComponent(t *testing.T) (root string, loop string) {
	t.Helper()
	root = t.TempDir()
	loopDir := loopModuleDir(root)
	if err := os.MkdirAll(loopDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(loopDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(loopDir, "go.mod"), []byte("module vrooli-autoheal-loop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loop = loopPath(root, "linux")
	if err := os.WriteFile(loop, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, loop
}

// stampLoopManifest records the manifest the engine would write for the
// current loop sources. The write time is placed in the future so the
// stat-cache trusts unchanged files; content edits are still re-hashed.
func stampLoopManifest(t *testing.T, root, loop string) {
	t.Helper()
	spec := cliutil.FreshnessSpec{
		SourceRoot:   loopModuleDir(root),
		ContextRoot:  root,
		Inputs:       []string{"scenarios/vrooli-autoheal/cli/loop"},
		SkipFiles:    []string{filepath.Base(loop)},
		SkipSuffixes: []string{"_test.go", cliutil.FreshnessManifestSuffix},
	}
	manifest, err := cliutil.ComputeFreshnessManifest(spec, "go_module", map[string]string{"toolchain": "go1.25.0"}, time.Now().Add(time.Minute).UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	if err := cliutil.WriteFreshnessManifest(cliutil.FreshnessManifestPath(loop), manifest); err != nil {
		t.Fatal(err)
	}
}

func TestLoopBinaryVerdictStaleWhenMissing(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	if err := os.Remove(loop); err != nil {
		t.Fatal(err)
	}
	verdict, reason := loopBinaryVerdict(root, loop)
	if verdict != verdictStale {
		t.Fatalf("verdict = %q, want stale for a missing binary", verdict)
	}
	if reason != "binary missing" {
		t.Errorf("reason = %q", reason)
	}
}

// A binary the engine never stamped is unproven. Reporting it fresh is how the
// old mtime path let a months-old watchdog read "Already present".
func TestLoopBinaryVerdictUnknownWithoutManifest(t *testing.T) {
	root, loop := writeLoopComponent(t)
	verdict, reason := loopBinaryVerdict(root, loop)
	if verdict != verdictUnknown {
		t.Fatalf("verdict = %q, want unknown without a manifest", verdict)
	}
	if reason == "" {
		t.Error("unknown verdict must name the missing manifest")
	}
}

func TestLoopBinaryVerdictFreshWhenManifestMatches(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	if verdict, reason := loopBinaryVerdict(root, loop); verdict != verdictFresh {
		t.Fatalf("verdict = %q (%s), want fresh when sources match the manifest", verdict, reason)
	}
}

// The 2026-09-01 case: the binary was months older than its source, the
// safeguard reported "Already present", and the fix never shipped.
func TestLoopBinaryVerdictStaleWhenSourceChanged(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	if err := os.WriteFile(filepath.Join(loopModuleDir(root), "main.go"), []byte("package main\n\n// changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	verdict, reason := loopBinaryVerdict(root, loop)
	if verdict != verdictStale {
		t.Fatalf("verdict = %q, want stale after a source edit", verdict)
	}
	if reason == "" {
		t.Error("stale result must explain what changed")
	}
}

// Test files do not change the built binary, so they must not stale it.
func TestLoopBinaryVerdictIgnoresTestSources(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	if err := os.WriteFile(filepath.Join(loopModuleDir(root), "main_test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if verdict, reason := loopBinaryVerdict(root, loop); verdict != verdictFresh {
		t.Fatalf("a new test file must not stale the loop, got %q (%s)", verdict, reason)
	}
}

// A corrupt manifest is unknown, not fresh and not an endless rebuild loop.
func TestLoopBinaryVerdictUnknownWhenManifestCorrupt(t *testing.T) {
	root, loop := writeLoopComponent(t)
	if err := os.WriteFile(cliutil.FreshnessManifestPath(loop), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if verdict, _ := loopBinaryVerdict(root, loop); verdict != verdictUnknown {
		t.Fatalf("verdict = %q, want unknown for a corrupt manifest", verdict)
	}
}

package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/scenario"
)

// TestEffectiveSourceDirRealFloorIsolatesContent is the deterministic encoding of
// the Baseline Modes shadow-mode live spike (plan §9/§10). Where TestEffectiveSourceDir
// asserts the decision returns the right path STRING against a fake resolver, this
// drives the real floor end-to-end: it captures a real restore point with
// baselinefloor.Capture, edits the working tree out from under it, and reads the
// BYTES at whichever directory effectiveSourceDir resolves to. It proves the spike's
// core guarantee as a permanent test — while a shadow engagement is open, the live
// (serving) instance is pinned to the frozen OLD copy even after the working tree
// changes to NEW; the @shadow instance follows the NEW working tree; and once the
// engagement closes (promote/abandon), live follows the working tree again.
//
// It lives in package lifecycle (not lifecycle_test) so it can call the unexported
// effectiveSourceDir; baselinefloor does not import lifecycle, so there is no cycle.
// The floor→fact mapping the production resolver performs is covered separately by
// internal/floorengagement/resolver_test.go; here the EngagementInfo is fed from a
// REAL captured restore-point path so the decision routes to real frozen content,
// not just a string.
func TestEffectiveSourceDirRealFloorIsolatesContent(t *testing.T) {
	const marker = "marker.txt"

	// 1. A working tree holding OLD code.
	workingTree := t.TempDir()
	writeFile(t, filepath.Join(workingTree, marker), "OLD")

	// 2. Capture a real restore point (the "baseline start" capture-before-merge step).
	store := baselinefloor.NewStore(t.TempDir())
	restorePoint := store.RestorePointPath("demo", "spike")
	if _, err := baselinefloor.Capture(workingTree, restorePoint, nil); err != nil {
		t.Fatalf("capture restore point: %v", err)
	}

	// 3. The candidate merges into the working tree: OLD -> NEW. The frozen copy
	//    must NOT see this.
	writeFile(t, filepath.Join(workingTree, marker), "NEW")

	item := func(variant string) scenario.Scenario {
		return scenario.Scenario{Slug: "demo", Path: workingTree, Variant: variant}
	}

	// 4. While the shadow engagement is open, the resolver reports the split with
	//    the real captured copy path.
	engaged := &fakeEngagementResolver{
		engaged: true,
		info: EngagementInfo{
			RestorePointDir: restorePoint,
			Slug:            "spike",
			Mode:            "shadow",
		},
	}
	r := &Runner{Engagements: engaged}

	// Live (empty variant) must resolve to the frozen copy and read OLD, even
	// though the working tree now holds NEW — this is the isolation guarantee.
	liveDir, err := r.effectiveSourceDir(item(""))
	if err != nil {
		t.Fatalf("live effectiveSourceDir while engaged: %v", err)
	}
	if got := readFile(t, filepath.Join(liveDir, marker)); got != "OLD" {
		t.Errorf("live serves %q from %q, want OLD (frozen copy)", got, liveDir)
	}

	// Shadow (@shadow) must resolve to the working tree and read NEW — the
	// candidate runs from the working tree the agent edited.
	shadowDir, err := r.effectiveSourceDir(item("shadow"))
	if err != nil {
		t.Fatalf("shadow effectiveSourceDir while engaged: %v", err)
	}
	if got := readFile(t, filepath.Join(shadowDir, marker)); got != "NEW" {
		t.Errorf("shadow serves %q from %q, want NEW (working tree)", got, shadowDir)
	}

	// 5. Engagement closes (promote/abandon both end the split). Live now follows
	//    the working tree again — read NEW.
	closed := &fakeEngagementResolver{engaged: false}
	r = &Runner{Engagements: closed}
	postDir, err := r.effectiveSourceDir(item(""))
	if err != nil {
		t.Fatalf("live effectiveSourceDir after close: %v", err)
	}
	if got := readFile(t, filepath.Join(postDir, marker)); got != "NEW" {
		t.Errorf("after engagement close live serves %q from %q, want NEW (working tree)", got, postDir)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}

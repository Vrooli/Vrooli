package changedetect_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver/changedetect"
	"workspace-sandbox/internal/types"
)

// I-CHANGE-1 contract test: GetChangedFiles is deterministic for a
// given filesystem state. Both strategies share the same fixture
// matrix here so a future change that subtly diverges the two surfaces
// fails loudly.

// stableFileID is the same hash function the driver package exposes,
// inlined into the test to avoid an import cycle.
func stableFileID(_ uuid.UUID, p string) uuid.UUID {
	// uuid.NewSHA1 keyed by namespace constant + path is what
	// driver.StableFileID does. We re-implement here so the test isn't
	// coupled to the driver package's helper.
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(p))
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func writeWhiteout(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	// Filename-style whiteout (.wh.NAME) — matches one of two ways the
	// strategy detects deletions.
	if err := os.WriteFile(filepath.Join(dir, ".wh."+name), nil, 0o644); err != nil {
		t.Fatalf("write whiteout: %v", err)
	}
}

func runStrategy(t *testing.T, strategy changedetect.Strategy, lower, upper string) []*types.FileChange {
	t.Helper()
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	changes, err := changedetect.Walk(context.Background(),
		changedetect.WalkOpts{Lower: lower, Upper: upper, SandboxID: uuid.New()},
		strategy,
		now,
	)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	return changes
}

type fixedIgnoreMatcher map[string]struct{}

func (m fixedIgnoreMatcher) Ignored(_ context.Context, _ []string) (map[string]struct{}, error) {
	return m, nil
}

func mustHave(t *testing.T, changes []*types.FileChange, wantPath string, wantType types.ChangeType) {
	t.Helper()
	for _, c := range changes {
		if c.FilePath == wantPath {
			if c.ChangeType != wantType {
				t.Errorf("path %q ChangeType = %q, want %q", wantPath, c.ChangeType, wantType)
			}
			return
		}
	}
	var paths []string
	for _, c := range changes {
		paths = append(paths, c.FilePath)
	}
	t.Errorf("missing path %q (want %s); got %v", wantPath, wantType, paths)
}

func mustNotHave(t *testing.T, changes []*types.FileChange, badPath string) {
	t.Helper()
	for _, c := range changes {
		if c.FilePath == badPath {
			t.Errorf("unexpected change for path %q (type=%s)", badPath, c.ChangeType)
		}
	}
}

// TestStrategy_AddedFiles covers a clean Added detection for both
// strategies — the most common case.
func TestStrategy_AddedFiles(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			writeFile(t, filepath.Join(upper, "added.txt"), "new")
			changes := runStrategy(t, tc.strategy, lower, upper)
			mustHave(t, changes, "added.txt", types.ChangeTypeAdded)
		})
	}
}

// TestStrategy_ModifiedFiles covers Modified detection: same path,
// different content.
func TestStrategy_ModifiedFiles(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			writeFile(t, filepath.Join(lower, "exists.txt"), "old")
			writeFile(t, filepath.Join(upper, "exists.txt"), "new")
			changes := runStrategy(t, tc.strategy, lower, upper)
			mustHave(t, changes, "exists.txt", types.ChangeTypeModified)
		})
	}
}

// TestStrategy_TrackedDotFilesCaptured ensures driver policy never drops
// tracked configuration merely because its name begins with a dot.
func TestStrategy_TrackedDotFilesCaptured(t *testing.T) {
	t.Run("copy", func(t *testing.T) {
		lower := t.TempDir()
		upper := t.TempDir()
		writeFile(t, filepath.Join(upper, ".local"), "x")
		writeFile(t, filepath.Join(upper, "visible.txt"), "v")
		changes := runStrategy(t, copyStrategyImpl(), lower, upper)
		mustHave(t, changes, ".local", types.ChangeTypeAdded)
		mustHave(t, changes, "visible.txt", types.ChangeTypeAdded)
	})
}

func TestWalk_IgnoreMatcherDropsGitignoredPath(t *testing.T) {
	lower, upper := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(upper, "ignored", "cache.bin"), "noise")
	writeFile(t, filepath.Join(upper, ".vrooli", "service.json"), "{}")
	changes, err := changedetect.Walk(context.Background(), changedetect.WalkOpts{
		Lower: lower, Upper: upper, SandboxID: uuid.New(),
		IgnoreMatcher: fixedIgnoreMatcher{"ignored": {}},
	}, copyStrategyImpl(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	mustNotHave(t, changes, filepath.Join("ignored", "cache.bin"))
	mustHave(t, changes, filepath.Join(".vrooli", "service.json"), types.ChangeTypeAdded)
}

func TestWalk_CrossStrategyEquivalenceWithSharedIgnorePolicy(t *testing.T) {
	lower, upper := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(upper, "main.go"), "package main")
	writeFile(t, filepath.Join(upper, ".vrooli", "service.json"), "{}")
	writeFile(t, filepath.Join(upper, "ignored", "cache.bin"), "noise")
	writeFile(t, filepath.Join(upper, ".git", "HEAD"), "ref: refs/heads/main")
	writeFile(t, filepath.Join(upper, ".overlay", "work", "artifact"), "internal")
	opts := changedetect.WalkOpts{Lower: lower, Upper: upper, SandboxID: uuid.New(), IgnoreMatcher: fixedIgnoreMatcher{"ignored": {}}}
	got := make([][]*types.FileChange, 2)
	var err error
	got[0], err = changedetect.Walk(context.Background(), opts, overlayStrategyImpl(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	got[1], err = changedetect.Walk(context.Background(), opts, copyStrategyImpl(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	paths := func(changes []*types.FileChange) []string {
		out := make([]string, 0, len(changes))
		for _, c := range changes {
			out = append(out, c.FilePath)
		}
		return out
	}
	if diff := strings.Join(paths(got[0]), ",") + "|" + strings.Join(paths(got[1]), ","); strings.Join(paths(got[0]), ",") != strings.Join(paths(got[1]), ",") {
		t.Fatalf("strategies diverged: %s", diff)
	}
	for _, changes := range got {
		mustHave(t, changes, filepath.Join(".vrooli", "service.json"), types.ChangeTypeAdded)
		mustNotHave(t, changes, filepath.Join("ignored", "cache.bin"))
		mustNotHave(t, changes, filepath.Join(".git", "HEAD"))
		mustNotHave(t, changes, filepath.Join(".overlay", "work", "artifact"))
	}
}

// TestStrategy_GitDirSkipped pins walker-owned .git filtering.
func TestStrategy_GitDirSkipped(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			writeFile(t, filepath.Join(upper, ".git", "HEAD"), "ref: refs/heads/main")
			writeFile(t, filepath.Join(upper, "main.go"), "package main")
			changes := runStrategy(t, tc.strategy, lower, upper)
			mustHave(t, changes, "main.go", types.ChangeTypeAdded)
			for _, c := range changes {
				if c.FilePath == ".git" || strings.HasPrefix(c.FilePath, ".git"+string(filepath.Separator)) {
					t.Errorf("unexpected .git entry: %s", c.FilePath)
				}
			}
		})
	}
}

// TestStrategy_DeletionsDetected covers strategy-specific deletion
// semantics: overlay uses whiteouts, copy uses absence-in-upper.
func TestStrategy_OverlayDeletionViaFilenameWhiteout(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	writeFile(t, filepath.Join(lower, "removed.txt"), "old")
	writeWhiteout(t, upper, "removed.txt")
	changes := runStrategy(t, overlayStrategyImpl(), lower, upper)
	mustHave(t, changes, "removed.txt", types.ChangeTypeDeleted)
}

func TestStrategy_CopyDeletionViaAbsence(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	writeFile(t, filepath.Join(lower, "removed.txt"), "old")
	// upper does not contain removed.txt → copy strategy reports Deleted
	changes := runStrategy(t, copyStrategyImpl(), lower, upper)
	mustHave(t, changes, "removed.txt", types.ChangeTypeDeleted)
}

// TestStrategy_OpaqueWhiteoutSkipped pins that overlayfs opaque markers
// (.wh..opq) don't surface as changes.
func TestStrategy_OpaqueWhiteoutSkipped(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			writeFile(t, filepath.Join(upper, "tmp", ".wh..opq"), "opaque")
			changes := runStrategy(t, tc.strategy, lower, upper)
			for _, c := range changes {
				if c.FilePath == ".wh..opq" || c.FilePath == "tmp/.wh..opq" {
					t.Errorf("opaque marker should be ignored: %s", c.FilePath)
				}
			}
		})
	}
}

// TestStrategy_DeterministicOrdering pins I-CHANGE-1: results sort by
// FilePath for both strategies.
func TestStrategy_DeterministicOrdering(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			writeFile(t, filepath.Join(upper, "z.txt"), "z")
			writeFile(t, filepath.Join(upper, "a.txt"), "a")
			writeFile(t, filepath.Join(upper, "m.txt"), "m")
			changes := runStrategy(t, tc.strategy, lower, upper)
			var paths []string
			for _, c := range changes {
				paths = append(paths, c.FilePath)
			}
			gotSorted := append([]string(nil), paths...)
			sort.Strings(gotSorted)
			for i := range paths {
				if paths[i] != gotSorted[i] {
					t.Fatalf("unsorted output %v (want %v)", paths, gotSorted)
				}
			}
		})
	}
}

// TestStrategy_DirectoriesNotEmittedAsChanges pins that bare dir
// presence does not produce a FileChange row — only files do.
func TestStrategy_DirectoriesNotEmittedAsChanges(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			if err := os.MkdirAll(filepath.Join(upper, "newdir"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			changes := runStrategy(t, tc.strategy, lower, upper)
			for _, c := range changes {
				if c.FilePath == "newdir" {
					t.Errorf("directory must not surface as a change row: %+v", c)
				}
			}
		})
	}
}

// TestStrategy_UnicodeFilenames covers a basic UTF-8 path.
func TestStrategy_UnicodeFilenames(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			path := "🚀-rocket.md"
			writeFile(t, filepath.Join(upper, path), "lift-off")
			changes := runStrategy(t, tc.strategy, lower, upper)
			mustHave(t, changes, path, types.ChangeTypeAdded)
		})
	}
}

// TestStrategy_FileReplacedByDirectory covers an edge case: in the
// upper layer, the same path that was a regular file in lower is now
// a directory. Both strategies should produce some non-empty signal
// rather than crashing or yielding nothing.
func TestStrategy_FileReplacedByDirectory(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			writeFile(t, filepath.Join(lower, "thing"), "was a file")
			if err := os.MkdirAll(filepath.Join(upper, "thing", "child"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			writeFile(t, filepath.Join(upper, "thing", "child", "leaf.txt"), "leaf")
			changes := runStrategy(t, tc.strategy, lower, upper)
			// "thing" now being a directory should at minimum yield the leaf
			// file as Added. Strategies may or may not also report "thing"
			// as Deleted; what matters is the leaf reaches us.
			mustHave(t, changes, filepath.Join("thing", "child", "leaf.txt"), types.ChangeTypeAdded)
		})
	}
}

// TestStrategy_EmptyUpper covers the no-changes path. Walk must succeed
// with zero changes.
func TestStrategy_EmptyUpper(t *testing.T) {
	for _, tc := range strategies() {
		t.Run(tc.name, func(t *testing.T) {
			lower := t.TempDir()
			upper := t.TempDir()
			changes := runStrategy(t, tc.strategy, lower, upper)
			if len(changes) != 0 {
				t.Errorf("expected 0 changes for empty upper, got %d (%+v)", len(changes), changes)
			}
		})
	}
}

// TestWalk_RejectsEmptyUpper covers the precondition guard.
func TestWalk_RejectsEmptyUpper(t *testing.T) {
	_, err := changedetect.Walk(context.Background(),
		changedetect.WalkOpts{Lower: "/tmp", Upper: ""},
		overlayStrategyImpl(),
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error for empty Upper, got nil")
	}
}

// TestWalk_RejectsNilStrategy covers the precondition guard.
func TestWalk_RejectsNilStrategy(t *testing.T) {
	_, err := changedetect.Walk(context.Background(),
		changedetect.WalkOpts{Lower: "/tmp", Upper: "/tmp"},
		nil,
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error for nil strategy, got nil")
	}
}

// TestWalk_ContextCancellation pins ctx propagation.
func TestWalk_ContextCancellation(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	for i := 0; i < 50; i++ {
		writeFile(t, filepath.Join(upper, "f"+string(rune('0'+i%10))), "x")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := changedetect.Walk(ctx,
		changedetect.WalkOpts{Lower: lower, Upper: upper, SandboxID: uuid.New()},
		overlayStrategyImpl(),
		time.Now(),
	)
	if err == nil {
		t.Error("expected context.Canceled error, got nil")
	}
}

// TestStrategy_ShouldSkipOverlayInternals pins overlay's filtering of
// .overlay subtree and .wh..opq markers.
func TestStrategy_ShouldSkipOverlayInternals(t *testing.T) {
	strat := overlayStrategyImpl()
	for _, p := range []string{".overlay", ".overlay/work", ".wh..opq", "deep/.wh..opq"} {
		if !strat.ShouldSkip(p) {
			t.Errorf("ShouldSkip(%q) = false, want true", p)
		}
	}
	for _, p := range []string{"src/main.go", "README.md"} {
		if strat.ShouldSkip(p) {
			t.Errorf("ShouldSkip(%q) = true, want false", p)
		}
	}
}

// TestStrategy_OverlaySkipDir pins the SkipDir contract for overlay.
func TestStrategy_OverlaySkipDir(t *testing.T) {
	strat := overlayStrategyImpl()
	for _, p := range []string{".overlay", ".overlay/sub"} {
		if !strat.SkipDir(p) {
			t.Errorf("SkipDir(%q) = false, want true", p)
		}
	}
	if strat.SkipDir("regular_dir") {
		t.Error("SkipDir(regular_dir) = true, want false")
	}
}

// TestStrategy_CopyClassifyRequiresLower pins the precondition check.
func TestStrategy_CopyClassifyRequiresLower(t *testing.T) {
	upper := t.TempDir()
	writeFile(t, filepath.Join(upper, "a.txt"), "x")
	_, err := changedetect.Walk(context.Background(),
		changedetect.WalkOpts{Lower: "", Upper: upper, SandboxID: uuid.New()},
		copyStrategyImpl(),
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error when copy strategy receives empty Lower with files in upper")
	}
}

// TestStrategy_FilesAreDifferent_ByMode covers the mode-only diff path
// in filesAreDifferent.
func TestStrategy_FilesAreDifferent_ByMode(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	writeFile(t, filepath.Join(lower, "exec.sh"), "#!/bin/sh")
	writeFile(t, filepath.Join(upper, "exec.sh"), "#!/bin/sh")
	if err := os.Chmod(filepath.Join(upper, "exec.sh"), 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	changes := runStrategy(t, copyStrategyImpl(), lower, upper)
	mustHave(t, changes, "exec.sh", types.ChangeTypeModified)
}

// TestStrategy_FilesAreDifferent_LargeFileByMtime exercises the large-
// file branch (≥64KB) which falls back to mtime comparison.
func TestStrategy_FilesAreDifferent_LargeFileByMtime(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	big := make([]byte, 70*1024)
	for i := range big {
		big[i] = byte(i % 251)
	}
	lowerPath := filepath.Join(lower, "big.bin")
	upperPath := filepath.Join(upper, "big.bin")
	if err := os.WriteFile(lowerPath, big, 0o644); err != nil {
		t.Fatalf("write lower: %v", err)
	}
	// Same content but written later — different mtime triggers Modified.
	if err := os.WriteFile(upperPath, big, 0o644); err != nil {
		t.Fatalf("write upper: %v", err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(lowerPath, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	changes := runStrategy(t, copyStrategyImpl(), lower, upper)
	mustHave(t, changes, "big.bin", types.ChangeTypeModified)
}

// TestStrategy_FilesIdenticalNoChange pins that identical files don't
// surface as changes.
func TestStrategy_FilesIdenticalNoChange(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	writeFile(t, filepath.Join(lower, "same.txt"), "abc")
	writeFile(t, filepath.Join(upper, "same.txt"), "abc")
	now := time.Now()
	if err := os.Chtimes(filepath.Join(lower, "same.txt"), now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	if err := os.Chtimes(filepath.Join(upper, "same.txt"), now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	changes := runStrategy(t, copyStrategyImpl(), lower, upper)
	for _, c := range changes {
		if c.FilePath == "same.txt" {
			t.Errorf("identical file surfaced as change: %+v", c)
		}
	}
}

// TestStrategy_CopyDeletionsKeepTrackedDotFiles pins that the lower-walk
// deletion pass retains dot-prefixed tracked content while ignoring markers.
func TestStrategy_CopyDeletionsKeepTrackedDotFiles(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	// Dot file in lower that's not in upper is legitimate tracked content.
	writeFile(t, filepath.Join(lower, ".hidden"), "h")
	// Whiteout-named file in lower (defensive) — must NOT surface as Deleted.
	writeFile(t, filepath.Join(lower, ".wh.foo"), "x")
	// Plus a real visible file that's been removed.
	writeFile(t, filepath.Join(lower, "real.txt"), "r")
	changes := runStrategy(t, copyStrategyImpl(), lower, upper)
	mustHave(t, changes, ".hidden", types.ChangeTypeDeleted)
	mustNotHave(t, changes, ".wh.foo")
	mustHave(t, changes, "real.txt", types.ChangeTypeDeleted)
}

// TestStrategy_CopyDeletions_LowerMissing pins behaviour when Lower
// path doesn't exist at all.
func TestStrategy_CopyDeletions_LowerMissing(t *testing.T) {
	upper := t.TempDir()
	writeFile(t, filepath.Join(upper, "a.txt"), "x")
	changes, err := changedetect.Walk(context.Background(),
		changedetect.WalkOpts{Lower: filepath.Join(t.TempDir(), "no-such-dir"), Upper: upper, SandboxID: uuid.New()},
		copyStrategyImpl(),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	mustHave(t, changes, "a.txt", types.ChangeTypeAdded)
}

// TestStrategy_OverlayWhiteoutEmptyTarget pins that a degenerate
// `.wh.` (no target name) is dropped instead of crashing.
func TestStrategy_OverlayWhiteoutEmptyTarget(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	// Just the bare ".wh." file with no target name.
	if err := os.WriteFile(filepath.Join(upper, ".wh."), nil, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	changes := runStrategy(t, overlayStrategyImpl(), lower, upper)
	for _, c := range changes {
		if c.FilePath == "" || c.FilePath == "." {
			t.Errorf("degenerate whiteout produced spurious change: %+v", c)
		}
	}
}

// TestStrategy_OverlayLowerEmptyAlwaysAdded pins detectChangeType when
// Lower is empty: every upper file is Added.
func TestStrategy_OverlayLowerEmptyAlwaysAdded(t *testing.T) {
	upper := t.TempDir()
	writeFile(t, filepath.Join(upper, "a.txt"), "x")
	changes, err := changedetect.Walk(context.Background(),
		changedetect.WalkOpts{Lower: "", Upper: upper, SandboxID: uuid.New()},
		overlayStrategyImpl(),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	mustHave(t, changes, "a.txt", types.ChangeTypeAdded)
}

// TestStrategy_OverlayDirectoryWhiteoutWithDirParent pins the dir-rel
// path joining branch in classifyFilenameWhiteout (when the .wh.X marker
// lives inside a subdirectory).
func TestStrategy_OverlayDirectoryWhiteoutWithDirParent(t *testing.T) {
	lower := t.TempDir()
	upper := t.TempDir()
	writeFile(t, filepath.Join(lower, "sub", "removed.txt"), "old")
	writeWhiteout(t, filepath.Join(upper, "sub"), "removed.txt")
	changes := runStrategy(t, overlayStrategyImpl(), lower, upper)
	mustHave(t, changes, filepath.Join("sub", "removed.txt"), types.ChangeTypeDeleted)
}

func overlayStrategyImpl() *changedetect.OverlayStrategy {
	return &changedetect.OverlayStrategy{FileIDFn: stableFileID}
}

func copyStrategyImpl() *changedetect.CopyStrategy {
	return &changedetect.CopyStrategy{FileIDFn: stableFileID}
}

type namedStrategy struct {
	name     string
	strategy changedetect.Strategy
}

// strategies enumerates every concrete Strategy. Tests that apply to
// both range over this; tests that only apply to one call the
// constructor directly (overlayStrategyImpl / copyStrategyImpl).
func strategies() []namedStrategy {
	return []namedStrategy{
		{"overlay", overlayStrategyImpl()},
		{"copy", copyStrategyImpl()},
	}
}

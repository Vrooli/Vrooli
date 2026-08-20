package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

const tmpRoot = "/tmp"

func at(day int, hour int) time.Time {
	return time.Date(2026, 7, day, hour, 0, 0, 0, time.UTC)
}

// now is the observation instant every test in this file evaluates against.
var now = at(31, 12)

func file(path string, size int64, mod time.Time) cleanup.FileInfo {
	return cleanup.FileInfo{Path: path, Size: size, ModTime: mod}
}

func dir(path string, mod time.Time) cleanup.FileInfo {
	return cleanup.FileInfo{Path: path, Size: 4096, ModTime: mod, IsDir: true}
}

func newTmpProvider(t *testing.T, files ...cleanup.FileInfo) (*FileProvider, *cleanupfakes.FileSystem) {
	t.Helper()
	indexed := make(map[string]cleanup.FileInfo, len(files))
	for _, f := range files {
		indexed[f.Path] = f
	}
	fsys := &cleanupfakes.FileSystem{Root: tmpRoot, Files: indexed, AllowRemove: true}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID:              "tmp",
		Name:            "Temporary files",
		Roots:           []string{tmpRoot},
		Description:     "Remove aged temporary entries below configured roots",
		TopLevelEntries: true,
	})
	return provider, fsys
}

func previewWithMinAge(t *testing.T, provider *FileProvider, minAge time.Duration) cleanup.Preview {
	t.Helper()
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: minAge, ApprovalMode: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	return preview
}

// TestTopLevelEntries_CollapsesSubtreeIntoOneItem asserts a staging directory is
// one cleanup unit sized by its whole subtree, not N separate file items.
//
// Per-file granularity was the original behaviour and it left the directory
// itself behind. On the incident host that meant 24,882 surviving directories
// after a "successful" cleanup.
func TestTopLevelEntries_CollapsesSubtreeIntoOneItem(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "vrooli-release-stage.ZYWslJ")
	provider, _ := newTmpProvider(t,
		dir(staging, at(20, 0)),
		file(filepath.Join(staging, "a.bin"), 1000, at(20, 0)),
		file(filepath.Join(staging, "nested", "b.bin"), 2000, at(20, 0)),
		dir(filepath.Join(staging, "nested"), at(20, 0)),
	)

	preview := previewWithMinAge(t, provider, 24*time.Hour)

	if len(preview.Items) != 1 {
		t.Fatalf("got %d items, want exactly 1 for one staging directory: %#v", len(preview.Items), preview.Items)
	}
	item := preview.Items[0]
	if item.Path != staging {
		t.Errorf("item path = %q, want the top-level entry %q", item.Path, staging)
	}
	// 1000 + 2000 payload + two directory inodes at 4096 each.
	if want := int64(1000 + 2000 + 4096 + 4096); item.Bytes != want {
		t.Errorf("item bytes = %d, want %d (whole subtree)", item.Bytes, want)
	}
}

// TestTopLevelEntries_AgeUsesNewestDescendant is the safety-critical test.
//
// A directory's own mtime changes only when entries are added to or removed
// from it directly — never when a file inside is modified. So a build staging
// directory created eleven days ago and written to a minute ago still reports
// an eleven-day-old mtime. Aging on that value would delete live work while
// every timestamp consulted said it was stale.
func TestTopLevelEntries_AgeUsesNewestDescendant(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "vrooli-build-active")
	provider, _ := newTmpProvider(t,
		// The directory and most of its contents look long abandoned...
		dir(staging, at(20, 0)),
		file(filepath.Join(staging, "old.bin"), 1000, at(20, 0)),
		// ...but a build wrote this one an hour ago.
		file(filepath.Join(staging, "in-progress.bin"), 10, at(31, 11)),
	)

	preview := previewWithMinAge(t, provider, 7*24*time.Hour)

	if len(preview.Items) != 0 {
		t.Fatalf("selected an actively-written directory for deletion: %#v", preview.Items)
	}
}

func TestTopLevelEntries_FreshOwnMtimeSkipsSubtreeWalk(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "vrooli-build-fresh-entry")
	fsys := &countingFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		staging:                           dir(staging, at(31, 11)),
		filepath.Join(staging, "old.bin"): file(filepath.Join(staging, "old.bin"), 1000, at(20, 0)),
	}}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "tmp", Name: "Temporary files", Roots: []string{tmpRoot}, TopLevelEntries: true,
	})

	preview := previewWithMinAge(t, provider, 24*time.Hour)
	if len(preview.Items) != 0 {
		t.Fatalf("fresh top-level entry was selected: %#v", preview.Items)
	}
	if fsys.walks != 0 {
		t.Fatalf("fresh top-level entry walked %d times; sound own-mtime prefilter should skip the walk", fsys.walks)
	}
}

func TestTopLevelEntries_OldOwnMtimeStillWalksForFreshDescendant(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "vrooli-build-old-entry-active")
	fsys := &countingFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		staging:                               dir(staging, at(20, 0)),
		filepath.Join(staging, "in-progress"): file(filepath.Join(staging, "in-progress"), 1000, at(31, 11)),
	}}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "tmp", Name: "Temporary files", Roots: []string{tmpRoot}, TopLevelEntries: true,
	})

	preview := previewWithMinAge(t, provider, 24*time.Hour)
	if len(preview.Items) != 0 {
		t.Fatalf("active descendant was selected: %#v", preview.Items)
	}
	if fsys.walks != 1 {
		t.Fatalf("old top-level entry walk count = %d, want 1", fsys.walks)
	}
}

func TestTopLevelEntries_CompleteCensusIgnoresRequestBudget(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "vrooli-build-complete-census")
	provider := NewTmpProvider(&cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		staging:                           dir(staging, at(20, 0)),
		filepath.Join(staging, "payload"): file(filepath.Join(staging, "payload"), 1000, at(20, 0)),
	}}, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "tmp", Name: "Temporary files", Roots: []string{tmpRoot}, TopLevelEntries: true,
		MeasureBudget: time.Nanosecond,
	})

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{CompleteCensus: true},
		Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview.Items) != 1 || len(preview.Warnings) != 0 {
		t.Fatalf("complete census preview = %#v, want one measured item and no budget warning", preview)
	}
}

// TestTopLevelEntries_FullyStaleDirectoryIsSelected is the counterpart: when
// nothing in the subtree is recent, the entry is genuinely reclaimable.
func TestTopLevelEntries_FullyStaleDirectoryIsSelected(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "vrooli-build-abandoned")
	provider, _ := newTmpProvider(t,
		dir(staging, at(20, 0)),
		file(filepath.Join(staging, "old.bin"), 1000, at(21, 0)),
		file(filepath.Join(staging, "older.bin"), 2000, at(19, 0)),
	)

	preview := previewWithMinAge(t, provider, 7*24*time.Hour)

	if len(preview.Items) != 1 || preview.Items[0].Path != staging {
		t.Fatalf("expected the abandoned directory to be selected, got %#v", preview.Items)
	}
}

// TestTopLevelEntries_SkipsActiveAndHiddenEntries asserts the dot-prefixed and
// socket-like entries that /tmp is full of are never candidates. These are
// live session infrastructure — .X11-unix, .ICE-unix — whose removal breaks a
// running desktop.
func TestTopLevelEntries_SkipsActiveAndHiddenEntries(t *testing.T) {
	t.Parallel()

	provider, _ := newTmpProvider(t,
		dir(filepath.Join(tmpRoot, ".X11-unix"), at(1, 0)),
		file(filepath.Join(tmpRoot, ".X11-unix", "X0"), 0, at(1, 0)),
		file(filepath.Join(tmpRoot, "agent.sock"), 0, at(1, 0)),
		file(filepath.Join(tmpRoot, "build.lock"), 0, at(1, 0)),
		dir(filepath.Join(tmpRoot, "claude-1000"), at(1, 0)),
		dir(filepath.Join(tmpRoot, "systemd-private-boot"), at(1, 0)),
	)
	// These exclusions are hard safety boundaries: the session directory and
	// systemd private directories must remain present even when old.

	preview := previewWithMinAge(t, provider, 24*time.Hour)

	if len(preview.Items) != 0 {
		t.Fatalf("selected active session entries: %#v", preview.Items)
	}
}

// TestTopLevelEntries_MultipleEntriesAreDeterministicallyOrdered guards the
// byte-cap path: when MaxBytes truncates the selection, the same entries must be
// chosen on every run rather than whichever the map happened to yield first.
func TestTopLevelEntries_MultipleEntriesAreDeterministicallyOrdered(t *testing.T) {
	t.Parallel()

	var infos []cleanup.FileInfo
	for _, name := range []string{"zeta", "alpha", "mid"} {
		entry := filepath.Join(tmpRoot, name)
		infos = append(infos, dir(entry, at(20, 0)), file(filepath.Join(entry, "payload"), 1000, at(20, 0)))
	}
	provider, _ := newTmpProvider(t, infos...)

	for i := 0; i < 8; i++ {
		preview := previewWithMinAge(t, provider, 24*time.Hour)
		if len(preview.Items) != 3 {
			t.Fatalf("run %d: got %d items, want 3", i, len(preview.Items))
		}
		want := []string{
			filepath.Join(tmpRoot, "alpha"),
			filepath.Join(tmpRoot, "mid"),
			filepath.Join(tmpRoot, "zeta"),
		}
		for j, item := range preview.Items {
			if item.Path != want[j] {
				t.Fatalf("run %d: item %d = %q, want %q", i, j, item.Path, want[j])
			}
		}
	}
}

// TestApply_ContinuesPastUnremovableItem asserts one failure does not abandon
// the rest of the run, and that what did succeed is still counted.
//
// Under disk pressure the whole point is to reclaim what can be reclaimed.
// Hitting an entry owned by another user, or one that vanished mid-run, is
// routine in a shared /tmp.
func TestApply_ContinuesPastUnremovableItem(t *testing.T) {
	t.Parallel()

	good := filepath.Join(tmpRoot, "removable")
	provider, _ := newTmpProvider(t,
		dir(good, at(20, 0)),
		file(filepath.Join(good, "payload"), 1000, at(20, 0)),
	)

	preview := previewWithMinAge(t, provider, 24*time.Hour)
	if len(preview.Items) != 1 {
		t.Fatalf("expected one candidate, got %#v", preview.Items)
	}

	// Prepend an item the fake filesystem refuses to remove. The root itself
	// qualifies: it is inside the configured cleanup root (so it reaches
	// RemoveAll rather than being filtered out as out-of-bounds) but is not
	// strictly beneath the fake's root, which is how the fake models a refusal.
	blocked := cleanup.PreviewItem{
		ID:     "tmp:blocked",
		Path:   tmpRoot,
		Bytes:  50,
		Action: "tmp-remove",
	}

	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: provider.Metadata().Version,
		ApprovalMode:    cleanup.ApprovalModeOperator,
		IdempotencyKey:  "apply-resilience",
		Preview:         cleanup.Preview{Items: append([]cleanup.PreviewItem{blocked}, preview.Items...)},
	})
	if err != nil {
		t.Fatalf("Apply() aborted on a single unremovable item: %v", err)
	}
	if len(result.SkippedItems) != 1 || result.SkippedItems[0] != blocked.ID {
		t.Errorf("skipped items = %#v, want just the blocked one", result.SkippedItems)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want one describing the blocked item", result.Warnings)
	}
	// The good item's bytes must still be counted; an aborted run would have
	// reported zero.
	if result.ReclaimedBytes != preview.Items[0].Bytes {
		t.Errorf("reclaimed = %d, want %d from the item that did succeed", result.ReclaimedBytes, preview.Items[0].Bytes)
	}
	if !result.Applied {
		t.Error("Applied = false despite reclaiming bytes")
	}
	// Warnings are persisted to the audit log, so the path must be stripped.
	if warning := result.Warnings[0]; strings.Contains(warning, tmpRoot) || !strings.Contains(warning, "[path]") {
		t.Errorf("warning %q should have its path redacted", warning)
	}
}

// budgetClock advances a fixed step on every reading, so a measurement budget
// can be exhausted deterministically without sleeping.
type budgetClock struct {
	current time.Time
	step    time.Duration
}

func (c *budgetClock) Now() time.Time {
	now := c.current
	c.current = c.current.Add(c.step)
	return now
}

// TestMeasureBudget_DropsUnmeasuredEntriesAndWarns is the correctness bar for
// the time budget.
//
// Measuring this host's trash meant stat-ing 5,064,705 files, which exceeded
// the API write timeout and failed the entire plan with a 500. The budget makes
// the pass bounded. What it must never do is guess: an entry that ran out of
// budget is excluded from the plan entirely, and the shortfall is reported.
func TestMeasureBudget_DropsUnmeasuredEntriesAndWarns(t *testing.T) {
	t.Parallel()

	var infos []cleanup.FileInfo
	for _, name := range []string{"a-first", "b-second", "c-third", "d-fourth"} {
		entry := filepath.Join(tmpRoot, name)
		infos = append(infos, dir(entry, at(20, 0)), file(filepath.Join(entry, "payload"), 1000, at(20, 0)))
	}
	indexed := make(map[string]cleanup.FileInfo, len(infos))
	for _, info := range infos {
		indexed[info.Path] = info
	}

	// Each clock reading advances 4s against a 10s budget, so only the first
	// few entries fit.
	clk := &budgetClock{current: now, step: 4 * time.Second}
	provider := NewTmpProvider(
		&cleanupfakes.FileSystem{Root: tmpRoot, Files: indexed, AllowRemove: true},
		clk,
		FileProviderConfig{
			ID:              "tmp",
			Name:            "Temporary files",
			Roots:           []string{tmpRoot},
			TopLevelEntries: true,
			MeasureBudget:   10 * time.Second,
		},
	)

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{Now: now},
		Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if len(preview.Items) == 0 {
		t.Fatal("budget exhaustion dropped everything; the measured entries should still be reported")
	}
	if len(preview.Items) == 4 {
		t.Fatal("no entries were dropped; the budget was not enforced")
	}
	// The shortfall must be visible. A silently partial estimate is
	// indistinguishable from a clean host.
	var warned bool
	for _, warning := range preview.Warnings {
		if strings.Contains(warning, "were not measured") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("no warning reported the unmeasured entries: %#v", preview.Warnings)
	}
	// Whatever survived must be a genuine top-level entry, never a partial
	// subtree masquerading as one.
	for _, item := range preview.Items {
		if filepath.Dir(item.Path) != tmpRoot {
			t.Errorf("item %q is not a top-level entry", item.Path)
		}
	}
}

// TestMeasureBudget_DefaultAppliesWhenUnset guards against a zero budget being
// interpreted as "no time at all", which would silently disable cleanup
// everywhere.
func TestMeasureBudget_DefaultAppliesWhenUnset(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "stale")
	provider, _ := newTmpProvider(t,
		dir(staging, at(20, 0)),
		file(filepath.Join(staging, "payload"), 1000, at(20, 0)),
	)

	preview := previewWithMinAge(t, provider, 24*time.Hour)
	if len(preview.Items) != 1 {
		t.Fatalf("default budget produced %d items, want 1", len(preview.Items))
	}
	for _, warning := range preview.Warnings {
		if strings.Contains(warning, "were not measured") {
			t.Errorf("default budget reported exhaustion on a tiny tree: %q", warning)
		}
	}
}

// TestMeasureBudget_DropsEntryThatOverrunsMidTraversal asserts the budget bounds
// a SINGLE oversized entry, not just the count of entries attempted.
//
// Checking the deadline only before starting each entry bounds nothing on its
// own: one directory holding millions of files overshoots by any amount. That
// is not hypothetical — the trash on this project's development host held
// 5,064,705 files, and an unbounded traversal of it exhausted the API write
// timeout and failed the whole plan.
//
// The entry must be dropped rather than half-measured, because a subtree
// measured halfway reports the newest mtime among only the files reached, which
// can look stale while a file the walk never saw is minutes old.
func TestMeasureBudget_DropsEntryThatOverrunsMidTraversal(t *testing.T) {
	t.Parallel()

	huge := filepath.Join(tmpRoot, "a-huge")
	small := filepath.Join(tmpRoot, "b-small")

	indexed := map[string]cleanup.FileInfo{
		huge:  dir(huge, at(20, 0)),
		small: dir(small, at(20, 0)),
	}
	// Enough files under the first entry to cross the clock-sampling stride.
	for i := 0; i < 5000; i++ {
		path := filepath.Join(huge, fmt.Sprintf("f%05d.bin", i))
		indexed[path] = file(path, 10, at(20, 0))
	}
	indexed[filepath.Join(small, "payload")] = file(filepath.Join(small, "payload"), 1000, at(20, 0))

	// Tuned so the entry PASSES the pre-entry check and only runs out of
	// budget partway through its traversal — which is the path under test.
	// Readings: t+0 sets the t+10 deadline, t+4 admits the entry, then the
	// sampled checks inside the walk land at t+8 (fine) and t+12 (expired).
	clk := &budgetClock{current: now, step: 4 * time.Second}
	provider := NewTmpProvider(
		&cleanupfakes.FileSystem{Root: tmpRoot, Files: indexed, AllowRemove: true},
		clk,
		FileProviderConfig{
			ID: "tmp", Name: "Temporary files",
			Roots:           []string{tmpRoot},
			TopLevelEntries: true,
			MeasureBudget:   10 * time.Second,
		},
	)

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{Now: now},
		Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v; budget exhaustion must not fail the plan", err)
	}

	for _, item := range preview.Items {
		if item.Path == huge {
			t.Errorf("an entry whose traversal ran out of budget was reported as a candidate: %#v", item)
		}
	}
	var warned bool
	for _, warning := range preview.Warnings {
		if strings.Contains(warning, "were not measured") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("the dropped entry was not reported: %#v", preview.Warnings)
	}
}

// countingFS records how many times each root was walked.
type countingFS struct {
	*cleanupfakes.FileSystem
	readDirs int
	walks    int
}

func (c *countingFS) ReadDir(ctx context.Context, path string) ([]cleanup.FileInfo, error) {
	c.readDirs++
	return c.FileSystem.ReadDir(ctx, path)
}

func (c *countingFS) Walk(ctx context.Context, root string, visit func(cleanup.FileInfo) error) error {
	c.walks++
	return c.FileSystem.Walk(ctx, root, visit)
}

// TestPreviewMemo_PlanMeasuresEachTreeOnce asserts a plan's Estimate-then-
// Preview pair costs one measurement rather than two.
//
// orchestrator.Plan calls Estimate and then Preview on every provider, and a
// file provider's Estimate IS a full preview. Without the memo every plan walks
// every tree twice: measured on this project's development host as 22.5s per
// plan against a 30s API write timeout, which the cache providers would push
// straight over the edge.
func TestPreviewMemo_PlanMeasuresEachTreeOnce(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "staging")
	indexed := map[string]cleanup.FileInfo{
		staging:                           dir(staging, at(20, 0)),
		filepath.Join(staging, "payload"): file(filepath.Join(staging, "payload"), 1000, at(20, 0)),
	}
	fsys := &countingFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: indexed, AllowRemove: true}}

	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "tmp", Name: "Temporary files",
		Roots:           []string{tmpRoot},
		TopLevelEntries: true,
	})

	scope := cleanup.ObservationScope{Now: now}
	policy := cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator}

	// Exactly what orchestrator.Plan does.
	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: scope, Policy: policy})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: scope, Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if fsys.readDirs != 1 {
		t.Errorf("root was listed %d times, want 1: the plan measured the tree twice", fsys.readDirs)
	}
	// The results must still be consistent with each other.
	if estimate.ItemCount != len(preview.Items) {
		t.Errorf("estimate reported %d items but preview returned %d", estimate.ItemCount, len(preview.Items))
	}
	if estimate.EstimatedBytes != sumPreviewBytes(preview.Items) {
		t.Errorf("estimate %d bytes != preview total %d", estimate.EstimatedBytes, sumPreviewBytes(preview.Items))
	}
}

// TestPreviewMemo_HandsOutIndependentCopies asserts one caller mutating its
// preview cannot corrupt what the next caller sees. The slice in question is a
// list of files about to be deleted, so aliasing it would be a genuinely
// dangerous bug rather than a cosmetic one.
func TestPreviewMemo_HandsOutIndependentCopies(t *testing.T) {
	t.Parallel()

	staging := filepath.Join(tmpRoot, "staging")
	provider, _ := newTmpProvider(t,
		dir(staging, at(20, 0)),
		file(filepath.Join(staging, "payload"), 1000, at(20, 0)),
	)

	first := previewWithMinAge(t, provider, 24*time.Hour)
	if len(first.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(first.Items))
	}
	first.Items[0].Path = "/mutated"
	first.Items = append(first.Items, cleanup.PreviewItem{ID: "injected", Path: "/etc"})

	second := previewWithMinAge(t, provider, 24*time.Hour)
	if len(second.Items) != 1 {
		t.Fatalf("second preview has %d items; the first caller's append leaked", len(second.Items))
	}
	if second.Items[0].Path != staging {
		t.Errorf("second preview path = %q, want %q; the first caller's mutation leaked", second.Items[0].Path, staging)
	}
}

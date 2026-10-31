package providers

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

func TestTemporaryApplyRejectsLegacyPreviewVersion(t *testing.T) {
	provider, fsys := newTmpProvider(t, dir(filepath.Join(tmpRoot, "old"), at(20, 0)))
	_, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "legacy", Preview: cleanup.Preview{Items: []cleanup.PreviewItem{{Path: filepath.Join(tmpRoot, "old")}}}})
	if err == nil || len(fsys.Removed) != 0 {
		t.Fatalf("legacy preview reached cleanup: err=%v removed=%v", err, fsys.Removed)
	}
}

func TestTemporaryRecoveryResumesLargeEntryWithoutOfferingPartialTree(t *testing.T) {
	staging := filepath.Join(tmpRoot, "old-large-fixture")
	fsys := &resumableTestFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		staging: dir(staging, at(20, 0)),
	}}, items: []cleanup.FileInfo{dir(staging, at(20, 0)), file(staging+"/old", 3<<30, at(20, 0)), file(staging+"/new", 100, at(31, 11))}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "tmp", Roots: []string{tmpRoot}, TopLevelEntries: true})
	for i := 0; i < 4; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		fsys.cancel = cancel
		preview, err := provider.Preview(ctx, cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: now, Recovery: true}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 7 * 24 * time.Hour}})
		cancel()
		if err != nil {
			t.Fatalf("bounded observation %d discarded progress: %v", i, err)
		}
		if len(preview.Items) != 0 {
			t.Fatalf("offered a partial or recently modified tree: %#v", preview.Items)
		}
		if i == 3 && preview.BlockedReason != "" {
			t.Fatalf("complete active tree remains pending: %#v", preview)
		}
	}
	if fsys.opens != 1 {
		t.Fatalf("reopened large tree %d times; want one resumable walk", fsys.opens)
	}
}

func TestTemporaryApplyRechecksDescendantsAndRootProtection(t *testing.T) {
	staging := filepath.Join(tmpRoot, "old-apply-fixture")
	provider, fsys := newTmpProvider(t, dir(staging, at(20, 0)), file(staging+"/payload", 100, at(20, 0)))
	preview := previewWithMinAge(t, provider, 7*24*time.Hour)
	fsys.Files[staging+"/fresh"] = file(staging+"/fresh", 1, now)
	req := cleanup.ApplyRequest{ProviderVersion: provider.Metadata().Version, IdempotencyKey: "fresh-after-preview", Preview: preview}
	result, err := provider.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied || len(fsys.Removed) != 0 || len(result.SkippedItems) != 1 {
		t.Fatalf("removed newly active temporary tree: %#v %v", result, fsys.Removed)
	}
	req.Preview = cleanup.Preview{Items: []cleanup.PreviewItem{{ID: "root", Path: tmpRoot, Bytes: 100}}}
	result, err = provider.Apply(context.Background(), req)
	if err != nil || result.Applied || len(fsys.Removed) != 0 {
		t.Fatalf("configured temporary root was removable: %#v %v", result, err)
	}
}

func TestTemporaryRecoveryAllowsOneFullyMeasuredOversizedEntry(t *testing.T) {
	staging := filepath.Join(tmpRoot, "old-large-fixture")
	fsys := &resumableTestFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{staging: dir(staging, at(20, 0))}}, items: []cleanup.FileInfo{dir(staging, at(20, 0)), file(staging+"/payload", 3<<30, at(20, 0))}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "tmp", Roots: []string{tmpRoot}, TopLevelEntries: true})
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: now, Recovery: true}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 7 * 24 * time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || !preview.AllowSingleOvershoot {
		t.Fatalf("large expired temporary entry is unreachable: %#v", preview)
	}
}

func TestTemporaryRecoveryAllowsExpiredHiddenPayloadButProtectsLeaseMarkers(t *testing.T) {
	for _, lease := range []bool{false, true} {
		t.Run(fmt.Sprintf("lease=%t", lease), func(t *testing.T) {
			staging := filepath.Join(tmpRoot, "expired-home")
			entries := []cleanup.FileInfo{dir(staging, at(20, 0)), dir(staging+"/.cache", at(20, 0)), file(staging+"/.cache/payload", 3<<30, at(20, 0))}
			if lease {
				entries = append(entries, file(staging+"/.lease", 0, at(20, 0)))
			}
			fsys := &resumableTestFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{staging: entries[0]}}, items: entries}
			provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "tmp", Roots: []string{tmpRoot}, TopLevelEntries: true})
			preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: now, Recovery: true}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 72 * time.Hour}})
			if err != nil {
				t.Fatal(err)
			}
			if (!lease && len(preview.Items) != 1) || (lease && len(preview.Items) != 0) {
				t.Fatalf("hidden/lease classification: %#v", preview)
			}
		})
	}
}

type resumableTestFS struct {
	*cleanupfakes.FileSystem
	items  []cleanup.FileInfo
	cancel context.CancelFunc
	opens  int
}

func (f *resumableTestFS) OpenWalk(context.Context, string) (cleanup.FileWalk, error) {
	f.opens++
	return &resumableTestWalk{fsys: f}, nil
}

type resumableTestWalk struct {
	fsys  *resumableTestFS
	index int
}

func (w *resumableTestWalk) Next(ctx context.Context) (cleanup.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return cleanup.FileInfo{}, err
	}
	if w.index == len(w.fsys.items) {
		return cleanup.FileInfo{}, io.EOF
	}
	item := w.fsys.items[w.index]
	w.index++
	if w.fsys.cancel != nil {
		w.fsys.cancel()
	}
	return item, nil
}
func (*resumableTestWalk) Close() error { return nil }

const tmpRoot = "/tmp"

func TestIntersectRootsRestrictsToScopedChild(t *testing.T) {
	got := intersectRoots([]string{"/tmp/go-work-dirs"}, []string{"/tmp/go-work-dirs/chaos"})
	if len(got) != 1 || got[0] != "/tmp/go-work-dirs/chaos" {
		t.Fatalf("intersectRoots = %v, want scoped child", got)
	}
}

func TestIntersectRootsRootPartitionIncludesConfiguredRoots(t *testing.T) {
	got := intersectRoots([]string{"/tmp", "/home/vrooli/cache"}, []string{"/"})
	if len(got) != 2 || got[0] != "/tmp" || got[1] != "/home/vrooli/cache" {
		t.Fatalf("intersectRoots(root partition) = %v, want configured roots", got)
	}
}

func TestFileProviderHonorsConfiguredRetentionByteCeiling(t *testing.T) {
	root := filepath.Join(tmpRoot, "bounded-cache")
	fsys := &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		root:                     dir(root, at(1, 0)),
		filepath.Join(root, "a"): file(filepath.Join(root, "a"), 80, at(1, 0)),
		filepath.Join(root, "b"): file(filepath.Join(root, "b"), 80, at(1, 0)),
	}, AllowRemove: true}
	provider := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "spec-bounded", Name: "Bounded cache", Roots: []string{root}, Tier: cleanup.SafetyTierRegenerable,
		RetentionMaxBytes: 100,
	})
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope: cleanup.ObservationScope{Now: now}, Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeNone},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := sumPreviewBytes(preview.Items); got > 100 {
		t.Fatalf("preview bytes = %d, exceeded configured ceiling", got)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != filepath.Join(root, "a") {
		t.Fatalf("cap preview = %#v, want oldest single overshoot candidate", preview.Items)
	}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: provider.Metadata().Version, IdempotencyKey: "bounded-cache",
		ApprovalMode: cleanup.ApprovalModeNone, Preview: preview,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || result.ReclaimedBytes != 80 || len(fsys.Removed) != 1 || fsys.Removed[0] != filepath.Join(root, "a") {
		t.Fatalf("non-Go cache reclamation changed: result=%#v removed=%v", result, fsys.Removed)
	}
}

func TestFileProviderHonorsAgeCeilingUnlessFreshReclaimIsAuthorized(t *testing.T) {
	root := filepath.Join(tmpRoot, "age-bounded-cache")
	path := filepath.Join(root, "fresh")
	fsys := &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		root: dir(root, now.Add(-6*time.Hour)), path: file(path, 80, now.Add(-6*time.Hour)),
	}, AllowRemove: true}
	provider := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
		ID: "spec-age-bounded", Name: "Age bounded cache", Roots: []string{root}, Tier: cleanup.SafetyTierRegenerable,
		RetentionMaxAge: 24 * time.Hour,
	})
	ordinary, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope: cleanup.ObservationScope{Now: now}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: time.Hour, ApprovalMode: cleanup.ApprovalModeNone},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ordinary.Items) != 0 {
		t.Fatalf("ordinary preview selected a younger-than-contract item: %#v", ordinary.Items)
	}
	fresh, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope: cleanup.ObservationScope{Now: now}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 0, ApprovalMode: cleanup.ApprovalModeNone, AllowFreshReclaim: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Items) != 1 || fresh.Items[0].Path != path {
		t.Fatalf("fresh-reclaim preview = %#v, want %q", fresh.Items, path)
	}
}

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

// treeWalkFS gives the in-memory fake a resumable walk over each entry's own
// subtree, so temporary recovery can be exercised across several entries.
type treeWalkFS struct{ *cleanupfakes.FileSystem }

func (f *treeWalkFS) OpenWalk(_ context.Context, path string) (cleanup.FileWalk, error) {
	var items []cleanup.FileInfo
	for candidate, info := range f.Files {
		if candidate == path || strings.HasPrefix(candidate, path+"/") {
			items = append(items, info)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return &sliceWalk{items: items}, nil
}

type sliceWalk struct {
	items []cleanup.FileInfo
	index int
}

func (w *sliceWalk) Next(ctx context.Context) (cleanup.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return cleanup.FileInfo{}, err
	}
	if w.index == len(w.items) {
		return cleanup.FileInfo{}, io.EOF
	}
	w.index++
	return w.items[w.index-1], nil
}
func (*sliceWalk) Close() error { return nil }

func recoveryPreview(t *testing.T, provider *FileProvider, minAge time.Duration) cleanup.Preview {
	t.Helper()
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: now, Recovery: true}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: minAge}})
	if err != nil {
		t.Fatalf("recovery preview: %v", err)
	}
	return preview
}

// A temporary entry whose removal failed must not be re-offered first on the
// next pass. On the incident host one read-only fixture was retried 130 times
// while 57 GB of removable entries behind it were never reached.
func TestTemporaryRecoveryFailedEntryDoesNotBlockLaterCandidates(t *testing.T) {
	bad, good := filepath.Join(tmpRoot, "bad-fixture"), filepath.Join(tmpRoot, "good-fixture")
	clock := &budgetClock{current: now}
	fsys := &treeWalkFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, AllowRemove: true, Files: map[string]cleanup.FileInfo{
		bad: {Path: bad, IsDir: true, Size: 4096, ModTime: at(1, 0), Identity: "1:10"}, bad + "/payload": file(bad+"/payload", 100, at(1, 0)),
		good: {Path: good, IsDir: true, Size: 4096, ModTime: at(2, 0), Identity: "1:20"}, good + "/payload": file(good+"/payload", 100, at(2, 0)),
	}}}
	provider := NewTmpProvider(fsys, clock, FileProviderConfig{ID: "tmp", Roots: []string{tmpRoot}, TopLevelEntries: true})
	first := recoveryPreview(t, provider, 7*24*time.Hour)
	var badItem cleanup.PreviewItem
	for _, item := range first.Items {
		if item.Path == bad {
			badItem = item
		}
	}
	if badItem.Path == "" {
		t.Fatalf("first preview = %v, want the stale fixture offered", previewPaths(first))
	}
	fsys.RemoveErrors = []error{fmt.Errorf("unlinkat %s: permission denied", bad)}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: provider.Metadata().Version, IdempotencyKey: "fails", Preview: cleanup.Preview{MinAge: 7 * 24 * time.Hour, Items: []cleanup.PreviewItem{badItem}}})
	if err != nil || result.ReclaimedBytes != 0 || len(result.SkippedItems) != 1 {
		t.Fatalf("failing apply = %#v, %v", result, err)
	}

	// Both the retained ready set and a fresh walk must skip the failed entry.
	for _, fresh := range []bool{false, true} {
		if fresh {
			provider.CloseRecoveryWalks()
		}
		next := recoveryPreview(t, provider, 7*24*time.Hour)
		if got := previewPaths(next); len(got) != 1 || got[0] != good {
			t.Fatalf("fresh=%t preview = %v, want only %s past the failed entry", fresh, got, good)
		}
	}

	// A different entry under the same name is a new candidate.
	replaced := fsys.Files[bad]
	replaced.Identity = "1:99"
	fsys.Files[bad] = replaced
	provider.CloseRecoveryWalks()
	if got := previewPaths(recoveryPreview(t, provider, 7*24*time.Hour)); len(got) != 2 {
		t.Fatalf("replaced entry preview = %v, want the new entry measured again", got)
	}

	// The original entry is retried once its back-off expires.
	replaced.Identity = "1:10"
	fsys.Files[bad] = replaced
	fsys.RemoveErrors = []error{fmt.Errorf("unlinkat %s: permission denied", bad)}
	badItem.Identity = "1:10"
	if _, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: provider.Metadata().Version, IdempotencyKey: "fails-again", Preview: cleanup.Preview{MinAge: 7 * 24 * time.Hour, Items: []cleanup.PreviewItem{badItem}}}); err != nil {
		t.Fatal(err)
	}
	provider.CloseRecoveryWalks()
	if got := previewPaths(recoveryPreview(t, provider, 7*24*time.Hour)); len(got) != 1 {
		t.Fatalf("preview inside back-off = %v, want failed entry withheld", got)
	}
	clock.current = clock.current.Add(failedCandidateBackoff + time.Minute)
	provider.CloseRecoveryWalks()
	if got := previewPaths(recoveryPreview(t, provider, 7*24*time.Hour)); len(got) != 2 {
		t.Fatalf("preview after back-off = %v, want failed entry retried", got)
	}
}

// One recovery observation must make byte-meaningful progress: it collects
// every complete unit its budget reaches and puts the largest first, so a
// large stale tree is not queued behind thousands of tiny entries.
func TestTemporaryRecoveryAccumulatesUnitsLargestFirst(t *testing.T) {
	files := map[string]cleanup.FileInfo{}
	for i := 0; i < 5; i++ {
		small := filepath.Join(tmpRoot, fmt.Sprintf("small-%d", i))
		files[small] = dir(small, at(1, i))
		files[small+"/f"] = file(small+"/f", 10, at(1, i))
	}
	large := filepath.Join(tmpRoot, "large-fixture")
	files[large] = dir(large, at(10, 0))
	files[large+"/payload"] = file(large+"/payload", 20<<30, at(10, 0))
	fsys := &treeWalkFS{FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: files}}
	provider := NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "tmp", Roots: []string{tmpRoot}, TopLevelEntries: true})
	preview := recoveryPreview(t, provider, 7*24*time.Hour)
	if len(preview.Items) != 6 || preview.Items[0].Path != large || !preview.AllowSingleOvershoot {
		t.Fatalf("preview = %v, want all six complete units with the large tree first", previewPaths(preview))
	}
}

// A regenerable root proven over its declared ceiling may shed one whole
// indivisible unit larger than the controller batch; a root under its ceiling
// may not. The stalest unit (by newest descendant, not by name or entry mtime)
// is offered first.
func TestRegenerableRecoveryOvershootRequiresProvenCeilingBreach(t *testing.T) {
	root := filepath.Join(tmpRoot, "store")
	for _, tc := range []struct {
		name      string
		ceiling   int64
		overshoot bool
	}{{"over ceiling", 100, true}, {"under ceiling", 200, false}} {
		t.Run(tc.name, func(t *testing.T) {
			fsys := &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
				root: dir(root, at(1, 0)),
				// "a-v10" sorts first and has the older entry mtime, but its
				// contents were written most recently.
				root + "/a-v10": {Path: root + "/a-v10", IsDir: true, ModTime: at(1, 0)}, root + "/a-v10/f": file(root+"/a-v10/f", 60, at(30, 0)),
				root + "/b-v3": {Path: root + "/b-v3", IsDir: true, ModTime: at(20, 0)}, root + "/b-v3/f": file(root+"/b-v3/f", 80, at(2, 0)),
			}}
			provider := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "store", Roots: []string{root}, Tier: cleanup.SafetyTierRegenerable, RetentionMaxBytes: tc.ceiling})
			preview := recoveryPreview(t, provider, 0)
			if preview.AllowSingleOvershoot != tc.overshoot {
				t.Fatalf("overshoot = %t, want %t", preview.AllowSingleOvershoot, tc.overshoot)
			}
			if len(preview.Items) == 0 || preview.Items[0].Path != root+"/b-v3" {
				t.Fatalf("preview = %v, want least recently written unit first", previewPaths(preview))
			}
		})
	}
}

// A contract glob such as "cache/lock" is relative to the declared root, and a
// unit is removed whole, so a protected descendant protects its unit.
func TestTopLevelEntries_RootRelativeProtectedDescendantProtectsUnit(t *testing.T) {
	root := filepath.Join(tmpRoot, "mod")
	fsys := &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		root:                         dir(root, at(1, 0)),
		root + "/cache":              dir(root+"/cache", at(1, 0)),
		root + "/cache/lock":         file(root+"/cache/lock", 0, at(1, 0)),
		root + "/cache/download/zip": file(root+"/cache/download/zip", 50, at(1, 0)),
		root + "/example.com":        dir(root+"/example.com", at(1, 0)),
		root + "/example.com/m@v1/a": file(root+"/example.com/m@v1/a", 50, at(1, 0)),
	}}
	provider := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "mod", Roots: []string{root}, Tier: cleanup.SafetyTierRegenerable, RetentionMaxBytes: 1 << 20, ProtectedGlobs: []string{"cache/lock"}})
	if got := previewPaths(recoveryPreview(t, provider, 0)); len(got) != 1 || got[0] != root+"/example.com" {
		t.Fatalf("preview = %v, want the unit holding cache/lock withheld", got)
	}
}

// slowEntryFS blocks the walk of one entry until the caller's context ends,
// standing in for a subtree too large to measure within the budget.
type slowEntryFS struct {
	*cleanupfakes.FileSystem
	slow string
}

func (f *slowEntryFS) Walk(ctx context.Context, root string, visit func(cleanup.FileInfo) error) error {
	if root == f.slow {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.FileSystem.Walk(ctx, root, visit)
}

// The provider's own measurement budget expiring mid-subtree is a bounded
// observation: units measured before it are returned with a warning instead
// of the whole estimate failing with "context deadline exceeded".
func TestMeasureBudget_ContextExpiryKeepsMeasuredUnits(t *testing.T) {
	root := filepath.Join(tmpRoot, "cache-root")
	fast, slow := root+"/fast", root+"/slow"
	fsys := &slowEntryFS{slow: slow, FileSystem: &cleanupfakes.FileSystem{Root: tmpRoot, Files: map[string]cleanup.FileInfo{
		root: dir(root, at(1, 0)), fast: dir(fast, at(1, 0)), fast + "/f": file(fast+"/f", 50, at(1, 0)),
		slow: dir(slow, at(2, 0)), slow + "/f": file(slow+"/f", 50, at(2, 0)),
	}}}
	provider := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{ID: "cache", Roots: []string{root}, Tier: cleanup.SafetyTierRegenerable, RetentionMaxBytes: 1 << 20, MeasureBudget: 50 * time.Millisecond})
	preview := recoveryPreview(t, provider, 0)
	if got := previewPaths(preview); len(got) != 1 || got[0] != fast {
		t.Fatalf("preview = %v, want the unit measured before the budget expired", got)
	}
	if len(preview.Warnings) == 0 || !strings.Contains(strings.Join(preview.Warnings, ";"), "measurement budget") {
		t.Fatalf("warnings = %v, want the exhausted budget surfaced", preview.Warnings)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Preview(cancelled, cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: now, Recovery: true}, Policy: cleanup.ProviderPolicy{Enabled: true}}); err == nil {
		t.Fatal("caller cancellation was reported as a successful partial measurement")
	}
}

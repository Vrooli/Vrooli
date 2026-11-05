package cleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/retention"
)

type fakeOutputIndex struct {
	outputs []internaljobs.Job
	active  []internaljobs.Job
}

func (f fakeOutputIndex) ListManagedOutputs(_ context.Context, limit int) ([]internaljobs.Job, error) {
	if limit < len(f.outputs) {
		return f.outputs[:limit], nil
	}
	return f.outputs, nil
}

func (f fakeOutputIndex) ListActive(context.Context) ([]internaljobs.Job, error) {
	return f.active, nil
}

var retentionNow = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func succeeded(id, ref string, finished time.Time, meta map[string]string) internaljobs.Job {
	f := finished
	return internaljobs.Job{ID: id, ResultRef: ref, State: internaljobs.StateSucceeded, Meta: meta, CreatedAt: finished, FinishedAt: &f}
}

func newOutputStore(t *testing.T, keys ...string) (*storage.Store, string) {
	t.Helper()
	root := t.TempDir()
	store := storage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	for _, key := range keys {
		if err := store.Put(context.Background(), key, bytes.NewReader(bytes.Repeat([]byte("x"), 100)), "image/png"); err != nil {
			t.Fatal(err)
		}
	}
	return store, root
}

func exists(t *testing.T, root, key string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(key)))
	return err == nil
}

func TestOutputRetentionDeletesExpiredAndProtectsKeptOutputs(t *testing.T) {
	store, root := newOutputStore(t, "out/expired.png", "out/recent.png", "out/pinned.png",
		"out/in-use.png", "out/shared.zip", "out/writing.png")
	old := retentionNow.Add(-40 * 24 * time.Hour)
	index := fakeOutputIndex{
		outputs: []internaljobs.Job{
			succeeded("expired", "out/expired.png", old, nil),
			succeeded("pinned", "out/pinned.png", old, map[string]string{"retention_pinned": "true"}),
			succeeded("in-use", "out/in-use.png", old, nil),
			// A content-addressed key reused by a recent job takes the newest age.
			succeeded("shared-old", "out/shared.zip", old, nil),
			succeeded("shared-new", "out/shared.zip", retentionNow.Add(-time.Hour), nil),
			succeeded("recent", "out/recent.png", retentionNow.Add(-24*time.Hour), nil),
		},
		active: []internaljobs.Job{{ID: "chain", State: internaljobs.StateRunning, Payload: []byte(`{"input_ref":"out/in-use.png"}`)}},
	}
	// out/writing.png has no successful job: an in-flight job's result.
	if err := os.Chtimes(filepath.Join(root, "out", "writing.png"), old, old); err != nil {
		t.Fatal(err)
	}

	owner := OutputRetention{Jobs: index, Store: store, Now: func() time.Time { return retentionNow }}
	budget := retention.Budget{Name: OutputBudgetName, MaxAge: 30 * 24 * time.Hour, MaxBytes: 1 << 30}
	res, err := owner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatal(err)
	}
	if exists(t, root, "out/expired.png") || exists(t, root, "out/expired.png.meta") {
		t.Fatalf("expired output and its metadata should be deleted")
	}
	for _, kept := range []string{"out/recent.png", "out/pinned.png", "out/in-use.png", "out/shared.zip", "out/writing.png"} {
		if !exists(t, root, kept) {
			t.Fatalf("%s must be kept", kept)
		}
	}
	if res.Deleted != 1 || res.FreedBytes <= 0 {
		t.Fatalf("result = %+v, want one deletion", res)
	}
	want := map[string]int{ProtectedPinned: 1, ProtectedActiveInput: 1, ProtectedUnindexed: 1}
	for reason, n := range want {
		if res.Protected[reason] != n {
			t.Fatalf("protected[%s] = %d, want %d (all: %s)", reason, res.Protected[reason], n, res.ProtectedSummary())
		}
	}
	if res.AfterBytes != res.BeforeBytes-res.FreedBytes {
		t.Fatalf("after bytes %d != before %d - freed %d", res.AfterBytes, res.BeforeBytes, res.FreedBytes)
	}
	if err := OutputReceiptError(res, budget, nil); err != nil {
		t.Fatalf("an in-budget complete cycle is enforced, got %v", err)
	}
}

func TestOutputRetentionByteBudgetDeletesOldestButRespectsGrace(t *testing.T) {
	store, root := newOutputStore(t, "out/a.png", "out/b.png", "out/c.png")
	index := fakeOutputIndex{outputs: []internaljobs.Job{
		succeeded("a", "out/a.png", retentionNow.Add(-3*time.Hour), nil),
		succeeded("b", "out/b.png", retentionNow.Add(-2*time.Hour), nil),
		succeeded("c", "out/c.png", retentionNow.Add(-10*time.Minute), nil),
	}}
	owner := OutputRetention{Jobs: index, Store: store, Now: func() time.Time { return retentionNow }}
	// Each blob plus metadata is a little over 100 bytes; a 50-byte ceiling
	// cannot be met, so the cycle must delete every output past its grace and
	// keep the one the caller was just handed.
	budget := retention.Budget{Name: OutputBudgetName, MaxAge: 30 * 24 * time.Hour, MaxBytes: 50}
	res, err := owner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatal(err)
	}
	if exists(t, root, "out/a.png") || exists(t, root, "out/b.png") {
		t.Fatalf("oldest outputs past grace should be deleted under byte pressure")
	}
	if !exists(t, root, "out/c.png") {
		t.Fatalf("an output inside the grace period must survive byte pressure")
	}
	if res.Protected[ProtectedRecentGrace] != 1 || !res.OverBudget(budget) {
		t.Fatalf("result = %+v, want grace protection and over budget", res)
	}
	err = OutputReceiptError(res, budget, nil)
	if err == nil || !strings.Contains(err.Error(), "recent_grace=1") {
		t.Fatalf("receipt error must name the protection that kept bytes over budget, got %v", err)
	}
}

func TestOutputRetentionBoundsDeletionsPerCycle(t *testing.T) {
	store, root := newOutputStore(t, "out/1.png", "out/2.png", "out/3.png")
	old := retentionNow.Add(-60 * 24 * time.Hour)
	index := fakeOutputIndex{outputs: []internaljobs.Job{
		succeeded("1", "out/1.png", old, nil),
		succeeded("2", "out/2.png", old.Add(time.Minute), nil),
		succeeded("3", "out/3.png", old.Add(2*time.Minute), nil),
	}}
	owner := OutputRetention{Jobs: index, Store: store, Now: func() time.Time { return retentionNow }, MaxDeletes: 2}
	budget := retention.Budget{Name: OutputBudgetName, MaxAge: 30 * 24 * time.Hour, MaxBytes: 1 << 30}
	res, err := owner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 2 || !res.Incomplete || exists(t, root, "out/1.png") || !exists(t, root, "out/3.png") {
		t.Fatalf("result = %+v, want the two oldest deleted and the cycle incomplete", res)
	}
	if err := OutputReceiptError(res, budget, nil); err == nil {
		t.Fatalf("an incomplete cycle must not be recorded as enforced")
	}
}

func TestOutputBudgetComesFromOwnerManifest(t *testing.T) {
	manifest := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	budget, err := OutputBudget(manifest, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if budget.MaxAge != 30*24*time.Hour || budget.MaxBytes != 32<<30 {
		t.Fatalf("budget = %+v, want the declared 30d / 32GiB", budget)
	}
	override, err := OutputBudget(manifest, "14d", "8GiB")
	if err != nil {
		t.Fatal(err)
	}
	if override.MaxAge != 14*24*time.Hour || override.MaxBytes != 8<<30 {
		t.Fatalf("override = %+v", override)
	}
	if _, err := OutputBudget(manifest, "0d", ""); err == nil {
		t.Fatalf("a zero age override must be rejected rather than disable retention")
	}
}

// The live Python environment and sidecar never change after provisioning, so
// a generic age pruner would select every top-level entry (lib, bin,
// pyvenv.cfg) and leave a half-deleted runtime under a running sidecar. The
// owner manifest must keep both out of generic age pruning.
func TestLiveRuntimeEntriesAreNeverGenericallyAgePruned(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Storage struct {
			Entries map[string]struct {
				Budget *struct {
					MaxAge   string `json:"max_age"`
					MaxBytes string `json:"max_bytes"`
				} `json:"budget"`
				Reclaim *struct {
					Pruner string `json:"pruner"`
				} `json:"reclaim"`
			} `json:"entries"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"python_runtime", "sidecar_runtime"} {
		entry, ok := manifest.Storage.Entries[name]
		if !ok {
			t.Fatalf("%s is not declared", name)
		}
		if entry.Reclaim == nil || entry.Reclaim.Pruner != "custom" {
			t.Fatalf("%s must be owner-enforced (reclaim.pruner=custom) so the generic pruner refuses it", name)
		}
		if entry.Budget == nil || entry.Budget.MaxAge != "" || entry.Budget.MaxBytes == "" {
			t.Fatalf("%s budget = %+v; want a byte ceiling and no age bound", name, entry.Budget)
		}
	}
}

func TestRecordOutputReceiptPersistsProtectedCounts(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	budget := retention.Budget{Name: OutputBudgetName, MaxAge: 30 * 24 * time.Hour, MaxBytes: 100}
	res := OutputResult{AfterBytes: 500, Deleted: 3, Protected: map[string]int{ProtectedPinned: 2}}
	if err := RecordOutputReceipt("image-tools", retentionNow, budget, res, nil); err != nil {
		t.Fatal(err)
	}
	shared, err := retention.ReadEnforcementReceipt("image-tools")
	if err != nil {
		t.Fatal(err)
	}
	if shared.LastEnforcementTime != nil || !strings.Contains(shared.LastError, "pinned=2") {
		t.Fatalf("over-budget cycle must not read as enforced: %+v", shared)
	}
	path, err := retention.EnforcementReceiptPath("image-tools")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), outputCycleReceiptName))
	if err != nil {
		t.Fatal(err)
	}
	var detail outputCycleReceipt
	if err := json.Unmarshal(raw, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Result.Protected[ProtectedPinned] != 2 || detail.Result.Deleted != 3 || detail.Error == "" {
		t.Fatalf("detail receipt = %+v", detail)
	}
}

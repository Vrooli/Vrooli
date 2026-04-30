package sandbox

// reconciler_archive_test.go — invariants for the archive-retention pass.
//
// What the tests pin:
//   - Age-based eviction unconditionally drops archives older than the cutoff.
//   - Per-project cap evicts oldest archives within a project beyond the cap.
//   - Global size budget evicts oldest archives until total <= budget.
//   - Each lever is independently disable-able by setting it to 0.
//   - All three levers active simultaneously compose without double-counting.
//   - Idempotent: running twice in a row evicts the same set the first time
//     and an empty set the second.
//   - Blob-delete failure leaves the row in place (retried next pass) and
//     is reported via BlobFailures + LastError.
//   - When the archive seam is not configured, the reconciler is a no-op.
//
// All tests use real SQLite + a real blobstore tree under a per-test
// temp dir, mirroring service_archive_test.go's archiveTestEnv.

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/types"
)

// seedArchive inserts a sandbox row + an archive row + a single
// content blob so the retention pass has something concrete to evict
// (and so the size budget computation has non-zero bytes to work
// with). The sandbox is required because sandbox_diff_archives.
// sandbox_id is a foreign key into sandboxes(id).
//
// snapshotAt controls the row's snapshot_at; project is the
// project_root group; status is one of Approved/Rejected/Deleted.
//
// Returns the inserted archive's sandbox_id for follow-up assertions.
func seedArchive(t *testing.T, env *archiveTestEnv, snapshotAt time.Time, project string, status types.Status, blobBytes int) uuid.UUID {
	t.Helper()
	id := uuid.New()

	// Parent row first to satisfy the FK.
	now := time.Now().UTC()
	sb := &types.Sandbox{
		ID:            id,
		ScopePath:     project + "/scope",
		ProjectRoot:   project,
		Owner:         "tester",
		OwnerType:     types.OwnerTypeUser,
		Status:        status,
		DriverID:      "mock",
		DriverVersion: "1.0",
		LowerDir:      project + "/lower",
		UpperDir:      project + "/upper",
		WorkDir:       project + "/work",
		MergedDir:     project + "/merged",
		CreatedAt:     now,
		LastUsedAt:    now,
		UpdatedAt:     now,
		Version:       1,
	}
	if err := env.repo.Create(context.Background(), sb); err != nil {
		t.Fatalf("seedArchive: repo.Create: %v", err)
	}
	if err := env.repo.Update(context.Background(), sb); err != nil {
		t.Fatalf("seedArchive: repo.Update: %v", err)
	}

	if blobBytes > 0 {
		content := make([]byte, blobBytes)
		for i := range content {
			content[i] = byte(i % 251)
		}
		salt := id.String()
		copy(content[:min(len(content), len(salt))], []byte(salt))
		if _, err := env.blobs.Put(context.Background(), id.String(), content); err != nil {
			t.Fatalf("seedArchive: blobs.Put: %v", err)
		}
	}

	archive := &types.DiffArchive{
		SandboxID:      id,
		SnapshotAt:     snapshotAt,
		ArchiveState:   types.ArchiveStateComplete,
		SandboxStatus:  status,
		ProjectRoot:    project,
		Owner:          "tester",
		TotalBlobBytes: int64(blobBytes),
	}
	if err := env.archiveRepo.Insert(context.Background(), nil, archive); err != nil {
		t.Fatalf("seedArchive: Insert: %v", err)
	}
	return id
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// archiveExists is a small helper that returns whether a row with id
// is present in the archive table.
func archiveExists(t *testing.T, env *archiveTestEnv, id uuid.UUID) bool {
	t.Helper()
	a, err := env.archiveRepo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("archiveExists Get: %v", err)
	}
	return a != nil
}

// --- Age-based eviction ---

func TestRetention_AgeOnly_EvictsOldArchives(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	old := seedArchive(t, env, now.Add(-100*24*time.Hour), "/proj", types.StatusApproved, 100)
	young := seedArchive(t, env, now.Add(-10*24*time.Hour), "/proj", types.StatusApproved, 100)

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveAgeDays: 90,
	})

	if report.EvictedAge != 1 {
		t.Errorf("EvictedAge = %d, want 1", report.EvictedAge)
	}
	if report.EvictedSize != 0 || report.EvictedPerProject != 0 {
		t.Errorf("only age should evict; got size=%d perProject=%d", report.EvictedSize, report.EvictedPerProject)
	}
	if archiveExists(t, env, old) {
		t.Errorf("old archive should be evicted")
	}
	if !archiveExists(t, env, young) {
		t.Errorf("young archive should remain")
	}
}

func TestRetention_AgeZeroDisabled(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()
	old := seedArchive(t, env, now.Add(-365*24*time.Hour), "/proj", types.StatusApproved, 100)

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveAgeDays: 0,
	})
	if report.TotalEvicted() != 0 {
		t.Errorf("MaxArchiveAgeDays=0 should disable; total evicted=%d", report.TotalEvicted())
	}
	if !archiveExists(t, env, old) {
		t.Errorf("archive must remain when retention disabled")
	}
}

// --- Size-based eviction ---

func TestRetention_SizeOnly_EvictsOldestUntilBudgetMet(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	// Three archives: 100, 200, 300 bytes. Total = 600. Budget = 350.
	// Oldest first eviction should remove 100 (total=500 still over)
	// then 200 (total=300, under budget). 300 stays.
	a1 := seedArchive(t, env, now.Add(-30*24*time.Hour), "/proj", types.StatusApproved, 100)
	a2 := seedArchive(t, env, now.Add(-20*24*time.Hour), "/proj", types.StatusApproved, 200)
	a3 := seedArchive(t, env, now.Add(-10*24*time.Hour), "/proj", types.StatusApproved, 300)

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveSizeBytes: 350,
	})
	if report.EvictedSize != 2 {
		t.Errorf("EvictedSize = %d, want 2", report.EvictedSize)
	}
	if report.EvictedAge != 0 || report.EvictedPerProject != 0 {
		t.Errorf("only size lever expected; got age=%d perProject=%d", report.EvictedAge, report.EvictedPerProject)
	}
	if archiveExists(t, env, a1) || archiveExists(t, env, a2) {
		t.Errorf("oldest two archives should be gone")
	}
	if !archiveExists(t, env, a3) {
		t.Errorf("newest archive should remain")
	}
}

func TestRetention_SizeUnderBudget_NoOp(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()
	a := seedArchive(t, env, now.Add(-1*time.Hour), "/proj", types.StatusApproved, 100)

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveSizeBytes: 1000, // way over the 100 we have
	})
	if report.TotalEvicted() != 0 {
		t.Errorf("under-budget should be no-op; got %d evicted", report.TotalEvicted())
	}
	if !archiveExists(t, env, a) {
		t.Errorf("archive must remain when under budget")
	}
}

// --- Per-project cap eviction ---

func TestRetention_PerProjectCap(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	// /proj-a has 4 archives, cap=2 → evict 2 oldest.
	// /proj-b has 1 archive, cap=2 → keep.
	a1 := seedArchive(t, env, now.Add(-40*24*time.Hour), "/proj-a", types.StatusApproved, 50)
	a2 := seedArchive(t, env, now.Add(-30*24*time.Hour), "/proj-a", types.StatusApproved, 50)
	a3 := seedArchive(t, env, now.Add(-20*24*time.Hour), "/proj-a", types.StatusApproved, 50)
	a4 := seedArchive(t, env, now.Add(-10*24*time.Hour), "/proj-a", types.StatusApproved, 50)
	b1 := seedArchive(t, env, now.Add(-5*24*time.Hour), "/proj-b", types.StatusApproved, 50)

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchivesPerProject: 2,
	})
	if report.EvictedPerProject != 2 {
		t.Errorf("EvictedPerProject = %d, want 2", report.EvictedPerProject)
	}
	if report.EvictedAge != 0 || report.EvictedSize != 0 {
		t.Errorf("only per-project lever expected; got age=%d size=%d", report.EvictedAge, report.EvictedSize)
	}
	if archiveExists(t, env, a1) || archiveExists(t, env, a2) {
		t.Errorf("two oldest /proj-a archives should be gone")
	}
	if !archiveExists(t, env, a3) || !archiveExists(t, env, a4) || !archiveExists(t, env, b1) {
		t.Errorf("kept archives must remain (a3,a4,b1)")
	}
}

// --- All levers combined ---

func TestRetention_AllLevers_NoDoubleCount(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	// Set up so an archive is "selected by" multiple levers; the
	// counter should attribute it to age (which runs first).
	old := seedArchive(t, env, now.Add(-200*24*time.Hour), "/proj", types.StatusApproved, 1000)
	mid := seedArchive(t, env, now.Add(-30*24*time.Hour), "/proj", types.StatusApproved, 500)
	yng := seedArchive(t, env, now.Add(-1*24*time.Hour), "/proj", types.StatusApproved, 100)

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveAgeDays:     90,   // evicts old (-200d)
		MaxArchivesPerProject: 1,    // would also have evicted old + mid; mid stays as the survivor
		MaxArchiveSizeBytes:   1500, // post-age survivors total 600 (< 1500), so no size eviction
	})

	// old: by age (1)
	// after age, /proj has [mid, yng]; cap=1 keeps yng (newest), evicts mid.
	// after per-project: /proj has [yng]; total = 100, well under 1500, so no size evictions.
	if report.EvictedAge != 1 {
		t.Errorf("EvictedAge = %d, want 1", report.EvictedAge)
	}
	if report.EvictedPerProject != 1 {
		t.Errorf("EvictedPerProject = %d, want 1", report.EvictedPerProject)
	}
	if report.EvictedSize != 0 {
		t.Errorf("EvictedSize = %d, want 0", report.EvictedSize)
	}
	total := report.EvictedAge + report.EvictedSize + report.EvictedPerProject
	if total != 2 {
		t.Errorf("TotalEvicted = %d, want 2 (no double-count)", total)
	}
	if archiveExists(t, env, old) || archiveExists(t, env, mid) {
		t.Errorf("old and mid should be evicted")
	}
	if !archiveExists(t, env, yng) {
		t.Errorf("yng must remain")
	}
}

// --- Idempotency ---

func TestRetention_IsIdempotent(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	seedArchive(t, env, now.Add(-200*24*time.Hour), "/p", types.StatusApproved, 100)
	seedArchive(t, env, now.Add(-1*24*time.Hour), "/p", types.StatusApproved, 100)

	first := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{MaxArchiveAgeDays: 90})
	if first.EvictedAge != 1 {
		t.Fatalf("first run EvictedAge = %d, want 1", first.EvictedAge)
	}
	second := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{MaxArchiveAgeDays: 90})
	if second.TotalEvicted() != 0 {
		t.Errorf("second run should be no-op; got total=%d", second.TotalEvicted())
	}
}

// --- Empty store ---

func TestRetention_EmptyStore_ReturnsCleanReport(t *testing.T) {
	env := newArchiveTestEnv(t)
	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveAgeDays:     30,
		MaxArchiveSizeBytes:   1 << 30,
		MaxArchivesPerProject: 5,
	})
	if report.Scanned != 0 || report.TotalEvicted() != 0 || report.LastError != "" {
		t.Errorf("empty store report = %+v, want zero", report)
	}
}

// --- Blob-delete failure ---

// deleteFailingBlobs wraps a real BlobStore but injects a failure on
// DeleteSandbox for a specific sandbox ID. Other operations (Put/Get/
// Stat) pass through.
type deleteFailingBlobs struct {
	inner blobstore.BlobStore
	fail  uuid.UUID
}

func (f *deleteFailingBlobs) Put(ctx context.Context, sandboxID string, content []byte) (blobstore.PutResult, error) {
	return f.inner.Put(ctx, sandboxID, content)
}

func (f *deleteFailingBlobs) Get(ctx context.Context, sandboxID, sha string) ([]byte, error) {
	return f.inner.Get(ctx, sandboxID, sha)
}

func (f *deleteFailingBlobs) Stat(ctx context.Context, sandboxID, sha string) (int64, bool, error) {
	return f.inner.Stat(ctx, sandboxID, sha)
}

func (f *deleteFailingBlobs) DeleteSandbox(ctx context.Context, sandboxID string) error {
	if sandboxID == f.fail.String() {
		return errors.New("induced failure")
	}
	return f.inner.DeleteSandbox(ctx, sandboxID)
}

func TestRetention_BlobFailure_LeavesRow(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	bad := seedArchive(t, env, now.Add(-200*24*time.Hour), "/p", types.StatusApproved, 100)
	good := seedArchive(t, env, now.Add(-200*24*time.Hour), "/p", types.StatusApproved, 100)

	// Swap the service's blobstore for a failing wrapper. Both blobs
	// still exist on the inner store; only the DeleteSandbox seam fails.
	env.svc.blobs = &deleteFailingBlobs{inner: env.blobs, fail: bad}

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{MaxArchiveAgeDays: 90})

	if report.BlobFailures != 1 {
		t.Errorf("BlobFailures = %d, want 1", report.BlobFailures)
	}
	if report.LastError == "" || !strings.Contains(report.LastError, "induced failure") {
		t.Errorf("LastError should mention the induced failure, got %q", report.LastError)
	}
	if !archiveExists(t, env, bad) {
		t.Errorf("bad archive row should remain (row delete must wait until blob delete succeeds)")
	}
	if archiveExists(t, env, good) {
		t.Errorf("good archive should be evicted")
	}
}

// --- Per-archive observability ---

func TestRetention_LargeBatch_AllLeversSeeWholePicture(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()

	// Seed 50 archives across 5 projects, ages spread out, varying sizes.
	want := 0
	for i := 0; i < 50; i++ {
		project := "/proj-" + strconv.Itoa(i%5)
		age := time.Duration(i) * 24 * time.Hour
		size := 100 + i*10
		seedArchive(t, env, now.Add(-age), project, types.StatusApproved, size)
		want++
	}

	report := env.svc.ReconcileArchiveRetention(context.Background(), RetentionPolicy{
		MaxArchiveAgeDays:     30, // archives older than 30d
		MaxArchivesPerProject: 3,  // each project caps at 3
		MaxArchiveSizeBytes:   1 << 20,
	})
	if report.Scanned != want {
		t.Errorf("Scanned = %d, want %d", report.Scanned, want)
	}
	if report.LastError != "" {
		t.Errorf("LastError = %q, want clean", report.LastError)
	}
}

// --- Reconciler wrapper ---

func TestArchiveRetentionReconciler_Disabled_FastPath(t *testing.T) {
	env := newArchiveTestEnv(t)
	// Even if archives exist, all-zero policy means the reconciler
	// must skip the DB call entirely. We assert by creating an
	// archive that ages out under any non-zero age policy and
	// verifying it's untouched.
	now := time.Now().UTC()
	a := seedArchive(t, env, now.Add(-1000*24*time.Hour), "/p", types.StatusApproved, 100)

	rec := NewArchiveRetentionReconciler(env.svc, func() RetentionPolicy { return RetentionPolicy{} })
	report := rec.Run(context.Background())
	if report.ItemsProcessed != 0 || report.ItemsFailed != 0 || report.LastError != "" {
		t.Errorf("disabled-policy report = %+v, want zero", report)
	}
	if !archiveExists(t, env, a) {
		t.Errorf("archive must not be evicted when policy is all-zero")
	}
}

func TestArchiveRetentionReconciler_PropagatesEviction(t *testing.T) {
	env := newArchiveTestEnv(t)
	now := time.Now().UTC()
	old := seedArchive(t, env, now.Add(-200*24*time.Hour), "/p", types.StatusApproved, 100)

	rec := NewArchiveRetentionReconciler(env.svc, func() RetentionPolicy {
		return RetentionPolicy{MaxArchiveAgeDays: 90}
	})
	report := rec.Run(context.Background())

	if report.ItemsProcessed != 1 {
		t.Errorf("ItemsProcessed = %d, want 1", report.ItemsProcessed)
	}
	if report.Details == nil {
		t.Fatalf("Details map missing")
	}
	if got, _ := report.Details["evictedAge"].(int); got != 1 {
		t.Errorf("Details.evictedAge = %v, want 1", report.Details["evictedAge"])
	}
	if got, _ := report.Details["totalEvicted"].(int); got != 1 {
		t.Errorf("Details.totalEvicted = %v, want 1", report.Details["totalEvicted"])
	}
	if archiveExists(t, env, old) {
		t.Errorf("archive should be evicted")
	}
}

func TestArchiveRetentionReconciler_NilProvider_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil provider")
		}
	}()
	env := newArchiveTestEnv(t)
	NewArchiveRetentionReconciler(env.svc, nil)
}

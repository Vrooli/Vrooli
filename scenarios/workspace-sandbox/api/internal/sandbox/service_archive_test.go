package sandbox

// Tests for the snapshot-on-terminal-transition seam (Phase 2). The
// invariants exercised here are normative: future refactors must not
// regress them.
//
// Scope:
//   - Approve full → archive row exists, blobs on disk, status=Approved.
//   - Approve partial → no archive, status remains Active.
//   - Reject → archive row exists, status=Rejected.
//   - Delete from Active without prior archive → archive row exists.
//   - Delete from Error → archive_state="not_captured", no blobs.
//   - Delete from Approved (auto-lifecycle) → archive untouched, status=Deleted.
//   - Snapshot failure aborts the transition.
//   - Live GetDiff vs archive equivalence.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/storage"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
)

// archiveTestEnv bundles a real SQLite-backed Service with a real
// archive repo + blobstore wired through a per-test storage root. The
// driver, gitops, and patcher are fakes so tests don't shell out.
type archiveTestEnv struct {
	t           *testing.T
	svc         *Service
	repo        *repository.SandboxRepository
	archiveRepo *repository.SandboxArchiveRepository
	blobs       *blobstore.Store
	drv         *mocks.FakeDriver
	tmp         string
}

func newArchiveTestEnv(t *testing.T) *archiveTestEnv {
	t.Helper()

	tmp := t.TempDir()
	// Pin storage class roots under the test temp dir via the sanctioned
	// override; the user-profile default now resolves under the operator runtime
	// home (~/.vrooli) and no longer honors XDG env vars.
	t.Setenv("VROOLI_STORAGE_ROOT", tmp)

	sqliteDB := db.NewSQLite(t)
	repo := repository.NewSandboxRepository(sqliteDB, schedule.System())
	archiveRepo := repository.NewArchiveRepository(sqliteDB, schedule.System())

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli-archive-test",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		t.Fatalf("storage.NewResolver: %v", err)
	}
	blobs, err := blobstore.New(resolver)
	if err != nil {
		t.Fatalf("blobstore.New: %v", err)
	}

	drv := mocks.NewFakeDriver()
	clk := schedule.System()
	svc := NewService(
		repo, drv, ServiceConfig{DefaultProjectRoot: tmp, MaxSandboxes: 100},
		clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(),
		WithGitOps(mocks.NewFakeGitOps()),
		WithArchive(archiveRepo, blobs),
	)

	return &archiveTestEnv{
		t:           t,
		svc:         svc,
		repo:        repo,
		archiveRepo: archiveRepo,
		blobs:       blobs,
		drv:         drv,
		tmp:         tmp,
	}
}

// makeSandbox creates a sandbox row in the DB plus on-disk upper/lower
// dirs. Files in upperContents are written to the upper dir; ChangedFiles
// are populated on the FakeDriver to match.
func (e *archiveTestEnv) makeSandbox(status types.Status, upperContents map[string]string, deletedFiles []string) *types.Sandbox {
	e.t.Helper()

	id := uuid.New()
	root := filepath.Join(e.tmp, id.String())
	upper := filepath.Join(root, "upper")
	lower := filepath.Join(root, "lower")
	work := filepath.Join(root, "work")
	merged := filepath.Join(root, "merged")
	for _, d := range []string{upper, lower, work, merged} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			e.t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	changes := make([]*types.FileChange, 0, len(upperContents)+len(deletedFiles))
	for path, content := range upperContents {
		full := filepath.Join(upper, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			e.t.Fatalf("mkdir upper: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			e.t.Fatalf("write upper %s: %v", path, err)
		}
		changes = append(changes, &types.FileChange{
			ID:         uuid.New(),
			SandboxID:  id,
			FilePath:   path,
			ChangeType: types.ChangeTypeAdded,
			FileSize:   int64(len(content)),
		})
	}
	for _, path := range deletedFiles {
		// "Deleted" files are in lower (the original) and absent in upper.
		full := filepath.Join(lower, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			e.t.Fatalf("mkdir lower: %v", err)
		}
		original := "OLD CONTENT for " + path
		if err := os.WriteFile(full, []byte(original), 0o644); err != nil {
			e.t.Fatalf("write lower %s: %v", path, err)
		}
		changes = append(changes, &types.FileChange{
			ID:         uuid.New(),
			SandboxID:  id,
			FilePath:   path,
			ChangeType: types.ChangeTypeDeleted,
			FileSize:   int64(len(original)),
		})
	}
	e.drv.ChangedFiles = changes

	now := time.Now().UTC()
	sb := &types.Sandbox{
		ID:            id,
		ScopePath:     filepath.Join(e.tmp, "src"),
		ProjectRoot:   e.tmp,
		Owner:         "test-user",
		OwnerType:     types.OwnerTypeUser,
		Status:        status,
		DriverID:      "mock",
		DriverVersion: "1.0",
		LowerDir:      lower,
		UpperDir:      upper,
		WorkDir:       work,
		MergedDir:     merged,
		CreatedAt:     now,
		LastUsedAt:    now,
		UpdatedAt:     now,
		Version:       1,
		Metadata: map[string]interface{}{
			"agent_manager_run_id": "run-archive-test",
		},
	}
	if err := e.repo.Create(context.Background(), sb); err != nil {
		e.t.Fatalf("repo.Create: %v", err)
	}
	// Create persists the column subset that's known at creation time;
	// mount paths and computed status fields land via Update. Re-stamp
	// the row so subsequent Get() calls see the on-disk overlay paths.
	if err := e.repo.Update(context.Background(), sb); err != nil {
		e.t.Fatalf("repo.Update (paths): %v", err)
	}
	return sb
}

// hashContent returns the lowercase hex SHA-256 of content.
func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// blobBytes fetches a blob via the BlobStore and asserts it round-trips.
func (e *archiveTestEnv) blobBytes(sandboxID uuid.UUID, hashHex string) []byte {
	e.t.Helper()
	b, err := e.blobs.Get(context.Background(), sandboxID.String(), hashHex)
	if err != nil {
		e.t.Fatalf("blobs.Get %s/%s: %v", sandboxID, hashHex, err)
	}
	return b
}

// --- Approve full branch ---

func TestSnapshot_ApproveFull_WritesArchive(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"new.txt":     "hello world\n",
		"sub/dir.txt": "nested content\n",
	}, nil)

	result, err := env.svc.Approve(context.Background(), &types.ApprovalRequest{
		SandboxID:         sb.ID,
		Mode:              "all",
		Actor:             "test-user",
		AgentManagerRunID: "run-archive-test",
	})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if !result.Success {
		t.Fatalf("Approve.Success = false, ErrorMsg=%q", result.ErrorMsg)
	}

	// Status flipped to Approved.
	got, err := env.repo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if got.Status != types.StatusApproved {
		t.Errorf("status = %s, want approved", got.Status)
	}

	// Archive row exists with state=complete.
	archive, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get: %v", err)
	}
	if archive == nil {
		t.Fatal("archive row not written")
	}
	if archive.ArchiveState != types.ArchiveStateComplete {
		t.Errorf("archive_state = %s, want complete", archive.ArchiveState)
	}
	if archive.SandboxStatus != types.StatusApproved {
		t.Errorf("sandbox_status = %s, want approved", archive.SandboxStatus)
	}
	if archive.AgentManagerRunID != "run-archive-test" {
		t.Errorf("run_id = %q, want run-archive-test", archive.AgentManagerRunID)
	}
	if archive.ProjectRoot != sb.ProjectRoot {
		t.Errorf("project_root = %q, want %q", archive.ProjectRoot, sb.ProjectRoot)
	}
	if archive.Owner != sb.Owner {
		t.Errorf("owner = %q, want %q", archive.Owner, sb.Owner)
	}
	if archive.UnifiedDiffSHA256 == "" {
		t.Error("unified_diff_sha256 empty")
	}
	if archive.TotalBlobBytes <= 0 {
		t.Errorf("total_blob_bytes = %d, want > 0", archive.TotalBlobBytes)
	}
	if len(archive.Files) != 2 {
		t.Errorf("Files count = %d, want 2", len(archive.Files))
	}

	// Each file's blob is fetchable and matches its declared hash.
	wantContent := map[string]string{
		"new.txt":     "hello world\n",
		"sub/dir.txt": "nested content\n",
	}
	for _, e := range archive.Files {
		if e.BlobSHA256 == "" {
			t.Errorf("file %s has no blob hash", e.Path)
			continue
		}
		if e.BlobSHA256 != hashContent([]byte(wantContent[e.Path])) {
			t.Errorf("file %s blob hash mismatch", e.Path)
		}
	}
	for _, e := range archive.Files {
		if e.BlobSHA256 == "" {
			continue
		}
		got := env.blobBytes(sb.ID, e.BlobSHA256)
		if string(got) != wantContent[e.Path] {
			t.Errorf("file %s blob content = %q, want %q", e.Path, got, wantContent[e.Path])
		}
	}
}

// --- Approve partial: no archive ---

func TestSnapshot_ApprovePartial_NoArchive(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"a.txt": "AAA\n",
		"b.txt": "BBB\n",
	}, nil)

	// Approve only one file. The other remains pending; the sandbox
	// stays Active. No archive should be written until full approval
	// (or rejection).
	picked := env.drv.ChangedFiles[0]
	_, err := env.svc.Approve(context.Background(), &types.ApprovalRequest{
		SandboxID: sb.ID,
		Mode:      "files",
		Actor:     "test-user",
		FileIDs:   []uuid.UUID{picked.ID},
	})
	if err != nil {
		t.Fatalf("Approve partial: %v", err)
	}

	got, err := env.repo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if got.Status != types.StatusActive {
		t.Errorf("status = %s, want active (partial leaves sandbox open)", got.Status)
	}
	archive, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get: %v", err)
	}
	if archive != nil {
		t.Errorf("partial approval should not write archive (got %v)", archive)
	}
}

// --- Reject ---

func TestSnapshot_Reject_WritesArchive(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"r.txt": "rejected content\n",
	}, nil)

	if _, err := env.svc.Reject(context.Background(), sb.ID, "test-user"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	got, err := env.repo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if got.Status != types.StatusRejected {
		t.Errorf("status = %s, want rejected", got.Status)
	}
	archive, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get: %v", err)
	}
	if archive == nil {
		t.Fatal("Reject did not write archive")
	}
	if archive.SandboxStatus != types.StatusRejected {
		t.Errorf("archive sandbox_status = %s, want rejected", archive.SandboxStatus)
	}
	if archive.ArchiveState != types.ArchiveStateComplete {
		t.Errorf("archive_state = %s, want complete", archive.ArchiveState)
	}
	if len(archive.Files) != 1 {
		t.Errorf("Files count = %d, want 1", len(archive.Files))
	}
}

// --- Delete from Active without prior archive ---

func TestSnapshot_Delete_FromActive_WritesArchive(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"del.txt": "to be archived\n",
	}, nil)

	if err := env.svc.Delete(context.Background(), sb.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err := env.repo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if got.Status != types.StatusDeleted {
		t.Errorf("status = %s, want deleted", got.Status)
	}
	if got.DeletedAt == nil {
		t.Error("DeletedAt should be stamped")
	}
	archive, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get: %v", err)
	}
	if archive == nil {
		t.Fatal("Delete from Active did not write archive")
	}
	if archive.SandboxStatus != types.StatusDeleted {
		t.Errorf("archive sandbox_status = %s, want deleted", archive.SandboxStatus)
	}
	if archive.ArchiveState != types.ArchiveStateComplete {
		t.Errorf("archive_state = %s, want complete", archive.ArchiveState)
	}
}

// --- Delete from Error: not_captured ---

func TestSnapshot_Delete_FromError_NotCaptured(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusError, nil, nil)

	if err := env.svc.Delete(context.Background(), sb.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	archive, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get: %v", err)
	}
	if archive == nil {
		t.Fatal("Delete from Error did not write archive marker")
	}
	if archive.ArchiveState != types.ArchiveStateNotCaptured {
		t.Errorf("archive_state = %s, want not_captured", archive.ArchiveState)
	}
	if len(archive.Files) != 0 {
		t.Errorf("Files count = %d, want 0 for not_captured", len(archive.Files))
	}
	if archive.UnifiedDiffSHA256 != "" {
		t.Errorf("unified_diff_sha256 = %q, want empty for not_captured", archive.UnifiedDiffSHA256)
	}
	if archive.TotalBlobBytes != 0 {
		t.Errorf("total_blob_bytes = %d, want 0 for not_captured", archive.TotalBlobBytes)
	}
}

// --- Delete from Approved: archive untouched (lifecycle path) ---

func TestSnapshot_Delete_AfterApprove_PreservesArchive(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"keep.txt": "keep me\n",
	}, nil)

	if _, err := env.svc.Approve(context.Background(), &types.ApprovalRequest{
		SandboxID: sb.ID,
		Mode:      "all",
		Actor:     "test-user",
	}); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	first, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get after Approve: %v", err)
	}
	if first == nil {
		t.Fatal("Approve did not write archive")
	}
	firstSnap := first.SnapshotAt
	firstStatus := first.SandboxStatus
	firstHash := first.UnifiedDiffSHA256

	// Lifecycle path: explicit Delete after Approve is exactly what
	// applyLifecycleOnTerminal would call. The archive must not be
	// re-snapshotted: it should retain the SandboxStatus=approved
	// captured at the apply moment.
	if err := env.svc.Delete(context.Background(), sb.ID); err != nil {
		t.Fatalf("Delete after Approve: %v", err)
	}

	second, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get after Delete: %v", err)
	}
	if second == nil {
		t.Fatal("archive lost after Delete")
	}
	if !second.SnapshotAt.Equal(firstSnap) {
		t.Errorf("snapshot_at changed: was %v, now %v (Delete should not re-snapshot)", firstSnap, second.SnapshotAt)
	}
	if second.SandboxStatus != firstStatus {
		t.Errorf("archive sandbox_status changed: was %s, now %s", firstStatus, second.SandboxStatus)
	}
	if second.UnifiedDiffSHA256 != firstHash {
		t.Errorf("unified_diff_sha256 changed: was %s, now %s", firstHash, second.UnifiedDiffSHA256)
	}

	// Sandbox row reflects the post-Delete status.
	got, err := env.repo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if got.Status != types.StatusDeleted {
		t.Errorf("sandbox status = %s, want deleted", got.Status)
	}
}

// --- Snapshot failure aborts the transition ---

// failingBlobs simulates a blob-write failure on the second Put. The
// first (unified-diff) Put succeeds, then per-file Put fails — so we
// exercise both the rollback and the partial-blob cleanup path.
type failingBlobs struct {
	blobstore.BlobStore
	calls    int
	failOn   int
	deleteCb func(string)
}

func (f *failingBlobs) Put(ctx context.Context, sandboxID string, content []byte) (blobstore.PutResult, error) {
	f.calls++
	if f.calls == f.failOn {
		return blobstore.PutResult{}, errors.New("disk full (simulated)")
	}
	return f.BlobStore.Put(ctx, sandboxID, content)
}

func (f *failingBlobs) DeleteSandbox(ctx context.Context, sandboxID string) error {
	if f.deleteCb != nil {
		f.deleteCb(sandboxID)
	}
	return f.BlobStore.DeleteSandbox(ctx, sandboxID)
}

func TestSnapshot_BlobFailure_AbortsTransition(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"a.txt": "content A\n",
		"b.txt": "content B\n",
	}, nil)

	cleanedUp := 0
	failing := &failingBlobs{
		BlobStore: env.blobs,
		failOn:    2, // unified-diff (1) succeeds, first per-file (2) fails
		deleteCb:  func(string) { cleanedUp++ },
	}
	// Re-wire the service with the failing blob store.
	env.svc.blobs = failing

	// Reject is the simplest path to exercise — no patch/commit side
	// effects to disentangle.
	_, err := env.svc.Reject(context.Background(), sb.ID, "test-user")
	if err == nil {
		t.Fatal("Reject must fail when blob write fails")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error chain = %v, want to wrap simulated failure", err)
	}

	// Sandbox stays Active; no archive row.
	got, err := env.repo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if got.Status != types.StatusActive {
		t.Errorf("status = %s, want active (snapshot failure must not flip status)", got.Status)
	}
	archive, err := env.archiveRepo.Get(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("archiveRepo.Get: %v", err)
	}
	if archive != nil {
		t.Errorf("archive row exists despite snapshot failure: %+v", archive)
	}

	// Cleanup ran exactly once.
	if cleanedUp != 1 {
		t.Errorf("cleanup callbacks = %d, want 1", cleanedUp)
	}
}

// --- Live vs archive equivalence ---

// On the same sandbox state, the live GetDiff and the archived
// GetArchive must produce structurally equivalent DiffResults: same
// file count, same per-file change types, same Stats. Per-file
// FileChange ID and DetectedAt are not compared (the live path
// re-derives those from the driver; the archive freezes them at
// snapshot time).
func TestSnapshot_LiveAndArchive_Equivalent(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"x.txt":      "X\n",
		"y/inner.go": "package y\n",
	}, nil)

	live, err := env.svc.GetDiff(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("live GetDiff: %v", err)
	}

	if _, err := env.svc.Reject(context.Background(), sb.ID, "test-user"); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	archived, err := env.svc.GetArchive(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("GetArchive: %v", err)
	}
	if archived == nil {
		t.Fatal("archive missing")
	}

	if len(archived.Files) != len(live.Files) {
		t.Fatalf("file count: live=%d archive=%d", len(live.Files), len(archived.Files))
	}
	if archived.Stats.FilesAdded != live.Stats.FilesAdded ||
		archived.Stats.FilesModified != live.Stats.FilesModified ||
		archived.Stats.FilesDeleted != live.Stats.FilesDeleted {
		t.Errorf("Stats mismatch: live=%+v archive=%+v", live.Stats, archived.Stats)
	}
	if archived.UnifiedDiff != live.UnifiedDiff {
		t.Errorf("unified_diff content drift: live len=%d archive len=%d",
			len(live.UnifiedDiff), len(archived.UnifiedDiff))
	}
	if archived.ArchiveState != types.ArchiveStateComplete {
		t.Errorf("archive_state = %s, want complete", archived.ArchiveState)
	}
}

// --- FetchArchiveFile returns content; missing returns ErrNotFound ---

func TestFetchArchiveFile(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"hello.txt": "hello\n",
	}, nil)
	if _, err := env.svc.Reject(context.Background(), sb.ID, "test"); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	got, err := env.svc.FetchArchiveFile(context.Background(), sb.ID, "hello.txt")
	if err != nil {
		t.Fatalf("FetchArchiveFile: %v", err)
	}
	if !bytes.Equal(got, []byte("hello\n")) {
		t.Errorf("content = %q, want %q", got, "hello\n")
	}

	if _, err := env.svc.FetchArchiveFile(context.Background(), sb.ID, "nope.txt"); !errors.Is(err, blobstore.ErrNotFound) {
		t.Errorf("missing path err = %v, want ErrNotFound", err)
	}

	if _, err := env.svc.FetchArchiveFile(context.Background(), uuid.New(), "any.txt"); !errors.Is(err, blobstore.ErrNotFound) {
		t.Errorf("unknown sandbox err = %v, want ErrNotFound", err)
	}
}

// --- ListHistory respects filters ---

func TestListHistory_FiltersAndPagination(t *testing.T) {
	env := newArchiveTestEnv(t)

	// Three sandboxes: one Approved, one Rejected, one Deleted-from-Error.
	sb1 := env.makeSandbox(types.StatusActive, map[string]string{"a.txt": "1\n"}, nil)
	if _, err := env.svc.Approve(context.Background(), &types.ApprovalRequest{
		SandboxID: sb1.ID, Mode: "all", Actor: "u1",
	}); err != nil {
		t.Fatalf("Approve sb1: %v", err)
	}

	sb2 := env.makeSandbox(types.StatusActive, map[string]string{"b.txt": "2\n"}, nil)
	if _, err := env.svc.Reject(context.Background(), sb2.ID, "u2"); err != nil {
		t.Fatalf("Reject sb2: %v", err)
	}

	sb3 := env.makeSandbox(types.StatusError, nil, nil)
	if err := env.svc.Delete(context.Background(), sb3.ID); err != nil {
		t.Fatalf("Delete sb3: %v", err)
	}

	// All three.
	all, total, err := env.svc.ListHistory(context.Background(), types.ArchiveListFilter{})
	if err != nil {
		t.Fatalf("ListHistory all: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Errorf("all = %d (total %d), want 3", len(all), total)
	}

	// Filter by Approved only.
	approved, total, err := env.svc.ListHistory(context.Background(), types.ArchiveListFilter{
		Statuses: []types.Status{types.StatusApproved},
	})
	if err != nil {
		t.Fatalf("ListHistory approved: %v", err)
	}
	if total != 1 || len(approved) != 1 {
		t.Errorf("approved count = %d (total %d), want 1", len(approved), total)
	}
	if len(approved) > 0 && approved[0].SandboxID != sb1.ID {
		t.Errorf("approved[0].SandboxID = %s, want %s", approved[0].SandboxID, sb1.ID)
	}

	// Pagination.
	page, total, err := env.svc.ListHistory(context.Background(), types.ArchiveListFilter{
		Limit: 2, Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListHistory page: %v", err)
	}
	if total != 3 {
		t.Errorf("page total = %d, want 3 (total ignores limit)", total)
	}
	if len(page) != 2 {
		t.Errorf("page len = %d, want 2", len(page))
	}
}

// --- Idempotent Approve does not double-snapshot ---

// Re-approving an already-approved sandbox returns immediately without
// touching the archive. Per the existing idempotency contract in
// service_review.go.
func TestSnapshot_Approve_Idempotent(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"a.txt": "x\n"}, nil)

	if _, err := env.svc.Approve(context.Background(), &types.ApprovalRequest{
		SandboxID: sb.ID, Mode: "all", Actor: "test",
	}); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	first, _ := env.archiveRepo.Get(context.Background(), sb.ID)
	if first == nil {
		t.Fatal("first Approve did not write archive")
	}

	if _, err := env.svc.Approve(context.Background(), &types.ApprovalRequest{
		SandboxID: sb.ID, Mode: "all", Actor: "test",
	}); err != nil {
		t.Fatalf("re-Approve: %v", err)
	}
	second, _ := env.archiveRepo.Get(context.Background(), sb.ID)
	if !second.SnapshotAt.Equal(first.SnapshotAt) {
		t.Errorf("re-Approve mutated archive snapshot_at: was %v, now %v",
			first.SnapshotAt, second.SnapshotAt)
	}
}

// Sanity check: Approve when no archive seam wired falls back to a
// plain status flip.
func TestSnapshot_NoArchiveSeam_Bypassed(t *testing.T) {
	tmp := t.TempDir()
	// Pin storage class roots under the test temp dir via the sanctioned
	// override; the user-profile default now resolves under the operator runtime
	// home (~/.vrooli) and no longer honors XDG env vars.
	t.Setenv("VROOLI_STORAGE_ROOT", tmp)

	sqliteDB := db.NewSQLite(t)
	repo := repository.NewSandboxRepository(sqliteDB, schedule.System())
	drv := mocks.NewFakeDriver()
	clk := schedule.System()
	svc := NewService(
		repo, drv, ServiceConfig{},
		clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(),
		WithGitOps(mocks.NewFakeGitOps()),
	)

	id := uuid.New()
	now := time.Now().UTC()
	sb := &types.Sandbox{
		ID: id, ScopePath: filepath.Join(tmp, "p"), ProjectRoot: tmp,
		Owner: "u", OwnerType: types.OwnerTypeUser, Status: types.StatusActive,
		DriverID: "mock", DriverVersion: "1.0",
		LowerDir: filepath.Join(tmp, "lower"), UpperDir: filepath.Join(tmp, "upper"),
		WorkDir: filepath.Join(tmp, "work"), MergedDir: filepath.Join(tmp, "merged"),
		CreatedAt: now, LastUsedAt: now, UpdatedAt: now, Version: 1,
	}
	for _, d := range []string{sb.LowerDir, sb.UpperDir, sb.WorkDir, sb.MergedDir} {
		_ = os.MkdirAll(d, 0o755)
	}
	if err := repo.Create(context.Background(), sb); err != nil {
		t.Fatalf("Create: %v", err)
	}
	drv.ChangedFiles = []*types.FileChange{}

	if _, err := svc.Reject(context.Background(), id, "test"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	got, _ := repo.Get(context.Background(), id)
	if got.Status != types.StatusRejected {
		t.Errorf("status = %s, want rejected (no-archive bypass)", got.Status)
	}
}

// --- Defensive: live diff still reuses Service.GetDiff ---

// Confirm the snapshot path uses the same generator as live; if a future
// refactor introduces a parallel diff path, this test fails because the
// archive's UnifiedDiff would diverge from a fresh diff regenerated via
// a clean-slate Generator.
func TestSnapshot_DiffEquivalence_VsFreshGenerator(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{
		"a.txt": "alpha\n",
	}, nil)

	gen := diff.NewGenerator(process.NewOSExecStarter())
	expected, err := gen.GenerateDiff(context.Background(), sb,
		[]*types.FileChange{env.drv.ChangedFiles[0]},
		&diff.GenerateOptions{PathPrefix: scopePathPrefix(sb)},
	)
	if err != nil {
		t.Fatalf("fresh GenerateDiff: %v", err)
	}

	if _, err := env.svc.Reject(context.Background(), sb.ID, "test"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	got, err := env.svc.GetArchive(context.Background(), sb.ID)
	if err != nil {
		t.Fatalf("GetArchive: %v", err)
	}
	if got.UnifiedDiff != expected.UnifiedDiff {
		t.Errorf("archive UnifiedDiff diverges from fresh generator: archive=%q expected=%q",
			truncForDiag(got.UnifiedDiff), truncForDiag(expected.UnifiedDiff))
	}
}

func truncForDiag(s string) string {
	const max = 200
	if len(s) <= max {
		return s
	}
	return fmt.Sprintf("%s...[%d more bytes]", s[:max], len(s)-max)
}

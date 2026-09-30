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
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
// driver and gitops are fakes; diff generation and application use real tools
// against temporary source trees, with commit creation disabled.
type archiveTestEnv struct {
	t           *testing.T
	svc         *Service
	repo        *repository.SandboxRepository
	archiveRepo *repository.SandboxArchiveRepository
	blobs       *blobstore.Store
	drv         *mocks.FakeDriver
	tmp         string
}

func TestApprovalRefusesChangedReviewedPatch(t *testing.T) {
	for _, mutation := range []string{"text", "binary", "removed", "force"} {
		t.Run(mutation, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed\n"}, nil)
			reviewed, err := env.svc.GetDiff(t.Context(), sb.ID)
			if err != nil {
				t.Fatal(err)
			}
			request := reviewedApproval(t, sb.ID, hashContent([]byte(reviewed.UnifiedDiff)))
			request.Force = mutation == "force"
			if mutation == "removed" {
				env.drv.ChangedFiles = nil
			} else {
				content := "unreviewed\n"
				if mutation == "binary" {
					content = "unreviewed\x00binary"
				}
				if err := os.WriteFile(filepath.Join(sb.UpperDir, "review.txt"), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			result, err := env.svc.Approve(t.Context(), request)
			if err == nil {
				t.Fatalf("changed reviewed patch must be refused before apply, got %+v", result)
			}
			if _, err := os.Stat(filepath.Join(sb.ScopePath, "review.txt")); !os.IsNotExist(err) {
				t.Fatalf("refused approval changed canonical source: %v", err)
			}
			stored, err := env.repo.Get(t.Context(), sb.ID)
			if err != nil || stored.Status != types.StatusActive {
				t.Fatalf("refused approval changed sandbox status: %+v, %v", stored, err)
			}
		})
	}
}

func TestApprovalRetainsPatchBeforeChangingItsLiveBaseline(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "after review\n"}, nil)
	if err := os.MkdirAll(sb.ScopePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sb.ScopePath, "review.txt"), []byte("before review\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// A real overlay lower layer can refer to the canonical project. Applying
	// changes mutates that baseline; a later diff is not the approved patch.
	sb.LowerDir = sb.ScopePath
	env.drv.ChangedFiles[0].ChangeType = types.ChangeTypeModified
	if err := env.repo.Update(t.Context(), sb); err != nil {
		t.Fatal(err)
	}
	reviewed, err := env.svc.GetDiff(t.Context(), sb.ID)
	if err != nil || reviewed.UnifiedDiff == "" {
		t.Fatalf("review fixture: %+v, %v", reviewed, err)
	}
	result, err := env.svc.Approve(t.Context(), reviewedApproval(t, sb.ID, reviewed.PatchSHA256))
	if err != nil || !result.Success {
		t.Fatalf("apply reviewed patch: %+v, %v", result, err)
	}
	archived, err := env.svc.GetDiff(t.Context(), sb.ID)
	if err != nil || archived.UnifiedDiff != reviewed.UnifiedDiff || archived.PatchSHA256 != result.AppliedPatchSHA256 {
		t.Fatalf("archive must retain the actual applied patch, not rediff a changed baseline: %+v, %v", archived, err)
	}
}

func TestApprovalArchivePreparationFailureDoesNotWriteSource(t *testing.T) {
	for _, failedWrite := range []int{1, 2} {
		t.Run(fmt.Sprintf("blob-%d", failedWrite), func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed\n"}, nil)
			env.svc.blobs = &failingBlobs{BlobStore: env.blobs, failOn: failedWrite}
			result, err := env.svc.Approve(t.Context(), &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"})
			if err == nil {
				t.Fatalf("unretained approval must fail before apply: %+v", result)
			}
			if _, err := os.Stat(filepath.Join(sb.ScopePath, "review.txt")); !os.IsNotExist(err) {
				t.Fatalf("archive preparation failure must precede source effects: %v", err)
			}
			stored, err := env.repo.Get(t.Context(), sb.ID)
			if err != nil || stored.Status != types.StatusActive {
				t.Fatalf("preparation failure changed status: %+v, %v", stored, err)
			}
			archive, err := env.archiveRepo.Get(t.Context(), sb.ID)
			if err != nil || archive != nil {
				t.Fatalf("failed preparation published an archive: %+v, %v", archive, err)
			}
		})
	}
}

type failingArchiveInsert struct {
	repository.ArchiveRepository
}

func (f *failingArchiveInsert) Insert(context.Context, *sql.Tx, *types.DiffArchive) error {
	return errors.New("archive insert failed (simulated)")
}

type lostPreparedResponse struct {
	repository.ArchiveRepository
}

func (f *lostPreparedResponse) PutPreparedApproval(ctx context.Context, intent *types.PreparedApproval) error {
	if err := f.ArchiveRepository.PutPreparedApproval(ctx, intent); err != nil {
		return err
	}
	return errors.New("prepared response lost (simulated)")
}

func (e *archiveTestEnv) freshService() *Service {
	e.t.Helper()
	clk := schedule.System()
	return NewService(e.repo, e.drv, e.svc.config, clk,
		audit.NewRepoEmitter(e.repo.LogAuditEvent, clk), process.NewOSExecStarter(),
		WithGitOps(mocks.NewFakeGitOps()), WithArchive(e.archiveRepo, e.blobs))
}

func TestPreparedApprovalLostResponseRecoveryAndGuards(t *testing.T) {
	env := newArchiveTestEnv(t)
	contents := map[string]string{"review.txt": "reviewed\n", "binary.bin": "reviewed\x00binary"}
	sb := env.makeSandbox(types.StatusActive, contents, nil)
	reviewed, err := env.svc.GetDiff(t.Context(), sb.ID)
	if err != nil {
		t.Fatal(err)
	}
	req := reviewedApproval(t, sb.ID, reviewed.PatchSHA256)
	env.svc.archiveRepo = &lostPreparedResponse{ArchiveRepository: env.archiveRepo}
	if _, err := env.svc.Approve(t.Context(), req); err == nil || !strings.Contains(err.Error(), "prepared response lost") {
		t.Fatalf("did not interrupt after durable preparation: %v", err)
	}
	svc := env.freshService()
	intent, err := env.archiveRepo.GetPreparedApproval(t.Context(), sb.ID)
	if err != nil || intent == nil || intent.Archive.UnifiedDiffSHA256 != reviewed.PatchSHA256 {
		t.Fatalf("missing durable original intent: %+v, %v", intent, err)
	}
	guards := map[string]func() error{
		"delete":  func() error { return svc.Delete(t.Context(), sb.ID) },
		"reject":  func() error { _, err := svc.Reject(t.Context(), sb.ID, "operator"); return err },
		"start":   func() error { _, err := svc.Start(t.Context(), sb.ID); return err },
		"resume":  func() error { _, err := svc.Resume(t.Context(), sb.ID); return err },
		"discard": func() error { _, err := svc.Discard(t.Context(), &types.DiscardRequest{SandboxID: sb.ID}); return err },
		"rebase":  func() error { _, err := svc.Rebase(t.Context(), &types.RebaseRequest{SandboxID: sb.ID}); return err },
		"checkpoint": func() error {
			_, err := svc.TurnCheckpoint(t.Context(), &types.TurnCheckpointRequest{SandboxID: sb.ID, AgentManagerRunID: "run", Source: types.SourceAgentManagerAutoApply})
			return err
		},
	}
	for name, invoke := range guards {
		if err := invoke(); err == nil || !strings.Contains(err.Error(), "pending prepared approval") {
			t.Errorf("%s must preserve pending evidence: %v", name, err)
		}
	}
	wrong := *req
	wrong.ExpectedPatchSHA256 = hashContent([]byte("different patch"))
	if _, err := svc.Approve(t.Context(), &wrong); err == nil {
		t.Fatal("recovery accepted another patch identity")
	}
	wrong = *req
	wrong.Actor = "different actor"
	if _, err := svc.Approve(t.Context(), &wrong); err == nil {
		t.Fatal("recovery rewrote the original attribution")
	}
	if err := os.MkdirAll(sb.ScopePath, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sb.ScopePath, "review.txt")
	if err := os.WriteFile(path, []byte("independent source edit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(t.Context(), req); err == nil {
		t.Fatal("recovery overwrote divergent canonical source")
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "independent source edit\n" {
		t.Fatalf("divergent source was changed: %q, %v", got, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	for path := range contents {
		if err := os.WriteFile(filepath.Join(sb.UpperDir, path), []byte("later overlay bytes"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Empty selections and omitted selections have the same request meaning
	// across JSON/Connect serialization and must retain the same operation.
	req.FileIDs = []uuid.UUID{}
	req.HunkRanges = []types.HunkRange{}
	result, err := svc.Approve(t.Context(), req)
	if err != nil || result == nil || !result.Success || result.Applied != len(contents) || result.AppliedPatchSHA256 != reviewed.PatchSHA256 {
		t.Fatalf("recovery did not apply the retained patch: %+v, %v", result, err)
	}
	for path, expected := range contents {
		actual, err := os.ReadFile(filepath.Join(sb.ScopePath, path))
		if err != nil || string(actual) != expected {
			t.Errorf("recovered %s = %q, %v; want retained %q", path, actual, err, expected)
		}
	}
	if pending, err := env.archiveRepo.GetPreparedApproval(t.Context(), sb.ID); err != nil || pending != nil {
		t.Fatalf("published approval did not consume its pending record: %+v, %v", pending, err)
	}
}

func TestPreparedApprovalMissingEvidenceRefusesRecovery(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed\n"}, nil)
	env.svc.archiveRepo = &lostPreparedResponse{ArchiveRepository: env.archiveRepo}
	req := &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"}
	if _, err := env.svc.Approve(t.Context(), req); err == nil {
		t.Fatal("lost-response fixture unexpectedly succeeded")
	}
	if err := env.blobs.DeleteSandbox(t.Context(), sb.ID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := env.freshService().Approve(t.Context(), req); err == nil {
		t.Fatal("missing retained bytes became successful recovery")
	}
	if _, err := os.Stat(filepath.Join(sb.ScopePath, "review.txt")); !os.IsNotExist(err) {
		t.Fatalf("missing-evidence recovery changed source: %v", err)
	}
	if intent, err := env.archiveRepo.GetPreparedApproval(t.Context(), sb.ID); err != nil || intent == nil {
		t.Fatalf("failed recovery lost its pending identity: %+v, %v", intent, err)
	}
}

func TestApprovalOwnerLockAcrossServiceInstances(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed\n"}, nil)
	entered, proceed := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(proceed) }) }
	defer release()
	env.svc.blobs = &failingBlobs{BlobStore: env.blobs, beforePut: func() {
		close(entered)
		<-proceed
	}}
	req := &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"}
	type response struct {
		result *types.ApprovalResult
		err    error
	}
	finished := make(chan response, 1)
	go func() {
		result, err := env.svc.Approve(t.Context(), req)
		finished <- response{result, err}
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("first approval did not reach archive preparation")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if _, err := env.freshService().Approve(ctx, req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("another service must wait cancellably without stealing approval: %v", err)
	}
	release()
	select {
	case first := <-finished:
		if first.err != nil || first.result == nil || !first.result.Success || first.result.Applied != 1 {
			t.Fatalf("first approval failed: %+v, %v", first.result, first.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("first approval did not release owner lock")
	}
	result, err := env.freshService().Approve(t.Context(), req)
	if err != nil || result == nil || !result.Success || result.Applied != 0 {
		t.Fatalf("later caller must observe the single published approval: %+v, %v", result, err)
	}
}

func TestApprovalPersistenceFailureRetainsAppliedEvidence(t *testing.T) {
	env := newArchiveTestEnv(t)
	content := []byte("reviewed\n")
	sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": string(content)}, nil)
	reviewed, err := env.svc.GetDiff(t.Context(), sb.ID)
	if err != nil {
		t.Fatal(err)
	}
	env.svc.archiveRepo = &failingArchiveInsert{ArchiveRepository: env.archiveRepo}
	result, err := env.svc.Approve(t.Context(), reviewedApproval(t, sb.ID, reviewed.PatchSHA256))
	if err != nil || result == nil || result.Success || result.Applied != 1 || result.AppliedPatchSHA256 != reviewed.PatchSHA256 {
		t.Fatalf("persistence failure must report the applied patch without success: %+v, %v", result, err)
	}
	if !strings.Contains(result.ErrorMsg, "archive insert failed") {
		t.Fatalf("missing persistence failure: %+v", result)
	}
	applied, err := os.ReadFile(filepath.Join(sb.ScopePath, "review.txt"))
	if err != nil || !bytes.Equal(applied, content) {
		t.Fatalf("test must reach source application: %q, %v", applied, err)
	}
	stored, err := env.repo.Get(t.Context(), sb.ID)
	if err != nil || stored.Status != types.StatusActive {
		t.Fatalf("failed transaction published approval: %+v, %v", stored, err)
	}
	archive, err := env.archiveRepo.Get(t.Context(), sb.ID)
	if err != nil || archive != nil {
		t.Fatalf("failed transaction published archive: %+v, %v", archive, err)
	}
	if got := env.blobBytes(sb.ID, reviewed.PatchSHA256); string(got) != reviewed.UnifiedDiff {
		t.Fatalf("persistence failure erased applied patch: %q", got)
	}
	if got := env.blobBytes(sb.ID, hashContent(content)); !bytes.Equal(got, content) {
		t.Fatalf("persistence failure erased applied body: %q", got)
	}
	// A fresh service must recover the original intent, even if the worker's
	// overlay has moved on. It must not apply that new content or require the
	// caller to regenerate a patch against already-modified canonical source.
	if err := os.WriteFile(filepath.Join(sb.UpperDir, "review.txt"), []byte("later unreviewed bytes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	restarted := env.freshService()
	recovered, err := restarted.Approve(t.Context(), reviewedApproval(t, sb.ID, reviewed.PatchSHA256))
	if err != nil || recovered == nil || !recovered.Success || recovered.Applied != 0 || recovered.AppliedPatchSHA256 != reviewed.PatchSHA256 {
		t.Fatalf("restart must complete the original approval without reapplying: %+v, %v", recovered, err)
	}
	archived, err := restarted.GetDiff(t.Context(), sb.ID)
	if err != nil || archived.UnifiedDiff != reviewed.UnifiedDiff {
		t.Fatalf("recovery replaced original evidence: %+v, %v", archived, err)
	}
	if got, err := os.ReadFile(filepath.Join(sb.ScopePath, "review.txt")); err != nil || !bytes.Equal(got, content) {
		t.Fatalf("recovery changed applied source: %q, %v", got, err)
	}
	provenance, err := env.repo.GetFileProvenance(t.Context(), filepath.Join(sb.ScopePath, "review.txt"), sb.ProjectRoot, 10)
	if err != nil || len(provenance) != 1 || provenance[0].ContentDigest != "sha256:"+hashContent(content) {
		t.Fatalf("recovery duplicated or replaced provenance: %+v, %v", provenance, err)
	}
}

func TestApprovalUsesCapturedBytesWhenOverlayChangesDuringBlobWrites(t *testing.T) {
	env := newArchiveTestEnv(t)
	contents := map[string]string{"review.txt": "reviewed\n", "binary.bin": "reviewed\x00binary", "empty.txt": ""}
	sb := env.makeSandbox(types.StatusActive, contents, nil)
	env.svc.blobs = &failingBlobs{BlobStore: env.blobs, beforePut: func() {
		for path := range contents {
			if err := os.WriteFile(filepath.Join(sb.UpperDir, path), []byte("unreviewed later bytes"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}}
	result, err := env.svc.Approve(t.Context(), &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"})
	if err != nil || !result.Success {
		t.Fatalf("approve captured bytes: %+v, %v", result, err)
	}
	for path, want := range contents {
		applied, err := os.ReadFile(filepath.Join(sb.ScopePath, path))
		if err != nil || string(applied) != want {
			t.Errorf("applied %s = %q, %v; want %q", path, applied, err, want)
		}
		archived, err := env.svc.FetchArchiveFile(t.Context(), sb.ID, path)
		if err != nil || string(archived) != want {
			t.Errorf("archived %s = %q, %v; want %q", path, archived, err, want)
		}
	}
}

func TestApprovalRetainsDeletedFileContentFromLiveBaseline(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, nil, []string{"gone.txt"})
	if err := os.MkdirAll(sb.ScopePath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sb.ScopePath, "gone.txt"), []byte("original evidence\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sb.LowerDir = sb.ScopePath
	if err := env.repo.Update(t.Context(), sb); err != nil {
		t.Fatal(err)
	}
	result, err := env.svc.Approve(t.Context(), &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"})
	if err != nil || !result.Success {
		t.Fatalf("delete reviewed file: %+v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(sb.ScopePath, "gone.txt")); !os.IsNotExist(err) {
		t.Fatalf("deletion not applied: %v", err)
	}
	archived, err := env.svc.FetchArchiveFile(t.Context(), sb.ID, "gone.txt")
	if err != nil || string(archived) != "original evidence\n" {
		t.Fatalf("deleted-file evidence lost: %q, %v", archived, err)
	}
}

func TestApprovalCapturePreservesLinksAndRefusesParentEscape(t *testing.T) {
	source, target, outside := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "private"), []byte("private bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(outside, "private")
	if err := os.Symlink(link, filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	content, mode, err := copyApprovalFile(source, target, "link", 0)
	if err != nil || string(content) != link || mode&os.ModeSymlink == 0 {
		t.Fatalf("capture followed a symlink instead of retaining its target: %q, %v, %v", content, mode, err)
	}
	if err := os.Mkdir(filepath.Join(source, "parent"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "parent", "private"), []byte("overwrite"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(target, "parent")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := copyApprovalFile(source, target, "parent/private", 0); err == nil {
		t.Fatal("staging copy escaped through an existing parent symlink")
	}
	got, err := os.ReadFile(filepath.Join(outside, "private"))
	if err != nil || string(got) != "private bytes" {
		t.Fatalf("staging changed outside content: %q, %v", got, err)
	}
}

func TestApprovalReplayCannotAcknowledgeAnotherPatch(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed\n"}, nil)
	reviewed, err := env.svc.GetDiff(t.Context(), sb.ID)
	if err != nil {
		t.Fatal(err)
	}
	request := reviewedApproval(t, sb.ID, hashContent([]byte(reviewed.UnifiedDiff)))
	result, err := env.svc.Approve(t.Context(), request)
	if err != nil || !result.Success || result.AppliedPatchSHA256 != reviewed.PatchSHA256 {
		t.Fatalf("exact reviewed patch should apply: %+v, %v", result, err)
	}
	stored, err := env.repo.Get(t.Context(), sb.ID)
	if err != nil || metadataString(stored.Metadata, metadataApprovedPatchSHA256) != reviewed.PatchSHA256 {
		t.Fatalf("approval identity must survive repository reload: %+v, %v", stored, err)
	}
	replay, err := env.svc.Approve(t.Context(), request)
	if err != nil || !replay.Success || replay.Applied != 0 {
		t.Fatalf("matching terminal replay should not reapply: %+v, %v", replay, err)
	}
	wrong := reviewedApproval(t, sb.ID, hashContent([]byte("another patch")))
	if result, err := env.svc.Approve(t.Context(), wrong); err == nil {
		t.Fatalf("terminal replay acknowledged another patch: %+v", result)
	}
	if err := env.svc.Delete(t.Context(), sb.ID); err != nil {
		t.Fatal(err)
	}
	if replay, err := env.svc.Approve(t.Context(), request); err != nil || !replay.Success || replay.Applied != 0 || replay.AppliedPatchSHA256 != reviewed.PatchSHA256 {
		t.Fatalf("cleanup must preserve approval replay identity: %+v, %v", replay, err)
	}
}

func TestReviewedApprovalRejectsPartialOrUnverifiableRequests(t *testing.T) {
	for _, invalid := range []string{"digest", "files", "hunks", "filtered", "old-approval"} {
		t.Run(invalid, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed\n"}, nil)
			reviewed, err := env.svc.GetDiff(t.Context(), sb.ID)
			if err != nil {
				t.Fatal(err)
			}
			req := reviewedApproval(t, sb.ID, reviewed.PatchSHA256)
			switch invalid {
			case "digest":
				req.ExpectedPatchSHA256 = "not-a-digest"
			case "files":
				req.FileIDs = []uuid.UUID{reviewed.Files[0].ID}
			case "hunks":
				req.Mode = "hunks"
			case "filtered":
				sb.Behavior.Acceptance.Deny.PathGlobs = []string{"src/review.txt"}
			case "old-approval":
				sb.Status = types.StatusApproved
				now := env.svc.clock.Now()
				sb.ApprovedAt = &now
			}
			if err := env.repo.Update(t.Context(), sb); err != nil {
				t.Fatal(err)
			}
			if result, err := env.svc.Approve(t.Context(), req); err == nil {
				t.Fatalf("unverifiable/partial reviewed approval succeeded: %+v", result)
			}
			if _, err := os.Stat(filepath.Join(sb.ScopePath, "review.txt")); !os.IsNotExist(err) {
				t.Fatalf("refused approval changed canonical source: %v", err)
			}
		})
	}
}

func reviewedApproval(t *testing.T, id uuid.UUID, digest string) *types.ApprovalRequest {
	t.Helper()
	var request types.ApprovalRequest
	// Exercise the public request shape, including before the field existed.
	if err := json.Unmarshal([]byte(fmt.Sprintf(`{"sandboxId":%q,"mode":"all","expectedPatchSha256":%q}`, id, digest)), &request); err != nil {
		t.Fatal(err)
	}
	return &request
}

func TestGetDiffUsesRetainedTerminalEvidence(t *testing.T) {
	for _, status := range []types.Status{types.StatusApproved, types.StatusRejected, types.StatusDeleted} {
		t.Run(string(status), func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "reviewed content\n"}, nil)
			ctx := t.Context()
			before, err := env.svc.GetDiff(ctx, sb.ID)
			if err != nil {
				t.Fatal(err)
			}
			switch status {
			case types.StatusApproved:
				result, err := env.svc.Approve(ctx, &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"})
				if err != nil || !result.Success {
					t.Fatalf("approve fixture: %+v, %v", result, err)
				}
			case types.StatusRejected:
				if _, err := env.svc.Reject(ctx, sb.ID, "review-test"); err != nil {
					t.Fatal(err)
				}
			case types.StatusDeleted:
				if err := env.svc.Delete(ctx, sb.ID); err != nil {
					t.Fatal(err)
				}
			}
			// The fake driver retains the overlay. Mutating it distinguishes
			// the immutable archive from an accidental fresh live read.
			if err := os.WriteFile(filepath.Join(sb.UpperDir, "review.txt"), []byte("unreviewed later content\n"), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := env.svc.GetDiff(ctx, sb.ID)
			if err != nil || got == nil || got.ArchiveState != types.ArchiveStateComplete || got.UnifiedDiff != before.UnifiedDiff {
				t.Fatalf("terminal review lost its original evidence: %+v, %v", got, err)
			}
		})
	}
}

func TestGetArchiveDoesNotConcealUnavailableContent(t *testing.T) {
	for _, unavailable := range []string{"missing-blob", "missing-reader"} {
		t.Run(unavailable, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusActive, map[string]string{"review.txt": "review evidence\n"}, nil)
			ctx := t.Context()
			if _, err := env.svc.Reject(ctx, sb.ID, "review-test"); err != nil {
				t.Fatal(err)
			}
			if unavailable == "missing-blob" {
				if err := env.blobs.DeleteSandbox(ctx, sb.ID.String()); err != nil {
					t.Fatal(err)
				}
			} else {
				env.svc.blobs = nil
			}
			if got, err := env.svc.GetArchive(ctx, sb.ID); err == nil || got != nil {
				t.Fatalf("unavailable evidence must not become an empty complete diff: %+v, %v", got, err)
			}
		})
	}
}

func TestGetDiffDistinguishesEmptyFromUncaptured(t *testing.T) {
	for _, status := range []types.Status{types.StatusCreating, types.StatusApproved, types.StatusRejected, types.StatusDeleted} {
		t.Run(string(status), func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(status, nil, nil)
			got, err := env.svc.GetDiff(t.Context(), sb.ID)
			want := types.ArchiveStateNotCaptured
			if status == types.StatusCreating {
				want = ""
			}
			if err != nil || got == nil || got.Files == nil || len(got.Files) != 0 || got.ArchiveState != want {
				t.Fatalf("empty evidence source lost: %+v, %v; want %q", got, err, want)
			}
		})
	}
	t.Run("captured-empty", func(t *testing.T) {
		env := newArchiveTestEnv(t)
		sb := env.makeSandbox(types.StatusActive, nil, nil)
		if _, err := env.svc.Reject(t.Context(), sb.ID, "test"); err != nil {
			t.Fatal(err)
		}
		got, err := env.svc.GetDiff(t.Context(), sb.ID)
		if err != nil || got == nil || got.ArchiveState != types.ArchiveStateComplete || got.UnifiedDiff != "" {
			t.Fatalf("a genuinely captured empty diff must remain valid: %+v, %v", got, err)
		}
	})
}

func TestArchiveCapturesSymlinkTargetNotReferencedBytes(t *testing.T) {
	upper := t.TempDir()
	for _, target := range []string{"../missing", filepath.Join(t.TempDir(), "private")} {
		if filepath.IsAbs(target) {
			if err := os.WriteFile(target, []byte("PRIVATE_TARGET_NOT_EVIDENCE"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		path := filepath.Join(upper, "link")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		got, err := readFileForArchive(&types.Sandbox{UpperDir: upper}, &types.FileChange{FilePath: "link", ChangeType: types.ChangeTypeAdded})
		if err != nil || string(got) != target {
			t.Errorf("archive bytes = %q, %v; want link %q", got, err, target)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
}

func newArchiveTestEnv(t *testing.T) *archiveTestEnv {
	t.Helper()

	tmp := t.TempDir()
	// Pin storage class roots under the test temp dir via the sanctioned
	// override; the user-profile default now resolves under the operator runtime
	// home (~/.vrooli) and no longer honors XDG env vars.
	t.Setenv("VROOLI_STORAGE_ROOT", tmp)

	sqliteDB := db.NewSQLite(t)
	if err := repository.EnsureSchema(context.Background(), sqliteDB, schedule.System()); err != nil {
		t.Fatalf("ensure workspace schema: %v", err)
	}
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

func TestSnapshot_Delete_WithDanglingSymlinkPreservesArchive(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, nil, nil)
	if err := os.Symlink("../controls/boundary.json", filepath.Join(sb.UpperDir, "link")); err != nil {
		t.Fatal(err)
	}
	env.drv.ChangedFiles = []*types.FileChange{{ID: uuid.New(), SandboxID: sb.ID, FilePath: "link", ChangeType: types.ChangeTypeAdded, FileMode: int(os.ModeSymlink | 0777)}}
	ctx := context.Background()
	if err := env.svc.Delete(ctx, sb.ID); err != nil {
		t.Fatalf("delete must archive the link without following it: %v", err)
	}
	got, err := env.svc.FetchArchiveFile(ctx, sb.ID, "link")
	if err != nil || string(got) != "../controls/boundary.json" {
		t.Fatalf("archived link = %q, %v", got, err)
	}
	archived, err := env.archiveRepo.Get(ctx, sb.ID)
	if err != nil || archived == nil || archived.ArchiveState != types.ArchiveStateComplete || len(archived.Files) != 1 || os.FileMode(archived.Files[0].FileMode)&os.ModeSymlink == 0 {
		t.Fatalf("archive did not preserve link metadata: %+v, %v", archived, err)
	}
}

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
	calls     int
	failOn    int
	deleteCb  func(string)
	beforePut func()
}

func (f *failingBlobs) Put(ctx context.Context, sandboxID string, content []byte) (blobstore.PutResult, error) {
	f.calls++
	if f.calls == 1 && f.beforePut != nil {
		f.beforePut()
	}
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

func TestSnapshotPublicationConflictPreservesOriginalBlobs(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"a.txt": "new content\n"}, nil)
	originalBody := []byte("original retained evidence\n")
	blob, err := env.blobs.Put(t.Context(), sb.ID.String(), originalBody)
	if err != nil {
		t.Fatal(err)
	}
	original := newDiffArchive(sb, types.StatusRejected)
	original.ArchiveState = types.ArchiveStateComplete
	original.UnifiedDiffSHA256 = blob.SHA256Hex
	original.TotalBlobBytes = blob.SizeOnDisk
	// Simulate an older/uncoordinated writer publishing between preflight and
	// the transaction. A conflicting capture must not delete its entire tree.
	env.svc.blobs = &failingBlobs{BlobStore: env.blobs, beforePut: func() {
		if err := env.archiveRepo.Insert(t.Context(), nil, original); err != nil {
			t.Fatal(err)
		}
	}}
	if _, err := env.svc.Reject(t.Context(), sb.ID, "retry"); err == nil {
		t.Fatal("conflicting publication must refuse replacement")
	}
	retained, err := env.archiveRepo.Get(t.Context(), sb.ID)
	if err != nil || retained == nil || retained.UnifiedDiffSHA256 != blob.SHA256Hex {
		t.Fatalf("original archive lost: %+v, %v", retained, err)
	}
	content, err := env.blobs.Get(t.Context(), sb.ID.String(), blob.SHA256Hex)
	if err != nil || !bytes.Equal(content, originalBody) {
		t.Fatalf("rollback erased retained content: %q, %v", content, err)
	}
	stored, err := env.repo.Get(t.Context(), sb.ID)
	if err != nil || stored.Status != types.StatusActive {
		t.Fatalf("conflicting publication changed status: %+v, %v", stored, err)
	}
}

func TestApprovalRefusesPreexistingArchiveBeforeSourceEffects(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"a.txt": "unreviewed\n"}, nil)
	original := newDiffArchive(sb, types.StatusRejected)
	if err := env.archiveRepo.Insert(t.Context(), nil, original); err != nil {
		t.Fatal(err)
	}
	if result, err := env.svc.Approve(t.Context(), &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"}); err == nil {
		t.Fatalf("inconsistent published evidence must refuse before apply: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(sb.ScopePath, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("source modified despite published evidence: %v", err)
	}
}

func TestSnapshotCleanupPreservesUncertainOrPendingEvidence(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprintf("pending=%t", pending), func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusActive, nil, nil)
			content := []byte("retained recovery evidence")
			blob, err := env.blobs.Put(t.Context(), sb.ID.String(), content)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if pending {
				archive := newDiffArchive(sb, types.StatusApproved)
				archive.ArchiveState = types.ArchiveStateComplete
				archive.UnifiedDiffSHA256 = blob.SHA256Hex
				intent := &types.PreparedApproval{Archive: *archive, Request: types.ApprovalRequest{SandboxID: sb.ID, Mode: "all"}, ScopePath: sb.ScopePath}
				if err := env.archiveRepo.PutPreparedApproval(ctx, intent); err != nil {
					t.Fatal(err)
				}
			} else {
				cancel() // ownership read fails; absent evidence was not established
			}
			release, err := env.blobs.LockReview(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			env.svc.cleanupUnpublishedBlobs(ctx, sb.ID)
			got, err := env.blobs.Get(t.Context(), sb.ID.String(), blob.SHA256Hex)
			if err != nil || !bytes.Equal(got, content) {
				t.Fatalf("cleanup erased pinned or uncertain evidence: %q, %v", got, err)
			}
		})
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
	if err := repository.EnsureSchema(context.Background(), sqliteDB, schedule.System()); err != nil {
		t.Fatalf("ensure workspace schema: %v", err)
	}
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

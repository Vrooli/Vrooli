package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"
)

type reviewDrainer func(context.Context, uuid.UUID) error

func (f reviewDrainer) Drain(ctx context.Context, id uuid.UUID) error { return f(ctx, id) }

func TestReviewAdmissionFencePrecedesStopAndCapture(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusActive, map[string]string{"a": "reviewed"}, nil)
	calls := 0
	env.svc.processDrainer = reviewDrainer(func(context.Context, uuid.UUID) error { calls++; return nil })
	_, release, err := env.svc.BeginProcess(t.Context(), sb.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := env.svc.Stop(ctx, sb.ID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stop bypassed process registration fence: %v", err)
	}
	if calls != 0 {
		t.Fatal("drain ran before process registration completed")
	}
	release()
	if _, err := env.svc.Stop(t.Context(), sb.ID); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("stop did not drain processes")
	}
	req := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
	if _, err := env.svc.CaptureReviewSnapshot(t.Context(), req); err != nil {
		t.Fatal(err)
	}
}

func TestStopRefusesUnverifiedDrain(t *testing.T) {
	env := newArchiveTestEnv(t)
	env.drv.Mounted = true
	sb := env.makeSandbox(types.StatusActive, nil, nil)
	env.svc.processDrainer = reviewDrainer(func(context.Context, uuid.UUID) error { return errors.New("exit unavailable") })
	if _, err := env.svc.Stop(t.Context(), sb.ID); err == nil {
		t.Fatal("failed drain became stopped success")
	}
	got, err := env.svc.Get(t.Context(), sb.ID)
	if err != nil || got.Status != types.StatusActive {
		t.Fatalf("failed drain changed lifecycle state: %+v, %v", got, err)
	}
	if !env.drv.Mounted {
		t.Fatal("failed drain unmounted a live process workspace")
	}
}

func TestReviewSnapshotRetainsCompleteSelectedInputAcrossResume(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusStopped, map[string]string{"new.txt": "new review\n", "binary": "\x00\x01binary", "empty": "", "script": "#!/bin/sh\n"}, []string{"deleted.txt"})
	if err := os.WriteFile(filepath.Join(sb.LowerDir, "context.txt"), []byte("unchanged context\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(sb.UpperDir, "script"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/outside/missing", filepath.Join(sb.LowerDir, "link")); err != nil {
		t.Fatal(err)
	}
	req := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{".", "new.txt"}}
	snapshot, err := env.svc.CaptureReviewSnapshot(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SHA256 == "" || snapshot.SHA256 != snapshot.ContentSHA256() || len(snapshot.Paths) != 1 {
		t.Fatalf("invalid immutable manifest: %+v", snapshot)
	}
	for _, check := range []struct{ side, path, body string }{
		{"before", "context.txt", "unchanged context\n"}, {"after", "context.txt", "unchanged context\n"},
		{"after", "new.txt", "new review\n"}, {"after", "binary", "\x00\x01binary"}, {"after", "empty", ""},
		{"before", "link", "/outside/missing"}, {"after", "link", "/outside/missing"},
	} {
		got, err := env.svc.FetchReviewFile(t.Context(), sb.ID, req.RequestID, check.side, check.path)
		if err != nil || string(got) != check.body {
			t.Fatalf("%s/%s: %q, %v", check.side, check.path, got, err)
		}
	}
	for _, file := range snapshot.After {
		if file.Path == "script" && file.Mode != 100755 {
			t.Fatalf("executable mode lost: %+v", file)
		}
		if file.Path == "deleted.txt" {
			t.Fatal("deleted file exists in candidate")
		}
		if file.Path == "link" && file.Mode != 120000 {
			t.Fatalf("symlink identity lost: %+v", file)
		}
	}
	patch, err := env.svc.FetchReviewFile(t.Context(), sb.ID, req.RequestID, "patch", "")
	if err != nil || !bytes.Contains(patch, []byte("new review")) {
		t.Fatalf("patch missing: %q, %v", patch, err)
	}
	workspace, err := env.svc.MaterializeReviewSnapshot(t.Context(), sb.ID, req.RequestID, snapshot.SHA256)
	if err != nil || workspace.SHA256 != snapshot.SHA256 {
		t.Fatalf("materialize: %+v, %v", workspace, err)
	}
	for _, side := range []struct {
		name  string
		files []types.ReviewFile
	}{{"before", snapshot.Before}, {"after", snapshot.After}} {
		for _, file := range side.files {
			path := filepath.Join(workspace.Root, side.name, file.Path)
			body, err := env.svc.FetchReviewFile(t.Context(), sb.ID, req.RequestID, side.name, file.Path)
			if err != nil {
				t.Fatal(err)
			}
			if file.Mode == 120000 {
				link, err := os.Readlink(path)
				if err != nil || link != string(body) {
					t.Fatalf("link %s: %q, %v", path, link, err)
				}
			} else {
				got, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(body, got) {
					t.Fatalf("body %s: %q, %v", path, got, err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				want := os.FileMode(0444)
				if file.Mode == 100755 {
					want = 0555
				}
				if info.Mode().Perm() != want {
					t.Fatalf("mode %s: %v", path, info.Mode())
				}
			}
		}
	}
	if _, err := os.Lstat(filepath.Join(workspace.Root, "after", "deleted.txt")); !os.IsNotExist(err) {
		t.Fatalf("deleted candidate file exists: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sb.LowerDir, "context.txt"), []byte("later baseline"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sb.UpperDir, "new.txt"), []byte("later worker edit"), 0644); err != nil {
		t.Fatal(err)
	}
	fresh := env.freshService()
	got, err := fresh.CaptureReviewSnapshot(t.Context(), req)
	if err != nil || !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("resume replaced original input: %+v, %v", got, err)
	}
	other := *req
	other.Paths = []string{"new.txt"}
	if _, err := fresh.CaptureReviewSnapshot(t.Context(), &other); err == nil {
		t.Fatal("changed selection reused request identity")
	}
	if err := fresh.Delete(t.Context(), sb.ID); err != nil {
		t.Fatal(err)
	}
	got, err = fresh.GetReviewSnapshot(t.Context(), sb.ID, req.RequestID)
	if err != nil || !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("sandbox teardown erased review: %+v, %v", got, err)
	}
	body, err := fresh.FetchReviewFile(t.Context(), sb.ID, req.RequestID, "after", "new.txt")
	if err != nil || string(body) != "new review\n" {
		t.Fatalf("teardown lost captured body: %q, %v", body, err)
	}
	replayed, err := fresh.MaterializeReviewSnapshot(t.Context(), sb.ID, req.RequestID, snapshot.SHA256)
	if err != nil || *replayed != *workspace {
		t.Fatalf("materialization replay changed: %+v, %v", replayed, err)
	}
	for _, digest := range []string{"", strings.Repeat("0", 64)} {
		if _, err := fresh.MaterializeReviewSnapshot(t.Context(), sb.ID, req.RequestID, digest); err == nil {
			t.Fatal("materialized without exact expected digest")
		}
	}
}

func TestReviewWorkspaceRefusesTamperingAndMissingEvidence(t *testing.T) {
	for _, kind := range []string{"content", "mode", "symlink", "extra-file", "extra-directory", "missing-directory", "missing-blob"} {
		t.Run(kind, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusStopped, map[string]string{"candidate": "retained"}, nil)
			req := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
			snapshot, err := env.svc.CaptureReviewSnapshot(t.Context(), req)
			if err != nil {
				t.Fatal(err)
			}
			workspace, err := env.svc.MaterializeReviewSnapshot(t.Context(), sb.ID, req.RequestID, snapshot.SHA256)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(workspace.Root, "after", "candidate")
			switch kind {
			case "content":
				if err = os.Chmod(path, 0644); err == nil {
					err = os.WriteFile(path, []byte("unreviewed"), 0644)
				}
				if err == nil {
					err = os.Chmod(path, 0444)
				}
			case "mode":
				err = os.Chmod(path, 0644)
			case "symlink":
				if err = os.Remove(path); err == nil {
					err = os.Symlink(filepath.Join(sb.UpperDir, "candidate"), path)
				}
			case "extra-file":
				err = os.WriteFile(filepath.Join(workspace.Root, "unexpected"), []byte("extra"), 0444)
			case "extra-directory":
				err = os.Mkdir(filepath.Join(workspace.Root, "unexpected"), 0700)
			case "missing-directory":
				err = os.Remove(filepath.Join(workspace.Root, "before"))
			case "missing-blob":
				namespace := filepath.Dir(workspace.Root)
				err = os.Remove(filepath.Join(namespace, snapshot.PatchSHA256+".gz"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.freshService().MaterializeReviewSnapshot(t.Context(), sb.ID, req.RequestID, snapshot.SHA256); err == nil {
				t.Fatal("accepted changed derived tree or missing authoritative evidence")
			}
		})
	}
}

type lostReviewResponse struct{ repository.ArchiveRepository }

func (r lostReviewResponse) PutReviewSnapshot(ctx context.Context, snapshot *types.ReviewSnapshot) error {
	if err := r.ArchiveRepository.PutReviewSnapshot(ctx, snapshot); err != nil {
		return err
	}
	return errors.New("lost review publication response")
}

func TestReviewSnapshotLostResponsePreservesEvidence(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusStopped, map[string]string{"review.txt": "original"}, nil)
	env.svc.archiveRepo = lostReviewResponse{env.archiveRepo}
	req := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
	if _, err := env.svc.CaptureReviewSnapshot(t.Context(), req); err == nil {
		t.Fatal("fixture must lose response")
	}
	if err := os.WriteFile(filepath.Join(sb.UpperDir, "review.txt"), []byte("later"), 0644); err != nil {
		t.Fatal(err)
	}
	fresh := env.freshService()
	if _, err := fresh.CaptureReviewSnapshot(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	body, err := fresh.FetchReviewFile(t.Context(), sb.ID, req.RequestID, "after", "review.txt")
	if err != nil || string(body) != "original" {
		t.Fatalf("lost response erased original evidence: %q, %v", body, err)
	}
}

func TestReviewSnapshotRefusesDriftAndIncompleteSelection(t *testing.T) {
	for _, mutation := range []string{"selection", "baseline", "candidate", "active", "blob-failure"} {
		t.Run(mutation, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusStopped, map[string]string{"review.txt": "original"}, nil)
			req := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
			switch mutation {
			case "selection":
				req.Paths = []string{"unrelated"}
			case "active":
				sb.Status = types.StatusActive
				if err := env.repo.Update(t.Context(), sb); err != nil {
					t.Fatal(err)
				}
			case "blob-failure":
				env.svc.blobs = &failingBlobs{BlobStore: env.blobs, failOn: 2}
			default:
				env.svc.blobs = &failingBlobs{BlobStore: env.blobs, beforePut: func() {
					root, path := sb.UpperDir, "review.txt"
					if mutation == "baseline" {
						root, path = sb.LowerDir, "new-context"
					}
					if err := os.WriteFile(filepath.Join(root, path), []byte("raced"), 0644); err != nil {
						t.Fatal(err)
					}
				}}
			}
			if _, err := env.svc.CaptureReviewSnapshot(t.Context(), req); err == nil {
				t.Fatal("incomplete or raced review input was published")
			}
			if got, err := env.archiveRepo.GetReviewSnapshot(t.Context(), sb.ID, req.RequestID); err != nil || got != nil {
				t.Fatalf("failed capture published: %+v, %v", got, err)
			}
		})
	}
}

func TestReviewSnapshotApprovalBindsContextAndRetainedEvidence(t *testing.T) {
	for _, mutation := range []string{"none", "context", "candidate", "missing-evidence", "wrong-digest", "no-review-id", "resumed"} {
		t.Run(mutation, func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusStopped, map[string]string{"review.txt": "reviewed\n"}, nil)
			// A real canonical scope, separate from the retained baseline layer.
			if err := os.MkdirAll(sb.ScopePath, 0755); err != nil {
				t.Fatal(err)
			}
			for _, root := range []string{sb.ScopePath, sb.LowerDir} {
				if err := os.WriteFile(filepath.Join(root, "context.txt"), []byte("context\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			capture := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
			snapshot, err := env.svc.CaptureReviewSnapshot(t.Context(), capture)
			if err != nil {
				t.Fatal(err)
			}
			req := &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all", Force: true, Actor: "independent-owner", ReviewRequestID: capture.RequestID, ExpectedReviewSHA256: snapshot.SHA256}
			switch mutation {
			case "context":
				err = os.WriteFile(filepath.Join(sb.ScopePath, "context.txt"), []byte("changed without changing patch\n"), 0644)
			case "candidate":
				err = os.WriteFile(filepath.Join(sb.UpperDir, "review.txt"), []byte("not reviewed\n"), 0644)
			case "missing-evidence":
				err = env.blobs.DeleteSandbox(t.Context(), snapshot.ID.String())
			case "wrong-digest":
				req.ExpectedReviewSHA256 = strings.Repeat("0", 64)
			case "no-review-id":
				req.ReviewRequestID = uuid.Nil
			case "resumed":
				sb.Status = types.StatusActive
				err = env.repo.Update(t.Context(), sb)
			}
			if err != nil {
				t.Fatal(err)
			}
			result, err := env.freshService().Approve(t.Context(), req)
			if mutation != "none" {
				if err == nil {
					t.Fatalf("approval accepted %s: %+v", mutation, result)
				}
				if _, err := os.Stat(filepath.Join(sb.ScopePath, "review.txt")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("refusal mutated canonical source: %v", err)
				}
				return
			}
			if err != nil || result == nil || !result.Success || result.AppliedPatchSHA256 != snapshot.PatchSHA256 {
				t.Fatalf("review-bound approval failed: %+v, %v", result, err)
			}
			body, err := os.ReadFile(filepath.Join(sb.ScopePath, "review.txt"))
			if err != nil || string(body) != "reviewed\n" {
				t.Fatalf("reviewed source not applied: %q, %v", body, err)
			}
			if req.ExpectedPatchSHA256 != "" {
				t.Fatal("approval mutated caller request")
			}
			if _, err := env.freshService().Approve(t.Context(), req); err != nil {
				t.Fatalf("matching terminal replay failed: %v", err)
			}
		})
	}
}

func TestReviewSnapshotRejectsOversizedFileBeforePublication(t *testing.T) {
	env := newArchiveTestEnv(t)
	sb := env.makeSandbox(types.StatusStopped, map[string]string{"large": ""}, nil)
	if err := os.Truncate(filepath.Join(sb.UpperDir, "large"), types.MaxReviewFileBytes+1); err != nil {
		t.Fatal(err)
	}
	req := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
	if _, err := env.svc.CaptureReviewSnapshot(t.Context(), req); err == nil {
		t.Fatal("oversized capture admitted")
	}
	if got, err := env.archiveRepo.GetReviewSnapshot(t.Context(), sb.ID, req.RequestID); err != nil || got != nil {
		t.Fatalf("published oversized snapshot: %+v, %v", got, err)
	}
}

func TestReviewCaptureBoundsStagingAndPatch(t *testing.T) {
	for _, limit := range []int64{3, 8} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusStopped, map[string]string{"a": "1234", "b": "5678"}, nil)
			capture, err := env.svc.captureApprovalDiff(t.Context(), sb, env.drv.ChangedFiles, 4, limit)
			if capture != nil {
				capture.close()
			}
			if err == nil {
				t.Fatal("staging/patch exceeded total limit")
			}
		})
	}
}

func TestReviewApprovalRecoveryKeepsContextAndRequestBinding(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(fmt.Sprint(applied), func(t *testing.T) {
			env := newArchiveTestEnv(t)
			sb := env.makeSandbox(types.StatusStopped, map[string]string{"review.txt": "reviewed\n"}, nil)
			if err := os.MkdirAll(sb.ScopePath, 0755); err != nil {
				t.Fatal(err)
			}
			for _, root := range []string{sb.ScopePath, sb.LowerDir} {
				if err := os.WriteFile(filepath.Join(root, "context"), []byte("original"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			capture := &types.ReviewSnapshotRequest{SandboxID: sb.ID, RequestID: uuid.New(), Paths: []string{"."}}
			snapshot, err := env.svc.CaptureReviewSnapshot(t.Context(), capture)
			if err != nil {
				t.Fatal(err)
			}
			req := &types.ApprovalRequest{SandboxID: sb.ID, Mode: "all", Force: true, Actor: "review-owner", ReviewRequestID: capture.RequestID, ExpectedReviewSHA256: snapshot.SHA256}
			env.svc.archiveRepo = &lostPreparedResponse{ArchiveRepository: env.archiveRepo}
			if _, err := env.svc.Approve(t.Context(), req); err == nil || !strings.Contains(err.Error(), "prepared response lost") {
				t.Fatalf("fixture did not lose prepared response: %v", err)
			}
			if applied {
				if err := os.WriteFile(filepath.Join(sb.ScopePath, "review.txt"), []byte("reviewed\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(sb.ScopePath, "context"), []byte("later context"), 0644); err != nil {
				t.Fatal(err)
			}
			fresh := env.freshService()
			if _, err := fresh.Approve(t.Context(), req); err == nil {
				t.Fatal("recovery ignored unchanged-context drift")
			}
			if err := os.WriteFile(filepath.Join(sb.ScopePath, "context"), []byte("original"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(sb.UpperDir, "review.txt"), []byte("later worker bytes"), 0644); err != nil {
				t.Fatal(err)
			}
			result, err := fresh.Approve(t.Context(), req)
			if err != nil || !result.Success || result.AppliedPatchSHA256 != snapshot.PatchSHA256 {
				t.Fatalf("recovery lost retained review: %+v, %v", result, err)
			}
			body, err := os.ReadFile(filepath.Join(sb.ScopePath, "review.txt"))
			if err != nil || string(body) != "reviewed\n" {
				t.Fatalf("recovery applied different candidate: %q, %v", body, err)
			}
		})
	}
}

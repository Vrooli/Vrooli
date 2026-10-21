package jobs_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"device-studio/internal/jobs"
	_ "modernc.org/sqlite"
)

var baseTime = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

func openRepo(t *testing.T, path string) (*sql.DB, jobs.Repository) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	if _, err = db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, jobs.NewSQLiteRepository(db)
}

func newService(repo jobs.Repository) jobs.Service {
	return jobs.NewService(repo, func() time.Time { return baseTime })
}

func input() jobs.Input {
	return jobs.Input{
		ProjectID: "project-a", ShotRevisionID: "shot-rev-7", ShotRevision: 7,
		Bindings:          []jobs.ArtifactBinding{{Role: "plate", ArtifactID: "artifact-plate", RevisionID: "plate-r3", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, {Role: "screen", ArtifactID: "artifact-screen", RevisionID: "screen-r2", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		SourceClockDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", ToolProfileDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", RendererVersion: "compositor-v1", OutputProfile: "png-1536-srgb",
	}
}

func submit(t *testing.T, s jobs.Service) jobs.Job {
	t.Helper()
	j, created, err := s.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "op-1", Input: input()})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected created job")
	}
	return j
}

func executionOwner(scope, attemptID string, attemptNumber int) jobs.ExecutionClaimInput {
	return jobs.ExecutionClaimInput{AttemptID: attemptID, AttemptNumber: attemptNumber, OwnerID: "platform-go", ScopeID: scope, ArtifactID: "execution-claim-" + scope, SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}
}

func proofFor(j jobs.Job, a jobs.Attempt, reason string) jobs.QuiescenceProof {
	return jobs.QuiescenceProof{JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ExecutionClaimID: a.Execution.ID, OwnerID: a.Execution.OwnerID, ScopeID: a.Execution.ScopeID, ProofArtifactID: "quiescence-" + a.ID, ProofSHA256: "1313131313131313131313131313131313131313131313131313131313131313", ObservedAt: baseTime, NoOwnedExecution: true, Reason: reason}
}

func outputs() jobs.CompletionEvidence {
	return jobs.CompletionEvidence{Outputs: []jobs.OutputRef{{Kind: "png", ArtifactID: "artifact-output", SHA256: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", MIMEType: "image/png", SizeBytes: 4096}}, ReceiptArtifactID: "render-receipt", ReceiptSHA256: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", CompletedAt: baseTime}
}

func TestSubmitIsDurableAndIdempotentAcrossRepositoryReopen(t *testing.T) { // [REQ:DS-JOB-DURABILITY]
	path := filepath.Join(t.TempDir(), "jobs.db")
	db, repo := openRepo(t, path)
	svc := newService(repo)
	first := submit(t, svc)
	_ = db.Close()
	db2, repo2 := openRepo(t, path)
	defer db2.Close()
	svc2 := newService(repo2)
	retry, created, err := svc2.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "op-1", Input: input()})
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("same immutable request must reuse job")
	}
	if retry.ID != first.ID || retry.InputFingerprint != first.InputFingerprint || len(retry.Attempts) != 1 {
		t.Fatalf("reopened job differs: %#v", retry)
	}
	byKey, err := svc2.FindByIdempotencyKey(context.Background(), "op-1")
	if err != nil || byKey.ID != first.ID || byKey.InputFingerprint != first.InputFingerprint {
		t.Fatalf("lookup by durable idempotency key: job=%+v err=%v", byKey, err)
	}
	if _, err := svc2.FindByIdempotencyKey(context.Background(), "missing"); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("missing idempotency key should be not found, got %v", err)
	}
	changed := input()
	changed.Bindings[0].SHA256 = "1111111111111111111111111111111111111111111111111111111111111111"
	if _, _, err = svc2.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "op-1", Input: changed}); !errors.Is(err, jobs.ErrIdempotencyConflict) {
		t.Fatalf("changed payload with same key: %v", err)
	}
}

func TestProgressPagesAndReleaseReceiptSurviveReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.db")
	db, repo := openRepo(t, path)
	svc := newService(repo)
	j := submit(t, svc)
	running, err := svc.Start(context.Background(), j.ID, executionOwner("scope-release", j.Attempts[0].ID, 1), j.Version)
	if err != nil {
		t.Fatal(err)
	}
	progress, err := svc.ListProgress(context.Background(), jobs.Cursor{}, 10)
	if err != nil || len(progress) != 1 || progress[0].ID != j.ID {
		t.Fatalf("claimed job missing from progress: rows=%#v err=%v", progress, err)
	}
	observed, err := svc.RecordObserved(context.Background(), j.ID, running.Attempts[0].ID, running.Version)
	if err != nil || !observed.Attempts[0].ExecutionObserved {
		t.Fatalf("explicit observation not persisted: job=%#v err=%v", observed, err)
	}
	completion := outputs()
	completion.AttemptID = observed.Attempts[0].ID
	completion.ExecutionClaimID = observed.Attempts[0].Execution.ID
	completion.InputFingerprint = observed.InputFingerprint
	terminal, err := svc.Complete(context.Background(), j.ID, observed.Attempts[0].ID, completion, observed.Version)
	if err != nil {
		t.Fatal(err)
	}
	if rows, err := svc.ListProgress(context.Background(), jobs.Cursor{}, 10); err != nil || len(rows) != 1 || rows[0].State != jobs.Succeeded {
		t.Fatalf("terminal release-pending job was not discoverable: rows=%#v err=%v", rows, err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db2, repo2 := openRepo(t, path)
	defer db2.Close()
	svc2 := newService(repo2)
	rows, err := svc2.ListProgress(context.Background(), jobs.Cursor{}, 10)
	if err != nil || len(rows) != 1 || rows[0].ID != terminal.ID {
		t.Fatalf("release work did not survive repository reopen: rows=%#v err=%v", rows, err)
	}
	released, err := svc2.RecordReleased(context.Background(), terminal.ID, terminal.Attempts[0].ID, terminal.Version, baseTime.Add(time.Second))
	if err != nil || released.Attempts[0].ExecutionReleasedAt == nil {
		t.Fatalf("release receipt was not persisted: job=%#v err=%v", released, err)
	}
	rows, err = svc2.ListProgress(context.Background(), jobs.Cursor{}, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("released terminal job remains in progress: rows=%#v err=%v", rows, err)
	}
	var count int
	if err = db2.QueryRow(`SELECT COUNT(*) FROM render_job_release_receipts WHERE job_id=? AND attempt_id=?`, j.ID, running.Attempts[0].ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("durable release receipt count=%d err=%v", count, err)
	}
}

func TestSuccessRequiresValidatedWorkerEvidenceAndPersistsAttempt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.db")
	db, repo := openRepo(t, path)
	svc := newService(repo)
	j := submit(t, svc)
	running, err := svc.Start(context.Background(), j.ID, executionOwner("scope-1", j.Attempts[0].ID, 1), j.Version)
	if err != nil {
		t.Fatal(err)
	}
	bad := outputs()
	bad.InputFingerprint = "wrong"
	if _, err = svc.Complete(context.Background(), j.ID, running.Attempts[0].ID, bad, running.Version); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("mismatched worker result accepted: %v", err)
	}
	good := outputs()
	good.AttemptID = running.Attempts[0].ID
	good.ExecutionClaimID = running.Attempts[0].Execution.ID
	good.InputFingerprint = running.InputFingerprint
	completed, err := svc.Complete(context.Background(), j.ID, running.Attempts[0].ID, good, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != jobs.Succeeded || completed.Attempts[0].State != jobs.AttemptSucceeded {
		t.Fatalf("completion not terminal: %#v", completed)
	}
	_ = db.Close()
	db2, repo2 := openRepo(t, path)
	defer db2.Close()
	svc2 := newService(repo2)
	got, err := repo2.Get(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != jobs.Succeeded || got.InputFingerprint != j.InputFingerprint || got.Input.SourceClockDigest != j.Input.SourceClockDigest ||
		got.Attempts[0].Completion == nil || len(got.Attempts[0].Completion.Outputs) != 1 ||
		got.Attempts[0].Completion.Outputs[0].ArtifactID != good.Outputs[0].ArtifactID || got.Attempts[0].Completion.Outputs[0].SHA256 != good.Outputs[0].SHA256 {
		t.Fatalf("completion receipt did not survive reopen: %#v", got)
	}
	replayed, created, err := svc2.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "op-1", Input: j.Input})
	if err != nil || created || replayed.ID != j.ID || replayed.State != jobs.Succeeded || replayed.Attempts[0].Completion == nil {
		t.Fatalf("same render request did not replay its accepted result after reopen: created=%v job=%+v err=%v", created, replayed, err)
	}
	rerender, created, err := svc2.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "op-1-explicit-rerender", Input: j.Input})
	if err != nil || !created || rerender.ID == j.ID || rerender.State != jobs.Queued || rerender.InputFingerprint != j.InputFingerprint || rerender.Input.SourceClockDigest != j.Input.SourceClockDigest || rerender.Attempts[0].Completion != nil {
		t.Fatalf("explicit rerender should create a fresh job over the same immutable source clock: created=%v job=%+v err=%v", created, rerender, err)
	}
	stillAccepted, err := svc2.Get(context.Background(), j.ID)
	if err != nil || stillAccepted.State != jobs.Succeeded || stillAccepted.Attempts[0].Completion == nil || stillAccepted.Attempts[0].Completion.Outputs[0].SHA256 != good.Outputs[0].SHA256 {
		t.Fatalf("new rerender changed the previous accepted output: job=%+v err=%v", stillAccepted, err)
	}
	if _, err = svc2.Complete(context.Background(), j.ID, running.Attempts[0].ID, good, got.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("terminal job accepted second completion: %v", err)
	}
}

func TestCancelAndSuccessHaveOneSerializedWinner(t *testing.T) { // [REQ:DS-JOB-CANCEL]
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "race.db"))
	svc := newService(repo)
	j := submit(t, svc)
	running, err := svc.Start(context.Background(), j.ID, executionOwner("scope-1", j.Attempts[0].ID, 1), j.Version)
	if err != nil {
		t.Fatal(err)
	}
	good := outputs()
	good.AttemptID = running.Attempts[0].ID
	good.ExecutionClaimID = running.Attempts[0].Execution.ID
	good.InputFingerprint = running.InputFingerprint
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	var completed jobs.Job
	var completeErr, cancelErr error
	go func() {
		defer wg.Done()
		<-start
		completed, completeErr = svc.Complete(context.Background(), j.ID, running.Attempts[0].ID, good, running.Version)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, cancelErr = svc.RequestCancel(context.Background(), j.ID, running.Version)
	}()
	close(start)
	wg.Wait()
	final, err := svc.Get(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	switch final.State {
	case jobs.Succeeded:
		if completeErr != nil {
			t.Fatalf("success winner reported error: %v", completeErr)
		}
		if cancelErr == nil {
			t.Fatal("cancel after success must report terminal/conflict")
		}
	case jobs.CancelRequested:
		if cancelErr != nil {
			t.Fatalf("cancel winner reported error: %v", cancelErr)
		}
		if completeErr == nil {
			t.Fatal("late success must not replace cancellation intent")
		}
		if _, err = svc.Complete(context.Background(), j.ID, running.Attempts[0].ID, good, final.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
			t.Fatalf("late success promoted cancel-requested job: %v", err)
		}
	default:
		t.Fatalf("race produced incoherent state: %s (complete=%v cancel=%v result=%#v)", final.State, completeErr, cancelErr, completed)
	}
}

func TestCheckpointIdentityAndRecoveryClassification(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "checkpoint.db"))
	svc := newService(repo)
	j := submit(t, svc)
	running, err := svc.Start(context.Background(), j.ID, executionOwner("scope-1", j.Attempts[0].ID, 1), j.Version)
	if err != nil {
		t.Fatal(err)
	}
	a := running.Attempts[0]
	cp := jobs.Checkpoint{AttemptID: a.ID, AttemptNumber: a.Number, ExecutionClaimID: a.Execution.ID, InputFingerprint: running.InputFingerprint, ToolProfileDigest: running.Input.ToolProfileDigest, SourceClockDigest: running.Input.SourceClockDigest, Unit: 12, ArtifactID: "checkpoint", SHA256: "9999999999999999999999999999999999999999999999999999999999999999", CreatedAt: baseTime}
	bad := cp
	bad.InputFingerprint = "other"
	if _, err = svc.SaveCheckpoint(context.Background(), j.ID, a.ID, bad, proofFor(running, a, "checkpoint attempt"), running.Version); !errors.Is(err, jobs.ErrCheckpointMismatch) {
		t.Fatalf("mismatched checkpoint accepted: %v", err)
	}
	proof := proofFor(running, a, "checkpoint owner reports execution stopped")
	forged := proof
	forged.ExecutionClaimID = "unrelated-claim"
	if _, err = svc.SaveCheckpoint(context.Background(), j.ID, a.ID, cp, forged, running.Version); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("checkpoint with another scope proof accepted: %v", err)
	}
	stillRunning, err := svc.Get(context.Background(), j.ID)
	if err != nil || stillRunning.State != jobs.Running {
		t.Fatalf("invalid proof changed state: state=%s err=%v", stillRunning.State, err)
	}
	checkpointed, err := svc.SaveCheckpoint(context.Background(), j.ID, a.ID, cp, proof, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	recs, err := svc.Recoveries(context.Background(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Class != jobs.RecoveryResume || recs[0].Checkpoint == nil {
		t.Fatalf("checkpoint recovery not classified: %#v", recs)
	}
	if _, err = svc.Start(context.Background(), j.ID, executionOwner("scope-2", "attempt-resumed-2", 2), checkpointed.Version); !errors.Is(err, jobs.ErrReleasePending) {
		t.Fatalf("new attempt admitted before prior execution release: %v", err)
	}
	releasedCheckpoint, err := svc.RecordReleased(context.Background(), j.ID, a.ID, checkpointed.Version, baseTime.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := svc.Start(context.Background(), j.ID, executionOwner("scope-2", "attempt-resumed-2", 2), releasedCheckpoint.Version)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State != jobs.Running || len(resumed.Attempts) != 2 || resumed.Attempts[1].ID == a.ID || resumed.Attempts[1].ResumeFrom == nil || resumed.Attempts[1].Execution.ScopeID == a.Execution.ScopeID {
		t.Fatalf("resume lost checkpoint identity: %#v", resumed)
	}
	stale := outputs()
	stale.AttemptID = a.ID
	stale.ExecutionClaimID = a.Execution.ID
	stale.InputFingerprint = running.InputFingerprint
	if _, err = svc.Complete(context.Background(), j.ID, a.ID, stale, resumed.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("prior checkpoint attempt completed over new attempt: %v", err)
	}
	latest, err := svc.Get(context.Background(), j.ID)
	if err != nil || latest.State != jobs.Running || len(latest.Attempts) != 2 {
		t.Fatalf("stale result mutated resumed job: state=%s err=%v", latest.State, err)
	}
	current := resumed.Attempts[1]
	cp2 := jobs.Checkpoint{AttemptID: current.ID, AttemptNumber: current.Number, ExecutionClaimID: current.Execution.ID, InputFingerprint: resumed.InputFingerprint, ToolProfileDigest: resumed.Input.ToolProfileDigest, SourceClockDigest: resumed.Input.SourceClockDigest, Unit: 13, ArtifactID: "checkpoint-13", SHA256: "8888888888888888888888888888888888888888888888888888888888888888", CreatedAt: baseTime}
	paused, err := svc.SaveCheckpoint(context.Background(), j.ID, current.ID, cp2, proofFor(resumed, current, "second worker scope stopped"), resumed.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Start(context.Background(), j.ID, executionOwner("scope-1", "attempt-resumed-3", 3), paused.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("historical execution scope reused: %v", err)
	}
	if _, err = svc.Start(context.Background(), j.ID, executionOwner("scope-3", a.ID, 3), paused.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("older attempt ID reused as a new attempt: %v", err)
	}
}

func TestCancellationRequiresQuiescenceAndSettlementIsOrthogonal(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "cancel.db"))
	svc := newService(repo)
	in := input()
	in.AccountingRequired = true
	in.AccountingOperationID = "operation-55"
	j, created, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "paid-op", Input: in})
	if err != nil || !created {
		t.Fatalf("submit: created=%v err=%v", created, err)
	}
	running, err := svc.Start(context.Background(), j.ID, executionOwner("scope-1", j.Attempts[0].ID, 1), j.Version)
	if err != nil {
		t.Fatal(err)
	}
	requested, err := svc.RequestCancel(context.Background(), j.ID, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	if requested.State != jobs.CancelRequested {
		t.Fatalf("state=%s", requested.State)
	}
	proof := proofFor(running, running.Attempts[0], "owner reported execution quiescent")
	wrong := proof
	wrong.ScopeID = "unrelated-scope"
	if _, err = svc.ConfirmCancelled(context.Background(), j.ID, running.Attempts[0].ID, wrong, requested.Version); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("unrelated owner scope accepted: %v", err)
	}
	wrong = proof
	wrong.ExecutionClaimID = "other-claim"
	if _, err = svc.ConfirmCancelled(context.Background(), j.ID, running.Attempts[0].ID, wrong, requested.Version); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("proof for another claim accepted: %v", err)
	}
	cancelled, err := svc.ConfirmCancelled(context.Background(), j.ID, running.Attempts[0].ID, proof, requested.Version)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != jobs.Cancelled || cancelled.Settlement != jobs.SettlementPending {
		t.Fatalf("execution and accounting state conflated: %#v", cancelled)
	}
	settled, err := svc.Settle(context.Background(), j.ID, jobs.SettlementSettled, cancelled.Version)
	if err != nil {
		t.Fatal(err)
	}
	if settled.State != jobs.Cancelled || settled.Settlement != jobs.SettlementSettled {
		t.Fatalf("settlement changed execution result: %#v", settled)
	}
}

func TestRecoveryClassifiesQueuedLiveAndInterruptedWork(t *testing.T) { // [REQ:DS-JOB-RECOVERY]
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "recovery.db"))
	svc := newService(repo)
	queued := submit(t, svc)
	if _, err := svc.Start(context.Background(), queued.ID, jobs.ExecutionClaimInput{}, queued.Version); !errors.Is(err, jobs.ErrInvalid) {
		t.Fatalf("start without persisted execution owner claim accepted: %v", err)
	}
	running, err := svc.Start(context.Background(), queued.ID, executionOwner("scope-1", queued.Attempts[0].ID, 1), queued.Version)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := svc.Recoveries(context.Background(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].Class != jobs.RecoveryLive {
		t.Fatalf("live lease classification: %#v", classes)
	}
	// Reconstruct the service with a later clock to model restart after lease expiry.
	svc = jobs.NewService(repo, func() time.Time { return baseTime.Add(10 * time.Minute) })
	classes, err = svc.Recoveries(context.Background(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].Class != jobs.RecoveryReconcile || classes[0].AttemptID != running.Attempts[0].ID {
		t.Fatalf("stale execution must reconcile, never blindly retry: %#v", classes)
	}
}

func TestQueuedCancelIsTerminalWithoutClaimingProcessCleanup(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "queued-cancel.db"))
	svc := newService(repo)
	j := submit(t, svc)
	cancelled, err := svc.RequestCancel(context.Background(), j.ID, j.Version)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != jobs.Cancelled || cancelled.Quiescence == nil || !cancelled.Quiescence.NoOwnedExecution || cancelled.Attempts[0].State != jobs.AttemptCancelled {
		t.Fatalf("pre-dispatch cancel lacks no-worker evidence: %#v", cancelled)
	}
	if _, err = svc.RequestCancel(context.Background(), j.ID, cancelled.Version); !errors.Is(err, jobs.ErrAlreadyTerminal) {
		t.Fatalf("terminal cancellation not stable: %v", err)
	}
}

func TestRecoveryDoesNotDispatchCheckpointWithoutPersistedQuiescenceProof(t *testing.T) {
	db, repo := openRepo(t, filepath.Join(t.TempDir(), "corrupt-checkpoint.db"))
	svc := newService(repo)
	j := submit(t, svc)
	running, err := svc.Start(context.Background(), j.ID, executionOwner("scope-proof", j.Attempts[0].ID, 1), j.Version)
	if err != nil {
		t.Fatal(err)
	}
	a := running.Attempts[0]
	cp := jobs.Checkpoint{AttemptID: a.ID, AttemptNumber: a.Number, ExecutionClaimID: a.Execution.ID, InputFingerprint: j.InputFingerprint, ToolProfileDigest: j.Input.ToolProfileDigest, SourceClockDigest: j.Input.SourceClockDigest, Unit: 3, ArtifactID: "checkpoint-3", SHA256: "9999999999999999999999999999999999999999999999999999999999999999", CreatedAt: baseTime}
	proof := proofFor(running, a, "worker owner confirms stopped")
	checkpointed, err := svc.SaveCheckpoint(context.Background(), j.ID, a.ID, cp, proof, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	if err = db.QueryRow("SELECT record_json FROM render_job_attempts WHERE id=?", a.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var stored jobs.Attempt
	if err = json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatal(err)
	}
	stored.CheckpointQuiescence = nil
	tampered, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE render_job_attempts SET record_json=? WHERE id=?", string(tampered), a.ID); err != nil {
		t.Fatal(err)
	}
	recovery, err := svc.Recoveries(context.Background(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovery) != 1 || recovery[0].Class != jobs.RecoveryReconcile {
		t.Fatalf("checkpoint missing durable owner proof was treated as resumable: %#v", recovery)
	}
	if _, err = svc.Start(context.Background(), j.ID, executionOwner("scope-next", "attempt-next-2", 2), checkpointed.Version); !errors.Is(err, jobs.ErrCheckpointMismatch) {
		t.Fatalf("resume proceeded without durable quiescence proof: %v", err)
	}
}

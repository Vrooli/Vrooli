package jobs_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"device-studio/internal/jobs"
)

func failedInspection(t *testing.T, accounting bool) (jobs.Service, jobs.Job) {
	t.Helper()
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "inspection-retry.db"))
	svc := newService(repo)
	in := typedInput()
	if accounting {
		in.AccountingRequired = true
		in.AccountingOperationID = "inspection-accounting"
	}
	job, created, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "inspection-retry", Input: in})
	if err != nil || !created {
		t.Fatalf("submit inspection: created=%v err=%v", created, err)
	}
	running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{
		AttemptID: job.Attempts[0].ID, AttemptNumber: 1, OwnerID: "inspection-owner", ScopeID: "inspection-scope",
		ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212",
	}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := svc.Fail(context.Background(), job.ID, running.Attempts[0].ID, "known tool failure", running.Version)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err = svc.RecordReleased(context.Background(), terminal.ID, terminal.Attempts[0].ID, terminal.Version, terminal.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.PendingSubjectFinalization(context.Background(), terminal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.CompleteSubjectFinalization(context.Background(), terminal.ID, pending.AttemptID, pending.Kind); err != nil {
		t.Fatal(err)
	}
	if accounting {
		terminal, err = svc.Settle(context.Background(), terminal.ID, jobs.SettlementSettled, terminal.Version)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		terminal, err = svc.Get(context.Background(), terminal.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	return svc, terminal
}

func TestRetrySourceInspectionPersistsOneAttemptAndReplaysByTerminalVersion(t *testing.T) {
	svc, failed := failedInspection(t, false)
	retried, created, err := svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version)
	if err != nil || !created {
		t.Fatalf("retry: created=%v err=%v", created, err)
	}
	if retried.State != jobs.Queued || retried.Version != failed.Version+1 || len(retried.Attempts) != 2 {
		t.Fatalf("retry transition: %+v", retried)
	}
	if retried.InputFingerprint != failed.InputFingerprint || retried.Input.SubjectID != failed.Input.SubjectID || retried.Input.ToolProfileDigest != failed.Input.ToolProfileDigest {
		t.Fatalf("retry changed immutable input: before=%+v after=%+v", failed.Input, retried.Input)
	}
	newAttempt := retried.Attempts[1]
	if newAttempt.State != jobs.AttemptQueued || newAttempt.Number != 2 || newAttempt.RetryFromVersion != failed.Version || newAttempt.ID == failed.Attempts[0].ID {
		t.Fatalf("retry attempt relation is incomplete: %+v", newAttempt)
	}
	replayed, created, err := svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version)
	if err != nil || created || replayed.Version != retried.Version || len(replayed.Attempts) != 2 || replayed.Attempts[1].ID != newAttempt.ID {
		t.Fatalf("same-version replay created another attempt: created=%v job=%+v err=%v", created, replayed, err)
	}
	if _, _, err = svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version-1); !errors.Is(err, jobs.ErrConcurrentTransition) {
		t.Fatalf("stale unrelated retry version = %v, want conflict", err)
	}
}

func TestRetrySourceInspectionStartsLatestAttemptAndPreservesReleasedHistory(t *testing.T) {
	svc, failed := failedInspection(t, false)
	retried, created, err := svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version)
	if err != nil || !created {
		t.Fatalf("retry: created=%v err=%v", created, err)
	}
	latest := retried.Attempts[len(retried.Attempts)-1]
	running, err := svc.Start(context.Background(), retried.ID, jobs.ExecutionClaimInput{
		AttemptID: latest.ID, AttemptNumber: latest.Number, OwnerID: "inspection-owner", ScopeID: "inspection-retry-scope",
		ArtifactID: "retry-claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212",
	}, retried.Version)
	if err != nil {
		t.Fatalf("start latest retry attempt: %v", err)
	}
	if running.State != jobs.Running || len(running.Attempts) != 2 || running.Attempts[0].State != jobs.AttemptFailed || running.Attempts[0].ExecutionReleasedAt == nil || running.Attempts[1].ID != latest.ID || running.Attempts[1].State != jobs.AttemptRunning || running.Attempts[1].Execution == nil {
		t.Fatalf("retry start did not preserve history and claim latest attempt: %+v", running)
	}
	if _, err := svc.Start(context.Background(), running.ID, jobs.ExecutionClaimInput{
		AttemptID: latest.ID, AttemptNumber: latest.Number, OwnerID: "inspection-owner", ScopeID: "second-claim",
		ArtifactID: "retry-claim-2", SHA256: "1212121212121212121212121212121212121212121212121212121212121212",
	}, running.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("running retry was claimable a second time: %v", err)
	}
}

func TestRetrySourceInspectionRequiresReleaseAndFinalization(t *testing.T) {
	for _, tc := range []struct {
		name        string
		acknowledge bool
		release     bool
	}{
		{name: "pending finalization", acknowledge: false, release: true},
		{name: "unreleased execution", acknowledge: true, release: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, repo := openRepo(t, filepath.Join(t.TempDir(), "blocked.db"))
			svc := newService(repo)
			job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "blocked", Input: typedInput()})
			if err != nil {
				t.Fatal(err)
			}
			running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{
				AttemptID: job.Attempts[0].ID, AttemptNumber: 1, OwnerID: "owner", ScopeID: "scope", ArtifactID: "claim",
				SHA256: "1212121212121212121212121212121212121212121212121212121212121212",
			}, job.Version)
			if err != nil {
				t.Fatal(err)
			}
			failed, err := svc.Fail(context.Background(), job.ID, running.Attempts[0].ID, "failure", running.Version)
			if err != nil {
				t.Fatal(err)
			}
			if tc.release {
				failed, err = svc.RecordReleased(context.Background(), job.ID, failed.Attempts[0].ID, failed.Version, failed.UpdatedAt)
				if err != nil {
					t.Fatal(err)
				}
			}
			if tc.acknowledge {
				pending, err := svc.PendingSubjectFinalization(context.Background(), job.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err = svc.CompleteSubjectFinalization(context.Background(), job.ID, pending.AttemptID, pending.Kind); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err = svc.RetrySourceInspection(context.Background(), job.ID, failed.Version); !errors.Is(err, jobs.ErrInvalidTransition) && !errors.Is(err, jobs.ErrReleasePending) {
				t.Fatalf("retry crossed lifecycle fence: %v", err)
			}
		})
	}
}

func TestRetrySourceInspectionRequiresDurableMatchingReleaseReceipt(t *testing.T) {
	db, repo := openRepo(t, filepath.Join(t.TempDir(), "missing-release-receipt.db"))
	svc := newService(repo)
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "missing-release", Input: typedInput()})
	if err != nil {
		t.Fatal(err)
	}
	running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: job.Attempts[0].ID, AttemptNumber: 1, OwnerID: "owner", ScopeID: "scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := svc.Fail(context.Background(), job.ID, running.Attempts[0].ID, "failure", running.Version)
	if err != nil {
		t.Fatal(err)
	}
	failed, err = svc.RecordReleased(context.Background(), failed.ID, failed.Attempts[0].ID, failed.Version, failed.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.PendingSubjectFinalization(context.Background(), failed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.CompleteSubjectFinalization(context.Background(), failed.ID, pending.AttemptID, pending.Kind); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`DELETE FROM render_job_release_receipts WHERE job_id=? AND attempt_id=?`, failed.ID, failed.Attempts[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version); !errors.Is(err, jobs.ErrReleasePending) {
		t.Fatalf("retry without durable release receipt = %v", err)
	}
}

func TestRetrySourceInspectionRequiresResolvedSettlement(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "settlement.db"))
	svc := newService(repo)
	in := typedInput()
	in.AccountingRequired = true
	in.AccountingOperationID = "accounting-op"
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "settlement", Input: in})
	if err != nil {
		t.Fatal(err)
	}
	running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: job.Attempts[0].ID, AttemptNumber: 1, OwnerID: "owner", ScopeID: "scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := svc.Fail(context.Background(), job.ID, running.Attempts[0].ID, "failure", running.Version)
	if err != nil {
		t.Fatal(err)
	}
	failed, err = svc.RecordReleased(context.Background(), failed.ID, failed.Attempts[0].ID, failed.Version, failed.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.PendingSubjectFinalization(context.Background(), failed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.CompleteSubjectFinalization(context.Background(), failed.ID, pending.AttemptID, pending.Kind); err != nil {
		t.Fatal(err)
	}
	if _, _, err = svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("unsettled retry = %v", err)
	}
	settled, err := svc.Settle(context.Background(), failed.ID, jobs.SettlementSettled, failed.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := svc.RetrySourceInspection(context.Background(), settled.ID, settled.Version); err != nil || !created {
		t.Fatalf("settled retry: created=%v err=%v", created, err)
	}
}

func TestRetrySourceInspectionAcceptsVerifiedCancellationOnlyAfterQuiescence(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "cancel-retry.db"))
	svc := newService(repo)
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "cancel-retry", Input: typedInput()})
	if err != nil {
		t.Fatal(err)
	}
	running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: job.Attempts[0].ID, AttemptNumber: 1, OwnerID: "owner", ScopeID: "scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	cancelRequested, err := svc.RequestCancel(context.Background(), running.ID, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	proof := proofFor(cancelRequested, cancelRequested.Attempts[0], "worker process and descendants are quiescent")
	cancelled, err := svc.ConfirmCancelled(context.Background(), cancelRequested.ID, cancelRequested.Attempts[0].ID, proof, cancelRequested.Version)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err = svc.RecordReleased(context.Background(), cancelled.ID, cancelled.Attempts[0].ID, cancelled.Version, cancelled.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.PendingSubjectFinalization(context.Background(), cancelled.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.CompleteSubjectFinalization(context.Background(), cancelled.ID, pending.AttemptID, pending.Kind); err != nil {
		t.Fatal(err)
	}
	retried, created, err := svc.RetrySourceInspection(context.Background(), cancelled.ID, cancelled.Version)
	if err != nil || !created {
		t.Fatalf("verified cancellation retry: created=%v err=%v", created, err)
	}
	if retried.State != jobs.Queued || len(retried.Attempts) != 2 || retried.Attempts[0].State != jobs.AttemptCancelled || retried.Attempts[1].RetryFromVersion != cancelled.Version || retried.Quiescence != nil {
		t.Fatalf("cancelled attempt did not remain immutable under retry: %+v", retried)
	}
}

func TestRetrySourceInspectionRejectsSuccessfulTerminalJob(t *testing.T) {
	_, repo := openRepo(t, filepath.Join(t.TempDir(), "success-not-retryable.db"))
	svc := newService(repo)
	job, _, err := svc.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: "success-not-retryable", Input: typedInput()})
	if err != nil {
		t.Fatal(err)
	}
	running, err := svc.Start(context.Background(), job.ID, jobs.ExecutionClaimInput{AttemptID: job.Attempts[0].ID, AttemptNumber: 1, OwnerID: "owner", ScopeID: "scope", ArtifactID: "claim", SHA256: "1212121212121212121212121212121212121212121212121212121212121212"}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	completion := outputs()
	completion.AttemptID, completion.ExecutionClaimID, completion.InputFingerprint = running.Attempts[0].ID, running.Attempts[0].Execution.ID, running.InputFingerprint
	succeeded, err := svc.Complete(context.Background(), running.ID, running.Attempts[0].ID, completion, running.Version)
	if err != nil {
		t.Fatal(err)
	}
	succeeded, err = svc.RecordReleased(context.Background(), succeeded.ID, succeeded.Attempts[0].ID, succeeded.Version, succeeded.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.PendingSubjectFinalization(context.Background(), succeeded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.CompleteSubjectFinalization(context.Background(), succeeded.ID, pending.AttemptID, pending.Kind); err != nil {
		t.Fatal(err)
	}
	if _, _, err = svc.RetrySourceInspection(context.Background(), succeeded.ID, succeeded.Version); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("successful inspection was retried: %v", err)
	}
}

func TestConcurrentRetrySourceInspectionConvergesOnOneAttempt(t *testing.T) {
	svc, failed := failedInspection(t, false)
	const callers = 12
	var wg sync.WaitGroup
	var mu sync.Mutex
	createdCount := 0
	ids := map[string]bool{}
	errs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, created, err := svc.RetrySourceInspection(context.Background(), failed.ID, failed.Version)
			if err != nil {
				errs <- err
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if created {
				createdCount++
			}
			if len(job.Attempts) != 2 {
				errs <- errors.New("concurrent replay did not return the single persisted retry")
				return
			}
			ids[job.Attempts[1].ID] = true
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent retry: %v", err)
	}
	if createdCount != 1 || len(ids) != 1 {
		t.Fatalf("concurrent callers created %d attempts with ids %v", createdCount, ids)
	}
}

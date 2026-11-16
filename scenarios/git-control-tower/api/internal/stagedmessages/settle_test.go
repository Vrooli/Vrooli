package stagedmessages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

type settleStatusProvider struct {
	*fakeProvider
	statuses map[string]string
}

func (p settleStatusProvider) Get(_ context.Context, id string, _ DraftInput) (Execution, error) {
	status := p.statuses[id]
	result := (*v1.StagedMessageSuggestion)(nil)
	if status == "succeeded" {
		result = validResult()
	}
	return Execution{ID: id, Status: status, Result: result}, nil
}

type unreachableProvider struct{ *fakeProvider }

func (unreachableProvider) Get(context.Context, string, DraftInput) (Execution, error) {
	return Execution{}, errors.New("owner unreachable")
}

func fixtureDirectory(context.Context, string) (string, error) { return "fixture", nil }

func TestSettleRecordsFinishedOwnerResultWithoutAReader(t *testing.T) {
	s, store, _, p := serviceFixture(t)
	_, j := admission(t, s, "settle")
	if n, err := s.Settle(context.Background(), fixtureDirectory); err != nil || n != 0 {
		t.Fatalf("running job settled: n=%d err=%v", n, err)
	}
	p.complete("succeeded", validResult())
	if n, err := s.Settle(context.Background(), fixtureDirectory); err != nil || n != 1 {
		t.Fatalf("settle n=%d err=%v", n, err)
	}
	job, err := store.Get(context.Background(), j.Id)
	if err != nil || job.State != reviewjobs.Succeeded || job.ResultJSON == "" {
		t.Fatalf("settled job %#v err=%v", job, err)
	}
	if n, _ := s.Settle(context.Background(), fixtureDirectory); n != 0 {
		t.Fatalf("terminal job settled again: %d", n)
	}
}

func TestSettleRecoversDurableAdmissionWithoutDispatchingTwice(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	collection, err := s.Collect(t.Context(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(DraftInput{FormatVersion: 2, Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK, Snapshot: collection.Snapshot, Contexts: collection.Contexts})
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(t.Context(), reviewjobs.Job{
		ID: "crash-before-dispatch", IdempotencyKey: "repo:crash-before-dispatch", InputDigest: digest(input), InputJSON: string(input),
		Operation: operation, RepositoryID: "repo", PolicyVersion: "staged-message-v1", State: reviewjobs.Queued,
	})
	if err != nil {
		t.Fatal(err)
	}

	// A status read observes the durable row and must not become a hidden retry.
	observed, err := s.Job(t.Context(), "repo", "fixture", job.ID, "", "get", 0)
	if err != nil || observed.Status != string(reviewjobs.Queued) || provider.starts != 0 {
		t.Fatalf("observational read dispatched queued work: status=%s starts=%d err=%v", observed.GetStatus(), provider.starts, err)
	}

	if settled, err := s.Settle(t.Context(), fixtureDirectory); err != nil || settled != 0 {
		t.Fatalf("recovery sweep result=%d err=%v", settled, err)
	}
	recovered, err := store.Get(t.Context(), job.ID)
	if err != nil || recovered.State != reviewjobs.Running || recovered.ExecutionID == "" || !recovered.DispatchAttempted || provider.starts != 1 || len(provider.startKeys) != 1 || provider.startKeys[0] != job.IdempotencyKey {
		t.Fatalf("durable admission was not recovered by its saved key: job=%+v starts=%d keys=%v err=%v", recovered, provider.starts, provider.startKeys, err)
	}
	if settled, err := s.Settle(t.Context(), fixtureDirectory); err != nil || settled != 0 || provider.starts != 1 {
		t.Fatalf("later recovery duplicated owner start: settled=%d starts=%d err=%v", settled, provider.starts, err)
	}
}

func TestSettleRecoveryHonorsCancellationRacingDispatchClaim(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	collection, err := s.Collect(t.Context(), "repo", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(DraftInput{FormatVersion: 2, Mode: v1.StagedMessageMode_STAGED_MESSAGE_MODE_QUICK, Snapshot: collection.Snapshot, Contexts: collection.Contexts})
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.Create(t.Context(), reviewjobs.Job{
		ID: "cancel-before-recovery-dispatch", IdempotencyKey: "repo:cancel-before-recovery-dispatch", InputDigest: digest(input), InputJSON: string(input),
		Operation: operation, RepositoryID: "repo", PolicyVersion: "staged-message-v1", State: reviewjobs.Queued,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.beforeDispatch = func(ctx context.Context, claimed reviewjobs.Job) {
		if claimed.ID != job.ID {
			return
		}
		if err := store.RequestAdvisoryCancel(ctx, claimed.ID, "repo"); err != nil {
			t.Errorf("request cancellation: %v", err)
		}
	}
	if _, err := s.Settle(t.Context(), fixtureDirectory); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.Get(t.Context(), job.ID)
	if err != nil || recovered.State != reviewjobs.Cancelled || provider.starts != 0 {
		t.Fatalf("recovery dispatched after cancellation: state=%s starts=%d err=%v", recovered.State, provider.starts, err)
	}
}

func TestSettleLeavesJobUntouchedWhenOwnerIsUnreachable(t *testing.T) {
	s, store, _, p := serviceFixture(t)
	_, j := admission(t, s, "unreachable")
	before, _ := store.Get(context.Background(), j.Id)
	s.provider = unreachableProvider{p}
	if n, err := s.Settle(context.Background(), fixtureDirectory); err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	after, _ := store.Get(context.Background(), j.Id)
	if after.Error != "" || after.State != before.State || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("background outage changed the job: before=%#v after=%#v", before, after)
	}
}

func TestSettleContinuesPendingCancellationUntilTerminal(t *testing.T) {
	s, store, _, p := serviceFixture(t)
	_, j := admission(t, s, "cancel-pending")
	if _, err := s.Job(context.Background(), "repo", "fixture", j.Id, "", "cancel", 0); err != nil {
		t.Fatal(err)
	}
	p.complete("succeeded", validResult())
	if n, err := s.Settle(context.Background(), fixtureDirectory); err != nil || n != 1 {
		t.Fatalf("cancellation was not settled after owner terminal confirmation: n=%d err=%v", n, err)
	}
	if job, _ := store.Get(context.Background(), j.Id); job.State != reviewjobs.Cancelled || job.ResultJSON != "" {
		t.Fatalf("cancelled job adopted a result or remained active: %+v", job)
	}
}

func TestSettleDoesNotStarveOlderDispatchedJobsBehindUnclaimedRows(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	_, older := admission(t, s, "older-dispatched")
	provider.complete("succeeded", validResult())
	base := time.Now().UTC().Add(time.Minute)
	for index := 0; index < MaxSettledPerPass; index++ {
		id := fmt.Sprintf("newer-unclaimed-%02d", index)
		_, _, err := store.Create(t.Context(), reviewjobs.Job{
			ID: id, IdempotencyKey: id, InputDigest: id, Operation: operation,
			RepositoryID: "repo", PolicyVersion: "staged-message-v1", State: reviewjobs.Running,
			CreatedAt: base.Add(time.Duration(index) * time.Second), InputJSON: `{}`,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	settled, err := s.Settle(t.Context(), fixtureDirectory)
	if err != nil || settled != 1 {
		t.Fatalf("older dispatched job was starved: settled=%d err=%v", settled, err)
	}
	updated, err := store.Get(t.Context(), older.Id)
	if err != nil || updated.State != reviewjobs.Succeeded || updated.ResultJSON == "" {
		t.Fatalf("older job result not made durable: state=%s result=%t err=%v", updated.State, updated.ResultJSON != "", err)
	}
}

func TestSettleRotatesPastDispatchedJobsThatRemainRunning(t *testing.T) {
	s, store, _, provider := serviceFixture(t)
	_, oldest := admission(t, s, "oldest-finished")
	seedInput, err := store.Get(t.Context(), oldest.Id)
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the valid immutable draft input while assigning stable IDs whose
	// first page remains running on every sweep.
	input := seedInput.InputJSON
	statuses := map[string]string{seedInput.ExecutionID: "succeeded"}
	for index := 0; index < MaxSettledPerPass; index++ {
		id := fmt.Sprintf("active-%02d", index)
		_, _, err := store.Create(t.Context(), reviewjobs.Job{
			ID: id, IdempotencyKey: id, InputDigest: id, InputJSON: input, Operation: operation,
			RepositoryID: "repo", ExecutionID: id, PolicyVersion: "staged-message-v1", State: reviewjobs.Running,
			CreatedAt: time.Now().UTC().Add(time.Duration(index) * time.Second),
		})
		if err != nil {
			t.Fatal(err)
		}
		statuses[id] = "running"
	}
	s.provider = settleStatusProvider{fakeProvider: provider, statuses: statuses}

	if settled, err := s.Settle(t.Context(), fixtureDirectory); err != nil || settled != 0 {
		t.Fatalf("first bounded sweep settled=%d err=%v", settled, err)
	}
	settled, err := s.Settle(t.Context(), fixtureDirectory)
	if err != nil || settled != 1 {
		t.Fatalf("second sweep failed to reach older completed execution: settled=%d err=%v", settled, err)
	}
	job, err := store.Get(t.Context(), oldest.Id)
	if err != nil || job.State != reviewjobs.Succeeded || job.ResultJSON == "" {
		t.Fatalf("older completed result not made durable: state=%s result=%t err=%v", job.State, job.ResultJSON != "", err)
	}
	if settled, err := s.Settle(t.Context(), fixtureDirectory); err != nil || settled != 0 {
		t.Fatalf("sweep did not wrap back to remaining active jobs: settled=%d err=%v", settled, err)
	}
}

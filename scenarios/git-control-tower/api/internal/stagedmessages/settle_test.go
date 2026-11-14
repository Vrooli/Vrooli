package stagedmessages

import (
	"context"
	"errors"
	"testing"

	"git-control-tower/internal/reviewjobs"
)

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

func TestSettleSkipsPendingCancellation(t *testing.T) {
	s, store, _, p := serviceFixture(t)
	_, j := admission(t, s, "cancel-pending")
	if _, err := s.Job(context.Background(), "repo", "fixture", j.Id, "", "cancel", 0); err != nil {
		t.Fatal(err)
	}
	p.complete("succeeded", validResult())
	if n, _ := s.Settle(context.Background(), fixtureDirectory); n != 0 {
		t.Fatalf("cancellation settled in background: %d", n)
	}
	if job, _ := store.Get(context.Background(), j.Id); job.ResultJSON != "" {
		t.Fatalf("cancelled job adopted a result")
	}
}

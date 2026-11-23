package proposals

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestWaitingListsOpenProposalsByRepositoryBeforeCutoffOldestFirst(t *testing.T) {
	ctx := context.Background()
	service := testService(t)
	firstRepo := repoFor(NewMemoryGit(map[string]string{"a.go": "before"}))
	firstRepo.ID = "1"
	firstRepo.Git.(*MemoryGit).Write("a.go", "after")
	first := createProposal(t, service, firstRepo, "a.go")

	otherRepo := repoFor(NewMemoryGit(map[string]string{"b.go": "before"}))
	otherRepo.ID = "2"
	otherRepo.Git.(*MemoryGit).Write("b.go", "after")
	other := createProposal(t, service, otherRepo, "b.go")

	latest := createProposal(t, service, firstRepo, "a.go")
	if !first.CreatedAt.Before(other.CreatedAt) || !other.CreatedAt.Before(latest.CreatedAt) {
		t.Fatalf("fixture times did not increase: first=%s latest=%s other=%s", first.CreatedAt, latest.CreatedAt, other.CreatedAt)
	}
	if _, err := service.Close(ctx, firstRepo, latest.ID, latest.Revision, "replaced", first.ID, humanActor); err != nil {
		t.Fatalf("close latest proposal: %v", err)
	}

	cutoff := other.CreatedAt.Add(-time.Nanosecond)
	items, err := service.Waiting(ctx, []string{"2", "1"}, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("waiting items = %#v, want only the earlier proposal before cutoff", items)
	}
	want := WaitingItem{
		ID: first.ID, RepositoryID: "1", EffortRef: "effort:bas", Epoch: "E27",
		Subject: first.Message.Subject, Files: len(first.Files), CreatedAt: first.CreatedAt,
	}
	if !reflect.DeepEqual(items[0], want) {
		t.Fatalf("waiting item = %#v, want %#v", items[0], want)
	}

	items, err = service.Waiting(ctx, []string{"2", "1"}, other.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != first.ID || items[1].ID != other.ID {
		t.Fatalf("waiting order = %#v, want first repository's oldest then second repository", items)
	}
}

func TestWaitingDigestIsIdempotentPerDayAndReportsStoreErrors(t *testing.T) {
	ctx := context.Background()
	service := testService(t)
	day := "2026-11-23"
	if sent, err := service.DigestSent(ctx, day); err != nil || sent {
		t.Fatalf("initial digest state = (%v, %v), want (false, nil)", sent, err)
	}

	sentAt := time.Date(2026, 11, 23, 17, 30, 0, 0, time.FixedZone("UTC-5", -5*60*60))
	if err := service.RecordDigest(ctx, day, 2); err != nil {
		t.Fatal(err)
	}
	if sent, err := service.DigestSent(ctx, day); err != nil || !sent {
		t.Fatalf("recorded digest state = (%v, %v), want (true, nil)", sent, err)
	}
	if err := service.store.RecordDigest(ctx, day, 99, sentAt); err != nil {
		t.Fatal(err)
	}
	var count int
	var storedAt string
	if err := service.store.db.QueryRowContext(ctx, `SELECT proposal_count, sent_at FROM gct_proposal_digests WHERE day = ?`, day).Scan(&count, &storedAt); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("duplicate digest replaced first record: count=%d sent_at=%q", count, storedAt)
	}
	if err := service.store.RecordDigest(ctx, "2026-11-24", 1, sentAt); err != nil {
		t.Fatal(err)
	}
	if err := service.store.db.QueryRowContext(ctx, `SELECT sent_at FROM gct_proposal_digests WHERE day = ?`, "2026-11-24").Scan(&storedAt); err != nil {
		t.Fatal(err)
	}
	if storedAt != sentAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("digest timestamp = %q, want UTC %q", storedAt, sentAt.UTC().Format(time.RFC3339Nano))
	}

	if err := service.store.db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.DigestSent(ctx, day); err == nil {
		t.Fatal("closed digest store returned no error")
	}
	if err := service.RecordDigest(ctx, day, 1); err == nil {
		t.Fatal("closed digest store accepted a write")
	}
	if _, err := service.Waiting(ctx, []string{"1"}, time.Now()); err == nil {
		t.Fatal("closed proposal store returned a waiting list")
	}
}

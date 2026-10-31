package main

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"git-control-tower/internal/proposals"
	"git-control-tower/internal/reviewjobs"

	"github.com/vrooli/api-core/eventbus"
	"google.golang.org/protobuf/types/known/structpb"
	_ "modernc.org/sqlite"
)

type notificationPublisher struct {
	mu     sync.Mutex
	events []eventbus.DomainEvent
	fail   bool
}

func (p *notificationPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fail {
		return errors.New("events unavailable")
	}
	p.events = append(p.events, event)
	return nil
}

func notifierFixture(t *testing.T, now time.Time) (*gctNotifier, *reviewjobs.Store, *notificationPublisher) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	jobs, err := reviewjobs.New(db)
	if err != nil {
		t.Fatal(err)
	}
	publisher := &notificationPublisher{}
	n := &gctNotifier{publisher: publisher, db: db, jobs: jobs, now: func() time.Time { return now }}
	n.resolveBase = func(context.Context) string { return "https://gct.example/" }
	n.repoName = func(context.Context, string) string { return "Vrooli" }
	if err := n.ensureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return n, jobs, publisher
}

func createDraftJob(t *testing.T, jobs *reviewjobs.Store, id string, state reviewjobs.State, at time.Time, result, failure string) {
	t.Helper()
	if _, _, err := jobs.Create(context.Background(), reviewjobs.Job{ID: id, IdempotencyKey: id, InputDigest: "sha256:x", Operation: "staged-message", RepositoryID: "1", PolicyVersion: "staged-message-v1", State: state, ResultJSON: result, Error: failure, CreatedAt: at}); err != nil {
		t.Fatal(err)
	}
}

func TestProposalNotificationEventCarriesNeutralFacts(t *testing.T) {
	created := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	view := proposals.View{Proposal: proposals.Proposal{
		ID: "gctp-abc", RepositoryID: "1", CreatedAt: created,
		Work:    proposals.Work{EffortRef: "effort:bas-rehab", Epoch: "E26"},
		Message: proposals.Message{Subject: "Fix replay timing"}, Files: []proposals.File{{Path: "a"}, {Path: "b"}},
	}}
	event := proposalNotificationEvent(view, "Vrooli", "https://gct.example")
	if event.EventType != proposalNotificationEventType || event.Source != "git-control-tower" || !event.Occurred.Equal(created) {
		t.Fatalf("envelope %#v", event)
	}
	p := event.Payload
	if p["category"] != notificationCategoryProposalReceived || p["dedupe_key"] != "gctp-abc" || p["sensitivity"] != "private" {
		t.Fatalf("payload %#v", p)
	}
	if p["title"] != "Commit proposal to review · Vrooli" || p["body"] != "Fix replay timing\n2 files · bas-rehab E26" || p["context_url"] != "https://gct.example/?proposal=gctp-abc" {
		t.Fatalf("copy %#v", p)
	}
	if _, err := structpb.NewStruct(p); err != nil {
		t.Fatal(err)
	}
}

func TestProposalNotificationSkipsVerifiedHumanAuthor(t *testing.T) {
	n, _, publisher := notifierFixture(t, time.Now().UTC())
	human := proposals.View{Proposal: proposals.Proposal{ID: "gctp-1", CreatedBy: proposals.Actor{Kind: "human", Verified: true}}}
	n.NotifyProposalCreated(context.Background(), "Vrooli", human)
	agent := proposals.View{Proposal: proposals.Proposal{ID: "gctp-2", CreatedBy: proposals.Actor{Kind: "agent"}}}
	n.NotifyProposalCreated(context.Background(), "Vrooli", agent)
	if len(publisher.events) != 1 || publisher.events[0].Payload["dedupe_key"] != "gctp-2" {
		t.Fatalf("events %#v", publisher.events)
	}
}

func TestFinishedDraftsAreAnnouncedOnceAfterTheWatermark(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	n, jobs, publisher := notifierFixture(t, now)
	createDraftJob(t, jobs, "message-old", reviewjobs.Succeeded, now.Add(-time.Minute), `{"title":"Before notifications"}`, "")
	createDraftJob(t, jobs, "message-ready", reviewjobs.Succeeded, now.Add(time.Minute), `{"title":"Add continuation trailer"}`, "")
	createDraftJob(t, jobs, "message-failed", reviewjobs.Failed, now.Add(time.Minute), "", "provider refused input")
	createDraftJob(t, jobs, "message-cancelled", reviewjobs.Cancelled, now.Add(time.Minute), "", "")
	createDraftJob(t, jobs, "message-running", reviewjobs.Running, now.Add(time.Minute), "", "")
	for pass := 0; pass < 2; pass++ {
		if _, err := n.NotifyFinishedDrafts(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	got := map[string]map[string]any{}
	for _, event := range publisher.events {
		if event.EventType != commitMessageNotificationEventType {
			t.Fatalf("event type %q", event.EventType)
		}
		got[event.Payload["dedupe_key"].(string)] = event.Payload
	}
	if len(publisher.events) != 2 || got["message-ready"] == nil || got["message-failed"] == nil {
		t.Fatalf("events %#v", publisher.events)
	}
	if got["message-ready"]["category"] != notificationCategoryCommitMessageReady || got["message-ready"]["title"] != "Commit message ready · Vrooli" || got["message-ready"]["body"] != "Add continuation trailer" {
		t.Fatalf("ready %#v", got["message-ready"])
	}
	if got["message-failed"]["category"] != notificationCategoryCommitMessageFailed || got["message-failed"]["title"] != "Commit message suggestion failed · Vrooli" || got["message-failed"]["body"] != "provider refused input" {
		t.Fatalf("failed %#v", got["message-failed"])
	}
}

func TestFinishedDraftIsRetriedAfterPublishFailure(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	n, jobs, publisher := notifierFixture(t, now)
	createDraftJob(t, jobs, "message-ready", reviewjobs.Succeeded, now.Add(time.Second), `{"title":"Retry me"}`, "")
	publisher.fail = true
	if count, _ := n.NotifyFinishedDrafts(context.Background()); count != 0 {
		t.Fatalf("published while failing: %d", count)
	}
	publisher.fail = false
	if count, _ := n.NotifyFinishedDrafts(context.Background()); count != 1 {
		t.Fatalf("retry published %d", count)
	}
}

func TestDraftEventIdentityIsStable(t *testing.T) {
	row := reviewjobs.AdvisorySummary{ID: "message-1", RepositoryID: "1", State: reviewjobs.Succeeded, ResultJSON: `{"title":"T"}`, UpdatedAt: time.Date(2026, 10, 28, 1, 2, 3, 4, time.UTC)}
	a, ok := commitMessageNotificationEvent(row, "Vrooli", "")
	b, _ := commitMessageNotificationEvent(row, "Vrooli", "")
	if !ok || a.EventType != b.EventType || !a.Occurred.Equal(b.Occurred) || a.Payload["body"] != b.Payload["body"] || a.Payload["context_url"] != nil {
		t.Fatalf("unstable event %#v %#v", a, b)
	}
	if _, ok := commitMessageNotificationEvent(reviewjobs.AdvisorySummary{State: reviewjobs.Cancelled}, "", ""); ok {
		t.Fatal("cancelled job announced")
	}
}

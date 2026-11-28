package supervision

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/operatordecision"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// revisableFixture raises decisions at a time after the epoch from which they
// are published under a subject key, with a clock the test advances.
func revisableFixture(t *testing.T) (*EffortService, *recordingPublisher, *time.Time) {
	t.Helper()
	s, publisher, _, _ := decisionFixture(t)
	now := subjectKeyEpoch.Add(time.Hour)
	s.now = func() time.Time { return now }
	return s, publisher, &now
}

func revisedRequest(question string) *pb.RaiseEffortDecisionRequest {
	request := decisionRequest()
	request.Decision.Question = question
	return request
}

func TestAChangedRaiseOfAPendingDecisionRevisesItInPlace(t *testing.T) {
	s, publisher, now := revisableFixture(t)
	ctx := context.Background()
	first, err := s.RaiseDecision(ctx, decisionRequest(), "run-1")
	if err != nil || !first.Created || publisher.count() != 1 {
		t.Fatalf("first raise: %+v %v", first, err)
	}
	*now = now.Add(10 * time.Minute)
	revised, err := s.RaiseDecision(ctx, revisedRequest("Close the goal now, given the baseline is missing?"), "run-2")
	if err != nil || revised.Created || !revised.Revised || !revised.Published || publisher.count() != 2 {
		t.Fatalf("a changed raise of a pending decision must revise it and publish the revision: %+v %v (published %d)", revised, err, publisher.count())
	}
	d := revised.Decision
	if d.Question != "Close the goal now, given the baseline is missing?" || d.State != DecisionPending || d.RaisedByRunId != "run-2" {
		t.Fatalf("revised record: %+v", d)
	}
	// created_at stays the original raise; revised_at is the revision and stamps its event.
	if !d.CreatedAt.AsTime().Equal(first.Decision.CreatedAt.AsTime()) || !d.RevisedAt.AsTime().Equal(now.UTC().Truncate(time.Millisecond)) || !publisher.events[1].Occurred.Equal(d.RevisedAt.AsTime()) {
		t.Fatalf("created_at %v revised_at %v event occurred %v", d.CreatedAt.AsTime(), d.RevisedAt.AsTime(), publisher.events[1].Occurred)
	}
	if first.Revised || first.Decision.RevisedAt != nil {
		t.Fatalf("a first raise is not a revision: %+v", first)
	}
	if d.EventId == first.Decision.EventId || d.EventId != effortdecision.EventID(publisher.events[1]) {
		t.Fatalf("the record must name the revision's event: %s vs first %s", d.EventId, first.Decision.EventId)
	}
	if publisher.events[0].Payload["subject_key"] != publisher.events[1].Payload["subject_key"] {
		t.Fatal("a revision is the same Hub subject, so the Hub updates the ask in place")
	}
	if found, err := s.repo.FindEffortDecisionByEvent(ctx, d.EventId); err != nil || found.Question != d.Question {
		t.Fatalf("the Hub's resolution names the revision's event, which must find the decision: %v %v", found, err)
	}
	if _, err := s.repo.FindEffortDecisionByEvent(ctx, first.Decision.EventId); !errors.Is(err, ErrNotFound) {
		t.Fatalf("the replaced revision's event no longer names the decision: %v", err)
	}
	again, err := s.RaiseDecision(ctx, revisedRequest("Close the goal now, given the baseline is missing?"), "run-2")
	if err != nil || again.Created || again.Revised || again.Published || publisher.count() != 2 {
		t.Fatalf("repeating the revision is idempotent: %+v %v (published %d)", again, err, publisher.count())
	}
}

// Returning to an earlier text must still be a new revision for the Hub; the same
// bytes would be a redelivery it ignores, leaving the operator on the wrong text.
func TestRevisingBackToAnEarlierTextIsANewEventTheHubWillApply(t *testing.T) {
	s, publisher, now := revisableFixture(t)
	ctx := context.Background()
	original := decisionRequest().Decision.Question
	for _, question := range []string{original, "Changed once?", original} {
		if _, err := s.RaiseDecision(ctx, revisedRequest(question), "run-1"); err != nil {
			t.Fatal(err)
		}
		*now = now.Add(time.Minute)
	}
	if publisher.count() != 3 {
		t.Fatalf("each revision publishes: %d", publisher.count())
	}
	seen := map[string]bool{}
	for _, event := range publisher.events {
		id := effortdecision.EventID(event)
		if seen[id] {
			t.Fatalf("event %s was published twice; the Hub would treat the second as a redelivery", id)
		}
		seen[id] = true
	}
}

func TestOnlyAPendingDecisionCreatedUnderSubjectKeysCanBeRevised(t *testing.T) {
	s, _, now := revisableFixture(t)
	ctx := context.Background()
	out, err := s.RaiseDecision(ctx, decisionRequest(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	answered := DecisionResolution{
		AskID: "ask-1", State: "answered", Answer: "close", AnsweredAt: now.Format(time.RFC3339Nano), AnsweredBy: "operator",
		SourceEventType: goalhome.DecisionRequestedEventType, RequesterRef: out.Decision.EventId,
		Correlation: map[string]string{"goal_home": decisionHome, "decision_id": "BAS-D-070"},
	}
	if _, err := s.ResolveDecisionFromHub(ctx, answered); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RaiseDecision(ctx, revisedRequest("A different question?"), "run-1"); !errors.Is(err, ErrDecisionClosed) {
		t.Fatalf("an answered decision must not be revised: %v", err)
	}
}

// A decision stored before subject keys has an open Hub ask with no key, so a
// revision would open a second ask beside it.
func TestADecisionRaisedBeforeSubjectKeysKeepsTheOldRefusal(t *testing.T) {
	ctx := context.Background()
	legacy, _, _, _ := decisionFixture(t)
	legacy.now = func() time.Time { return subjectKeyEpoch.Add(-time.Hour) }
	if _, err := legacy.RaiseDecision(ctx, decisionRequest(), "run-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.RaiseDecision(ctx, revisedRequest("A different question?"), "run-1"); !errors.Is(err, ErrDecisionChanged) {
		t.Fatalf("a decision raised before subject keys keeps the old rule: %v", err)
	}
}

func TestAResolutionOfAReplacedRevisionDoesNotAnswerTheCurrentOne(t *testing.T) {
	s, _, now := revisableFixture(t)
	ctx := context.Background()
	first, err := s.RaiseDecision(ctx, decisionRequest(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Minute)
	second, err := s.RaiseDecision(ctx, revisedRequest("Close the goal now?"), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	resolution := func(ref string) DecisionResolution {
		return DecisionResolution{
			AskID: "ask-1", State: "answered", Answer: "close", AnsweredAt: now.Format(time.RFC3339Nano), AnsweredBy: "operator",
			SourceEventType: goalhome.DecisionRequestedEventType, RequesterRef: ref,
			Correlation: map[string]string{"goal_home": decisionHome, "decision_id": "BAS-D-070"},
		}
	}
	if d, err := s.ResolveDecisionFromHub(ctx, resolution(first.Decision.EventId)); err != nil || (d != nil && d.State != DecisionPending) {
		t.Fatalf("an answer to the replaced revision must not settle the current one: %+v %v", d, err)
	}
	if d, err := s.ResolveDecisionFromHub(ctx, resolution(second.Decision.EventId)); err != nil || d == nil || d.State != DecisionAnswered {
		t.Fatalf("an answer to the current revision settles it: %+v %v", d, err)
	}
}

// A decision stored before its event gained a subject key is rebuilt with the key
// when it is first published; the record must then name what was published, since
// the Hub's resolution does.
func TestAnUnpublishedDecisionRecordsTheEventIDItIsPublishedUnder(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	publisher.fail = errors.New("vrooli-events unavailable")
	out, err := s.RaiseDecision(ctx, decisionRequest(), "run-1")
	if err != nil || out.Published {
		t.Fatalf("setup: %+v %v", out, err)
	}
	stale := out.Decision
	stale.EventId = "event-id-of-the-key-less-shape"
	if err := s.repo.ReviseEffortDecision(ctx, stale, decisionContentDigest(stale)); err != nil {
		t.Fatal(err)
	}
	publisher.fail = nil
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	found, err := s.repo.FindEffortDecisionByEvent(ctx, effortdecision.EventID(publisher.events[0]))
	if err != nil || found.PublishedAt == nil {
		t.Fatalf("the published event must find the decision: %+v %v", found, err)
	}
}

func TestAContextRequestFindsOnlyTheCurrentPendingDecisionItIsAbout(t *testing.T) {
	s, _, now := revisableFixture(t)
	ctx := context.Background()
	out, err := s.RaiseDecision(ctx, decisionRequest(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	request := func(mutate func(*operatordecision.ContextRequest)) operatordecision.ContextRequest {
		r := operatordecision.ContextRequest{
			AskID: "ask-1", RequestID: "req-1", Question: "Which baseline?", SourceEventType: goalhome.DecisionRequestedEventType,
			RequesterRef: out.Decision.EventId, SubjectKey: effortdecision.SubjectKey(decisionHome, "BAS-D-070"),
			Correlation: map[string]string{"goal_home": decisionHome, "decision_id": "BAS-D-070"},
		}
		mutate(&r)
		return r
	}
	if d, err := s.PendingDecisionForContextRequest(ctx, request(func(*operatordecision.ContextRequest) {})); err != nil || d == nil || d.RaisedByRunId != "run-1" {
		t.Fatalf("the question reaches the decision and its raising run: %+v %v", d, err)
	}
	for name, mutate := range map[string]func(*operatordecision.ContextRequest){
		"another producer's event": func(r *operatordecision.ContextRequest) {
			r.SourceEventType = "research_desk.needs_you.decision_requested.v1"
		},
		"no correlation":   func(r *operatordecision.ContextRequest) { r.Correlation = nil },
		"unknown decision": func(r *operatordecision.ContextRequest) { r.Correlation["decision_id"] = "NOPE" },
		"stale revision":   func(r *operatordecision.ContextRequest) { r.RequesterRef = "an-earlier-event" },
	} {
		if d, err := s.PendingDecisionForContextRequest(ctx, request(mutate)); err != nil || d != nil {
			t.Errorf("%s: want a benign no-op, got %+v %v", name, d, err)
		}
	}
	answered := DecisionResolution{
		AskID: "ask-1", State: "answered", Answer: "close", AnsweredAt: now.Format(time.RFC3339Nano), AnsweredBy: "operator",
		SourceEventType: goalhome.DecisionRequestedEventType, RequesterRef: out.Decision.EventId,
		Correlation: map[string]string{"goal_home": decisionHome, "decision_id": "BAS-D-070"},
	}
	if _, err := s.ResolveDecisionFromHub(ctx, answered); err != nil {
		t.Fatal(err)
	}
	if d, err := s.PendingDecisionForContextRequest(ctx, request(func(*operatordecision.ContextRequest) {})); err != nil || d != nil {
		t.Fatalf("a question about a resolved decision is a no-op: %+v %v", d, err)
	}
}

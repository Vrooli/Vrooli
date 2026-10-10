package supervision

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/effortdecision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const decisionHome = "scenarios/browser-automation-studio/docs/internal/goal"

type recordingPublisher struct {
	mu     sync.Mutex
	events []eventbus.DomainEvent
	fail   error
}

func (p *recordingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fail != nil {
		return p.fail
	}
	p.events = append(p.events, event)
	return nil
}

func (p *recordingPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

func decisionRequest() *pb.RaiseEffortDecisionRequest {
	return &pb.RaiseEffortDecisionRequest{Decision: &pb.EffortDecision{
		GoalHome: decisionHome + "/GOAL.md", DecisionId: "BAS-D-070", Question: "Close the BAS goal or re-aim at quality?",
		Options:     []*pb.EffortDecisionOption{{Key: "re-aim", Label: "Re-aim at quality"}, {Key: "close", Label: "Close the goal"}},
		Recommended: "re-aim", DefaultAnswer: "re-aim", Reversible: true, Urgency: "high",
		Deadline: timestamppb.New(time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)),
	}}
}

// decisionFixture enrolls an effort registered to the goal home, so answers
// signal it.
func decisionFixture(t *testing.T) (*EffortService, *recordingPublisher, string, *fakeActionController) {
	t.Helper()
	s, _, c := effortFixture(t)
	e := enrollmentFixture(t, s, c, "running")
	e.DestinationRef = "path:" + decisionHome + "/GOAL.md"
	if _, err := s.ReconcileMetadata(context.Background(), &pb.ReconcileEffortMetadataRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "goal-home"}, EffortActor{ID: "owner", Operator: true}); err != nil {
		t.Fatal(err)
	}
	publisher := &recordingPublisher{}
	s.SetDecisionPublisher(publisher)
	return s, publisher, e.EffortRef, c
}

func TestRaiseDecisionStoresOnePerIDAndPublishesOnce(t *testing.T) {
	s, publisher, effortRef, _ := decisionFixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	created := make(chan bool, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := s.RaiseDecision(ctx, decisionRequest(), "run-1")
			if err != nil {
				t.Error(err)
				return
			}
			created <- out.Created
		}()
	}
	wg.Wait()
	close(created)
	firsts := 0
	for c := range created {
		if c {
			firsts++
		}
	}
	if firsts != 1 || publisher.count() != 1 {
		t.Fatalf("concurrent identical raises must store and publish once: created=%d published=%d", firsts, publisher.count())
	}
	event := publisher.events[0]
	stored, err := s.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: decisionHome})
	if err != nil || len(stored.Decisions) != 1 {
		t.Fatalf("list: %v %v", stored, err)
	}
	d := stored.Decisions[0]
	if d.GoalHome != decisionHome || d.EffortRef != effortRef || d.State != DecisionPending || d.PublishedAt == nil || d.EventId != effortdecision.EventID(event) {
		t.Fatalf("stored record: %+v", d)
	}
	if event.EventType != "agent_manager.effort.decision_requested.v1" || event.Payload["recommended"] != "re-aim" || event.Payload["correlation"].(map[string]any)["decision_id"] != "BAS-D-070" {
		t.Fatalf("event is not in the notification-hub decision shape: %+v", event)
	}
	changed := decisionRequest()
	changed.Decision.Question = "Close the goal now?"
	if _, err := s.RaiseDecision(ctx, changed, "run-1"); !errors.Is(err, ErrDecisionChanged) {
		t.Fatalf("a changed question under the same id must be refused: %v", err)
	}
	noDefault := decisionRequest()
	noDefault.Decision.DecisionId, noDefault.Decision.Reversible = "BAS-D-071", false
	if _, err := s.RaiseDecision(ctx, noDefault, "run-1"); err == nil {
		t.Fatal("a default on an irreversible decision must be refused")
	}
}

func TestRaiseDecisionRetriesAnUnpublishedEventOnTick(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	publisher.fail = errors.New("vrooli-events unavailable")
	out, err := s.RaiseDecision(ctx, decisionRequest(), "")
	if err != nil || out.Published || out.Decision.PublishError == "" {
		t.Fatalf("a failed publish must be stored and reported: %+v %v", out, err)
	}
	publisher.fail = nil
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if publisher.count() != 1 {
		t.Fatalf("the retry must publish exactly once: %d", publisher.count())
	}
}

func TestDecisionResolutionsApplyOnceAndNeverReplaceAnOperatorAnswer(t *testing.T) {
	s, _, effortRef, c := decisionFixture(t)
	ctx := context.Background()
	out, err := s.RaiseDecision(ctx, decisionRequest(), "")
	if err != nil {
		t.Fatal(err)
	}
	effort, _, err := s.repo.GetEffort(ctx, effortRef)
	if err != nil {
		t.Fatal(err)
	}
	orchestrator := c.runs[uuid.MustParse(effort.Subjects[0].RunId)]
	orchestrator.Status = domain.RunStatusParked
	resolution := DecisionResolution{AskID: "ask-1", State: "defaulted", Answer: "re-aim", AnsweredAt: "2026-09-05T12:00:00Z", SourceEventType: "agent_manager.effort.decision_requested.v1", RequesterRef: out.Decision.EventId}
	for range 2 {
		if _, err := s.ResolveDecisionFromHub(ctx, resolution); err != nil {
			t.Fatal(err)
		}
	}
	late := DecisionResolution{AskID: "ask-1", State: "answered", Answer: "close", Note: "keep macOS last", AnsweredBy: "operator", AnsweredAt: "2026-09-05T13:00:00Z", Late: true, Correlation: map[string]string{"goal_home": decisionHome, "decision_id": "BAS-D-070"}}
	d, err := s.ResolveDecisionFromHub(ctx, late)
	if err != nil || d.State != DecisionAnswered || d.Answer != "close" || d.AnswerNote != "keep macOS last" || !d.Late || d.AnsweredVia != DecisionViaPhone {
		t.Fatalf("a late operator answer must supersede the default verbatim: %+v %v", d, err)
	}
	// A redelivered default after the operator answered changes nothing.
	if d, err = s.ResolveDecisionFromHub(ctx, resolution); err != nil || d.State != DecisionAnswered || d.Answer != "close" {
		t.Fatalf("a default replaced an operator answer: %+v %v", d, err)
	}
	signals, _, err := s.repo.ListEffortSignals(ctx, effortRef, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 2 || signals[0].Kind != SignalOperatorAnswer || signals[1].Kind != SignalOperatorAnswer {
		t.Fatalf("want one operator_answer per answer (default, then late answer): %+v", signals)
	}
	// The parked orchestrator is woken by the first answer; the late answer
	// finds it running and leaves it alone.
	if c.woken != 1 {
		t.Fatalf("a parked orchestrator must be woken once by an answer, woken %d times", c.woken)
	}
	if unknown, err := s.ResolveDecisionFromHub(ctx, DecisionResolution{State: "answered", RequesterRef: "some-other-ask"}); err != nil || unknown != nil {
		t.Fatalf("an ask Agent Manager did not raise must be ignored: %v %v", unknown, err)
	}
}

func TestMarkDecisionRecordedIsFencedOnTheAnswerTime(t *testing.T) {
	s, _, _, _ := decisionFixture(t)
	ctx := context.Background()
	if _, err := s.RaiseDecision(ctx, decisionRequest(), ""); err != nil {
		t.Fatal(err)
	}
	mark := &pb.MarkEffortDecisionRecordedRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", AnsweredAt: timestamppb.New(time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC))}
	if _, err := s.MarkDecisionRecorded(ctx, mark); !errors.Is(err, ErrDecisionNotAnswered) {
		t.Fatalf("an unanswered decision was marked recorded: %v", err)
	}
	if _, err := s.AnswerDecision(ctx, &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", Answer: "re-aim"}, EffortActor{ID: "agent"}); err == nil {
		t.Fatal("a non-operator answered a decision")
	}
	answered, err := s.AnswerDecision(ctx, &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", Answer: "re-aim", Note: "go"}, EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.MarkDecisionRecorded(ctx, mark); !errors.Is(err, ErrDecisionNotAnswered) {
		t.Fatalf("a stale answer time marked a newer answer recorded: %v", err)
	}
	mark.AnsweredAt = answered.AnsweredAt
	recorded, err := s.MarkDecisionRecorded(ctx, mark)
	if err != nil || recorded.RecordedAt == nil {
		t.Fatalf("mark recorded: %+v %v", recorded, err)
	}
	open, err := s.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: decisionHome, UnrecordedOnly: true})
	if err != nil || len(open.Decisions) != 0 {
		t.Fatalf("a recorded decision is still listed as unrecorded: %v %v", open, err)
	}
}

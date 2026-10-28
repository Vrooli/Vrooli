package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
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
	ownerRunID := uuid.MustParse(e.Subjects[0].GetRunId())
	c.runs[ownerRunID].WorkReferences = []*eventpb.WorkReference{effortReference(e.EffortRef, "orchestrator")}
	s.SetEffortSignalCursorStore(memoryEffortSignalCursorStore{runs: c.runs})
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

func TestDecisionContentDigestPreservesPreContextTerms(t *testing.T) {
	decision := &pb.EffortDecision{
		GoalHome: "scenarios/demo", DecisionId: "A172-legacy", Question: "Continue?",
		Options:     []*pb.EffortDecisionOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}},
		Recommended: "yes", Urgency: "normal", Note: "Legacy technical details.",
	}
	if got := decisionContentDigest(decision); got != "dbb47e39fb9f20df4e9d58c03c309b79564399195c019a38f2cf579fbd3a5912" {
		t.Fatalf("empty additive context fields changed the pre-A172 decision digest: %s", got)
	}
}

func TestRaiseDecisionContextRoundTripsAndRetryKeepsEventBytes(t *testing.T) {
	s, publisher, _, _ := decisionFixture(t)
	ctx := context.Background()
	request := decisionRequest()
	request.Decision.BigPicture = "This determines whether the original quality goal can be accepted."
	request.Decision.TechnicalDetails = "The historical baseline is missing; no replacement run can reconstruct it."
	publisher.fail = errors.New("vrooli-events unavailable")
	first, err := s.RaiseDecision(ctx, request, "run-context")
	if err != nil || first.Published || first.Decision.GetPublishError() == "" {
		t.Fatalf("failed publication should retain the exact decision for retry: %+v %v", first, err)
	}
	listed, err := s.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: decisionHome})
	if err != nil || len(listed.GetDecisions()) != 1 {
		t.Fatalf("decision readback: %+v %v", listed, err)
	}
	stored := listed.GetDecisions()[0]
	if stored.GetBigPicture() != request.Decision.GetBigPicture() || stored.GetTechnicalDetails() != request.Decision.GetTechnicalDetails() {
		t.Fatalf("context fields did not round-trip through the durable record: %+v", stored)
	}
	wantEvent := decisionEvent(stored)
	wantBytes, err := json.Marshal(wantEvent)
	if err != nil {
		t.Fatal(err)
	}
	publisher.fail = nil
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.publishPendingDecisions(ctx); err != nil {
		t.Fatal(err)
	}
	if publisher.count() != 1 {
		t.Fatalf("pending retry should publish exactly once, got %d", publisher.count())
	}
	gotBytes, err := json.Marshal(publisher.events[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBytes) != string(wantBytes) {
		t.Fatalf("retry event changed bytes\nwant %s\n got %s", wantBytes, gotBytes)
	}
	changed := proto.Clone(request).(*pb.RaiseEffortDecisionRequest)
	changed.Decision.BigPicture += " A changed purpose."
	if _, err := s.RaiseDecision(ctx, changed, "run-context"); !errors.Is(err, ErrDecisionChanged) {
		t.Fatalf("changed purpose under an existing decision id must be refused: %v", err)
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
	late := DecisionResolution{AskID: "ask-1", State: "answered", Answer: "close", Note: "keep macOS last", AnsweredBy: "operator", AnsweredAt: "2026-09-05T13:00:00Z", Late: true, RequesterRef: out.Decision.EventId, Correlation: map[string]string{"goal_home": decisionHome, "decision_id": "BAS-D-070"}}
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

func TestDecisionResolutionWithMismatchedRequesterRefIsIgnored(t *testing.T) {
	for _, tc := range []struct {
		name         string
		requesterRef string
	}{
		{name: "missing"},
		{name: "foreign", requesterRef: "different-request-generation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, effortRef, c := decisionFixture(t)
			ctx := context.Background()
			out, err := s.RaiseDecision(ctx, decisionRequest(), "run-generation-binding")
			if err != nil {
				t.Fatal(err)
			}
			effort, _, err := s.repo.GetEffort(ctx, effortRef)
			if err != nil || len(effort.Subjects) == 0 {
				t.Fatalf("read enrolled effort owner: %+v %v", effort, err)
			}
			orchestrator := c.runs[uuid.MustParse(effort.Subjects[0].RunId)]
			if orchestrator == nil {
				t.Fatal("the enrolled orchestrator fixture must exist")
			}
			orchestrator.Status = domain.RunStatusParked
			before, _, err := s.repo.GetEffortDecision(ctx, decisionHome, out.Decision.DecisionId)
			if err != nil {
				t.Fatal(err)
			}
			signalsBefore, _, err := s.repo.ListEffortSignals(ctx, effortRef, 0, 20)
			if err != nil {
				t.Fatal(err)
			}

			resolution := DecisionResolution{
				AskID: "foreign-ask", State: "answered", Answer: "close", AnsweredBy: "operator",
				SourceEventType: goalhome.DecisionRequestedEventType, RequesterRef: tc.requesterRef,
				Correlation: map[string]string{"goal_home": decisionHome, "decision_id": out.Decision.DecisionId},
			}
			got, err := s.ResolveDecisionFromHub(ctx, resolution)
			if err != nil || got != nil {
				t.Fatalf("a resolution with matching logical correlation but %s requester_ref must be ignored: got=%+v err=%v", tc.name, got, err)
			}
			after, _, err := s.repo.GetEffortDecision(ctx, decisionHome, out.Decision.DecisionId)
			if err != nil || !proto.Equal(before, after) {
				t.Fatalf("%s requester_ref mutated the stored decision: before=%+v after=%+v err=%v", tc.name, before, after, err)
			}
			signalsAfter, _, err := s.repo.ListEffortSignals(ctx, effortRef, 0, 20)
			if err != nil || len(signalsAfter) != len(signalsBefore) {
				t.Fatalf("%s requester_ref appended an answer signal: before=%d after=%d err=%v", tc.name, len(signalsBefore), len(signalsAfter), err)
			}
			if c.woken != 0 || orchestrator.Status != domain.RunStatusParked {
				t.Fatalf("%s requester_ref woke parked orchestrator: wake count=%d status=%s", tc.name, c.woken, orchestrator.Status)
			}
		})
	}
}

func TestDecisionAnswerRedeliveryRepairsFailedSignal(t *testing.T) {
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
	resolution := DecisionResolution{
		AskID: "ask-fault", State: "answered", Answer: "re-aim", Note: "retain the same scope",
		AnsweredBy: "operator", AnsweredAt: "2026-09-05T12:00:00Z",
		SourceEventType: "agent_manager.effort.decision_requested.v1", RequesterRef: out.Decision.EventId,
	}
	repo := s.repo.(*Repository)
	if _, err := repo.db.Exec(`CREATE TRIGGER fail_operator_answer BEFORE INSERT ON supervision_effort_signals WHEN NEW.kind='operator_answer' BEGIN SELECT RAISE(FAIL, 'injected signal fault'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveDecisionFromHub(ctx, resolution); err == nil {
		t.Fatal("first resolution must report the signal write failure")
	}
	d, _, err := s.repo.GetEffortDecision(ctx, decisionHome, out.Decision.DecisionId)
	if err != nil || d.State != DecisionAnswered || d.Answer != resolution.Answer {
		t.Fatalf("answer must remain durably recorded across signal failure: %+v %v", d, err)
	}
	if _, err := repo.db.Exec(`DROP TRIGGER fail_operator_answer`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := s.ResolveDecisionFromHub(ctx, resolution); err != nil {
			t.Fatalf("redelivery %d: %v", i, err)
		}
	}
	signals, _, err := s.repo.ListEffortSignals(ctx, effortRef, 0, 10)
	if err != nil || len(signals) != 1 || signals[0].Kind != SignalOperatorAnswer {
		t.Fatalf("redelivery must repair exactly one answer signal: %+v %v", signals, err)
	}
	if c.woken != 1 {
		t.Fatalf("signal repair should wake the parked owner once, got %d wakes", c.woken)
	}
}

func TestDefaultedResolutionRedeliveryRepairsFailedSignal(t *testing.T) {
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
	resolution := DecisionResolution{
		AskID: "ask-default-fault", State: "defaulted", Answer: "re-aim",
		AnsweredAt: "2026-09-05T12:00:00Z", Reason: "deadline elapsed",
		SourceEventType: "agent_manager.effort.decision_requested.v1", RequesterRef: out.Decision.EventId,
	}
	repo := s.repo.(*Repository)
	if _, err := repo.db.Exec(`CREATE TRIGGER fail_operator_answer BEFORE INSERT ON supervision_effort_signals WHEN NEW.kind='operator_answer' BEGIN SELECT RAISE(FAIL, 'injected signal fault'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveDecisionFromHub(ctx, resolution); err == nil {
		t.Fatal("first default must report the signal write failure")
	}
	first, _, err := s.repo.GetEffortDecision(ctx, decisionHome, out.Decision.DecisionId)
	if err != nil || first.State != DecisionDefaulted || first.AnsweredAt == nil {
		t.Fatalf("default must remain durable across signal failure: %+v %v", first, err)
	}
	firstAnsweredAt := first.AnsweredAt.AsTime()
	if _, err := repo.db.Exec(`DROP TRIGGER fail_operator_answer`); err != nil {
		t.Fatal(err)
	}
	resolution.AnsweredAt = "2026-09-05T12:30:00Z"
	resolution.Reason = "redelivered with changed transport metadata"
	for i := 0; i < 2; i++ {
		if _, err := s.ResolveDecisionFromHub(ctx, resolution); err != nil {
			t.Fatalf("default redelivery %d: %v", i, err)
		}
	}
	got, _, err := s.repo.GetEffortDecision(ctx, decisionHome, out.Decision.DecisionId)
	if err != nil || got.State != DecisionDefaulted || got.Answer != resolution.Answer || !got.GetAnsweredAt().AsTime().Equal(firstAnsweredAt) || got.AnsweredVia != DecisionViaDefault {
		t.Fatalf("redelivery must preserve the first default terms and time: %+v %v", got, err)
	}
	signals, _, err := s.repo.ListEffortSignals(ctx, effortRef, 0, 10)
	if err != nil || len(signals) != 1 || signals[0].Kind != SignalOperatorAnswer {
		t.Fatalf("redelivery must repair exactly one default signal: %+v %v", signals, err)
	}
	if c.woken != 1 {
		t.Fatalf("signal repair should wake the parked owner once, got %d wakes", c.woken)
	}
}

func TestDesktopAnswerRedeliveryRepairsFailedSignal(t *testing.T) {
	s, _, effortRef, c := decisionFixture(t)
	ctx := context.Background()
	if _, err := s.RaiseDecision(ctx, decisionRequest(), ""); err != nil {
		t.Fatal(err)
	}
	effort, _, err := s.repo.GetEffort(ctx, effortRef)
	if err != nil {
		t.Fatal(err)
	}
	orchestrator := c.runs[uuid.MustParse(effort.Subjects[0].RunId)]
	orchestrator.Status = domain.RunStatusParked
	repo := s.repo.(*Repository)
	if _, err := repo.db.Exec(`CREATE TRIGGER fail_operator_answer BEFORE INSERT ON supervision_effort_signals WHEN NEW.kind='operator_answer' BEGIN SELECT RAISE(FAIL, 'injected signal fault'); END`); err != nil {
		t.Fatal(err)
	}
	answer := &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: "BAS-D-070", Answer: "re-aim", Note: "preserve this note"}
	actor := EffortActor{ID: "owner", Operator: true}
	if _, err := s.AnswerDecision(ctx, answer, actor); err == nil {
		t.Fatal("first desktop answer must report the signal write failure")
	}
	first, _, err := s.repo.GetEffortDecision(ctx, decisionHome, answer.DecisionId)
	if err != nil || first.State != DecisionAnswered || first.AnswerNote != answer.Note || first.AnsweredAt == nil {
		t.Fatalf("desktop answer must remain durable across signal failure: %+v %v", first, err)
	}
	firstAnsweredAt := first.AnsweredAt.AsTime()
	if _, err := repo.db.Exec(`DROP TRIGGER fail_operator_answer`); err != nil {
		t.Fatal(err)
	}
	answer.Note = "changed duplicate note must not replace original"
	for i := 0; i < 2; i++ {
		if _, err := s.AnswerDecision(ctx, answer, actor); err != nil {
			t.Fatalf("desktop answer retry %d: %v", i, err)
		}
	}
	got, _, err := s.repo.GetEffortDecision(ctx, decisionHome, answer.DecisionId)
	if err != nil || got.AnswerNote != "preserve this note" || !got.GetAnsweredAt().AsTime().Equal(firstAnsweredAt) || got.AnsweredVia != DecisionViaDesktop {
		t.Fatalf("duplicate desktop answer replaced original attribution or terms: %+v %v", got, err)
	}
	signals, _, err := s.repo.ListEffortSignals(ctx, effortRef, 0, 10)
	if err != nil || len(signals) != 1 || signals[0].Kind != SignalOperatorAnswer {
		t.Fatalf("desktop retry must repair exactly one answer signal: %+v %v", signals, err)
	}
	if c.woken != 1 {
		t.Fatalf("signal repair should wake the parked owner once, got %d wakes", c.woken)
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

func TestDecisionFollowsEffortHomeWithoutChangingQuestionOrAnswer(t *testing.T) {
	for _, answered := range []bool{false, true} {
		t.Run(fmt.Sprint(answered), func(t *testing.T) {
			s, publisher, ref, _ := decisionFixture(t)
			ctx := context.Background()
			original, err := s.RaiseDecision(ctx, decisionRequest(), "worker")
			if err != nil {
				t.Fatal(err)
			}
			if answered {
				original.Decision, err = s.AnswerDecision(ctx, &pb.AnswerEffortDecisionRequest{GoalHome: decisionHome, DecisionId: original.Decision.DecisionId, Answer: "close", Note: "retain original answer"}, EffortActor{ID: "owner", Operator: true})
				if err != nil {
					t.Fatal(err)
				}
				original.Decision, err = s.MarkDecisionRecorded(ctx, &pb.MarkEffortDecisionRecordedRequest{GoalHome: decisionHome, DecisionId: original.Decision.DecisionId, AnsweredAt: original.Decision.AnsweredAt})
				if err != nil {
					t.Fatal(err)
				}
			}
			frozen := proto.Clone(original.Decision).(*pb.EffortDecision)
			frozenEvent := effortdecision.EventID(decisionEvent(frozen))
			e, _, err := s.repo.GetEffort(ctx, ref)
			if err != nil {
				t.Fatal(err)
			}
			newHome := ".vrooli/work/delivery/browser-automation-studio-delivery"
			e.DestinationRef = "path:" + newHome + "/CHARTER.md"
			if _, err = s.ReconcileMetadata(ctx, &pb.ReconcileEffortMetadataRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: "move-decision-home"}, EffortActor{ID: "owner", Operator: true}); err != nil {
				t.Fatal(err)
			}
			listed, err := s.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: newHome})
			if err != nil || len(listed.GetDecisions()) != 1 {
				t.Fatalf("moved home lost decision: %v %v", listed, err)
			}
			projected := proto.Clone(frozen).(*pb.EffortDecision)
			projected.GoalHome = newHome
			if !proto.Equal(projected, listed.Decisions[0]) {
				t.Fatalf("answer/provenance changed: %v", listed.Decisions[0])
			}
			// Grow beyond the API's list window before resolving the old ID.
			// Identity lookup must still retain its original answer/event.
			for i := range 210 {
				other := proto.Clone(frozen).(*pb.EffortDecision)
				other.DecisionId = fmt.Sprintf("newer-%d", i)
				other.CreatedAt = timestamppb.New(frozen.CreatedAt.AsTime().Add(time.Duration(i+1) * time.Second))
				other.EventId = effortdecision.EventID(decisionEvent(other))
				if _, err := s.repo.InsertEffortDecision(ctx, other, decisionContentDigest(other)); err != nil {
					t.Fatal(err)
				}
			}
			retry := decisionRequest()
			retry.Decision.GoalHome = newHome
			wrongEffort := proto.Clone(retry).(*pb.RaiseEffortDecisionRequest)
			wrongEffort.Decision.EffortRef = "effort:other"
			if _, err := s.RaiseDecision(ctx, wrongEffort, "worker"); !errors.Is(err, ErrConflict) {
				t.Fatalf("cross-effort re-raise accepted: %v", err)
			}
			reraised, err := s.RaiseDecision(ctx, retry, "new-worker")
			if err != nil || reraised.Created || publisher.count() != 1 {
				t.Fatalf("relocation republished/recreated decision: %v %v count=%d", reraised, err, publisher.count())
			}
			if !proto.Equal(frozen, reraised.Decision) {
				t.Fatal("re-raise changed original record")
			}
			if answered {
				marked, err := s.MarkDecisionRecorded(ctx, &pb.MarkEffortDecisionRecordedRequest{GoalHome: newHome, DecisionId: frozen.DecisionId, AnsweredAt: frozen.AnsweredAt})
				if err != nil || !proto.Equal(frozen, marked) {
					t.Fatalf("moved-home mark changed answer: %v %v", marked, err)
				}
				if _, err = s.AnswerDecision(ctx, &pb.AnswerEffortDecisionRequest{GoalHome: newHome, DecisionId: frozen.DecisionId, Answer: "re-aim"}, EffortActor{ID: "owner", Operator: true}); !errors.Is(err, ErrDecisionAnswered) {
					t.Fatalf("contradictory answer allowed after move: %v", err)
				}
			}
			stored, _, err := s.repo.GetEffortDecision(ctx, decisionHome, frozen.DecisionId)
			if err != nil || effortdecision.EventID(decisionEvent(stored)) != frozenEvent {
				t.Fatal("original publication identity changed")
			}
			if _, err = s.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: newHome, EffortRef: "effort:other"}); err == nil {
				t.Fatal("cross-effort lookup accepted")
			}
			conflict := proto.Clone(frozen).(*pb.EffortDecision)
			conflict.GoalHome = newHome
			conflict.EventId = effortdecision.EventID(decisionEvent(conflict))
			if _, err := s.repo.InsertEffortDecision(ctx, conflict, decisionContentDigest(conflict)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: newHome}); !errors.Is(err, ErrConflict) {
				t.Fatalf("list projected ambiguous historical answers: %v", err)
			}
			if _, _, err := s.decisionForHome(ctx, newHome, frozen.DecisionId); !errors.Is(err, ErrConflict) {
				t.Fatalf("direct-home row hid ambiguous original decision: %v", err)
			}
		})
	}
}

func TestDecisionRecoveryPaginationIncludesOldRecordedPolicyAndRejectsChangedScope(t *testing.T) {
	s, _, ref, _ := decisionFixture(t)
	ctx := context.Background()
	for i := range 211 {
		d := proto.Clone(decisionRequest().Decision).(*pb.EffortDecision)
		d.GoalHome, d.EffortRef, d.DecisionId = decisionHome, ref, fmt.Sprintf("policy-%03d", i)
		d.State, d.Answer = DecisionAnswered, "close"
		d.AnsweredAt, d.RecordedAt = timestamppb.Now(), timestamppb.Now()
		if _, err := s.repo.InsertEffortDecision(ctx, d, decisionContentDigest(d)); err != nil {
			t.Fatal(err)
		}
	}
	seen, pages := map[string]bool{}, 0
	request := &pb.ListEffortDecisionsRequest{EffortRef: ref, Limit: 200}
	for {
		page, err := s.ListDecisions(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		for _, d := range page.Decisions {
			if seen[d.DecisionId] {
				t.Fatalf("decision repeated across pages: %s", d.DecisionId)
			}
			seen[d.DecisionId] = true
		}
		if page.NextPageToken == "" {
			break
		}
		changed := &pb.ListEffortDecisionsRequest{EffortRef: "effort:other", Limit: 200, PageToken: page.NextPageToken}
		if _, err := s.ListDecisions(ctx, changed); err == nil {
			t.Fatal("pagination accepted changed scope")
		}
		request.PageToken = page.NextPageToken
	}
	if len(seen) != 211 || !seen["policy-000"] || !seen["policy-210"] || pages != 2 {
		t.Fatalf("incomplete policy recovery: %d decisions, %d pages", len(seen), pages)
	}
	s.config.ProjectRoot = t.TempDir()
	goalHomeTree(t, s.config.ProjectRoot, nil)
	enrollment, _, err := s.repo.GetEffort(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	projected := s.projectGoalHome(ctx, enrollment, &pb.EffortBoardRow{})
	if projected.State != "partial" || !strings.Contains(strings.Join(projected.Limitations, " "), "exceeds this board projection") {
		t.Fatalf("truncated decision history looked complete: %+v", projected)
	}

}

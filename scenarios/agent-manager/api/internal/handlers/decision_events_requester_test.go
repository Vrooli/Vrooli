package handlers

import (
	"context"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// fakeWaker records how each run was reached: WakeRun (parked) vs ContinueRun.
type fakeWaker struct {
	runs      map[uuid.UUID]*domain.Run
	woke      map[uuid.UUID]string
	continued map[uuid.UUID]string
	prompted  map[string]string // web-console session id → text
	continErr error
}

func newFakeWaker(runs ...*domain.Run) *fakeWaker {
	m := map[uuid.UUID]*domain.Run{}
	for _, r := range runs {
		m[r.ID] = r
	}
	return &fakeWaker{runs: m, woke: map[uuid.UUID]string{}, continued: map[uuid.UUID]string{}, prompted: map[string]string{}}
}

func (f *fakeWaker) GetRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	run, ok := f.runs[id]
	if !ok {
		return nil, &domain.NotFoundError{}
	}
	return run, nil
}

func (f *fakeWaker) WakeRun(_ context.Context, in orchestration.WakeRunInput) (*domain.Run, error) {
	f.woke[in.RunID] = in.Result
	return nil, nil
}

func (f *fakeWaker) ContinueRun(_ context.Context, req orchestration.ContinueRunRequest) (*domain.Run, error) {
	if f.continErr != nil {
		return nil, f.continErr
	}
	f.continued[req.RunID] = req.Message
	return nil, nil
}

func (f *fakeWaker) SendInteractivePrompt(_ context.Context, sessionID, text, _ string) error {
	f.prompted[sessionID] = text
	return nil
}

func runIn(status domain.RunStatus) *domain.Run {
	return &domain.Run{ID: uuid.New(), Status: status}
}

func parkedOn(producer, key string) *domain.Run {
	return &domain.Run{
		ID:          uuid.New(),
		Status:      domain.RunStatusParked,
		AwaitHandle: &domain.AwaitHandle{Producer: producer, Key: key},
	}
}

func resolution(runID, awaitKey string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		State: "answered", Answer: "approve", AnswerLabel: "Approve deletion", AnsweredBy: "op",
		Correlation: map[string]string{
			operatordecision.CorrelationRaisedByRunID: runID,
			operatordecision.CorrelationAgentAwaitKey: awaitKey,
		},
	}
}

func TestRequesterApplyPassesThroughWithoutRunID(t *testing.T) {
	e := requesterWakeEffect{waker: newFakeWaker()}
	handled, err := e.Apply(context.Background(), operatordecision.DecisionResolution{State: "answered", Answer: "approve"})
	if handled || err != nil {
		t.Fatalf("no run id ⇒ pass-through (false,nil), got handled=%v err=%v", handled, err)
	}
}

func TestRequesterApplyNeverClaimsResolution(t *testing.T) {
	run := parkedOn(orchestration.ProducerOperatorDecision, "d-1")
	e := requesterWakeEffect{waker: newFakeWaker(run)}
	handled, err := e.Apply(context.Background(), resolution(run.ID.String(), "d-1"))
	if handled || err != nil {
		t.Fatalf("observer must not claim the resolution, got handled=%v err=%v", handled, err)
	}
}

func TestDeliverWakesRunParkedOnThisDecision(t *testing.T) {
	run := parkedOn(orchestration.ProducerOperatorDecision, "d-1")
	f := newFakeWaker(run)
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "", "d-1", resolution(run.ID.String(), "d-1"))
	if msg, ok := f.woke[run.ID]; !ok || !strings.Contains(msg, "operator decision resolved") {
		t.Fatalf("parked-on-decision run must be woken with the answer, got ok=%v msg=%q", ok, msg)
	}
	if len(f.continued) != 0 {
		t.Fatal("a parked run must be woken, not continued")
	}
}

func TestDeliverDoesNotDisturbRunParkedOnOtherWork(t *testing.T) {
	run := parkedOn(orchestration.ProducerChildren, "something")
	f := newFakeWaker(run)
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "", "d-1", resolution(run.ID.String(), "d-1"))
	if len(f.woke) != 0 || len(f.continued) != 0 {
		t.Fatalf("a run parked on unrelated work must be left alone, woke=%v continued=%v", f.woke, f.continued)
	}
}

func TestDeliverContinuesRunningInteractiveRun(t *testing.T) {
	run := runIn(domain.RunStatusRunning)
	f := newFakeWaker(run)
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "", "d-1", resolution(run.ID.String(), "d-1"))
	if msg, ok := f.continued[run.ID]; !ok || !strings.Contains(msg, "operator decision resolved") {
		t.Fatalf("a running run must be delivered via ContinueRun, got ok=%v msg=%q", ok, msg)
	}
	if len(f.woke) != 0 {
		t.Fatal("a running run must not be woken")
	}
}

func TestDeliverContinuesTerminalRun(t *testing.T) {
	run := runIn(domain.RunStatusComplete)
	f := newFakeWaker(run)
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "", "d-1", resolution(run.ID.String(), "d-1"))
	if _, ok := f.continued[run.ID]; !ok {
		t.Fatal("a terminal run must be delivered via ContinueRun (continuation)")
	}
}

func TestDeliverSkipsRunAlreadyWokenForThisDecision(t *testing.T) {
	// A redelivery after the run was already woken for this decision must not
	// then also continue it (double-delivery).
	run := runIn(domain.RunStatusRunning)
	run.LastAwaitKey = orchestration.ProducerOperatorDecision + ":d-1"
	f := newFakeWaker(run)
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "", "d-1", resolution(run.ID.String(), "d-1"))
	if len(f.continued) != 0 {
		t.Fatalf("already-woken run must not be re-delivered, continued=%v", f.continued)
	}
}

func TestDeliverUnknownRunIsBenign(t *testing.T) {
	f := newFakeWaker()
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), uuid.New().String(), "", "d-1", resolution(uuid.New().String(), "d-1"))
	if len(f.woke) != 0 || len(f.continued) != 0 {
		t.Fatal("an unknown run must be a benign no-op")
	}
}

func sessionResolution(wcSession string) operatordecision.DecisionResolution {
	return operatordecision.DecisionResolution{
		State: "answered", Answer: "approve", AnswerLabel: "Approve deletion", AnsweredBy: "op",
		Correlation: map[string]string{operatordecision.CorrelationWebConsoleSessionID: wcSession},
	}
}

func TestDeliverUsesSessionLaneWhenNoRunID(t *testing.T) {
	// A web-console-hosted session whose token expired carries only the stable
	// session id — deliver straight to it via SendInteractivePrompt.
	f := newFakeWaker()
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), "", "wc-858b", "", sessionResolution("wc-858b"))
	if msg, ok := f.prompted["wc-858b"]; !ok || !strings.Contains(msg, "operator decision resolved") {
		t.Fatalf("expected direct session delivery, got ok=%v msg=%q", ok, msg)
	}
}

func TestDeliverFallsBackToSessionLaneWhenRunUnknown(t *testing.T) {
	// A run id that no longer resolves (e.g. ended) falls back to the stable
	// session id if one is present.
	f := newFakeWaker()
	gone := uuid.New().String()
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), gone, "wc-858b", "d-1",
		operatordecision.DecisionResolution{State: "answered", Answer: "approve", Correlation: map[string]string{
			operatordecision.CorrelationRaisedByRunID:       gone,
			operatordecision.CorrelationWebConsoleSessionID: "wc-858b",
			operatordecision.CorrelationAgentAwaitKey:       "d-1",
		}})
	if _, ok := f.prompted["wc-858b"]; !ok {
		t.Fatal("an unresolvable run must fall back to the session lane")
	}
}

func TestDeliverPrefersRunLaneOverSessionLane(t *testing.T) {
	// When the run resolves, deliver via the run (which itself reaches the WC
	// session); do NOT also send a direct prompt (would double-deliver).
	run := runIn(domain.RunStatusRunning)
	f := newFakeWaker(run)
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "wc-858b", "d-1",
		operatordecision.DecisionResolution{State: "answered", Answer: "approve", Correlation: map[string]string{
			operatordecision.CorrelationRaisedByRunID:       run.ID.String(),
			operatordecision.CorrelationWebConsoleSessionID: "wc-858b",
			operatordecision.CorrelationAgentAwaitKey:       "d-1",
		}})
	if _, ok := f.continued[run.ID]; !ok {
		t.Fatal("a resolvable run must be delivered via the run lane")
	}
	if len(f.prompted) != 0 {
		t.Fatalf("must not also direct-prompt when the run lane handled it, prompted=%v", f.prompted)
	}
}

func TestApplyTriggersForSessionOnlyResolution(t *testing.T) {
	// A resolution carrying only a web-console session id (no run id) must still
	// trigger the tie-back (pass-through only when BOTH are absent).
	e := requesterWakeEffect{waker: newFakeWaker()}
	handled, err := e.Apply(context.Background(), sessionResolution("wc-858b"))
	if handled || err != nil {
		t.Fatalf("observer must not claim, got handled=%v err=%v", handled, err)
	}
}

func TestDeliverToleratesContinueRefusal(t *testing.T) {
	// A running non-interactive run refuses mid-turn text; delivery must swallow
	// it (no panic, no error path).
	run := runIn(domain.RunStatusRunning)
	f := newFakeWaker(run)
	f.continErr = context.DeadlineExceeded
	requesterWakeEffect{waker: f}.deliverToRequester(context.Background(), run.ID.String(), "", "d-1", resolution(run.ID.String(), "d-1"))
}

func TestFormatDecisionWakeMessage(t *testing.T) {
	approved := formatDecisionWakeMessage(operatordecision.DecisionResolution{
		State: "answered", Answer: "approve", AnswerLabel: "Approve deletion", AnsweredBy: "matthalloran8",
	})
	if !strings.Contains(approved, "answered by matthalloran8") || !strings.Contains(approved, "Approve deletion") || !strings.Contains(approved, "do NOT re-raise") {
		t.Fatalf("answered message missing expected content: %q", approved)
	}
	expired := formatDecisionWakeMessage(operatordecision.DecisionResolution{State: "expired", Reason: "deadline"})
	if !strings.Contains(expired, "EXPIRED") || !strings.Contains(expired, "NOT granted") {
		t.Fatalf("expired message missing expected content: %q", expired)
	}
	defaulted := formatDecisionWakeMessage(operatordecision.DecisionResolution{State: "defaulted", Answer: "deny", AnswerLabel: "Do not delete"})
	if !strings.Contains(defaulted, "default answer") || !strings.Contains(defaulted, "Do not delete") {
		t.Fatalf("defaulted message missing expected content: %q", defaulted)
	}
}

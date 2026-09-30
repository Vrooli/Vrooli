package attempt

import (
	"context"
	"testing"
)

func TestRouterRoutesOnlyRegisteredTypedSubjects(t *testing.T) {
	router := NewRouter()
	if err := router.Register("fixture", DeciderFunc(func(_ context.Context, request DecisionRequest) (DecisionResult, error) {
		return DecisionResult{Decision: request.Decision, Status: "applied"}, nil
	})); err != nil {
		t.Fatal(err)
	}
	result, err := router.DecideAttempt(context.Background(), DecisionRequest{SubjectKind: "fixture", SubjectRef: "one", RoundNum: 1, Decision: "accept", Actor: "operator@example.test"})
	if err != nil || result.Status != "applied" {
		t.Fatalf("DecideAttempt() = %#v, %v", result, err)
	}
	if _, err := router.DecideAttempt(context.Background(), DecisionRequest{SubjectKind: "unknown", SubjectRef: "one", RoundNum: 1, Decision: "accept", Actor: "operator@example.test"}); err == nil {
		t.Fatal("unregistered subject was accepted")
	}
}

// TestDecisionStateMatrixRoutesEveryDriverAcrossAttemptStatuses keeps the
// Connect decision envelope honest as new domains join it. Domain adapters own
// their storage-specific transitions; this matrix proves every lifecycle state
// is delivered to every acting driver without requiring a live agent workflow.
func TestDecisionStateMatrixRoutesEveryDriverAcrossAttemptStatuses(t *testing.T) {
	type driver struct {
		subjectKind string
		decision    string
		terminal    string
	}
	drivers := []driver{
		{subjectKind: "backlog-item", decision: "accept", terminal: "completed"},
		{subjectKind: "agent-session-proposal", decision: "accept", terminal: "applied"},
		{subjectKind: "plan-workshop-candidate", decision: "drop", terminal: "discarded"},
	}
	statuses := []string{"gathering", "ready", "needs_followup", "failed", "completed"}
	router := NewRouter()
	for _, d := range drivers {
		driver := d
		if err := router.Register(driver.subjectKind, DeciderFunc(func(_ context.Context, request DecisionRequest) (DecisionResult, error) {
			return DecisionResult{Decision: request.Decision, Status: driver.terminal}, nil
		})); err != nil {
			t.Fatal(err)
		}
	}
	for _, status := range statuses {
		for _, driver := range drivers {
			t.Run(status+"/"+driver.subjectKind, func(t *testing.T) {
				result, err := router.DecideAttempt(context.Background(), DecisionRequest{
					SubjectKind: driver.subjectKind,
					SubjectRef:  "fixture/" + status,
					RoundNum:    1,
					Decision:    driver.decision,
					Actor:       "operator@example.test",
				})
				if err != nil {
					t.Fatal(err)
				}
				if result.Status != driver.terminal {
					t.Fatalf("terminal status = %q, want %q", result.Status, driver.terminal)
				}
			})
		}
	}
}

package checks

import (
	"context"
	"testing"
	"time"
)

type healIncidentReporterSpy struct {
	opened   []healIncidentRecord
	resolved []string
}

type healIncidentRecord struct {
	checkID             string
	actionID            string
	lastError           string
	consecutiveFailures int
}

func (s *healIncidentReporterSpy) OpenHealIncident(_ context.Context, checkID, actionID, lastError string, consecutiveFailures int) error {
	s.opened = append(s.opened, healIncidentRecord{
		checkID: checkID, actionID: actionID, lastError: lastError, consecutiveFailures: consecutiveFailures,
	})
	return nil
}

func (s *healIncidentReporterSpy) ResolveHealIncident(_ context.Context, checkID, actionID string) error {
	s.resolved = append(s.resolved, checkID+":"+actionID)
	return nil
}

func newIncidentEscalationRegistry(reporter HealIncidentReporter, actionResult ActionResult) (*Registry, *fixedClock) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(100, 0)}
	reg.SetClock(clock)
	reg.SetHealIncidentReporter(reporter)
	reg.SetConfigProvider(&mockConfigProvider{
		enabledChecks:  map[string]bool{"resource-qdrant": true},
		autoHealChecks: map[string]bool{"resource-qdrant": true},
	})
	reg.Register(&mockHealableCheck{
		id:            "resource-qdrant",
		result:        Result{CheckID: "resource-qdrant", Status: StatusCritical, Message: "qdrant stopped"},
		actions:       []RecoveryAction{{ID: "start", Available: true}},
		executeResult: actionResult,
	})
	return reg, clock
}

func TestRunAutoHealEscalatesOnceThresholdIsReachedAndResolvesOnSuccess(t *testing.T) {
	reporter := &healIncidentReporterSpy{}
	reg, clock := newIncidentEscalationRegistry(reporter, ActionResult{
		Success: false,
		Error:   "Runtime error: start qdrant: managed-service health check did not pass before startup timeout",
		Message: "qdrant start failed",
	})
	result := Result{CheckID: "resource-qdrant", Status: StatusCritical, Message: "qdrant stopped"}

	for i := 0; i < 3; i++ {
		reg.RunAutoHeal(context.Background(), []Result{result})
		clock.current = clock.current.Add(10 * time.Minute)
	}

	if len(reporter.opened) != 1 {
		t.Fatalf("opened incidents = %d, want exactly one", len(reporter.opened))
	}
	opened := reporter.opened[0]
	if opened.checkID != "resource-qdrant" || opened.actionID != "start" {
		t.Fatalf("incident identity = %q/%q, want resource-qdrant/start", opened.checkID, opened.actionID)
	}
	if opened.lastError != "Runtime error: start qdrant: managed-service health check did not pass before startup timeout" {
		t.Fatalf("incident last error = %q, want verbatim action error", opened.lastError)
	}

	check, _ := reg.GetHealableCheck("resource-qdrant")
	_ = check
	// Replace the action outcome while retaining the tracker and incident.
	reg.Unregister("resource-qdrant")
	reg.Register(&mockHealableCheck{
		id:            "resource-qdrant",
		result:        Result{CheckID: "resource-qdrant", Status: StatusCritical, Message: "qdrant stopped"},
		actions:       []RecoveryAction{{ID: "start", Available: true}},
		executeResult: ActionResult{Success: true, Message: "qdrant started"},
	})
	reg.RunAutoHeal(context.Background(), []Result{result})

	if len(reporter.resolved) != 1 || reporter.resolved[0] != "resource-qdrant:start" {
		t.Fatalf("resolved incidents = %#v, want resource-qdrant:start", reporter.resolved)
	}
}

func TestRunAutoHealDoesNotEscalateBelowFailureThreshold(t *testing.T) {
	reporter := &healIncidentReporterSpy{}
	reg, clock := newIncidentEscalationRegistry(reporter, ActionResult{Error: "temporary start failure"})
	result := Result{CheckID: "resource-qdrant", Status: StatusCritical, Message: "qdrant stopped"}
	for i := 0; i < 2; i++ {
		reg.RunAutoHeal(context.Background(), []Result{result})
		clock.current = clock.current.Add(10 * time.Minute)
	}
	if len(reporter.opened) != 0 {
		t.Fatalf("opened incidents = %d, want none below threshold", len(reporter.opened))
	}
}

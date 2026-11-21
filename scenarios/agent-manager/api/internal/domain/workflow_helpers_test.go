package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorkflowTerminalStatusAndTriggerDefaults(t *testing.T) {
	for _, status := range []WorkflowExecutionStatus{WorkflowExecutionSucceeded, WorkflowExecutionBlocked, WorkflowExecutionAbstained, WorkflowExecutionBudgetExhausted, WorkflowExecutionFailed, WorkflowExecutionCancelled} {
		if !status.Terminal() {
			t.Fatalf("terminal status %q reported non-terminal", status)
		}
	}
	for _, status := range []WorkflowExecutionStatus{WorkflowExecutionPending, WorkflowExecutionRunning, WorkflowExecutionWaiting, WorkflowExecutionCancelling} {
		if status.Terminal() {
			t.Fatalf("active status %q reported terminal", status)
		}
	}
	defaultPolicy := WorkflowTriggerPolicy{}
	if !defaultPolicy.Allows(WorkflowInitiatorHuman) || defaultPolicy.SelfTriggerMode() != WorkflowSelfTriggerDeny {
		t.Fatalf("unsafe trigger defaults: %+v", defaultPolicy)
	}
	policy := WorkflowTriggerPolicy{Initiators: []WorkflowInitiator{WorkflowInitiatorAgent}, SelfTrigger: WorkflowSelfTriggerPolicy{Mode: WorkflowSelfTriggerAllow, MaxDepth: 2}}
	if !policy.Allows(WorkflowInitiatorAgent) || policy.Allows(WorkflowInitiatorHuman) || policy.SelfTriggerMode() != WorkflowSelfTriggerAllow {
		t.Fatalf("trigger policy did not enforce author intent: %+v", policy)
	}
}

func TestWorkflowExecutionTransitionMatrix(t *testing.T) {
	statuses := []WorkflowExecutionStatus{
		WorkflowExecutionPending, WorkflowExecutionRunning, WorkflowExecutionWaiting,
		WorkflowExecutionCancelling, WorkflowExecutionSucceeded, WorkflowExecutionBlocked,
		WorkflowExecutionAbstained, WorkflowExecutionBudgetExhausted,
		WorkflowExecutionFailed, WorkflowExecutionCancelled,
	}
	allowed := map[WorkflowExecutionStatus]map[WorkflowExecutionStatus]bool{
		WorkflowExecutionPending: {
			WorkflowExecutionPending: true, WorkflowExecutionRunning: true,
			WorkflowExecutionCancelling: true, WorkflowExecutionFailed: true,
			WorkflowExecutionBudgetExhausted: true,
		},
		WorkflowExecutionRunning: {
			WorkflowExecutionRunning: true, WorkflowExecutionWaiting: true,
			WorkflowExecutionCancelling: true, WorkflowExecutionSucceeded: true,
			WorkflowExecutionBlocked: true, WorkflowExecutionAbstained: true,
			WorkflowExecutionBudgetExhausted: true, WorkflowExecutionFailed: true,
		},
		WorkflowExecutionWaiting: {
			WorkflowExecutionWaiting: true, WorkflowExecutionRunning: true,
			WorkflowExecutionCancelling: true, WorkflowExecutionSucceeded: true,
			WorkflowExecutionBlocked: true, WorkflowExecutionAbstained: true,
			WorkflowExecutionBudgetExhausted: true, WorkflowExecutionFailed: true,
			WorkflowExecutionCancelled: true,
		},
		WorkflowExecutionCancelling:      {WorkflowExecutionCancelling: true, WorkflowExecutionCancelled: true},
		WorkflowExecutionSucceeded:       {WorkflowExecutionSucceeded: true},
		WorkflowExecutionBlocked:         {WorkflowExecutionBlocked: true},
		WorkflowExecutionAbstained:       {WorkflowExecutionAbstained: true},
		WorkflowExecutionBudgetExhausted: {WorkflowExecutionBudgetExhausted: true},
		WorkflowExecutionFailed:          {WorkflowExecutionFailed: true, WorkflowExecutionRunning: true},
		WorkflowExecutionCancelled:       {WorkflowExecutionCancelled: true},
	}
	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				if got := from.CanTransitionTo(to); got != allowed[from][to] {
					t.Fatalf("CanTransitionTo(%q, %q) = %v", from, to, got)
				}
			})
		}
	}
	if WorkflowExecutionStatus("corrupt").CanTransitionTo(WorkflowExecutionStatus("corrupt")) ||
		WorkflowExecutionRunning.CanTransitionTo(WorkflowExecutionStatus("corrupt")) ||
		WorkflowExecutionStatus("corrupt").CanTransitionTo(WorkflowExecutionRunning) {
		t.Fatal("unknown status was accepted by the transition graph")
	}
	execution := &WorkflowExecution{Status: WorkflowExecutionSucceeded}
	if err := execution.TransitionStatus(WorkflowExecutionRunning); !errors.Is(err, ErrInvalidWorkflowExecutionStatusTransition) {
		t.Fatal("terminal execution was resurrected outside retry")
	}
	if execution.Status != WorkflowExecutionSucceeded {
		t.Fatalf("rejected transition mutated source to %q", execution.Status)
	}
}

func TestFinalizationMarkersPreserveTerminalOutcomeAndAuditTimes(t *testing.T) {
	run := &Run{}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	MarkFinalizationRunning(run, now)
	if run.FinalizationStatus != RunFinalizationStatusRunning || run.FinalizedAt != nil || !run.UpdatedAt.Equal(now) {
		t.Fatalf("running marker = %+v", run)
	}
	MarkFinalizationSucceeded(run, now.Add(time.Second))
	if run.FinalizationStatus != RunFinalizationStatusSucceeded || run.FinalizedAt == nil || run.FinalizationError != "" {
		t.Fatalf("succeeded marker = %+v", run)
	}
	MarkFinalizationSkipped(run, "  no sandbox changes  ", now.Add(2*time.Second))
	if run.FinalizationStatus != RunFinalizationStatusSkipped || run.FinalizationError != "no sandbox changes" {
		t.Fatalf("skipped marker = %+v", run)
	}
	MarkFinalizationFailed(run, errors.New("  copy failed  "), now.Add(3*time.Second))
	if run.FinalizationStatus != RunFinalizationStatusFailed || run.FinalizationError != "copy failed" || run.FinalizedAt == nil {
		t.Fatalf("failed marker = %+v", run)
	}
	MarkFinalizationFailed(run, nil, now.Add(4*time.Second))
	if run.FinalizationError != "" {
		t.Fatalf("nil failure error = %q", run.FinalizationError)
	}
	MarkFinalizationRunning(nil, now)
	MarkFinalizationSucceeded(nil, now)
	MarkFinalizationSkipped(nil, "ignored", now)
	MarkFinalizationFailed(nil, errors.New("ignored"), now)
}

func TestValidateEventsCountsTypedWarningsWithoutRejectingEvents(t *testing.T) {
	events := []*RunEvent{
		nil,
		{EventType: EventTypeToolCall, Data: &ToolCallEventData{ToolName: "unknown_tool"}},
		{EventType: EventTypeToolResult, Data: &ToolResultEventData{}},
		{EventType: EventTypeMessage, Data: &MessageEventData{Content: ""}},
		{EventType: EventTypeMetric, Data: &UsageEventData{}},
		{EventType: EventTypeError},
		{EventType: EventTypeLog},
		{EventType: EventTypeMessageDeleted, Data: &MessageDeletedEventData{}},
	}
	stats := ValidateEvents(events)
	if stats.TotalEvents != len(events) || stats.ToolCallCount != 1 || stats.ToolResultCount != 1 || stats.MessageCount != 1 || stats.MetricCount != 1 || stats.ErrorCount != 1 || stats.LogCount != 1 {
		t.Fatalf("event counts = %+v", stats)
	}
	if stats.EmptyToolNames != 1 || stats.EmptyInputs != 1 || stats.EmptyMessages != 1 || stats.WarningCount != 3 {
		t.Fatalf("event warnings = %+v", stats)
	}
	// Validation is diagnostic only; every supported event shape must remain
	// safe to process, including legacy and malformed payloads.
	for _, event := range events {
		ValidateEvent(event)
	}
	ValidateEvent(&RunEvent{EventType: EventTypeMetric, Data: &MetricEventData{}})
	ValidateEvent(&RunEvent{EventType: EventTypeMetric, Data: &MessageEventData{}})
}

func TestValidateEventsNeverCountsQuotaObservationsAsErrors(t *testing.T) {
	exhausted := 100.0
	stats := ValidateEvents([]*RunEvent{
		NewRateLimitEvent(uuid.New(), "300m", "", nil, 0),
		{EventType: EventTypeQuota, Data: &RateLimitEventData{Provider: "openai", Pool: "primary", UsedPercent: &exhausted}},
		NewErrorEvent(uuid.New(), "execution_error", "failed", false),
	})
	if stats.ErrorCount != 1 {
		t.Fatalf("quota observations counted as errors: %+v", stats)
	}
}

package tasks

import (
	"context"
	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/tasks/fix"
	"scenario-to-desktop-api/tasks/shared"
	"strings"
	"testing"
	"time"
)

func TestShutdownCancelsActiveWorkersAndRejectsNewTasks(t *testing.T) {
	svc, _, pipelineStore, _, _ := newTestService()
	pipelineStore.statuses["pipe-1"] = &pipeline.Status{PipelineID: "pipe-1"}

	if _, err := svc.TriggerTask(context.Background(), validInvestigateRequest("pipe-1")); err != nil {
		t.Fatalf("TriggerTask() error = %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if _, err := svc.TriggerTask(context.Background(), validInvestigateRequest("pipe-1")); err == nil || !strings.Contains(err.Error(), "shutting down") {
		t.Fatalf("TriggerTask() after Shutdown() error = %v, want shutdown rejection", err)
	}
}

// --- StopTask tests ---

func TestStopTask_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	err := svc.StopTask(ctx, "pipe-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing task")
	}
	if !strings.Contains(err.Error(), "task not found") {
		t.Errorf("error = %q, want 'task not found'", err.Error())
	}
}

func TestStopTask_NotRunning(t *testing.T) {
	svc, invStore, _, _, _ := newTestService()
	ctx := context.Background()

	invStore.investigations["inv-1"] = &domain.Investigation{
		ID:         "inv-1",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusCompleted,
	}

	err := svc.StopTask(ctx, "pipe-1", "inv-1")
	if err == nil {
		t.Fatal("expected error for non-running task")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error = %q, want 'not running'", err.Error())
	}
}

func TestStopTask_Running(t *testing.T) {
	svc, invStore, _, _, hub := newTestService()
	ctx := context.Background()

	cancelled := false
	invStore.investigations["inv-1"] = &domain.Investigation{
		ID:         "inv-1",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusRunning,
	}
	invStore.cancelFuncs["inv-1"] = func() { cancelled = true }

	err := svc.StopTask(ctx, "pipe-1", "inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cancelled {
		t.Error("cancel function should have been called")
	}
	if invStore.investigations["inv-1"].Status != domain.InvestigationStatusCancelled {
		t.Errorf("status = %q, want cancelled", invStore.investigations["inv-1"].Status)
	}
	// Verify progress event was broadcast
	found := false
	for _, e := range hub.events {
		if e.eventType == EventTaskStopped {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected EventTaskStopped broadcast")
	}
}

func TestStopTask_PendingTask(t *testing.T) {
	svc, invStore, _, _, _ := newTestService()
	ctx := context.Background()

	invStore.investigations["inv-1"] = &domain.Investigation{
		ID:         "inv-1",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusPending,
	}

	err := svc.StopTask(ctx, "pipe-1", "inv-1")
	if err != nil {
		t.Fatalf("should be able to stop pending task: %v", err)
	}
}

func TestStopTask_WithAgentRunID(t *testing.T) {
	svc, invStore, _, _, _ := newTestService()
	ctx := context.Background()

	runID := "agent-run-xyz"
	invStore.investigations["inv-1"] = &domain.Investigation{
		ID:         "inv-1",
		PipelineID: "pipe-1",
		Status:     domain.InvestigationStatusRunning,
		AgentRunID: &runID,
	}

	err := svc.StopTask(ctx, "pipe-1", "inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetTask / ListTasks tests ---

func TestGetTask(t *testing.T) {
	svc, invStore, _, _, _ := newTestService()
	ctx := context.Background()

	invStore.investigations["inv-1"] = &domain.Investigation{
		ID:         "inv-1",
		PipelineID: "pipe-1",
	}

	inv, err := svc.GetTask(ctx, "pipe-1", "inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected non-nil investigation")
	}
	if inv.ID != "inv-1" {
		t.Errorf("ID = %q, want inv-1", inv.ID)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	inv, err := svc.GetTask(ctx, "pipe-1", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv != nil {
		t.Error("expected nil for missing investigation")
	}
}

func TestListTasks(t *testing.T) {
	svc, invStore, _, _, _ := newTestService()
	ctx := context.Background()

	invStore.investigations["inv-1"] = &domain.Investigation{ID: "inv-1", PipelineID: "pipe-1"}
	invStore.investigations["inv-2"] = &domain.Investigation{ID: "inv-2", PipelineID: "pipe-1"}
	invStore.investigations["inv-3"] = &domain.Investigation{ID: "inv-3", PipelineID: "pipe-2"}

	results, err := svc.ListTasks(ctx, "pipe-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len = %d, want 2", len(results))
	}
}

// --- IsAgentAvailable / GetAgentManagerURL tests ---

func TestIsAgentAvailable(t *testing.T) {
	svc, _, _, agentExec, _ := newTestService()
	ctx := context.Background()

	agentExec.available = true
	if !svc.IsAgentAvailable(ctx) {
		t.Error("should be available")
	}

	agentExec.available = false
	if svc.IsAgentAvailable(ctx) {
		t.Error("should not be available")
	}
}

func TestGetAgentManagerURL(t *testing.T) {
	svc, _, _, agentExec, _ := newTestService()
	ctx := context.Background()

	agentExec.url = "http://agent:8080"
	url, err := svc.GetAgentManagerURL(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://agent:8080" {
		t.Errorf("url = %q, want http://agent:8080", url)
	}
}

// --- buildFixFindingsSummary tests ---

func TestBuildFixFindingsSummary_Empty(t *testing.T) {
	state := &fix.LoopState{
		FinalStatus: "success",
		Config:      fix.LoopConfig{MaxIterations: 5},
	}
	got := buildFixFindingsSummary(state)
	if !strings.Contains(got, "# Fix Task Summary") {
		t.Error("should contain header")
	}
	if !strings.Contains(got, "**Status:** success") {
		t.Error("should contain status")
	}
	if !strings.Contains(got, "**Iterations:** 0/5") {
		t.Error("should contain iteration count")
	}
}

func TestBuildFixFindingsSummary_WithIterations(t *testing.T) {
	state := &fix.LoopState{
		FinalStatus:      "success",
		CurrentIteration: 2,
		Config:           fix.LoopConfig{MaxIterations: 5},
		Iterations: []domain.FixIterationRecord{
			{
				Number:           1,
				DiagnosisSummary: "Found missing import",
				ChangesSummary:   "Added import statement",
				RebuildTriggered: true,
				VerifyResult:     "fail",
				Outcome:          "continue",
			},
			{
				Number:           2,
				DiagnosisSummary: "Fixed build config",
				ChangesSummary:   "Updated webpack config",
				RebuildTriggered: true,
				VerifyResult:     "pass",
				Outcome:          "success",
			},
		},
	}
	got := buildFixFindingsSummary(state)

	if !strings.Contains(got, "## Iteration 1") {
		t.Error("should contain iteration 1 header")
	}
	if !strings.Contains(got, "## Iteration 2") {
		t.Error("should contain iteration 2 header")
	}
	if !strings.Contains(got, "Found missing import") {
		t.Error("should contain diagnosis from iteration 1")
	}
	if !strings.Contains(got, "Updated webpack config") {
		t.Error("should contain changes from iteration 2")
	}
	if !strings.Contains(got, "**Rebuild Triggered:** true") {
		t.Error("should show rebuild triggered")
	}
}

// --- HandlerRegistry tests ---

func TestHandlerRegistry_RegisterAndGet(t *testing.T) {
	reg := NewHandlerRegistry()

	handler := &mockTaskHandler{taskType: domain.TaskTypeInvestigate, tag: "test"}
	reg.Register(handler)

	got, ok := reg.Get(domain.TaskTypeInvestigate)
	if !ok {
		t.Fatal("handler should be found")
	}
	if got.AgentTag() != "test" {
		t.Errorf("AgentTag() = %q, want 'test'", got.AgentTag())
	}
}

func TestHandlerRegistry_GetMissing(t *testing.T) {
	reg := NewHandlerRegistry()

	_, ok := reg.Get(domain.TaskTypeFix)
	if ok {
		t.Error("should not find unregistered handler")
	}
}

func TestHandlerRegistry_OverwritesPrevious(t *testing.T) {
	reg := NewHandlerRegistry()

	reg.Register(&mockTaskHandler{taskType: domain.TaskTypeInvestigate, tag: "first"})
	reg.Register(&mockTaskHandler{taskType: domain.TaskTypeInvestigate, tag: "second"})

	got, ok := reg.Get(domain.TaskTypeInvestigate)
	if !ok {
		t.Fatal("handler should be found")
	}
	if got.AgentTag() != "second" {
		t.Errorf("should use latest registered handler, got tag %q", got.AgentTag())
	}
}

// --- broadcastProgress tests ---

func TestBroadcastProgress_NilHub(t *testing.T) {
	svc := &Service{progressHub: nil}
	// Should not panic
	svc.broadcastProgress("pipe", "inv", "event", 50, "msg")
}

func TestBroadcastProgress_WithHub(t *testing.T) {
	hub := &mockProgressHub{}
	svc := &Service{progressHub: hub}

	svc.broadcastProgress("pipe-1", "inv-1", EventTaskProgress, 50.0, "halfway")

	if len(hub.events) != 1 {
		t.Fatalf("events len = %d, want 1", len(hub.events))
	}
	e := hub.events[0]
	if e.pipelineID != "pipe-1" || e.invID != "inv-1" || e.eventType != EventTaskProgress {
		t.Errorf("event = %+v", e)
	}
	if e.progress != 50.0 {
		t.Errorf("progress = %f, want 50", e.progress)
	}
}

// --- ptrTo helper test ---

func TestPtrTo(t *testing.T) {
	s := ptrTo("hello")
	if *s != "hello" {
		t.Errorf("ptrTo = %q, want 'hello'", *s)
	}

	i := ptrTo(42)
	if *i != 42 {
		t.Errorf("ptrTo = %d, want 42", *i)
	}
}

// Ensure the shared import is used.
var _ = shared.PromptResult{}

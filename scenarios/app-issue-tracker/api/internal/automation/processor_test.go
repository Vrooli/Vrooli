package automation

import (
	"context"
	"testing"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
)

type mockHost struct {
	clearExpiredRateLimitCalled  bool
	loadIssuesFromFolderCalled   bool
	clearRateLimitMetadataCalled bool
	triggerInvestigationCalled   bool
	cleanupOldTranscriptsCalled  bool
	loadIssuesFromFolderResult   []issuespkg.Issue
	triggerInvestigationErr      error
	loadIssuesFromFolderErr      error
}

func (m *mockHost) ClearExpiredRateLimitMetadata() {
	m.clearExpiredRateLimitCalled = true
}

func (m *mockHost) LoadIssuesFromFolder(folder string) ([]issuespkg.Issue, error) {
	m.loadIssuesFromFolderCalled = true
	return m.loadIssuesFromFolderResult, m.loadIssuesFromFolderErr
}

func (m *mockHost) ClearRateLimitMetadata(issueID string) {
	m.clearRateLimitMetadataCalled = true
}

func (m *mockHost) TriggerInvestigation(issueID, agentID string, autoResolve bool) error {
	m.triggerInvestigationCalled = true
	return m.triggerInvestigationErr
}

func (m *mockHost) CleanupOldTranscripts() {
	m.cleanupOldTranscriptsCalled = true
}

func TestNewProcessor(t *testing.T) {
	host := &mockHost{}
	p := NewProcessor(host)

	if p == nil {
		t.Fatal("NewProcessor returned nil")
	}

	state := p.CurrentState()
	if state.Active {
		t.Error("new processor should not be active")
	}
	if state.ConcurrentSlots != 2 {
		t.Errorf("expected 2 concurrent slots, got %d", state.ConcurrentSlots)
	}
	if state.RefreshInterval != 45 {
		t.Errorf("expected refresh interval of 45, got %d", state.RefreshInterval)
	}
}

func TestProcessorStateManagement(t *testing.T) {
	host := &mockHost{}
	p := NewProcessor(host)

	// Test UpdateState
	active := true
	slots := 5
	interval := 60
	maxIssues := 10
	maxIssuesDisabled := false

	p.UpdateState(&active, &slots, &interval, &maxIssues, &maxIssuesDisabled)

	state := p.CurrentState()
	if !state.Active {
		t.Error("state should be active after update")
	}
	if state.ConcurrentSlots != 5 {
		t.Errorf("expected 5 concurrent slots, got %d", state.ConcurrentSlots)
	}
	if state.RefreshInterval != 60 {
		t.Errorf("expected refresh interval of 60, got %d", state.RefreshInterval)
	}
	if state.MaxIssues != 10 {
		t.Errorf("expected max issues of 10, got %d", state.MaxIssues)
	}
	if state.MaxIssuesDisabled {
		t.Error("max issues should not be disabled")
	}
}

func TestProcessorCounter(t *testing.T) {
	host := &mockHost{}
	p := NewProcessor(host)

	// Initial count should be 0
	if count := p.ProcessedCount(); count != 0 {
		t.Errorf("expected initial count of 0, got %d", count)
	}

	// Increment and check
	count := p.IncrementProcessedCount()
	if count != 1 {
		t.Errorf("expected count of 1 after increment, got %d", count)
	}

	// Reset and check
	p.ResetCounter()
	if count := p.ProcessedCount(); count != 0 {
		t.Errorf("expected count of 0 after reset, got %d", count)
	}
}

func TestProcessorRunningProcesses(t *testing.T) {
	host := &mockHost{}
	p := NewProcessor(host)

	issueID := "test-issue-1"
	agentID := "test-agent"
	startTime := time.Now().UTC().Format(time.RFC3339)

	// Register a running process
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.RegisterRunningProcess(issueID, agentID, startTime, cancel)

	// Check if running
	if !p.IsRunning(issueID) {
		t.Error("process should be registered as running")
	}

	// Get running processes
	processes := p.RunningProcesses()
	if len(processes) != 1 {
		t.Fatalf("expected 1 running process, got %d", len(processes))
	}

	if processes[0].IssueID != issueID {
		t.Errorf("expected issue ID %s, got %s", issueID, processes[0].IssueID)
	}

	// Unregister
	p.UnregisterRunningProcess(issueID)

	if p.IsRunning(issueID) {
		t.Error("process should no longer be running after unregister")
	}
}

func TestProcessorCancelRunningProcess(t *testing.T) {
	host := &mockHost{}
	p := NewProcessor(host)

	issueID := "test-issue-1"
	agentID := "test-agent"
	startTime := time.Now().UTC().Format(time.RFC3339)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.RegisterRunningProcess(issueID, agentID, startTime, cancel)

	// Cancel the process
	cancelled := p.CancelRunningProcess(issueID, "test reason")
	if !cancelled {
		t.Error("should have successfully cancelled the process")
	}

	// Check cancellation info
	isCancelled, reason := p.CancellationInfo(issueID)
	if !isCancelled {
		t.Error("process should be marked as cancelled")
	}
	if reason != "test reason" {
		t.Errorf("expected reason 'test reason', got '%s'", reason)
	}

	// Try to cancel non-existent process
	cancelled = p.CancelRunningProcess("non-existent", "test")
	if cancelled {
		t.Error("should not be able to cancel non-existent process")
	}
}

func TestProcessorStartStop(t *testing.T) {
	host := &mockHost{}
	p := NewProcessor(host)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the processor
	p.Start(ctx)

	// Starting again should be a no-op (shouldn't panic)
	p.Start(ctx)

	// Stop the processor
	p.Stop()

	// Stopping again should be a no-op (shouldn't panic)
	p.Stop()
}

package server

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"app-issue-tracker-api/internal/agents"
)

func TestExecuteInvestigationUpdatesProcessedCount(t *testing.T) {
	cleanupLogger := setupTestLogger()
	defer cleanupLogger()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	svc := env.Server.investigations
	fixed := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	svc.now = func() time.Time { return fixed }
	var restarted []string
	svc.restartScenario = func(ctx context.Context, scenario string) (string, error) {
		restarted = append(restarted, scenario)
		return "restart ok", nil
	}

	factory := &captureFactory{}
	env.Server.commandFactory = factory.build

	// Successful execution should increment exactly once.
	issueSuccess := createTestIssue("issue-success", "Success Case", "bug", "high", "test-app")
	if _, err := env.Server.saveIssue(issueSuccess, "open"); err != nil {
		t.Fatalf("failed to seed success issue: %v", err)
	}

	factory.cmd = &fakeCommand{stdout: "Investigation summary\nFinal response"}
	ctx, cancel := context.WithCancel(context.Background())
	svc.executeInvestigation(ctx, cancel, issueSuccess.ID, agents.UnifiedResolverID, fixed.Format(time.RFC3339))

	if got := env.Server.processor.ProcessedCount(); got != 1 {
		t.Fatalf("expected processed count 1 after success, got %d", got)
	}

	if len(restarted) != 1 || restarted[0] != "test-app" {
		t.Fatalf("expected restart of test-app once, got %v", restarted)
	}

	// Failed execution should add one more.
	issueFailure := createTestIssue("issue-failure", "Failure Case", "bug", "high", "test-app")
	if _, err := env.Server.saveIssue(issueFailure, "open"); err != nil {
		t.Fatalf("failed to seed failure issue: %v", err)
	}

	factory.cmd = &fakeCommand{waitErr: errors.New("boom")}
	ctx, cancel = context.WithCancel(context.Background())
	svc.executeInvestigation(ctx, cancel, issueFailure.ID, agents.UnifiedResolverID, fixed.Format(time.RFC3339))

	if got := env.Server.processor.ProcessedCount(); got != 2 {
		t.Fatalf("expected processed count 2 after failure, got %d", got)
	}

	// NEW BEHAVIOR: Scenario restart now happens on failure too (to recover from broken state)
	if len(restarted) != 2 || restarted[1] != "test-app" {
		t.Fatalf("expected restart on failure too (to recover), got %v", restarted)
	}
}

func TestHandleInvestigationSuccessRestartFailure(t *testing.T) {
	cleanupLogger := setupTestLogger()
	defer cleanupLogger()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	svc := env.Server.investigations
	fixed := time.Date(2024, 6, 1, 10, 11, 12, 0, time.UTC)
	svc.now = func() time.Time { return fixed }

	issue := createTestIssue("issue-restart-failure", "Restart Failure", "bug", "high", "scenario-x")
	if _, err := env.Server.saveIssue(issue, "active"); err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}

	var restarts int
	svc.restartScenario = func(ctx context.Context, scenario string) (string, error) {
		restarts++
		return "restart output", errors.New("restart boom")
	}

	result := &ClaudeExecutionResult{
		Success:         true,
		LastMessage:     "All good",
		Output:          "All good",
		ExecutionTime:   2 * time.Second,
		TranscriptPath:  "",
		LastMessagePath: "",
	}

	svc.handleInvestigationSuccess(issue.ID, agents.UnifiedResolverID, result)

	if restarts != 1 {
		t.Fatalf("expected restart attempt once, got %d", restarts)
	}

	// NEW BEHAVIOR: Investigation success is not invalidated by restart failure
	// The issue should be completed, but restart failure should be logged in the event
	completedIssue := assertIssueExists(t, env.Server, issue.ID, "completed")
	if status := strings.TrimSpace(completedIssue.Status); status != "completed" {
		t.Fatalf("expected completed status (restart failure doesn't invalidate investigation), got %s", status)
	}
	// Investigation report should contain the agent's successful output, not restart error
	if !strings.Contains(completedIssue.Investigation.Report, "All good") {
		t.Fatalf("expected investigation report to contain agent output, got: %s", completedIssue.Investigation.Report)
	}
}

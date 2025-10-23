package server

import (
	"context"
	"errors"
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
}

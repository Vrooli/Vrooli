package server

import (
	"testing"
	"time"

	"app-issue-tracker-api/internal/agents"
)

func TestPersistInvestigationStartSetsRunningStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue(
		"issue-status-start",
		"Status Start",
		"bug",
		"medium",
		"status-suite",
	)
	if _, err := env.Server.saveIssue(issue, "open"); err != nil {
		t.Fatalf("Failed to save issue: %v", err)
	}

	loadedIssue, issueDir, _, err := env.Server.loadIssueWithStatus(issue.ID)
	if err != nil {
		t.Fatalf("Failed to load issue: %v", err)
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)
	if err := env.Server.investigations.persistInvestigationStart(loadedIssue, issueDir, agents.UnifiedResolverID, startedAt); err != nil {
		t.Fatalf("persistInvestigationStart failed: %v", err)
	}

	reloaded, _, _, err := env.Server.loadIssueWithStatus(issue.ID)
	if err != nil {
		t.Fatalf("Failed to reload issue: %v", err)
	}

	if reloaded.Metadata.Extra == nil {
		t.Fatalf("Expected metadata extra to be initialized")
	}

	if status := reloaded.Metadata.Extra[AgentStatusExtraKey]; status != AgentStatusRunning {
		t.Fatalf("Expected agent status %q, got %q", AgentStatusRunning, status)
	}

	if ts := reloaded.Metadata.Extra[AgentStatusTimestampExtraKey]; ts != startedAt {
		t.Fatalf("Expected status timestamp %q, got %q", startedAt, ts)
	}
}

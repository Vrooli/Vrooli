package server

import "testing"

func TestHandleInvestigationRateLimitMovesIssueToFailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	issue := createTestIssue("issue-rate-limit", "Rate limit scenario", "bug", "high", "rate-limit-app")
	if _, err := env.Server.saveIssue(issue, "active"); err != nil {
		t.Fatalf("Failed to persist test issue: %v", err)
	}

	resetTime := "2035-01-02T03:04:05Z"
	result := &ClaudeExecutionResult{
		Success: false,
		Error:   "RATE_LIMIT: API rate limit reached",
		Output:  "upstream responded with 429 Too Many Requests; next window resumes at " + resetTime,
	}

	handled := env.Server.handleInvestigationRateLimit(issue.ID, "unified-resolver", result)
	if !handled {
		t.Fatal("Expected rate limit handler to consume result")
	}

	updated, _, status, err := env.Server.loadIssueWithStatus(issue.ID)
	if err != nil {
		t.Fatalf("Failed to reload issue: %v", err)
	}

	if status != "failed" {
		t.Fatalf("Expected issue to move to failed, found %s", status)
	}

	if updated.Metadata.Extra == nil {
		t.Fatal("Expected metadata extras to be initialized")
	}

	if got := updated.Metadata.Extra["rate_limit_agent"]; got != "unified-resolver" {
		t.Errorf("Expected rate_limit_agent to be 'unified-resolver', got %q", got)
	}

	if got := updated.Metadata.Extra["rate_limit_until"]; got != resetTime {
		t.Errorf("Expected rate_limit_until to be %s, got %s", resetTime, got)
	}
}

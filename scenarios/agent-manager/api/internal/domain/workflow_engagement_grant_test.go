package domain

import "testing"

func TestWorkflowEngagementGrantValidationRequiresCoreLimits(t *testing.T) {
	if err := (WorkflowEngagementGrant{}).Validate(); err == nil {
		t.Fatal("empty grant unexpectedly accepted")
	}
	if err := (WorkflowEngagementGrant{MaxTokens: 10}).Validate(); err == nil {
		t.Fatal("grant without wall-time limit unexpectedly accepted")
	}
	if err := (WorkflowEngagementGrant{MaxTokens: 10, MaxWallTimeSeconds: 5, MaxRetries: -1}).Validate(); err == nil {
		t.Fatal("negative grant limit unexpectedly accepted")
	}
}

func TestWorkflowEngagementGrantValidationAcceptsBoundedGrant(t *testing.T) {
	grant := WorkflowEngagementGrant{MaxTokens: 100, MaxWallTimeSeconds: 30}
	if err := grant.Validate(); err != nil {
		t.Fatalf("valid grant rejected: %v", err)
	}
}

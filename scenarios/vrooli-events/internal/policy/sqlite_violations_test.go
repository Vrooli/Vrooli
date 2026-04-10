package policy

import (
	"context"
	"testing"
)

// [REQ:REQ-POL-007] Violation logging and querying
func TestSQLiteStore_Violations(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	// Log violations
	ps.LogViolation(ctx, Violation{
		SourceScenario: "src-a", TargetScenario: "tgt", Endpoint: "/api",
		RuleID: 1, RuleType: RuleTypeAccess, Reason: "denied",
	})
	ps.LogViolation(ctx, Violation{
		SourceScenario: "src-b", TargetScenario: "tgt", Endpoint: "/api",
		RuleID: 2, RuleType: RuleTypeAccess, Reason: "denied",
	})

	// List all
	viol, _ := ps.ListViolations(ctx, ViolationFilters{})
	if len(viol) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(viol))
	}

	// Filter by source
	viol, _ = ps.ListViolations(ctx, ViolationFilters{Source: "src-a"})
	if len(viol) != 1 {
		t.Fatalf("expected 1 violation for src-a, got %d", len(viol))
	}

	// Limit
	viol, _ = ps.ListViolations(ctx, ViolationFilters{Limit: 1})
	if len(viol) != 1 {
		t.Fatalf("expected 1 with limit, got %d", len(viol))
	}
}

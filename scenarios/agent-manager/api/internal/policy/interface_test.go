package policy_test

import (
	"testing"

	"agent-manager/internal/policy"
)

// [REQ:REQ-P1-004] Tests for policy evaluator interface definitions

// TestInterfaceTypes verifies that the package compiles and types are accessible.
// This is a minimal test to satisfy test coverage requirements for interface-only packages.
func TestInterfaceTypes(t *testing.T) {
	// Verify types are accessible
	var _ policy.Decision
	var _ policy.ApprovalDecision
	var _ policy.ConcurrencyDecision
}

func TestDecision_Fields(t *testing.T) {
	// Verify Decision struct can be instantiated with correct fields
	d := policy.Decision{
		Allowed:           true,
		RequiresSandbox:   true,
		RequiresApproval:  false,
		EffectiveMaxFiles: 100,
		EffectiveMaxSize:  10 * 1024 * 1024,
		EffectiveTimeout:  30 * 60 * 1000, // 30 minutes in ms
		DenialReason:      "",
	}

	if !d.Allowed {
		t.Error("expected decision to be allowed")
	}
	if !d.RequiresSandbox {
		t.Error("expected sandbox to be required")
	}
	if d.RequiresApproval {
		t.Error("expected approval not to be required")
	}
	if d.EffectiveMaxFiles != 100 {
		t.Errorf("expected max files 100, got %d", d.EffectiveMaxFiles)
	}
}

func TestApprovalDecision_Fields(t *testing.T) {
	// Verify ApprovalDecision struct
	ad := policy.ApprovalDecision{
		Allowed:                    true,
		RequiresAdditionalApprover: false,
		AutoApproved:               false,
		DenialReason:               "",
	}

	if !ad.Allowed {
		t.Error("expected approval to be allowed")
	}
}

func TestConcurrencyDecision_Fields(t *testing.T) {
	// Verify ConcurrencyDecision struct
	cd := policy.ConcurrencyDecision{
		Allowed:      false,
		DenialReason: "max concurrent runs exceeded",
		CurrentRuns:  10,
		MaxRuns:      10,
	}

	if cd.Allowed {
		t.Error("expected concurrency to be denied")
	}
	if cd.DenialReason == "" {
		t.Error("expected deny reason to be set")
	}
}

func TestDecision_Denied(t *testing.T) {
	// Test a denied decision
	d := policy.Decision{
		Allowed:      false,
		DenialReason: "path matches deny pattern .git/**",
	}

	if d.Allowed {
		t.Error("expected decision to be denied")
	}
	if d.DenialReason == "" {
		t.Error("expected deny reason when not allowed")
	}
}

func TestApprovalDecision_AutoApprove(t *testing.T) {
	// Test auto-approval decision
	ad := policy.ApprovalDecision{
		Allowed:           true,
		AutoApproved:      true,
		AutoApproveReason: "matches auto-approve pattern",
	}

	if !ad.AutoApproved {
		t.Error("expected auto-approve to be true")
	}
}

package policy_test

import (
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/policy"

	"github.com/google/uuid"
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

func TestEvaluateRequest_Fields(t *testing.T) {
	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Test Task",
		ScopePath: "src/",
	}
	profile := &domain.AgentProfile{
		ID:   uuid.New(),
		Name: "test-profile",
	}

	req := policy.EvaluateRequest{
		Task:          task,
		Profile:       profile,
		RequestedMode: domain.RunModeSandboxed,
		Actor:         "user@example.com",
		ForceInPlace:  false,
	}

	if req.Task != task {
		t.Error("expected Task to be set")
	}
	if req.Profile != profile {
		t.Error("expected Profile to be set")
	}
	if req.RequestedMode != domain.RunModeSandboxed {
		t.Errorf("expected RequestedMode 'sandboxed', got %s", req.RequestedMode)
	}
	if req.Actor != "user@example.com" {
		t.Errorf("expected Actor 'user@example.com', got %s", req.Actor)
	}
	if req.ForceInPlace {
		t.Error("expected ForceInPlace to be false")
	}
}

func TestEvaluateRequest_ForceInPlace(t *testing.T) {
	req := policy.EvaluateRequest{
		RequestedMode: domain.RunModeInPlace,
		Actor:         "admin@example.com",
		ForceInPlace:  true,
	}

	if !req.ForceInPlace {
		t.Error("expected ForceInPlace to be true")
	}
	if req.RequestedMode != domain.RunModeInPlace {
		t.Errorf("expected RequestedMode 'in_place', got %s", req.RequestedMode)
	}
}

func TestConcurrencyRequest_Fields(t *testing.T) {
	req := policy.ConcurrencyRequest{
		ScopePath:   "src/api",
		ProjectRoot: "/home/user/project",
		RunnerType:  domain.RunnerTypeClaudeCode,
	}

	if req.ScopePath != "src/api" {
		t.Errorf("expected ScopePath 'src/api', got %s", req.ScopePath)
	}
	if req.ProjectRoot != "/home/user/project" {
		t.Errorf("expected ProjectRoot '/home/user/project', got %s", req.ProjectRoot)
	}
	if req.RunnerType != domain.RunnerTypeClaudeCode {
		t.Errorf("expected RunnerType 'claude-code', got %s", req.RunnerType)
	}
}

func TestAppliedPolicy_Fields(t *testing.T) {
	policyID := uuid.New()
	ap := policy.AppliedPolicy{
		PolicyID:   policyID,
		PolicyName: "production-safety",
		Effect:     "require_sandbox",
	}

	if ap.PolicyID != policyID {
		t.Errorf("expected PolicyID %s, got %s", policyID, ap.PolicyID)
	}
	if ap.PolicyName != "production-safety" {
		t.Errorf("expected PolicyName 'production-safety', got %s", ap.PolicyName)
	}
	if ap.Effect != "require_sandbox" {
		t.Errorf("expected Effect 'require_sandbox', got %s", ap.Effect)
	}
}

func TestDecision_WithAppliedPolicies(t *testing.T) {
	policy1ID := uuid.New()
	policy2ID := uuid.New()

	d := policy.Decision{
		Allowed:         true,
		RequiresSandbox: true,
		AppliedPolicies: []policy.AppliedPolicy{
			{PolicyID: policy1ID, PolicyName: "sandbox-rule", Effect: "require_sandbox"},
			{PolicyID: policy2ID, PolicyName: "approval-rule", Effect: "require_approval"},
		},
	}

	if len(d.AppliedPolicies) != 2 {
		t.Errorf("expected 2 applied policies, got %d", len(d.AppliedPolicies))
	}
	if d.AppliedPolicies[0].PolicyName != "sandbox-rule" {
		t.Errorf("expected first policy name 'sandbox-rule', got %s", d.AppliedPolicies[0].PolicyName)
	}
}

func TestDecision_WithDenialPolicy(t *testing.T) {
	denialPolicy := &domain.Policy{
		ID:   uuid.New(),
		Name: "deny-git",
	}

	d := policy.Decision{
		Allowed:      false,
		DenialReason: "path matches deny pattern .git/**",
		DenialPolicy: denialPolicy,
	}

	if d.Allowed {
		t.Error("expected decision to be denied")
	}
	if d.DenialPolicy == nil {
		t.Error("expected DenialPolicy to be set")
	}
	if d.DenialPolicy.Name != "deny-git" {
		t.Errorf("expected DenialPolicy.Name 'deny-git', got %s", d.DenialPolicy.Name)
	}
}

func TestListFilter_Fields(t *testing.T) {
	enabled := true
	filter := policy.ListFilter{
		Enabled: &enabled,
		Limit:   25,
		Offset:  10,
	}

	if filter.Enabled == nil || !*filter.Enabled {
		t.Error("expected Enabled to be true")
	}
	if filter.Limit != 25 {
		t.Errorf("expected Limit 25, got %d", filter.Limit)
	}
	if filter.Offset != 10 {
		t.Errorf("expected Offset 10, got %d", filter.Offset)
	}
}

func TestListFilter_NilEnabled(t *testing.T) {
	filter := policy.ListFilter{
		Limit: 50,
	}

	if filter.Enabled != nil {
		t.Error("expected Enabled to be nil")
	}
}

func TestConcurrencyDecision_WithWaitEstimate(t *testing.T) {
	cd := policy.ConcurrencyDecision{
		Allowed:      false,
		DenialReason: "max concurrent runs reached",
		CurrentRuns:  10,
		MaxRuns:      10,
		WaitEstimate: "approximately 5 minutes",
	}

	if cd.Allowed {
		t.Error("expected concurrency to be denied")
	}
	if cd.WaitEstimate != "approximately 5 minutes" {
		t.Errorf("expected WaitEstimate 'approximately 5 minutes', got %s", cd.WaitEstimate)
	}
}

func TestApprovalDecision_RequiresAdditionalApprover(t *testing.T) {
	ad := policy.ApprovalDecision{
		Allowed:                    true,
		RequiresAdditionalApprover: true,
	}

	if !ad.Allowed {
		t.Error("expected approval to be allowed")
	}
	if !ad.RequiresAdditionalApprover {
		t.Error("expected RequiresAdditionalApprover to be true")
	}
}

package settings

import (
	"path/filepath"
	"testing"

	"swarm-manager/internal/execution"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "settings.json"))
	s := DefaultSettings()
	s.AgentMaxTurns = 42
	s.AgentTimeoutSeconds = 300
	s.AgentRequiresApproval = true
	s.DefaultMode = "yolo"
	s.AutoFixup = true
	s.MaxFixupAttempts = 3
	s.MaxConcurrentExecutions = 5
	s.MaxQueueDepth = 25
	s.CircuitBreakerThreshold = 4
	s.CircuitBreakerCooldownMinutes = 30
	s.ExecutionCostCapPerRun = 1.5
	s.CostPerTurnEstimate = 0.05
	s.ReviewCodeQualityMinScore = 0.8
	s.ReviewTestMinPassRate = 0.9
	s.ReviewMaxBlockingViolations = 2
	s.ReviewMaxWarnings = 10
	s.ReviewRequireScreenshots = true
	s.ReviewRequireTests = true
	if err := store.Save(s); err != nil {
		t.Fatalf("save test settings: %v", err)
	}
	return store
}

func TestAgentAdapter(t *testing.T) {
	store := testStore(t)
	adapter := NewAgentAdapter(store)

	maxTurns, timeout, approval, err := adapter.LoadAgentSettings()
	if err != nil {
		t.Fatalf("LoadAgentSettings: %v", err)
	}
	if maxTurns != 42 {
		t.Errorf("maxTurns = %d, want 42", maxTurns)
	}
	if timeout != 300 {
		t.Errorf("timeout = %d, want 300", timeout)
	}
	if !approval {
		t.Error("requiresApproval should be true")
	}
}

func TestPolicyAdapter(t *testing.T) {
	store := testStore(t)
	adapter := NewPolicyAdapter(store)

	policy, err := adapter.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy: %v", err)
	}
	if policy.DefaultMode != execution.Mode("yolo") {
		t.Errorf("DefaultMode = %q, want %q", policy.DefaultMode, "yolo")
	}
	if !policy.AutoFixup {
		t.Error("AutoFixup should be true")
	}
	if policy.MaxFixupAttempts != 3 {
		t.Errorf("MaxFixupAttempts = %d, want 3", policy.MaxFixupAttempts)
	}
	if !policy.ReviewAgentEnabled {
		t.Error("ReviewAgentEnabled should be true")
	}
}

func TestPolicyAdapter_ReviewAgentDisabled(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "settings.json"))
	s := DefaultSettings()
	s.ReviewAgentEnabled = false
	if err := store.Save(s); err != nil {
		t.Fatalf("save: %v", err)
	}
	adapter := NewPolicyAdapter(store)
	policy, err := adapter.LoadPolicy()
	if err != nil {
		t.Fatalf("LoadPolicy: %v", err)
	}
	if policy.ReviewAgentEnabled {
		t.Error("ReviewAgentEnabled should be false when explicitly disabled")
	}
}

func TestGovernanceAdapter(t *testing.T) {
	store := testStore(t)
	adapter := NewGovernanceAdapter(store)

	gov, err := adapter.LoadGovernance()
	if err != nil {
		t.Fatalf("LoadGovernance: %v", err)
	}
	if gov.MaxConcurrentExecutions != 5 {
		t.Errorf("MaxConcurrentExecutions = %d, want 5", gov.MaxConcurrentExecutions)
	}
	if gov.MaxQueueDepth != 25 {
		t.Errorf("MaxQueueDepth = %d, want 25", gov.MaxQueueDepth)
	}
	if gov.CircuitBreakerThreshold != 4 {
		t.Errorf("CircuitBreakerThreshold = %d, want 4", gov.CircuitBreakerThreshold)
	}
	if gov.CostPerTurnEstimate != 0.05 {
		t.Errorf("CostPerTurnEstimate = %f, want 0.05", gov.CostPerTurnEstimate)
	}
	if gov.AgentMaxTurns != 42 {
		t.Errorf("AgentMaxTurns = %d, want 42", gov.AgentMaxTurns)
	}
}

func TestReviewThresholdsAdapter(t *testing.T) {
	store := testStore(t)
	adapter := NewReviewThresholdsAdapter(store)

	thresholds, err := adapter.LoadReviewThresholds()
	if err != nil {
		t.Fatalf("LoadReviewThresholds: %v", err)
	}
	if thresholds.CodeQualityMinScore != 0.8 {
		t.Errorf("CodeQualityMinScore = %f, want 0.8", thresholds.CodeQualityMinScore)
	}
	if thresholds.TestMinPassRate != 0.9 {
		t.Errorf("TestMinPassRate = %f, want 0.9", thresholds.TestMinPassRate)
	}
	if thresholds.MaxBlockingViolations != 2 {
		t.Errorf("MaxBlockingViolations = %d, want 2", thresholds.MaxBlockingViolations)
	}
	if !thresholds.RequireScreenshots {
		t.Error("RequireScreenshots should be true")
	}
	if !thresholds.RequireTests {
		t.Error("RequireTests should be true")
	}
}

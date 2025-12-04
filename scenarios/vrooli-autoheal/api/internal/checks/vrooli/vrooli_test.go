// Package vrooli tests for Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001] [REQ:HEAL-ACTION-001]
package vrooli

import (
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
)

// TestResourceCheckInterface verifies ResourceCheck implements Check
// [REQ:RESOURCE-CHECK-001]
func TestResourceCheckInterface(t *testing.T) {
	var _ checks.Check = (*ResourceCheck)(nil)

	check := NewResourceCheck("postgres")
	if check.ID() != "resource-postgres" {
		t.Errorf("ID() = %q, want %q", check.ID(), "resource-postgres")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	// Resource checks should run on all platforms
	if check.Platforms() != nil {
		t.Error("ResourceCheck should run on all platforms")
	}
}

// TestResourceCheckCreation verifies ResourceCheck creation with different names
func TestResourceCheckCreation(t *testing.T) {
	resources := []string{"postgres", "redis", "ollama", "qdrant"}

	for _, res := range resources {
		check := NewResourceCheck(res)
		expectedID := "resource-" + res
		if check.ID() != expectedID {
			t.Errorf("NewResourceCheck(%q).ID() = %q, want %q", res, check.ID(), expectedID)
		}
		if check.resourceName != res {
			t.Errorf("resourceName = %q, want %q", check.resourceName, res)
		}
	}
}

// TestScenarioCheckInterface verifies ScenarioCheck implements Check
// [REQ:SCENARIO-CHECK-001]
func TestScenarioCheckInterface(t *testing.T) {
	var _ checks.Check = (*ScenarioCheck)(nil)

	check := NewScenarioCheck("test-scenario", true)
	if check.ID() != "scenario-test-scenario" {
		t.Errorf("ID() = %q, want %q", check.ID(), "scenario-test-scenario")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	// Scenario checks should run on all platforms
	if check.Platforms() != nil {
		t.Error("ScenarioCheck should run on all platforms")
	}
}

// TestScenarioCheckCriticality verifies critical flag affects status
func TestScenarioCheckCriticality(t *testing.T) {
	criticalCheck := NewScenarioCheck("test-crit", true)
	nonCriticalCheck := NewScenarioCheck("test-non-crit", false)

	if !criticalCheck.critical {
		t.Error("Critical check should have critical=true")
	}
	if nonCriticalCheck.critical {
		t.Error("Non-critical check should have critical=false")
	}
}

// TestAPICheckInterface verifies APICheck implements Check
// [REQ:VROOLI-API-001]
func TestAPICheckInterface(t *testing.T) {
	var _ checks.Check = (*APICheck)(nil)

	check := NewAPICheck()
	if check.ID() != "vrooli-api" {
		t.Errorf("ID() = %q, want %q", check.ID(), "vrooli-api")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	// API check should run on all platforms
	if check.Platforms() != nil {
		t.Error("APICheck should run on all platforms")
	}
}

// TestAPICheckHealable verifies APICheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestAPICheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*APICheck)(nil)

	check := NewAPICheck()

	// Test recovery actions with nil result
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"restart":   false,
		"kill-port": false,
		"logs":      false,
		"diagnose":  false,
	}
	for _, action := range actions {
		if _, exists := expectedActions[action.ID]; exists {
			expectedActions[action.ID] = true
		}
	}
	for id, found := range expectedActions {
		if !found {
			t.Errorf("Expected action %q not found in RecoveryActions()", id)
		}
	}
}

// TestAPICheckOptions verifies APICheck configuration options
func TestAPICheckOptions(t *testing.T) {
	check := NewAPICheck(
		WithAPIURL("http://localhost:9000/health"),
		WithAPITimeout(10*time.Second),
	)

	if check.url != "http://localhost:9000/health" {
		t.Errorf("url = %q, want %q", check.url, "http://localhost:9000/health")
	}
	if check.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want %v", check.timeout, 10*time.Second)
	}
}

// TestResourceCheckHealable verifies ResourceCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestResourceCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*ResourceCheck)(nil)

	check := NewResourceCheck("postgres")

	// Test recovery actions
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"start":   false,
		"stop":    false,
		"restart": false,
		"logs":    false,
	}
	for _, action := range actions {
		if _, exists := expectedActions[action.ID]; exists {
			expectedActions[action.ID] = true
		}
	}
	for id, found := range expectedActions {
		if !found {
			t.Errorf("Expected action %q not found in RecoveryActions()", id)
		}
	}
}

// TestScenarioCheckHealable verifies ScenarioCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestScenarioCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*ScenarioCheck)(nil)

	check := NewScenarioCheck("test-scenario", true)

	// Test recovery actions
	actions := check.RecoveryActions(nil)
	if len(actions) == 0 {
		t.Error("RecoveryActions() should return actions")
	}

	// Verify expected actions exist
	expectedActions := map[string]bool{
		"start":         false,
		"stop":          false,
		"restart":       false,
		"restart-clean": false,
		"cleanup-ports": false,
		"logs":          false,
		"diagnose":      false,
	}
	for _, action := range actions {
		if _, exists := expectedActions[action.ID]; exists {
			expectedActions[action.ID] = true
		}
	}
	for id, found := range expectedActions {
		if !found {
			t.Errorf("Expected action %q not found in RecoveryActions()", id)
		}
	}
}

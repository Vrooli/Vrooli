// Package vrooli tests for Vrooli-specific health checks
// [REQ:RESOURCE-CHECK-001] [REQ:SCENARIO-CHECK-001]
package vrooli

import (
	"testing"

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

// Package health tests
// [REQ:SCS-HEALTH-001] Test health status API
// [REQ:SCS-HEALTH-002] Test collector status tracking
// [REQ:SCS-HEALTH-003] Test collector test endpoint
package health

import (
	"errors"
	"testing"
	"time"

	"scenario-completeness-scoring/pkg/circuitbreaker"
)

// [REQ:SCS-HEALTH-002] Test RecordCheck success
func TestRecordCheckSuccess(t *testing.T) {
	tracker := NewTracker(nil)

	tracker.RecordCheck("requirements", true, "")
	health := tracker.GetCollectorHealth("requirements")

	if health.Status != StatusOK {
		t.Errorf("expected status OK, got %v", health.Status)
	}
	if health.LastSuccess == nil {
		t.Error("expected LastSuccess to be set")
	}
}

// [REQ:SCS-HEALTH-002] Test RecordCheck failure
func TestRecordCheckFailure(t *testing.T) {
	tracker := NewTracker(nil)

	tracker.RecordCheck("requirements", false, "file not found")
	health := tracker.GetCollectorHealth("requirements")

	if health.Status != StatusDegraded {
		t.Errorf("expected status Degraded after single failure, got %v", health.Status)
	}
	if health.Message != "file not found" {
		t.Errorf("expected error message, got %q", health.Message)
	}
}

// [REQ:SCS-HEALTH-002] Test consecutive failures lead to Failed status
func TestConsecutiveFailures(t *testing.T) {
	tracker := NewTracker(nil)

	// 3 consecutive failures
	tracker.RecordCheck("tests", false, "error 1")
	tracker.RecordCheck("tests", false, "error 2")
	tracker.RecordCheck("tests", false, "error 3")

	health := tracker.GetCollectorHealth("tests")
	if health.Status != StatusFailed {
		t.Errorf("expected status Failed after 3 failures, got %v", health.Status)
	}
}

// [REQ:SCS-HEALTH-002] Test success resets failure count
func TestSuccessResetsFailures(t *testing.T) {
	tracker := NewTracker(nil)

	// 2 failures
	tracker.RecordCheck("ui", false, "error 1")
	tracker.RecordCheck("ui", false, "error 2")

	// Then success
	tracker.RecordCheck("ui", true, "")

	health := tracker.GetCollectorHealth("ui")
	if health.Status != StatusOK {
		t.Errorf("expected status OK after success, got %v", health.Status)
	}
}

// [REQ:SCS-HEALTH-001] Test GetOverallHealth
func TestGetOverallHealth(t *testing.T) {
	tracker := NewTracker(nil)

	// Record some checks
	tracker.RecordCheck("requirements", true, "")
	tracker.RecordCheck("tests", true, "")
	tracker.RecordCheck("ui", false, "template detected")

	overall := tracker.GetOverallHealth()

	if overall.Total < 4 {
		t.Errorf("expected at least 4 total collectors, got %d", overall.Total)
	}
	if overall.Healthy < 2 {
		t.Errorf("expected at least 2 healthy collectors, got %d", overall.Healthy)
	}
}

// [REQ:SCS-HEALTH-001] Test overall status reflects worst collector
func TestOverallStatusReflectsWorst(t *testing.T) {
	tracker := NewTracker(nil)

	// All healthy
	tracker.RecordCheck("requirements", true, "")
	tracker.RecordCheck("tests", true, "")
	overall := tracker.GetOverallHealth()

	if overall.Status != StatusOK {
		t.Errorf("expected overall OK when all healthy, got %v", overall.Status)
	}

	// Add a degraded collector
	tracker.RecordCheck("ui", false, "minor issue")
	overall = tracker.GetOverallHealth()

	if overall.Status != StatusDegraded {
		t.Errorf("expected overall Degraded when one is degraded, got %v", overall.Status)
	}

	// Add a failed collector
	tracker.RecordCheck("custom", false, "error")
	tracker.RecordCheck("custom", false, "error")
	tracker.RecordCheck("custom", false, "error")
	overall = tracker.GetOverallHealth()

	if overall.Status != StatusFailed {
		t.Errorf("expected overall Failed when one is failed, got %v", overall.Status)
	}
}

// [REQ:SCS-HEALTH-003] Test TestCollector success
func TestTestCollectorSuccess(t *testing.T) {
	tracker := NewTracker(nil)

	result := tracker.TestCollector("requirements", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	if !result.Success {
		t.Error("expected test to succeed")
	}
	if result.Status != StatusOK {
		t.Errorf("expected status OK, got %v", result.Status)
	}
	if result.Duration < 5*time.Millisecond {
		t.Error("expected duration to be at least 5ms")
	}
}

// [REQ:SCS-HEALTH-003] Test TestCollector failure
func TestTestCollectorFailure(t *testing.T) {
	tracker := NewTracker(nil)

	result := tracker.TestCollector("tests", func() error {
		return errors.New("test execution failed")
	})

	if result.Success {
		t.Error("expected test to fail")
	}
	if result.Status != StatusFailed {
		t.Errorf("expected status Failed, got %v", result.Status)
	}
	if result.Error != "test execution failed" {
		t.Errorf("expected error message, got %q", result.Error)
	}
}

// [REQ:SCS-HEALTH-002] Test integration with circuit breaker
func TestCircuitBreakerIntegration(t *testing.T) {
	cfg := circuitbreaker.DefaultConfig()
	cfg.FailureThreshold = 1
	registry := circuitbreaker.NewRegistry(cfg)

	tracker := NewTracker(registry)

	// Get the breaker and trip it
	cb := registry.Get("requirements")
	cb.RecordFailure()

	// Health should show failed due to circuit breaker
	health := tracker.GetCollectorHealth("requirements")
	if health.Status != StatusFailed {
		t.Errorf("expected status Failed when circuit is open, got %v", health.Status)
	}
	if health.CircuitState != "open" {
		t.Errorf("expected circuit state 'open', got %q", health.CircuitState)
	}
}

// [REQ:SCS-HEALTH-002] Test unknown collector returns OK
func TestUnknownCollectorStatus(t *testing.T) {
	tracker := NewTracker(nil)

	health := tracker.GetCollectorHealth("unknown")
	if health.Status != StatusOK {
		t.Errorf("expected status OK for unknown collector, got %v", health.Status)
	}
	if health.Message != "Not yet checked" {
		t.Errorf("expected 'Not yet checked' message, got %q", health.Message)
	}
}

// [REQ:SCS-HEALTH-001] Test CollectorStatus constants
func TestCollectorStatusConstants(t *testing.T) {
	if StatusOK != "ok" {
		t.Errorf("expected StatusOK to be 'ok', got %q", StatusOK)
	}
	if StatusDegraded != "degraded" {
		t.Errorf("expected StatusDegraded to be 'degraded', got %q", StatusDegraded)
	}
	if StatusFailed != "failed" {
		t.Errorf("expected StatusFailed to be 'failed', got %q", StatusFailed)
	}
}

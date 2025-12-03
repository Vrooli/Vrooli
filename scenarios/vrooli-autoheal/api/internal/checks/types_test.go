// Package checks tests for types
package checks

import (
	"testing"
	"time"
)

// TestStatusConstants verifies Status constants have expected values
func TestStatusConstants(t *testing.T) {
	if string(StatusOK) != "ok" {
		t.Errorf("StatusOK = %q, want %q", StatusOK, "ok")
	}
	if string(StatusWarning) != "warning" {
		t.Errorf("StatusWarning = %q, want %q", StatusWarning, "warning")
	}
	if string(StatusCritical) != "critical" {
		t.Errorf("StatusCritical = %q, want %q", StatusCritical, "critical")
	}
}

// TestResultStructure verifies Result has all expected fields
func TestResultStructure(t *testing.T) {
	result := Result{
		CheckID:   "test",
		Status:    StatusOK,
		Message:   "Test message",
		Details:   map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}

	if result.CheckID == "" {
		t.Error("CheckID not set")
	}
	if result.Status == "" {
		t.Error("Status not set")
	}
	if result.Message == "" {
		t.Error("Message not set")
	}
	if result.Details == nil {
		t.Error("Details not set")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp not set")
	}
	if result.Duration == 0 {
		t.Error("Duration not set")
	}
}

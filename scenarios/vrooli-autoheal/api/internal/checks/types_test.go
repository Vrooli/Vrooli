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

// TestWorstStatus verifies status comparison logic
func TestWorstStatus(t *testing.T) {
	tests := []struct {
		name string
		a, b Status
		want Status
	}{
		{"ok + ok", StatusOK, StatusOK, StatusOK},
		{"ok + warning", StatusOK, StatusWarning, StatusWarning},
		{"warning + ok", StatusWarning, StatusOK, StatusWarning},
		{"ok + critical", StatusOK, StatusCritical, StatusCritical},
		{"critical + ok", StatusCritical, StatusOK, StatusCritical},
		{"warning + critical", StatusWarning, StatusCritical, StatusCritical},
		{"critical + warning", StatusCritical, StatusWarning, StatusCritical},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := WorstStatus(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("WorstStatus(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestAggregateStatus verifies overall status calculation
func TestAggregateStatus(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    Status
	}{
		{"empty returns ok", []Result{}, StatusOK},
		{"all ok", []Result{{Status: StatusOK}, {Status: StatusOK}}, StatusOK},
		{"one warning", []Result{{Status: StatusOK}, {Status: StatusWarning}}, StatusWarning},
		{"one critical", []Result{{Status: StatusOK}, {Status: StatusCritical}}, StatusCritical},
		{"mix all", []Result{{Status: StatusOK}, {Status: StatusWarning}, {Status: StatusCritical}}, StatusCritical},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AggregateStatus(tc.results)
			if got != tc.want {
				t.Errorf("AggregateStatus() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestComputeSummary verifies summary calculation from results
func TestComputeSummary(t *testing.T) {
	results := []Result{
		{CheckID: "a", Status: StatusOK},
		{CheckID: "b", Status: StatusOK},
		{CheckID: "c", Status: StatusWarning},
		{CheckID: "d", Status: StatusCritical},
	}

	summary := ComputeSummary(results)

	if summary.TotalCount != 4 {
		t.Errorf("TotalCount = %d, want 4", summary.TotalCount)
	}
	if summary.OkCount != 2 {
		t.Errorf("OkCount = %d, want 2", summary.OkCount)
	}
	if summary.WarnCount != 1 {
		t.Errorf("WarnCount = %d, want 1", summary.WarnCount)
	}
	if summary.CritCount != 1 {
		t.Errorf("CritCount = %d, want 1", summary.CritCount)
	}
	if summary.Status != StatusCritical {
		t.Errorf("Status = %v, want %v", summary.Status, StatusCritical)
	}
	if len(summary.Checks) != 4 {
		t.Errorf("Checks length = %d, want 4", len(summary.Checks))
	}
}

// TestComputeSummaryEmpty verifies empty results handling
func TestComputeSummaryEmpty(t *testing.T) {
	summary := ComputeSummary([]Result{})

	if summary.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", summary.TotalCount)
	}
	if summary.Status != StatusOK {
		t.Errorf("Empty summary Status = %v, want %v", summary.Status, StatusOK)
	}
}

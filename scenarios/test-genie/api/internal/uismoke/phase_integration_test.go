package uismoke

import (
	"testing"
	"time"
)

func TestResultToPhaseResult_Passed(t *testing.T) {
	result := &Result{
		Scenario:   "test-scenario",
		Status:     StatusPassed,
		Message:    "UI loaded successfully",
		DurationMs: 5000,
	}

	pr := resultToPhaseResult(result)

	if !pr.Success {
		t.Error("Success should be true for passed status")
	}
	if pr.Skipped {
		t.Error("Skipped should be false")
	}
	if pr.Blocked {
		t.Error("Blocked should be false")
	}
	if pr.Message != "ui smoke passed (5000ms)" {
		t.Errorf("Message = %q, want %q", pr.Message, "ui smoke passed (5000ms)")
	}
}

func TestResultToPhaseResult_PassedZeroDuration(t *testing.T) {
	result := &Result{
		Scenario:   "test-scenario",
		Status:     StatusPassed,
		Message:    "UI loaded successfully",
		DurationMs: 0,
	}

	pr := resultToPhaseResult(result)

	if pr.Message != "ui smoke passed" {
		t.Errorf("Message = %q, want %q", pr.Message, "ui smoke passed")
	}
}

func TestResultToPhaseResult_Skipped(t *testing.T) {
	result := &Result{
		Scenario: "test-scenario",
		Status:   StatusSkipped,
		Message:  "No UI directory",
	}

	pr := resultToPhaseResult(result)

	if !pr.Success {
		t.Error("Success should be true for skipped (not a failure)")
	}
	if !pr.Skipped {
		t.Error("Skipped should be true")
	}
	if pr.Blocked {
		t.Error("Blocked should be false")
	}
}

func TestResultToPhaseResult_Blocked(t *testing.T) {
	result := &Result{
		Scenario: "test-scenario",
		Status:   StatusBlocked,
		Message:  "Browserless offline",
	}

	pr := resultToPhaseResult(result)

	if pr.Success {
		t.Error("Success should be false for blocked")
	}
	if pr.Skipped {
		t.Error("Skipped should be false")
	}
	if !pr.Blocked {
		t.Error("Blocked should be true")
	}
}

func TestResultToPhaseResult_Failed(t *testing.T) {
	result := &Result{
		Scenario: "test-scenario",
		Status:   StatusFailed,
		Message:  "Handshake timeout",
	}

	pr := resultToPhaseResult(result)

	if pr.Success {
		t.Error("Success should be false for failed")
	}
	if pr.Skipped {
		t.Error("Skipped should be false")
	}
	if pr.Blocked {
		t.Error("Blocked should be false")
	}
}

func TestPhaseResult_FormatObservation(t *testing.T) {
	tests := []struct {
		name   string
		pr     PhaseResult
		expect string
	}{
		{
			name:   "skipped",
			pr:     PhaseResult{Skipped: true, Message: "No UI directory"},
			expect: "UI smoke skipped: No UI directory",
		},
		{
			name:   "blocked",
			pr:     PhaseResult{Blocked: true, Message: "Browserless offline"},
			expect: "UI smoke blocked: Browserless offline",
		},
		{
			name:   "success",
			pr:     PhaseResult{Success: true, Message: "ui smoke passed (5000ms)"},
			expect: "ui smoke passed (5000ms)",
		},
		{
			name:   "failed",
			pr:     PhaseResult{Success: false, Message: "Handshake timeout"},
			expect: "UI smoke failed: Handshake timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pr.FormatObservation(); got != tt.expect {
				t.Errorf("FormatObservation() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestPhaseResult_ToError(t *testing.T) {
	tests := []struct {
		name    string
		pr      PhaseResult
		wantErr bool
	}{
		{
			name:    "success - no error",
			pr:      PhaseResult{Success: true},
			wantErr: false,
		},
		{
			name:    "skipped - no error",
			pr:      PhaseResult{Skipped: true},
			wantErr: false,
		},
		{
			name: "failed - error",
			pr: PhaseResult{
				Success: false,
				Message: "Handshake timeout",
				Result:  &Result{Status: StatusFailed},
			},
			wantErr: true,
		},
		{
			name: "blocked - error",
			pr: PhaseResult{
				Success: false,
				Blocked: true,
				Message: "Browserless offline",
				Result:  &Result{Status: StatusBlocked},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pr.ToError()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPhaseResult_GetBundleStatus(t *testing.T) {
	tests := []struct {
		name       string
		pr         PhaseResult
		wantFresh  bool
		wantReason string
	}{
		{
			name:       "no result",
			pr:         PhaseResult{Result: nil},
			wantFresh:  true,
			wantReason: "",
		},
		{
			name:       "no bundle info",
			pr:         PhaseResult{Result: &Result{}},
			wantFresh:  true,
			wantReason: "",
		},
		{
			name: "fresh bundle",
			pr: PhaseResult{
				Result: &Result{
					Bundle: &BundleStatus{Fresh: true},
				},
			},
			wantFresh:  true,
			wantReason: "",
		},
		{
			name: "stale bundle",
			pr: PhaseResult{
				Result: &Result{
					Bundle: &BundleStatus{
						Fresh:  false,
						Reason: "Source files newer than bundle",
					},
				},
			},
			wantFresh:  false,
			wantReason: "Source files newer than bundle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fresh, reason := tt.pr.GetBundleStatus()
			if fresh != tt.wantFresh {
				t.Errorf("GetBundleStatus() fresh = %v, want %v", fresh, tt.wantFresh)
			}
			if reason != tt.wantReason {
				t.Errorf("GetBundleStatus() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestDefaultBrowserlessURL(t *testing.T) {
	if DefaultBrowserlessURL != "http://localhost:4110" {
		t.Errorf("DefaultBrowserlessURL = %q, want %q", DefaultBrowserlessURL, "http://localhost:4110")
	}
}

func TestDefaultTimeout(t *testing.T) {
	if DefaultTimeout != 90*time.Second {
		t.Errorf("DefaultTimeout = %v, want %v", DefaultTimeout, 90*time.Second)
	}
}

func TestDefaultHandshakeTimeout(t *testing.T) {
	if DefaultHandshakeTimeout != 15*time.Second {
		t.Errorf("DefaultHandshakeTimeout = %v, want %v", DefaultHandshakeTimeout, 15*time.Second)
	}
}

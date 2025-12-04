package smoke

import (
	"testing"
	"time"
)

func TestStatusIsSuccess(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPassed, true},
		{StatusFailed, false},
		{StatusSkipped, false},
		{StatusBlocked, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsSuccess(); got != tt.want {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPassed, true},
		{StatusFailed, true},
		{StatusSkipped, true},
		{StatusBlocked, true},
		{Status("running"), false},
		{Status(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewResult(t *testing.T) {
	r := NewResult("test-scenario", StatusPassed, "test passed")

	if r.Scenario != "test-scenario" {
		t.Errorf("Scenario = %q, want %q", r.Scenario, "test-scenario")
	}
	if r.Status != StatusPassed {
		t.Errorf("Status = %v, want %v", r.Status, StatusPassed)
	}
	if r.Message != "test passed" {
		t.Errorf("Message = %q, want %q", r.Message, "test passed")
	}
	if r.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestResultWithDuration(t *testing.T) {
	r := NewResult("test", StatusPassed, "ok")
	r.WithDuration(5 * time.Second)

	if r.DurationMs != 5000 {
		t.Errorf("DurationMs = %d, want 5000", r.DurationMs)
	}
}

func TestResultWithHandshake(t *testing.T) {
	r := NewResult("test", StatusPassed, "ok")
	h := HandshakeResult{
		Signaled:   true,
		DurationMs: 100,
	}
	r.WithHandshake(h)

	if !r.Handshake.Signaled {
		t.Error("Handshake.Signaled should be true")
	}
	if r.Handshake.DurationMs != 100 {
		t.Errorf("Handshake.DurationMs = %d, want 100", r.Handshake.DurationMs)
	}
}

func TestResultWithArtifacts(t *testing.T) {
	r := NewResult("test", StatusPassed, "ok")
	a := ArtifactPaths{
		Screenshot: "/path/to/screenshot.png",
		Console:    "/path/to/console.json",
	}
	r.WithArtifacts(a)

	if r.Artifacts.Screenshot != "/path/to/screenshot.png" {
		t.Errorf("Artifacts.Screenshot = %q, want %q", r.Artifacts.Screenshot, "/path/to/screenshot.png")
	}
	if r.Artifacts.Console != "/path/to/console.json" {
		t.Errorf("Artifacts.Console = %q, want %q", r.Artifacts.Console, "/path/to/console.json")
	}
}

func TestPassed(t *testing.T) {
	r := Passed("my-scenario", "http://localhost:3000", 2*time.Second)

	if r.Status != StatusPassed {
		t.Errorf("Status = %v, want %v", r.Status, StatusPassed)
	}
	if r.Scenario != "my-scenario" {
		t.Errorf("Scenario = %q, want %q", r.Scenario, "my-scenario")
	}
	if r.UIURL != "http://localhost:3000" {
		t.Errorf("UIURL = %q, want %q", r.UIURL, "http://localhost:3000")
	}
	if r.DurationMs != 2000 {
		t.Errorf("DurationMs = %d, want 2000", r.DurationMs)
	}
	if r.Message != "UI loaded successfully" {
		t.Errorf("Message = %q, want %q", r.Message, "UI loaded successfully")
	}
}

func TestFailed(t *testing.T) {
	r := Failed("my-scenario", "handshake timeout")

	if r.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", r.Status, StatusFailed)
	}
	if r.Scenario != "my-scenario" {
		t.Errorf("Scenario = %q, want %q", r.Scenario, "my-scenario")
	}
	if r.Message != "handshake timeout" {
		t.Errorf("Message = %q, want %q", r.Message, "handshake timeout")
	}
}

func TestSkipped(t *testing.T) {
	r := Skipped("my-scenario", "no UI directory")

	if r.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v", r.Status, StatusSkipped)
	}
	if r.Message != "no UI directory" {
		t.Errorf("Message = %q, want %q", r.Message, "no UI directory")
	}
}

func TestBlocked(t *testing.T) {
	r := Blocked("my-scenario", "browserless offline", BlockedReasonBrowserlessOffline)

	if r.Status != StatusBlocked {
		t.Errorf("Status = %v, want %v", r.Status, StatusBlocked)
	}
	if r.Message != "browserless offline" {
		t.Errorf("Message = %q, want %q", r.Message, "browserless offline")
	}
	if r.BlockedReason != BlockedReasonBrowserlessOffline {
		t.Errorf("BlockedReason = %v, want %v", r.BlockedReason, BlockedReasonBrowserlessOffline)
	}
}

func TestBlockedReason_ExitCode(t *testing.T) {
	tests := []struct {
		reason   BlockedReason
		expected int
	}{
		{BlockedReasonBrowserlessOffline, 50},
		{BlockedReasonBundleStale, 60},
		{BlockedReasonUIPortMissing, 61},
		{BlockedReasonNone, 1},
		{BlockedReason("unknown"), 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			if got := tt.reason.ExitCode(); got != tt.expected {
				t.Errorf("ExitCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

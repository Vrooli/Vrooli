package handshake

import (
	"testing"

	"test-genie/internal/uismoke/orchestrator"
)

func TestDetector_Evaluate_Success(t *testing.T) {
	d := NewDetector()
	raw := &orchestrator.HandshakeRaw{
		Signaled:   true,
		TimedOut:   false,
		DurationMs: 500,
	}

	result := d.Evaluate(raw)

	if !result.Signaled {
		t.Error("Signaled should be true")
	}
	if result.TimedOut {
		t.Error("TimedOut should be false")
	}
	if result.DurationMs != 500 {
		t.Errorf("DurationMs = %d, want 500", result.DurationMs)
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %q", result.Error)
	}
}

func TestDetector_Evaluate_Timeout(t *testing.T) {
	d := NewDetector()
	raw := &orchestrator.HandshakeRaw{
		Signaled:   false,
		TimedOut:   true,
		DurationMs: 15000,
		Error:      "Timeout exceeded",
	}

	result := d.Evaluate(raw)

	if result.Signaled {
		t.Error("Signaled should be false")
	}
	if !result.TimedOut {
		t.Error("TimedOut should be true")
	}
	if result.Error != "Timeout exceeded" {
		t.Errorf("Error = %q, want %q", result.Error, "Timeout exceeded")
	}
}

func TestDetector_Evaluate_Nil(t *testing.T) {
	d := NewDetector()
	result := d.Evaluate(nil)

	if result.Signaled {
		t.Error("Signaled should be false for nil input")
	}
	if result.Error == "" {
		t.Error("Error should be set for nil input")
	}
}

func TestIsSuccessful(t *testing.T) {
	tests := []struct {
		name   string
		result orchestrator.HandshakeResult
		want   bool
	}{
		{
			name: "successful",
			result: orchestrator.HandshakeResult{
				Signaled:   true,
				TimedOut:   false,
				DurationMs: 500,
			},
			want: true,
		},
		{
			name: "not signaled",
			result: orchestrator.HandshakeResult{
				Signaled: false,
			},
			want: false,
		},
		{
			name: "timed out",
			result: orchestrator.HandshakeResult{
				Signaled: true,
				TimedOut: true,
			},
			want: false,
		},
		{
			name: "has error",
			result: orchestrator.HandshakeResult{
				Signaled: true,
				Error:    "some error",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSuccessful(tt.result); got != tt.want {
				t.Errorf("IsSuccessful() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSuggestedAction(t *testing.T) {
	tests := []struct {
		name   string
		result orchestrator.HandshakeResult
		want   string
	}{
		{
			name: "successful - no action",
			result: orchestrator.HandshakeResult{
				Signaled: true,
			},
			want: "",
		},
		{
			name: "timed out",
			result: orchestrator.HandshakeResult{
				Signaled: false,
				TimedOut: true,
			},
			want: "The iframe-bridge did not signal ready within the timeout. " +
				"Ensure the UI properly initializes the bridge and calls its ready method.",
		},
		{
			name: "has error",
			result: orchestrator.HandshakeResult{
				Signaled: false,
				Error:    "connection failed",
			},
			want: "The handshake encountered an error: connection failed",
		},
		{
			name: "not signaled - generic",
			result: orchestrator.HandshakeResult{
				Signaled: false,
			},
			want: "The iframe-bridge handshake did not complete. " +
				"Verify that @vrooli/iframe-bridge is properly imported and initialized in the UI.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SuggestedAction(tt.result); got != tt.want {
				t.Errorf("SuggestedAction() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKnownSignals(t *testing.T) {
	if len(KnownSignals) == 0 {
		t.Error("KnownSignals should not be empty")
	}

	for i, signal := range KnownSignals {
		if signal.Property == "" {
			t.Errorf("KnownSignals[%d].Property should not be empty", i)
		}
		if signal.Description == "" {
			t.Errorf("KnownSignals[%d].Description should not be empty", i)
		}
	}
}

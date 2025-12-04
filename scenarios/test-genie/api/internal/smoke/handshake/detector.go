package handshake

import (
	"test-genie/internal/smoke/orchestrator"
)

// Signal defines a known iframe-bridge handshake signal.
type Signal struct {
	// Property is the window property path to check.
	Property string

	// Description describes what this signal indicates.
	Description string
}

// KnownSignals lists all recognized iframe-bridge handshake signals.
var KnownSignals = []Signal{
	{Property: "__vrooliBridgeChildInstalled", Description: "Legacy bridge marker"},
	{Property: "IFRAME_BRIDGE_READY", Description: "Global ready flag"},
	{Property: "IframeBridge.ready", Description: "Bridge instance ready property"},
	{Property: "iframeBridge.ready", Description: "Bridge instance ready (lowercase)"},
	{Property: "IframeBridge.getState().ready", Description: "Bridge state ready via getter"},
}

// Detector evaluates handshake results from browser responses.
type Detector struct{}

// NewDetector creates a new handshake Detector.
func NewDetector() *Detector {
	return &Detector{}
}

// Ensure Detector implements orchestrator.HandshakeDetector.
var _ orchestrator.HandshakeDetector = (*Detector)(nil)

// Evaluate interprets raw handshake data into a structured result.
func (d *Detector) Evaluate(raw *orchestrator.HandshakeRaw) orchestrator.HandshakeResult {
	if raw == nil {
		return orchestrator.HandshakeResult{
			Signaled: false,
			Error:    "no handshake data available",
		}
	}

	return orchestrator.HandshakeResult{
		Signaled:   raw.Signaled,
		TimedOut:   raw.TimedOut,
		DurationMs: raw.DurationMs,
		Error:      raw.Error,
	}
}

// IsSuccessful returns true if the handshake completed successfully.
func IsSuccessful(result orchestrator.HandshakeResult) bool {
	return result.Signaled && !result.TimedOut && result.Error == ""
}

// SuggestedAction returns a human-readable suggestion based on the handshake result.
func SuggestedAction(result orchestrator.HandshakeResult) string {
	if result.Signaled {
		return ""
	}

	if result.TimedOut {
		return "The iframe-bridge did not signal ready within the timeout. " +
			"Ensure the UI properly initializes the bridge and calls its ready method."
	}

	if result.Error != "" {
		return "The handshake encountered an error: " + result.Error
	}

	return "The iframe-bridge handshake did not complete. " +
		"Verify that @vrooli/iframe-bridge is properly imported and initialized in the UI."
}

package capabilities_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/capabilities"
)

// TestCheckersWithoutDoerAreUnknownNotPanics covers a partial composition: a
// checker wired without its HTTP doer used to panic inside the health path.
// Unknown is the honest state (the probe could not run), and it must not take
// down the caller.
func TestCheckersWithoutDoerAreUnknownNotPanics(t *testing.T) {
	checkers := map[string]capabilities.Checker{
		"resource":   &capabilities.ResourceChecker{URL: "http://127.0.0.1:1/ready"},
		"whisper":    &capabilities.WhisperChecker{BaseURL: "http://127.0.0.1:1"},
		"kokoro":     &capabilities.KokoroChecker{BaseURL: "http://127.0.0.1:1"},
		"ollama":     &capabilities.OllamaChecker{BaseURL: "http://127.0.0.1:1"},
		"openrouter": &capabilities.BYOKCredentialChecker{ProviderID: "openrouter", Capability: "summarize"},
	}
	for name, checker := range checkers {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				status, message := checker.Check(context.Background())
				require.Equal(t, capabilities.StatusUnknown, status)
				require.NotEmpty(t, message)
			})
		})
	}
}

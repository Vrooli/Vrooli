// Package backends selects per-harness behavior for agent sessions.
package backends

import (
	"web-console/backends/claude"
	"web-console/backends/codex"
	"web-console/backends/grok"
	"web-console/internal/backend"
)

// PromptDetectorFor returns the screen detector for an agent harness
// ("claude", "codex", "grok", "opencode"). OpenCode reports its state through
// its event stream instead of the screen, and unknown harnesses have no
// detector: both return nil and the session relies on its output clock.
func PromptDetectorFor(agent string) backend.PromptDetector {
	switch agent {
	case "claude":
		return claude.DefaultPromptDetector()
	case "codex":
		return codex.DefaultPromptDetector()
	case "grok":
		return grok.DefaultPromptDetector()
	default:
		return nil
	}
}

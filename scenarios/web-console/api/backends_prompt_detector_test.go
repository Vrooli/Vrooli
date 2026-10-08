package main

import (
	"testing"

	"web-console/backends"
)

// Prompt detection is selected by the session's agent harness.
func TestPromptDetectorForHarness(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "grok"} {
		if backends.PromptDetectorFor(agent) == nil {
			t.Fatalf("PromptDetectorFor(%q) = nil, want a screen detector", agent)
		}
	}
	// OpenCode reports state through its event stream; unknown agents have none.
	for _, agent := range []string{"opencode", "none", ""} {
		if backends.PromptDetectorFor(agent) != nil {
			t.Fatalf("PromptDetectorFor(%q) returned a screen detector, want nil", agent)
		}
	}
}

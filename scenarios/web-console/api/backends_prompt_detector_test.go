package main

import (
	"testing"

	"web-console/backends/codex"
)

func TestBackendPromptDetectorConstructs(t *testing.T) {
	if detector := codex.DefaultPromptDetector(); detector == nil {
		t.Fatal("codex prompt detector is nil")
	}
}

package main

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateDraft_LiveOpenRouter_Compliance(t *testing.T) {
	if strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	// Avoid forcing a real OpenRouter call unless explicitly enabled.
	if strings.TrimSpace(os.Getenv("VROOLI_LIVE_OPENROUTER_TESTS")) != "1" {
		t.Skip("set VROOLI_LIVE_OPENROUTER_TESTS=1 to enable live OpenRouter test")
	}

	draft := Draft{EntityType: "scenario", EntityName: "live-openrouter-test"}
	text, _, err := generateAIContent(draft, "🎯 Full PRD", "Single-line brief: build a tiny scenario that proves PRD generation works.", "", false, nil, "")
	if err != nil {
		t.Fatalf("generateAIContent error: %v", err)
	}
	result := ValidatePRDTemplateV2(text)
	if !isPRDTemplateCompliant(result) {
		t.Fatalf("generated PRD not compliant: %s", summarizePRDTemplateIssues(result))
	}
}

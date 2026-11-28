package transcriptindex

import (
	"testing"

	"agent-manager/internal/runsignal"
)

// Turn tool kinds are runsignal's kinds; MCP calls are named by server and
// tool and a shell call that runs a scenario's CLI by the catalogued scenario.
func TestToolKindMapsThroughTheRunsignalTaxonomy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input map[string]any
		want  string
	}{
		{"Read", nil, runsignal.KindFileRead},
		{"WebFetch", nil, runsignal.KindWebFetch},
		{"spawn_agent", nil, runsignal.KindDelegate},
		{"mcp__chrome__navigate", nil, "mcp:chrome.navigate"},
		{"Bash", map[string]any{"command": "image-tools generate --prompt cat"}, "scenario:image-tools"},
		{"Bash", map[string]any{"command": "echo three attempts"}, runsignal.KindShell},
		{"exec", map[string]any{"input": `await tools.exec_command({cmd:"prompt-manager skill read goal-loop"})`}, "scenario:prompt-manager"},
	} {
		if got := ToolKind(tc.name, tc.input); got != tc.want {
			t.Errorf("ToolKind(%q, %v) = %q, want %q", tc.name, tc.input, got, tc.want)
		}
	}
	var facts factAccumulator
	facts.addTool(ToolKind("WebSearch", nil))
	facts.addTool(ToolKind("webfetch", nil))
	if facts.facts.WebSearches != 1 || facts.facts.WebFetches != 1 {
		t.Fatalf("web counts=%+v", facts.facts)
	}
}

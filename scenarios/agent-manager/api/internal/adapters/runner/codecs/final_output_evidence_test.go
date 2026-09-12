package codecs

import (
	"encoding/json"
	"os"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestSupportedCodecTracesResolveCanonicalFinalOutput(t *testing.T) {
	tests := []struct {
		name  string
		trace string
		codec func() Codec
		want  string
	}{
		{name: "claude", trace: "claude_trace.jsonl", codec: func() Codec { return NewClaudeForTest() }, want: "Done."},
		{name: "codex", trace: "codex_trace.jsonl", codec: func() Codec { return NewCodexForTest() }, want: "Created `test123.txt`."},
		{name: "opencode", trace: "opencode_trace.jsonl", codec: func() Codec { return NewOpenCodeForTest() }, want: "All done."},
		{name: "grok", trace: "grok_trace.jsonl", codec: func() Codec { return NewGrokForTest() }, want: "pong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := tt.codec()
			state := codec.NewState()
			runID := uuid.New()
			var events []*domain.RunEvent
			for _, line := range readGoldenLines(t, tt.trace) {
				decoded, err := codec.DecodeStreamLine(state, runID, line)
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				events = append(events, decoded...)
			}
			result := domain.ResolveRunResult(events, true, 0, "completed")
			if result.Selection.Status != domain.FinalOutputSelectionSelected {
				t.Fatalf("selection = %#v", result.Selection)
			}
			if result.FinalOutput != tt.want {
				t.Fatalf("final output = %q, want %q", result.FinalOutput, tt.want)
			}
			var selected domain.FinalOutputCandidate
			for _, candidate := range result.Candidates {
				if candidate.ID == result.Selection.SelectedCandidateID {
					selected = candidate
					break
				}
			}
			if !selected.Terminal || selected.ProviderOrigin == "" || selected.ProviderEventType == "" || selected.RawEvidenceRef == "" {
				t.Fatalf("selected candidate lacks terminal provider provenance: %#v", selected)
			}
		})
	}
}

func TestAdversarialFinalOutputFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/final_output_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Name           string                            `json:"name"`
			Runner         string                            `json:"runner"`
			Events         []json.RawMessage                 `json:"events"`
			ExpectedStatus domain.FinalOutputSelectionStatus `json:"expectedStatus"`
			ExpectedText   string                            `json:"expectedText"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			var codec Codec
			switch tc.Runner {
			case "claude-code":
				codec = NewClaudeForTest()
			case "codex":
				codec = NewCodexForTest()
			case "opencode":
				codec = NewOpenCodeForTest()
			case "grok":
				codec = NewGrokForTest()
			default:
				t.Fatalf("unknown runner %q", tc.Runner)
			}
			parser := codec.NewTranscriptParser()
			runID := uuid.New()
			var events []*domain.RunEvent
			for _, raw := range tc.Events {
				parsed := parser.ParseTranscriptLine(runID, string(raw)+"\n")
				if parsed.Err != nil {
					t.Fatalf("parse: %v", parsed.Err)
				}
				events = append(events, parsed.Events...)
			}
			result := domain.ResolveRunResult(events, true, 0, "completed")
			if result.Selection.Status != tc.ExpectedStatus || result.FinalOutput != tc.ExpectedText {
				t.Fatalf("result = status %q text %q; want status %q text %q", result.Selection.Status, result.FinalOutput, tc.ExpectedStatus, tc.ExpectedText)
			}
		})
	}
}

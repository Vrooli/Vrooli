package interactive

import "testing"

func TestClassifyScreenInterruption(t *testing.T) {
	tests := []struct {
		name, screen, kind string
		want               bool
	}{
		{"capacity", "Selected model is at capacity. Please try a different model.", "model_capacity", true},
		{"rate limit", "429: rate limit exceeded; retry later", "rate_limit", true},
		{"provider unavailable", "Provider temporarily unavailable", "provider_unavailable", true},
		{"ordinary prose", "The agent discussed model capacity in the report", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyScreenInterruption(tt.screen)
			if ok != tt.want || (ok && got.Kind != tt.kind) {
				t.Fatalf("classification = (%+v, %v), want kind=%q detected=%v", got, ok, tt.kind, tt.want)
			}
		})
	}
}

func TestClassifyCodexStalledGoalFooter(t *testing.T) {
	screen := "› Ask Codex to do anything\n\n" +
		"GPT-6-Luna … Goal stalled (/goal resume) │ active-pane status\n" +
		"? for shortcuts                  ⚠ 2 · f2 │ right pane\n\n\n"
	got, ok := ClassifyScreenInterruption(screen)
	if !ok || got.Kind != "goal_stalled" || got.Message != "Goal stalled (/goal resume)" {
		t.Fatalf("classification = (%+v, %v), want current stalled-goal footer", got, ok)
	}
}

func TestClassifyCodexStalledGoalFooterRejectsHistoryAndIncompleteStates(t *testing.T) {
	tests := []struct {
		name   string
		screen string
	}{
		{
			name: "old footer followed by current ordinary prompt",
			screen: "GPT-6-Luna \u2026 Goal stalled (/goal resume)\n" +
				"? for shortcuts\n" +
				"\u203a Ask Codex to do anything",
		},
		{
			name: "quoted footer in transcript",
			screen: "The log says Goal stalled (/goal resume), then recovered.\n" +
				"\u203a Ask Codex to do anything\n? for shortcuts",
		},
		{
			name: "working screen after previous footer",
			screen: "\u203a Ask Codex to do anything\n\n" +
				"GPT-6-Luna \u2026 Goal stalled (/goal resume)\n? for shortcuts\n" +
				"Working (3s \u00b7 esc to interrupt)",
		},
		{
			name: "stalled text without resume command",
			screen: "\u203a Ask Codex to do anything\n\n" +
				"GPT-6-Luna \u2026 Goal stalled\n? for shortcuts",
		},
		{
			name: "status without prompt separator",
			screen: "\u203a Ask Codex to do anything\n" +
				"GPT-6-Luna \u2026 Goal stalled (/goal resume)\n? for shortcuts",
		},
		{
			name: "trailing status commentary after resume marker",
			screen: "\u203a Ask Codex to do anything\n\n" +
				"GPT-6-Luna \u2026 Goal stalled (/goal resume) but continuing\n" +
				"? for shortcuts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyScreenInterruption(tt.screen)
			if ok && got.Kind == "goal_stalled" {
				t.Fatalf("classification = (%+v, %v), want no stalled-goal classification", got, ok)
			}
		})
	}
}

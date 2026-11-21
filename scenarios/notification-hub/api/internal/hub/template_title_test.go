package hub

import "testing"

func TestTemplateConversationTitle(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want string
	}{
		{"daily-briefing", map[string]any{"title": "Portfolio status for 2026-11-16"}, "Daily briefing · Mon 16 Nov"},
		{"opportunity-roadmap", map[string]any{"title": "Opportunity roadmap for 2026-11-16"}, "Opportunity roadmap · 16 Nov"},
		{"research-report", map[string]any{"title": "Work progressed", "topic": "Local model performance"}, "Research report · Local model performance"},
		{"research-report", map[string]any{"title": "Work progressed", "topic": "local-model-performance"}, "Research report · Local model performance"},
		{"usage-mode", map[string]any{"title": "Keep Claude on low power?"}, "Usage mode: Keep Claude on low power?"},
		{"usage-mode", map[string]any{"title": "Usage mode: choose?"}, "Usage mode: choose?"},
		{"structural-change", map[string]any{"title": "Change the check?"}, "Structural change: Change the check?"},
		{"other", map[string]any{"title": "A named update"}, "A named update"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := templateConversationTitle(tt.name, map[string]any{"meta": tt.meta}, "Summary"); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

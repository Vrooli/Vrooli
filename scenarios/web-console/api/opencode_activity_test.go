package main

import (
	"testing"

	"web-console/backends/opencode"
)

// [REQ:P0-017i] OpenCode's own events carry the waiting prompt.
func TestOpenCodePromptFromPermissionAsked(t *testing.T) {
	ev := opencode.Event{Type: "permission.asked", Properties: map[string]any{
		"id": "per_1", "sessionID": "ses_a", "permission": "bash", "patterns": []any{"git status"},
	}}
	prompt := opencodePrompt(ev)
	if prompt == nil || prompt.Kind != "permission" || prompt.Text != "bash git status" {
		t.Fatalf("prompt = %+v, want a bash permission for git status", prompt)
	}
	keys := []string{}
	for _, option := range prompt.Options {
		keys = append(keys, option.Key)
	}
	if len(keys) != 3 || keys[0] != "once" || keys[1] != "always" || keys[2] != "reject" {
		t.Fatalf("option keys = %v, want OpenCode's replies once/always/reject", keys)
	}
}

func TestOpenCodePromptFromQuestionAsked(t *testing.T) {
	ev := opencode.Event{Type: "question.asked", Properties: map[string]any{
		"id": "que_1", "sessionID": "ses_a",
		"questions": []any{map[string]any{
			"question": "Which database?", "header": "DB",
			"options": []any{map[string]any{"label": "SQLite", "description": "local"}, map[string]any{"label": "Postgres", "description": "server"}},
			"custom":  false,
		}},
	}}
	prompt := opencodePrompt(ev)
	if prompt == nil || prompt.Kind != "question" || prompt.Text != "Which database?" {
		t.Fatalf("prompt = %+v, want the question", prompt)
	}
	if len(prompt.Options) != 2 || prompt.Options[1].Key != "Postgres" {
		t.Fatalf("options = %+v, want SQLite and Postgres keyed by label", prompt.Options)
	}
	if prompt.FreeTextHint != "" {
		t.Fatalf("custom=false must not offer a typed answer, got %q", prompt.FreeTextHint)
	}
}

func TestOpenCodeEventsForUnclaimedSessionsAreIgnored(t *testing.T) {
	w := &OpenCodeWatcher{server: newFakeTestServer(), claimed: map[string]string{}}
	// No claim, no live pane: must not panic or route anywhere.
	w.applyActivity(opencode.Event{Type: "permission.asked", Properties: map[string]any{"sessionID": "ses_unclaimed"}})
}

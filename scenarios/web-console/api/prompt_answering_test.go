package main

import (
	"os"
	"path/filepath"
	"testing"

	"web-console/internal/backend"
)

var answerTestQuestion = backend.PendingPrompt{Kind: "question", Text: "Pick", Options: []backend.PromptOption{{Key: "1", Label: "Red"}}}

func TestPromptAnsweringIsOffByDefault(t *testing.T) {
	// [REQ:P0-017i] Level 3 is opt-in per host: with no setting nothing is answerable.
	policy := newPromptAnswering("", func() string { return "2.1.268" })
	if _, ok := policy.decide("claude", answerTestQuestion); ok {
		t.Fatal("claude prompt answerable with answering off")
	}
	permission := backend.PendingPrompt{Kind: "permission", Options: answerTestQuestion.Options}
	if _, ok := policy.decide("opencode", permission); ok {
		t.Fatal("opencode prompt answerable with answering off")
	}
}

func TestPromptAnsweringFollowsHarnessAndVersion(t *testing.T) {
	// [REQ:P0-017i] With claude and opencode enabled, a Claude prompt is
	// answerable only on a verified version; OpenCode only for permissions.
	verified := newPromptAnswering(" claude , opencode ", func() string { return "2.1.268" })
	if version, ok := verified.decide("claude", answerTestQuestion); !ok || version != "2.1.268" {
		t.Fatalf("verified claude = (%q, %v), want answerable on 2.1.268", version, ok)
	}
	unverified := newPromptAnswering("claude", func() string { return "9.9.9" })
	if version, ok := unverified.decide("claude", answerTestQuestion); ok || version != "9.9.9" {
		t.Fatalf("unverified claude = (%q, %v), want the version reported and not answerable", version, ok)
	}
	if _, ok := verified.decide("claude", backend.PendingPrompt{Kind: "unknown"}); ok {
		t.Fatal("an unread prompt is not answerable")
	}
	permission := backend.PendingPrompt{Kind: "permission", Text: "bash", Options: []backend.PromptOption{{Key: "once", Label: "Allow once"}}}
	if _, ok := verified.decide("opencode", permission); !ok {
		t.Fatal("opencode permission not answerable with opencode enabled")
	}
	if _, ok := verified.decide("opencode", answerTestQuestion); ok {
		t.Fatal("opencode questions stay read-only")
	}
	if _, ok := newPromptAnswering("opencode", func() string { return "2.1.268" }).decide("claude", answerTestQuestion); ok {
		t.Fatal("claude answerable with only opencode enabled")
	}
	if _, ok := verified.decide("codex", answerTestQuestion); ok {
		t.Fatal("codex has no answering path")
	}
}

func TestPromptAnsweringSettingComesFromTheEnvironmentOrTheHostFile(t *testing.T) {
	// [REQ:P0-017i] The operator turns answering on per host: the environment
	// variable wins; otherwise a prompt-answering file in the data directory.
	path := filepath.Join(t.TempDir(), "prompt-answering")
	if got := promptAnsweringSetting("", path); got != "" {
		t.Fatalf("no env and no file = %q, want off", got)
	}
	if err := os.WriteFile(path, []byte("claude\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := promptAnsweringSetting("", path); got != "claude" {
		t.Fatalf("host file = %q, want claude", got)
	}
	if got := promptAnsweringSetting("opencode", path); got != "opencode" {
		t.Fatalf("env with a host file = %q, want the env", got)
	}
}

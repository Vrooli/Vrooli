package codecs

import (
	"strings"
	"testing"
)

// TestCapabilitiesConformance pins the per-runner capability contract to the
// matrix documented in scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md
// ("Codec capability contract"). It is the drift guard required by the
// coding-agent parity plan: identical-upstream-capability ⇒ identical
// contract. All three runners now expose the same boolean capabilities; the
// only intentional divergences are the curated cloud model lists and
// claude-code's lack of a local-Ollama path (it is Anthropic-native).
func TestCapabilitiesConformance(t *testing.T) {
	type want struct {
		messages, toolEvents, cost, streaming, cancel, continuation, image bool
		maxTurns                                                           int
		curatedModels                                                      []string
		allowsLocalOllama                                                  bool
	}
	// The uniform boolean contract: every coding agent supports all six.
	uniform := want{
		messages: true, toolEvents: true, cost: true,
		streaming: true, cancel: true, continuation: true, image: true,
		maxTurns: 0,
	}

	cases := []struct {
		name  string
		codec Codec
		want  want
	}{
		{
			name:  "claude",
			codec: NewClaudeForTest(),
			want: func() want {
				w := uniform
				w.curatedModels = []string{"sonnet", "opus", "haiku", "claude-sonnet-4-6"}
				w.allowsLocalOllama = false // Anthropic-native, documented difference
				return w
			}(),
		},
		{
			name:  "codex",
			codec: NewCodexForTest(),
			want: func() want {
				w := uniform
				w.curatedModels = []string{"gpt-5.5", "gpt-5.4", "gpt-5.3-codex"}
				w.allowsLocalOllama = true
				return w
			}(),
		},
		{
			name:  "opencode",
			codec: NewOpenCodeForTest(),
			want: func() want {
				w := uniform
				w.curatedModels = []string{"openrouter/deepseek/deepseek-v4-pro", "openrouter/anthropic/claude-opus-4.8"}
				w.allowsLocalOllama = true
				return w
			}(),
		},
		{
			// Grok intentionally diverges from the uniform contract: its
			// headless stdout surfaces assistant text + session id but NOT
			// tool events or token/cost (trace-verified, R3). These bools are
			// honest, not aspirational.
			name:  "grok",
			codec: NewGrokForTest(),
			want: want{
				messages: true, toolEvents: false, cost: false,
				streaming: true, cancel: true, continuation: true, image: false,
				maxTurns:          0,
				curatedModels:     []string{"grok-build", "grok-composer-2.5-fast"},
				allowsLocalOllama: false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caps := tc.codec.Capabilities()
			checkBool(t, "SupportsMessages", caps.SupportsMessages, tc.want.messages)
			checkBool(t, "SupportsToolEvents", caps.SupportsToolEvents, tc.want.toolEvents)
			checkBool(t, "SupportsCostTracking", caps.SupportsCostTracking, tc.want.cost)
			checkBool(t, "SupportsStreaming", caps.SupportsStreaming, tc.want.streaming)
			checkBool(t, "SupportsCancellation", caps.SupportsCancellation, tc.want.cancel)
			checkBool(t, "SupportsContinuation", caps.SupportsContinuation, tc.want.continuation)
			checkBool(t, "SupportsImageAttachments", caps.SupportsImageAttachments, tc.want.image)
			if caps.MaxTurns != tc.want.maxTurns {
				t.Errorf("MaxTurns = %d, want %d", caps.MaxTurns, tc.want.maxTurns)
			}
			// Curated cloud models must all be present (subset check, tolerant
			// of dynamically-appended ollama/* entries).
			for _, m := range tc.want.curatedModels {
				if indexOf(caps.SupportedModels, m) == -1 {
					t.Errorf("curated model %q missing from SupportedModels %v", m, caps.SupportedModels)
				}
			}
			// ForTest codecs leave the ollama lister nil, so no ollama/* entry
			// should surface regardless of allowsLocalOllama. The dynamic-append
			// behaviour is covered by TestCodex_Capabilities_AppendsOllamaModels.
			for _, m := range caps.SupportedModels {
				if strings.HasPrefix(m, ollamaModelPrefix) {
					t.Errorf("ForTest codec must not surface ollama model %q", m)
				}
			}
		})
	}
}

func checkBool(t *testing.T, field string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

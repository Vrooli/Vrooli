package codecs

import (
	"reflect"
	"testing"

	"agent-manager/internal/adapters/runner"
)

// TestCapabilitiesConformance pins the per-runner capability contract to the
// matrix documented in scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md
// ("Codec capability contract"). It is the drift guard required by the
// coding-agent parity plan: identical-upstream-capability ⇒ identical
// contract. The print-only Antigravity adapter intentionally exposes a smaller
// honest contract because its stable surface does not include tools, usage, or
// image attachments.
func TestCapabilitiesConformance(t *testing.T) {
	cases := []struct {
		name  string
		codec Codec
		want  runner.Capabilities
	}{
		{
			name:  "claude",
			codec: NewClaudeForTest(),
			want: runner.Capabilities{
				SupportsMessages: true, SupportsToolEvents: true, SupportsCostTracking: true,
				SupportsStreaming: true, SupportsCancellation: true, SupportsContinuation: true,
				SupportsWarmIteration: true, SupportsImageAttachments: true,
				SupportsToolRestriction: true,
				ToolRestrictionMappings: map[string]string{"read": "Read", "write": "Write", "edit": "Edit", "glob": "Glob", "grep": "Grep", "shell": "Bash", "web_search": "WebSearch", "web_fetch": "WebFetch"},
				SupportsEffort:          true, EffortMappings: map[string]string{"low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": "max"},
				SupportsRunnerDefault: true, SupportedFeatures: []string{"EnableBrowser"},
			},
		},
		{
			name:  "codex",
			codec: NewCodexForTest(),
			want: runner.Capabilities{
				SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected", "tracking", "off"}}, {ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true}},
				SupportsMessages:  true, SupportsToolEvents: true, SupportsCostTracking: true,
				SupportsStreaming: true, SupportsCancellation: true, SupportsContinuation: true,
				SupportsWarmIteration: true, SupportsImageAttachments: true,
				ToolRestrictionMappings: map[string]string{}, SupportsEffort: true,
				EffortMappings:        map[string]string{"low": "model_reasoning_effort=low", "medium": "model_reasoning_effort=medium", "high": "model_reasoning_effort=high", "xhigh": "model_reasoning_effort=xhigh"},
				SupportsRunnerDefault: true, DynamicModelPrefixes: []string{ollamaModelPrefix},
				SupportedFeatures: []string{}, AllowedExtraFlags: []string{"--verbose", "-c"},
			},
		},
		{
			name:  "opencode",
			codec: NewOpenCodeForTest(),
			want: runner.Capabilities{
				SupportsMessages: true, SupportsToolEvents: true, SupportsCostTracking: true,
				SupportsStreaming: true, SupportsCancellation: true, SupportsContinuation: true,
				SupportsWarmIteration: true, SupportsImageAttachments: true,
				ToolRestrictionMappings: map[string]string{}, SupportsEffort: true,
				EffortMappings: map[string]string{}, EffortModelSpecific: true,
				SupportsRunnerDefault: true, DynamicModelPrefixes: []string{ollamaModelPrefix},
				SupportedFeatures: []string{}, AllowedExtraFlags: []string{"--verbose"},
			},
		},
		{
			// Grok intentionally diverges from the uniform contract: its
			// headless stdout surfaces assistant text + session id but NOT
			// tool events or token/cost (trace-verified, R3). These bools are
			// honest, not aspirational.
			name:  "grok",
			codec: NewGrokForTest(),
			want: runner.Capabilities{
				SupportsMessages: true, SupportsStreaming: true, SupportsCancellation: true,
				SupportsContinuation: true, SupportsWarmIteration: true,
				SupportsToolRestriction: true,
				ToolRestrictionMappings: map[string]string{"read": "Read", "write": "Write", "edit": "Edit", "glob": "Glob", "grep": "Grep", "shell": "Bash", "web_search": "WebSearch", "web_fetch": "WebFetch"},
				SupportsEffort:          true, EffortMappings: map[string]string{"low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": "max"},
				SupportsRunnerDefault: true, SupportedFeatures: []string{},
			},
		},
		{
			name:  "antigravity",
			codec: NewAntigravityForTest(),
			want: runner.Capabilities{
				SupportsMessages: true, SupportsStreaming: true, SupportsCancellation: true,
				SupportsContinuation: true, SupportsWarmIteration: true,
				SupportsRunnerDefault: true, SupportedFeatures: []string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.codec.(CommandExtractor); !ok {
				t.Fatal("codec does not implement CommandExtractor")
			}
			if got := tc.codec.Capabilities(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("capabilities changed:\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func CanonicalToolNamesForTest() []string {
	return []string{"read", "write", "edit", "glob", "grep", "shell", "web_search", "web_fetch"}
}

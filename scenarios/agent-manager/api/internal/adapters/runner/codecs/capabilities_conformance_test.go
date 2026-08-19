package codecs

import (
	"slices"
	"testing"
)

// TestCapabilitiesConformance pins the per-runner capability contract to the
// matrix documented in scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md
// ("Codec capability contract"). It is the drift guard required by the
// coding-agent parity plan: identical-upstream-capability ⇒ identical
// contract. The print-only Antigravity adapter intentionally exposes a smaller
// honest contract because its stable surface does not include tools, usage, or
// image attachments.
func TestCapabilitiesConformance(t *testing.T) {
	type want struct {
		messages, toolEvents, cost, streaming, cancel, continuation, image bool
		maxTurns                                                           int
		supportsRunnerDefault                                              bool
		supportsToolRestriction                                            bool
		supportsEffort                                                     bool
		effortModelSpecific                                                bool
		effortMappingCount                                                 int
		mappingCount                                                       int
		dynamicModelPrefixes                                               []string
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
				w.supportsRunnerDefault = true
				w.supportsToolRestriction = true
				w.mappingCount = len(CanonicalToolNamesForTest())
				w.supportsEffort = true
				w.effortMappingCount = 5
				return w
			}(),
		},
		{
			name:  "codex",
			codec: NewCodexForTest(),
			want: func() want {
				w := uniform
				w.supportsRunnerDefault = true
				w.dynamicModelPrefixes = []string{ollamaModelPrefix}
				w.supportsEffort = true
				w.effortMappingCount = 4
				return w
			}(),
		},
		{
			name:  "opencode",
			codec: NewOpenCodeForTest(),
			want: func() want {
				w := uniform
				w.supportsRunnerDefault = true
				w.dynamicModelPrefixes = []string{ollamaModelPrefix}
				w.supportsEffort = true
				w.effortModelSpecific = true
				w.effortMappingCount = 0
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
				maxTurns:                0,
				supportsRunnerDefault:   true,
				supportsToolRestriction: true,
				supportsEffort:          true,
				effortMappingCount:      5,
				mappingCount:            len(CanonicalToolNamesForTest()),
			},
		},
		{
			name:  "antigravity",
			codec: NewAntigravityForTest(),
			want: want{
				messages: true, toolEvents: false, cost: false,
				streaming: true, cancel: true, continuation: true, image: false,
				maxTurns:              0,
				supportsRunnerDefault: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.codec.(CommandExtractor); !ok {
				t.Fatal("codec does not implement CommandExtractor")
			}
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
			if caps.SupportsRunnerDefault != tc.want.supportsRunnerDefault {
				t.Errorf("SupportsRunnerDefault = %v, want %v", caps.SupportsRunnerDefault, tc.want.supportsRunnerDefault)
			}
			if caps.SupportsToolRestriction != tc.want.supportsToolRestriction {
				t.Errorf("SupportsToolRestriction = %v, want %v", caps.SupportsToolRestriction, tc.want.supportsToolRestriction)
			}
			if caps.SupportsEffort != tc.want.supportsEffort {
				t.Errorf("SupportsEffort = %v, want %v", caps.SupportsEffort, tc.want.supportsEffort)
			}
			if caps.EffortModelSpecific != tc.want.effortModelSpecific {
				t.Errorf("EffortModelSpecific = %v, want %v", caps.EffortModelSpecific, tc.want.effortModelSpecific)
			}
			if len(caps.EffortMappings) != tc.want.effortMappingCount {
				t.Errorf("EffortMappings = %v, want %d mappings", caps.EffortMappings, tc.want.effortMappingCount)
			}
			if len(caps.ToolRestrictionMappings) != tc.want.mappingCount {
				t.Errorf("ToolRestrictionMappings = %v, want %d entries", caps.ToolRestrictionMappings, tc.want.mappingCount)
			}
			if !slices.Equal(caps.DynamicModelPrefixes, tc.want.dynamicModelPrefixes) {
				t.Errorf("DynamicModelPrefixes = %v, want %v", caps.DynamicModelPrefixes, tc.want.dynamicModelPrefixes)
			}
			if len(caps.SupportedModels) != 0 {
				t.Errorf("raw codec compiled static models into capabilities: %v", caps.SupportedModels)
			}
		})
	}
}

func CanonicalToolNamesForTest() []string {
	return []string{"read", "write", "edit", "glob", "grep", "shell", "web_search", "web_fetch"}
}

func checkBool(t *testing.T, field string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

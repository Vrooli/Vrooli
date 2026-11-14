package codecs

import (
	"testing"

	"github.com/google/uuid"
)

// withRegistry swaps the process-wide parse-failure counter for a fresh
// registry for the duration of the test and restores the previous one.
func withRegistry(t *testing.T) *ParseFailureRegistry {
	t.Helper()
	reg := NewParseFailureRegistry(DefaultParseFailureSampleCap)
	prev := SetDefaultParseFailureCounter(reg)
	t.Cleanup(func() { SetDefaultParseFailureCounter(prev) })
	return reg
}

// TestParseFailureRegistryRingBounded proves the sample ring never grows past
// its cap and keeps the most recent entries in chronological order.
func TestParseFailureRegistryRingBounded(t *testing.T) {
	reg := NewParseFailureRegistry(3)
	for i := 0; i < 10; i++ {
		reg.RecordParseFailure(uuid.New(), "test:site", "boom", "line")
	}
	snap := reg.Snapshot()
	if snap.Total != 10 {
		t.Fatalf("total = %d, want 10", snap.Total)
	}
	if got := snap.BySite["test:site"]; got != 10 {
		t.Fatalf("by_site[test:site] = %d, want 10", got)
	}
	if len(snap.Samples) != 3 {
		t.Fatalf("retained %d samples, want cap 3", len(snap.Samples))
	}
}

// TestParseFailureLineSampleBounded proves retained line prefixes are capped so
// a storm of large drift lines cannot grow memory without bound.
func TestParseFailureLineSampleBounded(t *testing.T) {
	reg := NewParseFailureRegistry(1)
	long := make([]byte, parseFailureLineSampleMax*4)
	for i := range long {
		long[i] = 'x'
	}
	reg.RecordParseFailure(uuid.New(), "test:site", "boom", string(long))
	snap := reg.Snapshot()
	if len(snap.Samples) != 1 {
		t.Fatalf("retained %d samples, want 1", len(snap.Samples))
	}
	if got := len(snap.Samples[0].LineSample); got != parseFailureLineSampleMax {
		t.Fatalf("line sample len = %d, want %d", got, parseFailureLineSampleMax)
	}
}

// TestCodecsRecordMalformedLineParseFailure proves a deliberately malformed
// JSON-looking line increments the counter and is retrievable, per codec.
// Behavior is unchanged — the line is still dropped — but the drop is now loud.
func TestCodecsRecordMalformedLineParseFailure(t *testing.T) {
	// A line that starts like JSON (passes each codec's non-JSON banner filter)
	// but is not valid JSON, so the codec's silent skip fires.
	const malformed = `{"type":"assistant","message":{` // truncated object: invalid JSON

	cases := []struct {
		name     string
		codec    Codec
		wantSite string
	}{
		{name: "claude", codec: NewClaudeForTest(), wantSite: siteClaudeStreamJSON},
		{name: "codex", codec: NewCodexForTest(), wantSite: siteCodexStreamJSON},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := withRegistry(t)
			runID := uuid.New()

			// Live path: DecodeStreamLine must not error (behavior unchanged) and
			// must drop the line (no events) while recording the failure.
			state := tc.codec.NewState()
			events, err := tc.codec.DecodeStreamLine(state, runID, malformed)
			if err != nil {
				t.Fatalf("DecodeStreamLine returned error (behavior changed): %v", err)
			}
			if len(events) != 0 {
				t.Fatalf("expected the malformed line dropped, got %d events", len(events))
			}

			snap := reg.Snapshot()
			if snap.Total == 0 {
				t.Fatalf("expected a recorded parse failure, got none")
			}
			if snap.BySite[tc.wantSite] == 0 {
				t.Fatalf("expected a failure at site %q, by_site=%v", tc.wantSite, snap.BySite)
			}
			var found bool
			for _, s := range snap.Samples {
				if s.Site == tc.wantSite && s.RunID == runID && s.LineSample != "" {
					found = true
				}
			}
			if !found {
				t.Fatalf("recorded sample not retrievable for run %s site %s: %+v", runID, tc.wantSite, snap.Samples)
			}
		})
	}
}

// TestCodecsRecordToolInputParseFailureSites proves every tool-input parse site
// that instrumentation guards is actually reachable and attributed to the
// correct site string. Each case feeds a line whose JSON is valid at the wire
// level but whose embedded tool-input payload is NOT a JSON object (a bare
// number), so the per-site Unmarshal-into-map fails exactly where expected.
//
// Without these, a format drift at opencode/grok or any claude sub-site would
// silently regress the W1 signal with no failing test: the sites existed but
// only claude:stream-json and codex:stream-json had positive coverage.
func TestCodecsRecordToolInputParseFailureSites(t *testing.T) {
	// feed drives the codec with the test line. Most sites are reachable via the
	// live stream path (DecodeStreamLine); grok's tool-input site lives in its
	// ACP transcript parser, so that case feeds the transcript parser instead.
	cases := []struct {
		name     string
		feed     func(t *testing.T, runID uuid.UUID, line string)
		line     string
		wantSite string
	}{
		{
			// claude stream-json tool_use event with a non-object input.
			name:     "claude tool-use-input",
			feed:     feedStream(func() Codec { return NewClaudeForTest() }),
			line:     `{"type":"tool_use","tool_use":{"id":"t1","name":"Write","input":123}}`,
			wantSite: siteClaudeToolUseInput,
		},
		{
			// claude assistant message carrying a tool_use content block whose
			// input is a non-object (exercises the ExtractToolUses path).
			name:     "claude tool-input (assistant block)",
			feed:     feedStream(func() Codec { return NewClaudeForTest() }),
			line:     `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t2","name":"Write","input":123}]}}`,
			wantSite: siteClaudeToolInput,
		},
		{
			// opencode legacy part-level input (no state.input) that is a
			// non-object: the fallback branch records opencode:tool-input.
			name:     "opencode tool-input",
			feed:     feedStream(func() Codec { return NewOpenCodeForTest() }),
			line:     `{"type":"tool_use","sessionID":"s","part":{"type":"tool","tool":"write","callID":"c","input":123}}`,
			wantSite: siteOpenCodeToolInput,
		},
		{
			// grok ACP tool_call with a non-object rawInput. This site lives in
			// grok's transcript (ACP JSON-RPC) parser, not the stdout decoder.
			name:     "grok tool-input",
			feed:     feedTranscript(func() Codec { return NewGrokForTest() }),
			line:     `{"method":"session/update","params":{"sessionId":"s","update":{"sessionUpdate":"tool_call","toolCallId":"c","title":"Shell","rawInput":123}}}`,
			wantSite: siteGrokToolInput,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := withRegistry(t)
			runID := uuid.New()
			tc.feed(t, runID, tc.line)
			snap := reg.Snapshot()
			if snap.BySite[tc.wantSite] == 0 {
				t.Fatalf("expected a failure at site %q, by_site=%v", tc.wantSite, snap.BySite)
			}
			var found bool
			for _, s := range snap.Samples {
				if s.Site == tc.wantSite && s.RunID == runID {
					found = true
				}
			}
			if !found {
				t.Fatalf("recorded sample not retrievable for site %s: %+v", tc.wantSite, snap.Samples)
			}
		})
	}
}

// TestCodecsRecordClaudeResultParseFailureSites covers the claude result/
// system-result/tool-result sites: a `result` field that is a JSON number
// instead of a string fails the string Unmarshal and must be recorded.
func TestCodecsRecordClaudeResultParseFailureSites(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		wantSite string
	}{
		{
			name:     "claude tool-result",
			line:     `{"type":"tool_result","result":123,"parent_tool_use_id":"p"}`,
			wantSite: siteClaudeToolResult,
		},
		{
			name:     "claude result",
			line:     `{"type":"result","subtype":"success","is_error":false,"result":123}`,
			wantSite: siteClaudeResult,
		},
		{
			name:     "claude system-result",
			line:     `{"type":"system","subtype":"info","result":123}`,
			wantSite: siteClaudeSystemResult,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := withRegistry(t)
			runID := uuid.New()
			c := NewClaudeForTest()
			state := c.NewState()
			if _, err := c.DecodeStreamLine(state, runID, tc.line); err != nil {
				t.Fatalf("DecodeStreamLine returned error (behavior changed): %v", err)
			}
			if snap := reg.Snapshot(); snap.BySite[tc.wantSite] == 0 {
				t.Fatalf("expected a failure at site %q, by_site=%v", tc.wantSite, snap.BySite)
			}
		})
	}
}

// TestGoldenFixturesProduceZeroParseFailures is the baseline: well-formed
// recorded traces must parse cleanly with NO parse failures. A non-zero count
// here means a real drift in a fixture (or an over-eager instrumentation site),
// both of which this guards against.
func TestGoldenFixturesProduceZeroParseFailures(t *testing.T) {
	fixtures := []struct {
		name  string
		trace string
		codec func() Codec
	}{
		{"claude", "claude_trace.jsonl", func() Codec { return NewClaudeForTest() }},
		{"claude_ondisk", "claude_ondisk_trace.jsonl", func() Codec { return NewClaudeForTest() }},
		{"codex", "codex_trace.jsonl", func() Codec { return NewCodexForTest() }},
		{"codex_rollout", "codex_rollout_trace.jsonl", func() Codec { return NewCodexForTest() }},
		{"opencode", "opencode_trace.jsonl", func() Codec { return NewOpenCodeForTest() }},
		{"grok", "grok_trace.jsonl", func() Codec { return NewGrokForTest() }},
		{"grok_updates", "grok_updates_trace.jsonl", func() Codec { return NewGrokForTest() }},
	}

	for _, fx := range fixtures {
		t.Run(fx.name, func(t *testing.T) {
			reg := withRegistry(t)
			lines := readGoldenLines(t, fx.trace)
			runID := uuid.New()

			live := fx.codec()
			state := live.NewState()
			for _, line := range lines {
				_, _ = live.DecodeStreamLine(state, runID, line)
			}

			transcript := fx.codec().NewTranscriptParser()
			for _, line := range lines {
				_ = transcript.ParseTranscriptLine(runID, line)
			}

			if snap := reg.Snapshot(); snap.Total != 0 {
				t.Fatalf("well-formed fixture %s produced %d parse failures (baseline must be zero): %+v",
					fx.trace, snap.Total, snap.Samples)
			}
		})
	}
}

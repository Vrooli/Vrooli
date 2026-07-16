// Golden-trace tests per codec.
//
// Each codec's testdata/<name>_trace.jsonl is a recorded representative
// stream from a real session. The test replays every line through both
// the live decoder (DecodeStreamLine) and the durable-transcript decoder
// (ParseTranscriptLine) and asserts that the resulting event sequence
// has the expected shape. The point is to catch schema drift loudly —
// when a codec bumps its JSON shape and the decoder silently starts
// dropping or mis-classifying events, this is the test that fails.
//
// Asserts in this file are intentionally structural: the sequence of
// event types per line, plus a small set of payload-field checks at
// the load-bearing positions. Bit-for-bit equality would calcify
// internal representations the rest of the test suite already covers.

package codecs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// golden is the contract for a codec under test.
type golden struct {
	name           string
	traceFile      string
	newCodec       func(t *testing.T) Codec
	expectedTypes  [][]domain.RunEventType // events[i] expected types (live path)
	requireMessage bool                    // at least one Message event total
	requireTool    bool                    // at least one ToolCall+ToolResult pair
}

func TestCodecGoldenTrace(t *testing.T) {
	cases := []golden{
		{
			name:      "claude",
			traceFile: "claude_trace.jsonl",
			newCodec:  func(t *testing.T) Codec { t.Helper(); return NewClaudeForTest() },
			// L0 system.init             → log
			// L1 assistant text          → message
			// L2 assistant tool_use      → tool_call
			// L3 user tool_result        → tool_result
			// L4 result.success          → terminal message + metric (cost/usage rollup)
			expectedTypes: [][]domain.RunEventType{
				{domain.EventTypeLog},
				{domain.EventTypeMessage},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolResult},
				{domain.EventTypeMessage, domain.EventTypeMetric},
			},
			requireMessage: true,
			requireTool:    true,
		},
		{
			name:      "codex",
			traceFile: "codex_trace.jsonl",
			newCodec:  func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() },
			// L0 thread.started                 → log
			// L1 turn.started                   → log
			// L2 reasoning                      → log
			// L3 command_execution started      → tool_call
			// L4 command_execution completed    → tool_result
			// L5 file_change                    → tool_call
			// L6 agent_message                  → message
			// L7 turn.completed                 → terminal evidence + metric
			expectedTypes: [][]domain.RunEventType{
				{domain.EventTypeLog},
				{domain.EventTypeLog},
				{domain.EventTypeLog},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolResult},
				{domain.EventTypeToolCall},
				{domain.EventTypeMessage},
				{domain.EventTypeMessage, domain.EventTypeMetric},
			},
			requireMessage: true,
			requireTool:    true,
		},
		{
			name:      "opencode",
			traceFile: "opencode_trace.jsonl",
			newCodec:  func(t *testing.T) Codec { t.Helper(); return NewOpenCodeForTest() },
			// L0 step_start                    → log
			// L1 text                          → message
			// L2 tool_use pending              → tool_call
			// L3 tool_use completed            → tool_call + tool_result
			// L4 step_finish stop              → metric + message
			expectedTypes: [][]domain.RunEventType{
				{domain.EventTypeLog},
				{domain.EventTypeMessage},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolCall, domain.EventTypeToolResult},
				{domain.EventTypeMetric, domain.EventTypeMessage},
			},
			requireMessage: true,
			requireTool:    true,
		},
	}

	for _, gc := range cases {
		gc := gc
		t.Run(gc.name, func(t *testing.T) {
			lines := readGoldenLines(t, gc.traceFile)
			if len(lines) == 0 {
				t.Fatalf("trace file %s is empty", gc.traceFile)
			}

			// Live decode path: single state, walk every line.
			liveCodec := gc.newCodec(t)
			liveState := liveCodec.NewState()
			runID := uuid.New()
			liveByLine := make([][]domain.RunEventType, len(lines))
			liveAll := []domain.RunEventType{}
			for i, line := range lines {
				events, err := liveCodec.DecodeStreamLine(liveState, runID, line)
				if err != nil {
					t.Fatalf("line %d DecodeStreamLine: %v\nline=%q", i, err, truncate(line))
				}
				types := make([]domain.RunEventType, 0, len(events))
				for _, e := range events {
					if e == nil {
						t.Fatalf("line %d emitted nil event", i)
					}
					types = append(types, e.EventType)
					liveAll = append(liveAll, e.EventType)
				}
				liveByLine[i] = types
			}

			// Per-line expected-type assertions (only when supplied).
			for i, want := range gc.expectedTypes {
				if want == nil {
					continue
				}
				got := liveByLine[i]
				if len(got) != len(want) {
					t.Errorf("line %d: live-path event count = %d (%v), want %d (%v)",
						i, len(got), got, len(want), want)
					continue
				}
				for j, wt := range want {
					if got[j] != wt {
						t.Errorf("line %d event %d: got %s, want %s",
							i, j, got[j], wt)
					}
				}
			}

			// Aggregate invariants.
			if gc.requireMessage && !containsType(liveAll, domain.EventTypeMessage) {
				t.Errorf("trace produced no Message event (live path); aggregate=%v", liveAll)
			}
			if gc.requireTool {
				if !containsType(liveAll, domain.EventTypeToolCall) {
					t.Errorf("trace produced no ToolCall event (live path); aggregate=%v", liveAll)
				}
				if !containsType(liveAll, domain.EventTypeToolResult) {
					t.Errorf("trace produced no ToolResult event (live path); aggregate=%v", liveAll)
				}
			}

			// Transcript path: ParseTranscriptLine over the same lines.
			// A codec's live and transcript decoders share line shape for
			// the runners we ship today, so transcript output should be a
			// permutation/subset of live output (Terminal events on the
			// final result line may shift between paths). The minimum
			// invariant we pin is "no panics, no errors."
			transcriptCodec := gc.newCodec(t)
			transcriptParser := transcriptCodec.NewTranscriptParser()
			transcriptAll := []domain.RunEventType{}
			for i, line := range lines {
				res := transcriptParser.ParseTranscriptLine(runID, line)
				if res.Err != nil {
					t.Errorf("line %d ParseTranscriptLine: %v\nline=%q",
						i, res.Err, truncate(line))
					continue
				}
				for _, e := range res.Events {
					if e == nil {
						t.Errorf("line %d transcript emitted nil event", i)
						continue
					}
					transcriptAll = append(transcriptAll, e.EventType)
				}
			}

			// Transcript replay should produce *at least as many* of each
			// user-meaningful event type as the live path: the transcript
			// decoder reads the same line shapes plus has access to final
			// state, so it can legitimately emit additional terminal
			// events the live path skipped. The drift signal we want to
			// catch is the inverse — a transcript path that *drops*
			// messages or tool events the live path saw, indicating a
			// schema bump landed in only one decoder.
			liveHist := histogram(liveAll)
			transcriptHist := histogram(transcriptAll)
			for typ, count := range liveHist {
				if typ != domain.EventTypeMessage &&
					typ != domain.EventTypeToolCall &&
					typ != domain.EventTypeToolResult {
					continue
				}
				if transcriptHist[typ] < count {
					t.Errorf("event-type %s: live=%d but transcript=%d — transcript decoder dropped events (schema drift?)",
						typ, count, transcriptHist[typ])
				}
			}
		})
	}
}

// onDiskGolden is the contract for a codec's on-disk (interactive) transcript
// dialect, which differs from the stdout stream dialect the stdout golden above
// exercises. Each testdata/<name>_trace.jsonl is a real, sanitized slice of the
// file the interactive CLI writes to its own session home (design §3): claude's
// ~/.claude/projects/*/<id>.jsonl, codex's rollout-*.jsonl, grok's
// updates.jsonl. The replay proves the transcript parser extracts the same
// structured events pipe runs produce, captures the session id, and synthesizes
// the terminal marker Phase 4 consumes.
type onDiskGolden struct {
	name                string
	traceFile           string
	newCodec            func(t *testing.T) Codec
	expectedTypes       [][]domain.RunEventType // per-line event types (transcript path)
	wantSessionID       bool
	wantTerminalSuccess bool
	// forbidLog asserts the replay emits zero Log events — proving claude's
	// interactive-only record types (mode/permission-mode/ai-title/…) are
	// dropped rather than surfaced as "unhandled type" debug noise.
	forbidLog bool
}

func TestCodecOnDiskTranscriptTrace(t *testing.T) {
	cases := []onDiskGolden{
		{
			name:      "claude",
			traceFile: "claude_ondisk_trace.jsonl",
			newCodec:  func(t *testing.T) Codec { t.Helper(); return NewClaudeForTest() },
			// L0 mode                     → (dropped, interactive-only)
			// L1 permission-mode          → (dropped)
			// L2 ai-title                 → (dropped)
			// L3 file-history-snapshot    → (dropped)
			// L4 assistant tool_use       → tool_call
			// L5 user tool_result         → tool_result
			// L6 assistant end_turn text  → message (+ terminal)
			expectedTypes: [][]domain.RunEventType{
				{},
				{},
				{},
				{},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolResult},
				{domain.EventTypeMessage},
			},
			wantSessionID:       true,
			wantTerminalSuccess: true,
			forbidLog:           true,
		},
		{
			name:      "codex",
			traceFile: "codex_rollout_trace.jsonl",
			newCodec:  func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() },
			// L0 session_meta               → (session id)
			// L1 user_message               → (suppressed)
			// L2 function_call              → tool_call
			// L3 function_call_output       → tool_result
			// L4 custom_tool_call           → tool_call
			// L5 custom_tool_call_output    → tool_result
			// L6 token_count                → (dropped)
			// L7 agent_message              → message
			// L8 task_complete              → terminal evidence
			expectedTypes: [][]domain.RunEventType{
				{},
				{},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolResult},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolResult},
				{},
				{domain.EventTypeMessage},
				{domain.EventTypeMessage},
			},
			wantSessionID:       true,
			wantTerminalSuccess: true,
		},
		{
			name:      "grok",
			traceFile: "grok_updates_trace.jsonl",
			newCodec:  func(t *testing.T) Codec { t.Helper(); return NewGrokForTest() },
			// L0 user_message_chunk         → (suppressed)
			// L1 agent_thought_chunk        → (reasoning, dropped)
			// L2 tool_call                  → tool_call
			// L3 tool_call_update completed → tool_result
			// L4 agent_message_chunk        → (accumulated, not yet emitted)
			// L5 turn_completed             → message (flushed) + terminal
			expectedTypes: [][]domain.RunEventType{
				{},
				{},
				{domain.EventTypeToolCall},
				{domain.EventTypeToolResult},
				{},
				{domain.EventTypeMessage},
			},
			wantSessionID:       true,
			wantTerminalSuccess: true,
		},
	}

	for _, gc := range cases {
		gc := gc
		t.Run(gc.name, func(t *testing.T) {
			lines := readGoldenLines(t, gc.traceFile)
			if len(lines) == 0 {
				t.Fatalf("trace file %s is empty", gc.traceFile)
			}

			parser := gc.newCodec(t).NewTranscriptParser()
			runID := uuid.New()
			var sessionID string
			var terminal *runner.TranscriptTerminal
			gotLog := false

			for i, line := range lines {
				res := parser.ParseTranscriptLine(runID, line)
				if res.Err != nil {
					t.Fatalf("line %d ParseTranscriptLine: %v\nline=%q", i, res.Err, truncate(line))
				}
				if res.SessionID != "" {
					sessionID = res.SessionID
				}
				if res.Terminal != nil {
					terminal = res.Terminal
				}
				types := make([]domain.RunEventType, 0, len(res.Events))
				for _, e := range res.Events {
					if e == nil {
						t.Fatalf("line %d emitted nil event", i)
					}
					types = append(types, e.EventType)
					if e.EventType == domain.EventTypeLog {
						gotLog = true
					}
				}
				if i < len(gc.expectedTypes) {
					want := gc.expectedTypes[i]
					if len(types) != len(want) {
						t.Errorf("line %d: event types = %v, want %v", i, types, want)
						continue
					}
					for j, wt := range want {
						if types[j] != wt {
							t.Errorf("line %d event %d: got %s, want %s", i, j, types[j], wt)
						}
					}
				}
			}

			if gc.wantSessionID && sessionID == "" {
				t.Errorf("no session id captured from on-disk transcript")
			}
			if gc.wantTerminalSuccess {
				if terminal == nil {
					t.Errorf("no terminal marker synthesized from on-disk transcript")
				} else if !terminal.Success {
					t.Errorf("terminal = failure (%q), want success", terminal.ErrorMessage)
				}
			}
			if gc.forbidLog && gotLog {
				t.Errorf("on-disk replay emitted Log events — interactive-only records should be dropped silently, not logged as debug/unhandled noise")
			}
		})
	}
}

func readGoldenLines(t *testing.T, name string) []string {
	t.Helper()
	path := filepath.Join("testdata", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	out := []string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return out
}

func containsType(types []domain.RunEventType, want domain.RunEventType) bool {
	for _, t := range types {
		if t == want {
			return true
		}
	}
	return false
}

func histogram(types []domain.RunEventType) map[domain.RunEventType]int {
	out := map[domain.RunEventType]int{}
	for _, t := range types {
		out[t]++
	}
	return out
}

func truncate(s string) string {
	const max = 200
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

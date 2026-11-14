package codecs

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// These retained fixtures are trimmed, redacted real transcripts (provenance in
// testdata/README.md). Each test feeds one through the capture shapes that
// inflated totals before (E-5): a fresh parse, a fresh parser per turn starting
// at a mid-file cursor (interactive Tail), a recovery replay of the last turn,
// and a full re-tail after a restart. The run's usage account must equal the
// transcript's own final counter every time.

func fixtureLines(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

// parseSegment runs one fresh parser over lines, as every capture path does.
func parseSegment(parser runner.TranscriptParser, lines []string) []*domain.RunEvent {
	var events []*domain.RunEvent
	for _, line := range lines {
		events = append(events, parser.ParseTranscriptLine(uuid.Nil, line).Events...)
	}
	return events
}

// segmentsAt splits lines into turns that each begin at a line marking a turn.
func segmentsAt(lines []string, starts func(string) bool) [][]string {
	var out [][]string
	begin := 0
	for index := 1; index < len(lines); index++ {
		if starts(lines[index]) {
			out = append(out, lines[begin:index])
			begin = index
		}
	}
	return append(out, lines[begin:])
}

func accountTotal(events []*domain.RunEvent) int64 {
	return runsignal.AccountUsage(events).TotalTokens()
}

type captureCase struct {
	name   string
	events []*domain.RunEvent
}

// captureShapes builds the E-5 capture shapes from per-turn segments. fresh
// returns a parser that reads from the transcript start; resumed returns one
// that starts at a mid-file cursor of a known session.
func captureShapes(segments [][]string, fresh, resumed func() runner.TranscriptParser) []captureCase {
	var all []string
	for _, segment := range segments {
		all = append(all, segment...)
	}
	whole := parseSegment(fresh(), all)
	var perTurn []*domain.RunEvent
	for index, segment := range segments {
		parser := resumed()
		if index == 0 {
			parser = fresh()
		}
		perTurn = append(perTurn, parseSegment(parser, segment)...)
	}
	replayLast := append(append([]*domain.RunEvent{}, perTurn...), parseSegment(resumed(), segments[len(segments)-1])...)
	restart := append(append([]*domain.RunEvent{}, replayLast...), parseSegment(fresh(), all)...)
	return []captureCase{
		{"fresh parse", whole},
		{"fresh parser per turn from mid-file cursors", perTurn},
		{"recovery replay of the last turn", replayLast},
		{"full re-tail after restart", restart},
	}
}

func TestCodexRolloutUsageMatchesFinalCounterUnderEveryCaptureShape(t *testing.T) {
	lines := fixtureLines(t, "codex_usage_rollout.jsonl")
	var session string
	var final int64
	for _, line := range lines {
		var record struct {
			Type    string `json:"type"`
			Payload struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Info *struct {
					Total struct {
						Total int64 `json:"total_tokens"`
					} `json:"total_token_usage"`
				} `json:"info"`
			} `json:"payload"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record.Type == "session_meta" {
			session = record.Payload.ID
		}
		if record.Payload.Type == "token_count" && record.Payload.Info != nil {
			final = record.Payload.Info.Total.Total
		}
	}
	if session == "" || final == 0 {
		t.Fatal("fixture lacks a session or a final counter")
	}
	codec := NewCodexForTest()
	fresh := func() runner.TranscriptParser { return codec.NewTranscriptParser() }
	resumed := func() runner.TranscriptParser {
		parser := codec.NewTranscriptParser()
		parser.(runner.TranscriptSessionSetter).SetTranscriptSession(session, false)
		return parser
	}
	segments := segmentsAt(lines, func(line string) bool { return strings.Contains(line, `"type":"task_started"`) })
	if len(segments) < 3 {
		t.Fatalf("fixture has %d turns, want several", len(segments))
	}
	for _, shape := range captureShapes(segments, fresh, resumed) {
		if got := accountTotal(shape.events); got != final {
			t.Errorf("%s: account total = %d, want the rollout's final counter %d", shape.name, got, final)
		}
	}
}

// The exec stream reports only the thread counter per completed turn. Its
// first invocations were stopped before turn.completed, so the stream starts
// with a counter that already covers their work: that part is recovered from
// the rollout by RecoverInterruptedUsage, never from the stream. What the
// stream does report must count exactly once under replay.
func TestCodexExecUsageCountsTheStreamOnceUnderReplay(t *testing.T) {
	lines := fixtureLines(t, "codex_usage_exec.jsonl")
	var counters []domain.UsageCounters
	session := ""
	for _, line := range lines {
		var record struct {
			Type     string      `json:"type"`
			ThreadID string      `json:"thread_id"`
			Usage    *CodexUsage `json:"usage"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record.ThreadID != "" {
			session = record.ThreadID
		}
		if record.Usage != nil {
			counters = append(counters, codexCounters(record.Usage))
		}
	}
	want := counters[len(counters)-1].Total() - counters[0].Total()
	codec := NewCodexForTest()
	// Live: one stream state per invocation (each begins at thread.started),
	// the first a fresh `codex exec`, the rest `codex exec resume`, each
	// preceded by the owner's running boundary.
	invocations := segmentsAt(lines, func(line string) bool { return strings.Contains(line, `"thread.started"`) })
	var live []*domain.RunEvent
	for index, invocation := range invocations {
		state := codec.NewState()
		if index == 0 {
			codec.BuildArgs(state, runner.ExecuteRequest{})
		} else {
			codec.BuildContinueArgs(state, runner.ContinueRequest{SessionID: session})
		}
		live = append(live, domain.NewStatusEvent(uuid.Nil, "starting", string(domain.RunStatusRunning), "invocation"))
		for _, line := range invocation {
			events, err := codec.DecodeStreamLine(state, uuid.Nil, line)
			if err != nil {
				t.Fatal(err)
			}
			live = append(live, events...)
		}
	}
	if got := accountTotal(live); got != want {
		t.Fatalf("live account = %d, want %d", got, want)
	}
	// Recovery after a restart drains the last invocation again with a fresh
	// parser at a mid-file cursor (the 2x of run aee95b01).
	drain := codec.NewTranscriptParser()
	drain.(runner.TranscriptSessionSetter).SetTranscriptSession(session, false)
	replayed := append(append([]*domain.RunEvent{}, live...), parseSegment(drain, invocations[len(invocations)-1])...)
	if got := accountTotal(replayed); got != want {
		t.Fatalf("recovery replay account = %d, want %d", got, want)
	}
}

// claudeUniqueMessageUsage is the independent expectation: Claude repeats a
// message's usage on each streamed content block, sometimes with partial
// output on early blocks, so a message's usage is its largest report.
func claudeUniqueMessageUsage(t *testing.T, lines []string) (int64, int) {
	t.Helper()
	largest := map[string]int64{}
	reports := 0
	for _, line := range lines {
		var record struct {
			Type    string `json:"type"`
			Message struct {
				ID    string `json:"id"`
				Usage *struct {
					Input         int64 `json:"input_tokens"`
					CacheCreation int64 `json:"cache_creation_input_tokens"`
					CacheRead     int64 `json:"cache_read_input_tokens"`
					Output        int64 `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record.Type != "assistant" || record.Message.Usage == nil {
			continue
		}
		reports++
		u := record.Message.Usage
		largest[record.Message.ID] = max(largest[record.Message.ID], u.Input+u.CacheCreation+u.CacheRead+u.Output)
	}
	var total int64
	for _, value := range largest {
		total += value
	}
	return total, reports - len(largest)
}

func TestClaudeOnDiskUsageCountsEachMessageOnceUnderEveryCaptureShape(t *testing.T) {
	for _, fixture := range []string{"claude_usage_ondisk.jsonl", "claude_usage_subagent.jsonl"} {
		t.Run(fixture, func(t *testing.T) {
			lines := fixtureLines(t, fixture)
			want, repeats := claudeUniqueMessageUsage(t, lines)
			if repeats == 0 {
				t.Fatal("fixture has no repeated message usage")
			}
			codec := NewClaudeForTest()
			parser := func() runner.TranscriptParser { return codec.NewTranscriptParser() }
			segments := segmentsAt(lines, func(line string) bool {
				return strings.Contains(line, `"message":{"role":"user","content":"[redacted]"}`)
			})
			for _, shape := range captureShapes(segments, parser, parser) {
				if got := accountTotal(shape.events); got != want {
					t.Errorf("%s: account total = %d, want summed unique message usage %d", shape.name, got, want)
				}
			}
		})
	}
}

// A sub-agent's streamed usage names the parent tool call that spawned it,
// so its consumption stays attributable after the account counts it.
func TestClaudeStreamSubagentUsageKeepsItsParentToolCall(t *testing.T) {
	codec := NewClaudeForTest()
	state := codec.NewState()
	line := `{"type":"assistant","parent_tool_use_id":"toolu_parent","session_id":"s","message":{"id":"msg_child","role":"assistant","content":[{"type":"text","text":"[redacted]"}],"usage":{"input_tokens":3,"output_tokens":2}}}`
	events, err := codec.DecodeStreamLine(state, uuid.Nil, line)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if usage, ok := event.Data.(*domain.UsageEventData); ok {
			if usage.MessageID != "msg_child" || usage.ParentToolUseID != "toolu_parent" {
				t.Fatalf("sub-agent usage lost its identity: %+v", usage)
			}
			return
		}
	}
	t.Fatal("sub-agent message emitted no usage")
}

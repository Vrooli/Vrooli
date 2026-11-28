package agenttranscript

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestContainsMessageMarkerAcrossLargeNativeTranscriptCorpus(t *testing.T) {
	const marker = "WC2-CONSUMED-7f30c51e"
	for _, harness := range []string{HarnessClaude, HarnessCodex, HarnessOpenCode, HarnessGrok} {
		lines := make([][]byte, 5005)
		for i := range lines {
			text := fmt.Sprintf("corpus message %d", i+1)
			role := "assistant"
			if i == len(lines)-2 {
				role = "user"
				text = "delivered handoff marker: " + marker
			} else if i == len(lines)-1 {
				text = "target consumed this exact marker: " + marker
			}
			switch harness {
			case HarnessClaude:
				lines[i] = []byte(fmt.Sprintf(`{"type":%q,"message":{"content":[{"type":"text","text":%q}]}}`, role, text))
			case HarnessCodex:
				kind := "agent_message"
				wire := "response_item"
				if role == "user" {
					kind, wire = "user_message", "event_msg"
				}
				lines[i] = []byte(fmt.Sprintf(`{"type":%q,"payload":{"type":%q,"text":%q}}`, wire, kind, text))
			case HarnessOpenCode:
				lines[i] = []byte(fmt.Sprintf(`{"info":{"role":%q},"parts":[{"type":"text","text":%q}]}`, role, text))
			case HarnessGrok:
				kind := "agent_message_chunk"
				if role == "user" {
					kind = "user_message_chunk"
				}
				lines[i] = []byte(fmt.Sprintf(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":%q,"content":{"type":"text","text":%q}}}}`, kind, text))
			}
		}
		if got := InspectHandoffTranscript(harness, marker, lines); !got.Delivered || !got.Consumed {
			t.Errorf("%s target transcript receipt = %+v, want delivered and consumed over 5,005 events", harness, got)
		}
		lines[len(lines)-1] = []byte(`{"type":"progress","message":"` + marker + `"}`)
		if got := InspectHandoffTranscript(harness, marker, lines); !got.Delivered || got.Consumed {
			t.Errorf("%s non-conversational marker must not prove consumption: %+v", harness, got)
		}
	}
	if ContainsMessageMarker(HarnessClaude, "  ", nil) {
		t.Fatal("empty receipt marker must never prove consumption")
	}
}

// This is an offline parser/format corpus, not a live handoff receipt. It
// exercises every source/target dialect pair while keeping the selected
// content exact, including a tool call that carries the clue needed by the
// target's answer. Live consumption still requires observing the target's own
// transcript after a real, authorized handoff.
func TestHandoffNativeDialectSourceTargetMatrix(t *testing.T) {
	const marker = "WC2-MATRIX-9b1740"
	const clue = "opportunity-code: amber-otter-731"
	sources := map[string][][]byte{
		HarnessClaude: {
			[]byte(`{"type":"user","message":{"content":[{"type":"text","text":"Selected task context: ` + clue + `"}]}}`),
			[]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"matrix-claude","name":"Read","input":{"path":"` + clue + `"}}]}}`),
		},
		HarnessCodex: {
			[]byte(`{"type":"event_msg","payload":{"type":"user_message","text":"Selected task context: ` + clue + `"}}`),
			[]byte(`{"type":"response_item","payload":{"type":"function_call","call_id":"matrix-codex","name":"read","arguments":"{\"path\":\"` + clue + `\"}"}}`),
		},
		HarnessOpenCode: {
			[]byte(`{"info":{"role":"user"},"parts":[{"type":"text","text":"Selected task context: ` + clue + `"}]}`),
			[]byte(`{"info":{"role":"assistant"},"parts":[{"type":"tool","callID":"matrix-opencode","tool":"read","input":{"path":"` + clue + `"}}]}`),
		},
		HarnessGrok: {
			[]byte(`{"method":"session/update","params":{"sessionId":"source","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"Selected task context: ` + clue + `"}}}}`),
			[]byte(`{"method":"session/update","params":{"sessionId":"source","update":{"sessionUpdate":"tool_call","toolCallId":"matrix-grok","title":"Read","rawInput":{"path":"` + clue + `"}}}}`),
		},
	}
	harnesses := []string{HarnessClaude, HarnessCodex, HarnessOpenCode, HarnessGrok}
	for _, source := range harnesses {
		var selected []string
		for _, line := range sources[source] {
			for _, message := range ParseMessages(source, line) {
				selected = append(selected, message.Text)
			}
		}
		payload := strings.Join(selected, "\n")
		if !strings.Contains(payload, clue) || !strings.Contains(payload, "Tool call") {
			t.Fatalf("%s selected payload lost source text or tool call: %q", source, payload)
		}
		for _, target := range harnesses {
			t.Run(source+"-to-"+target, func(t *testing.T) {
				delivered := "Selected context follows. `" + marker + "`\n" + payload
				answer := "The opportunity-code is amber-otter-731. `" + marker + "`"
				lines := [][]byte{nativeTextLine(target, "user", delivered), nativeTextLine(target, "assistant", answer)}
				if got := InspectHandoffTranscript(target, marker, lines); !got.Delivered || !got.Consumed {
					t.Fatalf("offline %s target-format corpus receipt = %+v", target, got)
				}
				if got := InspectHandoffTranscript(target, marker, [][]byte{lines[1], lines[0]}); !got.Delivered || got.Consumed {
					t.Fatalf("%s assistant marker before delivery must not count as consumption: %+v", target, got)
				}
				messages := ParseMessages(target, lines[0])
				if len(messages) != 1 || !strings.Contains(messages[0].Text, clue) || !strings.Contains(messages[0].Text, "Tool call") {
					t.Fatalf("%s target user record did not preserve exact selected context: %#v", target, messages)
				}
			})
		}
	}
}

func nativeTextLine(harness, role, text string) []byte {
	switch harness {
	case HarnessClaude:
		return []byte(fmt.Sprintf(`{"type":%q,"message":{"content":[{"type":"text","text":%q}]}}`, role, text))
	case HarnessCodex:
		if role == "user" {
			return []byte(fmt.Sprintf(`{"type":"event_msg","payload":{"type":"user_message","text":%q}}`, text))
		}
		return []byte(fmt.Sprintf(`{"type":"response_item","payload":{"type":"agent_message","text":%q}}`, text))
	case HarnessOpenCode:
		return []byte(fmt.Sprintf(`{"info":{"role":%q},"parts":[{"type":"text","text":%q}]}`, role, text))
	case HarnessGrok:
		kind := "agent_message_chunk"
		if role == "user" {
			kind = "user_message_chunk"
		}
		return []byte(fmt.Sprintf(`{"method":"session/update","params":{"sessionId":"target","update":{"sessionUpdate":%q,"content":{"type":"text","text":%q}}}}`, kind, text))
	default:
		return nil
	}
}

func TestFormatsAreSharedAndStable(t *testing.T) {
	for _, harness := range []string{HarnessClaude, HarnessCodex, "grok", "opencode", "antigravity"} {
		if format, ok := FormatFor(harness); !ok || format.Harness != harness || format.Transcript == "" || format.MessageDialect == "" {
			t.Fatalf("FormatFor(%q) = %#v, %v", harness, format, ok)
		}
	}
	if _, ok := FormatFor("unlisted"); ok {
		t.Fatal("unknown harness unexpectedly has a format")
	}
	formats := Formats()
	for i := 1; i < len(formats); i++ {
		if formats[i-1].Harness >= formats[i].Harness {
			t.Fatalf("formats are not uniquely sorted: %#v", formats)
		}
	}
}

func TestDiscoveryRootsAreDataDefined(t *testing.T) {
	claude, _ := FormatFor("claude")
	codex, _ := FormatFor("codex")
	if len(claude.HomeRoots) != 1 || claude.HomeRoots[0] != ".claude/projects" || len(claude.RunRoots) != 2 {
		t.Fatalf("Claude discovery roots = %#v", claude)
	}
	if len(codex.HomeRoots) != 2 || len(codex.RunRoots) != 4 || len(codex.WebConsoleRoots) != 2 {
		t.Fatalf("Codex discovery roots = %#v", codex)
	}
}

func TestParseMessagesClaudeAndCodex(t *testing.T) {
	for _, tc := range []struct{ harness, line, role, text string }{
		{HarnessClaude, `{"type":"user","message":{"content":[{"type":"text","text":"question"}]}}`, "user", "question"},
		{HarnessClaude, `{"type":"assistant","message":{"content":"answer"}}`, "assistant", "answer"},
		{HarnessCodex, `{"type":"event_msg","payload":{"type":"user_message","text":"question"}}`, "user", "question"},
		{HarnessCodex, `{"type":"event_msg","payload":{"type":"user_message","message":"question"}}`, "user", "question"},
		{HarnessCodex, `{"type":"response_item","payload":{"role":"assistant","content":[{"type":"output_text","text":"answer"}]}}`, "assistant", "answer"},
		{HarnessCodex, `{"type":"response_item","item":{"type":"agent_message","text":"answer"}}`, "assistant", "answer"},
		{"opencode", `{"info":{"role":"assistant"},"parts":[{"type":"text","text":"answer"}]}`, "assistant", "answer"},
		{"antigravity", `{"text":"answer","session_id":"thread-1"}`, "assistant", "answer"},
	} {
		messages := ParseMessages(tc.harness, []byte(tc.line))
		if len(messages) != 1 || messages[0].Role != tc.role || messages[0].Text != tc.text {
			t.Errorf("ParseMessages(%s) = %#v", tc.harness, messages)
		}
	}
	got := ParseMessages("antigravity", []byte(`{"text":"answer","session_id":"thread-1"}`))
	if len(got) != 1 || got[0].SessionID != "thread-1" {
		t.Fatalf("antigravity session = %#v", got)
	}
	if got := ParseMessages(HarnessClaude, []byte(`{"type":"user","message":{"content":[{"type":"tool_result","content":"not user text"}]}}`)); len(got) != 0 {
		t.Fatalf("tool result became user message: %#v", got)
	}
}

func TestParseMessagesPreservesToolCallsForSpanSelection(t *testing.T) {
	cases := []struct{ harness, line, callID string }{
		{HarnessClaude, `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"claude-call","name":"Bash","input":{"command":"pwd"}}]}}`, "claude-call"},
		{HarnessCodex, `{"type":"response_item","payload":{"type":"function_call","call_id":"codex-call","name":"exec","arguments":"{\"cmd\":\"pwd\"}"}}`, "codex-call"},
		{HarnessOpenCode, `{"info":{"role":"assistant"},"parts":[{"type":"tool","callID":"oc-call","tool":"bash","input":{"command":"pwd"}}]}`, "oc-call"},
		{HarnessGrok, `{"method":"session/update","params":{"sessionId":"grok-session","update":{"sessionUpdate":"tool_call","toolCallId":"grok-call","title":"Shell","rawInput":{"command":"pwd"}}}}`, "grok-call"},
	}
	for _, tc := range cases {
		messages := ParseMessages(tc.harness, []byte(tc.line))
		if len(messages) != 1 || messages[0].Role != "assistant" || !strings.Contains(messages[0].Text, "Tool call") || !strings.Contains(messages[0].Text, tc.callID) || !strings.Contains(messages[0].Text, "pwd") {
			t.Errorf("%s tool call was not preserved for selection: %#v", tc.harness, messages)
		}
	}
}

func TestParseGrokACPUpdates(t *testing.T) {
	user := []byte(`{"method":"session/update","params":{"sessionId":"thread-7","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"question"}}}}`)
	assistant := []byte(`{"method":"session/update","params":{"sessionId":"thread-7","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"answer"}}}}`)
	completion := []byte(`{"method":"_x.ai/session/update","params":{"sessionId":"thread-7","update":{"sessionUpdate":"turn_completed","stop_reason":"end_turn"}}}`)
	for _, tc := range []struct {
		line       []byte
		role, text string
	}{{user, "user", "question"}, {assistant, "assistant", "answer"}} {
		messages := ParseMessages("grok", tc.line)
		if len(messages) != 1 || messages[0].Role != tc.role || messages[0].Text != tc.text || messages[0].SessionID != "thread-7" {
			t.Errorf("ParseMessages(grok) = %#v", messages)
		}
	}
	if messages := ParseMessages("grok", completion); len(messages) != 0 {
		t.Fatalf("completion became prose: %#v", messages)
	}
	update, ok := ParseGrokUpdate(completion)
	if !ok || update.Kind != "turn_completed" || update.StopReason != "end_turn" {
		t.Fatalf("ParseGrokUpdate(completion) = %#v, %v", update, ok)
	}
}

func TestOpenCodeArchiveMessagesCarryRecordOccurrenceTime(t *testing.T) {
	got := ParseMessages(HarnessOpenCode, []byte(`{"info":{"role":"assistant","time":{"created":1782095815108}},"parts":[{"type":"text","text":"first"},{"type":"tool","tool":"shell","callID":"c1","input":{"cmd":"pwd"}}]}`))
	want := time.UnixMilli(1782095815108).UTC()
	if len(got) != 2 || !got[0].OccurredAt.Equal(want) || !got[1].OccurredAt.Equal(want) {
		t.Fatalf("OpenCode record occurrence timestamps = %#v, want %s on each emitted part", got, want)
	}
	for _, raw := range []string{`{"info":{"role":"assistant","time":{"created":"bad"}},"parts":[{"type":"text","text":"x"}]}`, `{"info":{"role":"assistant"},"parts":[{"type":"text","text":"x"}]}`} {
		messages := ParseMessages(HarnessOpenCode, []byte(raw))
		if len(messages) != 1 || !messages[0].OccurredAt.IsZero() {
			t.Errorf("invalid or missing OpenCode time should remain unknown: %#v", messages)
		}
	}
}

func TestClaudeAndCodexMessagesCarryOuterRecordTime(t *testing.T) {
	want := time.Date(2026, 7, 22, 2, 36, 55, 108000000, time.UTC)
	cases := []struct {
		name, harness, line string
		wantCount           int
		wantAt              time.Time
	}{
		{"Claude text and tool parts", HarnessClaude, `{"type":"assistant","timestamp":"2026-07-22T02:36:55.108Z","message":{"content":[{"type":"text","text":"answer"},{"type":"tool_use","id":"call-1","name":"Read","input":{"path":"x"}}]}}`, 2, want},
		{"Codex content parts", HarnessCodex, `{"timestamp":"2026-07-22T02:36:55.108Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"one"},{"type":"output_text","text":"two"}]}}`, 2, want},
		{"Codex user message", HarnessCodex, `{"timestamp":1782095815,"type":"event_msg","payload":{"type":"user_message","message":"question"}}`, 1, time.Unix(1782095815, 0).UTC()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messages := ParseMessages(tc.harness, []byte(tc.line))
			if len(messages) != tc.wantCount {
				t.Fatalf("message count=%d want %d: %#v", len(messages), tc.wantCount, messages)
			}
			for _, message := range messages {
				if !message.OccurredAt.Equal(tc.wantAt) {
					t.Errorf("OccurredAt=%s want %s", message.OccurredAt, tc.wantAt)
				}
			}
		})
	}
	for _, raw := range []struct{ harness, line string }{
		{HarnessClaude, `{"type":"assistant","timestamp":"invalid","message":{"content":"answer"}}`},
		{HarnessCodex, `{"type":"event_msg","payload":{"type":"user_message","message":"question"}}`},
	} {
		messages := ParseMessages(raw.harness, []byte(raw.line))
		if len(messages) != 1 || !messages[0].OccurredAt.IsZero() {
			t.Errorf("missing/invalid outer time must remain unknown: %#v", messages)
		}
	}
}

func TestParseMessagesAgainstRealGrokReplayCorpus(t *testing.T) {
	file, err := os.Open("../../../scenarios/agent-manager/api/internal/adapters/runner/codecs/testdata/corpus/grok-updates.jsonl")
	if err != nil {
		t.Fatalf("open redacted provider-native Grok corpus: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var users, assistants int
	for scanner.Scan() {
		for _, message := range ParseMessages("grok", scanner.Bytes()) {
			switch message.Role {
			case "user":
				users++
			case "assistant":
				assistants++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan Grok corpus: %v", err)
	}
	if users == 0 || assistants == 0 {
		t.Fatalf("real Grok corpus yielded %d user and %d assistant messages", users, assistants)
	}
}

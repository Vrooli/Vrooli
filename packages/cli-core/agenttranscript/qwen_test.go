package agenttranscript

import (
	"strings"
	"testing"
	"time"
)

func TestParseQwenMessagesKeepsVisibleTextAndNativeTime(t *testing.T) {
	line := []byte(`{"type":"assistant","sessionId":"qwen-session","timestamp":"2026-11-23T16:00:00.123Z","message":{"role":"model","parts":[{"text":"visible answer"},{"text":"private thought","thought":true},{"text":"second part"}]}}`)
	got := ParseMessages("qwen-code", line)
	if len(got) != 1 || got[0].Role != "assistant" || got[0].Text != "visible answer\nsecond part" || got[0].SessionID != "qwen-session" {
		t.Fatalf("Qwen messages = %+v", got)
	}
	want := time.Date(2026, time.November, 23, 16, 0, 0, 123000000, time.UTC)
	if !got[0].OccurredAt.Equal(want) {
		t.Fatalf("OccurredAt = %s, want %s", got[0].OccurredAt, want)
	}
}

func TestParseQwenMessagesSkipsMetadataAndNonConversation(t *testing.T) {
	for _, line := range []string{
		`{"type":"system","subtype":"custom_title","systemPayload":{"customTitle":"title"}}`,
		`{"type":"tool_result","message":{"role":"function","parts":[{"text":"tool output"}]}}`,
		`{"type":"assistant","message":{"role":"model","parts":[{"text":"thought only","thought":true}]}}`,
		`{"type":"assistant","message":{"role":"user","parts":[{"text":"wrong role"}]}}`,
		`{broken`,
	} {
		if got := ParseMessages("qwen-code", []byte(line)); len(got) != 0 {
			t.Errorf("ParseMessages(%s) = %+v, want no selectable messages", line, got)
		}
	}
}

func TestParseQwenMessagesRetainsToolCallsAndResults(t *testing.T) {
	call := ParseMessages("qwen-code", []byte(`{"type":"assistant","sessionId":"qwen-session","timestamp":"2026-11-23T16:00:00Z","message":{"role":"model","parts":[{"functionCall":{"name":"read_file","id":"call-1","args":{"path":"main.go"}}}]}}`))
	if len(call) != 1 || call[0].Role != "assistant" || !strings.Contains(call[0].Text, "Tool call: read_file") || !strings.Contains(call[0].Text, "main.go") {
		t.Fatalf("Qwen tool call = %+v", call)
	}
	result := ParseMessages("qwen-code", []byte(`{"type":"tool_result","sessionId":"qwen-session","timestamp":"2026-11-23T16:00:01Z","message":{"role":"function","parts":[{"functionResponse":{"name":"read_file","id":"call-1","response":{"content":"source result"}}}]}}`))
	if len(result) != 1 || result[0].Role != "assistant" || !strings.Contains(result[0].Text, "Tool result: read_file") || !strings.Contains(result[0].Text, "source result") {
		t.Fatalf("Qwen tool result = %+v", result)
	}
}

func TestQwenNativeArchiveDescriptor(t *testing.T) {
	format, ok := FormatFor("qwen-code")
	if !ok || format.ResumeFlag != "--resume" || format.MessageDialect != "qwen-code-jsonl" {
		t.Fatalf("Qwen format = %+v, found=%v", format, ok)
	}
	path := "/home/example/.qwen/projects/-home-example-project/chats/session-id.jsonl"
	if !IsNativeArchiveFile("qwen-code", path) {
		t.Fatalf("Qwen chat transcript not recognized: %s", path)
	}
	archived := "/home/example/.qwen/projects/-home-example-project/chats/archive/old-session.jsonl"
	if !IsNativeArchiveFile("qwen-code", archived) {
		t.Fatalf("archived Qwen chat transcript not recognized: %s", archived)
	}
	if IsNativeArchiveFile("qwen-code", "/home/example/.qwen/projects/-home-example-project/session-id.jsonl") {
		t.Fatal("non-chats JSONL file was admitted as a Qwen transcript")
	}
}

func TestBuildQwenNativeResumeInvocation(t *testing.T) {
	prompt := `continue "the selected task"`
	got, err := BuildNativeResumeInvocation("qwen-code", "qwen-session-id", prompt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != "qwen" || len(got.Args) != 4 || got.Args[0] != "--resume" || got.Args[1] != "qwen-session-id" || got.Args[2] != "-p" || got.Args[3] != prompt {
		t.Fatalf("Qwen resume invocation = %+v", got)
	}
	if _, err := BuildNativeResumeInvocation("qwen-code", "", prompt); err == nil {
		t.Fatal("resume accepted an empty native session ID")
	}
	if _, err := BuildNativeResumeInvocation("claude", "session-id", prompt); err == nil {
		t.Fatal("resume built for a harness without an explicit shared binary contract")
	}
}

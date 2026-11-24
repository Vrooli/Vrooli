package agenttranscript

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAntigravityCLITranscriptCorpus(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		role   string
		text   string
		second int
	}{
		{
			name: "user input",
			line: `{"step_index":1,"type":"USER_INPUT","source":"USER_EXPLICIT","status":"DONE","created_at":"2026-11-24T10:00:00Z","content":"sanitized user fixture"}`,
			role: "user", text: "sanitized user fixture", second: 0,
		},
		{
			name: "planner response",
			line: `{"step_index":2,"type":"PLANNER_RESPONSE","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:01Z","content":"sanitized assistant fixture"}`,
			role: "assistant", text: "sanitized assistant fixture", second: 1,
		},
		{
			name: "tool result",
			line: `{"step_index":3,"type":"RUN_COMMAND","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:02Z","content":"sanitized tool output"}`,
			role: "assistant", text: "[Tool result: RUN_COMMAND]\n\"sanitized tool output\"", second: 2,
		},
		{
			name: "view file result",
			line: `{"step_index":4,"type":"VIEW_FILE","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:03Z","content":"sanitized view output"}`,
			role: "assistant", text: "[Tool result: VIEW_FILE]\n\"sanitized view output\"", second: 3,
		},
		{
			name: "grep search result",
			line: `{"step_index":5,"type":"GREP_SEARCH","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:04Z","content":"sanitized grep output"}`,
			role: "assistant", text: "[Tool result: GREP_SEARCH]\n\"sanitized grep output\"", second: 4,
		},
		{
			name: "web search result",
			line: `{"step_index":6,"type":"SEARCH_WEB","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:05Z","content":"sanitized search output"}`,
			role: "assistant", text: "[Tool result: SEARCH_WEB]\n\"sanitized search output\"", second: 5,
		},
		{
			name: "URL content result",
			line: `{"step_index":7,"type":"READ_URL_CONTENT","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:06Z","content":"sanitized URL output"}`,
			role: "assistant", text: "[Tool result: READ_URL_CONTENT]\n\"sanitized URL output\"", second: 6,
		},
		{
			name: "directory listing result",
			line: `{"step_index":8,"type":"LIST_DIRECTORY","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:07Z","content":"sanitized directory output"}`,
			role: "assistant", text: "[Tool result: LIST_DIRECTORY]\n\"sanitized directory output\"", second: 7,
		},
		{
			name: "question tool result",
			line: `{"step_index":9,"type":"ASK_QUESTION","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:08Z","content":"sanitized question output"}`,
			role: "assistant", text: "[Tool result: ASK_QUESTION]\n\"sanitized question output\"", second: 8,
		},
		{
			name: "conversation history marker",
			line: `{"step_index":0,"type":"CONVERSATION_HISTORY","source":"SYSTEM","status":"DONE","created_at":"2026-11-24T09:59:00Z"}`,
		},
		{
			name: "checkpoint state marker",
			line: `{"step_index":10,"type":"CHECKPOINT","source":"SYSTEM","status":"DONE","created_at":"2026-11-24T10:00:09Z","content":"sanitized checkpoint state"}`,
		},
		{
			name: "unclassified generic record",
			line: `{"step_index":5,"type":"GENERIC","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:03Z","content":"sanitized unclassified fixture"}`,
		},
		{
			name: "malformed native record",
			line: `{"step_index":6,"type":"PLANNER_RESPONSE","created_at":`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseMessages("antigravity", []byte(tc.line))
			if tc.role == "" {
				if len(got) != 0 {
					t.Fatalf("ParseMessages() = %+v, want no conversation message", got)
				}
				return
			}
			if len(got) != 1 || got[0].Role != tc.role || got[0].Text != tc.text {
				t.Fatalf("ParseMessages() = %+v, want %s %q", got, tc.role, tc.text)
			}
			wantTime := time.Date(2026, 11, 24, 10, 0, tc.second, 0, time.UTC)
			if !got[0].OccurredAt.Equal(wantTime) {
				t.Fatalf("OccurredAt = %v, want %v", got[0].OccurredAt, wantTime)
			}
		})
	}
}

func TestParseAntigravityCLIPlannerToolCalls(t *testing.T) {
	line := `{"type":"PLANNER_RESPONSE","source":"MODEL","created_at":"2026-11-24T10:00:01Z","tool_calls":[{"name":"RUN_COMMAND","args":{"command":"fixture"}}]}`
	got := ParseMessages("antigravity", []byte(line))
	want := FormatToolEvent("RUN_COMMAND", "call", "", json.RawMessage(`{"command":"fixture"}`))
	if len(got) != 1 || got[0].Role != "assistant" || got[0].Text != want {
		t.Fatalf("Antigravity tool call messages = %+v, want assistant tool event %q", got, want)
	}
}

func TestParseAntigravityCLIPlannerContentAndToolCallFromObservedShape(t *testing.T) {
	line := `{"step_index":2,"type":"PLANNER_RESPONSE","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:01Z","thinking":"sanitized hidden reasoning","content":"sanitized response","tool_calls":[{"name":"RUN_COMMAND","args":{"command":"fixture"}}]}`
	got := ParseMessages("antigravity", []byte(line))
	if len(got) != 2 {
		t.Fatalf("Antigravity planner record decoded to %d messages, want assistant content and tool call", len(got))
	}
	if got[0].Role != "assistant" || got[0].Text != "sanitized response" {
		t.Fatalf("Antigravity planner content = %+v, want sanitized assistant response", got[0])
	}
	wantCall := FormatToolEvent("RUN_COMMAND", "call", "", json.RawMessage(`{"command":"fixture"}`))
	if got[1].Role != "assistant" || got[1].Text != wantCall {
		t.Fatalf("Antigravity planner tool call = %+v, want %q", got[1], wantCall)
	}
	wantTime := time.Date(2026, 11, 24, 10, 0, 1, 0, time.UTC)
	for i, message := range got {
		if !message.OccurredAt.Equal(wantTime) {
			t.Errorf("message %d OccurredAt = %v, want %v", i, message.OccurredAt, wantTime)
		}
		if strings.Contains(message.Text, "sanitized hidden reasoning") {
			t.Errorf("message %d leaked thinking field", i)
		}
	}
}

func TestAntigravityCLIArchiveDescriptorSelectsFullTranscriptAndConversationID(t *testing.T) {
	format, ok := FormatFor("antigravity")
	if !ok || format.MessageDialect != "antigravity-cli-jsonl" || format.ArchiveFileName != "transcript_full.jsonl" || format.ArchiveID != "antigravity-brain" {
		t.Fatalf("Antigravity CLI descriptor = %+v, found=%v", format, ok)
	}
	roots := HomeRootPaths("/operator-home", "antigravity")
	wantRoot := filepath.Join("/operator-home", ".gemini", "antigravity-cli", "brain")
	if len(roots) != 1 || roots[0] != wantRoot {
		t.Fatalf("Antigravity transcript roots = %#v, want %q", roots, wantRoot)
	}
	path := filepath.Join(wantRoot, "native-session", ".system_generated", "logs", "transcript_full.jsonl")
	if !IsNativeArchiveFile("antigravity", path) {
		t.Fatalf("expected native Antigravity transcript %q to match", path)
	}
	if IsNativeArchiveFile("antigravity", filepath.Join(filepath.Dir(path), "transcript.jsonl")) {
		t.Fatal("reduced transcript.jsonl must not be indexed alongside transcript_full.jsonl")
	}
	sessionID, found := NativeArchiveSessionID("antigravity", path, nil)
	if !found || sessionID != "native-session" {
		t.Fatalf("NativeArchiveSessionID() = %q, %v; want native-session", sessionID, found)
	}
	identity := ReadIdentity(path, "antigravity", nil)
	if identity.SessionID != "native-session" || identity.HeaderSessionID != "native-session" {
		t.Fatalf("ReadIdentity() = %+v, want path-derived native session identity", identity)
	}
}

package agenttranscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadIdentitySharesNativeHeadersAndLineage(t *testing.T) {
	parent := "11111111-1111-4111-8111-111111111111"
	child := "22222222-2222-4222-8222-222222222222"
	tests := []struct {
		name, path, harness, contents string
		want                          TranscriptIdentity
	}{
		{
			name: "opencode archive session", path: "archive/session.jsonl", harness: "opencode",
			contents: `{"info":{"sessionID":"ses_native"},"parts":[]}`,
			want:     TranscriptIdentity{Harness: "opencode", SessionID: "ses_native", HeaderSessionID: "ses_native"},
		},
		{
			name: "grok acp session", path: "sessions/native.jsonl", harness: "grok",
			contents: `{"params":{"sessionId":"grok-native"}}`,
			want:     TranscriptIdentity{Harness: "grok", SessionID: "grok-native", HeaderSessionID: "grok-native"},
		},
		{
			name: "Gemini CLI document session", path: "chats/session-display-name.json", harness: "gemini-cli",
			contents: `{"sessionId":"gemini-native","messages":[]}`,
			want:     TranscriptIdentity{Harness: "gemini-cli", SessionID: "gemini-native", HeaderSessionID: "gemini-native"},
		},
		{
			name: "codex child lineage", path: "rollout-1-" + parent + "_" + child + ".jsonl", harness: "codex",
			contents: `{"type":"session_meta","payload":{"id":"` + child + `","source":{"subagent":{"thread_spawn":{"parent_thread_id":"` + parent + `"}}}}}`,
			want:     TranscriptIdentity{Harness: "codex", SessionID: child, HeaderSessionID: child, ParentSessionID: parent},
		},
		{
			name: "claude sidechain", path: "project/" + parent + "/subagents/agent.jsonl", harness: "claude-code",
			contents: `{"sessionId":"claude-child"}`,
			want:     TranscriptIdentity{Harness: "claude", SessionID: "agent", HeaderSessionID: "claude-child", ParentSessionID: parent, Sidechain: true},
		},
		{
			name: "configured root sniff", path: "unknown.jsonl", harness: "configured",
			contents: `{"type":"thread.started","thread_id":"codex-sniffed"}`,
			want:     TranscriptIdentity{Harness: "codex", SessionID: "codex-sniffed", HeaderSessionID: "codex-sniffed"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), filepath.FromSlash(tc.path))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.contents+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			got := ReadIdentity(path, tc.harness, nil)
			if tc.name == "claude sidechain" {
				tc.want.ParentPath = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(path))), parent+".jsonl")
			}
			if got != tc.want {
				t.Fatalf("ReadIdentity() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

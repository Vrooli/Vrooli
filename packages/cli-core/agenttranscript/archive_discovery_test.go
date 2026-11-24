package agenttranscript

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeArchiveFileRulesAndSessionIDs(t *testing.T) {
	tests := []struct {
		name, harness, path, contents, wantID string
		wantMatch                             bool
	}{
		{"Claude project transcript", "claude", "/home/.claude/projects/p/session-1.jsonl", "{}\n", "session-1", true},
		{"Codex rollout metadata", "codex", "/home/.codex/sessions/2026/01/01/rollout-1.jsonl", `{"type":"session_meta","payload":{"id":"codex-native"}}` + "\n", "codex-native", true},
		{"Grok updates", "grok", "/home/.grok/sessions/project/session-2/updates.jsonl", "{}\n", "session-2", true},
		{"OpenCode message", "opencode", "/home/.local/share/opencode/storage/message/session-3/msg.json", "{}", "session-3", true},
		{"Gemini chat", "gemini-cli", "/home/.gemini/tmp/project/chats/session-4.json", `{"sessionId":"gemini-native","messages":[]}`, "gemini-native", true},
		{"Cursor database", "cursor", "/home/.config/Cursor/User/workspaceStorage/workspace-4/state.vscdb", "db", "", true},
		{"Grok rejects other JSONL", "grok", "/home/.grok/sessions/project/session-2/other.jsonl", "{}\n", "", false},
		{"Gemini rejects settings", "gemini-cli", "/home/.gemini/tmp/project/chats/settings.json", `{}`, "", false},
		{"Cursor rejects unrelated SQLite", "cursor", "/home/.config/Cursor/User/workspaceStorage/workspace-4/other.vscdb", "db", "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), filepath.Base(filepath.Dir(test.path)), filepath.Base(test.path))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			matched := IsNativeArchiveFile(test.harness, path)
			if matched != test.wantMatch {
				t.Fatalf("IsNativeArchiveFile(%q, %q) = %v, want %v", test.harness, path, matched, test.wantMatch)
			}
			id, ok := NativeArchiveSessionID(test.harness, path, info)
			if ok != test.wantMatch {
				t.Fatalf("NativeArchiveSessionID found=%v, want %v", ok, test.wantMatch)
			}
			if test.harness == "cursor" && ok {
				sum := sha256.Sum256([]byte(filepath.Dir(path)))
				test.wantID = hex.EncodeToString(sum[:8])
			}
			if ok && id != test.wantID {
				t.Fatalf("native session ID = %q, want %q", id, test.wantID)
			}
		})
	}
}

func TestHomeRootPathsResolveFormatData(t *testing.T) {
	got := HomeRootPaths("/operator", "opencode")
	if len(got) != 1 || got[0] != filepath.Join("/operator", ".local/share/opencode/storage/message") {
		t.Fatalf("OpenCode home roots = %#v", got)
	}
	if got := HomeRootPaths("/operator", "unknown"); len(got) != 0 {
		t.Fatalf("unknown home roots = %#v", got)
	}
}

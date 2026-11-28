package transcriptindex_test

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/transcriptindex"
	"github.com/vrooli/cli-core/agenttranscript"
)

func TestGrokACPHomeRootIsDiscoveredAndKeepsSessionIdentity(t *testing.T) {
	home := t.TempDir()
	transcript := filepath.Join(home, ".grok", "sessions", "workspace-hash", "session-42", "updates.jsonl")
	if err := os.MkdirAll(filepath.Dir(transcript), 0o700); err != nil {
		t.Fatal(err)
	}
	line := `{"method":"session/update","params":{"sessionId":"session-42","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"indexed Grok response"}}}}` + "\n"
	if err := os.WriteFile(transcript, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	var grokRoot *transcriptindex.Root
	for _, root := range transcriptindex.Roots(transcriptindex.RootConfig{Home: home}) {
		if root.Harness == transcriptindex.HarnessGrok {
			copy := root
			grokRoot = &copy
			break
		}
	}
	if grokRoot == nil || grokRoot.Path != filepath.Dir(filepath.Dir(filepath.Dir(transcript))) || grokRoot.Kind != transcriptindex.RootGrokHome {
		t.Fatalf("Grok root = %#v, want native sessions directory", grokRoot)
	}
	candidates, err := transcriptindex.Walk(*grokRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Path != transcript {
		t.Fatalf("Grok candidates = %#v, want %s", candidates, transcript)
	}
	identity := transcriptindex.ReadIdentity(transcript, grokRoot.Harness)
	if identity.SessionID != "session-42" || identity.HeaderSessionID != "session-42" || identity.Harness != transcriptindex.HarnessGrok {
		t.Fatalf("Grok identity = %#v", identity)
	}
	messages := agenttranscript.ParseMessages(agenttranscript.HarnessGrok, []byte(line))
	if len(messages) != 1 || messages[0].Text != "indexed Grok response" {
		t.Fatalf("shared Grok decode = %#v", messages)
	}
}

func TestGrokDiscoveryIncludesRunScopedAndConfiguredHomes(t *testing.T) {
	home, runState, configured := t.TempDir(), t.TempDir(), t.TempDir()
	runID := "run-17"
	runSession := filepath.Join(runState, runID, "grok", "sessions", "cwd-hash", "session-17", "updates.jsonl")
	if err := os.MkdirAll(filepath.Dir(runSession), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runSession, []byte(`{"method":"session/update","params":{"sessionId":"session-17","update":{"sessionUpdate":"turn_completed"}}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	customSession := filepath.Join(configured, "cwd-hash", "session-custom", "updates.jsonl")
	if err := os.MkdirAll(filepath.Dir(customSession), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(customSession, []byte(`{"method":"session/update","params":{"sessionId":"session-custom","update":{"sessionUpdate":"turn_completed"}}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	roots := transcriptindex.Roots(transcriptindex.RootConfig{Home: home, RunStateRoot: runState, GrokSessionsRoot: configured})
	foundRun, foundConfigured := false, false
	for _, root := range roots {
		if root.Harness != transcriptindex.HarnessGrok {
			continue
		}
		candidates, err := transcriptindex.Walk(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, candidate := range candidates {
			identity := transcriptindex.ReadIdentity(candidate.Path, root.Harness)
			if candidate.Path == runSession && root.Kind == transcriptindex.RootRunHome && root.RunID == runID && identity.SessionID == "session-17" {
				foundRun = true
			}
			if candidate.Path == customSession && root.Kind == transcriptindex.RootGrokHome && identity.SessionID == "session-custom" {
				foundConfigured = true
			}
		}
	}
	if !foundRun || !foundConfigured {
		t.Fatalf("Grok discovery: run-scoped=%v configured=%v roots=%+v", foundRun, foundConfigured, roots)
	}
}

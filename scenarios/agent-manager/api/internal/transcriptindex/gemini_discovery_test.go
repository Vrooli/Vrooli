package transcriptindex_test

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/transcriptindex"
)

func TestGeminiCLIFullDocumentIsDiscoveredBySharedFormatRoot(t *testing.T) {
	home := t.TempDir()
	chats := filepath.Join(home, ".gemini", "tmp", "project-hash", "chats")
	if err := os.MkdirAll(chats, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(chats, "session-native.json")
	if err := os.WriteFile(transcript, []byte(`{"sessionId":"gemini-native","messages":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chats, "settings.json"), []byte(`{"enabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var geminiRoot *transcriptindex.Root
	for _, root := range transcriptindex.Roots(transcriptindex.RootConfig{Home: home}) {
		if root.Kind == transcriptindex.RootGeminiCLI {
			found := root
			geminiRoot = &found
			break
		}
	}
	if geminiRoot == nil {
		t.Fatal("shared Gemini CLI format root was not discovered")
	}
	candidates, err := transcriptindex.Walk(*geminiRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Path != transcript {
		t.Fatalf("Gemini candidates = %+v, want only the session archive", candidates)
	}
	identity := transcriptindex.ReadIdentity(transcript, geminiRoot.Harness)
	if identity.Harness != "gemini-cli" || identity.SessionID != "gemini-native" || identity.HeaderSessionID != "gemini-native" {
		t.Fatalf("Gemini identity = %+v", identity)
	}
}

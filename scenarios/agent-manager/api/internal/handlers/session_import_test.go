package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"
)

func TestScanRunnerSessionsUsesOpaqueKeysAndReadsCodexConversationTitle(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "2026", "07", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	contents := "{\"type\":\"session_meta\",\"payload\":{\"id\":\"session-1\"}}\n{\"type\":\"response_item\",\"payload\":{\"role\":\"user\",\"content\":[{\"text\":\"Repair the import workflow\"}]}}\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	sessions, err := scanRunnerSessions(runnerSessionSource{RunnerType: domain.RunnerTypeCodex, Root: root}, map[string]string{"session-1": "run-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(sessions))
	}
	if got := sessions[0]; got.Key != "2026/07/rollout.jsonl" || got.Title != "Repair the import workflow" || got.ImportedRunID != "run-1" {
		t.Fatalf("unexpected session: %#v", got)
	}
}

func TestSafeSessionPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	source := runnerSessionSource{Root: root}
	if _, ok := safeSessionPath(source, "../secrets.jsonl"); ok {
		t.Fatal("path traversal accepted")
	}
	if _, ok := safeSessionPath(source, "/tmp/session.jsonl"); ok {
		t.Fatal("absolute path accepted")
	}
	path := filepath.Join(root, "session.jsonl")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ok := safeSessionPath(source, "session.jsonl"); !ok || got != path {
		t.Fatalf("safe path = %q, %v", got, ok)
	}
}

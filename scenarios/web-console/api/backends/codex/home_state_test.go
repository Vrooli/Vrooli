package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRolloutRootUsesSharedSessionMeta(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\"thread-1\",\"session_id\":\"root-1\",\"cwd\":\"/workspace\",\"thread_source\":\"subagent\"}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, ok := ReadRolloutRoot(path)
	if !ok {
		t.Fatal("valid session_meta did not resolve rollout root")
	}
	if root.ThreadID != "thread-1" || root.RootID != "root-1" || root.CWD != "/workspace" || !root.SubAgent {
		t.Fatalf("rollout root=%+v", root)
	}
}

func TestReadRolloutRootRejectsMissingThreadID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"session_meta\",\"payload\":{\"session_id\":\"root-1\"}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if root, ok := ReadRolloutRoot(path); ok || root != (RolloutRoot{}) {
		t.Fatalf("missing thread id produced root=%+v ok=%v", root, ok)
	}
}

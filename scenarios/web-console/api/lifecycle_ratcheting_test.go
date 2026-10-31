package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLifecycleDeletionHasOneApplicationOwner is a structural ratchet. The
// repository may implement the primitive delete, but only the sessions
// adapter may invoke it as a lifecycle operation. Process, websocket, expiry,
// and workspace code must preserve metadata and route policy decisions through
// that owner.
func TestLifecycleDeletionHasOneApplicationOwner(t *testing.T) {
	allowed := map[string]bool{
		filepath.Clean("internal/sessionstore/store.go"): true,
		filepath.Clean("handlers/sessions/adapter.go"):   true,
	}
	conversationOwner := filepath.Clean("handlers/sessions/adapter.go")
	retentionOwner := filepath.Clean("conversation_retention.go")
	err := filepath.Walk(".", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			cleanPath := filepath.Clean(path)
			if strings.Contains(line, ".Store.Delete(") && !allowed[cleanPath] {
				t.Errorf("unauthorized lifecycle delete caller %s", path)
			}
			if strings.Contains(line, "DELETE FROM sessions") && !allowed[cleanPath] {
				t.Errorf("unauthorized session metadata delete in %s", path)
			}
			for _, call := range []string{
				".Conversations.DeleteSession(",
				".CodexCheckpoints.DeleteSession(",
				".AgentCheckpoints.DeleteSession(",
			} {
				if strings.Contains(line, call) && cleanPath != conversationOwner {
					t.Errorf("unauthorized durable evidence delete caller %s", path)
				}
			}
			if strings.Contains(line, ".PruneEvents(") && cleanPath != retentionOwner {
				t.Errorf("unauthorized conversation retention caller %s", path)
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
}

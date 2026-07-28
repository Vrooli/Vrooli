package heartbeat

import (
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/store"
)

func newFileStore(t testing.TB, roots paths.Roots) *store.FileStore {
	t.Helper()
	fileStore, err := store.NewFileStore(roots)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	return fileStore
}

// newTestExecutor builds an Executor and drains its completion waiters before
// the test's temporary directories are removed. Execute starts those waiters
// on a background context, so a test that triggers a heartbeat and returns
// leaves them writing into a directory t.TempDir is concurrently deleting.
// Tests must use this instead of calling NewExecutor directly.
func newTestExecutor(
	t testing.TB,
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	agentClient AgentClient,
	vrooliRoot string,
	runRegistry *RunRegistry,
	handoffExtractor HandoffExtractor,
) *Executor {
	t.Helper()
	executor := NewExecutor(teamStore, agentStore, agentClient, vrooliRoot, runRegistry, handoffExtractor)
	t.Cleanup(executor.Shutdown)
	return executor
}

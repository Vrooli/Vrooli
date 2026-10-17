package heartbeat

import (
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
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

// testHandlersOption customizes what a test's Handlers is wired with.
type testHandlersOption func(*testHandlersConfig)

type testHandlersConfig struct {
	agentClient      AgentClient
	withExecutor     bool
	vrooliRoot       string
	runRegistry      *RunRegistry
	teamExecStore    *TeamExecutionStore
	scheduler        *Scheduler
	handoffExtractor HandoffExtractor
}

// withAgentClient wires an agent client. A test that asserts on spawn behavior

// withoutStores builds Handlers with no file stores at all, for tests that only

// testHandlers is what newTestHandlers hands back: the handlers plus the stores
// a test usually needs to seed. Returning a struct rather than a tuple means
// adding a store later does not touch every call site.
type testHandlers struct {
	Handlers      *Handlers
	TeamStore     *store.FileTeamStore
	AgentStore    *store.FileAgentStore
	RelationStore store.RelationStore
	Executor      *Executor
	Root          string
}

// newTestHandlers replaces the six near-identical setup*TestHandlers functions
// that each rebuilt the same store/executor/handlers chain and differed only in
// which collaborator they wired and which subset they returned.
//
// The executor's Shutdown is registered with t.Cleanup here, once, so a test
// cannot forget it. Execute starts completion waiters on a background context;
// a test that returns without draining them leaves goroutines writing into a
// directory t.TempDir is concurrently removing.
func newTestHandlers(t *testing.T, opts ...testHandlersOption) testHandlers {
	t.Helper()
	cfg := testHandlersConfig{withExecutor: true}
	for _, opt := range opts {
		opt(&cfg)
	}
	if !cfg.withExecutor {
		return testHandlers{Handlers: NewHandlers(HandlersDeps{
			AgentClient:   cfg.agentClient,
			RunRegistry:   cfg.runRegistry,
			TeamExecStore: cfg.teamExecStore,
			Scheduler:     cfg.scheduler,
		})}
	}

	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := newTestExecutor(t, teamStore, agentStore, cfg.agentClient, cfg.vrooliRoot, cfg.runRegistry, cfg.handoffExtractor)

	return testHandlers{
		Handlers: NewHandlers(HandlersDeps{
			TeamStore:     teamStore,
			AgentStore:    agentStore,
			RelationStore: relationStore,
			Scheduler:     cfg.scheduler,
			Executor:      executor,
			RunRegistry:   cfg.runRegistry,
			AgentClient:   cfg.agentClient,
			TeamExecStore: cfg.teamExecStore,
		}),
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: relationStore,
		Executor:      executor,
		Root:          roots.Config,
	}
}

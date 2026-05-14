package heartbeat

import (
	"context"
	"log"
	"prompt-manager/store"
	"sync"
)

// TeamExecutionStore manages TeamExecutionContexts for all teams.
type TeamExecutionStore struct {
	mu          sync.RWMutex
	contexts    map[string]*TeamExecutionContext
	executor    HeartbeatExecutor
	persistDir  string
	teamStore   *store.FileTeamStore
	agentClient AgentClient
}

// NewTeamExecutionStore creates a new store for team execution contexts.
// agentClient may be nil in tests that don't exercise Recover reconciliation.
func NewTeamExecutionStore(teamStore *store.FileTeamStore, executor HeartbeatExecutor, persistDir string, agentClient AgentClient) *TeamExecutionStore {
	return &TeamExecutionStore{
		contexts:    make(map[string]*TeamExecutionContext),
		executor:    executor,
		persistDir:  persistDir,
		teamStore:   teamStore,
		agentClient: agentClient,
	}
}

// GetOrCreate returns the TeamExecutionContext for the given team, creating one if needed.
func (s *TeamExecutionStore) GetOrCreate(teamID string) *TeamExecutionContext {
	s.mu.RLock()
	ctx, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if ok {
		return ctx
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if ctx, ok := s.contexts[teamID]; ok {
		return ctx
	}

	ctx = newTeamExecutionContext(teamID, s.executor, s.persistDir, s.agentClient)
	s.contexts[teamID] = ctx
	return ctx
}

// SetRunningRunID routes a RunID update to the context for teamID.
// No-op if no context exists yet for the team.
func (s *TeamExecutionStore) SetRunningRunID(teamID, agentID, runID string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	tec.SetRunningRunID(agentID, runID)
}

// ClearRunning clears a single running entry on the given team's context.
// Returns ErrRunningEntryNotFound if the context or entry is missing.
func (s *TeamExecutionStore) ClearRunning(ctx context.Context, teamID, agentID string, force bool) error {
	tec, err := s.configureContext(ctx, teamID)
	if err != nil {
		return err
	}
	return tec.ClearRunning(ctx, agentID, force)
}

// Enqueue delegates to the team's execution context.
func (s *TeamExecutionStore) Enqueue(ctx context.Context, teamID, agentID, profileKey string) (*EnqueueResult, error) {
	tec, err := s.configureContext(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return tec.Enqueue(ctx, agentID, profileKey)
}

// Status returns the execution status for a team.
func (s *TeamExecutionStore) Status(teamID string) TeamExecutionStatus {
	if tec, err := s.configureContext(context.Background(), teamID); err == nil {
		return tec.Status()
	}

	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()

	if !ok {
		return TeamExecutionStatus{
			TeamID: teamID,
			State:  "idle",
			Queue:  []string{},
		}
	}
	return tec.Status()
}

// OnComplete is the callback invoked when a heartbeat execution finishes.
// It routes the completion to the correct TeamExecutionContext.
func (s *TeamExecutionStore) OnComplete(teamID, agentID string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()

	if !ok {
		return
	}
	tec.OnMemberComplete(agentID)
}

// Recover loads persisted queue state for all known teams from disk and
// reconciles each running entry against agent-manager via the per-context
// AgentClient. See TeamExecutionContext.Recover for details.
func (s *TeamExecutionStore) Recover(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := readDirSafe(s.persistDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !isTeamQueueFile(name) {
			continue
		}
		teamID := extractTeamID(name)
		if teamID == "" {
			continue
		}

		tec := newTeamExecutionContext(teamID, s.executor, s.persistDir, s.agentClient)
		tec.Recover(ctx)
		s.contexts[teamID] = tec
	}

	if len(s.contexts) > 0 {
		log.Printf("team_execution_store: recovered %d team context(s)", len(s.contexts))
	}
}

func (s *TeamExecutionStore) configureContext(ctx context.Context, teamID string) (*TeamExecutionContext, error) {
	tec := s.GetOrCreate(teamID)
	if s.teamStore == nil {
		return tec, nil
	}
	team, err := s.teamStore.Get(ctx, teamID)
	if err != nil {
		return nil, err
	}
	tec.Configure(team.Execution.QueuePolicy, team.Execution.MaxConcurrentRuns)
	return tec, nil
}

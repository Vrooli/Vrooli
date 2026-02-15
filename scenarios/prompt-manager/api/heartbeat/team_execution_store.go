package heartbeat

import (
	"context"
	"log"
	"sync"
)

// TeamExecutionStore manages TeamExecutionContexts for all teams.
type TeamExecutionStore struct {
	mu         sync.RWMutex
	contexts   map[string]*TeamExecutionContext
	executor   HeartbeatExecutor
	persistDir string
}

// NewTeamExecutionStore creates a new store for team execution contexts.
func NewTeamExecutionStore(executor HeartbeatExecutor, persistDir string) *TeamExecutionStore {
	return &TeamExecutionStore{
		contexts:   make(map[string]*TeamExecutionContext),
		executor:   executor,
		persistDir: persistDir,
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

	ctx = newTeamExecutionContext(teamID, s.executor, s.persistDir)
	s.contexts[teamID] = ctx
	return ctx
}

// Enqueue delegates to the team's execution context.
func (s *TeamExecutionStore) Enqueue(ctx context.Context, teamID, agentID, profileKey string) (*EnqueueResult, error) {
	return s.GetOrCreate(teamID).Enqueue(ctx, agentID, profileKey)
}

// Status returns the execution status for a team.
func (s *TeamExecutionStore) Status(teamID string) TeamExecutionStatus {
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

// Recover loads persisted queue state for all known teams from disk.
func (s *TeamExecutionStore) Recover(_ context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Scan persist dir for team-queue-*.json files
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

		tec := newTeamExecutionContext(teamID, s.executor, s.persistDir)
		tec.Recover()
		s.contexts[teamID] = tec
	}

	if len(s.contexts) > 0 {
		log.Printf("team_execution_store: recovered %d team context(s)", len(s.contexts))
	}
}

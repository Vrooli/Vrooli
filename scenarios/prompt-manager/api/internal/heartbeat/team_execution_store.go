package heartbeat

import (
	"context"
	"log"
	"sync"
	"time"

	"prompt-manager/internal/store"
)

// TeamExecutionStore manages TeamExecutionContexts for all teams.
type TeamExecutionStore struct {
	mu          sync.RWMutex
	contexts    map[string]*TeamExecutionContext
	executor    HeartbeatExecutor
	persistDir  string
	teamStore   *store.FileTeamStore
	agentClient AgentClient
	// resumer and clock are defaults applied to every context the store
	// creates, so a store configured once covers contexts recovered later.
	resumer PausedRunResumer
	clock   func() time.Time
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
	s.applyDefaults(ctx)
	s.contexts[teamID] = ctx
	return ctx
}

// applyDefaults copies store-level pause/resume configuration onto a freshly
// created context. Caller holds s.mu; the context is not yet shared.
func (s *TeamExecutionStore) applyDefaults(tec *TeamExecutionContext) {
	if s.resumer != nil {
		tec.resumer = s.resumer
	}
	if s.clock != nil {
		tec.clock = s.clock
	}
}

// SetPausedRunResumer installs the runtime-owner pause/resume port on the store
// and every existing context.
func (s *TeamExecutionStore) SetPausedRunResumer(resumer PausedRunResumer) {
	s.mu.Lock()
	s.resumer = resumer
	contexts := make([]*TeamExecutionContext, 0, len(s.contexts))
	for _, tec := range s.contexts {
		contexts = append(contexts, tec)
	}
	s.mu.Unlock()
	for _, tec := range contexts {
		tec.SetPausedRunResumer(resumer)
	}
}

// SetClock overrides the reset-eligibility clock on the store and every
// existing context (tests only).
func (s *TeamExecutionStore) SetClock(now func() time.Time) {
	s.mu.Lock()
	s.clock = now
	contexts := make([]*TeamExecutionContext, 0, len(s.contexts))
	for _, tec := range s.contexts {
		contexts = append(contexts, tec)
	}
	s.mu.Unlock()
	for _, tec := range contexts {
		tec.SetClock(now)
	}
}

// ConsumeResetEligibility routes one runtime-owner reset-eligibility event to
// the context for its team.
func (s *TeamExecutionStore) ConsumeResetEligibility(ctx context.Context, ev ResetEligibility) (ResetConsumeOutcome, error) {
	if ev.TeamID == "" {
		return ResetConsumeNone, nil
	}
	tec := s.GetOrCreate(ev.TeamID)
	return tec.ConsumeResetEligibility(ctx, ev)
}

// RunResetEligibility consumes owner reset-eligibility events until the source
// is exhausted or ctx is done. It is event-driven: it blocks on the source and
// never wakes on a timer, so an unknown reset is not polled.
func (s *TeamExecutionStore) RunResetEligibility(ctx context.Context, src ResetEligibilitySource) {
	if src == nil {
		return
	}
	for {
		ev, ok, err := src.Next(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("team_execution_store: reset eligibility source stopped: %v", err)
			}
			return
		}
		if !ok {
			return
		}
		if _, err := s.ConsumeResetEligibility(ctx, ev); err != nil {
			log.Printf("team_execution_store: consuming reset eligibility for %s/%s: %v", ev.TeamID, ev.AgentID, err)
		}
	}
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

// BeginDispatch routes a pre-request dispatch intent to the context for teamID.
// No-op if no context exists yet for the team.
func (s *TeamExecutionStore) BeginDispatch(teamID, agentID string, intent DispatchIntent) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	tec.BeginDispatch(agentID, intent)
}

// ReconcileDispatch routes an owner-mediated replay of a retained uncertain
// dispatch to the context for teamID.
func (s *TeamExecutionStore) ReconcileDispatch(ctx context.Context, teamID, agentID string) error {
	tec, err := s.configureContext(ctx, teamID)
	if err != nil {
		return err
	}
	return tec.ReconcileDispatch(ctx, agentID)
}

// MarkDispatchUncertain routes a retained uncertain obligation to the context
// for teamID. No-op if no context exists yet for the team.
func (s *TeamExecutionStore) MarkDispatchUncertain(teamID, agentID, reason string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	tec.MarkDispatchUncertain(agentID, reason)
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
		s.applyDefaults(tec)
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

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

// Shutdown waits for all context dispatch goroutines to finish.
func (s *TeamExecutionStore) Shutdown() {
	s.mu.RLock()
	contexts := make([]*TeamExecutionContext, 0, len(s.contexts))
	for _, ctx := range s.contexts {
		contexts = append(contexts, ctx)
	}
	s.mu.RUnlock()
	for _, ctx := range contexts {
		ctx.Shutdown()
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
func (s *TeamExecutionStore) SetRunningRunID(teamID, agentID, runID string, ownerStartedAt ...string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	tec.SetRunningRunID(agentID, runID, ownerStartedAt...)
}

// BeginDispatch routes a pre-request dispatch intent to the context for teamID.
// A missing context is an error: callers must not dispatch without proving
// that the durable queue obligation was updated.
func (s *TeamExecutionStore) BeginDispatch(teamID, agentID string, intent DispatchIntent) error {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return ErrRunningEntryNotFound
	}
	return tec.BeginDispatch(agentID, intent)
}

// ConfirmTaskDispatch persists the verified task phase before CreateRun.
func (s *TeamExecutionStore) ConfirmTaskDispatch(teamID, agentID string) error {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return ErrRunningEntryNotFound
	}
	return tec.ConfirmTaskDispatch(agentID)
}

// CancelUnsentTaskDispatch releases only the exact durable task intent whose
// owner request was proven not to have been sent.
func (s *TeamExecutionStore) CancelUnsentTaskDispatch(teamID, agentID, taskID, idempotencyKey, taskSnapshot string) error {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return ErrRunningEntryNotFound
	}
	return tec.CancelUnsentTaskDispatch(agentID, taskID, idempotencyKey, taskSnapshot)
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

func (s *TeamExecutionStore) MarkApprovalPending(teamID, agentID string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if ok {
		tec.MarkApprovalPending(agentID)
	}
}

// MarkAdmissionDeferred retains the exact durable dispatch after a typed
// owner capacity refusal so the existing schedule can replay it later.
func (s *TeamExecutionStore) MarkAdmissionDeferred(teamID, agentID string) error {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return ErrRunningEntryNotFound
	}
	return tec.MarkAdmissionDeferred(agentID)
}

// RetryApprovalPending replays an existing run-stage intent on a scheduled
// callback. It returns retried=true even when approval remains pending so the
// same callback cannot also admit a fresh task/run.
func (s *TeamExecutionStore) RetryApprovalPending(ctx context.Context, teamID, agentID string) (retried bool, err error) {
	tec, err := s.configureContext(ctx, teamID)
	if err != nil {
		return false, err
	}
	for _, pendingAgentID := range tec.Status().ApprovalPendingAgentIDs {
		if pendingAgentID == agentID {
			return true, tec.ReconcileDispatch(ctx, agentID)
		}
	}
	return false, nil
}

// RetryAdmissionDeferred replays a single durable request only when the
// scheduler has already established current team/member/config eligibility.
func (s *TeamExecutionStore) RetryAdmissionDeferred(ctx context.Context, teamID, agentID string) (bool, error) {
	tec, err := s.configureContext(ctx, teamID)
	if err != nil {
		return false, err
	}
	return tec.RetryAdmissionDeferred(ctx, agentID)
}

// RetryManualDispatch replays a same-caller retained manual intent through
// the existing manual-trigger route, after that route checks current owner
// eligibility and the fresh caller binding.
func (s *TeamExecutionStore) RetryManualDispatch(ctx context.Context, teamID, agentID string) (bool, error) {
	tec, err := s.configureContext(ctx, teamID)
	if err != nil {
		return false, err
	}
	return tec.RetryManualDispatch(ctx, agentID)
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
			Owners: []TeamExecutionOwner{},
		}
	}
	return tec.Status()
}

// OnRunComplete routes completion only to the exact retained run owner.
func (s *TeamExecutionStore) OnRunComplete(teamID, agentID, runID string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	tec.OnMemberCompleteRun(agentID, runID)
}

func (s *TeamExecutionStore) IsCurrentRun(teamID, agentID, runID string) bool {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	return ok && tec.OwnsRun(agentID, runID)
}

func (s *TeamExecutionStore) ProfileKeyForRun(teamID, agentID, runID string) (string, bool) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	tec.mu.Lock()
	defer tec.mu.Unlock()
	entry, ok := tec.running[agentID]
	if !ok || entry.RunID != runID {
		return "", false
	}
	return entry.ProfileKey, true
}

func (s *TeamExecutionStore) MarkWatcherUnavailableForRun(teamID, agentID, runID string) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if ok {
		tec.MarkWatcherUnavailableForRun(agentID, runID)
	}
}

// MarkWatcherUnavailable retains the member slot when completion observation
// ends without definitive terminal owner evidence.
func (s *TeamExecutionStore) MarkWatcherUnavailable(teamID, agentID string) {
	tec := s.GetOrCreate(teamID)
	tec.MarkWatcherUnavailable(agentID)
}

// RetryUnavailableWatcher performs one scheduled owner reconciliation and
// reattaches a waiter only when the existing run remains nonterminal.
func (s *TeamExecutionStore) RetryUnavailableWatcher(ctx context.Context, teamID, agentID string) (bool, error) {
	tec := s.GetOrCreate(teamID)
	return tec.RetryUnavailableWatcher(ctx, agentID)
}

// Recover loads persisted queue state for all known teams from disk and
// reconciles each running entry against agent-manager via the per-context
// AgentClient. See TeamExecutionContext.Recover for details.
func (s *TeamExecutionStore) Recover(ctx context.Context) map[runOwnerIdentity]struct{} {
	entries, err := readDirSafe(s.persistDir)
	if err != nil {
		return nil
	}

	// Publish each context while holding the store lock, then release it before
	// recovery. Recover may synchronously reconcile a terminal run, whose
	// completion callback looks this context up through the same store.
	contexts := make([]*TeamExecutionContext, 0, len(entries))
	s.mu.Lock()
	for _, entry := range entries {
		name := entry.Name()
		if !isTeamQueueFile(name) {
			continue
		}
		teamID := extractTeamID(name)
		if teamID == "" {
			continue
		}
		if _, exists := s.contexts[teamID]; exists {
			continue
		}

		tec := newTeamExecutionContext(teamID, s.executor, s.persistDir, s.agentClient)
		s.applyDefaults(tec)
		// Publish before Recover can attach a waiter or invoke a synchronous
		// completion callback. GetOrCreate now returns this same context.
		s.contexts[teamID] = tec
		contexts = append(contexts, tec)
	}
	recoveredCount := len(s.contexts)
	s.mu.Unlock()

	queueOwners := make(map[runOwnerIdentity]struct{})
	for _, tec := range contexts {
		for _, owner := range tec.Recover(ctx) {
			queueOwners[runOwnerIdentity{TeamID: owner.TeamID, AgentID: owner.AgentID, RunID: owner.RunID}] = struct{}{}
		}
	}
	if recoveredCount > 0 {
		log.Printf("team_execution_store: recovered %d team context(s)", recoveredCount)
	}
	return queueOwners
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

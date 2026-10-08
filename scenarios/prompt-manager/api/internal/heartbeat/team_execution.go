package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/vrooli/api-core/effortauthority"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"prompt-manager/internal/teamconfig"
)

// TeamExecutionManager defines the interface for team-level execution coordination.
type TeamExecutionManager interface {
	Enqueue(ctx context.Context, teamID, agentID, profileKey string) (*EnqueueResult, error)
	Status(teamID string) TeamExecutionStatus
}

// EnqueueResult reports what happened when an agent was enqueued.
type EnqueueResult struct {
	TeamID   string `json:"teamId"`
	AgentID  string `json:"agentId"`
	Status   string `json:"status"`   // "started" | "queued"
	Position int    `json:"position"` // 0 if started, 1-based queue position if queued
}

// TeamExecutionStatus is a snapshot of the team's execution state.
type TeamExecutionStatus struct {
	TeamID            string   `json:"teamId"`
	State             string   `json:"state"` // "idle" | "active"
	RunningAgentIDs   []string `json:"runningAgentIds"`
	Queue             []string `json:"queue"`
	QueuePolicy       string   `json:"queuePolicy"`
	MaxConcurrentRuns int      `json:"maxConcurrentRuns"`
	// Unresolved ownership classes. Each listed member still holds its slot, so
	// no duplicate heartbeat can be admitted until it is reconciled or cleared.
	UncertainAgentIDs   []string `json:"uncertainAgentIds"`
	UnreachableAgentIDs []string `json:"unreachableAgentIds"`
	PausedAgentIDs      []string `json:"pausedAgentIds"`
}

// MemberAlreadyQueuedError is returned when a member is already queued or running.
type MemberAlreadyQueuedError struct {
	TeamID  string
	AgentID string
}

func (e *MemberAlreadyQueuedError) Error() string {
	return fmt.Sprintf("member %s/%s is already queued or running", e.TeamID, e.AgentID)
}

// IsMemberAlreadyQueued checks if an error is a MemberAlreadyQueuedError.
func IsMemberAlreadyQueued(err error) bool {
	_, ok := err.(*MemberAlreadyQueuedError)
	return ok
}

// ErrRunningEntryNotFound is returned by ClearRunning when no running entry
// exists for the given (teamID, agentID).
var ErrRunningEntryNotFound = fmt.Errorf("running entry not found")

// RunningStillActiveError is returned by ClearRunning when the backing
// agent-manager run is still in a non-terminal status. Operator must
// either stop the run via agent-manager or pass force=true.
type RunningStillActiveError struct {
	TeamID  string
	AgentID string
	RunID   string
	Status  string
}

func (e *RunningStillActiveError) Error() string {
	return fmt.Sprintf("running entry %s/%s has active run %s (status=%s); stop the run first or use --force", e.TeamID, e.AgentID, e.RunID, e.Status)
}

// IsRunningStillActive checks if an error is a RunningStillActiveError.
func IsRunningStillActive(err error) bool {
	_, ok := err.(*RunningStillActiveError)
	return ok
}

// Durable obligation states for a member's allocated execution slot. An empty
// state means an ordinary live run. The other states are nonterminal
// obligations that still hold the slot so a duplicate heartbeat cannot be
// admitted while ownership is unresolved. They are released only on definitive
// terminal owner evidence or an explicit operator recovery action.
const (
	// ObligationRunning is a normal live run (or a run reconciled as active).
	ObligationRunning = ""
	// ObligationDispatchUncertain means a dispatch intent was persisted (or a
	// start response was lost) but the run identity cannot be confirmed. The
	// slot is retained and the obligation is surfaced for owner reconciliation.
	ObligationDispatchUncertain = "dispatch_uncertain"
	// ObligationOwnerUnreachable means reconciliation could not reach the
	// owner. Absence of evidence is not terminal evidence; the slot is retained.
	ObligationOwnerUnreachable = "owner_unreachable"
	// ObligationPaused means the owner reports the run as suspended (parked or
	// quota-paused). It is nonterminal retained ownership, not a missing run.
	ObligationPaused = "paused"
)

type queuedExecution struct {
	EffortBinding   *effortauthority.Binding `json:"effortBinding,omitempty"`
	ManualAdmission bool                     `json:"manual_admission,omitempty"`
	// Only in-memory caller proof. Recovery requires fresh authorized admission.
	caller     context.Context `json:"-"`
	AgentID    string          `json:"AgentID"`
	ProfileKey string          `json:"ProfileKey"`
	// RunID is the agent-manager run ID for entries in the running slice.
	// Queued entries (not yet dispatched) carry an empty RunID; it is
	// populated by Executor.Execute via SetRunningRunID once CreateRun
	// returns. Recover uses RunID to reconcile against agent-manager.
	RunID string `json:"RunID,omitempty"`
	// State carries a durable obligation classification across restart. Empty
	// for ordinary running entries; see the Obligation* constants.
	State string `json:"State,omitempty"`
	// IdempotencyKey is the durable dispatch intent recorded before the owner
	// start request. It lets an uncertain obligation be replayed to the owner
	// (returning the same run) instead of starting a duplicate.
	IdempotencyKey string `json:"IdempotencyKey,omitempty"`
	// TaskID is the owner task the dispatch referenced. It is persisted with
	// the intent so an uncertain dispatch can be replayed against the owner
	// without rebuilding the prompt or creating a second task.
	TaskID string `json:"TaskID,omitempty"`
	// RunTag is the owner-visible run tag recorded with the intent. It is
	// preserved so a replayed run keeps the same attribution label.
	RunTag string `json:"RunTag,omitempty"`
	// Workload identity is persisted with the request so an uncertain replay
	// cannot silently lose the accounting key that was present on the original
	// dispatch.
	WorkloadKind     string `json:"WorkloadKind,omitempty"`
	WorkloadKey      string `json:"WorkloadKey,omitempty"`
	WorkloadInstance string `json:"WorkloadInstance,omitempty"`
	// ResetConsumedEventID marks the runtime-owner reset-eligibility event that
	// already resumed this paused obligation. It makes consumption idempotent
	// so a replayed owner event cannot wake the same run twice.
	ResetConsumedEventID string `json:"ResetConsumedEventID,omitempty"`
}

type runningEntry struct {
	EffortBinding        *effortauthority.Binding
	ProfileKey           string
	RunID                string
	State                string
	IdempotencyKey       string
	TaskID               string
	RunTag               string
	WorkloadKind         string
	WorkloadKey          string
	WorkloadInstance     string
	ResetConsumedEventID string
}

// DispatchIntent is the durable record of a heartbeat's owner start request,
// persisted before the request leaves the process. TaskID and RunTag let a
// retained uncertain obligation be replayed with the same owner request (and
// the same idempotency key) after a crash or lost response.
type DispatchIntent struct {
	IdempotencyKey   string
	TaskID           string
	RunTag           string
	WorkloadKind     string
	WorkloadKey      string
	WorkloadInstance string
}

// TeamExecutionContext manages execution for a single team according to queue policy.
type TeamExecutionContext struct {
	mu                sync.Mutex
	teamID            string
	queuePolicy       string
	maxConcurrentRuns int
	running           map[string]runningEntry // agentID -> {profileKey, runID}
	queue             []queuedExecution
	queued            map[string]bool
	executor          HeartbeatExecutor
	persistDir        string
	agentClient       AgentClient
	// resumer resumes a quota-paused (parked) owner run after the runtime owner
	// reports reset eligibility. It is the narrow pause/resume port and is
	// separate from AgentClient so consumption is qualified with synthetic
	// owner events before live integration.
	resumer PausedRunResumer
	// clock supplies "now" for reset-eligibility timing. Tests inject a fake
	// clock; production defaults to time.Now.
	clock func() time.Time
	// executionWG tracks the short-lived dispatch goroutines. Completion
	// waiters themselves are owned and drained by Executor.
	executionWG sync.WaitGroup
}

// newTeamExecutionContext creates a new TeamExecutionContext for a single team.
func newTeamExecutionContext(teamID string, executor HeartbeatExecutor, persistDir string, agentClient AgentClient) *TeamExecutionContext {
	return &TeamExecutionContext{
		teamID:            teamID,
		queuePolicy:       teamconfig.QueuePolicySerialized,
		maxConcurrentRuns: 1,
		running:           make(map[string]runningEntry),
		queue:             make([]queuedExecution, 0),
		queued:            make(map[string]bool),
		executor:          executor,
		persistDir:        persistDir,
		agentClient:       agentClient,
		clock:             time.Now,
	}
}

// SetPausedRunResumer installs the narrow runtime-owner pause/resume port.
func (c *TeamExecutionContext) SetPausedRunResumer(resumer PausedRunResumer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resumer = resumer
}

// SetClock overrides the clock used for reset-eligibility timing (tests only).
func (c *TeamExecutionContext) SetClock(now func() time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clock = now
}

func (c *TeamExecutionContext) Configure(queuePolicy string, maxConcurrentRuns int) {
	c.mu.Lock()
	c.queuePolicy = queuePolicy
	c.maxConcurrentRuns = maxConcurrentRuns
	dispatches := c.dispatchAvailableLocked()
	c.mu.Unlock()
	c.startExecutions(dispatches)
}

// Enqueue adds an agent to the team's execution queue. If capacity is available,
// execution starts immediately. Otherwise, the member is queued.
func (c *TeamExecutionContext) Enqueue(ctx context.Context, agentID, profileKey string) (*EnqueueResult, error) {
	worker := context.Background()
	manualAdmission := false
	if client, ok := c.agentClient.(*AgentManagerClient); ok {
		manualAdmission = true
		qualified, err := client.prepareCreateRunCaller(ctx)
		if err != nil {
			return nil, NewDispatchRejectedError(err)
		}
		carried, err := carryQualifiedCaller(qualified, context.Background())
		if err != nil {
			return nil, NewDispatchRejectedError(err)
		}
		proof := createRunCaller(ctx)
		worker = withCreateRunCaller(carried, proof.authorization, proof.runIdentity)
	}
	c.mu.Lock()
	if _, ok := c.running[agentID]; ok {
		c.mu.Unlock()
		return nil, &MemberAlreadyQueuedError{TeamID: c.teamID, AgentID: agentID}
	}
	if c.queued[agentID] {
		c.mu.Unlock()
		return nil, &MemberAlreadyQueuedError{TeamID: c.teamID, AgentID: agentID}
	}

	result := &EnqueueResult{
		TeamID:   c.teamID,
		AgentID:  agentID,
		Status:   "queued",
		Position: len(c.queue) + 1,
	}

	if len(c.running) < c.maxConcurrentRuns {
		c.running[agentID] = runningEntry{ProfileKey: profileKey, EffortBinding: qualifiedEffortBinding(worker)}
		if err := c.persistLocked(); err != nil && qualifiedEffortBinding(worker) != nil {
			delete(c.running, agentID)
			c.mu.Unlock()
			return nil, NewDispatchUncertainError(fmt.Errorf("finite queue save failed: %w", err))
		}
		c.mu.Unlock()
		c.startExecution(agentID, profileKey, worker)
		result.Status = "started"
		result.Position = 0
		return result, nil
	}

	c.queue = append(c.queue, queuedExecution{AgentID: agentID, ProfileKey: profileKey, caller: worker, ManualAdmission: manualAdmission, EffortBinding: qualifiedEffortBinding(worker)})
	c.queued[agentID] = true
	if err := c.persistLocked(); err != nil && qualifiedEffortBinding(worker) != nil {
		c.queue = c.queue[:len(c.queue)-1]
		delete(c.queued, agentID)
		c.mu.Unlock()
		return nil, NewDispatchUncertainError(fmt.Errorf("finite queue save failed: %w", err))
	}
	c.mu.Unlock()
	return result, nil
}

// OnMemberComplete is called when an execution finishes. It frees a slot and
// starts queued members until the current concurrency limit is full.
func (c *TeamExecutionContext) OnMemberComplete(agentID string) {
	c.mu.Lock()
	if _, ok := c.running[agentID]; !ok {
		c.mu.Unlock()
		return
	}
	delete(c.running, agentID)
	dispatches := c.dispatchAvailableLocked()
	c.mu.Unlock()
	c.startExecutions(dispatches)
}

// Status returns a snapshot of the team's execution state.
func (c *TeamExecutionContext) Status() TeamExecutionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := "idle"
	if len(c.running) > 0 {
		state = "active"
	}

	running := make([]string, 0, len(c.running))
	uncertain := make([]string, 0)
	unreachable := make([]string, 0)
	paused := make([]string, 0)
	for agentID, entry := range c.running {
		running = append(running, agentID)
		switch entry.State {
		case ObligationDispatchUncertain:
			uncertain = append(uncertain, agentID)
		case ObligationOwnerUnreachable:
			unreachable = append(unreachable, agentID)
		case ObligationPaused:
			paused = append(paused, agentID)
		}
	}
	sort.Strings(running)
	sort.Strings(uncertain)
	sort.Strings(unreachable)
	sort.Strings(paused)

	queue := make([]string, 0, len(c.queue))
	for _, item := range c.queue {
		queue = append(queue, item.AgentID)
	}

	return TeamExecutionStatus{
		TeamID:              c.teamID,
		State:               state,
		RunningAgentIDs:     running,
		Queue:               queue,
		QueuePolicy:         c.queuePolicy,
		MaxConcurrentRuns:   c.maxConcurrentRuns,
		UncertainAgentIDs:   uncertain,
		UnreachableAgentIDs: unreachable,
		PausedAgentIDs:      paused,
	}
}

type persistedTeamQueue struct {
	TeamID            string            `json:"teamId"`
	QueuePolicy       string            `json:"queuePolicy"`
	MaxConcurrentRuns int               `json:"maxConcurrentRuns"`
	Running           []queuedExecution `json:"running"`
	Queue             []queuedExecution `json:"queue"`
}

func (c *TeamExecutionContext) dispatchAvailableLocked() []queuedExecution {
	var dispatches []queuedExecution
	savedQueue := append([]queuedExecution(nil), c.queue...)
	savedRunning := make(map[string]runningEntry, len(c.running))
	for k, v := range c.running {
		savedRunning[k] = v
	}
	savedQueued := make(map[string]bool, len(c.queued))
	for k, v := range c.queued {
		savedQueued[k] = v
	}
	// A bounded scan keeps expired/recovered entries held without blocking other
	// callers. Their proof is never renewed from a different queued request.
	scans := len(c.queue)
	for len(c.running) < c.maxConcurrentRuns && len(c.queue) > 0 && scans > 0 {
		scans--
		next := c.queue[0]
		c.queue = c.queue[1:]
		if _, real := c.agentClient.(*AgentManagerClient); real {
			if next.caller == nil || requireQualifiedCaller(next.caller) != nil {
				next.State = "caller_required"
				c.queue = append(c.queue, next)
				continue
			}
		}
		delete(c.queued, next.AgentID)
		c.running[next.AgentID] = runningEntry{ProfileKey: next.ProfileKey, EffortBinding: next.EffortBinding}
		dispatches = append(dispatches, next)
	}
	if err := c.persistLocked(); err != nil {
		for _, d := range dispatches {
			if d.EffortBinding != nil {
				c.queue, c.running, c.queued = savedQueue, savedRunning, savedQueued
				log.Printf("team_execution: finite dispatch held because queue save failed: %v", err)
				return nil
			}
		}
	}
	return dispatches
}

func (c *TeamExecutionContext) startExecutions(dispatches []queuedExecution) {
	for _, next := range dispatches {
		c.startExecution(next.AgentID, next.ProfileKey, next.caller)
	}
}

func (c *TeamExecutionContext) startExecution(agentID, profileKey string, caller context.Context) {
	c.executionWG.Add(1)
	go func() {
		defer c.executionWG.Done()
		ctx := caller
		if ctx == nil {
			ctx = context.Background()
		}
		result, err := c.executor.Execute(ctx, c.teamID, agentID, profileKey)
		if err != nil {
			if IsDispatchUncertain(err) {
				// The owner may have accepted the run. Retain the slot as an
				// uncertain obligation instead of freeing it for a duplicate.
				log.Printf("team_execution: retaining uncertain dispatch obligation for %s/%s: %v", c.teamID, agentID, err)
				c.MarkDispatchUncertain(agentID, err.Error())
				return
			}
			log.Printf("team_execution: execution failed for %s/%s: %v", c.teamID, agentID, err)
			c.OnMemberComplete(agentID)
			return
		}
		log.Printf("team_execution: execution started for %s/%s, run ID: %s", c.teamID, agentID, result.RunID)
	}()
}

// Shutdown waits for dispatch goroutines to finish their durable state update.
func (c *TeamExecutionContext) Shutdown() {
	c.executionWG.Wait()
}

// BeginDispatch records the durable dispatch intent (including the owner
// idempotency key and the owner request essentials) for a member before the
// start request leaves the process. A crash between here and the response
// recovers as a visible, replayable obligation.
func (c *TeamExecutionContext) BeginDispatch(agentID string, intent DispatchIntent) {
	_ = c.BeginDispatchDurable(agentID, intent)
}
func (c *TeamExecutionContext) BeginDispatchDurable(agentID string, intent DispatchIntent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		return ErrRunningEntryNotFound
	}
	entry.State = ObligationDispatchUncertain
	entry.IdempotencyKey = intent.IdempotencyKey
	entry.TaskID = intent.TaskID
	entry.RunTag = intent.RunTag
	entry.WorkloadKind = intent.WorkloadKind
	entry.WorkloadKey = intent.WorkloadKey
	entry.WorkloadInstance = intent.WorkloadInstance
	c.running[agentID] = entry
	return c.persistLocked()
}

// ReconcileDispatch resolves a retained dispatch_uncertain obligation by
// replaying the recorded owner start request with its idempotency key. The
// owner deduplicates on that key, so a lost start response converges on the
// one run it accepted; if the request never reached the owner, the replay
// creates the intended run for the already-created task.
//
// The outcome is owner-mediated and therefore definite for reconciliation:
//   - success            -> bind the confirmed run and re-attach a completion waiter
//   - uncertain error    -> retain the obligation (owner still unreachable)
//   - definitive refusal -> no run will ever exist for this intent; release the slot
//
// It returns ErrRunningEntryNotFound when no entry exists and nil when the
// entry is not an uncertain dispatch (nothing to reconcile).
func (c *TeamExecutionContext) ReconcileDispatch(ctx context.Context, agentID string) error {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	if !ok {
		c.mu.Unlock()
		return ErrRunningEntryNotFound
	}
	if entry.State != ObligationDispatchUncertain {
		c.mu.Unlock()
		return nil
	}
	if entry.IdempotencyKey == "" || entry.TaskID == "" {
		c.mu.Unlock()
		return fmt.Errorf("cannot replay %s/%s dispatch: recorded intent is incomplete", c.teamID, agentID)
	}
	if c.agentClient == nil {
		c.mu.Unlock()
		return fmt.Errorf("cannot replay %s/%s dispatch: agent client is not configured", c.teamID, agentID)
	}
	if client, ok := c.agentClient.(*AgentManagerClient); ok && entry.EffortBinding != nil {
		if client.efforts == nil {
			c.mu.Unlock()
			return effortauthority.ErrRefused
		}
		restored, e := client.efforts.Restore(ctx, *entry.EffortBinding, c.teamID, agentID, entry.ProfileKey)
		if e != nil {
			c.mu.Unlock()
			return e
		}
		ctx = restored
	}
	profileKey := entry.ProfileKey
	intent := DispatchIntent{
		IdempotencyKey:   entry.IdempotencyKey,
		TaskID:           entry.TaskID,
		RunTag:           entry.RunTag,
		WorkloadKind:     entry.WorkloadKind,
		WorkloadKey:      entry.WorkloadKey,
		WorkloadInstance: entry.WorkloadInstance,
	}
	c.mu.Unlock()

	attribKey, attribValue := buildHeartbeatAttributionEnv(c.teamID, agentID)
	environment := map[string]string{attribKey: attribValue}
	addWorkloadEnvironment(environment, intent.WorkloadKind, intent.WorkloadKey, intent.WorkloadInstance)
	req := &CreateRunRequest{
		TaskID:         intent.TaskID,
		ProfileRef:     &ProfileRef{ProfileKey: profileKey},
		Tag:            &intent.RunTag,
		IdempotencyKey: intent.IdempotencyKey,
		Environment:    environment,
	}

	run, err := c.agentClient.CreateRun(ctx, req)
	if err != nil {
		if IsDispatchUncertain(err) {
			c.MarkDispatchUncertain(agentID, err.Error())
			return err
		}
		if entry.EffortBinding != nil {
			return err
		}
		// Definitive owner refusal: the intent can never become a run.
		log.Printf("team_execution: releasing dispatch obligation for %s/%s after definitive replay refusal: %v", c.teamID, agentID, err)
		c.releaseObligation(agentID)
		return err
	}
	if run == nil || run.ID == "" {
		reason := "owner accepted replay but returned no run identity"
		c.MarkDispatchUncertain(agentID, reason)
		return fmt.Errorf("%s/%s: %s", c.teamID, agentID, reason)
	}

	c.SetRunningRunID(agentID, run.ID)
	c.watchRetainedRun(agentID, run.ID, profileKey, intent.TaskID, intent.RunTag)
	log.Printf("team_execution: resolved uncertain dispatch for %s/%s to run %s", c.teamID, agentID, run.ID)
	return nil
}

// releaseObligation drops a member's retained slot and admits any member that
// was waiting on capacity. It mirrors OnMemberComplete for the definite-refusal
// path of a replayed dispatch.
func (c *TeamExecutionContext) releaseObligation(agentID string) {
	c.mu.Lock()
	if _, ok := c.running[agentID]; !ok {
		c.mu.Unlock()
		return
	}
	delete(c.running, agentID)
	dispatches := c.dispatchAvailableLocked()
	c.mu.Unlock()
	c.startExecutions(dispatches)
}

// retainedRunWatcher is implemented by an executor that can re-attach a
// completion waiter to a run it did not create in this process. A recovered
// obligation bound during Recover otherwise has no goroutine to release its
// slot when the owner reports terminal state.
type retainedRunWatcher interface {
	WatchRecoveredRun(teamID, agentID, runID, profileKey, taskID, runTag string)
}

func (c *TeamExecutionContext) watchRetainedRun(agentID, runID, profileKey, taskID, runTag string) {
	watcher, ok := c.executor.(retainedRunWatcher)
	if !ok || watcher == nil {
		return
	}
	watcher.WatchRecoveredRun(c.teamID, agentID, runID, profileKey, taskID, runTag)
}

// MarkDispatchUncertain retains a member's slot after a dispatch whose outcome
// is unknown. It is idempotent and never releases the obligation.
func (c *TeamExecutionContext) MarkDispatchUncertain(agentID, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		log.Printf("team_execution: MarkDispatchUncertain called for %s/%s not in running map", c.teamID, agentID)
		return
	}
	entry.State = ObligationDispatchUncertain
	c.running[agentID] = entry
	c.persistLocked()
}

// SetRunningRunID records the agent-manager run ID for an in-flight running
// entry. Called by Executor.Execute immediately after CreateRun returns so
// that subsequent Recover() calls can reconcile the entry against
// agent-manager. No-op (with a debug log) if the agent is not currently
// recorded as running — bookkeeping must not affect run progress.
func (c *TeamExecutionContext) SetRunningRunID(agentID, runID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		log.Printf("team_execution: SetRunningRunID called for %s/%s not in running map", c.teamID, agentID)
		return
	}
	entry.RunID = runID
	// Binding a confirmed run identity resolves any prior uncertainty.
	entry.State = ObligationRunning
	c.running[agentID] = entry
	c.persistLocked()
}

// Recover loads persisted queue state from disk and reconciles each running
// entry against agent-manager. Ownership is released only on definitive
// terminal evidence from the owner; every other outcome is retained as a
// visible, recoverable obligation:
//
//   - terminal run                 -> release the slot
//   - active run                   -> retain as running
//   - parked/paused run            -> retain as paused (nonterminal)
//   - empty RunID                  -> retain as dispatch_uncertain
//   - GetRun error (owner down)    -> retain as owner_unreachable
//   - run not found by owner       -> retain as dispatch_uncertain
//
// This deliberately reverses the previous conservative-drop behavior. Dropping
// an unresolved entry after an outage allowed the next tick to admit a
// duplicate heartbeat, recreating the stuck-running ambiguity in a worse form.
// The retained obligation blocks Enqueue for that member until an operator
// resolves it (ClearRunning) or the owner reports definitive terminal state.
//
// Stale queued (not-yet-dispatched) ticks are intentionally not restored: the
// cron schedule is the source of truth after a restart, and replaying the
// backlog would start a catch-up queue the acceptance forbids.
func (c *TeamExecutionContext) Recover(ctx context.Context) {
	c.mu.Lock()

	data, err := os.ReadFile(c.queueFilePath())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("team_execution: failed to read queue for %s: %v", c.teamID, err)
		}
		c.mu.Unlock()
		return
	}

	var persisted persistedTeamQueue
	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("team_execution: failed to unmarshal queue for %s: %v", c.teamID, err)
		c.mu.Unlock()
		return
	}

	if persisted.QueuePolicy != "" {
		c.queuePolicy = persisted.QueuePolicy
	}
	if persisted.MaxConcurrentRuns > 0 {
		c.maxConcurrentRuns = persisted.MaxConcurrentRuns
	}

	c.running = make(map[string]runningEntry, len(persisted.Running))
	var replayable []string
	for _, item := range persisted.Running {
		entry, keep := c.reconcileRunningEntry(ctx, item)
		if !keep {
			continue
		}
		c.running[item.AgentID] = entry
		// A recorded uncertain dispatch with a complete owner intent can be
		// resolved by replaying the idempotent start request. Doing so here
		// (after the lock is released) converges a lost response on the one
		// run the owner accepted instead of waiting for an operator.
		if entry.State == ObligationDispatchUncertain && entry.IdempotencyKey != "" && entry.TaskID != "" {
			replayable = append(replayable, item.AgentID)
		}
	}

	c.queue = make([]queuedExecution, 0)
	c.queued = make(map[string]bool)
	for _, item := range persisted.Queue {
		if !item.ManualAdmission {
			continue
		} // stale scheduled ticks remain discarded
		item.caller = nil
		item.State = "caller_required"
		if client, ok := c.agentClient.(*AgentManagerClient); ok && client.efforts != nil && item.EffortBinding != nil {
			item.caller, _ = client.efforts.Restore(ctx, *item.EffortBinding, c.teamID, item.AgentID, item.ProfileKey)
			if item.caller != nil {
				item.State = ""
			}
		}
		c.queue = append(c.queue, item)
		c.queued[item.AgentID] = true
	}

	// Write cleaned/reconciled state back to disk before any replay.
	saveErr := c.persistLocked()
	c.mu.Unlock()
	if saveErr != nil {
		log.Printf("team_execution: recovery held because queue save failed: %v", saveErr)
		return
	}

	for _, agentID := range replayable {
		if err := c.ReconcileDispatch(ctx, agentID); err != nil {
			// Retained uncertainty is not a recovery failure: the obligation
			// stays visible and the owner will be reconciled next time.
			log.Printf("team_execution: uncertain dispatch for %s/%s not resolved on recovery: %v", c.teamID, agentID, err)
		}
	}
}

// reconcileRunningEntry decides the durable obligation for one persisted
// running entry. keep is false only on definitive terminal owner evidence.
func (c *TeamExecutionContext) reconcileRunningEntry(ctx context.Context, item queuedExecution) (runningEntry, bool) {
	base := runningEntry{
		ProfileKey:           item.ProfileKey,
		EffortBinding:        item.EffortBinding,
		RunID:                item.RunID,
		IdempotencyKey:       item.IdempotencyKey,
		TaskID:               item.TaskID,
		RunTag:               item.RunTag,
		WorkloadKind:         item.WorkloadKind,
		WorkloadKey:          item.WorkloadKey,
		WorkloadInstance:     item.WorkloadInstance,
		ResetConsumedEventID: item.ResetConsumedEventID,
	}
	// A prior recovery already recorded a non-resolvable obligation. Preserve it
	// without re-probing the owner: unknown reset waits must not poll.
	if item.State != ObligationRunning {
		base.State = item.State
		return base, true
	}
	if item.RunID == "" {
		log.Printf("team_execution: retaining dispatch_uncertain obligation for %s/%s (no RunID)", c.teamID, item.AgentID)
		base.State = ObligationDispatchUncertain
		return base, true
	}
	if c.agentClient == nil {
		// No client configured (e.g. unit tests not exercising reconciliation).
		// Preserve the entry — caller takes responsibility for what's on disk.
		return base, true
	}
	run, err := c.agentClient.GetRun(ctx, item.RunID)
	if err != nil {
		log.Printf("team_execution: retaining owner_unreachable obligation for %s/%s (run %s: %v)", c.teamID, item.AgentID, item.RunID, err)
		base.State = ObligationOwnerUnreachable
		return base, true
	}
	if run == nil {
		log.Printf("team_execution: retaining dispatch_uncertain obligation for %s/%s (run %s not found by owner)", c.teamID, item.AgentID, item.RunID)
		base.State = ObligationDispatchUncertain
		return base, true
	}
	if IsTerminalStatus(run.Status) {
		log.Printf("team_execution: releasing terminal obligation for %s/%s (run %s status=%s)", c.teamID, item.AgentID, item.RunID, run.Status)
		return runningEntry{}, false
	}
	if IsParkedStatus(run.Status) {
		base.State = ObligationPaused
	}
	return base, true
}

// ClearRunning removes a single running entry for the team. If the entry
// has a RunID and the backing run is still active in agent-manager, returns
// an error unless force is true. If the agent isn't recorded as running,
// returns ErrRunningEntryNotFound.
func (c *TeamExecutionContext) ClearRunning(ctx context.Context, agentID string, force bool) error {
	c.mu.Lock()

	entry, ok := c.running[agentID]
	if !ok {
		c.mu.Unlock()
		return ErrRunningEntryNotFound
	}

	if !force && entry.RunID != "" && c.agentClient != nil {
		// Release the lock for the network call.
		c.mu.Unlock()
		run, err := c.agentClient.GetRun(ctx, entry.RunID)
		if err == nil && run != nil && !IsTerminalStatus(run.Status) {
			return &RunningStillActiveError{TeamID: c.teamID, AgentID: agentID, RunID: entry.RunID, Status: run.Status}
		}
		c.mu.Lock()
	}

	delete(c.running, agentID)
	dispatches := c.dispatchAvailableLocked()
	c.mu.Unlock()
	c.startExecutions(dispatches)
	return nil
}

func (c *TeamExecutionContext) persistLocked() error {
	running := make([]queuedExecution, 0, len(c.running))
	for agentID, entry := range c.running {
		running = append(running, queuedExecution{
			AgentID:              agentID,
			ProfileKey:           entry.ProfileKey,
			EffortBinding:        entry.EffortBinding,
			RunID:                entry.RunID,
			State:                entry.State,
			IdempotencyKey:       entry.IdempotencyKey,
			TaskID:               entry.TaskID,
			RunTag:               entry.RunTag,
			WorkloadKind:         entry.WorkloadKind,
			WorkloadKey:          entry.WorkloadKey,
			WorkloadInstance:     entry.WorkloadInstance,
			ResetConsumedEventID: entry.ResetConsumedEventID,
		})
	}
	sort.Slice(running, func(i, j int) bool {
		return running[i].AgentID < running[j].AgentID
	})

	data := persistedTeamQueue{
		TeamID:            c.teamID,
		QueuePolicy:       c.queuePolicy,
		MaxConcurrentRuns: c.maxConcurrentRuns,
		Running:           running,
		Queue:             append([]queuedExecution(nil), c.queue...),
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	filePath := c.queueFilePath()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	// Queue obligations must survive a process crash before native dispatch.
	f, err := os.CreateTemp(filepath.Dir(filePath), ".team-queue-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(bytes); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, filePath); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(filePath))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (c *TeamExecutionContext) queueFilePath() string {
	return filepath.Join(c.persistDir, fmt.Sprintf("team-queue-%s.json", c.teamID))
}

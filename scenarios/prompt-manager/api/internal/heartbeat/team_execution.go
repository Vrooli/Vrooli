package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

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

type queuedExecution struct {
	AgentID    string `json:"AgentID"`
	ProfileKey string `json:"ProfileKey"`
	// RunID is the agent-manager run ID for entries in the running slice.
	// Queued entries (not yet dispatched) carry an empty RunID; it is
	// populated by Executor.Execute via SetRunningRunID once CreateRun
	// returns. Recover uses RunID to reconcile against agent-manager.
	RunID string `json:"RunID,omitempty"`
}

type runningEntry struct {
	ProfileKey string
	RunID      string
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
	}
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
func (c *TeamExecutionContext) Enqueue(_ context.Context, agentID, profileKey string) (*EnqueueResult, error) {
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
		c.running[agentID] = runningEntry{ProfileKey: profileKey}
		c.persistLocked()
		c.mu.Unlock()
		c.startExecution(agentID, profileKey)
		result.Status = "started"
		result.Position = 0
		return result, nil
	}

	c.queue = append(c.queue, queuedExecution{AgentID: agentID, ProfileKey: profileKey})
	c.queued[agentID] = true
	c.persistLocked()
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
	for agentID := range c.running {
		running = append(running, agentID)
	}
	sort.Strings(running)

	queue := make([]string, 0, len(c.queue))
	for _, item := range c.queue {
		queue = append(queue, item.AgentID)
	}

	return TeamExecutionStatus{
		TeamID:            c.teamID,
		State:             state,
		RunningAgentIDs:   running,
		Queue:             queue,
		QueuePolicy:       c.queuePolicy,
		MaxConcurrentRuns: c.maxConcurrentRuns,
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
	for len(c.running) < c.maxConcurrentRuns && len(c.queue) > 0 {
		next := c.queue[0]
		c.queue = c.queue[1:]
		delete(c.queued, next.AgentID)
		c.running[next.AgentID] = runningEntry{ProfileKey: next.ProfileKey}
		dispatches = append(dispatches, next)
	}
	c.persistLocked()
	return dispatches
}

func (c *TeamExecutionContext) startExecutions(dispatches []queuedExecution) {
	for _, next := range dispatches {
		c.startExecution(next.AgentID, next.ProfileKey)
	}
}

func (c *TeamExecutionContext) startExecution(agentID, profileKey string) {
	go func() {
		result, err := c.executor.Execute(context.Background(), c.teamID, agentID, profileKey)
		if err != nil {
			log.Printf("team_execution: execution failed for %s/%s: %v", c.teamID, agentID, err)
			c.OnMemberComplete(agentID)
			return
		}
		log.Printf("team_execution: execution started for %s/%s, run ID: %s", c.teamID, agentID, result.RunID)
	}()
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
	c.running[agentID] = entry
	c.persistLocked()
}

// Recover loads persisted queue state from disk and reconciles each running
// entry against agent-manager. Entries whose RunID is empty, whose run no
// longer exists in agent-manager, whose run has reached a terminal status,
// or whose status cannot be determined (network error) are dropped. The
// cleaned state is persisted back to disk.
//
// Mirrors RunRegistry.Recover. Conservative-drop on transient error is
// intentional: re-creating the original stuck-running bug is a worse failure
// than briefly losing visibility on a run that may still be alive (the run
// itself continues; the next heartbeat trigger simply succeeds).
func (c *TeamExecutionContext) Recover(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.queueFilePath())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("team_execution: failed to read queue for %s: %v", c.teamID, err)
		}
		return
	}

	var persisted persistedTeamQueue
	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("team_execution: failed to unmarshal queue for %s: %v", c.teamID, err)
		return
	}

	if persisted.QueuePolicy != "" {
		c.queuePolicy = persisted.QueuePolicy
	}
	if persisted.MaxConcurrentRuns > 0 {
		c.maxConcurrentRuns = persisted.MaxConcurrentRuns
	}

	c.running = make(map[string]runningEntry, len(persisted.Running))
	for _, item := range persisted.Running {
		if item.RunID == "" {
			log.Printf("team_execution: dropping stale running entry %s/%s (no RunID)", c.teamID, item.AgentID)
			continue
		}
		if c.agentClient == nil {
			// No client configured (e.g. unit tests not exercising
			// reconciliation). Preserve the entry — caller takes
			// responsibility for what's on disk.
			c.running[item.AgentID] = runningEntry{ProfileKey: item.ProfileKey, RunID: item.RunID}
			continue
		}
		run, err := c.agentClient.GetRun(ctx, item.RunID)
		if err != nil {
			log.Printf("team_execution: dropping running entry %s/%s (run %s GetRun error: %v)", c.teamID, item.AgentID, item.RunID, err)
			continue
		}
		if run == nil {
			log.Printf("team_execution: dropping running entry %s/%s (run %s not found)", c.teamID, item.AgentID, item.RunID)
			continue
		}
		if IsTerminalStatus(run.Status) {
			log.Printf("team_execution: dropping running entry %s/%s (run %s terminal status=%s)", c.teamID, item.AgentID, item.RunID, run.Status)
			continue
		}
		c.running[item.AgentID] = runningEntry{ProfileKey: item.ProfileKey, RunID: item.RunID}
	}

	c.queue = persisted.Queue
	if c.queue == nil {
		c.queue = make([]queuedExecution, 0)
	}
	c.queued = make(map[string]bool, len(c.queue))
	for _, item := range c.queue {
		c.queued[item.AgentID] = true
	}

	// Write cleaned state back to disk.
	c.persistLocked()
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

func (c *TeamExecutionContext) persistLocked() {
	running := make([]queuedExecution, 0, len(c.running))
	for agentID, entry := range c.running {
		running = append(running, queuedExecution{
			AgentID:    agentID,
			ProfileKey: entry.ProfileKey,
			RunID:      entry.RunID,
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
		log.Printf("team_execution: failed to marshal queue for %s: %v", c.teamID, err)
		return
	}

	filePath := c.queueFilePath()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		log.Printf("team_execution: failed to create dir for %s: %v", c.teamID, err)
		return
	}

	if err := os.WriteFile(filePath, bytes, 0o644); err != nil {
		log.Printf("team_execution: failed to write queue for %s: %v", c.teamID, err)
	}
}

func (c *TeamExecutionContext) queueFilePath() string {
	return filepath.Join(c.persistDir, fmt.Sprintf("team-queue-%s.json", c.teamID))
}

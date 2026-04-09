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

	"prompt-manager/teamconfig"
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

type queuedExecution struct {
	AgentID    string
	ProfileKey string
}

// TeamExecutionContext manages execution for a single team according to queue policy.
type TeamExecutionContext struct {
	mu                sync.Mutex
	teamID            string
	queuePolicy       string
	maxConcurrentRuns int
	running           map[string]string // agentID -> profileKey
	queue             []queuedExecution
	queued            map[string]bool
	executor          HeartbeatExecutor
	persistDir        string
}

// newTeamExecutionContext creates a new TeamExecutionContext for a single team.
func newTeamExecutionContext(teamID string, executor HeartbeatExecutor, persistDir string) *TeamExecutionContext {
	return &TeamExecutionContext{
		teamID:            teamID,
		queuePolicy:       teamconfig.QueuePolicySerialized,
		maxConcurrentRuns: 1,
		running:           make(map[string]string),
		queue:             make([]queuedExecution, 0),
		queued:            make(map[string]bool),
		executor:          executor,
		persistDir:        persistDir,
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
		c.running[agentID] = profileKey
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
	TeamID            string                     `json:"teamId"`
	QueuePolicy       string                     `json:"queuePolicy"`
	MaxConcurrentRuns int                        `json:"maxConcurrentRuns"`
	Running           []queuedExecution          `json:"running"`
	Queue             []queuedExecution          `json:"queue"`
}

func (c *TeamExecutionContext) dispatchAvailableLocked() []queuedExecution {
	var dispatches []queuedExecution
	for len(c.running) < c.maxConcurrentRuns && len(c.queue) > 0 {
		next := c.queue[0]
		c.queue = c.queue[1:]
		delete(c.queued, next.AgentID)
		c.running[next.AgentID] = next.ProfileKey
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

// Recover loads persisted queue state from disk.
func (c *TeamExecutionContext) Recover() {
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

	c.running = make(map[string]string, len(persisted.Running))
	for _, item := range persisted.Running {
		c.running[item.AgentID] = item.ProfileKey
	}
	c.queue = persisted.Queue
	if c.queue == nil {
		c.queue = make([]queuedExecution, 0)
	}
	c.queued = make(map[string]bool, len(c.queue))
	for _, item := range c.queue {
		c.queued[item.AgentID] = true
	}
}

func (c *TeamExecutionContext) persistLocked() {
	running := make([]queuedExecution, 0, len(c.running))
	for agentID, profileKey := range c.running {
		running = append(running, queuedExecution{AgentID: agentID, ProfileKey: profileKey})
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

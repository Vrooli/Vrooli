package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
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
	TeamID  string   `json:"teamId"`
	State   string   `json:"state"` // "idle" | "active"
	Running *string  `json:"running,omitempty"`
	Queue   []string `json:"queue"`
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

// TeamExecutionContext manages serialized execution for a single team.
type TeamExecutionContext struct {
	mu         sync.Mutex
	teamID     string
	running    *string           // agentID currently running, nil if idle
	queue      []string          // FIFO queue of agentIDs waiting
	queued     map[string]bool   // fast dedup lookup
	profiles   map[string]string // agentID -> profileKey for queued entries
	executor   HeartbeatExecutor
	persistDir string
}

// newTeamExecutionContext creates a new TeamExecutionContext for a single team.
func newTeamExecutionContext(teamID string, executor HeartbeatExecutor, persistDir string) *TeamExecutionContext {
	return &TeamExecutionContext{
		teamID:     teamID,
		queue:      make([]string, 0),
		queued:     make(map[string]bool),
		profiles:   make(map[string]string),
		executor:   executor,
		persistDir: persistDir,
	}
}

// Enqueue adds an agent to the team's execution queue. If the team is idle,
// execution starts immediately. If the agent is already queued or running,
// a MemberAlreadyQueuedError is returned.
func (c *TeamExecutionContext) Enqueue(ctx context.Context, agentID, profileKey string) (*EnqueueResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already running
	if c.running != nil && *c.running == agentID {
		return nil, &MemberAlreadyQueuedError{TeamID: c.teamID, AgentID: agentID}
	}

	// Check if already queued
	if c.queued[agentID] {
		return nil, &MemberAlreadyQueuedError{TeamID: c.teamID, AgentID: agentID}
	}

	// If idle, start immediately
	if c.running == nil {
		c.running = &agentID
		c.persist()

		// Start execution (don't hold lock during execution)
		// Use context.Background() because the goroutine outlives the HTTP
		// request that created ctx — by the time Execute runs, the request
		// handler has already written the 202 response and ctx is cancelled.
		go func() {
			result, err := c.executor.Execute(context.Background(), c.teamID, agentID, profileKey)
			if err != nil {
				log.Printf("team_execution: execution failed for %s/%s: %v", c.teamID, agentID, err)
				// Clear running state so the team isn't permanently stuck.
				// When Execute succeeds, OnMemberComplete is called later via
				// executor.OnComplete from waitForCompletion.
				c.OnMemberComplete(agentID)
			} else {
				log.Printf("team_execution: execution started for %s/%s, run ID: %s", c.teamID, agentID, result.RunID)
			}
		}()

		return &EnqueueResult{
			TeamID:   c.teamID,
			AgentID:  agentID,
			Status:   "started",
			Position: 0,
		}, nil
	}

	// Active: queue the agent
	c.queue = append(c.queue, agentID)
	c.queued[agentID] = true
	c.profiles[agentID] = profileKey
	position := len(c.queue)
	c.persist()

	return &EnqueueResult{
		TeamID:   c.teamID,
		AgentID:  agentID,
		Status:   "queued",
		Position: position,
	}, nil
}

// OnMemberComplete is called when an execution finishes. It pops the next
// agent from the queue and starts it, or sets the team to idle.
func (c *TeamExecutionContext) OnMemberComplete(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only clear running if it matches the completed agent
	if c.running == nil || *c.running != agentID {
		return
	}

	if len(c.queue) > 0 {
		// Pop next from queue
		nextAgent := c.queue[0]
		c.queue = c.queue[1:]
		delete(c.queued, nextAgent)
		profileKey := c.profiles[nextAgent]
		delete(c.profiles, nextAgent)

		c.running = &nextAgent
		c.persist()

		go func() {
			ctx := context.Background()
			result, err := c.executor.Execute(ctx, c.teamID, nextAgent, profileKey)
			if err != nil {
				log.Printf("team_execution: queued execution failed for %s/%s: %v", c.teamID, nextAgent, err)
				c.OnMemberComplete(nextAgent)
			} else {
				log.Printf("team_execution: queued execution started for %s/%s, run ID: %s", c.teamID, nextAgent, result.RunID)
			}
		}()
	} else {
		c.running = nil
		c.persist()
	}
}

// Status returns a snapshot of the team's execution state.
func (c *TeamExecutionContext) Status() TeamExecutionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := "idle"
	if c.running != nil {
		state = "active"
	}

	queue := make([]string, len(c.queue))
	copy(queue, c.queue)

	return TeamExecutionStatus{
		TeamID:  c.teamID,
		State:   state,
		Running: c.running,
		Queue:   queue,
	}
}

// persistedTeamQueue is the on-disk representation of a team's execution queue.
type persistedTeamQueue struct {
	TeamID   string            `json:"teamId"`
	Running  *string           `json:"running,omitempty"`
	Queue    []string          `json:"queue"`
	Profiles map[string]string `json:"profiles,omitempty"`
}

// persist writes the current queue state to disk. Caller must hold mu.
func (c *TeamExecutionContext) persist() {
	data := persistedTeamQueue{
		TeamID:   c.teamID,
		Running:  c.running,
		Queue:    c.queue,
		Profiles: c.profiles,
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

	c.running = persisted.Running
	c.queue = persisted.Queue
	if c.queue == nil {
		c.queue = make([]string, 0)
	}
	c.queued = make(map[string]bool, len(c.queue))
	for _, id := range c.queue {
		c.queued[id] = true
	}
	c.profiles = persisted.Profiles
	if c.profiles == nil {
		c.profiles = make(map[string]string)
	}

	if c.running != nil || len(c.queue) > 0 {
		log.Printf("team_execution: recovered queue for %s (running=%v, queued=%d)", c.teamID, c.running, len(c.queue))
	}
}

// queueFilePath returns the path for this team's persisted queue file.
func (c *TeamExecutionContext) queueFilePath() string {
	return filepath.Join(c.persistDir, fmt.Sprintf("team-queue-%s.json", c.teamID))
}

package heartbeat

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ActiveRun tracks a currently executing heartbeat run.
type ActiveRun struct {
	TeamID    string             `json:"teamId"`
	AgentID   string             `json:"agentId"`
	RunID     string             `json:"runId"`
	StartedAt time.Time          `json:"startedAt"`
	CancelFn  context.CancelFunc `json:"-"`
}

// RunRegistry is an in-memory registry of active heartbeat runs with disk
// persistence for restart recovery.
type RunRegistry struct {
	mu       sync.RWMutex
	active   map[string]*ActiveRun // key: "teamID/agentID"
	filePath string
}

// NewRunRegistry creates a new run registry that persists state to the given directory.
func NewRunRegistry(persistDir string) *RunRegistry {
	return &RunRegistry{
		active:   make(map[string]*ActiveRun),
		filePath: filepath.Join(persistDir, "heartbeat-active-runs.json"),
	}
}

func registryKey(teamID, agentID string) string {
	return teamID + "/" + agentID
}

// Register adds an active run to the registry.
func (r *RunRegistry) Register(teamID, agentID, runID string, startedAt time.Time, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.active[registryKey(teamID, agentID)] = &ActiveRun{
		TeamID:    teamID,
		AgentID:   agentID,
		RunID:     runID,
		StartedAt: startedAt,
		CancelFn:  cancel,
	}
	r.persistLocked()
}

// Unregister removes an active run from the registry. Safe to call if key is not found.
func (r *RunRegistry) Unregister(teamID, agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.active, registryKey(teamID, agentID))
	r.persistLocked()
}

// ListActive returns a snapshot of all active runs.
func (r *RunRegistry) ListActive() []ActiveRun {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runs := make([]ActiveRun, 0, len(r.active))
	for _, run := range r.active {
		runs = append(runs, *run)
	}
	return runs
}

// GetActiveRun looks up a single active run.
func (r *RunRegistry) GetActiveRun(teamID, agentID string) (*ActiveRun, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, ok := r.active[registryKey(teamID, agentID)]
	if !ok {
		return nil, false
	}
	copy := *run
	return &copy, true
}

// Count returns the number of active runs.
func (r *RunRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.active)
}

// persistedRun is the on-disk representation (no CancelFn).
type persistedRun struct {
	TeamID    string    `json:"teamId"`
	AgentID   string    `json:"agentId"`
	RunID     string    `json:"runId"`
	StartedAt time.Time `json:"startedAt"`
}

// persistLocked writes the current active map to disk. Caller must hold mu.
func (r *RunRegistry) persistLocked() {
	runs := make([]persistedRun, 0, len(r.active))
	for _, run := range r.active {
		runs = append(runs, persistedRun{
			TeamID:    run.TeamID,
			AgentID:   run.AgentID,
			RunID:     run.RunID,
			StartedAt: run.StartedAt,
		})
	}

	data, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		log.Printf("run_registry: failed to marshal: %v", err)
		return
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		log.Printf("run_registry: failed to create dir: %v", err)
		return
	}

	if err := os.WriteFile(r.filePath, data, 0o644); err != nil {
		log.Printf("run_registry: failed to write: %v", err)
	}
}

// Recover loads persisted runs from disk and checks each against agent-manager.
// Terminal runs are removed; active ones are kept (without CancelFn since the
// goroutine is gone after a restart).
func (r *RunRegistry) Recover(ctx context.Context, client *AgentManagerClient) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("run_registry: failed to read persisted runs: %v", err)
		}
		return
	}

	var runs []persistedRun
	if err := json.Unmarshal(data, &runs); err != nil {
		log.Printf("run_registry: failed to unmarshal persisted runs: %v", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, pr := range runs {
		// Check run status with agent-manager
		run, err := client.GetRun(ctx, pr.RunID)
		if err != nil {
			log.Printf("run_registry: failed to check run %s: %v", pr.RunID, err)
			continue
		}

		// If run is nil or terminal, skip it
		if run == nil {
			continue
		}
		switch run.Status {
		case "RUN_STATUS_COMPLETE", "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED",
			"complete", "failed", "cancelled":
			continue
		}

		// Still active — add to registry without CancelFn
		r.active[registryKey(pr.TeamID, pr.AgentID)] = &ActiveRun{
			TeamID:    pr.TeamID,
			AgentID:   pr.AgentID,
			RunID:     pr.RunID,
			StartedAt: pr.StartedAt,
		}
	}

	r.persistLocked()
	if len(r.active) > 0 {
		log.Printf("run_registry: recovered %d active run(s)", len(r.active))
	}
}

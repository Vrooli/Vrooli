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
	// State carries a durable obligation classification across restart. Empty
	// for ordinary live runs; see the Obligation* constants.
	State               string `json:"state,omitempty"`
	ownerStartedAtKnown bool
}

// RunObserver receives run lifecycle notifications from the registry. The
// world feed is the consumer; the registry never blocks on it.
type RunObserver interface {
	RunStarted(run ActiveRun)
	// RunEnded reports the outcome the registry knows: failed with a message,
	// or ended cleanly (message "stopped" when an operator stopped it).
	RunEnded(run ActiveRun, endedAt time.Time, failed bool, message string)
}

// RunRegistry is an in-memory registry of active heartbeat runs with disk
// persistence for restart recovery.
type RunRegistry struct {
	mu       sync.RWMutex
	active   map[string]*ActiveRun // key: "teamID/agentID"
	filePath string
	observer RunObserver
}

// SetObserver installs the run lifecycle observer.
func (r *RunRegistry) SetObserver(observer RunObserver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observer = observer
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
	r.register(teamID, agentID, runID, startedAt, "", cancel, true, false)
}

// RegisterOwnerRun registers a new run using its owner-reported original start
// time. A missing start remains unknown instead of falling back to the local
// request or waiter start time.
func (r *RunRegistry) RegisterOwnerRun(teamID, agentID, runID string, startedAt time.Time, cancel context.CancelFunc, ownerState ...string) {
	state := ""
	if len(ownerState) > 0 {
		state = ownerState[0]
	}
	r.register(teamID, agentID, runID, startedAt, state, cancel, true, !startedAt.IsZero())
}

// RegisterRecovered records an existing owner run while reattaching its local
// waiter. startedAt must be the owner's original start time; a zero value
// means that historical time is unavailable. It does not publish RunStarted
// because recovery did not create a new execution.
func (r *RunRegistry) RegisterRecovered(teamID, agentID, runID string, startedAt time.Time, state string, cancel context.CancelFunc) {
	r.register(teamID, agentID, runID, startedAt, state, cancel, false, !startedAt.IsZero())
}

func (r *RunRegistry) register(teamID, agentID, runID string, startedAt time.Time, state string, cancel context.CancelFunc, notifyStarted, ownerStartedAtKnown bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	run := &ActiveRun{
		TeamID:              teamID,
		AgentID:             agentID,
		RunID:               runID,
		StartedAt:           startedAt,
		CancelFn:            cancel,
		State:               state,
		ownerStartedAtKnown: ownerStartedAtKnown,
	}
	r.active[registryKey(teamID, agentID)] = run
	r.persistLocked()
	if notifyStarted && r.observer != nil {
		r.observer.RunStarted(*run)
	}
}

// Unregister removes an active run from the registry. Safe to call if key is
// not found. Reported to the observer as a clean stop.
func (r *RunRegistry) Unregister(teamID, agentID string) {
	r.Complete(teamID, agentID, false, "stopped")
}

// Complete removes an active run and reports its outcome to the observer.
func (r *RunRegistry) Complete(teamID, agentID string, failed bool, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := registryKey(teamID, agentID)
	run, ok := r.active[key]
	delete(r.active, key)
	r.persistLocked()
	if ok && r.observer != nil {
		r.observer.RunEnded(*run, time.Now().UTC(), failed, message)
	}
}

// MarkObserverUnavailable records that the local waiter ended without
// terminal owner evidence. It deliberately keeps the run registered and clears
// the stale cancellation function so a later scheduled reconciliation can
// safely reattach observation.
func (r *RunRegistry) MarkObserverUnavailable(teamID, agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.active[registryKey(teamID, agentID)]
	if !ok {
		return
	}
	run.State = ObligationWatcherUnavailable
	run.CancelFn = nil
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
	TeamID              string    `json:"teamId"`
	AgentID             string    `json:"agentId"`
	RunID               string    `json:"runId"`
	StartedAt           time.Time `json:"startedAt"`
	OwnerStartedAtKnown bool      `json:"ownerStartedAtKnown,omitempty"`
	State               string    `json:"state,omitempty"`
}

// persistLocked writes the current active map to disk. Caller must hold mu.
func (r *RunRegistry) persistLocked() {
	runs := make([]persistedRun, 0, len(r.active))
	for _, run := range r.active {
		runs = append(runs, persistedRun{
			TeamID:              run.TeamID,
			AgentID:             run.AgentID,
			RunID:               run.RunID,
			StartedAt:           run.StartedAt,
			OwnerStartedAtKnown: run.ownerStartedAtKnown,
			State:               run.State,
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
// Definitive terminal runs are removed; active runs and unresolved obligations
// are kept (without CancelFn since the goroutine is gone after a restart).
//
// A GetRun error means the owner is unavailable: retaining the entry prevents a
// later tick from mistaking an outage for a dead run. A run the owner no longer
// knows is retained as uncertain rather than dropped, because absence of
// terminal evidence is not terminal evidence.
func (r *RunRegistry) Recover(ctx context.Context, client AgentClient) {
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
		base := ActiveRun{
			TeamID:  pr.TeamID,
			AgentID: pr.AgentID,
			RunID:   pr.RunID,
			State:   pr.State,
		}
		if pr.OwnerStartedAtKnown {
			base.StartedAt = pr.StartedAt
			base.ownerStartedAtKnown = true
		}

		// A prior recovery already recorded a non-resolvable obligation.
		// Preserve it without re-probing the owner.
		if pr.State != ObligationRunning && pr.State != ObligationWatcherUnavailable {
			base.State = pr.State
			r.active[registryKey(pr.TeamID, pr.AgentID)] = &base
			continue
		}

		run, err := client.GetRun(ctx, pr.RunID)
		if err != nil {
			log.Printf("run_registry: retaining owner_unreachable run %s: %v", pr.RunID, err)
			base.State = ObligationOwnerUnreachable
			r.active[registryKey(pr.TeamID, pr.AgentID)] = &base
			continue
		}
		if run == nil {
			log.Printf("run_registry: retaining dispatch_uncertain run %s (owner reports not found)", pr.RunID)
			base.State = ObligationDispatchUncertain
			r.active[registryKey(pr.TeamID, pr.AgentID)] = &base
			continue
		}
		if IsTerminalStatus(run.Status) {
			continue
		}
		if ownerStartedAt, ok := parseOwnerRunStartedAt(run.StartedAt); ok {
			base.StartedAt = ownerStartedAt
			base.ownerStartedAtKnown = true
		} else if !pr.OwnerStartedAtKnown {
			base.StartedAt = time.Time{}
			base.ownerStartedAtKnown = false
		}
		if IsParkedStatus(run.Status) {
			base.State = ObligationPaused
		} else if IsNeedsReviewStatus(run.Status) {
			base.State = ObligationNeedsReview
		}
		r.active[registryKey(pr.TeamID, pr.AgentID)] = &base
	}

	r.persistLocked()
	if len(r.active) > 0 {
		log.Printf("run_registry: recovered %d active run(s)", len(r.active))
	}
}

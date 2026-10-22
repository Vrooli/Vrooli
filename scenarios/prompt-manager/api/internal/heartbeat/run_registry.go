package heartbeat

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vrooli/api-core/storage"
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
	mu              sync.RWMutex
	active          map[string]*ActiveRun // key: "teamID/agentID"
	filePath        string
	observer        RunObserver
	recoveryLoading bool
	recoveryRows    []persistedRun
}

type runOwnerIdentity struct {
	TeamID  string
	AgentID string
	RunID   string
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

// CompleteRun removes only the named run, so a stale waiter cannot end a
// newer run registered for the same team member.
func (r *RunRegistry) CompleteRun(teamID, agentID, runID string, failed bool, message string) {
	if runID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	key := registryKey(teamID, agentID)
	run, ok := r.active[key]
	if !ok || run.RunID != runID {
		return
	}
	delete(r.active, key)
	r.persistLocked()
	if r.observer != nil {
		r.observer.RunEnded(*run, time.Now().UTC(), failed, message)
	}
}

// MarkRunObserverUnavailable changes only the named run's observation state.
func (r *RunRegistry) MarkRunObserverUnavailable(teamID, agentID, runID string) {
	if runID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.active[registryKey(teamID, agentID)]
	if !ok || run.RunID != runID {
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

// persistLocked writes the primary presentation and unresolved legacy owner
// references to disk. Caller must hold mu.
func (r *RunRegistry) persistLocked() {
	runs := make([]persistedRun, 0, len(r.active)+len(r.recoveryRows))
	seen := make(map[runOwnerIdentity]struct{}, len(r.active)+len(r.recoveryRows))
	for _, run := range r.active {
		pr := persistedRun{
			TeamID:              run.TeamID,
			AgentID:             run.AgentID,
			RunID:               run.RunID,
			StartedAt:           run.StartedAt,
			OwnerStartedAtKnown: run.ownerStartedAtKnown,
			State:               run.State,
		}
		runs = append(runs, pr)
		seen[runOwnerIdentity{TeamID: pr.TeamID, AgentID: pr.AgentID, RunID: pr.RunID}] = struct{}{}
	}
	// During startup reconciliation, retain the original modeled owner
	// references until each one has an exact owner classification. Queue
	// hydration can persist before the registry fallback runs.
	for _, pr := range r.recoveryRows {
		identity := runOwnerIdentity{TeamID: pr.TeamID, AgentID: pr.AgentID, RunID: pr.RunID}
		if _, duplicate := seen[identity]; duplicate {
			continue
		}
		runs = append(runs, pr)
		seen[identity] = struct{}{}
	}
	data, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		log.Printf("run_registry: failed to marshal: %v", err)
		return
	}

	perm := os.FileMode(0o644)
	if info, err := os.Stat(r.filePath); err == nil {
		perm = info.Mode().Perm()
		if perm == 0 {
			log.Printf("run_registry: refusing to replace file with unsupported zero permissions: %s", r.filePath)
			return
		}
	} else if !os.IsNotExist(err) {
		log.Printf("run_registry: failed to inspect file permissions: %v", err)
		return
	}

	if err := storage.WriteFileAtomic(r.filePath, data, perm); err != nil {
		log.Printf("run_registry: failed to write: %v", err)
	}
}

// LoadPersisted stages the registry's local presentation without owner reads.
// Startup queue recovery uses this first so its persistence cannot erase
// registry-only legacy rows before their fallback reconciliation.
func (r *RunRegistry) LoadPersisted() {
	r.mu.Lock()
	if r.recoveryLoading {
		r.mu.Unlock()
		return
	}
	r.recoveryLoading = true
	r.recoveryRows = make([]persistedRun, 0)
	r.mu.Unlock()

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
	r.recoveryRows = append([]persistedRun(nil), runs...)
	for _, pr := range runs {
		key := registryKey(pr.TeamID, pr.AgentID)
		if current, ok := r.active[key]; ok && current.RunID != pr.RunID {
			continue
		}
		base := activeRunFromPersisted(pr)
		r.active[key] = &base
	}
}

func activeRunFromPersisted(pr persistedRun) ActiveRun {
	base := ActiveRun{TeamID: pr.TeamID, AgentID: pr.AgentID, RunID: pr.RunID, State: pr.State}
	if pr.OwnerStartedAtKnown {
		base.StartedAt = pr.StartedAt
		base.ownerStartedAtKnown = true
	}
	return base
}

// HydrateQueueOwners replaces the presentation for exact queue-owned members
// without emitting new-start events. Team-execution recovery already owns the
// classification and the persisted queue remains the admission authority.
func (r *RunRegistry) HydrateQueueOwners(owners []ActiveRun) {
	if len(owners) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range owners {
		owner := owners[i]
		if owner.RunID == "" {
			continue
		}
		owner.CancelFn = nil
		r.active[registryKey(owner.TeamID, owner.AgentID)] = &owner
	}
	r.persistLocked()
}

// Recover loads persisted runs from disk and checks each against agent-manager.
// Definitive terminal runs are removed; active runs and unresolved obligations
// are kept (without CancelFn since the goroutine is gone after a restart).
//
// A GetRun error means the owner is unavailable: retaining the entry prevents a
// later tick from mistaking an outage for a dead run. A run the owner no longer
// knows is retained as uncertain rather than dropped, because absence of
// terminal evidence is not terminal evidence.
func (r *RunRegistry) Recover(ctx context.Context, client AgentClient, queueOwners map[runOwnerIdentity]struct{}) {
	r.LoadPersisted()
	r.mu.Lock()
	defer r.mu.Unlock()
	stagedRows := append([]persistedRun(nil), r.recoveryRows...)
	unresolvedRows := make([]persistedRun, 0, len(stagedRows))

	for _, pr := range stagedRows {
		if _, covered := queueOwners[runOwnerIdentity{TeamID: pr.TeamID, AgentID: pr.AgentID, RunID: pr.RunID}]; covered {
			continue
		}
		base := activeRunFromPersisted(pr)
		key := registryKey(pr.TeamID, pr.AgentID)
		current := r.active[key]
		if current != nil && current.RunID != pr.RunID {
			// The queue-backed current run owns this member's presentation. Still
			// inspect a distinct legacy identity; retain any unresolved owner
			// reference on disk, never as a replacement or terminal inference.
			if retained, keep := recoverConflictingLegacyRun(ctx, client, *current, base); keep {
				unresolvedRows = append(unresolvedRows, retained)
			}
			continue
		}

		// A prior recovery already recorded a non-resolvable obligation.
		// Preserve it without re-probing the owner.
		if pr.State != ObligationRunning && pr.State != ObligationWatcherUnavailable {
			base.State = pr.State
			r.active[key] = &base
			continue
		}

		if client == nil {
			base.State = ObligationOwnerUnreachable
			r.active[key] = &base
			unresolvedRows = append(unresolvedRows, pr)
			continue
		}
		run, err := client.GetRun(ctx, pr.RunID)
		observation := observeOwnerRun(pr.RunID, run, err)
		if observation.state == ObligationOwnerUnreachable || observation.state == ObligationDispatchUncertain {
			log.Printf("run_registry: retaining %s run %s (owner status %q)", observation.state, pr.RunID, observation.status)
			base.State = observation.state
			r.active[key] = &base
			unresolvedRows = append(unresolvedRows, pr)
			continue
		}
		if observation.terminal {
			if current := r.active[key]; current != nil && current.RunID == pr.RunID {
				delete(r.active, key)
			}
			continue
		}
		if observation.startedAtKnown {
			base.StartedAt = observation.startedAt
			base.ownerStartedAtKnown = true
		} else if !pr.OwnerStartedAtKnown {
			base.StartedAt = time.Time{}
			base.ownerStartedAtKnown = false
		}
		if observation.state == ObligationPaused || observation.state == ObligationNeedsReview {
			base.State = observation.state
		}
		r.active[key] = &base
	}

	r.recoveryRows = unresolvedRows
	r.recoveryLoading = false
	r.persistLocked()
	if len(r.active) > 0 {
		log.Printf("run_registry: recovered %d active run(s)", len(r.active))
	}
}

func recoverConflictingLegacyRun(ctx context.Context, client AgentClient, queueOwner, legacy ActiveRun) (persistedRun, bool) {
	retained := persistedRun{
		TeamID: legacy.TeamID, AgentID: legacy.AgentID, RunID: legacy.RunID,
		StartedAt: legacy.StartedAt, OwnerStartedAtKnown: legacy.ownerStartedAtKnown,
	}
	if client == nil {
		retained.State = ObligationOwnerUnreachable
		log.Printf("run_registry: startup owner conflict team=%s agent=%s queue_run_id=%s legacy_run_id=%s legacy_owner_state=owner_unreachable action=retain_queue_presentation_report_legacy_state", queueOwner.TeamID, queueOwner.AgentID, queueOwner.RunID, legacy.RunID)
		return retained, true
	}
	run, err := client.GetRun(ctx, legacy.RunID)
	observation := observeOwnerRun(legacy.RunID, run, err)
	if observation.terminal {
		log.Printf("run_registry: startup owner conflict team=%s agent=%s queue_run_id=%s legacy_run_id=%s legacy_owner_state=%s action=drop_terminal_legacy_reference_keep_queue_presentation", queueOwner.TeamID, queueOwner.AgentID, queueOwner.RunID, legacy.RunID, observation.status)
		return persistedRun{}, false
	}
	retained.State = observation.state
	if observation.startedAtKnown {
		retained.StartedAt = observation.startedAt
		retained.OwnerStartedAtKnown = true
	}
	log.Printf("run_registry: startup owner conflict team=%s agent=%s queue_run_id=%s legacy_run_id=%s legacy_owner_state=%s action=retain_queue_presentation_report_legacy_state", queueOwner.TeamID, queueOwner.AgentID, queueOwner.RunID, legacy.RunID, observation.status)
	return retained, true
}

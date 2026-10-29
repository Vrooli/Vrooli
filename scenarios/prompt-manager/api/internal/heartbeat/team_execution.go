package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/vrooli/api-core/owneridentity"

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
	UncertainAgentIDs              []string `json:"uncertainAgentIds"`
	UnreachableAgentIDs            []string `json:"unreachableAgentIds"`
	PausedAgentIDs                 []string `json:"pausedAgentIds"`
	ApprovalPendingAgentIDs        []string `json:"approvalPendingAgentIds"`
	NeedsReviewAgentIDs            []string `json:"needsReviewAgentIds"`
	WatcherUnavailableAgentIDs     []string `json:"watcherUnavailableAgentIds"`
	PreparationInterruptedAgentIDs []string `json:"preparationInterruptedAgentIds"`
	AdmissionDeferredAgentIDs      []string `json:"admissionDeferredAgentIds"`
	// Owners exposes only the identifiers and dispatch classification needed
	// to diagnose a retained slot. It intentionally omits frozen prompts,
	// request snapshots, caller bindings, and other private execution payloads.
	Owners []TeamExecutionOwner `json:"owners"`
}

// TeamExecutionOwner identifies the retained-slot or queued owner record
// without implying that an external model process is currently active and
// without exposing its task or run payload.
type TeamExecutionOwner struct {
	AgentID              string `json:"agentId"`
	Phase                string `json:"phase"`
	State                string `json:"state,omitempty"`
	DispatchStage        string `json:"dispatchStage,omitempty"`
	DispatchFailurePhase string `json:"dispatchFailurePhase,omitempty"`
	DispatchFailureCode  string `json:"dispatchFailureCode,omitempty"`
	TaskID               string `json:"taskId,omitempty"`
	RunID                string `json:"runId,omitempty"`
	AdmissionOrigin      string `json:"admissionOrigin,omitempty"`
	OwnerStartedAt       string `json:"ownerStartedAt,omitempty"`
	ObligationID         string `json:"obligationId,omitempty"`
}

const (
	dispatchFailurePhaseDispatch           = "dispatch"
	dispatchFailurePhaseTaskAuthorization  = "task_authorization"
	dispatchFailurePhaseTaskResolution     = "task_resolution"
	dispatchFailurePhaseTaskLookup         = "task_lookup"
	dispatchFailurePhaseTaskMatch          = "task_match"
	dispatchFailurePhaseTaskCreate         = "task_create"
	dispatchFailurePhaseTaskReconciliation = "task_reconciliation"
	dispatchFailurePhaseTaskConfirmation   = "task_confirmation"
	dispatchFailurePhaseRunCreation        = "run_creation"
	dispatchFailurePhaseRunReconciliation  = "run_reconciliation"
	dispatchFailurePhaseTaskCancellation   = "task_cancellation"
	dispatchFailurePhaseCapacityDeferral   = "capacity_deferral"
)

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

// RunningOwnerUnverifiedError is returned when ClearRunning cannot prove that
// the retained slot's recorded run is terminal. An explicit force request is
// still available to an operator after checking the owner independently.
type RunningOwnerUnverifiedError struct {
	TeamID  string
	AgentID string
	RunID   string
	Reason  string
}

func (e *RunningOwnerUnverifiedError) Error() string {
	owner := e.RunID
	if owner == "" {
		owner = "no run ID is recorded"
	}
	return fmt.Sprintf("cannot safely clear running entry %s/%s (%s): %s; verify the owner or use --force", e.TeamID, e.AgentID, owner, e.Reason)
}

// IsRunningOwnerUnverified reports a ClearRunning refusal due to missing or
// inconsistent owner evidence.
func IsRunningOwnerUnverified(err error) bool {
	_, ok := err.(*RunningOwnerUnverifiedError)
	return ok
}

// RunningOwnerLookupError is returned when Agent Manager cannot be queried.
// Unlike an owner mismatch, this is an upstream failure rather than evidence
// that the retained slot is stale.
type RunningOwnerLookupError struct {
	TeamID  string
	AgentID string
	RunID   string
	Err     error
}

func (e *RunningOwnerLookupError) Error() string {
	return fmt.Sprintf("cannot verify running entry %s/%s owner %s: %v", e.TeamID, e.AgentID, e.RunID, e.Err)
}

func (e *RunningOwnerLookupError) Unwrap() error { return e.Err }

func IsRunningOwnerLookupFailure(err error) bool {
	_, ok := err.(*RunningOwnerLookupError)
	return ok
}

// Durable obligation states for a member's allocated execution slot. An empty
// state means an ordinary live run. The other states are nonterminal
// obligations that still hold the slot so a duplicate heartbeat cannot be
// admitted while ownership is unresolved. They are released only on definitive
// terminal owner evidence or an explicit operator recovery action.
const (
	AdmissionOriginUnknown   = ""
	AdmissionOriginScheduled = "scheduled"
	AdmissionOriginManual    = "manual"

	// DispatchStagePreparing is durably written before a newly admitted worker
	// starts. It proves that no Agent Manager task or run POST has been issued.
	DispatchStagePreparing = "preparing"
	// DispatchStageIntent is written atomically with the frozen task/run request
	// snapshots before the first owner POST.
	DispatchStageIntent = "task_intent"

	// ObligationRunning is a normal live run (or a run reconciled as active).
	ObligationRunning = ""
	// ObligationDispatchUncertain means a dispatch intent was persisted (or a
	// start response was lost) but the run identity cannot be confirmed. The
	// slot is retained and the obligation is surfaced for owner reconciliation.
	ObligationDispatchUncertain = "dispatch_uncertain"
	// ObligationDispatchCanceled is a durable no-replay tombstone for an
	// unconfirmed task request proven not to have been sent. The current worker
	// still owns the in-memory slot until its normal completion callback.
	ObligationDispatchCanceled = "dispatch_canceled_before_send"
	// ObligationOwnerUnreachable means reconciliation could not reach the
	// owner. Absence of evidence is not terminal evidence; the slot is retained.
	ObligationOwnerUnreachable = "owner_unreachable"
	// ObligationPaused means the owner reports the run as suspended (parked or
	// quota-paused). It is nonterminal retained ownership, not a missing run.
	ObligationPaused = "paused"
	// ObligationApprovalPending retains a run-stage request and its exact
	// idempotent dispatch intent while operator approval is pending.
	ObligationApprovalPending = "approval_pending"
	// ObligationAdmissionDeferred retains an exact dispatch intent after
	// Agent Manager definitively refused it only because its current grant
	// capacity is full. The scheduled callback may replay it once; admission
	// is re-evaluated by Agent Manager on every request.
	ObligationAdmissionDeferred = "admission_deferred"
	// ObligationNeedsReview is a stopped owner run awaiting operator review.
	// It remains retained ownership and must not cause duplicate dispatch.
	ObligationNeedsReview = "needs_review"
	// ObligationWatcherUnavailable means a local completion observer ended
	// without terminal owner evidence. The slot remains held until a scheduled
	// reconciliation can reattach it or prove the owner run is terminal.
	ObligationWatcherUnavailable = "watcher_unavailable"
	// ObligationPreparationInterrupted is an accepted admission recovered from
	// a pre-owner-effect preparation record. It holds the slot until a fresh
	// normally-gated admission replaces it.
	ObligationPreparationInterrupted = "preparation_interrupted"
)

type queuedExecution struct {
	ManualAdmission bool `json:"manual_admission,omitempty"`
	// AdmissionOrigin is independent of the authenticated owner caller mode.
	// It records whether an accepted intent came from cron or an explicit
	// operator trigger so recovery does not infer fresh-work authority.
	AdmissionOrigin string `json:"admission_origin,omitempty"`
	DispatchStage   string `json:"DispatchStage,omitempty"`
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
	// TaskRequestSnapshot freezes the caller-authored task before its first
	// POST. TaskConfirmed is persisted only after the exact task is found or
	// created and its immutable terms are checked.
	TaskRequestSnapshot  string `json:"TaskRequestSnapshot,omitempty"`
	TaskConfirmed        bool   `json:"TaskConfirmed,omitempty"`
	DispatchFailurePhase string `json:"DispatchFailurePhase,omitempty"`
	DispatchFailureCode  string `json:"DispatchFailureCode,omitempty"`
	// CreateRunRequestSnapshot freezes the exact serialized owner request before
	// its first send so recovery does not reconstruct terms from mutable state.
	CreateRunRequestSnapshot string `json:"CreateRunRequestSnapshot,omitempty"`
	CreateRunCallerMode      string `json:"CreateRunCallerMode,omitempty"`
	CreateRunCallerBinding   string `json:"CreateRunCallerBinding,omitempty"`
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
	ResetConsumedEventID string                                `json:"ResetConsumedEventID,omitempty"`
	OwnerObligationID    string                                `json:"OwnerObligationID,omitempty"`
	OwnerStartedAt       string                                `json:"OwnerStartedAt,omitempty"`
	OwnerStateSnapshots  map[string]retainedOwnerStateSnapshot `json:"OwnerStateSnapshots,omitempty"`
}

type runningEntry struct {
	DispatchStage            string
	AdmissionOrigin          string
	ProfileKey               string
	RunID                    string
	State                    string
	IdempotencyKey           string
	TaskID                   string
	TaskRequestSnapshot      string
	TaskConfirmed            bool
	DispatchFailurePhase     string
	DispatchFailureCode      string
	CreateRunRequestSnapshot string
	CreateRunCallerMode      string
	CreateRunCallerBinding   string
	RunTag                   string
	WorkloadKind             string
	WorkloadKey              string
	WorkloadInstance         string
	ResetConsumedEventID     string
	OwnerObligationID        string
	OwnerStartedAt           string
	OwnerStateSnapshots      map[string]retainedOwnerStateSnapshot
}

// retainedOwnerStateSnapshot freezes one observed state per obligation so
// later startup reconciliation can replay identical immutable event terms.
type retainedOwnerStateSnapshot struct {
	OwnerObligationID string `json:"ownerObligationId"`
	State             string `json:"state"`
	StartedAt         string `json:"startedAt,omitempty"`
	ObservedAt        string `json:"observedAt"`
	TaskID            string `json:"taskId,omitempty"`
	RunID             string `json:"runId,omitempty"`
	ReviewURL         string `json:"reviewUrl,omitempty"`
}

// DispatchIntent is the durable record of a heartbeat's owner start request,
// persisted before the request leaves the process. TaskID and RunTag let a
// retained uncertain obligation be replayed with the same owner request (and
// the same idempotency key) after a crash or lost response.
type DispatchIntent struct {
	AdmissionOrigin          string
	IdempotencyKey           string
	TaskID                   string
	TaskRequestSnapshot      string
	TaskConfirmed            bool
	RunTag                   string
	WorkloadKind             string
	WorkloadKey              string
	WorkloadInstance         string
	CreateRunRequestSnapshot string
	CreateRunCallerMode      string
	CreateRunCallerBinding   string
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
	origin := heartbeatAdmissionOrigin(ctx)
	if origin == AdmissionOriginUnknown {
		// Direct Enqueue callers are explicit operator/service API requests unless
		// the cron scheduler marks its origin before entering this owner.
		origin = AdmissionOriginManual
	}
	manualAdmission := false
	callerMode := createRunCallerOriginUnqualified
	callerBinding := ""
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
		proof := createRunCaller(qualified)
		callerMode = proof.origin
		if !validCreateRunCallerMode(callerMode) {
			callerMode = createRunCallerOriginUnqualified
			if proof.authorization != "" || proof.runIdentity != "" {
				callerMode = createRunCallerOriginCaller
			}
		}
		callerBinding = proof.binding
		if proof.origin == createRunCallerOriginService {
			// Queue service mode, but not the minted bearer. Executor refreshes a
			// request-local service token before each owner interaction.
			worker = withCreateRunCallerOrigin(carried, "", "", proof.origin)
		} else {
			worker = withCreateRunCallerOrigin(carried, proof.authorization, proof.runIdentity, proof.origin)
			worker = withCreateRunCallerBinding(worker, proof.binding)
		}
	}
	worker = withHeartbeatAdmissionOrigin(worker, origin)
	c.mu.Lock()
	previous, hasRunning := c.running[agentID]
	if hasRunning && !isReplaceableInterruptedPreparation(previous) {
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

	if hasRunning || len(c.running) < c.maxConcurrentRuns {
		c.running[agentID] = runningEntry{
			DispatchStage:   DispatchStagePreparing,
			AdmissionOrigin: origin,
			ProfileKey:      profileKey,
		}
		if err := c.persistLocked(); err != nil {
			if hasRunning {
				c.running[agentID] = previous
			} else {
				delete(c.running, agentID)
			}
			c.mu.Unlock()
			return nil, fmt.Errorf("persisting execution preparation: %w", err)
		}
		c.mu.Unlock()
		c.startExecution(agentID, profileKey, worker)
		result.Status = "started"
		result.Position = 0
		return result, nil
	}

	queued := queuedExecution{
		AgentID:                agentID,
		ProfileKey:             profileKey,
		AdmissionOrigin:        origin,
		caller:                 worker,
		ManualAdmission:        manualAdmission,
		CreateRunCallerMode:    callerMode,
		CreateRunCallerBinding: callerBinding,
	}
	c.queue = append(c.queue, queued)
	c.queued[agentID] = true
	if err := c.persistLocked(); err != nil {
		c.queue = c.queue[:len(c.queue)-1]
		delete(c.queued, agentID)
		c.mu.Unlock()
		return nil, fmt.Errorf("persisting queued execution: %w", err)
	}
	c.mu.Unlock()
	return result, nil
}

func isReplaceableInterruptedPreparation(entry runningEntry) bool {
	if entry.State != ObligationPreparationInterrupted || entry.DispatchStage != DispatchStagePreparing ||
		entry.RunID != "" || entry.IdempotencyKey != "" || entry.TaskID != "" || entry.TaskRequestSnapshot != "" || entry.TaskConfirmed ||
		entry.CreateRunRequestSnapshot != "" || entry.CreateRunCallerMode != "" || entry.CreateRunCallerBinding != "" ||
		entry.RunTag != "" || entry.WorkloadKind != "" || entry.WorkloadKey != "" || entry.WorkloadInstance != "" ||
		entry.ResetConsumedEventID != "" || entry.OwnerStartedAt != "" {
		return false
	}
	if entry.OwnerObligationID == "" {
		return len(entry.OwnerStateSnapshots) == 0
	}
	if len(entry.OwnerStateSnapshots) != 1 {
		return false
	}
	snapshot, ok := entry.OwnerStateSnapshots[ObligationPreparationInterrupted]
	return ok && validInterruptedPreparationSnapshot(snapshot, entry.OwnerObligationID)
}

func validInterruptedPreparationSnapshot(snapshot retainedOwnerStateSnapshot, ownerID string) bool {
	_, observedAtErr := time.Parse(time.RFC3339Nano, snapshot.ObservedAt)
	return observedAtErr == nil && snapshot.State == ObligationPreparationInterrupted &&
		snapshot.OwnerObligationID == ownerID && snapshot.StartedAt == "" &&
		snapshot.ObservedAt != "" && snapshot.TaskID == "" && snapshot.RunID == ""
}

// OnMemberComplete is called when an execution finishes. It frees a slot and
// starts queued members until the current concurrency limit is full.
func (c *TeamExecutionContext) OnMemberComplete(agentID string) {
	c.completeMember(agentID, "")
}

// OnMemberCompleteRun releases a slot only when it is still owned by runID.
func (c *TeamExecutionContext) OnMemberCompleteRun(agentID, runID string) {
	c.completeMember(agentID, runID)
}

func (c *TeamExecutionContext) OwnsRun(agentID, runID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.running[agentID]
	return ok && runID != "" && entry.RunID == runID
}

func (c *TeamExecutionContext) completeMember(agentID, expectedRunID string) bool {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	if !ok || (expectedRunID != "" && entry.RunID != expectedRunID) {
		c.mu.Unlock()
		return false
	}
	delete(c.running, agentID)
	dispatches, persistErr := c.dispatchAvailableLockedWithPersistence()
	if persistErr != nil {
		// Keep ownership in memory when its completion could not be persisted.
		c.running[agentID] = entry
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()
	c.startExecutions(dispatches)
	return true
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
	preparationInterrupted := make([]string, 0)
	uncertain := make([]string, 0)
	unreachable := make([]string, 0)
	paused := make([]string, 0)
	approvalPending := make([]string, 0)
	needsReview := make([]string, 0)
	watcherUnavailable := make([]string, 0)
	admissionDeferred := make([]string, 0)
	owners := make([]TeamExecutionOwner, 0, len(c.running)+len(c.queue))
	for agentID, entry := range c.running {
		owners = append(owners, TeamExecutionOwner{
			AgentID: agentID, Phase: "retained", State: entry.State,
			DispatchStage: entry.DispatchStage, DispatchFailurePhase: entry.DispatchFailurePhase,
			DispatchFailureCode: entry.DispatchFailureCode, TaskID: entry.TaskID, RunID: entry.RunID,
			AdmissionOrigin: entry.AdmissionOrigin, OwnerStartedAt: entry.OwnerStartedAt,
			ObligationID: entry.OwnerObligationID,
		})
		if entry.State == ObligationPreparationInterrupted {
			preparationInterrupted = append(preparationInterrupted, agentID)
		} else if entry.State == ObligationAdmissionDeferred {
			admissionDeferred = append(admissionDeferred, agentID)
		} else {
			running = append(running, agentID)
		}
		switch entry.State {
		case ObligationDispatchUncertain:
			uncertain = append(uncertain, agentID)
		case ObligationOwnerUnreachable:
			unreachable = append(unreachable, agentID)
		case ObligationPaused:
			paused = append(paused, agentID)
		case ObligationApprovalPending:
			approvalPending = append(approvalPending, agentID)
		case ObligationNeedsReview:
			needsReview = append(needsReview, agentID)
		case ObligationWatcherUnavailable:
			watcherUnavailable = append(watcherUnavailable, agentID)
		case ObligationAdmissionDeferred:
			// This is a retained, definite no-admission result. It owns a slot
			// for exact retry but is not a running model execution or unknown
			// dispatch outcome.
		}
	}
	sort.Strings(running)
	sort.Strings(uncertain)
	sort.Strings(unreachable)
	sort.Strings(paused)
	sort.Strings(approvalPending)
	sort.Strings(needsReview)
	sort.Strings(watcherUnavailable)
	sort.Strings(preparationInterrupted)
	sort.Strings(admissionDeferred)

	queue := make([]string, 0, len(c.queue))
	for _, item := range c.queue {
		queue = append(queue, item.AgentID)
		owners = append(owners, TeamExecutionOwner{
			AgentID: item.AgentID, Phase: "queued", State: item.State,
			DispatchStage: item.DispatchStage, DispatchFailurePhase: item.DispatchFailurePhase,
			DispatchFailureCode: item.DispatchFailureCode, TaskID: item.TaskID, RunID: item.RunID,
			AdmissionOrigin: item.AdmissionOrigin, OwnerStartedAt: item.OwnerStartedAt,
			ObligationID: item.OwnerObligationID,
		})
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].AgentID != owners[j].AgentID {
			return owners[i].AgentID < owners[j].AgentID
		}
		return owners[i].Phase < owners[j].Phase
	})

	return TeamExecutionStatus{
		TeamID:                         c.teamID,
		State:                          state,
		RunningAgentIDs:                running,
		Queue:                          queue,
		QueuePolicy:                    c.queuePolicy,
		MaxConcurrentRuns:              c.maxConcurrentRuns,
		UncertainAgentIDs:              uncertain,
		UnreachableAgentIDs:            unreachable,
		PausedAgentIDs:                 paused,
		ApprovalPendingAgentIDs:        approvalPending,
		NeedsReviewAgentIDs:            needsReview,
		WatcherUnavailableAgentIDs:     watcherUnavailable,
		PreparationInterruptedAgentIDs: preparationInterrupted,
		AdmissionDeferredAgentIDs:      admissionDeferred,
		Owners:                         owners,
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
	dispatches, _ := c.dispatchAvailableLockedWithPersistence()
	return dispatches
}

func (c *TeamExecutionContext) dispatchAvailableLockedWithPersistence() ([]queuedExecution, error) {
	previousRunning := make(map[string]runningEntry, len(c.running))
	for agentID, entry := range c.running {
		previousRunning[agentID] = entry
	}
	previousQueue := append([]queuedExecution(nil), c.queue...)
	previousQueued := make(map[string]bool, len(c.queued))
	for agentID, queued := range c.queued {
		previousQueued[agentID] = queued
	}
	var dispatches []queuedExecution
	// A bounded scan keeps expired/recovered entries held without blocking other
	// callers. Their proof is never renewed from a different queued request.
	scans := len(c.queue)
	for len(c.running) < c.maxConcurrentRuns && len(c.queue) > 0 && scans > 0 {
		scans--
		next := c.queue[0]
		c.queue = c.queue[1:]
		realClient, real := c.agentClient.(*AgentManagerClient)
		serviceMode := next.CreateRunCallerMode == createRunCallerOriginService
		blocked := serviceMode && (!real || realClient == nil)
		if real && (next.caller == nil || requireQualifiedCaller(next.caller) != nil) {
			blocked = true
		}
		if blocked {
			next.State = "caller_required"
			c.queue = append(c.queue, next)
			continue
		}
		delete(c.queued, next.AgentID)
		c.running[next.AgentID] = runningEntry{
			DispatchStage:   DispatchStagePreparing,
			AdmissionOrigin: next.AdmissionOrigin,
			ProfileKey:      next.ProfileKey,
		}
		dispatches = append(dispatches, next)
	}
	if err := c.persistLocked(); err != nil {
		c.running = previousRunning
		c.queue = previousQueue
		c.queued = previousQueued
		log.Printf("team_execution: queue promotion held because preparation save failed for %s: %v", c.teamID, err)
		return nil, err
	}
	return dispatches, nil
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
			if IsApprovalPending(err) {
				if approvalPendingStage(err) == "creating_task" {
					// The owner guarantees a typed task-stage 202 created neither
					// task nor run, so a later schedule can safely retry admission.
					c.OnMemberComplete(agentID)
				}
				return
			}
			if IsDispatchUncertain(err) {
				// The owner may have accepted the run. Retain the slot as an
				// uncertain obligation instead of freeing it for a duplicate.
				log.Printf("team_execution: retaining uncertain dispatch obligation for %s/%s: %v", c.teamID, agentID, err)
				c.MarkDispatchUncertain(agentID, dispatchFailurePhaseFromError(err, dispatchFailurePhaseDispatch), err)
				return
			}
			if c.isAdmissionDeferred(agentID) {
				log.Printf("team_execution: retaining capacity-deferred dispatch for %s/%s: %v", c.teamID, agentID, err)
				return
			}
			log.Print(formatExecutionFailure(c.teamID, agentID, err))
			c.OnMemberComplete(agentID)
			return
		}
		log.Printf("team_execution: execution started for %s/%s, run ID: %s", c.teamID, agentID, result.RunID)
	}()
}

func formatExecutionFailure(teamID, agentID string, err error) string {
	message := fmt.Sprintf("team_execution: execution failed for %s/%s", teamID, agentID)
	if reason := admissionReasonFromError(err); reason != "" {
		message += " admission_reason=" + reason
	}
	return fmt.Sprintf("%s: %v", message, err)
}

// Shutdown waits for dispatch goroutines to finish their durable state update.
func (c *TeamExecutionContext) Shutdown() {
	c.executionWG.Wait()
}

// BeginDispatch records the immutable task and run requests before the first
// owner POST. A crash at either phase resumes from the saved task identity and
// then the saved idempotent run request.
func (c *TeamExecutionContext) BeginDispatch(agentID string, intent DispatchIntent) error {
	request, err := createRunRequestFromSnapshot(intent.CreateRunRequestSnapshot, intent.IdempotencyKey, intent.TaskID)
	if err != nil {
		return fmt.Errorf("cannot persist run dispatch intent: %w", err)
	}
	if intent.TaskConfirmed && intent.TaskRequestSnapshot == "" {
		return fmt.Errorf("cannot persist task confirmation without a frozen task request")
	}
	if intent.TaskRequestSnapshot != "" {
		task, taskErr := parseTaskDispatchRequest(intent.TaskRequestSnapshot)
		if taskErr != nil {
			return fmt.Errorf("cannot persist task dispatch intent: %w", taskErr)
		}
		if task.ID != intent.TaskID || request.TaskID != task.ID {
			return fmt.Errorf("cannot persist dispatch intent: task and run request identities differ")
		}
	}
	if !validCreateRunCallerMode(intent.CreateRunCallerMode) {
		return fmt.Errorf("cannot persist run dispatch intent: caller mode is missing or invalid")
	}
	if intent.CreateRunCallerMode == createRunCallerOriginCaller && intent.CreateRunCallerBinding == "" {
		return fmt.Errorf("cannot persist run dispatch intent: verified caller binding is missing")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		return ErrRunningEntryNotFound
	}
	if intent.AdmissionOrigin != AdmissionOriginUnknown {
		if intent.AdmissionOrigin != AdmissionOriginScheduled && intent.AdmissionOrigin != AdmissionOriginManual {
			return fmt.Errorf("cannot persist dispatch intent: admission origin is invalid")
		}
		if entry.AdmissionOrigin != AdmissionOriginUnknown && entry.AdmissionOrigin != intent.AdmissionOrigin {
			return fmt.Errorf("cannot persist dispatch intent: admission origin changed")
		}
		entry.AdmissionOrigin = intent.AdmissionOrigin
	}
	previous := entry
	entry.State = ObligationDispatchUncertain
	entry.DispatchStage = DispatchStageIntent
	entry.IdempotencyKey = intent.IdempotencyKey
	entry.TaskID = intent.TaskID
	entry.TaskRequestSnapshot = intent.TaskRequestSnapshot
	entry.TaskConfirmed = intent.TaskConfirmed
	entry.DispatchFailurePhase = ""
	entry.DispatchFailureCode = ""
	entry.CreateRunRequestSnapshot = intent.CreateRunRequestSnapshot
	entry.CreateRunCallerMode = intent.CreateRunCallerMode
	entry.CreateRunCallerBinding = intent.CreateRunCallerBinding
	entry.ProfileKey = request.ProfileRef.ProfileKey
	entry.RunTag = intent.RunTag
	entry.WorkloadKind = intent.WorkloadKind
	entry.WorkloadKey = intent.WorkloadKey
	entry.WorkloadInstance = intent.WorkloadInstance
	entry.OwnerObligationID = heartbeatOwnerObligationID(c.teamID, agentID, intent.IdempotencyKey, "", "")
	entry.OwnerStartedAt = ""
	entry.OwnerStateSnapshots = nil
	c.running[agentID] = entry
	if err := c.persistLocked(); err != nil {
		// The owner start request is still gated on this save. Restore the
		// in-memory entry so it does not claim a durable uncertain intent that
		// was never committed.
		c.running[agentID] = previous
		return err
	}
	return nil
}

// ConfirmTaskDispatch durably advances a task-stage obligation after the
// exact owner task and its immutable caller-authored terms have been verified.
// CreateRun must remain gated on this save.
func (c *TeamExecutionContext) ConfirmTaskDispatch(agentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		return ErrRunningEntryNotFound
	}
	if entry.TaskRequestSnapshot == "" {
		return errors.New("cannot confirm task dispatch without a frozen task request")
	}
	if entry.TaskConfirmed {
		return nil
	}
	previous := entry
	entry.TaskConfirmed = true
	entry.DispatchFailurePhase = ""
	entry.DispatchFailureCode = ""
	c.running[agentID] = entry
	if err := c.persistLocked(); err != nil {
		c.running[agentID] = previous
		return err
	}
	return nil
}

// CancelUnsentTaskDispatch durably marks only the exact unconfirmed task intent
// whose pre-send cancellation has been proven. The worker keeps its slot until
// its normal completion callback, preventing that callback from releasing a
// later attempt for the same member.
func (c *TeamExecutionContext) CancelUnsentTaskDispatch(agentID, taskID, idempotencyKey, taskSnapshot string) error {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	if !ok {
		c.mu.Unlock()
		return ErrRunningEntryNotFound
	}
	if entry.TaskConfirmed || entry.TaskRequestSnapshot == "" || entry.TaskRequestSnapshot != taskSnapshot || entry.RunID != "" || entry.TaskID != taskID || entry.IdempotencyKey != idempotencyKey {
		c.mu.Unlock()
		return fmt.Errorf("refusing to cancel a task dispatch that no longer matches the proven unsent intent")
	}
	previous := entry
	entry.State = ObligationDispatchCanceled
	c.running[agentID] = entry
	if err := c.persistLocked(); err != nil {
		c.running[agentID] = previous
		c.mu.Unlock()
		return fmt.Errorf("persist canceled task dispatch: %w", err)
	}
	c.mu.Unlock()
	return nil
}

func createRunRequestFromSnapshot(snapshot, expectedKey, expectedTaskID string) (*CreateRunRequest, error) {
	if snapshot == "" {
		return nil, fmt.Errorf("complete original CreateRun request snapshot is unavailable")
	}
	var request CreateRunRequest
	if err := json.Unmarshal([]byte(snapshot), &request); err != nil {
		return nil, fmt.Errorf("original CreateRun request snapshot is invalid: %w", err)
	}
	if request.IdempotencyKey == "" || request.IdempotencyKey != expectedKey || request.TaskID == "" || request.TaskID != expectedTaskID {
		return nil, fmt.Errorf("original CreateRun request snapshot does not match the retained task and idempotency key")
	}
	if request.ProfileRef == nil || request.ProfileRef.ProfileKey == "" {
		return nil, fmt.Errorf("original CreateRun request snapshot has no resolved profile key")
	}
	return &request, nil
}

func validCreateRunCallerMode(mode string) bool {
	switch mode {
	case createRunCallerOriginUnqualified, createRunCallerOriginCaller, createRunCallerOriginService:
		return true
	default:
		return false
	}
}

func prepareReplayCaller(ctx context.Context, agentClient AgentClient, mode, binding string) (context.Context, error) {
	switch mode {
	case createRunCallerOriginService:
		client, ok := agentClient.(*AgentManagerClient)
		if !ok || client == nil {
			return nil, fmt.Errorf("original service caller cannot be reauthorized by the configured owner client")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// A new request/human proof must not replace the original service
		// identity. The existing client mints a fresh service credential here.
		// Clear the request-local proof while preserving caller cancellation and
		// deadline for both token exchange and the owner request.
		serviceCtx := context.WithValue(ctx, createRunCallerKey{}, createRunCallerProof{})
		qualified, err := client.prepareCreateRunCaller(serviceCtx)
		if err != nil {
			return nil, err
		}
		proof := createRunCaller(qualified)
		if proof.origin != createRunCallerOriginService || proof.authorization == "" {
			return nil, fmt.Errorf("fresh service caller proof is unavailable")
		}
		return qualified, nil
	case createRunCallerOriginCaller:
		if binding == "" {
			return nil, fmt.Errorf("original verified caller binding is unavailable")
		}
		proof := createRunCaller(ctx)
		if proof.origin != "" && proof.origin != createRunCallerOriginCaller {
			return nil, fmt.Errorf("current proof origin does not match the original caller mode")
		}
		if proof.authorization == "" && proof.runIdentity == "" {
			return nil, fmt.Errorf("original caller proof is unavailable; it is not persisted with the queue")
		}
		if client, ok := agentClient.(*AgentManagerClient); ok && client != nil {
			qualified, err := client.prepareCreateRunCaller(ctx)
			if err != nil {
				return nil, err
			}
			qualifiedProof := createRunCaller(qualified)
			if qualifiedProof.origin != createRunCallerOriginCaller || qualifiedProof.binding == "" || qualifiedProof.binding != binding {
				return nil, fmt.Errorf("current proof does not match the original caller mode")
			}
			return qualified, nil
		}
		return nil, fmt.Errorf("original caller cannot be freshly verified by the configured owner client")
	case createRunCallerOriginUnqualified:
		if owneridentity.CreateRunCallerEnforced() {
			return nil, fmt.Errorf("original request had no caller proof and current owner policy requires one")
		}
		return ctx, nil
	default:
		return nil, fmt.Errorf("original caller mode is unavailable")
	}
}

// ReconcileDispatch resolves a retained uncertain or approval-pending run
// obligation by replaying the recorded owner start request with its idempotency key. The
// owner deduplicates on that key, so a lost start response converges on the
// one run it accepted; if the request never reached the owner, the replay
// creates the intended run for the already-created task.
//
// The outcome is owner-mediated and therefore definite for reconciliation:
//   - success            -> bind the confirmed run and re-attach a completion waiter
//   - uncertain error    -> retain the obligation (owner still unreachable)
//   - replay rejection    -> retain the obligation unless the owner proves no run exists
//
// It returns ErrRunningEntryNotFound when no entry exists and nil when the
// entry is not an uncertain or approval-pending dispatch (nothing to reconcile).
func (c *TeamExecutionContext) ReconcileDispatch(ctx context.Context, agentID string) error {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	if !ok {
		c.mu.Unlock()
		return ErrRunningEntryNotFound
	}
	if entry.State != ObligationDispatchUncertain && entry.State != ObligationApprovalPending && entry.State != ObligationAdmissionDeferred {
		c.mu.Unlock()
		return nil
	}
	if entry.TaskRequestSnapshot != "" && !entry.TaskConfirmed {
		c.mu.Unlock()
		return c.ReconcileTaskDispatch(ctx, agentID)
	}
	request, snapshotErr := createRunRequestFromSnapshot(entry.CreateRunRequestSnapshot, entry.IdempotencyKey, entry.TaskID)
	if snapshotErr != nil {
		c.mu.Unlock()
		return fmt.Errorf("cannot replay %s/%s dispatch: %w", c.teamID, agentID, snapshotErr)
	}
	if !validCreateRunCallerMode(entry.CreateRunCallerMode) {
		c.mu.Unlock()
		return fmt.Errorf("cannot replay %s/%s dispatch: original caller mode is unavailable", c.teamID, agentID)
	}
	if c.agentClient == nil {
		c.mu.Unlock()
		return fmt.Errorf("cannot replay %s/%s dispatch: agent client is not configured", c.teamID, agentID)
	}
	profileKey := ""
	if request.ProfileRef != nil {
		profileKey = request.ProfileRef.ProfileKey
	}
	callerMode := entry.CreateRunCallerMode
	c.mu.Unlock()

	callerBinding := entry.CreateRunCallerBinding
	replayCtx, authErr := prepareReplayCaller(ctx, c.agentClient, callerMode, callerBinding)
	if authErr != nil {
		return fmt.Errorf("cannot replay %s/%s dispatch: %w", c.teamID, agentID, authErr)
	}

	run, err := c.agentClient.CreateRun(replayCtx, request)
	if err != nil {
		if IsApprovalPending(err) {
			c.MarkApprovalPending(agentID)
			return err
		}
		if IsDispatchUncertain(err) {
			c.MarkDispatchUncertain(agentID, dispatchFailurePhaseRunReconciliation, err)
			return err
		}
		if admissionReasonFromError(err) == "grant_limit_reached" {
			if markErr := c.MarkAdmissionDeferred(agentID); markErr != nil {
				c.MarkDispatchUncertain(agentID, dispatchFailurePhaseCapacityDeferral, errors.Join(err, markErr))
				return fmt.Errorf("retaining capacity-deferred dispatch: %w", errors.Join(err, markErr))
			}
			return err
		}
		// A replay-time rejection is not proof that the key has no accepted run:
		// the owner can reject changed request terms before resolving the key.
		// Keep the slot held for explicit owner reconciliation.
		log.Printf("team_execution: retaining dispatch obligation for %s/%s after replay refusal: %v", c.teamID, agentID, err)
		c.MarkDispatchUncertain(agentID, dispatchFailurePhaseRunReconciliation, err)
		return err
	}
	if run == nil || run.ID == "" {
		reason := "owner accepted replay but returned no run identity"
		c.MarkDispatchUncertain(agentID, dispatchFailurePhaseRunReconciliation, errors.New(reason))
		return fmt.Errorf("%s/%s: %s", c.teamID, agentID, reason)
	}

	c.SetRunningRunID(agentID, run.ID, run.Status, run.StartedAt)
	runTag := ""
	if request.Tag != nil {
		runTag = *request.Tag
	}
	if !c.watchRetainedRun(agentID, run.ID, profileKey, request.TaskID, runTag, run.Status, run.StartedAt, run.EndedAt, entry.AdmissionOrigin) {
		c.MarkWatcherUnavailableForRun(agentID, run.ID)
	}
	log.Printf("team_execution: resolved retained dispatch for %s/%s to run %s", c.teamID, agentID, run.ID)
	return nil
}

// ReconcileTaskDispatch resolves only the persisted, unconfirmed task stage.
// Once confirmation is durable, run recovery bypasses this phase and replays
// the frozen CreateRun request even if the task has since moved to running.
func (c *TeamExecutionContext) ReconcileTaskDispatch(ctx context.Context, agentID string) error {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	if !ok {
		c.mu.Unlock()
		return ErrRunningEntryNotFound
	}
	if entry.TaskRequestSnapshot == "" || entry.TaskConfirmed {
		c.mu.Unlock()
		return nil
	}
	if c.agentClient == nil {
		c.mu.Unlock()
		return errors.New("cannot reconcile task dispatch: agent client is not configured")
	}
	snapshot := entry.TaskRequestSnapshot
	callerMode := entry.CreateRunCallerMode
	callerBinding := entry.CreateRunCallerBinding
	client := c.agentClient
	c.mu.Unlock()

	replayCtx, authErr := prepareReplayCaller(ctx, client, callerMode, callerBinding)
	if authErr != nil {
		c.MarkDispatchUncertain(agentID, dispatchFailurePhaseTaskAuthorization, authErr)
		return fmt.Errorf("reauthorize retained task dispatch: %w", authErr)
	}
	if _, err := resolveTaskDispatch(replayCtx, client, snapshot); err != nil {
		if IsApprovalPending(err) {
			c.MarkApprovalPending(agentID)
			return err
		}
		if admissionReasonFromError(err) == "grant_limit_reached" {
			if markErr := c.MarkAdmissionDeferred(agentID); markErr != nil {
				c.MarkDispatchUncertain(agentID, dispatchFailurePhaseCapacityDeferral, errors.Join(err, markErr))
				return fmt.Errorf("retaining capacity-deferred task dispatch: %w", errors.Join(err, markErr))
			}
			return err
		}
		c.MarkDispatchUncertain(agentID, dispatchFailurePhaseFromError(err, dispatchFailurePhaseTaskResolution), err)
		return err
	}
	if err := c.ConfirmTaskDispatch(agentID); err != nil {
		c.MarkDispatchUncertain(agentID, dispatchFailurePhaseTaskConfirmation, err)
		return fmt.Errorf("confirm recovered task dispatch: %w", err)
	}
	return c.ReconcileDispatch(ctx, agentID)
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

// retainedRunProjectionWatcher is implemented by an executor that can re-attach a
// completion waiter to a run it did not create in this process. A recovered
// obligation bound during Recover otherwise has no goroutine to release its
// slot when the owner reports terminal state.
type retainedRunProjectionWatcher interface {
	WatchRecoveredRunWithProjection(teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin string) bool
}

func (c *TeamExecutionContext) watchRetainedRun(agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin string) bool {
	watcher, ok := c.executor.(retainedRunProjectionWatcher)
	if !ok || watcher == nil {
		return false
	}
	return watcher.WatchRecoveredRunWithProjection(c.teamID, agentID, runID, profileKey, taskID, runTag, ownerStatus, ownerStartedAt, ownerTerminalAt, admissionOrigin)
}

// MarkDispatchUncertain retains a member's slot after a dispatch whose outcome
// is unknown. It is idempotent and never releases the obligation.
func (c *TeamExecutionContext) MarkDispatchUncertain(agentID, phase string, cause error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		log.Printf("team_execution: MarkDispatchUncertain called for %s/%s not in running map", c.teamID, agentID)
		return
	}
	entry.State = ObligationDispatchUncertain
	entry.DispatchFailurePhase = normalizeDispatchFailurePhase(phase)
	entry.DispatchFailureCode = dispatchFailureCode(cause)
	c.running[agentID] = entry
	c.persistLocked()
}

func normalizeDispatchFailurePhase(phase string) string {
	switch phase {
	case dispatchFailurePhaseDispatch, dispatchFailurePhaseTaskAuthorization, dispatchFailurePhaseTaskResolution,
		dispatchFailurePhaseTaskLookup, dispatchFailurePhaseTaskMatch, dispatchFailurePhaseTaskCreate,
		dispatchFailurePhaseTaskReconciliation,
		dispatchFailurePhaseTaskConfirmation, dispatchFailurePhaseRunReconciliation,
		dispatchFailurePhaseRunCreation, dispatchFailurePhaseTaskCancellation, dispatchFailurePhaseCapacityDeferral:
		return phase
	default:
		return "dispatch"
	}
}

func dispatchFailureCode(err error) string {
	var stageFailure *dispatchStageFailure
	if errors.As(err, &stageFailure) && stageFailure != nil {
		switch stageFailure.code {
		case "capacity_deferral_failed", "task_cancellation_failed", "task_confirmation_failed", "run_create_uncertain":
			return stageFailure.code
		}
	}
	if reason := admissionReasonFromError(err); reason != "" {
		return reason
	}
	if status, ok := AgentManagerAuthorizationStatus(err); ok {
		if status == http.StatusUnauthorized {
			return "owner_unauthorized"
		}
		return "owner_forbidden"
	}
	if errors.As(err, &stageFailure) && stageFailure != nil {
		switch stageFailure.code {
		case "task_lookup_failed", "task_terms_mismatch", "task_create_uncertain", "task_create_reconciliation_failed",
			"capacity_deferral_failed", "task_cancellation_failed", "task_confirmation_failed", "run_create_uncertain":
			return stageFailure.code
		}
	}
	if IsApprovalPending(err) {
		return "approval_pending"
	}
	if IsDispatchRejected(err) {
		return "owner_rejected"
	}
	if IsDispatchUncertain(err) {
		return "outcome_uncertain"
	}
	return "dispatch_error"
}

func dispatchFailurePhaseFromError(err error, fallback string) string {
	var stageFailure *dispatchStageFailure
	if errors.As(err, &stageFailure) && stageFailure != nil {
		switch stageFailure.phase {
		case dispatchFailurePhaseTaskLookup, dispatchFailurePhaseTaskMatch,
			dispatchFailurePhaseTaskCreate, dispatchFailurePhaseTaskReconciliation,
			dispatchFailurePhaseTaskConfirmation, dispatchFailurePhaseTaskCancellation,
			dispatchFailurePhaseRunCreation, dispatchFailurePhaseCapacityDeferral:
			return stageFailure.phase
		}
	}
	return fallback
}

// MarkApprovalPending retains a run-stage request and its exact idempotent
// dispatch intent for a later scheduled retry after the operator answers.
func (c *TeamExecutionContext) MarkApprovalPending(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.running[agentID]
	if !ok {
		return
	}
	entry.State = ObligationApprovalPending
	entry.DispatchFailurePhase = ""
	entry.DispatchFailureCode = ""
	c.running[agentID] = entry
	_ = c.persistLocked()
}

// MarkAdmissionDeferred retains the immutable dispatch after the owner proves
// that only current grant capacity prevented admission. It is deliberately a
// separate state from approval_pending and dispatch_uncertain.
func (c *TeamExecutionContext) MarkAdmissionDeferred(agentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.running[agentID]
	if !ok {
		return ErrRunningEntryNotFound
	}
	task, taskErr := parseTaskDispatchRequest(entry.TaskRequestSnapshot)
	run, runErr := createRunRequestFromSnapshot(entry.CreateRunRequestSnapshot, entry.IdempotencyKey, entry.TaskID)
	if entry.DispatchStage != DispatchStageIntent || entry.RunID != "" || entry.IdempotencyKey == "" || entry.TaskID == "" ||
		taskErr != nil || task.ID != entry.TaskID || runErr != nil || run.TaskID != entry.TaskID ||
		!validCreateRunCallerMode(entry.CreateRunCallerMode) ||
		(entry.CreateRunCallerMode == createRunCallerOriginCaller && entry.CreateRunCallerBinding == "") ||
		(entry.State != ObligationDispatchUncertain && entry.State != ObligationAdmissionDeferred) {
		return errors.New("cannot defer admission without a complete immutable dispatch intent")
	}
	previous := entry
	entry.State = ObligationAdmissionDeferred
	entry.DispatchFailurePhase = ""
	entry.DispatchFailureCode = ""
	c.running[agentID] = entry
	if err := c.persistLocked(); err != nil {
		c.running[agentID] = previous
		return fmt.Errorf("persisting deferred admission: %w", err)
	}
	return nil
}

func (c *TeamExecutionContext) isAdmissionDeferred(agentID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.running[agentID]
	return ok && entry.State == ObligationAdmissionDeferred
}

// RetryAdmissionDeferred is called only by an otherwise-eligible scheduled
// callback. It replays the exact retained request only when its persisted
// trigger origin is known to be scheduled; legacy origin is never inferred.
func (c *TeamExecutionContext) RetryAdmissionDeferred(ctx context.Context, agentID string) (bool, error) {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	c.mu.Unlock()
	if !ok || entry.AdmissionOrigin != AdmissionOriginScheduled || entry.RunID != "" ||
		(entry.State != ObligationAdmissionDeferred && entry.State != ObligationDispatchUncertain) {
		return false, nil
	}
	return true, c.ReconcileDispatch(ctx, agentID)
}

// RetryManualDispatch is called by the existing explicit manual-trigger route
// after its normal team/member/pause checks. It retries only the exact retained
// manual intent. Caller-backed intents require the same freshly verified
// binding; service-backed intents retain the service's own fresh authority.
func (c *TeamExecutionContext) RetryManualDispatch(ctx context.Context, agentID string) (bool, error) {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	c.mu.Unlock()
	if !ok || entry.AdmissionOrigin != AdmissionOriginManual || entry.RunID != "" ||
		(entry.State != ObligationAdmissionDeferred && entry.State != ObligationDispatchUncertain) {
		return false, nil
	}
	switch entry.CreateRunCallerMode {
	case createRunCallerOriginService:
		if entry.CreateRunCallerBinding != "" {
			return false, nil
		}
		// The manual route has already checked current trigger eligibility and
		// caller authority. Replay mints fresh service proof through the existing
		// owner client; the triggering caller never replaces that authority.
	case createRunCallerOriginCaller:
		proof := createRunCaller(ctx)
		if proof.origin != createRunCallerOriginCaller || entry.CreateRunCallerBinding == "" ||
			proof.binding == "" || proof.binding != entry.CreateRunCallerBinding {
			return false, nil
		}
	default:
		return false, nil
	}
	return true, c.ReconcileDispatch(ctx, agentID)
}

// MarkWatcherUnavailable keeps the run obligation after a local waiter fails
// without terminal evidence. A later scheduled callback may probe the owner
// and attach one replacement waiter; this does not release queue capacity.
func (c *TeamExecutionContext) MarkWatcherUnavailable(agentID string) {
	c.markWatcherUnavailable(agentID, "")
}

func (c *TeamExecutionContext) MarkWatcherUnavailableForRun(agentID, runID string) {
	c.markWatcherUnavailable(agentID, runID)
}

func (c *TeamExecutionContext) markWatcherUnavailable(agentID, expectedRunID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.running[agentID]
	if !ok || entry.RunID == "" || (expectedRunID != "" && entry.RunID != expectedRunID) {
		return
	}
	entry.State = ObligationWatcherUnavailable
	c.running[agentID] = entry
	_ = c.persistLocked()
}

// RetryUnavailableWatcher is called only by the ordinary schedule. It makes
// one owner status read, then either retains the slot, releases on definitive
// terminal evidence, or attaches a replacement waiter for a nonterminal run.
// There is no timer loop or background retry.
func (c *TeamExecutionContext) RetryUnavailableWatcher(ctx context.Context, agentID string) (bool, error) {
	outcome, err := c.retryUnavailableWatcher(ctx, agentID)
	return outcome != watcherRetryNone, err
}

type unavailableWatcherRetry uint8

const (
	watcherRetryNone unavailableWatcherRetry = iota
	watcherRetryHandled
	watcherRetryTerminalReconciled
)

// retryUnavailableWatcher separates a retained owner that was fully projected
// and released from retries that only reattach or remain unresolved.
func (c *TeamExecutionContext) retryUnavailableWatcher(ctx context.Context, agentID string) (unavailableWatcherRetry, error) {
	c.mu.Lock()
	entry, ok := c.running[agentID]
	if !ok || (entry.State != ObligationWatcherUnavailable && entry.State != ObligationOwnerUnreachable && entry.State != ObligationPaused) {
		c.mu.Unlock()
		return watcherRetryNone, nil
	}
	c.mu.Unlock()
	if entry.RunID == "" || c.agentClient == nil {
		return watcherRetryHandled, nil
	}
	run, err := c.agentClient.GetRun(ctx, entry.RunID)
	if err != nil {
		return watcherRetryHandled, err
	}
	if run == nil {
		return watcherRetryHandled, fmt.Errorf("run %s was not found by agent-manager", entry.RunID)
	}
	if run.ID != entry.RunID {
		return watcherRetryHandled, fmt.Errorf("agent-manager returned run %s while reconciling retained run %s", run.ID, entry.RunID)
	}
	ownerStartedAt := resolvedOwnerStartedAt(run.StartedAt, entry.OwnerStartedAt)
	if IsTerminalStatus(run.Status) {
		// Project terminal state synchronously through the same executor owner
		// used by the completion waiter. Only then release this exact slot.
		projector, supportsProjection := c.executor.(interface {
			ProjectRetainedTerminal(teamID, agentID, runID, profileKey, taskID, tag, admissionOrigin string, run *Run) retainedTerminalProjection
		})
		if supportsProjection {
			projection := projector.ProjectRetainedTerminal(c.teamID, agentID, entry.RunID, entry.ProfileKey, entry.TaskID, entry.RunTag, entry.AdmissionOrigin, run)
			if projection == retainedTerminalUnconfirmed {
				// A production projector's refusal means terminal state was not
				// durably confirmed. Do not attach a second asynchronous projection
				// path in this callback; retain the slot for ordinary retry.
				c.MarkWatcherUnavailableForRun(agentID, entry.RunID)
				return watcherRetryHandled, nil
			}
			if !c.completeMember(agentID, entry.RunID) {
				return watcherRetryHandled, nil
			}
			if projection == retainedTerminalPersisted {
				return watcherRetryTerminalReconciled, nil
			}
			// The exact old owner is terminal, but a newer heartbeat result was
			// preserved by the canonical projection. Release only this run's slot;
			// let the next scheduled callback reload gates before admitting work.
			return watcherRetryHandled, nil
		}
		// Only executors without synchronous terminal projection retain the
		// prior asynchronous observer behavior; they cannot authorize
		// same-callback admission.
		if !c.watchRetainedRun(agentID, entry.RunID, entry.ProfileKey, entry.TaskID, entry.RunTag, run.Status, ownerStartedAt, ownerTerminalTimestamp(run), entry.AdmissionOrigin) {
			// No confirmed synchronous projection or replacement waiter means the
			// owner slot remains held for the next scheduled reconciliation.
			c.MarkWatcherUnavailableForRun(agentID, entry.RunID)
		}
		return watcherRetryHandled, nil
	}
	c.mu.Lock()
	current, stillOwned := c.running[agentID]
	if !stillOwned || current.RunID != entry.RunID || (current.State != ObligationWatcherUnavailable && current.State != ObligationOwnerUnreachable && current.State != ObligationPaused) {
		c.mu.Unlock()
		return watcherRetryHandled, nil
	}
	switch {
	case IsParkedStatus(run.Status):
		current.State = ObligationPaused
	case IsNeedsReviewStatus(run.Status):
		current.State = ObligationNeedsReview
	default:
		current.State = ObligationRunning
	}
	if ownerStarted, ok := parseOwnerRunStartedAt(run.StartedAt); ok {
		current.OwnerStartedAt = ownerStarted.Format(time.RFC3339Nano)
	}
	c.running[agentID] = current
	_ = c.persistLocked()
	c.mu.Unlock()
	if !IsParkedStatus(run.Status) {
		if !c.watchRetainedRun(agentID, run.ID, entry.ProfileKey, entry.TaskID, entry.RunTag, run.Status, ownerStartedAt, ownerTerminalTimestamp(run), entry.AdmissionOrigin) {
			c.MarkWatcherUnavailableForRun(agentID, entry.RunID)
		}
	}
	return watcherRetryHandled, nil
}

func (s *TeamExecutionStore) retryUnavailableWatcher(ctx context.Context, teamID, agentID string) (unavailableWatcherRetry, error) {
	s.mu.RLock()
	tec, ok := s.contexts[teamID]
	s.mu.RUnlock()
	if !ok {
		return watcherRetryNone, nil
	}
	return tec.retryUnavailableWatcher(ctx, agentID)
}

func resolvedOwnerStartedAt(ownerValue, retainedValue string) string {
	if startedAt, ok := parseOwnerRunStartedAt(ownerValue); ok {
		return startedAt.Format(time.RFC3339Nano)
	}
	if startedAt, ok := parseOwnerRunStartedAt(retainedValue); ok {
		return startedAt.Format(time.RFC3339Nano)
	}
	return ""
}

// SetRunningRunID records the agent-manager run ID for an in-flight running
// entry. Called by Executor.Execute immediately after CreateRun returns so
// that subsequent Recover() calls can reconcile the entry against
// agent-manager. No-op (with a debug log) if the agent is not currently
// recorded as running — bookkeeping must not affect run progress.
func (c *TeamExecutionContext) SetRunningRunID(agentID, runID string, ownerStartedAt ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.running[agentID]
	if !ok {
		log.Printf("team_execution: SetRunningRunID called for %s/%s not in running map", c.teamID, agentID)
		return
	}
	if entry.RunID != runID {
		entry.OwnerStartedAt = ""
	}
	entry.RunID = runID
	entry.DispatchFailurePhase = ""
	entry.DispatchFailureCode = ""
	if len(ownerStartedAt) > 1 {
		entry.State = ownerRunObligationState(ownerStartedAt[0])
	} else {
		entry.State = ObligationRunning
	}
	startedAtValue := ""
	if len(ownerStartedAt) > 1 {
		startedAtValue = ownerStartedAt[1]
	} else if len(ownerStartedAt) > 0 {
		startedAtValue = ownerStartedAt[0]
	}
	if startedAt, ok := parseOwnerRunStartedAt(startedAtValue); ok {
		entry.OwnerStartedAt = startedAt.Format(time.RFC3339Nano)
	}
	// Binding a confirmed run identity resolves dispatch uncertainty while
	// preserving owner-reported paused/review states.
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
//   - needs_review run             -> retain as review-required ownership
//   - approval_pending run intent  -> retain the exact intent for scheduled replay
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
func (c *TeamExecutionContext) Recover(ctx context.Context) []ActiveRun {
	c.mu.Lock()

	data, err := os.ReadFile(c.queueFilePath())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("team_execution: failed to read queue for %s: %v", c.teamID, err)
		}
		c.mu.Unlock()
		return nil
	}

	var persisted persistedTeamQueue
	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("team_execution: failed to unmarshal queue for %s: %v", c.teamID, err)
		c.mu.Unlock()
		return nil
	}

	if persisted.QueuePolicy != "" {
		c.queuePolicy = persisted.QueuePolicy
	}
	if persisted.MaxConcurrentRuns > 0 {
		c.maxConcurrentRuns = persisted.MaxConcurrentRuns
	}

	c.running = make(map[string]runningEntry, len(persisted.Running))
	type terminalProjection struct {
		entry   queuedExecution
		endedAt string
	}
	terminalRuns := make([]terminalProjection, 0)
	watchers := make([]queuedExecution, 0, len(persisted.Running))
	recoveredOwners := make([]ActiveRun, 0, len(persisted.Running))
	for _, item := range persisted.Running {
		if item.DispatchStage == DispatchStagePreparing && isProvenPreDispatchPreparation(item) {
			// Keep the accepted obligation visible, but do not replay stale
			// scheduler eligibility or caller context after restart. A fresh,
			// normally-gated Enqueue must replace it before any owner request.
			c.running[item.AgentID] = runningEntry{
				DispatchStage: DispatchStagePreparing,
				ProfileKey:    item.ProfileKey,
				State:         ObligationPreparationInterrupted,
			}
			log.Printf("team_execution: retaining pre-dispatch preparation for %s/%s; fresh admission is required", c.teamID, item.AgentID)
			continue
		}
		if isProvenInterruptedPreparation(item) {
			c.running[item.AgentID] = runningEntry{
				DispatchStage:       DispatchStagePreparing,
				ProfileKey:          item.ProfileKey,
				State:               ObligationPreparationInterrupted,
				OwnerObligationID:   item.OwnerObligationID,
				OwnerStartedAt:      item.OwnerStartedAt,
				OwnerStateSnapshots: cloneRetainedOwnerStateSnapshots(item.OwnerStateSnapshots),
			}
			continue
		}
		entry, keep, terminalRun := c.reconcileRunningEntry(ctx, item)
		if !keep {
			continue
		}
		if terminalRun != nil {
			// Keep the exact slot durable until the common executor completion
			// path projects the already-observed terminal run and releases this exact run.
			item.State = terminalRun.Status
			item.OwnerStartedAt = resolvedOwnerStartedAt(terminalRun.StartedAt, item.OwnerStartedAt)
			terminalRuns = append(terminalRuns, terminalProjection{entry: item, endedAt: ownerTerminalTimestamp(terminalRun)})
		}
		if ownerStartedAt, ok := parseOwnerRunStartedAt(entry.OwnerStartedAt); ok {
			entry.OwnerStartedAt = ownerStartedAt.Format(time.RFC3339Nano)
		} else {
			entry.OwnerStartedAt = ""
		}
		entry.OwnerObligationID = item.OwnerObligationID
		entry.OwnerStateSnapshots = cloneRetainedOwnerStateSnapshots(item.OwnerStateSnapshots)
		if entry.OwnerObligationID == "" {
			entry.OwnerObligationID = heartbeatOwnerObligationID(c.teamID, item.AgentID, entry.IdempotencyKey, entry.RunID, entry.OwnerStartedAt)
		}
		c.running[item.AgentID] = entry
		if entry.RunID != "" {
			state := entry.State
			if state == "" {
				state = ObligationRunning
			}
			owner := ActiveRun{TeamID: c.teamID, AgentID: item.AgentID, RunID: entry.RunID, State: state}
			if startedAt, ok := parseOwnerRunStartedAt(entry.OwnerStartedAt); ok {
				owner.StartedAt = startedAt
				owner.ownerStartedAtKnown = true
			}
			recoveredOwners = append(recoveredOwners, owner)
		}
		if terminalRun == nil && entry.RunID != "" && entry.State != ObligationDispatchUncertain && entry.State != ObligationOwnerUnreachable && entry.State != ObligationPaused && entry.State != ObligationApprovalPending {
			ownerStatus := "RUN_STATUS_RUNNING"
			if entry.State == ObligationNeedsReview {
				ownerStatus = "RUN_STATUS_NEEDS_REVIEW"
			}
			watchers = append(watchers, queuedExecution{AgentID: item.AgentID, AdmissionOrigin: entry.AdmissionOrigin, ProfileKey: entry.ProfileKey, RunID: entry.RunID, TaskID: entry.TaskID, RunTag: entry.RunTag, State: ownerStatus, OwnerStartedAt: entry.OwnerStartedAt})
		}
		// Fresh POST retries are owned by a currently eligible explicit trigger.
		// Startup may reconcile a known RunID with GET/watch above, but must not
		// infer a missing trigger origin or replay a no-RunID intent here.
	}

	c.queue = make([]queuedExecution, 0)
	c.queued = make(map[string]bool)
	for _, item := range persisted.Queue {
		if !item.ManualAdmission {
			continue
		} // stale scheduled ticks remain discarded
		if item.CreateRunCallerMode == createRunCallerOriginService {
			// Only a service mode explicitly captured on the original queued
			// admission survives restart. Restore no credential; the worker will
			// mint a fresh service token before it can produce owner effects.
			if _, ok := c.agentClient.(*AgentManagerClient); ok {
				item.caller = withCreateRunCallerOrigin(context.Background(), "", "", createRunCallerOriginService)
				item.caller = withHeartbeatAdmissionOrigin(item.caller, item.AdmissionOrigin)
				item.State = ""
			} else {
				item.caller = nil
				item.State = "caller_required"
			}
			c.queue = append(c.queue, item)
			c.queued[item.AgentID] = true
			continue
		}
		// With AUTH-01 off (P-18) an ordinary manual intent is a stale tick as
		// before AUTH-01: the schedule, not a held intent, restarts the member.
		if !owneridentity.CreateRunCallerEnforced() {
			continue
		}
		item.caller = nil
		item.State = "caller_required"
		c.queue = append(c.queue, item)
		c.queued[item.AgentID] = true
	}

	// Write cleaned/reconciled state back to disk before attaching watchers.
	saveErr := c.persistLocked()
	c.mu.Unlock()
	if hydrator, ok := c.executor.(interface{ HydrateRecoveredRuns([]ActiveRun) }); ok {
		hydrator.HydrateRecoveredRuns(recoveredOwners)
	}
	if saveErr != nil {
		log.Printf("team_execution: recovery held because queue save failed: %v", saveErr)
		return recoveredOwners
	}
	// Terminal owner evidence discovered during recovery also needs the
	// executor's normal completion projection. Invoke watchers only after the
	// queue mutex is released; the run-scoped completion callback may re-enter this context.
	for _, terminal := range terminalRuns {
		entry := terminal.entry
		if !c.watchRetainedRun(entry.AgentID, entry.RunID, entry.ProfileKey, entry.TaskID, entry.RunTag, entry.State, entry.OwnerStartedAt, terminal.endedAt, entry.AdmissionOrigin) {
			log.Printf("team_execution: terminal run projection unavailable for %s/%s (run %s); releasing proven terminal slot", c.teamID, entry.AgentID, entry.RunID)
			c.OnMemberCompleteRun(entry.AgentID, entry.RunID)
		}
	}
	for _, retained := range watchers {
		if !c.watchRetainedRun(retained.AgentID, retained.RunID, retained.ProfileKey, retained.TaskID, retained.RunTag, retained.State, retained.OwnerStartedAt, "", retained.AdmissionOrigin) {
			c.MarkWatcherUnavailableForRun(retained.AgentID, retained.RunID)
		}
	}

	c.publishRetainedOwnerStateFacts(ctx)
	return recoveredOwners
}

// reconcileRunningEntry decides the durable obligation for one persisted
// running entry. keep is false only on definitive terminal owner evidence.
func isProvenPreDispatchPreparation(item queuedExecution) bool {
	return item.DispatchStage == DispatchStagePreparing &&
		item.RunID == "" &&
		item.State == "" &&
		item.IdempotencyKey == "" &&
		item.TaskID == "" &&
		item.TaskRequestSnapshot == "" &&
		!item.TaskConfirmed &&
		item.CreateRunRequestSnapshot == "" &&
		item.CreateRunCallerMode == "" &&
		item.CreateRunCallerBinding == "" &&
		item.RunTag == "" &&
		item.WorkloadKind == "" &&
		item.WorkloadKey == "" &&
		item.WorkloadInstance == "" &&
		item.ResetConsumedEventID == "" &&
		item.OwnerObligationID == "" &&
		item.OwnerStartedAt == "" &&
		len(item.OwnerStateSnapshots) == 0
}

func isProvenInterruptedPreparation(item queuedExecution) bool {
	if item.State != ObligationPreparationInterrupted || item.DispatchStage != DispatchStagePreparing {
		return false
	}
	base := item
	base.State = ""
	base.OwnerObligationID = ""
	base.OwnerStartedAt = ""
	base.OwnerStateSnapshots = nil
	if !isProvenPreDispatchPreparation(base) || item.OwnerStartedAt != "" {
		return false
	}
	if item.OwnerObligationID == "" {
		return len(item.OwnerStateSnapshots) == 0
	}
	if len(item.OwnerStateSnapshots) != 1 {
		return false
	}
	snapshot, ok := item.OwnerStateSnapshots[ObligationPreparationInterrupted]
	return ok && validInterruptedPreparationSnapshot(snapshot, item.OwnerObligationID)
}

func isProvenCanceledTaskDispatch(teamID string, item queuedExecution) bool {
	if item.DispatchStage != DispatchStageIntent || item.TaskConfirmed || item.RunID != "" || item.TaskID == "" || item.IdempotencyKey == "" {
		return false
	}
	task, err := parseTaskDispatchRequest(item.TaskRequestSnapshot)
	if err != nil || task.ID != item.TaskID {
		return false
	}
	if _, err := createRunRequestFromSnapshot(item.CreateRunRequestSnapshot, item.IdempotencyKey, item.TaskID); err != nil {
		return false
	}
	if !validCreateRunCallerMode(item.CreateRunCallerMode) {
		return false
	}
	switch item.CreateRunCallerMode {
	case createRunCallerOriginService, createRunCallerOriginUnqualified:
		if item.CreateRunCallerBinding != "" {
			return false
		}
	case createRunCallerOriginCaller:
		if item.CreateRunCallerBinding == "" {
			return false
		}
	}
	if item.ResetConsumedEventID != "" || item.OwnerObligationID != heartbeatOwnerObligationID(teamID, item.AgentID, item.IdempotencyKey, "", "") || item.OwnerStartedAt != "" || len(item.OwnerStateSnapshots) != 0 {
		return false
	}
	return true
}

func (c *TeamExecutionContext) reconcileRunningEntry(ctx context.Context, item queuedExecution) (runningEntry, bool, *Run) {
	if item.State == ObligationDispatchCanceled && isProvenCanceledTaskDispatch(c.teamID, item) {
		// Execute persisted this no-replay tombstone after a typed pre-send
		// cancellation, with the complete exact intent, but crashed before its
		// worker completion callback.
		return runningEntry{}, false, nil
	}
	base := runningEntry{
		DispatchStage:            item.DispatchStage,
		AdmissionOrigin:          item.AdmissionOrigin,
		ProfileKey:               item.ProfileKey,
		RunID:                    item.RunID,
		IdempotencyKey:           item.IdempotencyKey,
		TaskID:                   item.TaskID,
		TaskRequestSnapshot:      item.TaskRequestSnapshot,
		TaskConfirmed:            item.TaskConfirmed,
		DispatchFailurePhase:     item.DispatchFailurePhase,
		DispatchFailureCode:      item.DispatchFailureCode,
		CreateRunRequestSnapshot: item.CreateRunRequestSnapshot,
		CreateRunCallerMode:      item.CreateRunCallerMode,
		CreateRunCallerBinding:   item.CreateRunCallerBinding,
		RunTag:                   item.RunTag,
		WorkloadKind:             item.WorkloadKind,
		WorkloadKey:              item.WorkloadKey,
		WorkloadInstance:         item.WorkloadInstance,
		ResetConsumedEventID:     item.ResetConsumedEventID,
		OwnerObligationID:        item.OwnerObligationID,
		OwnerStartedAt:           item.OwnerStartedAt,
		OwnerStateSnapshots:      cloneRetainedOwnerStateSnapshots(item.OwnerStateSnapshots),
	}
	// A prior recovery already recorded a non-resolvable obligation. Preserve it
	// without re-probing the owner: unknown reset waits must not poll.
	needsOwnerRead := item.State == ObligationRunning || item.State == ObligationWatcherUnavailable || (item.State == ObligationDispatchUncertain && item.RunID != "")
	if !needsOwnerRead {
		base.State = item.State
		return base, true, nil
	}
	if item.RunID == "" {
		log.Printf("team_execution: retaining dispatch_uncertain obligation for %s/%s (no RunID)", c.teamID, item.AgentID)
		base.State = ObligationDispatchUncertain
		return base, true, nil
	}
	if c.agentClient == nil {
		// No client configured (e.g. unit tests not exercising reconciliation).
		// Preserve the entry — caller takes responsibility for what's on disk.
		return base, true, nil
	}
	run, err := c.agentClient.GetRun(ctx, item.RunID)
	observation := observeOwnerRun(item.RunID, run, err)
	if observation.state == ObligationOwnerUnreachable {
		log.Printf("team_execution: retaining owner_unreachable obligation for %s/%s (run %s: %v)", c.teamID, item.AgentID, item.RunID, err)
		base.State = observation.state
		return base, true, nil
	}
	if observation.state == ObligationDispatchUncertain {
		log.Printf("team_execution: retaining dispatch_uncertain obligation for %s/%s (requested run %s, owner returned %q)", c.teamID, item.AgentID, item.RunID, observation.status)
		base.State = observation.state
		return base, true, nil
	}
	if observation.terminal {
		log.Printf("team_execution: reconciling terminal heartbeat projection for %s/%s (run %s status=%s)", c.teamID, item.AgentID, item.RunID, run.Status)
		base.State = ObligationRunning
		if observation.startedAtKnown {
			base.OwnerStartedAt = observation.startedAt.Format(time.RFC3339Nano)
		}
		return base, true, run
	}
	if observation.state == ObligationPaused || observation.state == ObligationNeedsReview {
		base.State = observation.state
	}
	if observation.startedAtKnown {
		base.OwnerStartedAt = observation.startedAt.Format(time.RFC3339Nano)
	}
	return base, true, nil
}

// ClearRunning removes a single running entry for the team. Without force,
// it requires exact terminal run evidence, or a cancelled task with no current
// run and a complete empty attempt inventory. Active or unverified owners are
// preserved. If the agent isn't recorded as running, returns
// ErrRunningEntryNotFound.
func (c *TeamExecutionContext) ClearRunning(ctx context.Context, agentID string, force bool) error {
	c.mu.Lock()

	entry, ok := c.running[agentID]
	if !ok {
		c.mu.Unlock()
		return ErrRunningEntryNotFound
	}

	if !force {
		if entry.RunID == "" && (entry.TaskID == "" || entry.DispatchStage != DispatchStageIntent) {
			c.mu.Unlock()
			return &RunningOwnerUnverifiedError{TeamID: c.teamID, AgentID: agentID, Reason: "no run ID is recorded"}
		}
		client := c.agentClient
		if client == nil {
			c.mu.Unlock()
			return &RunningOwnerLookupError{TeamID: c.teamID, AgentID: agentID, RunID: entry.RunID, Err: errors.New("Agent Manager client is unavailable")}
		}
		// Release the lock for the network call, then verify the same slot still
		// belongs to this run before changing queue state.
		c.mu.Unlock()
		if entry.RunID == "" {
			if err := c.verifyCancelledTaskWithoutAttempts(ctx, client, agentID, entry.TaskID); err != nil {
				return err
			}
		} else {
			run, err := client.GetRun(ctx, entry.RunID)
			if err != nil {
				return &RunningOwnerLookupError{TeamID: c.teamID, AgentID: agentID, RunID: entry.RunID, Err: err}
			}
			if run == nil || run.ID != entry.RunID {
				return &RunningOwnerUnverifiedError{TeamID: c.teamID, AgentID: agentID, RunID: entry.RunID, Reason: "Agent Manager did not confirm the exact run ID"}
			}
			if !IsTerminalStatus(run.Status) {
				return &RunningStillActiveError{TeamID: c.teamID, AgentID: agentID, RunID: entry.RunID, Status: run.Status}
			}
		}
		c.mu.Lock()
		current, stillPresent := c.running[agentID]
		if !stillPresent {
			c.mu.Unlock()
			return ErrRunningEntryNotFound
		}
		if current.RunID != entry.RunID || current.TaskID != entry.TaskID || current.IdempotencyKey != entry.IdempotencyKey || current.DispatchStage != entry.DispatchStage {
			c.mu.Unlock()
			return &RunningOwnerUnverifiedError{TeamID: c.teamID, AgentID: agentID, RunID: entry.RunID, Reason: "the retained slot changed while owner status was checked"}
		}
	}

	delete(c.running, agentID)
	dispatches, persistErr := c.dispatchAvailableLockedWithPersistence()
	if persistErr != nil {
		// The dispatch helper has rolled back its promotion attempt. Restore the
		// explicitly-cleared entry too, so memory and the durable queue continue
		// to agree and no queued execution starts without a committed removal.
		c.running[agentID] = entry
		c.mu.Unlock()
		return fmt.Errorf("persist queue after clearing %s/%s: %w", c.teamID, agentID, persistErr)
	}
	c.mu.Unlock()
	c.startExecutions(dispatches)
	return nil
}

// Cancellation is owned by Agent Manager and fences future task admission.
// Absence alone, a terminal-looking task with attempts, or a partial page is
// insufficient to release a task-only intent whose run response may be lost.
func (c *TeamExecutionContext) verifyCancelledTaskWithoutAttempts(ctx context.Context, client AgentClient, agentID, taskID string) error {
	unverified := func(reason string) error {
		return &RunningOwnerUnverifiedError{TeamID: c.teamID, AgentID: agentID, Reason: reason}
	}
	task, err := client.GetTask(ctx, taskID)
	if err != nil {
		return &RunningOwnerLookupError{TeamID: c.teamID, AgentID: agentID, Err: err}
	}
	if task == nil || task.ID != taskID || task.Status != "TASK_STATUS_CANCELLED" || task.CurrentRunID != "" {
		return unverified("Agent Manager must confirm the exact cancelled task with no current run")
	}
	attempts, err := client.ListRuns(ctx, ListRunsOptions{TaskID: taskID, Limit: 1})
	if err != nil {
		return &RunningOwnerLookupError{TeamID: c.teamID, AgentID: agentID, Err: err}
	}
	if attempts == nil || attempts.HasMore || attempts.Total != 0 || len(attempts.Runs) != 0 {
		return unverified("Agent Manager did not confirm a complete empty task attempt inventory")
	}
	return nil
}

func (c *TeamExecutionContext) persistLocked() error {
	running := make([]queuedExecution, 0, len(c.running))
	for agentID, entry := range c.running {
		running = append(running, queuedExecution{
			AgentID:                  agentID,
			DispatchStage:            entry.DispatchStage,
			AdmissionOrigin:          entry.AdmissionOrigin,
			ProfileKey:               entry.ProfileKey,
			RunID:                    entry.RunID,
			State:                    entry.State,
			IdempotencyKey:           entry.IdempotencyKey,
			TaskID:                   entry.TaskID,
			TaskRequestSnapshot:      entry.TaskRequestSnapshot,
			TaskConfirmed:            entry.TaskConfirmed,
			DispatchFailurePhase:     entry.DispatchFailurePhase,
			DispatchFailureCode:      entry.DispatchFailureCode,
			CreateRunRequestSnapshot: entry.CreateRunRequestSnapshot,
			CreateRunCallerMode:      entry.CreateRunCallerMode,
			CreateRunCallerBinding:   entry.CreateRunCallerBinding,
			RunTag:                   entry.RunTag,
			WorkloadKind:             entry.WorkloadKind,
			WorkloadKey:              entry.WorkloadKey,
			WorkloadInstance:         entry.WorkloadInstance,
			ResetConsumedEventID:     entry.ResetConsumedEventID,
			OwnerObligationID:        entry.OwnerObligationID,
			OwnerStartedAt:           entry.OwnerStartedAt,
			OwnerStateSnapshots:      cloneRetainedOwnerStateSnapshots(entry.OwnerStateSnapshots),
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

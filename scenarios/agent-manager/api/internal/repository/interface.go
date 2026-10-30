// Package repository provides persistence interfaces for domain entities.
//
// This package defines the SEAM for data persistence. All domain entities
// are persisted through these interfaces, enabling:
// - Easy testing with mock/stub repositories
// - Database migrations without changing domain logic
package repository

import (
	"context"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Common Types
// -----------------------------------------------------------------------------

// ListFilter specifies pagination for list operations.
type ListFilter struct {
	Limit  int
	Offset int
}

// RunListFilter extends ListFilter with run-specific filters.
type RunListFilter struct {
	ListFilter
	// RunnerPIDs narrows owner checks to exact process IDs already observed in a peer ancestry.
	RunnerPIDs []int
	// NonTerminalOnly narrows lifecycle-owner lookups to statuses that can
	// still own a live executor. It is used by security-sensitive peer/run
	// binding and avoids paging historical terminal runs.
	NonTerminalOnly           bool
	TaskID                    *uuid.UUID
	AgentProfileID            *uuid.UUID
	Status                    *domain.RunStatus
	EndedFrom                 *time.Time
	EndedTo                   *time.Time
	TagPrefix                 string // Filter runs by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
	ScopePrefix               string // Filter runs by the joined task's scope_path prefix (e.g., "scenarios/agent-manager" to drain runs targeting a scenario)
	InvestigatesRunID         *uuid.UUID
	AppliesInvestigationRunID *uuid.UUID
	ParentRunID               *uuid.UUID // Filter to direct child runs of one parent run
	TopLevelOnly              bool       // Exclude runs that are a sub-agent of another run
	Origin                    string     // agent_manager | operator_sessions | workflows; empty matches every origin
}

// -----------------------------------------------------------------------------
// ProfileRepository - AgentProfile persistence
// -----------------------------------------------------------------------------

// ProfileRepository provides persistence for AgentProfile entities.
type ProfileRepository interface {
	// Create stores a new agent profile.
	Create(ctx context.Context, profile *domain.AgentProfile) error

	// Get retrieves a profile by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error)

	// GetByName retrieves a profile by name.
	GetByName(ctx context.Context, name string) (*domain.AgentProfile, error)

	// GetByKey retrieves a profile by key.
	GetByKey(ctx context.Context, key string) (*domain.AgentProfile, error)

	// List retrieves profiles with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.AgentProfile, error)

	// Update modifies an existing profile.
	Update(ctx context.Context, profile *domain.AgentProfile) error

	// Delete removes a profile by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// WorkflowRepository stores immutable, digest-addressed workflow revisions.
// ActivateBatch validates and activates an entire scenario reload atomically;
// a failed reload therefore leaves every previously active revision intact.
type WorkflowRepository interface {
	ActivateBatch(ctx context.Context, revisions []*domain.WorkflowRevision) error
	GetActive(ctx context.Context, owner, key string) (*domain.WorkflowRevision, error)
	GetByDigest(ctx context.Context, digest string) (*domain.WorkflowRevision, error)
	List(ctx context.Context, owner, key string, filter ListFilter) ([]*domain.WorkflowRevision, error)
}

type WorkflowCommit struct {
	ExpectedVersion int64
	Execution       *domain.WorkflowExecution
	Attempt         *domain.WorkflowNodeAttempt
	Attempts        []*domain.WorkflowNodeAttempt
	Journal         []*domain.WorkflowJournalEntry
}

// WorkflowExecutionListFilter bounds operator history queries. Empty owner,
// workflow key, and status values mean "all"; pagination is always bounded by
// the repository implementation.
type WorkflowExecutionListFilter struct {
	ListFilter
	Owner       string
	WorkflowKey string
	Status      domain.WorkflowExecutionStatus
}

// WorkflowExecutionRepository is the durable interpreter commit boundary.
// Commit is compare-and-swap and atomically persists execution state, one
// attempt projection, and append-only journal entries.
type WorkflowExecutionRepository interface {
	Create(ctx context.Context, execution *domain.WorkflowExecution, initial *domain.WorkflowJournalEntry) error
	Get(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowExecution, error)
	List(ctx context.Context, filter WorkflowExecutionListFilter) ([]*domain.WorkflowExecution, error)
	Commit(ctx context.Context, commit WorkflowCommit) (bool, error)
	GetAttemptByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowNodeAttempt, error)
	ListAttempts(ctx context.Context, executionID uuid.UUID) ([]*domain.WorkflowNodeAttempt, error)
	// ExecutionIDForRun resolves the workflow execution that dispatched the
	// given run through a node attempt. Returns uuid.Nil (nil error) when the
	// run does not belong to any workflow attempt — the common non-workflow run.
	ExecutionIDForRun(ctx context.Context, runID uuid.UUID) (uuid.UUID, error)
	ListJournal(ctx context.Context, executionID uuid.UUID, afterSequence int64, limit int) ([]*domain.WorkflowJournalEntry, error)
	ListRecoverable(ctx context.Context, limit int) ([]*domain.WorkflowExecution, error)
}

// -----------------------------------------------------------------------------
// TaskRepository - Task persistence
// -----------------------------------------------------------------------------

// TaskRepository provides persistence for Task entities.
type TaskRepository interface {
	// Create stores a new task.
	Create(ctx context.Context, task *domain.Task) error

	// Get retrieves a task by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Task, error)

	// List retrieves tasks with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.Task, error)

	// ListByStatus retrieves tasks with a specific status.
	ListByStatus(ctx context.Context, status domain.TaskStatus, filter ListFilter) ([]*domain.Task, error)

	// Update modifies an existing task.
	Update(ctx context.Context, task *domain.Task) error
	// UpdateQueuedBrief updates only title and description when the exact task
	// is still queued, undispatched, and owned by createdBy. It returns false
	// when the owner/status fence no longer matches.
	UpdateQueuedBrief(ctx context.Context, id uuid.UUID, createdBy, expectedTitle, expectedDescription, title, description string) (bool, error)
	// Delete removes a task by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// -----------------------------------------------------------------------------
// RunRepository - Run persistence
// -----------------------------------------------------------------------------

// RunRepository provides persistence for Run entities.
type RunRepository interface {
	// Create stores a run that is not claiming a Task attempt (for example an import).
	Create(ctx context.Context, run *domain.Run) error
	// CreateTaskAttempt atomically claims the Task and inserts the Run/creation
	// receipt only if the prompt-bearing fields still match expectedTaskSnapshot.
	// That snapshot must be the persisted task before admission-time normalization
	// or recovery overrides. expectedCurrentRunID is nil for an unowned queued or
	// legacy task with no live historical attempt, or the exact terminal attempt
	// being replaced.
	CreateTaskAttempt(ctx context.Context, task, expectedTaskSnapshot *domain.Task, run *domain.Run, expectedCurrentRunID *uuid.UUID) error
	// CancelTaskAttempt atomically records explicit Task cancellation and fences
	// the exact current Run. Pending Runs become terminal before recovery can
	// enqueue them; active Runs receive the ordinary cancellation intent and
	// must still be stopped through the existing StopRun path.
	CancelTaskAttempt(ctx context.Context, taskID uuid.UUID, runID *uuid.UUID, activeMayHaveStarted bool, at time.Time) (domain.RunStatus, bool, error)
	// EnsureCurrentRun accepts only the exact requested Run. A legacy NULL
	// pointer is lazily bound only when no competing nonterminal Run exists.
	EnsureCurrentRun(ctx context.Context, taskID, runID uuid.UUID, at time.Time) (bool, error)
	// AdmitCurrentRun reactivates only the exact current Run while its owning
	// Task remains nonterminal, preventing stale ResumeRun snapshots from
	// resurrecting a canceled attempt.
	AdmitCurrentRun(ctx context.Context, run *domain.Run, expectedStatus domain.RunStatus, at time.Time) (bool, error)

	// Get retrieves a run by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Run, error)

	// List retrieves runs with optional filtering.
	List(ctx context.Context, filter RunListFilter) ([]*domain.Run, error)

	// Count returns how many runs match filter, ignoring Limit and Offset.
	Count(ctx context.Context, filter RunListFilter) (int, error)

	// ListByTask retrieves runs for a specific task.
	ListByTask(ctx context.Context, taskID uuid.UUID, filter ListFilter) ([]*domain.Run, error)

	// Update modifies an existing run. A transition into an active status is
	// checked against the durable effort coordinator owner in the same
	// transaction, including retained-session continuation lifecycle updates.
	Update(ctx context.Context, run *domain.Run) error
	// UpdateInteractiveContinuation atomically commits a new invocation stamp for
	// a running, parked, or terminal interactive run. The caller supplies the
	// exact prior status and invocation observed before mutation so the database
	// can reject a stale continuation without weakening ordinary Update's
	// current-invocation guard.
	UpdateInteractiveContinuation(ctx context.Context, run *domain.Run, expectedStatus domain.RunStatus, expectedInvocationStartedAt *time.Time) error

	// TouchHeartbeat atomically updates ONLY a run's last_heartbeat (and
	// updated_at), and ONLY while it is still in an actively-executing status
	// (running or starting). It returns true when a row was updated.
	//
	// This exists so a heartbeat tick can never resurrect a run that has moved
	// on. The heartbeat loop holds an in-memory *Run whose Status is whatever it
	// was when the loop started; a full-row Update from that stale pointer would
	// clobber a concurrent park/stop transition (e.g. running→parked written by
	// another goroutine), rewriting status back to running. The status-guarded,
	// single-column update closes that race at the SQL layer (no TOCTOU).
	TouchHeartbeat(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)

	// BeginInteractiveInvocation atomically advances the invocation evidence for
	// the exact current interactive owner before its first prompt or a continuation
	// is sent. The expected status and prior stamp fence stale callers. It updates
	// no transcript, heartbeat, or task fields, so concurrent tail callbacks
	// cannot overwrite their stream state.
	BeginInteractiveInvocation(ctx context.Context, run *domain.Run, expectedStatus domain.RunStatus, expectedStartedAt *time.Time, at time.Time) (bool, error)

	// RequestCancellation atomically stamps the durable cancellation intent on
	// a run that is still actively executing (running or starting). It returns
	// true when the intent was newly recorded and false when it was already
	// stamped or the run is no longer active. Like TouchHeartbeat, the guard is
	// applied in SQL so a stale caller cannot stamp a terminal run, and the
	// intent is monotonic (a repeat stop request is a no-op). The persisted
	// stamp lets a delayed process exit reconcile to cancelled exactly once
	// instead of resurrecting the run to a different terminal state.
	RequestCancellation(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)

	// UpdateRunnerStreamState atomically persists ONLY the runner streaming
	// columns (session id, runner pid/pgid, transcript path/cursor/seq) from the
	// supplied run, and ONLY while it is still actively executing (running or
	// starting). It returns true when a row was updated.
	//
	// Same race class as TouchHeartbeat: the executor's OnAdvance/OnProcessStart/
	// OnSessionID transcript callbacks fire on every agent output chunk and hold
	// an in-memory *Run whose Status is the stale "running" value. A full-row
	// Update from that pointer during the park turn-end grace window would rewrite
	// a just-persisted running→parked transition back to running (clobbering the
	// park before detectParked ever reads it). The status-guarded targeted update
	// closes that race at the SQL layer (no TOCTOU): once the run is parked/
	// terminal, the streaming write no-ops instead of resurrecting it.
	UpdateRunnerStreamState(ctx context.Context, run *domain.Run) (bool, error)

	// Delete removes a terminal run only if the observed status and lifecycle version still match.
	Delete(ctx context.Context, id uuid.UUID, expectedStatus domain.RunStatus, expectedLifecycleVersion int64) error

	// CountByStatus returns the count of runs by status.
	CountByStatus(ctx context.Context, status domain.RunStatus) (int, error)

	// CountCapacityByStatus counts runs in status that hold a dispatch slot.
	// Attached and imported runs are excluded: they record sessions Agent
	// Manager did not launch, whose resources the agent session slice already
	// bounds (an indexed external transcript is `running` while it grows).
	CountCapacityByStatus(ctx context.Context, status domain.RunStatus) (int, error)

	// GetByTokenHash retrieves a run by its identity token hash.
	// Returns nil, nil if no matching run is found.
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Run, error)

	// GetByImportProvenance returns the run that adopted an external source
	// session, or nil when that session has not been imported.
	GetByImportProvenance(ctx context.Context, sourceHarness, sourceSessionID string) (*domain.Run, error)
	// FindImportProvenance returns at most two matching IDs. Two means ambiguous;
	// unlike GetByImportProvenance it must not silently choose one duplicate.
	FindImportProvenance(ctx context.Context, sourceHarness, sourceSessionID string) ([]uuid.UUID, error)

	// ListByWebConsoleSessionIDs returns every run whose web_console_session_id
	// is one of sessionIDs (several runs may share one session). Unknown and
	// empty ids contribute nothing; an empty input returns no runs.
	ListByWebConsoleSessionIDs(ctx context.Context, sessionIDs []string) ([]*domain.Run, error)

	// ListRunTranscriptPaths maps each transcript path an interactive or
	// codec-pipe run reads itself to that run, so the Transcript Indexer can
	// register those files as run-owned instead of counting them again.
	ListRunTranscriptPaths(ctx context.Context) (map[string]uuid.UUID, error)

	// GetByIdempotencyKey resolves the run durably created under a creation
	// idempotency key, or nil when no creation was accepted. A retained receipt
	// with a missing run returns an error, never permission to create again.
	// It exists so a
	// caller that lost the accepted start response can reconcile the original
	// dispatch from durable owner state instead of starting a duplicate.
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Run, error)

	// GetCreationReceipt returns immutable accepted-creation identity, including
	// when the original run was deleted. Nil means no retained acceptance.
	GetCreationReceipt(ctx context.Context, key string) (*domain.RunCreationReceipt, error)
}

// EffortSignalCursorRepository stores the per-effort sequence positions used
// by a retained orchestrator. Cursor writes are narrow so owner check-ins do
// not rewrite unrelated live Run state.
type EffortSignalCursorRepository interface {
	// QueueEffortSignalCursor raises the queued high-water without moving the
	// delivered position. It returns false when the run no longer owns the exact
	// active orchestrator reference. An absent cursor coalesces one latest-row
	// catch-up because preexisting native consumption cannot be proven.
	QueueEffortSignalCursor(ctx context.Context, runID uuid.UUID, effortRef string, through int64) (domain.EffortSignalCursor, bool, error)
}

// RunRecoveryAttacher records ownership of an existing live executor. The
// lifecycle/status/cancellation fence and field-limited write are atomic:
// attaching cannot replace a run or overwrite concurrent progress/stream data.
type RunRecoveryAttacher interface {
	AttachRecovery(ctx context.Context, id uuid.UUID, lifecycleVersion int64, at time.Time) (bool, error)
}

// AttachedRunRecoveryRepository corrects only an observed interrupted attached
// row after its owner proves the same live process. The complete expected
// binding and lifecycle fence prevent cancellation or identity races.
type AttachedRunRecoveryRepository interface {
	RestoreAttachedRun(context.Context, *domain.Run, time.Time) (bool, error)
}

// RunRecoveryProcessIdentityClearer retires a runner PID/PGID from a terminal
// run after recovery has positively established that the executor is absent.
// The guarded write prevents a stale recovery observation from clearing a
// newly attached executor.
type RunRecoveryProcessIdentityClearer interface {
	ClearTerminalRunnerProcessIdentity(ctx context.Context, id uuid.UUID, lifecycleVersion, ownerEpoch, runnerPID, runnerPGID int64) (bool, error)
}

// RunRecoveryOwnerClaimer atomically transfers the right to advance a live
// run to a new Agent Manager owner. The returned epoch must be carried by all
// subsequent lifecycle, stream, and finalization writes.
type RunRecoveryOwnerClaimer interface {
	ClaimRecoveryOwnership(ctx context.Context, id uuid.UUID, lifecycleVersion int64, ownerIdentity string, at time.Time) (int64, bool, error)
}

// RunFreshRecoveryClaimer consumes a source lifecycle exactly once before a
// replacement may have effects. The request hash never expires or resets when
// the replacement or the idempotency cache is deleted. Only a definitive
// pre-effect refusal permits guarded release; missing acceptance alone does not.
type RunFreshRecoveryClaimer interface {
	ClaimFreshRecovery(ctx context.Context, sourceID uuid.UUID, lifecycleVersion int64, requestHash string, allowCancelled bool) (bool, error)
	GetFreshRecoveryClaim(ctx context.Context, sourceID uuid.UUID) (string, error)
	ReleaseFreshRecoveryClaim(ctx context.Context, sourceID uuid.UUID, claimedLifecycleVersion int64, requestHash string) (bool, error)
}

// RunLabelUpdater is an intentionally narrow maintenance seam. Label
// backfills must not use RunRepository.Update because that would rewrite the
// complete run row, including identity and attribution fields.
type RunLabelUpdater interface {
	UpdateRunLabel(ctx context.Context, id uuid.UUID, label string, source domain.RunLabelSource) error
}

// RunSubjectUpdater updates only the derived subject projection. Keeping this
// seam separate prevents analytics projection from overwriting live run state.
type RunSubjectUpdater interface {
	UpdateRunSubject(ctx context.Context, id uuid.UUID, subject []string) error
}

// -----------------------------------------------------------------------------
// PolicyRepository - Policy persistence
// -----------------------------------------------------------------------------

// PolicyRepository provides persistence for Policy entities.
type PolicyRepository interface {
	// Create stores a new policy.
	Create(ctx context.Context, policy *domain.Policy) error

	// Get retrieves a policy by ID.
	Get(ctx context.Context, id uuid.UUID) (*domain.Policy, error)

	// List retrieves policies with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*domain.Policy, error)

	// ListEnabled retrieves only enabled policies.
	ListEnabled(ctx context.Context) ([]*domain.Policy, error)

	// Update modifies an existing policy.
	Update(ctx context.Context, policy *domain.Policy) error

	// Delete removes a policy by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByScope finds policies matching a scope pattern.
	FindByScope(ctx context.Context, scopePath string) ([]*domain.Policy, error)
}

// -----------------------------------------------------------------------------
// CheckpointRepository - Run checkpoint persistence
// -----------------------------------------------------------------------------

// CheckpointRepository provides persistence for run checkpoints.
// Checkpoints enable safe interruption and resumption of runs.
type CheckpointRepository interface {
	// Save stores a checkpoint, overwriting any existing checkpoint for the run.
	Save(ctx context.Context, checkpoint *domain.RunCheckpoint) error

	// Get retrieves the latest checkpoint for a run.
	Get(ctx context.Context, runID uuid.UUID) (*domain.RunCheckpoint, error)

	// Delete removes all checkpoints for a run.
	Delete(ctx context.Context, runID uuid.UUID) error

	// ListStale finds runs with checkpoints older than the given duration.
	// This helps identify runs that may have stalled.
	ListStale(ctx context.Context, olderThan time.Duration) ([]*domain.RunCheckpoint, error)

	// Heartbeat updates the last heartbeat time for a checkpoint.
	Heartbeat(ctx context.Context, runID uuid.UUID) error
}

// -----------------------------------------------------------------------------
// IdempotencyRepository - Idempotency record persistence
// -----------------------------------------------------------------------------

// IdempotencyRepository provides persistence for idempotency records.
// This enables replay-safe operations that can be safely retried.
type IdempotencyRepository interface {
	// Check looks up an existing idempotency record.
	// Returns nil, nil if no record exists.
	Check(ctx context.Context, key string) (*domain.IdempotencyRecord, error)

	// Reserve creates a pending idempotency record.
	// Returns an error if the key already exists.
	Reserve(ctx context.Context, key string, ttl time.Duration) (*domain.IdempotencyRecord, error)

	// Complete marks an idempotency record as complete with the result.
	Complete(ctx context.Context, key string, entityID uuid.UUID, entityType string, response []byte) error

	// Fail marks an idempotency record as failed (allows retry).
	Fail(ctx context.Context, key string) error

	// CleanupExpired removes expired idempotency records.
	CleanupExpired(ctx context.Context) (int, error)
}

// ContinuationIdempotencyRepository provides the stronger receipt semantics
// required by run continuation, whose input may have reached an external
// runner before the caller receives a response. Implementations must retain
// typed continuation records past expiry and must never reclaim a pending
// continuation by timeout alone.
type ContinuationIdempotencyRepository interface {
	// CheckContinuation reads a receipt regardless of its expiry. A legacy
	// untyped pending row is returned so the owner can fail closed on ambiguity.
	CheckContinuation(ctx context.Context, key string) (*domain.IdempotencyRecord, error)

	// ReserveContinuation atomically binds a fresh/pre-effect-failed key to the
	// exact run and request hash before effects. acquired is true only when this
	// call acquired the key; otherwise record describes the existing owner.
	ReserveContinuation(ctx context.Context, key string, runID uuid.UUID, requestHash []byte, ttl time.Duration) (record *domain.IdempotencyRecord, acquired bool, err error)

	// CompleteContinuation records confirmed completion only if the pending
	// receipt still has the exact run and request identity supplied here.
	CompleteContinuation(ctx context.Context, key string, runID uuid.UUID, requestHash []byte) error

	// FailContinuation makes a receipt reusable only after the caller proves
	// that no continuation effect occurred.
	FailContinuation(ctx context.Context, key string, runID uuid.UUID, requestHash []byte) error
}

// -----------------------------------------------------------------------------
// InvestigationSettingsRepository - Investigation settings persistence
// -----------------------------------------------------------------------------

// InvestigationSettingsRepository provides persistence for investigation settings.
// This is a singleton table - only one row exists.
type InvestigationSettingsRepository interface {
	// Get retrieves the current investigation settings.
	// Returns default settings if none exist.
	Get(ctx context.Context) (*domain.InvestigationSettings, error)

	// Update modifies the investigation settings.
	Update(ctx context.Context, settings *domain.InvestigationSettings) error

	// Reset restores settings to defaults.
	Reset(ctx context.Context) error
}

// StatsRepository ranks models by recorded run usage for the pricing model
// picker. Run analytics (status, duration, cost, breakdowns) are Measures
// procedures over the invocation read model; they are not served here.
type StatsRepository interface {
	// GetPopularModels returns the most used models by run count since a time.
	GetPopularModels(ctx context.Context, since time.Time, limit int) ([]string, error)

	// GetRecentModels returns recently used models ordered by most recent use (system-wide).
	GetRecentModels(ctx context.Context, limit int) ([]string, error)
}

// AttachedIdentityRenewalRepository is the narrow persistence seam for
// same-run ordinary identity rotation. It is optional so unrelated in-memory
// RunRepository implementations do not acquire renewal authority implicitly.
type AttachedIdentityRenewalRepository interface {
	GetByIdentityRenewalAnchorHash(context.Context, string) (*domain.Run, error)
	RenewAttachedIdentity(context.Context, *domain.Run, string, string, time.Time) (bool, error)
}

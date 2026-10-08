// Responsibility: retain types declarations within their original package.
package domain

import (
	"encoding/json"
	"strings"
	"time"

	"agent-manager/internal/tokenaccounting"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

// RunLabelSource records how the human-readable run label was obtained.
// Empty is retained only for legacy rows; new runs must set both Label and
// LabelSource.
type RunLabelSource string

const (
	// RunLabelSourceHarness is the title the coding-agent harness itself wrote
	// (claude's ai-title record, for example).
	RunLabelSourceHarness RunLabelSource = "harness"
	// RunLabelSourceDerived is taken verbatim from session content, such as the
	// first non-injected user prompt.
	RunLabelSourceDerived RunLabelSource = "derived"
	// RunLabelSourceGenerated means an inference provider wrote the label from
	// session content. It must never be applied to a deterministic fallback:
	// consumers filter on it expecting prose about the work.
	RunLabelSourceGenerated RunLabelSource = "generated"
	// RunLabelSourcePlaceholder means nothing named the work — no harness
	// title, no usable session content, and no successful generation. The label
	// identifies the run but says nothing about it, so analysis that needs a
	// subject must treat these as unlabelled rather than as generated prose.
	RunLabelSourcePlaceholder RunLabelSource = "placeholder"
	// RunLabelSourceManual is an operator-supplied label.
	RunLabelSourceManual RunLabelSource = "manual"
)

// RunCreationReceipt records persistence of the original creation, not runner
// completion. It survives run/task deletion and ordinary replay-cache expiry.
// It is read-only evidence and never authorizes another executor.
type RunCreationReceipt struct {
	IdempotencyKey string
	RunID          uuid.UUID
	TaskID         uuid.UUID
	OwnerSubject   string
	CreatedAt      time.Time
}

type Run struct {
	// LifecycleVersion fences snapshots from an earlier status/continuation.
	// Repository-owned; heartbeat and stream updates do not advance it.
	LifecycleVersion int64 `json:"-" db:"lifecycle_version"`
	// OwnerIdentity and OwnerEpoch identify the current Agent Manager process
	// that may advance this run. OwnerEpoch is claimed transactionally during
	// recovery so a replaced owner cannot continue writing with an old
	// snapshot, even when the run status itself remains running.
	OwnerIdentity  string     `json:"-" db:"owner_identity"`
	OwnerEpoch     int64      `json:"-" db:"owner_epoch"`
	ID             uuid.UUID  `json:"id" db:"id"`
	TaskID         uuid.UUID  `json:"taskId,omitempty" db:"task_id"`
	AgentProfileID *uuid.UUID `json:"agentProfileId,omitempty" db:"agent_profile_id"` // Optional if inline config provided

	// Custom tag for identification (defaults to ID if not set)
	// Used for agent tracking, log filtering, and external process identification
	Tag             string                       `json:"tag,omitempty" db:"tag"`
	Label           string                       `json:"label,omitempty" db:"label"`
	LabelSource     RunLabelSource               `json:"labelSource,omitempty" db:"label_source"`
	Subject         []string                     `json:"subject,omitempty" db:"subject"`
	OwnerSubject    string                       `json:"ownerSubject,omitempty" db:"owner_subject"`
	OwnerScopes     []string                     `json:"ownerScopes" db:"owner_scopes"`
	RequestedScopes []string                     `json:"requestedScopes" db:"requested_scopes"`
	OwnerExpiresAt  *time.Time                   `json:"ownerExpiresAt,omitempty" db:"owner_expires_at"`
	WorkReferences  []*eventdomain.WorkReference `json:"workReferences,omitempty" db:"work_references"`
	Workload        WorkloadRef                  `json:"workload,omitempty" db:"workload"`
	Billing         BillingSnapshot              `json:"billing,omitempty" db:"billing"`
	// Sandbox integration
	SandboxID     *uuid.UUID     `json:"sandboxId,omitempty" db:"sandbox_id"`
	RunMode       RunMode        `json:"runMode" db:"run_mode"`
	SandboxConfig *SandboxConfig `json:"sandboxConfig,omitempty" db:"sandbox_config"`
	// ExecutionMode selects the CLI-driving substrate; empty defaults to codec-pipe. Orthogonal to RunMode.
	ExecutionMode    ExecutionMode `json:"executionMode,omitempty" db:"execution_mode"`
	HarnessKind      string        `json:"harnessKind,omitempty" db:"harness_kind"`
	HarnessSessionID string        `json:"harnessSessionId,omitempty" db:"harness_session_id"`

	// GoalDelivery records how the completion objective reached the harness:
	// native_verified, native_unverified, prompt_carried, or empty.
	GoalDelivery string `json:"goalDelivery,omitempty" db:"goal_delivery"`
	// Read-only projection from qualified runner events. These fields are not
	// persisted on the run row, so coordinator heartbeats cannot change them.
	ObservedGoalStatus string     `json:"observedGoalStatus,omitempty"`
	ObservedGoalAt     *time.Time `json:"observedGoalAt,omitempty"`
	ProviderActivityAt *time.Time `json:"providerActivityAt,omitempty"`

	// TerminalClass and StopReason are the typed terminal pair every terminal
	// run carries so Swarm can decide finalization versus resume without
	// parsing error strings. LastHandoff carries the last structured handoff or
	// assistant message for a resume prompt.
	TerminalClass RunTerminalClass `json:"terminalClass,omitempty" db:"terminal_class"`
	StopReason    RunStopReason    `json:"stopReason,omitempty" db:"stop_reason"`
	LastHandoff   string           `json:"lastHandoff,omitempty" db:"last_handoff"`

	// WebConsoleSessionID is the id of the web-console session hosting the
	// interactive agent CLI, set only for ExecutionModeInteractive runs. It
	// backs the run-detail deep link to the live session and routes the
	// interactive Continue/Stop terminal input + session teardown.
	WebConsoleSessionID string `json:"webConsoleSessionId,omitempty" db:"web_console_session_id"`

	// WebConsoleSessionURL is the resolved deep link to the live web-console
	// session (computed at read time from WebConsoleSessionID + the web-console
	// UI base, not persisted). Empty for non-interactive runs and when the
	// web-console UI base cannot be resolved server-side.
	WebConsoleSessionURL string `json:"webConsoleSessionUrl,omitempty"`

	// Execution state
	Status    RunStatus  `json:"status" db:"status"`
	StartedAt *time.Time `json:"startedAt,omitempty" db:"started_at"`
	// InteractiveInvocationStartedAt identifies the current admitted interactive
	// turn. Persist it with continuation admission; original StartedAt is history.
	InteractiveInvocationStartedAt *time.Time `json:"interactiveInvocationStartedAt,omitempty" db:"interactive_invocation_started_at"`
	EndedAt                        *time.Time `json:"endedAt,omitempty" db:"ended_at"`

	// CancelRequestedAt is the durable cancellation intent for a run. An owner
	// stop request stamps it before the runner process is terminated, so a
	// process that exits after the request reconciles to cancelled instead of
	// resurrecting the run to a different terminal state. It is monotonic:
	// once stamped it is never cleared, and a repeat stop request is a no-op.
	// Nil for runs with no stop request.
	CancelRequestedAt *time.Time `json:"cancelRequestedAt,omitempty" db:"cancel_requested_at"`

	// GoalID is retained as a stable historical cohort key. The degenerate
	// self-reported goal status field is intentionally not part of Run.
	GoalID string `json:"goalId,omitempty" db:"goal_id"`

	// Progress tracking (for resumption and visibility)
	Phase            RunPhase   `json:"phase" db:"phase"`
	LastCheckpointID *uuid.UUID `json:"lastCheckpointId,omitempty" db:"last_checkpoint_id"`
	LastHeartbeat    *time.Time `json:"lastHeartbeat,omitempty" db:"last_heartbeat"`
	ProgressPercent  int        `json:"progressPercent" db:"progress_percent"`

	// Idempotency (for replay safety)
	IdempotencyKey string `json:"idempotencyKey,omitempty" db:"idempotency_key"`

	// Results
	// Result is the canonical, provenance-bearing terminal output projection.
	// Summary is a compatibility view derived from Result and must never be
	// independently authored for new runner completions.
	Result   *RunResult  `json:"result,omitempty" db:"run_result"`
	Summary  *RunSummary `json:"summary,omitempty" db:"summary"`
	ErrorMsg string      `json:"errorMsg,omitempty" db:"error_msg"`
	ExitCode *int        `json:"exitCode,omitempty" db:"exit_code"`

	// Approval workflow
	ApprovalState ApprovalState `json:"approvalState" db:"approval_state"`
	ApprovedBy    string        `json:"approvedBy,omitempty" db:"approved_by"`
	ApprovedAt    *time.Time    `json:"approvedAt,omitempty" db:"approved_at"`

	// Post-run sandbox finalization. This tracks apply/checkpoint effects
	// separately from the runner turn status so infrastructure cleanup cannot
	// make a completed turn appear to still be running.
	FinalizationStatus RunFinalizationStatus `json:"finalizationStatus" db:"finalization_status"`
	FinalizationError  string                `json:"finalizationError,omitempty" db:"finalization_error"`
	FinalizedAt        *time.Time            `json:"finalizedAt,omitempty" db:"finalized_at"`

	// Inline config (used when no profile provided, or to store resolved config)
	ResolvedConfig *RunConfig `json:"resolvedConfig,omitempty" db:"resolved_config"`

	// Artifacts
	DiffPath       string `json:"diffPath,omitempty" db:"diff_path"`
	LogPath        string `json:"logPath,omitempty" db:"log_path"`
	ChangedFiles   int    `json:"changedFiles" db:"changed_files"`
	TotalSizeBytes int64  `json:"totalSizeBytes" db:"total_size_bytes"`
	CommitHash     string `json:"commitHash,omitempty" db:"commit_hash"`

	// Session continuation support
	// Stores the runner-specific session identifier for conversation resumption.
	// For Claude Code: session_id from stream events
	// For Codex: thread_id from stream events
	// For OpenCode: sessionID from stream events
	SessionID string `json:"sessionId,omitempty" db:"session_id"`

	// Transcript recovery metadata for restart-safe run reconciliation.
	RunnerPID         int    `json:"runnerPid,omitempty" db:"runner_pid"`
	RunnerPGID        int    `json:"runnerPgid,omitempty" db:"runner_pgid"`
	TranscriptPath    string `json:"transcriptPath,omitempty" db:"transcript_path"`
	TranscriptCursor  int64  `json:"transcriptCursor,omitempty" db:"transcript_cursor"`
	TranscriptLastSeq int64  `json:"transcriptLastSeq,omitempty" db:"transcript_last_seq"`

	// Import provenance is populated for read-only transcripts adopted from an
	// external runner store. Source harness plus source session is a stable,
	// runner-qualified identity used to make corpus imports idempotent. The
	// harness half must be compared through [NormalizeImportHarness] — see that
	// function for why the raw label is descriptive, not identifying.
	ImportSourceHarness   string     `json:"importSourceHarness,omitempty" db:"import_source_harness"`
	ImportSourceSessionID string     `json:"importSourceSessionId,omitempty" db:"import_source_session_id"`
	ImportedAt            *time.Time `json:"importedAt,omitempty" db:"imported_at"`

	// Model provenance — requested is the first concrete entry the preset chain expanded to
	// when the run was created; actual is the model the CLI actually executed with once
	// model-fallback (if any) converged. When they differ the run degraded through the chain.
	RequestedModel string `json:"requestedModel,omitempty" db:"requested_model"`
	ActualModel    string `json:"actualModel,omitempty" db:"actual_model"`
	// CanaryArm is duplicated for operator ergonomics; the immutable policy
	// snapshot remains the authoritative replay record.
	CanaryArm string `json:"canaryArm,omitempty" db:"canary_arm"`

	// Investigation lineage fields
	// SourceRunIDs links investigation runs back to the run(s) being investigated.
	SourceRunIDs []uuid.UUID `json:"sourceRunIds,omitempty" db:"source_run_ids"`
	// SourceInvestigationRunID links apply runs back to the investigation run they apply.
	SourceInvestigationRunID *uuid.UUID `json:"sourceInvestigationRunId,omitempty" db:"source_investigation_run_id"`

	// ParentRunID is the generic "parent run" link for conversation continuity.
	// When a spawner is creating a follow-up run as a continuation of an
	// existing agent thread (e.g. swarm-manager queue resuming a swarm,
	// agent-manager UI "continue conversation"), it sets ParentRunID to the
	// originating run. The run-creation path uses ParentRunID to inherit
	// ConversationID — see ResolveConversationID.
	//
	// ParentRunID is a separate concept from SourceInvestigationRunID
	// (apply-from-investigation linkage) and SourceRunIDs (investigation
	// targets); a run can have any combination of those plus ParentRunID.
	ParentRunID *uuid.UUID `json:"parentRunId,omitempty" db:"parent_run_id"`

	// ConversationID groups runs that belong to the same agent thread for
	// auditability. One ID per agent-thread; child runs inherit from
	// ParentRunID's run when set, otherwise the run-creation path generates
	// a fresh UUID. Spawn surfaces that already know they are continuing a
	// thread (e.g. swarm-manager queue resuming a swarm) populate this value
	// directly; standalone runs get a new ID. See
	// scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md Finding 2.
	//
	// IMPORTANT: this is NOT the same as Run.SessionID, which is a
	// runner-specific resume token (Claude Code session_id, Codex thread_id,
	// etc.) with an unrelated lifetime.
	ConversationID string `json:"conversationId,omitempty" db:"conversation_id"`

	// Identity token fields
	IdentityTokenHash      string     `json:"identityTokenHash,omitempty" db:"identity_token_hash"`
	IdentityTokenRevokedAt *time.Time `json:"identityTokenRevokedAt,omitempty" db:"identity_token_revoked_at"`

	// CustomEnv holds the caller-supplied VROOLI_-prefixed environment
	// variables passed at run creation (CreateRunRequest.Environment).
	// Persisted so the continue/wake path can re-inject them: a continued
	// turn that bypassed this would silently drop scenario-injected custom
	// env (the latent bug Phase 0 of the park/resume work fixes). Values are
	// VROOLI_*-validated at the API boundary (≤20 entries / ≤4096 bytes).
	CustomEnv map[string]string `json:"customEnv,omitempty" db:"custom_env"`

	// AwaitHandle describes the externally-owned async work a parked run is
	// waiting on. It is set when the run transitions running→parked and cleared
	// on wake (parked→running) or cancel (parked→cancelled). Persisted (JSON
	// column) so an agent-manager restart can re-spawn the waiter for every
	// parked run (one open handle per run — a second park while parked is
	// rejected). Nil for every non-parked run.
	AwaitHandle *AwaitHandle `json:"awaitHandle,omitempty" db:"await_handle"`

	// LastAwaitKey / LastAwaitResult / LastAwaitResolvedAt record the most
	// recently RESOLVED await: the producer:key, the full result string that was
	// injected into the woken turn, and when it resolved. They are the durable
	// SSOT behind the re-fetch path (GET /runs/{id}/await-result): a woken agent
	// that did not see — or wants to re-read — the result can retrieve it cheaply
	// without re-running the blocking producer. Set on wake (parked→running),
	// retained across subsequent turns until the next await resolves.
	LastAwaitKey        string     `json:"lastAwaitKey,omitempty" db:"last_await_key"`
	LastAwaitResult     string     `json:"lastAwaitResult,omitempty" db:"last_await_result"`
	LastAwaitResolvedAt *time.Time `json:"lastAwaitResolvedAt,omitempty" db:"last_await_resolved_at"`

	// LastWakeSeq snapshots TranscriptLastSeq at the moment of the last wake. The
	// no-progress re-park guard compares it against the live TranscriptLastSeq to
	// best-effort detect whether the agent did any work since being woken before
	// it tries to park again.
	LastWakeSeq int64 `json:"lastWakeSeq,omitempty" db:"last_wake_seq"`

	// SameKeyParkStreak counts how many times in a row this run has tried to park
	// on the SAME await key without forward progress in between. It is the
	// timing-independent backstop the re-park guard uses to refuse a degenerate
	// "wake → immediately re-run the same blocking command → re-park" loop (the
	// coding-agent limitation park exists to absorb). Reset to 0 on a different-key
	// park or on detected progress.
	SameKeyParkStreak int `json:"sameKeyParkStreak,omitempty" db:"same_key_park_streak"`

	// First ~120 chars of the associated task description (computed, not persisted).
	PromptPreview string `json:"promptPreview,omitempty"`

	// Action availability (computed, not persisted)
	Actions *RunActions `json:"actions,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// GetTag returns the tag for this run, defaulting to the run ID if no custom tag is set.
func (r *Run) GetTag() string {
	if r.Tag != "" {
		return r.Tag
	}
	return r.ID.String()
}

// IsResumable returns whether this run can be resumed from its current state.
func (r *Run) IsResumable() bool {
	// Can only resume runs that are in a non-terminal state
	switch r.Status {
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled, RunStatusUnknown:
		return false
	}
	// Check if the phase supports resumption
	return r.Phase.CanResumeFromPhase()
}

// IsStale returns whether this run appears to have stalled.
func (r *Run) IsStale(staleDuration time.Duration) bool {
	if r.LastHeartbeat == nil {
		// No heartbeat recorded, check based on started time
		if r.StartedAt == nil {
			return false
		}
		return time.Since(*r.StartedAt) > staleDuration
	}
	return time.Since(*r.LastHeartbeat) > staleDuration
}

// UpdateProgress updates the run's progress tracking fields.
func (r *Run) UpdateProgress(phase RunPhase, percent int) {
	r.Phase = phase
	r.ProgressPercent = percent
	now := time.Now()
	r.LastHeartbeat = &now
	r.UpdatedAt = now
}

// IsInvestigationRun returns true if this run is an investigation run (not an apply run).
// Investigation runs have a tag starting with "agent-manager-investigation" but not ending in "-apply".
func (r *Run) IsInvestigationRun() bool {
	return strings.HasPrefix(r.Tag, "agent-manager-investigation") &&
		!strings.HasSuffix(r.Tag, "-apply")
}

// RunMode indicates whether the run uses sandbox isolation.
type RunMode string

const (
	RunModeSandboxed RunMode = "sandboxed"
	RunModeInPlace   RunMode = "in_place"
)

// ExecutionMode indicates how agent-manager drives the agent CLI for a run.
// It is orthogonal to [RunMode] (sandbox isolation): a run picks an execution
// substrate independently of whether it is sandboxed.
//
//   - ExecutionModeCodecPipe (default): agent-manager owns the CLI process and
//     reads events off its stdout pipe via the codec decoders. This is the
//     historical path and the only path for protected (sandboxed) runs.
//   - ExecutionModeInteractive: agent-manager launches the real interactive
//     agent CLI inside a web-console (persistent/tmux) session and reads events
//     by tailing the agent-owned on-disk transcript. Allowed only for
//     non-protected (in-place) runs.
type ExecutionMode string

const (
	ExecutionModeCodecPipe   ExecutionMode = "codec_pipe"
	ExecutionModeInteractive ExecutionMode = "interactive"
	// ExecutionModeImported marks a terminal, read-only run adopted from an external harness transcript.
	ExecutionModeImported ExecutionMode = "imported"
	ExecutionModeAttached ExecutionMode = "attached"
)

// Normalized returns the mode with the empty value defaulted to
// ExecutionModeCodecPipe, so rows written before the column existed (and
// callers that leave the field unset) behave as codec-pipe runs.
func (m ExecutionMode) Normalized() ExecutionMode {
	if m == "" {
		return ExecutionModeCodecPipe
	}
	return m
}

// IsValid reports whether the mode is one of the known execution modes.
func (m ExecutionMode) IsValid() bool {
	switch m {
	case ExecutionModeCodecPipe, ExecutionModeInteractive, ExecutionModeImported, ExecutionModeAttached:
		return true
	default:
		return false
	}
}

// RunStatus represents the current state of a run.
type RunStatus string

const (
	RunStatusPending     RunStatus = "pending"
	RunStatusStarting    RunStatus = "starting"
	RunStatusRunning     RunStatus = "running"
	RunStatusNeedsReview RunStatus = "needs_review"
	RunStatusComplete    RunStatus = "complete"
	RunStatusFailed      RunStatus = "failed"
	RunStatusCancelled   RunStatus = "cancelled"
	// RunStatusParked: the run is suspended waiting on externally-owned async
	// work (a test-genie run, a git-control-tower baseline diff). The agent
	// process has exited (zero tokens burned) but the run is NOT terminal — its
	// sandbox is preserved and agent-manager re-spawns ("wakes") the conversation
	// with the awaited result injected once it resolves. Modeled on needs_review
	// (process-exited, non-terminal, resume-with-injected-message) but for
	// infrastructure-driven waiting rather than operator review. See
	// scenarios/agent-manager/docs/internal/SEAMS.md (park/wake) and the
	// LivenessPolicy table in decisions.go (parked is scanned but never
	// heartbeat-reaped).
	RunStatusParked RunStatus = "parked"
	// RunStatusUnknown means historical evidence did not contain a trustworthy
	// terminal signal. It is distinct from a provider failure.
	RunStatusUnknown RunStatus = "unknown"
)

// RunTerminalClass separates a deliberate agent verdict from an involuntary
// interruption. Swarm trusts a verdict and resumes an interruption.
type RunTerminalClass string

const (
	RunTerminalClassVerdict      RunTerminalClass = "verdict"
	RunTerminalClassInterruption RunTerminalClass = "interruption"
)

// RunStopReason is the typed reason a run reached terminal. Verdict reasons are
// complete, blocked and abstained; interruption reasons are usage_window,
// timeout, crash, session_lost and token_cap.
type RunStopReason string

const (
	RunStopReasonComplete    RunStopReason = "complete"
	RunStopReasonBlocked     RunStopReason = "blocked"
	RunStopReasonAbstained   RunStopReason = "abstained"
	RunStopReasonUsageWindow RunStopReason = "usage_window"
	RunStopReasonTimeout     RunStopReason = "timeout"
	RunStopReasonCrash       RunStopReason = "crash"
	RunStopReasonSessionLost RunStopReason = "session_lost"
	// RunStopReasonTokenCap: Agent Manager stopped the run at its weighted-token
	// cap (DL-8). The orchestrator treats it as a step-back.
	RunStopReasonTokenCap RunStopReason = "token_cap"
)

// TerminalClassForStopReason returns the class a stop reason belongs to. It is
// the single mapping Swarm relies on; a reason outside the known set returns
// the empty class so callers fail closed.
func TerminalClassForStopReason(reason RunStopReason) RunTerminalClass {
	switch reason {
	case RunStopReasonComplete, RunStopReasonBlocked, RunStopReasonAbstained:
		return RunTerminalClassVerdict
	case RunStopReasonUsageWindow, RunStopReasonTimeout, RunStopReasonCrash, RunStopReasonSessionLost, RunStopReasonTokenCap:
		return RunTerminalClassInterruption
	default:
		return ""
	}
}

// AwaitHandle identifies the externally-owned async work a parked run is
// blocked on. agent-manager (which owns the agent process) performs the
// blocking wait on the agent's behalf via a per-producer Waiter seam and wakes
// the run when the handle resolves. The handle is the unit persisted for
// restart recovery and the key the waiter de-duplicates on so wake is
// idempotent (a double-resolve must not double-wake).
type AwaitHandle struct {
	// Producer identifies which Waiter resolves this handle (e.g. "test-genie",
	// "git-control-tower"). The await-handle registry (Phase 3) maps it to a
	// concrete Waiter implementation.
	Producer string `json:"producer"`
	// Key is the producer-scoped identifier of the awaited work (e.g. a
	// test-genie run ID, a baseline diff request key). Producer+Key together
	// uniquely identify the work being awaited.
	Key string `json:"key"`
	// Deadline bounds the wait. When it elapses agent-manager wakes the run with
	// a typed "timed-out / unknown" result rather than hanging forever. Nil ⇒
	// the orchestrator default ParkTTL is applied at park time, so a persisted
	// handle always carries a concrete deadline.
	Deadline *time.Time `json:"deadline,omitempty"`
	// RegisteredAt records when the park happened (for observability / ETA).
	RegisteredAt time.Time `json:"registeredAt"`
}

// RunFinalizationStatus represents post-run sandbox apply/checkpoint state.
type RunFinalizationStatus string

const (
	RunFinalizationStatusNone      RunFinalizationStatus = "none"
	RunFinalizationStatusPending   RunFinalizationStatus = "pending"
	RunFinalizationStatusRunning   RunFinalizationStatus = "running"
	RunFinalizationStatusSucceeded RunFinalizationStatus = "succeeded"
	RunFinalizationStatusFailed    RunFinalizationStatus = "failed"
	RunFinalizationStatusSkipped   RunFinalizationStatus = "skipped"
)

// ApprovalState represents the approval workflow state.
type ApprovalState string

const (
	ApprovalStateNone              ApprovalState = "none"
	ApprovalStatePending           ApprovalState = "pending"
	ApprovalStatePartiallyApproved ApprovalState = "partially_approved"
	ApprovalStateApproved          ApprovalState = "approved"
	ApprovalStateRejected          ApprovalState = "rejected"
)

// RunSummary contains the structured summary from an agent run.
type RunSummary struct {
	Description   string   `json:"description,omitempty"`
	FilesModified []string `json:"filesModified,omitempty"`
	FilesCreated  []string `json:"filesCreated,omitempty"`
	FilesDeleted  []string `json:"filesDeleted,omitempty"`
	TokensUsed    int      `json:"tokensUsed,omitempty"`
	TurnsUsed     int      `json:"turnsUsed,omitempty"`
	CostEstimate  float64  `json:"costEstimate,omitempty"`
	ContextTokens int      `json:"contextTokens,omitempty"`
}

// FinalOutputSelectionStatus describes whether terminal evidence identifies a
// unique final assistant handoff. Historical runs may have no RunResult at all;
// a present result always carries one of these explicit outcomes.
type FinalOutputSelectionStatus string

const (
	FinalOutputSelectionSelected    FinalOutputSelectionStatus = "selected"
	FinalOutputSelectionAmbiguous   FinalOutputSelectionStatus = "ambiguous"
	FinalOutputSelectionUnavailable FinalOutputSelectionStatus = "unavailable"
)

// FinalOutputCandidate is an immutable projection of one assistant message
// considered by the final-output resolver.
type FinalOutputCandidate struct {
	ID                string `json:"id"`
	EventID           string `json:"eventId,omitempty"`
	Sequence          int64  `json:"sequence,omitempty"`
	Content           string `json:"content"`
	MessageID         string `json:"messageId,omitempty"`
	ConversationID    string `json:"conversationId,omitempty"`
	TurnID            string `json:"turnId,omitempty"`
	ProviderOrigin    string `json:"providerOrigin,omitempty"`
	CompletionReason  string `json:"completionReason,omitempty"`
	Terminal          bool   `json:"terminal,omitempty"`
	ParentMessageID   string `json:"parentMessageId,omitempty"`
	ProviderEventType string `json:"providerEventType,omitempty"`
	RawEvidenceRef    string `json:"rawEvidenceRef,omitempty"`
	EvidenceTier      int    `json:"evidenceTier"`
}

// FinalOutputSelection records the deterministic resolver decision and the
// exact rule/version needed to explain or reproduce it.
type FinalOutputSelection struct {
	Status              FinalOutputSelectionStatus `json:"status"`
	SelectedCandidateID string                     `json:"selectedCandidateId,omitempty"`
	Rule                string                     `json:"rule"`
	AlgorithmVersion    string                     `json:"algorithmVersion"`
	Evidence            []string                   `json:"evidence,omitempty"`
}

// RunResult is the canonical terminal result for one execute or continue turn.
// It intentionally remains useful when selection is ambiguous/unavailable.
type RunResult struct {
	FinalOutput    string                 `json:"finalOutput,omitempty"`
	Selection      FinalOutputSelection   `json:"selection"`
	Candidates     []FinalOutputCandidate `json:"candidates,omitempty"`
	Success        bool                   `json:"success"`
	ExitCode       int                    `json:"exitCode"`
	TerminalReason string                 `json:"terminalReason,omitempty"`
	Structured     *StructuredResult      `json:"structured,omitempty"`
}

// ResultSpecKind selects the one canonical typed-result contract. Enum
// classification is represented as a schema-shaped ResultSpec rather than a
// separate classifier persistence model.
type ResultSpecKind string

const (
	ResultSpecKindNone           ResultSpecKind = "none"
	ResultSpecKindJSONSchema     ResultSpecKind = "json_schema"
	ResultSpecKindClassification ResultSpecKind = "classification"
)

// StructuredExtractionMode controls whether deterministic parsing may fall
// back to the portable extraction seam. The fallback is never trusted without
// the same local schema validation as deterministic candidates.
type StructuredExtractionMode string

const (
	StructuredExtractionDeterministic StructuredExtractionMode = "deterministic_only"
	StructuredExtractionConstrained   StructuredExtractionMode = "constrained_fallback"
)

// ResultSpec is the versioned request for a typed result. Schema contains
// canonical JSON bytes after creation-time normalization. ClassificationValues
// is a create-surface convenience that is compiled into Schema and then
// cleared, keeping Schema as the sole persisted validation authority.
type ResultSpec struct {
	Version              string                   `json:"version"`
	Kind                 ResultSpecKind           `json:"kind"`
	Schema               json.RawMessage          `json:"schema,omitempty"`
	SchemaDigest         string                   `json:"schemaDigest,omitempty"`
	ClassificationValues []string                 `json:"classificationValues,omitempty"`
	ExtractionMode       StructuredExtractionMode `json:"extractionMode,omitempty"`
	ExtractionRole       string                   `json:"extractionRole,omitempty"`
	// SchemaRepairAttempts is nil for the safe workflow default of one repair,
	// zero to disable repair, and one to request the single bounded correction.
	SchemaRepairAttempts *int `json:"schemaRepairAttempts,omitempty"`
}

// StructuredResultStatus separates all honest terminal outcomes. Only
// StructuredResultSuccess may carry Value.
type StructuredResultStatus string

const (
	StructuredResultSuccess     StructuredResultStatus = "success"
	StructuredResultUnavailable StructuredResultStatus = "unavailable"
	StructuredResultInvalid     StructuredResultStatus = "invalid"
	StructuredResultAmbiguous   StructuredResultStatus = "ambiguous"
	StructuredResultAbstained   StructuredResultStatus = "abstained"
)

// StructuredDiagnostic is bounded, normalized, and safe to expose. It never
// includes source output or schema fragments, which prevents secret-bearing
// agent text from leaking through validation errors.
type StructuredDiagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// StructuredExtractionProvenance explains how a fallback candidate was
// produced without making provider output authoritative.
type StructuredExtractionProvenance struct {
	RoleRef        string                   `json:"roleRef,omitempty"`
	Provider       string                   `json:"provider,omitempty"`
	Model          string                   `json:"model,omitempty"`
	PolicySnapshot *ExecutionPolicySnapshot `json:"policySnapshot,omitempty"`
}

// StructuredResult is the locally validated typed projection attached to the
// canonical RunResult. Requested schema digest, method, and diagnostics remain
// durable even when resolution abstains or fails.
type StructuredResult struct {
	Status            StructuredResultStatus          `json:"status"`
	SpecKind          ResultSpecKind                  `json:"specKind"`
	SchemaDigest      string                          `json:"schemaDigest"`
	Value             json.RawMessage                 `json:"value,omitempty"`
	Method            string                          `json:"method,omitempty"`
	SourceCandidateID string                          `json:"sourceCandidateId,omitempty"`
	Extractor         *StructuredExtractionProvenance `json:"extractor,omitempty"`
	Diagnostics       []StructuredDiagnostic          `json:"diagnostics,omitempty"`
}

// RunAdmission is the immutable creation-time record that binds the caller's
// requested settings to the owner-resolved effective settings and the policy
// source they were resolved from. It is embedded in RunConfig so it persists in
// the same resolved_config snapshot and is exposed on every run read without a
// separate lookup. The requested side is taken verbatim from the create request
// (empty means the caller did not request the field); the effective side mirrors
// the enclosing RunConfig.
// CreateRunCaller is non-secret, server-verified admission provenance. It is
// persisted inside resolved_config; it is not a credential or a wire request.
// Absence on historical rows is unknown proof, never anonymous authority.
type CreateRunCaller struct {
	Kind         string    `json:"kind"`
	EffortID     string    `json:"effortId,omitempty"`
	EffortEpoch  uint64    `json:"effortEpoch,omitempty"`
	EffortDigest string    `json:"effortDigest,omitempty"`
	Subject      string    `json:"subject,omitempty"`
	RunID        uuid.UUID `json:"runId,omitempty"`
}

type RunAdmission struct {
	// Native-owned finite task projection, bound by EffortIntent.InputDigest.
	// Stored privately with RunConfig; never a caller identity projection.
	EffortTaskProjection []byte `json:"effortTaskProjection,omitempty"`
	// CreateCaller is internal admission evidence. Proto projection does not
	// carry it; authentication/replay must read the full authoritative row.
	Effort            *effortauthority.Binding `json:"effort,omitempty"`
	EffortIntent      *effortauthority.Intent  `json:"effortIntent,omitempty"`
	CreateCaller      *CreateRunCaller         `json:"createCaller,omitempty"`
	RequestedRunner   string                   `json:"requestedRunner,omitempty"`
	RequestedModel    string                   `json:"requestedModel,omitempty"`
	RequestedRoleRef  string                   `json:"requestedRoleRef,omitempty"`
	RequestedEffort   string                   `json:"requestedEffort,omitempty"`
	RequestedTimeout  time.Duration            `json:"requestedTimeout,omitempty"`
	RequestedMaxTurns int                      `json:"requestedMaxTurns,omitempty"`

	// RequestedGoalMode is "until" when the caller supplied a completion test,
	// otherwise empty.
	RequestedGoalMode string `json:"requestedGoalMode,omitempty"`

	EffectiveRunner   string        `json:"effectiveRunner,omitempty"`
	EffectiveModel    string        `json:"effectiveModel,omitempty"`
	EffectiveEffort   string        `json:"effectiveEffort,omitempty"`
	EffectiveTimeout  time.Duration `json:"effectiveTimeout,omitempty"`
	EffectiveMaxTurns int           `json:"effectiveMaxTurns,omitempty"`
	EffectiveUntil    string        `json:"effectiveUntil,omitempty"`

	CatalogDigest   string `json:"catalogDigest,omitempty"`
	PolicyDigest    string `json:"policyDigest,omitempty"`
	PolicyPath      string `json:"policyPath,omitempty"`
	SelectionReason string `json:"selectionReason,omitempty"`

	// PassedControlArgs is the "passed" layer between owner-resolved effective
	// values and provider acknowledgment: the runner-native control arguments
	// the selected codec emits for the resolved configuration (model, reasoning
	// effort, tool and permission controls). Empty when the runner registry or
	// codec was unavailable or refused the configuration at admission.
	PassedControlArgs []string `json:"passedControlArgs,omitempty"`

	// TranslationDiagnostics explains how the canonical controls became native
	// arguments, or why translation was refused. It is the reader's evidence
	// that a requested setting was honored rather than silently dropped. An
	// entry never substitutes for a launch; the empty passed layer plus the
	// diagnostic is the truthful record.
	TranslationDiagnostics []string `json:"translationDiagnostics,omitempty"`

	// RuntimeVersion is the concrete runner/runtime version observed for this
	// run. It is only observable at a live launch, so it stays empty for runs
	// admitted before that evidence exists; the bounded live qualification
	// probe supplies it. Never inferred from the requested or passed layers.
	RuntimeVersion string `json:"runtimeVersion,omitempty"`

	// ProviderAcknowledgment is the provider's own confirmation of the
	// effective runner, model and reasoning effort, captured from provider
	// evidence during a bounded live launch. Empty until that evidence exists;
	// it never substitutes for the requested, effective or passed layers.
	ProviderAcknowledgment []string `json:"providerAcknowledgment,omitempty"`

	// Receipt is the reusable route-keyed qualification receipt derived from a
	// bounded live run. It persists in the same resolved_config snapshot so a
	// fresh coordinator can read it without a separate lookup and the
	// configuration-sensitive delegation gate can compare it against a
	// requested identity. Nil until live qualification evidence exists.
	Receipt *QualificationReceipt `json:"receipt,omitempty"`
}

// RunConfig contains the resolved configuration for a run.
// This can be loaded from a profile, provided inline, or a combination of both.
type RunConfig struct {
	// Runner configuration
	RunnerType RunnerType `json:"runnerType"`
	// ManifestIndexSnapshot pins the CLI catalog index used by an imported
	// transcript. Imported episode attribution remains historical evidence.
	ManifestIndexSnapshot string  `json:"manifestIndexSnapshot,omitempty"`
	TranscriptCodec       string  `json:"transcriptCodec,omitempty"`
	TranscriptCodecScore  float64 `json:"transcriptCodecScore,omitempty"`
	Until                 string  `json:"until,omitempty"`
	Model                 string  `json:"model,omitempty"`
	RoleRef               string  `json:"roleRef,omitempty"`
	PreferredRunner       string  `json:"preferredRunner,omitempty"`
	MaxTurns              int     `json:"maxTurns,omitempty"`
	// MaxToolCalls is an owner-declared hard ceiling for one runner process.
	// Unlike aggregate token accounting, it remains enforceable when a provider
	// emits usage only at terminal completion. Zero preserves unlimited behavior.
	MaxToolCalls int           `json:"maxToolCalls,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	Effort       Effort        `json:"effort,omitempty"`

	// PolicySnapshot pins the exact active catalog revision and ordered
	// candidate sequence selected before this run was persisted.
	PolicySnapshot *ExecutionPolicySnapshot `json:"policySnapshot,omitempty"`
	Billing        BillingSnapshot          `json:"billing,omitempty"`

	// ResultSpec is normalized before the run is persisted. Nil/none preserves
	// the historical unstructured behavior.
	ResultSpec *ResultSpec `json:"resultSpec,omitempty"`

	// Tool permissions
	AllowedTools          []string              `json:"allowedTools,omitempty"`
	DeniedTools           []string              `json:"deniedTools,omitempty"`
	ToolRestrictionPolicy ToolRestrictionPolicy `json:"toolRestrictionPolicy,omitempty"`

	// Execution flags
	SkipPermissionPrompt bool `json:"skipPermissionPrompt,omitempty"`

	// Feature flags (typed, discoverable capabilities)
	Features FeatureFlags `json:"features,omitempty"`

	// Extra CLI flags per runner type (validated escape hatch)
	ExtraFlags RunnerExtraFlags `json:"extraFlags,omitempty"`

	// Policy flags
	NetworkAccess NetworkAccess `json:"networkAccess"`

	// PreambleInjectedTokens is the estimator result recorded at run creation
	// for the instructions Agent Manager injects into the provider context.
	// It is metadata about this run, not a runner-selection input.
	PreambleInjectedTokens int64                 `json:"preambleInjectedTokens,omitempty"`
	PreambleTokenBasis     tokenaccounting.Basis `json:"preambleTokenBasis,omitempty"`

	// Sandbox behavior settings.
	//
	// SandboxConfig.Mode is the single source of truth for whether the
	// run is sandboxed. See [orchestration.DeriveRunMode]. A nil
	// SandboxConfig is treated as "unspecified" — orchestration spawn
	// surfaces clone [DefaultSandboxConfig] before resolving so the
	// nil case only arises in legacy tests.
	SandboxConfig *SandboxConfig `json:"sandboxConfig,omitempty"`

	// Path restrictions
	AllowedPaths []string `json:"allowedPaths,omitempty"`
	DeniedPaths  []string `json:"deniedPaths,omitempty"`
	// AllowedEffects is the owner-issued effect ceiling for this run.
	AllowedEffects []string `json:"allowedEffects,omitempty"`
	// RequireEffectContainment prevents effect-bearing runs from silently
	// downgrading to a host launcher when the workspace boundary is absent.
	RequireEffectContainment bool     `json:"requireEffectContainment,omitempty"`
	SkillPack                []string `json:"skillPack,omitempty"`
	SkillExperimentID        string   `json:"skillExperimentId,omitempty"`

	// Admission records the requested-versus-effective configuration proven
	// for this admitted run. Nil on historical rows created before adoption.
	Admission *RunAdmission `json:"admission,omitempty"`
}

// ApplyProfile applies values from an AgentProfile as the base configuration.
func (c *RunConfig) ApplyProfile(profile *AgentProfile) {
	if profile == nil {
		return
	}
	c.RoleRef = profile.RoleRef
	c.MaxTurns = profile.MaxTurns
	c.Timeout = profile.Timeout
	c.Effort = profile.Effort
	c.AllowedTools = profile.AllowedTools
	c.DeniedTools = profile.DeniedTools
	c.ToolRestrictionPolicy = profile.ToolRestrictionPolicy.Effective()
	c.SkipPermissionPrompt = profile.SkipPermissionPrompt
	c.Features = profile.Features
	if len(profile.ExtraFlags) > 0 {
		c.ExtraFlags = make(RunnerExtraFlags, len(profile.ExtraFlags))
		for rt, flags := range profile.ExtraFlags {
			c.ExtraFlags[rt] = append([]string(nil), flags...)
		}
	}
	c.NetworkAccess = profile.NetworkAccess
	// Only overwrite SandboxConfig when the profile actually provides
	// one. A nil profile.SandboxConfig means "use the run-config
	// default"; copying it would silently clobber DefaultSandboxConfig
	// and reintroduce the zero-value-bool bypass class of bug.
	if profile.SandboxConfig != nil {
		c.SandboxConfig = profile.SandboxConfig
	}
	c.AllowedPaths = profile.AllowedPaths
	c.DeniedPaths = profile.DeniedPaths
	c.SkillPack = append([]string(nil), profile.SkillPack...)
	c.SkillExperimentID = profile.SkillExperimentID
}

// DefaultRunConfig returns sensible defaults for run configuration.
//
// The auditability-contract apply defaults (ManualReview=false,
// AutoApply=true, ApplyOnFailure=true, Mode=Protected) live on
// SandboxConfig — see DefaultSandboxConfig. Embedding the default
// SandboxConfig here is what makes the "sandbox by default" invariant
// hold even when callers ApplyProfile a profile with no SandboxConfig
// of its own.
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		RunnerType:            RunnerTypeClaudeCode,
		MaxTurns:              30,
		Timeout:               60 * time.Minute,
		NetworkAccess:         NetworkAccessLocalhost,
		ToolRestrictionPolicy: ToolRestrictionPolicyEnforced,
		SandboxConfig:         DefaultSandboxConfig(),
	}
}

// -----------------------------------------------------------------------------
// RunEvent - Append-only event stream
// -----------------------------------------------------------------------------
//
// TAGGED UNION PATTERN:
// RunEvent uses a tagged union for type-safe event payloads. Each event type
// has a specific payload struct, ensuring you can only set relevant fields.
//
// Usage:
//   event := NewLogEvent(runID, "info", "Starting execution")
//   event := NewToolCallEvent(runID, "Read", "toolu_123", map[string]interface{}{"path": "/foo"})
//
// The Data field contains a type-specific payload that can be type-asserted:
//   if log, ok := event.Data.(*LogEventData); ok { ... }

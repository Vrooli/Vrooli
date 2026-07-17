package opsrunner

import (
	"context"
	"encoding/json"
	"errors"

	"swarm-manager/internal/agentops"
)

// TargetRef names the unit of work an operation runs against. ID is the domain
// entity's stable identity: a backlog item ref ("kind/name") or an initiative
// name. The runner never interprets ID beyond passing it to the target adapter
// via the ModePreparer and to the workflow repository — it does not branch on
// Kind.
type TargetRef struct {
	Kind agentops.TargetKind
	ID   string
}

// InvokeRequest is the typed request to run one operation against one target.
type InvokeRequest struct {
	Target    TargetRef
	Operation agentops.OperationID
	// OperationVersion pins an exact contract version; empty resolves the highest
	// authored version from the catalog.
	OperationVersion string
	// CallerInputs are the typed caller-supplied values the operation contract
	// declares. They are validated against the compiled input contract and their
	// canonical digest is pinned in provenance; sensitive values are rejected per
	// the contract's retention policy before anything is persisted.
	CallerInputs map[string]any
	// AuthorizedInvocationBinding, when non-nil, is an explicit highest-precedence
	// binding pinned by an authorized caller for this one run.
	AuthorizedInvocationBinding *agentops.OperationBinding
	// IdempotencyKey deduplicates a retried Invoke: a second Invoke with the same
	// key against the same workflow returns the prior execution's result without
	// starting a second run or applying a second action.
	IdempotencyKey string
	// RequestedBy is a provenance label for who initiated the operation.
	RequestedBy string
	// Simulate drives the deterministic in-memory driver instead of a live agent
	// spawn. Preset selects a simulation example-run; empty uses the default.
	Simulate         bool
	SimulationPreset string
}

// Disposition is the workflow-level classification of an operation outcome,
// derived from the operation contract's declared outcome dispositions.
type Disposition string

// OperationResult is the typed outcome the runner returns.
type OperationResult struct {
	WorkflowInstanceID string
	ExecutionID        string
	// Provenance is the immutable record pinned before the run started.
	Provenance       agentops.ExecutionProvenance
	ProvenanceDigest string
	// Outcome is the contract outcome name the execution terminated with.
	Outcome     string
	Disposition Disposition
	// Result is the structured result payload the execution produced.
	Result json.RawMessage
	// Action is the domain action the transition policy selected for this
	// outcome, if any fired. Empty when the policy had no matching transition.
	Action agentops.ActionName
	// WorkflowState is the workflow's coordination state after the transition.
	WorkflowState agentops.WorkflowState
	// EvidenceRefs are the run-owner attribution keys under which this
	// execution's evidence is recorded in the ledger.
	EvidenceRefs []EvidenceRef
	// StartHandle is set on a non-blocking live Invoke: it carries the run
	// association (agent run/task ids) the caller returns to its client while the
	// operation runs. Nil on the synchronous simulation path (which returns a
	// terminal Outcome instead).
	StartHandle *StartHandle
	// Replayed is true when an idempotency key matched and the prior result was
	// returned without a second run.
	Replayed bool
}

// EvidenceRef attributes an execution's evidence to (target, operation,
// workflow, execution, mode revision) in the run-owner index.
type EvidenceRef struct {
	RunID        string
	WorkflowID   string
	ExecutionID  string
	Operation    agentops.OperationID
	Mode         string
	ModeRevision string
}

// PrepareRequest asks the ModePreparer to compile and pin the bound mode for a
// target and validate caller inputs.
type PrepareRequest struct {
	Target       TargetRef
	Operation    agentops.OperationID
	Mode         string
	ModeRevision string
	CallerInputs map[string]any
}

// Prepared is the immutable, reproducible snapshot the ModePreparer returns. The
// runner hashes CompiledMode, PromptCatalog, and EffectiveInputs into the
// provenance digests, so all hashing lives in the runner and stays canonical.
type Prepared struct {
	ExecutionID  string
	Mode         string
	ModeRevision string
	// CompiledMode is the canonical bytes of the pinned mode bundle (parent +
	// delegated definitions) at ModeRevision. Hashed into compiled_mode_digest.
	CompiledMode json.RawMessage
	// PromptCatalog is the canonical bytes of the reachable prompt-source
	// projection. Hashed into prompt_catalog_digest.
	PromptCatalog         json.RawMessage
	PromptCatalogRevision string
	// EffectiveInputs is the validated, normalized caller-input snapshot shared
	// by simulation and live runs. Hashed into caller_input_digest.
	EffectiveInputs json.RawMessage
}

// ExecutionOutcome is what an ExecutionDriver reports after one run.
type ExecutionOutcome struct {
	Outcome     string
	Disposition Disposition
	Result      json.RawMessage
	RunID       string
}

// RunHandle carries the pinned identity of the execution to a driver.
type RunHandle struct {
	ExecutionID string
	WorkflowID  string
	Target      TargetRef
	Operation   agentops.OperationID
	Preset      string
	Simulate    bool
}

// ModePreparer compiles+pins the bound mode and validates caller inputs. It is
// the seam through which the (target-specific, engine-owned) work enters the
// generic runner; the runner itself never resolves a target adapter or a mode
// definition. RevisionExists/ModeCompatible back the binding checker.
type ModePreparer interface {
	Prepare(ctx context.Context, req PrepareRequest) (Prepared, error)
	RevisionExists(mode, revision string) bool
	ModeCompatible(mode string, op agentops.OperationID, target agentops.TargetKind) bool
}

// ExecutionDriver drives one prepared execution to a terminal outcome. It backs
// the synchronous simulation/test path: no agent spawn, a deterministic outcome
// the runner records and transitions on inline.
type ExecutionDriver interface {
	Drive(ctx context.Context, prep Prepared, run RunHandle) (ExecutionOutcome, error)
}

// StartHandle is the run association a non-blocking live start returns
// immediately. The operation stays running until its round is delivered to
// CommitResult; these ids let the caller surface the in-flight run to its client
// and correlate the eventual result back to this execution.
type StartHandle struct {
	RunID     string
	TaskID    string
	BaseURL   string
	CreatedAt string
}

// RunStarter starts one live operating-mode execution and returns immediately,
// WITHOUT waiting for the round to finish. It spawns the agent through the
// operating-mode engine's agentactivity chokepoint (the only allowed live spawn
// path) and returns the run association. The eventual outcome is delivered back
// to the runner via CommitResult when the round completes — this seam never
// classifies or transitions.
type RunStarter interface {
	Start(ctx context.Context, prep Prepared, run RunHandle) (StartHandle, error)
}

// Typed runner errors.
var (
	ErrUnknownOperation   = errors.New("operation is not declared in the catalog")
	ErrIncompatibleTarget = errors.New("operation is incompatible with the target kind")
	// ErrNoRunID is returned when a live start reports success but yields no run
	// id: nothing trackable was started, so the operation could never be delivered
	// to CommitResult. The runner fails closed and reaps the operation rather than
	// returning a phantom running record.
	ErrNoRunID          = errors.New("live start produced no run id")
	ErrDigestMismatch   = errors.New("pinned digest does not match recomputed digest")
	ErrWorkflowConflict = errors.New("workflow compare-and-swap conflict")
	ErrAmbiguousResume  = errors.New("ambiguous resumable execution")
	// ErrLegacyImportNotReproducible is returned when reproducibility is asked of
	// a Phase-8 legacy-execution-import snapshot: compiled-mode/prompt-catalog
	// digests never existed for pre-cutover runs, so there is nothing to reverify.
	ErrLegacyImportNotReproducible = errors.New("legacy execution import has no reproducibility provenance")
)

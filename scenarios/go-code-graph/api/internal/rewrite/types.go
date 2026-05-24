// Package rewrite is the domain-scoped home for the two-step file-move +
// import-rewrite surface. It owns the in-process plan store, the
// RewriteExecutor seam, plan normalization, and the typed RewriteError
// sentinel set the handler maps to Connect codes.
//
// Layering follows the canonical Vrooli pattern:
//
//	handler → Service → seam(PlanStore, RewriteExecutor)
//
// The Service is pure orchestration; every filesystem-touching path
// goes through the RewriteExecutor seam so tests can wire FakeExecutor
// and never mutate disk.
package rewrite

import "fmt"

// PlanID is the sha256-hex identifier the Service derives for every
// successful Plan call. Stable across identical inputs.
type PlanID string

// OperationKind enumerates the concrete operation shapes. Used as the
// stable sort prefix in normalize.go so canonical JSON serialization
// (and therefore PlanID derivation) is deterministic.
type OperationKind string

const (
	OperationKindFileMove      OperationKind = "file_move"
	OperationKindImportRewrite OperationKind = "import_rewrite"
)

// Operation is the sealed sum type the Service consumes. The two
// concrete variants live in this file; isOperation() keeps third
// parties from accidentally satisfying the interface.
type Operation interface {
	Kind() OperationKind
	isOperation()
}

// FileMove relocates a single file from From to To. Both paths are
// relative to the scenario root, no leading slash, no ".." segments.
type FileMove struct {
	From string
	To   string
}

// Kind reports the OperationKind discriminator.
func (FileMove) Kind() OperationKind { return OperationKindFileMove }
func (FileMove) isOperation()        {}

// ImportRewrite replaces import path Old with New across every .go
// file under the scenario root.
type ImportRewrite struct {
	Old string
	New string
}

// Kind reports the OperationKind discriminator.
func (ImportRewrite) Kind() OperationKind { return OperationKindImportRewrite }
func (ImportRewrite) isOperation()        {}

// Plan is the immutable record persisted in the PlanStore between Plan
// and Apply. ScenarioPath is recorded so Apply can reject mismatched
// requests with PathMismatch.
type Plan struct {
	ID           PlanID
	ScenarioPath string
	Operations   []Operation
}

// PlanInput is the validated payload threaded from handler to
// Service.Plan.
type PlanInput struct {
	ScenarioPath string
	Operations   []Operation
}

// ApplyInput is the validated payload threaded from handler to
// Service.Apply. Apply must be true; DryRun threads the X-Dry-Run
// header through so the Service short-circuits before invoking the
// executor.
type ApplyInput struct {
	ScenarioPath string
	PlanID       PlanID
	Apply        bool
	DryRun       bool
}

// OperationStatus mirrors the proto enum for the apply log.
type OperationStatus int

const (
	OperationStatusUnspecified OperationStatus = 0
	OperationStatusOK          OperationStatus = 1
	OperationStatusFailed      OperationStatus = 2
)

// OperationResult is one row of the per-op apply log returned by
// Service.Apply.
type OperationResult struct {
	Operation Operation
	Status    OperationStatus
	Message   string
}

// ApplyResult is the full apply log plus a dry-run echo.
type ApplyResult struct {
	PlanID  PlanID
	Results []OperationResult
	DryRun  bool
}

// RewriteErrorKind names the catastrophic conditions the Service
// returns as typed sentinels. The set is held in lockstep with proto's
// errors_v1.RewriteError_* (see packages/proto/schemas/go-code-graph/v1/errors).
type RewriteErrorKind string

const (
	// RewriteErrorNoOperations means the caller passed an empty
	// operations list.
	RewriteErrorNoOperations RewriteErrorKind = "no_operations"
	// RewriteErrorMalformedOperation means an operation was missing
	// required fields or contained forbidden path segments.
	RewriteErrorMalformedOperation RewriteErrorKind = "malformed_operation"
	// RewriteErrorPlanNotFound means the caller asked Apply for an
	// unknown plan_id.
	RewriteErrorPlanNotFound RewriteErrorKind = "plan_not_found"
	// RewriteErrorPathMismatch means the apply request's scenario_path
	// differs from the path the plan was authored against.
	RewriteErrorPathMismatch RewriteErrorKind = "path_mismatch"
	// RewriteErrorApplyNotSet means the caller forgot to set apply=true
	// on an Apply request.
	RewriteErrorApplyNotSet RewriteErrorKind = "apply_not_set"
	// RewriteErrorInternal means an unexpected failure (executor
	// boom, store boom, etc.).
	RewriteErrorInternal RewriteErrorKind = "internal"
)

// RewriteError is the typed sentinel the Service returns. Handlers
// translate the Kind to a Connect error code via ErrorToConnectCode.
type RewriteError struct {
	Kind    RewriteErrorKind
	Message string
	Cause   error
}

// Error renders the typed sentinel.
func (e RewriteError) Error() string {
	if e.Message != "" && e.Cause != nil {
		return fmt.Sprintf("rewrite %s: %s: %v", e.Kind, e.Message, e.Cause)
	}
	if e.Message != "" {
		return fmt.Sprintf("rewrite %s: %s", e.Kind, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("rewrite %s: %v", e.Kind, e.Cause)
	}
	return fmt.Sprintf("rewrite %s", e.Kind)
}

// Unwrap exposes the underlying cause for errors.Is / errors.As.
func (e RewriteError) Unwrap() error { return e.Cause }

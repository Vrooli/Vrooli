// Package rewrite is the domain-scoped home for the TypeScript rewrite
// surface: two-step plan/apply where the plan is a deterministic ID
// over a normalized operation list, and apply delegates execution to
// the Node sidecar (which owns ts-morph and the consumer project file
// system).
//
// Layering follows the canonical Vrooli pattern: handler → Service →
// seam (sidecar.SidecarClient + PlanStore). The Service itself is pure
// orchestration; every side-effecting capability is reachable only
// through a seam.
//
// Substrate boundary: this package MUST NOT import time, os, net/http,
// or os/exec — see no_prod_import_test.go. The package also MUST NOT
// import os/exec or name git/tsc/pnpm in source — see
// no_external_command_test.go (PRD non-goals §12).
package rewrite

import (
	"fmt"
)

// PlanID is the sha256-hex digest over the normalized operation list.
// Two identical normalized lists always produce the same PlanID; the
// store scopes plans by (scenario_path, PlanID) so two scenarios with
// the same operation list cannot replay each other's plans.
type PlanID string

// FileMove relocates a TypeScript source file inside the project. Both
// paths are scenario-relative (canonicalized by Normalize); absolute
// paths and "../" escape sequences are rejected during validation.
type FileMove struct {
	FromPath string
	ToPath   string
}

// ImportRewrite changes every import / export specifier referring to
// OldPath so it refers to NewPath instead. As with FileMove, paths are
// scenario-relative.
type ImportRewrite struct {
	OldPath string
	NewPath string
}

// Operation is the rewrite domain's discriminated union. Exactly one
// of the embedded pointers must be non-nil; both nil or both set is
// rejected with ErrInvalidOperation.
//
// The shape mirrors the proto Operation oneof (FileMove at tag 1,
// ImportRewrite at tag 2). Adapter code in handlers/rewrite/adapter.go
// translates between this domain type and the proto.
type Operation struct {
	FileMove      *FileMove
	ImportRewrite *ImportRewrite
}

// OperationTag returns a short string identifying which oneof arm is
// set ("file_move" or "import_rewrite"). Used by Normalize for stable
// sort and by Hash for canonical JSON encoding.
//
// Returns "" when neither arm is set (invalid).
func (o Operation) OperationTag() string {
	switch {
	case o.FileMove != nil && o.ImportRewrite == nil:
		return "file_move"
	case o.ImportRewrite != nil && o.FileMove == nil:
		return "import_rewrite"
	default:
		return ""
	}
}

// PrimaryPath returns the "first" path of the operation under canonical
// ordering: the from-side of a FileMove, the old-side of an
// ImportRewrite. Used for stable sort.
func (o Operation) PrimaryPath() string {
	switch {
	case o.FileMove != nil:
		return o.FileMove.FromPath
	case o.ImportRewrite != nil:
		return o.ImportRewrite.OldPath
	default:
		return ""
	}
}

// SecondaryPath returns the destination path (to-side / new-side).
// Used as a tiebreaker in stable sort.
func (o Operation) SecondaryPath() string {
	switch {
	case o.FileMove != nil:
		return o.FileMove.ToPath
	case o.ImportRewrite != nil:
		return o.ImportRewrite.NewPath
	default:
		return ""
	}
}

// Plan is what the store holds after Service.Plan succeeds. The
// ScenarioPath is captured at plan time and re-checked at apply time
// to prevent cross-scenario plan replay.
type Plan struct {
	ID           PlanID
	ScenarioPath string
	Operations   []Operation
}

// PlanInput is the validated request payload threaded from handler to
// Service.Plan.
type PlanInput struct {
	ScenarioPath string
	Operations   []Operation
}

// PlanOutput bundles the assigned PlanID and the normalized operation
// list so the caller can persist the canonical form and re-render the
// plan back to a user.
type PlanOutput struct {
	PlanID               PlanID
	NormalizedOperations []Operation
}

// ApplyInput is the validated request payload threaded from handler to
// Service.Apply.
type ApplyInput struct {
	ScenarioPath string
	PlanID       PlanID
	DryRun       bool
}

// ApplyResult is one row of the per-op apply log. The Status string
// uses the proto OperationStatus enum-name values
// ("OPERATION_STATUS_OK" / "OPERATION_STATUS_FAILED") so the adapter
// can pass them straight through.
type ApplyResult struct {
	Operation Operation
	Status    string
	Message   string
}

// ApplyOutput bundles everything the handler needs to project onto the
// proto RewriteApplyResponse.
type ApplyOutput struct {
	PlanID  PlanID
	Results []ApplyResult
	DryRun  bool
}

// Status constants — match the proto OperationStatus enum names so
// adapter mapping is mechanical.
const (
	StatusOK     = "OPERATION_STATUS_OK"
	StatusFailed = "OPERATION_STATUS_FAILED"
)

// RewriteErrorKind enumerates the catastrophic conditions Service.Plan
// and Service.Apply return. Handlers map each Kind to a Connect code
// via ErrorToConnectCode.
type RewriteErrorKind string

const (
	// RewriteErrorInvalidInput means the request payload itself was
	// malformed (empty scenario_path, empty operations, etc.).
	RewriteErrorInvalidInput RewriteErrorKind = "invalid_input"
	// RewriteErrorInvalidOperation means at least one Operation was
	// malformed: wrong number of oneof arms set, invalid paths, etc.
	RewriteErrorInvalidOperation RewriteErrorKind = "invalid_operation"
	// RewriteErrorPlanNotFound means the (scenario_path, plan_id)
	// composite key did not match a stored plan.
	RewriteErrorPlanNotFound RewriteErrorKind = "plan_not_found"
	// RewriteErrorSidecarUnavailable means the Node sidecar is not in a
	// state to accept requests.
	RewriteErrorSidecarUnavailable RewriteErrorKind = "sidecar_unavailable"
	// RewriteErrorSidecarTimeout means a sidecar call exceeded its
	// deadline.
	RewriteErrorSidecarTimeout RewriteErrorKind = "sidecar_timeout"
	// RewriteErrorInternal is the default for unexpected errors.
	RewriteErrorInternal RewriteErrorKind = "internal"
)

// RewriteError is the typed sentinel the Service returns. Handlers map
// Kind → Connect code via ErrorToConnectCode.
type RewriteError struct {
	Kind    RewriteErrorKind
	Path    string
	PlanID  PlanID
	Message string
	Cause   error
}

func (e RewriteError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("rewrite %s (%s): %s", e.Kind, e.Path, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("rewrite %s (%s): %v", e.Kind, e.Path, e.Cause)
	}
	return fmt.Sprintf("rewrite %s (%s)", e.Kind, e.Path)
}

func (e RewriteError) Unwrap() error { return e.Cause }

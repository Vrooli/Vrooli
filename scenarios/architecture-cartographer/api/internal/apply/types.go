// Package apply is the domain-scoped home for fix-plan generation
// and (in a later plan) execution.
//
// v0.1 ships the type surface, planner (deterministic — derives the
// operation list from current resolved conflicts without any toolchain
// dependency), the BuildGuard interface, and the Recipe interface. The
// Service.RunApply method returns CodeUnimplemented; execution body
// lands in a later plan.
package apply

import (
	"fmt"
	"time"
)

// OperationKind enumerates the closed set of atomic apply steps.
type OperationKind string

const (
	OperationKindMoveFile      OperationKind = "move_file"
	OperationKindRewriteImport OperationKind = "rewrite_import"
	OperationKindDeleteFile    OperationKind = "delete_file"
	OperationKindCreateFile    OperationKind = "create_file"
)

// ApplyStatus is the lifecycle state of an apply run.
type ApplyStatus string

const (
	ApplyStatusPlanned    ApplyStatus = "planned"
	ApplyStatusRunning    ApplyStatus = "running"
	ApplyStatusBuildGreen ApplyStatus = "build_green"
	ApplyStatusBuildRed   ApplyStatus = "build_red"
	ApplyStatusReverted   ApplyStatus = "reverted"
	ApplyStatusCommitted  ApplyStatus = "committed"
)

// Operation is one atomic step in a plan.
type Operation struct {
	ID                  string
	Kind                OperationKind
	FromPath            string
	ToPath              string
	Payload             []byte
	ResolvesConflictIDs []string
}

// Plan is an ordered set of Operations targeting one domain.
type Plan struct {
	ID          string
	Scenario    string
	Domain      string
	Operations  []Operation
	ConflictIDs []string
	PlannedAt   time.Time
}

// ApplyRun records one execution of a Plan.
type ApplyRun struct {
	ID         string
	PlanID     string
	Scenario   string
	Domain     string
	Status     ApplyStatus
	BuildLog   string
	StartedAt  time.Time
	FinishedAt time.Time
}

// BuildBaseline records the toolchain pre-apply state.
type BuildBaseline struct {
	Scenario   string
	Green      bool
	Toolchain  string
	Log        string
	CapturedAt time.Time
}

// ErrInvalidPlanRequest is the typed sentinel returned by PlanApply
// when input is incomplete.
type ErrInvalidPlanRequest struct {
	Field  string
	Reason string
}

func (e ErrInvalidPlanRequest) Error() string {
	return fmt.Sprintf("invalid plan request: %s: %s", e.Field, e.Reason)
}

// ErrApplyUnimplemented is the typed sentinel RunApply returns. v0.1
// surfaces this as connect.CodeUnimplemented; later plans replace it
// with real execution.
type ErrApplyUnimplemented struct {
	NextPlan string
}

func (e ErrApplyUnimplemented) Error() string {
	if e.NextPlan == "" {
		return "apply run not implemented in v0.1"
	}
	return fmt.Sprintf("apply run not implemented in v0.1; tracked by %s", e.NextPlan)
}

// ErrSuppressionUnconfigured is returned by WriteSuppression when the apply
// service was constructed without a marker writer/locator.
type ErrSuppressionUnconfigured struct{}

func (ErrSuppressionUnconfigured) Error() string {
	return "suppression-marker writing is not configured on this apply service"
}

// Package validation is the plan-health domain. It resolves a plan's code
// references against code-facts, computes staleness tiers from referenced-code
// change, derives each phase's exact baseline/validation command set across all
// affected locations, runs those baselines on request (compute + run, the agent
// in the loop), and verifies Definition-of-Done against the regression anchor as
// an oracle. Every cross-scenario call is behind a seam and degrades to a marked
// gap (UNKNOWN), never a false PASS.
//
// Layering:
//
//	handler → Service → {PlanSource, ReferenceResolver, StalenessComputer, CommandRunner}
//	            ↑            ↑ plans domain   ↑ code-facts      ↑ fs/git       ↑ git-control-tower
//	        (proto edge)   (all faked in tests; all degrade gracefully)
//
// The structured plan/phase/reference Go types live in the neutral planmodel kernel;
// validation imports that kernel and annotates
// references with resolution/staleness. It does NOT own project-level validation
// of resources/packages — that is consumed from test-genie / scenario-validation.
package validation

import (
	"context"

	planmodel "plan-manager/internal/planmodel"
)

// OperationStatus is the durable lifecycle of a validation operation.
type OperationStatus string

const (
	OperationQueued   OperationStatus = "queued"
	OperationRunning  OperationStatus = "running"
	OperationTerminal OperationStatus = "terminal"
)

// ChildStatus is the durable lifecycle of one command child.
type ChildStatus string

const (
	ChildQueued   ChildStatus = "queued"
	ChildRunning  ChildStatus = "running"
	ChildTerminal ChildStatus = "terminal"
)

// OperationError is a typed, persisted failure. Detail is operator-facing;
// callers branch on Code rather than parsing it.
type OperationError struct {
	Code   string
	Detail string
}

// ValidationChild is one durable oracle or informational command checkpoint.
type ValidationChild struct {
	ID string
	// Check is the durable semantic identity. Command is only the deterministic
	// dispatch projection retained for operators and legacy compatibility.
	Check      ValidationCheck
	Command    string
	Oracle     bool
	Status     ChildStatus
	Attempt    int
	ExternalID string
	Verdict    Verdict
	Detail     string
	Error      *OperationError
	QueuedAt   string
	StartedAt  string
	TerminalAt string
}

// ValidationCheckKind is deliberately small: each kind has one stable meaning
// and therefore one semantic deduplication key.
type ValidationCheckKind string

const (
	ValidationCheckScenarioDiff     ValidationCheckKind = "scenario_baseline_diff"
	ValidationCheckCollectionDiff   ValidationCheckKind = "collection_diff"
	ValidationCheckPathSnapshotDiff ValidationCheckKind = "path_snapshot_diff"
	ValidationCheckRepoDiff         ValidationCheckKind = "repo_diff"
	ValidationCheckCustom           ValidationCheckKind = "custom_command"
)

// ValidationCheck is a typed validation work item. SemanticKey is independent
// of presentation flags such as --json, so rendering variants cannot multiply
// expensive work.
type ValidationCheck struct {
	Kind        ValidationCheckKind
	SemanticKey string
	Scenario    string
	Baseline    string
	Paths       []string
	Scenarios   []string
	Branch      string
	Command     string
	Oracle      bool
}

// ValidationOperation is the server-owned, reattachable validation record.
// Queue, execution, and transport budgets are deliberately independent.
type ValidationOperation struct {
	SchemaVersion              int
	ID                         string
	PlanID                     string
	PhaseID                    string
	IdempotencyKey             string
	Status                     OperationStatus
	Attempt                    int
	Children                   []ValidationChild
	Result                     *Result
	ResultRef                  string
	Error                      *OperationError
	QueuedAt                   string
	StartedAt                  string
	TerminalAt                 string
	QueueBudgetSeconds         int
	ExecutionBudgetSeconds     int
	TransportWaitBudgetSeconds int
	RecommendedWaitSeconds     int
	// ScopeFingerprint pins the plan content/scope compiled into this operation.
	// A changed plan gets a fresh validation only after the active operation ends.
	ScopeFingerprint string
	// QueueReason is durable operator guidance while the operation is not terminal.
	QueueReason string
}

const CurrentOperationSchemaVersion = 2

// Terminal reports whether no more server-owned work remains.
func (o ValidationOperation) Terminal() bool { return o.Status == OperationTerminal }

// Verdict is the outcome of a validation/DoD check. Unknown is the honest
// degraded result when a composed dependency is unavailable — never a false pass.
type Verdict string

const (
	VerdictUnspecified Verdict = ""
	VerdictPass        Verdict = "pass"
	VerdictFail        Verdict = "fail"
	VerdictUnknown     Verdict = "unknown"
)

// Result is a validation/baseline outcome for a plan or phase. Computed by the
// service (verdict from baseline diff exit-0 as oracle).
type Result struct {
	ID              string
	PlanID          string
	PhaseID         string
	Verdict         Verdict
	Staleness       planmodel.StalenessTier
	CommandsRun     []string
	CommandFindings []CommandFinding
	Detail          string
	RanAt           string
}

// ReferenceReport is the resolved-reference view returned by ResolveReferences /
// ComputeStaleness: the annotated references plus whether a dependency degraded.
type ReferenceReport struct {
	References []planmodel.Reference
	Overall    planmodel.StalenessTier
	Degraded   bool
}

// BaselineScope is the derived command set across all affected locations for a
// phase (or plan), plus the distinct locations the commands cover.
type BaselineScope struct {
	Commands  []string
	Locations []string
}

// BaselineCapture reports the outcome of capturing the regression-anchor's
// baseline snapshot at execution start. It is honest: Captured=false with a
// Detail when git-control-tower is unavailable or the anchor intent is still a
// placeholder — never a fabricated capture.
type BaselineCapture struct {
	Captured         bool
	Scenario         string
	BaselineName     string
	ScenarioTargets  []string
	RepoPaths        []string
	CollectionBranch string
	Members          []BaselineCollectionMember
	PathSnapshots    []BaselinePathSnapshot
	Required         int
	Ready            int
	Pending          int
	Failed           int
	Skipped          int
	Stale            int
	RunID            string
	SchemaVersion    int
	DegradedReasons  []string
	Detail           string
}

type CommandReferenceValidator interface {
	ValidateCommandReference(context.Context, CommandReferenceRequest) (CommandReferenceResult, error)
}

type CommandReferenceRequest struct {
	CommandText string
	Qualifiers  []string
}

type CommandReferenceResult struct {
	Verdict         string
	ValidationLevel string
	Issues          []CommandIssue
	Suggestions     []string
	Guidance        []string
}

type CommandIssue struct {
	Code    string
	Message string
}

type CommandFinding struct {
	CommandText string
	Verdict     string
	Level       string
	Message     string
	Location    string
	IssueCodes  []string
	Suggestions []string
	Guidance    []string
}

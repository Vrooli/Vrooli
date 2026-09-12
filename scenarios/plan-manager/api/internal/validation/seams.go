package validation

import (
	"context"

	planmodel "plan-manager/internal/planmodel"
)

// PlanSource is the read seam onto the plans SSOT. Production wraps the plans
// domain Service; tests inject a fake. validation only reads plans — it never
// mutates them.
type PlanSource interface {
	GetPlan(ctx context.Context, idOrSlug string) (planmodel.Plan, error)
}

// ReferenceResolver resolves a single code/req/doc reference against code-facts.
// Production wires a code-facts-backed resolver; tests inject a fake. A nil
// resolver (or one returning an error) degrades the reference to UNRESOLVED —
// recorded but unresolved, never a false "resolved".
type ReferenceResolver interface {
	Resolve(ctx context.Context, ref planmodel.Reference) (planmodel.Reference, error)
}

// StalenessComputer computes the staleness tier + change factor of a resolved
// reference. Production wires the filesystem existence floor plus git-sourced
// per-reference drift refinement; tests inject a fake. A nil computer (or an
// error) degrades staleness to UNKNOWN.
type StalenessComputer interface {
	Compute(ctx context.Context, ref planmodel.Reference) (planmodel.StalenessTier, float64, error)
}

// ResultStore persists validation results and reads back the last-known result
// per plan/phase. RunValidation (the explicit, agent-in-the-loop check) writes;
// the execution context server reads the LAST STORED result so status/next never
// shell a subprocess. A nil store means "no persistence" — RunValidation still
// returns its live result, but nothing is cached for later cheap reads.
type ResultStore interface {
	SaveResult(ctx context.Context, r Result) error
	GetResult(ctx context.Context, id string) (Result, bool, error)
	LastResult(ctx context.Context, planID, phaseID string) (Result, bool, error)
}

// OperationStore is the durable checkpoint ledger for validation operations.
// CreateOperation atomically deduplicates an explicit key by (plan, phase,
// execution, scope generation, key); unkeyed active starts coalesce only in
// that same execution-local scope.
type OperationStore interface {
	CreateOperation(ctx context.Context, op ValidationOperation) (stored ValidationOperation, created bool, err error)
	SaveOperation(ctx context.Context, op ValidationOperation) error
	GetOperation(ctx context.Context, id string) (ValidationOperation, bool, error)
	ListNonTerminalOperations(ctx context.Context) ([]ValidationOperation, error)
}

// BaselineCollectionClient is Plan Manager's typed seam to Git Control Tower.
// It carries collection policy and coverage as data, avoiding CLI text parsing
// or reconstruction of one child command per scenario. The concrete Connect
// adapter belongs at the module edge; tests inject this small interface.
type BaselineCollectionClient interface {
	StartCollectionCapture(ctx context.Context, req BaselineCollectionCaptureRequest) (BaselineCollectionCaptureResult, error)
	StartCollectionDiff(ctx context.Context, req BaselineCollectionDiffRequest) (BaselineCollectionDiffResult, error)
	GetCollection(ctx context.Context, name, branch string) (BaselineCollectionCaptureResult, error)
	GetCollectionDiff(ctx context.Context, name, branch, operationID string) (BaselineCollectionDiffResult, error)
	DiffPathEvidence(ctx context.Context, req BaselinePathDiffRequest) (BaselinePathDiffResult, error)
}

// TestRunClient reads a durable Test Genie run snapshot. It deliberately has no
// wait method: Test Genie remains the only owner of its native wait, timeout,
// recovery, and Agent Manager parking contract.
type TestRunClient interface {
	GetRun(ctx context.Context, scenario, runID string) (TestRunEvidence, error)
}

type BaselineCollectionCaptureRequest struct {
	Name      string
	Scenarios []string
	RepoPaths []string
}

type BaselineCollectionCaptureResult struct {
	Name          string
	Branch        string
	Required      int
	Ready         int
	Pending       int
	Failed        int
	Skipped       int
	Stale         int
	Members       []BaselineCollectionMember
	PathSnapshots []BaselinePathSnapshot
}

// BaselineCollectionMember and BaselinePathSnapshot are the durable GCT
// provenance that Plan Manager stores on its execution checkpoint. They are
// evidence references, never reconstructed from plan prose on resume.
type BaselineCollectionMember struct {
	Scenario     string
	BaselineName string
	Required     bool
	Status       string
	RunID        string
	GitSHA       string
	Error        string
}

type BaselinePathSnapshot struct {
	Name      string
	Branch    string
	CreatedAt string
}

type BaselineCollectionDiffRequest struct {
	Name        string
	OperationID string
	Scenarios   []string
}

type BaselineCollectionDiffResult struct {
	OperationID    string
	Classification string
	Detail         string
}

type BaselinePathDiffRequest struct {
	BeforeName  string
	Branch      string
	Paths       []string
	OperationID string
}

type BaselinePathDiffResult struct {
	AfterName string
	Deltas    int
	Detail    string
}

// BaselineInventorySource supplies the latest execution-owned collection
// checkpoint. Validation uses its captured target list in preference to mutable
// plan prose, while keeping plan policy as the fallback before execution starts.
type BaselineInventorySource interface {
	LatestBaselineInventory(ctx context.Context, planID string) (BaselineInventory, bool, error)
}

type BaselineInventory struct {
	Name            string
	Branch          string
	ScenarioTargets []string
	PathSnapshots   []BaselinePathSnapshot
	Complete        bool
}

func (r BaselineCollectionCaptureResult) Complete() bool {
	return r.Required > 0 && r.Required == r.Ready
}

package investigationbrief

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	internalexecution "plan-manager/internal/execution"
	planmodel "plan-manager/internal/planmodel"
	internalvalidation "plan-manager/internal/validation"

	"github.com/vrooli/api-core/schedule"
)

const (
	SchemaVersion = "plan-investigation-brief/v1"
	maxText       = 4096
	maxIDs        = 8
)

var (
	ErrInvalidRequest   = errors.New("invalid investigation brief request")
	ErrExecutionMissing = errors.New("investigation brief execution was not found")
	ErrPhaseStale       = errors.New("investigation brief phase is stale or not current")
	ErrRunSetMismatch   = errors.New("investigation brief run set does not match authoritative execution linkage")
	ErrOperationScope   = errors.New("investigation validation operation is outside the requested execution scope")
)

// Request identifies one execution-local, phase-local brief. RunIDs are a
// caller assertion and are checked against the execution row; they are never
// treated as authoritative input.
type Request struct {
	ExecutionID           string
	PhaseID               string
	RunIDs                []string
	ValidationOperationID string
}

type PlanSource interface {
	GetPlan(ctx context.Context, id string) (planmodel.Plan, error)
}

type Provider interface {
	Get(ctx context.Context, req Request) (Brief, error)
}

type Brief struct {
	SchemaVersion      string
	Status             string
	ExecutionID        string
	PlanID             string
	PlanRevision       string
	PhaseID            string
	PhaseGeneration    int
	RunIDs             []string
	ExpectedOutcome    string
	AcceptanceCriteria string
	MaterialProgress   []ProgressMarker
	ProducerWait       *ProducerWait
	WallTimeSeconds    int64
	ActiveWorkSeconds  int64
	KnownWaitSeconds   int64
	Budget             Budget
	OmittedReasons     []string
}

type ProgressMarker struct {
	Kind        string
	EvidenceRef string
	Detail      string
	ObservedAt  string
}

type ProducerWait struct {
	OperationID string
	Status      string
	QueueReason string
	WaitArgv    []string
	SyncArgv    []string
	Explicit    bool
}

type Budget struct {
	Known                bool
	Basis                string
	QueueSeconds         int
	ExecutionSeconds     int
	TransportWaitSeconds int
}

type provider struct {
	plans      PlanSource
	executions internalexecution.Repository
	operations internalvalidation.OperationStore
	clock      schedule.Clock
}

func NewProvider(plans PlanSource, executions internalexecution.Repository, operations internalvalidation.OperationStore, clock schedule.Clock) Provider {
	if clock == nil {
		clock = schedule.System()
	}
	return &provider{plans: plans, executions: executions, operations: operations, clock: clock}
}

func (p *provider) Get(ctx context.Context, req Request) (Brief, error) {
	req.ExecutionID = strings.TrimSpace(req.ExecutionID)
	req.PhaseID = strings.TrimSpace(req.PhaseID)
	req.ValidationOperationID = strings.TrimSpace(req.ValidationOperationID)
	if req.ExecutionID == "" || req.PhaseID == "" || len(req.RunIDs) > maxIDs {
		return Brief{}, fmt.Errorf("%w: execution_id, phase_id, and at most %d run_ids are required", ErrInvalidRequest, maxIDs)
	}
	requestedRuns := normalizeIDs(req.RunIDs)
	execution, found, err := p.executions.GetExecution(ctx, req.ExecutionID)
	if err != nil {
		return Brief{}, fmt.Errorf("read execution: %w", err)
	}
	if !found {
		return Brief{}, ErrExecutionMissing
	}
	if strings.TrimSpace(execution.CurrentPhaseID) != req.PhaseID {
		return Brief{}, fmt.Errorf("%w: execution current phase is %q", ErrPhaseStale, execution.CurrentPhaseID)
	}
	authoritativeRuns := normalizeIDs([]string{execution.RunID})
	if len(requestedRuns) > 0 && !sameIDs(requestedRuns, authoritativeRuns) {
		return Brief{}, fmt.Errorf("%w: requested=%v authoritative=%v", ErrRunSetMismatch, requestedRuns, authoritativeRuns)
	}

	plan, err := p.plans.GetPlan(ctx, execution.PlanID)
	if err != nil {
		return Brief{}, fmt.Errorf("read plan: %w", err)
	}
	phase, ok := phaseByID(plan, req.PhaseID)
	if !ok {
		return Brief{}, fmt.Errorf("%w: phase %q is absent from plan %q", ErrPhaseStale, req.PhaseID, plan.ID)
	}

	status := "ready"
	omitted := make([]string, 0, 4)
	if len(authoritativeRuns) == 0 {
		status = "unresolved"
		omitted = append(omitted, "authoritative execution run identity is unavailable")
	}
	brief := Brief{
		SchemaVersion:      SchemaVersion,
		Status:             status,
		ExecutionID:        execution.ID,
		PlanID:             plan.ID,
		PlanRevision:       bounded(plan.ContentHash),
		PhaseID:            phase.ID,
		PhaseGeneration:    execution.PhaseValidationGenerations[phase.ID],
		RunIDs:             authoritativeRuns,
		ExpectedOutcome:    bounded(firstNonEmpty(plan.TargetOutcome, plan.Purpose)),
		AcceptanceCriteria: bounded(phase.Acceptance),
		OmittedReasons:     omitted,
	}
	if brief.ExpectedOutcome == "" {
		brief.OmittedReasons = append(brief.OmittedReasons, "plan has no authored target outcome or purpose")
	}
	if brief.AcceptanceCriteria == "" {
		brief.OmittedReasons = append(brief.OmittedReasons, "phase has no authored acceptance criteria")
	}

	now := p.clock.Now().UTC()
	brief.WallTimeSeconds = elapsedSeconds(execution.StartedAt, now)
	brief.ActiveWorkSeconds = brief.WallTimeSeconds

	operationID := req.ValidationOperationID
	var op internalvalidation.ValidationOperation
	operationFound := false
	if operationID != "" {
		if p.operations == nil {
			return Brief{}, fmt.Errorf("%w: operation store unavailable", ErrOperationScope)
		}
		var found bool
		op, found, err = p.operations.GetOperation(ctx, operationID)
		if err != nil {
			return Brief{}, fmt.Errorf("read validation operation: %w", err)
		}
		if !found {
			return Brief{}, fmt.Errorf("%w: operation %q not found", ErrOperationScope, operationID)
		}
		operationFound = true
	} else if p.operations != nil {
		pending, listErr := p.operations.ListNonTerminalOperations(ctx)
		if listErr != nil {
			return Brief{}, fmt.Errorf("list current validation operations: %w", listErr)
		}
		for _, candidate := range pending {
			if candidate.ExecutionID == execution.ID && candidate.PlanID == plan.ID && candidate.PhaseID == phase.ID {
				if operationFound {
					brief.OmittedReasons = append(brief.OmittedReasons, "multiple current validation operations exist; producer wait is unresolved")
					operationFound = false
					op = internalvalidation.ValidationOperation{}
					break
				}
				op = candidate
				operationFound = true
			}
		}
		if operationFound {
			operationID = op.ID
		}
	}
	if operationFound {
		if op.ExecutionID != "" && op.ExecutionID != execution.ID || op.PlanID != "" && op.PlanID != plan.ID || op.PhaseID != "" && op.PhaseID != phase.ID {
			return Brief{}, fmt.Errorf("%w: operation=%q", ErrOperationScope, op.ID)
		}
		brief.Budget = budgetFrom(op)
		if !op.Terminal() {
			brief.ProducerWait = &ProducerWait{OperationID: op.ID, Status: string(op.Status), QueueReason: bounded(op.QueueReason), WaitArgv: boundedArgs(op.ProducerWaitArgv), SyncArgv: boundedArgs(op.SyncArgv), Explicit: true}
			brief.KnownWaitSeconds = knownWaitSeconds(op, now)
			if brief.KnownWaitSeconds > brief.WallTimeSeconds {
				brief.KnownWaitSeconds = brief.WallTimeSeconds
			}
			brief.ActiveWorkSeconds = brief.WallTimeSeconds - brief.KnownWaitSeconds
		}
		appendReceipt(&brief, op.Result)
	} else {
		if operationID == "" {
			brief.OmittedReasons = append(brief.OmittedReasons, "no current validation operation was found; material progress is not inferred from unrelated commits")
		}
	}
	if len(brief.MaterialProgress) == 0 {
		if store, ok := p.operations.(internalvalidation.ResultStore); ok {
			result, found, resultErr := latestScopedResult(ctx, store, plan.ID, phase.ID, execution.ID, brief.PhaseGeneration)
			if resultErr != nil {
				return Brief{}, fmt.Errorf("read current validation receipt: %w", resultErr)
			}
			if found {
				appendReceipt(&brief, &result)
			}
		}
	}
	if len(brief.MaterialProgress) == 0 {
		brief.OmittedReasons = append(brief.OmittedReasons, "current validation operation has no terminal receipt")
	}
	return brief, nil
}

func appendReceipt(brief *Brief, result *internalvalidation.Result) {
	if result == nil || strings.TrimSpace(result.ID) == "" {
		return
	}
	brief.MaterialProgress = append(brief.MaterialProgress, ProgressMarker{Kind: "validation_receipt", EvidenceRef: "validation:" + bounded(result.ID), Detail: bounded(result.Detail), ObservedAt: bounded(result.RanAt)})
}

func latestScopedResult(ctx context.Context, store internalvalidation.ResultStore, planID, phaseID, executionID string, generation int) (internalvalidation.Result, bool, error) {
	result, found, err := store.LastResult(ctx, planID, phaseID)
	if err != nil || !found {
		return result, found, err
	}
	if result.ExecutionID != executionID || result.ScopeGeneration != generation {
		return internalvalidation.Result{}, false, nil
	}
	return result, true, nil
}

func phaseByID(plan planmodel.Plan, id string) (planmodel.Phase, bool) {
	for _, phase := range plan.Phases {
		if phase.ID == id {
			return phase, true
		}
	}
	return planmodel.Phase{}, false
}

func budgetFrom(op internalvalidation.ValidationOperation) Budget {
	known := op.QueueBudgetSeconds > 0 || op.ExecutionBudgetSeconds > 0 || op.TransportWaitBudgetSeconds > 0
	basis := "explicit validation operation budget"
	if !known {
		basis = "unknown: validation operation did not record a budget"
	}
	return Budget{Known: known, Basis: basis, QueueSeconds: op.QueueBudgetSeconds, ExecutionSeconds: op.ExecutionBudgetSeconds, TransportWaitSeconds: op.TransportWaitBudgetSeconds}
}

func knownWaitSeconds(op internalvalidation.ValidationOperation, now time.Time) int64 {
	queued, qOK := parseTime(op.QueuedAt)
	started, sOK := parseTime(op.StartedAt)
	if !qOK {
		return 0
	}
	end := now
	if sOK {
		end = started
	}
	if end.Before(queued) {
		return 0
	}
	return int64(end.Sub(queued) / time.Second)
}

func elapsedSeconds(started string, now time.Time) int64 {
	t, ok := parseTime(started)
	if !ok || now.Before(t) {
		return 0
	}
	return int64(now.Sub(t) / time.Second)
}

func parseTime(value string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return t, err == nil
}

func normalizeIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				out = append(out, id)
			}
		}
	}
	sort.Strings(out)
	return out
}

func sameIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func bounded(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxText {
		return value[:maxText]
	}
	return value
}

func boundedArgs(values []string) []string {
	if len(values) > maxIDs {
		values = values[:maxIDs]
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, bounded(value))
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

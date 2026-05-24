package rewrite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"

	intgraph "go-code-graph/internal/graph"
)

// Service orchestrates Plan and Apply. The pattern matches the graph
// domain: validate → acquire per-path mutex → call seam → return.
// All filesystem-touching work flows through RewriteExecutor; all
// persistence through PlanStore.
type Service struct {
	store    PlanStore
	executor RewriteExecutor
	mu       *intgraph.PathMutex
	oplog    OperationLog
}

// NewService wires the production Service with a no-op operation log.
// The same *intgraph.PathMutex instance must be shared with the graph
// domain so concurrent Extract/Apply calls for the same scenario_path
// serialize (OT-P0-006). Callers that want REQ-P1-002 persistent
// audit rows should use NewServiceWithLog.
func NewService(store PlanStore, executor RewriteExecutor, mu *intgraph.PathMutex) *Service {
	return NewServiceWithLog(store, executor, mu, NoopOperationLog{})
}

// NewServiceWithLog wires the production Service with an explicit
// OperationLog (REQ-P1-002). Pass *SQLiteStore (which satisfies both
// PlanStore and OperationLog) to get durable audit rows; tests wire
// NoopOperationLog (the default) so apply paths stay disk-free.
func NewServiceWithLog(store PlanStore, executor RewriteExecutor, mu *intgraph.PathMutex, oplog OperationLog) *Service {
	if oplog == nil {
		oplog = NoopOperationLog{}
	}
	return &Service{store: store, executor: executor, mu: mu, oplog: oplog}
}

// Plan validates, normalizes, hashes, persists, and returns the plan.
// The PlanID is sha256-hex(canonical JSON of ScenarioPath +
// normalized operations) — identical inputs always produce identical
// plan_ids.
func (s *Service) Plan(ctx context.Context, in PlanInput) (Plan, error) {
	if strings.TrimSpace(in.ScenarioPath) == "" {
		return Plan{}, RewriteError{
			Kind:    RewriteErrorMalformedOperation,
			Message: "scenario_path is required",
		}
	}
	if len(in.Operations) == 0 {
		return Plan{}, RewriteError{
			Kind:    RewriteErrorNoOperations,
			Message: "operations list must contain at least one entry",
		}
	}
	if err := ValidateOperations(in.Operations); err != nil {
		return Plan{}, err
	}
	normalized := Normalize(in.Operations)
	if len(normalized) == 0 {
		return Plan{}, RewriteError{
			Kind:    RewriteErrorNoOperations,
			Message: "operations list normalized to empty",
		}
	}

	id, err := derivePlanID(in.ScenarioPath, normalized)
	if err != nil {
		return Plan{}, RewriteError{Kind: RewriteErrorInternal, Message: "derive plan id", Cause: err}
	}
	plan := Plan{
		ID:           id,
		ScenarioPath: in.ScenarioPath,
		Operations:   normalized,
	}
	if err := s.store.Save(ctx, plan); err != nil {
		return Plan{}, RewriteError{Kind: RewriteErrorInternal, Message: "save plan", Cause: err}
	}
	return plan, nil
}

// Apply loads the plan, validates apply=true, optionally short-circuits
// for dry-run, then iterates ops calling the executor and records each
// per-op status. A failed op does NOT abort the remaining ops — partial
// state is intentional per OT-P0-003 and surfaced via OperationResult.
func (s *Service) Apply(ctx context.Context, in ApplyInput) (ApplyResult, error) {
	if !in.Apply {
		return ApplyResult{}, RewriteError{
			Kind:    RewriteErrorApplyNotSet,
			Message: "apply must be true; dry-run callers must set X-Dry-Run header instead",
		}
	}
	if strings.TrimSpace(string(in.PlanID)) == "" {
		return ApplyResult{}, RewriteError{
			Kind:    RewriteErrorPlanNotFound,
			Message: "plan_id is required",
		}
	}
	plan, err := s.store.Load(ctx, in.PlanID)
	if err != nil {
		return ApplyResult{}, err
	}
	if strings.TrimSpace(in.ScenarioPath) == "" {
		return ApplyResult{}, RewriteError{
			Kind:    RewriteErrorMalformedOperation,
			Message: "scenario_path is required",
		}
	}
	if !samePath(plan.ScenarioPath, in.ScenarioPath) {
		return ApplyResult{}, RewriteError{
			Kind:    RewriteErrorPathMismatch,
			Message: "plan was authored against a different scenario_path",
		}
	}

	abs, err := filepath.Abs(in.ScenarioPath)
	if err != nil {
		return ApplyResult{}, RewriteError{Kind: RewriteErrorInternal, Message: "resolve scenario path", Cause: err}
	}

	// Dry-run: skip the executor entirely, return synthetic OK for
	// every op so the caller sees the realistic shape.
	if in.DryRun {
		results := make([]OperationResult, 0, len(plan.Operations))
		for _, op := range plan.Operations {
			results = append(results, OperationResult{Operation: op, Status: OperationStatusOK})
		}
		return ApplyResult{PlanID: plan.ID, Results: results, DryRun: true}, nil
	}

	unlock := s.mu.Lock(abs)
	defer unlock()

	results := make([]OperationResult, 0, len(plan.Operations))
	for _, op := range plan.Operations {
		execErr := s.executor.Execute(ctx, abs, op)
		if execErr == nil {
			results = append(results, OperationResult{Operation: op, Status: OperationStatusOK})
			continue
		}
		results = append(results, OperationResult{
			Operation: op,
			Status:    OperationStatusFailed,
			Message:   execErr.Error(),
		})
	}
	// Record the per-op outcome to the operation log. We swallow the
	// log error and surface only the apply result — audit failure must
	// not mask a successful (or partially successful) apply.
	_ = s.oplog.RecordApply(ctx, plan.ID, results)
	return ApplyResult{PlanID: plan.ID, Results: results, DryRun: false}, nil
}

// derivePlanID computes the deterministic plan id from the scenario
// path and normalized operations. JSON is the canonical serialization
// because every field is a primitive string and Go's encoding/json
// produces stable output for structs.
func derivePlanID(scenarioPath string, ops []Operation) (PlanID, error) {
	payload := struct {
		ScenarioPath string           `json:"scenario_path"`
		Operations   []map[string]any `json:"operations"`
	}{
		ScenarioPath: scenarioPath,
		Operations:   make([]map[string]any, 0, len(ops)),
	}
	for _, op := range ops {
		switch o := op.(type) {
		case FileMove:
			payload.Operations = append(payload.Operations, map[string]any{
				"kind": string(OperationKindFileMove),
				"from": o.From,
				"to":   o.To,
			})
		case ImportRewrite:
			payload.Operations = append(payload.Operations, map[string]any{
				"kind": string(OperationKindImportRewrite),
				"old":  o.Old,
				"new":  o.New,
			})
		}
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf)
	return PlanID(hex.EncodeToString(sum[:])), nil
}

// samePath compares two scenario paths via their absolute forms so
// "./x" and "/abs/x" match when the caller's cwd makes them equivalent.
// Falls back to literal string compare when filepath.Abs fails.
func samePath(a, b string) bool {
	aa, aerr := filepath.Abs(a)
	bb, berr := filepath.Abs(b)
	if aerr != nil || berr != nil {
		return a == b
	}
	return aa == bb
}

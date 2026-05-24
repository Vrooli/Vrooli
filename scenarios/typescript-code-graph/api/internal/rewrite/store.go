package rewrite

import "errors"

// ErrPlanNotFound is the typed sentinel PlanStore implementations
// return when (scenario_path, plan_id) does not exist. Service.Apply
// wraps this in a RewriteError{Kind:RewriteErrorPlanNotFound} for the
// handler to translate to CodeFailedPrecondition.
var ErrPlanNotFound = errors.New("plan not found")

// PlanStore persists Plan values keyed by (scenario_path, PlanID). The
// composite key prevents cross-scenario replay: two scenarios that
// happen to derive the same PlanID (same operation list) cannot apply
// each other's plans.
//
// seam: production wires *MemoryPlanStore; tests wire either the same
// in-memory store directly or mocks.PlanStore when they need to inject
// failures.
type PlanStore interface {
	// Save records the plan. Calling Save twice with the same
	// (ScenarioPath, ID) is idempotent — the operation list is identical
	// because PlanID is derived from it.
	Save(plan Plan) error

	// Get returns the plan for (scenarioPath, id). Returns
	// ErrPlanNotFound if no such entry exists.
	Get(scenarioPath string, id PlanID) (Plan, error)
}

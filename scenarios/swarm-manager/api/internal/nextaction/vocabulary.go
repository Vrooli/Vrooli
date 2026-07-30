// Package nextaction owns the stable, cross-domain vocabulary used by the
// operator inbox. Domain packages resolve their own facts, but do not invent
// action IDs or classify blockers from presentation text.
package nextaction

import "fmt"

type ID string

const (
	None                ID = "none"
	AcceptSuggestion    ID = "accept_suggestion"
	AuthorPlan          ID = "author_plan"
	AcceptPlan          ID = "accept_plan"
	RepairPlan          ID = "repair_plan"
	ResolveDependencies ID = "resolve_dependencies"
	Review              ID = "review"
	ViewExecution       ID = "view_execution"
	Run                 ID = "run"
	Retry               ID = "retry"
	Archive             ID = "archive"
	Decide              ID = "decide"
	DispatchFollowup    ID = "dispatch_followup"
	AuthorFollowup      ID = "author_followup"
	PlanGoal            ID = "plan_goal"
	DefineCriteria      ID = "define_criteria"
	CloseOut            ID = "close_out"
	Chain               ID = "chain"
)

type BlockerCode string

const (
	PlanChanged       BlockerCode = "plan_changed"
	PlanNotAccepted   BlockerCode = "plan_not_accepted"
	PlanInvalid       BlockerCode = "plan_invalid"
	UnmetDependencies BlockerCode = "unmet_dependencies"
	QueueCap          BlockerCode = "queue_cap"
	CostCap           BlockerCode = "cost_cap"
	CircuitOpen       BlockerCode = "circuit_open"
)

// ValidateBlockerCode fails closed so a new execution preflight blocker must
// be given an intentional operator action before it reaches the inbox.
func ValidateBlockerCode(code string) error {
	switch BlockerCode(code) {
	case PlanChanged, PlanNotAccepted, PlanInvalid, UnmetDependencies, QueueCap, CostCap, CircuitOpen:
		return nil
	default:
		return fmt.Errorf("unmapped next-action blocker code %q", code)
	}
}

// ActionForBlocker is policy over typed codes only; message wording is never
// used to infer a UI action.
func ActionForBlocker(code string) ID {
	switch BlockerCode(code) {
	case PlanChanged, PlanNotAccepted:
		return AcceptPlan
	case PlanInvalid:
		return RepairPlan
	case UnmetDependencies:
		return ResolveDependencies
	default:
		return Run
	}
}

// TransitionKey is the one server-owned bridge from an action decision to a
// declared transition. Clients consume the projection and do not duplicate it.
func TransitionKey(id ID) string {
	switch id {
	case AuthorPlan:
		return "plan.author"
	case RepairPlan:
		return "plan.repair"
	case DispatchFollowup:
		return "follow_up.dispatch"
	case PlanGoal, DefineCriteria:
		return "goal.plan"
	case CloseOut:
		return "goal.close_out"
	default:
		return ""
	}
}

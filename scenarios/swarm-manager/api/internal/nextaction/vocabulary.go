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

// Effect declares what performing an action does to the system.
//
// It exists because TransitionKey cannot answer that question. Several actions
// dispatch agent work through transitions whose subject the client does not
// hold — `run` starts plan.execute against an execution ID, `review` starts
// work.review against a round-scoped ref — so their TransitionKey is
// deliberately empty. Without a separate declaration the client had no way to
// distinguish "this saves a field" from "this spends minutes of agent time",
// and was reduced to pattern-matching action ids.
//
// Effect is about the system, not the presentation: it says an agent runs, not
// how a button should look.
type Effect string

const (
	// EffectNone performs no server-side work; the client only navigates.
	EffectNone Effect = "none"
	// EffectStateChange applies an immediate, cheap server-side change.
	EffectStateChange Effect = "state_change"
	// EffectAgentRun dispatches an autonomous agent run: minutes, and tokens.
	EffectAgentRun Effect = "agent_run"
	// EffectAgentSession opens an interactive agent session.
	EffectAgentSession Effect = "agent_session"
)

// EffectFor classifies an action.
//
// The default is EffectStateChange, not EffectNone: a new action that nobody
// classified should be presented as doing something, because under-promising
// a side effect is safer than implying there is none.
func EffectFor(id ID) Effect {
	switch id {
	case None, Decide, ResolveDependencies, ViewExecution, Chain:
		return EffectNone
	case Run, Retry, Review, AuthorPlan, RepairPlan, PlanGoal, AuthorFollowup:
		// Each of these ends in a declared `workflow` transition:
		// plan.execute, work.correct/work.follow_up, work.review or
		// milestone.review, plan.author, plan.repair, goal.plan.
		return EffectAgentRun
	case AcceptSuggestion, AcceptPlan, Archive, DispatchFollowup, DefineCriteria, CloseOut:
		// follow_up.dispatch and goal.close_out are declared `deterministic`;
		// the rest write a field directly.
		return EffectStateChange
	default:
		return EffectStateChange
	}
}

// IsDestructive marks actions that remove or interrupt state. It is separate
// from Effect because the two are orthogonal: an agent run is expensive but
// not destructive, and archiving is destructive but cheap.
func IsDestructive(id ID) bool {
	return id == Archive
}

// TransitionKey is the one server-owned bridge from an action decision to a
// declared transition. Clients consume the projection and do not duplicate it.
//
// It is empty for actions whose transition takes a subject the client does not
// hold (an execution ID, a review round). Those still declare an Effect.
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

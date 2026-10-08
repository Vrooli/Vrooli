package goalhome

import (
	"fmt"
	"strings"
	"time"
)

// Park policy for an orchestrator (DL-1, decision P-06). A park waits on
// something: a live direct child (a short backstop for a worker that hangs
// without ending) or an operator decision recorded under ## Needs operator.
// With neither, parking only idles the goal.
//
// `agent-manager effort park` applies this in the CLI. Wave 2 applies the
// same DecidePark server-side in orchestration.ParkRunFromAgent, next to the
// re-park guard, so a raw `run park` cannot bypass it.
const (
	ChildBackstop = time.Hour
	DecisionWait  = 72 * time.Hour
)

// Park rules.
const (
	ParkRuleLiveChild     = "live-child"
	ParkRuleNeedsOperator = "needs-operator"
	ParkRuleAdmissible    = "admissible-slice"
)

// ParkDecision is the policy's answer for one park request.
type ParkDecision struct {
	Allowed bool          `json:"allowed"`
	Rule    string        `json:"rule"`
	Max     time.Duration `json:"-"`
	Timeout time.Duration `json:"-"`
	Reason  string        `json:"reason"`
}

// DecidePark picks the longest allowed park and checks requested against it;
// requested 0 means the maximum. A request over the maximum is refused rather
// than shortened, so the caller sees the rule.
func DecidePark(liveChildren []string, needsOperator []Item, requested time.Duration) ParkDecision {
	var decision ParkDecision
	switch {
	case len(liveChildren) > 0:
		decision = ParkDecision{Rule: ParkRuleLiveChild, Max: ChildBackstop, Reason: fmt.Sprintf("live direct child %s; the timer is only a backstop for a worker that hangs", strings.Join(liveChildren, ", "))}
	case len(needsOperator) > 0:
		decision = ParkDecision{Rule: ParkRuleNeedsOperator, Max: DecisionWait, Reason: fmt.Sprintf("%d item(s) under ## Needs operator, first at QUEUE.md line %d; feedback or an answer wakes the run sooner", len(needsOperator), needsOperator[0].Line)}
	default:
		return ParkDecision{Rule: ParkRuleAdmissible, Reason: "no live direct child and nothing under ## Needs operator. Admissible-slice rule: admit the next [ready] slice (or plan one), or record a decision with options under ## Needs operator, then park"}
	}
	decision.Timeout = requested
	if requested == 0 {
		decision.Timeout = decision.Max
	}
	if decision.Timeout > decision.Max {
		decision.Reason = fmt.Sprintf("requested %s exceeds the %s maximum (%s): %s", requested, decision.Max, decision.Rule, decision.Reason)
		return decision
	}
	decision.Allowed = true
	return decision
}

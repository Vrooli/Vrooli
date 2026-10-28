package goalhome

import (
	"fmt"
	"strings"
	"time"
)

// Park policy for an orchestrator (DL-1, decision P-06). A park waits on
// something: a live direct child (a short backstop for a worker that hangs
// without ending) or an operator decision recorded under ## Needs operator
// when no independent [ready] slice remains. With neither, parking only idles
// the goal.
//
// The CLI and native agent park entry point use DecideHomePark, which first
// requires disposition of new steering. Legacy duration/admissible-work rules
// below retain the server's report/enforce setting.
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

// DecideHomePark applies the same feedback obligation as Lint before a leader
// can wait again. An unchanged signal cursor does not negate changed home state.
// This does not stop the worker, resolve feedback or require product acceptance.
func DecideHomePark(liveChildren []string, home *Home, requested time.Duration) ParkDecision {
	if decision := DecideFeedbackPark(home); !decision.Allowed {
		return decision
	}
	return DecidePark(liveChildren, home.Queue, requested)
}

// DecideFeedbackPark checks steering for every producer, without applying the
// child-specific duration and admissible-work rules.
func DecideFeedbackPark(home *Home) ParkDecision {
	if pending := unacknowledgedFeedback(home); len(pending) > 0 {
		ids := make([]string, 0, len(pending))
		for _, entry := range pending {
			ids = append(ids, entry.ID)
		}
		return ParkDecision{Rule: "feedback-unacknowledged", Reason: "act on open steering and record its ownership/disposition in Handoff before parking: " + strings.Join(ids, ", ")}
	}
	return ParkDecision{Allowed: true}
}

// DecidePark picks the longest allowed park and checks requested against it;
// requested 0 means the maximum. A request over the maximum is refused rather
// than shortened, so the caller sees the rule.
func DecidePark(liveChildren []string, queue Queue, requested time.Duration) ParkDecision {
	var decision ParkDecision
	ready := queue.ReadyNext()
	switch {
	case len(liveChildren) > 0:
		decision = ParkDecision{Rule: ParkRuleLiveChild, Max: ChildBackstop, Reason: fmt.Sprintf("live direct child %s; the timer is only a backstop for a worker that hangs", strings.Join(liveChildren, ", "))}
	case len(ready) > 0:
		return ParkDecision{Rule: ParkRuleAdmissible, Reason: fmt.Sprintf("Admissible-slice rule: %d [ready] slice(s) remain under ## Next; continue independent work before parking on ## Needs operator", len(ready))}
	case len(queue.NeedsOperator) > 0:
		decision = ParkDecision{Rule: ParkRuleNeedsOperator, Max: DecisionWait, Reason: fmt.Sprintf("%d item(s) under ## Needs operator, first at line %d; an answered decision or an explicit wake can resume the run sooner", len(queue.NeedsOperator), queue.NeedsOperator[0].Line)}
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

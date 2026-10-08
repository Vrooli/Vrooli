package capstatus

import (
	"fmt"
	"slices"
	"sort"
)

// Status is a capability's derived provability. Never authored.
type Status string

const (
	// StatusProvable means every axis, evidence kind, and claim type this
	// capability needs is available right now.
	StatusProvable Status = "provable"
	// StatusAxisUnavailable means an axis exists in the engine but the
	// reconciler does not drive it — plumbing, not engineering.
	StatusAxisUnavailable Status = "axis-unavailable"
	// StatusAxisMissing means no axis value is capturable by any means yet.
	StatusAxisMissing Status = "axis-missing"
	// StatusEvidenceMissing means the axes are fine and the observation is not
	// obtainable. computed-style is the dominant case.
	StatusEvidenceMissing Status = "evidence-missing"
	// StatusNoChecker means axes and evidence are fine and no evaluator exists
	// for at least one referenced claim type.
	StatusNoChecker Status = "no-checker"
	// StatusPortOnly marks capabilities with no promises facet. They are
	// satisfied structurally by an asset or a template, not proven by capture.
	StatusPortOnly Status = "port-only"
)

// Support is what the reconciler can currently do, supplied by the caller so
// this package stays free of a dependency on the reconciler's internals.
type Support struct {
	// Axes maps axis id to the values that can actually be captured.
	Axes map[string][]string
	// Evidence is the set of obtainable evidence kinds.
	Evidence []string
	// ClaimTypes is the set of claim types with a deterministic evaluator.
	ClaimTypes []string
}

// Result is one capability's derived state plus the specific reasons it is not
// provable. Blockers are the actionable half: "hover-contrast is blocked" is a
// complaint, "hover-contrast needs axis interaction-state and evidence
// computed-style" is a work item.
type Result struct {
	Capability string
	Title      string
	Group      string
	Status     Status
	Blockers   []string
}

// Report is the derived view over the whole registry.
type Report struct {
	Results []Result
	Totals  map[Status]int
	// PromiseTotal is the number of capabilities carrying the promises facet;
	// it is the denominator that matters, since port-only entries are never
	// proven by capture.
	PromiseTotal  int
	ProvableTotal int
}

// Derive computes status for every capability in the registry.
//
// Precedence when a capability is blocked several ways is deliberate: axis
// problems outrank evidence problems, which outrank missing checkers. An axis
// that cannot be captured makes the evidence question moot, and reporting the
// deepest blocker first keeps the work queue in dependency order.
func Derive(reg Registry, sup Support) Report {
	rep := Report{Totals: map[Status]int{}}
	evidence := map[string]struct{}{}
	for _, e := range sup.Evidence {
		evidence[e] = struct{}{}
	}
	claims := map[string]struct{}{}
	for _, c := range sup.ClaimTypes {
		claims[c] = struct{}{}
	}

	for _, cap := range reg.Capabilities {
		res := Result{Capability: cap.ID, Title: cap.Title, Group: cap.Group}
		if cap.Proves == nil {
			res.Status = StatusPortOnly
			rep.Results = append(rep.Results, res)
			rep.Totals[res.Status]++
			continue
		}
		rep.PromiseTotal++

		var axisMissing, axisUnavailable, evidenceMissing, noChecker []string

		axisIDs := make([]string, 0, len(cap.Proves.Axes))
		for id := range cap.Proves.Axes {
			axisIDs = append(axisIDs, id)
		}
		sort.Strings(axisIDs)
		for _, axisID := range axisIDs {
			want := cap.Proves.Axes[axisID]
			have, wired := sup.Axes[axisID]
			if !wired {
				// Declared in the registry but not driven at all. Whether the
				// engine could drive it is a separate question the axis
				// registry's mechanism block answers; from the reconciler's
				// point of view it is simply unavailable.
				axisUnavailable = append(axisUnavailable,
					fmt.Sprintf("axis %s is not driven by the reconciler", axisID))
				continue
			}
			if _, known := reg.Axes[axisID]; !known {
				axisMissing = append(axisMissing,
					fmt.Sprintf("axis %s is not declared in the registry", axisID))
				continue
			}
			for _, value := range want {
				if !slices.Contains(have, value) {
					axisUnavailable = append(axisUnavailable,
						fmt.Sprintf("axis %s value %q is not captured", axisID, value))
				}
			}
		}
		for _, want := range sortedCopy(cap.Proves.Evidence) {
			if _, ok := evidence[want]; !ok {
				evidenceMissing = append(evidenceMissing,
					fmt.Sprintf("evidence %s is not produced", want))
			}
		}
		for _, want := range sortedCopy(cap.Proves.ClaimTypes) {
			if _, ok := claims[want]; !ok {
				noChecker = append(noChecker,
					fmt.Sprintf("claim type %s has no evaluator", want))
			}
		}

		switch {
		case len(axisMissing) > 0:
			res.Status = StatusAxisMissing
		case len(axisUnavailable) > 0:
			res.Status = StatusAxisUnavailable
		case len(evidenceMissing) > 0:
			res.Status = StatusEvidenceMissing
		case len(noChecker) > 0:
			res.Status = StatusNoChecker
		default:
			res.Status = StatusProvable
			rep.ProvableTotal++
		}
		// Every blocker is reported, not only the one that set the status, so a
		// capability three problems deep is not fixed once and found broken again.
		res.Blockers = append(res.Blockers, axisMissing...)
		res.Blockers = append(res.Blockers, axisUnavailable...)
		res.Blockers = append(res.Blockers, evidenceMissing...)
		res.Blockers = append(res.Blockers, noChecker...)

		rep.Results = append(rep.Results, res)
		rep.Totals[res.Status]++
	}
	return rep
}

// BlockerCounts aggregates how many capabilities each individual blocker holds
// back. This is the work-ordering signal: the blocker at the top of this list
// unlocks the most capabilities for one piece of work.
func BlockerCounts(rep Report) []BlockerCount {
	counts := map[string]int{}
	for _, res := range rep.Results {
		for _, b := range res.Blockers {
			counts[b]++
		}
	}
	out := make([]BlockerCount, 0, len(counts))
	for blocker, n := range counts {
		out = append(out, BlockerCount{Blocker: blocker, Capabilities: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Capabilities != out[j].Capabilities {
			return out[i].Capabilities > out[j].Capabilities
		}
		return out[i].Blocker < out[j].Blocker
	})
	return out
}

// BlockerCount is one blocker and how many capabilities it holds back.
type BlockerCount struct {
	Blocker      string
	Capabilities int
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

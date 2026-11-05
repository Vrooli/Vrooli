package verify

import (
	"fmt"
	"sort"
	"strings"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation/compile"
	"flow-verifier/internal/flows/kinds/navigation/contract"
	"flow-verifier/internal/flows/kinds/navigation/predicate"
)

// Trace is the kind-specific payload attached to a Finding for a
// reachability-invariant failure.
type Trace struct {
	InvariantID string
	World       World
	Target      string
	// Path is populated when must_reach failed because the target was
	// reached but over budget; Steps holds the shortest path found.
	Path *Path
	// MustNotReachPath is populated when must_not_reach failed because
	// the target IS reachable in this world.
	MustNotReachPath *Path
	// Budget is the click budget that applied to this world (0 means
	// "no budget enforced").
	Budget int
}

// CheckInvariants runs every reachability invariant against the graph
// and returns one Finding per invariant. Findings are sorted by id.
func CheckInvariants(g compile.Graph) ([]kind.Finding, error) {
	worlds, err := EnumerateWorlds(g.Contract.Contexts)
	if err != nil {
		return nil, err
	}
	if len(worlds) == 0 {
		return nil, fmt.Errorf("no valid context worlds; check valid_when predicates")
	}
	findings := make([]kind.Finding, 0, len(g.Contract.ReachabilityInvariants))
	for _, inv := range g.Contract.ReachabilityInvariants {
		f, err := checkOne(g, inv, worlds)
		if err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })
	return findings, nil
}

func checkOne(g compile.Graph, inv contract.ReachabilityInvariant, worlds []World) (kind.Finding, error) {
	givenPred, err := predicate.Parse(inv.Given)
	if err != nil {
		return kind.Finding{}, fmt.Errorf("invariant %q given: %w", inv.ID, err)
	}
	matching, err := WorldsMatching(worlds, inv.Given)
	if err != nil {
		return kind.Finding{}, fmt.Errorf("invariant %q: %w", inv.ID, err)
	}
	if len(matching) == 0 {
		return kind.Finding{
			ID:      inv.ID,
			Passed:  false,
			Message: fmt.Sprintf("invariant %q: no worlds satisfy given %q", inv.ID, inv.Given),
		}, nil
	}

	var failures []string
	var firstFailTrace *Trace

	for _, w := range matching {
		paths, err := ShortestPaths(g, inv.From, w, givenPred)
		if err != nil {
			return kind.Finding{}, fmt.Errorf("invariant %q world %s: %w", inv.ID, w.Key(), err)
		}
		for _, target := range inv.MustReach {
			p, reachable := paths[target]
			if !reachable {
				failures = append(failures, fmt.Sprintf("must_reach %q not reachable in world %s", target, w.Key()))
				if firstFailTrace == nil {
					firstFailTrace = &Trace{InvariantID: inv.ID, World: w, Target: target}
				}
				continue
			}
			budget := budgetFor(inv.MaxClicks, w)
			if budget > 0 && p.Clicks > budget {
				failures = append(failures, fmt.Sprintf("must_reach %q took %d clicks (>budget %d) in world %s", target, p.Clicks, budget, w.Key()))
				if firstFailTrace == nil {
					pp := p
					firstFailTrace = &Trace{InvariantID: inv.ID, World: w, Target: target, Path: &pp, Budget: budget}
				}
			}
		}
		for _, target := range inv.MustNotReach {
			if p, reachable := paths[target]; reachable {
				failures = append(failures, fmt.Sprintf("must_not_reach %q IS reachable in world %s (%d clicks)", target, w.Key(), p.Clicks))
				if firstFailTrace == nil {
					pp := p
					firstFailTrace = &Trace{InvariantID: inv.ID, World: w, Target: target, MustNotReachPath: &pp}
				}
			}
		}
	}

	if len(failures) == 0 {
		return kind.Finding{
			ID:      inv.ID,
			Passed:  true,
			Message: fmt.Sprintf("invariant %q: %d worlds OK", inv.ID, len(matching)),
		}, nil
	}
	sort.Strings(failures)
	return kind.Finding{
		ID:       inv.ID,
		Passed:   false,
		Severity: "error",
		Message:  fmt.Sprintf("invariant %q failed:\n  - %s", inv.ID, strings.Join(failures, "\n  - ")),
		Trace:    firstFailTrace,
	}, nil
}

// budgetFor resolves the click budget for one world. Returns 0 when no
// budget was declared (no enforcement).
func budgetFor(mc *contract.MaxClicks, w World) int {
	if mc == nil || !mc.IsSet() {
		return 0
	}
	if mc.ByViewport != nil {
		vp := w["viewport"]
		if vp == "" {
			// Viewport not declared as a context — treat per-viewport
			// budget as the max across declared viewports so we don't
			// false-fail.
			max := 0
			for _, v := range mc.ByViewport {
				if v > max {
					max = v
				}
			}
			return max
		}
		if v, ok := mc.ByViewport[vp]; ok {
			return v
		}
		// Viewport value not listed in the per-viewport budget — leave
		// unenforced (interpret as "any" for this viewport).
		return 0
	}
	return mc.Scalar
}

package findings

import "github.com/vrooli/maturity-go/dimensions"

// Diff is the per-dimension change in the open-findings set across one
// controller iteration, computed by stable finding ID. It is the signal credit
// assignment attributes to the skill that ran (see pkg/effectiveness.CreditEvent).
type Diff struct {
	// ClosedByDimension counts findings present before and absent after, bucketed
	// by the dimension they belonged to.
	ClosedByDimension map[dimensions.Dimension]int
	// IntroducedByDimension counts findings absent before and present after,
	// bucketed by the dimension they now belong to.
	IntroducedByDimension map[dimensions.Dimension]int
}

// NetClosed returns the total findings closed minus introduced across all
// dimensions (the iteration's net findings flow; can be negative on a regression).
func (d Diff) NetClosed() int {
	net := 0
	for _, n := range d.ClosedByDimension {
		net += n
	}
	for _, n := range d.IntroducedByDimension {
		net -= n
	}
	return net
}

// DiffStates computes the per-dimension findings diff between two states by
// stable finding ID. A finding in both states is unchanged regardless of audit
// ordering; this is what lets a targeted re-audit (which carries stale findings
// forward unchanged) register zero churn outside the re-audited dimensions.
func DiffStates(before, after FindingsState) Diff {
	beforeByID := indexByID(before.Findings)
	afterByID := indexByID(after.Findings)

	d := Diff{
		ClosedByDimension:     map[dimensions.Dimension]int{},
		IntroducedByDimension: map[dimensions.Dimension]int{},
	}
	for id, f := range beforeByID {
		if _, still := afterByID[id]; !still {
			d.ClosedByDimension[f.Dimension]++
		}
	}
	for id, f := range afterByID {
		if _, had := beforeByID[id]; !had {
			d.IntroducedByDimension[f.Dimension]++
		}
	}
	return d
}

func indexByID(fs []Finding) map[string]Finding {
	out := make(map[string]Finding, len(fs))
	for _, f := range fs {
		out[f.ID] = f
	}
	return out
}

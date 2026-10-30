package usageattribution

import "sort"

// minPlanSaving is the combined saving, in points, below which a plan is
// folded into a count: it would display as 0.0.
const minPlanSaving = 0.1

// floorPlanMinShare is the share of a target's points the starting-context
// floor must make up for an R-floor-only plan to be worth showing, and
// floorPlanMinPoints the least the target must have used.
const (
	floorPlanMinShare  = 0.2
	floorPlanMinPoints = 1.0
)

// Plan is every lever for one target in one window, with a combined saving
// that does not count overlapping levers twice.
type Plan struct {
	Rank        int
	TargetKind  string
	TargetID    string
	TargetLabel string
	Provider    string
	Pool        string
	Window      string
	// Levers hold the rules that fired for the target, largest saving first;
	// each lever's Marginal is what it adds after the larger ones.
	Levers []Recommendation
	// Saved is the combined saving over the window's whole horizon at this
	// rate (a week for weekly windows); Recoverable is the part still to be had
	// before the window resets.
	Saved, Recoverable float64
	// CrossCut marks a plan whose target overlaps the others: a friction
	// problem's points are already inside the session plans that ran it, so
	// savings and shares must never be summed across target kinds.
	CrossCut bool
	// Use is the target's consumption so far and projected, shared by its levers.
	Use Use
}

// planKey identifies a target within a window.
func planKey(r Recommendation) [4]string {
	return [4]string{r.Provider + "/" + r.Pool, r.Window, r.TargetKind, r.TargetID}
}

// buildPlans folds levers by target and ranks the groups. Levers of one
// target overlap (each reduces the same use), so the largest is applied to the
// target's projected use first and each next lever to what remains:
// combined = projected × (1 − Π(1 − saving/projected)). Plans saving less
// than minPlanSaving are counted, not listed.
func buildPlans(levers []Recommendation) (plans, folded []Plan) {
	byKey := map[[4]string]*Plan{}
	var order [][4]string
	for _, lever := range levers {
		key := planKey(lever)
		g := byKey[key]
		if g == nil {
			g = &Plan{TargetKind: lever.TargetKind, TargetID: lever.TargetID, TargetLabel: lever.TargetLabel, Provider: lever.Provider, Pool: lever.Pool, Window: lever.Window}
			byKey[key] = g
			order = append(order, key)
		}
		g.Levers = append(g.Levers, lever)
	}
	for _, key := range order {
		g := byKey[key]
		sort.SliceStable(g.Levers, func(i, j int) bool { return g.Levers[i].Saved > g.Levers[j].Saved })
		g.Use = g.Levers[0].Use
		g.Saved = applyLevers(g.Levers, g.Use.AtThisRate)
		g.Recoverable = recoverable(g.Saved, g.Use)
		g.CrossCut = g.TargetKind == "friction"
		if g.Saved < minPlanSaving && !informative(g.Levers) {
			folded = append(folded, *g)
			continue
		}
		plans = append(plans, *g)
	}
	sort.SliceStable(plans, func(i, j int) bool { return plans[i].Saved > plans[j].Saved })
	for i := range plans {
		plans[i].Rank = i + 1
	}
	return plans, folded
}

// applyLevers sets each lever's Marginal and returns the combined saving.
func applyLevers(levers []Recommendation, projected float64) float64 {
	if projected <= 0 {
		// Nothing to compound against: only the largest lever counts.
		for i := range levers {
			levers[i].Marginal = 0
		}
		levers[0].Marginal = levers[0].Saved
		return levers[0].Saved
	}
	remaining := projected
	for i := range levers {
		fraction := min(1, levers[i].Saved/projected)
		levers[i].Marginal = remaining * fraction
		remaining -= levers[i].Marginal
	}
	return projected - remaining
}

// Top returns the largest plan stated in the window.
func (r Report) Top(key WindowKey) (Plan, bool) {
	for _, g := range r.Plans {
		if g.Provider == key.Provider && g.Pool == key.Pool && g.Window == key.Window {
			return g, true
		}
	}
	return Plan{}, false
}

// informative is a plan worth showing although it claims no saving: an
// unproven model move whose upper bound is a real amount (its range, not a
// point estimate, is the finding), or a starting-context floor that is a
// material share of a target that used real points.
func informative(levers []Recommendation) bool {
	for _, lever := range levers {
		if lever.Rule == RuleModelMix && lever.Facts.Model != nil &&
			lever.Facts.Model.ExpensiveWeeklyPoints*(1-lever.Facts.Model.PriceRatio) >= minPlanSaving {
			return true
		}
		if lever.Rule == RuleFloor && lever.Facts.Context != nil &&
			lever.Facts.Context.CachedShare >= floorPlanMinShare && lever.Use.ConsumerPoints >= floorPlanMinPoints {
			return true
		}
	}
	return false
}

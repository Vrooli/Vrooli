package goals

import (
	"sort"

	"swarm-manager/internal/depgraph"
)

// StatusCompleted is the backlog status that counts an item as achieved for
// progress. StatusDropped resolves an item without achieving it. Kept as
// literals so the goals package does not import backlog.
const (
	StatusCompleted = "completed"
	StatusDropped   = "dropped"
)

// isResolved reports whether a status means nothing depending on the item is
// still waiting. Both a completed item and a dropped one satisfy dependents;
// only completed counts as progress.
func isResolved(status string) bool {
	return status == StatusCompleted || status == StatusDropped
}

// ScopeInput carries the pure inputs for item-root scope computation.
type ScopeInput struct {
	// Targets are the goal's raw item refs: "<kind>/<name>".
	Targets []string
	// ItemDeps maps every backlog item ref to its depends_on item refs.
	ItemDeps map[string][]string
	// ItemStatus maps every backlog item ref to its backlog status. Membership
	// in this map is the definition of "a known item".
	ItemStatus map[string]string
	// ItemEffort maps every backlog item ref to its effort class (XS..XL, or ""
	// when unsized). Read by the ETA rollup; absent from a ref means unsized.
	ItemEffort map[string]string
	// Milestones partitions the computed scope for read-time rollups only.
	Milestones []Milestone
}

// Scope is the computed goal scope.
type Scope struct {
	Targets        []string         `json:"targets"`
	Closure        []string         `json:"closure"`
	Completed      []string         `json:"completed"`
	Dropped        []string         `json:"dropped,omitempty"`
	Ready          []string         `json:"ready"`
	Blocked        []string         `json:"blocked"`
	Total          int              `json:"total"`
	CompletedCount int              `json:"completed_count"`
	DroppedCount   int              `json:"dropped_count,omitempty"`
	BlockedCount   int              `json:"blocked_count"`
	ProgressPct    float64          `json:"progress_pct"`
	Milestones     []MilestoneScope `json:"milestones,omitempty"`
	Unassigned     []string         `json:"unassigned,omitempty"`
}

// MilestoneScope is the live status rollup for one goal-owned milestone.
type MilestoneScope struct {
	Milestone Milestone `json:"milestone"`
	Items     []string  `json:"items"`
	// Orphaned lists members the milestone claims that are outside the goal's
	// derived closure. They are counted in no rollup, so without this field the
	// discrepancy is invisible. A non-empty list means either the item should be
	// a goal target or the membership is stale.
	Orphaned       []string `json:"orphaned,omitempty"`
	CompletedCount int      `json:"completed_count"`
	DroppedCount   int      `json:"dropped_count,omitempty"`
	ReadyCount     int      `json:"ready_count"`
	BlockedCount   int      `json:"blocked_count"`
}

// ComputeScope resolves a goal's transitive prerequisite closure, progress, and
// readiness. It is pure and cycle-safe.
func ComputeScope(in ScopeInput) Scope {
	roots := append([]string(nil), in.Targets...)

	// Build the item dependency graph over all known items and walk the
	// transitive closure from the expanded roots.
	itemGraph := depgraph.New()
	for ref := range in.ItemStatus {
		itemGraph.AddNode(ref, in.ItemDeps[ref])
	}
	// Roots that are not known items (e.g. a target typo) still enter the
	// closure as leaves so the scope is not silently narrowed.
	for _, r := range roots {
		if _, ok := in.ItemStatus[r]; !ok {
			itemGraph.AddNode(r, in.ItemDeps[r])
		}
	}
	rawClosure := itemGraph.TransitiveClosure(roots)

	// Restrict the reported closure to known backlog items.
	closure := make([]string, 0, len(rawClosure))
	for _, ref := range rawClosure {
		if _, ok := in.ItemStatus[ref]; ok {
			closure = append(closure, ref)
		}
	}
	sort.Strings(closure)

	// Build the readiness graph over items. The gate is keyed on *resolved*,
	// not *completed*: a dropped prerequisite is never coming back, so holding
	// its dependents in `blocked` would strand them permanently.
	graphMap := make(map[string][]string, len(in.ItemStatus))
	resolved := make(map[string]bool, len(in.ItemStatus))
	for ref, status := range in.ItemStatus {
		graphMap[ref] = in.ItemDeps[ref]
		resolved[ref] = isResolved(status)
	}

	gateGraph := depgraph.New()
	for key, deps := range graphMap {
		gateGraph.AddNode(key, deps)
	}
	unblocked := make(map[string]bool)
	for _, ref := range gateGraph.UnblockedItems(resolved) {
		unblocked[ref] = true
	}
	// A milestone dependency is a goal-owned gate: an item assigned to a
	// milestone cannot become ready until every predecessor milestone is
	// complete. This keeps milestone sequencing in the item graph without
	// inventing a second top-level work entity.
	applyMilestoneGate(unblocked, resolved, in.Milestones)

	scope := Scope{Targets: append([]string(nil), in.Targets...), Closure: closure}
	for _, ref := range closure {
		switch {
		case in.ItemStatus[ref] == StatusCompleted:
			scope.Completed = append(scope.Completed, ref)
		case in.ItemStatus[ref] == StatusDropped:
			scope.Dropped = append(scope.Dropped, ref)
		case unblocked[ref]:
			scope.Ready = append(scope.Ready, ref)
		default:
			scope.Blocked = append(scope.Blocked, ref)
		}
	}
	scope.Total = len(closure)
	scope.CompletedCount = len(scope.Completed)
	scope.DroppedCount = len(scope.Dropped)
	scope.BlockedCount = len(scope.Blocked)
	// Progress is measured against the work still considered in scope. Dropped
	// items leave the denominator entirely: counting them as incomplete would
	// hold a goal below 100% forever, and counting them as complete would claim
	// abandoned work as an achievement.
	if inScope := scope.Total - scope.DroppedCount; inScope > 0 {
		scope.ProgressPct = float64(scope.CompletedCount) / float64(inScope) * 100.0
	}
	populateMilestonePartition(&scope, in.Milestones)
	return scope
}

// applyMilestoneGate holds back items whose predecessor milestones are not yet
// settled. `resolved` marks items nothing is still waiting on — a milestone
// whose remaining items were all dropped is settled and must stop gating its
// successors, exactly as if they had been completed.
func applyMilestoneGate(unblocked, resolved map[string]bool, milestones []Milestone) {
	byName := make(map[string]Milestone, len(milestones))
	settledMilestones := make(map[string]bool, len(milestones))
	for _, milestone := range milestones {
		if milestone.ArchivedAt != nil {
			continue
		}
		byName[milestone.Name] = milestone
		done := len(milestone.Items) > 0
		for _, ref := range milestone.Items {
			if !resolved[ref] {
				done = false
				break
			}
		}
		settledMilestones[milestone.Name] = done
	}
	for _, milestone := range byName {
		for _, dependency := range milestone.DependsOn {
			if settledMilestones[dependency] {
				continue
			}
			for _, ref := range milestone.Items {
				delete(unblocked, ref)
			}
			break
		}
	}
}

func populateMilestonePartition(scope *Scope, milestones []Milestone) {
	if len(milestones) == 0 {
		return
	}
	inScope := make(map[string]bool, len(scope.Closure))
	completed := make(map[string]bool, len(scope.Completed))
	dropped := make(map[string]bool, len(scope.Dropped))
	ready := make(map[string]bool, len(scope.Ready))
	blocked := make(map[string]bool, len(scope.Blocked))
	assigned := make(map[string]bool, len(scope.Closure))
	for _, ref := range scope.Closure {
		inScope[ref] = true
	}
	for _, ref := range scope.Completed {
		completed[ref] = true
	}
	for _, ref := range scope.Dropped {
		dropped[ref] = true
	}
	for _, ref := range scope.Ready {
		ready[ref] = true
	}
	for _, ref := range scope.Blocked {
		blocked[ref] = true
	}
	for _, milestone := range milestones {
		rollup := MilestoneScope{Milestone: milestone, Items: []string{}}
		for _, ref := range milestone.Items {
			if !inScope[ref] {
				// The milestone claims an item the goal's closure does not
				// contain. Skipping it silently is how membership rots
				// unnoticed: the rollup still looks plausible while real work
				// goes uncounted. Report it so the discrepancy is visible.
				rollup.Orphaned = append(rollup.Orphaned, ref)
				continue
			}
			if assigned[ref] {
				continue
			}
			assigned[ref] = true
			rollup.Items = append(rollup.Items, ref)
			if completed[ref] {
				rollup.CompletedCount++
			}
			if dropped[ref] {
				rollup.DroppedCount++
			}
			if ready[ref] {
				rollup.ReadyCount++
			}
			if blocked[ref] {
				rollup.BlockedCount++
			}
		}
		sort.Strings(rollup.Items)
		scope.Milestones = append(scope.Milestones, rollup)
	}
	for _, ref := range scope.Closure {
		if !assigned[ref] {
			scope.Unassigned = append(scope.Unassigned, ref)
		}
	}
}

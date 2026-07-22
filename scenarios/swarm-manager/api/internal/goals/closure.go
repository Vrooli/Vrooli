package goals

import (
	"sort"

	"swarm-manager/internal/depgraph"
)

// StatusCompleted is the backlog status that counts an item as done for
// progress. Kept as a literal so the goals package does not import backlog.
const StatusCompleted = "completed"

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
	Ready          []string         `json:"ready"`
	Blocked        []string         `json:"blocked"`
	Total          int              `json:"total"`
	CompletedCount int              `json:"completed_count"`
	BlockedCount   int              `json:"blocked_count"`
	ProgressPct    float64          `json:"progress_pct"`
	Milestones     []MilestoneScope `json:"milestones,omitempty"`
	Unassigned     []string         `json:"unassigned,omitempty"`
}

// MilestoneScope is the live status rollup for one goal-owned milestone.
type MilestoneScope struct {
	Milestone      Milestone `json:"milestone"`
	Items          []string  `json:"items"`
	CompletedCount int       `json:"completed_count"`
	ReadyCount     int       `json:"ready_count"`
	BlockedCount   int       `json:"blocked_count"`
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

	// Build the readiness graph over items.
	graphMap := make(map[string][]string, len(in.ItemStatus))
	completed := make(map[string]bool, len(in.ItemStatus))
	for ref, status := range in.ItemStatus {
		graphMap[ref] = in.ItemDeps[ref]
		completed[ref] = status == StatusCompleted
	}

	gateGraph := depgraph.New()
	for key, deps := range graphMap {
		gateGraph.AddNode(key, deps)
	}
	unblocked := make(map[string]bool)
	for _, ref := range gateGraph.UnblockedItems(completed) {
		unblocked[ref] = true
	}
	// A milestone dependency is a goal-owned gate: an item assigned to a
	// milestone cannot become ready until every predecessor milestone is
	// complete. This keeps milestone sequencing in the item graph without
	// inventing a second top-level work entity.
	applyMilestoneGate(unblocked, completed, in.Milestones)

	scope := Scope{Targets: append([]string(nil), in.Targets...), Closure: closure}
	for _, ref := range closure {
		switch {
		case in.ItemStatus[ref] == StatusCompleted:
			scope.Completed = append(scope.Completed, ref)
		case unblocked[ref]:
			scope.Ready = append(scope.Ready, ref)
		default:
			scope.Blocked = append(scope.Blocked, ref)
		}
	}
	scope.Total = len(closure)
	scope.CompletedCount = len(scope.Completed)
	scope.BlockedCount = len(scope.Blocked)
	if scope.Total > 0 {
		scope.ProgressPct = float64(scope.CompletedCount) / float64(scope.Total) * 100.0
	}
	populateMilestonePartition(&scope, in.Milestones)
	return scope
}

func applyMilestoneGate(unblocked, completed map[string]bool, milestones []Milestone) {
	byName := make(map[string]Milestone, len(milestones))
	completedMilestones := make(map[string]bool, len(milestones))
	for _, milestone := range milestones {
		if milestone.ArchivedAt != nil {
			continue
		}
		byName[milestone.Name] = milestone
		done := len(milestone.Items) > 0
		for _, ref := range milestone.Items {
			if !completed[ref] {
				done = false
				break
			}
		}
		completedMilestones[milestone.Name] = done
	}
	for _, milestone := range byName {
		for _, dependency := range milestone.DependsOn {
			if completedMilestones[dependency] {
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
	ready := make(map[string]bool, len(scope.Ready))
	blocked := make(map[string]bool, len(scope.Blocked))
	assigned := make(map[string]bool, len(scope.Closure))
	for _, ref := range scope.Closure {
		inScope[ref] = true
	}
	for _, ref := range scope.Completed {
		completed[ref] = true
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
			if !inScope[ref] || assigned[ref] {
				continue
			}
			assigned[ref] = true
			rollup.Items = append(rollup.Items, ref)
			if completed[ref] {
				rollup.CompletedCount++
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

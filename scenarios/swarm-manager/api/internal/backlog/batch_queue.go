// Batch helper functions: milestone resolution, rollback, and utility
// functions that support atomic batch operations in batch_handler.go.
package backlog

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/depgraph"
)

func isValidMilestoneStatus(status string) bool {
	switch status {
	case "active", "completed":
		return true
	default:
		return false
	}
}

// normalizeMilestoneDeps trims, dedupes, and sorts depends_on entries for an
// milestone. Returns errors for blanks, self-references, or kind/name-form
// strings (which belong on items, not milestones).
func normalizeMilestoneDeps(raw []string, self string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, d := range raw {
		d = strings.TrimSpace(d)
		if d == "" {
			return nil, fmt.Errorf("depends_on contains a blank entry")
		}
		if strings.Contains(d, "/") {
			return nil, fmt.Errorf("depends_on must be milestone names, not kind/name: %q", d)
		}
		if d == self {
			return nil, fmt.Errorf("depends_on cannot reference self %q", self)
		}
		if seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	sort.Strings(out)
	return out, nil
}

// validateMilestoneDepRefs ensures every depends_on entry points at either
// another milestone in the batch or an existing milestone on disk.
func (h *Handler) validateMilestoneDepRefs(provided map[string]batchCreateMilestone) error {
	if h.milestoneAssigner == nil {
		for _, init := range provided {
			if init.DependsOn != nil && len(*init.DependsOn) > 0 {
				return apierr.Internal("milestone support not configured")
			}
		}
		return nil
	}
	for name, init := range provided {
		if init.DependsOn == nil {
			continue
		}
		for _, dep := range *init.DependsOn {
			if _, inBatch := provided[dep]; inBatch {
				continue
			}
			existing, err := h.milestoneAssigner.Get(dep)
			if err != nil && !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("failed to load milestone %q: %w", dep, err)
			}
			if existing == nil {
				return fmt.Errorf("milestones[%s]: depends_on references unknown milestone %q", name, dep)
			}
		}
	}
	return nil
}

// orderedMilestonePlans returns plan names in an order that satisfies
// intra-batch depends_on (dep before dependent). Independent nodes are
// ordered alphabetically for determinism. On cycle, returns an error.
func orderedMilestonePlans(plans map[string]resolvedMilestonePlan) ([]string, error) {
	g := depgraph.New()
	for name, plan := range plans {
		deps := make([]string, 0, len(plan.spec.DependsOn))
		for _, d := range plan.spec.DependsOn {
			if _, ok := plans[d]; ok {
				deps = append(deps, d)
			}
		}
		g.AddNode(name, deps)
	}
	if cycle, found := g.DetectCycle(); found {
		return nil, fmt.Errorf("milestone dependency cycle detected: %s", strings.Join(cycle, " -> "))
	}
	return g.TopologicalSort()
}

func groupItemRefsByMilestone(items []BacklogItem) map[string][]string {
	grouped := make(map[string][]string)
	for _, item := range items {
		if strings.TrimSpace(item.Milestone) == "" {
			continue
		}
		ref := string(item.Kind) + "/" + item.Name
		grouped[item.Milestone] = append(grouped[item.Milestone], ref)
	}
	return grouped
}

func rollbackBatchCreate(createdDirs []string, appliedMilestones []resolvedMilestonePlan, assigner MilestoneAssigner) {
	for _, dir := range createdDirs {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			slog.Debug("backlog: rollback batch-created dir failed", "err", rmErr, "dir", dir)
		}
	}
	if assigner == nil {
		return
	}
	for i := len(appliedMilestones) - 1; i >= 0; i-- {
		plan := appliedMilestones[i]
		switch plan.action {
		case "create":
			_ = assigner.Delete(plan.spec.Name)
		default:
			if plan.existing != nil {
				_ = assigner.Replace(*plan.existing)
			}
		}
	}
}

func (h *Handler) resolveMilestonePlans(
	referenced map[string]bool,
	provided map[string]batchCreateMilestone,
) (map[string]resolvedMilestonePlan, []batchCreateMilestoneResult, error) {
	plans := make(map[string]resolvedMilestonePlan, len(referenced))
	results := make([]batchCreateMilestoneResult, 0, len(referenced))

	for name := range referenced {
		providedSpec, hasProvided := provided[name]
		existing, err := h.milestoneAssigner.Get(name)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return nil, nil, fmt.Errorf("failed to load milestone %q: %w", name, err)
		}

		if !hasProvided {
			if existing == nil {
				return nil, nil, fmt.Errorf("milestone %q does not exist; include it in the milestones list with metadata", name)
			}
			deps := append([]string(nil), existing.DependsOn...)
			plans[name] = resolvedMilestonePlan{
				spec: MilestoneSpec{
					Name:        existing.Name,
					Title:       existing.Title,
					Description: existing.Description,
					Status:      existing.Status,
					Priority:    existing.Priority,
					DependsOn:   deps,
					PlanRef:     existing.PlanRef,
				},
				existing: existing,
				action:   "reuse",
			}
			results = append(results, batchCreateMilestoneResult{
				Name:        existing.Name,
				Title:       existing.Title,
				Description: existing.Description,
				Status:      existing.Status,
				Priority:    existing.Priority,
				DependsOn:   append([]string(nil), existing.DependsOn...),
				Action:      "reuse",
			})
			continue
		}

		description := ""
		if providedSpec.Description != nil {
			description = *providedSpec.Description
		} else if existing != nil {
			description = existing.Description
		}
		status := "active"
		switch {
		case providedSpec.Status != nil:
			status = *providedSpec.Status
		case existing != nil:
			status = existing.Status
		}
		priority := 0
		switch {
		case providedSpec.Priority != nil:
			priority = *providedSpec.Priority
		case existing != nil:
			priority = existing.Priority
		}
		var deps []string
		switch {
		case providedSpec.DependsOn != nil:
			deps = append([]string(nil), (*providedSpec.DependsOn)...)
		case existing != nil:
			deps = append([]string(nil), existing.DependsOn...)
		}
		planRef := normalizePlanRef(providedSpec.PlanRef)
		if planRef == nil && existing != nil {
			planRef = existing.PlanRef
		}
		spec := MilestoneSpec{
			Name:        name,
			Title:       providedSpec.Title,
			Description: description,
			Status:      status,
			Priority:    priority,
			DependsOn:   deps,
			PlanRef:     planRef,
		}
		action := "create"
		if existing != nil {
			action = "reuse"
			if existing.Title != spec.Title ||
				existing.Description != spec.Description ||
				existing.Status != spec.Status ||
				existing.Priority != spec.Priority ||
				!stringSetEqual(existing.DependsOn, spec.DependsOn) ||
				!planRefsEqual(existing.PlanRef, spec.PlanRef) {
				action = "update"
			}
		}

		plans[name] = resolvedMilestonePlan{
			spec:     spec,
			existing: existing,
			action:   action,
		}
		results = append(results, batchCreateMilestoneResult{
			Name:        spec.Name,
			Title:       spec.Title,
			Description: spec.Description,
			Status:      spec.Status,
			Priority:    spec.Priority,
			DependsOn:   append([]string(nil), spec.DependsOn...),
			Action:      action,
		})
	}

	return plans, results, nil
}

// stringSetEqual returns true if a and b contain the same elements (set
// equality). Handles the common mixed-order case where existing deps come
// back from the store in insertion order and provided deps come in sorted
// by normalizeMilestoneDeps.
func stringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa := append([]string(nil), a...)
	sb := append([]string(nil), b...)
	sort.Strings(sa)
	sort.Strings(sb)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

func planRefsEqual(a, b *PlanRef) bool {
	a = normalizePlanRef(a)
	b = normalizePlanRef(b)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Provider == b.Provider &&
		a.PlanID == b.PlanID &&
		a.Slug == b.Slug &&
		a.Role == b.Role
}

// Batch helper functions: initiative resolution, rollback, and utility
// functions that support atomic batch operations in batch_handler.go.
package backlog

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func isValidInitiativeStatus(status string) bool {
	switch status {
	case "active", "completed":
		return true
	default:
		return false
	}
}

func orderedInitiativeNames(plans map[string]resolvedInitiativePlan) []string {
	names := make([]string, 0, len(plans))
	for name := range plans {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func groupItemRefsByInitiative(items []BacklogItem) map[string][]string {
	grouped := make(map[string][]string)
	for _, item := range items {
		if strings.TrimSpace(item.Initiative) == "" {
			continue
		}
		ref := string(item.Kind) + "/" + item.Name
		grouped[item.Initiative] = append(grouped[item.Initiative], ref)
	}
	return grouped
}

func rollbackBatchCreate(createdDirs []string, appliedInitiatives []resolvedInitiativePlan, assigner InitiativeAssigner) {
	for _, dir := range createdDirs {
		_ = os.RemoveAll(dir)
	}
	if assigner == nil {
		return
	}
	for i := len(appliedInitiatives) - 1; i >= 0; i-- {
		plan := appliedInitiatives[i]
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

func (h *Handler) resolveInitiativePlans(
	referenced map[string]bool,
	provided map[string]batchCreateInitiative,
) (map[string]resolvedInitiativePlan, []batchCreateInitiativeResult, error) {
	plans := make(map[string]resolvedInitiativePlan, len(referenced))
	results := make([]batchCreateInitiativeResult, 0, len(referenced))

	for name := range referenced {
		providedSpec, hasProvided := provided[name]
		existing, err := h.initiativeAssigner.Get(name)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return nil, nil, fmt.Errorf("failed to load initiative %q: %w", name, err)
		}

		if !hasProvided {
			if existing == nil {
				return nil, nil, fmt.Errorf("initiative %q does not exist; include it in the initiatives list with metadata", name)
			}
			plans[name] = resolvedInitiativePlan{
				spec: InitiativeSpec{
					Name:        existing.Name,
					Title:       existing.Title,
					Description: existing.Description,
					Status:      existing.Status,
				},
				existing: existing,
				action:   "reuse",
			}
			results = append(results, batchCreateInitiativeResult{
				Name:        existing.Name,
				Title:       existing.Title,
				Description: existing.Description,
				Status:      existing.Status,
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
		spec := InitiativeSpec{
			Name:        name,
			Title:       providedSpec.Title,
			Description: description,
			Status:      status,
		}
		action := "create"
		if existing != nil {
			action = "reuse"
			if existing.Title != spec.Title || existing.Description != spec.Description || existing.Status != spec.Status {
				action = "update"
			}
		}

		plans[name] = resolvedInitiativePlan{
			spec: InitiativeSpec{
				Name:        spec.Name,
				Title:       spec.Title,
				Description: spec.Description,
				Status:      spec.Status,
			},
			existing: existing,
			action:   action,
		}
		results = append(results, batchCreateInitiativeResult{
			Name:        spec.Name,
			Title:       spec.Title,
			Description: spec.Description,
			Status:      spec.Status,
			Action:      action,
		})
	}

	return plans, results, nil
}

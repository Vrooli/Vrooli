package main

import (
	"net/http"
)

// orderedResource represents a resource with its computed setup order.
type orderedResource struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Order        int      `json:"order"`
	Dependencies []string `json:"dependencies"`
}

// topoSortResources performs a dependency-aware topological sort on resources.
// Resources with no dependencies are placed first (order 1), then resources
// whose dependencies are all satisfied, and so on. Resources with unmet
// dependencies are placed last with a final catch-all order.
func topoSortResources(resources []Resource, deps map[string][]string) []orderedResource {
	ordered := make([]orderedResource, 0, len(resources))
	remaining := make([]orderedResource, 0)
	placed := make(map[string]bool, len(resources))

	for _, res := range resources {
		d := deps[res.Name]
		entry := orderedResource{
			Name:         res.Name,
			Category:     res.Category,
			Dependencies: d,
		}
		if len(d) == 0 {
			entry.Order = 1
			ordered = append(ordered, entry)
			placed[res.Name] = true
		} else {
			remaining = append(remaining, entry)
		}
	}

	order := 2
	for len(remaining) > 0 {
		var stillRemaining []orderedResource
		for _, r := range remaining {
			allMet := true
			for _, dep := range r.Dependencies {
				if !placed[dep] {
					allMet = false
					break
				}
			}
			if allMet {
				r.Order = order
				ordered = append(ordered, r)
				placed[r.Name] = true
			} else {
				stillRemaining = append(stillRemaining, r)
			}
		}

		if len(stillRemaining) == len(remaining) {
			for i := range stillRemaining {
				stillRemaining[i].Order = order + 1
				ordered = append(ordered, stillRemaining[i])
			}
			break
		}

		remaining = stillRemaining
		order++
	}

	return ordered
}

// handleSetupOrder returns resources sorted by recommended setup order based on dependencies.
func (s *Server) handleSetupOrder(w http.ResponseWriter, r *http.Request) {
	resources, err := loadResources()
	if err != nil {
		writeResourceLoadError(w, err)
		return
	}

	ordered := topoSortResources(resources, knownDependencies)

	writeJSON(w, http.StatusOK, map[string]any{
		"setup_order": ordered,
		"total":       len(ordered),
	})
}

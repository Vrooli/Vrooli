package main

import (
	"net/http"
)

// handleSetupOrder returns resources sorted by recommended setup order based on dependencies.
func (s *Server) handleSetupOrder(w http.ResponseWriter, r *http.Request) {
	resources, err := loadResources()
	if err != nil {
		writeResourceLoadError(w, err)
		return
	}

	// Build dependency-aware ordering: resources with no deps first
	type orderedResource struct {
		Name         string   `json:"name"`
		Category     string   `json:"category"`
		Order        int      `json:"order"`
		Dependencies []string `json:"dependencies"`
	}

	// Topological sort: place resources with no unmet dependencies first
	ordered := make([]orderedResource, 0, len(resources))
	remaining := make([]orderedResource, 0)
	placed := make(map[string]bool, len(resources))

	for _, res := range resources {
		deps := knownDependencies[res.Name]
		entry := orderedResource{
			Name:         res.Name,
			Category:     res.Category,
			Dependencies: deps,
		}
		if len(deps) == 0 {
			entry.Order = 1
			ordered = append(ordered, entry)
			placed[res.Name] = true
		} else {
			remaining = append(remaining, entry)
		}
	}

	// Iteratively place resources whose deps are all satisfied
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

	writeJSON(w, http.StatusOK, map[string]any{
		"setup_order": ordered,
		"total":       len(ordered),
	})
}

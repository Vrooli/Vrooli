package main

import (
	"net/http"
	"time"
)

// resourceHealthStatus represents the health state of a single resource.
type resourceHealthStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Category    string `json:"category"`
	Available   bool   `json:"available"`
	LastChecked string `json:"last_checked"`
}

func (s *Server) handleResourceHealth(w http.ResponseWriter, _ *http.Request) {
	resources, err := loadResources()
	if err != nil {
		writeResourceLoadError(w, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	statuses := make([]resourceHealthStatus, 0, len(resources))
	healthy := 0

	for _, res := range resources {
		available := res.Status == "running"
		if available {
			healthy++
		}
		statuses = append(statuses, resourceHealthStatus{
			Name:        res.Name,
			Status:      res.Status,
			Category:    res.Category,
			Available:   available,
			LastChecked: now,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resources":     statuses,
		"total":         len(statuses),
		"healthy_count": healthy,
		"checked_at":    now,
	})
}

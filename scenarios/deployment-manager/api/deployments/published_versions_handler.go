package deployments

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// PublishedVersionsHandler handles HTTP endpoints for querying published versions.
type PublishedVersionsHandler struct {
	repo PublishedVersionsRepository
	log  func(string, map[string]interface{})
}

// NewPublishedVersionsHandler creates a new published versions handler.
func NewPublishedVersionsHandler(repo PublishedVersionsRepository, log func(string, map[string]interface{})) *PublishedVersionsHandler {
	return &PublishedVersionsHandler{repo: repo, log: log}
}

// GetPublishedVersions handles GET /api/v1/profiles/{id}/published-versions
// Query params: ?history=true&platform=X&limit=N
func (h *PublishedVersionsHandler) GetPublishedVersions(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]

	history := r.URL.Query().Get("history") == "true"
	platform := r.URL.Query().Get("platform")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	var (
		versions []PublishedVersion
		err      error
	)

	if history {
		versions, err = h.repo.GetHistory(r.Context(), profileID, platform, limit)
	} else {
		versions, err = h.repo.GetLatestByProfile(r.Context(), profileID)
	}

	if err != nil {
		h.log("failed to get published versions", map[string]interface{}{
			"error":      err.Error(),
			"profile_id": profileID,
		})
		http.Error(w, `{"error":"failed to get published versions"}`, http.StatusInternalServerError)
		return
	}

	if versions == nil {
		versions = []PublishedVersion{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"profile_id": profileID,
		"versions":   versions,
	})
}

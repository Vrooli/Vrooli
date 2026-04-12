package backlog

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// validateGlobsRequest is the JSON body for POST /backlog/validate-globs.
type validateGlobsRequest struct {
	Patterns []string `json:"patterns"`
}

// validateGlobResult describes the validation outcome for a single pattern.
type validateGlobResult struct {
	Pattern    string `json:"pattern"`
	MatchCount int    `json:"matchCount"`
	Valid      bool   `json:"valid"`
	Error      string `json:"error,omitempty"`
	Warning    string `json:"warning,omitempty"`
}

// validateGlobsResponse is the JSON response for POST /backlog/validate-globs.
type validateGlobsResponse struct {
	Results []validateGlobResult `json:"results"`
}

// ValidateGlobs checks an array of glob patterns for syntax validity and
// counts how many files match each pattern within the project directory.
//
// Deferred migration note: this handler still derives repo root from the local
// scenario layout and still couples bespoke matching behavior to legacy
// validation logic. It is intentionally not Phase 1 contract authority.
func (h *Handler) ValidateGlobs(w http.ResponseWriter, r *http.Request) {
	var req validateGlobsRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[backlog] validate-globs", apierr.BadRequest("%s", err.Error()))
		return
	}

	// Resolve the project root (parent of the swarm-manager scenario).
	projectRoot := filepath.Dir(filepath.Dir(h.rootDir))

	results := make([]validateGlobResult, 0, len(req.Patterns))
	for _, pattern := range req.Patterns {
		res := validateGlobResult{Pattern: pattern, Valid: true}

		// Reuse the server-side validation logic for syntax checks.
		if err := validateGlobs([]string{pattern}); err != nil {
			res.Valid = false
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		// Count matching files using doublestar for ** support.
		fullPattern := filepath.Join(projectRoot, pattern)
		matches, err := doublestar.FilepathGlob(fullPattern)
		if err != nil {
			res.Valid = false
			res.Error = "invalid glob syntax: " + err.Error()
			results = append(results, res)
			continue
		}

		// Filter out directories — only count files.
		count := 0
		for _, m := range matches {
			info, err := os.Stat(m)
			if err == nil && !info.IsDir() {
				count++
			}
		}
		res.MatchCount = count
		if count == 0 {
			res.Warning = "No files match this pattern"
		}
		results = append(results, res)
	}

	if err := httputil.JSON(w, validateGlobsResponse{Results: results}); err != nil {
		apierr.MapError(w, "[backlog] validate-globs", apierr.Internal("failed to encode response"))
	}
}

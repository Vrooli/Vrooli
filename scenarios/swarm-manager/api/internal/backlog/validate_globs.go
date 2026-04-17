package backlog

import (
	"net/http"
	"os"
	"path/filepath"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	repocontract "github.com/vrooli/repo-contract-go"
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
// counts how many files match each pattern under the active repo contract.
func (h *Handler) ValidateGlobs(w http.ResponseWriter, r *http.Request) {
	var req validateGlobsRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[backlog] validate-globs", apierr.BadRequest("%s", err.Error()))
		return
	}

	projectRoot, err := resolveRepoRoot(h.rootDir)
	if err != nil {
		apierr.MapError(w, "[backlog] validate-globs", apierr.Internal("failed to resolve repo root"))
		return
	}

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

		count, err := repocontract.FileMatchCount(projectRoot, pattern)
		if err != nil {
			res.Valid = false
			res.Error = err.Error()
			results = append(results, res)
			continue
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

func resolveRepoRoot(rootDir string) (string, error) {
	if projectRoot, err := repocontract.FindRepoRootFromPath(rootDir); err == nil {
		return projectRoot, nil
	}

	current := rootDir
	for {
		if _, err := os.Stat(filepath.Join(current, ".vrooli", "repo-contract.json")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", os.ErrNotExist
}

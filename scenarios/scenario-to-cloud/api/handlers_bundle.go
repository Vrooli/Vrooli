package main

import (
	"net/http"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"time"
)

// handleManifestValidate validates a domain.CloudManifest and returns normalized output.
// POST /api/v1/manifest/validate
func (s *Server) handleManifestValidate(w http.ResponseWriter, r *http.Request) {
	raw, err := httputil.DecodeJSON[map[string]interface{}](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues, err := manifest.ValidateRaw(raw)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_validate_failed",
			Message: "Failed to validate manifest",
			Hint:    err.Error(),
		})
		return
	}

	resp := domain.ManifestValidateResponse{
		Valid:      !manifest.HasBlockingIssues(issues),
		Issues:     issues,
		Manifest:   normalized,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		SchemaHint: "Use /api/v1/manifest/schema for structural contract; semantic checks are validated server-side.",
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// handleBundleBuild builds a mini-Vrooli bundle from a manifest.
// POST /api/v1/bundle/build
func (s *Server) handleBundleBuild(w http.ResponseWriter, r *http.Request) {
	m, err := httputil.DecodeJSON[domain.CloudManifest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := manifest.ValidateAndNormalize(m)
	if manifest.HasBlockingIssues(issues) {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		bundle.WriteRepoNotFoundError(w, err)
		return
	}

	outDir, err := bundle.GetLocalBundlesDir()
	if err != nil {
		bundle.WriteRepoNotFoundError(w, err)
		return
	}

	artifact, err := bundle.BuildMiniVrooliBundle(repoRoot, outDir, normalized)
	if err != nil {
		bundle.WriteBundlesDirError(w, "build mini-Vrooli bundle", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"artifact":  artifact,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

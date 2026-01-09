package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/ssh"
)

// DeploymentContext bundles the common results of fetching and parsing a deployment.
// This reduces cognitive load by eliminating the repeated pattern of:
//   - Extract ID from URL
//   - Fetch deployment from database
//   - Handle errors and not-found
//   - Parse manifest JSON
//   - Validate and normalize manifest
//   - Check VPS target exists
type DeploymentContext struct {
	ID         string
	Deployment *domain.Deployment
	Manifest   domain.CloudManifest
	SSHConfig  ssh.Config
	Workdir    string
}

// FetchDeploymentContext extracts deployment ID from the request, fetches the deployment,
// parses the manifest, and returns all the common context needed by VPS-related handlers.
// Returns nil and writes an error response if any step fails.
// Use this for handlers that need VPS access (most deployment handlers).
func (s *Server) FetchDeploymentContext(w http.ResponseWriter, r *http.Request) *DeploymentContext {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return nil
	}

	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return nil
	}

	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return nil
	}

	normalized, _ := manifest.ValidateAndNormalize(m)

	if normalized.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return nil
	}

	return &DeploymentContext{
		ID:         id,
		Deployment: deployment,
		Manifest:   normalized,
		SSHConfig:  ssh.ConfigFromManifest(normalized),
		Workdir:    normalized.Target.VPS.Workdir,
	}
}

// FetchDeploymentOnly fetches a deployment without requiring VPS target.
// Use this for handlers that work with deployments regardless of target type.
// Returns (id, deployment) or writes an error response and returns ("", nil) on failure.
func (s *Server) FetchDeploymentOnly(w http.ResponseWriter, r *http.Request) (string, *domain.Deployment) {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return "", nil
	}

	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return "", nil
	}

	return id, deployment
}

// DeploymentRepository is the interface used by FetchDeploymentContext.
// This is defined here to avoid import cycles with persistence package.
type DeploymentRepository interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
}

// DecodeAndValidateManifest consolidates the common pattern of:
//   - Decoding a request body containing a manifest field
//   - Validating and normalizing the manifest
//   - Writing error response if validation fails with blocking issues
//
// Returns (request, normalized manifest, issues, ok). If ok is false, an error
// response was already written and the handler should return immediately.
// This reduces ~20 lines of repeated boilerplate per handler.
func DecodeAndValidateManifest[T any](w http.ResponseWriter, body io.Reader, maxBytes int64, extractManifest func(T) domain.CloudManifest) (T, domain.CloudManifest, []domain.ValidationIssue, bool) {
	var zero T
	req, err := httputil.DecodeJSON[T](body, maxBytes)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return zero, domain.CloudManifest{}, nil, false
	}

	m := extractManifest(req)
	normalized, issues := manifest.ValidateAndNormalize(m)
	if manifest.HasBlockingIssues(issues) {
		httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return zero, domain.CloudManifest{}, nil, false
	}

	return req, normalized, issues, true
}

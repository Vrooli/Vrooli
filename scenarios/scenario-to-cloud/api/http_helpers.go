package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

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
	Manifest   CloudManifest
	SSHConfig  SSHConfig
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
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return nil
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return nil
	}

	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return nil
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)

	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return nil
	}

	return &DeploymentContext{
		ID:         id,
		Deployment: deployment,
		Manifest:   normalized,
		SSHConfig:  sshConfigFromManifest(normalized),
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
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return "", nil
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
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

// writeRepoNotFoundError writes a standardized error response for when the repo root cannot be found.
func writeRepoNotFoundError(w http.ResponseWriter, err error) {
	writeAPIError(w, http.StatusInternalServerError, APIError{
		Code:    "repo_not_found",
		Message: "Could not find repository root",
		Hint:    err.Error(),
	})
}

// writeBundlesDirError writes a standardized error response for bundles directory errors.
func writeBundlesDirError(w http.ResponseWriter, operation string, err error) {
	writeAPIError(w, http.StatusInternalServerError, APIError{
		Code:    operation + "_failed",
		Message: "Failed to " + operation,
		Hint:    err.Error(),
	})
}

type APIErrorEnvelope struct {
	Error APIError `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, apiErr APIError) {
	writeJSON(w, status, APIErrorEnvelope{Error: apiErr})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// decodeRequestBody decodes a JSON request body into the provided pointer.
// Returns false and writes an error response if decoding fails.
// This consolidates the common pattern of:
//   - json.NewDecoder(r.Body).Decode(&req)
//   - if err != nil { writeAPIError(...); return }
func decodeRequestBody[T any](w http.ResponseWriter, r *http.Request, dest *T) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Invalid request body",
			Hint:    err.Error(),
		})
		return false
	}
	return true
}

// DecodeAndValidateManifest consolidates the common pattern of:
//   - Decoding a request body containing a manifest field
//   - Validating and normalizing the manifest
//   - Writing error response if validation fails with blocking issues
//
// Returns (request, normalized manifest, issues, ok). If ok is false, an error
// response was already written and the handler should return immediately.
// This reduces ~20 lines of repeated boilerplate per handler.
func DecodeAndValidateManifest[T any](w http.ResponseWriter, body io.Reader, maxBytes int64, extractManifest func(T) CloudManifest) (T, CloudManifest, []ValidationIssue, bool) {
	var zero T
	req, err := decodeJSON[T](body, maxBytes)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return zero, CloudManifest{}, nil, false
	}

	manifest := extractManifest(req)
	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if hasBlockingIssues(issues) {
		writeJSON(w, http.StatusUnprocessableEntity, ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return zero, CloudManifest{}, nil, false
	}

	return req, normalized, issues, true
}

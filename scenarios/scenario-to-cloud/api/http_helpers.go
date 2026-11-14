package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"

	"github.com/gorilla/mux"
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
	Workdir    string
}

// FetchDeploymentContext extracts deployment ID from the request, fetches the deployment,
// parses the manifest, and returns all the common context needed by VPS-related handlers.
// Returns nil and writes an error response if any step fails.
// Use this for handlers that need VPS access (most deployment handlers).
func (s *Server) FetchDeploymentContext(w http.ResponseWriter, r *http.Request) *DeploymentContext {
	dc, err := s.loadDeploymentContext(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		apierrors.Write(w, err)
		return nil
	}
	return dc
}

// loadDeploymentContext is the transport-independent half of
// FetchDeploymentContext, shared by REST handlers and Connect services.
func (s *Server) loadDeploymentContext(ctx context.Context, id string) (*DeploymentContext, *apierrors.Error) {
	repo := s.deploymentRepo
	if repo == nil {
		repo = s.repo
	}
	deployment, err := repo.GetDeployment(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Failed to get deployment", err)
	}

	if deployment == nil {
		return nil, deploymentNotFound(id)
	}

	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		return nil, apierrors.Internal("Failed to parse deployment manifest", err)
	}

	normalized, _ := manifest.ValidateAndNormalize(m)

	if normalized.Target.VPS == nil {
		return nil, apierrors.New(apierrors.CodeUnsupportedCapability, "Deployment does not have a VPS target").WithDetail("id", id)
	}

	return &DeploymentContext{
		ID:         id,
		Deployment: deployment,
		Manifest:   normalized,
		Workdir:    normalized.Target.VPS.Workdir,
	}, nil
}

// FetchDeploymentOnly fetches a deployment without requiring VPS target.
// Use this for handlers that work with deployments regardless of target type.
// Returns (id, deployment) or writes an error response and returns ("", nil) on failure.
func (s *Server) FetchDeploymentOnly(w http.ResponseWriter, r *http.Request) (string, *domain.Deployment) {
	id := mux.Vars(r)["id"]

	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to get deployment", err))
		return "", nil
	}

	if deployment == nil {
		apierrors.Write(w, deploymentNotFound(id))
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

// targetFor is the reach target of a deployment: its recorded binding, or
// the manifest locator for a record that predates target bindings. The
// workdir always comes from the normalized manifest.
func (s *Server) targetFor(dc *DeploymentContext) identity.TargetRef {
	target := dc.Deployment.Target
	if target.IsZero() {
		target = domain.TargetRefFromManifest(dc.Manifest)
	}
	if target.Locator.Host == "" && dc.Manifest.Target.VPS != nil {
		target.Locator.Host = dc.Manifest.Target.VPS.Host
		target.Locator.Port = dc.Manifest.Target.VPS.Port
	}
	if target.Locator.User == "" && dc.Manifest.Target.VPS != nil {
		target.Locator.User = dc.Manifest.Target.VPS.User
	}
	if target.Locator.Workdir == "" {
		target.Locator.Workdir = dc.Workdir
	}
	if target.Transport == "" {
		target.Transport = identity.TransportSSH
	}
	return target
}

// reachFor is the transport a deployment's reads and lifecycle verbs go
// through: the router in production; the bounded SSH adapter over the
// target locator when no router is wired (tests, ad hoc runs).
func (s *Server) reachFor(*DeploymentContext) reach.Reach {
	return s.targetReach()
}

// proberFor is the read-only probe surface for a deployment.
func (s *Server) proberFor(dc *DeploymentContext) vps.Prober {
	return vps.Prober{Reach: s.reachFor(dc), Target: s.targetFor(dc)}
}

// runtimeFor is the target-facing runtime for management actions on a
// deployment (host repairs through the target owner).
func (s *Server) runtimeFor(dc *DeploymentContext) vps.Runtime {
	return vps.Runtime{Reach: s.reachFor(dc), Target: s.targetFor(dc)}
}

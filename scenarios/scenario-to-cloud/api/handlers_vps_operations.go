package main

import (
	"context"
	"net/http"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/vps"
)

// Type aliases for backward compatibility with API documentation and request parsing.
type (
	VPSSetupRequest   = vps.SetupRequest
	VPSDeployRequest  = vps.DeployRequest
	VPSInspectRequest = vps.InspectRequest
)

// handleVPSSetupPlan generates a plan for VPS setup without executing.
// POST /api/v1/vps/setup/plan
func (s *Server) handleVPSSetupPlan(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSSetupRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	plan, err := vps.BuildSetupPlan(manifest, req.BundlePath)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_setup_request",
			Message: "Unable to build VPS setup plan",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"plan":      plan,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSSetupApply executes VPS setup: copies bundle and runs Vrooli setup.
// POST /api/v1/vps/setup/apply
func (s *Server) handleVPSSetupApply(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSSetupRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	resp := vps.RunSetup(ctx, manifest, req.BundlePath, s.sshRunner, s.scpRunner)
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":    resp,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSDeployPlan generates a plan for VPS deploy without executing.
// POST /api/v1/vps/deploy/plan
func (s *Server) handleVPSDeployPlan(w http.ResponseWriter, r *http.Request) {
	_, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSDeployRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	plan, err := vps.BuildDeployPlan(manifest)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_deploy_request",
			Message: "Unable to build VPS deploy plan",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"plan":      plan,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSDeployApply executes VPS deploy: starts resources and scenario.
// POST /api/v1/vps/deploy/apply
func (s *Server) handleVPSDeployApply(w http.ResponseWriter, r *http.Request) {
	_, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSDeployRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	resp := vps.RunDeploy(ctx, manifest, s.sshRunner, s.secretsGenerator, nil)
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":    resp,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSInspectPlan generates a plan for VPS inspection without executing.
// POST /api/v1/vps/inspect/plan
func (s *Server) handleVPSInspectPlan(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSInspectRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	opts, err := req.Options.Normalize(manifest)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_inspect_request",
			Message: "Unable to build VPS inspect plan",
			Hint:    err.Error(),
		})
		return
	}

	plan, err := vps.BuildInspectPlan(manifest, opts)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_inspect_request",
			Message: "Unable to build VPS inspect plan",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"plan":      plan,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSInspectApply executes VPS inspection: retrieves status and logs.
// POST /api/v1/vps/inspect/apply
func (s *Server) handleVPSInspectApply(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSInspectRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	opts, err := req.Options.Normalize(manifest)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_inspect_request",
			Message: "Unable to run VPS inspect",
			Hint:    err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result := vps.RunInspect(ctx, manifest, opts, s.sshRunner)
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

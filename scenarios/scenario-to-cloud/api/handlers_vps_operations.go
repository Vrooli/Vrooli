package main

import (
	"context"
	"net/http"
	"time"
)

// handleVPSSetupPlan generates a plan for VPS setup without executing.
// POST /api/v1/vps/setup/plan
func (s *Server) handleVPSSetupPlan(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSSetupRequest) CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	plan, err := BuildVPSSetupPlan(manifest, req.BundlePath)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_setup_request",
			Message: "Unable to build VPS setup plan",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plan":      plan,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSSetupApply executes VPS setup: copies bundle and runs Vrooli setup.
// POST /api/v1/vps/setup/apply
func (s *Server) handleVPSSetupApply(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSSetupRequest) CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	resp := RunVPSSetup(ctx, manifest, req.BundlePath, s.sshRunner, s.scpRunner)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    resp,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSDeployPlan generates a plan for VPS deploy without executing.
// POST /api/v1/vps/deploy/plan
func (s *Server) handleVPSDeployPlan(w http.ResponseWriter, r *http.Request) {
	_, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSDeployRequest) CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	plan, err := BuildVPSDeployPlan(manifest)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_deploy_request",
			Message: "Unable to build VPS deploy plan",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plan":      plan,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSDeployApply executes VPS deploy: starts resources and scenario.
// POST /api/v1/vps/deploy/apply
func (s *Server) handleVPSDeployApply(w http.ResponseWriter, r *http.Request) {
	_, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSDeployRequest) CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	resp := RunVPSDeploy(ctx, manifest, s.sshRunner, s.secretsGenerator)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    resp,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSInspectPlan generates a plan for VPS inspection without executing.
// POST /api/v1/vps/inspect/plan
func (s *Server) handleVPSInspectPlan(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSInspectRequest) CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	opts, err := req.Options.Normalize(manifest)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_inspect_request",
			Message: "Unable to build VPS inspect plan",
			Hint:    err.Error(),
		})
		return
	}

	plan, err := BuildVPSInspectPlan(manifest, opts)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_inspect_request",
			Message: "Unable to build VPS inspect plan",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plan":      plan,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSInspectApply executes VPS inspection: retrieves status and logs.
// POST /api/v1/vps/inspect/apply
func (s *Server) handleVPSInspectApply(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSInspectRequest) CloudManifest { return r.Manifest })
	if !ok {
		return
	}

	opts, err := req.Options.Normalize(manifest)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_inspect_request",
			Message: "Unable to run VPS inspect",
			Hint:    err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result := RunVPSInspect(ctx, manifest, opts, s.sshRunner)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/vps"
)

// Type aliases for backward compatibility with API documentation and request parsing.
type (
	VPSSetupRequest   = vps.SetupRequest
	VPSDeployRequest  = vps.DeployRequest
	VPSInspectRequest = vps.InspectRequest
)

// planResponse is the shared body of the plan endpoints. `plan` keeps the
// legacy step shape (id = action id, command = shell preview); the
// executable plan and its digest are what apply consumes.
func planResponse(plan *execplan.Plan, manifest domain.CloudManifest, issues []domain.ValidationIssue) map[string]interface{} {
	return map[string]interface{}{
		"plan":            vps.RenderSteps(plan, manifest),
		"plan_digest":     plan.MustDigest(),
		"executable_plan": plan,
		"preview":         vps.RenderPreview(plan, manifest),
		"issues":          issues,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
}

// reviewedPlan recompiles the plan for an apply request and refuses when the
// reviewed digest is missing or does not match. Nothing is executed and no
// record is written before this returns a plan.
func reviewedPlan(w http.ResponseWriter, compile func() (*execplan.Plan, error), reviewedDigest, invalidCode string) (*execplan.Plan, bool) {
	reviewedDigest = strings.TrimSpace(reviewedDigest)
	if reviewedDigest == "" {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "plan_digest is required: review the plan first and submit its digest").
			WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "replan", Reference: "plan", Label: "Call the plan endpoint and submit plan_digest"}))
		return nil, false
	}
	plan, err := compile()
	if err != nil {
		if typed := apierrors.As(err); typed != nil {
			apierrors.Write(w, typed)
			return nil, false
		}
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: invalidCode, Message: "Unable to build VPS plan", Hint: err.Error()})
		return nil, false
	}
	current := plan.MustDigest()
	if current != reviewedDigest {
		apierrors.Write(w, execplan.DigestMismatchError(reviewedDigest, current))
		return nil, false
	}
	if plan.Outcome == execplan.OutcomeNeedsInput {
		apierrors.Write(w, execplan.NeedsInputError(plan.Handoff))
		return nil, false
	}
	return plan, true
}

// admitAdHocOperation records a durable operation when the caller binds the
// ad hoc apply to a stored deployment. Without a deployment id the apply is
// executed unrecorded (legacy behaviour, documented).
func (s *Server) admitAdHocOperation(ctx context.Context, w http.ResponseWriter, deploymentID, requestKey string, plan *execplan.Plan) (string, bool) {
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" || s.repo == nil {
		return "", true
	}
	op, _, err := operations.Admit(ctx, s.repo, deploymentID, requestKey, plan)
	if err != nil {
		apierrors.Write(w, err)
		return "", false
	}
	return op.ID, true
}

// handleVPSSetupPlan compiles the install-scope plan without executing.
// POST /api/v1/vps/setup/plan
func (s *Server) handleVPSSetupPlan(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSSetupRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}
	plan, err := vps.BuildSetupExecutablePlan(r.Context(), manifest, req.BundlePath)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_setup_request",
			Message: "Unable to build VPS setup plan",
			Hint:    err.Error(),
		})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, planResponse(plan, manifest, issues))
}

// handleVPSSetupApply executes the reviewed install plan. The request must
// carry the plan_digest returned by the plan endpoint; the plan is recompiled
// and refused on mismatch, so preview and apply are the same action graph.
// POST /api/v1/vps/setup/apply
func (s *Server) handleVPSSetupApply(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSSetupRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}
	plan, ok := reviewedPlan(w, func() (*execplan.Plan, error) {
		return vps.BuildSetupExecutablePlan(r.Context(), manifest, req.BundlePath)
	}, req.PlanDigest, "invalid_setup_request")
	if !ok {
		return
	}
	operationID, ok := s.admitAdHocOperation(r.Context(), w, req.DeploymentID, req.RequestKey, plan)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	progress := 0.0
	resp := vps.RunSetupPlanWithProgress(ctx, plan, manifest, req.BundlePath, s.executionRuntime(manifest, req.DeploymentID, operationID), vps.NoopProgressHub{}, vps.NoopProgressRepo{}, req.DeploymentID, &progress)
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":       resp,
		"plan_digest":  resp.PlanDigest,
		"operation_id": operationID,
		"issues":       issues,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVPSDeployPlan compiles the runtime-scope plan without executing.
// POST /api/v1/vps/deploy/plan
func (s *Server) handleVPSDeployPlan(w http.ResponseWriter, r *http.Request) {
	_, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSDeployRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}
	plan, err := vps.BuildDeployExecutablePlan(r.Context(), manifest, execplan.ScopeRuntime)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_deploy_request",
			Message: "Unable to build VPS deploy plan",
			Hint:    err.Error(),
		})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, planResponse(plan, manifest, issues))
}

// handleVPSDeployApply executes the reviewed runtime plan (see setup apply).
// POST /api/v1/vps/deploy/apply
func (s *Server) handleVPSDeployApply(w http.ResponseWriter, r *http.Request) {
	req, manifest, issues, ok := DecodeAndValidateManifest(w, r.Body, 2<<20, func(r VPSDeployRequest) domain.CloudManifest { return r.Manifest })
	if !ok {
		return
	}
	plan, ok := reviewedPlan(w, func() (*execplan.Plan, error) {
		return vps.BuildDeployExecutablePlan(r.Context(), manifest, execplan.ScopeRuntime)
	}, req.PlanDigest, "invalid_deploy_request")
	if !ok {
		return
	}
	operationID, ok := s.admitAdHocOperation(r.Context(), w, req.DeploymentID, req.RequestKey, plan)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	progress := 0.0
	resp := vps.RunDeployPlanWithProgress(ctx, plan, manifest, s.executionRuntime(manifest, req.DeploymentID, operationID), vps.NoopProgressHub{}, vps.NoopProgressRepo{}, req.DeploymentID, &progress, vps.DeployOptions{})
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":       resp,
		"plan_digest":  resp.PlanDigest,
		"operation_id": operationID,
		"issues":       issues,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
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

	result := vps.RunInspect(ctx, manifest, opts, vps.Prober{Reach: s.targetReach(), Target: domain.TargetRefFromManifest(manifest)})
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"result":    result,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

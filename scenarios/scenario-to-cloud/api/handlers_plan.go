package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/plansvc"
	"scenario-to-cloud/vps"
)

// planAdapter exposes the server's compile/apply path to the Connect
// PlansService so REST and Connect share one implementation.
type planAdapter struct{ s *Server }

// PlansService returns the Connect service over this server.
func (s *Server) PlansService() *plansvc.Service { return plansvc.New(planAdapter{s: s}) }

func (a planAdapter) CompilePlan(ctx context.Context, deploymentID, scope string, forceBundleBuild bool) (*plansvc.Compiled, error) {
	compiled, err := a.s.compileDeploymentPlanRequest(ctx, deploymentID, PlanCompileRequest{Scope: scope, ForceBundleBuild: forceBundleBuild})
	if err != nil {
		return nil, err
	}
	return &plansvc.Compiled{Plan: compiled.Plan, PlanDigest: compiled.PlanDigest, Preview: compiled.Preview, ClosureStatus: compiled.ClosureStatus}, nil
}

func (a planAdapter) ApplyPlan(ctx context.Context, deploymentID, planDigest, requestKey, scope string, runPreflight bool) (*plansvc.Applied, error) {
	result, err := a.s.applyDeploymentPlan(ctx, deploymentID, PlanApplyRequest{PlanDigest: planDigest, RequestKey: requestKey, Scope: scope, RunPreflight: runPreflight})
	if err != nil {
		return nil, err
	}
	return &plansvc.Applied{OperationID: result.OperationID, PlanDigest: result.PlanDigest, State: result.State}, nil
}

// PlanObserver supplies the target observations a deployment plan binds to.
// The default knows nothing about the target (every digest unobserved), so a
// plan is never a no-op until the health observation (P16) is wired in.
type PlanObserver interface {
	Observe(ctx context.Context, dep *domain.Deployment, manifest domain.CloudManifest, closure *domain.Closure) (execplan.Observations, error)
}

// planObserverOverride lets tests substitute an observer. Production uses
// recordObserver until the health projection owns observations.
var planObserverOverride PlanObserver

// recordObserver reads only the deployment record: revision (fence), target
// enrollment and the manifest's legacy preserve paths. Closure credential
// descriptors count as satisfied unless the manifest's secrets plan declares
// them as required operator prompts: the secrets plan is the authority on
// which inputs an operator must supply; generated and infrastructure
// credentials are provisioned by credentials.provision.
type recordObserver struct{}

func (recordObserver) Observe(_ context.Context, dep *domain.Deployment, manifest domain.CloudManifest, closure *domain.Closure) (execplan.Observations, error) {
	obs := execplan.Observations{}
	if dep != nil {
		obs.DeploymentRevision = dep.Fence
		obs.TargetEnrollment = dep.Target.EnrollmentGeneration
	}
	if manifest.Target.VPS != nil {
		obs.LegacyDataInventory = append(obs.LegacyDataInventory, manifest.Target.VPS.PreservePaths...)
	}
	obs.PersistentDataBindings = persistentDataBindings(dep)
	prompted := map[string]bool{}
	if manifest.Secrets != nil {
		for _, secret := range manifest.Secrets.BundleSecrets {
			if secret.Class == "user_prompt" && secret.Required && secret.Descriptor != nil {
				prompted[strings.TrimSpace(secret.Descriptor.LogicalID)+":"+strings.TrimSpace(secret.Descriptor.Field)] = true
			}
		}
	}
	if closure != nil {
		for _, component := range closure.ComponentsOfKind(domain.ClosureKindCredentialDescriptor) {
			if component.Credential == nil {
				continue
			}
			address := strings.TrimSpace(component.Credential.LogicalID) + ":" + strings.TrimSpace(component.Credential.Field)
			if !prompted[address] {
				obs.SatisfiedInputs = append(obs.SatisfiedInputs, address)
			}
		}
	}
	return obs, nil
}

func planObserver() PlanObserver {
	if planObserverOverride != nil {
		return planObserverOverride
	}
	return recordObserver{}
}

// CompiledPlan is what the plan endpoint and the Connect service return.
type CompiledPlan struct {
	SchemaVersion string               `json:"schema_version"`
	Plan          *execplan.Plan       `json:"plan"`
	PlanDigest    string               `json:"plan_digest"`
	Preview       execplan.Preview     `json:"preview"`
	Steps         []domain.VPSPlanStep `json:"steps"`
	ClosureStatus string               `json:"closure_status"`
}

// PlanCompileRequest is the preview body. ForceBundleBuild rebuilds the
// release archive before compiling; a deployment without a recorded bundle
// is built regardless, so the compiled plan pins a real release digest
// exactly as the execute path does.
type PlanCompileRequest struct {
	Scope            string `json:"scope,omitempty"`
	ForceBundleBuild bool   `json:"force_bundle_build,omitempty"`
}

// PlanApplyRequest is the apply body: the reviewed digest and the caller's
// idempotency key. RunPreflight runs the target preflight before the first
// effectful step of the admitted operation.
type PlanApplyRequest struct {
	PlanDigest   string `json:"plan_digest"`
	RequestKey   string `json:"request_key"`
	Scope        string `json:"scope,omitempty"`
	RunPreflight bool   `json:"run_preflight,omitempty"`
	// Plan is the reviewed envelope (optional). When present, a mismatch is
	// classified: material precondition drift is plan_stale.
	Plan *execplan.Plan `json:"plan,omitempty"`
}

// PlanApplyResult is the apply response.
type PlanApplyResult struct {
	SchemaVersion string `json:"schema_version"`
	OperationID   string `json:"operation_id,omitempty"`
	PlanDigest    string `json:"plan_digest"`
	State         string `json:"state"`
}

// registerPlanRoutes mounts the deployment plan endpoints. Preview creates no
// records and no target effects; apply admits an operation for the exact
// reviewed digest.
func (s *Server) registerPlanRoutes(api *mux.Router) {
	api.HandleFunc("/deployments/{id}/plan", s.handleCompileDeploymentPlan).Methods("POST")
	api.HandleFunc("/deployments/{id}/plan/apply", s.handleApplyDeploymentPlan).Methods("POST")
}

// compileDeploymentPlan compiles the plan for a stored deployment. It is the
// single compile path shared by preview, apply and the Connect service.
func (s *Server) compileDeploymentPlan(ctx context.Context, deploymentID, scope string) (*CompiledPlan, *apierrors.Error) {
	return s.compileDeploymentPlanWith(ctx, deploymentID, scope, execplan.Policy{})
}

// compileDeploymentPlanRequest is the preview entry point: it makes sure the
// release archive exists (building or rebuilding it through the
// orchestrator) and then compiles. The archive is a cloud-local input of the
// plan, not a target effect, so preview and execute pin the same digest.
func (s *Server) compileDeploymentPlanRequest(ctx context.Context, deploymentID string, req PlanCompileRequest) (*CompiledPlan, *apierrors.Error) {
	if err := s.ensureReleaseBundle(ctx, deploymentID, req.ForceBundleBuild); err != nil {
		return nil, err
	}
	return s.compileDeploymentPlan(ctx, deploymentID, req.Scope)
}

// ensureReleaseBundle builds the release archive when the record has none or
// when a rebuild is forced. Without an orchestrator a forced rebuild is
// refused rather than silently skipped; a missing bundle is left for compile
// to report as a pending release digest.
func (s *Server) ensureReleaseBundle(ctx context.Context, deploymentID string, force bool) *apierrors.Error {
	dctx, err := s.loadDeploymentContext(ctx, deploymentID)
	if err != nil {
		return err
	}
	missing := dctx.Deployment.BundlePath == nil || strings.TrimSpace(*dctx.Deployment.BundlePath) == ""
	if !force && !missing {
		return nil
	}
	if s.orchestrator == nil {
		if force {
			return apierrors.New(apierrors.CodeInvalidRequest, "force_bundle_build requires the release builder, which is not configured").WithDetail("step", "bundle_build")
		}
		return nil
	}
	if _, berr := s.orchestrator.EnsureBundle(ctx, deploymentID, dctx.Manifest, dctx.Deployment.BundlePath, force); berr != nil {
		return apierrors.New(apierrors.CodeInvalidRequest, "Bundle build failed").WithDetail("cause", berr.Error()).WithDetail("step", "bundle_build")
	}
	return nil
}

// compileDeploymentPlanWith compiles under an explicit policy (retention
// for the retire scope, a pinned activation strategy).
func (s *Server) compileDeploymentPlanWith(ctx context.Context, deploymentID, scope string, policy execplan.Policy) (*CompiledPlan, *apierrors.Error) {
	dctx, err := s.loadDeploymentContext(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	if scope == "" {
		scope = execplan.ScopeFull
	}
	req := vps.PlanRequest{
		Manifest:   dctx.Manifest,
		Deployment: dctx.Deployment,
		Scope:      scope,
		Policy:     policy,
	}
	if dctx.Deployment.BundlePath != nil {
		req.BundlePath = *dctx.Deployment.BundlePath
	}
	closureStatus := "derived"
	if svc, svcErr := closureService(); svcErr == nil && svc != nil {
		derived, derr := svc.Derive(ctx, closure.Request{
			ScenarioID:  dctx.Manifest.Scenario.ID,
			Environment: dctx.Deployment.Environment,
			OS:          svc.DefaultPlatform.OS,
			Arch:        svc.DefaultPlatform.Arch,
			Scope:       string(closure.ScopeBundle),
		})
		if derr != nil {
			closureStatus = "unavailable: " + derr.Error()
		} else {
			req.Closure = &derived
		}
	} else {
		closureStatus = "unavailable: closure service not configured"
	}
	obs, oerr := planObserver().Observe(ctx, dctx.Deployment, dctx.Manifest, req.Closure)
	if oerr != nil {
		return nil, apierrors.Internal("Failed to observe deployment", oerr)
	}
	req.Observations = obs
	plan, cerr := vps.CompilePlan(ctx, req)
	if cerr != nil {
		if typed := apierrors.As(cerr); typed != nil {
			return nil, typed
		}
		return nil, apierrors.New(apierrors.CodeManifestInvalid, "Unable to compile the deployment plan").WithDetail("cause", cerr.Error())
	}
	return &CompiledPlan{
		SchemaVersion: execplan.SchemaVersion,
		Plan:          plan,
		PlanDigest:    plan.MustDigest(),
		Preview:       vps.RenderPreview(plan, dctx.Manifest),
		Steps:         vps.RenderSteps(plan, dctx.Manifest),
		ClosureStatus: closureStatus,
	}, nil
}

// applyDeploymentPlan recompiles, compares the reviewed digest, validates
// material preconditions and admits a durable operation. It returns before
// any target effect; the pipeline runs asynchronously.
func (s *Server) applyDeploymentPlan(ctx context.Context, deploymentID string, req PlanApplyRequest) (*PlanApplyResult, *apierrors.Error) {
	reviewed := strings.TrimSpace(req.PlanDigest)
	if reviewed == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "plan_digest is required: review the plan first and submit its digest")
	}
	if strings.TrimSpace(req.RequestKey) == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "request_key is required")
	}
	compiled, err := s.compileDeploymentPlan(ctx, deploymentID, req.Scope)
	if err != nil {
		return nil, err
	}
	if req.Plan != nil && req.Plan.MustDigest() != reviewed {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "The submitted plan does not match plan_digest")
	}
	if compiled.PlanDigest != reviewed {
		// With the reviewed envelope in hand the difference can be classified:
		// a changed material precondition is plan_stale (replan required); any
		// other difference is a digest mismatch.
		if req.Plan != nil {
			if reasons := execplan.Compare(req.Plan, compiled.Plan); execplan.Material(reasons) {
				return nil, execplan.StaleError(reasons)
			}
		}
		return nil, execplan.DigestMismatchError(reviewed, compiled.PlanDigest)
	}
	switch compiled.Plan.Outcome {
	case execplan.OutcomeNeedsInput:
		return nil, execplan.NeedsInputError(compiled.Plan.Handoff)
	case execplan.OutcomeNoOp:
		return &PlanApplyResult{SchemaVersion: execplan.SchemaVersion, PlanDigest: compiled.PlanDigest, State: "no_op"}, nil
	}
	if s.repo == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "Operation store is not configured")
	}
	op, created, aerr := operations.Admit(ctx, s.repo, deploymentID, req.RequestKey, compiled.Plan)
	if aerr != nil {
		return nil, aerr
	}
	result := &PlanApplyResult{SchemaVersion: execplan.SchemaVersion, OperationID: op.ID, PlanDigest: op.PlanDigest, State: string(op.State)}
	if !created {
		// Replay: the same request key already admitted this digest.
		return result, nil
	}
	if s.operations == nil {
		// No owner wired (tests of the pure plan path): the record stays
		// admitted for reconciliation by whichever owner starts next.
		return result, nil
	}
	s.operations.Submit(ctx, op.ID, operations.ExecuteOptions{RunPreflight: req.RunPreflight})
	return result, nil
}

// handleCompileDeploymentPlan compiles and previews the plan.
// POST /api/v1/deployments/{id}/plan
func (s *Server) handleCompileDeploymentPlan(w http.ResponseWriter, r *http.Request) {
	var body PlanCompileRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidJSON, "Request body must be valid JSON").WithDetail("cause", err.Error()))
			return
		}
	}
	compiled, err := s.compileDeploymentPlanRequest(r.Context(), mux.Vars(r)["id"], body)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, compiled)
}

// handleApplyDeploymentPlan admits the reviewed plan.
// POST /api/v1/deployments/{id}/plan/apply
func (s *Server) handleApplyDeploymentPlan(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[PlanApplyRequest](r.Body, 1<<20)
	if err != nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidJSON, "Request body must be valid JSON").WithDetail("cause", err.Error()))
		return
	}
	result, aerr := s.applyDeploymentPlan(r.Context(), mux.Vars(r)["id"], req)
	if aerr != nil {
		apierrors.Write(w, aerr)
		return
	}
	httputil.WriteJSON(w, http.StatusAccepted, result)
}

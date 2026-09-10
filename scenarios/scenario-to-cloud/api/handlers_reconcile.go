package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/health"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reconcile"
	"scenario-to-cloud/vps"
)

// Reconciliation, retirement and the legacy data-binding conversion share
// one model (api/reconcile): desired state is what the deployment record
// says, observed state is what the health and target owners report, and
// every correction is an executable plan admitted through the operation
// owner. Observation never mutates a target.

// registerReconcileRoutes mounts the reconciliation surface.
func (s *Server) registerReconcileRoutes(api *mux.Router) {
	api.HandleFunc("/deployments/{id}/desired-state", s.handleGetDesiredState).Methods("GET")
	api.HandleFunc("/deployments/{id}/desired-state", s.handleSetDesiredState).Methods("PUT")
	api.HandleFunc("/deployments/{id}/reconcile", s.handleReconcileDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/retire/plan", s.handleRetirePlan).Methods("POST")
	api.HandleFunc("/deployments/{id}/retire/apply", s.handleRetireApply).Methods("POST")
	api.HandleFunc("/deployments/{id}/data-bindings/inventory", s.handleDataBindingsInventory).Methods("POST")
	api.HandleFunc("/deployments/{id}/data-bindings/adopt", s.handleDataBindingsAdopt).Methods("POST")
}

// TargetObserver reads the target owner's durable release state for
// reconciliation. Tests substitute it; production reads `release list`
// through reach.
type TargetObserver interface {
	ObserveTarget(ctx context.Context, dctx *DeploymentContext) (*TargetState, error)
}

// TargetState is the reconciliation view of `cloud-target release list`.
type TargetState struct {
	ActiveRelease         string   `json:"active_release,omitempty"`
	PreviousRelease       string   `json:"previous_release,omitempty"`
	Releases              []string `json:"releases"`
	ActivationInterrupted bool     `json:"activation_interrupted"`
	Candidate             string   `json:"candidate,omitempty"`
}

// targetObserverOverride lets tests substitute the target owner reader.
var targetObserverOverride TargetObserver

// healthObserverOverride lets tests substitute the health observation.
var healthObserverOverride func(ctx context.Context, dctx *DeploymentContext) reconcile.Observed

func (s *Server) observeTarget(ctx context.Context, dctx *DeploymentContext) (*TargetState, error) {
	if targetObserverOverride != nil {
		return targetObserverOverride.ObserveTarget(ctx, dctx)
	}
	if s.reach == nil {
		return nil, apierrors.New(apierrors.CodeReachUnavailable, "target reach is not configured")
	}
	rt := s.executionRuntime(dctx.Manifest, dctx.ID, "")
	res, err := rt.Reach.Exec(ctx, rt.Target, reach.Command{Verb: "cloud-target release list", Args: []string{"--deployment", dctx.ID, "--json"}, RequiredScope: "vrooli:read", Timeout: 20 * time.Second})
	if err != nil {
		return nil, reach.APIError(err)
	}
	var listing struct {
		Active *struct {
			ActiveRelease   string `json:"active_release"`
			PreviousRelease string `json:"previous_release"`
		} `json:"active"`
		Interrupted *struct {
			Candidate string `json:"candidate"`
		} `json:"interrupted_activation"`
		Releases []struct {
			Digest string `json:"digest"`
		} `json:"releases"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &listing); err != nil {
		return nil, apierrors.New(apierrors.CodeReachProtocolUnsupported, "release list reply is not JSON").WithDetail("cause", err.Error())
	}
	state := &TargetState{Releases: []string{}}
	if listing.Active != nil {
		state.ActiveRelease, state.PreviousRelease = listing.Active.ActiveRelease, listing.Active.PreviousRelease
	}
	if listing.Interrupted != nil {
		state.ActivationInterrupted, state.Candidate = true, listing.Interrupted.Candidate
	}
	for _, r := range listing.Releases {
		state.Releases = append(state.Releases, r.Digest)
	}
	return state, nil
}

// observeHealth projects the typed health observation onto the
// reconciliation model. Unknown values stay unknown.
func (s *Server) observeHealth(ctx context.Context, dctx *DeploymentContext) reconcile.Observed {
	if healthObserverOverride != nil {
		return healthObserverOverride(ctx, dctx)
	}
	obs := s.inspectDeploymentHealth(ctx, dctx).observation
	if obs == nil {
		return reconcile.Observed{Status: "unknown", Freshness: "unknown"}
	}
	out := reconcile.Observed{ReleaseDigest: health.NormalizeDigest(obs.GetObservedReleaseDigest()), ConfigurationDigest: obs.GetObservedConfigurationDigest(), ProducerRef: obs.GetProducerRef()}
	if at := obs.GetObservedAt(); at != nil {
		out.ObservedAt = at.AsTime()
	}
	out.Status = strings.ToLower(strings.TrimPrefix(obs.GetStatus().String(), "HEALTH_STATUS_"))
	out.Freshness = strings.ToLower(strings.TrimPrefix(obs.GetFreshness().String(), "FRESHNESS_"))
	if out.Status == "healthy" || out.Status == "degraded" {
		running := true
		out.WorkloadRunning = &running
	}
	return out
}

func (s *Server) desiredFor(dctx *DeploymentContext) (reconcile.Desired, *domain.ClosureSupervision) {
	dep := dctx.Deployment
	desired := reconcile.Desired{DeploymentID: dep.ID, Revision: dep.Fence, State: dep.DesiredState.Normalized(), RecordedAt: dep.UpdatedAt}
	if dep.BundleSHA256 != nil && strings.TrimSpace(*dep.BundleSHA256) != "" {
		desired.ReleaseDigest = "sha256:" + strings.TrimPrefix(strings.TrimSpace(*dep.BundleSHA256), "sha256:")
	}
	if digest, err := vps.ConfigurationDigest(dctx.Manifest); err == nil {
		desired.ConfigurationDigest = digest
	}
	var supervision *domain.ClosureSupervision
	if closure, _ := s.deploymentClosure(context.Background(), dctx); closure != nil {
		desired.ClosureDigest = closure.Digest
		for _, component := range closure.ComponentsOfKind(domain.ClosureKindScenario) {
			if component.Supervision != nil && (component.ID == dctx.Manifest.Scenario.ID || supervision == nil) {
				supervision = component.Supervision
			}
		}
	}
	return desired, supervision
}

// DesiredStateResponse is the desired-state contract consumers (including
// vrooli-autoheal) read before touching a cloud workload.
type DesiredStateResponse struct {
	SchemaVersion   string                 `json:"schema_version"`
	DeploymentID    string                 `json:"deployment_id"`
	DesiredState    domain.DesiredState    `json:"desired_state"`
	DesiredRevision uint64                 `json:"desired_revision"`
	ReleaseDigest   string                 `json:"release_digest,omitempty"`
	RecordedAt      time.Time              `json:"recorded_at"`
	RebootPolicy    reconcile.RebootPolicy `json:"reboot_policy"`
	// ObservationMayRestart is the contract in one bit: false means no
	// observer (health, autoheal, drift) may start this workload.
	ObservationMayRestart bool `json:"observation_may_restart"`
}

// GET /api/v1/deployments/{id}/desired-state
func (s *Server) handleGetDesiredState(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	desired, supervision := s.desiredFor(dctx)
	reboot := reconcile.Reboot(desired.State, supervision)
	httputil.WriteJSON(w, http.StatusOK, DesiredStateResponse{
		SchemaVersion: reconcile.SchemaVersion, DeploymentID: dctx.ID, DesiredState: desired.State, DesiredRevision: desired.Revision,
		ReleaseDigest: desired.ReleaseDigest, RecordedAt: desired.RecordedAt, RebootPolicy: reboot,
		ObservationMayRestart: desired.State == domain.DesiredRunning && reboot.AutoRestart,
	})
}

// PUT /api/v1/deployments/{id}/desired-state {desired_state}
func (s *Server) handleSetDesiredState(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	var body struct {
		DesiredState domain.DesiredState `json:"desired_state"`
	}
	if !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	switch body.DesiredState {
	case domain.DesiredRunning, domain.DesiredStopped:
	default:
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "desired_state must be running or stopped; retirement goes through /retire/plan").WithDetail("desired_state", string(body.DesiredState)))
		return
	}
	if err := s.repo.UpdateDesiredState(r.Context(), dctx.ID, body.DesiredState); err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to record desired state", err))
		return
	}
	s.handleGetDesiredState(w, r)
}

// ReconcileRequest is the reconcile body: without a plan digest the call
// observes and proposes; with the reviewed digest of the proposed correction
// it admits the correction as a durable operation.
type ReconcileRequest struct {
	PlanDigest string `json:"plan_digest,omitempty"`
	RequestKey string `json:"request_key,omitempty"`
}

// ReconcileResponse is the drift report plus the proposed correction plan.
type ReconcileResponse struct {
	SchemaVersion string                `json:"schema_version"`
	Report        reconcile.DriftReport `json:"report"`
	Target        *TargetState          `json:"target,omitempty"`
	TargetError   string                `json:"target_error,omitempty"`
	// Correction is the compiled plan the report proposes (nil when nothing
	// may be applied). Apply it by resubmitting plan_digest.
	Correction *CompiledPlan  `json:"correction,omitempty"`
	Operation  map[string]any `json:"operation,omitempty"`
}

// POST /api/v1/deployments/{id}/reconcile
func (s *Server) handleReconcileDeployment(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	var req ReconcileRequest
	if r.ContentLength != 0 && !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	desired, supervision := s.desiredFor(dctx)
	observed := s.observeHealth(r.Context(), dctx)
	resp := ReconcileResponse{SchemaVersion: reconcile.SchemaVersion}
	if state, err := s.observeTarget(r.Context(), dctx); err != nil {
		resp.TargetError = err.Error()
	} else {
		resp.Target = state
		observed.ActivationInterrupted = state.ActivationInterrupted
	}
	resp.Report = reconcile.Detect(desired, observed, supervision, time.Now().UTC())
	if resp.Report.Correction == nil || resp.Report.Correction.Scope == "" {
		httputil.WriteJSON(w, http.StatusOK, resp)
		return
	}
	compiled, cerr := s.compileDeploymentPlan(r.Context(), dctx.ID, resp.Report.Correction.Scope)
	if cerr != nil {
		apierrors.Write(w, cerr)
		return
	}
	resp.Correction = compiled
	if strings.TrimSpace(req.PlanDigest) == "" {
		httputil.WriteJSON(w, http.StatusOK, resp)
		return
	}
	if strings.TrimSpace(req.PlanDigest) != compiled.PlanDigest {
		apierrors.Write(w, execplan.DigestMismatchError(req.PlanDigest, compiled.PlanDigest))
		return
	}
	requestKey := strings.TrimSpace(req.RequestKey)
	if requestKey == "" {
		requestKey = "reconcile:" + dctx.ID + ":" + compiled.PlanDigest
	}
	op, created, aerr := s.admitAndSubmit(r, dctx.ID, requestKey, compiled, operations.ExecuteOptions{})
	if aerr != nil {
		apierrors.Write(w, aerr)
		return
	}
	resp.Operation = map[string]any{"operation_id": op.ID, "plan_digest": op.PlanDigest, "state": op.State, "replayed": !created, "wait": "/api/v1/operations/" + op.ID + "/wait"}
	httputil.WriteJSON(w, http.StatusAccepted, resp)
}

// RetireRequest carries the explicit data disposition.
type RetireRequest struct {
	RetentionPolicy string `json:"retention_policy,omitempty"`
	PlanDigest      string `json:"plan_digest,omitempty"`
	RequestKey      string `json:"request_key,omitempty"`
}

// RetireResponse is the retained/deleted listing plus the executable plan.
type RetireResponse struct {
	SchemaVersion string                   `json:"schema_version"`
	Retirement    reconcile.RetirementPlan `json:"retirement"`
	Plan          *CompiledPlan            `json:"plan,omitempty"`
	PlanError     *apierrors.Error         `json:"plan_error,omitempty"`
	Target        *TargetState             `json:"target,omitempty"`
	TargetError   string                   `json:"target_error,omitempty"`
}

func (s *Server) retirementFor(r *http.Request, dctx *DeploymentContext, policy string) RetireResponse {
	resp := RetireResponse{SchemaVersion: reconcile.SchemaVersion}
	in := reconcile.RetirementInputs{DeploymentID: dctx.ID, Domain: strings.TrimSpace(dctx.Manifest.Edge.Domain), Scenarios: []string{dctx.Manifest.Scenario.ID}, RetentionPolicy: policy}
	if dctx.Manifest.Secrets != nil {
		for _, secret := range dctx.Manifest.Secrets.BundleSecrets {
			in.CredentialRefs = append(in.CredentialRefs, secret.ID)
		}
	}
	if closure, _ := s.deploymentClosure(r.Context(), dctx); closure != nil {
		for _, data := range closure.PersistentData {
			in.DataBindings = append(in.DataBindings, data.ID)
		}
	}
	if dctx.Manifest.Target.VPS != nil {
		in.LegacyData = append(in.LegacyData, dctx.Manifest.Target.VPS.PreservePaths...)
	}
	for _, rp := range s.recoveryPointsFor(r.Context(), dctx.ID) {
		in.RecoveryPoints = append(in.RecoveryPoints, rp.ID)
		in.RecoveryPointReleases = append(in.RecoveryPointReleases, strings.TrimPrefix(rp.ReleaseDigest, "sha256:"))
	}
	if state, err := s.observeTarget(r.Context(), dctx); err != nil {
		resp.TargetError = err.Error()
	} else {
		resp.Target = state
		in.ActiveRelease, in.PreviousRelease, in.Releases = state.ActiveRelease, state.PreviousRelease, state.Releases
	}
	resp.Retirement = reconcile.Retirement(in)
	compiled, cerr := s.compileDeploymentPlanWith(r.Context(), dctx.ID, execplan.ScopeRetire, execplan.Policy{RetentionPolicy: policy})
	if cerr != nil {
		resp.PlanError = cerr
	} else {
		resp.Plan = compiled
	}
	return resp
}

// POST /api/v1/deployments/{id}/retire/plan {retention_policy}
func (s *Server) handleRetirePlan(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	var req RetireRequest
	if r.ContentLength != 0 && !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s.retirementFor(r, dctx, strings.TrimSpace(req.RetentionPolicy)))
}

// POST /api/v1/deployments/{id}/retire/apply {retention_policy, plan_digest, request_key}
func (s *Server) handleRetireApply(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	var req RetireRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.PlanDigest) == "" || strings.TrimSpace(req.RequestKey) == "" {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "plan_digest and request_key are required: preview the retirement first"))
		return
	}
	resp := s.retirementFor(r, dctx, strings.TrimSpace(req.RetentionPolicy))
	if resp.PlanError != nil {
		apierrors.Write(w, resp.PlanError)
		return
	}
	if resp.Retirement.Outcome == reconcile.Blocked {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "Retirement is blocked: "+strings.Join(resp.Retirement.Blocked, "; ")).WithDetail("blocked", resp.Retirement.Blocked))
		return
	}
	if strings.TrimSpace(req.PlanDigest) != resp.Plan.PlanDigest {
		apierrors.Write(w, execplan.DigestMismatchError(req.PlanDigest, resp.Plan.PlanDigest))
		return
	}
	op, created, aerr := s.admitAndSubmit(r, dctx.ID, req.RequestKey, resp.Plan, operations.ExecuteOptions{})
	if aerr != nil {
		apierrors.Write(w, aerr)
		return
	}
	if created {
		if err := s.repo.UpdateDesiredState(r.Context(), dctx.ID, domain.DesiredRetired); err != nil {
			s.log("failed to record retired desired state", map[string]interface{}{"deployment_id": dctx.ID, "error": err.Error()})
		}
	}
	writeOperationAccepted(w, dctx.Deployment, op, !created, "Retirement admitted. Wait on /operations/{id}/wait.")
}

// DataBindingsInventoryResponse compares the target's mutable directories
// with the declared and recorded bindings. It is read-only.
type DataBindingsInventoryResponse struct {
	SchemaVersion string                           `json:"schema_version"`
	DeploymentID  string                           `json:"deployment_id"`
	Declared      []domain.PersistentDataMapping   `json:"declared_bindings"`
	Recorded      []domain.PersistentDataMapping   `json:"recorded_mappings"`
	Inventory     []domain.PersistentDataInventory `json:"inventory"`
	Unmapped      []domain.PersistentDataInventory `json:"unmapped"`
}

// declaredMappings lists the closure's filesystem bindings as mappings.
func (s *Server) declaredMappings(ctx context.Context, dctx *DeploymentContext) []domain.PersistentDataMapping {
	var out []domain.PersistentDataMapping
	if closure, _ := s.deploymentClosure(ctx, dctx); closure != nil {
		for _, data := range closure.PersistentData {
			kind, locator, ok := strings.Cut(strings.TrimSpace(data.Binding), ":")
			if !ok || kind != "dir" {
				continue
			}
			out = append(out, domain.PersistentDataMapping{BindingID: data.ID, Scenario: data.Owner, Path: strings.Trim(locator, "/")})
		}
	}
	return out
}

func recordedMappings(dep *domain.Deployment) []domain.PersistentDataMapping {
	if dep == nil || !dep.PersistentData.Valid {
		return nil
	}
	var doc domain.PersistentDataBindings
	if err := json.Unmarshal(dep.PersistentData.Data, &doc); err != nil {
		return nil
	}
	return doc.Mappings
}

// POST /api/v1/deployments/{id}/data-bindings/inventory
func (s *Server) handleDataBindingsInventory(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	if s.reach == nil {
		apierrors.Write(w, apierrors.New(apierrors.CodeReachUnavailable, "target reach is not configured"))
		return
	}
	resp := DataBindingsInventoryResponse{SchemaVersion: reconcile.SchemaVersion, DeploymentID: dctx.ID, Declared: s.declaredMappings(r.Context(), dctx), Recorded: recordedMappings(dctx.Deployment), Inventory: []domain.PersistentDataInventory{}, Unmapped: []domain.PersistentDataInventory{}}
	known := map[string][]map[string]string{}
	for _, m := range append(append([]domain.PersistentDataMapping{}, resp.Declared...), resp.Recorded...) {
		known[m.Scenario] = append(known[m.Scenario], map[string]string{"id": m.BindingID, "path": m.Path})
	}
	scenarios := append([]string{}, dctx.Manifest.Bundle.Scenarios...)
	if len(scenarios) == 0 {
		scenarios = []string{dctx.Manifest.Scenario.ID}
	}
	sort.Strings(scenarios)
	rt := s.executionRuntime(dctx.Manifest, dctx.ID, "")
	for _, scenario := range scenarios {
		raw, _ := json.Marshal(known[scenario])
		encoded := vps.JSONArgPrefix + base64.RawURLEncoding.EncodeToString(raw)
		res, err := rt.Reach.Exec(r.Context(), rt.Target, reach.Command{Verb: "cloud-target data inventory", Args: []string{"--deployment", dctx.ID, "--workdir", dctx.Workdir, "--scenario", scenario, "--bindings", encoded, "--json"}, RequiredScope: "vrooli:read", Timeout: 30 * time.Second})
		if err != nil {
			apierrors.Write(w, reach.APIError(err))
			return
		}
		var report struct {
			Entries []struct {
				Path    string `json:"path"`
				Bytes   int64  `json:"bytes"`
				Files   int64  `json:"files"`
				Covered bool   `json:"covered"`
			} `json:"entries"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &report); err != nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeReachProtocolUnsupported, "data inventory reply is not JSON").WithDetail("scenario", scenario))
			return
		}
		for _, entry := range report.Entries {
			item := domain.PersistentDataInventory{Scenario: scenario, Path: entry.Path, Bytes: entry.Bytes, Files: entry.Files, Mapped: entry.Covered}
			resp.Inventory = append(resp.Inventory, item)
			if !entry.Covered {
				resp.Unmapped = append(resp.Unmapped, item)
			}
		}
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// DataBindingsAdoptRequest records the legacy conversion mapping.
type DataBindingsAdoptRequest struct {
	Mappings []domain.PersistentDataMapping `json:"mappings"`
}

// POST /api/v1/deployments/{id}/data-bindings/adopt
func (s *Server) handleDataBindingsAdopt(w http.ResponseWriter, r *http.Request) {
	dctx := s.FetchDeploymentContext(w, r)
	if dctx == nil {
		return
	}
	var req DataBindingsAdoptRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	if len(req.Mappings) == 0 {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "mappings are required"))
		return
	}
	declared := map[string]bool{}
	if closure, _ := s.deploymentClosure(r.Context(), dctx); closure != nil {
		for _, data := range closure.PersistentData {
			declared[data.ID] = true
		}
	}
	seen := map[string]bool{}
	for i := range req.Mappings {
		m := &req.Mappings[i]
		m.BindingID, m.Scenario, m.Path = strings.TrimSpace(m.BindingID), strings.TrimSpace(m.Scenario), strings.Trim(strings.TrimSpace(m.Path), "/")
		if m.BindingID == "" || m.Scenario == "" || m.Path == "" || strings.Contains(m.Path, "..") || strings.ContainsAny(m.BindingID+m.Scenario+m.Path, " ;|&$`<>'\"\\") {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "each mapping needs binding_id, scenario and a relative path without traversal or shell syntax").WithDetail("mapping", m))
			return
		}
		if !declared[m.BindingID] {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "binding_id is not a declared persistent-data binding of the closure; declare it before adopting data into it").WithDetail("binding_id", m.BindingID))
			return
		}
		if seen[m.BindingID] {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "binding_id mapped twice").WithDetail("binding_id", m.BindingID))
			return
		}
		seen[m.BindingID] = true
	}
	doc := domain.PersistentDataBindings{SchemaVersion: 1, Mappings: req.Mappings, RecordedAt: time.Now().UTC()}
	if err := s.repo.UpdatePersistentData(r.Context(), dctx.ID, doc); err != nil {
		apierrors.Write(w, apierrors.Internal("Failed to record data bindings", err))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": reconcile.SchemaVersion, "deployment_id": dctx.ID, "persistent_data": doc, "note": "mapping recorded; no data moved. The next release.activate binds these paths and adopts the directories by rename; unmapped directories are carried and never deleted."})
}

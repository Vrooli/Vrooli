package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"test-genie/agentmanager"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/remediation"
	"test-genie/internal/runmanager"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type remediationJobRequest struct {
	SourceExecutionID string   `json:"sourceExecutionId"`
	FindingIDs        []string `json:"findingIds"`
	RequirementIDs    []string `json:"requirementIds"`
	RoleRef           string   `json:"roleRef"`
	AdditionalContext string   `json:"additionalContext"`
}

func (s *Server) handleGetRemediationPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := s.remediationPlan(r.Context(), mux.Vars(r)["name"], mux.Vars(r)["executionID"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleCreateRemediationJob(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil || s.remediationLauncher == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation service is unavailable")
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request remediationJobRequest
	if err := decoder.Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "unsupported remediation payload: use sourceExecutionId, findingIds, requirementIds, roleRef, and additionalContext")
		return
	}
	if strings.TrimSpace(request.RoleRef) == "" {
		s.writeError(w, http.StatusBadRequest, "roleRef is required")
		return
	}
	plan, err := s.remediationPlan(r.Context(), mux.Vars(r)["name"], request.SourceExecutionID)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	job, err := s.remediationService.Create(r.Context(), plan, request.FindingIDs, request.RequirementIDs, request.AdditionalContext)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	job, err = s.remediationService.PrepareLaunch(r.Context(), job.ID, request.RoleRef)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	job, err = s.launchRemediation(r.Context(), job)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to launch remediation through agent-manager")
		return
	}
	s.writeJSON(w, http.StatusCreated, job)
}

// launchRemediation is replay-safe: PrepareLaunch persists the stable intent
// first, and the adapter uses that key/tag to reconnect to an already-created
// Agent Manager run after a crash or an interrupted HTTP response.
func (s *Server) launchRemediation(ctx context.Context, job remediation.Job) (remediation.Job, error) {
	attribution, err := s.remediationLauncher.Launch(ctx, job, job.Attribution.RoleRef)
	if err != nil {
		_, _ = s.remediationService.RecordLaunchFailure(context.Background(), job.ID, err.Error())
		return remediation.Job{}, err
	}
	return s.remediationService.MarkRunning(ctx, job.ID, attribution)
}

func (s *Server) handleListRemediationJobs(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation service is unavailable")
		return
	}
	jobs, err := s.remediationService.List(r.Context(), mux.Vars(r)["name"], 50)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": jobs, "count": len(jobs)})
}

func (s *Server) handleGetRemediationJob(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation service is unavailable")
		return
	}
	job, err := s.remediationService.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	if job.Scenario != mux.Vars(r)["name"] {
		s.writeError(w, http.StatusNotFound, "remediation job not found")
		return
	}
	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCancelRemediationJob(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation service is unavailable")
		return
	}
	job, err := s.remediationService.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	if job.Scenario != mux.Vars(r)["name"] {
		s.writeError(w, http.StatusNotFound, "remediation job not found")
		return
	}
	if s.remediationLauncher != nil {
		if err := s.remediationLauncher.Cancel(r.Context(), job); err != nil {
			s.writeError(w, http.StatusBadGateway, "failed to cancel agent-manager run")
			return
		}
	}
	job, err = s.remediationService.Cancel(r.Context(), job.ID)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleRecoverRemediationJob(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil || s.remediationLauncher == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation recovery is unavailable")
		return
	}
	job, err := s.remediationService.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	if job.Scenario != mux.Vars(r)["name"] {
		s.writeError(w, http.StatusNotFound, "remediation job not found")
		return
	}
	if job.Status != remediation.JobStatusLaunchPending {
		s.writeJSON(w, http.StatusOK, job)
		return
	}
	job, err = s.launchRemediation(r.Context(), job)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to recover remediation through agent-manager")
		return
	}
	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleRetryRemediationJob(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil || s.remediationLauncher == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation retry is unavailable")
		return
	}
	job, err := s.remediationService.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	if job.Scenario != mux.Vars(r)["name"] {
		s.writeError(w, http.StatusNotFound, "remediation job not found")
		return
	}
	job, err = s.remediationService.RetryLaunch(r.Context(), job.ID)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	job, err = s.launchRemediation(r.Context(), job)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to retry remediation through agent-manager")
		return
	}
	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleRefreshRemediationAgent(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation service is unavailable")
		return
	}
	job, err := s.remediationService.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	if job.Scenario != mux.Vars(r)["name"] {
		s.writeError(w, http.StatusNotFound, "remediation job not found")
		return
	}
	if job.Status != remediation.JobStatusRunning {
		s.writeJSON(w, http.StatusOK, job)
		return
	}
	run, err := s.agentService.GetRun(r.Context(), job.Attribution.RunID)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "failed to load remediation agent status")
		return
	}
	if run == nil {
		s.writeError(w, http.StatusNotFound, "agent-manager run not found")
		return
	}
	switch agentmanager.MapRunStatus(run.Status) {
	case "completed":
		job, err = s.remediationService.MarkAgentCompleted(r.Context(), job.ID, run.GetSummary().GetDescription())
	case "failed", "stopped":
		job, err = s.remediationService.Fail(r.Context(), job.ID, run.GetErrorMsg())
	}
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleVerifyRemediationJob(w http.ResponseWriter, r *http.Request) {
	if s.remediationService == nil || s.runManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "remediation verification is unavailable")
		return
	}
	job, err := s.remediationService.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	if job.Scenario != mux.Vars(r)["name"] {
		s.writeError(w, http.StatusNotFound, "remediation job not found")
		return
	}
	job, err = s.remediationService.ReserveVerification(r.Context(), job.ID)
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	phases := selectedPhaseNames(job)
	started, err := s.runManager.Start(runmanager.StartOptions{Input: execution.SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{
		ScenarioName: job.Scenario,
		Phases:       phases,
	}}})
	if err != nil {
		_, _ = s.remediationService.ReleaseVerificationReservation(context.Background(), job.ID)
		s.writeError(w, http.StatusConflict, "unable to start server-owned verification: "+err.Error())
		return
	}
	job, err = s.remediationService.SetVerificationRun(r.Context(), job.ID, remediation.Verification{RunID: started.RunID})
	if err != nil {
		s.writeRemediationError(w, err)
		return
	}
	go s.completeRemediationVerification(job, started.RunID)
	s.writeJSON(w, http.StatusAccepted, job)
}

func selectedPhaseNames(job remediation.Job) []string {
	selected := make(map[string]struct{}, len(job.SelectedFindingIDs))
	for _, id := range job.SelectedFindingIDs {
		selected[id] = struct{}{}
	}
	set := map[string]struct{}{}
	for _, finding := range job.Source.Findings {
		if _, ok := selected[finding.StableID]; ok && finding.Phase != "" {
			set[finding.Phase] = struct{}{}
		}
	}
	phases := make([]string, 0, len(set))
	for phase := range set {
		phases = append(phases, phase)
	}
	sort.Strings(phases)
	return phases
}

func (s *Server) completeRemediationVerification(job remediation.Job, runID string) {
	status, err := s.runManager.Wait(context.Background(), job.Scenario, runID)
	if err != nil || status.Result == nil {
		_, _ = s.remediationService.CompleteVerification(context.Background(), job.ID, remediation.Verification{RunID: runID}, remediation.ComparePlan(job.Source, job.SelectedFindingIDs, nil, false, nil), remediation.CompareRequirements(job.Source, job.SelectedRequirementIDs, nil, false), "verification execution did not produce findings")
		return
	}
	result := status.Result
	dir := filepath.Join(s.scenarios.ScenarioRoot(), job.Scenario)
	snapshot, snapshotErr := sharedruns.ReadDescriptorSnapshot(dir, result.RunID)
	plan := remediation.BuildPlan(remediation.EvidenceFromExecution(result.ExecutionID.String(), result, &snapshot, snapshotErr))
	if _, err := os.Stat(sharedartifacts.RunFindingsArtifactPath(dir, result.RunID)); err != nil {
		plan.Degraded, plan.DegradedReasons = true, append(plan.DegradedReasons, "combined findings artifact unavailable: "+err.Error())
	}
	delta := remediation.ComparePlan(job.Source, job.SelectedFindingIDs, plan.Findings, !plan.Degraded, result.PlannedPhases)
	requirementDelta := remediation.CompareRequirements(job.Source, job.SelectedRequirementIDs, nil, false)
	if requirementPlan, requirementErr := s.remediationPlan(context.Background(), job.Scenario, result.ExecutionID.String()); requirementErr == nil {
		requirementDelta = remediation.CompareRequirements(job.Source, job.SelectedRequirementIDs, requirementPlan.Requirements, !requirementPlan.Degraded)
	} else {
		plan.DegradedReasons = append(plan.DegradedReasons, "verification requirements evidence unavailable: "+requirementErr.Error())
	}
	_, _ = s.remediationService.CompleteVerification(context.Background(), job.ID, remediation.Verification{ExecutionID: result.ExecutionID.String(), RunID: result.RunID}, delta, requirementDelta, strings.Join(plan.DegradedReasons, "; "))
}

func (s *Server) remediationPlan(ctx context.Context, scenario, executionID string) (remediation.Plan, error) {
	if s.executionHistory == nil {
		return remediation.Plan{}, fmt.Errorf("execution history is unavailable")
	}
	if _, err := s.scenarios.GetSummary(ctx, scenario); err != nil {
		return remediation.Plan{}, fmt.Errorf("scenario not found: %w", err)
	}
	id, err := uuid.Parse(strings.TrimSpace(executionID))
	if err != nil {
		return remediation.Plan{}, fmt.Errorf("%w: sourceExecutionId must be a UUID", remediation.ErrInvalidSelector)
	}
	result, err := s.executionHistory.Get(ctx, id)
	if err != nil {
		return remediation.Plan{}, err
	}
	if result.ScenarioName != scenario {
		return remediation.Plan{}, fmt.Errorf("%w: source execution belongs to another scenario", remediation.ErrInvalidSelector)
	}
	dir := filepath.Join(s.scenarios.ScenarioRoot(), scenario)
	snapshot, snapshotErr := sharedruns.ReadDescriptorSnapshot(dir, result.RunID)
	evidence := remediation.EvidenceFromExecution(id.String(), result, &snapshot, snapshotErr)
	if _, err := os.Stat(sharedartifacts.RunFindingsArtifactPath(dir, result.RunID)); err != nil {
		evidence.DegradedReasons = append(evidence.DegradedReasons, "combined findings artifact unavailable: "+err.Error())
	}
	plan := remediation.BuildPlan(evidence)
	modules, _, _, requirementsErr := s.loadRequirementModules(dir)
	if requirementsErr != nil {
		plan.Degraded = true
		plan.DegradedReasons = append(plan.DegradedReasons, "requirements evidence unavailable: "+requirementsErr.Error())
		return plan, nil
	}
	for _, module := range modules {
		for _, requirement := range module.requirements {
			validations := make([]string, 0, len(requirement.Validations))
			for _, validation := range requirement.Validations {
				validations = append(validations, validation.Type+":"+validation.Ref+":"+validation.LiveStatus)
			}
			plan.Requirements = append(plan.Requirements, remediation.RequirementEvidence{ID: requirement.ID, Title: requirement.Title, Description: requirement.Description, Status: requirement.Status, LiveStatus: requirement.LiveStatus, Criticality: requirement.Criticality, Validations: validations})
		}
	}
	return plan, nil
}

func (s *Server) writeRemediationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, remediation.ErrNotFound):
		s.writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, remediation.ErrActiveJob):
		s.writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, remediation.ErrInvalidSelector), errors.Is(err, remediation.ErrInvalidState):
		s.writeError(w, http.StatusBadRequest, err.Error())
	default:
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

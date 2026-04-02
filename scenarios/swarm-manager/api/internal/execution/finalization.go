package execution

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
)

// FinalizationStatus represents the lifecycle state of a finalization step
// (restart, health check, review, or the finalization as a whole).
type FinalizationStatus string

const (
	FinalizationStatusPending   FinalizationStatus = "pending"
	FinalizationStatusRunning   FinalizationStatus = "running"
	FinalizationStatusCompleted FinalizationStatus = "completed"
	FinalizationStatusSkipped   FinalizationStatus = "skipped"
	FinalizationStatusFailed    FinalizationStatus = "failed"
)

const (
	FinalizationPhaseScopeDetection    = "scope_detection"
	FinalizationPhaseRestarting        = "restarting"
	FinalizationPhaseHealthCheck       = "health_check"
	FinalizationPhaseReviewing         = "reviewing"
	FinalizationPhaseEvidenceGathering = "evidence_gathering"
	FinalizationPhaseCompleted         = "completed"
	FinalizationPhaseSkipped           = "skipped"
	FinalizationPhaseFailed            = "failed"
)

const (
	FinalizationScopeNone                        = "none"
	FinalizationScopeSandboxDiff                 = "sandbox_diff"
	FinalizationScopeAcceptanceAllow             = "acceptance_allow"
	FinalizationScopeSandboxDiffPlusAcceptance   = "sandbox_diff_plus_acceptance_allow"
	FinalizationAggregateReady                   = "ready"
	FinalizationAggregateReadyWithNotes          = "ready_with_notes"
	FinalizationAggregateNeedsWork               = "needs_work"
	FinalizationAggregateNotAssessable           = "not_assessable"
	FinalizationAggregateSkipped                 = "skipped"
	finalizationWarningScopeDiffUnavailable      = "scope_diff_unavailable"
	finalizationWarningScopeSharedPathBroadening = "scope_shared_path_broadening"
	finalizationWarningRestartRetry              = "restart_retry"
	finalizationWarningHealthRetry               = "health_retry"
	finalizationWarningHealthSchemaInvalid       = "health_schema_invalid"
	finalizationWarningHealthChecksMissing       = "health_checks_missing"
	finalizationWarningReviewSkipped             = "review_skipped"
	finalizationWarningFinalizationInfra         = "finalization_infrastructure"
	finalizationWarningReviewAgentFailed         = "review_agent_failed"
)

// ScenarioLifecycle restarts affected scenarios after execution completion.
type ScenarioLifecycle interface {
	Restart(ctx context.Context, name string) error
}

// ScenarioHealthChecker probes scenario health using the standard Vrooli status
// contract.
type ScenarioHealthChecker interface {
	Check(ctx context.Context, name string) (ScenarioHealthSnapshot, error)
}

// RunDiffer resolves changed files for sandboxed agent-manager runs.
type RunDiffer interface {
	GetRunDiff(ctx context.Context, runID string) (agentmanager.RunDiff, error)
}

// ScenarioHealthSnapshot is the execution package's neutral view of scenario
// health.
type ScenarioHealthSnapshot struct {
	ScenarioStatus string
	HealthStatus   string
	SchemaValid    bool
	Healthy        bool
	Details        string
	CheckedAt      string
}

// Finalization captures the full post-run orchestration state for an
// execution.
type Finalization struct {
	Eligible                bool                   `json:"eligible"`
	Status                  FinalizationStatus     `json:"status,omitempty"`
	Phase                   string                 `json:"phase,omitempty"`
	ScopeSource             string                 `json:"scope_source,omitempty"`
	SkipReason              string                 `json:"skip_reason,omitempty"`
	StartedAt               string                 `json:"started_at,omitempty"`
	CompletedAt             string                 `json:"completed_at,omitempty"`
	Warnings                []FinalizationWarning  `json:"warnings,omitempty"`
	AffectedScenarios       []string               `json:"affected_scenarios,omitempty"`
	AggregateClassification string                 `json:"aggregate_classification,omitempty"`
	AggregateSummary        string                 `json:"aggregate_summary,omitempty"`
	Scenarios               []ScenarioFinalization `json:"scenarios,omitempty"`
}

// FinalizationWarning captures a non-fatal issue encountered while running the
// finalization flow.
type FinalizationWarning struct {
	Code         string `json:"code,omitempty"`
	ScenarioName string `json:"scenario_name,omitempty"`
	Message      string `json:"message,omitempty"`
	Retryable    bool   `json:"retryable,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// ScenarioFinalization captures the restart, health, and review work for one
// affected scenario.
type ScenarioFinalization struct {
	ScenarioName string             `json:"scenario_name,omitempty"`
	ChangedPaths []string           `json:"changed_paths,omitempty"`
	Restart      RestartResult      `json:"restart,omitempty"`
	Health       HealthCheckResult  `json:"health,omitempty"`
	Review       ScenarioReviewStep `json:"review,omitempty"`
}

// RestartResult captures restart attempts for one scenario.
type RestartResult struct {
	Status     FinalizationStatus `json:"status,omitempty"`
	Attempts   int                `json:"attempts,omitempty"`
	LastError  string             `json:"last_error,omitempty"`
	StartedAt  string             `json:"started_at,omitempty"`
	FinishedAt string             `json:"finished_at,omitempty"`
}

// HealthCheckResult captures the structured health outcome for one scenario.
type HealthCheckResult struct {
	Status         FinalizationStatus `json:"status,omitempty"`
	ScenarioStatus string             `json:"scenario_status,omitempty"`
	HealthStatus   string             `json:"health_status,omitempty"`
	SchemaValid    bool               `json:"schema_valid"`
	Details        string             `json:"details,omitempty"`
	CheckedAt      string             `json:"checked_at,omitempty"`
}

// ScenarioReviewStep captures one scenario's review job and result.
type ScenarioReviewStep struct {
	Status     FinalizationStatus `json:"status,omitempty"`
	JobID      string             `json:"job_id,omitempty"`
	SkipReason string             `json:"skip_reason,omitempty"`
	Result     *ReviewResult      `json:"result,omitempty"`
}

type finalizationScope struct {
	source                 string
	affectedScenarios      []string
	changedPathsByScenario map[string][]string
	sandboxID              string
	warnings               []FinalizationWarning
}

func isFinalizationEligible(record Record) bool {
	if record.ArchiveContext != nil {
		return false
	}

	runType := strings.ToLower(strings.TrimSpace(record.effectiveRunType()))
	switch runType {
	case "", "process", "fixup", "followup", "custom":
		return true
	default:
		return false
	}
}

func (r Record) effectiveRunType() string {
	if r.PromptTrace != nil && strings.TrimSpace(r.PromptTrace.Purpose) != "" {
		return strings.ToLower(strings.TrimSpace(r.PromptTrace.Purpose))
	}
	switch strings.ToLower(strings.TrimSpace(r.Operation)) {
	case "fixup", "followup", "custom":
		return strings.ToLower(strings.TrimSpace(r.Operation))
	default:
		return "process"
	}
}

func (s *Service) processFinalization(ctx context.Context, executionID string) error {
	if !s.beginFinalization(executionID) {
		return nil
	}
	defer s.endFinalization(executionID)

	record, item, err := s.loadExecutionForFinalization(executionID)
	if err != nil {
		return s.failFinalization(executionID, "", fmt.Sprintf("load execution for finalization: %v", err))
	}

	if !isFinalizationEligible(record) {
		return s.completeFinalizationSkipped(executionID, "execution type does not use post-run checks")
	}

	if err := s.markFinalizationPhase(executionID, FinalizationPhaseScopeDetection); err != nil {
		return err
	}

	scope, err := s.resolveFinalizationScope(ctx, record, item)
	if err != nil {
		return s.failFinalization(executionID, "", fmt.Sprintf("resolve affected scenarios: %v", err))
	}
	if len(scope.affectedScenarios) == 0 {
		skipReason := "no affected scenarios could be resolved from sandbox diff or acceptance_allow"
		if err := s.applyFinalizationScope(executionID, scope, skipReason); err != nil {
			return err
		}
		return s.completeFinalizationSkipped(executionID, skipReason)
	}
	if err := s.applyFinalizationScope(executionID, scope, ""); err != nil {
		return err
	}

	for _, scenarioName := range scope.affectedScenarios {
		if err := s.runScenarioRestartAndHealth(ctx, executionID, scenarioName); err != nil {
			return err
		}
	}

	if err := s.markFinalizationPhase(executionID, FinalizationPhaseReviewing); err != nil {
		return err
	}
	for _, scenarioName := range scope.affectedScenarios {
		if err := s.runScenarioReview(ctx, executionID, scenarioName, scope.sandboxID, item.AcceptanceAllow); err != nil {
			return err
		}
	}

	// Evidence gathering phase (optional, policy-gated). The review agent
	// spawns asynchronously; its failure is non-fatal to finalization.
	if s.isReviewAgentEnabled() {
		if err := s.markFinalizationPhase(executionID, FinalizationPhaseEvidenceGathering); err != nil {
			return err
		}
		if err := s.triggerReviewAgent(ctx, executionID, scope, item); err != nil {
			s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningReviewAgentFailed, "", err.Error(), false,
			))
		}
	}

	return s.finishFinalization(executionID)
}

func (s *Service) resolveFinalizationScope(ctx context.Context, record Record, item backlogItem) (finalizationScope, error) {
	scope := finalizationScope{
		source:                 FinalizationScopeNone,
		changedPathsByScenario: map[string][]string{},
	}

	acceptanceScenarios := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
	var sandboxDiff agentmanager.RunDiff
	if s.differ != nil && strings.TrimSpace(record.RunID) != "" {
		diff, err := s.differ.GetRunDiff(ctx, record.RunID)
		if err == nil {
			sandboxDiff = diff
			scope.sandboxID = strings.TrimSpace(diff.SandboxID)
		} else {
			scope.warnings = append(scope.warnings, newFinalizationWarning(
				finalizationWarningScopeDiffUnavailable,
				"",
				fmt.Sprintf("sandbox diff unavailable for %s: %v", record.RunID, err),
				false,
			))
		}
	}

	if len(sandboxDiff.Files) > 0 {
		paths := make([]string, 0, len(sandboxDiff.Files))
		for _, file := range sandboxDiff.Files {
			paths = append(paths, file.Path)
		}
		grouped := pathutil.GroupChangedPaths(paths)
		for scenarioName, changedPaths := range grouped.DirectScenarioPaths {
			scope.changedPathsByScenario[scenarioName] = append([]string(nil), changedPaths...)
		}

		directScenarios := mapKeysSorted(grouped.DirectScenarioPaths)
		switch {
		case len(grouped.SharedPaths) == 0 && len(directScenarios) > 0:
			scope.source = FinalizationScopeSandboxDiff
			scope.affectedScenarios = directScenarios
			return scope, nil
		case len(grouped.SharedPaths) > 0 && len(directScenarios) > 0:
			scope.source = FinalizationScopeSandboxDiffPlusAcceptance
			scope.affectedScenarios = unionSortedStrings(directScenarios, acceptanceScenarios)
			for _, scenarioName := range scope.affectedScenarios {
				if len(scope.changedPathsByScenario[scenarioName]) == 0 {
					scope.changedPathsByScenario[scenarioName] = append([]string(nil), grouped.SharedPaths...)
				}
			}
			scope.warnings = append(scope.warnings, newFinalizationWarning(
				finalizationWarningScopeSharedPathBroadening,
				"",
				fmt.Sprintf("shared repo changes required acceptance_allow broadening: %s", strings.Join(grouped.SharedPaths, ", ")),
				false,
			))
			return scope, nil
		case len(grouped.SharedPaths) > 0:
			scope.source = FinalizationScopeAcceptanceAllow
			scope.affectedScenarios = acceptanceScenarios
			for _, scenarioName := range scope.affectedScenarios {
				scope.changedPathsByScenario[scenarioName] = append([]string(nil), grouped.SharedPaths...)
			}
			scope.warnings = append(scope.warnings, newFinalizationWarning(
				finalizationWarningScopeSharedPathBroadening,
				"",
				fmt.Sprintf("sandbox diff only exposed shared paths; falling back to acceptance_allow: %s", strings.Join(grouped.SharedPaths, ", ")),
				false,
			))
			return scope, nil
		}
	}

	scope.source = FinalizationScopeAcceptanceAllow
	scope.affectedScenarios = acceptanceScenarios
	return scope, nil
}

func (s *Service) runScenarioRestartAndHealth(ctx context.Context, executionID string, scenarioName string) error {
	if err := s.markFinalizationPhase(executionID, FinalizationPhaseRestarting); err != nil {
		return err
	}

	if s.scenarioLifecycle == nil || s.scenarioHealth == nil {
		return s.failFinalization(executionID, scenarioName, "scenario restart/health seams are not configured")
	}

	for attempt := 1; attempt <= s.finalizationCfg.MaxRestartAttempts; attempt++ {
		restartStartedAt := nowRFC3339()
		if err := s.updateScenarioRestartState(executionID, scenarioName, RestartResult{
			Status:    FinalizationStatusRunning,
			Attempts:  attempt,
			StartedAt: restartStartedAt,
		}); err != nil {
			return err
		}

		restartErr := s.scenarioLifecycle.Restart(ctx, scenarioName)
		restartFinishedAt := nowRFC3339()
		if restartErr != nil {
			restartResult := RestartResult{
				Status:     FinalizationStatusFailed,
				Attempts:   attempt,
				LastError:  restartErr.Error(),
				StartedAt:  restartStartedAt,
				FinishedAt: restartFinishedAt,
			}
			if err := s.updateScenarioRestartState(executionID, scenarioName, restartResult); err != nil {
				return err
			}
			if attempt < s.finalizationCfg.MaxRestartAttempts {
				if err := s.appendFinalizationWarning(executionID, newFinalizationWarning(
					finalizationWarningRestartRetry,
					scenarioName,
					fmt.Sprintf("restart attempt %d failed: %v; retrying once", attempt, restartErr),
					true,
				)); err != nil {
					return err
				}
				continue
			}
			if err := s.updateScenarioHealthState(executionID, scenarioName, HealthCheckResult{
				Status:      FinalizationStatusFailed,
				SchemaValid: false,
				Details:     "restart command failed; health check skipped",
				CheckedAt:   restartFinishedAt,
			}); err != nil {
				return err
			}
			return nil
		}

		if err := s.updateScenarioRestartState(executionID, scenarioName, RestartResult{
			Status:     FinalizationStatusCompleted,
			Attempts:   attempt,
			StartedAt:  restartStartedAt,
			FinishedAt: restartFinishedAt,
		}); err != nil {
			return err
		}

		if err := s.markFinalizationPhase(executionID, FinalizationPhaseHealthCheck); err != nil {
			return err
		}
		healthSnapshot, healthErr := s.waitForScenarioHealth(ctx, scenarioName)
		if healthErr == nil {
			return s.updateScenarioHealthState(executionID, scenarioName, HealthCheckResult{
				Status:         FinalizationStatusCompleted,
				ScenarioStatus: healthSnapshot.ScenarioStatus,
				HealthStatus:   healthSnapshot.HealthStatus,
				SchemaValid:    healthSnapshot.SchemaValid,
				Details:        healthSnapshot.Details,
				CheckedAt:      healthSnapshot.CheckedAt,
			})
		}

		if err := s.updateScenarioHealthState(executionID, scenarioName, HealthCheckResult{
			Status:         FinalizationStatusFailed,
			ScenarioStatus: healthSnapshot.ScenarioStatus,
			HealthStatus:   healthSnapshot.HealthStatus,
			SchemaValid:    healthSnapshot.SchemaValid,
			Details:        healthSnapshot.Details,
			CheckedAt:      healthSnapshot.CheckedAt,
		}); err != nil {
			return err
		}

		warningCode := finalizationWarningHealthRetry
		if !healthSnapshot.SchemaValid {
			warningCode = finalizationWarningHealthSchemaInvalid
		} else if strings.Contains(strings.ToLower(healthSnapshot.Details), "no health checks") {
			warningCode = finalizationWarningHealthChecksMissing
		}
		if attempt < s.finalizationCfg.MaxRestartAttempts {
			if err := s.appendFinalizationWarning(executionID, newFinalizationWarning(
				warningCode,
				scenarioName,
				fmt.Sprintf("health check failed after restart attempt %d: %s; retrying once", attempt, healthSnapshot.Details),
				true,
			)); err != nil {
				return err
			}
			continue
		}
		return nil
	}

	return nil
}

func (s *Service) waitForScenarioHealth(ctx context.Context, scenarioName string) (ScenarioHealthSnapshot, error) {
	var last ScenarioHealthSnapshot
	deadline := time.Now().Add(s.finalizationCfg.HealthPollTimeout)
	for {
		snapshot, err := s.scenarioHealth.Check(ctx, scenarioName)
		if err == nil {
			last = snapshot
			if snapshot.Healthy {
				return snapshot, nil
			}
		} else {
			last = ScenarioHealthSnapshot{
				SchemaValid: false,
				Healthy:     false,
				Details:     err.Error(),
				CheckedAt:   nowRFC3339(),
			}
		}

		if time.Now().After(deadline) {
			if strings.TrimSpace(last.Details) == "" {
				last.Details = "scenario did not become healthy before timeout"
				last.CheckedAt = nowRFC3339()
			}
			return last, fmt.Errorf("%s", last.Details)
		}

		select {
		case <-ctx.Done():
			last.Details = ctx.Err().Error()
			last.CheckedAt = nowRFC3339()
			return last, ctx.Err()
		case <-time.After(s.finalizationCfg.HealthPollInterval):
		}
	}
}

func (s *Service) runScenarioReview(ctx context.Context, executionID, scenarioName, sandboxID string, acceptanceAllow []string) error {
	scenarioState, err := s.loadScenarioFinalization(executionID, scenarioName)
	if err != nil {
		return err
	}
	if scenarioState.Health.Status != FinalizationStatusCompleted {
		warning := newFinalizationWarning(
			finalizationWarningReviewSkipped,
			scenarioName,
			"review skipped because restart/health checks did not pass",
			false,
		)
		if err := s.appendFinalizationWarning(executionID, warning); err != nil {
			return err
		}
		return s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
			Status:     FinalizationStatusSkipped,
			SkipReason: "restart/health checks did not pass",
		})
	}

	if s.reviewClient == nil {
		return s.failFinalization(executionID, scenarioName, "review client is not configured")
	}

	if err := s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
		Status: FinalizationStatusRunning,
	}); err != nil {
		return err
	}

	req := ReviewRequest{
		ScenarioName: scenarioName,
		SandboxID:    sandboxID,
	}
	if len(scenarioState.ChangedPaths) > 0 {
		req.ExpectedPaths = append([]string(nil), scenarioState.ChangedPaths...)
	} else {
		req.ExpectedPaths = append([]string(nil), acceptanceAllow...)
	}
	if s.reviewThresholdsProvider != nil {
		if th, thErr := s.reviewThresholdsProvider.LoadReviewThresholds(); thErr == nil {
			req.Thresholds = th
		}
	}

	jobID, err := s.reviewClient.TriggerReview(ctx, req)
	if err != nil {
		return s.failFinalization(executionID, scenarioName, fmt.Sprintf("trigger review for %s: %v", scenarioName, err))
	}
	if err := s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
		Status: FinalizationStatusRunning,
		JobID:  jobID,
	}); err != nil {
		return err
	}

	deadline := time.Now().Add(s.finalizationCfg.ReviewPollTimeout)
	for {
		result, done, pollErr := s.reviewClient.PollReview(ctx, jobID)
		if pollErr != nil {
			return s.failFinalization(executionID, scenarioName, fmt.Sprintf("poll review for %s: %v", scenarioName, pollErr))
		}
		if done {
			return s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
				Status: FinalizationStatusCompleted,
				JobID:  jobID,
				Result: result,
			})
		}
		if time.Now().After(deadline) {
			return s.failFinalization(executionID, scenarioName, fmt.Sprintf("review for %s timed out after %s", scenarioName, s.finalizationCfg.ReviewPollTimeout))
		}
		select {
		case <-ctx.Done():
			return s.failFinalization(executionID, scenarioName, ctx.Err().Error())
		case <-time.After(s.finalizationCfg.ReviewPollInterval):
		}
	}
}

func (s *Service) finishFinalization(executionID string) error {
	s.mu.Lock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)

	finalization.Status = FinalizationStatusCompleted
	finalization.Phase = FinalizationPhaseCompleted
	finalization.CompletedAt = nowRFC3339()
	classification, summary, hasActionableFailure := summarizeFinalization(*finalization)
	finalization.AggregateClassification = classification
	finalization.AggregateSummary = summary
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	record.FinishedAt = nowRFC3339()

	item, loadErr := s.loadBacklogItemByRecord(record)
	autoSpawnFixup := false
	if loadErr == nil {
		switch {
		case hasActionableFailure:
			policy, _ := s.policyProvider.LoadPolicy()
			if policy.AutoFixup && record.FixupAttempt < policy.MaxFixupAttempts {
				record.Status = StatusNeedsFixup
				record.FailureReason = ""
				autoSpawnFixup = true
			} else {
				record.Status = StatusNeedsFixup
				record.FailureReason = ""
			}
			_ = s.updateBacklogStatus(item, backlogStatusFailed)
		default:
			record.Status = StatusCompleted
			record.FailureReason = ""
			_ = s.updateBacklogStatus(item, backlogStatusCompleted)
		}
	} else {
		record.Status = StatusNeedsFixup
		record.FailureReason = fmt.Sprintf("failed to load backlog item while finishing finalization: %v", loadErr)
	}

	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	s.dispatchStatusUpdate(*record)
	if autoSpawnFixup {
		s.spawnFixupRun(context.Background(), record, item)
	}
	return nil
}

func summarizeFinalization(finalization Finalization) (classification string, summary string, hasActionableFailure bool) {
	if finalization.Status == FinalizationStatusSkipped {
		return FinalizationAggregateSkipped, strings.TrimSpace(finalization.SkipReason), false
	}

	hasReadyWithNotes := false
	summaries := make([]string, 0, len(finalization.Scenarios))
	for _, scenario := range finalization.Scenarios {
		if scenario.Restart.Status != "" && scenario.Restart.Status != FinalizationStatusCompleted {
			hasActionableFailure = true
			summaries = append(summaries, fmt.Sprintf("%s restart failed", scenario.ScenarioName))
			continue
		}
		if scenario.Health.Status != "" && scenario.Health.Status != FinalizationStatusCompleted {
			hasActionableFailure = true
			if strings.TrimSpace(scenario.Health.Details) != "" {
				summaries = append(summaries, fmt.Sprintf("%s health failed: %s", scenario.ScenarioName, scenario.Health.Details))
			} else {
				summaries = append(summaries, fmt.Sprintf("%s health failed", scenario.ScenarioName))
			}
			continue
		}
		switch {
		case scenario.Review.Status == FinalizationStatusSkipped:
			hasActionableFailure = true
			summaries = append(summaries, fmt.Sprintf("%s review skipped: %s", scenario.ScenarioName, scenario.Review.SkipReason))
		case scenario.Review.Result == nil:
			hasActionableFailure = true
			summaries = append(summaries, fmt.Sprintf("%s review unavailable", scenario.ScenarioName))
		default:
			switch scenario.Review.Result.Classification {
			case FinalizationAggregateReady:
				summaries = append(summaries, fmt.Sprintf("%s ready", scenario.ScenarioName))
			case FinalizationAggregateReadyWithNotes:
				hasReadyWithNotes = true
				summaries = append(summaries, fmt.Sprintf("%s ready with notes", scenario.ScenarioName))
			default:
				hasActionableFailure = true
				summaries = append(summaries, fmt.Sprintf("%s needs follow-up: %s", scenario.ScenarioName, scenario.Review.Result.Summary))
			}
		}
	}

	classification = FinalizationAggregateReady
	switch {
	case hasActionableFailure:
		classification = FinalizationAggregateNeedsWork
	case hasReadyWithNotes:
		classification = FinalizationAggregateReadyWithNotes
	}

	if len(finalization.Warnings) > 0 {
		summaries = append(summaries, fmt.Sprintf("%d warning(s)", len(finalization.Warnings)))
	}
	summary = strings.Join(summaries, "; ")
	return classification, summary, hasActionableFailure
}

func (s *Service) completeFinalizationSkipped(executionID string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusSkipped
	finalization.Phase = FinalizationPhaseSkipped
	finalization.SkipReason = strings.TrimSpace(reason)
	finalization.CompletedAt = nowRFC3339()
	finalization.AggregateClassification = FinalizationAggregateSkipped
	finalization.AggregateSummary = strings.TrimSpace(reason)
	record.Finalization = finalization
	record.Status = StatusCompleted
	record.FinishedAt = nowRFC3339()
	record.FailureReason = ""
	record.UpdatedAt = nowRFC3339()
	if item, loadErr := s.loadBacklogItemByRecord(record); loadErr == nil {
		_ = s.updateBacklogStatus(item, backlogStatusCompleted)
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) failFinalization(executionID, scenarioName, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusFailed
	finalization.Phase = FinalizationPhaseFailed
	finalization.CompletedAt = nowRFC3339()
	finalization.AggregateClassification = FinalizationAggregateNotAssessable
	finalization.AggregateSummary = strings.TrimSpace(message)
	if warning := strings.TrimSpace(message); warning != "" {
		finalization.Warnings = append(finalization.Warnings, newFinalizationWarning(
			finalizationWarningFinalizationInfra,
			scenarioName,
			warning,
			false,
		))
	}
	record.Finalization = finalization
	record.Status = StatusFailed
	record.FailureReason = strings.TrimSpace(message)
	record.FinishedAt = nowRFC3339()
	record.UpdatedAt = nowRFC3339()
	if item, loadErr := s.loadBacklogItemByRecord(record); loadErr == nil {
		_ = s.updateBacklogStatus(item, backlogStatusFailed)
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func ensureFinalization(record *Record) *Finalization {
	if record.Finalization != nil {
		record.Finalization.Warnings = append([]FinalizationWarning(nil), record.Finalization.Warnings...)
		record.Finalization.AffectedScenarios = append([]string(nil), record.Finalization.AffectedScenarios...)
		record.Finalization.Scenarios = append([]ScenarioFinalization(nil), record.Finalization.Scenarios...)
		return record.Finalization
	}
	record.Finalization = &Finalization{
		Eligible:                isFinalizationEligible(*record),
		Status:                  FinalizationStatusPending,
		Phase:                   FinalizationPhaseScopeDetection,
		ScopeSource:             FinalizationScopeNone,
		Warnings:                []FinalizationWarning{},
		AffectedScenarios:       []string{},
		Scenarios:               []ScenarioFinalization{},
		StartedAt:               nowRFC3339(),
		AggregateSummary:        "",
		AggregateClassification: "",
	}
	return record.Finalization
}

func newFinalizationWarning(code, scenarioName, message string, retryable bool) FinalizationWarning {
	return FinalizationWarning{
		Code:         strings.TrimSpace(code),
		ScenarioName: strings.TrimSpace(scenarioName),
		Message:      strings.TrimSpace(message),
		Retryable:    retryable,
		CreatedAt:    nowRFC3339(),
	}
}

func (s *Service) applyFinalizationScope(executionID string, scope finalizationScope, skipReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusRunning
	finalization.Phase = FinalizationPhaseScopeDetection
	finalization.ScopeSource = scope.source
	finalization.SkipReason = strings.TrimSpace(skipReason)
	finalization.AffectedScenarios = append([]string(nil), scope.affectedScenarios...)
	finalization.Warnings = append(finalization.Warnings, scope.warnings...)
	finalization.Scenarios = make([]ScenarioFinalization, 0, len(scope.affectedScenarios))
	for _, scenarioName := range scope.affectedScenarios {
		finalization.Scenarios = append(finalization.Scenarios, ScenarioFinalization{
			ScenarioName: scenarioName,
			ChangedPaths: append([]string(nil), scope.changedPathsByScenario[scenarioName]...),
			Restart:      RestartResult{Status: FinalizationStatusPending},
			Health:       HealthCheckResult{Status: FinalizationStatusPending},
			Review:       ScenarioReviewStep{Status: FinalizationStatusPending},
		})
	}
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) markFinalizationPhase(executionID, phase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusRunning
	finalization.Phase = strings.TrimSpace(phase)
	record.Finalization = finalization
	record.Status = StatusValidating
	record.UpdatedAt = nowRFC3339()
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) appendFinalizationWarning(executionID string, warning FinalizationWarning) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Warnings = append(finalization.Warnings, warning)
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) updateScenarioRestartState(executionID, scenarioName string, restart RestartResult) error {
	return s.updateScenarioFinalization(executionID, scenarioName, func(scenario *ScenarioFinalization) {
		scenario.Restart = restart
	})
}

func (s *Service) updateScenarioHealthState(executionID, scenarioName string, health HealthCheckResult) error {
	return s.updateScenarioFinalization(executionID, scenarioName, func(scenario *ScenarioFinalization) {
		scenario.Health = health
	})
}

func (s *Service) updateScenarioReviewState(executionID, scenarioName string, review ScenarioReviewStep) error {
	return s.updateScenarioFinalization(executionID, scenarioName, func(scenario *ScenarioFinalization) {
		scenario.Review = review
	})
}

func (s *Service) updateScenarioFinalization(executionID, scenarioName string, mutate func(*ScenarioFinalization)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	for i := range finalization.Scenarios {
		if finalization.Scenarios[i].ScenarioName != scenarioName {
			continue
		}
		mutate(&finalization.Scenarios[i])
		record.Finalization = finalization
		record.UpdatedAt = nowRFC3339()
		if err := s.store.Save(records); err != nil {
			return err
		}
		s.dispatchStatusUpdate(*record)
		return nil
	}
	return fmt.Errorf("scenario finalization not found for %s/%s", executionID, scenarioName)
}

func (s *Service) loadScenarioFinalization(executionID, scenarioName string) (ScenarioFinalization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return ScenarioFinalization{}, err
	}
	record := records[idx]
	finalization := effectiveFinalization(record)
	if finalization == nil {
		return ScenarioFinalization{}, fmt.Errorf("execution %s has no finalization state", executionID)
	}
	for _, scenario := range finalization.Scenarios {
		if scenario.ScenarioName == scenarioName {
			return scenario, nil
		}
	}
	return ScenarioFinalization{}, fmt.Errorf("scenario finalization not found for %s/%s", executionID, scenarioName)
}

func (s *Service) loadExecutionForFinalization(executionID string) (Record, backlogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, backlogItem{}, err
	}
	record := records[idx]
	item, loadErr := s.loadBacklogItemByRecord(&record)
	if loadErr != nil {
		return Record{}, backlogItem{}, loadErr
	}
	return record, item, nil
}

func (s *Service) beginFinalization(executionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.processingFinalizations == nil {
		s.processingFinalizations = map[string]struct{}{}
	}
	if _, exists := s.processingFinalizations[executionID]; exists {
		return false
	}
	s.processingFinalizations[executionID] = struct{}{}
	return true
}

func (s *Service) endFinalization(executionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.processingFinalizations == nil {
		return
	}
	delete(s.processingFinalizations, executionID)
}

func effectiveFinalization(record Record) *Finalization {
	if record.Finalization != nil {
		return record.Finalization
	}
	if record.LegacyReviewResult == nil && record.LegacyReviewJobID == "" && record.LegacyReviewSkipReason == "" && record.LegacyReviewStartedAt == "" {
		return nil
	}
	finalization := &Finalization{
		Eligible:          true,
		Status:            FinalizationStatusCompleted,
		Phase:             FinalizationPhaseCompleted,
		ScopeSource:       FinalizationScopeAcceptanceAllow,
		StartedAt:         record.LegacyReviewStartedAt,
		CompletedAt:       record.FinishedAt,
		AffectedScenarios: []string{},
		Warnings:          []FinalizationWarning{},
		Scenarios:         []ScenarioFinalization{},
	}
	if record.Status == StatusValidating {
		finalization.Status = FinalizationStatusRunning
		finalization.Phase = FinalizationPhaseReviewing
	}
	if record.LegacyReviewSkipReason != "" {
		finalization.Status = FinalizationStatusSkipped
		finalization.Phase = FinalizationPhaseSkipped
		finalization.SkipReason = record.LegacyReviewSkipReason
		finalization.AggregateClassification = FinalizationAggregateSkipped
		finalization.AggregateSummary = record.LegacyReviewSkipReason
		return finalization
	}
	if record.LegacyReviewResult != nil {
		finalization.AggregateClassification = record.LegacyReviewResult.Classification
		finalization.AggregateSummary = record.LegacyReviewResult.Summary
		finalization.Scenarios = []ScenarioFinalization{{
			ScenarioName: record.BacklogName,
			Restart:      RestartResult{Status: FinalizationStatusCompleted},
			Health:       HealthCheckResult{Status: FinalizationStatusCompleted, SchemaValid: true},
			Review: ScenarioReviewStep{
				Status: FinalizationStatusCompleted,
				JobID:  record.LegacyReviewResult.JobID,
				Result: record.LegacyReviewResult,
			},
		}}
		return finalization
	}
	finalization.AggregateClassification = FinalizationAggregateNotAssessable
	return finalization
}

func unionSortedStrings(base []string, extras []string) []string {
	return pathutil.UniqueSortedStrings(append(append([]string{}, base...), extras...))
}

func mapKeysSorted(values map[string][]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func migrateLegacyFinalizationState(record *Record, item backlogItem) bool {
	if record == nil || record.Finalization != nil {
		return false
	}
	finalization := effectiveFinalization(*record)
	if finalization == nil {
		return false
	}
	if len(finalization.Scenarios) == 1 && finalization.Scenarios[0].ScenarioName == record.BacklogName {
		scenarios := pathutil.ScenariosFromGlobs(item.AcceptanceAllow)
		if len(scenarios) > 0 {
			finalization.Scenarios[0].ScenarioName = scenarios[0]
			finalization.AffectedScenarios = []string{scenarios[0]}
		}
	}
	record.Finalization = finalization
	record.LegacyReviewResult = nil
	record.LegacyReviewJobID = ""
	record.LegacyReviewSkipReason = ""
	record.LegacyReviewStartedAt = ""
	return true
}

func logFinalizationError(executionID string, err error) {
	if err == nil {
		return
	}
	log.Printf("[execution] finalization error for %s: %v", executionID, err)
}

// isReviewAgentEnabled checks the execution policy for the review_agent_enabled flag.
func (s *Service) isReviewAgentEnabled() bool {
	policy, err := s.policyProvider.LoadPolicy()
	if err != nil {
		return false
	}
	return policy.ReviewAgentEnabled
}

// triggerReviewAgent spawns the review agent to gather evidence after finalization.
// The agent runs asynchronously; this method returns after the spawn request is sent.
func (s *Service) triggerReviewAgent(ctx context.Context, executionID string, scope finalizationScope, item backlogItem) error {
	if s.reviewService == nil {
		return fmt.Errorf("review service not configured")
	}
	return s.reviewService.StartReviewForExecution(ctx,
		executionID, item.Kind, item.Name, item.Title,
		s.itemDir(item.Kind, item.Name),
		scope.affectedScenarios, scope.changedPathsByScenario,
	)
}

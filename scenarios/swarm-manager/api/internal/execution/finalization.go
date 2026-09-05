package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/experiment"
	"swarm-manager/internal/promptmanager"
)

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

	// Filter out self to avoid restarting our own process mid-finalization.
	scope.affectedScenarios = s.filterSelfScenario(executionID, scope.affectedScenarios)

	if err := s.markFinalizationPhase(executionID, FinalizationPhaseRestarting); err != nil {
		return err
	}
	for _, scenarioName := range scope.affectedScenarios {
		// Route restart/health to @shadow when this scenario is shadow-engaged
		// under the owner (plan P-b.5); bare name otherwise.
		target := s.shadowTargetFor(record, scenarioName)
		if err := s.runScenarioRestartAndHealth(ctx, executionID, scenarioName, target); err != nil {
			return err
		}
	}

	// Before/after baseline diff: compare each affected scenario against the
	// baseline captured before execution so the review agent can separate
	// regressions this item caused from pre-existing failures. Additive to the
	// absolute review below — best-effort, never fails finalization.
	if err := s.runBaselineDiffs(ctx, executionID, scope, record.PreExecBaselines); err != nil {
		return err
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
	//
	// reviewStarted records whether a review round was actually spawned. It
	// drives the terminal status decision in finishFinalization: a started
	// review lands the item in `in_review` (a round is actively gathering);
	// a review that was disabled or failed to spawn lands the item directly
	// in `review_pending` so the user can still decide a terminal state via
	// review-decide instead of being stranded in an `in_review` with no
	// review round behind it (the orphaned-in_review dead-end).
	reviewStarted, reviewSkipReason, err := s.runEvidenceGathering(ctx, executionID, scope, item)
	if err != nil {
		return err
	}

	// Pre-exec baselines have served their purpose (the diff results are
	// persisted on the record and handed to the review agent); reclaim them
	// unless configured to retain. Best-effort — never blocks finalization.
	s.cleanupPreExecBaselines(ctx, record.PreExecBaselines)

	return s.finishFinalization(executionID, reviewStarted, reviewSkipReason)
}

// filterSelfScenario drops the running scenario from the affected set so
// finalization never restarts its own process, recording a warning when it does.
func (s *Service) filterSelfScenario(executionID string, affected []string) []string {
	if s.selfScenarioName == "" {
		return affected
	}
	filtered := make([]string, 0, len(affected))
	for _, name := range affected {
		if name == s.selfScenarioName {
			slog.Warn("skipping self-restart during finalization",
				"execution_id", executionID,
				"scenario", s.selfScenarioName,
			)
			_ = s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningSelfRestartSkipped,
				s.selfScenarioName,
				fmt.Sprintf("Scenario %q was in scope but skipped because restarting it would kill this running process. If changes to %s require a restart, restart it manually after finalization completes.", s.selfScenarioName, s.selfScenarioName),
				false,
			))
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}

// runEvidenceGathering spawns the (policy-gated) review agent and reports
// whether a review round actually started plus a skip reason when it did not.
func (s *Service) runEvidenceGathering(ctx context.Context, executionID string, scope finalizationScope, item backlogItem) (reviewStarted bool, reviewSkipReason string, err error) {
	switch enabled, reason := s.checkReviewAgentEnabled(); {
	case enabled:
		if err := s.markFinalizationPhase(executionID, FinalizationPhaseEvidenceGathering); err != nil {
			return false, "", err
		}
		if err := s.triggerReviewAgent(ctx, executionID, scope, item); err != nil {
			slog.Warn("review agent spawn failed", "execution_id", executionID, "err", err)
			_ = s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningReviewAgentFailed, "", err.Error(), false,
			))
			return false, "review agent spawn failed: " + err.Error(), nil
		}
		return true, "", nil
	default:
		slog.Info("evidence gathering skipped", "execution_id", executionID, "reason", reason)
		_ = s.appendFinalizationWarning(executionID, newFinalizationWarning(
			reason, "", s.evidenceSkipMessage(reason), false,
		))
		return false, s.evidenceSkipMessage(reason), nil
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

// finishFinalization writes the terminal finalization state and moves the
// backlog item into the review gate.
//
// reviewStarted reports whether processFinalization actually spawned a review
// round. When true the item enters `in_review` (a round is gathering). When
// false — the review agent was disabled or its spawn failed — the item enters
// `review_pending` directly so the user can decide a terminal state, instead of
// being stranded in `in_review` with no review round to ever advance it (the
// orphaned-in_review dead-end). reviewSkipReason explains the latter case and
// is recorded as a synthetic review round for UI/audit clarity.
func (s *Service) finishFinalization(executionID string, reviewStarted bool, reviewSkipReason string) error {
	s.mu.Lock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	record := &records[idx]
	prevStatus := record.Status
	finalization := ensureFinalization(record)

	finalization.Status = FinalizationStatusCompleted
	finalization.Phase = FinalizationPhaseCompleted
	finalization.CompletedAt = nowRFC3339()
	classification, summary, hasActionableFailure := summarizeFinalization(*finalization, s.finalizationCfg.BaselineRegressionGateEnabled)
	finalization.AggregateClassification = classification
	finalization.AggregateSummary = summary
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	record.FinishedAt = nowRFC3339()

	// A started review gathers evidence in `in_review`; without one, route the
	// item straight to the human-decidable `review_pending` so it can never be
	// orphaned in `in_review` with no round behind it.
	reviewStatus := backlogStatusInReview
	if !reviewStarted {
		reviewStatus = backlogStatusReviewPending
	}

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
			if err := s.updateBacklogStatus(item, reviewStatus); err != nil {
				slog.Warn("failed to set backlog review status after finalization",
					"execution_id", executionID, "backlog_ref", item.Kind+"/"+item.Name, "status", reviewStatus, "err", err)
			}
		default:
			record.Status = StatusCompleted
			record.FailureReason = ""
			if err := s.updateBacklogStatus(item, reviewStatus); err != nil {
				slog.Warn("failed to set backlog review status after finalization",
					"execution_id", executionID, "backlog_ref", item.Kind+"/"+item.Name, "status", reviewStatus, "err", err)
			}
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

	// Record why no evidence was gathered when we routed straight to
	// review_pending, so the review surface explains the empty round set.
	// Best-effort: never blocks finalization. Done outside the lock (disk I/O).
	if loadErr == nil && !reviewStarted && s.reviewService != nil {
		reason := strings.TrimSpace(reviewSkipReason)
		if reason == "" {
			reason = "review agent did not run; routed to review_pending for manual decision"
		}
		if recErr := s.reviewService.RecordUnavailableReview(item.Kind, item.Name, executionID, reason); recErr != nil {
			slog.Warn("failed to record review-unavailable marker",
				"execution_id", executionID, "backlog_ref", item.Kind+"/"+item.Name, "err", recErr)
		}
	}

	s.dispatchStatusAndLog(*record, prevStatus)

	// Fire-and-forget: record experiment outcome if this execution was part of an experiment.
	s.recordExperimentOutcome(record)

	// Baseline Modes engagements are NOT closed here (plan P-c). Finalization is
	// the agent's verdict, not the user's: the owner's engagement set is held
	// across finalization into review_pending and promoted/abandoned as a whole
	// at review-decide (the atomic accept/reject), so a candidate is never blessed
	// into live before the human accept. The owner keying (ownerKeyFor) means a
	// fixup transparently shares the same set — no per-record inheritance needed.

	if autoSpawnFixup {
		s.spawnFixupRun(context.Background(), record, item)
	}
	return nil
}

// recordExperimentOutcome posts an outcome to prompt-manager if the execution
// was part of an A/B experiment. Failures are logged but never block finalization.
func (s *Service) recordExperimentOutcome(record *Record) {
	if record.PromptTrace == nil || strings.TrimSpace(record.PromptTrace.ExperimentID) == "" {
		return
	}
	if s.experimentClient == nil {
		slog.Warn("experiment outcome skipped: no experiment client configured",
			"execution_id", record.ExecutionID,
			"experiment_id", record.PromptTrace.ExperimentID,
		)
		return
	}

	classification := ""
	if record.Finalization != nil {
		classification = record.Finalization.AggregateClassification
	}

	data := experiment.OutcomeDataV1{
		ExecutionID:    record.ExecutionID,
		Classification: classification,
		BacklogKind:    record.BacklogKind,
		BacklogName:    record.BacklogName,
		Purpose:        record.PromptTrace.Purpose,
		HadFixup:       record.FixupAttempt > 0,
		DurationSecs:   executionDuration(*record),
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		slog.Warn("experiment outcome marshal failed",
			"execution_id", record.ExecutionID,
			"err", err,
		)
		return
	}

	outcome := promptmanager.ExperimentOutcomeRequest{
		VariantID:     record.PromptTrace.VariantID,
		Source:        "swarm-manager",
		SchemaVersion: experiment.OutcomeSchemaVersion,
		Data:          dataJSON,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.experimentClient.RecordExperimentOutcome(ctx, record.PromptTrace.ExperimentID, outcome); err != nil {
		slog.Warn("experiment outcome recording failed",
			"execution_id", record.ExecutionID,
			"experiment_id", record.PromptTrace.ExperimentID,
			"err", err,
		)
	}
}

// checkReviewAgentEnabled checks the execution policy for the review_agent_enabled flag.
// Returns (true, "") when enabled, or (false, warningCode) with a machine-readable
// reason when disabled so callers can emit structured warnings.
func (s *Service) checkReviewAgentEnabled() (bool, string) {
	policy, err := s.policyProvider.LoadPolicy()
	if err != nil {
		slog.Error("failed to load policy for review agent check", "err", err)
		return false, finalizationWarningEvidenceSkippedPolicyErr
	}
	if !policy.ReviewAgentEnabled {
		return false, finalizationWarningEvidenceSkippedDisabled
	}
	return true, ""
}

// evidenceSkipMessage returns a human-readable message for an evidence-skip warning code.
func (s *Service) evidenceSkipMessage(code string) string {
	switch code {
	case finalizationWarningEvidenceSkippedDisabled:
		return "Review agent is disabled in settings. Enable it to gather evidence automatically."
	case finalizationWarningEvidenceSkippedPolicyErr:
		return "Could not load settings to check review agent policy. Evidence gathering was skipped."
	default:
		return "Evidence gathering was skipped."
	}
}

// triggerReviewAgent spawns the review agent to gather evidence after finalization.
// The agent runs asynchronously; this method returns after the spawn request is sent.
func (s *Service) triggerReviewAgent(ctx context.Context, executionID string, scope finalizationScope, item backlogItem) error {
	if s.reviewService == nil {
		return fmt.Errorf("review service not configured")
	}

	// Inject agent activity spec so the tracked agent service can record the spawn.
	ctx = agentactivity.WithSpec(ctx, backlogActivitySpec(
		item,
		executionID,
		agentactivity.PurposeReview,
		"finalization",
		map[string]string{
			"entrypoint": "finalization.evidence_gathering",
		},
	))

	// Load GCT review results for each affected scenario so the review agent
	// can evaluate existing coverage before gathering evidence.
	type scenarioGCTResult struct {
		Classification string            `json:"classification"`
		Dimensions     []ReviewDimension `json:"dimensions,omitempty"`
		RawDimensions  json.RawMessage   `json:"raw_dimensions,omitempty"`
		Summary        string            `json:"summary"`
	}
	resultsByScenario := make(map[string]*scenarioGCTResult)
	baselineByScenario := make(map[string]BaselineDiffResult)
	for _, name := range scope.affectedScenarios {
		sf, err := s.loadScenarioFinalization(executionID, name)
		if err != nil {
			continue
		}
		if sf.Review.Result != nil {
			r := sf.Review.Result
			resultsByScenario[name] = &scenarioGCTResult{
				Classification: r.Classification,
				Dimensions:     r.Dimensions,
				RawDimensions:  r.RawDimensions,
				Summary:        r.Summary,
			}
		}
		if sf.BaselineDiff != nil {
			baselineByScenario[name] = *sf.BaselineDiff
		}
	}

	gctJSON := ""
	if len(resultsByScenario) > 0 {
		if b, err := json.Marshal(resultsByScenario); err == nil {
			gctJSON = string(b)
		}
	}
	baselineJSON := MarshalBaselineDiffResults(baselineByScenario)

	machineEvidence := resolveCriterionChecks(ctx, item.AcceptanceCriteria, defaultCriterionCommandRunner{})
	return s.reviewService.StartReviewForExecution(ctx,
		executionID, item.Kind, item.Name, item.Title, item.Description,
		s.itemDir(item.Kind, item.Name),
		item.AcceptanceCriteria,
		machineEvidence,
		scope.affectedScenarios, scope.changedPathsByScenario,
		gctJSON, baselineJSON,
	)
}

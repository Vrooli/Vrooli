// DOC: docs/reference/api-architecture.md#pipeline-system-core-engine
package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

// runPipelineAsync executes the pipeline stages sequentially.
func (o *DefaultOrchestrator) runPipelineAsync(ctx context.Context, pipelineID string, config *Config) {
	defer o.cancelManager.Clear(pipelineID)

	success := false
	rollback := config.takeVersionRollback()
	defer func() {
		if rollback == nil || success {
			return
		}
		if err := rollback.Restore(); err != nil {
			o.logger.Error("Failed to rollback version update", "pipeline_id", pipelineID, "error", err)
			return
		}
		o.logger.Info("Rolled back version update", "pipeline_id", pipelineID)
	}()

	// Update status to running and transition to initializing state
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusRunning
		s.TransitionTo(PipelineStateInitializing, "Pipeline starting execution")
		s.UpdateProgress()
	})

	// Capture git provenance for build traceability
	scenarioPath := filepath.Join(o.scenarioRoot, config.ScenarioName)
	provenance := CaptureProvenance(scenarioPath, config.Version)
	o.store.Update(pipelineID, func(s *Status) {
		s.Provenance = provenance
	})

	// Build stage input
	input := &StageInput{
		Config:       config,
		PipelineID:   pipelineID,
		ScenarioPath: scenarioPath,
		Provenance:   provenance,
		Logger:       o.logger,
	}

	// If resuming, restore input from parent pipeline
	o.restoreResumeInput(input, config, pipelineID)

	// Track whether we've reached the resume stage (if resuming)
	resumeFromStage := config.GetResumeFromStage()
	reachedResumeStage := resumeFromStage == "" // If not resuming, consider it reached

	// Filter stages if specific stages were requested
	stagesToRun := o.stages
	if requestedStages := config.GetStages(); len(requestedStages) > 0 {
		stagesToRun = o.filterStages(requestedStages)
		o.logger.Info("Filtered stages for execution",
			"pipeline_id", pipelineID,
			"requested", requestedStages,
			"count", len(stagesToRun),
		)
	}

	// Execute stages sequentially
	for _, stage := range stagesToRun {
		stageName := stage.Name()

		// If resuming, skip stages until we reach the resume point
		if !reachedResumeStage {
			if stageName == resumeFromStage {
				reachedResumeStage = true
				o.logger.Info("Reached resume stage", "pipeline_id", pipelineID, "stage", stageName)
			} else {
				o.markStageSkipped(pipelineID, stageName, "Stage skipped - resuming from later stage")
				continue
			}
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			o.markPipelineCancelled(pipelineID)
			return
		default:
		}

		outcome := o.executeStage(ctx, stage, input, config, pipelineID)
		switch outcome {
		case stageOutcomeFailed:
			return
		case stageOutcomeCancelled:
			return
		case stageOutcomeStopped:
			success = true
			return
		}
	}

	// Mark pipeline as completed
	finalArtifacts := collectArtifacts(input)
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusCompleted
		s.CompletedAt = o.timeProvider.Now()
		s.CurrentStage = ""
		s.FinalArtifacts = finalArtifacts
		s.TransitionTo(PipelineStateCompleted, "Pipeline completed successfully")
		s.UpdateProgress()
	})

	success = true
	o.logger.Info("Pipeline completed", "pipeline_id", pipelineID)
}

type stageOutcome int

const (
	stageOutcomeContinue stageOutcome = iota
	stageOutcomeFailed
	stageOutcomeCancelled
	stageOutcomeStopped
)

// executeStage runs a single pipeline stage and returns the outcome.
func (o *DefaultOrchestrator) executeStage(ctx context.Context, stage Stage, input *StageInput, config *Config, pipelineID string) stageOutcome {
	stageName := stage.Name()

	o.store.Update(pipelineID, func(s *Status) {
		s.CurrentStage = stageName
		s.TransitionTo(PipelineStateQueueingStage, fmt.Sprintf("Queueing stage: %s", stageName))
		s.UpdateProgress()
	})

	o.logger.Info("Starting stage", "pipeline_id", pipelineID, "stage", stageName)

	// Check if stage can be skipped
	if stage.CanSkip(input) {
		o.markStageSkipped(pipelineID, stageName, "Stage skipped based on configuration")
		if config.GetStopAfterStage() == stageName {
			o.stopAfterStage(pipelineID, stageName, input)
			return stageOutcomeStopped
		}
		return stageOutcomeContinue
	}

	o.store.Update(pipelineID, func(s *Status) {
		s.TransitionTo(PipelineStateExecutingStage, fmt.Sprintf("Executing stage: %s", stageName))
	})

	result := stage.Execute(ctx, input)

	o.store.Update(pipelineID, func(s *Status) {
		s.TransitionTo(PipelineStateProcessingResult, fmt.Sprintf("Processing result: %s", stageName))
	})
	o.store.UpdateStage(pipelineID, stageName, result)

	o.logger.Info("Stage completed", "pipeline_id", pipelineID, "stage", stageName, "status", result.Status)

	if result.Status == StatusFailed && config.GetStopOnFailure() {
		o.store.Update(pipelineID, func(s *Status) {
			s.Status = StatusFailed
			s.CompletedAt = o.timeProvider.Now()
			s.Error = fmt.Sprintf("stage %s failed: %s", stageName, result.Error)
			s.TransitionTo(PipelineStateFailed, fmt.Sprintf("Stage %s failed", stageName))
			s.UpdateProgress()
		})
		o.logger.Error("Pipeline failed", "pipeline_id", pipelineID, "stage", stageName, "error", result.Error)
		return stageOutcomeFailed
	}
	if result.Status == StatusFailed {
		o.logger.Warn("Stage failed but continuing", "pipeline_id", pipelineID, "stage", stageName)
	}

	if result.Status == StatusCancelled {
		o.store.Update(pipelineID, func(s *Status) {
			s.Status = StatusCancelled
			s.CompletedAt = o.timeProvider.Now()
			s.Error = "pipeline cancelled"
			s.TransitionTo(PipelineStateCancelled, fmt.Sprintf("Stage %s cancelled", stageName))
			s.UpdateProgress()
		})
		o.logger.Info("Pipeline cancelled at stage", "pipeline_id", pipelineID, "stage", stageName)
		return stageOutcomeCancelled
	}

	if config.GetStopAfterStage() == stageName {
		o.stopAfterStage(pipelineID, stageName, input)
		return stageOutcomeStopped
	}

	return stageOutcomeContinue
}

// markStageSkipped records a stage as skipped.
func (o *DefaultOrchestrator) markStageSkipped(pipelineID, stageName, reason string) {
	result := &StageResult{
		Stage:       stageName,
		Status:      StatusSkipped,
		StartedAt:   o.timeProvider.Now(),
		CompletedAt: o.timeProvider.Now(),
		Logs:        []string{reason},
	}
	o.store.UpdateStage(pipelineID, stageName, result)
	o.logger.Info("Stage skipped", "pipeline_id", pipelineID, "stage", stageName)
}

// markPipelineCancelled records the pipeline as cancelled via context.
func (o *DefaultOrchestrator) markPipelineCancelled(pipelineID string) {
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusCancelled
		s.CompletedAt = o.timeProvider.Now()
		s.Error = "pipeline cancelled"
		s.TransitionTo(PipelineStateCancelled, "Pipeline cancelled by context")
		s.UpdateProgress()
	})
	o.logger.Info("Pipeline cancelled", "pipeline_id", pipelineID)
}

// restoreResumeInput copies saved input from a parent pipeline when resuming.
func (o *DefaultOrchestrator) restoreResumeInput(input *StageInput, config *Config, pipelineID string) {
	if config.ParentPipelineID == "" {
		return
	}
	parentStatus, ok := o.store.Get(config.ParentPipelineID)
	if !ok || parentStatus.ResumedInput == nil {
		return
	}
	input.BundleResult = parentStatus.ResumedInput.BundleResult
	input.PreflightResult = parentStatus.ResumedInput.PreflightResult
	input.GenerationResult = parentStatus.ResumedInput.GenerationResult
	input.BuildResult = parentStatus.ResumedInput.BuildResult
	input.SmokeTestResult = parentStatus.ResumedInput.SmokeTestResult
	input.DeployResult = parentStatus.ResumedInput.DeployResult
	input.ScenarioMetadata = parentStatus.ResumedInput.ScenarioMetadata
	input.DesktopPath = parentStatus.ResumedInput.DesktopPath
	o.logger.Info("Restored input from parent pipeline", "pipeline_id", pipelineID, "parent_id", config.ParentPipelineID)
}

// stopAfterStage marks the pipeline as completed after the specified stage and saves input for resumption.
func (o *DefaultOrchestrator) stopAfterStage(pipelineID, stageName string, input *StageInput) {
	finalArtifacts := collectArtifacts(input)

	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusCompleted
		s.CompletedAt = o.timeProvider.Now()
		s.CurrentStage = ""
		s.StoppedAfterStage = stageName
		s.FinalArtifacts = finalArtifacts
		// Save the input so it can be restored when resuming
		s.ResumedInput = input
		s.TransitionTo(PipelineStateCompleted, fmt.Sprintf("Pipeline stopped after stage: %s", stageName))
		s.UpdateProgress()
	})

	o.logger.Info("Pipeline stopped after stage",
		"pipeline_id", pipelineID,
		"stopped_after", stageName,
	)
}

// pollForCompletion polls for pipeline completion until it finishes or times out.
// Returns the final status when complete, failed, or cancelled.
// Returns an error (with partial status) if the timeout is exceeded or the pipeline disappears.
func (o *DefaultOrchestrator) pollForCompletion(ctx context.Context, pipelineID string, timeoutSecs int) (*Status, error) {
	timeout := time.Duration(timeoutSecs) * time.Second
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPipelinePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			current, _ := o.GetStatus(pipelineID)
			return current, ctx.Err()
		case <-ticker.C:
			current, ok := o.GetStatus(pipelineID)
			if !ok {
				return nil, fmt.Errorf("pipeline %s disappeared", pipelineID)
			}

			if current.IsComplete() {
				return current, nil
			}

			if time.Now().After(deadline) {
				return current, fmt.Errorf("timeout after %d seconds", timeoutSecs)
			}
		}
	}
}

// currentPlatform returns the current platform identifier.
func currentPlatform() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map to electron-builder platform names
	switch goos {
	case "darwin":
		if goarch == "arm64" {
			return "mac-arm64"
		}
		return "mac"
	case "windows":
		return "win"
	case "linux":
		return "linux"
	default:
		return goos
	}
}

// collectArtifacts gathers final artifact paths from the pipeline input.
func collectArtifacts(input *StageInput) map[string]string {
	if input.BuildResult == nil {
		return make(map[string]string)
	}
	return GetReadyArtifacts(input.BuildResult.PlatformResults)
}

// filterStages returns only the stages that match the requested stage names.
// The returned stages preserve the original pipeline order, not the order of the requested list.
func (o *DefaultOrchestrator) filterStages(requested []string) []Stage {
	requestedSet := make(map[string]bool, len(requested))
	for _, name := range requested {
		requestedSet[name] = true
	}

	filtered := make([]Stage, 0, len(requested))
	for _, stage := range o.stages {
		if requestedSet[stage.Name()] {
			filtered = append(filtered, stage)
		}
	}
	return filtered
}

// This file provides high-level retrieval and control operations for runs.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"

	agentconfig "agent-manager/internal/config"

	"github.com/google/uuid"
)

func (o *Orchestrator) GetRunnerStatus(ctx context.Context) ([]*RunnerStatus, error) {
	if o.runners == nil {
		return nil, nil
	}

	var statuses []*RunnerStatus
	for _, r := range o.runners.List() {
		available, msg := r.IsAvailable(ctx)
		statuses = append(statuses, &RunnerStatus{
			Type:         r.Type(),
			Available:    available,
			Message:      msg,
			Capabilities: r.Capabilities(),
		})
	}
	return statuses, nil
}

// ProbeRunner sends a real, bounded request through the registered runner
// adapter. Execution therefore follows the same launcher and environment seam
// as a managed run rather than spawning a coding-agent binary from orchestration.
func (o *Orchestrator) ProbeRunner(ctx context.Context, runnerType domain.RunnerType) (*ProbeResult, error) {
	if o.runners == nil {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    "no runner registry configured",
		}, nil
	}

	r, err := o.runners.Get(runnerType)
	if err != nil {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("runner not found: %v", err),
		}, nil
	}

	// First check if the runner reports itself as available
	available, msg := r.IsAvailable(ctx)
	if !available {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    msg,
		}, nil
	}

	start := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result, execErr := r.Execute(probeCtx, runner.ExecuteRequest{
		RunID:          uuid.New(),
		Tag:            "agent-manager-runner-probe",
		ResolvedConfig: &domain.RunConfig{RunnerType: runnerType, MaxTurns: 1},
		WorkingDir:     os.TempDir(),
		Prompt:         "Reply with exactly one word: PROBE_OK",
	})
	duration := time.Since(start)
	if probeCtx.Err() == context.DeadlineExceeded {
		return &ProbeResult{RunnerType: runnerType, Success: false, Message: "runner probe timed out after 30s", DurationMs: duration.Milliseconds()}, nil
	}
	if execErr != nil {
		return &ProbeResult{RunnerType: runnerType, Success: false, Message: fmt.Sprintf("runner probe failed: %v", execErr), DurationMs: duration.Milliseconds()}, nil
	}
	if result == nil || !result.Success {
		message := "runner probe returned an unsuccessful result"
		if result != nil && result.ErrorMessage != "" {
			message = result.ErrorMessage
		}
		return &ProbeResult{RunnerType: runnerType, Success: false, Message: message, DurationMs: duration.Milliseconds()}, nil
	}
	return &ProbeResult{
		RunnerType: runnerType,
		Success:    true,
		Message:    "runner completed managed probe",
		DurationMs: duration.Milliseconds(),
	}, nil
}

// PurgeData deletes profiles, tasks, or runs matching a regex pattern.
func (o *Orchestrator) PurgeData(ctx context.Context, req PurgeRequest) (*PurgeResult, error) {
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		return nil, domain.NewValidationError("pattern", "pattern is required")
	}
	if len(req.Targets) == 0 {
		return nil, domain.NewValidationError("targets", "at least one target is required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, domain.NewValidationError("pattern", "invalid regex pattern")
	}

	targets := map[PurgeTarget]bool{}
	for _, t := range req.Targets {
		targets[t] = true
	}

	result := &PurgeResult{
		Matched: PurgeCounts{},
		Deleted: PurgeCounts{},
		DryRun:  req.DryRun,
	}

	var profileIDs []uuid.UUID
	if targets[PurgeTargetProfiles] {
		profiles, err := o.profiles.List(ctx, repository.ListFilter{})
		if err != nil {
			return nil, err
		}
		for _, profile := range profiles {
			if re.MatchString(profile.ProfileKey) {
				result.Matched.Profiles++
				profileIDs = append(profileIDs, profile.ID)
			}
		}
	}

	var taskIDs []uuid.UUID
	if targets[PurgeTargetTasks] {
		tasks, err := o.tasks.List(ctx, repository.ListFilter{})
		if err != nil {
			return nil, err
		}
		for _, task := range tasks {
			if re.MatchString(task.Title) {
				result.Matched.Tasks++
				taskIDs = append(taskIDs, task.ID)
			}
		}
	}

	var runIDs []uuid.UUID
	if targets[PurgeTargetRuns] {
		runs, err := o.runs.List(ctx, repository.RunListFilter{})
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			if re.MatchString(run.GetTag()) {
				result.Matched.Runs++
				runIDs = append(runIDs, run.ID)
			}
		}
	}

	if req.DryRun {
		return result, nil
	}

	for _, id := range runIDs {
		if o.events != nil {
			if err := o.events.Delete(ctx, id); err != nil {
				return nil, err
			}
		}
		if o.checkpoints != nil {
			if err := o.checkpoints.Delete(ctx, id); err != nil {
				return nil, err
			}
		}
		if err := o.runs.Delete(ctx, id); err != nil {
			return nil, err
		}
		result.Deleted.Runs++
	}

	for _, id := range taskIDs {
		if err := o.tasks.Delete(ctx, id); err != nil {
			return nil, err
		}
		result.Deleted.Tasks++
	}

	for _, id := range profileIDs {
		if err := o.profiles.Delete(ctx, id); err != nil {
			return nil, err
		}
		result.Deleted.Profiles++
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// Helper Types
// -----------------------------------------------------------------------------

// EventBroadcaster is a callback for broadcasting events in real-time.
// This is typically implemented by the WebSocket hub.
//
// The canonical definition lives in the phases package so per-phase
// functions can reference it without an import cycle. The alias here keeps
// existing orchestration call sites compiling without per-site rewrites.
type EventBroadcaster = phases.EventBroadcaster

func appendAndBroadcastEvents(ctx context.Context, store event.Store, broadcaster EventBroadcaster, runID uuid.UUID, events ...*domain.RunEvent) error {
	persistable := make([]*domain.RunEvent, 0, len(events))
	for _, evt := range events {
		if evt != nil {
			persistable = append(persistable, evt)
		}
	}
	if len(persistable) == 0 {
		return nil
	}
	if store == nil {
		return fmt.Errorf("event store is required before broadcasting run events")
	}
	if err := store.Append(ctx, runID, persistable...); err != nil {
		return err
	}
	if broadcaster != nil {
		for _, evt := range persistable {
			broadcaster.BroadcastEvent(evt)
		}
	}
	return nil
}

func (o *Orchestrator) appendAndBroadcastEvents(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	return appendAndBroadcastEvents(ctx, o.events, o.broadcaster, runID, events...)
}

// eventStoreAdapter adapts event.Store to runner.EventSink
type eventStoreAdapter struct {
	store        event.Store
	runID        uuid.UUID
	lastSequence int64
}

func (e *eventStoreAdapter) Emit(evt *domain.RunEvent) error {
	if err := e.store.Append(context.Background(), e.runID, evt); err != nil {
		return err
	}
	e.lastSequence = evt.Sequence
	return nil
}

func (e *eventStoreAdapter) Close() error {
	return nil
}

func (e *eventStoreAdapter) LastSequence() int64 {
	return e.lastSequence
}

// broadcastingEventSink stores events AND broadcasts them via WebSocket.
type broadcastingEventSink struct {
	store        event.Store
	runID        uuid.UUID
	broadcaster  EventBroadcaster
	lastSequence int64
}

func (b *broadcastingEventSink) Emit(evt *domain.RunEvent) error {
	// Validate event and log warnings for missing data
	domain.ValidateEvent(evt)

	if err := appendAndBroadcastEvents(context.Background(), b.store, b.broadcaster, b.runID, evt); err != nil {
		obs.Component("broadcast-sink").Warn("event store append failed",
			obs.KeyRunID, b.runID.String(),
			obs.KeyError, err.Error(),
		)
		return err
	}
	b.lastSequence = evt.Sequence

	if b.broadcaster != nil {
		// Also emit progress events for status changes
		if data, ok := evt.Data.(*domain.StatusEventData); ok {
			b.broadcaster.BroadcastProgress(b.runID, domain.RunPhase(data.NewStatus), 0, data.Reason)
		}
		if data, ok := evt.Data.(*domain.ProgressEventData); ok {
			b.broadcaster.BroadcastProgress(b.runID, data.Phase, data.PercentComplete, data.CurrentAction)
		}
	}

	return nil
}

func (b *broadcastingEventSink) Close() error {
	return nil
}

func (b *broadcastingEventSink) LastSequence() int64 {
	return b.lastSequence
}

func (o *Orchestrator) runEventSink(runID uuid.UUID) runner.EventSink {
	switch {
	case o.events != nil && o.broadcaster != nil:
		return &broadcastingEventSink{
			store:       o.events,
			runID:       runID,
			broadcaster: o.broadcaster,
		}
	case o.events != nil:
		return &eventStoreAdapter{store: o.events, runID: runID}
	default:
		return &noOpEventSink{}
	}
}

// runnerTypeOrEmpty returns the runner type from a run's resolved
// config, or "" when no resolved config is set yet (e.g. during
// pre-spawn validation). Used for lifecycle event tagging.
func runnerTypeOrEmpty(run *domain.Run) domain.RunnerType {
	if run == nil || run.ResolvedConfig == nil {
		return ""
	}
	return run.ResolvedConfig.RunnerType
}

// dispatcherSink returns an obs.Sink for emitting lifecycle events
// (spawn-enqueued, spawn-started) from the spawn dispatcher path. It
// uses the same store + broadcaster as the per-run gate, so the
// timeline shows a continuous lifecycle from "queued" through "exited"
// regardless of where in the run-executor stack the event originated.
//
// Returned sink is non-nil even when the orchestrator has no event
// store wired (defaults to the noOp sink so dispatcher.Enqueue still
// emits its log line).
func (o *Orchestrator) dispatcherSink(runID uuid.UUID) obs.Sink {
	return o.runEventSink(runID)
}

// noOpEventSink discards events
type noOpEventSink struct{}

func (n *noOpEventSink) Emit(_ *domain.RunEvent) error { return nil }
func (n *noOpEventSink) Close() error                  { return nil }

// valueOrDefault returns the pointer value or default
func valueOrDefault(ptr *domain.RunMode, def domain.RunMode) domain.RunMode {
	if ptr != nil {
		return *ptr
	}
	return def
}

// -----------------------------------------------------------------------------
// Investigation Settings Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetInvestigationSettings(ctx context.Context) (*domain.InvestigationSettings, error) {
	var settings *domain.InvestigationSettings
	if o.investigationSettings == nil {
		settings = domain.DefaultInvestigationSettings()
	} else {
		var err error
		settings, err = o.investigationSettings.Get(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Overlay prompt templates from prompt-manager skills (overrides DB values)
	if o.promptClient != nil {
		if prompt, err := o.promptClient.ReadSkill(ctx, "agent-manager-process-investigation", nil, false); err == nil {
			settings.PromptTemplate = prompt
		}
		if applyPrompt, err := o.promptClient.ReadSkill(ctx, "agent-manager-process-investigation-apply", nil, false); err == nil {
			settings.ApplyPromptTemplate = applyPrompt
		}
	}

	return settings, nil
}

func (o *Orchestrator) UpdateInvestigationSettings(ctx context.Context, settings *domain.InvestigationSettings) error {
	if o.investigationSettings == nil {
		return domain.NewConfigMissingError("investigationSettings", "repository not configured", nil)
	}

	// Validate operational settings
	if !settings.DefaultDepth.IsValid() {
		return domain.NewValidationError("defaultDepth", "invalid depth value")
	}

	// Write prompt templates to prompt-manager skills
	if o.promptClient != nil {
		if adminClient, ok := o.promptClient.(promptmanager.AdminClient); ok {
			if settings.PromptTemplate != "" {
				content := settings.PromptTemplate
				if _, err := adminClient.UpdateSkill(ctx, "agent-manager-process-investigation",
					promptmanager.PromptSkillUpdate{Content: &content}); err != nil {
					return fmt.Errorf("update investigation skill: %w", err)
				}
			}
			if settings.ApplyPromptTemplate != "" {
				content := settings.ApplyPromptTemplate
				if _, err := adminClient.UpdateSkill(ctx, "agent-manager-process-investigation-apply",
					promptmanager.PromptSkillUpdate{Content: &content}); err != nil {
					return fmt.Errorf("update apply investigation skill: %w", err)
				}
			}
		}
	}

	// Operational config still saved to local DB
	return o.investigationSettings.Update(ctx, settings)
}

func (o *Orchestrator) ResetInvestigationSettings(ctx context.Context) error {
	if o.investigationSettings == nil {
		return domain.NewConfigMissingError("investigationSettings", "repository not configured", nil)
	}

	// Revert prompt-manager skills to original version
	if o.promptClient != nil {
		if adminClient, ok := o.promptClient.(promptmanager.AdminClient); ok {
			_ = adminClient.RevertSkillVersion(ctx, "agent-manager-process-investigation", 1)
			_ = adminClient.RevertSkillVersion(ctx, "agent-manager-process-investigation-apply", 1)
		}
	}

	// Reset operational config in DB
	return o.investigationSettings.Reset(ctx)
}

// -----------------------------------------------------------------------------
// Orchestration Settings Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetOrchestrationSettings(_ context.Context) (*agentconfig.OrchestrationSettings, error) {
	if o.orchestrationSettings == nil {
		defaults := agentconfig.DefaultOrchestrationSettings()
		return &defaults, nil
	}
	settings := o.orchestrationSettings.Get()
	return &settings, nil
}

func (o *Orchestrator) UpdateOrchestrationSettings(_ context.Context, settings *agentconfig.OrchestrationSettings) error {
	if o.orchestrationSettings == nil {
		return domain.NewConfigMissingError("orchestrationSettings", "store not configured", nil)
	}
	if err := o.orchestrationSettings.Update(*settings); err != nil {
		return err
	}
	o.propagateOrchestrationSettings(settings)
	return nil
}

func (o *Orchestrator) ResetOrchestrationSettings(_ context.Context) error {
	if o.orchestrationSettings == nil {
		return domain.NewConfigMissingError("orchestrationSettings", "store not configured", nil)
	}
	if err := o.orchestrationSettings.Reset(); err != nil {
		return err
	}
	defaults := agentconfig.DefaultOrchestrationSettings()
	o.propagateOrchestrationSettings(&defaults)
	return nil
}

// propagateOrchestrationSettings applies updated settings to running components.
func (o *Orchestrator) propagateOrchestrationSettings(s *agentconfig.OrchestrationSettings) {
	// Update orchestrator config (affects new runs).
	o.config.DefaultTimeout = time.Duration(s.RunExecution.RunTimeoutMinutes) * time.Minute
	o.config.MaxConcurrentRuns = s.RunExecution.MaxConcurrentRuns
	o.config.RequireSandboxByDefault = s.SafetyIsolation.RequireSandbox

	// Propagate to reconciler.
	if o.reconciler != nil {
		o.reconciler.UpdateConfig(ReconcilerConfig{
			Interval:          time.Duration(s.HealthDetection.ReconcilerIntervalSeconds) * time.Second,
			StaleThreshold:    time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second,
			MaxRecoveryAge:    time.Duration(s.HealthDetection.MaxRecoveryAgeSeconds) * time.Second,
			OrphanGracePeriod: time.Duration(s.ProcessTermination.OrphanGracePeriodSeconds) * time.Second,
			MaxStaleRuns:      10,
			KillOrphans:       s.ProcessTermination.KillOrphans,
			AutoRecover:       true,
		})
	}

	// Propagate to terminator.
	if o.terminator != nil {
		o.terminator.UpdateConfig(TerminatorConfig{
			GracePeriod:      time.Duration(s.ProcessTermination.GracePeriodSeconds) * time.Second,
			MaxRetries:       s.ProcessTermination.TerminationMaxRetries,
			BaseBackoff:      500 * time.Millisecond,
			MaxBackoff:       5 * time.Second,
			VerifyTimeout:    2 * time.Second,
			KillProcessGroup: s.ProcessTermination.KillProcessGroup,
		})
	}
}

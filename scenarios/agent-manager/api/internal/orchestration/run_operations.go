// This file provides high-level retrieval and control operations for runs.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

// ProbeRunner sends a real test request to a runner to verify end-to-end functionality.
// This invokes the agent with a minimal prompt to verify CLI + auth + API all work.
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

	// Build the probe command - uses a minimal prompt to reduce cost/time
	// The prompt asks for a specific response so we can validate it
	start := o.now()
	var probeCmd *exec.Cmd
	var cmdName string
	var codexOutputFile string
	probePrompt := "Reply with exactly one word: PROBE_OK"

	// Use a timeout context for the probe (30 seconds should be plenty)
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch runnerType {
	case domain.RunnerTypeClaudeCode:
		cmdName = "claude"
		// Use print mode for non-interactive, max tokens to limit response
		probeCmd = exec.CommandContext(probeCtx, cmdName, "-p", "--output-format", "text", probePrompt)
	case domain.RunnerTypeCodex:
		cmdName = "codex"
		// Use exec subcommand for non-interactive execution
		// --skip-git-repo-check allows running from /tmp without a git repo
		// -o writes just the response to a file (avoids session metadata in stdout)
		codexOutputFile = fmt.Sprintf("/tmp/codex-probe-%s.txt", uuid.New().String()[:8])
		probeCmd = exec.CommandContext(probeCtx, cmdName, "exec", "--skip-git-repo-check", "-o", codexOutputFile, probePrompt)
	case domain.RunnerTypeOpenCode:
		cmdName = "opencode"
		// Use run subcommand
		probeCmd = exec.CommandContext(probeCtx, cmdName, "run", probePrompt)
	case domain.RunnerTypeGrok:
		cmdName = "grok"
		// Headless single-turn; plain output is enough for a one-word probe
		// and one turn cannot reach a tool that would need approval.
		probeCmd = exec.CommandContext(probeCtx, cmdName, "-p", probePrompt, "--output-format", "plain", "--max-turns", "1")
	default:
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("unknown runner type: %s", runnerType),
		}, nil
	}

	// Run from a safe directory (temp) to avoid any project-specific behavior
	probeCmd.Dir = "/tmp"

	output, err := probeCmd.CombinedOutput()
	duration := time.Since(start)

	// For Codex, read the clean output from the file instead of stdout
	var outputStr string
	if codexOutputFile != "" {
		defer os.Remove(codexOutputFile) // Clean up temp file
		if fileContent, readErr := os.ReadFile(codexOutputFile); readErr == nil {
			outputStr = strings.TrimSpace(string(fileContent))
		} else {
			// Fall back to stdout if file read fails
			outputStr = strings.TrimSpace(string(output))
		}
	} else {
		outputStr = strings.TrimSpace(string(output))
	}

	// Strip ANSI escape codes for cleaner output and matching
	outputClean := stripANSI(outputStr)

	// Check for timeout
	if probeCtx.Err() == context.DeadlineExceeded {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("%s probe timed out after 30s", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Check for command execution error (non-zero exit code)
	if err != nil {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("%s probe failed: %v", cmdName, err),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Check for error patterns in output (some CLIs return exit 0 on failure)
	outputLower := strings.ToLower(outputClean)
	if strings.Contains(outputLower, "error:") ||
		strings.Contains(outputLower, "unauthorized") ||
		strings.Contains(outputLower, "authentication failed") ||
		strings.Contains(outputLower, "api key") ||
		strings.Contains(outputLower, "rate limit") {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("%s returned error in output", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Validate we got a meaningful response
	// The agent should have responded with something containing "PROBE_OK" or similar
	if strings.Contains(strings.ToUpper(outputClean), "PROBE_OK") ||
		strings.Contains(strings.ToUpper(outputClean), "PROBE OK") {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    true,
			Message:    fmt.Sprintf("%s responded correctly", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Got a response but not the expected one - still counts as working
	// (the agent might rephrase or add context, which is fine)
	if len(outputClean) > 0 {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    true,
			Message:    fmt.Sprintf("%s responded (content varies)", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Empty response is suspicious
	return &ProbeResult{
		RunnerType: runnerType,
		Success:    false,
		Message:    fmt.Sprintf("%s returned empty response", cmdName),
		Response:   "",
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

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Match ANSI escape sequences: ESC[ followed by params and a letter
	// This handles color codes, cursor movement, etc.
	result := strings.Builder{}
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip the '['
			continue
		}
		if inEscape {
			// End of escape sequence is a letter (A-Z, a-z)
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
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

// This file provides high-level retrieval and control operations for runs.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/pricing"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"

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

func (o *Orchestrator) ListExecutionOptions(ctx context.Context, roleRef string) ([]ExecutionOption, error) {
	if o.runners == nil {
		return nil, nil
	}
	// Resolve the role's candidate order and models when a role is supplied and
	// role policy is configured. The resolved policy is the catalog's model
	// truth; the local runner probe is merged only as additional entries.
	var resolved []rolepolicy.ResolvedCandidate
	if strings.TrimSpace(roleRef) != "" && o.rolePolicy != nil && o.roleResolver != nil {
		if res, err := o.rolePolicy.Resolve(ctx, o.roleResolver, roleRef); err == nil && res != nil {
			resolved = res.Candidates
		}
	}
	byRunner := map[domain.RunnerType]rolepolicy.ResolvedCandidate{}
	order := make([]domain.RunnerType, 0, len(resolved))
	for _, candidate := range resolved {
		if _, ok := byRunner[candidate.Runner]; ok {
			continue
		}
		byRunner[candidate.Runner] = candidate
		order = append(order, candidate.Runner)
	}
	options := make([]ExecutionOption, 0)
	if len(order) == 0 {
		for _, r := range o.runners.List() {
			options = append(options, executionOptionForRunner(ctx, r, nil))
		}
		return options, nil
	}
	for _, rt := range order {
		r, err := o.runners.Get(rt)
		if err != nil {
			continue
		}
		candidate := byRunner[rt]
		options = append(options, executionOptionForRunner(ctx, r, &candidate))
	}
	return options, nil
}

// executionOptionForRunner keeps a malformed optional adapter from taking
// down the read-only catalog. A runner that cannot publish capabilities is
// represented as unavailable with its diagnostic, which is safer than
// claiming support based on its type alone. When resolved is non-nil, its
// role-policy model is the default and its models are listed with
// source=role_policy; the runner's own locally-probed models are merged as
// source=local_probe.
func executionOptionForRunner(ctx context.Context, r runner.Runner, resolved *rolepolicy.ResolvedCandidate) (option ExecutionOption) {
	option.Message = "runner adapter did not publish execution capabilities"
	defer func() {
		if recovered := recover(); recovered != nil {
			option.Available = false
			option.Models = nil
			option.DefaultModel = ""
			option.Message = fmt.Sprintf("runner capability probe failed: %v", recovered)
		}
	}()
	if r == nil {
		option.Message = "runner adapter is nil"
		return option
	}
	option.RunnerType = string(r.Type())
	available, message := r.IsAvailable(ctx)
	option.Available = available
	option.Message = message
	capabilities := r.Capabilities()
	for _, capability := range capabilities.SpawnCapabilities {
		if capability.NativeObjective {
			option.NativeObjective = true
			option.SandboxModesWithNativeObjective = append(option.SandboxModesWithNativeObjective, capability.SandboxModes...)
		}
	}
	if !available {
		return option
	}
	seenModel := map[string]bool{}
	addModel := func(id, canonical, source string, isDefault bool) {
		id = strings.TrimSpace(id)
		if id == "" || seenModel[id] {
			return
		}
		seenModel[id] = true
		option.Models = append(option.Models, ExecutionModelOption{ID: id, CanonicalModel: canonical, Source: source, IsDefault: isDefault})
	}
	if resolved != nil {
		option.DefaultModel = strings.TrimSpace(resolved.Model)
		if option.DefaultModel == "" {
			option.DefaultModel = strings.TrimSpace(resolved.CanonicalModel)
		}
		if option.DefaultModel != "" {
			option.DefaultModelSource = executionModelSourceRolePolicy
		}
		addModel(option.DefaultModel, resolved.CanonicalModel, executionModelSourceRolePolicy, option.DefaultModel != "")
		for _, fallback := range resolved.Fallbacks {
			addModel(fallback, "", executionModelSourceRolePolicy, false)
		}
	} else if len(capabilities.SupportedModels) > 0 {
		// No role resolved: the first probed model is the runner's default.
		option.DefaultModel = capabilities.SupportedModels[0]
		option.DefaultModelSource = executionModelSourceLocalProbe
		addModel(option.DefaultModel, option.DefaultModel, executionModelSourceLocalProbe, true)
	}
	for _, model := range capabilities.SupportedModels {
		addModel(model, model, executionModelSourceLocalProbe, option.DefaultModel == "" && len(option.Models) == 0)
	}
	if option.DefaultModel == "" && len(option.Models) > 0 {
		option.DefaultModel = option.Models[0].ID
		option.Models[0].IsDefault = true
	}
	option.EffortLevels = orderedEffortLevels(capabilities.EffortMappings)
	return option
}

const (
	executionModelSourceRolePolicy = "role_policy"
	executionModelSourceLocalProbe = "local_probe"
)

// effortLevelOrder is the deterministic display order for reasoning effort.
var effortLevelOrder = []domain.Effort{
	domain.EffortLow, domain.EffortMedium, domain.EffortHigh, domain.EffortXHigh, domain.EffortMax,
}

// orderedEffortLevels returns the declared effort levels in fixed low→max order
// instead of Go map order, with any unknown level appended sorted.
func orderedEffortLevels(mappings map[string]string) []string {
	declared := map[string]bool{}
	for level := range mappings {
		declared[level] = true
	}
	ordered := make([]string, 0, len(declared))
	for _, effort := range effortLevelOrder {
		level := string(effort)
		if declared[level] {
			ordered = append(ordered, level)
			delete(declared, level)
		}
	}
	extra := make([]string, 0, len(declared))
	for level := range declared {
		extra = append(extra, level)
	}
	sort.Strings(extra)
	return append(ordered, extra...)
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
		o.notifyConversationSearch(ctx, "delete_run", id.String(), "")
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
	if err := appendAndBroadcastEvents(ctx, o.events, o.broadcaster, runID, events...); err != nil {
		return err
	}
	for _, evt := range events {
		if evt == nil {
			continue
		}
		o.nudgeWorkflowUsage(runID, evt)
		switch evt.EventType {
		case domain.EventTypeMessage, domain.EventTypeToolCall, domain.EventTypeToolResult:
			o.notifyConversationSearch(ctx, "upsert_run", runID.String(), evt.ID.String())
		case domain.EventTypeMessageDeleted:
			target := ""
			if data, ok := evt.Data.(*domain.MessageDeletedEventData); ok {
				target = data.TargetEventID
			}
			o.notifyConversationSearch(ctx, "delete_event", runID.String(), target)
		}
	}
	return nil
}

// eventStoreAdapter adapts event.Store to runner.EventSink
type eventStoreAdapter struct {
	mu           sync.Mutex
	store        event.Store
	runID        uuid.UUID
	lastSequence int64
	afterPersist func(*domain.RunEvent)
}

func (e *eventStoreAdapter) Emit(evt *domain.RunEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.store.Append(context.Background(), e.runID, evt); err != nil {
		return err
	}
	e.lastSequence = evt.Sequence
	if e.afterPersist != nil {
		e.afterPersist(evt)
	}
	return nil
}

func (e *eventStoreAdapter) Close() error {
	return nil
}

func (e *eventStoreAdapter) LastSequence() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastSequence
}

// broadcastingEventSink stores events AND broadcasts them via WebSocket.
type broadcastingEventSink struct {
	store        event.Store
	runID        uuid.UUID
	broadcaster  EventBroadcaster
	lastSequence int64
	afterPersist func(*domain.RunEvent)
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
	if b.afterPersist != nil {
		b.afterPersist(evt)
	}

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
	afterPersist := func(evt *domain.RunEvent) {
		o.nudgeWorkflowUsage(runID, evt)
		o.observeQuotaEvent(evt)
	}
	switch {
	case o.events != nil && o.broadcaster != nil:
		return &broadcastingEventSink{
			store:        o.events,
			runID:        runID,
			broadcaster:  o.broadcaster,
			afterPersist: afterPersist,
		}
	case o.events != nil:
		return &eventStoreAdapter{store: o.events, runID: runID, afterPersist: afterPersist}
	default:
		return &noOpEventSink{}
	}
}

func (o *Orchestrator) observeQuotaEvent(evt *domain.RunEvent) {
	if o == nil || o.quotaObservations == nil || evt == nil {
		return
	}
	data, ok := evt.Data.(*domain.RateLimitEventData)
	if !ok || strings.TrimSpace(data.Provider) == "" || strings.TrimSpace(data.Pool) == "" {
		return
	}
	provider := data.Provider
	pool := data.Pool
	provenance := strings.TrimSpace(data.Provenance)
	if provenance == "" {
		provenance = "agent-manager:run-event.rate-limit"
	}
	observation, err := pricing.FromRateLimitEvent(provider, pool, provenance, evt.Timestamp, data)
	if err != nil {
		obs.Component("quota-observation").Warn("provider quota frame rejected", obs.KeyRunID, evt.RunID.String(), obs.KeyError, err.Error())
		return
	}
	observation.SourceRunID = evt.RunID.String()
	observation.EvidenceRef = "agent-manager://runs/" + evt.RunID.String() + "/events/" + evt.ID.String()
	observation.ID = pricing.QuotaObservationID(observation.SourceRunID, observation)
	now := o.now().UTC()
	if !evt.Timestamp.IsZero() && !evt.Timestamp.After(now) {
		observation.Freshness = pricing.ObservationFresh
	} else {
		observation.Freshness = pricing.ObservationUnknown
	}
	if err := o.quotaObservations.RecordQuotaObservation(context.Background(), observation); err != nil {
		obs.Component("quota-observation").Warn("provider quota frame persistence failed", obs.KeyRunID, evt.RunID.String(), obs.KeyError, err.Error())
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

// Package fixtures provides domain-object factories for tests.
//
// Each factory returns a sane default object and accepts option functions for
// fields that matter to a specific test. Defaults should be valid enough for
// repository seeding and orchestration setup without repeated literals.
package fixtures

import (
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// AgentProfileOpt mutates an AgentProfile during construction.
type AgentProfileOpt func(*domain.AgentProfile)

// TaskOpt mutates a Task during construction.
type TaskOpt func(*domain.Task)

// RunOpt mutates a Run during construction.
type RunOpt func(*domain.Run)

// SandboxConfigOpt mutates a SandboxConfig during construction.
type SandboxConfigOpt func(*domain.SandboxConfig)

// RunEventOpt mutates a RunEvent during construction.
type RunEventOpt func(*domain.RunEvent)

// NewAgentProfile returns a Claude Code profile with deterministic-enough
// defaults for repository-backed tests.
func NewAgentProfile(t *testing.T, opts ...AgentProfileOpt) *domain.AgentProfile {
	if t != nil {
		t.Helper()
	}
	now := time.Now()
	profile := &domain.AgentProfile{
		ID:            uuid.New(),
		Name:          "test-profile",
		ProfileKey:    "test-profile",
		Description:   "Test agent profile",
		RunnerType:    domain.RunnerTypeClaudeCode,
		Model:         "claude-3-opus",
		MaxTurns:      100,
		NetworkAccess: domain.NetworkAccessLocalhost,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	for _, opt := range opts {
		opt(profile)
	}
	return profile
}

func WithAgentProfileID(id uuid.UUID) AgentProfileOpt {
	return func(p *domain.AgentProfile) { p.ID = id }
}

func WithAgentProfileName(name string) AgentProfileOpt {
	return func(p *domain.AgentProfile) {
		p.Name = name
		p.ProfileKey = name
	}
}

func WithAgentProfileRunner(runnerType domain.RunnerType) AgentProfileOpt {
	return func(p *domain.AgentProfile) { p.RunnerType = runnerType }
}

func WithAgentProfileModel(model string) AgentProfileOpt {
	return func(p *domain.AgentProfile) { p.Model = model }
}

func WithAgentProfileSandboxConfig(cfg *domain.SandboxConfig) AgentProfileOpt {
	return func(p *domain.AgentProfile) { p.SandboxConfig = cfg }
}

// NewTask returns a queued task scoped to a test project.
func NewTask(t *testing.T, opts ...TaskOpt) *domain.Task {
	if t != nil {
		t.Helper()
	}
	now := time.Now()
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "A test task",
		ScopePath:   "src/",
		ProjectRoot: "/project",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	for _, opt := range opts {
		opt(task)
	}
	return task
}

func WithTaskID(id uuid.UUID) TaskOpt {
	return func(t *domain.Task) { t.ID = id }
}

func WithTaskTitle(title string) TaskOpt {
	return func(t *domain.Task) { t.Title = title }
}

func WithTaskDescription(description string) TaskOpt {
	return func(t *domain.Task) { t.Description = description }
}

func WithTaskScope(scopePath, projectRoot string) TaskOpt {
	return func(t *domain.Task) {
		t.ScopePath = scopePath
		t.ProjectRoot = projectRoot
	}
}

func WithTaskStatus(status domain.TaskStatus) TaskOpt {
	return func(t *domain.Task) { t.Status = status }
}

// NewRun returns a pending sandboxed run linked to the provided task/profile
// IDs. Pass uuid.Nil for either ID when the relationship is intentionally
// absent.
func NewRun(t *testing.T, taskID uuid.UUID, profileID uuid.UUID, opts ...RunOpt) *domain.Run {
	if t != nil {
		t.Helper()
	}
	now := time.Now()
	var profileIDPtr *uuid.UUID
	if profileID != uuid.Nil {
		id := profileID
		profileIDPtr = &id
	}
	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         taskID,
		AgentProfileID: profileIDPtr,
		Status:         domain.RunStatusPending,
		Phase:          domain.RunPhaseQueued,
		RunMode:        domain.RunModeSandboxed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for _, opt := range opts {
		opt(run)
	}
	return run
}

func WithRunID(id uuid.UUID) RunOpt {
	return func(r *domain.Run) { r.ID = id }
}

func WithRunStatus(status domain.RunStatus) RunOpt {
	return func(r *domain.Run) { r.Status = status }
}

func WithRunPhase(phase domain.RunPhase) RunOpt {
	return func(r *domain.Run) { r.Phase = phase }
}

func WithRunMode(mode domain.RunMode) RunOpt {
	return func(r *domain.Run) { r.RunMode = mode }
}

func WithRunSandboxConfig(cfg *domain.SandboxConfig) RunOpt {
	return func(r *domain.Run) { r.SandboxConfig = cfg }
}

func WithRunSandboxID(id uuid.UUID) RunOpt {
	return func(r *domain.Run) { r.SandboxID = &id }
}

func WithRunConversationID(id string) RunOpt {
	return func(r *domain.Run) { r.ConversationID = id }
}

// NewSandboxConfig returns a copy of the domain default sandbox contract.
func NewSandboxConfig(t *testing.T, opts ...SandboxConfigOpt) *domain.SandboxConfig {
	if t != nil {
		t.Helper()
	}
	cfg := domain.DefaultSandboxConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func WithSandboxMode(mode domain.SandboxMode) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) { c.Mode = mode }
}

func WithSandboxManualReview(manualReview bool) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) { c.ManualReview = manualReview }
}

func WithSandboxAutoApply(autoApply bool) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) { c.AutoApply = &autoApply }
}

func WithSandboxApplyOnFailure(applyOnFailure bool) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) { c.ApplyOnFailure = &applyOnFailure }
}

func WithSandboxDeleteOn(events ...domain.SandboxLifecycleEvent) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) {
		c.Lifecycle.DeleteOn = append([]domain.SandboxLifecycleEvent(nil), events...)
	}
}

func WithSandboxStopOn(events ...domain.SandboxLifecycleEvent) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) {
		c.Lifecycle.StopOn = append([]domain.SandboxLifecycleEvent(nil), events...)
	}
}

// NewRunEvent returns a log event for the given run, with override hooks for
// tests that need a different event type or deterministic metadata.
func NewRunEvent(t *testing.T, runID uuid.UUID, opts ...RunEventOpt) *domain.RunEvent {
	if t != nil {
		t.Helper()
	}
	evt := domain.NewLogEvent(runID, "info", "test event")
	for _, opt := range opts {
		opt(evt)
	}
	return evt
}

func WithRunEventID(id uuid.UUID) RunEventOpt {
	return func(e *domain.RunEvent) { e.ID = id }
}

func WithRunEventSequence(sequence int64) RunEventOpt {
	return func(e *domain.RunEvent) { e.Sequence = sequence }
}

func WithRunEventTimestamp(ts time.Time) RunEventOpt {
	return func(e *domain.RunEvent) { e.Timestamp = ts }
}

func WithRunEventPayload(eventType domain.RunEventType, payload domain.EventPayload) RunEventOpt {
	return func(e *domain.RunEvent) {
		e.EventType = eventType
		e.Data = payload
	}
}

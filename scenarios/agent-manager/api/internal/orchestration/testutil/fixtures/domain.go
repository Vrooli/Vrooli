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
		RoleRef:       "code.default",
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

func WithAgentProfileName(name string) AgentProfileOpt {
	return func(p *domain.AgentProfile) {
		p.Name = name
		p.ProfileKey = name
	}
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

func WithTaskTitle(title string) TaskOpt {
	return func(t *domain.Task) { t.Title = title }
}

func WithTaskDescription(description string) TaskOpt {
	return func(t *domain.Task) { t.Description = description }
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

func WithRunStatus(status domain.RunStatus) RunOpt {
	return func(r *domain.Run) { r.Status = status }
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

func WithSandboxManualReview(manualReview bool) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) { c.ManualReview = manualReview }
}

func WithSandboxAutoApply(autoApply bool) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) { c.AutoApply = &autoApply }
}

func WithSandboxDeleteOn(events ...domain.SandboxLifecycleEvent) SandboxConfigOpt {
	return func(c *domain.SandboxConfig) {
		c.Lifecycle.DeleteOn = append([]domain.SandboxLifecycleEvent(nil), events...)
	}
}

// NewRunEvent returns an info log event for the given run.
func NewRunEvent(t *testing.T, runID uuid.UUID) *domain.RunEvent {
	if t != nil {
		t.Helper()
	}
	return domain.NewLogEvent(runID, "info", "test event")
}

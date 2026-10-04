// Responsibility: operate profile and task records through orchestration repositories.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"context"
	"github.com/google/uuid"
	"strings"
)

func (o *Orchestrator) CreateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error) {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	profile.CreatedAt = o.now()
	profile.UpdatedAt = profile.CreatedAt

	if err := normalizeProfileInput(profile); err != nil {
		return nil, err
	}

	if err := o.profiles.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (o *Orchestrator) GetProfile(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error) {
	profile, err := o.profiles.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, domain.NewNotFoundError("AgentProfile", id)
	}
	return profile, nil
}

func (o *Orchestrator) ListProfiles(ctx context.Context, opts ListOptions) ([]*domain.AgentProfile, error) {
	return o.profiles.List(ctx, repository.ListFilter{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (o *Orchestrator) UpdateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error) {
	existing, err := o.profiles.Get(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.SourcePath != "" {
		profile.OwnerScenario = existing.OwnerScenario
		profile.SourcePath = existing.SourcePath
		profile.SourceHash = existing.SourceHash
		profile.LastAppliedHash = existing.LastAppliedHash
		profile.SourceUpdatedAt = existing.SourceUpdatedAt
		profile.LocalOverride = true
	}
	profile.UpdatedAt = o.now()
	if existing != nil {
		profile.CreatedAt = existing.CreatedAt
	}

	if err := normalizeProfileInput(profile); err != nil {
		return nil, err
	}

	if err := o.profiles.Update(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (o *Orchestrator) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	return o.profiles.Delete(ctx, id)
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed.
func (o *Orchestrator) EnsureProfile(ctx context.Context, req EnsureProfileRequest) (*EnsureProfileResult, error) {
	key := strings.TrimSpace(req.ProfileKey)
	if key == "" {
		return nil, domain.NewValidationErrorWithHint("profileKey", "field is required",
			"Provide a stable profile key for lookup or creation")
	}

	existing, err := o.profiles.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if existing != nil && !req.UpdateExisting {
		return &EnsureProfileResult{Profile: existing}, nil
	}

	defaults := req.Defaults
	if defaults == nil {
		return nil, domain.NewValidationErrorWithHint("defaults", "field is required",
			"Provide default profile settings to create a new profile")
	}

	candidate := *defaults
	candidate.ProfileKey = key
	if strings.TrimSpace(candidate.Name) == "" {
		candidate.Name = key
	}

	now := o.now()
	if existing == nil {
		if candidate.ID == uuid.Nil {
			candidate.ID = uuid.New()
		}
		candidate.CreatedAt = now
		candidate.UpdatedAt = now

		if err := normalizeProfileInput(&candidate); err != nil {
			return nil, err
		}
		if err := o.profiles.Create(ctx, &candidate); err != nil {
			return nil, err
		}
		return &EnsureProfileResult{Profile: &candidate, Created: true}, nil
	}

	candidate.ID = existing.ID
	candidate.CreatedAt = existing.CreatedAt
	candidate.UpdatedAt = now
	if candidate.CreatedBy == "" {
		candidate.CreatedBy = existing.CreatedBy
	}

	if err := normalizeProfileInput(&candidate); err != nil {
		return nil, err
	}
	if err := o.profiles.Update(ctx, &candidate); err != nil {
		return nil, err
	}

	return &EnsureProfileResult{Profile: &candidate, Updated: true}, nil
}

func normalizeProfileInput(profile *domain.AgentProfile) error {
	if profile == nil {
		return domain.NewValidationError("profile", "cannot be nil")
	}

	name := strings.TrimSpace(profile.Name)
	key := strings.TrimSpace(profile.ProfileKey)
	if key == "" && name != "" {
		profile.ProfileKey = name
		key = name
	}
	if name == "" && key != "" {
		profile.Name = key
	}

	return profile.Validate()
}

// -----------------------------------------------------------------------------
// Task Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	task.Status = domain.TaskStatusQueued
	task.CreatedAt = o.now()
	task.UpdatedAt = task.CreatedAt

	if err := o.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (o *Orchestrator) GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, err := o.tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.NewNotFoundError("Task", id)
	}
	return task, nil
}

func (o *Orchestrator) ListTasks(ctx context.Context, opts ListOptions) ([]*domain.Task, error) {
	return o.tasks.List(ctx, repository.ListFilter{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (o *Orchestrator) UpdateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if task == nil {
		return nil, domain.NewValidationError("task", "cannot be nil")
	}

	existing, err := o.tasks.Get(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.NewNotFoundError("Task", task.ID)
	}

	// Preserve immutable/system-managed fields.
	updated := *existing
	updated.Title = task.Title
	updated.Description = task.Description
	updated.ScopePath = task.ScopePath
	updated.ProjectRoot = task.ProjectRoot
	updated.ContextAttachments = task.ContextAttachments
	updated.UpdatedAt = o.now()

	if err := o.tasks.Update(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (o *Orchestrator) CancelTask(ctx context.Context, id uuid.UUID) error {
	task, err := o.GetTask(ctx, id)
	if err != nil {
		return err
	}

	if task.Status != domain.TaskStatusQueued && task.Status != domain.TaskStatusRunning {
		return domain.NewStateError("Task", string(task.Status), "cancel", "can only cancel queued or running tasks")
	}

	task.Status = domain.TaskStatusCancelled
	task.UpdatedAt = o.now()
	return o.tasks.Update(ctx, task)
}

func (o *Orchestrator) DeleteTask(ctx context.Context, id uuid.UUID) error {
	task, err := o.GetTask(ctx, id)
	if err != nil {
		return err
	}

	if task.Status != domain.TaskStatusCancelled {
		return domain.NewStateError("Task", string(task.Status), "delete", "can only delete cancelled tasks")
	}

	return o.tasks.Delete(ctx, id)
}

// -----------------------------------------------------------------------------
// Run Operations
// -----------------------------------------------------------------------------

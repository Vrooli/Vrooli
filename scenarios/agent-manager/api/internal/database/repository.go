package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Repositories holds all PostgreSQL repository implementations.
type Repositories struct {
	Profiles     repository.ProfileRepository
	Tasks        repository.TaskRepository
	Runs         repository.RunRepository
	Events       repository.EventRepository
	Checkpoints  repository.CheckpointRepository
	Idempotency  repository.IdempotencyRepository
	Policies     repository.PolicyRepository
	Locks        repository.LockRepository
}

// NewRepositories creates all repository implementations using the given database connection.
func NewRepositories(db *DB, log *logrus.Logger) *Repositories {
	return &Repositories{
		Profiles:     &profileRepository{db: db, log: log},
		Tasks:        &taskRepository{db: db, log: log},
		Runs:         &runRepository{db: db, log: log},
		Events:       &eventRepository{db: db, log: log},
		Checkpoints:  &checkpointRepository{db: db, log: log},
		Idempotency:  &idempotencyRepository{db: db, log: log},
		Policies:     &policyRepository{db: db, log: log},
		Locks:        &lockRepository{db: db, log: log},
	}
}

// Helper for pagination
func appendLimitOffset(query string, limit, offset int) (string, []interface{}) {
	var args []interface{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}
	return query, args
}

// ============================================================================
// ProfileRepository Implementation
// ============================================================================

type profileRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.ProfileRepository = (*profileRepository)(nil)

// profileRow is the database row representation for agent_profiles.
type profileRow struct {
	ID                   uuid.UUID   `db:"id"`
	Name                 string      `db:"name"`
	Description          string      `db:"description"`
	RunnerType           string      `db:"runner_type"`
	Model                string      `db:"model"`
	MaxTurns             int         `db:"max_turns"`
	TimeoutMs            int64       `db:"timeout_ms"`
	AllowedTools         StringSlice `db:"allowed_tools"`
	DeniedTools          StringSlice `db:"denied_tools"`
	SkipPermissionPrompt bool        `db:"skip_permission_prompt"`
	RequiresSandbox      bool        `db:"requires_sandbox"`
	RequiresApproval     bool        `db:"requires_approval"`
	AllowedPaths         StringSlice `db:"allowed_paths"`
	DeniedPaths          StringSlice `db:"denied_paths"`
	CreatedBy            string      `db:"created_by"`
	CreatedAt            SQLiteTime  `db:"created_at"`
	UpdatedAt            SQLiteTime  `db:"updated_at"`
}

func (r *profileRow) toDomain() *domain.AgentProfile {
	return &domain.AgentProfile{
		ID:                   r.ID,
		Name:                 r.Name,
		Description:          r.Description,
		RunnerType:           domain.RunnerType(r.RunnerType),
		Model:                r.Model,
		MaxTurns:             r.MaxTurns,
		Timeout:              time.Duration(r.TimeoutMs) * time.Millisecond,
		AllowedTools:         r.AllowedTools,
		DeniedTools:          r.DeniedTools,
		SkipPermissionPrompt: r.SkipPermissionPrompt,
		RequiresSandbox:      r.RequiresSandbox,
		RequiresApproval:     r.RequiresApproval,
		AllowedPaths:         r.AllowedPaths,
		DeniedPaths:          r.DeniedPaths,
		CreatedBy:            r.CreatedBy,
		CreatedAt:            r.CreatedAt.Time(),
		UpdatedAt:            r.UpdatedAt.Time(),
	}
}

func profileFromDomain(p *domain.AgentProfile) *profileRow {
	return &profileRow{
		ID:                   p.ID,
		Name:                 p.Name,
		Description:          p.Description,
		RunnerType:           string(p.RunnerType),
		Model:                p.Model,
		MaxTurns:             p.MaxTurns,
		TimeoutMs:            int64(p.Timeout / time.Millisecond),
		AllowedTools:         p.AllowedTools,
		DeniedTools:          p.DeniedTools,
		SkipPermissionPrompt: p.SkipPermissionPrompt,
		RequiresSandbox:      p.RequiresSandbox,
		RequiresApproval:     p.RequiresApproval,
		AllowedPaths:         p.AllowedPaths,
		DeniedPaths:          p.DeniedPaths,
		CreatedBy:            p.CreatedBy,
		CreatedAt:            SQLiteTime(p.CreatedAt),
		UpdatedAt:            SQLiteTime(p.UpdatedAt),
	}
}

const profileColumns = `id, name, description, runner_type, model, max_turns, timeout_ms,
	allowed_tools, denied_tools, skip_permission_prompt, requires_sandbox, requires_approval,
	allowed_paths, denied_paths, created_by, created_at, updated_at`

func (r *profileRepository) Create(ctx context.Context, profile *domain.AgentProfile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	row := profileFromDomain(profile)
	query := `INSERT INTO agent_profiles (id, name, description, runner_type, model, max_turns, timeout_ms,
		allowed_tools, denied_tools, skip_permission_prompt, requires_sandbox, requires_approval,
		allowed_paths, denied_paths, created_by, created_at, updated_at)
		VALUES (:id, :name, :description, :runner_type, :model, :max_turns, :timeout_ms,
		:allowed_tools, :denied_tools, :skip_permission_prompt, :requires_sandbox, :requires_approval,
		:allowed_paths, :denied_paths, :created_by, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create agent profile")
		return fmt.Errorf("failed to create profile: %w", err)
	}
	return nil
}

func (r *profileRepository) Get(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM agent_profiles WHERE id = ?", profileColumns))
	var row profileRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return row.toDomain(), nil
}

func (r *profileRepository) GetByName(ctx context.Context, name string) (*domain.AgentProfile, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM agent_profiles WHERE name = ?", profileColumns))
	var row profileRow
	if err := r.db.GetContext(ctx, &row, query, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get profile by name: %w", err)
	}
	return row.toDomain(), nil
}

func (r *profileRepository) List(ctx context.Context, filter repository.ListFilter) ([]*domain.AgentProfile, error) {
	base := fmt.Sprintf("SELECT %s FROM agent_profiles ORDER BY updated_at DESC", profileColumns)
	queryWithPaging, args := appendLimitOffset(base, filter.Limit, filter.Offset)
	query := r.db.Rebind(queryWithPaging)

	var rows []profileRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	result := make([]*domain.AgentProfile, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *domain.AgentProfile) error {
	profile.UpdatedAt = time.Now()
	row := profileFromDomain(profile)

	query := `UPDATE agent_profiles SET name = :name, description = :description,
		runner_type = :runner_type, model = :model, max_turns = :max_turns, timeout_ms = :timeout_ms,
		allowed_tools = :allowed_tools, denied_tools = :denied_tools,
		skip_permission_prompt = :skip_permission_prompt, requires_sandbox = :requires_sandbox,
		requires_approval = :requires_approval, allowed_paths = :allowed_paths, denied_paths = :denied_paths,
		created_by = :created_by, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}
	return nil
}

func (r *profileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM agent_profiles WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	return nil
}

// ============================================================================
// TaskRepository Implementation
// ============================================================================

type taskRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.TaskRepository = (*taskRepository)(nil)

// taskRow is the database row representation for tasks.
type taskRow struct {
	ID                 uuid.UUID              `db:"id"`
	Title              string                 `db:"title"`
	Description        string                 `db:"description"`
	ScopePath          string                 `db:"scope_path"`
	ProjectRoot        string                 `db:"project_root"`
	PhasePromptIDs     UUIDSlice              `db:"phase_prompt_ids"`
	ContextAttachments ContextAttachmentSlice `db:"context_attachments"`
	Status             string                 `db:"status"`
	CreatedBy          string                 `db:"created_by"`
	CreatedAt          SQLiteTime             `db:"created_at"`
	UpdatedAt          SQLiteTime             `db:"updated_at"`
}

func (row *taskRow) toDomain() *domain.Task {
	return &domain.Task{
		ID:                 row.ID,
		Title:              row.Title,
		Description:        row.Description,
		ScopePath:          row.ScopePath,
		ProjectRoot:        row.ProjectRoot,
		PhasePromptIDs:     row.PhasePromptIDs,
		ContextAttachments: row.ContextAttachments,
		Status:             domain.TaskStatus(row.Status),
		CreatedBy:          row.CreatedBy,
		CreatedAt:          row.CreatedAt.Time(),
		UpdatedAt:          row.UpdatedAt.Time(),
	}
}

func taskFromDomain(t *domain.Task) *taskRow {
	return &taskRow{
		ID:                 t.ID,
		Title:              t.Title,
		Description:        t.Description,
		ScopePath:          t.ScopePath,
		ProjectRoot:        t.ProjectRoot,
		PhasePromptIDs:     t.PhasePromptIDs,
		ContextAttachments: t.ContextAttachments,
		Status:             string(t.Status),
		CreatedBy:          t.CreatedBy,
		CreatedAt:          SQLiteTime(t.CreatedAt),
		UpdatedAt:          SQLiteTime(t.UpdatedAt),
	}
}

const taskColumns = `id, title, description, scope_path, project_root,
	phase_prompt_ids, context_attachments, status, created_by, created_at, updated_at`

func (r *taskRepository) Create(ctx context.Context, task *domain.Task) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	row := taskFromDomain(task)
	query := `INSERT INTO tasks (id, title, description, scope_path, project_root,
		phase_prompt_ids, context_attachments, status, created_by, created_at, updated_at)
		VALUES (:id, :title, :description, :scope_path, :project_root,
		:phase_prompt_ids, :context_attachments, :status, :created_by, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create task")
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

func (r *taskRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM tasks WHERE id = ?", taskColumns))
	var row taskRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return row.toDomain(), nil
}

func (r *taskRepository) List(ctx context.Context, filter repository.ListFilter) ([]*domain.Task, error) {
	base := fmt.Sprintf("SELECT %s FROM tasks ORDER BY updated_at DESC", taskColumns)
	queryWithPaging, args := appendLimitOffset(base, filter.Limit, filter.Offset)
	query := r.db.Rebind(queryWithPaging)

	var rows []taskRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	result := make([]*domain.Task, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *taskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, filter repository.ListFilter) ([]*domain.Task, error) {
	base := fmt.Sprintf("SELECT %s FROM tasks WHERE status = ? ORDER BY updated_at DESC", taskColumns)
	queryWithPaging, args := appendLimitOffset(base, filter.Limit, filter.Offset)
	args = append([]interface{}{string(status)}, args...)
	query := r.db.Rebind(queryWithPaging)

	var rows []taskRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list tasks by status: %w", err)
	}

	result := make([]*domain.Task, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *taskRepository) Update(ctx context.Context, task *domain.Task) error {
	task.UpdatedAt = time.Now()
	row := taskFromDomain(task)

	query := `UPDATE tasks SET title = :title, description = :description,
		scope_path = :scope_path, project_root = :project_root,
		phase_prompt_ids = :phase_prompt_ids, context_attachments = :context_attachments,
		status = :status, created_by = :created_by, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM tasks WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

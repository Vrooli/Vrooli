package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/durability"
	"agent-manager/internal/findings"
	"agent-manager/internal/investigation"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Repositories holds all repository implementations.
type Repositories struct {
	Profiles              repository.ProfileRepository
	Workflows             repository.WorkflowRepository
	WorkflowExecutions    repository.WorkflowExecutionRepository
	Tasks                 repository.TaskRepository
	Runs                  repository.RunRepository
	Events                repository.EventRepository
	Checkpoints           repository.CheckpointRepository
	Idempotency           repository.IdempotencyRepository
	Policies              repository.PolicyRepository
	Stats                 repository.StatsRepository
	InvestigationSettings repository.InvestigationSettingsRepository
	Findings              findings.Repository
	Investigations        investigation.Repository
	ReceiptEvidence       runreport.ReceiptJoinStore
	InvestigationLedger   runreport.LedgerStore
	InvocationReadModel   invocationreadmodel.Store
	DurabilityBoundary    durability.BoundaryStore
}

// NewRepositories creates all repository implementations using the given database connection.
func NewRepositories(db *DB, log *logrus.Logger) *Repositories {
	return &Repositories{
		Profiles:              &profileRepository{db: db, log: log},
		Workflows:             &workflowRepository{db: db, log: log},
		WorkflowExecutions:    &workflowExecutionRepository{db: db, log: log},
		Tasks:                 &taskRepository{db: db, log: log},
		Runs:                  &runRepository{db: db, log: log},
		Events:                &eventRepository{db: db, log: log},
		Checkpoints:           &checkpointRepository{db: db, log: log},
		Idempotency:           &idempotencyRepository{db: db, log: log},
		Policies:              &policyRepository{db: db, log: log},
		Stats:                 &statsRepository{db: db, log: log},
		InvestigationSettings: &investigationSettingsRepository{db: db, log: log},
		Findings:              findings.NewSQLiteRepository(db),
		Investigations:        investigation.NewSQLiteRepository(db),
		ReceiptEvidence:       &receiptEvidenceRepository{db: db},
		InvestigationLedger:   &receiptEvidenceRepository{db: db},
		InvocationReadModel:   &invocationReadModelRepository{db: db},
		DurabilityBoundary:    &durabilityBoundaryRepository{db: db},
	}
}

// Helper for pagination
func appendLimitOffset(query string, limit, offset int) (string, []interface{}) {
	var args []interface{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	} else if offset > 0 {
		// SQLite requires LIMIT before OFFSET; use -1 for unlimited
		query += " LIMIT -1"
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
	ID                    uuid.UUID                `db:"id"`
	Name                  string                   `db:"name"`
	ProfileKey            string                   `db:"profile_key"`
	Description           string                   `db:"description"`
	RoleRef               string                   `db:"role_ref"`
	MaxTurns              int                      `db:"max_turns"`
	TimeoutMs             int64                    `db:"timeout_ms"`
	Effort                string                   `db:"effort"`
	MaxWeightedTokens     int64                    `db:"max_weighted_tokens"`
	AllowedTools          StringSlice              `db:"allowed_tools"`
	DeniedTools           StringSlice              `db:"denied_tools"`
	ToolRestrictionPolicy string                   `db:"tool_restriction_policy"`
	SkipPermissionPrompt  bool                     `db:"skip_permission_prompt"`
	Features              NullableFeatureFlags     `db:"features"`
	ExtraFlags            NullableRunnerExtraFlags `db:"extra_flags"`
	NetworkAccess         string                   `db:"network_access"`
	SandboxConfig         NullableSandboxConfig    `db:"sandbox_config"`
	SpawnPolicy           string                   `db:"spawn_policy"`
	AllowedPaths          StringSlice              `db:"allowed_paths"`
	DeniedPaths           StringSlice              `db:"denied_paths"`
	DeclaredScopes        StringSlice              `db:"declared_scopes"`
	SkillPack             StringSlice              `db:"skill_pack"`
	SkillExperimentID     string                   `db:"skill_experiment_id"`
	CreatedBy             string                   `db:"created_by"`
	OwnerScenario         string                   `db:"owner_scenario"`
	SourcePath            string                   `db:"source_path"`
	SourceHash            string                   `db:"source_hash"`
	LastAppliedHash       string                   `db:"last_applied_hash"`
	SourceUpdatedAt       SQLiteTime               `db:"source_updated_at"`
	LocalOverride         bool                     `db:"local_override"`
	CreatedAt             SQLiteTime               `db:"created_at"`
	UpdatedAt             SQLiteTime               `db:"updated_at"`
}

func (r *profileRow) toDomain() *domain.AgentProfile {
	return &domain.AgentProfile{
		ID:                    r.ID,
		Name:                  r.Name,
		ProfileKey:            r.ProfileKey,
		Description:           r.Description,
		RoleRef:               r.RoleRef,
		MaxTurns:              r.MaxTurns,
		Timeout:               time.Duration(r.TimeoutMs) * time.Millisecond,
		Effort:                domain.Effort(r.Effort),
		MaxWeightedTokens:     r.MaxWeightedTokens,
		AllowedTools:          r.AllowedTools,
		DeniedTools:           r.DeniedTools,
		ToolRestrictionPolicy: domain.ToolRestrictionPolicy(r.ToolRestrictionPolicy).Effective(),
		SkipPermissionPrompt:  r.SkipPermissionPrompt,
		Features:              r.Features.V,
		ExtraFlags:            r.ExtraFlags.V,
		NetworkAccess:         domain.NetworkAccess(r.NetworkAccess),
		SandboxConfig:         r.SandboxConfig.V,
		SpawnPolicy:           decodeSpawnPolicy(r.SpawnPolicy),
		AllowedPaths:          r.AllowedPaths,
		DeniedPaths:           r.DeniedPaths,
		DeclaredScopes:        r.DeclaredScopes,
		SkillPack:             r.SkillPack,
		SkillExperimentID:     r.SkillExperimentID,
		CreatedBy:             r.CreatedBy,
		OwnerScenario:         r.OwnerScenario,
		SourcePath:            r.SourcePath,
		SourceHash:            r.SourceHash,
		LastAppliedHash:       r.LastAppliedHash,
		SourceUpdatedAt:       r.SourceUpdatedAt.Time(),
		LocalOverride:         r.LocalOverride,
		CreatedAt:             r.CreatedAt.Time(),
		UpdatedAt:             r.UpdatedAt.Time(),
	}
}

func profileFromDomain(p *domain.AgentProfile) *profileRow {
	spawnPolicy := "{}"
	if p.SpawnPolicy != nil {
		if data, err := json.Marshal(p.SpawnPolicy); err == nil {
			spawnPolicy = string(data)
		}
	}
	return &profileRow{
		ID:                    p.ID,
		Name:                  p.Name,
		ProfileKey:            p.ProfileKey,
		Description:           p.Description,
		RoleRef:               p.RoleRef,
		MaxTurns:              p.MaxTurns,
		TimeoutMs:             int64(p.Timeout / time.Millisecond),
		Effort:                string(p.Effort),
		MaxWeightedTokens:     p.MaxWeightedTokens,
		AllowedTools:          p.AllowedTools,
		DeniedTools:           p.DeniedTools,
		ToolRestrictionPolicy: string(p.ToolRestrictionPolicy.Effective()),
		SkipPermissionPrompt:  p.SkipPermissionPrompt,
		Features:              NullableFeatureFlags{V: p.Features},
		ExtraFlags:            NullableRunnerExtraFlags{V: p.ExtraFlags},
		NetworkAccess:         string(p.NetworkAccess),
		SandboxConfig:         NullableSandboxConfig{V: p.SandboxConfig},
		SpawnPolicy:           spawnPolicy,
		AllowedPaths:          p.AllowedPaths,
		DeniedPaths:           p.DeniedPaths,
		DeclaredScopes:        p.DeclaredScopes,
		SkillPack:             p.SkillPack,
		SkillExperimentID:     p.SkillExperimentID,
		CreatedBy:             p.CreatedBy,
		OwnerScenario:         p.OwnerScenario,
		SourcePath:            p.SourcePath,
		SourceHash:            p.SourceHash,
		LastAppliedHash:       p.LastAppliedHash,
		SourceUpdatedAt:       SQLiteTime(p.SourceUpdatedAt),
		LocalOverride:         p.LocalOverride,
		CreatedAt:             SQLiteTime(p.CreatedAt),
		UpdatedAt:             SQLiteTime(p.UpdatedAt),
	}
}

func decodeSpawnPolicy(raw string) *domain.SpawnPolicy {
	if raw == "" || raw == "{}" {
		return nil
	}
	var policy domain.SpawnPolicy
	if json.Unmarshal([]byte(raw), &policy) != nil || len(policy.AxisOrder) == 0 {
		return nil
	}
	return &policy
}

const profileColumns = `id, name, profile_key, description, role_ref, max_turns, timeout_ms,
	effort, max_weighted_tokens, allowed_tools, denied_tools, tool_restriction_policy, skip_permission_prompt, features, extra_flags,
	network_access, sandbox_config, spawn_policy, allowed_paths, denied_paths, declared_scopes, skill_pack, skill_experiment_id, created_by, owner_scenario, source_path,
	source_hash, last_applied_hash, source_updated_at, local_override, created_at, updated_at`

func (r *profileRepository) Create(ctx context.Context, profile *domain.AgentProfile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	row := profileFromDomain(profile)
	query := `INSERT INTO agent_profiles (id, name, profile_key, description, role_ref, max_turns, timeout_ms,
		effort, max_weighted_tokens, allowed_tools, denied_tools, tool_restriction_policy, skip_permission_prompt, features, extra_flags,
		network_access, sandbox_config, spawn_policy, allowed_paths, denied_paths, declared_scopes, skill_pack, skill_experiment_id, created_by, owner_scenario, source_path,
		source_hash, last_applied_hash, source_updated_at, local_override, created_at, updated_at)
		VALUES (:id, :name, :profile_key, :description, :role_ref, :max_turns, :timeout_ms,
		:effort, :max_weighted_tokens, :allowed_tools, :denied_tools, :tool_restriction_policy, :skip_permission_prompt, :features, :extra_flags,
		:network_access, :sandbox_config, :spawn_policy, :allowed_paths, :denied_paths, :declared_scopes, :skill_pack, :skill_experiment_id, :created_by, :owner_scenario, :source_path,
		:source_hash, :last_applied_hash, :source_updated_at, :local_override, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create agent profile")
		return wrapDBError("create", "AgentProfile", profile.ID.String(), err)
	}
	return nil
}

func (r *profileRepository) Get(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error) {
	query := fmt.Sprintf("SELECT %s FROM agent_profiles WHERE id = ?", profileColumns)
	var row profileRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get", "AgentProfile", id.String(), err)
	}
	return row.toDomain(), nil
}

func (r *profileRepository) GetByName(ctx context.Context, name string) (*domain.AgentProfile, error) {
	query := fmt.Sprintf("SELECT %s FROM agent_profiles WHERE name = ?", profileColumns)
	var row profileRow
	if err := r.db.GetContext(ctx, &row, query, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_by_name", "AgentProfile", name, err)
	}
	return row.toDomain(), nil
}

func (r *profileRepository) GetByKey(ctx context.Context, key string) (*domain.AgentProfile, error) {
	query := fmt.Sprintf("SELECT %s FROM agent_profiles WHERE profile_key = ?", profileColumns)
	var row profileRow
	if err := r.db.GetContext(ctx, &row, query, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_by_key", "AgentProfile", key, err)
	}
	return row.toDomain(), nil
}

func (r *profileRepository) List(ctx context.Context, filter repository.ListFilter) ([]*domain.AgentProfile, error) {
	base := fmt.Sprintf("SELECT %s FROM agent_profiles ORDER BY updated_at DESC", profileColumns)
	queryWithPaging, args := appendLimitOffset(base, filter.Limit, filter.Offset)
	query := queryWithPaging

	var rows []profileRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list", "AgentProfile", "", err)
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

	query := `UPDATE agent_profiles SET name = :name, profile_key = :profile_key, description = :description,
	role_ref = :role_ref, max_turns = :max_turns, timeout_ms = :timeout_ms, effort = :effort, max_weighted_tokens = :max_weighted_tokens,
		allowed_tools = :allowed_tools, denied_tools = :denied_tools, tool_restriction_policy = :tool_restriction_policy,
		skip_permission_prompt = :skip_permission_prompt, features = :features, extra_flags = :extra_flags,
		network_access = :network_access,
		sandbox_config = :sandbox_config, spawn_policy = :spawn_policy, allowed_paths = :allowed_paths, denied_paths = :denied_paths,
		declared_scopes = :declared_scopes, skill_pack = :skill_pack, skill_experiment_id = :skill_experiment_id,
		created_by = :created_by, owner_scenario = :owner_scenario, source_path = :source_path,
		source_hash = :source_hash, last_applied_hash = :last_applied_hash, source_updated_at = :source_updated_at,
		local_override = :local_override, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("update", "AgentProfile", profile.ID.String(), err)
	}
	return nil
}

func (r *profileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM agent_profiles WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return wrapDBError("delete", "AgentProfile", id.String(), err)
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
	CurrentRunID       NullableUUID           `db:"current_run_id"`
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
		CurrentRunID:       row.CurrentRunID.ToPtr(),
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
		CurrentRunID:       NewNullableUUID(t.CurrentRunID),
		CreatedBy:          t.CreatedBy,
		CreatedAt:          SQLiteTime(t.CreatedAt),
		UpdatedAt:          SQLiteTime(t.UpdatedAt),
	}
}

const taskColumns = `id, title, description, scope_path, project_root,
	phase_prompt_ids, context_attachments, status, current_run_id, created_by, created_at, updated_at`

func (r *taskRepository) Create(ctx context.Context, task *domain.Task) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	row := taskFromDomain(task)
	query := `INSERT INTO tasks (id, title, description, scope_path, project_root,
		phase_prompt_ids, context_attachments, status, current_run_id, created_by, created_at, updated_at)
		VALUES (:id, :title, :description, :scope_path, :project_root,
		:phase_prompt_ids, :context_attachments, :status, :current_run_id, :created_by, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create task")
		return wrapDBError("create", "Task", task.ID.String(), err)
	}
	return nil
}

func (r *taskRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	query := fmt.Sprintf("SELECT %s FROM tasks WHERE id = ?", taskColumns)
	var row taskRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get", "Task", id.String(), err)
	}
	return row.toDomain(), nil
}

func (r *taskRepository) List(ctx context.Context, filter repository.ListFilter) ([]*domain.Task, error) {
	base := fmt.Sprintf("SELECT %s FROM tasks ORDER BY updated_at DESC", taskColumns)
	queryWithPaging, args := appendLimitOffset(base, filter.Limit, filter.Offset)
	query := queryWithPaging

	var rows []taskRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list", "Task", "", err)
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
	query := queryWithPaging

	var rows []taskRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list_by_status", "Task", "", err)
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
		updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("update", "Task", task.ID.String(), err)
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = ? AND status = ? AND NOT EXISTS (
		SELECT 1 FROM runs WHERE runs.task_id = tasks.id AND runs.status NOT IN (?, ?, ?))`
	result, err := r.db.ExecContext(ctx, query, id, string(domain.TaskStatusCancelled), string(domain.RunStatusComplete), string(domain.RunStatusFailed), string(domain.RunStatusCancelled))
	if err != nil {
		return wrapDBError("delete", "Task", id.String(), err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return wrapDBError("delete_rows", "Task", id.String(), err)
	}
	if deleted != 1 {
		return domain.NewStateErrorWithID("Task", id.String(), "changed", "delete", "only a cancelled task with terminal attached runs can be deleted")
	}
	return nil
}

// CreateTaskAttempt atomically claims the Task projection and records the new
// Run plus its idempotency receipt. expectedCurrentRunID is nil for an
// unowned queued/legacy task with no live historical attempt, and otherwise
// must name the exact terminal attempt being replaced.
func (r *runRepository) CreateTaskAttempt(ctx context.Context, task *domain.Task, run *domain.Run, expectedCurrentRunID *uuid.UUID, expectedProjectRoot string) error {
	if task == nil || run == nil || run.TaskID == uuid.Nil || run.TaskID != task.ID {
		return fmt.Errorf("task attempt requires a matching task and run")
	}
	prepareRunForInsert(run)
	now := SQLiteTime(run.CreatedAt)
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("claim_task_attempt", "Task", task.ID.String(), err)
	}
	defer tx.Rollback()
	query := `UPDATE tasks SET status = ?, current_run_id = ?, project_root = ?, updated_at = ? WHERE id = ? AND project_root = ?`
	args := []any{string(domain.TaskStatusRunning), run.ID.String(), task.ProjectRoot, now, task.ID, expectedProjectRoot}
	if expectedCurrentRunID == nil {
		query += ` AND current_run_id IS NULL AND status IN (?, ?, ?) AND NOT EXISTS (
			SELECT 1 FROM runs WHERE runs.task_id = tasks.id AND runs.status IN (?, ?, ?, ?, ?))`
		args = append(args, string(domain.TaskStatusQueued), string(domain.TaskStatusCompleted), string(domain.TaskStatusFailed), string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning), string(domain.RunStatusNeedsReview), string(domain.RunStatusParked))
	} else {
		query += ` AND current_run_id = ? AND status IN (?, ?) AND EXISTS (
			SELECT 1 FROM runs WHERE runs.id = ? AND runs.task_id = tasks.id AND runs.status IN (?, ?, ?))`
		args = append(args, expectedCurrentRunID.String(), string(domain.TaskStatusCompleted), string(domain.TaskStatusFailed), expectedCurrentRunID.String(), string(domain.RunStatusComplete), string(domain.RunStatusFailed), string(domain.RunStatusCancelled))
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapDBError("claim_task_attempt", "Task", task.ID.String(), err)
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if claimed != 1 {
		return domain.NewStateErrorWithID("Task", task.ID.String(), string(task.Status), "start run", "task attempt changed or is not eligible; no run was admitted")
	}
	if err := r.insertRun(ctx, tx, run); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrapDBError("claim_task_attempt", "Task", task.ID.String(), err)
	}
	task.Status = domain.TaskStatusRunning
	id := run.ID
	task.CurrentRunID = &id
	task.UpdatedAt = run.UpdatedAt
	return nil
}

// CancelTaskAttempt atomically fences a Task's exact current Run before the
// orchestration layer cancels its in-memory dispatcher admission or invokes
// StopRun. Pending work is terminalized in the same transaction so startup
// recovery cannot enqueue a canceled attempt after process restart.
func (r *runRepository) CancelTaskAttempt(ctx context.Context, taskID uuid.UUID, runID *uuid.UUID, activeMayHaveStarted bool, at time.Time) (domain.RunStatus, bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", false, wrapDBError("cancel_task_attempt", "Task", taskID.String(), err)
	}
	defer tx.Rollback()

	var runStatus domain.RunStatus
	var runExists bool
	if runID != nil {
		if err := tx.QueryRowContext(ctx, `SELECT status FROM runs WHERE id = ? AND task_id = ?`, runID.String(), taskID).Scan(&runStatus); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return "", false, wrapDBError("cancel_task_attempt_read_run", "Run", runID.String(), err)
			}
			return "", false, domain.NewStateErrorWithID("Task", taskID.String(), "unknown", "cancel", "current Run association could not be verified")
		}
		runExists = true
	}

	taskQuery := `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ? AND current_run_id IS NULL
		AND NOT EXISTS (SELECT 1 FROM runs WHERE runs.task_id = tasks.id AND runs.status IN (?, ?, ?, ?, ?))`
	taskArgs := []any{string(domain.TaskStatusCancelled), SQLiteTime(at), taskID, string(domain.TaskStatusQueued),
		string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning), string(domain.RunStatusNeedsReview), string(domain.RunStatusParked)}
	if runID != nil {
		taskQuery = `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status IN (?, ?) AND current_run_id = ?`
		taskArgs = []any{
			string(domain.TaskStatusCancelled), SQLiteTime(at), taskID,
			string(domain.TaskStatusQueued), string(domain.TaskStatusRunning), runID.String(),
		}
	}
	result, err := tx.ExecContext(ctx, taskQuery, taskArgs...)
	if err != nil {
		return "", false, wrapDBError("cancel_task_attempt", "Task", taskID.String(), err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return "", false, err
	}
	if changed != 1 {
		return runStatus, false, nil
	}
	if runExists {
		runResult, err := tx.ExecContext(ctx, `UPDATE runs SET
			lifecycle_version = lifecycle_version + 1,
			status = CASE WHEN status = ? AND ? = 0 THEN ? ELSE status END,
			phase = CASE WHEN status = ? AND ? = 0 THEN ? ELSE phase END,
			ended_at = CASE WHEN status = ? AND ? = 0 THEN ? ELSE ended_at END,
			cancel_requested_at = CASE WHEN status IN (?, ?) OR (status = ? AND ? = 1) THEN COALESCE(cancel_requested_at, ?) ELSE cancel_requested_at END,
			updated_at = ?
			WHERE id = ? AND task_id = ? AND status IN (?, ?, ?, ?)`,
			string(domain.RunStatusPending), boolInt(activeMayHaveStarted), string(domain.RunStatusCancelled),
			string(domain.RunStatusPending), boolInt(activeMayHaveStarted), string(domain.RunPhaseCompleted),
			string(domain.RunStatusPending), boolInt(activeMayHaveStarted), SQLiteTime(at),
			string(domain.RunStatusStarting), string(domain.RunStatusRunning), string(domain.RunStatusPending), boolInt(activeMayHaveStarted), SQLiteTime(at), SQLiteTime(at),
			runID, taskID, string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning), string(domain.RunStatusParked))
		if err != nil {
			return "", false, wrapDBError("cancel_task_attempt_fence_run", "Run", runID.String(), err)
		}
		fenced, err := runResult.RowsAffected()
		if err != nil {
			return "", false, err
		}
		if fenced != 1 {
			return "", false, fmt.Errorf("task cancellation committed without fencing exact current run %s", runID)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", false, wrapDBError("cancel_task_attempt_commit", "Task", taskID.String(), err)
	}
	return runStatus, true, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// EnsureCurrentRun lazily associates a legacy Task with the exact Run named by
// the caller. It never chooses by timestamp or list order and refuses when a
// different current Run or any competing nonterminal historical Run exists.
func (r *runRepository) EnsureCurrentRun(ctx context.Context, taskID, runID uuid.UUID, at time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE tasks SET current_run_id = ?, updated_at = ?
		WHERE id = ? AND status NOT IN (?, ?, ?)
		AND EXISTS (SELECT 1 FROM runs target WHERE target.id = ? AND target.task_id = tasks.id)
		AND (current_run_id = ? OR (current_run_id IS NULL AND NOT EXISTS (
			SELECT 1 FROM runs other WHERE other.task_id = tasks.id AND other.id != ?
			AND other.status IN (?, ?, ?, ?, ?))))`,
		runID.String(), SQLiteTime(at), taskID,
		string(domain.TaskStatusCancelled), string(domain.TaskStatusApproved), string(domain.TaskStatusRejected),
		runID, runID.String(), runID,
		string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning), string(domain.RunStatusNeedsReview), string(domain.RunStatusParked))
	if err != nil {
		return false, wrapDBError("ensure_current_run", "Task", taskID.String(), err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return changed == 1, nil
}

func (r *runRepository) AdmitCurrentRun(ctx context.Context, run *domain.Run, expectedStatus domain.RunStatus, at time.Time) (bool, error) {
	if run == nil || run.TaskID == uuid.Nil {
		return false, fmt.Errorf("admit current run requires an exact task/run")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, wrapDBError("admit_current_run", "Run", run.ID.String(), err)
	}
	defer tx.Rollback()
	var lifecycleVersion int64
	err = tx.QueryRowContext(ctx, `UPDATE runs SET status = ?, updated_at = ?,
		lifecycle_version = lifecycle_version + CASE WHEN status <> ? THEN 1 ELSE 0 END
		WHERE id = ? AND task_id = ? AND status = ? AND lifecycle_version = ? AND cancel_requested_at IS NULL
		AND EXISTS (SELECT 1 FROM tasks WHERE tasks.id = runs.task_id
			AND tasks.current_run_id = runs.id AND tasks.status NOT IN (?, ?, ?))
		RETURNING lifecycle_version`,
		string(domain.RunStatusRunning), SQLiteTime(at), string(domain.RunStatusRunning), run.ID, run.TaskID, string(expectedStatus), run.LifecycleVersion,
		string(domain.TaskStatusCancelled), string(domain.TaskStatusApproved), string(domain.TaskStatusRejected)).Scan(&lifecycleVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, wrapDBError("admit_current_run", "Run", run.ID.String(), err)
	}
	taskResult, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ?, updated_at = ?
		WHERE id = ? AND current_run_id = ? AND status IN (?, ?, ?, ?, ?)`,
		string(domain.TaskStatusRunning), SQLiteTime(at), run.TaskID, run.ID,
		string(domain.TaskStatusQueued), string(domain.TaskStatusRunning), string(domain.TaskStatusNeedsReview), string(domain.TaskStatusCompleted), string(domain.TaskStatusFailed))
	if err != nil {
		return false, wrapDBError("project_admitted_current_run", "Task", run.TaskID.String(), err)
	}
	projected, err := taskResult.RowsAffected()
	if err != nil {
		return false, err
	}
	if projected != 1 {
		return false, fmt.Errorf("admitted Run %s lost its exact Task projection", run.ID)
	}
	if err := tx.Commit(); err != nil {
		return false, wrapDBError("admit_current_run_commit", "Run", run.ID.String(), err)
	}
	run.Status = domain.RunStatusRunning
	run.UpdatedAt = at
	run.LifecycleVersion = lifecycleVersion
	return true, nil
}

// ============================================================================
// InvestigationSettingsRepository Implementation
// ============================================================================

type investigationSettingsRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.InvestigationSettingsRepository = (*investigationSettingsRepository)(nil)

// investigationSettingsRow is the database row representation for investigation_settings.
// Prompt templates are no longer stored in the DB — they live in prompt-manager skills.
type investigationSettingsRow struct {
	DefaultDepth   string                        `db:"default_depth"`
	DefaultContext InvestigationContextFlagsJSON `db:"default_context"`
	TagAllowlist   InvestigationTagAllowlistJSON `db:"investigation_tag_allowlist"`
	UpdatedAt      SQLiteTime                    `db:"updated_at"`
}

// InvestigationContextFlagsJSON handles JSON serialization for context flags.
type InvestigationContextFlagsJSON domain.InvestigationContextFlags

func (j InvestigationContextFlagsJSON) Value() (interface{}, error) {
	return json.Marshal(domain.InvestigationContextFlags(j))
}

// InvestigationTagAllowlistJSON handles JSON serialization for tag allowlist rules.
type InvestigationTagAllowlistJSON []domain.InvestigationTagRule

func (j InvestigationTagAllowlistJSON) Value() (interface{}, error) {
	return json.Marshal([]domain.InvestigationTagRule(j))
}

func (j *InvestigationTagAllowlistJSON) Scan(src interface{}) error {
	if src == nil {
		*j = InvestigationTagAllowlistJSON(domain.DefaultInvestigationTagAllowlist())
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		*j = InvestigationTagAllowlistJSON(domain.DefaultInvestigationTagAllowlist())
		return nil
	}

	var rules []domain.InvestigationTagRule
	if err := json.Unmarshal(data, &rules); err != nil {
		*j = InvestigationTagAllowlistJSON(domain.DefaultInvestigationTagAllowlist())
		return nil
	}
	if len(rules) == 0 {
		rules = domain.DefaultInvestigationTagAllowlist()
	}
	*j = InvestigationTagAllowlistJSON(rules)
	return nil
}

func (j *InvestigationContextFlagsJSON) Scan(src interface{}) error {
	if src == nil {
		*j = InvestigationContextFlagsJSON(domain.DefaultInvestigationContextFlags())
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		*j = InvestigationContextFlagsJSON(domain.DefaultInvestigationContextFlags())
		return nil
	}

	var flags domain.InvestigationContextFlags
	if err := json.Unmarshal(data, &flags); err != nil {
		// On error, return defaults
		*j = InvestigationContextFlagsJSON(domain.DefaultInvestigationContextFlags())
		return nil
	}
	*j = InvestigationContextFlagsJSON(flags)
	return nil
}

func (row *investigationSettingsRow) toDomain() *domain.InvestigationSettings {
	return &domain.InvestigationSettings{
		// PromptTemplate and ApplyPromptTemplate are populated by the orchestration
		// layer from prompt-manager skills (with hardcoded constants as fallback).
		PromptTemplate:            domain.DefaultInvestigationPromptTemplate,
		ApplyPromptTemplate:       domain.DefaultApplyInvestigationPromptTemplate,
		DefaultDepth:              domain.InvestigationDepth(row.DefaultDepth),
		DefaultContext:            domain.InvestigationContextFlags(row.DefaultContext),
		InvestigationTagAllowlist: []domain.InvestigationTagRule(row.TagAllowlist),
		UpdatedAt:                 row.UpdatedAt.Time(),
	}
}

func (r *investigationSettingsRepository) Get(ctx context.Context) (*domain.InvestigationSettings, error) {
	query := `SELECT default_depth, default_context, investigation_tag_allowlist, updated_at
		FROM investigation_settings WHERE id = 1`

	var row investigationSettingsRow
	if err := r.db.GetContext(ctx, &row, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return defaults if no settings exist
			return domain.DefaultInvestigationSettings(), nil
		}
		return nil, wrapDBError("get", "InvestigationSettings", "singleton", err)
	}
	return row.toDomain(), nil
}

func (r *investigationSettingsRepository) Update(ctx context.Context, settings *domain.InvestigationSettings) error {
	settings.UpdatedAt = time.Now()

	contextJSON, err := json.Marshal(settings.DefaultContext)
	if err != nil {
		return wrapDBError("update", "InvestigationSettings", "singleton", err)
	}
	allowlistJSON, err := json.Marshal(domain.NormalizeInvestigationTagAllowlist(settings.InvestigationTagAllowlist))
	if err != nil {
		return wrapDBError("update", "InvestigationSettings", "singleton", err)
	}

	query := `
		INSERT INTO investigation_settings (id, default_depth, default_context, investigation_tag_allowlist, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			default_depth = EXCLUDED.default_depth,
			default_context = EXCLUDED.default_context,
			investigation_tag_allowlist = EXCLUDED.investigation_tag_allowlist,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		string(settings.DefaultDepth),
		contextJSON,
		allowlistJSON,
		time.Now().UTC().Format("2006-01-02 15:04:05.999999999"),
	)
	if err != nil {
		return wrapDBError("update", "InvestigationSettings", "singleton", err)
	}
	return nil
}

func (r *investigationSettingsRepository) Reset(ctx context.Context) error {
	defaults := domain.DefaultInvestigationSettings()
	return r.Update(ctx, defaults)
}

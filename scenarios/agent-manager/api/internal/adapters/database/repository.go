package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/findings"
	"agent-manager/internal/repository"

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
	Locks                 repository.LockRepository
	Stats                 repository.StatsRepository
	InvestigationSettings repository.InvestigationSettingsRepository
	Findings              findings.Repository
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
		Locks:                 &lockRepository{db: db, log: log},
		Stats:                 &statsRepository{db: db, log: log},
		InvestigationSettings: &investigationSettingsRepository{db: db, log: log},
		Findings:              findings.NewSQLiteRepository(db),
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
	AllowedTools          StringSlice              `db:"allowed_tools"`
	DeniedTools           StringSlice              `db:"denied_tools"`
	ToolRestrictionPolicy string                   `db:"tool_restriction_policy"`
	SkipPermissionPrompt  bool                     `db:"skip_permission_prompt"`
	Features              NullableFeatureFlags     `db:"features"`
	ExtraFlags            NullableRunnerExtraFlags `db:"extra_flags"`
	NetworkAccess         string                   `db:"network_access"`
	SandboxConfig         NullableSandboxConfig    `db:"sandbox_config"`
	AllowedPaths          StringSlice              `db:"allowed_paths"`
	DeniedPaths           StringSlice              `db:"denied_paths"`
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
		AllowedTools:          r.AllowedTools,
		DeniedTools:           r.DeniedTools,
		ToolRestrictionPolicy: domain.ToolRestrictionPolicy(r.ToolRestrictionPolicy).Effective(),
		SkipPermissionPrompt:  r.SkipPermissionPrompt,
		Features:              r.Features.V,
		ExtraFlags:            r.ExtraFlags.V,
		NetworkAccess:         domain.NetworkAccess(r.NetworkAccess),
		SandboxConfig:         r.SandboxConfig.V,
		AllowedPaths:          r.AllowedPaths,
		DeniedPaths:           r.DeniedPaths,
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
	return &profileRow{
		ID:                    p.ID,
		Name:                  p.Name,
		ProfileKey:            p.ProfileKey,
		Description:           p.Description,
		RoleRef:               p.RoleRef,
		MaxTurns:              p.MaxTurns,
		TimeoutMs:             int64(p.Timeout / time.Millisecond),
		Effort:                string(p.Effort),
		AllowedTools:          p.AllowedTools,
		DeniedTools:           p.DeniedTools,
		ToolRestrictionPolicy: string(p.ToolRestrictionPolicy.Effective()),
		SkipPermissionPrompt:  p.SkipPermissionPrompt,
		Features:              NullableFeatureFlags{V: p.Features},
		ExtraFlags:            NullableRunnerExtraFlags{V: p.ExtraFlags},
		NetworkAccess:         string(p.NetworkAccess),
		SandboxConfig:         NullableSandboxConfig{V: p.SandboxConfig},
		AllowedPaths:          p.AllowedPaths,
		DeniedPaths:           p.DeniedPaths,
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

const profileColumns = `id, name, profile_key, description, role_ref, max_turns, timeout_ms,
	effort, allowed_tools, denied_tools, tool_restriction_policy, skip_permission_prompt, features, extra_flags,
	network_access, sandbox_config, allowed_paths, denied_paths, created_by, owner_scenario, source_path,
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
		effort, allowed_tools, denied_tools, tool_restriction_policy, skip_permission_prompt, features, extra_flags,
		network_access, sandbox_config, allowed_paths, denied_paths, created_by, owner_scenario, source_path,
		source_hash, last_applied_hash, source_updated_at, local_override, created_at, updated_at)
		VALUES (:id, :name, :profile_key, :description, :role_ref, :max_turns, :timeout_ms,
		:effort, :allowed_tools, :denied_tools, :tool_restriction_policy, :skip_permission_prompt, :features, :extra_flags,
		:network_access, :sandbox_config, :allowed_paths, :denied_paths, :created_by, :owner_scenario, :source_path,
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
	role_ref = :role_ref, max_turns = :max_turns, timeout_ms = :timeout_ms, effort = :effort,
		allowed_tools = :allowed_tools, denied_tools = :denied_tools, tool_restriction_policy = :tool_restriction_policy,
		skip_permission_prompt = :skip_permission_prompt, features = :features, extra_flags = :extra_flags,
		network_access = :network_access,
		sandbox_config = :sandbox_config, allowed_paths = :allowed_paths, denied_paths = :denied_paths,
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
		status = :status, created_by = :created_by, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("update", "Task", task.ID.String(), err)
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return wrapDBError("delete", "Task", id.String(), err)
	}
	return nil
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

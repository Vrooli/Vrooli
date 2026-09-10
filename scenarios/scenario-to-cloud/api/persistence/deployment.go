package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
)

// deploymentColumns is the single projection every deployment read uses so a
// new column is added in one place and scanned by one helper.
const deploymentColumns = `
			id, name, scenario_id, environment, target_binding, fence, desired_state, persistent_data, status, manifest,
			bundle_path, bundle_sha256, bundle_size_bytes,
			setup_result, deploy_result, preflight_result, last_inspect_result, ssh_identity,
			error_message, error_step,
			progress_step, progress_percent,
			created_at, updated_at, last_deployed_at, last_inspected_at`

type rowScanner interface {
	Scan(dest ...any) error
}

// scanDeployment reads one row of deploymentColumns.
func scanDeployment(row rowScanner) (*domain.Deployment, error) {
	d := &domain.Deployment{}
	// manifest and target_binding are scanned through NullRawMessage because
	// PostgreSQL returns jsonb as bytes while a SQLite pool may hold TEXT.
	var binding, manifest domain.NullRawMessage
	var desired sql.NullString
	if err := row.Scan(
		&d.ID,
		&d.Name,
		&d.ScenarioID,
		&d.Environment,
		&binding,
		&d.Fence,
		&desired,
		&d.PersistentData,
		&d.Status,
		&manifest,
		&d.BundlePath,
		&d.BundleSHA256,
		&d.BundleSizeBytes,
		&d.SetupResult,
		&d.DeployResult,
		&d.PreflightResult,
		&d.LastInspectResult,
		&d.SSHIdentity,
		&d.ErrorMessage,
		&d.ErrorStep,
		&d.ProgressStep,
		&d.ProgressPercent,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.LastDeployedAt,
		&d.LastInspectedAt,
	); err != nil {
		return nil, err
	}
	d.Environment = identity.NormalizeEnvironment(d.Environment)
	d.Manifest = manifest.Data
	d.DesiredState = domain.DesiredState(desired.String).Normalized()
	if binding.Valid && len(binding.Data) > 0 {
		if err := json.Unmarshal(binding.Data, &d.Target); err != nil {
			return nil, fmt.Errorf("decode target binding for deployment %s: %w", d.ID, err)
		}
	}
	return d, nil
}

// UpdateDesiredState records the operator's intent for the workload. It
// never touches the observed status: a stopped intent is honoured by
// reconciliation, not by the health projection.
func (r *Repository) UpdateDesiredState(ctx context.Context, id string, state domain.DesiredState) error {
	switch state.Normalized() {
	case domain.DesiredRunning, domain.DesiredStopped, domain.DesiredRetired:
	default:
		return fmt.Errorf("unknown desired state %q", state)
	}
	const q = `UPDATE deployments SET desired_state = $2, updated_at = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id, string(state.Normalized()), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update desired_state: %w", err)
	}
	return requireRowsAffected(result, id)
}

// UpdatePersistentData records the legacy data-binding conversion.
func (r *Repository) UpdatePersistentData(ctx context.Context, id string, bindings domain.PersistentDataBindings) error {
	raw, err := json.Marshal(bindings)
	if err != nil {
		return fmt.Errorf("encode persistent data bindings: %w", err)
	}
	const q = `UPDATE deployments SET persistent_data = $2, updated_at = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id, raw, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update persistent_data: %w", err)
	}
	return requireRowsAffected(result, id)
}

// bindingForWrite serialises the target binding and its uniqueness key. A
// deployment created without an explicit binding derives one from the
// manifest so the identity index is always populated on write.
func bindingForWrite(d *domain.Deployment) ([]byte, *string, error) {
	if d.Target.IsZero() && len(d.Manifest) > 0 {
		var m domain.CloudManifest
		if err := json.Unmarshal(d.Manifest, &m); err == nil {
			d.Target = domain.TargetRefFromManifest(m)
		}
	}
	raw, err := json.Marshal(d.Target)
	if err != nil {
		return nil, nil, fmt.Errorf("encode target binding: %w", err)
	}
	return raw, targetKeyPtr(d.Target), nil
}

func targetKeyPtr(t identity.TargetRef) *string {
	key := t.Key()
	if key == "" {
		return nil
	}
	return &key
}

// isUniqueViolation recognises the identity index conflict on both engines
// without leaking driver types into callers.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key")
}

// CreateDeployment inserts a new deployment record. A second record for the
// same (scenario, environment, target) is a typed deployment_identity_conflict.
func (r *Repository) CreateDeployment(ctx context.Context, d *domain.Deployment) error {
	d.Environment = identity.NormalizeEnvironment(d.Environment)
	binding, targetKey, err := bindingForWrite(d)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO deployments (
			id, name, scenario_id, environment, target_binding, target_key, fence, desired_state, status, manifest,
			bundle_path, bundle_sha256, bundle_size_bytes,
			preflight_result, ssh_identity,
			error_message, error_step,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13,
			$14, $15,
			$16, $17,
			$18, $19
		)
	`
	_, err = r.db.ExecContext(ctx, q,
		d.ID,
		d.Name,
		d.ScenarioID,
		d.Environment,
		binding,
		targetKey,
		d.Fence,
		string(d.DesiredState.Normalized()),
		d.Status,
		d.Manifest,
		d.BundlePath,
		d.BundleSHA256,
		d.BundleSizeBytes,
		d.PreflightResult,
		d.SSHIdentity,
		d.ErrorMessage,
		d.ErrorStep,
		d.CreatedAt,
		d.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return identityConflict(d.ScenarioID, d.Environment, d.Target)
	}
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	return nil
}

// GetDeployment retrieves a deployment by ID.
func (r *Repository) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	q := `SELECT ` + deploymentColumns + ` FROM deployments WHERE id = $1`
	d, err := scanDeployment(r.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	return d, nil
}

// GetDeploymentRef returns only the stable identity of a deployment, or nil
// when the record does not exist.
func (r *Repository) GetDeploymentRef(ctx context.Context, id string) (*identity.DeploymentRef, error) {
	const q = `SELECT id, scenario_id, environment, target_binding FROM deployments WHERE id = $1`
	ref, err := scanDeploymentRef(r.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment ref: %w", err)
	}
	return &ref, nil
}

func scanDeploymentRef(row rowScanner) (identity.DeploymentRef, error) {
	var ref identity.DeploymentRef
	var binding domain.NullRawMessage
	if err := row.Scan(&ref.ID, &ref.ScenarioID, &ref.Environment, &binding); err != nil {
		return identity.DeploymentRef{}, err
	}
	ref.Environment = identity.NormalizeEnvironment(ref.Environment)
	if binding.Valid && len(binding.Data) > 0 {
		if err := json.Unmarshal(binding.Data, &ref.Target); err != nil {
			return identity.DeploymentRef{}, fmt.Errorf("decode target binding for deployment %s: %w", ref.ID, err)
		}
	}
	return ref, nil
}

// ResolveDeployments returns every deployment matching the selector. Facet
// matching is a conjunction; the identity package decides how many matches
// are acceptable. The host facet reads the bound locator, and the domain facet
// reads the manifest edge domain; both JSON paths are valid on PostgreSQL and
// SQLite so routed test pools resolve identically to production.
func (r *Repository) ResolveDeployments(ctx context.Context, selector identity.Selector) ([]identity.DeploymentRef, error) {
	selector = selector.Normalized()
	q := `SELECT id, scenario_id, environment, target_binding FROM deployments WHERE 1=1`
	args := []any{}
	add := func(clause, value string) {
		args = append(args, value)
		q += fmt.Sprintf(" AND %s = $%d", clause, len(args))
	}
	if selector.ID != "" {
		add("id", selector.ID)
	}
	if selector.ScenarioID != "" {
		add("scenario_id", selector.ScenarioID)
	}
	if selector.Environment != "" {
		add("environment", identity.NormalizeEnvironment(selector.Environment))
	}
	if selector.Domain != "" {
		add("manifest->'edge'->>'domain'", selector.Domain)
	}
	if selector.Host != "" {
		add("target_binding->'locator'->>'host'", selector.Host)
	}
	q += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve deployments: %w", err)
	}
	defer rows.Close()
	var refs []identity.DeploymentRef
	for rows.Next() {
		ref, err := scanDeploymentRef(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment ref: %w", err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deployment refs: %w", err)
	}
	return refs, nil
}

// UpdateTargetBinding rebinds the target of a deployment. The deployment ID
// is untouched: a machine address change or a Bridge enrollment never creates
// a new deployment. Binding onto a target already owned by another deployment
// of the same scenario and environment is a typed conflict.
func (r *Repository) UpdateTargetBinding(ctx context.Context, id string, target identity.TargetRef) error {
	raw, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("encode target binding: %w", err)
	}
	const q = `
		UPDATE deployments SET
			target_binding = $2,
			target_key = $3,
			updated_at = $4
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, q, id, raw, targetKeyPtr(target), time.Now().UTC())
	if isUniqueViolation(err) {
		ref, refErr := r.GetDeploymentRef(ctx, id)
		if refErr == nil && ref != nil {
			return identityConflict(ref.ScenarioID, ref.Environment, target)
		}
		return identityConflict("", "", target)
	}
	if err != nil {
		return fmt.Errorf("failed to update target binding: %w", err)
	}
	return requireRowsAffected(result, id)
}

// BumpFence increments the deployment fence and returns the new value. The
// caller stamps every write and target effect with it; a target that has seen
// a higher fence refuses the lower one.
func (r *Repository) BumpFence(ctx context.Context, id string) (uint64, error) {
	const q = `
		UPDATE deployments SET fence = fence + 1, updated_at = $2
		WHERE id = $1
		RETURNING fence
	`
	var fence uint64
	err := r.db.QueryRowContext(ctx, q, id, time.Now().UTC()).Scan(&fence)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("deployment not found: %s", id)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to bump fence: %w", err)
	}
	return fence, nil
}

// ListDeployments retrieves deployments with optional filtering.
func (r *Repository) ListDeployments(ctx context.Context, filter domain.ListFilter) ([]*domain.Deployment, error) {
	q := `SELECT ` + deploymentColumns + ` FROM deployments WHERE 1=1`
	args := []interface{}{}

	if filter.Status != nil {
		args = append(args, *filter.Status)
		q += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if filter.ScenarioID != nil {
		args = append(args, *filter.ScenarioID)
		q += fmt.Sprintf(" AND scenario_id = $%d", len(args))
	}
	if filter.Environment != nil {
		args = append(args, identity.NormalizeEnvironment(*filter.Environment))
		q += fmt.Sprintf(" AND environment = $%d", len(args))
	}

	q += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*domain.Deployment
	for rows.Next() {
		d, err := scanDeployment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		deployments = append(deployments, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deployments: %w", err)
	}

	return deployments, nil
}

// UpdateDeployment updates an existing deployment record. The target binding
// is rewritten from the record so a manifest host change keeps the locator
// and the uniqueness key coherent; the ID and fence are never touched here.
func (r *Repository) UpdateDeployment(ctx context.Context, d *domain.Deployment) error {
	d.Environment = identity.NormalizeEnvironment(d.Environment)
	binding, targetKey, err := bindingForWrite(d)
	if err != nil {
		return err
	}
	const q = `
		UPDATE deployments SET
			name = $2,
			scenario_id = $3,
			environment = $4,
			target_binding = $5,
			target_key = $6,
			status = $7,
			manifest = $8,
			bundle_path = $9,
			bundle_sha256 = $10,
			bundle_size_bytes = $11,
			setup_result = $12,
			deploy_result = $13,
			preflight_result = $14,
			last_inspect_result = $15,
			ssh_identity = $16,
			error_message = $17,
			error_step = $18,
			updated_at = $19,
			last_deployed_at = $20,
			last_inspected_at = $21
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, q,
		d.ID,
		d.Name,
		d.ScenarioID,
		d.Environment,
		binding,
		targetKey,
		d.Status,
		d.Manifest,
		d.BundlePath,
		d.BundleSHA256,
		d.BundleSizeBytes,
		d.SetupResult,
		d.DeployResult,
		d.PreflightResult,
		d.LastInspectResult,
		d.SSHIdentity,
		d.ErrorMessage,
		d.ErrorStep,
		d.UpdatedAt,
		d.LastDeployedAt,
		d.LastInspectedAt,
	)
	if isUniqueViolation(err) {
		return identityConflict(d.ScenarioID, d.Environment, d.Target)
	}
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return requireRowsAffected(result, d.ID)
}

func requireRowsAffected(result sql.Result, id string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("deployment not found: %s", id)
	}
	return nil
}

// UpdateDeploymentStatus updates just the status and optional error fields.
func (r *Repository) UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMsg, errorStep *string) error {
	const q = `
		UPDATE deployments SET
			status = $2,
			error_message = $3,
			error_step = $4,
			updated_at = $5
		WHERE id = $1
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, q, id, status, errorMsg, errorStep, now)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}
	return requireRowsAffected(result, id)
}

// UpdateDeploymentSetupResult stores the VPS setup result.
func (r *Repository) UpdateDeploymentSetupResult(ctx context.Context, id string, result json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			setup_result = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, result, now)
	if err != nil {
		return fmt.Errorf("failed to update setup result: %w", err)
	}
	return nil
}

// UpdateDeploymentDeployResult stores the VPS deploy result.
func (r *Repository) UpdateDeploymentDeployResult(ctx context.Context, id string, result json.RawMessage, success bool) error {
	const q = `
		UPDATE deployments SET
			deploy_result = $2,
			status = $3,
			last_deployed_at = $4,
			updated_at = $4
		WHERE id = $1
	`
	now := time.Now()
	status := domain.StatusDeployed
	if !success {
		status = domain.StatusFailed
	}
	_, err := r.db.ExecContext(ctx, q, id, result, status, now)
	if err != nil {
		return fmt.Errorf("failed to update deploy result: %w", err)
	}
	return nil
}

// UpdateDeploymentPreflightResult stores the latest preflight result.
func (r *Repository) UpdateDeploymentPreflightResult(ctx context.Context, id string, result json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			preflight_result = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, result, now)
	if err != nil {
		return fmt.Errorf("failed to update preflight result: %w", err)
	}
	return nil
}

// UpdateDeploymentInspectResult stores the latest VPS inspect result.
func (r *Repository) UpdateDeploymentInspectResult(ctx context.Context, id string, result json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			last_inspect_result = $2,
			last_inspected_at = $3,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, result, now)
	if err != nil {
		return fmt.Errorf("failed to update inspect result: %w", err)
	}
	return nil
}

// UpdateDeploymentBundle stores bundle information after building.
func (r *Repository) UpdateDeploymentBundle(ctx context.Context, id, bundlePath, bundleSHA256 string, sizeBytes int64) error {
	const q = `
		UPDATE deployments SET
			bundle_path = $2,
			bundle_sha256 = $3,
			bundle_size_bytes = $4,
			updated_at = $5
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, bundlePath, bundleSHA256, sizeBytes, now)
	if err != nil {
		return fmt.Errorf("failed to update bundle info: %w", err)
	}
	return nil
}

// UpdateDeploymentManifest updates the manifest for a deployment.
// Used when refreshing the manifest to reflect current scenario state.
func (r *Repository) UpdateDeploymentManifest(ctx context.Context, id string, manifest json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			manifest = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, q, id, manifest, now)
	if err != nil {
		return fmt.Errorf("failed to update manifest: %w", err)
	}
	return requireRowsAffected(result, id)
}

// UpdateDeploymentSSHIdentity persists the canonical SSH identity model.
func (r *Repository) UpdateDeploymentSSHIdentity(ctx context.Context, id string, identityJSON json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			ssh_identity = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, q, id, identityJSON, now)
	if err != nil {
		return fmt.Errorf("failed to update ssh_identity: %w", err)
	}
	return requireRowsAffected(result, id)
}

// DeleteDeployment removes a deployment record.
func (r *Repository) DeleteDeployment(ctx context.Context, id string) error {
	const q = `DELETE FROM deployments WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}
	return requireRowsAffected(result, id)
}

// CountDeploymentsByBundleSHA256 returns the number of deployments using a specific bundle.
// Used to check if a bundle can be safely deleted (no other deployments reference it).
func (r *Repository) CountDeploymentsByBundleSHA256(ctx context.Context, sha256 string) (int, error) {
	const q = `SELECT COUNT(*) FROM deployments WHERE bundle_sha256 = $1`
	var count int
	if err := r.db.QueryRowContext(ctx, q, sha256).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count deployments: %w", err)
	}
	return count, nil
}

// UpdateDeploymentProgress updates the current progress step and percentage.
func (r *Repository) UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error {
	const q = `
		UPDATE deployments SET
			progress_step = $2,
			progress_percent = $3,
			updated_at = $4
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, step, percent, now)
	if err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}
	return nil
}

// ResetDeploymentProgress clears the progress fields when starting a new deployment.
func (r *Repository) ResetDeploymentProgress(ctx context.Context, id string) error {
	const q = `
		UPDATE deployments SET
			progress_step = NULL,
			progress_percent = 0,
			updated_at = $2
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, now)
	if err != nil {
		return fmt.Errorf("failed to reset progress: %w", err)
	}
	return nil
}

// AppendHistoryEvent adds a new event to the deployment history.
func (r *Repository) AppendHistoryEvent(ctx context.Context, id string, event domain.HistoryEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal history event: %w", err)
	}

	const q = `
		UPDATE deployments SET
			deployment_history = COALESCE(deployment_history, '[]'::jsonb) || $2::jsonb,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err = r.db.ExecContext(ctx, q, id, eventJSON, now)
	if err != nil {
		return fmt.Errorf("failed to append history event: %w", err)
	}
	return nil
}

// GetDeploymentHistory retrieves the deployment history for a deployment.
func (r *Repository) GetDeploymentHistory(ctx context.Context, id string) ([]domain.HistoryEvent, error) {
	const q = `
		SELECT COALESCE(deployment_history, '[]'::jsonb)
		FROM deployments
		WHERE id = $1
	`
	var historyJSON []byte
	err := r.db.QueryRowContext(ctx, q, id).Scan(&historyJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	var history []domain.HistoryEvent
	if err := json.Unmarshal(historyJSON, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment history: %w", err)
	}

	return history, nil
}

// BeginDeploymentRun projects "an operation is executing" onto the
// deployment record: status, cleared error and progress. It is a projection
// written from the operation state machine, never an ownership claim — the
// durable operation row (cloud_operations) owns execution, so there is no
// status CAS here and an owner restart can never wedge a deployment.
func (r *Repository) BeginDeploymentRun(ctx context.Context, id string, status domain.DeploymentStatus) error {
	const q = `
		UPDATE deployments SET
			status = $2,
			error_message = NULL,
			error_step = NULL,
			progress_step = NULL,
			progress_percent = 0,
			updated_at = $3
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, q, id, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to begin deployment run: %w", err)
	}
	return requireRowsAffected(result, id)
}

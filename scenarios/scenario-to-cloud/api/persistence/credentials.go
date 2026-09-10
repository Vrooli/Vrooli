package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// Credential ledger DDL. Bindings, acknowledgements and lifecycle operations
// are references and metadata only; no column ever holds a credential value.
// The rotation document is stored whole (JSON) because its receipts and
// consumer progress are read and written as one unit by the lifecycle owner.
// #nosec G101 -- schema identifiers describe metadata columns; no credential value is embedded.
const cloudCredentialsPostgresDDL = `
	CREATE TABLE IF NOT EXISTS cloud_credential_bindings (
		id TEXT PRIMARY KEY,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		logical_id TEXT NOT NULL,
		field TEXT NOT NULL,
		class TEXT NOT NULL,
		source_class TEXT NOT NULL DEFAULT '',
		target_type TEXT NOT NULL DEFAULT 'env',
		target_name TEXT NOT NULL DEFAULT '',
		version_number BIGINT NOT NULL DEFAULT 0,
		version_content_ref TEXT NOT NULL DEFAULT '',
		version_created_at TIMESTAMPTZ,
		previous_version JSONB,
		consumer_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
		grant_ref TEXT NOT NULL DEFAULT '',
		recovery_key_ref TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT 'planned'
			CHECK (state IN ('planned', 'materialized', 'revoked')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE (deployment_id, logical_id, field)
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_credential_bindings_descriptor ON cloud_credential_bindings(logical_id, field);
	CREATE TABLE IF NOT EXISTS cloud_credential_acks (
		binding_id TEXT NOT NULL REFERENCES cloud_credential_bindings(id) ON DELETE CASCADE,
		consumer TEXT NOT NULL,
		version BIGINT NOT NULL,
		verified_at TIMESTAMPTZ NOT NULL,
		PRIMARY KEY (binding_id, consumer)
	);
	CREATE TABLE IF NOT EXISTS cloud_credential_rotations (
		id TEXT PRIMARY KEY,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		binding_id TEXT NOT NULL DEFAULT '',
		kind TEXT NOT NULL,
		from_version BIGINT NOT NULL DEFAULT 0,
		to_version BIGINT NOT NULL DEFAULT 0,
		state TEXT NOT NULL,
		document JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		completed_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_credential_rotations_deployment ON cloud_credential_rotations(deployment_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_cloud_credential_rotations_binding ON cloud_credential_rotations(binding_id);
`

const cloudCredentialsSQLiteDDL = `
CREATE TABLE IF NOT EXISTS cloud_credential_bindings (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 logical_id TEXT NOT NULL,
	 field TEXT NOT NULL,
	 class TEXT NOT NULL,
	 source_class TEXT NOT NULL DEFAULT '',
	 target_type TEXT NOT NULL DEFAULT 'env',
	 target_name TEXT NOT NULL DEFAULT '',
	 version_number INTEGER NOT NULL DEFAULT 0,
	 version_content_ref TEXT NOT NULL DEFAULT '',
	 version_created_at TIMESTAMP,
	 previous_version TEXT,
	 consumer_refs TEXT NOT NULL DEFAULT '[]',
	 grant_ref TEXT NOT NULL DEFAULT '',
	 recovery_key_ref TEXT NOT NULL DEFAULT '',
	 state TEXT NOT NULL DEFAULT 'planned' CHECK (state IN ('planned', 'materialized', 'revoked')),
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 UNIQUE (deployment_id, logical_id, field)
);
CREATE INDEX IF NOT EXISTS idx_cloud_credential_bindings_descriptor ON cloud_credential_bindings(logical_id, field);
CREATE TABLE IF NOT EXISTS cloud_credential_acks (
	 binding_id TEXT NOT NULL REFERENCES cloud_credential_bindings(id) ON DELETE CASCADE,
	 consumer TEXT NOT NULL,
	 version INTEGER NOT NULL,
	 verified_at TIMESTAMP NOT NULL,
	 PRIMARY KEY (binding_id, consumer)
);
CREATE TABLE IF NOT EXISTS cloud_credential_rotations (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 binding_id TEXT NOT NULL DEFAULT '',
	 kind TEXT NOT NULL,
	 from_version INTEGER NOT NULL DEFAULT 0,
	 to_version INTEGER NOT NULL DEFAULT 0,
	 state TEXT NOT NULL,
	 document TEXT NOT NULL,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 completed_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cloud_credential_rotations_deployment ON cloud_credential_rotations(deployment_id, created_at);
CREATE INDEX IF NOT EXISTS idx_cloud_credential_rotations_binding ON cloud_credential_rotations(binding_id);
`

// #nosec G101 -- fixed projection identifiers; no credential value is embedded.
const credentialBindingColumns = `
		id, deployment_id, logical_id, field, class, source_class, target_type, target_name,
		version_number, version_content_ref, version_created_at, previous_version, consumer_refs,
		grant_ref, recovery_key_ref, state, created_at, updated_at`

func scanCredentialBinding(row rowScanner) (*domain.CredentialBinding, error) {
	b := &domain.CredentialBinding{}
	var (
		versionCreated sql.NullTime
		previous       domain.NullRawMessage
		consumers      domain.NullRawMessage
	)
	if err := row.Scan(
		&b.ID, &b.DeploymentID, &b.Descriptor.LogicalID, &b.Descriptor.Field, &b.Class, &b.SourceClass, &b.Target.Type, &b.Target.Name,
		&b.Version.Number, &b.Version.ContentRef, &versionCreated, &previous, &consumers,
		&b.GrantRef, &b.RecoveryKeyRef, &b.State, &b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if versionCreated.Valid {
		b.Version.CreatedAt = versionCreated.Time.UTC()
	}
	if previous.Valid && len(previous.Data) > 0 && string(previous.Data) != "null" {
		var prev domain.CredentialVersion
		if err := json.Unmarshal(previous.Data, &prev); err != nil {
			return nil, fmt.Errorf("decode previous version: %w", err)
		}
		b.PreviousVersion = &prev
	}
	b.ConsumerRefs = []string{}
	if consumers.Valid && len(consumers.Data) > 0 {
		if err := json.Unmarshal(consumers.Data, &b.ConsumerRefs); err != nil {
			return nil, fmt.Errorf("decode consumer refs: %w", err)
		}
	}
	b.CreatedAt = b.CreatedAt.UTC()
	b.UpdatedAt = b.UpdatedAt.UTC()
	return b, nil
}

// UpsertBinding inserts or replaces a binding by id. Only references and
// metadata are stored.
func (r *Repository) UpsertBinding(ctx context.Context, binding *domain.CredentialBinding) error {
	if binding == nil || strings.TrimSpace(binding.ID) == "" || strings.TrimSpace(binding.DeploymentID) == "" || binding.Descriptor.IsZero() {
		return fmt.Errorf("credential binding id, deployment id and descriptor are required")
	}
	now := time.Now().UTC()
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = now
	}
	if binding.UpdatedAt.IsZero() {
		binding.UpdatedAt = now
	}
	if binding.State == "" {
		binding.State = domain.CredentialBindingPlanned
	}
	consumers, err := json.Marshal(nonNilStrings(binding.ConsumerRefs))
	if err != nil {
		return err
	}
	var previous any
	if binding.PreviousVersion != nil {
		raw, err := json.Marshal(binding.PreviousVersion)
		if err != nil {
			return err
		}
		previous = string(raw)
	}
	var versionCreated any
	if !binding.Version.CreatedAt.IsZero() {
		versionCreated = binding.Version.CreatedAt.UTC()
	}
	const q = `
		INSERT INTO cloud_credential_bindings (` + credentialBindingColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			class = EXCLUDED.class, source_class = EXCLUDED.source_class,
			target_type = EXCLUDED.target_type, target_name = EXCLUDED.target_name,
			version_number = EXCLUDED.version_number, version_content_ref = EXCLUDED.version_content_ref,
			version_created_at = EXCLUDED.version_created_at, previous_version = EXCLUDED.previous_version,
			consumer_refs = EXCLUDED.consumer_refs, grant_ref = EXCLUDED.grant_ref,
			recovery_key_ref = EXCLUDED.recovery_key_ref, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at`
	_, err = r.db.ExecContext(ctx, q,
		binding.ID, binding.DeploymentID, binding.Descriptor.LogicalID, binding.Descriptor.Field, string(binding.Class), binding.SourceClass, binding.Target.Type, binding.Target.Name,
		binding.Version.Number, binding.Version.ContentRef, versionCreated, previous, string(consumers),
		binding.GrantRef, binding.RecoveryKeyRef, string(binding.State), binding.CreatedAt.UTC(), binding.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert credential binding: %w", err)
	}
	return nil
}

// GetBinding returns one binding of a deployment, or nil when absent.
func (r *Repository) GetBinding(ctx context.Context, deploymentID, bindingID string) (*domain.CredentialBinding, error) {
	q := `SELECT ` + credentialBindingColumns + ` FROM cloud_credential_bindings WHERE deployment_id = $1 AND id = $2`
	b, err := scanCredentialBinding(r.db.QueryRowContext(ctx, q, deploymentID, bindingID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get credential binding: %w", err)
	}
	return b, nil
}

// ListBindings returns every binding of a deployment ordered by id.
func (r *Repository) ListBindings(ctx context.Context, deploymentID string) ([]domain.CredentialBinding, error) {
	q := `SELECT ` + credentialBindingColumns + ` FROM cloud_credential_bindings WHERE deployment_id = $1 ORDER BY id`
	return r.queryBindings(ctx, q, deploymentID)
}

// FindBindingsByDescriptor returns every binding of an exact descriptor
// across deployments; the caller decides whether several are an ambiguity.
func (r *Repository) FindBindingsByDescriptor(ctx context.Context, descriptor domain.CredentialDescriptor) ([]domain.CredentialBinding, error) {
	q := `SELECT ` + credentialBindingColumns + ` FROM cloud_credential_bindings WHERE logical_id = $1 AND field = $2 ORDER BY id`
	return r.queryBindings(ctx, q, descriptor.LogicalID, descriptor.Field)
}

func (r *Repository) queryBindings(ctx context.Context, q string, args ...any) ([]domain.CredentialBinding, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list credential bindings: %w", err)
	}
	defer rows.Close()
	out := []domain.CredentialBinding{}
	for rows.Next() {
		b, err := scanCredentialBinding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan credential binding: %w", err)
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

// RecordAck upserts a consumer's acknowledgement of a version.
func (r *Repository) RecordAck(ctx context.Context, ack domain.CredentialAck) error {
	if strings.TrimSpace(ack.BindingID) == "" || strings.TrimSpace(ack.Consumer) == "" {
		return fmt.Errorf("credential ack binding id and consumer are required")
	}
	if ack.VerifiedAt.IsZero() {
		ack.VerifiedAt = time.Now().UTC()
	}
	const q = `
		INSERT INTO cloud_credential_acks (binding_id, consumer, version, verified_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (binding_id, consumer) DO UPDATE SET version = EXCLUDED.version, verified_at = EXCLUDED.verified_at`
	if _, err := r.db.ExecContext(ctx, q, ack.BindingID, ack.Consumer, ack.Version, ack.VerifiedAt.UTC()); err != nil {
		return fmt.Errorf("record credential ack: %w", err)
	}
	return nil
}

// ListAcks returns a binding's acknowledgements ordered by consumer.
func (r *Repository) ListAcks(ctx context.Context, bindingID string) ([]domain.CredentialAck, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT binding_id, consumer, version, verified_at FROM cloud_credential_acks WHERE binding_id = $1 ORDER BY consumer`, bindingID)
	if err != nil {
		return nil, fmt.Errorf("list credential acks: %w", err)
	}
	defer rows.Close()
	out := []domain.CredentialAck{}
	for rows.Next() {
		var ack domain.CredentialAck
		if err := rows.Scan(&ack.BindingID, &ack.Consumer, &ack.Version, &ack.VerifiedAt); err != nil {
			return nil, fmt.Errorf("scan credential ack: %w", err)
		}
		ack.VerifiedAt = ack.VerifiedAt.UTC()
		out = append(out, ack)
	}
	return out, rows.Err()
}

// SaveRotation inserts or replaces a lifecycle operation. The document is
// the whole operation; the indexed columns are projections of it.
func (r *Repository) SaveRotation(ctx context.Context, rotation *domain.CredentialRotation) error {
	if rotation == nil || strings.TrimSpace(rotation.ID) == "" || strings.TrimSpace(rotation.DeploymentID) == "" {
		return fmt.Errorf("credential rotation id and deployment id are required")
	}
	now := time.Now().UTC()
	if rotation.CreatedAt.IsZero() {
		rotation.CreatedAt = now
	}
	if rotation.UpdatedAt.IsZero() {
		rotation.UpdatedAt = now
	}
	document, err := json.Marshal(rotation)
	if err != nil {
		return err
	}
	var completed any
	if rotation.CompletedAt != nil {
		completed = rotation.CompletedAt.UTC()
	}
	const q = `
		INSERT INTO cloud_credential_rotations (id, deployment_id, binding_id, kind, from_version, to_version, state, document, created_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			binding_id = EXCLUDED.binding_id, kind = EXCLUDED.kind, from_version = EXCLUDED.from_version,
			to_version = EXCLUDED.to_version, state = EXCLUDED.state, document = EXCLUDED.document,
			updated_at = EXCLUDED.updated_at, completed_at = EXCLUDED.completed_at`
	if _, err := r.db.ExecContext(ctx, q, rotation.ID, rotation.DeploymentID, rotation.BindingID, string(rotation.Kind), rotation.FromVersion, rotation.ToVersion, string(rotation.State), string(document), rotation.CreatedAt.UTC(), rotation.UpdatedAt.UTC(), completed); err != nil {
		return fmt.Errorf("save credential rotation: %w", err)
	}
	return nil
}

// GetRotation returns one operation by id, or nil when absent.
func (r *Repository) GetRotation(ctx context.Context, id string) (*domain.CredentialRotation, error) {
	var document domain.NullRawMessage
	err := r.db.QueryRowContext(ctx, `SELECT document FROM cloud_credential_rotations WHERE id = $1`, id).Scan(&document)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get credential rotation: %w", err)
	}
	return decodeRotation(document.Data)
}

// ListRotations returns a deployment's operations oldest first.
func (r *Repository) ListRotations(ctx context.Context, deploymentID string) ([]domain.CredentialRotation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT document FROM cloud_credential_rotations WHERE deployment_id = $1 ORDER BY created_at, id`, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("list credential rotations: %w", err)
	}
	defer rows.Close()
	out := []domain.CredentialRotation{}
	for rows.Next() {
		var document domain.NullRawMessage
		if err := rows.Scan(&document); err != nil {
			return nil, fmt.Errorf("scan credential rotation: %w", err)
		}
		rotation, err := decodeRotation(document.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, *rotation)
	}
	return out, rows.Err()
}

func decodeRotation(data []byte) (*domain.CredentialRotation, error) {
	var rotation domain.CredentialRotation
	if err := json.Unmarshal(data, &rotation); err != nil {
		return nil, fmt.Errorf("decode credential rotation: %w", err)
	}
	if rotation.Consumers == nil {
		rotation.Consumers = []domain.CredentialConsumerProgress{}
	}
	if rotation.Receipts == nil {
		rotation.Receipts = []domain.CredentialReceipt{}
	}
	return &rotation, nil
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/evidence"
)

// cloudEvidencePostgresDDL holds the append-only evidence ledger and the
// governed publication records. cloud_evidence_records has no UPDATE or
// DELETE path in this repository: a rerun appends, a withdrawal appends.
const cloudEvidencePostgresDDL = `
	CREATE TABLE IF NOT EXISTS cloud_evidence_records (
		id TEXT PRIMARY KEY,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		release_digest TEXT NOT NULL,
		target_key TEXT NOT NULL,
		profile_id TEXT NOT NULL,
		case_id TEXT NOT NULL,
		lane TEXT NOT NULL,
		disposition TEXT NOT NULL,
		record JSONB NOT NULL,
		observed_at TIMESTAMPTZ NOT NULL,
		recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_evidence_records_release ON cloud_evidence_records(release_digest);
	CREATE INDEX IF NOT EXISTS idx_cloud_evidence_records_deployment ON cloud_evidence_records(deployment_id);
	CREATE TABLE IF NOT EXISTS cloud_publications (
		id TEXT PRIMARY KEY,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		request_key TEXT NOT NULL,
		review_ref TEXT,
		identity_digest TEXT NOT NULL,
		release_digest TEXT NOT NULL,
		state TEXT NOT NULL,
		publication JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		published_at TIMESTAMPTZ,
		UNIQUE (deployment_id, request_key)
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_publications_deployment ON cloud_publications(deployment_id);
`

const cloudEvidenceSQLiteDDL = `
CREATE TABLE IF NOT EXISTS cloud_evidence_records (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 release_digest TEXT NOT NULL,
	 target_key TEXT NOT NULL,
	 profile_id TEXT NOT NULL,
	 case_id TEXT NOT NULL,
	 lane TEXT NOT NULL,
	 disposition TEXT NOT NULL,
	 record TEXT NOT NULL,
	 observed_at TIMESTAMP NOT NULL,
	 recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cloud_evidence_records_release ON cloud_evidence_records(release_digest);
CREATE INDEX IF NOT EXISTS idx_cloud_evidence_records_deployment ON cloud_evidence_records(deployment_id);
CREATE TABLE IF NOT EXISTS cloud_publications (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 request_key TEXT NOT NULL,
	 review_ref TEXT,
	 identity_digest TEXT NOT NULL,
	 release_digest TEXT NOT NULL,
	 state TEXT NOT NULL,
	 publication TEXT NOT NULL,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 published_at TIMESTAMP,
	 UNIQUE (deployment_id, request_key)
);
CREATE INDEX IF NOT EXISTS idx_cloud_publications_deployment ON cloud_publications(deployment_id);
`

// ErrEvidenceRecordExists is returned when a record id is appended twice.
var ErrEvidenceRecordExists = errors.New("evidence record already exists")

// AppendEvidenceRecord appends one record. An existing id is a conflict:
// records are immutable and never overwritten.
func (r *Repository) AppendEvidenceRecord(ctx context.Context, rec evidence.Record) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("evidence record requires an id")
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("encode evidence record: %w", err)
	}
	if rec.RecordedAt.IsZero() {
		rec.RecordedAt = time.Now().UTC()
	}
	const q = `
		INSERT INTO cloud_evidence_records (
			id, deployment_id, release_digest, target_key, profile_id, case_id, lane, disposition, record, observed_at, recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING`
	result, err := r.db.ExecContext(ctx, q, rec.ID, rec.Binding.DeploymentID, strings.ToLower(rec.Binding.ReleaseDigest), rec.Binding.TargetKey,
		rec.ProfileID, rec.CaseID, string(rec.Lane), string(rec.Disposition), string(raw), rec.ObservedAt.UTC(), rec.RecordedAt.UTC())
	if err != nil {
		return fmt.Errorf("append evidence record: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrEvidenceRecordExists
	}
	return nil
}

// ListEvidenceRecords returns every record for a release digest, optionally
// narrowed to one deployment, in observation order.
func (r *Repository) ListEvidenceRecords(ctx context.Context, releaseDigest, deploymentID string) ([]evidence.Record, error) {
	q := `SELECT record FROM cloud_evidence_records WHERE release_digest = $1`
	args := []any{strings.ToLower(strings.TrimSpace(releaseDigest))}
	if strings.TrimSpace(deploymentID) != "" {
		q += ` AND deployment_id = $2`
		args = append(args, deploymentID)
	}
	q += ` ORDER BY observed_at ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list evidence records: %w", err)
	}
	defer rows.Close()
	var out []evidence.Record
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var rec evidence.Record
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			return nil, fmt.Errorf("decode evidence record: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// GetEvidenceRecord reads one record by id.
func (r *Repository) GetEvidenceRecord(ctx context.Context, id string) (*evidence.Record, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT record FROM cloud_evidence_records WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get evidence record: %w", err)
	}
	var rec evidence.Record
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, fmt.Errorf("decode evidence record: %w", err)
	}
	return &rec, nil
}

// CreatePublication inserts a publication under its (deployment, request
// key). A replay with the same key returns the stored publication and false.
func (r *Repository) CreatePublication(ctx context.Context, pub evidence.Publication) (*evidence.Publication, bool, error) {
	if strings.TrimSpace(pub.ID) == "" || strings.TrimSpace(pub.DeploymentID) == "" || strings.TrimSpace(pub.RequestKey) == "" {
		return nil, false, fmt.Errorf("publication requires id, deployment and request key")
	}
	now := time.Now().UTC()
	if pub.RequestedAt.IsZero() {
		pub.RequestedAt = now
	}
	pub.UpdatedAt = now
	raw, err := json.Marshal(pub)
	if err != nil {
		return nil, false, fmt.Errorf("encode publication: %w", err)
	}
	const q = `
		INSERT INTO cloud_publications (id, deployment_id, request_key, review_ref, identity_digest, release_digest, state, publication, created_at, updated_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $10)
		ON CONFLICT (deployment_id, request_key) DO NOTHING`
	result, err := r.db.ExecContext(ctx, q, pub.ID, pub.DeploymentID, pub.RequestKey, pub.ReviewRef, pub.IdentityDigest, strings.ToLower(pub.ReleaseDigest), pub.State, string(raw), now, nullTime(pub.PublishedAt))
	if err != nil {
		return nil, false, fmt.Errorf("create publication: %w", err)
	}
	created := false
	if n, _ := result.RowsAffected(); n > 0 {
		created = true
	}
	stored, err := r.GetPublicationByRequestKey(ctx, pub.DeploymentID, pub.RequestKey)
	if err != nil {
		return nil, false, err
	}
	return stored, created, nil
}

// UpdatePublication replaces the mutable projection of a publication.
func (r *Repository) UpdatePublication(ctx context.Context, pub evidence.Publication) error {
	pub.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(pub)
	if err != nil {
		return fmt.Errorf("encode publication: %w", err)
	}
	const q = `UPDATE cloud_publications SET review_ref = $2, state = $3, publication = $4, updated_at = $5, published_at = $6 WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, pub.ID, pub.ReviewRef, pub.State, string(raw), pub.UpdatedAt, nullTime(pub.PublishedAt)); err != nil {
		return fmt.Errorf("update publication: %w", err)
	}
	return nil
}

// GetPublicationByRequestKey resolves an idempotent publication request.
func (r *Repository) GetPublicationByRequestKey(ctx context.Context, deploymentID, requestKey string) (*evidence.Publication, error) {
	return r.scanPublication(r.db.QueryRowContext(ctx, `SELECT publication FROM cloud_publications WHERE deployment_id = $1 AND request_key = $2`, deploymentID, requestKey))
}

// GetPublication reads one publication by id.
func (r *Repository) GetPublication(ctx context.Context, id string) (*evidence.Publication, error) {
	return r.scanPublication(r.db.QueryRowContext(ctx, `SELECT publication FROM cloud_publications WHERE id = $1`, id))
}

// ListPublications returns the deployment's publications, newest first.
func (r *Repository) ListPublications(ctx context.Context, deploymentID string) ([]evidence.Publication, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT publication FROM cloud_publications WHERE deployment_id = $1 ORDER BY created_at DESC, id DESC`, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("list publications: %w", err)
	}
	defer rows.Close()
	var out []evidence.Publication
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var pub evidence.Publication
		if err := json.Unmarshal([]byte(raw), &pub); err != nil {
			return nil, fmt.Errorf("decode publication: %w", err)
		}
		out = append(out, pub)
	}
	return out, rows.Err()
}

func (r *Repository) scanPublication(row *sql.Row) (*evidence.Publication, error) {
	var raw string
	err := row.Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get publication: %w", err)
	}
	var pub evidence.Publication
	if err := json.Unmarshal([]byte(raw), &pub); err != nil {
		return nil, fmt.Errorf("decode publication: %w", err)
	}
	return &pub, nil
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

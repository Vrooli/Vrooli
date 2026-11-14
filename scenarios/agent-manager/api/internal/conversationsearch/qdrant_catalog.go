package conversationsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/sqlcompat"
	aisearch "github.com/vrooli/ai-go/search"
)

// SQLiteQdrantCatalog is the durable catalog adapter for the agent-manager
// owner. The physical collections remain owned by Qdrant; this table records
// the metadata needed to make cleanup restart-safe and auditable.
type SQLiteQdrantCatalog struct {
	db sqlcompat.DB
}

func NewSQLiteQdrantCatalog(db sqlcompat.DB) *SQLiteQdrantCatalog {
	return &SQLiteQdrantCatalog{db: db}
}

type qdrantGenerationRow struct {
	GenerationID        string         `db:"generation_id"`
	Owner               string         `db:"owner"`
	Namespace           string         `db:"namespace"`
	AliasName           string         `db:"alias_name"`
	CollectionName      string         `db:"collection_name"`
	ContentIdentity     string         `db:"content_identity"`
	Model               string         `db:"model"`
	RecipeID            string         `db:"recipe_id"`
	RecipeCanonical     string         `db:"recipe_canonical"`
	ChunkPolicy         string         `db:"chunk_policy"`
	EmbeddingRole       string         `db:"embedding_role"`
	EmbeddingDimensions int            `db:"embedding_dimensions"`
	PolicySchemaVersion string         `db:"policy_schema_version"`
	EmbeddingTaskPrefix bool           `db:"embedding_task_prefix"`
	MigrationPhase      string         `db:"migration_phase"`
	MigrationSources    int            `db:"migration_sources"`
	MigrationPoints     int            `db:"migration_points"`
	State               string         `db:"state"`
	LeaseID             string         `db:"lease_id"`
	LeaseHolder         string         `db:"lease_holder"`
	LeaseExpiresAt      sql.NullString `db:"lease_expires_at"`
	Points              int            `db:"points"`
	Bytes               int64          `db:"bytes"`
	CleanupOutcome      string         `db:"cleanup_outcome"`
	CreatedAt           string         `db:"created_at"`
	UpdatedAt           string         `db:"updated_at"`
}

func (r qdrantGenerationRow) record() (aisearch.GenerationRecord, error) {
	created, err := parseTime(r.CreatedAt)
	if err != nil {
		return aisearch.GenerationRecord{}, err
	}
	updated, err := parseTime(r.UpdatedAt)
	if err != nil {
		return aisearch.GenerationRecord{}, err
	}
	var leaseExpires time.Time
	if r.LeaseExpiresAt.Valid && r.LeaseExpiresAt.String != "" {
		leaseExpires, err = parseTime(r.LeaseExpiresAt.String)
		if err != nil {
			return aisearch.GenerationRecord{}, err
		}
	}
	return aisearch.GenerationRecord{
		Metadata:       aisearch.GenerationMetadata{ID: r.GenerationID, CreatedAt: created, Owner: r.Owner, Namespace: r.Namespace, Alias: r.AliasName, ContentIdentity: r.ContentIdentity, Model: r.Model, RecipeID: r.RecipeID, RecipeCanonical: r.RecipeCanonical, ChunkPolicy: r.ChunkPolicy, EmbeddingRole: r.EmbeddingRole, EmbeddingDimensions: r.EmbeddingDimensions, PolicySchemaVersion: r.PolicySchemaVersion, EmbeddingTaskPrefix: r.EmbeddingTaskPrefix, MigrationPhase: r.MigrationPhase, MigrationSources: r.MigrationSources, MigrationPoints: r.MigrationPoints, LeaseID: r.LeaseID, LeaseExpiresAt: leaseExpires},
		CollectionName: r.CollectionName, State: aisearch.GenerationLifecycleState(r.State), Lease: aisearch.GenerationLease{ID: r.LeaseID, Holder: r.LeaseHolder, ExpiresAt: leaseExpires}, Points: r.Points, Bytes: r.Bytes, CleanupOutcome: r.CleanupOutcome, UpdatedAt: updated,
	}, nil
}

func (c *SQLiteQdrantCatalog) ListGenerationRecords(ctx context.Context, namespace, alias string) ([]aisearch.GenerationRecord, error) {
	var rows []qdrantGenerationRow
	if err := c.db.SelectContext(ctx, &rows, `SELECT * FROM conversation_search_qdrant_generations WHERE namespace = ? AND alias_name = ? AND state <> 'deleted' ORDER BY created_at DESC, generation_id DESC`, namespace, alias); err != nil {
		return nil, fmt.Errorf("list qdrant generation catalog: %w", err)
	}
	result := make([]aisearch.GenerationRecord, 0, len(rows))
	for _, row := range rows {
		record, err := row.record()
		if err != nil {
			return nil, fmt.Errorf("decode qdrant generation %q: %w", row.GenerationID, err)
		}
		result = append(result, record)
	}
	return result, nil
}

func (c *SQLiteQdrantCatalog) SaveGenerationRecord(ctx context.Context, record aisearch.GenerationRecord) error {
	if record.Metadata.ID == "" || record.CollectionName == "" || record.State == "" || record.Metadata.Owner == "" || record.Metadata.Namespace == "" || record.Metadata.Alias == "" || record.Metadata.CreatedAt.IsZero() {
		return errors.New("qdrant generation record is missing identity, owner, state, or timestamps")
	}
	leaseExpires := any(nil)
	if !record.Lease.ExpiresAt.IsZero() {
		leaseExpires = formatTime(record.Lease.ExpiresAt)
	}
	updated := record.UpdatedAt
	if updated.IsZero() {
		updated = time.Now().UTC()
	}
	_, err := c.db.ExecContext(ctx, `INSERT INTO conversation_search_qdrant_generations
        (generation_id, owner, namespace, alias_name, collection_name, content_identity, model, recipe_id, recipe_canonical, chunk_policy,
         embedding_role, embedding_dimensions, policy_schema_version, embedding_task_prefix, migration_phase,
         migration_sources, migration_points, state,
         lease_id, lease_holder, lease_expires_at, points, bytes, cleanup_outcome, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(generation_id) DO UPDATE SET owner=excluded.owner, namespace=excluded.namespace,
        alias_name=excluded.alias_name, collection_name=excluded.collection_name,
        content_identity=CASE WHEN excluded.content_identity='' THEN conversation_search_qdrant_generations.content_identity ELSE excluded.content_identity END,
        model=CASE WHEN excluded.model='' THEN conversation_search_qdrant_generations.model ELSE excluded.model END,
        recipe_id=CASE WHEN excluded.recipe_id='' THEN conversation_search_qdrant_generations.recipe_id ELSE excluded.recipe_id END,
        recipe_canonical=CASE WHEN excluded.recipe_canonical='' THEN conversation_search_qdrant_generations.recipe_canonical ELSE excluded.recipe_canonical END,
        chunk_policy=CASE WHEN excluded.chunk_policy='' THEN conversation_search_qdrant_generations.chunk_policy ELSE excluded.chunk_policy END,
        embedding_role=CASE WHEN excluded.embedding_role='' THEN conversation_search_qdrant_generations.embedding_role ELSE excluded.embedding_role END,
        embedding_dimensions=CASE WHEN excluded.embedding_dimensions=0 THEN conversation_search_qdrant_generations.embedding_dimensions ELSE excluded.embedding_dimensions END,
        policy_schema_version=CASE WHEN excluded.policy_schema_version='' THEN conversation_search_qdrant_generations.policy_schema_version ELSE excluded.policy_schema_version END,
        embedding_task_prefix=CASE WHEN excluded.model='' THEN conversation_search_qdrant_generations.embedding_task_prefix ELSE excluded.embedding_task_prefix END,
        migration_phase=CASE WHEN excluded.migration_phase='' THEN conversation_search_qdrant_generations.migration_phase ELSE excluded.migration_phase END,
        migration_sources=CASE WHEN excluded.migration_phase='' THEN conversation_search_qdrant_generations.migration_sources ELSE excluded.migration_sources END,
        migration_points=excluded.migration_points, state=excluded.state, lease_id=excluded.lease_id,
        lease_holder=excluded.lease_holder, lease_expires_at=excluded.lease_expires_at,
        points=excluded.points, bytes=excluded.bytes, cleanup_outcome=excluded.cleanup_outcome,
        updated_at=excluded.updated_at`,
		record.Metadata.ID, record.Metadata.Owner, record.Metadata.Namespace, record.Metadata.Alias, record.CollectionName,
		record.Metadata.ContentIdentity, record.Metadata.Model, record.Metadata.RecipeID, record.Metadata.RecipeCanonical, record.Metadata.ChunkPolicy,
		record.Metadata.EmbeddingRole, record.Metadata.EmbeddingDimensions, record.Metadata.PolicySchemaVersion,
		record.Metadata.EmbeddingTaskPrefix, record.Metadata.MigrationPhase, record.Metadata.MigrationSources,
		record.Metadata.MigrationPoints, string(record.State), record.Lease.ID, record.Lease.Holder, leaseExpires,
		record.Points, record.Bytes, record.CleanupOutcome, formatTime(record.Metadata.CreatedAt), formatTime(updated))
	if err != nil {
		return fmt.Errorf("save qdrant generation %q: %w", record.Metadata.ID, err)
	}
	return nil
}

func (c *SQLiteQdrantCatalog) LoadGenerationCleanupReceipt(ctx context.Context, key string) (aisearch.GenerationCleanupReceipt, bool, error) {
	var row struct {
		ReceiptJSON string `db:"receipt_json"`
	}
	err := c.db.GetContext(ctx, &row, `SELECT receipt_json FROM conversation_search_qdrant_cleanup_receipts WHERE idempotency_key = ?`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return aisearch.GenerationCleanupReceipt{}, false, nil
	}
	if err != nil {
		return aisearch.GenerationCleanupReceipt{}, false, err
	}
	var receipt aisearch.GenerationCleanupReceipt
	if err := json.Unmarshal([]byte(row.ReceiptJSON), &receipt); err != nil {
		return aisearch.GenerationCleanupReceipt{}, false, fmt.Errorf("decode qdrant cleanup receipt: %w", err)
	}
	return receipt, true, nil
}

func (c *SQLiteQdrantCatalog) SaveGenerationCleanupReceipt(ctx context.Context, receipt aisearch.GenerationCleanupReceipt) error {
	payload, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, `INSERT INTO conversation_search_qdrant_cleanup_receipts (idempotency_key, receipt_json, created_at)
        VALUES (?, ?, ?) ON CONFLICT(idempotency_key) DO NOTHING`, receipt.IdempotencyKey, string(payload), formatTime(receipt.CompletedAt))
	return err
}

func (c *SQLiteQdrantCatalog) AcquireFleetAliasMutation(ctx context.Context, operation aisearch.FleetAliasOperation) (aisearch.FleetAliasMutationFence, error) {
	boundary, err := aisearch.NewSQLiteFleetAliasMutationBoundary(c.db)
	if err != nil {
		return nil, err
	}
	return boundary.AcquireFleetAliasMutation(ctx, operation)
}

var (
	_ aisearch.GenerationCatalog             = (*SQLiteQdrantCatalog)(nil)
	_ aisearch.GenerationCleanupReceiptStore = (*SQLiteQdrantCatalog)(nil)
	_ aisearch.FleetAliasMutationBoundary    = (*SQLiteQdrantCatalog)(nil)
)

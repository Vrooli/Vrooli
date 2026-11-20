package conversationsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	StateReason         string         `db:"state_reason"`
	SupersededBy        string         `db:"superseded_by"`
	StateChangedAt      string         `db:"state_changed_at"`
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
	// An active transition is the one write that can violate the one-active
	// invariant, so it retires every other active row for this alias in the same
	// transaction that installs the invariant and writes the row. A crash
	// between the two is impossible, and the previous rollback target keeps its
	// collection and catalog row as evidence.
	conn, err := c.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open qdrant generation transaction for %q: %w", record.Metadata.ID, err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin qdrant generation transaction for %q: %w", record.Metadata.ID, err)
	}
	defer func() { _ = tx.Rollback() }()
	if record.State == aisearch.GenerationStateActive {
		now := formatTime(updated)
		reason := fmt.Sprintf("lifecycle: superseded by %s", record.Metadata.ID)
		if _, err := tx.ExecContext(ctx, `UPDATE conversation_search_qdrant_generations
            SET state = 'retired', state_reason = ?, superseded_by = ?, state_changed_at = ?, updated_at = ?
            WHERE namespace = ? AND alias_name = ? AND state = 'active' AND generation_id <> ?`,
			reason, record.Metadata.ID, now, now,
			record.Metadata.Namespace, record.Metadata.Alias, record.Metadata.ID); err != nil {
			return fmt.Errorf("retire the previous active qdrant generation for %q: %w", record.Metadata.ID, err)
		}
		if err := ensureOneActiveGenerationInvariant(ctx, tx); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO conversation_search_qdrant_generations
        (generation_id, owner, namespace, alias_name, collection_name, content_identity, model, recipe_id, recipe_canonical, chunk_policy,
         embedding_role, embedding_dimensions, policy_schema_version, embedding_task_prefix, migration_phase,
         migration_sources, migration_points, state, state_reason, superseded_by, state_changed_at,
         lease_id, lease_holder, lease_expires_at, points, bytes, cleanup_outcome, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
        migration_points=excluded.migration_points, state=excluded.state,
        state_reason=CASE WHEN excluded.state = 'active' THEN '' ELSE conversation_search_qdrant_generations.state_reason END,
        superseded_by=CASE WHEN excluded.state = 'active' THEN '' ELSE conversation_search_qdrant_generations.superseded_by END,
        state_changed_at=CASE WHEN conversation_search_qdrant_generations.state <> excluded.state THEN excluded.state_changed_at
                              WHEN excluded.state = 'active' THEN excluded.state_changed_at
                              ELSE conversation_search_qdrant_generations.state_changed_at END,
        lease_id=excluded.lease_id,
        lease_holder=excluded.lease_holder, lease_expires_at=excluded.lease_expires_at,
        points=excluded.points, bytes=excluded.bytes, cleanup_outcome=excluded.cleanup_outcome,
        updated_at=excluded.updated_at`,
		record.Metadata.ID, record.Metadata.Owner, record.Metadata.Namespace, record.Metadata.Alias, record.CollectionName,
		record.Metadata.ContentIdentity, record.Metadata.Model, record.Metadata.RecipeID, record.Metadata.RecipeCanonical, record.Metadata.ChunkPolicy,
		record.Metadata.EmbeddingRole, record.Metadata.EmbeddingDimensions, record.Metadata.PolicySchemaVersion,
		record.Metadata.EmbeddingTaskPrefix, record.Metadata.MigrationPhase, record.Metadata.MigrationSources,
		record.Metadata.MigrationPoints, string(record.State), formatTime(updated), record.Lease.ID, record.Lease.Holder, leaseExpires,
		record.Points, record.Bytes, record.CleanupOutcome, formatTime(record.Metadata.CreatedAt), formatTime(updated))
	if err != nil {
		return fmt.Errorf("save qdrant generation %q: %w", record.Metadata.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit qdrant generation %q: %w", record.Metadata.ID, err)
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
	fence, err := boundary.AcquireFleetAliasMutation(ctx, operation)
	if err == nil || !errors.Is(err, aisearch.ErrFleetStateConflict) {
		return fence, err
	}
	// The shared boundary only replaces a dead holder when the exact same
	// operation is retried. A rollback that died mid-flight on 2026-11-11 left
	// this alias fenced forever, which blocked every later promotion and rebuild
	// cutover. A provably dead holder is safe to displace for a new operation; a
	// live holder is never touched.
	released, releaseErr := c.releaseDeadAliasFence(ctx, operation)
	if releaseErr != nil {
		return nil, errors.Join(err, releaseErr)
	}
	if !released {
		return nil, err
	}
	return boundary.AcquireFleetAliasMutation(ctx, operation)
}

// releaseDeadAliasFence deletes a durable alias fence whose holder process is
// gone (PID absent, or PID reused with a different start time).
func (c *SQLiteQdrantCatalog) releaseDeadAliasFence(ctx context.Context, operation aisearch.FleetAliasOperation) (bool, error) {
	var row struct {
		OwnerID     string `db:"owner_id"`
		OperationID string `db:"operation_id"`
		HolderPID   int    `db:"holder_pid"`
		HolderStart string `db:"holder_start"`
	}
	err := c.db.GetContext(ctx, &row, `SELECT owner_id, operation_id, holder_pid, holder_start
        FROM qdrant_fleet_alias_fences WHERE endpoint_id = ? AND alias_name = ?`, operation.EndpointID, operation.Alias)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read the shared Qdrant alias fence: %w", err)
	}
	if row.OwnerID == operation.OwnerID && row.OperationID == operation.OperationID {
		// Same operation: the shared boundary owns dead-holder replay.
		return false, nil
	}
	if fleetHolderProcessLive(row.HolderPID, row.HolderStart) {
		return false, nil
	}
	result, err := c.db.ExecContext(ctx, `DELETE FROM qdrant_fleet_alias_fences
        WHERE endpoint_id = ? AND alias_name = ? AND owner_id = ? AND operation_id = ? AND holder_pid = ? AND holder_start = ?`,
		operation.EndpointID, operation.Alias, row.OwnerID, row.OperationID, row.HolderPID, row.HolderStart)
	if err != nil {
		return false, fmt.Errorf("release the dead shared Qdrant alias fence: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return changed == 1, nil
}

// fleetHolderProcessLive reports whether the fence holder still runs. A pid that
// no longer exists, or whose kernel start time differs (pid reuse), is dead.
// Anything unreadable is treated as live so a real holder is never displaced.
func fleetHolderProcessLive(pid int, start string) bool {
	if pid <= 0 {
		return false
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false
	}
	end := strings.LastIndex(string(raw), ") ")
	if end < 0 {
		return true
	}
	fields := strings.Fields(string(raw)[end+2:])
	if len(fields) <= 19 {
		return true
	}
	return strings.TrimSpace(fields[19]) == strings.TrimSpace(start)
}

var (
	_ aisearch.GenerationCatalog             = (*SQLiteQdrantCatalog)(nil)
	_ aisearch.GenerationCleanupReceiptStore = (*SQLiteQdrantCatalog)(nil)
	_ aisearch.FleetAliasMutationBoundary    = (*SQLiteQdrantCatalog)(nil)
)

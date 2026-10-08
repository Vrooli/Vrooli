package continuity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"web-console/internal/dbx"
)

// CatalogStore is intentionally narrower than the session store. It owns
// only the Web Console projection and cannot mutate native provider sources.
type CatalogStore interface {
	Upsert(ctx context.Context, record CatalogRecord) error
	AddAliases(ctx context.Context, record CatalogRecord) error
}

// ReconciliationMutationDetector lets the SQL implementation distinguish a
// reviewed repair that is still needed from an identical replay. Test fakes
// intentionally omit it and retain the simple apply semantics used by unit
// tests.
type ReconciliationMutationDetector interface {
	NeedsReconciliation(ctx context.Context, record CatalogRecord) (bool, error)
}

// LifecycleCatalog updates the searchable projection after the session
// lifecycle authority commits a visibility transition. Native agent sources
// remain outside this seam.
type LifecycleCatalog interface {
	UpdateLifecycleState(ctx context.Context, sessionID string, state State) error
}

// PublicationClient is an outbound seam. Web Console remains authoritative
// for lifecycle state; publication may be unavailable without affecting local
// search or recovery.
type PublicationClient interface {
	Publish(context.Context, PublicationRecord) error
	Tombstone(context.Context, PublicationRecord) error
}

type PublicationRecord struct {
	SessionID         string
	LifecycleState    State
	AgentType         string
	CurrentTitle      string
	RolloutRef        string
	SourceFingerprint string
}

type PublicationReport struct {
	Attempted int `json:"attempted"`
	Published int `json:"published"`
	Failed    int `json:"failed"`
	Next      int `json:"next"`
}

type ReconciliationProgress struct {
	OperationID  string
	ManifestHash string
	ActorID      string
	NextOffset   int
	TotalItems   int
	Status       string
	UpdatedAt    time.Time
}

type ReconciliationItemReceipt struct {
	OperationID  string
	ItemIndex    int
	ManifestHash string
	SessionID    string
	Action       ReconcileAction
	Status       string
	ErrorCode    string
	CreatedAt    time.Time
	CompletedAt  time.Time
}

type SQLCatalogStore struct{ db dbx.Handle }

func NewSQLCatalogStore(db dbx.Handle) *SQLCatalogStore { return &SQLCatalogStore{db: db} }

func (s *SQLCatalogStore) SaveReconciliationProgress(ctx context.Context, progress ReconciliationProgress) error {
	if progress.OperationID == "" || progress.ManifestHash == "" || progress.TotalItems < 0 || progress.NextOffset < 0 || progress.NextOffset > progress.TotalItems {
		return fmt.Errorf("invalid reconciliation progress")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO continuity_reconciliation_progress(operation_id,manifest_hash,actor_id,next_offset,total_items,status,updated_at)
		VALUES(?,?,?,?,?,?,?) ON CONFLICT(operation_id) DO UPDATE SET manifest_hash=excluded.manifest_hash, actor_id=excluded.actor_id,
		next_offset=excluded.next_offset, total_items=excluded.total_items, status=excluded.status, updated_at=excluded.updated_at`,
		progress.OperationID, progress.ManifestHash, progress.ActorID, progress.NextOffset, progress.TotalItems, progress.Status, formatCatalogTime(progress.UpdatedAt))
	return err
}

func (s *SQLCatalogStore) LoadReconciliationProgress(ctx context.Context, operationID string) (ReconciliationProgress, error) {
	var progress ReconciliationProgress
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT operation_id,manifest_hash,actor_id,next_offset,total_items,status,updated_at FROM continuity_reconciliation_progress WHERE operation_id=?`, operationID).
		Scan(&progress.OperationID, &progress.ManifestHash, &progress.ActorID, &progress.NextOffset, &progress.TotalItems, &progress.Status, &updated)
	if err != nil {
		return ReconciliationProgress{}, err
	}
	progress.UpdatedAt = parseCatalogTime(updated)
	return progress, nil
}

func (s *SQLCatalogStore) SaveReconciliationItemReceipt(ctx context.Context, receipt ReconciliationItemReceipt) error {
	if receipt.OperationID == "" || receipt.ManifestHash == "" || receipt.ItemIndex < 0 || receipt.SessionID == "" {
		return fmt.Errorf("invalid reconciliation item receipt")
	}
	var completed any
	if !receipt.CompletedAt.IsZero() {
		completed = formatCatalogTime(receipt.CompletedAt)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO continuity_reconciliation_item_receipts(operation_id,item_index,manifest_hash,session_id,action,status,error_code,created_at,completed_at)
		VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(operation_id,item_index) DO UPDATE SET status=excluded.status,error_code=excluded.error_code,completed_at=excluded.completed_at`,
		receipt.OperationID, receipt.ItemIndex, receipt.ManifestHash, receipt.SessionID, string(receipt.Action), receipt.Status, receipt.ErrorCode,
		formatCatalogTime(receipt.CreatedAt), completed)
	return err
}

func (s *SQLCatalogStore) LoadReconciliationItemReceipt(ctx context.Context, operationID string, itemIndex int) (ReconciliationItemReceipt, error) {
	var receipt ReconciliationItemReceipt
	var action, created, completed string
	err := s.db.QueryRowContext(ctx, `SELECT operation_id,item_index,manifest_hash,session_id,action,status,error_code,created_at,COALESCE(completed_at,'') FROM continuity_reconciliation_item_receipts WHERE operation_id=? AND item_index=?`, operationID, itemIndex).
		Scan(&receipt.OperationID, &receipt.ItemIndex, &receipt.ManifestHash, &receipt.SessionID, &action, &receipt.Status, &receipt.ErrorCode, &created, &completed)
	if err != nil {
		return ReconciliationItemReceipt{}, err
	}
	receipt.Action = ReconcileAction(action)
	receipt.CreatedAt = parseCatalogTime(created)
	receipt.CompletedAt = parseCatalogTime(completed)
	return receipt, nil
}

func (s *SQLCatalogStore) SaveManifest(ctx context.Context, manifest ReconciliationManifest) error {
	payload, err := json.Marshal(manifest.Items)
	if err != nil {
		return err
	}
	previous, err := json.Marshal(manifest.Previous)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO continuity_reconciliation_manifests(hash, generation, items_json, previous_json, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(hash) DO NOTHING`, manifest.Hash, manifest.Generation, string(payload), string(previous), formatCatalogTime(time.Now().UTC()))
	return err
}

func (s *SQLCatalogStore) LoadManifest(ctx context.Context, hash string) (ReconciliationManifest, error) {
	var generation, payload, previousPayload string
	if err := s.db.QueryRowContext(ctx, `SELECT generation, items_json, previous_json FROM continuity_reconciliation_manifests WHERE hash = ?`, hash).Scan(&generation, &payload, &previousPayload); err != nil {
		return ReconciliationManifest{}, err
	}
	var items []ReconcileItem
	if err := json.Unmarshal([]byte(payload), &items); err != nil {
		return ReconciliationManifest{}, err
	}
	previous := map[string]CatalogRecord{}
	if err := json.Unmarshal([]byte(previousPayload), &previous); err != nil {
		return ReconciliationManifest{}, err
	}
	return ReconciliationManifest{Hash: hash, Generation: generation, Items: items, Previous: previous}, nil
}

// RollbackManifest restores only the catalog projection preimage captured
// before apply. Durable events, checkpoints, panes, and native histories are
// intentionally outside this operation's mutation boundary.
func (s *SQLCatalogStore) RollbackManifest(ctx context.Context, hash string) error {
	manifest, err := s.LoadManifest(ctx, hash)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, item := range manifest.Items {
		if item.Action == ActionQuarantine {
			continue
		}
		id := item.Record.SessionID
		if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_aliases WHERE session_id = ?`, id); err != nil {
			_ = tx.Rollback()
			return err
		}
		prior, ok := manifest.Previous[id]
		if !ok {
			if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_catalog WHERE session_id = ?`, id); err != nil {
				_ = tx.Rollback()
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_catalog WHERE session_id = ?`, id); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_catalog(session_id,lifecycle_state,lifecycle_version,backend,agent_type,agent_session_id,agent_home_ref,rollout_ref,original_title,current_title,topic_summary,cwd,created_at,last_activity_at,source_fingerprint) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, prior.SessionID, string(prior.LifecycleState), 1, prior.Backend, prior.AgentType, prior.AgentSessionID, prior.AgentHomeRef, prior.RolloutRef, prior.OriginalTitle, prior.CurrentTitle, prior.TopicSummary, prior.CWD, formatCatalogTime(prior.CreatedAt), formatCatalogTime(prior.LastActivityAt), prior.SourceFingerprint); err != nil {
			_ = tx.Rollback()
			return err
		}
		for _, alias := range prior.Aliases {
			if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_aliases(session_id,alias_kind,alias_value,observed_at) VALUES (?,?,?,?)`, prior.SessionID, alias.Kind, alias.Value, formatCatalogTime(time.Now().UTC())); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

// Observe reads only Web Console-owned metadata and produces source evidence
// for reconciliation. The LEFT JOINs are intentional: missing projection rows
// are evidence of recoverability debt, not permission to discard native data.
func (s *SQLCatalogStore) Observe(ctx context.Context) ([]Evidence, error) {
	rows, err := s.db.QueryContext(ctx, `WITH ids(session_id) AS (
		SELECT id FROM sessions
		UNION SELECT session_id FROM conversation_sessions
		UNION SELECT session_id FROM conversation_events
		UNION SELECT web_console_session_id FROM agent_transcript_checkpoints
		UNION SELECT session_id FROM workspace_panes
	) SELECT ids.session_id, COALESCE(s.backend, ''), COALESCE(s.agent_type, ''),
		COALESCE(NULLIF(s.agent_session_id, ''), (SELECT p.source_key FROM agent_transcript_checkpoints p WHERE p.web_console_session_id=ids.session_id ORDER BY p.updated_at DESC LIMIT 1), ''), COALESCE(s.cwd, ''), COALESCE(s.display_label, ''), COALESCE(s.last_rollout_path, ''),
		COALESCE(s.created_at, ''), COALESCE(s.last_activity_at, ''),
		c.session_id IS NOT NULL OR EXISTS (SELECT 1 FROM conversation_events e WHERE e.session_id=ids.session_id),
		EXISTS (SELECT 1 FROM agent_transcript_checkpoints p WHERE p.web_console_session_id=ids.session_id)
		FROM ids LEFT JOIN sessions s ON s.id=ids.session_id
		LEFT JOIN conversation_sessions c ON c.session_id=ids.session_id ORDER BY ids.session_id`)
	if err != nil {
		return nil, fmt.Errorf("observe sessions: %w", err)
	}
	defer rows.Close()
	var out []Evidence
	for rows.Next() {
		var e Evidence
		var created, activity string
		if err := rows.Scan(&e.SessionID, &e.Backend, &e.AgentType,
			&e.AgentSessionID, &e.CWD, &e.CurrentTitle, &e.RolloutRef, &created, &activity,
			&e.HasConversation, &e.HasNativeHistory); err != nil {
			return nil, fmt.Errorf("observe session row: %w", err)
		}
		normalizeRolloutCheckpoint(&e)
		e.CreatedAt = parseCatalogTime(created)
		e.LastActivityAt = parseCatalogTime(activity)
		e.OriginalTitle = e.CurrentTitle
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("observe sessions rows: %w", err)
	}
	return out, nil
}

// normalizeRolloutCheckpoint repairs the common orphan shape where the
// checkpoint source key is the rollout filename rather than the provider's
// thread id. The path remains private rollout evidence; the stable UUID is
// the searchable/public alias used for cross-scenario attribution.
func normalizeRolloutCheckpoint(e *Evidence) {
	if e == nil || e.RolloutRef != "" || !strings.HasSuffix(strings.ToLower(e.AgentSessionID), ".jsonl") {
		return
	}
	path := e.AgentSessionID
	base := strings.TrimSuffix(path, ".jsonl")
	parts := strings.Split(base, "-")
	if len(parts) < 5 {
		return
	}
	candidate := strings.Join(parts[len(parts)-5:], "-")
	if len(candidate) != 36 || candidate[8] != '-' || candidate[13] != '-' || candidate[18] != '-' || candidate[23] != '-' {
		return
	}
	e.RolloutRef = path
	e.AgentSessionID = candidate
}

func (s *SQLCatalogStore) Existing(ctx context.Context) (map[string]CatalogRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT session_id, lifecycle_state,
		backend, agent_type, agent_session_id, agent_home_ref, rollout_ref,
		original_title, current_title, topic_summary, cwd, created_at,
		last_activity_at, source_fingerprint FROM conversation_catalog ORDER BY session_id`)
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}
	defer rows.Close()
	out := map[string]CatalogRecord{}
	for rows.Next() {
		var r CatalogRecord
		var state, created, activity string
		if err := rows.Scan(&r.SessionID, &state, &r.Backend, &r.AgentType,
			&r.AgentSessionID, &r.AgentHomeRef, &r.RolloutRef, &r.OriginalTitle,
			&r.CurrentTitle, &r.TopicSummary, &r.CWD, &created, &activity,
			&r.SourceFingerprint); err != nil {
			return nil, fmt.Errorf("load catalog row: %w", err)
		}
		r.LifecycleState = State(state)
		r.CreatedAt = parseCatalogTime(created)
		r.LastActivityAt = parseCatalogTime(activity)
		out[r.SessionID] = r
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load catalog rows: %w", err)
	}
	aliasRows, err := s.db.QueryContext(ctx, `SELECT session_id, alias_kind, alias_value FROM conversation_aliases ORDER BY alias_kind, alias_value`)
	if err != nil {
		return nil, fmt.Errorf("load catalog aliases: %w", err)
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var sessionID string
		var alias Alias
		if err := aliasRows.Scan(&sessionID, &alias.Kind, &alias.Value); err != nil {
			return nil, fmt.Errorf("load catalog alias: %w", err)
		}
		if record, ok := out[sessionID]; ok {
			record.Aliases = append(record.Aliases, alias)
			out[sessionID] = record
		}
	}
	if err := aliasRows.Err(); err != nil {
		return nil, fmt.Errorf("load catalog aliases rows: %w", err)
	}
	return out, nil
}

func (s *SQLCatalogStore) List(ctx context.Context, lifecycleState string, limit int) ([]CatalogRecord, bool, error) {
	if limit == 0 {
		limit = 100
	}
	unlimited := limit < 0
	query := `SELECT session_id, lifecycle_state, backend, agent_type, agent_session_id,
		agent_home_ref, rollout_ref, original_title, current_title, topic_summary, cwd,
		created_at, last_activity_at, source_fingerprint FROM conversation_catalog`
	args := []any{}
	if lifecycleState != "" {
		query += ` WHERE lifecycle_state = ?`
		args = append(args, lifecycleState)
	}
	query += ` ORDER BY last_activity_at DESC, session_id`
	if !unlimited {
		query += ` LIMIT ?`
		args = append(args, limit+1)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list catalog: %w", err)
	}
	defer rows.Close()
	capacity := 100
	if !unlimited && limit > capacity {
		capacity = limit
	}
	out := make([]CatalogRecord, 0, capacity)
	for rows.Next() {
		var r CatalogRecord
		var state, created, activity string
		if err := rows.Scan(&r.SessionID, &state, &r.Backend, &r.AgentType, &r.AgentSessionID, &r.AgentHomeRef, &r.RolloutRef, &r.OriginalTitle, &r.CurrentTitle, &r.TopicSummary, &r.CWD, &created, &activity, &r.SourceFingerprint); err != nil {
			return nil, false, err
		}
		r.LifecycleState, r.CreatedAt, r.LastActivityAt = State(state), parseCatalogTime(created), parseCatalogTime(activity)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	truncated := !unlimited && len(out) > limit
	if truncated {
		out = out[:limit]
	}
	return out, truncated, nil
}

// ListAll is used by complete archive projections. Unlike the operator-facing
// bounded List method, it does not turn a large catalog into a partial result.
func (s *SQLCatalogStore) ListAll(ctx context.Context, lifecycleState string) ([]CatalogRecord, error) {
	rows, _, err := s.List(ctx, lifecycleState, -1)
	return rows, err
}

func (s *SQLCatalogStore) Upsert(ctx context.Context, r CatalogRecord) error {
	if r.SessionID == "" || r.SourceFingerprint == "" {
		return fmt.Errorf("catalog session_id and source_fingerprint are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO conversation_catalog
		(session_id, lifecycle_state, lifecycle_version, backend, agent_type, agent_session_id,
		agent_home_ref, rollout_ref, original_title, current_title, topic_summary, cwd,
		created_at, last_activity_at, source_fingerprint)
		VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET lifecycle_state=excluded.lifecycle_state,
		backend=excluded.backend, agent_type=excluded.agent_type, agent_session_id=excluded.agent_session_id,
		agent_home_ref=excluded.agent_home_ref, rollout_ref=excluded.rollout_ref,
		original_title=excluded.original_title, current_title=excluded.current_title,
		topic_summary=excluded.topic_summary, cwd=excluded.cwd,
		last_activity_at=excluded.last_activity_at, source_fingerprint=excluded.source_fingerprint`,
		r.SessionID, string(r.LifecycleState), r.Backend, r.AgentType, r.AgentSessionID,
		r.AgentHomeRef, r.RolloutRef, r.OriginalTitle, r.CurrentTitle, r.TopicSummary, r.CWD,
		formatCatalogTime(r.CreatedAt), formatCatalogTime(r.LastActivityAt), r.SourceFingerprint)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO continuity_publication_queue(session_id, lifecycle_state, source_fingerprint, attempts, last_error, published_fingerprint, updated_at) VALUES (?, ?, ?, 0, '', '', ?) ON CONFLICT(session_id) DO UPDATE SET lifecycle_state=excluded.lifecycle_state, source_fingerprint=excluded.source_fingerprint, updated_at=excluded.updated_at WHERE continuity_publication_queue.published_fingerprint <> excluded.source_fingerprint`, r.SessionID, string(r.LifecycleState), r.SourceFingerprint, formatCatalogTime(time.Now().UTC()))
	return err
}

func (s *SQLCatalogStore) NeedsReconciliation(ctx context.Context, r CatalogRecord) (bool, error) {
	var fingerprint string
	err := s.db.QueryRowContext(ctx, `SELECT source_fingerprint FROM conversation_catalog WHERE session_id = ?`, r.SessionID).Scan(&fingerprint)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return fingerprint != r.SourceFingerprint, nil
}

func (s *SQLCatalogStore) AddAliases(ctx context.Context, r CatalogRecord) error {
	for _, alias := range r.Aliases {
		if alias.Kind == "" || alias.Value == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO conversation_aliases(session_id, alias_kind, alias_value, observed_at) VALUES (?, ?, ?, ?) ON CONFLICT(alias_kind, alias_value) DO NOTHING`, r.SessionID, alias.Kind, alias.Value, formatCatalogTime(time.Now().UTC())); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLCatalogStore) UpdateLifecycleState(ctx context.Context, sessionID string, state State) error {
	if strings.TrimSpace(sessionID) == "" || state == "" {
		return fmt.Errorf("session_id and lifecycle state are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var fingerprint string
	if err := tx.QueryRowContext(ctx, `SELECT source_fingerprint FROM conversation_catalog WHERE session_id = ?`, sessionID).Scan(&fingerprint); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE conversation_catalog
		SET lifecycle_state = ?, lifecycle_version = lifecycle_version + 1
		WHERE session_id = ?`, string(state), sessionID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO continuity_publication_queue
		(session_id, lifecycle_state, source_fingerprint, attempts, last_error, published_fingerprint, updated_at)
		VALUES (?, ?, ?, 0, '', '', ?)
		ON CONFLICT(session_id) DO UPDATE SET lifecycle_state=excluded.lifecycle_state,
			source_fingerprint=excluded.source_fingerprint, published_fingerprint='',
			last_error='', updated_at=excluded.updated_at`,
		sessionID, string(state), fingerprint, formatCatalogTime(time.Now().UTC())); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// EnqueueTombstone keeps a deletion intent after the local catalog row and
// native artifacts are removed. The remote derived index can therefore
// converge even after a retry or Web Console restart.
func (s *SQLCatalogStore) EnqueueTombstone(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	fingerprint := "tombstone:" + sessionID
	_, err := s.db.ExecContext(ctx, `INSERT INTO continuity_publication_queue(session_id, lifecycle_state, source_fingerprint, attempts, last_error, published_fingerprint, updated_at) VALUES (?, ?, ?, 0, '', '', ?) ON CONFLICT(session_id) DO UPDATE SET lifecycle_state=excluded.lifecycle_state, source_fingerprint=excluded.source_fingerprint, published_fingerprint='', last_error='', updated_at=excluded.updated_at`, sessionID, string(StateDeleted), fingerprint, formatCatalogTime(time.Now().UTC()))
	return err
}

// PublishPending drains a bounded durable queue. A successful call is marked
// only after Agent Manager acknowledges the source identity; failed calls stay
// queued for a later retry.
func (s *SQLCatalogStore) PublishPending(ctx context.Context, client PublicationClient, limit int) (PublicationReport, error) {
	if client == nil {
		return PublicationReport{}, fmt.Errorf("publication client is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	// Conversation-only records have no transcript source that Agent Manager
	// can import. They remain fully searchable locally, but must not create an
	// infinite remote retry loop. If a transcript is attached later, Upsert's
	// new fingerprint makes the record eligible again.
	if _, err := s.db.ExecContext(ctx, `UPDATE continuity_publication_queue
		SET published_fingerprint=source_fingerprint, last_error='local-only source', updated_at=?
		WHERE lifecycle_state <> ? AND published_fingerprint <> source_fingerprint
		  AND NOT EXISTS (SELECT 1 FROM conversation_catalog c
		    WHERE c.session_id=continuity_publication_queue.session_id
		      AND TRIM(COALESCE(c.rollout_ref, '')) <> '')`, formatCatalogTime(time.Now().UTC()), string(StateDeleted)); err != nil {
		return PublicationReport{}, fmt.Errorf("classify local-only publication records: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT q.session_id, q.lifecycle_state, COALESCE(c.agent_type, ''), COALESCE(c.current_title, ''), COALESCE(c.rollout_ref, ''), q.source_fingerprint FROM continuity_publication_queue q LEFT JOIN conversation_catalog c ON c.session_id=q.session_id WHERE q.published_fingerprint <> q.source_fingerprint ORDER BY q.updated_at, q.session_id LIMIT ?`, limit)
	if err != nil {
		return PublicationReport{}, fmt.Errorf("load publication queue: %w", err)
	}
	defer rows.Close()
	records := make([]PublicationRecord, 0, limit)
	for rows.Next() {
		var record PublicationRecord
		if err := rows.Scan(&record.SessionID, &record.LifecycleState, &record.AgentType, &record.CurrentTitle, &record.RolloutRef, &record.SourceFingerprint); err != nil {
			return PublicationReport{}, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return PublicationReport{}, err
	}
	if err := rows.Close(); err != nil {
		return PublicationReport{}, err
	}
	report := PublicationReport{}
	for _, record := range records {
		report.Attempted++
		var publishErr error
		if record.LifecycleState == StateDeleted {
			publishErr = client.Tombstone(ctx, record)
		} else {
			publishErr = client.Publish(ctx, record)
		}
		if publishErr != nil {
			report.Failed++
			if _, err := s.db.ExecContext(ctx, `UPDATE continuity_publication_queue SET attempts=attempts+1, last_error=?, updated_at=? WHERE session_id=?`, publishErr.Error(), formatCatalogTime(time.Now().UTC()), record.SessionID); err != nil {
				return report, fmt.Errorf("record publication failure for %s: %w", record.SessionID, err)
			}
			continue
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE continuity_publication_queue SET attempts=attempts+1, last_error='', published_fingerprint=?, updated_at=? WHERE session_id=?`, record.SourceFingerprint, formatCatalogTime(time.Now().UTC()), record.SessionID); err != nil {
			return report, err
		}
		report.Published++
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM continuity_publication_queue WHERE published_fingerprint <> source_fingerprint`).Scan(&report.Next); err != nil {
		return report, fmt.Errorf("count pending publications: %w", err)
	}
	return report, nil
}

// PendingPublicationCount reports queued remote work without exposing queue
// internals to transport handlers or metrics adapters.
func (s *SQLCatalogStore) PendingPublicationCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM continuity_publication_queue WHERE published_fingerprint <> source_fingerprint`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending publications: %w", err)
	}
	return count, nil
}

func ApplyReconciliation(ctx context.Context, store CatalogStore, items []ReconcileItem) (int, error) {
	return ApplyReconciliationBatch(ctx, store, items, 0, len(items))
}

func ApplyReconciliationItem(ctx context.Context, store CatalogStore, item ReconcileItem) (bool, error) {
	if item.Action == ActionQuarantine {
		return false, nil
	}
	if detector, ok := store.(ReconciliationMutationDetector); ok {
		needed, err := detector.NeedsReconciliation(ctx, item.Record)
		if err != nil {
			return false, err
		}
		if !needed {
			return false, nil
		}
	}
	if err := store.Upsert(ctx, item.Record); err != nil {
		return false, err
	}
	if err := store.AddAliases(ctx, item.Record); err != nil {
		return false, err
	}
	return true, nil
}

// ApplyReconciliationBatch applies a bounded slice. The caller persists the
// manifest hash and next offset in its operation receipt between batches.
func ApplyReconciliationBatch(ctx context.Context, store CatalogStore, items []ReconcileItem, offset, batchSize int) (int, error) {
	if offset < 0 || offset > len(items) {
		return 0, fmt.Errorf("reconciliation offset %d is outside %d items", offset, len(items))
	}
	if batchSize <= 0 || batchSize > 500 {
		batchSize = 500
	}
	end := offset + batchSize
	if end > len(items) {
		end = len(items)
	}
	mutations := 0
	for _, item := range items[offset:end] {
		changed, err := ApplyReconciliationItem(ctx, store, item)
		if err != nil {
			return mutations, err
		}
		if changed {
			mutations++
		}
	}
	return mutations, nil
}

func formatCatalogTime(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseCatalogTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

package conversationsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"
)

const maxSourcePageSize = 1000

type SQLiteSource struct {
	db         sqlcompat.DB
	normalizer *Normalizer
}

func NewSQLiteSource(db sqlcompat.DB, normalizer *Normalizer) (*SQLiteSource, error) {
	if db == nil {
		return nil, errors.New("conversation source database is required")
	}
	if normalizer == nil {
		return nil, errors.New("conversation source normalizer is required")
	}
	return &SQLiteSource{db: db, normalizer: normalizer}, nil
}

// SnapshotCursor captures the append-only event high-water mark without
// holding a read transaction open. Imports committed later are excluded from
// this traversal and remain in the projection change queue for catch-up.
func (s *SQLiteSource) SnapshotCursor(ctx context.Context) (*SourceCursor, error) {
	var maxRowID int64
	if err := s.db.GetContext(ctx, &maxRowID, `SELECT COALESCE(MAX(rowid), 0) FROM run_events`); err != nil {
		return nil, fmt.Errorf("snapshot canonical conversation source: %w", err)
	}
	if maxRowID == 0 {
		maxRowID = -1
	}
	return &SourceCursor{SnapshotMaxEventRowID: maxRowID}, nil
}

// LoadSourcePage reads canonical messages in a total order and resolves
// logical deletion before normalization. It selects metadata only; transcript
// paths, attachment bodies, and other raw run payloads never cross this seam.
func (s *SQLiteSource) LoadSourcePage(ctx context.Context, cursor *SourceCursor, limit int) (SourcePage, error) {
	if limit <= 0 || limit > maxSourcePageSize {
		return SourcePage{}, fmt.Errorf("source page limit must be between 1 and %d", maxSourcePageSize)
	}
	query := `SELECT
        e.id AS event_id, e.run_id, e.sequence, e.event_type, e.timestamp, e.schema_version, e.data,
        r.label AS run_label, r.status AS run_status,
        COALESCE(NULLIF(r.import_source_harness, ''), r.harness_kind, '') AS harness,
        COALESCE(r.import_source_harness, '') AS import_source_harness,
        COALESCE(NULLIF(r.import_source_session_id, ''), NULLIF(r.harness_session_id, ''), r.session_id, '') AS source_session_id,
        COALESCE(r.harness_kind, '') AS runner,
        COALESCE(NULLIF(r.actual_model, ''), r.requested_model, '') AS model,
        COALESCE(r.agent_profile_id, '') AS profile,
        COALESCE(r.tag, '') AS tag,
        COALESCE(r.workload_kind, '') AS workload_kind,
        COALESCE(r.workload_key, '') AS workload_key,
        COALESCE(t.project_root, '') AS project_scope,
        COALESCE(t.scope_path, '') AS cwd_scope
    FROM run_events e
    JOIN runs r ON r.id = e.run_id
    LEFT JOIN tasks t ON t.id = r.task_id
    WHERE e.event_type IN ('message', 'tool_call', 'tool_result')
      AND NOT EXISTS (
        SELECT 1 FROM run_events deletion
        WHERE deletion.run_id = e.run_id
          AND deletion.event_type = 'message_deleted'
          AND (? = 0 OR deletion.rowid <= ?)
          AND COALESCE(
            json_extract(deletion.data, '$.targetEventId'),
            json_extract(deletion.data, '$.target_event_id'),
            ''
          ) = e.id
      )`
	snapshotMaxRowID := int64(0)
	if cursor != nil {
		snapshotMaxRowID = cursor.SnapshotMaxEventRowID
	}
	args := []any{snapshotMaxRowID, snapshotMaxRowID}
	if cursor != nil && cursor.SnapshotMaxEventRowID != 0 {
		query += ` AND e.rowid <= ?`
		args = append(args, cursor.SnapshotMaxEventRowID)
	}
	pagingSet := cursor != nil && (!cursor.OccurredAt.IsZero() || cursor.SourceRunID != "" || cursor.EventSequence != 0)
	if cursor != nil && cursor.SnapshotMaxEventRowID == 0 && !pagingSet {
		return SourcePage{}, errors.New("source cursor requires snapshot or paging position")
	}
	if pagingSet {
		if cursor.OccurredAt.IsZero() || cursor.SourceRunID == "" || cursor.EventSequence < 0 {
			return SourcePage{}, errors.New("source cursor requires occurred time, run id, and non-negative sequence")
		}
		timestamp := formatTime(cursor.OccurredAt)
		query += ` AND (e.timestamp > ? OR
            (e.timestamp = ? AND e.run_id > ?) OR
            (e.timestamp = ? AND e.run_id = ? AND e.sequence > ?))`
		args = append(args, timestamp, timestamp, cursor.SourceRunID, timestamp, cursor.SourceRunID, cursor.EventSequence)
	}
	query += ` ORDER BY e.timestamp ASC, e.run_id ASC, e.sequence ASC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return SourcePage{}, fmt.Errorf("load canonical conversation page: %w", err)
	}
	defer rows.Close()

	raw := make([]sourceRow, 0, limit+1)
	for rows.Next() {
		var row sourceRow
		if err := rows.StructScan(&row); err != nil {
			return SourcePage{}, fmt.Errorf("scan canonical conversation source: %w", err)
		}
		raw = append(raw, row)
	}
	if err := rows.Err(); err != nil {
		return SourcePage{}, fmt.Errorf("iterate canonical conversation page: %w", err)
	}

	hasMore := len(raw) > limit
	if hasMore {
		raw = raw[:limit]
	}
	page := SourcePage{}
	for _, row := range raw {
		source, err := row.sourceMessage()
		if err != nil {
			return SourcePage{}, err
		}
		documents, err := s.normalizer.Normalize(source)
		if err != nil {
			return SourcePage{}, fmt.Errorf("normalize source event %q: %w", row.EventID, err)
		}
		page.Documents = append(page.Documents, documents...)
	}
	if hasMore && len(raw) > 0 {
		last := raw[len(raw)-1]
		occurredAt, err := parseSourceTime(last.Timestamp)
		if err != nil {
			return SourcePage{}, err
		}
		page.NextCursor = &SourceCursor{OccurredAt: occurredAt, SourceRunID: last.RunID, EventSequence: last.Sequence}
		if cursor != nil {
			page.NextCursor.SnapshotMaxEventRowID = cursor.SnapshotMaxEventRowID
		}
	}
	return page, nil
}

// LoadRunDocuments reads only the canonical events for one run. Incremental
// projection uses this path after the durable change queue has already named
// the affected run, avoiding a full-corpus scan for every queued upsert.
func (s *SQLiteSource) LoadRunDocuments(ctx context.Context, runID string) ([]Document, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, errors.New("source run id is required")
	}
	rows, err := s.db.QueryxContext(ctx, `SELECT
        e.id AS event_id, e.run_id, e.sequence, e.event_type, e.timestamp, e.schema_version, e.data,
        r.label AS run_label, r.status AS run_status,
        COALESCE(NULLIF(r.import_source_harness, ''), r.harness_kind, '') AS harness,
        COALESCE(r.import_source_harness, '') AS import_source_harness,
        COALESCE(NULLIF(r.import_source_session_id, ''), NULLIF(r.harness_session_id, ''), r.session_id, '') AS source_session_id,
        COALESCE(r.harness_kind, '') AS runner,
        COALESCE(NULLIF(r.actual_model, ''), r.requested_model, '') AS model,
        COALESCE(r.agent_profile_id, '') AS profile,
        COALESCE(r.tag, '') AS tag,
        COALESCE(r.workload_kind, '') AS workload_kind,
        COALESCE(r.workload_key, '') AS workload_key,
        COALESCE(t.project_root, '') AS project_scope,
        COALESCE(t.scope_path, '') AS cwd_scope
    FROM run_events e
    JOIN runs r ON r.id = e.run_id
    LEFT JOIN tasks t ON t.id = r.task_id
    WHERE e.run_id = ?
      AND e.event_type IN ('message', 'tool_call', 'tool_result')
      AND NOT EXISTS (
        SELECT 1 FROM run_events deletion
        WHERE deletion.run_id = e.run_id
          AND deletion.event_type = 'message_deleted'
          AND COALESCE(
            json_extract(deletion.data, '$.targetEventId'),
            json_extract(deletion.data, '$.target_event_id'),
            ''
          ) = e.id
      )
    ORDER BY e.timestamp ASC, e.sequence ASC, e.id ASC`, runID)
	if err != nil {
		return nil, fmt.Errorf("load canonical conversation run %q: %w", runID, err)
	}
	defer rows.Close()

	documents := make([]Document, 0)
	for rows.Next() {
		var row sourceRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan canonical conversation run %q: %w", runID, err)
		}
		source, err := row.sourceMessage()
		if err != nil {
			return nil, err
		}
		normalized, err := s.normalizer.Normalize(source)
		if err != nil {
			return nil, fmt.Errorf("normalize source event %q: %w", row.EventID, err)
		}
		documents = append(documents, normalized...)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate canonical conversation run %q: %w", runID, err)
	}
	return documents, nil
}

type sourceRow struct {
	EventID         string         `db:"event_id"`
	RunID           string         `db:"run_id"`
	Sequence        int64          `db:"sequence"`
	EventType       string         `db:"event_type"`
	Timestamp       string         `db:"timestamp"`
	SchemaVersion   int            `db:"schema_version"`
	Data            []byte         `db:"data"`
	RunLabel        string         `db:"run_label"`
	RunStatus       string         `db:"run_status"`
	Harness         string         `db:"harness"`
	ImportHarness   string         `db:"import_source_harness"`
	SourceSessionID string         `db:"source_session_id"`
	Runner          string         `db:"runner"`
	Model           string         `db:"model"`
	Profile         string         `db:"profile"`
	Tag             string         `db:"tag"`
	WorkloadKind    string         `db:"workload_kind"`
	WorkloadKey     string         `db:"workload_key"`
	ProjectScope    sql.NullString `db:"project_scope"`
	CWDScope        sql.NullString `db:"cwd_scope"`
}

func (row sourceRow) sourceMessage() (SourceMessage, error) {
	eventType := domain.RunEventType(row.EventType)
	payload, err := domain.DecodeEventPayload(eventType, row.Data)
	if err != nil {
		return SourceMessage{}, fmt.Errorf("decode conversation event %q schema %d: %w", row.EventID, row.SchemaVersion, err)
	}
	var role, content, messageID, providerOrigin, providerEventType, evidenceFor string
	var evidenceOnly bool
	switch value := payload.(type) {
	case *domain.MessageEventData:
		role, content, messageID = value.Role, value.Content, value.MessageID
		providerOrigin, providerEventType = value.ProviderOrigin, value.ProviderEventType
		evidenceOnly, evidenceFor = value.EvidenceOnly, value.EvidenceForEventID
	case *domain.ToolCallEventData:
		encoded, marshalErr := json.Marshal(value.Input)
		if marshalErr != nil {
			return SourceMessage{}, fmt.Errorf("encode tool call event %q: %w", row.EventID, marshalErr)
		}
		role, messageID, providerEventType = "tool_call", value.ToolCallID, string(domain.EventTypeToolCall)
		content = strings.TrimSpace(value.ToolName + " " + string(encoded))
	case *domain.ToolResultEventData:
		role, messageID, providerEventType = "tool_result", value.ToolCallID, string(domain.EventTypeToolResult)
		content = value.Output
		if content == "" {
			content = value.Error
		}
	default:
		return SourceMessage{}, fmt.Errorf("decode conversation event %q: unexpected payload %T", row.EventID, payload)
	}
	occurredAt, err := parseSourceTime(row.Timestamp)
	if err != nil {
		return SourceMessage{}, fmt.Errorf("event %q: %w", row.EventID, err)
	}
	importer := ""
	if row.ImportHarness != "" {
		importer = "agent-manager.transcript-import"
	}
	return SourceMessage{
		RunID: row.RunID, EventID: row.EventID, MessageID: messageID, Sequence: row.Sequence,
		Role: role, OccurredAt: occurredAt, Content: content,
		ProviderEventType: providerEventType, EvidenceOnly: evidenceOnly,
		EvidenceForEvent: evidenceFor, Harness: row.Harness,
		SourceSessionID: row.SourceSessionID, ProviderOrigin: providerOrigin,
		Importer: importer, ProjectScope: row.ProjectScope.String, CWDScope: row.CWDScope.String,
		Runner: row.Runner, Model: row.Model, Profile: row.Profile, RunStatus: row.RunStatus,
		RunLabel: row.RunLabel, Tags: compactStrings([]string{row.Tag}),
		Workloads: compactStrings([]string{row.WorkloadKind, row.WorkloadKey}),
	}, nil
}

func parseSourceTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid canonical timestamp %q", value)
}

var (
	_ SourceRepository  = (*SQLiteSource)(nil)
	_ RunDocumentSource = (*SQLiteSource)(nil)
)

package cases

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
	chainforensics "github.com/vrooli/chain-forensics"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository interface {
	Import(context.Context, chainforensics.Case) (Record, error)
	MergeCapture(context.Context, string, []chainforensics.Source, []chainforensics.Occurrence, ...[]chainforensics.EVMWalletSnapshot) (Record, error)
	Get(context.Context, string) (Record, error)
	List(context.Context, int) ([]Record, error)
	AddLabel(context.Context, AnalystLabel) (AnalystLabel, error)
	ListLabels(context.Context, string) ([]AnalystLabel, error)
	SavePacket(context.Context, chainforensics.PrivatePacket) (PacketRecord, error)
	Revise(context.Context, string, time.Time, RevisionAdditions) (Record, CaseRevision, error)
	ListRevisions(context.Context, string) ([]CaseRevision, error)
}

func (r *sqliteRepository) SavePacket(ctx context.Context, packet chainforensics.PrivatePacket) (PacketRecord, error) {
	if packet.Case.ID == "" || packet.SchemaVersion == "" || len(packet.Bytes) == 0 || packet.SHA256 == "" {
		return PacketRecord{}, ErrInvalid{Field: "packet", Reason: "case, schema version, digest, and JSON bytes are required"}
	}
	if err := chainforensics.Validate(packet.Case); err != nil || !packet.Case.Private {
		return PacketRecord{}, ErrInvalid{Field: "packet", Reason: "packet must contain a valid private case"}
	}
	canonical, err := chainforensics.BuildPrivatePacket(packet.Case)
	if err != nil || canonical.SchemaVersion != packet.SchemaVersion || canonical.SHA256 != packet.SHA256 || string(canonical.Bytes) != string(packet.Bytes) {
		return PacketRecord{}, ErrInvalid{Field: "packet", Reason: "packet bytes must match the canonical typed package representation"}
	}
	stamp := r.clock.Now().UTC()
	_, err = r.db.ExecContext(ctx, `INSERT INTO forensic_packets(case_id, schema_version, sha256, packet_json, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(case_id, sha256) DO NOTHING`, packet.Case.ID, packet.SchemaVersion, packet.SHA256, string(packet.Bytes), stamp.Format(timestampLayout))
	if err != nil {
		return PacketRecord{}, fmt.Errorf("persist private packet: %w", err)
	}
	record, err := r.getPacket(ctx, packet.Case.ID, packet.SHA256)
	if err != nil {
		return PacketRecord{}, err
	}
	if record.SchemaVersion != packet.SchemaVersion || string(record.JSON) != string(packet.Bytes) {
		return PacketRecord{}, ErrConflict{ID: packet.Case.ID}
	}
	return record, nil
}

func (r *sqliteRepository) getPacket(ctx context.Context, caseID, digest string) (PacketRecord, error) {
	var record PacketRecord
	var created string
	err := r.db.QueryRowContext(ctx, `SELECT case_id, schema_version, sha256, packet_json, created_at FROM forensic_packets WHERE case_id = ? AND sha256 = ?`, caseID, digest).Scan(&record.CaseID, &record.SchemaVersion, &record.SHA256, &record.JSON, &created)
	if err != nil {
		return PacketRecord{}, fmt.Errorf("read persisted private packet: %w", err)
	}
	parsed, err := time.Parse(timestampLayout, created)
	if err != nil {
		return PacketRecord{}, fmt.Errorf("parse packet created_at: %w", err)
	}
	record.CreatedAt = parsed
	return record, nil
}

func (r *sqliteRepository) MergeCapture(ctx context.Context, id string, sources []chainforensics.Source, occurrences []chainforensics.Occurrence, snapshots ...[]chainforensics.EVMWalletSnapshot) (Record, error) {
	record, err := r.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	changed := false
	for _, source := range sources {
		found := false
		for sourceIndex, old := range record.Case.Sources {
			if old.ID == source.ID {
				found = true
				oldSnapshot, newSnapshot := old, source
				// Retrieved-at differs on a repeat read even when the public
				// provider returned byte-identical result data. Keep the first
				// observation timestamp; the content hash is the replay identity.
				oldSnapshot.Retrieved, newSnapshot.Retrieved = "", ""
				// JSON-RPC envelopes include the caller's request id. A repeat
				// query can therefore have a different exact envelope while the
				// provider result bytes remain identical. Preserve the first raw
				// envelope as evidence and treat the result hash as the replay key.
				enrichRawResponse := oldSnapshot.SHA256 == newSnapshot.SHA256 && len(oldSnapshot.RawResponseBody) == 0 && len(newSnapshot.RawResponseBody) > 0
				if enrichRawResponse && strings.Contains(strings.ToLower(oldSnapshot.Limitations), "not retained") && !strings.Contains(strings.ToLower(newSnapshot.Limitations), "not retained") {
					// Replace a stale capture limitation only when the stronger
					// source now retains the exact envelope that was previously absent.
					oldSnapshot.Limitations = newSnapshot.Limitations
				}
				if oldSnapshot.SHA256 == newSnapshot.SHA256 {
					oldSnapshot.RawResponseBody, newSnapshot.RawResponseBody = nil, nil
					oldSnapshot.RawResponseSHA256, newSnapshot.RawResponseSHA256 = "", ""
				}
				if !reflect.DeepEqual(oldSnapshot, newSnapshot) {
					return Record{}, ErrConflict{ID: id}
				}
				if enrichRawResponse {
					source.Retrieved = old.Retrieved
					record.Case.Sources[sourceIndex] = source
					changed = true
				}
				break
			}
		}
		if !found {
			record.Case.Sources = append(record.Case.Sources, source)
			changed = true
		}
	}
	for _, occurrence := range occurrences {
		found := false
		for _, old := range record.Case.Occurrences {
			if old.ID == occurrence.ID {
				found = true
				oldJSON, _ := json.Marshal(old)
				newJSON, _ := json.Marshal(occurrence)
				if string(oldJSON) != string(newJSON) {
					return Record{}, ErrConflict{ID: id}
				}
				break
			}
		}
		if !found {
			record.Case.Occurrences = append(record.Case.Occurrences, occurrence)
			changed = true
		}
	}
	for _, batch := range snapshots {
		for _, snapshot := range batch {
			found := false
			for _, old := range record.Case.WalletSnapshots {
				if old.ID == snapshot.ID {
					found = true
					oldJSON, _ := json.Marshal(old)
					newJSON, _ := json.Marshal(snapshot)
					if string(oldJSON) != string(newJSON) {
						return Record{}, ErrConflict{ID: id}
					}
					break
				}
			}
			if !found {
				record.Case.WalletSnapshots = append(record.Case.WalletSnapshots, snapshot)
				changed = true
			}
		}
	}
	// Exact Bitcoin UTXO references make a transport-neutral observed edge
	// when both the prior output and the spending input are present. Do not
	// infer links from a shared address or similar amount/time.
	for _, edge := range chainforensics.BuildBitcoinOutpointEdges(record.Case.Occurrences) {
		found := false
		for _, old := range record.Case.Edges {
			if old.ID == edge.ID {
				found = true
				break
			}
		}
		if !found {
			record.Case.Edges = append(record.Case.Edges, edge)
			changed = true
		}
	}
	previousEdgeCount := len(record.Case.Edges)
	record.Case = chainforensics.MergeBridgeCorrelationEdges(record.Case)
	if len(record.Case.Edges) != previousEdgeCount {
		changed = true
	}
	if !changed {
		return record, nil
	}
	if err := chainforensics.Validate(record.Case); err != nil {
		return Record{}, ErrInvalid{Field: "capture", Reason: err.Error()}
	}
	encoded, err := json.Marshal(record.Case)
	if err != nil {
		return Record{}, err
	}
	expected := record.UpdatedAt.UTC().Format(timestampLayout)
	now := r.clock.Now().UTC()
	if !now.After(record.UpdatedAt) {
		now = record.UpdatedAt.Add(time.Nanosecond)
	}
	stamp := now.Format(timestampLayout)
	result, err := r.db.ExecContext(ctx, `UPDATE forensic_cases SET document_json = ?, updated_at = ? WHERE id = ? AND updated_at = ?`, string(encoded), stamp, id, expected)
	if err != nil {
		return Record{}, fmt.Errorf("merge case capture: %w", err)
	}
	changedRows, err := result.RowsAffected()
	if err != nil {
		return Record{}, fmt.Errorf("read capture update result: %w", err)
	}
	if changedRows != 1 {
		return Record{}, ErrConflict{ID: id}
	}
	record.UpdatedAt = now
	return record, nil
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

const timestampLayout = time.RFC3339Nano

func (r *sqliteRepository) Import(ctx context.Context, document chainforensics.Case) (Record, error) {
	if err := chainforensics.Validate(document); err != nil {
		return Record{}, ErrInvalid{Field: "case", Reason: err.Error()}
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return Record{}, fmt.Errorf("encode case: %w", err)
	}
	now := r.clock.Now().UTC()
	stamp := now.Format(timestampLayout)
	result, err := r.db.ExecContext(ctx, `INSERT INTO forensic_cases(id, document_json, created_at, updated_at) VALUES (?, ?, ?, ?)`, document.ID, string(encoded), stamp, stamp)
	if err == nil {
		if _, err = result.RowsAffected(); err != nil {
			return Record{}, fmt.Errorf("read case insert result: %w", err)
		}
		return Record{Case: document, CreatedAt: now, UpdatedAt: now}, nil
	}
	if existing, getErr := r.Get(ctx, document.ID); getErr == nil {
		previous, marshalErr := json.Marshal(existing.Case)
		if marshalErr == nil && string(previous) == string(encoded) {
			return existing, nil
		}
		return Record{}, ErrConflict{ID: document.ID}
	} else if _, ok := getErr.(ErrNotFound); !ok {
		return Record{}, fmt.Errorf("check duplicate case %q: %w", document.ID, getErr)
	}
	return Record{}, fmt.Errorf("insert case %q: %w", document.ID, err)
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Record, error) {
	var raw, created, updated string
	if err := r.db.QueryRowContext(ctx, `SELECT document_json, created_at, updated_at FROM forensic_cases WHERE id = ?`, id).Scan(&raw, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, ErrNotFound{ID: id}
		}
		return Record{}, fmt.Errorf("read case %q: %w", id, err)
	}
	document, err := chainforensics.DecodeCase(strings.NewReader(raw))
	if err != nil {
		return Record{}, fmt.Errorf("decode persisted case %q: %w", id, err)
	}
	createdAt, err := time.Parse(timestampLayout, created)
	if err != nil {
		return Record{}, fmt.Errorf("parse case created_at: %w", err)
	}
	updatedAt, err := time.Parse(timestampLayout, updated)
	if err != nil {
		return Record{}, fmt.Errorf("parse case updated_at: %w", err)
	}
	return Record{Case: document, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func (r *sqliteRepository) List(ctx context.Context, limit int) ([]Record, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM forensic_cases ORDER BY updated_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan case id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case list: %w", err)
	}
	out := make([]Record, 0, len(ids))
	for _, id := range ids {
		record, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func (r *sqliteRepository) AddLabel(ctx context.Context, label AnalystLabel) (AnalystLabel, error) {
	if strings.TrimSpace(label.CaseID) == "" || strings.TrimSpace(label.Subject) == "" || strings.TrimSpace(label.Label) == "" {
		return AnalystLabel{}, ErrInvalid{Field: "label", Reason: "case id, subject, and label are required"}
	}
	switch label.Grade {
	case chainforensics.GradeObserved, chainforensics.GradeProtocolCorrelated, chainforensics.GradeSourceCorroborated, chainforensics.GradeHypothesis, chainforensics.GradeUnresolved:
	default:
		return AnalystLabel{}, ErrInvalid{Field: "grade", Reason: "must be a supported evidence grade"}
	}
	parsed, err := url.ParseRequestURI(label.SourceURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return AnalystLabel{}, ErrInvalid{Field: "source_url", Reason: "must be an absolute HTTP(S) source URL"}
	}
	if strings.TrimSpace(label.SourceCitation) == "" {
		return AnalystLabel{}, ErrInvalid{Field: "source_citation", Reason: "a locator or source citation is required"}
	}
	if strings.TrimSpace(label.CounterExplanation) == "" || strings.TrimSpace(label.UnresolvedGap) == "" {
		return AnalystLabel{}, ErrInvalid{Field: "counter_explanation", Reason: "a counter-explanation and unresolved gap are required for attribution labels"}
	}
	if label.ID == "" {
		label.ID = uuid.NewString()
	}
	if label.CreatedAt.IsZero() {
		label.CreatedAt = r.clock.Now().UTC()
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO analyst_labels(id, case_id, subject, label, source_url, source_citation, grade, counter_explanation, unresolved_gap, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		label.ID, label.CaseID, label.Subject, label.Label, label.SourceURL, label.SourceCitation, string(label.Grade), label.CounterExplanation, label.UnresolvedGap, label.CreatedAt.UTC().Format(timestampLayout))
	if err != nil {
		return AnalystLabel{}, fmt.Errorf("insert analyst label: %w", err)
	}
	return label, nil
}

func (r *sqliteRepository) ListLabels(ctx context.Context, caseID string) ([]AnalystLabel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, case_id, subject, label, source_url, source_citation, grade, counter_explanation, unresolved_gap, created_at FROM analyst_labels WHERE case_id = ? ORDER BY created_at, id`, caseID)
	if err != nil {
		return nil, fmt.Errorf("list analyst labels: %w", err)
	}
	defer rows.Close()
	var labels []AnalystLabel
	for rows.Next() {
		var label AnalystLabel
		var grade, created string
		if err := rows.Scan(&label.ID, &label.CaseID, &label.Subject, &label.Label, &label.SourceURL, &label.SourceCitation, &grade, &label.CounterExplanation, &label.UnresolvedGap, &created); err != nil {
			return nil, fmt.Errorf("scan analyst label: %w", err)
		}
		label.Grade = chainforensics.Grade(grade)
		label.CreatedAt, err = time.Parse(timestampLayout, created)
		if err != nil {
			return nil, fmt.Errorf("parse analyst label created_at: %w", err)
		}
		labels = append(labels, label)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate analyst labels: %w", err)
	}
	return labels, nil
}

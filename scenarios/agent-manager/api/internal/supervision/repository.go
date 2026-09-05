package supervision

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/eventlog"
	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const cursorVersion = 1

var (
	ErrNotFound = errors.New("cohort watch not found")
	ErrConflict = errors.New("cohort watch revision or cursor conflict")
)

type CursorCheckpoint struct {
	Token               string
	Version             int
	RowID               int64
	RetentionGeneration int64
	FilterDigest        string
}

type Repository struct {
	db  sqlcompat.DB
	now func() time.Time
}

type Measures struct {
	ActiveWatches int64
	DueWatches    int64
	Decisions     int64
	Actions       int64
	DatabaseBytes int64
}

func NewRepository(db sqlcompat.DB) *Repository { return &Repository{db: db, now: time.Now} }

func (r *Repository) Create(ctx context.Context, spec *domainpb.WatchSpec, idempotencyKey string, retentionGeneration int64) (*domainpb.CohortWatch, CursorCheckpoint, bool, error) {
	canonical, digest, err := canonicalSpec(spec)
	if err != nil {
		return nil, CursorCheckpoint{}, false, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = "cohort-watch:" + digest
	}
	if existing, checkpoint, err := r.getByIdempotencyKey(ctx, idempotencyKey); err == nil {
		return existing, checkpoint, true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, CursorCheckpoint{}, false, err
	}

	now := r.now().UTC()
	nextWake := initialWake(now, canonical.GetTriggers())
	watchID, cursorToken := uuid.NewString(), uuid.NewString()
	checkpoint := CursorCheckpoint{Token: cursorToken, Version: cursorVersion, RetentionGeneration: retentionGeneration, FilterDigest: digest}
	specJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(canonical)
	if err != nil {
		return nil, CursorCheckpoint{}, false, fmt.Errorf("marshal watch spec: %w", err)
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, CursorCheckpoint{}, false, fmt.Errorf("watch connection: %w", err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, CursorCheckpoint{}, false, fmt.Errorf("begin watch create: %w", err)
	}
	defer tx.Rollback()
	// A family retains its reviewed policy even when it acquires another watch.
	priorRows, err := tx.QueryContext(ctx, `SELECT spec_json FROM cohort_watches WHERE family_execution_id=?`, canonical.GetFamilyExecutionId())
	if err != nil {
		return nil, CursorCheckpoint{}, false, err
	}
	for priorRows.Next() {
		var raw string
		if err = priorRows.Scan(&raw); err != nil {
			priorRows.Close()
			return nil, CursorCheckpoint{}, false, err
		}
		var prior domainpb.WatchSpec
		if err = protojson.Unmarshal([]byte(raw), &prior); err != nil {
			priorRows.Close()
			return nil, CursorCheckpoint{}, false, err
		}
		if prior.GetPolicyVersion() != canonical.GetPolicyVersion() {
			priorRows.Close()
			return nil, CursorCheckpoint{}, false, errors.New("family execution already binds a different supervision policy")
		}
	}
	err = priorRows.Err()
	priorRows.Close()
	if err != nil {
		return nil, CursorCheckpoint{}, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cohort_watches
		(watch_id,idempotency_key,revision,status,family_execution_id,parent_run_id,spec_json,cursor_token,cursor_version,cursor_rowid,retention_generation,filter_digest,next_wake_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, watchID, idempotencyKey, 1, int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE), canonical.GetFamilyExecutionId(), canonical.GetParentRunId(), string(specJSON), cursorToken, cursorVersion, 0, retentionGeneration, digest, formatTime(nextWake), formatTime(now), formatTime(now))
	if err != nil {
		_ = tx.Rollback()
		if existing, existingCheckpoint, getErr := r.getByIdempotencyKey(ctx, idempotencyKey); getErr == nil {
			return existing, existingCheckpoint, true, nil
		}
		return nil, CursorCheckpoint{}, false, fmt.Errorf("insert cohort watch: %w", err)
	}
	for _, subject := range canonical.GetSubjects() {
		if _, err := tx.ExecContext(ctx, `INSERT INTO cohort_watch_subjects (watch_id,family_execution_id,plan_id,run_id) VALUES (?,?,?,?)`, watchID, subject.GetFamilyExecutionId(), subject.GetPlanId(), subject.GetRunId()); err != nil {
			return nil, CursorCheckpoint{}, false, fmt.Errorf("insert watch subject: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, CursorCheckpoint{}, false, fmt.Errorf("commit watch create: %w", err)
	}
	watch := &domainpb.CohortWatch{WatchId: watchID, Revision: 1, Status: domainpb.WatchStatus_WATCH_STATUS_ACTIVE, Spec: canonical, Cursor: &domainpb.WatchCursor{Token: cursorToken}, NextWakeAt: timestamppb.New(nextWake), CreatedAt: timestamppb.New(now), UpdatedAt: timestamppb.New(now)}
	return watch, checkpoint, false, nil
}

func (r *Repository) Get(ctx context.Context, watchID string) (*domainpb.CohortWatch, CursorCheckpoint, error) {
	return r.scanWatch(r.db.QueryRowContext(ctx, watchSelect+` WHERE watch_id = ?`, strings.TrimSpace(watchID)))
}

func (r *Repository) List(ctx context.Context, familyExecutionID string, status domainpb.WatchStatus, pageSize uint32, pageToken string) ([]*domainpb.CohortWatch, string, error) {
	if pageSize == 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset, err := decodePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
	query, args := watchSelect+` WHERE 1=1`, []any{}
	if familyExecutionID = strings.TrimSpace(familyExecutionID); familyExecutionID != "" {
		query += ` AND family_execution_id=?`
		args = append(args, familyExecutionID)
	}
	if status != domainpb.WatchStatus_WATCH_STATUS_UNSPECIFIED {
		query += ` AND status=?`
		args = append(args, int32(status))
	}
	query += ` ORDER BY created_at,watch_id LIMIT ? OFFSET ?`
	args = append(args, int(pageSize)+1, offset)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list cohort watches: %w", err)
	}
	defer rows.Close()
	watches := make([]*domainpb.CohortWatch, 0, pageSize)
	for rows.Next() {
		watch, _, err := r.scanWatch(rows)
		if err != nil {
			return nil, "", err
		}
		watches = append(watches, watch)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(watches) > int(pageSize) {
		watches = watches[:pageSize]
		next = encodePageToken(offset + int(pageSize))
	}
	return watches, next, nil
}

func (r *Repository) Cancel(ctx context.Context, watchID string, expectedRevision uint64) (*domainpb.CohortWatch, error) {
	if expectedRevision == 0 {
		return nil, errors.New("expected revision is required")
	}
	now := r.now().UTC()
	result, err := r.db.ExecContext(ctx, `UPDATE cohort_watches SET revision=revision+1,status=?,updated_at=?,terminal_at=? WHERE watch_id=? AND revision=? AND status=?`, int32(domainpb.WatchStatus_WATCH_STATUS_CANCELED), formatTime(now), formatTime(now), strings.TrimSpace(watchID), expectedRevision, int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE))
	if err != nil {
		return nil, fmt.Errorf("cancel cohort watch: %w", err)
	}
	updated, _ := result.RowsAffected()
	if updated != 1 {
		if _, _, err := r.Get(ctx, watchID); errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, ErrConflict
	}
	watch, _, err := r.Get(ctx, watchID)
	return watch, err
}

func (r *Repository) Due(ctx context.Context, at time.Time, limit int) ([]*domainpb.CohortWatch, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.QueryxContext(ctx, watchSelect+` WHERE status=? AND next_wake_at<=? ORDER BY next_wake_at,watch_id LIMIT ?`, int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE), formatTime(at), limit)
	if err != nil {
		return nil, fmt.Errorf("list due cohort watches: %w", err)
	}
	defer rows.Close()
	watches := make([]*domainpb.CohortWatch, 0, limit)
	for rows.Next() {
		watch, _, err := r.scanWatch(rows)
		if err != nil {
			return nil, err
		}
		watches = append(watches, watch)
	}
	return watches, rows.Err()
}

func (r *Repository) NextDue(ctx context.Context) (*time.Time, error) {
	var raw sql.NullString
	if err := r.db.QueryRowContext(ctx, `SELECT MIN(next_wake_at) FROM cohort_watches WHERE status=?`, int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE)).Scan(&raw); err != nil {
		return nil, fmt.Errorf("next cohort watch due time: %w", err)
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339Nano, raw.String)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *Repository) Measures(ctx context.Context, at time.Time) (Measures, error) {
	var result Measures
	queries := []struct {
		target *int64
		query  string
		args   []any
	}{
		{&result.ActiveWatches, `SELECT COUNT(*) FROM cohort_watches WHERE status=?`, []any{int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE)}},
		{&result.DueWatches, `SELECT COUNT(*) FROM cohort_watches WHERE status=? AND next_wake_at<=?`, []any{int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE), formatTime(at)}},
		{&result.Decisions, `SELECT COUNT(*) FROM cohort_watch_decisions`, nil},
		{&result.Actions, `SELECT COUNT(*) FROM cohort_watch_actions`, nil},
	}
	for _, item := range queries {
		if err := r.db.QueryRowContext(ctx, item.query, item.args...).Scan(item.target); err != nil {
			return Measures{}, fmt.Errorf("read cohort watch measures: %w", err)
		}
	}
	var pages, pageSize int64
	if err := r.db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pages); err != nil {
		return Measures{}, err
	}
	if err := r.db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&pageSize); err != nil {
		return Measures{}, err
	}
	result.DatabaseBytes = pages * pageSize
	return result, nil
}

func (r *Repository) CommitDecision(ctx context.Context, watchID string, expectedRevision uint64, before CursorCheckpoint, decision *domainpb.WatchDecision, after CursorCheckpoint, inputs ...EvaluationInput) (*domainpb.CohortWatch, error) {
	if decision == nil || strings.TrimSpace(decision.GetIdempotencyKey()) == "" {
		return nil, errors.New("decision and idempotency key are required")
	}
	if before.Token == "" || after.Token == "" || after.RowID < before.RowID || after.FilterDigest != before.FilterDigest {
		return nil, errors.New("invalid cursor transition")
	}
	now := r.now().UTC()
	stored := proto.Clone(decision).(*domainpb.WatchDecision)
	if stored.DecisionId == "" {
		stored.DecisionId = uuid.NewString()
	}
	stored.NextCursor = &domainpb.WatchCursor{Token: after.Token}
	stored.CreatedAt = timestamppb.New(now)
	decisionJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(stored)
	if err != nil {
		return nil, fmt.Errorf("marshal watch decision: %w", err)
	}
	status := domainpb.WatchStatus_WATCH_STATUS_ACTIVE
	var terminalAt any
	if stored.GetDisposition() == domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL {
		status, terminalAt = domainpb.WatchStatus_WATCH_STATUS_TERMINAL, formatTime(now)
	}
	nextWake := now
	if stored.GetNextWakeAt().IsValid() {
		nextWake = stored.GetNextWakeAt().AsTime().UTC()
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO cohort_watch_decisions (decision_id,watch_id,idempotency_key,disposition,decision_json,cursor_before,cursor_after,created_at) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, stored.GetDecisionId(), watchID, stored.GetIdempotencyKey(), int32(stored.GetDisposition()), string(decisionJSON), before.Token, after.Token, formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("insert watch decision: %w", err)
	}
	inserted, _ := result.RowsAffected()
	if inserted == 0 {
		_ = tx.Rollback()
		watch, _, getErr := r.Get(ctx, watchID)
		return watch, getErr
	}
	if len(inputs) > 0 {
		snapshot := inputs[0]
		snapshot.Events = append([]eventlog.CohortEvent(nil), snapshot.Events...)
		for i := range snapshot.Events {
			snapshot.Events[i].Data = nil
		}
		raw, encodeErr := json.Marshal(snapshot)
		if encodeErr != nil {
			return nil, encodeErr
		}
		if len(raw) > 65536 {
			return nil, errors.New("supervision replay input exceeds 64 KiB")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO supervision_evaluation_inputs(decision_id,input_json,created_at) VALUES (?,?,?)`, stored.GetDecisionId(), string(raw), formatTime(now)); err != nil {
			return nil, err
		}
	}
	if len(inputs) > 0 {
		spec := inputs[0].Watch.GetSpec()
		evidence, _ := json.Marshal(stored.GetEvidenceIds())
		for _, subject := range spec.GetSubjects() {
			_, err := tx.ExecContext(ctx, `INSERT INTO supervision_outcomes(outcome_id,idempotency_key,policy_version,family_execution_id,watch_id,decision_id,action_id,child_run_id,evidence_json,predicted_class,observed_class,overridden,counterexample,safety_violation,completion_impact,supersedes_outcome_id,created_at,expires_at) SELECT ?,?,?,?,?,?, '',?,?,?,'',0,0,0,0,'',?,? WHERE EXISTS(SELECT 1 FROM supervision_policies WHERE version=?)`, uuid.NewString(), "decision-observation:"+stored.GetDecisionId()+":"+subject.GetRunId(), spec.GetPolicyVersion(), spec.GetFamilyExecutionId(), watchID, stored.GetDecisionId(), subject.GetRunId(), string(evidence), stored.GetClassification(), formatTime(now), formatTime(now.Add(180*24*time.Hour)), spec.GetPolicyVersion())
			if err != nil {
				return nil, err
			}
		}
	}
	result, err = tx.ExecContext(ctx, `UPDATE cohort_watches SET revision=revision+1,status=?,cursor_token=?,cursor_version=?,cursor_rowid=?,retention_generation=?,filter_digest=?,next_wake_at=?,updated_at=?,terminal_at=? WHERE watch_id=? AND revision=? AND cursor_token=? AND status=?`, int32(status), after.Token, after.Version, after.RowID, after.RetentionGeneration, after.FilterDigest, formatTime(nextWake), formatTime(now), terminalAt, watchID, expectedRevision, before.Token, int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE))
	if err != nil {
		return nil, fmt.Errorf("advance watch cursor: %w", err)
	}
	updated, _ := result.RowsAffected()
	if updated != 1 {
		return nil, ErrConflict
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit watch decision: %w", err)
	}
	watch, _, err := r.Get(ctx, watchID)
	if err == nil {
		watch.LastDecision = stored
	}
	return watch, err
}

const watchSelect = `SELECT watch_id,revision,status,spec_json,cursor_token,cursor_version,cursor_rowid,retention_generation,filter_digest,next_wake_at,created_at,updated_at,COALESCE(terminal_at,''),COALESCE((SELECT decision_json FROM cohort_watch_decisions decision WHERE decision.watch_id=cohort_watches.watch_id ORDER BY created_at DESC,decision_id DESC LIMIT 1),'') FROM cohort_watches`

type scanner interface{ Scan(...any) error }

func (r *Repository) getByIdempotencyKey(ctx context.Context, key string) (*domainpb.CohortWatch, CursorCheckpoint, error) {
	return r.scanWatch(r.db.QueryRowContext(ctx, watchSelect+` WHERE idempotency_key = ?`, key))
}

func (r *Repository) scanWatch(row scanner) (*domainpb.CohortWatch, CursorCheckpoint, error) {
	var id, specJSON, token, digest, nextRaw, createdRaw, updatedRaw, terminalRaw, decisionJSON string
	var revision uint64
	var status int32
	var version int
	var rowID, retention int64
	if err := row.Scan(&id, &revision, &status, &specJSON, &token, &version, &rowID, &retention, &digest, &nextRaw, &createdRaw, &updatedRaw, &terminalRaw, &decisionJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, CursorCheckpoint{}, ErrNotFound
		}
		return nil, CursorCheckpoint{}, fmt.Errorf("scan cohort watch: %w", err)
	}
	spec := &domainpb.WatchSpec{}
	if err := protojson.Unmarshal([]byte(specJSON), spec); err != nil {
		return nil, CursorCheckpoint{}, fmt.Errorf("decode watch spec: %w", err)
	}
	next, err := time.Parse(time.RFC3339Nano, nextRaw)
	if err != nil {
		return nil, CursorCheckpoint{}, err
	}
	created, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return nil, CursorCheckpoint{}, err
	}
	updated, err := time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		return nil, CursorCheckpoint{}, err
	}
	watch := &domainpb.CohortWatch{WatchId: id, Revision: revision, Status: domainpb.WatchStatus(status), Spec: spec, Cursor: &domainpb.WatchCursor{Token: token}, NextWakeAt: timestamppb.New(next), CreatedAt: timestamppb.New(created), UpdatedAt: timestamppb.New(updated)}
	if terminalRaw != "" {
		terminal, err := time.Parse(time.RFC3339Nano, terminalRaw)
		if err != nil {
			return nil, CursorCheckpoint{}, err
		}
		watch.TerminalAt = timestamppb.New(terminal)
	}
	if decisionJSON != "" {
		watch.LastDecision = &domainpb.WatchDecision{}
		if err := protojson.Unmarshal([]byte(decisionJSON), watch.LastDecision); err != nil {
			return nil, CursorCheckpoint{}, fmt.Errorf("decode watch decision: %w", err)
		}
	}
	return watch, CursorCheckpoint{Token: token, Version: version, RowID: rowID, RetentionGeneration: retention, FilterDigest: digest}, nil
}

func canonicalSpec(spec *domainpb.WatchSpec) (*domainpb.WatchSpec, string, error) {
	if spec == nil || strings.TrimSpace(spec.GetFamilyExecutionId()) == "" || len(spec.GetSubjects()) == 0 {
		return nil, "", errors.New("family execution and at least one subject are required")
	}
	canonical := proto.Clone(spec).(*domainpb.WatchSpec)
	canonical.FamilyExecutionId = strings.TrimSpace(canonical.GetFamilyExecutionId())
	seen := map[string]bool{}
	for _, subject := range canonical.GetSubjects() {
		subject.FamilyExecutionId = strings.TrimSpace(subject.GetFamilyExecutionId())
		if subject.FamilyExecutionId == "" {
			subject.FamilyExecutionId = canonical.FamilyExecutionId
		}
		if subject.FamilyExecutionId != canonical.FamilyExecutionId || strings.TrimSpace(subject.GetRunId()) == "" {
			return nil, "", errors.New("each subject must bind a run to the watch family execution")
		}
		subject.PlanId, subject.RunId = strings.TrimSpace(subject.GetPlanId()), strings.TrimSpace(subject.GetRunId())
		if seen[subject.RunId] {
			return nil, "", fmt.Errorf("duplicate watch run %q", subject.RunId)
		}
		seen[subject.RunId] = true
	}
	sort.Slice(canonical.Subjects, func(i, j int) bool { return canonical.Subjects[i].GetRunId() < canonical.Subjects[j].GetRunId() })
	parts := []string{"v1", canonical.GetFamilyExecutionId()}
	for _, subject := range canonical.GetSubjects() {
		parts = append(parts, subject.GetPlanId(), subject.GetRunId())
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return canonical, hex.EncodeToString(sum[:]), nil
}

func initialWake(now time.Time, triggers *domainpb.WatchTriggers) time.Time {
	wake := now
	if triggers.GetQuietTime().IsValid() && triggers.GetQuietTime().AsDuration() > 0 {
		wake = now.Add(triggers.GetQuietTime().AsDuration())
	}
	if triggers.GetDeadline().IsValid() {
		deadline := triggers.GetDeadline().AsTime().UTC()
		if !deadline.After(now) || deadline.Before(wake) {
			wake = deadline
		}
	}
	return wake
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func encodePageToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("v1:" + strconv.Itoa(offset)))
}

func decodePageToken(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || !strings.HasPrefix(string(raw), "v1:") {
		return 0, errors.New("invalid cohort watch page token")
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(string(raw), "v1:"))
	if err != nil || offset < 0 {
		return 0, errors.New("invalid cohort watch page token")
	}
	return offset, nil
}

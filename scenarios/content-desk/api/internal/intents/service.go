package intents

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CurrentRevision struct{ DraftID, Body, RevisionID string }
type RevisionSource interface {
	CurrentRevision(context.Context, string) (CurrentRevision, error)
}
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type PublicationIntent struct {
	ID, DraftID, SourceRevisionID, BodySHA256, Destination string
	DesiredLocalDate, TimeZone, Daypart, Status            string
	Version                                                int64
	CreatedAt                                              time.Time
}
type SetInput struct {
	DraftID, Destination, DesiredLocalDate, TimeZone, Daypart string
	ExpectedRevisionID, ExpectedBody, IdempotencyKey          string
	ExpectedIntentVersion                                     int64
}
type Service interface {
	List(context.Context, string, string) ([]PublicationIntent, error)
	Set(context.Context, SetInput) (PublicationIntent, error)
}
type service struct {
	db        SQLExecutor
	revisions RevisionSource
}
type transactionBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func NewService(db SQLExecutor, revisions RevisionSource) Service {
	return &service{db: db, revisions: revisions}
}
func Schema() string { return schema }

func (s *service) List(ctx context.Context, draftID, destination string) ([]PublicationIntent, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,draft_id,source_revision_id,body_sha256,destination,desired_local_date,time_zone,daypart,status,version,created_at FROM publication_intents WHERE (? = '' OR draft_id = ?) AND (? = '' OR destination = ?) ORDER BY desired_local_date,created_at,id", draftID, draftID, destination, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublicationIntent
	for rows.Next() {
		var x PublicationIntent
		var created string
		if err := rows.Scan(&x.ID, &x.DraftID, &x.SourceRevisionID, &x.BodySHA256, &x.Destination, &x.DesiredLocalDate, &x.TimeZone, &x.Daypart, &x.Status, &x.Version, &created); err != nil {
			return nil, err
		}
		x.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *service) Set(ctx context.Context, in SetInput) (PublicationIntent, error) {
	in.DraftID = strings.TrimSpace(in.DraftID)
	in.Destination = strings.TrimSpace(in.Destination)
	in.TimeZone = strings.TrimSpace(in.TimeZone)
	in.Daypart = strings.TrimSpace(in.Daypart)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.DraftID == "" {
		return PublicationIntent{}, fmt.Errorf("draft id is required")
	}
	if in.Destination == "" {
		return PublicationIntent{}, fmt.Errorf("destination is required")
	}
	if in.IdempotencyKey == "" {
		return PublicationIntent{}, fmt.Errorf("idempotency key is required")
	}
	if _, err := time.Parse("2006-01-02", in.DesiredLocalDate); err != nil {
		return PublicationIntent{}, fmt.Errorf("desired local date must be YYYY-MM-DD")
	}
	if in.TimeZone == "" {
		return PublicationIntent{}, fmt.Errorf("IANA time zone is required")
	}
	if _, err := time.LoadLocation(in.TimeZone); err != nil {
		return PublicationIntent{}, fmt.Errorf("time zone must be a valid IANA location")
	}
	switch in.Daypart {
	case "", "morning", "afternoon", "evening":
	default:
		return PublicationIntent{}, fmt.Errorf("daypart must be empty, morning, afternoon, or evening")
	}
	if in.ExpectedIntentVersion < 0 {
		return PublicationIntent{}, fmt.Errorf("expected intent version cannot be negative")
	}
	bodyHash := digest([]byte(in.ExpectedBody))
	basis, err := json.Marshal(struct {
		DraftID, Destination, Date, Zone, Daypart, RevisionID, BodySHA string
		IntentVersion                                                  int64
	}{in.DraftID, in.Destination, in.DesiredLocalDate, in.TimeZone, in.Daypart, in.ExpectedRevisionID, bodyHash, in.ExpectedIntentVersion})
	if err != nil {
		return PublicationIntent{}, err
	}
	fingerprint := digest(basis)
	if old, found, e := s.operation(ctx, in.IdempotencyKey); e != nil {
		return PublicationIntent{}, e
	} else if found {
		if old.fingerprint != fingerprint {
			return PublicationIntent{}, fmt.Errorf("idempotency key was reused with a different publication-intent request")
		}
		return s.getRevision(ctx, old.revisionID)
	}
	if s.revisions == nil {
		return PublicationIntent{}, fmt.Errorf("current draft revision source is unavailable")
	}
	current, err := s.revisions.CurrentRevision(ctx, in.DraftID)
	if err != nil {
		return PublicationIntent{}, fmt.Errorf("read current draft revision: %w", err)
	}
	if current.RevisionID != in.ExpectedRevisionID || current.Body != in.ExpectedBody {
		return PublicationIntent{}, fmt.Errorf("draft changed since the plan view loaded; refresh before saving the intention")
	}
	beginner, ok := s.db.(transactionBeginner)
	if !ok {
		return PublicationIntent{}, fmt.Errorf("transactional storage is required for versioned publication intentions")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return s.recover(ctx, in.IdempotencyKey, fingerprint, err)
	}
	defer tx.Rollback()
	var oldID, oldCreated string
	var oldVersion int64
	err = tx.QueryRowContext(ctx, "SELECT id,version,created_at FROM publication_intents WHERE draft_id=? AND destination=?", in.DraftID, in.Destination).Scan(&oldID, &oldVersion, &oldCreated)
	newIntent := err == sql.ErrNoRows
	if err != nil && !newIntent {
		return PublicationIntent{}, err
	}
	if newIntent && in.ExpectedIntentVersion != 0 {
		_ = tx.Rollback()
		return s.recover(ctx, in.IdempotencyKey, fingerprint, fmt.Errorf("publication intention changed; refresh before saving"))
	}
	if !newIntent && oldVersion != in.ExpectedIntentVersion {
		_ = tx.Rollback()
		return s.recover(ctx, in.IdempotencyKey, fingerprint, fmt.Errorf("publication intention changed; refresh before saving"))
	}
	version := int64(1)
	intentID := uuid.NewString()
	if !newIntent {
		version, intentID = oldVersion+1, oldID
	}
	now := time.Now().UTC()
	item := PublicationIntent{ID: intentID, DraftID: in.DraftID, SourceRevisionID: current.RevisionID, BodySHA256: bodyHash, Destination: in.Destination, DesiredLocalDate: in.DesiredLocalDate, TimeZone: in.TimeZone, Daypart: in.Daypart, Status: "desired", Version: version, CreatedAt: now}
	if newIntent {
		_, err = tx.ExecContext(ctx, "INSERT INTO publication_intents (id,draft_id,destination,version,source_revision_id,body_sha256,desired_local_date,time_zone,daypart,status,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)", item.ID, item.DraftID, item.Destination, item.Version, item.SourceRevisionID, item.BodySHA256, item.DesiredLocalDate, item.TimeZone, item.Daypart, item.Status, now.Format(time.RFC3339Nano))
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, "UPDATE publication_intents SET version=?,source_revision_id=?,body_sha256=?,desired_local_date=?,time_zone=?,daypart=?,status='desired',created_at=? WHERE id=? AND version=?", item.Version, item.SourceRevisionID, item.BodySHA256, item.DesiredLocalDate, item.TimeZone, item.Daypart, now.Format(time.RFC3339Nano), item.ID, in.ExpectedIntentVersion)
		if err == nil {
			if n, e := result.RowsAffected(); e != nil {
				err = e
			} else if n != 1 {
				err = fmt.Errorf("publication intention changed; refresh before saving")
			}
		}
	}
	if err != nil {
		_ = tx.Rollback()
		return s.recover(ctx, in.IdempotencyKey, fingerprint, err)
	}
	revisionID := uuid.NewString()
	_, err = tx.ExecContext(ctx, "INSERT INTO publication_intent_revisions (id,intent_id,version,source_revision_id,body_sha256,desired_local_date,time_zone,daypart,created_at) VALUES (?,?,?,?,?,?,?,?,?)", revisionID, item.ID, item.Version, item.SourceRevisionID, item.BodySHA256, item.DesiredLocalDate, item.TimeZone, item.Daypart, now.Format(time.RFC3339Nano))
	if err != nil {
		_ = tx.Rollback()
		return s.recover(ctx, in.IdempotencyKey, fingerprint, err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO publication_intent_operations (idempotency_key,request_fingerprint,intent_revision_id,created_at) VALUES (?,?,?,?)", in.IdempotencyKey, fingerprint, revisionID, now.Format(time.RFC3339Nano))
	if err != nil {
		_ = tx.Rollback()
		return s.recover(ctx, in.IdempotencyKey, fingerprint, err)
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
		return s.recover(ctx, in.IdempotencyKey, fingerprint, err)
	}
	return item, nil
}

type operation struct{ fingerprint, revisionID string }

func (s *service) operation(ctx context.Context, key string) (operation, bool, error) {
	var x operation
	err := s.db.QueryRowContext(ctx, "SELECT request_fingerprint,intent_revision_id FROM publication_intent_operations WHERE idempotency_key=?", key).Scan(&x.fingerprint, &x.revisionID)
	if err == sql.ErrNoRows {
		return operation{}, false, nil
	}
	if err != nil {
		return operation{}, false, err
	}
	return x, true, nil
}
func (s *service) recover(ctx context.Context, key, fingerprint string, cause error) (PublicationIntent, error) {
	x, found, err := s.operation(ctx, key)
	if err != nil {
		return PublicationIntent{}, err
	}
	if !found {
		return PublicationIntent{}, cause
	}
	if x.fingerprint != fingerprint {
		return PublicationIntent{}, fmt.Errorf("idempotency key was reused with a different publication-intent request")
	}
	return s.getRevision(ctx, x.revisionID)
}
func (s *service) getRevision(ctx context.Context, id string) (PublicationIntent, error) {
	var x PublicationIntent
	var created string
	err := s.db.QueryRowContext(ctx, "SELECT i.id,i.draft_id,r.source_revision_id,r.body_sha256,i.destination,r.desired_local_date,r.time_zone,r.daypart,i.status,r.version,r.created_at FROM publication_intent_revisions r JOIN publication_intents i ON i.id=r.intent_id WHERE r.id=?", id).Scan(&x.ID, &x.DraftID, &x.SourceRevisionID, &x.BodySHA256, &x.Destination, &x.DesiredLocalDate, &x.TimeZone, &x.Daypart, &x.Status, &x.Version, &created)
	if err != nil {
		return PublicationIntent{}, err
	}
	x.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return x, err
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

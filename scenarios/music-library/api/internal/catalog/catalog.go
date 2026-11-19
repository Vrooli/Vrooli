package catalog

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

var (
	hashPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	ErrInvalid     = errors.New("invalid catalog input")
	ErrNotFound    = errors.New("track not found")
	ErrUnavailable = errors.New("no eligible track is available")
	ErrReservation = errors.New("reservation is missing, expired, or owned by another consumer")
)

type (
	Use struct {
		Project  string
		Revision string
		Status   string
		Purpose  string
	}
	Track struct {
		SHA256       string
		Title        string
		SourcePath   string
		TakeID       string
		ReceiptPath  string
		Tags         []string
		Purposes     []string
		BankPriority int
		BankState    string
		Approval     string
		Feedback     string
		Uses         []Use
	}
	Filter struct {
		Tag        string
		Purpose    string
		UnusedOnly bool
		Project    string
	}
	Reservation struct {
		ID        string
		SHA256    string
		Consumer  string
		Purpose   string
		Project   string
		Revision  string
		State     string
		CreatedAt time.Time
		ExpiresAt time.Time
	}
	Repository interface {
		Put(context.Context, Track) (Track, error)
		List(context.Context, Filter) ([]Track, error)
		RecordUse(context.Context, string, Use) (Track, error)
		Reserve(context.Context, ReserveRequest) (Reservation, Track, error)
		Release(context.Context, string, string) (Track, error)
		CommitReservation(context.Context, string, string, string) (Track, Use, error)
		RollbackReservation(context.Context, string, string) (Track, error)
	}
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	}
	Store struct{ db SQLExecutor }
)

type ReserveRequest struct {
	SHA256       string
	Purpose      string
	Project      string
	Revision     string
	Consumer     string
	LeaseSeconds int
}

func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

func validate(t Track) error {
	if !hashPattern.MatchString(t.SHA256) || strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.SourcePath) == "" {
		return fmt.Errorf("%w: sha256, title and source path required", ErrInvalid)
	}
	if t.Approval != "liked" && t.Approval != "unreviewed" && t.Approval != "rejected" {
		return fmt.Errorf("%w: approval must be liked, unreviewed or rejected", ErrInvalid)
	}
	if len(t.Tags) > 32 || len(t.Purposes) > 16 || len(t.Uses) > 0 {
		return fmt.Errorf("%w: too many tags or purposes, or unexpected uses", ErrInvalid)
	}
	for _, x := range t.Tags {
		if strings.TrimSpace(x) != x || x == "" || len(x) > 80 {
			return fmt.Errorf("%w: invalid tag", ErrInvalid)
		}
	}
	seen := map[string]bool{}
	for _, purpose := range t.Purposes {
		if strings.TrimSpace(purpose) != purpose || purpose == "" || len(purpose) > 120 || seen[purpose] {
			return fmt.Errorf("%w: invalid or duplicate purpose", ErrInvalid)
		}
		seen[purpose] = true
	}
	if t.BankPriority < 0 || t.BankPriority > 10000 {
		return fmt.Errorf("%w: bank priority must be 0–10000", ErrInvalid)
	}
	return nil
}

func (s *Store) Put(ctx context.Context, t Track) (Track, error) {
	if err := validate(t); err != nil {
		return Track{}, err
	}
	b, err := json.Marshal(t.Tags)
	if err != nil {
		return Track{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Track{}, fmt.Errorf("begin catalog update: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO music_catalog_tracks (sha256,title,source_path,take_id,receipt_path,tags_json,approval,feedback) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(sha256) DO UPDATE SET title=excluded.title,source_path=excluded.source_path,take_id=excluded.take_id,receipt_path=excluded.receipt_path,tags_json=excluded.tags_json,approval=excluded.approval,feedback=excluded.feedback`, t.SHA256, t.Title, t.SourcePath, t.TakeID, t.ReceiptPath, string(b), t.Approval, t.Feedback)
	if err != nil {
		return Track{}, fmt.Errorf("save track: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM music_catalog_track_purposes WHERE sha256=?`, t.SHA256); err != nil {
		return Track{}, err
	}
	for _, purpose := range t.Purposes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO music_catalog_track_purposes (sha256,purpose) VALUES (?,?)`, t.SHA256, purpose); err != nil {
			return Track{}, fmt.Errorf("save track purpose: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO music_catalog_ranks (sha256,priority) VALUES (?,?) ON CONFLICT(sha256) DO UPDATE SET priority=excluded.priority`, t.SHA256, t.BankPriority); err != nil {
		return Track{}, fmt.Errorf("save bank priority: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Track{}, fmt.Errorf("commit catalog update: %w", err)
	}
	return s.get(ctx, t.SHA256)
}

func (s *Store) get(ctx context.Context, id string) (Track, error) {
	var t Track
	var tags string
	err := s.db.QueryRowContext(ctx, `SELECT sha256,title,source_path,take_id,receipt_path,tags_json,approval,feedback FROM music_catalog_tracks WHERE sha256=?`, id).Scan(&t.SHA256, &t.Title, &t.SourcePath, &t.TakeID, &t.ReceiptPath, &tags, &t.Approval, &t.Feedback)
	if errors.Is(err, sql.ErrNoRows) {
		return Track{}, ErrNotFound
	}
	if err != nil {
		return Track{}, err
	}
	if err = json.Unmarshal([]byte(tags), &t.Tags); err != nil {
		return Track{}, err
	}
	purposeRows, err := s.db.QueryContext(ctx, `SELECT purpose FROM music_catalog_track_purposes WHERE sha256=? ORDER BY purpose`, id)
	if err != nil {
		return Track{}, err
	}
	for purposeRows.Next() {
		var purpose string
		if err := purposeRows.Scan(&purpose); err != nil {
			purposeRows.Close()
			return Track{}, err
		}
		t.Purposes = append(t.Purposes, purpose)
	}
	if err := purposeRows.Err(); err != nil {
		purposeRows.Close()
		return Track{}, err
	}
	if err := purposeRows.Close(); err != nil {
		return Track{}, err
	}
	t.BankPriority = 100
	if err := s.db.QueryRowContext(ctx, `SELECT priority FROM music_catalog_ranks WHERE sha256=?`, id).Scan(&t.BankPriority); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Track{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT project,revision,status FROM music_catalog_uses WHERE sha256=? ORDER BY project,revision`, id)
	if err != nil {
		return Track{}, err
	}
	for rows.Next() {
		var u Use
		if err = rows.Scan(&u.Project, &u.Revision, &u.Status); err != nil {
			rows.Close()
			return Track{}, err
		}
		t.Uses = append(t.Uses, u)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Track{}, err
	}
	if err := rows.Close(); err != nil {
		return Track{}, err
	}
	for i := range t.Uses {
		u := &t.Uses[i]
		if err := s.db.QueryRowContext(ctx, `SELECT purpose FROM music_catalog_use_purposes WHERE sha256=? AND project=? AND revision=?`, id, u.Project, u.Revision).Scan(&u.Purpose); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Track{}, err
		}
	}
	if err := s.db.QueryRowContext(ctx, `SELECT CASE WHEN EXISTS(SELECT 1 FROM music_catalog_reservations WHERE sha256=? AND state='active' AND expires_at>?) THEN 'reserved' WHEN EXISTS(SELECT 1 FROM music_catalog_uses WHERE sha256=? AND status='accepted') THEN 'used' WHEN EXISTS(SELECT 1 FROM music_catalog_uses WHERE sha256=?) THEN 'draft' ELSE 'available' END`, id, stamp(time.Now().UTC()), id, id).Scan(&t.BankState); err != nil {
		return Track{}, err
	}
	return t, nil
}

func (s *Store) List(ctx context.Context, f Filter) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sha256 FROM music_catalog_tracks ORDER BY title,sha256`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []Track{}
	for _, id := range ids {
		t, e := s.get(ctx, id)
		if e != nil {
			return nil, e
		}
		if f.Tag != "" {
			found := false
			for _, tag := range t.Tags {
				if tag == f.Tag {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		if f.Purpose != "" {
			found := false
			for _, purpose := range t.Purposes {
				if purpose == f.Purpose {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		used := len(t.Uses) > 0
		project := f.Project == ""
		for _, u := range t.Uses {
			if u.Project == f.Project {
				project = true
			}
		}
		if f.UnusedOnly && (used || t.BankState == "reserved") {
			continue
		}
		if !project {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) RecordUse(ctx context.Context, id string, u Use) (Track, error) {
	if !hashPattern.MatchString(id) || strings.TrimSpace(u.Project) == "" || strings.TrimSpace(u.Revision) == "" || (u.Status != "draft" && u.Status != "accepted") || (u.Purpose != "" && (strings.TrimSpace(u.Purpose) != u.Purpose || len(u.Purpose) > 120)) {
		return Track{}, fmt.Errorf("%w: project, revision and draft/accepted status required", ErrInvalid)
	}
	if _, err := s.get(ctx, id); err != nil {
		return Track{}, err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO music_catalog_uses (sha256,project,revision,status) VALUES (?,?,?,?) ON CONFLICT(sha256,project,revision) DO UPDATE SET status=CASE WHEN music_catalog_uses.status='accepted' THEN 'accepted' ELSE excluded.status END`, id, u.Project, u.Revision, u.Status)
	if err != nil {
		return Track{}, err
	}
	if u.Purpose != "" {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO music_catalog_use_purposes (sha256,project,revision,purpose) VALUES (?,?,?,?) ON CONFLICT(sha256,project,revision) DO UPDATE SET purpose=excluded.purpose`, id, u.Project, u.Revision, u.Purpose); err != nil {
			return Track{}, err
		}
	}
	return s.get(ctx, id)
}

func (s *Store) Reserve(ctx context.Context, input ReserveRequest) (Reservation, Track, error) {
	if strings.TrimSpace(input.Purpose) != input.Purpose || input.Purpose == "" || len(input.Purpose) > 120 || strings.TrimSpace(input.Project) != input.Project || input.Project == "" || len(input.Project) > 240 || strings.TrimSpace(input.Revision) != input.Revision || input.Revision == "" || len(input.Revision) > 240 || strings.TrimSpace(input.Consumer) != input.Consumer || input.Consumer == "" || len(input.Consumer) > 240 || (input.SHA256 != "" && !hashPattern.MatchString(input.SHA256)) {
		return Reservation{}, Track{}, fmt.Errorf("%w: purpose, project, revision and consumer are required", ErrInvalid)
	}
	if input.LeaseSeconds == 0 {
		input.LeaseSeconds = 900
	}
	if input.LeaseSeconds < 30 || input.LeaseSeconds > 3600 {
		return Reservation{}, Track{}, fmt.Errorf("%w: lease must be between 30 and 3600 seconds", ErrInvalid)
	}
	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return Reservation{}, Track{}, fmt.Errorf("allocate reservation identity: %w", err)
	}
	reservationID := hex.EncodeToString(idBytes[:])
	now := time.Now().UTC()
	expires := now.Add(time.Duration(input.LeaseSeconds) * time.Second)
	if _, err := s.db.ExecContext(ctx, `UPDATE music_catalog_reservations SET state='expired',completed_at=? WHERE state='active' AND expires_at<=?`, stamp(now), stamp(now)); err != nil {
		return Reservation{}, Track{}, fmt.Errorf("expire stale bank reservations: %w", err)
	}
	query := `INSERT INTO music_catalog_reservations (reservation_id,sha256,consumer,purpose,project,revision,state,created_at,expires_at) SELECT ?,t.sha256,?,?,?,?,?,?,? FROM music_catalog_tracks t JOIN music_catalog_track_purposes p ON p.sha256=t.sha256 LEFT JOIN music_catalog_ranks bank_rank ON bank_rank.sha256=t.sha256 WHERE t.approval='liked' AND p.purpose=? AND (?='' OR t.sha256=?) AND NOT EXISTS (SELECT 1 FROM music_catalog_reservations r WHERE r.sha256=t.sha256 AND r.state='active') AND (?<>'' OR NOT EXISTS (SELECT 1 FROM music_catalog_uses u WHERE u.sha256=t.sha256)) ORDER BY COALESCE(bank_rank.priority,100),t.title,t.sha256 LIMIT 1`
	result, err := s.db.ExecContext(ctx, query, reservationID, input.Consumer, input.Purpose, input.Project, input.Revision, "active", stamp(now), stamp(expires), input.Purpose, input.SHA256, input.SHA256, input.SHA256)
	if err != nil {
		return Reservation{}, Track{}, fmt.Errorf("reserve approved library track: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Reservation{}, Track{}, err
	}
	if affected == 0 {
		return Reservation{}, Track{}, ErrUnavailable
	}
	reservation, err := s.reservation(ctx, reservationID)
	if err != nil {
		return Reservation{}, Track{}, err
	}
	track, err := s.get(ctx, reservation.SHA256)
	return reservation, track, err
}

func (s *Store) Release(ctx context.Context, reservationID, consumer string) (Track, error) {
	if strings.TrimSpace(reservationID) == "" || len(reservationID) > 64 || strings.TrimSpace(consumer) == "" || len(consumer) > 240 {
		return Track{}, fmt.Errorf("%w: reservation id and consumer are required", ErrInvalid)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE music_catalog_reservations SET state='released',completed_at=? WHERE reservation_id=? AND consumer=? AND state='active' AND expires_at>?`, stamp(time.Now().UTC()), reservationID, consumer, stamp(time.Now().UTC()))
	if err != nil {
		return Track{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Track{}, err
	}
	if count != 1 {
		return Track{}, ErrReservation
	}
	var sha string
	if err := s.db.QueryRowContext(ctx, `SELECT sha256 FROM music_catalog_reservations WHERE reservation_id=?`, reservationID).Scan(&sha); err != nil {
		return Track{}, err
	}
	return s.get(ctx, sha)
}

func (s *Store) CommitReservation(ctx context.Context, reservationID, consumer, status string) (Track, Use, error) {
	if strings.TrimSpace(reservationID) == "" || len(reservationID) > 64 || strings.TrimSpace(consumer) == "" || len(consumer) > 240 || (status != "draft" && status != "accepted") {
		return Track{}, Use{}, fmt.Errorf("%w: reservation id, consumer and draft/accepted status required", ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Track{}, Use{}, err
	}
	defer tx.Rollback()
	var sha, project, revision, purpose, state, expiresAt string
	if err := tx.QueryRowContext(ctx, `SELECT sha256,project,revision,purpose,state,expires_at FROM music_catalog_reservations WHERE reservation_id=? AND consumer=?`, reservationID, consumer).Scan(&sha, &project, &revision, &purpose, &state, &expiresAt); errors.Is(err, sql.ErrNoRows) {
		return Track{}, Use{}, ErrReservation
	} else if err != nil {
		return Track{}, Use{}, err
	}
	now := time.Now().UTC()
	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return Track{}, Use{}, err
	}
	if state != "active" || !expires.After(now) {
		if state == "active" {
			_, _ = tx.ExecContext(ctx, `UPDATE music_catalog_reservations SET state='expired',completed_at=? WHERE reservation_id=? AND state='active'`, stamp(now), reservationID)
			_ = tx.Commit()
		}
		return Track{}, Use{}, ErrReservation
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO music_catalog_uses (sha256,project,revision,status) VALUES (?,?,?,?) ON CONFLICT(sha256,project,revision) DO UPDATE SET status=CASE WHEN music_catalog_uses.status='accepted' THEN 'accepted' ELSE excluded.status END`, sha, project, revision, status)
	if err != nil {
		return Track{}, Use{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO music_catalog_use_purposes (sha256,project,revision,purpose) VALUES (?,?,?,?) ON CONFLICT(sha256,project,revision) DO UPDATE SET purpose=excluded.purpose`, sha, project, revision, purpose)
	if err != nil {
		return Track{}, Use{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO music_catalog_replenish_outbox (event_id,reservation_id,sha256,purpose,project,revision,state,available_at,created_at) VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT(event_id) DO NOTHING`, reservationID, reservationID, sha, purpose, project, revision, "pending", stamp(now), stamp(now)); err != nil {
		return Track{}, Use{}, fmt.Errorf("queue candidate replenishment: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE music_catalog_reservations SET state='committed',completed_at=? WHERE reservation_id=? AND consumer=? AND state='active' AND expires_at>?`, stamp(now), reservationID, consumer, stamp(now))
	if err != nil {
		return Track{}, Use{}, err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return Track{}, Use{}, errors.Join(ErrReservation, err)
	}
	if err := tx.Commit(); err != nil {
		return Track{}, Use{}, err
	}
	track, err := s.get(ctx, sha)
	return track, Use{Project: project, Revision: revision, Status: status, Purpose: purpose}, err
}

// RollbackReservation releases a committed draft association when a film cut
// is abandoned. Accepted uses are immutable. The same transaction cancels an
// undelivered refill or asks the downstream candidate queue to cancel it.
func (s *Store) RollbackReservation(ctx context.Context, reservationID, consumer string) (Track, error) {
	if strings.TrimSpace(reservationID) == "" || len(reservationID) > 64 || strings.TrimSpace(consumer) == "" || len(consumer) > 240 {
		return Track{}, fmt.Errorf("%w: reservation id and consumer are required", ErrInvalid)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Track{}, err
	}
	defer tx.Rollback()
	var sha, project, revision, state, status string
	err = tx.QueryRowContext(ctx, `SELECT r.sha256,r.project,r.revision,r.state,u.status FROM music_catalog_reservations r JOIN music_catalog_uses u ON u.sha256=r.sha256 AND u.project=r.project AND u.revision=r.revision WHERE r.reservation_id=? AND r.consumer=?`, reservationID, consumer).Scan(&sha, &project, &revision, &state, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return Track{}, ErrReservation
	}
	if err != nil {
		return Track{}, err
	}
	if state != "committed" || status != "draft" {
		return Track{}, fmt.Errorf("%w: only a committed draft can be rolled back", ErrReservation)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM music_catalog_use_purposes WHERE sha256=? AND project=? AND revision=?`, sha, project, revision); err != nil {
		return Track{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM music_catalog_uses WHERE sha256=? AND project=? AND revision=? AND status='draft'`, sha, project, revision); err != nil {
		return Track{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE music_catalog_reservations SET state='rolled_back',completed_at=? WHERE reservation_id=? AND consumer=? AND state='committed'`, stamp(time.Now().UTC()), reservationID, consumer); err != nil {
		return Track{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE music_catalog_replenish_outbox SET state=CASE WHEN state='delivered' THEN 'cancel_pending' WHEN state='pending' THEN 'cancelled' ELSE state END,available_at=?,last_error='' WHERE reservation_id=?`, stamp(time.Now().UTC()), reservationID); err != nil {
		return Track{}, err
	}
	if err := tx.Commit(); err != nil {
		return Track{}, err
	}
	return s.get(ctx, sha)
}

func (s *Store) reservation(ctx context.Context, id string) (Reservation, error) {
	var out Reservation
	var created, expires string
	err := s.db.QueryRowContext(ctx, `SELECT reservation_id,sha256,consumer,purpose,project,revision,state,created_at,expires_at FROM music_catalog_reservations WHERE reservation_id=?`, id).Scan(&out.ID, &out.SHA256, &out.Consumer, &out.Purpose, &out.Project, &out.Revision, &out.State, &created, &expires)
	if err != nil {
		return Reservation{}, err
	}
	if out.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Reservation{}, err
	}
	if out.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires); err != nil {
		return Reservation{}, err
	}
	return out, nil
}

func stamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/targetmodel"
)

// DesktopAdmissions stores owner-issued metadata before helper admission. It
// contains no account bearer, signing key, or owner session lease token.
type DesktopAdmissions interface {
	ReserveOpen(context.Context, string, string, DesktopOpenIntent, time.Time) (DesktopOpenAttempt, bool, error)
	BindOpen(context.Context, string, string, DesktopGrant) (bool, error)
	RejectOpen(context.Context, string, string) error
	ReconcileOpen(context.Context, string, string, DesktopOpenIntent) (DesktopOpenAttempt, error)
	ExpireOpenReservations(context.Context, time.Time) error
	Put(context.Context, DesktopGrant) error
	Get(context.Context, targetmodel.SessionRef, string) (DesktopGrant, error)
	List(context.Context, targetmodel.SurfaceRef, string, string, int) (DesktopAdmissionPage, error)
}

type DesktopAdmissionPage struct {
	Grants        []DesktopGrant
	NextPageToken string
}
type admissionDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}
type SQLiteDesktopAdmissions struct{ db admissionDB }

func NewSQLiteDesktopAdmissions(ctx context.Context, db admissionDB) (*SQLiteDesktopAdmissions, error) {
	if db == nil {
		return nil, ErrDesktopAdmission
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(func() string {
		return `
CREATE TABLE IF NOT EXISTS device_control_desktop_admissions (
 session_id TEXT PRIMARY KEY,
 actor TEXT NOT NULL,
 grant_metadata TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS device_control_desktop_admissions_actor ON device_control_desktop_admissions(actor,session_id);
CREATE TABLE IF NOT EXISTS device_control_desktop_open_attempts (
 actor TEXT NOT NULL,
 request_id TEXT NOT NULL,
 payload TEXT NOT NULL,
 expires_unix INTEGER NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('reserved','forwarding','not_admitted')),
 grant_metadata TEXT,
 PRIMARY KEY(actor,request_id)
);
CREATE INDEX IF NOT EXISTS device_control_desktop_open_attempts_expiry ON device_control_desktop_open_attempts(state,expires_unix);
`
	})); err != nil {
		return nil, err
	}
	return &SQLiteDesktopAdmissions{db: db}, nil
}

func (r *SQLiteDesktopAdmissions) Put(ctx context.Context, grant DesktopGrant) error {
	if grant.Revoked || !validDesktopGrant(grant, grant.IssuedAt) || time.Now().Before(grant.IssuedAt) {
		return ErrDesktopAdmission
	}
	data, err := json.Marshal(grant)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO device_control_desktop_admissions(session_id,actor,grant_metadata) VALUES(?,?,?) ON CONFLICT(session_id) DO UPDATE SET grant_metadata=excluded.grant_metadata WHERE actor=excluded.actor AND grant_metadata=excluded.grant_metadata`, grant.Lease.Ref.SessionID, grant.Lease.Actor, string(data))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrDesktopAdmission
	}
	return nil
}

// Actor is supplied by authenticated owner context, never request JSON.
func (r *SQLiteDesktopAdmissions) Get(ctx context.Context, ref targetmodel.SessionRef, actor string) (DesktopGrant, error) {
	if ref.Validate() != nil || actor == "" {
		return DesktopGrant{}, ErrDesktopAdmission
	}
	rows, err := r.db.QueryContext(ctx, `SELECT grant_metadata FROM device_control_desktop_admissions WHERE session_id=? AND actor=?`, ref.SessionID, actor)
	if err != nil {
		return DesktopGrant{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return DesktopGrant{}, err
		}
		return DesktopGrant{}, sql.ErrNoRows
	}
	var raw string
	if err := rows.Scan(&raw); err != nil {
		return DesktopGrant{}, err
	}
	var grant DesktopGrant
	if err := json.Unmarshal([]byte(raw), &grant); err != nil {
		return DesktopGrant{}, err
	}
	if grant.Lease.Ref != ref || grant.Lease.Actor != actor || !validDesktopGrant(grant, grant.IssuedAt) {
		return DesktopGrant{}, ErrDesktopAdmission
	}
	return grant, nil
}

// List enumerates immutable admission metadata, including expired grants and
// admissions whose helper reply was lost. Presence does not establish a live
// lease or completed cleanup. Actor and surface come from authenticated owner
// context. Cursors are traversal positions, never credentials or snapshots.
func (r *SQLiteDesktopAdmissions) List(ctx context.Context, surface targetmodel.SurfaceRef, actor, token string, size int) (DesktopAdmissionPage, error) {
	if surface.Validate() != nil || actor == "" || size < 0 || size > 100 {
		return DesktopAdmissionPage{}, ErrDesktopAdmission
	}
	if size == 0 {
		size = 50
	}
	var after int64
	if token != "" {
		var err error
		after, err = strconv.ParseInt(token, 10, 64)
		if err != nil || after <= 0 || strconv.FormatInt(after, 10) != token {
			return DesktopAdmissionPage{}, ErrDesktopAdmission
		}
	}
	rows, err := r.db.QueryContext(ctx, `SELECT rowid,grant_metadata FROM device_control_desktop_admissions
WHERE actor=? AND rowid>?
 AND json_extract(grant_metadata,'$.lease.ref.surface.owner_scenario')=?
 AND json_extract(grant_metadata,'$.lease.ref.surface.surface_id')=?
 AND json_extract(grant_metadata,'$.lease.ref.surface.target.owner_scenario')=?
 AND json_extract(grant_metadata,'$.lease.ref.surface.target.resource_id')=?
 AND COALESCE(json_extract(grant_metadata,'$.lease.ref.surface.target.host_node_id'),'')=?
ORDER BY rowid LIMIT ?`, actor, after, surface.OwnerScenario, surface.SurfaceID, surface.Target.OwnerScenario, surface.Target.ResourceID, surface.Target.HostNodeID, size+1)
	if err != nil {
		return DesktopAdmissionPage{}, err
	}
	defer rows.Close()
	page := DesktopAdmissionPage{Grants: make([]DesktopGrant, 0, size)}
	var last int64
	for rows.Next() {
		var position int64
		var raw string
		if err := rows.Scan(&position, &raw); err != nil {
			return DesktopAdmissionPage{}, err
		}
		var grant DesktopGrant
		if err := json.Unmarshal([]byte(raw), &grant); err != nil {
			return DesktopAdmissionPage{}, err
		}
		if grant.Lease.Actor != actor || grant.Lease.Ref.Surface != surface || !validDesktopGrant(grant, grant.IssuedAt) {
			return DesktopAdmissionPage{}, ErrDesktopAdmission
		}
		if len(page.Grants) == size {
			page.NextPageToken = strconv.FormatInt(last, 10)
			break
		}
		page.Grants = append(page.Grants, grant)
		last = position
	}
	if err := rows.Err(); err != nil {
		return DesktopAdmissionPage{}, err
	}
	return page, nil
}

// DesktopOpenIntent is the immutable request payload. Actor and deadline come
// from the owner, not browser claims. Correlation IDs never confer authority.
type DesktopOpenIntent struct {
	Surface    targetmodel.SurfaceRef `json:"surface"`
	Control    bool                   `json:"control"`
	TTLSeconds uint32                 `json:"ttl_seconds"`
}
type DesktopOpenAttempt struct {
	ID        string
	Actor     string
	Intent    DesktopOpenIntent
	ExpiresAt time.Time
	State     string
	Grant     *DesktopGrant
}

func validOpenIdentity(id, actor string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed != uuid.Nil && parsed.String() == id && actor != "" && len(actor) <= 256
}

// ReserveOpen admits one execution of an exact actor/request pair. A duplicate
// returns its prior state and created=false; callers must not repeat acquisition.
func (r *SQLiteDesktopAdmissions) ReserveOpen(ctx context.Context, id, actor string, intent DesktopOpenIntent, expires time.Time) (DesktopOpenAttempt, bool, error) {
	if !validOpenIdentity(id, actor) || intent.Surface.Validate() != nil || intent.TTLSeconds == 0 || intent.TTLSeconds > 600 || !expires.After(time.Now()) || time.Until(expires) > 30*time.Second {
		return DesktopOpenAttempt{}, false, ErrDesktopAdmission
	}
	payload, err := json.Marshal(intent)
	if err != nil {
		return DesktopOpenAttempt{}, false, err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO device_control_desktop_open_attempts(actor,request_id,payload,expires_unix,state) VALUES(?,?,?,?,'reserved') ON CONFLICT(actor,request_id) DO NOTHING`, actor, id, string(payload), expires.UnixNano())
	if err != nil {
		return DesktopOpenAttempt{}, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return DesktopOpenAttempt{}, false, err
	}
	attempt, err := r.ReadOpen(ctx, id, actor)
	if err != nil {
		return DesktopOpenAttempt{}, false, err
	}
	if attempt.Intent != intent {
		return DesktopOpenAttempt{}, false, ErrDesktopAdmission
	}
	return attempt, n == 1, nil
}

func (r *SQLiteDesktopAdmissions) ReadOpen(ctx context.Context, id, actor string) (DesktopOpenAttempt, error) {
	if !validOpenIdentity(id, actor) {
		return DesktopOpenAttempt{}, ErrDesktopAdmission
	}
	rows, err := r.db.QueryContext(ctx, `SELECT payload,expires_unix,state,grant_metadata FROM device_control_desktop_open_attempts WHERE actor=? AND request_id=?`, actor, id)
	if err != nil {
		return DesktopOpenAttempt{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return DesktopOpenAttempt{}, err
		}
		return DesktopOpenAttempt{}, sql.ErrNoRows
	}
	var payload string
	var expiry int64
	var metadata sql.NullString
	attempt := DesktopOpenAttempt{ID: id, Actor: actor}
	if err := rows.Scan(&payload, &expiry, &attempt.State, &metadata); err != nil {
		return DesktopOpenAttempt{}, err
	}
	if err := json.Unmarshal([]byte(payload), &attempt.Intent); err != nil {
		return DesktopOpenAttempt{}, err
	}
	attempt.ExpiresAt = time.Unix(0, expiry)
	if attempt.Intent.Surface.Validate() != nil || attempt.Intent.TTLSeconds == 0 || attempt.Intent.TTLSeconds > 600 {
		return DesktopOpenAttempt{}, ErrDesktopAdmission
	}
	if metadata.Valid {
		var grant DesktopGrant
		if err := json.Unmarshal([]byte(metadata.String), &grant); err != nil {
			return DesktopOpenAttempt{}, err
		}
		if !openGrantMatches(attempt, grant) {
			return DesktopOpenAttempt{}, ErrDesktopAdmission
		}
		attempt.Grant = &grant
	}
	if (attempt.State != "reserved" && attempt.State != "forwarding" && attempt.State != "not_admitted") || expiry <= 0 || (attempt.State == "forwarding") != (attempt.Grant != nil) {
		return DesktopOpenAttempt{}, ErrDesktopAdmission
	}
	return attempt, nil
}

func openGrantMatches(attempt DesktopOpenAttempt, grant DesktopGrant) bool {
	return !grant.Revoked && !time.Now().Before(grant.IssuedAt) && validDesktopGrant(grant, grant.IssuedAt) && grant.Lease.Actor == attempt.Actor && grant.Lease.Ref.Surface == attempt.Intent.Surface && grant.Lease.Control == attempt.Intent.Control && grant.ExpiresAt.Sub(grant.IssuedAt) <= time.Duration(attempt.Intent.TTLSeconds)*time.Second
}

// BindOpen is the durable boundary before helper Open. Only the caller receiving
// claimed=true may forward. Once cancelled, a delayed worker cannot cross it.
func (r *SQLiteDesktopAdmissions) BindOpen(ctx context.Context, id, actor string, grant DesktopGrant) (bool, error) {
	attempt, err := r.ReadOpen(ctx, id, actor)
	if err != nil {
		return false, err
	}
	if !openGrantMatches(attempt, grant) {
		return false, ErrDesktopAdmission
	}
	metadata, err := json.Marshal(grant)
	if err != nil {
		return false, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE device_control_desktop_open_attempts SET state='forwarding',grant_metadata=? WHERE actor=? AND request_id=? AND state='reserved' AND expires_unix>?`, string(metadata), actor, id, time.Now().UnixNano())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 1 {
		stored, err := tx.ExecContext(ctx, `INSERT INTO device_control_desktop_admissions(session_id,actor,grant_metadata) VALUES(?,?,?) ON CONFLICT(session_id) DO UPDATE SET grant_metadata=excluded.grant_metadata WHERE actor=excluded.actor AND grant_metadata=excluded.grant_metadata`, grant.Lease.Ref.SessionID, actor, string(metadata))
		if err != nil {
			return false, err
		}
		count, err := stored.RowsAffected()
		if err != nil {
			return false, err
		}
		if count != 1 {
			return false, ErrDesktopAdmission
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := tx.Rollback(); err != nil {
		return false, err
	}
	prior, err := r.ReadOpen(ctx, id, actor)
	if err != nil {
		return false, err
	}
	if prior.State != "forwarding" || prior.Grant == nil {
		return false, ErrDesktopAdmission
	}
	previous, err := json.Marshal(prior.Grant)
	if err != nil {
		return false, err
	}
	if string(previous) != string(metadata) {
		return false, ErrDesktopAdmission
	}
	return false, nil
}

// RejectOpen can certify no helper admission only before the forwarding claim.
// It cannot turn a lost helper reply into a claim that nothing happened.
func (r *SQLiteDesktopAdmissions) RejectOpen(ctx context.Context, id, actor string) error {
	if !validOpenIdentity(id, actor) {
		return ErrDesktopAdmission
	}
	result, err := r.db.ExecContext(ctx, `UPDATE device_control_desktop_open_attempts SET state='not_admitted' WHERE actor=? AND request_id=? AND state='reserved'`, actor, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	prior, err := r.ReadOpen(ctx, id, actor)
	if err != nil {
		return err
	}
	if prior.State != "not_admitted" {
		return ErrDesktopAdmission
	}
	return nil
}

// ExpireOpenReservations is owner lifecycle reconciliation. ReadOpen never
// interprets elapsed time alone as terminal; this transition fences late workers.
func (r *SQLiteDesktopAdmissions) ExpireOpenReservations(ctx context.Context, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE device_control_desktop_open_attempts SET state='not_admitted' WHERE state='reserved' AND expires_unix<=?`, now.UnixNano())
	return err
}

// ReconcileOpen cancels an unforwarded request, including a request that has not
// arrived yet. The durable reservation prevents a later Open with this identity
// from executing. A winner of BindOpen remains forwarding and must be reconciled
// through its exact helper lease, never relabelled as no admission.
func (r *SQLiteDesktopAdmissions) ReconcileOpen(ctx context.Context, id, actor string, intent DesktopOpenIntent) (DesktopOpenAttempt, error) {
	attempt, _, err := r.ReserveOpen(ctx, id, actor, intent, time.Now().Add(10*time.Second))
	if err != nil {
		return DesktopOpenAttempt{}, err
	}
	if attempt.State == "reserved" {
		err = r.RejectOpen(ctx, id, actor)
		if err != nil && !errors.Is(err, ErrDesktopAdmission) {
			return DesktopOpenAttempt{}, err
		}
	}
	return r.ReadOpen(ctx, id, actor)
}

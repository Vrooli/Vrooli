package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqlTransactor interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func workspaceOwner(ctx context.Context) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return "", ErrUnverifiedWorkspacePrincipal
	}
	owner := principal.OwnerSubject()
	if owner == "" {
		return "", ErrUnverifiedWorkspacePrincipal
	}
	return owner, nil
}

func (r *sqliteRepository) Get(ctx context.Context) (Profile, error) {
	owner, err := workspaceOwner(ctx)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	var updated int64
	err = r.db.QueryRowContext(ctx, `SELECT id,timezone,week_start,daily_capacity_minutes,reserve_minutes,focus_session_minutes,revision,updated_at FROM actor_planning_profiles WHERE owner_subject=? AND id=?`, owner, profileID).
		Scan(&p.ID, &p.Timezone, &p.WeekStart, &p.DailyCapacityMinutes, &p.ReserveMinutes, &p.FocusSessionMinutes, &p.Revision, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		p = Profile{ID: profileID, Timezone: "UTC", WeekStart: "monday", DailyCapacityMinutes: 480, ReserveMinutes: 60, FocusSessionMinutes: 45, Revision: 1}
		p.UpdatedAt = r.clock.Now().UTC()
		_, err = r.db.ExecContext(ctx, `INSERT OR IGNORE INTO actor_planning_profiles (owner_subject,id,timezone,week_start,daily_capacity_minutes,reserve_minutes,focus_session_minutes,revision,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, owner, p.ID, p.Timezone, p.WeekStart, p.DailyCapacityMinutes, p.ReserveMinutes, p.FocusSessionMinutes, p.Revision, p.UpdatedAt.Unix())
		if err != nil {
			return Profile{}, err
		}
		// A concurrent first read may have won the INSERT. Read again so both
		// callers observe the same authoritative profile and revision.
		return r.Get(ctx)
	}
	if err != nil {
		return Profile{}, err
	}
	p.UpdatedAt = time.Unix(updated, 0).UTC()
	return p, nil
}

func (r *sqliteRepository) Update(ctx context.Context, in UpdateInput) (Profile, error) {
	owner, err := workspaceOwner(ctx)
	if err != nil {
		return Profile{}, err
	}
	now := r.clock.Now().UTC()
	res, err := r.db.ExecContext(ctx, `UPDATE actor_planning_profiles SET timezone=?,week_start=?,daily_capacity_minutes=?,reserve_minutes=?,focus_session_minutes=?,revision=revision+1,updated_at=? WHERE owner_subject=? AND id=? AND revision=?`, in.Timezone, in.WeekStart, in.DailyCapacityMinutes, in.ReserveMinutes, in.FocusSessionMinutes, now.Unix(), owner, profileID, in.ExpectedRevision)
	if err != nil {
		return Profile{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Profile{}, err
	}
	if n != 1 {
		return Profile{}, ErrRevisionConflict{}
	}
	return r.Get(ctx)
}

func (r *sqliteRepository) ListAvailability(ctx context.Context) (Availability, error) {
	owner, err := workspaceOwner(ctx)
	if err != nil {
		return Availability{}, err
	}
	p, err := r.Get(ctx)
	if err != nil {
		return Availability{}, err
	}
	a := Availability{Revision: p.Revision}
	rows, err := r.db.QueryContext(ctx, `SELECT id,weekday,start_minute,end_minute,timezone,effective_start_date,effective_end_date,priority,revision FROM actor_availability_rules WHERE owner_subject=? ORDER BY weekday,start_minute,id`, owner)
	if err != nil {
		return Availability{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var w AvailabilityWindow
		if err := rows.Scan(&w.ID, &w.Weekday, &w.StartMinute, &w.EndMinute, &w.Timezone, &w.EffectiveStartDate, &w.EffectiveEndDate, &w.Priority, &w.Revision); err != nil {
			return Availability{}, err
		}
		a.Windows = append(a.Windows, w)
	}
	if err := rows.Err(); err != nil {
		return Availability{}, err
	}
	erows, err := r.db.QueryContext(ctx, `SELECT id,date,start_minute,end_minute,kind,reason,revision FROM actor_availability_exceptions WHERE owner_subject=? ORDER BY date,start_minute,id`, owner)
	if err != nil {
		return Availability{}, err
	}
	defer erows.Close()
	for erows.Next() {
		var e AvailabilityException
		if err := erows.Scan(&e.ID, &e.Date, &e.StartMinute, &e.EndMinute, &e.Kind, &e.Reason, &e.Revision); err != nil {
			return Availability{}, err
		}
		a.Exceptions = append(a.Exceptions, e)
	}
	return a, erows.Err()
}

func (r *sqliteRepository) ReplaceAvailability(ctx context.Context, in AvailabilityInput) (Availability, error) {
	owner, err := workspaceOwner(ctx)
	if err != nil {
		return Availability{}, err
	}
	p, err := r.Get(ctx)
	if err != nil {
		return Availability{}, err
	}
	if p.Revision != in.ExpectedRevision {
		return Availability{}, ErrRevisionConflict{}
	}
	db, ok := r.db.(sqlTransactor)
	if !ok {
		return Availability{}, errors.New("workspace database does not support transactions")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Availability{}, err
	}
	defer tx.Rollback()
	now := r.clock.Now().UTC().Unix()
	if _, err = tx.ExecContext(ctx, `DELETE FROM actor_availability_rules WHERE owner_subject=?`, owner); err != nil {
		return Availability{}, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM actor_availability_exceptions WHERE owner_subject=?`, owner); err != nil {
		return Availability{}, err
	}
	for i, w := range in.Windows {
		id := w.ID
		if id == "" {
			id = fmt.Sprintf("window-%d", i+1)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO actor_availability_rules (owner_subject,id,weekday,start_minute,end_minute,timezone,effective_start_date,effective_end_date,priority,revision) VALUES (?,?,?,?,?,?,?,?,?,?)`, owner, id, w.Weekday, w.StartMinute, w.EndMinute, w.Timezone, w.EffectiveStartDate, w.EffectiveEndDate, w.Priority, p.Revision+1); err != nil {
			return Availability{}, err
		}
	}
	for i, e := range in.Exceptions {
		id := e.ID
		if id == "" {
			id = fmt.Sprintf("exception-%d", i+1)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO actor_availability_exceptions (owner_subject,id,date,start_minute,end_minute,kind,reason,revision) VALUES (?,?,?,?,?,?,?,?)`, owner, id, e.Date, e.StartMinute, e.EndMinute, e.Kind, e.Reason, p.Revision+1); err != nil {
			return Availability{}, err
		}
	}
	res, err := tx.ExecContext(ctx, `UPDATE actor_planning_profiles SET revision=revision+1,updated_at=? WHERE owner_subject=? AND id=? AND revision=?`, now, owner, profileID, in.ExpectedRevision)
	if err != nil {
		return Availability{}, err
	}
	if n, affectedErr := res.RowsAffected(); affectedErr != nil {
		return Availability{}, affectedErr
	} else if n != 1 {
		return Availability{}, ErrRevisionConflict{}
	}
	if err := tx.Commit(); err != nil {
		return Availability{}, err
	}
	return r.ListAvailability(ctx)
}

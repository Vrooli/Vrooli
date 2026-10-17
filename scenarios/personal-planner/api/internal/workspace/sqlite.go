package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Get(ctx context.Context) (Profile, error) {
	var p Profile
	var updated int64
	err := r.db.QueryRowContext(ctx, `SELECT id,timezone,week_start,daily_capacity_minutes,reserve_minutes,focus_session_minutes,revision,updated_at FROM planning_profiles WHERE id=?`, profileID).
		Scan(&p.ID, &p.Timezone, &p.WeekStart, &p.DailyCapacityMinutes, &p.ReserveMinutes, &p.FocusSessionMinutes, &p.Revision, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		p = Profile{ID: profileID, Timezone: "UTC", WeekStart: "monday", DailyCapacityMinutes: 480, ReserveMinutes: 60, FocusSessionMinutes: 45, Revision: 1}
		p.UpdatedAt = r.clock.Now().UTC()
		_, err = r.db.ExecContext(ctx, `INSERT OR IGNORE INTO planning_profiles (id,timezone,week_start,daily_capacity_minutes,reserve_minutes,focus_session_minutes,revision,updated_at) VALUES (?,?,?,?,?,?,?,?)`, p.ID, p.Timezone, p.WeekStart, p.DailyCapacityMinutes, p.ReserveMinutes, p.FocusSessionMinutes, p.Revision, p.UpdatedAt.Unix())
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
	now := r.clock.Now().UTC()
	res, err := r.db.ExecContext(ctx, `UPDATE planning_profiles SET timezone=?,week_start=?,daily_capacity_minutes=?,reserve_minutes=?,focus_session_minutes=?,revision=revision+1,updated_at=? WHERE id=? AND revision=?`, in.Timezone, in.WeekStart, in.DailyCapacityMinutes, in.ReserveMinutes, in.FocusSessionMinutes, now.Unix(), profileID, in.ExpectedRevision)
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
	p, err := r.Get(ctx)
	if err != nil {
		return Availability{}, err
	}
	a := Availability{Revision: p.Revision}
	rows, err := r.db.QueryContext(ctx, `SELECT id,weekday,start_minute,end_minute,timezone,effective_start_date,effective_end_date,priority,revision FROM availability_rules ORDER BY weekday,start_minute,id`)
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
	erows, err := r.db.QueryContext(ctx, `SELECT id,date,start_minute,end_minute,kind,reason,revision FROM availability_exceptions ORDER BY date,start_minute,id`)
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
	p, err := r.Get(ctx)
	if err != nil {
		return Availability{}, err
	}
	if p.Revision != in.ExpectedRevision {
		return Availability{}, ErrRevisionConflict{}
	}
	now := r.clock.Now().UTC().Unix()
	if _, err = r.db.ExecContext(ctx, `DELETE FROM availability_rules`); err != nil {
		return Availability{}, err
	}
	if _, err = r.db.ExecContext(ctx, `DELETE FROM availability_exceptions`); err != nil {
		return Availability{}, err
	}
	for i, w := range in.Windows {
		id := w.ID
		if id == "" {
			id = fmt.Sprintf("window-%d", i+1)
		}
		if _, err = r.db.ExecContext(ctx, `INSERT INTO availability_rules (id,weekday,start_minute,end_minute,timezone,effective_start_date,effective_end_date,priority,revision) VALUES (?,?,?,?,?,?,?,?,?)`, id, w.Weekday, w.StartMinute, w.EndMinute, w.Timezone, w.EffectiveStartDate, w.EffectiveEndDate, w.Priority, p.Revision+1); err != nil {
			return Availability{}, err
		}
	}
	for i, e := range in.Exceptions {
		id := e.ID
		if id == "" {
			id = fmt.Sprintf("exception-%d", i+1)
		}
		if _, err = r.db.ExecContext(ctx, `INSERT INTO availability_exceptions (id,date,start_minute,end_minute,kind,reason,revision) VALUES (?,?,?,?,?,?,?)`, id, e.Date, e.StartMinute, e.EndMinute, e.Kind, e.Reason, p.Revision+1); err != nil {
			return Availability{}, err
		}
	}
	if _, err = r.db.ExecContext(ctx, `UPDATE planning_profiles SET revision=revision+1,updated_at=? WHERE id=? AND revision=?`, now, profileID, in.ExpectedRevision); err != nil {
		return Availability{}, err
	}
	return r.ListAvailability(ctx)
}

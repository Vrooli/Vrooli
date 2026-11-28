package hub

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"notification-hub/internal/dbvalue"
)

// Quiet windows are stored one per recipient and weekday. A window belongs to
// the weekday it starts on: when End is earlier than Start it is overnight, runs
// from Start on that weekday to End on the next one, and is read in the
// window's own timezone, so Saturday 22:00 to 07:00 covers Sunday morning and
// Sunday 22:00 to 07:00 covers Monday morning, wrapping the week.

// SetQuietWindow replaces the recipient's window for window.Weekday.
func (s *Service) SetQuietWindow(ctx context.Context, recipient string, window QuietWindow) (QuietWindow, error) {
	if recipient == "" || window.Weekday < 0 || window.Weekday > 6 || window.Start == "" || window.End == "" || window.Timezone == "" {
		return QuietWindow{}, fmt.Errorf("%w: recipient, weekday, start, end, and timezone are required", ErrInvalidArgument)
	}
	if _, _, ok := quietMinutes(window.Start, window.End); !ok {
		return QuietWindow{}, fmt.Errorf("%w: quiet window times must use 24-hour HH:MM", ErrInvalidArgument)
	}
	if window.Start == window.End {
		return QuietWindow{}, fmt.Errorf("%w: quiet window start and end must differ", ErrInvalidArgument)
	}
	if _, err := time.LoadLocation(window.Timezone); err != nil {
		return QuietWindow{}, fmt.Errorf("%w: invalid timezone", ErrInvalidArgument)
	}
	if err := s.EnsureRecipient(ctx, recipient); err != nil {
		return QuietWindow{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return QuietWindow{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM quiet_windows WHERE recipient_id = (SELECT id FROM recipients WHERE subject=?) AND weekday = ?`, recipient, window.Weekday); err != nil {
		return QuietWindow{}, err
	}
	window.ID = uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO quiet_windows (id, recipient_id, weekday, start_time, end_time, timezone, critical_override) VALUES (?, (SELECT id FROM recipients WHERE subject=?), ?, ?, ?, ?, ?)`, window.ID, recipient, window.Weekday, window.Start, window.End, window.Timezone, dbvalue.BoolInt(window.CriticalOverride)); err != nil {
		return QuietWindow{}, err
	}
	if err := tx.Commit(); err != nil {
		return QuietWindow{}, err
	}
	return window, nil
}

// inQuietWindow reports whether now falls inside any of the recipient's quiet
// windows. Every active window is considered, because an overnight window can
// overlap the next day's own window: a critical notification passes only when
// each active window allows the override.
func (s *Service) inQuietWindow(ctx context.Context, recipient, urgency string, now time.Time) (bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT weekday, start_time, end_time, timezone, critical_override FROM quiet_windows WHERE recipient_id = (SELECT id FROM recipients WHERE subject=?)`, recipient)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	quiet, overridable := false, true
	for rows.Next() {
		var weekday int
		var start, end, timezone string
		var criticalOverride int
		if err := rows.Scan(&weekday, &start, &end, &timezone, &criticalOverride); err != nil {
			return false, err
		}
		location, err := time.LoadLocation(timezone)
		if err != nil {
			continue
		}
		from, until, ok := quietMinutes(start, end)
		if !ok {
			continue
		}
		if !quietWindowCovers(weekday, from, until, now.In(location)) {
			continue
		}
		quiet = true
		if criticalOverride == 0 {
			overridable = false
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if quiet && urgency == "critical" && overridable {
		return false, nil
	}
	return quiet, nil
}

// quietMinutes parses two 24-hour HH:MM strings into minutes after midnight.
func quietMinutes(start, end string) (from, until int, ok bool) {
	parse := func(value string) (int, bool) {
		parsed, err := time.Parse("15:04", value)
		if err != nil || parsed.Format("15:04") != value {
			return 0, false
		}
		return parsed.Hour()*60 + parsed.Minute(), true
	}
	from, fromOK := parse(start)
	until, untilOK := parse(end)
	return from, until, fromOK && untilOK
}

// quietWindowCovers reports whether local, already in the window's timezone,
// falls inside the window anchored on weekday. A same-day window covers
// [from, until) on its weekday. An overnight window (until < from) covers
// [from, midnight) on its weekday and [midnight, until) on the next.
func quietWindowCovers(weekday, from, until int, local time.Time) bool {
	current := local.Hour()*60 + local.Minute()
	day := int(local.Weekday())
	if from < until {
		return day == weekday && current >= from && current < until
	}
	if day == weekday && current >= from {
		return true
	}
	return day == (weekday+1)%7 && current < until
}

package review

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestDailySeparatesRecordedActivityFromUnknownCoverage(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE focus_sessions (active_seconds INTEGER, started_at INTEGER)`,
		`CREATE TABLE manual_actuals (reported_minutes INTEGER, local_date TEXT)`,
		`CREATE TABLE goals (status TEXT)`,
		`CREATE TABLE calendar_allocations (local_date TEXT, duration_minutes INTEGER, state TEXT)`,
		`INSERT INTO focus_sessions(active_seconds, started_at) VALUES (1500, 1789808400)`,
		`INSERT INTO goals(status) VALUES ('active')`,
		`INSERT INTO calendar_allocations(local_date, duration_minutes, state) VALUES ('2026-09-19', 45, 'accepted')`,
		`INSERT INTO manual_actuals(reported_minutes, local_date) VALUES (10, '2026-09-19')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.NewFake(time.Unix(1789822800, 0).UTC())
	got, err := NewService(db, clock).Daily(context.Background(), "2026-09-19")
	if err != nil {
		t.Fatal(err)
	}
	if got.RecordedActiveMinutes != 35 || got.FocusSessionCount != 1 || got.ActiveGoalCount != 1 || got.PlannedMinutes != 45 || got.UnrecordedMinutes != 0 {
		t.Fatalf("unexpected summary: %#v", got)
	}
	if got.CoverageNote == "" {
		t.Fatal("expected coverage note")
	}
}

func TestWeeklyAggregatesSevenDailySummariesWithoutCompletionClaim(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-weekly-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE focus_sessions (active_seconds INTEGER, started_at INTEGER)`,
		`CREATE TABLE manual_actuals (reported_minutes INTEGER, local_date TEXT)`,
		`CREATE TABLE goals (status TEXT)`,
		`CREATE TABLE calendar_allocations (local_date TEXT, duration_minutes INTEGER, state TEXT)`,
		`INSERT INTO focus_sessions(active_seconds, started_at) VALUES (3600, 1789473600)`,
		`INSERT INTO manual_actuals(reported_minutes, local_date) VALUES (15, '2026-09-17')`,
		`INSERT INTO calendar_allocations(local_date, duration_minutes, state) VALUES ('2026-09-15', 120, 'accepted')`,
		`INSERT INTO calendar_allocations(local_date, duration_minutes, state) VALUES ('2026-09-17', 30, 'accepted')`,
		`INSERT INTO goals(status) VALUES ('active')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	got, err := NewService(db, schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC))).Weekly(context.Background(), "2026-09-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Days) != 7 || got.PlannedMinutes != 150 || got.RecordedActiveMinutes != 75 || got.FocusSessionCount != 1 || got.ActiveGoalCount != 1 {
		t.Fatalf("unexpected weekly summary: %#v", got)
	}
	if got.Days[1].PlannedMinutes != 120 || got.Days[3].PlannedMinutes != 30 || got.CoverageNote == "" {
		t.Fatalf("unexpected daily breakdown: %#v", got.Days)
	}
}

func TestReflectionIsOptionalAndDurable(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-reflection-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE review_reflections (local_date TEXT PRIMARY KEY, text TEXT NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	clock := schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC))
	service := NewService(db, clock)
	empty, err := service.Reflection(context.Background(), "2026-09-19")
	if err != nil || empty.Text != "" {
		t.Fatalf("expected empty optional reflection, got %#v, %v", empty, err)
	}
	saved, err := service.SaveReflection(context.Background(), "2026-09-19", "Protect the first hour.")
	if err != nil || saved.Text == "" {
		t.Fatalf("save reflection: %#v, %v", saved, err)
	}
	loaded, err := service.Reflection(context.Background(), "2026-09-19")
	if err != nil || loaded.Text != saved.Text {
		t.Fatalf("load reflection: %#v, %v", loaded, err)
	}
	if _, err := service.SaveReflection(context.Background(), "2026-09-19", "   "); err == nil {
		t.Fatal("expected blank reflection rejection")
	}
}

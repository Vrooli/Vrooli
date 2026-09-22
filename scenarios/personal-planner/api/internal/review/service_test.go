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

func TestWinIsOptionalAndDurable(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-win-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE review_wins (local_date TEXT PRIMARY KEY, text TEXT NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	service := NewService(db, schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)))
	empty, err := service.Win(context.Background(), "2026-09-19")
	if err != nil || empty.Text != "" {
		t.Fatalf("expected empty win, got %#v, %v", empty, err)
	}
	saved, err := service.SaveWin(context.Background(), "2026-09-19", "Protected the first hour.")
	if err != nil || saved.Text == "" {
		t.Fatalf("save win: %#v, %v", saved, err)
	}
	loaded, err := service.Win(context.Background(), "2026-09-19")
	if err != nil || loaded.Text != saved.Text {
		t.Fatalf("load win: %#v, %v", loaded, err)
	}
	if _, err := service.SaveWin(context.Background(), "2026-09-19", "   "); err == nil {
		t.Fatal("expected blank win rejection")
	}
}

func TestCalibrationPersistsLinkedPlanToActualReadModel(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-calibration-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE manual_actuals (reported_minutes INTEGER, allocation_id TEXT, local_date TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT PRIMARY KEY, duration_minutes INTEGER)`,
		`CREATE TABLE estimation_bias (period TEXT PRIMARY KEY, avg_error_percent REAL, over_ratio REAL, under_ratio REAL, by_category_json TEXT, accuracy_trend_json TEXT, sample_size INTEGER, updated_at TEXT)`,
		`INSERT INTO calendar_allocations(id, duration_minutes) VALUES ('allocation-1', 60), ('allocation-2', 30)`,
		`INSERT INTO manual_actuals(reported_minutes, allocation_id, local_date) VALUES (90, 'allocation-1', '2026-09-19'), (15, 'allocation-2', '2026-09-19')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(db, schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)))
	got, err := service.Calibration(context.Background(), "all_time")
	if err != nil {
		t.Fatal(err)
	}
	if got.SampleSize != 2 || got.AvgErrorPercent != 0 || got.OverRatio != .5 || got.UnderRatio != .5 {
		t.Fatalf("unexpected calibration: %#v", got)
	}
	var stored float64
	if err := db.QueryRow(`SELECT avg_error_percent FROM estimation_bias WHERE period='all_time'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != got.AvgErrorPercent {
		t.Fatalf("stored calibration %v does not match response %v", stored, got.AvgErrorPercent)
	}
}

func TestGoalVariancesReadStoredDatesAndLabelPositiveAndNegativeDelta(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-goal-variance-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE goals (id TEXT PRIMARY KEY, target_date TEXT, completed_date TEXT)`,
		`CREATE TABLE milestones (goal_id TEXT, original_due_date TEXT, due_date TEXT, completed_date TEXT)`,
		`INSERT INTO goals VALUES ('early','2026-09-20','2026-09-18'), ('over','2026-09-20','2026-09-23')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	items, err := NewService(db, schedule.NewFake(time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC))).GoalVariances(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Label != "2 days early" || items[0].DeltaDays != -2 || items[1].Label != "3 days over" || items[1].DeltaDays != 3 {
		t.Fatalf("unexpected goal variance read model: %#v", items)
	}
}

func TestTodaySignalsReportsOverdueWorkAndMomentum(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-today-signals-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE calendar_allocations (work_item_id TEXT, local_date TEXT, duration_minutes INTEGER, state TEXT)`,
		`CREATE TABLE work_items (id TEXT, status TEXT)`,
		`CREATE TABLE focus_sessions (started_at INTEGER)`,
		`CREATE TABLE manual_actuals (local_date TEXT)`,
		`INSERT INTO work_items VALUES ('w-1','open'),('w-2','complete')`,
		`INSERT INTO calendar_allocations VALUES ('w-1','2026-09-18',45,'accepted'),('w-2','2026-09-17',30,'accepted')`,
		`INSERT INTO focus_sessions VALUES (1789819200)`,
		`INSERT INTO manual_actuals VALUES ('2026-09-18')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewService(db, schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC))).TodaySignals(context.Background(), "2026-09-19")
	if err != nil {
		t.Fatal(err)
	}
	if got.OverdueCount != 1 || got.OverdueMinutes != 45 || got.MomentumDays != 2 || got.Label != "1 overdue" {
		t.Fatalf("unexpected today signals: %#v", got)
	}
}

func TestGoalDriftsCompareStoredProgressToElapsedPlan(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-goal-drift-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE goals (id TEXT, progress_basis_points INTEGER, target_date TEXT, created_at INTEGER, status TEXT)`); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC).Unix()
	if _, err := db.Exec(`INSERT INTO goals VALUES ('goal-1',1000,'2026-10-01',?,'active')`, created); err != nil {
		t.Fatal(err)
	}
	got, err := NewService(db, schedule.NewFake(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))).GoalDrifts(context.Background(), "2026-09-16")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].TargetDate != "2026-10-01" || got[0].Label != "Behind pace" || got[0].ActualBasis != 1000 || got[0].ExpectedBasis < 4800 {
		t.Fatalf("unexpected goal drift: %#v", got)
	}
}

func TestRemindersReadUpcomingOverdueAndGoalTargets(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-reminders-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE calendar_allocations (id TEXT, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT)`,
		`CREATE TABLE work_items (id TEXT, title TEXT, status TEXT)`,
		`CREATE TABLE goals (id TEXT, title TEXT, status TEXT, target_date TEXT)`,
		`CREATE TABLE reminder_preferences (id TEXT PRIMARY KEY, enabled INTEGER, quiet_start_minutes INTEGER, quiet_end_minutes INTEGER, lead_minutes INTEGER, updated_at TEXT)`,
		`INSERT INTO reminder_preferences VALUES ('workspace',1,1320,420,60,'2026-09-21T00:00:00Z')`,
		`INSERT INTO work_items VALUES ('w-overdue','Review notes','active')`,
		`INSERT INTO work_items VALUES ('w-upcoming','Write brief','active')`,
		`INSERT INTO calendar_allocations VALUES ('a-overdue','w-overdue','2026-09-21',480,30,'accepted')`,
		`INSERT INTO calendar_allocations VALUES ('a-upcoming','w-upcoming','2026-09-21',620,30,'accepted')`,
		`INSERT INTO goals VALUES ('g-1','Ship planner','active','2026-09-21')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewService(db, schedule.NewFake(time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC))).Reminders(context.Background(), "2026-09-21")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Kind != "overdue" || got[1].Kind != "upcoming" || got[2].Kind != "goal" {
		t.Fatalf("unexpected reminders: %#v", got)
	}
}

func TestReminderPreferencesPersistAndQuietHoursSuppressReminders(t *testing.T) {
	db, err := sql.Open("sqlite", "file:review-reminder-preferences-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE reminder_preferences (id TEXT PRIMARY KEY, enabled INTEGER, quiet_start_minutes INTEGER, quiet_end_minutes INTEGER, lead_minutes INTEGER, updated_at TEXT)`,
		`INSERT INTO reminder_preferences VALUES ('workspace',1,1320,420,60,'')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.NewFake(time.Date(2026, 9, 21, 23, 0, 0, 0, time.UTC))
	service := NewService(db, clock)
	preferences, err := service.SaveReminderPreferences(context.Background(), ReminderPreferences{Enabled: true, QuietStartMinutes: 22 * 60, QuietEndMinutes: 7 * 60, LeadMinutes: 30})
	if err != nil || preferences.LeadMinutes != 30 || preferences.UpdatedAt == "" {
		t.Fatalf("save reminder preferences: %#v, %v", preferences, err)
	}
	reminders, err := service.Reminders(context.Background(), "2026-09-21")
	if err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 0 {
		t.Fatalf("quiet hours should suppress reminders, got %#v", reminders)
	}
	if _, err := service.SaveReminderPreferences(context.Background(), ReminderPreferences{Enabled: true, QuietStartMinutes: 1440, QuietEndMinutes: 0, LeadMinutes: 30}); err == nil {
		t.Fatal("expected invalid quiet-hour rejection")
	}
}

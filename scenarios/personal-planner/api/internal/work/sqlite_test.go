package work

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestSQLiteSnoozeHidesUntilDateAndRecordsReason(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-snooze-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_snoozes (id TEXT PRIMARY KEY, work_item_id TEXT, until_date TEXT, reason TEXT, created_at TEXT)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','30',30,'open','', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepositoryWithOptions(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)), RepositoryOptions{IDGenerator: func() string { return "snooze-1" }})
	if err := repo.Snooze(context.Background(), "work-1", "2026-09-22", "blocked"); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("snoozed item was listed: %#v", items)
	}
	var reason string
	if err := db.QueryRow(`SELECT reason FROM work_item_snoozes WHERE id='snooze-1'`).Scan(&reason); err != nil || reason != "blocked" {
		t.Fatalf("reason=%q err=%v", reason, err)
	}
}

func TestSQLiteCompleteRecordsHistoryAndAdvancesLinkedGoal(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-complete-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_completion_history (id TEXT PRIMARY KEY, work_item_id TEXT, original_estimate_minutes INTEGER, final_actual_minutes INTEGER, variance_minutes INTEGER, created_date TEXT, completed_date TEXT)`,
		`CREATE TABLE milestones (id TEXT PRIMARY KEY, goal_id TEXT, status TEXT, completed_date TEXT, updated_at INTEGER, revision INTEGER)`,
		`CREATE TABLE milestone_work_links (milestone_id TEXT PRIMARY KEY, work_item_id TEXT)`,
		`CREATE TABLE milestone_prerequisites (milestone_id TEXT, prerequisite_milestone_id TEXT)`,
		`CREATE TABLE goals (id TEXT PRIMARY KEY, progress_method TEXT, status TEXT, progress_basis_points INTEGER, completed_date TEXT, updated_at INTEGER, revision INTEGER)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','30',60,'open','', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T00:00:00Z')`,
		`INSERT INTO goals VALUES ('goal-1','milestones','active',0,'',0,1)`,
		`INSERT INTO milestones VALUES ('milestone-1','goal-1','open','',0,1)`,
		`INSERT INTO milestone_work_links VALUES ('milestone-1','work-1')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepositoryWithOptions(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)), RepositoryOptions{IDGenerator: func() string { return "completion-1" }})
	if err := repo.Complete(context.Background(), "work-1"); err != nil {
		t.Fatal(err)
	}
	var status, goalStatus string
	if err := db.QueryRow(`SELECT status FROM work_items WHERE id='work-1'`).Scan(&status); err != nil || status != "complete" {
		t.Fatalf("work status=%q err=%v", status, err)
	}
	if err := db.QueryRow(`SELECT status FROM milestones WHERE id='milestone-1'`).Scan(&status); err != nil || status != "complete" {
		t.Fatalf("milestone status=%q err=%v", status, err)
	}
	if err := db.QueryRow(`SELECT status FROM goals WHERE id='goal-1'`).Scan(&goalStatus); err != nil || goalStatus != "complete" {
		t.Fatalf("goal status=%q err=%v", goalStatus, err)
	}
	var historyID string
	if err := db.QueryRow(`SELECT id FROM work_item_completion_history WHERE work_item_id='work-1'`).Scan(&historyID); err != nil || historyID != "completion-1" {
		t.Fatalf("completion history=%q err=%v", historyID, err)
	}
}

func TestSQLiteUpdateEstimatePreservesOriginalAndRecordsReason(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-estimate-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_estimation_changes (id TEXT PRIMARY KEY, work_item_id TEXT, previous_minutes INTEGER, new_minutes INTEGER, reason_code TEXT, changed_at TEXT)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','30',60,'open','', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepositoryWithOptions(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)), RepositoryOptions{IDGenerator: func() string { return "estimate-1" }})
	if err := repo.UpdateEstimate(context.Background(), "work-1", 75, "scope_changed"); err != nil {
		t.Fatal(err)
	}
	var remaining, original, previous, next int
	if err := db.QueryRow(`SELECT remaining_minutes,original_estimate_minutes FROM work_items WHERE id='work-1'`).Scan(&remaining, &original); err != nil || remaining != 75 || original != 60 {
		t.Fatalf("remaining=%d original=%d err=%v", remaining, original, err)
	}
	var reason string
	if err := db.QueryRow(`SELECT previous_minutes,new_minutes,reason_code FROM work_item_estimation_changes WHERE id='estimate-1'`).Scan(&previous, &next, &reason); err != nil || previous != 30 || next != 75 || reason != "scope_changed" {
		t.Fatalf("previous=%d next=%d reason=%q err=%v", previous, next, reason, err)
	}
}

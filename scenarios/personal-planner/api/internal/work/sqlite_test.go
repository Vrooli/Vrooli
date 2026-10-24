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
	clock := schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepositoryWithOptions(db, clock, RepositoryOptions{IDGenerator: func() string { return "snooze-1" }})
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
	clock.Advance(24 * time.Hour)
	items, err = repo.List(context.Background(), 100)
	if err != nil || len(items) != 1 || items[0].ID != "work-1" {
		t.Fatalf("item did not reappear after its snooze date: %#v err=%v", items, err)
	}
	var historyCount int
	var workUntil, workUpdatedAt, historyUntil, reason, historyCreatedAt string
	if err := db.QueryRow(`SELECT snoozed_until,updated_at FROM work_items WHERE id='work-1'`).Scan(&workUntil, &workUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*),MAX(until_date),MAX(reason),MAX(created_at) FROM work_item_snoozes WHERE work_item_id='work-1'`).Scan(&historyCount, &historyUntil, &reason, &historyCreatedAt); err != nil {
		t.Fatal(err)
	}
	if historyCount != 1 || workUntil != "2026-09-22" || historyUntil != workUntil || reason != "blocked" || workUpdatedAt != historyCreatedAt {
		t.Fatalf("snooze state/history mismatch: count=%d state=(%q,%q) history=(%q,%q,%q)", historyCount, workUntil, workUpdatedAt, historyUntil, reason, historyCreatedAt)
	}
}

func TestSQLiteSnoozeRollsBackWhenReasonInsertFails(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-snooze-rollback-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_snoozes (id TEXT PRIMARY KEY, work_item_id TEXT, until_date TEXT, reason TEXT, created_at TEXT)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','30',30,'open','', '2026-09-23', 'planner','2026-09-21T00:00:00Z','2026-09-20T00:00:00Z')`,
		`CREATE TRIGGER fail_snooze_history BEFORE INSERT ON work_item_snoozes BEGIN SELECT RAISE(ABORT, 'injected snooze history failure'); END`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepositoryWithOptions(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)), RepositoryOptions{IDGenerator: func() string { return "snooze-failure" }})
	if err := repo.Snooze(context.Background(), "work-1", "2026-09-24", "blocked"); err == nil {
		t.Fatal("expected injected snooze history failure")
	}
	var until, updatedAt string
	var historyCount int
	if err := db.QueryRow(`SELECT snoozed_until,updated_at FROM work_items WHERE id='work-1'`).Scan(&until, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if until != "2026-09-23" || updatedAt != "2026-09-20T00:00:00Z" {
		t.Fatalf("work item changed after history failure: until=%q updated_at=%q", until, updatedAt)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM work_item_snoozes WHERE work_item_id='work-1'`).Scan(&historyCount); err != nil || historyCount != 0 {
		t.Fatalf("history rows=%d err=%v", historyCount, err)
	}
}

func TestSQLiteSnoozeRejectsCompletedWork(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-snooze-completed-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_snoozes (id TEXT PRIMARY KEY, work_item_id TEXT, until_date TEXT, reason TEXT, created_at TEXT)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','0',30,'complete','2026-09-21T00:00:00Z', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepositoryWithOptions(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)), RepositoryOptions{IDGenerator: func() string { return "snooze-completed" }})
	if err := repo.Snooze(context.Background(), "work-1", "2026-09-22", "blocked"); err == nil {
		t.Fatal("expected completed work snooze rejection")
	}
	var until, updatedAt string
	var historyCount int
	if err := db.QueryRow(`SELECT snoozed_until,updated_at FROM work_items WHERE id='work-1'`).Scan(&until, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if until != "" || updatedAt != "2026-09-21T00:00:00Z" {
		t.Fatalf("completed work changed: until=%q updated_at=%q", until, updatedAt)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM work_item_snoozes`).Scan(&historyCount); err != nil || historyCount != 0 {
		t.Fatalf("history rows=%d err=%v", historyCount, err)
	}
}

func TestSQLiteCompleteLeavesActualUnknownAndAdvancesLinkedGoal(t *testing.T) {
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
	var workCompletedAt string
	var remaining, original, milestoneRevision, goalRevision int
	var milestoneDate, goalDate string
	if err := db.QueryRow(`SELECT completed_at,remaining_minutes,original_estimate_minutes FROM work_items WHERE id='work-1'`).Scan(&workCompletedAt, &remaining, &original); err != nil || workCompletedAt == "" || remaining != 0 || original != 60 {
		t.Fatalf("work completion=%q remaining=%d original=%d err=%v", workCompletedAt, remaining, original, err)
	}
	if err := db.QueryRow(`SELECT completed_date,revision FROM milestones WHERE id='milestone-1'`).Scan(&milestoneDate, &milestoneRevision); err != nil || milestoneDate != "2026-09-21" || milestoneRevision != 2 {
		t.Fatalf("milestone date=%q revision=%d err=%v", milestoneDate, milestoneRevision, err)
	}
	if err := db.QueryRow(`SELECT completed_date,revision FROM goals WHERE id='goal-1'`).Scan(&goalDate, &goalRevision); err != nil || goalDate != "2026-09-21" || goalRevision != 2 {
		t.Fatalf("goal date=%q revision=%d err=%v", goalDate, goalRevision, err)
	}
	var historyRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM work_item_completion_history WHERE work_item_id='work-1'`).Scan(&historyRows); err != nil || historyRows != 0 {
		t.Fatalf("unknown actual produced %d completion-history rows, err=%v", historyRows, err)
	}
	if err := repo.Complete(context.Background(), "work-1"); err != nil {
		t.Fatalf("repeat completion: %v", err)
	}
	var repeatedCompletedAt, repeatedMilestoneDate, repeatedGoalDate string
	var repeatedMilestoneRevision, repeatedGoalRevision int
	if err := db.QueryRow(`SELECT completed_at FROM work_items WHERE id='work-1'`).Scan(&repeatedCompletedAt); err != nil || repeatedCompletedAt != workCompletedAt {
		t.Fatalf("repeated work completed_at=%q want %q err=%v", repeatedCompletedAt, workCompletedAt, err)
	}
	if err := db.QueryRow(`SELECT completed_date,revision FROM milestones WHERE id='milestone-1'`).Scan(&repeatedMilestoneDate, &repeatedMilestoneRevision); err != nil || repeatedMilestoneDate != milestoneDate || repeatedMilestoneRevision != milestoneRevision {
		t.Fatalf("repeated milestone date=%q revision=%d err=%v", repeatedMilestoneDate, repeatedMilestoneRevision, err)
	}
	if err := db.QueryRow(`SELECT completed_date,revision FROM goals WHERE id='goal-1'`).Scan(&repeatedGoalDate, &repeatedGoalRevision); err != nil || repeatedGoalDate != goalDate || repeatedGoalRevision != goalRevision {
		t.Fatalf("repeated goal date=%q revision=%d err=%v", repeatedGoalDate, repeatedGoalRevision, err)
	}
}

func TestSQLiteCompleteRollsBackLinkedCascadeWhenGoalUpdateFails(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-complete-rollback-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE milestones (id TEXT PRIMARY KEY, goal_id TEXT, status TEXT, completed_date TEXT, updated_at INTEGER, revision INTEGER)`,
		`CREATE TABLE milestone_work_links (milestone_id TEXT PRIMARY KEY, work_item_id TEXT)`,
		`CREATE TABLE milestone_prerequisites (milestone_id TEXT, prerequisite_milestone_id TEXT)`,
		`CREATE TABLE goals (id TEXT PRIMARY KEY, progress_method TEXT, status TEXT, progress_basis_points INTEGER, completed_date TEXT, updated_at INTEGER, revision INTEGER)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','30',60,'open','', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T00:00:00Z')`,
		`INSERT INTO goals VALUES ('goal-1','milestones','active',0,'',0,1)`,
		`INSERT INTO milestones VALUES ('milestone-1','goal-1','open','',0,1)`,
		`INSERT INTO milestone_work_links VALUES ('milestone-1','work-1')`,
		`CREATE TRIGGER fail_goal_completion BEFORE UPDATE ON goals BEGIN SELECT RAISE(ABORT, 'injected goal update failure'); END`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)))
	if err := repo.Complete(context.Background(), "work-1"); err == nil {
		t.Fatal("expected injected goal update failure")
	}
	var workStatus, completedAt, milestoneStatus, milestoneDate, goalStatus, goalDate string
	var remaining, milestoneRevision, goalRevision int
	if err := db.QueryRow(`SELECT status,completed_at,remaining_minutes FROM work_items WHERE id='work-1'`).Scan(&workStatus, &completedAt, &remaining); err != nil || workStatus != "open" || completedAt != "" || remaining != 30 {
		t.Fatalf("work persisted after rollback: status=%q completed=%q remaining=%d err=%v", workStatus, completedAt, remaining, err)
	}
	if err := db.QueryRow(`SELECT status,completed_date,revision FROM milestones WHERE id='milestone-1'`).Scan(&milestoneStatus, &milestoneDate, &milestoneRevision); err != nil || milestoneStatus != "open" || milestoneDate != "" || milestoneRevision != 1 {
		t.Fatalf("milestone persisted after rollback: status=%q date=%q revision=%d err=%v", milestoneStatus, milestoneDate, milestoneRevision, err)
	}
	if err := db.QueryRow(`SELECT status,completed_date,revision FROM goals WHERE id='goal-1'`).Scan(&goalStatus, &goalDate, &goalRevision); err != nil || goalStatus != "active" || goalDate != "" || goalRevision != 1 {
		t.Fatalf("goal changed after failure: status=%q date=%q revision=%d err=%v", goalStatus, goalDate, goalRevision, err)
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
	var reason, changedAt string
	if err := db.QueryRow(`SELECT previous_minutes,new_minutes,reason_code,changed_at FROM work_item_estimation_changes WHERE id='estimate-1'`).Scan(&previous, &next, &reason, &changedAt); err != nil || previous != 30 || next != 75 || reason != "scope_changed" || changedAt != "2026-09-21T12:00:00Z" {
		t.Fatalf("previous=%d next=%d reason=%q changed_at=%q err=%v", previous, next, reason, changedAt, err)
	}
	var historyRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM work_item_estimation_changes WHERE work_item_id='work-1'`).Scan(&historyRows); err != nil || historyRows != 1 {
		t.Fatalf("history rows=%d err=%v", historyRows, err)
	}
}

func TestSQLiteUpdateEstimateRollsBackWhenHistoryInsertFails(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-estimate-rollback-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_estimation_changes (id TEXT PRIMARY KEY, work_item_id TEXT, previous_minutes INTEGER, new_minutes INTEGER, reason_code TEXT, changed_at TEXT)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','30',60,'open','', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T00:00:00Z')`,
		`CREATE TRIGGER fail_estimation_history BEFORE INSERT ON work_item_estimation_changes BEGIN SELECT RAISE(ABORT, 'injected history insert failure'); END`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepositoryWithOptions(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)), RepositoryOptions{IDGenerator: func() string { return "estimate-rollback-1" }})
	if err := repo.UpdateEstimate(context.Background(), "work-1", 75, "scope_changed"); err == nil {
		t.Fatal("expected injected estimate history insert failure")
	}
	var remaining, original, historyRows int
	if err := db.QueryRow(`SELECT remaining_minutes,original_estimate_minutes FROM work_items WHERE id='work-1'`).Scan(&remaining, &original); err != nil || remaining != 30 || original != 60 {
		t.Fatalf("remaining=%d original=%d err=%v", remaining, original, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM work_item_estimation_changes WHERE work_item_id='work-1'`).Scan(&historyRows); err != nil || historyRows != 0 {
		t.Fatalf("history rows=%d err=%v", historyRows, err)
	}
}

func TestSQLiteUpdateEstimateRejectsCompletedWork(t *testing.T) {
	db, err := sql.Open("sqlite", "file:work-estimate-completed-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, original_estimate_minutes INTEGER, status TEXT, completed_at TEXT, snoozed_until TEXT NOT NULL DEFAULT '', source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`CREATE TABLE work_item_estimation_changes (id TEXT PRIMARY KEY, work_item_id TEXT, previous_minutes INTEGER, new_minutes INTEGER, reason_code TEXT, changed_at TEXT)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','0',60,'complete','2026-09-21T12:00:00Z', '', 'planner','2026-09-21T00:00:00Z','2026-09-21T12:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)))
	if err := repo.UpdateEstimate(context.Background(), "work-1", 75, "scope_changed"); err == nil {
		t.Fatal("expected completed work estimate rejection")
	}
	var remaining, historyRows int
	if err := db.QueryRow(`SELECT remaining_minutes FROM work_items WHERE id='work-1'`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("remaining=%d err=%v", remaining, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM work_item_estimation_changes`).Scan(&historyRows); err != nil || historyRows != 0 {
		t.Fatalf("history rows=%d err=%v", historyRows, err)
	}
}

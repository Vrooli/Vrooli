package calendar

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestSQLiteListTodayUsesWorkspaceCapacityAndReserve(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-capacity-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT, created_at TEXT)`,
		`INSERT INTO planning_profiles VALUES ('default',420,60,'UTC')`,
		`INSERT INTO work_items VALUES ('work-1','Draft','manual')`,
		`INSERT INTO calendar_allocations VALUES ('allocation-1','work-1','2026-09-19',600,45,'accepted','2026-09-19T09:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	today, err := NewSQLiteRepository(db, schedule.System()).ListToday(context.Background(), "2026-09-19")
	if err != nil {
		t.Fatal(err)
	}
	if today.AvailableMinutes != 360 || today.PlannedMinutes != 45 || today.BreathingRoomMinutes != 315 {
		t.Fatalf("today=%#v", today)
	}
}

func TestSQLiteRoutineDemandSubtractsFromBreathingRoomAndPlacement(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-routine-demand-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT, created_at TEXT)`,
		`CREATE TABLE routines (id TEXT PRIMARY KEY, title TEXT, kind TEXT, timezone TEXT, start_date TEXT, end_date TEXT, weekdays TEXT, start_minute INTEGER, duration_minutes INTEGER, frequency_per_week INTEGER, revision INTEGER, active INTEGER, created_at TEXT)`,
		`CREATE TABLE routine_occurrence_overrides (routine_id TEXT, local_date TEXT, status TEXT, revision INTEGER, updated_at TEXT, PRIMARY KEY (routine_id, local_date))`,
		`CREATE TABLE routine_occurrence_reschedules (routine_id TEXT, local_date TEXT, start_minute INTEGER, revision INTEGER, updated_at TEXT, PRIMARY KEY (routine_id, local_date))`,
		`INSERT INTO planning_profiles VALUES ('default',480,60,'UTC')`,
		`INSERT INTO work_items VALUES ('work-1','Draft','manual')`,
		`INSERT INTO routines VALUES ('routine-1','Morning review','fixed','UTC','2026-09-21','','1',600,60,1,1,1,'2026-09-20T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepository(db, schedule.System())
	today, err := repo.ListToday(context.Background(), "2026-09-21")
	if err != nil {
		t.Fatal(err)
	}
	if today.AvailableMinutes != 420 || today.PlannedMinutes != 0 || today.BreathingRoomMinutes != 360 {
		t.Fatalf("today=%#v", today)
	}
	proposal, err := repo.Preview(context.Background(), PreviewInput{WorkItemID: "work-1", LocalDate: "2026-09-21", StartMinutes: 600, DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.State != "feasible" || proposal.StartMinutes != 660 {
		t.Fatalf("proposal=%#v", proposal)
	}
}

func TestSQLiteListTodaySubtractsExternalBusyUnionWithoutDoubleCountingOverlap(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-external-busy-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT, created_at TEXT)`,
		`CREATE TABLE provider_connections (id TEXT PRIMARY KEY, status TEXT)`,
		`CREATE TABLE imported_events (id TEXT, connection_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, busy INTEGER, status TEXT)`,
		`INSERT INTO planning_profiles VALUES ('default',480,60,'UTC')`,
		`INSERT INTO work_items VALUES ('work-1','Draft','manual')`,
		`INSERT INTO calendar_allocations VALUES ('allocation-1','work-1','2026-09-19',600,45,'accepted','2026-09-19T09:00:00Z')`,
		`INSERT INTO provider_connections VALUES ('provider-1','synced')`,
		`INSERT INTO imported_events VALUES ('event-1','provider-1','2026-09-19',630,60,1,'active')`,
		`INSERT INTO imported_events VALUES ('event-2','provider-1','2026-09-19',780,30,1,'active')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	today, err := NewSQLiteRepository(db, schedule.System()).ListToday(context.Background(), "2026-09-19")
	if err != nil {
		t.Fatal(err)
	}
	if today.ExternalBusyMinutes != 90 || today.ExternalEventCount != 2 || today.BreathingRoomMinutes != 300 {
		t.Fatalf("today=%#v", today)
	}
}

func TestSQLitePreviewMovesPastAcceptedAndImportedBusyIntervals(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-preview-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT, created_at TEXT)`,
		`CREATE TABLE provider_connections (id TEXT PRIMARY KEY, status TEXT)`,
		`CREATE TABLE imported_events (id TEXT, connection_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, busy INTEGER, status TEXT)`,
		`INSERT INTO planning_profiles VALUES ('default',480,60,'UTC')`,
		`INSERT INTO work_items VALUES ('work-1','Draft','manual')`,
		`INSERT INTO work_items VALUES ('work-2','Review','manual')`,
		`INSERT INTO calendar_allocations VALUES ('allocation-1','work-2','2026-09-19',600,45,'accepted','2026-09-19T09:00:00Z')`,
		`INSERT INTO provider_connections VALUES ('provider-1','synced')`,
		`INSERT INTO imported_events VALUES ('event-1','provider-1','2026-09-19',645,30,1,'active')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	proposal, err := NewSQLiteRepository(db, schedule.System()).Preview(context.Background(), PreviewInput{WorkItemID: "work-1", LocalDate: "2026-09-19", StartMinutes: 600, DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.State != "feasible" || proposal.StartMinutes != 675 || proposal.Reason == "" {
		t.Fatalf("proposal=%#v", proposal)
	}
}

func TestSQLiteApplyProposalIsRevisionCheckedAndIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-apply-proposal-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO work_items VALUES ('work-1','Draft','manual'),('work-2','Review','manual')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	proposal, err := repo.Preview(context.Background(), PreviewInput{WorkItemID: "work-1", LocalDate: "2026-09-19", StartMinutes: 600, DurationMinutes: 30})
	if err != nil || proposal.ID == "" || proposal.BaseRevision != 1 {
		t.Fatalf("proposal=%#v err=%v", proposal, err)
	}
	allocation, err := repo.ApplyProposal(context.Background(), ApplyProposalInput{ProposalID: proposal.ID, ExpectedRevision: proposal.BaseRevision, IdempotencyKey: "apply-1"})
	if err != nil {
		t.Fatal(err)
	}
	if allocation.State != "accepted" || allocation.WorkItemID != "work-1" {
		t.Fatalf("allocation=%#v", allocation)
	}
	retry, err := repo.ApplyProposal(context.Background(), ApplyProposalInput{ProposalID: proposal.ID, ExpectedRevision: proposal.BaseRevision, IdempotencyKey: "apply-1"})
	if err != nil || retry.ID != allocation.ID {
		t.Fatalf("retry=%#v err=%v", retry, err)
	}
	other, err := repo.Preview(context.Background(), PreviewInput{WorkItemID: "work-2", LocalDate: "2026-09-19", StartMinutes: 700, DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyProposal(context.Background(), ApplyProposalInput{ProposalID: other.ID, ExpectedRevision: 1, IdempotencyKey: "apply-2"}); err == nil {
		t.Fatal("expected stale schedule revision")
	}
}

func TestSQLiteScheduleProposalPlacesSelectedWorkInStableOrder(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-schedule-proposal-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, description TEXT, remaining_minutes INTEGER, source_label TEXT, created_at TEXT, updated_at TEXT)`,
		`INSERT INTO planning_profiles VALUES ('default',480,60,'UTC')`,
		`INSERT INTO work_items VALUES ('work-1','Draft','','45','manual','2026-09-19T00:00:00Z','2026-09-19T00:00:00Z')`,
		`INSERT INTO work_items VALUES ('work-2','Review','','30','manual','2026-09-19T00:00:00Z','2026-09-19T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepository(db, schedule.System())
	proposal, err := repo.PreviewSchedule(context.Background(), SchedulePreviewInput{LocalDate: "2026-09-19", StartMinutes: 540, WorkItemIDs: []string{"work-1", "work-2"}})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.State != "feasible" || len(proposal.Placements) != 2 || proposal.Placements[0].StartMinutes != 540 || proposal.Placements[1].StartMinutes != 585 {
		t.Fatalf("proposal=%#v", proposal)
	}
	allocations, err := repo.ApplyScheduleProposal(context.Background(), ApplyScheduleProposalInput{ProposalID: proposal.ID, ExpectedRevision: proposal.BaseRevision, IdempotencyKey: "schedule-apply-1"})
	if err != nil || len(allocations) != 2 {
		t.Fatalf("allocations=%#v err=%v", allocations, err)
	}
	retry, err := repo.ApplyScheduleProposal(context.Background(), ApplyScheduleProposalInput{ProposalID: proposal.ID, ExpectedRevision: proposal.BaseRevision, IdempotencyKey: "schedule-apply-1"})
	if err != nil || len(retry) != 2 || retry[0].ID != allocations[0].ID {
		t.Fatalf("retry=%#v err=%v", retry, err)
	}
	remaining, err := repo.PreviewSchedule(context.Background(), SchedulePreviewInput{LocalDate: "2026-09-19", StartMinutes: 720, WorkItemIDs: []string{"work-1", "work-2"}})
	if err != nil {
		t.Fatal(err)
	}
	if remaining.State != "blocked" || len(remaining.Placements) != 2 || remaining.Placements[0].State != "unresolved" || remaining.Placements[1].State != "unresolved" {
		t.Fatalf("expected demand-conserving blocked preview, got %#v", remaining)
	}
}

func TestSQLiteListTodayUsesConfiguredAvailabilityAndProtectedException(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-availability-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE planning_profiles (id TEXT PRIMARY KEY, daily_capacity_minutes INTEGER, reserve_minutes INTEGER, timezone TEXT)`,
		`CREATE TABLE availability_rules (id TEXT PRIMARY KEY, weekday INTEGER, start_minute INTEGER, end_minute INTEGER, timezone TEXT, effective_start_date TEXT, effective_end_date TEXT, priority INTEGER, revision INTEGER)`,
		`CREATE TABLE availability_exceptions (id TEXT PRIMARY KEY, date TEXT, start_minute INTEGER, end_minute INTEGER, kind TEXT, reason TEXT, revision INTEGER)`,
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT, created_at TEXT)`,
		`INSERT INTO planning_profiles VALUES ('default',600,60,'America/New_York')`,
		`INSERT INTO availability_rules VALUES ('monday-morning',1,540,1020,'America/New_York','','',0,1)`,
		`INSERT INTO availability_exceptions VALUES ('pickup','2026-09-21',720,780,'protected','school pickup',1)`,
		`INSERT INTO availability_exceptions VALUES ('pickup-overlap','2026-09-21',750,810,'protected','overlapping transition',1)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	today, err := NewSQLiteRepository(db, schedule.System()).ListToday(context.Background(), "2026-09-21")
	if err != nil {
		t.Fatal(err)
	}
	if today.AvailableMinutes != 330 {
		t.Fatalf("today=%#v", today)
	}
}

func TestIntervalUnionCountsOverlappingExceptionsOnce(t *testing.T) {
	if got := intervalUnionMinutes([]capacityInterval{{start: 10, end: 30}, {start: 20, end: 40}, {start: 50, end: 60}}); got != 40 {
		t.Fatalf("union=%d", got)
	}
}

func TestSQLiteRoutinesExpandFixedAndFlexibleOccurrences(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-routines-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE routines (id TEXT PRIMARY KEY, title TEXT, kind TEXT, timezone TEXT, start_date TEXT, end_date TEXT, weekdays TEXT, start_minute INTEGER, duration_minutes INTEGER, frequency_per_week INTEGER, revision INTEGER, active INTEGER, created_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE routine_occurrence_overrides (routine_id TEXT, local_date TEXT, status TEXT, revision INTEGER, updated_at TEXT, PRIMARY KEY (routine_id, local_date))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE routine_occurrence_reschedules (routine_id TEXT, local_date TEXT, start_minute INTEGER, revision INTEGER, updated_at TEXT, PRIMARY KEY (routine_id, local_date))`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO routines VALUES ('fixed','Review','fixed','UTC','2026-09-21','','1,3,5',540,45,3,1,1,'2026-09-20T00:00:00Z')`,
		`INSERT INTO routines VALUES ('flex','Runs','flexible','UTC','2026-09-21','','1,3,5',600,30,2,1,1,'2026-09-20T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	items, err := NewSQLiteRepository(db, schedule.System()).ListRoutineOccurrences(context.Background(), "2026-09-21", "2026-09-27")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("occurrences=%#v", items)
	}
	if items[0].LocalDate != "2026-09-21" || items[0].Kind != "fixed" {
		t.Fatalf("occurrences=%#v", items)
	}
	if err := NewSQLiteRepository(db, schedule.System()).SkipRoutineOccurrence(context.Background(), "fixed", "2026-09-23", 1); err != nil {
		t.Fatal(err)
	}
	items, err = NewSQLiteRepository(db, schedule.System()).ListRoutineOccurrences(context.Background(), "2026-09-21", "2026-09-27")
	if err != nil || len(items) != 4 {
		t.Fatalf("skipped occurrences=%#v err=%v", items, err)
	}
	if err := NewSQLiteRepository(db, schedule.System()).RescheduleRoutineOccurrence(context.Background(), "fixed", "2026-09-21", 600, 2); err != nil {
		t.Fatalf("reschedule occurrence: %v", err)
	}
	items, err = NewSQLiteRepository(db, schedule.System()).ListRoutineOccurrences(context.Background(), "2026-09-21", "2026-09-27")
	if err != nil || len(items) != 4 || items[0].StartMinute != 600 {
		t.Fatalf("rescheduled occurrences=%#v err=%v", items, err)
	}
}

func TestSQLiteCarryForwardPreservesHistoryAndIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file:calendar-carry-forward-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE work_items (id TEXT PRIMARY KEY, title TEXT, source_label TEXT)`,
		`CREATE TABLE calendar_allocations (id TEXT PRIMARY KEY, work_item_id TEXT, local_date TEXT, start_minutes INTEGER, duration_minutes INTEGER, state TEXT, created_at TEXT)`,
		`CREATE TABLE allocation_carry_forwards (source_allocation_id TEXT PRIMARY KEY, carried_allocation_id TEXT NOT NULL UNIQUE, target_local_date TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`INSERT INTO work_items VALUES ('work-1','Draft','manual')`,
		`INSERT INTO calendar_allocations VALUES ('allocation-1','work-1','2026-09-19',600,45,'accepted','2026-09-19T09:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSQLiteRepository(db, schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)))
	first, err := repo.CarryForward(context.Background(), CarryForwardInput{AllocationID: "allocation-1", TargetLocalDate: "2026-09-22", StartMinutes: 660})
	if err != nil {
		t.Fatal(err)
	}
	if first.CarriedFromID != "allocation-1" || first.LocalDate != "2026-09-22" || first.DurationMinutes != 45 {
		t.Fatalf("unexpected carried allocation: %#v", first)
	}
	var sourceState string
	if err := db.QueryRow(`SELECT state FROM calendar_allocations WHERE id='allocation-1'`).Scan(&sourceState); err != nil || sourceState != "carried_forward" {
		t.Fatalf("source state=%q err=%v", sourceState, err)
	}
	second, err := repo.CarryForward(context.Background(), CarryForwardInput{AllocationID: "allocation-1", TargetLocalDate: "2026-09-22", StartMinutes: 660})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.CarriedFromID != first.CarriedFromID {
		t.Fatalf("retry created a different allocation: first=%#v second=%#v", first, second)
	}
}

package calendar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
	id    func() string
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock, id: uuid.NewString}
}

func (r *sqliteRepository) WorkItem(ctx context.Context, id string) (Allocation, error) {
	var a Allocation
	err := r.db.QueryRowContext(ctx, `SELECT id,title,source_label FROM work_items WHERE id=?`, id).Scan(&a.WorkItemID, &a.Title, &a.SourceLabel)
	if errors.Is(err, sql.ErrNoRows) {
		return Allocation{}, ErrWorkItemNotFound{id}
	}
	return a, err
}

func (r *sqliteRepository) ListToday(ctx context.Context, date string) (Today, error) {
	rangeResult, err := r.ListRange(ctx, date, date)
	if err != nil {
		return Today{}, err
	}
	capacity, reserve, err := r.profileCapacity(ctx, date)
	if err != nil {
		return Today{}, err
	}
	external, err := r.externalBusy(ctx, date)
	if err != nil {
		return Today{}, err
	}
	routineIntervals, err := r.routineBusy(ctx, date)
	if err != nil {
		return Today{}, err
	}
	today := Today{AvailableMinutes: capacity - reserve, Allocations: rangeResult.Allocations, ExternalBusyMinutes: external.minutes, ExternalEventCount: external.events, ExternalFreshness: external.freshness}
	if today.AvailableMinutes < 0 {
		today.AvailableMinutes = 0
	}
	occupied := make([]capacityInterval, 0, len(today.Allocations)+len(external.intervals)+len(routineIntervals))
	for _, a := range today.Allocations {
		today.PlannedMinutes += a.DurationMinutes
		occupied = append(occupied, capacityInterval{start: a.StartMinutes, end: a.StartMinutes + a.DurationMinutes})
	}
	occupied = append(occupied, external.intervals...)
	occupied = append(occupied, routineIntervals...)
	today.BreathingRoomMinutes = today.AvailableMinutes - intervalUnionMinutes(occupied)
	if today.BreathingRoomMinutes < 0 {
		today.BreathingRoomMinutes = 0
	}
	return today, nil
}

func (r *sqliteRepository) Preview(ctx context.Context, in PreviewInput) (PlacementProposal, error) {
	today, err := r.ListToday(ctx, in.LocalDate)
	if err != nil {
		return PlacementProposal{}, err
	}
	routineIntervals, routineErr := r.routineBusy(ctx, in.LocalDate)
	if routineErr != nil {
		return PlacementProposal{}, routineErr
	}
	occupied := make([]capacityInterval, 0, len(today.Allocations)+len(routineIntervals)+4)
	for _, allocation := range today.Allocations {
		occupied = append(occupied, capacityInterval{start: allocation.StartMinutes, end: allocation.StartMinutes + allocation.DurationMinutes})
	}
	if external, externalErr := r.externalBusy(ctx, in.LocalDate); externalErr == nil {
		occupied = append(occupied, external.intervals...)
	}
	occupied = append(occupied, routineIntervals...)
	requested := in.StartMinutes
	proposal := PlacementProposal{ID: r.id(), WorkItemID: in.WorkItemID, LocalDate: in.LocalDate, StartMinutes: requested, DurationMinutes: in.DurationMinutes, BaseRevision: r.scheduleRevision(ctx, r.db)}
	if remaining, known, demandErr := r.remainingSchedulableMinutes(ctx, in.WorkItemID); demandErr != nil {
		return PlacementProposal{}, demandErr
	} else if known && in.DurationMinutes > remaining {
		proposal.State, proposal.Reason = "blocked", fmt.Sprintf("The requested %d-minute session exceeds the %d minutes of remaining effort.", in.DurationMinutes, remaining)
		return r.persistProposal(ctx, proposal)
	}
	for candidate := requested; candidate+in.DurationMinutes <= 1440; candidate += 15 {
		if !overlapsAny(occupied, candidate, candidate+in.DurationMinutes) {
			reason := "Requested time is available."
			if candidate != requested {
				reason = fmt.Sprintf("Moved to the next open 15-minute slot after the requested time (%s).", clockLabel(requested))
			}
			proposal.State, proposal.StartMinutes, proposal.Reason = "feasible", candidate, reason
			return r.persistProposal(ctx, proposal)
		}
	}
	proposal.State, proposal.Reason = "blocked", "No contiguous open slot remains after the requested time."
	return r.persistProposal(ctx, proposal)
}

func (r *sqliteRepository) persistProposal(ctx context.Context, proposal PlacementProposal) (PlacementProposal, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO placement_proposals (id,work_item_id,local_date,start_minutes,duration_minutes,state,reason,base_revision,created_at) VALUES (?,?,?,?,?,?,?,?,?)`, proposal.ID, proposal.WorkItemID, proposal.LocalDate, proposal.StartMinutes, proposal.DurationMinutes, proposal.State, proposal.Reason, proposal.BaseRevision, r.clock.Now().UTC().Format(time.RFC3339Nano))
	if err != nil && !strings.Contains(err.Error(), "no such table") {
		return PlacementProposal{}, fmt.Errorf("persist placement proposal: %w", err)
	}
	return proposal, nil
}

func (r *sqliteRepository) scheduleRevision(ctx context.Context, db SQLExecutor) int64 {
	var revision int64
	if err := db.QueryRowContext(ctx, `SELECT revision FROM calendar_schedule_state WHERE id='default'`).Scan(&revision); err != nil || revision < 1 {
		return 1
	}
	return revision
}

func bumpScheduleRevision(ctx context.Context, db SQLExecutor) error {
	_, err := db.ExecContext(ctx, `UPDATE calendar_schedule_state SET revision=revision+1 WHERE id='default'`)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

func (r *sqliteRepository) ApplyProposal(ctx context.Context, in ApplyProposalInput) (Allocation, error) {
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return Allocation{}, fmt.Errorf("calendar storage does not support transactional proposal application")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return Allocation{}, fmt.Errorf("begin apply proposal: %w", err)
	}
	fail := func(cause error) (Allocation, error) { _ = tx.Rollback(); return Allocation{}, cause }
	var proposal PlacementProposal
	var appliedID, appliedKey string
	err = tx.QueryRowContext(ctx, `SELECT id,work_item_id,local_date,start_minutes,duration_minutes,state,reason,base_revision,applied_allocation_id,applied_idempotency_key FROM placement_proposals WHERE id=?`, in.ProposalID).Scan(&proposal.ID, &proposal.WorkItemID, &proposal.LocalDate, &proposal.StartMinutes, &proposal.DurationMinutes, &proposal.State, &proposal.Reason, &proposal.BaseRevision, &appliedID, &appliedKey)
	if errors.Is(err, sql.ErrNoRows) {
		return fail(ErrProposalNotFound{in.ProposalID})
	}
	if err != nil {
		return fail(fmt.Errorf("read placement proposal: %w", err))
	}
	if appliedID != "" {
		if appliedKey == in.IdempotencyKey {
			allocation, loadErr := loadAllocation(ctx, tx, appliedID)
			if loadErr != nil {
				return fail(loadErr)
			}
			if commitErr := tx.Commit(); commitErr != nil {
				return Allocation{}, fmt.Errorf("commit idempotent proposal: %w", commitErr)
			}
			return allocation, nil
		}
		return fail(ErrProposalAlreadyApplied{in.ProposalID})
	}
	current := r.scheduleRevision(ctx, tx)
	if current != in.ExpectedRevision {
		return fail(ErrScheduleRevisionConflict{Expected: in.ExpectedRevision, Current: current})
	}
	if proposal.State != "feasible" {
		return fail(ErrProposalNotFeasible{in.ProposalID})
	}
	var item Allocation
	if err := tx.QueryRowContext(ctx, `SELECT id,title,source_label FROM work_items WHERE id=?`, proposal.WorkItemID).Scan(&item.WorkItemID, &item.Title, &item.SourceLabel); errors.Is(err, sql.ErrNoRows) {
		return fail(ErrWorkItemNotFound{proposal.WorkItemID})
	} else if err != nil {
		return fail(err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM calendar_allocations WHERE local_date=? AND state='accepted' AND start_minutes < ? AND start_minutes + duration_minutes > ?`, proposal.LocalDate, proposal.StartMinutes+proposal.DurationMinutes, proposal.StartMinutes).Scan(&count); err != nil {
		return fail(err)
	}
	if count > 0 {
		return fail(ErrAllocationConflict{})
	}
	busy, err := importedOverlap(ctx, tx, proposal.LocalDate, proposal.StartMinutes, proposal.DurationMinutes)
	if err != nil {
		return fail(err)
	}
	if busy {
		return fail(ErrAllocationConflict{})
	}
	if routineIntervals, routineErr := r.routineBusy(ctx, proposal.LocalDate); routineErr != nil {
		return fail(routineErr)
	} else if overlapsAny(routineIntervals, proposal.StartMinutes, proposal.StartMinutes+proposal.DurationMinutes) {
		return fail(ErrAllocationConflict{})
	}
	if remaining, known, demandErr := r.remainingSchedulableMinutesOn(ctx, tx, proposal.WorkItemID); demandErr != nil {
		return fail(demandErr)
	} else if known && proposal.DurationMinutes > remaining {
		return fail(ErrDemandExceeded{proposal.WorkItemID})
	}
	item.ID, item.LocalDate, item.StartMinutes, item.DurationMinutes, item.State, item.CreatedAt = r.id(), proposal.LocalDate, proposal.StartMinutes, proposal.DurationMinutes, "accepted", r.clock.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO calendar_allocations (id,work_item_id,local_date,start_minutes,duration_minutes,state,created_at) VALUES (?,?,?,?,?,?,?)`, item.ID, item.WorkItemID, item.LocalDate, item.StartMinutes, item.DurationMinutes, item.State, item.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return fail(fmt.Errorf("insert applied allocation: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `UPDATE placement_proposals SET applied_allocation_id=?,applied_idempotency_key=? WHERE id=? AND applied_allocation_id=''`, item.ID, in.IdempotencyKey, proposal.ID); err != nil {
		return fail(err)
	}
	if err := bumpScheduleRevision(ctx, tx); err != nil {
		return fail(err)
	}
	if err := tx.Commit(); err != nil {
		return Allocation{}, fmt.Errorf("commit proposal application: %w", err)
	}
	return item, nil
}

func (r *sqliteRepository) PreviewSchedule(ctx context.Context, in SchedulePreviewInput) (ScheduleProposal, error) {
	today, err := r.ListToday(ctx, in.LocalDate)
	if err != nil {
		return ScheduleProposal{}, err
	}
	routineIntervals, routineErr := r.routineBusy(ctx, in.LocalDate)
	if routineErr != nil {
		return ScheduleProposal{}, routineErr
	}
	occupied := make([]capacityInterval, 0, len(today.Allocations)+len(routineIntervals)+8)
	for _, allocation := range today.Allocations {
		occupied = append(occupied, capacityInterval{start: allocation.StartMinutes, end: allocation.StartMinutes + allocation.DurationMinutes})
	}
	if external, externalErr := r.externalBusy(ctx, in.LocalDate); externalErr == nil {
		occupied = append(occupied, external.intervals...)
	}
	occupied = append(occupied, routineIntervals...)
	proposal := ScheduleProposal{ID: r.id(), LocalDate: in.LocalDate, StartMinutes: in.StartMinutes, BaseRevision: r.scheduleRevision(ctx, r.db), Placements: []ProposedPlacement{}}
	remainingBudget := today.BreathingRoomMinutes
	for _, id := range in.WorkItemIDs {
		item, itemErr := r.WorkItem(ctx, id)
		if itemErr != nil {
			return ScheduleProposal{}, itemErr
		}
		placement := ProposedPlacement{WorkItemID: id, Title: item.Title, LocalDate: in.LocalDate, State: "unresolved"}
		remainingMinutes, demandKnown, remainingErr := r.remainingSchedulableMinutes(ctx, id)
		if remainingErr != nil {
			return ScheduleProposal{}, remainingErr
		}
		if !demandKnown {
			placement.Reason = "Remaining effort is unknown; no placement was invented."
			proposal.Placements = append(proposal.Placements, placement)
			continue
		}
		if remainingMinutes <= 0 {
			placement.Reason = "Remaining effort is unknown or already complete; no session was invented."
			proposal.Placements = append(proposal.Placements, placement)
			continue
		}
		duration := remainingMinutes
		if duration > 90 {
			duration = 90
		}
		if duration > remainingBudget {
			placement.Reason = fmt.Sprintf("The next %d-minute session exceeds the remaining breathing room.", duration)
			proposal.Placements = append(proposal.Placements, placement)
			continue
		}
		found := false
		for candidate := in.StartMinutes; candidate+duration <= 1440; candidate += 15 {
			if !overlapsAny(occupied, candidate, candidate+duration) {
				placement.State, placement.StartMinutes, placement.DurationMinutes = "feasible", candidate, duration
				placement.Reason = "First feasible session in stable request order."
				occupied = append(occupied, capacityInterval{start: candidate, end: candidate + duration})
				remainingBudget -= duration
				found = true
				break
			}
		}
		if !found {
			placement.Reason = "No contiguous open slot remains after the requested time."
		}
		proposal.Placements = append(proposal.Placements, placement)
	}
	feasible := 0
	for _, placement := range proposal.Placements {
		if placement.State == "feasible" {
			feasible++
		}
	}
	if feasible == 0 {
		proposal.State, proposal.Reason = "blocked", "No selected work item has a feasible session in the requested scope."
	} else if feasible < len(proposal.Placements) {
		proposal.State, proposal.Reason = "partial", fmt.Sprintf("Placed %d of %d selected work items; unresolved items remain visible.", feasible, len(proposal.Placements))
	} else {
		proposal.State, proposal.Reason = "feasible", fmt.Sprintf("Placed all %d selected work items in stable request order.", feasible)
	}
	return r.persistScheduleProposal(ctx, proposal)
}

func (r *sqliteRepository) remainingSchedulableMinutes(ctx context.Context, workItemID string) (int, bool, error) {
	return r.remainingSchedulableMinutesOn(ctx, r.db, workItemID)
}

func (r *sqliteRepository) remainingSchedulableMinutesOn(ctx context.Context, db SQLExecutor, workItemID string) (int, bool, error) {
	var remaining int
	err := db.QueryRowContext(ctx, `SELECT remaining_minutes FROM work_items WHERE id=?`, workItemID).Scan(&remaining)
	if err != nil && strings.Contains(err.Error(), "no such column") {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	var scheduled int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(duration_minutes),0) FROM calendar_allocations WHERE work_item_id=? AND state='accepted'`, workItemID).Scan(&scheduled); err != nil {
		return 0, false, err
	}
	remaining -= scheduled
	if remaining < 0 {
		remaining = 0
	}
	return remaining, true, nil
}

func (r *sqliteRepository) persistScheduleProposal(ctx context.Context, proposal ScheduleProposal) (ScheduleProposal, error) {
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return ScheduleProposal{}, fmt.Errorf("calendar storage does not support transactional schedule proposals")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return ScheduleProposal{}, err
	}
	rollback := func(cause error) (ScheduleProposal, error) { _ = tx.Rollback(); return ScheduleProposal{}, cause }
	if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_proposals (id,local_date,start_minutes,state,reason,base_revision,created_at) VALUES (?,?,?,?,?,?,?)`, proposal.ID, proposal.LocalDate, proposal.StartMinutes, proposal.State, proposal.Reason, proposal.BaseRevision, r.clock.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return rollback(err)
	}
	for _, placement := range proposal.Placements {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_proposal_placements (id,proposal_id,work_item_id,title,local_date,start_minutes,duration_minutes,state,reason) VALUES (?,?,?,?,?,?,?,?,?)`, r.id(), proposal.ID, placement.WorkItemID, placement.Title, placement.LocalDate, placement.StartMinutes, placement.DurationMinutes, placement.State, placement.Reason); err != nil {
			return rollback(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return ScheduleProposal{}, err
	}
	return proposal, nil
}

func (r *sqliteRepository) ApplyScheduleProposal(ctx context.Context, in ApplyScheduleProposalInput) ([]Allocation, error) {
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return nil, fmt.Errorf("calendar storage does not support transactional schedule proposal application")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	fail := func(cause error) ([]Allocation, error) { _ = tx.Rollback(); return nil, cause }
	var proposalID, state, appliedKey string
	var baseRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT id,state,base_revision,applied_idempotency_key FROM schedule_proposals WHERE id=?`, in.ProposalID).Scan(&proposalID, &state, &baseRevision, &appliedKey); errors.Is(err, sql.ErrNoRows) {
		return fail(ErrProposalNotFound{in.ProposalID})
	} else if err != nil {
		return fail(err)
	}
	if appliedKey != "" {
		if appliedKey != in.IdempotencyKey {
			return fail(ErrProposalAlreadyApplied{in.ProposalID})
		}
		allocations, loadErr := loadScheduleProposalAllocations(ctx, tx, in.ProposalID)
		if loadErr != nil {
			return fail(loadErr)
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return allocations, nil
	}
	current := r.scheduleRevision(ctx, tx)
	if current != in.ExpectedRevision {
		return fail(ErrScheduleRevisionConflict{Expected: in.ExpectedRevision, Current: current})
	}
	if state == "blocked" {
		return fail(ErrProposalNotFeasible{in.ProposalID})
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,work_item_id,title,local_date,start_minutes,duration_minutes,state FROM schedule_proposal_placements WHERE proposal_id=? AND state='feasible' ORDER BY id`, in.ProposalID)
	if err != nil {
		return fail(err)
	}
	defer rows.Close()
	allocations := []Allocation{}
	for rows.Next() {
		var placementID, workItemID, title, date, placementState string
		var start, duration int
		if err := rows.Scan(&placementID, &workItemID, &title, &date, &start, &duration, &placementState); err != nil {
			return fail(err)
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM calendar_allocations WHERE local_date=? AND state='accepted' AND start_minutes < ? AND start_minutes + duration_minutes > ?`, date, start+duration, start).Scan(&count); err != nil {
			return fail(err)
		}
		if count > 0 {
			return fail(ErrAllocationConflict{})
		}
		busy, busyErr := importedOverlap(ctx, tx, date, start, duration)
		if busyErr != nil {
			return fail(busyErr)
		}
		if busy {
			return fail(ErrAllocationConflict{})
		}
		routineIntervals, routineErr := r.routineBusy(ctx, date)
		if routineErr != nil {
			return fail(routineErr)
		}
		if overlapsAny(routineIntervals, start, start+duration) {
			return fail(ErrAllocationConflict{})
		}
		if remaining, known, demandErr := r.remainingSchedulableMinutesOn(ctx, tx, workItemID); demandErr != nil {
			return fail(demandErr)
		} else if known && duration > remaining {
			return fail(ErrDemandExceeded{workItemID})
		}
		allocation := Allocation{ID: r.id(), WorkItemID: workItemID, Title: title, LocalDate: date, StartMinutes: start, DurationMinutes: duration, State: "accepted", CreatedAt: r.clock.Now().UTC()}
		if _, err := tx.ExecContext(ctx, `INSERT INTO calendar_allocations (id,work_item_id,local_date,start_minutes,duration_minutes,state,created_at) VALUES (?,?,?,?,?,?,?)`, allocation.ID, allocation.WorkItemID, allocation.LocalDate, allocation.StartMinutes, allocation.DurationMinutes, allocation.State, allocation.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return fail(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE schedule_proposal_placements SET applied_allocation_id=? WHERE id=? AND applied_allocation_id=''`, allocation.ID, placementID); err != nil {
			return fail(err)
		}
		allocations = append(allocations, allocation)
	}
	if err := rows.Err(); err != nil {
		return fail(err)
	}
	if len(allocations) == 0 {
		return fail(ErrProposalNotFeasible{in.ProposalID})
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedule_proposals SET applied_idempotency_key=? WHERE id=? AND applied_idempotency_key=''`, in.IdempotencyKey, proposalID); err != nil {
		return fail(err)
	}
	if err := bumpScheduleRevision(ctx, tx); err != nil {
		return fail(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return allocations, nil
}

func loadScheduleProposalAllocations(ctx context.Context, db SQLExecutor, proposalID string) ([]Allocation, error) {
	rows, err := db.QueryContext(ctx, `SELECT a.id,a.work_item_id,w.title,w.source_label,a.local_date,a.start_minutes,a.duration_minutes,a.state,a.created_at FROM schedule_proposal_placements p JOIN calendar_allocations a ON a.id=p.applied_allocation_id JOIN work_items w ON w.id=a.work_item_id WHERE p.proposal_id=? AND p.applied_allocation_id<>'' ORDER BY p.id`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Allocation
	for rows.Next() {
		var allocation Allocation
		var created string
		if err := rows.Scan(&allocation.ID, &allocation.WorkItemID, &allocation.Title, &allocation.SourceLabel, &allocation.LocalDate, &allocation.StartMinutes, &allocation.DurationMinutes, &allocation.State, &created); err != nil {
			return nil, err
		}
		allocation.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		result = append(result, allocation)
	}
	return result, rows.Err()
}

func overlapsAny(items []capacityInterval, start, end int) bool {
	for _, item := range items {
		if start < item.end && end > item.start {
			return true
		}
	}
	return false
}

func clockLabel(minutes int) string { return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60) }

type externalBusyResult struct {
	intervals []capacityInterval
	minutes   int
	events    int
	freshness string
}

// routineBusy turns generated local routine demand into capacity intervals.
// Routines remain distinct from accepted work allocations, but they still
// consume the time in which a proposal may be placed. Older focused fixtures
// predate the routine tables, so their absence is treated as no routine demand.
func (r *sqliteRepository) routineBusy(ctx context.Context, date string) ([]capacityInterval, error) {
	occurrences, err := r.ListRoutineOccurrences(ctx, date, date)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	intervals := make([]capacityInterval, 0, len(occurrences))
	for _, occurrence := range occurrences {
		intervals = append(intervals, capacityInterval{
			start: occurrence.StartMinute,
			end:   occurrence.StartMinute + occurrence.DurationMinutes,
		})
	}
	return intervals, nil
}

func (r *sqliteRepository) externalBusy(ctx context.Context, date string) (externalBusyResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT e.start_minutes,e.duration_minutes FROM imported_events e JOIN provider_connections c ON c.id=e.connection_id WHERE e.local_date=? AND e.status='active' AND e.busy=1 AND c.status IN ('connected','synced') ORDER BY e.start_minutes,e.id`, date)
	if err != nil {
		// Calendar unit fixtures from before the integrations domain do not have
		// its optional tables. A fresh production schema always does.
		if strings.Contains(err.Error(), "no such table") {
			return externalBusyResult{}, nil
		}
		return externalBusyResult{}, err
	}
	defer rows.Close()
	result := externalBusyResult{}
	for rows.Next() {
		var start, duration int
		if err := rows.Scan(&start, &duration); err != nil {
			return externalBusyResult{}, err
		}
		result.intervals = append(result.intervals, capacityInterval{start: start, end: start + duration})
		result.events++
	}
	if err := rows.Err(); err != nil {
		return externalBusyResult{}, err
	}
	result.minutes = intervalUnionMinutes(append([]capacityInterval(nil), result.intervals...))
	if result.events > 0 {
		result.freshness = "fresh"
	}
	return result, nil
}

func (r *sqliteRepository) profileCapacity(ctx context.Context, date string) (int, int, error) {
	var capacity, reserve int
	var timezone string
	err := r.db.QueryRowContext(ctx, `SELECT daily_capacity_minutes,reserve_minutes,timezone FROM planning_profiles WHERE id='default'`).Scan(&capacity, &reserve, &timezone)
	if errors.Is(err, sql.ErrNoRows) {
		// A fresh workspace has the documented defaults until Settings first
		// materializes its profile row.
		return 480, 60, nil
	}
	if err != nil {
		return 0, 0, err
	}
	day, err := time.ParseInLocation("2006-01-02", date, func() *time.Location {
		loc, loadErr := time.LoadLocation(timezone)
		if loadErr != nil {
			return time.UTC
		}
		return loc
	}())
	if err != nil {
		return 0, 0, err
	}
	weekday := (int(day.Weekday())+6)%7 + 1
	rows, err := r.db.QueryContext(ctx, `SELECT start_minute,end_minute FROM availability_rules WHERE weekday=? AND (effective_start_date='' OR effective_start_date<=?) AND (effective_end_date='' OR effective_end_date>=?) ORDER BY priority,start_minute`, weekday, date, date)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	total := 0
	for rows.Next() {
		var start, end int
		if err := rows.Scan(&start, &end); err != nil {
			return 0, 0, err
		}
		total += end - start
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	if total == 0 {
		return capacity, reserve, nil
	}
	protectedRows, err := r.db.QueryContext(ctx, `SELECT start_minute,end_minute,kind FROM availability_exceptions WHERE date=?`, date)
	if err != nil {
		return 0, 0, err
	}
	var protected, extra []capacityInterval
	for protectedRows.Next() {
		var start, end int
		var kind string
		if err := protectedRows.Scan(&start, &end, &kind); err != nil {
			protectedRows.Close()
			return 0, 0, err
		}
		if kind == "protected" {
			protected = append(protected, capacityInterval{start, end})
		} else if kind == "extra" {
			extra = append(extra, capacityInterval{start, end})
		}
	}
	if err := protectedRows.Err(); err != nil {
		protectedRows.Close()
		return 0, 0, err
	}
	protectedRows.Close()
	total -= intervalUnionMinutes(protected)
	total += intervalUnionMinutes(extra)
	if total < 0 {
		total = 0
	}
	if capacity > 0 && total > capacity {
		total = capacity
	}
	return total, reserve, nil
}

type capacityInterval struct{ start, end int }

func intervalUnionMinutes(items []capacityInterval) int {
	if len(items) == 0 {
		return 0
	}
	sort.Slice(items, func(i, j int) bool { return items[i].start < items[j].start })
	start, end, total := items[0].start, items[0].end, 0
	for _, item := range items[1:] {
		if item.start > end {
			total += end - start
			start, end = item.start, item.end
		} else if item.end > end {
			end = item.end
		}
	}
	return total + end - start
}

func (r *sqliteRepository) ListRange(ctx context.Context, start, end string) (DateRange, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.work_item_id,w.title,w.source_label,a.local_date,a.start_minutes,a.duration_minutes,a.state,a.created_at FROM calendar_allocations a JOIN work_items w ON w.id=a.work_item_id WHERE a.local_date>=? AND a.local_date<=? AND a.state='accepted' ORDER BY a.local_date,a.start_minutes,a.id`, start, end)
	if err != nil {
		return DateRange{}, err
	}
	defer rows.Close()
	rangeResult := DateRange{Allocations: []Allocation{}}
	for rows.Next() {
		var a Allocation
		var created string
		if err := rows.Scan(&a.ID, &a.WorkItemID, &a.Title, &a.SourceLabel, &a.LocalDate, &a.StartMinutes, &a.DurationMinutes, &a.State, &created); err != nil {
			return DateRange{}, err
		}
		a.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return DateRange{}, err
		}
		rangeResult.Allocations = append(rangeResult.Allocations, a)
	}
	if err := rows.Err(); err != nil {
		return DateRange{}, err
	}
	return rangeResult, nil
}

func (r *sqliteRepository) Create(ctx context.Context, a Allocation) (Allocation, error) {
	a.ID = r.id()
	a.State = "accepted"
	a.CreatedAt = r.clock.Now().UTC()
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM calendar_allocations WHERE local_date=? AND state='accepted' AND start_minutes < ? AND start_minutes + duration_minutes > ?`, a.LocalDate, a.StartMinutes+a.DurationMinutes, a.StartMinutes).Scan(&count); err != nil {
		return Allocation{}, err
	}
	if count > 0 {
		return Allocation{}, ErrAllocationConflict{}
	}
	busy, err := importedOverlap(ctx, r.db, a.LocalDate, a.StartMinutes, a.DurationMinutes)
	if err != nil {
		return Allocation{}, err
	}
	if busy {
		return Allocation{}, ErrAllocationConflict{}
	}
	if remaining, known, demandErr := r.remainingSchedulableMinutes(ctx, a.WorkItemID); demandErr != nil {
		return Allocation{}, demandErr
	} else if known && a.DurationMinutes > remaining {
		return Allocation{}, ErrDemandExceeded{a.WorkItemID}
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO calendar_allocations (id,work_item_id,local_date,start_minutes,duration_minutes,state,created_at) VALUES (?,?,?,?,?,?,?)`, a.ID, a.WorkItemID, a.LocalDate, a.StartMinutes, a.DurationMinutes, a.State, a.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Allocation{}, fmt.Errorf("insert allocation: %w", err)
	}
	if err := bumpScheduleRevision(ctx, r.db); err != nil {
		return Allocation{}, fmt.Errorf("advance schedule revision: %w", err)
	}
	return a, nil
}

func importedOverlap(ctx context.Context, db SQLExecutor, date string, start, duration int) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM imported_events e JOIN provider_connections c ON c.id=e.connection_id WHERE e.local_date=? AND e.status='active' AND e.busy=1 AND c.status IN ('connected','synced') AND e.start_minutes < ? AND e.start_minutes + e.duration_minutes > ?`, date, start+duration, start).Scan(&count)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *sqliteRepository) CarryForward(ctx context.Context, in CarryForwardInput) (Allocation, error) {
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return Allocation{}, fmt.Errorf("calendar storage does not support transactional carry-forward")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return Allocation{}, fmt.Errorf("begin carry-forward: %w", err)
	}
	rollback := func(cause error) (Allocation, error) {
		_ = tx.Rollback()
		return Allocation{}, cause
	}

	var source Allocation
	var created string
	err = tx.QueryRowContext(ctx, `SELECT a.id,a.work_item_id,w.title,w.source_label,a.local_date,a.start_minutes,a.duration_minutes,a.state,a.created_at FROM calendar_allocations a JOIN work_items w ON w.id=a.work_item_id WHERE a.id=?`, in.AllocationID).Scan(&source.ID, &source.WorkItemID, &source.Title, &source.SourceLabel, &source.LocalDate, &source.StartMinutes, &source.DurationMinutes, &source.State, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return rollback(ErrAllocationNotFound{in.AllocationID})
	}
	if err != nil {
		return rollback(fmt.Errorf("read source allocation: %w", err))
	}
	if source.State != "accepted" {
		var carriedID string
		if lookupErr := tx.QueryRowContext(ctx, `SELECT carried_allocation_id FROM allocation_carry_forwards WHERE source_allocation_id=?`, source.ID).Scan(&carriedID); lookupErr == nil {
			carried, loadErr := loadAllocation(ctx, tx, carriedID)
			if loadErr != nil {
				return rollback(loadErr)
			}
			if commitErr := tx.Commit(); commitErr != nil {
				return Allocation{}, fmt.Errorf("commit idempotent carry-forward: %w", commitErr)
			}
			carried.CarriedFromID = source.ID
			return carried, nil
		}
		return rollback(ErrAllocationAlreadyCarried{source.ID})
	}
	if parsed, parseErr := time.Parse(time.RFC3339Nano, created); parseErr == nil {
		source.CreatedAt = parsed
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM calendar_allocations WHERE local_date=? AND state='accepted' AND start_minutes < ? AND start_minutes + duration_minutes > ?`, in.TargetLocalDate, in.StartMinutes+source.DurationMinutes, in.StartMinutes).Scan(&count); err != nil {
		return rollback(fmt.Errorf("check carry-forward overlap: %w", err))
	}
	if count > 0 {
		return rollback(ErrAllocationConflict{})
	}
	carried := Allocation{ID: r.id(), WorkItemID: source.WorkItemID, Title: source.Title, SourceLabel: source.SourceLabel, LocalDate: in.TargetLocalDate, StartMinutes: in.StartMinutes, DurationMinutes: source.DurationMinutes, State: "accepted", CarriedFromID: source.ID, CreatedAt: r.clock.Now().UTC()}
	if _, err := tx.ExecContext(ctx, `INSERT INTO calendar_allocations (id,work_item_id,local_date,start_minutes,duration_minutes,state,created_at) VALUES (?,?,?,?,?,?,?)`, carried.ID, carried.WorkItemID, carried.LocalDate, carried.StartMinutes, carried.DurationMinutes, carried.State, carried.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return rollback(fmt.Errorf("insert carried allocation: %w", err))
	}
	updated, err := tx.ExecContext(ctx, `UPDATE calendar_allocations SET state='carried_forward' WHERE id=? AND state='accepted'`, source.ID)
	if err != nil {
		return rollback(fmt.Errorf("close source allocation: %w", err))
	}
	if affected, _ := updated.RowsAffected(); affected != 1 {
		return rollback(ErrAllocationAlreadyCarried{source.ID})
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO allocation_carry_forwards (source_allocation_id,carried_allocation_id,target_local_date,created_at) VALUES (?,?,?,?)`, source.ID, carried.ID, carried.LocalDate, carried.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return rollback(fmt.Errorf("record carry-forward: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO reschedule_history (id,allocation_id,from_date,to_date,reason_code,rescheduled_at) VALUES (?,?,?,?,?,?)`, r.id(), source.ID, source.LocalDate, carried.LocalDate, in.ReasonCode, carried.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return rollback(fmt.Errorf("record reschedule history: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return Allocation{}, fmt.Errorf("commit carry-forward: %w", err)
	}
	if err := bumpScheduleRevision(ctx, r.db); err != nil {
		return Allocation{}, fmt.Errorf("advance schedule revision: %w", err)
	}
	return carried, nil
}

func loadAllocation(ctx context.Context, db SQLExecutor, id string) (Allocation, error) {
	var a Allocation
	var created string
	err := db.QueryRowContext(ctx, `SELECT a.id,a.work_item_id,w.title,w.source_label,a.local_date,a.start_minutes,a.duration_minutes,a.state,a.created_at FROM calendar_allocations a JOIN work_items w ON w.id=a.work_item_id WHERE a.id=?`, id).Scan(&a.ID, &a.WorkItemID, &a.Title, &a.SourceLabel, &a.LocalDate, &a.StartMinutes, &a.DurationMinutes, &a.State, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Allocation{}, ErrAllocationNotFound{id}
	}
	if err != nil {
		return Allocation{}, err
	}
	if a.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Allocation{}, err
	}
	a.CarriedFromID = ""
	return a, nil
}

func (r *sqliteRepository) ListRoutines(ctx context.Context) ([]Routine, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,kind,timezone,start_date,end_date,weekdays,start_minute,duration_minutes,frequency_per_week,revision,active FROM routines WHERE active=1 ORDER BY start_date,title,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Routine
	for rows.Next() {
		var routine Routine
		var weekdayText string
		var active int
		if err := rows.Scan(&routine.ID, &routine.Title, &routine.Kind, &routine.Timezone, &routine.StartDate, &routine.EndDate, &weekdayText, &routine.StartMinute, &routine.DurationMinutes, &routine.FrequencyPerWeek, &routine.Revision, &active); err != nil {
			return nil, err
		}
		routine.Weekdays = parseWeekdays(weekdayText)
		routine.Active = active == 1
		out = append(out, routine)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateRoutine(ctx context.Context, in CreateRoutineInput) (Routine, error) {
	routine := Routine{ID: r.id(), Title: in.Title, Kind: in.Kind, Timezone: in.Timezone, StartDate: in.StartDate, EndDate: in.EndDate, Weekdays: append([]int(nil), in.Weekdays...), StartMinute: in.StartMinute, DurationMinutes: in.DurationMinutes, FrequencyPerWeek: in.FrequencyPerWeek, Revision: 1, Active: true}
	_, err := r.db.ExecContext(ctx, `INSERT INTO routines (id,title,kind,timezone,start_date,end_date,weekdays,start_minute,duration_minutes,frequency_per_week,revision,active,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, routine.ID, routine.Title, routine.Kind, routine.Timezone, routine.StartDate, routine.EndDate, formatWeekdays(routine.Weekdays), routine.StartMinute, routine.DurationMinutes, routine.FrequencyPerWeek, routine.Revision, 1, r.clock.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Routine{}, fmt.Errorf("insert routine: %w", err)
	}
	return routine, nil
}

func (r *sqliteRepository) ListRoutineOccurrences(ctx context.Context, start, end string) ([]RoutineOccurrence, error) {
	routines, err := r.ListRoutines(ctx)
	if err != nil {
		return nil, err
	}
	skippedRows, err := r.db.QueryContext(ctx, `SELECT routine_id,local_date FROM routine_occurrence_overrides WHERE status='skipped' AND local_date>=? AND local_date<=?`, start, end)
	if err != nil {
		return nil, err
	}
	skipped := map[string]bool{}
	for skippedRows.Next() {
		var routineID, localDate string
		if err := skippedRows.Scan(&routineID, &localDate); err != nil {
			skippedRows.Close()
			return nil, err
		}
		skipped[routineID+"/"+localDate] = true
	}
	if err := skippedRows.Err(); err != nil {
		skippedRows.Close()
		return nil, err
	}
	skippedRows.Close()
	rescheduledRows, err := r.db.QueryContext(ctx, `SELECT routine_id,local_date,start_minute FROM routine_occurrence_reschedules WHERE local_date>=? AND local_date<=?`, start, end)
	if err != nil {
		return nil, err
	}
	rescheduled := map[string]int{}
	for rescheduledRows.Next() {
		var routineID, localDate string
		var startMinute int
		if err := rescheduledRows.Scan(&routineID, &localDate, &startMinute); err != nil {
			rescheduledRows.Close()
			return nil, err
		}
		rescheduled[routineID+"/"+localDate] = startMinute
	}
	if err := rescheduledRows.Err(); err != nil {
		rescheduledRows.Close()
		return nil, err
	}
	rescheduledRows.Close()
	startDate, _ := time.Parse("2006-01-02", start)
	endDate, _ := time.Parse("2006-01-02", end)
	var out []RoutineOccurrence
	for _, routine := range routines {
		first, _ := time.Parse("2006-01-02", routine.StartDate)
		last := endDate
		if routine.EndDate != "" {
			if candidate, parseErr := time.Parse("2006-01-02", routine.EndDate); parseErr == nil && candidate.Before(last) {
				last = candidate
			}
		}
		if first.After(last) {
			continue
		}
		rangeStart := startDate
		if first.After(rangeStart) {
			rangeStart = first
		}
		for date := rangeStart; !date.After(last); date = date.AddDate(0, 0, 1) {
			localDate := date.Format("2006-01-02")
			if skipped[routine.ID+"/"+localDate] {
				continue
			}
			day := (int(date.Weekday())+6)%7 + 1
			eligible := containsWeekday(routine.Weekdays, day)
			if !eligible {
				continue
			}
			if routine.Kind == "flexible" && !isFirstFlexibleDay(routine, date) {
				continue
			}
			startMinute := routine.StartMinute
			if moved, ok := rescheduled[routine.ID+"/"+localDate]; ok {
				startMinute = moved
			}
			out = append(out, RoutineOccurrence{RoutineID: routine.ID, Title: routine.Title, LocalDate: localDate, StartMinute: startMinute, DurationMinutes: routine.DurationMinutes, Kind: routine.Kind, Generated: true})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LocalDate != out[j].LocalDate {
			return out[i].LocalDate < out[j].LocalDate
		}
		return out[i].StartMinute < out[j].StartMinute
	})
	return out, nil
}

func (r *sqliteRepository) SkipRoutineOccurrence(ctx context.Context, routineID, localDate string, expectedRevision int64) error {
	var currentRevision int64
	err := r.db.QueryRowContext(ctx, `SELECT revision FROM routines WHERE id=? AND active=1`, routineID).Scan(&currentRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoutineNotFound{routineID}
	}
	if err != nil {
		return err
	}
	if currentRevision != expectedRevision {
		return ErrRoutineRevisionConflict{routineID}
	}
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := r.db.ExecContext(ctx, `UPDATE routines SET revision=revision+1 WHERE id=? AND revision=?`, routineID, expectedRevision); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO routine_occurrence_overrides (routine_id,local_date,status,revision,updated_at) VALUES (?,?,?,?,?) ON CONFLICT(routine_id,local_date) DO UPDATE SET status=excluded.status,revision=excluded.revision,updated_at=excluded.updated_at`, routineID, localDate, "skipped", expectedRevision+1, now)
	return err
}

func (r *sqliteRepository) RescheduleRoutineOccurrence(ctx context.Context, routineID, localDate string, startMinute int, expectedRevision int64) error {
	var currentRevision int64
	err := r.db.QueryRowContext(ctx, `SELECT revision FROM routines WHERE id=? AND active=1`, routineID).Scan(&currentRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoutineNotFound{routineID}
	}
	if err != nil {
		return err
	}
	if currentRevision != expectedRevision {
		return ErrRoutineRevisionConflict{routineID}
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE routines SET revision=revision+1 WHERE id=? AND revision=?`, routineID, expectedRevision); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO routine_occurrence_reschedules (routine_id,local_date,start_minute,revision,updated_at) VALUES (?,?,?,?,?) ON CONFLICT(routine_id,local_date) DO UPDATE SET start_minute=excluded.start_minute,revision=excluded.revision,updated_at=excluded.updated_at`, routineID, localDate, startMinute, expectedRevision+1, r.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func formatWeekdays(days []int) string {
	parts := make([]string, len(days))
	for i, day := range days {
		parts[i] = strconv.Itoa(day)
	}
	return strings.Join(parts, ",")
}

func parseWeekdays(value string) []int {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if day, err := strconv.Atoi(part); err == nil {
			out = append(out, day)
		}
	}
	return out
}

func containsWeekday(days []int, wanted int) bool {
	for _, day := range days {
		if day == wanted {
			return true
		}
	}
	return false
}

func isFirstFlexibleDay(routine Routine, date time.Time) bool {
	count := 0
	monday := date.AddDate(0, 0, -((int(date.Weekday()) + 6) % 7))
	for day := monday; !day.After(date); day = day.AddDate(0, 0, 1) {
		candidate := (int(day.Weekday())+6)%7 + 1
		if containsWeekday(routine.Weekdays, candidate) {
			count++
		}
	}
	return count <= routine.FrequencyPerWeek
}

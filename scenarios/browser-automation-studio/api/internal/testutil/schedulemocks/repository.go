package schedulemocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// Repository is a synchronized fake for the shared schedule persistence seam.
type Repository struct {
	ListErr, GetErr, CreateErr, UpdateErr, DeleteErr error
	NextRunErr, LastRunErr                           error

	mu           sync.RWMutex
	Items        map[uuid.UUID]*database.ScheduleIndex
	lastRun      map[uuid.UUID]time.Time
	NextRunCalls atomic.Int32
	LastRunCalls atomic.Int32
}

func NewRepository(schedules ...*database.ScheduleIndex) *Repository {
	repo := &Repository{Items: make(map[uuid.UUID]*database.ScheduleIndex), lastRun: make(map[uuid.UUID]time.Time)}
	for _, schedule := range schedules {
		if schedule != nil {
			repo.Items[schedule.ID] = schedule
		}
	}
	return repo
}

func (r *Repository) Add(schedule *database.ScheduleIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if schedule != nil {
		r.Items[schedule.ID] = schedule
	}
}

func (r *Repository) CreateSchedule(_ context.Context, schedule *database.ScheduleIndex) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = time.Now()
	}
	schedule.UpdatedAt = time.Now()
	r.Items[schedule.ID] = schedule
	return nil
}

func (r *Repository) GetSchedule(_ context.Context, id uuid.UUID) (*database.ScheduleIndex, error) {
	if r.GetErr != nil {
		return nil, r.GetErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	schedule, ok := r.Items[id]
	if !ok || schedule == nil {
		return nil, database.ErrNotFound
	}
	return schedule, nil
}

func (r *Repository) UpdateSchedule(_ context.Context, schedule *database.ScheduleIndex) error {
	if r.UpdateErr != nil {
		return r.UpdateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	schedule.UpdatedAt = time.Now()
	r.Items[schedule.ID] = schedule
	return nil
}

func (r *Repository) DeleteSchedule(_ context.Context, id uuid.UUID) error {
	if r.DeleteErr != nil {
		return r.DeleteErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.Items[id]; !ok {
		return database.ErrNotFound
	}
	delete(r.Items, id)
	return nil
}

func (r *Repository) ListSchedules(_ context.Context, workflowID *uuid.UUID, activeOnly bool, _, _ int) ([]*database.ScheduleIndex, error) {
	if r.ListErr != nil {
		return nil, r.ListErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*database.ScheduleIndex
	for _, schedule := range r.Items {
		if schedule == nil || workflowID != nil && schedule.WorkflowID != *workflowID || activeOnly && !schedule.IsActive {
			continue
		}
		out = append(out, schedule)
	}
	return out, nil
}

func (r *Repository) UpdateScheduleNextRun(_ context.Context, id uuid.UUID, nextRun time.Time) error {
	r.NextRunCalls.Add(1)
	if r.NextRunErr != nil {
		return r.NextRunErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	schedule, ok := r.Items[id]
	if !ok || schedule == nil {
		return database.ErrNotFound
	}
	schedule.NextRunAt = &nextRun
	return nil
}

func (r *Repository) UpdateScheduleLastRun(_ context.Context, id uuid.UUID, lastRun time.Time) error {
	r.LastRunCalls.Add(1)
	if r.LastRunErr != nil {
		return r.LastRunErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRun[id] = lastRun
	if schedule, ok := r.Items[id]; ok && schedule != nil {
		schedule.LastRunAt = &lastRun
	}
	return nil
}

func (r *Repository) LastRun(id uuid.UUID) (time.Time, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lastRun, ok := r.lastRun[id]
	return lastRun, ok
}

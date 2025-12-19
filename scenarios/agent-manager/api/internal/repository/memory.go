// Package repository provides persistence implementations for domain entities.
//
// This file contains in-memory implementations of the repository interfaces.
// These are suitable for development, testing, and lightweight deployments.
// For production with persistence, use the PostgreSQL implementations.
package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// =============================================================================
// In-Memory Profile Repository
// =============================================================================

// MemoryProfileRepository is an in-memory implementation of ProfileRepository.
type MemoryProfileRepository struct {
	mu       sync.RWMutex
	profiles map[uuid.UUID]*domain.AgentProfile
	byName   map[string]uuid.UUID
}

// NewMemoryProfileRepository creates a new in-memory profile repository.
func NewMemoryProfileRepository() *MemoryProfileRepository {
	return &MemoryProfileRepository{
		profiles: make(map[uuid.UUID]*domain.AgentProfile),
		byName:   make(map[string]uuid.UUID),
	}
}

func (r *MemoryProfileRepository) Create(ctx context.Context, profile *domain.AgentProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.profiles[profile.ID]; exists {
		return fmt.Errorf("profile already exists: %s", profile.ID)
	}
	if _, exists := r.byName[profile.Name]; exists {
		return fmt.Errorf("profile name already exists: %s", profile.Name)
	}

	// Deep copy to prevent external mutation
	copy := *profile
	r.profiles[profile.ID] = &copy
	r.byName[profile.Name] = profile.ID
	return nil
}

func (r *MemoryProfileRepository) Get(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, exists := r.profiles[id]
	if !exists {
		return nil, nil
	}
	// Return a copy
	copy := *profile
	return &copy, nil
}

func (r *MemoryProfileRepository) GetByName(ctx context.Context, name string) (*domain.AgentProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byName[name]
	if !exists {
		return nil, nil
	}
	profile := r.profiles[id]
	copy := *profile
	return &copy, nil
}

func (r *MemoryProfileRepository) List(ctx context.Context, filter ListFilter) ([]*domain.AgentProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.AgentProfile, 0, len(r.profiles))
	for _, p := range r.profiles {
		copy := *p
		result = append(result, &copy)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		return []*domain.AgentProfile{}, nil
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *MemoryProfileRepository) Update(ctx context.Context, profile *domain.AgentProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.profiles[profile.ID]
	if !exists {
		return fmt.Errorf("profile not found: %s", profile.ID)
	}

	// Update name index if name changed
	if existing.Name != profile.Name {
		delete(r.byName, existing.Name)
		r.byName[profile.Name] = profile.ID
	}

	copy := *profile
	r.profiles[profile.ID] = &copy
	return nil
}

func (r *MemoryProfileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	profile, exists := r.profiles[id]
	if !exists {
		return nil // Idempotent delete
	}

	delete(r.byName, profile.Name)
	delete(r.profiles, id)
	return nil
}

// Verify interface compliance
var _ ProfileRepository = (*MemoryProfileRepository)(nil)

// =============================================================================
// In-Memory Task Repository
// =============================================================================

// MemoryTaskRepository is an in-memory implementation of TaskRepository.
type MemoryTaskRepository struct {
	mu    sync.RWMutex
	tasks map[uuid.UUID]*domain.Task
}

// NewMemoryTaskRepository creates a new in-memory task repository.
func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{
		tasks: make(map[uuid.UUID]*domain.Task),
	}
}

func (r *MemoryTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return fmt.Errorf("task already exists: %s", task.ID)
	}

	copy := *task
	r.tasks[task.ID] = &copy
	return nil
}

func (r *MemoryTaskRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, nil
	}
	copy := *task
	return &copy, nil
}

func (r *MemoryTaskRepository) List(ctx context.Context, filter ListFilter) ([]*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		copy := *t
		result = append(result, &copy)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		return []*domain.Task{}, nil
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *MemoryTaskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, filter ListFilter) ([]*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Task
	for _, t := range r.tasks {
		if t.Status == status {
			copy := *t
			result = append(result, &copy)
		}
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		return []*domain.Task{}, nil
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *MemoryTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	copy := *task
	r.tasks[task.ID] = &copy
	return nil
}

func (r *MemoryTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tasks, id)
	return nil
}

// Verify interface compliance
var _ TaskRepository = (*MemoryTaskRepository)(nil)

// =============================================================================
// In-Memory Run Repository
// =============================================================================

// MemoryRunRepository is an in-memory implementation of RunRepository.
type MemoryRunRepository struct {
	mu   sync.RWMutex
	runs map[uuid.UUID]*domain.Run
}

// NewMemoryRunRepository creates a new in-memory run repository.
func NewMemoryRunRepository() *MemoryRunRepository {
	return &MemoryRunRepository{
		runs: make(map[uuid.UUID]*domain.Run),
	}
}

func (r *MemoryRunRepository) Create(ctx context.Context, run *domain.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runs[run.ID]; exists {
		return fmt.Errorf("run already exists: %s", run.ID)
	}

	copy := *run
	r.runs[run.ID] = &copy
	return nil
}

func (r *MemoryRunRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, exists := r.runs[id]
	if !exists {
		return nil, nil
	}
	copy := *run
	return &copy, nil
}

func (r *MemoryRunRepository) List(ctx context.Context, filter RunListFilter) ([]*domain.Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Run
	for _, run := range r.runs {
		// Apply filters
		if filter.TaskID != nil && run.TaskID != *filter.TaskID {
			continue
		}
		if filter.AgentProfileID != nil && run.AgentProfileID != *filter.AgentProfileID {
			continue
		}
		if filter.Status != nil && run.Status != *filter.Status {
			continue
		}

		copy := *run
		result = append(result, &copy)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		return []*domain.Run{}, nil
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (r *MemoryRunRepository) ListByTask(ctx context.Context, taskID uuid.UUID, filter ListFilter) ([]*domain.Run, error) {
	return r.List(ctx, RunListFilter{
		ListFilter: filter,
		TaskID:     &taskID,
	})
}

func (r *MemoryRunRepository) Update(ctx context.Context, run *domain.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runs[run.ID]; !exists {
		return fmt.Errorf("run not found: %s", run.ID)
	}

	copy := *run
	r.runs[run.ID] = &copy
	return nil
}

func (r *MemoryRunRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.runs, id)
	return nil
}

func (r *MemoryRunRepository) CountByStatus(ctx context.Context, status domain.RunStatus) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, run := range r.runs {
		if run.Status == status {
			count++
		}
	}
	return count, nil
}

// Verify interface compliance
var _ RunRepository = (*MemoryRunRepository)(nil)

// =============================================================================
// In-Memory Event Repository
// =============================================================================

// MemoryEventRepository is an in-memory implementation of EventRepository.
type MemoryEventRepository struct {
	mu       sync.RWMutex
	events   map[uuid.UUID][]*domain.RunEvent // runID -> events
	sequence map[uuid.UUID]int64              // runID -> next sequence
}

// NewMemoryEventRepository creates a new in-memory event repository.
func NewMemoryEventRepository() *MemoryEventRepository {
	return &MemoryEventRepository{
		events:   make(map[uuid.UUID][]*domain.RunEvent),
		sequence: make(map[uuid.UUID]int64),
	}
}

func (r *MemoryEventRepository) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, evt := range events {
		seq := r.sequence[runID]
		evt.RunID = runID
		evt.Sequence = seq
		r.sequence[runID] = seq + 1

		copy := *evt
		r.events[runID] = append(r.events[runID], &copy)
	}

	return nil
}

func (r *MemoryEventRepository) Get(ctx context.Context, runID uuid.UUID, afterSequence int64, limit int) ([]*domain.RunEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.events[runID]
	var result []*domain.RunEvent

	for _, evt := range events {
		if evt.Sequence > afterSequence {
			copy := *evt
			result = append(result, &copy)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

func (r *MemoryEventRepository) GetByType(ctx context.Context, runID uuid.UUID, types []domain.RunEventType, limit int) ([]*domain.RunEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := r.events[runID]
	var result []*domain.RunEvent

	typeSet := make(map[domain.RunEventType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	for _, evt := range events {
		if typeSet[evt.EventType] {
			copy := *evt
			result = append(result, &copy)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

func (r *MemoryEventRepository) Count(ctx context.Context, runID uuid.UUID) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.events[runID])), nil
}

func (r *MemoryEventRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.events, runID)
	delete(r.sequence, runID)
	return nil
}

// Verify interface compliance
var _ EventRepository = (*MemoryEventRepository)(nil)

// =============================================================================
// In-Memory Checkpoint Repository
// =============================================================================

// MemoryCheckpointRepository is an in-memory implementation of CheckpointRepository.
type MemoryCheckpointRepository struct {
	mu          sync.RWMutex
	checkpoints map[uuid.UUID]*domain.RunCheckpoint
}

// NewMemoryCheckpointRepository creates a new in-memory checkpoint repository.
func NewMemoryCheckpointRepository() *MemoryCheckpointRepository {
	return &MemoryCheckpointRepository{
		checkpoints: make(map[uuid.UUID]*domain.RunCheckpoint),
	}
}

func (r *MemoryCheckpointRepository) Save(ctx context.Context, checkpoint *domain.RunCheckpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copy := *checkpoint
	r.checkpoints[checkpoint.RunID] = &copy
	return nil
}

func (r *MemoryCheckpointRepository) Get(ctx context.Context, runID uuid.UUID) (*domain.RunCheckpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cp, exists := r.checkpoints[runID]
	if !exists {
		return nil, nil
	}
	copy := *cp
	return &copy, nil
}

func (r *MemoryCheckpointRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.checkpoints, runID)
	return nil
}

func (r *MemoryCheckpointRepository) ListStale(ctx context.Context, olderThan time.Duration) ([]*domain.RunCheckpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cutoff := time.Now().Add(-olderThan)
	var result []*domain.RunCheckpoint

	for _, cp := range r.checkpoints {
		if cp.LastHeartbeat.Before(cutoff) {
			copy := *cp
			result = append(result, &copy)
		}
	}

	return result, nil
}

func (r *MemoryCheckpointRepository) Heartbeat(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp, exists := r.checkpoints[runID]
	if !exists {
		return nil
	}

	cp.LastHeartbeat = time.Now()
	return nil
}

// Verify interface compliance
var _ CheckpointRepository = (*MemoryCheckpointRepository)(nil)

// =============================================================================
// In-Memory Idempotency Repository
// =============================================================================

// MemoryIdempotencyRepository is an in-memory implementation of IdempotencyRepository.
type MemoryIdempotencyRepository struct {
	mu      sync.RWMutex
	records map[string]*domain.IdempotencyRecord
}

// NewMemoryIdempotencyRepository creates a new in-memory idempotency repository.
func NewMemoryIdempotencyRepository() *MemoryIdempotencyRepository {
	return &MemoryIdempotencyRepository{
		records: make(map[string]*domain.IdempotencyRecord),
	}
}

func (r *MemoryIdempotencyRepository) Check(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rec, exists := r.records[key]
	if !exists {
		return nil, nil
	}

	// Check expiration
	if time.Now().After(rec.ExpiresAt) {
		return nil, nil
	}

	copy := *rec
	return &copy, nil
}

func (r *MemoryIdempotencyRepository) Reserve(ctx context.Context, key string, ttl time.Duration) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if key already exists and is not expired
	if existing, exists := r.records[key]; exists {
		if time.Now().Before(existing.ExpiresAt) {
			return nil, fmt.Errorf("idempotency key already reserved: %s", key)
		}
	}

	rec := &domain.IdempotencyRecord{
		Key:       key,
		Status:    domain.IdempotencyStatusPending,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	r.records[key] = rec
	copy := *rec
	return &copy, nil
}

func (r *MemoryIdempotencyRepository) Complete(ctx context.Context, key string, entityID uuid.UUID, entityType string, response []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, exists := r.records[key]
	if !exists {
		return fmt.Errorf("idempotency record not found: %s", key)
	}

	rec.Status = domain.IdempotencyStatusComplete
	rec.EntityID = &entityID
	rec.EntityType = entityType
	rec.Response = response
	return nil
}

func (r *MemoryIdempotencyRepository) Fail(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, exists := r.records[key]
	if !exists {
		return nil // Idempotent
	}

	rec.Status = domain.IdempotencyStatusFailed
	return nil
}

func (r *MemoryIdempotencyRepository) CleanupExpired(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	count := 0

	for key, rec := range r.records {
		if now.After(rec.ExpiresAt) {
			delete(r.records, key)
			count++
		}
	}

	return count, nil
}

// Verify interface compliance
var _ IdempotencyRepository = (*MemoryIdempotencyRepository)(nil)

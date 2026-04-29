// Package mocks contains hand-written fakes for every interface in the
// workspace-sandbox API surface. Each fake is the single canonical
// implementation; production tests must not author their own.
package mocks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"
)

// FakeRepository is the canonical in-memory repository.Repository
// implementation for tests. It keeps full state for sandboxes,
// audit events, applied changes, and heal-state rows so that tests
// can assert on durable side effects without a real SQLite database.
//
// All state is exported so tests can both seed (for arrange) and
// inspect (for assert) without going through accessor methods.
// Mutations are guarded by an internal mutex so concurrent
// reconcilers don't race.
//
// # Error injection
//
// Every method that returns an error has a matching `Err*` field.
// Setting `r.GetErr = errors.New("...")` makes Get return that error
// on the next call. The repository delete helper (DeleteFailIDs)
// supports per-ID failure for partial-failure tests.
type FakeRepository struct {
	mu sync.Mutex

	// State
	Sandboxes        map[uuid.UUID]*types.Sandbox
	IdempotencyIndex map[string]*types.Sandbox
	AuditEvents      []*types.AuditEvent
	AppliedChanges   []*types.AppliedChange
	HealStates       map[uuid.UUID]repository.HealStateRow
	ScopeConflicts   []types.PathConflict
	GCCandidates     []*types.Sandbox
	Stats            *types.SandboxStats
	DeletedIDs       []uuid.UUID

	// Per-method error injection. Zero value means no error.
	CreateErr                     error
	GetErr                        error
	UpdateErr                     error
	UpdateWithVersionCheckErr     error
	DeleteErr                     error
	ListErr                       error
	CheckScopeOverlapErr          error
	GetActiveSandboxesErr         error
	FindByIdempotencyKeyErr       error
	LogAuditEventErr              error
	GetAuditLogErr                error
	GetStatsErr                   error
	RecordAppliedChangesErr       error
	GetPendingChangesErr          error
	GetPendingChangeFilesErr      error
	GetFileProvenanceErr          error
	MarkChangesCommittedErr       error
	MarkChangesCommittedByPathErr error
	GetPendingChangesByRunErr     error
	GetHealStateErr               error
	UpsertHealStateErr            error
	ClearHealStateErr             error
	ListHealStateErr              error
	GetGCCandidatesErr            error
	BeginTxErr                    error

	// DeleteFailIDs lets gc-style tests fail Delete for specific IDs
	// only. When non-empty, Delete returns DeleteErr only for IDs in
	// the map; other IDs succeed even if DeleteErr is set.
	DeleteFailIDs map[uuid.UUID]bool

	// TxFactory builds the TxRepository returned from BeginTx. When
	// nil, the default FakeTxRepository wrapping this FakeRepository
	// is returned.
	TxFactory func(*FakeRepository) repository.TxRepository
}

// NewFakeRepository returns a fresh FakeRepository with empty state
// maps and no errors. The default Stats has zero counts.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		Sandboxes:        make(map[uuid.UUID]*types.Sandbox),
		IdempotencyIndex: make(map[string]*types.Sandbox),
		AuditEvents:      []*types.AuditEvent{},
		AppliedChanges:   []*types.AppliedChange{},
		HealStates:       make(map[uuid.UUID]repository.HealStateRow),
		DeleteFailIDs:    make(map[uuid.UUID]bool),
	}
}

// AuditEventCount returns the number of recorded audit events whose
// EventType matches `eventType`. Useful for asserting that a code
// path emitted exactly the audit it should have.
func (r *FakeRepository) AuditEventCount(eventType string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.AuditEvents {
		if e != nil && e.EventType == eventType {
			n++
		}
	}
	return n
}

// SetSandbox installs a sandbox at its ID, bypassing Create's
// idempotency-index logic. Convenience for tests that arrange state
// outside the Create code path.
func (r *FakeRepository) SetSandbox(s *types.Sandbox) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Sandboxes[s.ID] = s
}

// --- repository.Repository implementation ---

func (r *FakeRepository) Create(ctx context.Context, s *types.Sandbox) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	s.Version = 1
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.LastUsedAt.IsZero() {
		s.LastUsedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	r.Sandboxes[s.ID] = s
	if s.IdempotencyKey != "" {
		r.IdempotencyIndex[s.IdempotencyKey] = s
	}
	return nil
}

func (r *FakeRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	if r.GetErr != nil {
		return nil, r.GetErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.Sandboxes[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (r *FakeRepository) Update(ctx context.Context, s *types.Sandbox) error {
	if r.UpdateErr != nil {
		return r.UpdateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s.Version++
	s.UpdatedAt = time.Now()
	r.Sandboxes[s.ID] = s
	return nil
}

func (r *FakeRepository) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
	if r.UpdateWithVersionCheckErr != nil {
		return r.UpdateWithVersionCheckErr
	}
	return r.Update(ctx, s)
}

func (r *FakeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.DeleteFailIDs) > 0 {
		if r.DeleteFailIDs[id] && r.DeleteErr != nil {
			return r.DeleteErr
		}
	} else if r.DeleteErr != nil {
		return r.DeleteErr
	}
	if _, ok := r.Sandboxes[id]; !ok {
		return errors.New("sandbox not found or already deleted")
	}
	delete(r.Sandboxes, id)
	r.DeletedIDs = append(r.DeletedIDs, id)
	return nil
}

func (r *FakeRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	if r.ListErr != nil {
		return nil, r.ListErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]*types.Sandbox, 0, len(r.Sandboxes))
	for _, s := range r.Sandboxes {
		if !filterMatches(s, filter) {
			continue
		}
		all = append(all, s)
	}
	limit, offset := 0, 0
	if filter != nil {
		limit, offset = filter.Limit, filter.Offset
	}
	return &types.ListResult{
		Sandboxes:  all,
		TotalCount: len(all),
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// filterMatches applies the subset of ListFilter semantics that real
// tests rely on. The production repo applies more (sort orders,
// pagination); test code that needs those builds against a real
// sqlite DB via testutil/db.
func filterMatches(s *types.Sandbox, f *types.ListFilter) bool {
	if f == nil {
		return true
	}
	if len(f.Status) > 0 {
		hit := false
		for _, st := range f.Status {
			if s.Status == st {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	if f.ProjectRoot != "" && s.ProjectRoot != f.ProjectRoot {
		return false
	}
	if f.Owner != "" && s.Owner != f.Owner {
		return false
	}
	return true
}

func (r *FakeRepository) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	if r.CheckScopeOverlapErr != nil {
		return nil, r.CheckScopeOverlapErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]types.PathConflict, len(r.ScopeConflicts))
	copy(out, r.ScopeConflicts)
	return out, nil
}

func (r *FakeRepository) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	if r.GetActiveSandboxesErr != nil {
		return nil, r.GetActiveSandboxesErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	active := make([]*types.Sandbox, 0)
	for _, s := range r.Sandboxes {
		if s.Status == types.StatusActive && s.ProjectRoot == projectRoot {
			active = append(active, s)
		}
	}
	return active, nil
}

func (r *FakeRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	if r.FindByIdempotencyKeyErr != nil {
		return nil, r.FindByIdempotencyKeyErr
	}
	if key == "" {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.IdempotencyIndex[key], nil
}

func (r *FakeRepository) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	if r.LogAuditEventErr != nil {
		return r.LogAuditEventErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AuditEvents = append(r.AuditEvents, event)
	return nil
}

func (r *FakeRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	if r.GetAuditLogErr != nil {
		return nil, 0, r.GetAuditLogErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if sandboxID == nil {
		return r.AuditEvents, len(r.AuditEvents), nil
	}
	out := make([]*types.AuditEvent, 0, len(r.AuditEvents))
	for _, e := range r.AuditEvents {
		if e != nil && e.SandboxID != nil && *e.SandboxID == *sandboxID {
			out = append(out, e)
		}
	}
	return out, len(out), nil
}

func (r *FakeRepository) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	if r.GetStatsErr != nil {
		return nil, r.GetStatsErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Stats != nil {
		return r.Stats, nil
	}
	return &types.SandboxStats{
		TotalCount:  int64(len(r.Sandboxes)),
		ActiveCount: int64(len(r.Sandboxes)),
	}, nil
}

func (r *FakeRepository) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	if r.RecordAppliedChangesErr != nil {
		return r.RecordAppliedChangesErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AppliedChanges = append(r.AppliedChanges, changes...)
	return nil
}

func (r *FakeRepository) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	if r.GetPendingChangesErr != nil {
		return nil, r.GetPendingChangesErr
	}
	return &types.PendingChangesResult{}, nil
}

func (r *FakeRepository) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	if r.GetPendingChangeFilesErr != nil {
		return nil, r.GetPendingChangeFilesErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*types.AppliedChange, len(r.AppliedChanges))
	copy(out, r.AppliedChanges)
	return out, nil
}

func (r *FakeRepository) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	if r.GetFileProvenanceErr != nil {
		return nil, r.GetFileProvenanceErr
	}
	return []*types.AppliedChange{}, nil
}

func (r *FakeRepository) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	if r.MarkChangesCommittedErr != nil {
		return r.MarkChangesCommittedErr
	}
	return nil
}

func (r *FakeRepository) MarkChangesCommittedByPath(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) (int, int, error) {
	if r.MarkChangesCommittedByPathErr != nil {
		return 0, 0, r.MarkChangesCommittedByPathErr
	}
	return len(filePaths), 0, nil
}

func (r *FakeRepository) GetPendingChangesByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error) {
	if r.GetPendingChangesByRunErr != nil {
		return nil, r.GetPendingChangesByRunErr
	}
	return nil, nil
}

func (r *FakeRepository) GetHealState(ctx context.Context, id uuid.UUID) (*repository.HealStateRow, error) {
	if r.GetHealStateErr != nil {
		return nil, r.GetHealStateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.HealStates[id]; ok {
		copy := row
		return &copy, nil
	}
	return nil, nil
}

func (r *FakeRepository) UpsertHealState(ctx context.Context, row repository.HealStateRow) error {
	if r.UpsertHealStateErr != nil {
		return r.UpsertHealStateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.HealStates[row.SandboxID] = row
	return nil
}

func (r *FakeRepository) ClearHealState(ctx context.Context, id uuid.UUID) error {
	if r.ClearHealStateErr != nil {
		return r.ClearHealStateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.HealStates, id)
	return nil
}

func (r *FakeRepository) ListHealState(ctx context.Context) ([]repository.HealStateRow, error) {
	if r.ListHealStateErr != nil {
		return nil, r.ListHealStateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]repository.HealStateRow, 0, len(r.HealStates))
	for _, row := range r.HealStates {
		out = append(out, row)
	}
	return out, nil
}

func (r *FakeRepository) BeginTx(ctx context.Context) (repository.TxRepository, error) {
	if r.BeginTxErr != nil {
		return nil, r.BeginTxErr
	}
	if r.TxFactory != nil {
		return r.TxFactory(r), nil
	}
	return &FakeTxRepository{FakeRepository: r}, nil
}

func (r *FakeRepository) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	if r.GetGCCandidatesErr != nil {
		return nil, r.GetGCCandidatesErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.GCCandidates != nil {
		return r.GCCandidates, nil
	}
	out := make([]*types.Sandbox, 0, len(r.Sandboxes))
	for _, s := range r.Sandboxes {
		out = append(out, s)
	}
	return out, nil
}

// Compile-time interface guard.
var _ repository.Repository = (*FakeRepository)(nil)

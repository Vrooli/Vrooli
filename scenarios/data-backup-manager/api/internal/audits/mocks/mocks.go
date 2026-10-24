// Package mocks holds co-located test doubles for the audits domain seams, so
// the audit orchestration is unit-testable against fakes without standing up
// the engine, sources, or a real database.
package mocks

import (
	"context"
	"fmt"
	"sync"

	"data-backup-manager/internal/audits"
)

// FakeTargetLookup returns canned target specs keyed by target id.
type FakeTargetLookup struct {
	Targets map[string]audits.TargetForAudit
	Err     error
}

func (f *FakeTargetLookup) TargetForAudit(_ context.Context, targetID string) (audits.TargetForAudit, error) {
	if f.Err != nil {
		return audits.TargetForAudit{}, f.Err
	}
	t, ok := f.Targets[targetID]
	if !ok {
		return audits.TargetForAudit{}, &notFound{"target", targetID}
	}
	return t, nil
}

// FakeDestinationLookup returns canned destination specs keyed by destination id.
type FakeDestinationLookup struct {
	Destinations map[string]audits.DestinationForAudit
	Err          error
}

func (f *FakeDestinationLookup) DestinationForAudit(_ context.Context, destID string) (audits.DestinationForAudit, error) {
	if f.Err != nil {
		return audits.DestinationForAudit{}, f.Err
	}
	d, ok := f.Destinations[destID]
	if !ok {
		return audits.DestinationForAudit{}, &notFound{"destination", destID}
	}
	return d, nil
}

// SyncExecutor runs each submitted audit job inline and to completion on the
// caller's goroutine, so a service test can request an audit and immediately
// observe the terminal record without polling. It mirrors the production async
// contract (Bind once, then Submit) but collapses the asynchrony.
type SyncExecutor struct {
	baseCtx context.Context
	run     audits.AuditFunc
}

// NewSyncExecutor constructs an inline executor for tests.
func NewSyncExecutor() *SyncExecutor { return &SyncExecutor{} }

func (e *SyncExecutor) Bind(baseCtx context.Context, run audits.AuditFunc) {
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	e.baseCtx = baseCtx
	e.run = run
}

func (e *SyncExecutor) Submit(job audits.AuditJob) {
	if e.run != nil {
		e.run(e.baseCtx, job)
	}
}

func (e *SyncExecutor) Shutdown(context.Context) error { return nil }

// InMemoryRepository is an in-memory audits.Repository for service tests.
type InMemoryRepository struct {
	mu      sync.Mutex
	byID    map[string]audits.Audit
	order   []string
	seq     int
	CreErr  error
	FinErr  error
	StatErr error
}

// NewInMemoryRepository constructs an empty in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{byID: map[string]audits.Audit{}}
}

func (r *InMemoryRepository) CreateAudit(_ context.Context, a audits.Audit) (audits.Audit, error) {
	if r.CreErr != nil {
		return audits.Audit{}, r.CreErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if a.ID == "" {
		r.seq++
		a.ID = idForSeq(r.seq)
	}
	if a.Status == "" {
		a.Status = audits.AuditRequested
	}
	r.byID[a.ID] = a
	r.order = append(r.order, a.ID)
	return a, nil
}

func (r *InMemoryRepository) UpdateAuditStatus(_ context.Context, id string, status audits.AuditStatus) error {
	if r.StatErr != nil {
		return r.StatErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok {
		return audits.ErrAuditNotFound{ID: id}
	}
	a.Status = status
	r.byID[id] = a
	return nil
}

func (r *InMemoryRepository) FinishAudit(_ context.Context, a audits.Audit) error {
	if r.FinErr != nil {
		return r.FinErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[a.ID]; !ok {
		return audits.ErrAuditNotFound{ID: a.ID}
	}
	r.byID[a.ID] = a
	return nil
}

func (r *InMemoryRepository) ListNonTerminalAudits(_ context.Context) ([]audits.Audit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []audits.Audit
	for _, id := range r.order {
		a := r.byID[id]
		if a.Status == audits.AuditRequested || a.Status == audits.AuditRunning {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *InMemoryRepository) GetAudit(_ context.Context, id string) (audits.Audit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok {
		return audits.Audit{}, audits.ErrAuditNotFound{ID: id}
	}
	return a, nil
}

func (r *InMemoryRepository) ListAudits(_ context.Context, targetID string, limit int) ([]audits.Audit, error) {
	if limit <= 0 {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []audits.Audit
	for i := len(r.order) - 1; i >= 0; i-- {
		a := r.byID[r.order[i]]
		if targetID != "" && a.TargetID != targetID {
			continue
		}
		out = append(out, a)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// FakeSQLiteChecker returns a canned SqliteInventory per relative path, or a
// default ok result so a checker is always present for compare tests.
type FakeSQLiteChecker struct {
	ByRel   map[string]audits.SqliteInventory
	Default audits.SqliteInventory
}

func (f *FakeSQLiteChecker) Check(_ context.Context, _ string, rel string) audits.SqliteInventory {
	if inv, ok := f.ByRel[rel]; ok {
		inv.Path = rel
		return inv
	}
	d := f.Default
	d.Path = rel
	if d.IntegrityStatus == "" {
		d.IntegrityStatus = "ok"
	}
	return d
}

// FakeService satisfies audits.Service for handler tests.
type FakeService struct {
	RunOut  audits.Audit
	RunErr  error
	GetOut  audits.Audit
	GetErr  error
	ListOut []audits.Audit
	ListErr error

	RunCalls []RunCall
}

// RunCall records one RunSnapshotAudit invocation.
type RunCall struct {
	TargetID, DestinationID, SnapshotID string
	IncludeContentHash, IncludeSQLite   bool
}

func (f *FakeService) RunSnapshotAudit(_ context.Context, targetID, destinationID, snapshotID string, includeContentHash, includeSQLiteCheck bool) (audits.Audit, error) {
	f.RunCalls = append(f.RunCalls, RunCall{targetID, destinationID, snapshotID, includeContentHash, includeSQLiteCheck})
	if f.RunErr != nil {
		return audits.Audit{}, f.RunErr
	}
	return f.RunOut, nil
}

func (f *FakeService) GetAudit(_ context.Context, _ string) (audits.Audit, error) {
	if f.GetErr != nil {
		return audits.Audit{}, f.GetErr
	}
	return f.GetOut, nil
}

func (f *FakeService) ListAudits(_ context.Context, _ string, _ int) ([]audits.Audit, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.ListOut, nil
}

func (f *FakeService) Reconcile(_ context.Context) error { return nil }

func (f *FakeService) Shutdown(_ context.Context) error { return nil }

// idForSeq returns a short deterministic id (audit-0001 style) for tests.
func idForSeq(n int) string { return fmt.Sprintf("audit-%04d", n) }

// Compile-time guarantees.
var (
	_ audits.TargetLookup      = (*FakeTargetLookup)(nil)
	_ audits.DestinationLookup = (*FakeDestinationLookup)(nil)
	_ audits.Executor          = (*SyncExecutor)(nil)
	_ audits.Repository        = (*InMemoryRepository)(nil)
	_ audits.SQLiteChecker     = (*FakeSQLiteChecker)(nil)
	_ audits.Service           = (*FakeService)(nil)
)

type notFound struct {
	kind string
	id   string
}

func (e *notFound) Error() string { return e.kind + " " + e.id + " not found" }

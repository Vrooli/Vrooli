package indexcontrol

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"code-facts/internal/catalog"
	_ "modernc.org/sqlite"
)

type mutableClock struct{ now time.Time }

func (clock *mutableClock) Now() time.Time { return clock.now }

func openControlDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:control-%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repository := catalog.NewSQLiteRepository(db, controlCatalogClock{})
	if err := repository.BeginGeneration(context.Background(), catalog.Generation{ID: "g1", Policy: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpsertFiles(context.Background(), "g1", []catalog.SourceFile{{ID: "file:one", Path: "one.go", Language: "go", Role: catalog.RoleImplementation, Scope: "repo", Authority: "authoritative", Owner: "test", Hash: "sha256:one", Size: 10, Searchable: true}}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CompleteGeneration(context.Background(), "g1", "source", "descriptor"); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}
	return db
}

type controlCatalogClock struct{}

func (controlCatalogClock) Now() time.Time { return time.Unix(100, 0) }

func TestSQLiteJobStorePersistsCancellationAndRestartEvidence(t *testing.T) {
	db := openControlDB(t)
	store := NewSQLiteJobStore(db)
	now := time.Unix(200, 0)
	job := Job{ID: "job-1", Kind: "reconcile", State: "running", Generation: "g1", Cursor: "c1", Progress: 2, Total: 9, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	recovered, err := store.RecoverInterrupted(context.Background(), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 1 || recovered[0].State != "interrupted" || recovered[0].Cursor != "c1" || recovered[0].Error == "" {
		t.Fatalf("restart evidence mismatch: %+v", recovered)
	}
	if err := store.RequestCancel(context.Background(), job.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), job.ID)
	if err != nil || got.State != "cancellation_requested" || !got.CancellationRequested {
		t.Fatalf("durable cancellation mismatch: %+v err=%v", got, err)
	}
}

func TestSQLiteStatusReaderReportsDurableGenerationCountsAndJobs(t *testing.T) {
	db := openControlDB(t)
	jobs := NewSQLiteJobStore(db)
	now := time.Unix(200, 0)
	if err := jobs.Create(context.Background(), Job{ID: "status-job", Kind: "reconcile", State: "running", Generation: "g1", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	status, err := NewSQLiteStatusReader(db, jobs).Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.ActiveGeneration != "g1" || status.State != "updating" || status.SourceFiles != 1 || status.StorageBytes <= 0 || len(status.ActiveJobs) != 1 || status.LastReconcileOutcome != "running" {
		t.Fatalf("truthful durable status mismatch: %+v", status)
	}
}

func TestDebouncerDeduplicatesAndBoundsFreshnessLatency(t *testing.T) {
	start := time.Unix(100, 0)
	debouncer := NewDebouncer(time.Second, 10*time.Second)
	debouncer.Add(start, Change{Path: "service.go", Operation: "upsert", Hash: "old"})
	for second := 1; second <= 12; second++ {
		debouncer.Add(start.Add(time.Duration(second)*time.Second), Change{Path: "service.go", Operation: "upsert", Hash: fmt.Sprintf("hash-%d", second)})
	}
	if got := debouncer.Ready(start.Add(9*time.Second), 10); len(got) != 0 {
		t.Fatalf("event escaped before maximum debounce: %+v", got)
	}
	got := debouncer.Ready(start.Add(10*time.Second), 10)
	if len(got) != 1 || got[0].Hash != "hash-12" {
		t.Fatalf("debounced event mismatch: %+v", got)
	}
	if !AuditDue(start, start.Add(5*time.Minute)) || AuditDue(start, start.Add(5*time.Minute-time.Nanosecond)) {
		t.Fatal("periodic audit must run exactly at the five-minute repair bound")
	}
}

type memoryJobs struct {
	mu   sync.Mutex
	jobs map[string]Job
}

func (store *memoryJobs) Create(_ context.Context, job Job) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.jobs == nil {
		store.jobs = map[string]Job{}
	}
	store.jobs[job.ID] = job
	return nil
}

func (store *memoryJobs) Update(_ context.Context, job Job) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.jobs[job.ID] = job
	return nil
}

func (store *memoryJobs) Get(_ context.Context, id string) (Job, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	job, ok := store.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return job, nil
}
func (store *memoryJobs) ListActive(context.Context) ([]Job, error) { return nil, nil }
func (store *memoryJobs) RequestCancel(_ context.Context, id string, _ time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	job := store.jobs[id]
	job.CancellationRequested = true
	job.State = "cancellation_requested"
	store.jobs[id] = job
	return nil
}

func (store *memoryJobs) RecoverInterrupted(context.Context, time.Time) ([]Job, error) {
	return nil, nil
}

type pageSource struct {
	pages []ChangeBatch
	calls int
}

func (source *pageSource) Changes(_ context.Context, _ string, limit int) (ChangeBatch, error) {
	source.calls++
	if len(source.pages) == 0 {
		return ChangeBatch{Done: true}, nil
	}
	page := source.pages[0]
	source.pages = source.pages[1:]
	if len(page.Changes) > limit {
		return ChangeBatch{}, fmt.Errorf("test source limit exceeded")
	}
	return page, nil
}

type recordingProcessor struct {
	batches [][]Change
	err     error
}

func (processor *recordingProcessor) Apply(_ context.Context, _ string, changes []Change) (int64, error) {
	processor.batches = append(processor.batches, append([]Change(nil), changes...))
	return int64(len(changes)), processor.err
}

type fakeLifecycle struct {
	active      string
	activateErr error
	begun       []string
	completed   []string
}

func (lifecycle *fakeLifecycle) Active(context.Context) (string, error) { return lifecycle.active, nil }
func (lifecycle *fakeLifecycle) BeginShadow(_ context.Context, generation string) error {
	lifecycle.begun = append(lifecycle.begun, generation)
	return nil
}

func (lifecycle *fakeLifecycle) CompleteShadow(_ context.Context, generation string) error {
	lifecycle.completed = append(lifecycle.completed, generation)
	return nil
}

func (lifecycle *fakeLifecycle) Activate(_ context.Context, generation string) error {
	if lifecycle.activateErr != nil {
		return lifecycle.activateErr
	}
	lifecycle.active = generation
	return nil
}

func (lifecycle *fakeLifecycle) Rollback(_ context.Context, generation string) error {
	lifecycle.active = generation
	return nil
}

type fakeAliases struct{ calls []string }

func (aliases *fakeAliases) Promote(_ context.Context, generation string) error {
	aliases.calls = append(aliases.calls, "promote:"+generation)
	return nil
}

func (aliases *fakeAliases) Rollback(_ context.Context, generation string) error {
	aliases.calls = append(aliases.calls, "rollback:"+generation)
	return nil
}

type fakeValidator struct{ err error }

func (validator fakeValidator) Validate(context.Context, string) error { return validator.err }

type fakePromotions struct{ states []string }

func (promotions *fakePromotions) Prepare(_ context.Context, _, _, _ string, _ time.Time) error {
	promotions.states = append(promotions.states, "prepared")
	return nil
}

func (promotions *fakePromotions) Transition(_ context.Context, _, state, _ string, _ time.Time) error {
	promotions.states = append(promotions.states, state)
	return nil
}

func testCoordinator(source *pageSource, processor *recordingProcessor) (*Coordinator, *memoryJobs, *fakeLifecycle, *fakeAliases, *fakePromotions) {
	jobs := &memoryJobs{}
	lifecycle := &fakeLifecycle{active: "g1"}
	aliases := &fakeAliases{}
	promotions := &fakePromotions{}
	return &Coordinator{Jobs: jobs, Source: source, Processor: processor, Lifecycle: lifecycle, Aliases: aliases, Validator: fakeValidator{}, Promotions: promotions, Clock: &mutableClock{now: time.Unix(300, 0)}, BatchSize: 2}, jobs, lifecycle, aliases, promotions
}

func TestCoordinatorBoundsDeduplicatesAndCompletesShadow(t *testing.T) {
	source := &pageSource{pages: []ChangeBatch{
		{Cursor: "", NextCursor: "c1", Changes: []Change{{Path: "b.go", Hash: "old"}, {Path: "b.go", Hash: "new"}}, Done: false},
		{Cursor: "c1", NextCursor: "c2", Changes: []Change{{Path: "a.go", Operation: "delete"}}, Done: true},
	}}
	processor := &recordingProcessor{}
	coordinator, _, lifecycle, _, _ := testCoordinator(source, processor)
	job, err := coordinator.StartShadow(context.Background(), "g2")
	if err != nil {
		t.Fatal(err)
	}
	if job.State != "succeeded" || job.Progress != 2 || len(processor.batches) != 2 || processor.batches[0][0].Hash != "new" {
		t.Fatalf("bounded reconcile mismatch: job=%+v batches=%+v", job, processor.batches)
	}
	if !reflect.DeepEqual(lifecycle.begun, []string{"g2"}) || !reflect.DeepEqual(lifecycle.completed, []string{"g2"}) {
		t.Fatalf("shadow lifecycle mismatch: %+v", lifecycle)
	}
}

func TestCoordinatorNoChangeIsBoundedAndPromotionFailurePreservesActive(t *testing.T) {
	source := &pageSource{pages: []ChangeBatch{{Done: true}}}
	processor := &recordingProcessor{}
	coordinator, _, lifecycle, aliases, promotions := testCoordinator(source, processor)
	job, err := coordinator.Reconcile(context.Background(), "")
	if err != nil || job.State != "succeeded" || source.calls != 1 || len(processor.batches) != 0 {
		t.Fatalf("no-change reconcile mismatch: job=%+v calls=%d batches=%d err=%v", job, source.calls, len(processor.batches), err)
	}
	lifecycle.activateErr = errors.New("catalog promotion failed")
	if err := coordinator.Promote(context.Background(), "g2"); err == nil {
		t.Fatal("expected promotion failure")
	}
	if lifecycle.active != "g1" || !reflect.DeepEqual(aliases.calls, []string{"promote:g2", "rollback:g1"}) || !reflect.DeepEqual(promotions.states, []string{"prepared", "alias_promoted", "rolled_back"}) {
		t.Fatalf("promotion compensation mismatch: active=%s aliases=%v states=%v", lifecycle.active, aliases.calls, promotions.states)
	}
}

func TestCoordinatorCancellationAndRollbackAreTruthful(t *testing.T) {
	source := &pageSource{pages: []ChangeBatch{{Done: true}}}
	coordinator, jobs, lifecycle, aliases, _ := testCoordinator(source, &recordingProcessor{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	job, err := coordinator.Reconcile(ctx, "g1")
	if !errors.Is(err, context.Canceled) || job.State != "cancelled" {
		t.Fatalf("cancelled job mismatch: %+v err=%v", job, err)
	}
	if stored := jobs.jobs[job.ID]; stored.State != "cancelled" || !stored.CancellationRequested {
		t.Fatalf("cancelled job was not durable: %+v", stored)
	}
	lifecycle.active = "g2"
	if err := coordinator.Rollback(context.Background(), "g1"); err != nil {
		t.Fatal(err)
	}
	if lifecycle.active != "g1" || !reflect.DeepEqual(aliases.calls, []string{"rollback:g1"}) {
		t.Fatalf("rollback mismatch: active=%s aliases=%v", lifecycle.active, aliases.calls)
	}
}

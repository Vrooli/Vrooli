package maintenance

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"vrooli-memory/internal/harness"
	"vrooli-memory/internal/testutil/mocks"
)

type fakeImporter struct {
	runtimes []string
	order    *[]string
	mu       sync.Mutex
	started  chan struct{}
	release  chan struct{}
	err      map[string]error
}

func (f *fakeImporter) Runtimes() []string { return f.runtimes }
func (f *fakeImporter) Import(_ context.Context, runtime string, _ bool) (harness.ImportResult, error) {
	if f.order != nil {
		f.mu.Lock()
		*f.order = append(*f.order, "import:"+runtime)
		f.mu.Unlock()
	}
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
		<-f.release
	}
	return harness.ImportResult{Runtime: runtime, Seen: 1}, f.err[runtime]
}

type fakeProjector struct {
	runtimes []string
	order    *[]string
	err      map[string]error
}

func (f *fakeProjector) Runtimes() []string { return f.runtimes }
func (f *fakeProjector) Project(_ context.Context, runtime string, _ bool) (harness.ProjectionResult, error) {
	if f.order != nil {
		*f.order = append(*f.order, "project:"+runtime)
	}
	return harness.ProjectionResult{Path: runtime}, f.err[runtime]
}

type memoryStore struct {
	mu          sync.Mutex
	runs        []Run
	outcomes    map[string]map[string]Outcome
	compactions map[string]Compaction
}

func newMemoryStore() *memoryStore {
	return &memoryStore{outcomes: map[string]map[string]Outcome{}, compactions: map[string]Compaction{}}
}

func (s *memoryStore) PutCompaction(_ context.Context, id string, c Compaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.compactions[id] = c
	return nil
}

func (s *memoryStore) Begin(_ context.Context, run Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs = append(s.runs, run)
	s.outcomes[run.ID] = map[string]Outcome{}
	return nil
}

func (s *memoryStore) PutOutcome(_ context.Context, id string, o Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outcomes[id][o.Runtime] = o
	return nil
}

func (s *memoryStore) Complete(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.runs {
		if s.runs[i].ID == id {
			s.runs[i].CompletedAt = at
		}
	}
	return nil
}

func (s *memoryStore) Latest(_ context.Context) (Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.runs) == 0 {
		return Run{}, errors.New("no runs")
	}
	run := s.runs[len(s.runs)-1]
	for _, o := range s.outcomes[run.ID] {
		run.Outcomes = append(run.Outcomes, o)
	}
	return run, nil
}

func TestRunOnceImportsEveryRuntimeBeforeProjection(t *testing.T) {
	ctx := context.Background()
	order := []string{}
	store := newMemoryStore()
	service := NewService(store, &fakeImporter{runtimes: []string{"b", "a"}, order: &order, err: map[string]error{}}, &fakeProjector{runtimes: []string{"a"}, order: &order, err: map[string]error{}}, mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)), 0)
	run, err := service.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, run)
	require.ElementsMatch(t, []string{"import:a", "import:b"}, order[:2])
	require.Equal(t, "project:a", order[2])
	got, err := service.Latest(ctx)
	require.NoError(t, err)
	require.Len(t, got.Outcomes, 2)
}

func TestRunOnceSkipsOverlap(t *testing.T) {
	started, release := make(chan struct{}, 1), make(chan struct{})
	store := newMemoryStore()
	service := NewService(store, &fakeImporter{runtimes: []string{"a"}, started: started, release: release, err: map[string]error{}}, &fakeProjector{runtimes: []string{"a"}, err: map[string]error{}}, mocks.NewFakeClock(time.Time{}), 0)
	done := make(chan struct{})
	go func() { _, _ = service.RunOnce(context.Background()); close(done) }()
	<-started
	second, err := service.RunOnce(context.Background())
	require.NoError(t, err)
	require.False(t, second)
	close(release)
	<-done
	got, err := service.Latest(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)
}

func TestRunOnceRecordsFailureAndContinues(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store, &fakeImporter{runtimes: []string{"broken", "healthy"}, err: map[string]error{"broken": errors.New("store missing")}}, &fakeProjector{runtimes: []string{"healthy"}, err: map[string]error{}}, mocks.NewFakeClock(time.Time{}), 0)
	_, err := service.RunOnce(context.Background())
	require.NoError(t, err)
	run, err := service.Latest(context.Background())
	require.NoError(t, err)
	byRuntime := map[string]Outcome{}
	for _, o := range run.Outcomes {
		byRuntime[o.Runtime] = o
	}
	require.Equal(t, "failed", byRuntime["broken"].ImportStatus)
	require.Equal(t, "completed", byRuntime["healthy"].ImportStatus)
	require.Equal(t, "completed", byRuntime["healthy"].ProjectionStatus)
}

func TestIntervalFromEnv(t *testing.T) {
	d, err := IntervalFromEnv(func(string) (string, bool) { return "15m", true })
	require.NoError(t, err)
	require.Equal(t, 15*time.Minute, d)
	d, err = IntervalFromEnv(func(string) (string, bool) { return "0", true })
	require.NoError(t, err)
	require.Zero(t, d)
	_, err = IntervalFromEnv(func(string) (string, bool) { return "nonsense", true })
	require.Error(t, err)
}

type fakeCompactor struct {
	calls  []int
	result CompactionResult
	err    error
}

func (f *fakeCompactor) RunBounded(_ context.Context, limit int) (CompactionResult, error) {
	f.calls = append(f.calls, limit)
	return f.result, f.err
}

func TestRunOnceCompactsAfterProjectionSoAmbientMemoryIsNeverBlocked(t *testing.T) {
	order := []string{}
	store := newMemoryStore()
	compactor := &fakeCompactor{result: CompactionResult{CompactedCount: 3, EligibleFrontierBefore: 100, EligibleFrontierAfter: 97}}
	service := NewService(store,
		&fakeImporter{runtimes: []string{"a"}, order: &order, err: map[string]error{}},
		&fakeProjector{runtimes: []string{"a"}, order: &order, err: map[string]error{}},
		mocks.NewFakeClock(time.Time{}), 0).WithCompaction(compactor, 25)
	_, err := service.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, []int{25}, compactor.calls, "compaction runs once per pass, at the configured limit")
	require.Equal(t, []string{"import:a", "project:a"}, order,
		"projection completes before the backlog-scaled compaction pass starts")

	run, err := service.Latest(context.Background())
	require.NoError(t, err)
	require.Equal(t, "completed", store.compactions[run.ID].Status)
	require.Equal(t, 3, store.compactions[run.ID].Compacted)
	require.Equal(t, 100, store.compactions[run.ID].FrontierBefore)
	require.Equal(t, 97, store.compactions[run.ID].FrontierAfter)
}

func TestCompactionFailureDoesNotStopProjection(t *testing.T) {
	order := []string{}
	store := newMemoryStore()
	compactor := &fakeCompactor{err: errors.New("provider unavailable")}
	service := NewService(store,
		&fakeImporter{runtimes: []string{"a"}, order: &order, err: map[string]error{}},
		&fakeProjector{runtimes: []string{"a"}, order: &order, err: map[string]error{}},
		mocks.NewFakeClock(time.Time{}), 0).WithCompaction(compactor, 10)
	_, err := service.RunOnce(context.Background())
	require.NoError(t, err, "a compaction failure is recorded, not propagated")

	run, err := service.Latest(context.Background())
	require.NoError(t, err)
	require.Equal(t, "failed", store.compactions[run.ID].Status)
	require.Contains(t, store.compactions[run.ID].Error, "provider unavailable")
	require.Contains(t, order, "project:a", "ambient memory still refreshes when the canopy stalls")
}

func TestCompactionNotConfiguredWhenLimitIsZero(t *testing.T) {
	store := newMemoryStore()
	compactor := &fakeCompactor{}
	service := NewService(store,
		&fakeImporter{runtimes: []string{"a"}, err: map[string]error{}},
		&fakeProjector{runtimes: []string{"a"}, err: map[string]error{}},
		mocks.NewFakeClock(time.Time{}), 0).WithCompaction(compactor, 0)
	_, err := service.RunOnce(context.Background())
	require.NoError(t, err)
	require.Empty(t, compactor.calls)
	run, err := service.Latest(context.Background())
	require.NoError(t, err)
	require.Equal(t, "not_configured", store.compactions[run.ID].Status)
}

func TestCompactLimitFromEnv(t *testing.T) {
	n, err := CompactLimitFromEnv(func(string) (string, bool) { return "", false })
	require.NoError(t, err)
	require.Equal(t, DefaultCompactLimit, n)
	n, err = CompactLimitFromEnv(func(string) (string, bool) { return "0", true })
	require.NoError(t, err)
	require.Zero(t, n)
	_, err = CompactLimitFromEnv(func(string) (string, bool) { return "many", true })
	require.Error(t, err)
}

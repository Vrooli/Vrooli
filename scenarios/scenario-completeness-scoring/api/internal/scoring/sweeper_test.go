package scoring

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSweeperDigestSkipAvoidsScoring(t *testing.T) {
	repo := newFakeSnapshotRepo()
	repo.snapshots["alpha"] = []Snapshot{
		snapshotAt("alpha", "td:same", 70, time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)),
	}
	scorer := &fakeSweepScorer{}
	sweeper := newTestSweeper(t, repo, scorer, fakeDigester{digests: map[string]string{"alpha": "td:same"}}, []ScenarioRef{{Name: "alpha", Root: "/fleet/alpha"}})

	report, err := sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if report.Scanned != 1 || report.Skipped != 1 || report.Scored != 0 || report.Persisted != 0 || report.Failed != 0 {
		t.Fatalf("report = %+v, want skip-only", report)
	}
	if scorer.calls != 0 {
		t.Fatalf("scorer calls = %d, want 0", scorer.calls)
	}
}

func TestSweeperChangedDigestPersistsSnapshot(t *testing.T) {
	repo := newFakeSnapshotRepo()
	scorer := &fakeSweepScorer{results: map[string]Result{
		"alpha": {
			Scenario: "alpha",
			Category: "utility",
			Composite: Composite{
				Score:               82,
				Classification:      "mostly_complete",
				ClassificationLabel: "Mostly complete",
			},
			Maturity:     Maturity{WorkingRung: "R3"},
			CalculatedAt: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
		},
	}}
	sweeper := newTestSweeper(t, repo, scorer, fakeDigester{digests: map[string]string{"alpha": "td:new"}}, []ScenarioRef{{Name: "alpha", Root: "/fleet/alpha"}})

	report, err := sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if report.Scored != 1 || report.Persisted != 1 || report.Failed != 0 {
		t.Fatalf("report = %+v, want one persisted score", report)
	}
	got, ok, err := repo.LatestFor(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("LatestFor() error = %v", err)
	}
	if !ok {
		t.Fatal("LatestFor() ok = false, want true")
	}
	if got.Digest != "td:new" || got.Composite != 82 || got.WorkingRung != "R3" {
		t.Fatalf("snapshot = %+v, want converted result", got)
	}
	if got.Importance != nil {
		t.Fatalf("importance = %v, want nil when score enrichment is absent", *got.Importance)
	}
	if !strings.Contains(got.BreakdownJSON, "mostly_complete") {
		t.Fatalf("breakdown_json = %q, want composite payload", got.BreakdownJSON)
	}
}

func TestSweeperDigestFailureContinuesFleet(t *testing.T) {
	repo := newFakeSnapshotRepo()
	scorer := &fakeSweepScorer{results: map[string]Result{
		"beta": {
			Scenario:     "beta",
			Category:     "utility",
			Composite:    Composite{Score: 55, Classification: "partial"},
			CalculatedAt: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
		},
	}}
	digester := fakeDigester{
		digests: map[string]string{"beta": "td:beta"},
		errs:    map[string]error{"alpha": errors.New("boom")},
	}
	sweeper := newTestSweeper(t, repo, scorer, digester, []ScenarioRef{
		{Name: "alpha", Root: "/fleet/alpha"},
		{Name: "beta", Root: "/fleet/beta"},
	})

	report, err := sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if report.Scanned != 2 || report.Failed != 1 || report.Persisted != 1 {
		t.Fatalf("report = %+v, want one failure and one persisted snapshot", report)
	}
}

func TestSweeperCancellationStopsDispatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := newFakeSnapshotRepo()
	scorer := &fakeSweepScorer{}
	sweeper := newTestSweeper(t, repo, scorer, fakeDigester{digests: map[string]string{"alpha": "td:new"}}, []ScenarioRef{{Name: "alpha", Root: "/fleet/alpha"}})

	report, err := sweeper.RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunOnce() error = %v, want context.Canceled", err)
	}
	if report.Scored != 0 || report.Persisted != 0 {
		t.Fatalf("report = %+v, want no score work after cancellation", report)
	}
}

func TestSweeperSingleFlightRejectsOverlap(t *testing.T) {
	repo := newFakeSnapshotRepo()
	block := make(chan struct{})
	scorer := &fakeSweepScorer{
		block: block,
		results: map[string]Result{
			"alpha": {
				Scenario:     "alpha",
				Category:     "utility",
				Composite:    Composite{Score: 75, Classification: "mostly_complete"},
				CalculatedAt: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	sweeper := newTestSweeper(t, repo, scorer, fakeDigester{digests: map[string]string{"alpha": "td:new"}}, []ScenarioRef{{Name: "alpha", Root: "/fleet/alpha"}})

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		_, err := sweeper.RunOnce(context.Background())
		done <- err
	}()
	<-started
	for scorer.callCount() == 0 {
		time.Sleep(time.Millisecond)
	}

	if _, err := sweeper.RunOnce(context.Background()); !errors.Is(err, ErrSweepInProgress) {
		t.Fatalf("overlapping RunOnce() error = %v, want ErrSweepInProgress", err)
	}
	close(block)
	if err := <-done; err != nil {
		t.Fatalf("first RunOnce() error = %v", err)
	}
}

func TestSweeperRunLoopHonorsInitialJitter(t *testing.T) {
	repo := newFakeSnapshotRepo()
	scorer := &fakeSweepScorer{}
	sweeper := newTestSweeper(t, repo, scorer, fakeDigester{digests: map[string]string{"alpha": "td:new"}}, []ScenarioRef{{Name: "alpha", Root: "/fleet/alpha"}})
	sweeper.cfg.InitialJitter = 40 * time.Millisecond
	sweeper.cfg.Interval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		sweeper.RunLoop(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	if calls := scorer.callCount(); calls != 0 {
		t.Fatalf("scorer calls before initial jitter = %d, want 0", calls)
	}

	deadline := time.After(500 * time.Millisecond)
	for scorer.callCount() == 0 {
		select {
		case <-deadline:
			t.Fatal("RunLoop did not execute after initial jitter")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	<-done
}

func newTestSweeper(t *testing.T, repo SnapshotRepository, scorer SweepScorer, digester DigestComputer, scenarios []ScenarioRef) *Sweeper {
	t.Helper()
	sweeper, err := NewSweeper(SweeperConfig{
		ScenariosRoot: "/fleet",
		Repository:    repo,
		Scorer:        scorer,
		Lister:        fakeScenarioLister(scenarios),
		Digester:      digester,
		Now: func() time.Time {
			return time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
		},
		Logger:      log.New(testWriter{t: t}, "", 0),
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("NewSweeper() error = %v", err)
	}
	return sweeper
}

type fakeScenarioLister []ScenarioRef

var _ ScenarioLister = fakeScenarioLister{}

func (f fakeScenarioLister) ListScenarios(string) ([]ScenarioRef, error) {
	out := make([]ScenarioRef, len(f))
	copy(out, f)
	return out, nil
}

type fakeDigester struct {
	digests map[string]string
	errs    map[string]error
}

var _ DigestComputer = fakeDigester{}

func (f fakeDigester) ComputeDigest(root string) (string, error) {
	name := root[strings.LastIndex(root, "/")+1:]
	if f.errs != nil {
		if err := f.errs[name]; err != nil {
			return "", err
		}
	}
	return f.digests[name], nil
}

type fakeSweepScorer struct {
	mu      sync.Mutex
	calls   int
	results map[string]Result
	errs    map[string]error
	block   <-chan struct{}
}

var _ SweepScorer = (*fakeSweepScorer)(nil)

func (f *fakeSweepScorer) GetScore(scenario string) (Result, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.block != nil {
		<-f.block
	}
	if err := f.errs[scenario]; err != nil {
		return Result{}, err
	}
	if res, ok := f.results[scenario]; ok {
		return res, nil
	}
	return Result{Scenario: scenario, Category: "utility", Composite: Composite{Score: 1, Classification: "test"}}, nil
}

func (f *fakeSweepScorer) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeSnapshotRepo struct {
	mu        sync.Mutex
	snapshots map[string][]Snapshot
}

var _ SnapshotRepository = (*fakeSnapshotRepo)(nil)

func newFakeSnapshotRepo() *fakeSnapshotRepo {
	return &fakeSnapshotRepo{snapshots: map[string][]Snapshot{}}
}

func (f *fakeSnapshotRepo) LatestFor(_ context.Context, scenario string) (Snapshot, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	snaps := f.snapshots[scenario]
	if len(snaps) == 0 {
		return Snapshot{}, false, nil
	}
	return snaps[len(snaps)-1], true, nil
}

func (f *fakeSnapshotRepo) LatestDifferingDigest(_ context.Context, scenario, digest string) (Snapshot, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.snapshots[scenario]) - 1; i >= 0; i-- {
		if f.snapshots[scenario][i].Digest != digest {
			return f.snapshots[scenario][i], true, nil
		}
	}
	return Snapshot{}, false, nil
}

func (f *fakeSnapshotRepo) SeriesFor(_ context.Context, q TrendQuery) ([]Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]Snapshot(nil), f.snapshots[q.Scenario]...)
	return out, nil
}

func (f *fakeSnapshotRepo) UpsertSnapshot(_ context.Context, snap Snapshot) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.snapshots[snap.Scenario] {
		if existing.Digest == snap.Digest {
			return false, nil
		}
	}
	f.snapshots[snap.Scenario] = append(f.snapshots[snap.Scenario], snap)
	return true, nil
}

func (f *fakeSnapshotRepo) ListPage(context.Context, ListQuery) (ListResult, error) {
	return ListResult{}, nil
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}

package evalsched

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	internaleval "search-hub/internal/eval"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

type fakeSuites struct {
	mu     sync.Mutex
	suites []*evalv1.EvalSuite
}

func (s *fakeSuites) ListSuites(_ context.Context, _ internaleval.ListSuitesFilter) ([]*evalv1.EvalSuite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*evalv1.EvalSuite(nil), s.suites...), nil
}

type recordedRun struct {
	suite string
	tag   string
}

type fakeStore struct {
	mu          sync.Mutex
	runs        []recordedRun
	validations []string
}

func (s *fakeStore) AppendRun(_ context.Context, run *evalv1.EvalRun) error {
	s.mu.Lock()
	s.runs = append(s.runs, recordedRun{suite: run.GetSuiteId(), tag: run.GetTag()})
	s.mu.Unlock()
	return nil
}

func (s *fakeStore) AppendCorpusValidation(_ context.Context, suiteID string, _ *evalv1.ValidateCorpusResponse, _ time.Time) error {
	s.mu.Lock()
	s.validations = append(s.validations, suiteID)
	s.mu.Unlock()
	return nil
}

func (s *fakeStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.runs)
}

func (s *fakeStore) validationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.validations)
}

type fakeValidator struct {
	mu    sync.Mutex
	calls []string
}

func (v *fakeValidator) ValidateCorpus(_ context.Context, suite *evalv1.EvalSuite, _ int32) (*evalv1.ValidateCorpusResponse, error) {
	v.mu.Lock()
	v.calls = append(v.calls, suite.GetSuiteId())
	v.mu.Unlock()
	return &evalv1.ValidateCorpusResponse{Rollup: &evalv1.CorpusValidationRollup{Live: 1}}, nil
}

func (v *fakeValidator) count() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.calls)
}

type fakeRunner struct {
	name       string
	mu         sync.Mutex
	calls      []string
	active     int
	maxActive  int
	errFor     map[string]bool
	onStart    func(string)
	onComplete func()
}

func (r *fakeRunner) stats() (calls, maxActive int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls), r.maxActive
}

func (r *fakeRunner) Run(_ context.Context, suite *evalv1.EvalSuite, tag string, _ int32) (*evalv1.EvalRun, error) {
	id := suite.GetSuiteId()
	r.mu.Lock()
	r.calls = append(r.calls, id)
	if r.errFor[id] {
		r.mu.Unlock()
		return nil, context.DeadlineExceeded
	}
	r.active++
	if r.active > r.maxActive {
		r.maxActive = r.active
	}
	onStart := r.onStart
	onComplete := r.onComplete
	r.mu.Unlock()
	if onStart != nil {
		onStart(id)
	}
	if onComplete != nil {
		onComplete()
	}
	r.mu.Lock()
	r.active--
	r.mu.Unlock()
	return &evalv1.EvalRun{RunId: r.name + "-" + id, SuiteId: id, Tag: tag, Tier: r.name}, nil
}

func testScheduler(t *testing.T, suites []*evalv1.EvalSuite, concurrency int) (*Scheduler, *fakeSuites, *fakeRunner, *fakeRunner, *fakeStore, *fakeClock) {
	t.Helper()
	clk := &fakeClock{now: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)}
	source := &fakeSuites{suites: suites}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	store := &fakeStore{}
	scheduler := New(clk, source, direct, federated, store, Options{
		Cadence:     time.Hour,
		Concurrency: concurrency,
		CaseLimit:   3,
		Logger:      log.New(io.Discard, "", 0),
	})
	return scheduler, source, direct, federated, store, clk
}

func suite(id string) *evalv1.EvalSuite {
	return &evalv1.EvalSuite{SuiteId: id, ProviderId: "provider-" + id, State: "active"}
}

func TestSchedulerDiscoversBothTiersAndRespectsDynamicSuiteSet(t *testing.T) {
	scheduler, source, _, _, store, clk := testScheduler(t, []*evalv1.EvalSuite{suite("one"), suite("two")}, 2)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("first tick: %v", err)
	}
	if got := store.count(); got != 4 {
		t.Fatalf("first run count = %d, want 4", got)
	}

	source.mu.Lock()
	source.suites = []*evalv1.EvalSuite{suite("one"), suite("three")}
	source.mu.Unlock()
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("same-cadence tick: %v", err)
	}
	if got := store.count(); got != 6 {
		t.Fatalf("dynamic run count = %d, want new suite only (6 total)", got)
	}

	clk.Advance(time.Hour)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("cadence tick: %v", err)
	}
	if got := store.count(); got != 10 {
		t.Fatalf("cadence run count = %d, want 10", got)
	}
	store.mu.Lock()
	runs := append([]recordedRun(nil), store.runs...)
	store.mu.Unlock()
	for _, run := range runs {
		if run.tag != schedulerRunTag+":provider_direct" && run.tag != schedulerRunTag+":federated" {
			t.Fatalf("run tag = %q, want scheduler tier tag", run.tag)
		}
	}
}

func TestSchedulerContinuesOtherSuitesAndTierAfterRunnerFailure(t *testing.T) {
	scheduler, _, direct, federated, store, _ := testScheduler(t, []*evalv1.EvalSuite{suite("bad"), suite("good")}, 2)
	direct.errFor = map[string]bool{"bad": true}
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if got := store.count(); got != 4 {
		t.Fatalf("stored runs = %d, want degraded bad direct + bad federated + both good tiers", got)
	}
	if calls, _ := federated.stats(); calls != 2 {
		t.Fatalf("federated calls = %d, want 2", calls)
	}
}

func TestSchedulerBoundsSuiteConcurrency(t *testing.T) {
	scheduler, _, direct, federated, _, _ := testScheduler(t, []*evalv1.EvalSuite{suite("one"), suite("two"), suite("three"), suite("four")}, 2)
	direct.onStart = func(string) { time.Sleep(10 * time.Millisecond) }
	federated.onStart = func(string) { time.Sleep(10 * time.Millisecond) }
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	_, directMax := direct.stats()
	_, federatedMax := federated.stats()
	if directMax > 2 || federatedMax > 2 {
		t.Fatalf("tier concurrency direct/federated = %d/%d, want each <= 2", directMax, federatedMax)
	}
}

func TestSchedulerDoesNotOverlapCycles(t *testing.T) {
	scheduler, _, direct, _, store, _ := testScheduler(t, []*evalv1.EvalSuite{suite("one")}, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	direct.onStart = func(string) {
		close(started)
		<-release
	}
	finished := make(chan error, 1)
	go func() { finished <- scheduler.Tick(context.Background()) }()
	<-started
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("overlapping tick: %v", err)
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatalf("first tick: %v", err)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("stored runs = %d, want one direct and one federated run", got)
	}
}

func TestSchedulerValidatesOnSeparateCadenceAndPersistsVerdicts(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)}
	source := &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	store := &fakeStore{}
	validator := &fakeValidator{}
	scheduler := New(clk, source, direct, federated, store, Options{
		Cadence:           7 * 24 * time.Hour,
		ValidationCadence: 24 * time.Hour,
		Concurrency:       1,
		Validation:        validator,
		Logger:            log.New(io.Discard, "", 0),
	})

	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got := validator.count(); got != 1 {
		t.Fatalf("initial validation calls = %d, want 1", got)
	}
	if got := store.validationCount(); got != 1 {
		t.Fatalf("initial persisted validations = %d, want 1", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("initial eval runs = %d, want 2", got)
	}

	clk.Advance(12 * time.Hour)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("half-cadence tick: %v", err)
	}
	if got := validator.count(); got != 1 {
		t.Fatalf("half-cadence validation calls = %d, want 1", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("half-cadence eval runs = %d, want 2", got)
	}

	clk.Advance(12 * time.Hour)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("validation-cadence tick: %v", err)
	}
	if got := validator.count(); got != 2 {
		t.Fatalf("validation-cadence calls = %d, want 2", got)
	}
	if got := store.validationCount(); got != 2 {
		t.Fatalf("validation-cadence persisted validations = %d, want 2", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("validation-cadence eval runs = %d, want 2", got)
	}
}

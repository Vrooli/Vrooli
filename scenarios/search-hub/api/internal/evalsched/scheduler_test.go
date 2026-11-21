package evalsched

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	internaleval "search-hub/internal/eval"

	"github.com/vrooli/api-core/scheduletest"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

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

type historyStore struct {
	base       *fakeStore
	clock      *scheduletest.FakeClock
	runs       []*evalv1.EvalRun
	validation *internaleval.CorpusValidationRecord
}

func (s *historyStore) AppendRun(ctx context.Context, run *evalv1.EvalRun) error {
	if run.GetCreatedAt() == "" {
		run.CreatedAt = s.clock.Now().UTC().Format(time.RFC3339Nano)
	}
	if err := s.base.AppendRun(ctx, run); err != nil {
		return err
	}
	s.runs = append(s.runs, run)
	return nil
}

func (s *historyStore) AppendCorpusValidation(ctx context.Context, suiteID string, result *evalv1.ValidateCorpusResponse, observedAt time.Time) error {
	if err := s.base.AppendCorpusValidation(ctx, suiteID, result, observedAt); err != nil {
		return err
	}
	s.validation = &internaleval.CorpusValidationRecord{CreatedAt: observedAt, Result: result}
	return nil
}

func (s *historyStore) ListRuns(_ context.Context, filter internaleval.ListRunsFilter) ([]*evalv1.EvalRun, error) {
	for _, run := range s.runs {
		if run.GetSuiteId() == filter.SuiteID && run.GetTag() == filter.Tag {
			return []*evalv1.EvalRun{run}, nil
		}
	}
	return nil, nil
}

func (s *historyStore) LatestCorpusValidation(_ context.Context, _ string) (*internaleval.CorpusValidationRecord, error) {
	return s.validation, nil
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
	mu             sync.Mutex
	calls          []string
	failRemaining  int
	providerErrors int
}

func (v *fakeValidator) ValidateCorpus(_ context.Context, suite *evalv1.EvalSuite, _ int32) (*evalv1.ValidateCorpusResponse, error) {
	v.mu.Lock()
	v.calls = append(v.calls, suite.GetSuiteId())
	if v.failRemaining > 0 {
		v.failRemaining--
		v.mu.Unlock()
		return nil, context.DeadlineExceeded
	}
	v.mu.Unlock()
	return &evalv1.ValidateCorpusResponse{Rollup: &evalv1.CorpusValidationRollup{Live: 1, ProviderErrors: int32(v.providerErrors)}}, nil
}

func (v *fakeValidator) count() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.calls)
}

type fakeRunner struct {
	name          string
	mu            sync.Mutex
	calls         []string
	active        int
	maxActive     int
	errFor        map[string]bool
	failRemaining map[string]int
	onStart       func(string)
	onComplete    func()

	budget    time.Duration
	budgetSet bool
}

func (r *fakeRunner) stats() (calls, maxActive int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls), r.maxActive
}

// observedBudget returns the run budget the scheduler handed to the runner,
// measured from the context deadline installed by runTier.
func (r *fakeRunner) observedBudget() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.budget
}

func (r *fakeRunner) Run(ctx context.Context, suite *evalv1.EvalSuite, tag string, _ int32) (*evalv1.EvalRun, error) {
	id := suite.GetSuiteId()
	r.mu.Lock()
	r.calls = append(r.calls, id)
	if deadline, ok := ctx.Deadline(); ok {
		r.budget = time.Until(deadline)
		r.budgetSet = true
	}
	if r.errFor[id] {
		r.mu.Unlock()
		return nil, context.DeadlineExceeded
	}
	if r.failRemaining[id] > 0 {
		r.failRemaining[id]--
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

func testScheduler(t *testing.T, suites []*evalv1.EvalSuite, concurrency int) (*Scheduler, *fakeSuites, *fakeRunner, *fakeRunner, *fakeStore, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
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

func TestSchedulerGivesFederatedTierItsOwnCaseBudget(t *testing.T) {
	scheduler, _, direct, federated, store, _ := testScheduler(t, []*evalv1.EvalSuite{suite("one")}, 1)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("stored runs = %d, want one direct and one federated run", got)
	}
	directBudget := direct.observedBudget()
	federatedBudget := federated.observedBudget()
	wantDirect := DefaultCaseTimeout*time.Duration(3) + DefaultRunOverhead
	wantFederated := DefaultFederatedCaseTimeout*time.Duration(3) + DefaultRunOverhead
	if directBudget <= 0 || federatedBudget <= 0 {
		t.Fatalf("observed budgets direct=%s federated=%s, want both positive", directBudget, federatedBudget)
	}
	if directBudget > wantDirect {
		t.Fatalf("direct run budget = %s, want <= %s", directBudget, wantDirect)
	}
	if federatedBudget <= directBudget {
		t.Fatalf("federated run budget = %s, want wider than direct %s", federatedBudget, directBudget)
	}
	if federatedBudget > wantFederated {
		t.Fatalf("federated run budget = %s, want <= %s", federatedBudget, wantFederated)
	}
}

func TestOptionsDefaultCaseBudgetsAreTierAware(t *testing.T) {
	o := Options{}.withDefaults()
	if o.CaseTimeout != DefaultCaseTimeout {
		t.Fatalf("default case timeout = %s, want %s", o.CaseTimeout, DefaultCaseTimeout)
	}
	if o.FederatedCaseTimeout != DefaultFederatedCaseTimeout {
		t.Fatalf("default federated case timeout = %s, want %s", o.FederatedCaseTimeout, DefaultFederatedCaseTimeout)
	}
	if o.FederatedCaseTimeout <= o.CaseTimeout {
		t.Fatalf("federated case timeout %s must exceed provider case timeout %s", o.FederatedCaseTimeout, o.CaseTimeout)
	}
}

func TestOptionsFromEnvReadsFederatedTimeoutIndependently(t *testing.T) {
	t.Setenv("SEARCH_HUB_EVAL_SCHEDULER_TIMEOUT", "6s")
	t.Setenv("SEARCH_HUB_EVAL_SCHEDULER_FEDERATED_TIMEOUT", "45s")
	o := OptionsFromEnv(log.New(io.Discard, "", 0))
	if o.CaseTimeout != 6*time.Second {
		t.Fatalf("provider case timeout = %s, want 6s", o.CaseTimeout)
	}
	if o.FederatedCaseTimeout != 45*time.Second {
		t.Fatalf("federated case timeout = %s, want 45s", o.FederatedCaseTimeout)
	}
}

func TestOptionsFromEnvReadsBoundedRetryBackoff(t *testing.T) {
	t.Setenv("SEARCH_HUB_EVAL_SCHEDULER_RETRY_BACKOFF", "35m")
	if got := OptionsFromEnv(log.New(io.Discard, "", 0)).RetryBackoff; got != 35*time.Minute {
		t.Fatalf("retry backoff = %s, want 35m", got)
	}
	if got := (Options{RetryBackoff: 48 * time.Hour}).withDefaults().RetryBackoff; got != MaxRetryBackoff {
		t.Fatalf("oversized retry backoff = %s, want capped %s", got, MaxRetryBackoff)
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
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
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

func TestSchedulerDelaysFirstTickForDependencyStartup(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	store := &fakeStore{}
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler := New(clk, &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}, &fakeRunner{name: "direct"}, &fakeRunner{name: "federated"}, store, Options{
		Cadence:       time.Hour,
		StartupDelay:  2 * time.Second,
		StartupJitter: 0,
		Sleep: func(_ context.Context, delay time.Duration) error {
			if delay != 2*time.Second {
				t.Errorf("startup delay = %s, want 2s", delay)
			}
			close(started)
			<-release
			return nil
		},
		Logger: log.New(io.Discard, "", 0),
	})
	done := make(chan struct{})
	go func() { scheduler.Run(ctx); close(done) }()
	<-started
	// The injected sleep is the first observable action; the scheduler has not
	// been allowed to run a boot tick before dependency startup completes.
	if got := store.count(); got != 0 {
		t.Fatalf("runs before startup delay completed = %d, want 0", got)
	}
	close(release)
	cancel()
	<-done
}

func TestSchedulerRetriesFailedEvaluationAfterBoundedBackoff(t *testing.T) {
	scheduler, _, direct, federated, store, clk := testScheduler(t, []*evalv1.EvalSuite{suite("one")}, 1)
	direct.failRemaining = map[string]int{"one": 1}
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got, _ := direct.stats(); got != 1 {
		t.Fatalf("initial direct calls = %d, want 1", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("initial receipts = %d, want degraded direct plus successful federated", got)
	}

	clk.Advance(DefaultRetryBackoff - time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("early retry tick: %v", err)
	}
	if got, _ := direct.stats(); got != 1 {
		t.Fatalf("direct calls before retry deadline = %d, want 1", got)
	}

	clk.Advance(time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("retry tick: %v", err)
	}
	if got, _ := direct.stats(); got != 2 {
		t.Fatalf("direct calls at retry deadline = %d, want 2", got)
	}
	if got, _ := federated.stats(); got != 2 {
		t.Fatalf("federated calls = %d, want both tiers rerun for a complete suite receipt", got)
	}
	if got := store.count(); got != 4 {
		t.Fatalf("receipts after retry = %d, want 4", got)
	}

	clk.Advance(DefaultRetryBackoff - time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("post-success tick: %v", err)
	}
	if got, _ := direct.stats(); got != 2 {
		t.Fatalf("successful suite retried too early; direct calls = %d, want 2", got)
	}
}

func TestSchedulerTreatsMissingTierRunnerAsDegradedAndRetries(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	federated := &fakeRunner{name: "federated"}
	store := &fakeStore{}
	scheduler := New(clk, &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}, nil, federated, store, Options{
		Concurrency:  1,
		CaseLimit:    2,
		RetryBackoff: 15 * time.Minute,
		Logger:       log.New(io.Discard, "", 0),
	})
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("initial receipts = %d, want degraded missing direct runner plus federated result", got)
	}
	clk.Advance(15 * time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("retry tick: %v", err)
	}
	if got := store.count(); got != 4 {
		t.Fatalf("receipts after retry = %d, want missing tier re-attempt and federated result", got)
	}
}

func TestSchedulerRetriesCorpusValidationWithoutRepeatingFreshEvaluation(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	source := &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	store := &fakeStore{}
	validator := &fakeValidator{failRemaining: 1}
	scheduler := New(clk, source, direct, federated, store, Options{
		Cadence:           7 * 24 * time.Hour,
		ValidationCadence: 24 * time.Hour,
		RetryBackoff:      15 * time.Minute,
		Concurrency:       1,
		Validation:        validator,
		Logger:            log.New(io.Discard, "", 0),
	})
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("initial eval runs = %d, want 2", got)
	}
	if got := store.validationCount(); got != 0 {
		t.Fatalf("validation receipts after failed validation = %d, want 0", got)
	}

	clk.Advance(15 * time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("validation retry tick: %v", err)
	}
	if got := store.validationCount(); got != 1 {
		t.Fatalf("validation receipts after retry = %d, want 1", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("evaluation runs after validation-only retry = %d, want unchanged 2", got)
	}
	if got := validator.count(); got != 2 {
		t.Fatalf("validation calls = %d, want initial failure and one retry", got)
	}
}

func TestSchedulerRetriesPartialCorpusValidationWithProviderErrors(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	source := &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	store := &fakeStore{}
	validator := &fakeValidator{providerErrors: 1}
	scheduler := New(clk, source, direct, federated, store, Options{
		Cadence:           7 * 24 * time.Hour,
		ValidationCadence: 24 * time.Hour,
		RetryBackoff:      15 * time.Minute,
		Concurrency:       1,
		Validation:        validator,
		Logger:            log.New(io.Discard, "", 0),
	})
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got := store.validationCount(); got != 1 {
		t.Fatalf("partial validation receipts = %d, want retained evidence", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("initial eval runs = %d, want 2", got)
	}

	clk.Advance(15 * time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("validation retry tick: %v", err)
	}
	if got := store.validationCount(); got != 2 {
		t.Fatalf("validation receipts after retry = %d, want 2", got)
	}
	if got := store.count(); got != 2 {
		t.Fatalf("evaluation runs after validation-only retry = %d, want unchanged 2", got)
	}
	if got := validator.count(); got != 2 {
		t.Fatalf("validation calls = %d, want initial partial receipt and one retry", got)
	}
}

func TestSchedulerRestoresSuccessfulCadenceFromDurableHistory(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	base := &fakeStore{}
	storedAt := clk.Now().Add(-30 * time.Minute).Format(time.RFC3339Nano)
	store := &historyStore{base: base, clock: clk, runs: []*evalv1.EvalRun{
		{SuiteId: "one", Tag: schedulerRunTag + ":provider_direct", CreatedAt: storedAt},
		{SuiteId: "one", Tag: schedulerRunTag + ":federated", CreatedAt: storedAt},
	}}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	scheduler := New(clk, &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}, direct, federated, store, Options{
		Cadence:     time.Hour,
		Concurrency: 1,
		Logger:      log.New(io.Discard, "", 0),
	})
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got, _ := direct.stats(); got != 0 {
		t.Fatalf("direct calls before restored cadence = %d, want 0", got)
	}
	clk.Advance(29 * time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("pre-cadence tick: %v", err)
	}
	if got, _ := direct.stats(); got != 0 {
		t.Fatalf("direct calls before cadence = %d, want 0", got)
	}
	clk.Advance(time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("cadence tick: %v", err)
	}
	if got, _ := direct.stats(); got != 1 {
		t.Fatalf("direct calls at restored cadence = %d, want 1", got)
	}
}

func TestSchedulerRestoresFailedTierBackoffFromDurableHistory(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	base := &fakeStore{}
	storedAt := clk.Now().Add(-5 * time.Minute).Format(time.RFC3339Nano)
	store := &historyStore{base: base, clock: clk, runs: []*evalv1.EvalRun{
		{SuiteId: "one", Tag: schedulerRunTag + ":provider_direct", CreatedAt: storedAt, Degraded: true},
		{SuiteId: "one", Tag: schedulerRunTag + ":federated", CreatedAt: storedAt},
	}}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	scheduler := New(clk, &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}, direct, federated, store, Options{
		Cadence:      time.Hour,
		RetryBackoff: 15 * time.Minute,
		Concurrency:  1,
		Logger:       log.New(io.Discard, "", 0),
	})
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got, _ := direct.stats(); got != 0 {
		t.Fatalf("direct calls before restored retry deadline = %d, want 0", got)
	}
	clk.Advance(10 * time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("retry tick: %v", err)
	}
	if got, _ := direct.stats(); got != 1 {
		t.Fatalf("direct calls at restored retry deadline = %d, want 1", got)
	}
	if got, _ := federated.stats(); got != 1 {
		t.Fatalf("federated calls on complete-suite retry = %d, want 1", got)
	}
}

func TestSchedulerRestoresValidationBackoffWithoutRerunningEvaluation(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	base := &fakeStore{}
	storedAt := clk.Now().Add(-5 * time.Minute)
	store := &historyStore{base: base, clock: clk, runs: []*evalv1.EvalRun{
		{SuiteId: "one", Tag: schedulerRunTag + ":provider_direct", CreatedAt: storedAt.Format(time.RFC3339Nano)},
		{SuiteId: "one", Tag: schedulerRunTag + ":federated", CreatedAt: storedAt.Format(time.RFC3339Nano)},
	}, validation: &internaleval.CorpusValidationRecord{
		CreatedAt: storedAt,
		Result:    &evalv1.ValidateCorpusResponse{Rollup: &evalv1.CorpusValidationRollup{ProviderErrors: 1}},
	}}
	direct := &fakeRunner{name: "direct"}
	federated := &fakeRunner{name: "federated"}
	validator := &fakeValidator{}
	scheduler := New(clk, &fakeSuites{suites: []*evalv1.EvalSuite{suite("one")}}, direct, federated, store, Options{
		Cadence:           7 * 24 * time.Hour,
		ValidationCadence: 24 * time.Hour,
		RetryBackoff:      15 * time.Minute,
		Concurrency:       1,
		Validation:        validator,
		Logger:            log.New(io.Discard, "", 0),
	})
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("initial tick: %v", err)
	}
	if got, _ := direct.stats(); got != 0 {
		t.Fatalf("direct calls before restored validation retry = %d, want 0", got)
	}
	clk.Advance(10 * time.Minute)
	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("validation retry tick: %v", err)
	}
	if got := validator.count(); got != 1 {
		t.Fatalf("validation calls after restored retry = %d, want 1", got)
	}
	if got, _ := direct.stats(); got != 0 {
		t.Fatalf("eval calls during validation-only retry = %d, want 0", got)
	}
}

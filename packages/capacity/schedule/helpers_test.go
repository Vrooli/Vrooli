package schedule

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// testEpoch is a Wednesday at 09:00 UTC. Window tests use UTC explicitly so
// weekday and hour arithmetic is independent of the host timezone.
var testEpoch = time.Date(2026, time.November, 18, 9, 0, 0, 0, time.UTC)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func (c *fakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

type fakeLease struct {
	admission *fakeAdmission
	once      sync.Once
	released  bool
}

func (l *fakeLease) Release(context.Context) error {
	l.once.Do(func() {
		l.admission.mu.Lock()
		l.admission.active--
		l.admission.mu.Unlock()
		l.released = true
	})
	return nil
}

// fakeAdmission grants by default and consumes queued verdicts in order.
type fakeAdmission struct {
	mu       sync.Mutex
	verdicts []Verdict
	err      error
	requests []AdmissionRequest
	active   int
	granted  int
}

func (f *fakeAdmission) Acquire(_ context.Context, req AdmissionRequest) (Lease, Verdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, Verdict{Kind: VerdictDeny, Reason: f.err.Error()}, f.err
	}
	verdict := Verdict{Kind: VerdictGrant}
	if len(f.verdicts) > 0 {
		verdict = f.verdicts[0]
		f.verdicts = f.verdicts[1:]
	}
	if !verdict.Granted() {
		return nil, verdict, nil
	}
	f.active++
	f.granted++
	return &fakeLease{admission: f}, verdict, nil
}

func (f *fakeAdmission) push(verdicts ...Verdict) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.verdicts = append(f.verdicts, verdicts...)
}

func (f *fakeAdmission) snapshot() (requests []AdmissionRequest, active, granted int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]AdmissionRequest(nil), f.requests...), f.active, f.granted
}

type fakeExecutor struct {
	mu    sync.Mutex
	calls []Job
	fn    func(ctx context.Context, job Job) (Result, error)
}

func (f *fakeExecutor) Execute(ctx context.Context, job Job) (Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, job)
	fn := f.fn
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx, job)
	}
	return Result{
		ActualSeconds: job.Estimate.Seconds,
		Items:         job.Estimate.Items,
		Tokens:        job.Estimate.Tokens,
	}, nil
}

func (f *fakeExecutor) set(fn func(ctx context.Context, job Job) (Result, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fn = fn
}

func (f *fakeExecutor) seen() []Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Job(nil), f.calls...)
}

type fakeObserver struct {
	mu    sync.Mutex
	state HostState
	err   error
	calls int
}

func (o *fakeObserver) Observe(context.Context) (HostState, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls++
	return o.state, o.err
}

func (o *fakeObserver) set(state HostState) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state = state
}

type harness struct {
	store     *Store
	clock     *fakeClock
	admission *fakeAdmission
	executor  *fakeExecutor
	observer  *fakeObserver
	scheduler *Scheduler
}

type harnessOptions struct {
	Policy    Policy
	Admission Admission
	Executor  Executor
	Observer  HostObserver
	LeaseTTL  time.Duration
}

func newHarness(t *testing.T, opts harnessOptions) *harness {
	t.Helper()
	clock := &fakeClock{t: testEpoch}
	store, err := NewStore(context.Background(), Config{
		Path:  filepath.Join(t.TempDir(), "schedule.db"),
		Clock: clock.Now,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	policy := opts.Policy
	if policy.Location == nil {
		policy.Location = time.UTC
	}
	admission := &fakeAdmission{}
	admissionImpl := Admission(admission)
	if opts.Admission != nil {
		admissionImpl = opts.Admission
		if fake, ok := opts.Admission.(*fakeAdmission); ok {
			admission = fake
		}
	}
	executor := &fakeExecutor{}
	executorImpl := Executor(executor)
	if opts.Executor != nil {
		executorImpl = opts.Executor
		if fake, ok := opts.Executor.(*fakeExecutor); ok {
			executor = fake
		}
	}
	scheduler, err := New(SchedulerConfig{
		Store:      store,
		Policy:     policy,
		Admission:  admissionImpl,
		Executor:   executorImpl,
		Observer:   opts.Observer,
		LeaseOwner: "test-runner",
		LeaseTTL:   opts.LeaseTTL,
	})
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	observer, _ := opts.Observer.(*fakeObserver)
	return &harness{
		store:     store,
		clock:     clock,
		admission: admission,
		executor:  executor,
		observer:  observer,
		scheduler: scheduler,
	}
}

// newGrantHarness is the common case: fake admission grants, fake executor
// succeeds.
func newGrantHarness(t *testing.T, policy Policy) *harness {
	t.Helper()
	return newHarness(t, harnessOptions{
		Policy:    policy,
		Admission: &fakeAdmission{},
		Executor:  &fakeExecutor{},
	})
}

func (h *harness) submit(t *testing.T, jobs ...Job) {
	t.Helper()
	if _, err := h.store.SubmitBatch(context.Background(), jobs); err != nil {
		t.Fatalf("submit jobs: %v", err)
	}
}

func (h *harness) mustTick(t *testing.T) TickResult {
	t.Helper()
	result, err := h.scheduler.Tick(context.Background())
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	return result
}

func (h *harness) job(t *testing.T, id string) Job {
	t.Helper()
	job, err := h.store.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get job %s: %v", id, err)
	}
	return job
}

func (h *harness) status(t *testing.T) Status {
	t.Helper()
	status, err := h.scheduler.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	return status
}

// jobSpec builds a valid job with sensible test defaults.
func jobSpec(id, kind, owner string, mutate ...func(*Job)) Job {
	job := Job{
		ID:            id,
		Kind:          kind,
		Owner:         owner,
		Priority:      PriorityBatch,
		ResourceClass: ResourceClassCompute,
		ModelAffinity: "nomic-embed-text",
		Estimate:      Estimate{Seconds: 10, RAMBytes: 64 << 20, Items: 4, Tokens: 512},
		Value:         1,
	}
	for _, apply := range mutate {
		apply(&job)
	}
	return job
}

// containsSubstring reports whether any value contains want.
func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

// mustContain fails when want is absent from got.
func mustContain(t *testing.T, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, got)
}

var errTransient = errors.New("transient executor failure")

// nopCheckpointer lets a checkpoint-capable test executor also satisfy Executor.
type nopCheckpointer struct{}

func (nopCheckpointer) Save(context.Context, string) error { return nil }

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func describeJobs(jobs []Job) string {
	out := ""
	for _, job := range jobs {
		out += fmt.Sprintf("%s(%s/%s/%s) ", job.ID, job.State, job.Priority, job.ModelAffinity)
	}
	return out
}

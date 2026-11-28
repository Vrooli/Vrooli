package schedule

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	capacity "github.com/vrooli/vrooli/packages/capacity"
)

type fakeCapacityLease struct {
	mu         sync.Mutex
	released   bool
	heartbeats int
	identity   capacity.GPUIdentity
}

func (l *fakeCapacityLease) Release(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.released = true
	return nil
}

func (l *fakeCapacityLease) Heartbeat(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.heartbeats++
	return nil
}

func (l *fakeCapacityLease) Identity() capacity.GPUIdentity { return l.identity }

type gpuRequest struct {
	owner string
	uuid  string
	vram  int64
}

type fakeCapacityBroker struct {
	mu               sync.Mutex
	resourceRequests []capacity.Request
	gpuRequests      []gpuRequest
	verdict          capacity.Verdict
	err              error
	lease            *fakeCapacityLease
}

func newFakeCapacityBroker() *fakeCapacityBroker {
	return &fakeCapacityBroker{
		verdict: capacity.Verdict{Kind: VerdictGrant},
		lease:   &fakeCapacityLease{identity: capacity.GPUIdentity{Index: 0, UUID: "GPU-1234", Name: "RTX 4070"}},
	}
}

func (b *fakeCapacityBroker) AcquireResources(_ context.Context, req capacity.Request) (capacity.RenewableLease, capacity.Verdict, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resourceRequests = append(b.resourceRequests, req)
	if b.err != nil {
		return nil, b.verdict, b.err
	}
	if !verdictGranted(b.verdict.Kind) {
		return nil, b.verdict, nil
	}
	return b.lease, b.verdict, nil
}

func (b *fakeCapacityBroker) AcquireGPU(_ context.Context, ownerID, uuid string, vramBytes int64) (capacity.GPULease, capacity.Verdict, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.gpuRequests = append(b.gpuRequests, gpuRequest{owner: ownerID, uuid: uuid, vram: vramBytes})
	if b.err != nil {
		return nil, b.verdict, b.err
	}
	if !verdictGranted(b.verdict.Kind) {
		return nil, b.verdict, nil
	}
	return b.lease, b.verdict, nil
}

func verdictGranted(kind string) bool {
	return kind == VerdictGrant || kind == VerdictDegrade
}

func (b *fakeCapacityBroker) snapshot(t *testing.T) ([]capacity.Request, []gpuRequest) {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]capacity.Request(nil), b.resourceRequests...), append([]gpuRequest(nil), b.gpuRequests...)
}

func TestBrokerAdmissionComputeRequest(t *testing.T) {
	broker := newFakeCapacityBroker()
	admission, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:agent-manager:embeddings"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	lease, verdict, err := admission.Acquire(context.Background(), AdmissionRequest{
		JobID:         "job-1",
		Kind:          "embed",
		Owner:         "agent-manager",
		ResourceClass: ResourceClassCompute,
		Estimate:      Estimate{RAMBytes: 512 << 20},
		Priority:      PriorityInteractive,
		NoReclaim:     true,
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if !verdict.Granted() {
		t.Fatalf("verdict = %+v, want grant", verdict)
	}
	if lease == nil {
		t.Fatal("expected a lease")
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("release: %v", err)
	}
	requests, _ := broker.snapshot(t)
	if len(requests) != 1 {
		t.Fatalf("broker calls = %d, want 1", len(requests))
	}
	request := requests[0]
	if request.OwnerID != "schedule:agent-manager:embeddings:embed:job-1" {
		t.Fatalf("owner id = %q, want the namespaced job identity", request.OwnerID)
	}
	if request.RAMBytes != 512<<20 {
		t.Fatalf("ram = %d, want 512MiB", request.RAMBytes)
	}
	if request.Priority != string(PriorityInteractive) {
		t.Fatalf("priority = %q, want interactive", request.Priority)
	}
	if !request.NoReclaim {
		t.Fatal("deferred work must never reclaim resident capacity")
	}
}

func TestBrokerAdmissionPassesThroughVerdicts(t *testing.T) {
	for _, kind := range []string{VerdictQueue, VerdictDeny} {
		t.Run(kind, func(t *testing.T) {
			broker := newFakeCapacityBroker()
			broker.verdict = capacity.Verdict{Kind: kind, Reason: "no headroom"}
			admission, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:test"})
			if err != nil {
				t.Fatalf("new admission: %v", err)
			}
			lease, verdict, err := admission.Acquire(context.Background(), AdmissionRequest{
				JobID: "job", Kind: "embed", ResourceClass: ResourceClassCompute,
				Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch,
			})
			if err != nil {
				t.Fatalf("acquire: %v", err)
			}
			if lease != nil {
				t.Fatalf("lease = %v, want nil on refusal", lease)
			}
			if verdict.Kind != kind || verdict.Reason != "no headroom" {
				t.Fatalf("verdict = %+v, want %s preserved verbatim", verdict, kind)
			}
			if verdict.Granted() {
				t.Fatal("a queue/deny verdict must not be granted")
			}
		})
	}
}

func TestBrokerAdmissionFailsClosedOnUnknownVerdict(t *testing.T) {
	broker := newFakeCapacityBroker()
	broker.verdict = capacity.Verdict{Kind: "maybe", Reason: "malformed"}
	admission, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:test"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	_, verdict, err := admission.Acquire(context.Background(), AdmissionRequest{
		JobID: "job", Kind: "embed", ResourceClass: ResourceClassCompute,
		Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch,
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if verdict.Kind != VerdictDeny {
		t.Fatalf("verdict = %+v, want an unknown kind to fail closed", verdict)
	}
}

func TestBrokerAdmissionVRAmRequiresIdentity(t *testing.T) {
	broker := newFakeCapacityBroker()
	admission, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:test"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	request := AdmissionRequest{
		JobID: "job", Kind: "annotate", ResourceClass: ResourceClassVRAM,
		Estimate: Estimate{VRAMBytes: 6 << 30}, Priority: PriorityBatch,
	}
	_, verdict, err := admission.Acquire(context.Background(), request)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if verdict.Kind != VerdictDeny {
		t.Fatalf("verdict = %+v, want a denial without a bound GPU identity", verdict)
	}
	if _, gpus := broker.snapshot(t); len(gpus) != 0 {
		t.Fatalf("GPU admission was attempted without an identity: %v", gpus)
	}

	bound, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:test", GPUUUID: "GPU-1234"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	lease, verdict, err := bound.Acquire(context.Background(), request)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if !verdict.Granted() || lease == nil {
		t.Fatalf("verdict = %+v lease = %v, want a GPU grant", verdict, lease)
	}
	_, gpus := broker.snapshot(t)
	if len(gpus) != 1 || gpus[0].uuid != "GPU-1234" || gpus[0].vram != 6<<30 {
		t.Fatalf("gpu requests = %+v, want the bound identity and measured VRAM", gpus)
	}
}

func TestBrokerAdmissionNoneClassSkipsBroker(t *testing.T) {
	broker := newFakeCapacityBroker()
	admission, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:test"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	lease, verdict, err := admission.Acquire(context.Background(), AdmissionRequest{
		JobID: "job", Kind: "embed", ResourceClass: ResourceClassNone, Priority: PriorityBatch,
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if !verdict.Granted() || lease == nil {
		t.Fatalf("verdict = %+v lease = %v, want a grant", verdict, lease)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if requests, gpus := broker.snapshot(t); len(requests) != 0 || len(gpus) != 0 {
		t.Fatalf("no-admission class reached the broker: %v %v", requests, gpus)
	}
}

func TestNewBrokerAdmissionValidation(t *testing.T) {
	if _, err := NewBrokerAdmission(nil, BrokerAdmissionOptions{OwnerPrefix: "x"}); err == nil {
		t.Fatal("nil broker must be rejected")
	}
	if _, err := NewBrokerAdmission(newFakeCapacityBroker(), BrokerAdmissionOptions{}); err == nil {
		t.Fatal("empty owner prefix must be rejected")
	}
	admission, err := NewBrokerAdmission(newFakeCapacityBroker(), BrokerAdmissionOptions{OwnerPrefix: ":schedule:test:"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	if _, _, err := admission.Acquire(context.Background(), AdmissionRequest{
		JobID: "job", Kind: "embed", ResourceClass: ResourceClassCompute,
		Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch,
	}); err != nil {
		t.Fatalf("acquire: %v", err)
	}
}

func TestAdmissionRequestValidate(t *testing.T) {
	cases := map[string]AdmissionRequest{
		"missing job":      {Kind: "embed", ResourceClass: ResourceClassCompute, Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch},
		"unknown priority": {JobID: "job", ResourceClass: ResourceClassCompute, Estimate: Estimate{RAMBytes: 1}, Priority: "urgent"},
		"unknown class":    {JobID: "job", ResourceClass: "disk", Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch},
		"no measurement":   {JobID: "job", ResourceClass: ResourceClassCompute, Priority: PriorityBatch},
		"vram without ram": {JobID: "job", ResourceClass: ResourceClassVRAM, Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch},
	}
	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			if err := request.Validate(); err == nil {
				t.Fatal("expected the request to be rejected")
			}
		})
	}
	valid := AdmissionRequest{JobID: "job", Kind: "embed", ResourceClass: ResourceClassCompute, Estimate: Estimate{RAMBytes: 1}, Priority: PriorityBatch}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
}

type fakeObservationSource struct {
	observation capacity.HostObservation
	err         error
}

func (s fakeObservationSource) ObserveHostState(context.Context) (capacity.HostObservation, error) {
	return s.observation, s.err
}

func TestBrokerObserverMapsHostState(t *testing.T) {
	observer := BrokerObserver{Source: fakeObservationSource{observation: capacity.HostObservation{AvailableRAMBytes: 12 << 30, Load1: 0.75}}}
	state, err := observer.Observe(context.Background())
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if state.AvailableRAMBytes != 12<<30 || state.Load1 != 0.75 {
		t.Fatalf("state = %+v, want the broker's readings", state)
	}
	if state.ObservedAt.IsZero() {
		t.Fatal("observed at must be set")
	}
	if _, err := (BrokerObserver{}).Observe(context.Background()); err == nil {
		t.Fatal("a nil source must be an error, not a zero observation")
	}
	if _, err := (BrokerObserver{Source: fakeObservationSource{err: errors.New("sensor down")}}).Observe(context.Background()); err == nil {
		t.Fatal("observation errors must propagate")
	}
}

func TestSchedulerAdmitsThroughBroker(t *testing.T) {
	broker := newFakeCapacityBroker()
	admission, err := NewBrokerAdmission(broker, BrokerAdmissionOptions{OwnerPrefix: "schedule:agent-manager"})
	if err != nil {
		t.Fatalf("new admission: %v", err)
	}
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: admission,
		Executor:  &fakeExecutor{},
	})
	h.submit(t, jobSpec("job", "embed", "agent-manager", func(j *Job) { j.Priority = PriorityService }))
	result := h.mustTick(t)
	if result.Ran != 1 {
		t.Fatalf("tick = %+v, want the job to run", result)
	}
	requests, _ := broker.snapshot(t)
	if len(requests) != 1 {
		t.Fatalf("broker calls = %d, want 1", len(requests))
	}
	if requests[0].Priority != string(PriorityService) {
		t.Fatalf("priority = %q, want service", requests[0].Priority)
	}
	if !broker.lease.released {
		t.Fatal("the broker lease must be released after execution")
	}
}

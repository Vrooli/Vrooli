package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/testenv"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

type restartFenceTestService struct {
	commonv1connect.UnimplementedLifecycleMaintenanceServiceHandler
	mu                 sync.Mutex
	events             *[]string
	drainMode          string
	opID               string
	closed             bool
	failFirstResume    bool
	statusErr          error
	prepareErr         error
	port               int
	components         []*commonv1.LifecycleManagedComponent
	admitting          int32
	remaining          int32
	work               []*commonv1.LifecycleWorkItem
	statusBlockers     []*commonv1.LifecycleBlocker
	prepareDrained     bool
	statusDrained      bool
	requiresQuiescence bool
	providerInstanceID string
	fencePersistence   commonv1.LifecycleFencePersistence
	drainStarted       chan struct{}
	releaseDrain       chan struct{}
}

func (s *restartFenceTestService) record(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.events = append(*s.events, event)
}

func (s *restartFenceTestService) setDrainMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drainMode = mode
}

func (s *restartFenceTestService) operationID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opID
}

func (s *restartFenceTestService) Prepare(_ context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	s.mu.Lock()
	if s.prepareErr != nil {
		err := s.prepareErr
		s.mu.Unlock()
		return nil, err
	}
	mode := strings.TrimPrefix(s.drainMode, "emergency-")
	s.mu.Unlock()
	provider, instanceID, operationID := "alpha", s.providerInstance(), req.Msg.GetOperationId()
	closed := true
	if mode == "prepare-operation" {
		operationID = "different-operation"
	}
	if mode == "prepare-open" {
		closed = false
	}
	s.mu.Lock()
	s.opID = req.Msg.GetOperationId()
	s.closed = closed
	*s.events = append(*s.events, "prepare")
	s.mu.Unlock()
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: &commonv1.LifecycleStanding{
		Provider: provider, ProviderInstanceId: instanceID, OperationId: operationID,
		FenceToken: "fence-7", Revision: 7, AdmissionClosed: closed, InventoryComplete: true,
		FencePersistence:  s.providerFencePersistence(),
		ManagedComponents: s.components, Admitting: s.admitting,
		Remaining: s.remaining,
		Drained:   s.prepareDrained, ComponentStopReady: s.prepareDrained,
		QuiescencePending: closed && s.requiresQuiescence,
	}}), nil
}

func TestPrepareProviderFenceRefusesLiveProviderWithoutPrepareContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		code connect.Code
	}{{"not_found", connect.CodeNotFound}, {"unimplemented", connect.CodeUnimplemented}} {
		t.Run(tc.name, func(t *testing.T) {
			runner, service, events := restartFenceTestRunner(t, "allow")
			service.mu.Lock()
			service.prepareErr = connect.NewError(tc.code, errors.New("method unavailable"))
			service.mu.Unlock()
			_, err := runner.prepareProviderFence(context.Background(), scenario.Scenario{
				Slug: "alpha", Variant: "live",
				Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
			}, false, "", "operation-1", nil, nil)
			if err == nil || !strings.Contains(err.Error(), "does not implement the maintenance fence Prepare contract") {
				t.Fatalf("prepareProviderFence() error = %v, want fail-closed missing contract error", err)
			}
			if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
				t.Fatalf("provider mutation calls = %v, want no Drain after missing Prepare", got)
			}
		})
	}
}

func TestPrepareProviderFenceAllowsExplicitOverrideForNonAuthoritativeRuntime(t *testing.T) {
	runner, _, _ := restartFenceTestRunner(t, "allow")
	runner.deps.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
		return hostsession.Snapshot{BootID: "different-host-boot"}, nil
	}
	item := scenario.Scenario{
		Slug: "alpha", Variant: "live",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
	}
	if _, err := runner.prepareProviderFence(context.Background(), item, false, "", "operation-1", nil, nil); err == nil || !strings.Contains(err.Error(), "not authoritative") {
		t.Fatalf("prepareProviderFence() without override error = %v, want non-authoritative refusal", err)
	}
	fence, err := runner.prepareProviderFence(context.Background(), item, true, "replace stale generation", "operation-1", nil, nil)
	if err != nil {
		t.Fatalf("prepareProviderFence() with explicit override: %v", err)
	}
	if fence != nil {
		t.Fatalf("prepareProviderFence() fence = %#v, want no fence for unprovable stale generation", fence)
	}
}

func TestPrepareProviderFenceAllowsDeadPrelaunchStartupWithoutProviderRPC(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	clock := testenv.NewClock(time.Now().Add(-time.Hour))
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	ownerPID := 2_000_000_000
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		Scenario: "alpha", Variant: "live", Status: scenarioruntime.StatusStarting, Phase: "setup",
		OwnerKind: scenarioruntime.OwnerKindLifecycle, OwnerPID: &ownerPID, HostBootID: "provider-fence-test-boot",
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID: instance.InstanceID, Scenario: "alpha", Variant: "live", PortName: "api", EnvVar: "API_PORT",
		Port: port, Status: scenarioruntime.ClaimStatusReserved,
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	runner := &Runner{Home: home, deps: lifecycleDeps{
		runtimeRegistry: func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, Clock: clock})
		},
		hostSession: func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "provider-fence-test-boot"}, nil
		},
	}}
	item := scenario.Scenario{
		Slug: "alpha", Variant: "live",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
	}
	fence, err := runner.prepareProviderFence(ctx, item, false, "", "operation-1", nil, nil)
	if err != nil {
		t.Fatalf("prepareProviderFence() error = %v, want safe absent-provider result", err)
	}
	if fence != nil {
		t.Fatalf("prepareProviderFence() fence = %#v, want no RPC fence for absent provider", fence)
	}
}

func (s *restartFenceTestService) Status(context.Context, *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.statusErr != nil {
		return nil, s.statusErr
	}
	var blockers []*commonv1.LifecycleBlocker
	mode := strings.TrimPrefix(s.drainMode, "emergency-")
	if strings.HasPrefix(mode, "overrideable-") {
		blockers = []*commonv1.LifecycleBlocker{{Code: "operator_review", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE}}
	}
	if s.statusBlockers != nil {
		blockers = s.statusBlockers
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: &commonv1.LifecycleStanding{
		ManagedComponents: s.components, Admitting: s.admitting,
		Remaining: s.remaining,
		Work:      s.work,
		Blockers:  blockers,
		Drained:   s.statusDrained, ComponentStopReady: s.statusDrained,
		Provider: "alpha", ProviderInstanceId: s.providerInstance(), OperationId: s.opID,
		FenceToken: "fence-7", Revision: 7, AdmissionClosed: s.closed, InventoryComplete: true,
		FencePersistence:  s.providerFencePersistence(),
		QuiescencePending: s.closed && s.requiresQuiescence,
	}}), nil
}

func (s *restartFenceTestService) providerInstance() string {
	if s.providerInstanceID != "" {
		return s.providerInstanceID
	}
	return "provider-instance"
}

func (s *restartFenceTestService) providerFencePersistence() commonv1.LifecycleFencePersistence {
	if s.fencePersistence != commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_UNSPECIFIED {
		return s.fencePersistence
	}
	return commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE
}

func TestEmergencyRestartRefusesNegativeAdmittingWithOverrideableBlocker(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "emergency-overrideable-negative-admitting")
	service.admitting = -1
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	_, err := runner.RestartContext(t.Context(), "alpha", StartOptions{ForceLifecycle: true, LifecycleOverrideReason: "operator emergency"})
	if err == nil || !strings.Contains(err.Error(), "negative admitted or remaining") {
		t.Fatalf("RestartContext() error = %v, want negative provider work count refusal", err)
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
		t.Fatalf("lifecycle events = %v, want no Prepare, Drain, or Stop after malformed Status", got)
	}
}

func TestPrepareProviderFenceWarnsAndContinuesWithoutStatusContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		code connect.Code
	}{{"not_found", connect.CodeNotFound}, {"unimplemented", connect.CodeUnimplemented}} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			runner, service, events := restartFenceTestRunner(t, "allow")
			runner.Logger = slog.New(slog.NewTextHandler(&logs, nil))
			service.mu.Lock()
			service.statusErr = connect.NewError(tc.code, errors.New("method unavailable"))
			service.mu.Unlock()
			fence, err := runner.prepareProviderFence(context.Background(), scenario.Scenario{
				Slug: "alpha", Variant: "live",
				Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
			}, false, "", "operation-1", nil, nil)
			if err != nil || fence != nil {
				t.Fatalf("prepareProviderFence() = (%v, %v), want no fence and no error for unsupported Status", fence, err)
			}
			if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
				t.Fatalf("provider calls = %v, want no Prepare or Drain when Status is unavailable", got)
			}
			if got := logs.String(); !strings.Contains(got, "proceeding without a provider fence") || !strings.Contains(got, "scenario=alpha") {
				t.Fatalf("warning log = %q, want a warning naming alpha and the missing fence", got)
			}
		})
	}
}

func (s *restartFenceTestService) Drain(ctx context.Context, req *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) {
	if req.Msg.GetQuiesceComponents() {
		s.record("quiesce")
	} else {
		s.record("drain")
	}
	// A real provider bounds its own wait by the request timeout.
	if timeout := req.Msg.GetTimeout(); timeout != nil && timeout.AsDuration() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout.AsDuration())
		defer cancel()
	}
	s.mu.Lock()
	mode := strings.TrimPrefix(s.drainMode, "emergency-")
	opID := s.opID
	requiresQuiescence := s.requiresQuiescence
	drainStarted, releaseDrain := s.drainStarted, s.releaseDrain
	s.mu.Unlock()
	if drainStarted != nil {
		close(drainStarted)
		select {
		case <-releaseDrain:
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		}
	}
	if req.Msg.GetOperationId() != opID || req.Msg.GetFenceToken() != "fence-7" || req.Msg.GetRevision() != 7 {
		return nil, connect.NewError(connect.CodeAborted, errors.New("unexpected drain identity"))
	}
	if mode == "quiesce-zero-components" && req.Msg.GetQuiesceComponents() && requiresQuiescence {
		return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: &commonv1.LifecycleStanding{
			Provider: "alpha", ProviderInstanceId: "provider-instance", OperationId: opID,
			FenceToken: "fence-7", Revision: 7, AdmissionClosed: true, InventoryComplete: true,
			FencePersistence:   commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE,
			ComponentStopReady: true, QuiescencePending: false,
		}}), nil
	}
	switch mode {
	case "refuse":
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("active provider work remains"))
	case "timeout":
		<-ctx.Done()
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	case "slow":
		select {
		case <-time.After(11 * time.Second):
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		}
	}
	standing := &commonv1.LifecycleStanding{
		Provider: "alpha", ProviderInstanceId: "provider-instance", OperationId: s.opID,
		FenceToken: "fence-7", Revision: 7, AdmissionClosed: true, InventoryComplete: true, Drained: true,
		FencePersistence: commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE,
	}
	switch mode {
	case "drain-admitted":
		standing.Admitting = 1
	case "drain-remaining":
		standing.Remaining = 1
	case "drain-quiescence":
		standing.QuiescencePending = true
	case "drain-blocker":
		standing.Blockers = []*commonv1.LifecycleBlocker{{Code: "unknown-owner", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN}}
	case "drain-open":
		standing.AdmissionClosed = false
	case "drain-not-drained":
		standing.Drained = false
	case "drain-operation":
		standing.OperationId = "different-operation"
	case "drain-token":
		standing.FenceToken = "different-fence"
	case "drain-revision":
		standing.Revision++
	case "drain-provider":
		standing.ProviderInstanceId = "different-instance"
	}
	return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: standing}), nil
}

func (s *restartFenceTestService) Resume(_ context.Context, req *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	s.mu.Lock()
	opID := s.opID
	if req.Msg.GetOperationId() != opID || req.Msg.GetFenceToken() != "fence-7" || req.Msg.GetRevision() != 7 {
		s.mu.Unlock()
		return nil, connect.NewError(connect.CodeAborted, errors.New("unexpected resume identity"))
	}
	if s.failFirstResume {
		s.failFirstResume = false
		s.mu.Unlock()
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("replacement health recovery retained the fence"))
	}
	s.closed = false
	s.mu.Unlock()
	s.record("resume")
	s.mu.Lock()
	s.closed = false
	s.mu.Unlock()
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{}), nil
}

type restartFenceProgressSink struct {
	service *restartFenceTestService
}

type failingFenceLinkRuntimeStore struct {
	scenarioRuntimeStore
	err error
}

func (s failingFenceLinkRuntimeStore) SetStartOperationFenceOperationID(context.Context, string, string) error {
	return s.err
}

func (s restartFenceProgressSink) Publish(event ProgressEvent) {
	if event.Kind == EventStopStarted {
		s.service.record("stop")
	}
}

func TestRestartFencesProviderBeforeStopAndRetainsRunningOwnerOnDrainFailure(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		force bool
	}{
		{mode: "refuse"},
		{mode: "timeout"},
		{mode: "prepare-operation"},
		{mode: "prepare-open"},
		{mode: "drain-open"},
		{mode: "drain-not-drained"},
		{mode: "drain-operation"},
		{mode: "drain-token"},
		{mode: "drain-revision"},
		{mode: "drain-provider"},
		{mode: "drain-admitted"},
		{mode: "drain-remaining"},
		{mode: "drain-quiescence"},
		{mode: "drain-blocker"},
		{mode: "prepare-drained-admitted"},
		{mode: "emergency-prepare-operation", force: true},
		{mode: "emergency-prepare-open", force: true},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			runner, service, events := restartFenceTestRunner(t, tc.mode)
			runner.WithProgressSink(restartFenceProgressSink{service: service})
			if tc.mode == "prepare-drained-admitted" {
				service.prepareDrained = true
				service.admitting = 1
			}

			ctx := t.Context()
			var cancel context.CancelFunc
			if tc.mode == "timeout" {
				ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
			}
			opts := StartOptions{ForceLifecycle: tc.force}
			if tc.force {
				opts.LifecycleOverrideReason = "test drain-only override"
			}
			_, err := runner.RestartContext(ctx, "alpha", opts)
			if err == nil {
				t.Fatal("Restart() error = nil, want provider drain refusal")
			}
			if tc.mode == "timeout" && !strings.Contains(err.Error(), "deadline_exceeded") {
				t.Fatalf("Restart() error = %v, want provider drain deadline", err)
			}
			if service.opID == "" {
				t.Fatal("provider Prepare did not receive an operation ID")
			}
			got := snapshotRestartFenceEvents(service, events)
			want := []string{"prepare"}
			if tc.mode == "refuse" || tc.mode == "timeout" || strings.HasPrefix(tc.mode, "drain-") {
				want = append(want, "drain")
			}
			if len(got) != len(want) {
				t.Fatalf("lifecycle events = %v, want %v and no Stop", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("lifecycle events = %v, want %v and no Stop", got, want)
				}
			}
			store, err := runner.runtimeDeps().runtimeRegistry(context.Background(), runner.Home)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{Scenario: "alpha", Statuses: scenarioruntime.ActiveInstanceStatuses()})
			if err != nil {
				t.Fatal(err)
			}
			if len(instances) != 1 || instances[0].Status != scenarioruntime.StatusRunning {
				t.Fatalf("provider runtime after failed drain = %+v, want original running owner", instances)
			}
		})
	}
}

func TestRestartDrainsBeforeStopAndResumesSameFenceAfterStart(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	result, err := runner.Restart("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Restart(): %v", err)
	}
	if result.Health != "running" {
		t.Fatalf("Restart() health = %q, want running for fixture without health checks", result.Health)
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 4 || got[0] != "prepare" || got[1] != "drain" || got[2] != "stop" || got[3] != "resume" {
		t.Fatalf("lifecycle events = %v, want Prepare, Drain, Stop, Resume", got)
	}
	op := latestStartOperation(t, runner.Home, "alpha")
	if op.FenceOperationID != service.operationID() || op.OperationID == op.FenceOperationID {
		t.Fatalf("durable attempt/fence IDs = %q/%q, want distinct IDs linked to provider fence %q", op.OperationID, op.FenceOperationID, service.operationID())
	}
}

func TestRestartPersistsFenceOwnerBeforeBlockedDrain(t *testing.T) {
	runner, service, _ := restartFenceTestRunner(t, "allow")
	service.drainStarted = make(chan struct{})
	service.releaseDrain = make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(service.releaseDrain) })
	done := make(chan error, 1)
	go func() {
		_, err := runner.RestartContext(t.Context(), "alpha", StartOptions{})
		done <- err
	}()
	select {
	case <-service.drainStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("restart did not reach Drain")
	}
	op := latestStartOperation(t, runner.Home, "alpha")
	if op.FenceOperationID != service.operationID() {
		t.Fatalf("running start operation fence link = %q, want %q while Drain is blocked", op.FenceOperationID, service.operationID())
	}
	if op.CurrentStep != startStepProviderFence {
		t.Fatalf("running start operation step = %q, want %q while Drain is blocked", op.CurrentStep, startStepProviderFence)
	}
	view := EvaluateStartOperation(op, func(int) bool { return true }, time.Now(), nil)
	if view.FenceOperationID != service.operationID() || view.ETAKnown {
		t.Fatalf("in-flight fence view = %+v, want linked fence and unknown ETA", view)
	}
	releaseOnce.Do(func() { close(service.releaseDrain) })
	if err := <-done; err != nil {
		t.Fatalf("RestartContext() after releasing Drain: %v", err)
	}
}

func TestProviderFenceLinkFailureStopsBeforeDrain(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	wantErr := errors.New("registry link unavailable")
	_, err := runner.prepareProviderFenceWithLink(t.Context(), scenario.Scenario{
		Slug: "alpha", Variant: "live",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
	}, false, "", "operation-1", nil, nil, func(string) error { return wantErr })
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("prepareProviderFenceWithLink() error = %v, want link failure", err)
	}
	if got := snapshotRestartFenceEvents(service, events); strings.Join(got, ",") != "prepare" {
		t.Fatalf("provider events = %v, want Prepare only when durable link fails", got)
	}
}

func TestEmergencyRestartRecoversExactSuccessfulFenceOwnerAcrossUnlinkedStart(t *testing.T) {
	runner, service, _ := restartFenceTestRunner(t, "allow")
	service.mu.Lock()
	service.opID = "retained-fence"
	service.closed = true
	service.statusDrained = true
	service.prepareDrained = true
	service.mu.Unlock()

	store, err := runner.runtimeDeps().runtimeRegistry(t.Context(), runner.Home)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationRestart, FenceOperationID: "retained-fence",
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	owner.Status = scenarioruntime.StartOperationStatusSucceeded
	if _, err := store.UpdateStartOperation(t.Context(), owner); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	newer, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: "start",
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	newer.Status = scenarioruntime.StartOperationStatusSucceeded
	if _, err := store.UpdateStartOperation(t.Context(), newer); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	latest, err := store.GetLatestStartOperation(t.Context(), "alpha", "live")
	if closeErr := store.Close(); err != nil || closeErr != nil {
		t.Fatalf("load latest attempt: %v (close %v)", err, closeErr)
	}

	var linked string
	fence, err := runner.prepareProviderFenceWithLink(t.Context(), scenario.Scenario{
		Slug: "alpha", Variant: "live",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
	}, true, "verified durable fence owner retained across process-local provider transition", "", &latest, nil, func(id string) error {
		linked = id
		return nil
	})
	if err != nil {
		t.Fatalf("prepare exact retained fence recovery: %v", err)
	}
	if fence == nil || fence.recoveryFenceOperationID != "retained-fence" || linked != "retained-fence" {
		t.Fatalf("recovery fence = %+v, linked=%q; want exact retained fence owner", fence, linked)
	}
}

func TestRestartFenceLinkFailureLeavesOriginalOwnerServing(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	wantErr := errors.New("registry link unavailable")
	originalRegistry := runner.runtimeDeps().runtimeRegistry
	runner.deps.runtimeRegistry = func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
		store, err := originalRegistry(ctx, home)
		if err != nil {
			return nil, err
		}
		return failingFenceLinkRuntimeStore{scenarioRuntimeStore: store, err: wantErr}, nil
	}
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	_, err := runner.RestartContext(t.Context(), "alpha", StartOptions{})
	if err == nil || !strings.Contains(err.Error(), wantErr.Error()) {
		t.Fatalf("RestartContext() error = %v, want durable fence-link failure", err)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare" {
		t.Fatalf("provider/process events = %s, want Prepare only before Drain and Stop", got)
	}
	store, err := originalRegistry(context.Background(), runner.Home)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{Scenario: "alpha", Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].Status != scenarioruntime.StatusRunning {
		t.Fatalf("runtime after link failure = %+v, want original serving instance", instances)
	}
	op := latestStartOperation(t, runner.Home, "alpha")
	if !op.IsTerminal() || op.Status != scenarioruntime.StartOperationStatusFailed || op.FenceOperationID != "" {
		t.Fatalf("failed owner operation = %+v, want failed attempt with no fabricated fence link", op)
	}
}

func TestRestartWaitsForAdmittedWorkBeyondProbeTimeout(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "slow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if _, err := runner.RestartContext(ctx, "alpha", StartOptions{}); err != nil {
		t.Fatalf("drain must follow operation context, not the ten-second probe timeout: %v", err)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare,drain,stop,resume" {
		t.Fatalf("slow work was not drained before replacement: %s", got)
	}
}

func TestRestartQuiescesProviderOwnedWorkWithNoManagedComponents(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "quiesce-zero-components")
	service.requiresQuiescence = true
	runner.WithProgressSink(restartFenceProgressSink{service: service})

	if _, err := runner.RestartContext(t.Context(), "alpha", StartOptions{}); err != nil {
		t.Fatalf("RestartContext() error = %v, want success after owner quiescence", err)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare,quiesce,drain,stop,resume" {
		t.Fatalf("lifecycle events = %s, want quiescence and final drain before stop", got)
	}
}

func TestOrdinaryForcedReplacementLinksFenceBeforeStop(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	if _, err := runner.Start("alpha", StartOptions{ForceSetup: true}); err != nil {
		t.Fatalf("forced ordinary replacement Start(): %v", err)
	}
	op := latestStartOperation(t, runner.Home, "alpha")
	if op.FenceOperationID == "" || op.FenceOperationID != service.operationID() || op.OperationID == op.FenceOperationID {
		t.Fatalf("start operation/fence IDs = %q/%q, provider fence=%q", op.OperationID, op.FenceOperationID, service.operationID())
	}
	if got := snapshotRestartFenceEvents(service, events); strings.Join(got, ",") != "prepare,drain,stop,resume" {
		t.Fatalf("forced replacement events = %v, want Prepare, Drain, Stop, Resume", got)
	}
}

func TestRestartDrainTimeoutCanRetrySameServingOwner(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "timeout")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if _, err := runner.RestartContext(ctx, "alpha", StartOptions{}); err == nil {
		t.Fatal("first Restart() error = nil, want drain timeout")
	}
	firstOperationID := service.operationID()
	if firstOperationID == "" {
		t.Fatal("first Prepare did not receive an operation ID")
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 2 || got[0] != "prepare" || got[1] != "drain" {
		t.Fatalf("first lifecycle events = %v, want Prepare, Drain and no Stop", got)
	}

	// The provider deliberately retains the fence after timeout. A retry against
	// the same runtime instance must reuse its operation ID and complete without
	// manually reopening admission or clearing the retained fence.
	service.setDrainMode("allow")
	if _, err := runner.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("same-owner retry Restart(): %v", err)
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 6 ||
		got[0] != "prepare" || got[1] != "drain" || got[2] != "prepare" ||
		got[3] != "drain" || got[4] != "stop" || got[5] != "resume" {
		t.Fatalf("retry lifecycle events = %v, want same-fence Prepare, Drain, Prepare, Drain, Stop, Resume", got)
	}
	if got := service.operationID(); got != firstOperationID {
		t.Fatalf("retry operation ID = %q, first operation ID = %q", got, firstOperationID)
	}
	op := latestStartOperation(t, runner.Home, "alpha")
	if op.FenceOperationID != firstOperationID || op.OperationID == op.FenceOperationID {
		t.Fatalf("durable attempt/fence IDs = %q/%q, want distinct attempt linked to retained fence %q", op.OperationID, op.FenceOperationID, firstOperationID)
	}
}

func TestPrepareProviderFenceRefusesUnlinkedClosedFence(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	service.mu.Lock()
	service.opID = "foreign-maintenance-operation"
	service.closed = true
	service.mu.Unlock()
	_, err := runner.prepareProviderFence(context.Background(), scenario.Scenario{Slug: "alpha", Variant: "live", Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}, false, "", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "without an exact retained start-operation owner") {
		t.Fatalf("prepareProviderFence error = %v; want refusal for unlinked closed fence", err)
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
		t.Fatalf("unlinked closed fence caused provider mutation calls: %v", got)
	}
}

func TestPrepareProviderFenceFindsExactStartOwnerBehindDivergentLatestAttempt(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	service.mu.Lock()
	service.opID = "retained-fence"
	service.closed, service.statusDrained, service.prepareDrained = true, true, true
	service.mu.Unlock()

	store, err := runner.runtimeDeps().runtimeRegistry(t.Context(), runner.Home)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationStart, FenceOperationID: "retained-fence",
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	owner.Status = scenarioruntime.StartOperationStatusFailed
	if _, err := store.UpdateStartOperation(t.Context(), owner); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	latest, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationRestart,
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	latest.Status = scenarioruntime.StartOperationStatusFailed
	if _, err := store.UpdateStartOperation(t.Context(), latest); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	prior, err := runner.latestStartOperationSnapshot(t.Context(), "alpha", "live")
	if err != nil || prior == nil || prior.FenceOperationID != "retained-fence" || prior.Operation != lifecycleOperationStart {
		t.Fatalf("retained prior owner = %+v, %v; want exact linked start owner behind the newer unlinked attempt", prior, err)
	}
	var linked string
	fence, err := runner.prepareProviderFenceWithLink(t.Context(), scenario.Scenario{
		Slug: "alpha", Variant: "live",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
	}, false, "", "", prior, nil, func(id string) error { linked = id; return nil })
	if err != nil {
		t.Fatalf("recover exact retained start owner: %v", err)
	}
	if fence == nil || fence.operationID != "retained-fence" || linked != "retained-fence" {
		t.Fatalf("fence=%+v link=%q; want exact retained owner", fence, linked)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare" {
		t.Fatalf("provider lifecycle events = %s, want one exact Prepare", got)
	}
}

func TestPrepareProviderFenceAdoptsOnlyCurrentGenerationProcessLocalFence(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	view, err := runner.lookupRegistryRuntime(t.Context(), scenario.Scenario{Slug: "alpha", Variant: "live"})
	if err != nil || !view.Authoritative {
		t.Fatalf("runtime view = %+v, %v; want current authoritative fixture", view, err)
	}
	fenceID := "restart-" + view.Instance.InstanceID
	service.mu.Lock()
	service.providerInstanceID = view.Instance.InstanceID
	service.fencePersistence = commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL
	service.opID, service.closed = fenceID, true
	service.statusDrained, service.prepareDrained = true, true
	service.mu.Unlock()

	store, err := runner.runtimeDeps().runtimeRegistry(t.Context(), runner.Home)
	if err != nil {
		t.Fatal(err)
	}
	oldOwner, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationStart,
		FenceOperationID: "restart-old-provider-generation",
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	oldOwner.Status = scenarioruntime.StartOperationStatusFailed
	if _, err := store.UpdateStartOperation(t.Context(), oldOwner); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	interrupted, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationRestart,
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	interrupted.Status = scenarioruntime.StartOperationStatusFailed
	if _, err := store.UpdateStartOperation(t.Context(), interrupted); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	prior, err := runner.latestStartOperationSnapshot(t.Context(), "alpha", "live")
	if err != nil || prior == nil || prior.FenceOperationID == fenceID {
		t.Fatalf("prior operation = %+v, %v; want the divergent old generation link", prior, err)
	}
	recorder, err := runner.beginStartOperationRecord("alpha", StartOptions{Context: t.Context(), Variant: "live", Operation: lifecycleOperationRestart})
	if err != nil {
		t.Fatal(err)
	}
	defer recorder.close()
	fence, err := runner.prepareProviderFenceWithLink(t.Context(), scenario.Scenario{
		Slug: "alpha", Variant: "live",
		Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
	}, false, "", "", prior, nil, recorder.linkProviderFence)
	if err != nil {
		t.Fatalf("adopt current generation process-local fence: %v", err)
	}
	if fence == nil || fence.operationID != fenceID || recorder.op.FenceOperationID != fenceID {
		t.Fatalf("fence=%+v recorded link=%q; want new attempt linked to %q", fence, recorder.op.FenceOperationID, fenceID)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare" {
		t.Fatalf("provider lifecycle events = %s, want Prepare before any drain/stop", got)
	}
}

func TestPrepareProviderFenceRefusesAmbiguousProcessLocalFenceOwnership(t *testing.T) {
	for _, tc := range []struct {
		name            string
		persistence     commonv1.LifecycleFencePersistence
		providerMatches bool
		priorStatus     string
	}{
		{name: "durable fence without retained owner", persistence: commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE, providerMatches: true, priorStatus: scenarioruntime.StartOperationStatusFailed},
		{name: "wrong provider generation", persistence: commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL, providerMatches: false, priorStatus: scenarioruntime.StartOperationStatusFailed},
		{name: "concurrent active attempt", persistence: commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL, providerMatches: true, priorStatus: scenarioruntime.StartOperationStatusRunning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, service, events := restartFenceTestRunner(t, "allow")
			view, err := runner.lookupRegistryRuntime(t.Context(), scenario.Scenario{Slug: "alpha", Variant: "live"})
			if err != nil || !view.Authoritative {
				t.Fatalf("runtime view = %+v, %v", view, err)
			}
			fenceID := "restart-" + view.Instance.InstanceID
			service.mu.Lock()
			service.providerInstanceID = view.Instance.InstanceID
			if !tc.providerMatches {
				service.providerInstanceID = "other-generation"
			}
			service.fencePersistence = tc.persistence
			service.opID, service.closed = fenceID, true
			service.mu.Unlock()
			prior := &scenarioruntime.StartOperation{
				Scenario: "alpha", Variant: "live", Operation: lifecycleOperationRestart,
				Status: tc.priorStatus, FenceOperationID: "restart-previous-generation",
			}
			links := 0
			_, err = runner.prepareProviderFenceWithLink(t.Context(), scenario.Scenario{
				Slug: "alpha", Variant: "live",
				Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}},
			}, false, "", "", prior, nil, func(string) error { links++; return nil })
			if err == nil || !strings.Contains(err.Error(), "without an exact retained start-operation owner") {
				t.Fatalf("prepare error = %v; want fail-closed ambiguous ownership", err)
			}
			if links != 0 {
				t.Fatalf("link callback count = %d; want no ownership claim", links)
			}
			if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
				t.Fatalf("provider lifecycle events = %v; want no Prepare or Drain", got)
			}
		})
	}
}

func TestForceRecoveryRequiresTerminalExactOwnerForUnlinkedDerivedFence(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	service.mu.Lock()
	service.opID = "restart-provider-instance"
	service.closed = true
	service.mu.Unlock()
	item := scenario.Scenario{Slug: "alpha", Variant: "live", Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}
	running := &scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationRestart,
		Status: scenarioruntime.StartOperationStatusRunning,
	}
	_, err := runner.prepareProviderFence(context.Background(), item, true, "test stale-process override", "", running, nil)
	if err == nil || !strings.Contains(err.Error(), "without an exact retained start-operation owner") {
		t.Fatalf("force recovery error = %v, want refusal while exact owner is still running", err)
	}
	if got := snapshotRestartFenceEvents(service, events); len(got) != 0 {
		t.Fatalf("running-owner recovery calls = %v, want read-only refusal", got)
	}
}

func TestForceRestartRecoversTerminalUnlinkedFenceAndSkipsStaleRemainingWork(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	service.mu.Lock()
	service.opID = "restart-provider-instance"
	service.closed = true
	service.remaining = 1
	service.mu.Unlock()
	store := openWaitTestStore(t, runner.Home)
	prior, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{
		Scenario: "alpha", Variant: "live", Operation: lifecycleOperationRestart, StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed prior restart: %v", err)
	}
	prior, err = store.MarkStartOperationAbandoned(t.Context(), prior.OperationID, "legacy restart interrupted before fence link")
	if err != nil {
		t.Fatalf("abandon prior restart: %v", err)
	}
	if !prior.IsTerminal() || prior.FenceOperationID != "" {
		t.Fatalf("prior operation = %+v, want terminal unlinked restart", prior)
	}
	latest, latestErr := runner.latestStartOperationSnapshot(t.Context(), "alpha", "live")
	if latestErr != nil || latest == nil || latest.OperationID != prior.OperationID {
		t.Fatalf("latest recovery owner = %+v, err=%v; want seeded terminal operation %q", latest, latestErr, prior.OperationID)
	}
	result, err := runner.RestartContext(t.Context(), "alpha", StartOptions{
		ForceLifecycle: true, LifecycleOverrideReason: "verified PID reuse: tracked PID 123 exited before Chrome renderer reuse",
	})
	if err != nil {
		t.Fatalf("force restart recovery: %v", err)
	}
	if result.Scenario.Slug != "alpha" {
		t.Fatalf("restart result = %+v", result)
	}
	operation := latestStartOperation(t, runner.Home, "alpha")
	if operation.Status != scenarioruntime.StartOperationStatusSucceeded || operation.FenceOperationID != "restart-provider-instance" {
		t.Fatalf("recovered operation = %+v, want successful operation linked to exact retained fence", operation)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare,stop,resume" {
		t.Fatalf("recovery events = %s, want exact Prepare, Stop, Resume and no stale-work Drain", got)
	}
}

func TestForceRestartDoesNotBypassCombinedAdmittingAndRemainingCounts(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	service.remaining = 1
	service.admitting = 1
	if _, err := runner.RestartContext(t.Context(), "alpha", StartOptions{
		ForceLifecycle: true, LifecycleOverrideReason: "test explicit risk acceptance",
	}); err != nil {
		t.Fatalf("restart after normal drain: %v", err)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare,drain,stop,resume" {
		t.Fatalf("combined work counts were bypassed: events=%s, want Drain before Stop", got)
	}
}

// A handler that never returns holds the provider's drain forever. A restart
// that carries the operator's explicit risk acceptance must give up after the
// bounded wait, record what was stuck and why it proceeded, and replace the
// provider through the normal stop path. A normal restart keeps refusing.
func TestForceRestartGivesUpOnStuckWorkAfterBoundedDrain(t *testing.T) {
	previous := emergencyDrainTimeout
	emergencyDrainTimeout = 150 * time.Millisecond
	t.Cleanup(func() { emergencyDrainTimeout = previous })

	var logs bytes.Buffer
	runner, service, events := restartFenceTestRunner(t, "timeout")
	runner.Logger = slog.New(slog.NewTextHandler(&logs, nil))
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	service.admitting = 2
	service.remaining = 2 // a provider that counts admitted requests in remaining too
	service.work = []*commonv1.LifecycleWorkItem{{Id: "POST /api/v1/catalog/run", Kind: "http_request", Status: "running 9m57s since 2026-11-14T00:11:30Z"}}

	if _, err := runner.RestartContext(t.Context(), "alpha", StartOptions{
		ForceLifecycle: true, LifecycleOverrideReason: "abandoned fence, hung handler",
	}); err != nil {
		t.Fatalf("force restart with stuck work: %v", err)
	}
	if got := strings.Join(snapshotRestartFenceEvents(service, events), ","); got != "prepare,drain,stop,resume" {
		t.Fatalf("events = %s, want a bounded Drain followed by the normal Stop and Resume", got)
	}
	for _, want := range []string{"gave up on stuck work", "abandoned fence, hung handler", "POST /api/v1/catalog/run", "http_request", "admitting=2"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("lifecycle log does not record %q:\n%s", want, logs.String())
		}
	}
}

func TestNormalRestartKeepsWaitingOnStuckWorkAndForceNeverWaivesBlockers(t *testing.T) {
	previous := emergencyDrainTimeout
	emergencyDrainTimeout = 100 * time.Millisecond
	t.Cleanup(func() { emergencyDrainTimeout = previous })

	for _, tc := range []struct {
		name     string
		force    bool
		blockers []*commonv1.LifecycleBlocker
	}{
		{name: "normal restart", force: false},
		{name: "force with a blocking blocker", force: true, blockers: []*commonv1.LifecycleBlocker{{Code: "owner-job", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, service, events := restartFenceTestRunner(t, "timeout")
			runner.WithProgressSink(restartFenceProgressSink{service: service})
			service.admitting, service.remaining = 1, 1
			service.statusBlockers = tc.blockers
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			opts := StartOptions{ForceLifecycle: tc.force}
			if tc.force {
				opts.LifecycleOverrideReason = "operator risk acceptance"
			}
			if _, err := runner.RestartContext(ctx, "alpha", opts); err == nil {
				t.Fatal("restart replaced a provider that still had stuck work")
			}
			for _, event := range snapshotRestartFenceEvents(service, events) {
				if event == "stop" {
					t.Fatalf("events include stop; the provider must stay serving: %v", snapshotRestartFenceEvents(service, events))
				}
			}
		})
	}
}

func TestRestartRecoversClosedFenceLinkedToFailedReplacementAcrossNewInstance(t *testing.T) {
	runner, service, events := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	service.failFirstResume = true
	if _, err := runner.Restart("alpha", StartOptions{}); err == nil {
		t.Fatal("first Restart() error = nil, want retained-fence failure")
	}
	first := latestStartOperation(t, runner.Home, "alpha")
	if first.FenceOperationID == "" || first.Status != scenarioruntime.StartOperationStatusFailed {
		t.Fatalf("first failed replacement record = %+v, want retained fence link and failed attempt", first)
	}
	firstFenceID := service.operationID()
	if first.FenceOperationID != firstFenceID {
		t.Fatalf("first record fence link = %q, provider fence = %q", first.FenceOperationID, firstFenceID)
	}
	registerRestartFenceTestProvider(t, runner, service.port)
	// Intervening attempts may fail before reaching fence negotiation. More
	// attempts than ordinary history retains must still preserve the exact
	// first linked owner for the closed fence in the replacement generation.
	prepareFailure := errors.New("replacement source temporarily unavailable")
	runner.Engagements = &fakeEngagementResolver{err: prepareFailure}
	for i := 0; i < scenarioruntime.StartOperationKeepTerminal+2; i++ {
		if _, err := runner.Restart("alpha", StartOptions{}); !errors.Is(err, prepareFailure) {
			t.Fatalf("preparation retry %d: %v", i, err)
		}
		failed := latestStartOperation(t, runner.Home, "alpha")
		if failed.OperationID == first.OperationID || failed.FenceOperationID != "" || failed.Status != scenarioruntime.StartOperationStatusFailed {
			t.Fatalf("preparation retry %d record = %+v", i, failed)
		}
	}
	runner.Engagements = nil
	// A caller lost before preparation emits no terminal event. The next
	// admitted attempt supersedes that durable running record as abandoned.
	store := openWaitTestStore(t, runner.Home)
	lost, err := store.BeginStartOperation(t.Context(), scenarioruntime.StartOperation{Scenario: "alpha", Operation: lifecycleOperationRestart})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := runner.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("retry from exact retained fence: %v", err)
	}
	second := latestStartOperation(t, runner.Home, "alpha")
	if second.OperationID == first.OperationID {
		t.Fatalf("retry reused terminal start attempt ID %q", second.OperationID)
	}
	if second.FenceOperationID != firstFenceID {
		t.Fatalf("retry fence link = %q, want retained fence %q", second.FenceOperationID, firstFenceID)
	}
	interrupted, err := store.GetStartOperation(t.Context(), lost.OperationID)
	if err != nil || interrupted.Status != scenarioruntime.StartOperationStatusAbandoned {
		t.Fatalf("lost preparation was not superseded as abandoned: %+v, %v", interrupted, err)
	}
	got := snapshotRestartFenceEvents(service, events)
	if len(got) != 7 || strings.Join(got, ",") != "prepare,drain,stop,prepare,drain,stop,resume" {
		t.Fatalf("restart/recovery lifecycle events = %v", got)
	}
}

func TestRestartRecoversRetainedClosedFenceAfterWarmStart(t *testing.T) {
	runner, service, _ := restartFenceTestRunner(t, "allow")
	runner.WithProgressSink(restartFenceProgressSink{service: service})
	service.failFirstResume = true
	if _, err := runner.Restart("alpha", StartOptions{}); err == nil {
		t.Fatal("first replacement unexpectedly resumed its fence")
	}
	owner := latestStartOperation(t, runner.Home, "alpha")
	registerRestartFenceTestProvider(t, runner, service.port)
	warm, err := runner.Start("alpha", StartOptions{})
	if err != nil || !warm.AlreadyRunning {
		t.Fatalf("warm start under retained maintenance fence = %+v, %v", warm, err)
	}
	warmReceipt := latestStartOperation(t, runner.Home, "alpha")
	if warmReceipt.Status != scenarioruntime.StartOperationStatusSucceeded || warmReceipt.FenceOperationID != "" {
		t.Fatalf("warm receipt = %+v; want succeeded without a new fence", warmReceipt)
	}
	if _, err := runner.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("exact retained fence became unrecoverable after warm start: %v", err)
	}
	recovered := latestStartOperation(t, runner.Home, "alpha")
	if recovered.FenceOperationID != owner.FenceOperationID {
		t.Fatalf("recovered fence = %s, want exact %s", recovered.FenceOperationID, owner.FenceOperationID)
	}
}

func restartFenceTestRunner(t *testing.T, drainMode string) (*Runner, *restartFenceTestService, *[]string) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Version:   "1.0.0",
		Service:   scenario.ServiceMetadata{Name: "alpha", DisplayName: "Lifecycle alpha", Version: "0.1.0"},
		Ports:     map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "22000-22010"}},
		Lifecycle: scenario.Lifecycle{Version: "2.0.0"},
	})
	events := []string{}
	service := &restartFenceTestService{events: &events, drainMode: drainMode}
	_, handler := commonv1connect.NewLifecycleMaintenanceServiceHandler(service)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	_, portText, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	service.port = port

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "provider-fence-test-boot"}, nil
		}
		// The listener belongs to this test's provider RPC server. Stop must not
		// signal it; the current-host registry lease is the only stop target.
		deps.listeningPIDs = func(int) ([]int, error) { return nil, nil }
	})
	store, err := runner.runtimeDeps().runtimeRegistry(context.Background(), runner.Home)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := store.CreateInstance(context.Background(), scenarioruntime.Instance{
		Scenario: "alpha", Variant: "live", Status: scenarioruntime.StatusRunning,
		HostBootID: "provider-fence-test-boot",
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	now := time.Now().UTC()
	claim, err := store.AcquirePortClaim(context.Background(), scenarioruntime.PortClaim{
		InstanceID: instance.InstanceID, Scenario: "alpha", Variant: "live", PortName: "api", EnvVar: "API_PORT",
		Port: port, Status: scenarioruntime.ClaimStatusBound, LastBoundAt: &now,
	})
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if claim.Status != scenarioruntime.ClaimStatusBound {
		t.Fatalf("provider port claim status = %q, want bound", claim.Status)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	return runner, service, &events
}

func registerRestartFenceTestProvider(t *testing.T, runner *Runner, port int) {
	t.Helper()
	ctx := context.Background()
	store, err := runner.runtimeDeps().runtimeRegistry(ctx, runner.Home)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Scenario: "alpha", Variant: "live", Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil || len(instances) != 1 {
		t.Fatalf("active replacement instances = %d, err=%v; want one", len(instances), err)
	}
	pid := os.Getpid()
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{InstanceID: instances[0].InstanceID, PID: &pid, Status: "running"}); err != nil {
		t.Fatalf("register replacement test process: %v", err)
	}
	now := time.Now().UTC()
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID: instances[0].InstanceID, Scenario: "alpha", Variant: "live", PortName: "api", EnvVar: "API_PORT",
		Port: port, Status: scenarioruntime.ClaimStatusBound, LastBoundAt: &now,
	}); err != nil {
		t.Fatalf("register replacement provider API port: %v", err)
	}
}

func snapshotRestartFenceEvents(service *restartFenceTestService, events *[]string) []string {
	service.mu.Lock()
	defer service.mu.Unlock()
	return append([]string(nil), (*events)...)
}

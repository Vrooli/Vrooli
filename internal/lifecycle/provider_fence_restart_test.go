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
	prepareDrained     bool
	statusDrained      bool
	requiresQuiescence bool
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
	provider, instanceID, operationID := "alpha", "provider-instance", req.Msg.GetOperationId()
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
		ManagedComponents: s.components, Admitting: s.admitting,
		Drained: s.prepareDrained, ComponentStopReady: s.prepareDrained,
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
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: &commonv1.LifecycleStanding{
		ManagedComponents: s.components, Admitting: s.admitting,
		Blockers: blockers,
		Drained:  s.statusDrained, ComponentStopReady: s.statusDrained,
		Provider: "alpha", ProviderInstanceId: "provider-instance", OperationId: s.opID,
		FenceToken: "fence-7", Revision: 7, AdmissionClosed: s.closed, InventoryComplete: true,
		QuiescencePending: s.closed && s.requiresQuiescence,
	}}), nil
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
	s.mu.Lock()
	mode := strings.TrimPrefix(s.drainMode, "emergency-")
	opID := s.opID
	requiresQuiescence := s.requiresQuiescence
	s.mu.Unlock()
	if req.Msg.GetOperationId() != opID || req.Msg.GetFenceToken() != "fence-7" || req.Msg.GetRevision() != 7 {
		return nil, connect.NewError(connect.CodeAborted, errors.New("unexpected drain identity"))
	}
	if mode == "quiesce-zero-components" && req.Msg.GetQuiesceComponents() && requiresQuiescence {
		return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing: &commonv1.LifecycleStanding{
			Provider: "alpha", ProviderInstanceId: "provider-instance", OperationId: opID,
			FenceToken: "fence-7", Revision: 7, AdmissionClosed: true, InventoryComplete: true,
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

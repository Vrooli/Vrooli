package lifecycle

import (
	"context"
	"net"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

func TestValidateStatusStandingExplainsIncompleteInventoryWithoutProviderStrings(t *testing.T) {
	standing := &commonv1.LifecycleStanding{
		Provider:           "device-control",
		ProviderInstanceId: "instance-1",
		InventoryComplete:  false,
		Blockers: []*commonv1.LifecycleBlocker{{
			Code:        "private_token_ABC123",
			Message:     "private path /home/operator/runtime/helper.sock",
			Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN,
		}},
		Work: []*commonv1.LifecycleWorkItem{{Kind: "private_work_kind_SECRET99", Status: "private_work_status"}},
	}

	err := validateStatusStanding("device-control", standing)
	if err == nil {
		t.Fatal("validateStatusStanding() error = nil, want incomplete inventory error")
	}
	message := err.Error()
	for _, want := range []string{
		"incomplete lifecycle inventory",
		"inventory counts: blockers=1",
		"unknown=1",
		"work=1",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("error %q does not contain %q", message, want)
		}
	}
	for _, secret := range []string{"private_token_ABC123", "/home/operator/runtime/helper.sock", "private_work_kind_SECRET99", "private_work_status"} {
		if strings.Contains(message, secret) {
			t.Fatalf("error leaked provider-authored string %q: %q", secret, message)
		}
	}
}

func TestLifecycleFenceRejectsNegativeWorkCountersAndEmergencyOverride(t *testing.T) {
	standing := &commonv1.LifecycleStanding{
		Provider: "alpha", ProviderInstanceId: "instance-1", InventoryComplete: true,
		AdmissionClosed: true,
		Blockers:        []*commonv1.LifecycleBlocker{{Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE}},
	}
	if !onlyOverrideableLifecycleBlockers(standing) {
		t.Fatal("valid overrideable blocker standing should remain eligible for explicit emergency override")
	}
	for _, field := range []string{"admitting", "remaining"} {
		t.Run(field, func(t *testing.T) {
			candidate := &commonv1.LifecycleStanding{
				Provider: "alpha", ProviderInstanceId: "instance-1", InventoryComplete: true,
				AdmissionClosed: true,
				Blockers:        []*commonv1.LifecycleBlocker{{Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE}},
			}
			if field == "admitting" {
				candidate.Admitting = -1
			} else {
				candidate.Remaining = -1
			}
			if onlyOverrideableLifecycleBlockers(candidate) {
				t.Fatalf("negative %s count passed emergency override eligibility", field)
			}
			if err := validateStatusStanding("alpha", candidate); err == nil || !strings.Contains(err.Error(), "negative admitted or remaining") {
				t.Fatalf("validateStatusStanding() error = %v, want negative count rejection", err)
			}
		})
	}
}

func TestLifecycleInventoryDiagnosticIsOrderIndependentAndBounded(t *testing.T) {
	first := &commonv1.LifecycleStanding{
		Blockers: []*commonv1.LifecycleBlocker{
			{Code: "first-secret", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING},
			{Code: "second-secret", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN},
		},
		Work:              []*commonv1.LifecycleWorkItem{{Kind: "owner-secret", Status: "active"}},
		ManagedComponents: []*commonv1.LifecycleManagedComponent{{Name: "component-secret", Generation: "generation-secret"}},
	}
	second := &commonv1.LifecycleStanding{
		Blockers: []*commonv1.LifecycleBlocker{
			{Code: "second-secret", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN},
			{Code: "first-secret", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING},
		},
		Work:              []*commonv1.LifecycleWorkItem{{Kind: "another-secret", Status: "unknown"}},
		ManagedComponents: []*commonv1.LifecycleManagedComponent{{Name: "other-component", Generation: "other-generation"}},
	}

	firstDiagnostic := lifecycleInventoryDiagnostic(first)
	secondDiagnostic := lifecycleInventoryDiagnostic(second)
	if firstDiagnostic != secondDiagnostic {
		t.Fatalf("diagnostics differ for equal count snapshots: %q != %q", firstDiagnostic, secondDiagnostic)
	}
	if len(firstDiagnostic) > 200 {
		t.Fatalf("diagnostic length = %d, want at most 200: %q", len(firstDiagnostic), firstDiagnostic)
	}
	for _, secret := range []string{"first-secret", "owner-secret", "component-secret", "generation-secret"} {
		if strings.Contains(firstDiagnostic, secret) {
			t.Fatalf("diagnostic leaked provider string %q: %q", secret, firstDiagnostic)
		}
	}
}

type providerFenceTestService struct {
	commonv1connect.UnimplementedLifecycleMaintenanceServiceHandler
	prepares       int
	resumes        int
	statusStanding *commonv1.LifecycleStanding
}

func (s *providerFenceTestService) Prepare(_ context.Context, req *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) {
	s.prepares++
	return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing: &commonv1.LifecycleStanding{
		Provider:           "other-scenario",
		ProviderInstanceId: "instance-1",
		OperationId:        req.Msg.GetOperationId(),
		FenceToken:         "fence-token",
		Revision:           7,
		AdmissionClosed:    true,
		InventoryComplete:  true,
		Drained:            true,
	}}), nil
}

func (s *providerFenceTestService) Status(context.Context, *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) {
	if s.statusStanding != nil {
		return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: s.statusStanding}), nil
	}
	return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing: &commonv1.LifecycleStanding{
		Provider: "other-scenario", ProviderInstanceId: "instance-1", Phase: commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN,
		InventoryComplete: true,
	}}), nil
}

func (s *providerFenceTestService) Resume(context.Context, *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) {
	s.resumes++
	return connect.NewResponse(&commonv1.LifecycleResumeResponse{}), nil
}

func TestPrepareProviderFenceAllowsRestartWhenScenarioIsStopped(t *testing.T) {
	home := t.TempDir()
	runner := &Runner{Home: home, deps: lifecycleDeps{
		runtimeRegistry: func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		},
	}}

	fence, err := runner.prepareProviderFence(t.Context(), scenario.Scenario{Slug: "stopped-scenario"}, false, "", "operation-1", nil, nil)
	if err != nil {
		t.Fatalf("prepareProviderFence() error = %v, want nil", err)
	}
	if fence != nil {
		t.Fatalf("prepareProviderFence() fence = %#v, want nil for stopped scenario", fence)
	}
}

func TestPrepareProviderFenceKeepsOtherScenarioProviderProtocol(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	service := &providerFenceTestService{}
	_, handler := commonv1connect.NewLifecycleMaintenanceServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	_, portText, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	ownerPID := os.Getpid()
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		Scenario: "other-scenario", Status: scenarioruntime.StatusRunning,
		HostBootID: "test-boot", OwnerPID: &ownerPID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{InstanceID: instance.InstanceID, PID: &ownerPID}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID:  instance.InstanceID,
		Scenario:    instance.Scenario,
		PortName:    "API_PORT",
		EnvVar:      "API_PORT",
		Port:        port,
		Status:      scenarioruntime.ClaimStatusBound,
		LastBoundAt: &now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	runner := &Runner{Home: home, deps: lifecycleDeps{
		runtimeRegistry: func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		},
		hostSession: func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "test-boot"}, nil
		},
	}}
	fence, err := runner.prepareProviderFence(ctx, scenario.Scenario{Slug: "other-scenario", Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}, false, "", "operation-1", nil, nil)
	if err != nil {
		t.Fatalf("prepareProviderFence() error = %v", err)
	}
	if fence == nil || !fence.active || service.prepares != 1 {
		t.Fatalf("provider fence=%+v prepare calls=%d, want active fence and one Prepare", fence, service.prepares)
	}
	if err := fence.resume(ctx); err != nil {
		t.Fatalf("resume provider fence: %v", err)
	}
	if service.resumes != 1 {
		t.Fatalf("Resume calls=%d, want 1", service.resumes)
	}
}

func TestPrepareProviderFenceRefusesIncompleteInventoryBeforePrepare(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	service := &providerFenceTestService{statusStanding: &commonv1.LifecycleStanding{
		Provider:           "other-scenario",
		ProviderInstanceId: "instance-1",
		InventoryComplete:  false,
		Blockers: []*commonv1.LifecycleBlocker{{
			Code:        "opaque_provider_value",
			Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN,
		}},
		Work: []*commonv1.LifecycleWorkItem{{Kind: "opaque_work_value", Status: "unknown"}},
	}}
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

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	ownerPID := os.Getpid()
	instance, err := store.CreateInstance(ctx, scenarioruntime.Instance{Scenario: "other-scenario", Status: scenarioruntime.StatusRunning, HostBootID: "test-boot", OwnerPID: &ownerPID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{InstanceID: instance.InstanceID, PID: &ownerPID}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{InstanceID: instance.InstanceID, Scenario: instance.Scenario, PortName: "API_PORT", EnvVar: "API_PORT", Port: port, Status: scenarioruntime.ClaimStatusBound, LastBoundAt: &now}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{Home: home, deps: lifecycleDeps{
		runtimeRegistry: func(ctx context.Context, home string) (scenarioRuntimeStore, error) {
			return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		},
		hostSession: func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "test-boot"}, nil
		},
	}}
	item := scenario.Scenario{Slug: "other-scenario", Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}
	_, err = runner.prepareProviderFence(ctx, item, false, "", "operation-1", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "incomplete lifecycle inventory") || !strings.Contains(err.Error(), "blockers=1") {
		t.Fatalf("prepareProviderFence() error = %v, want incomplete inventory diagnostic", err)
	}
	if service.prepares != 0 {
		t.Fatalf("Prepare calls = %d, want zero after incomplete Status", service.prepares)
	}
}

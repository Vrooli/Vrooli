package capabilities

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	"web-console/internal/backend"
	internalcaps "web-console/internal/capabilities"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"
)

type fakeService struct {
	actionReq ActionRequest
	actionErr error
}

func (f *fakeService) Resolve(context.Context) Snapshot {
	return Snapshot{}
}

func (f *fakeService) Liveness(context.Context) Snapshot {
	return Snapshot{}
}

func (f *fakeService) RunAction(_ context.Context, req ActionRequest) (ActionResult, error) {
	f.actionReq = req
	if f.actionErr != nil {
		return ActionResult{}, f.actionErr
	}
	return ActionResult{
		Success:      true,
		Status:       "healthy",
		Message:      "lifecycle action completed",
		CapabilityID: req.CapabilityID,
		ActionKind:   req.ActionKind,
		Snapshot: Snapshot{
			Timestamp: "2026-03-17T00:01:00Z",
			Capabilities: []CapabilityState{{
				ID:             req.CapabilityID,
				Name:           "Audio Tools",
				DependencyKind: "scenario",
				DependencySlug: "audio-tools",
				Status:         "available",
				Message:        "scenario is healthy",
			}},
		},
	}, nil
}

func TestConnectHandlerRunAction(t *testing.T) {
	svc := &fakeService{}
	h := NewConnectHandler(Deps{Service: svc})

	resp, err := h.RunAction(context.Background(), connect.NewRequest(&capabilitiesv1.RunActionRequest{
		CapabilityId: "audio-tools",
		ActionKind:   "scenario_start",
	}))
	if err != nil {
		t.Fatalf("RunAction returned error: %v", err)
	}
	if svc.actionReq.CapabilityID != "audio-tools" || svc.actionReq.ActionKind != "scenario_start" {
		t.Fatalf("request = %+v", svc.actionReq)
	}
	if !resp.Msg.GetSuccess() || resp.Msg.GetStatus() != "healthy" {
		t.Fatalf("response = %+v, want success healthy", resp.Msg)
	}
	if len(resp.Msg.GetCapabilities()) != 1 || resp.Msg.GetCapabilities()[0].GetStatus() != "available" {
		t.Fatalf("capabilities = %+v", resp.Msg.GetCapabilities())
	}
}

func TestConnectHandlerGetAndLivenessProjectAllFields(t *testing.T) {
	svc := &fakeService{}
	// Override the zero snapshots through a small wrapper so both projection
	// paths exercise the same transport mapping.
	projected := &projectionService{fakeService: svc}
	h := NewConnectHandler(Deps{Service: projected})
	get, err := h.Get(context.Background(), connect.NewRequest(&capabilitiesv1.GetRequest{}))
	if err != nil || len(get.Msg.Capabilities) != 1 || len(get.Msg.SessionBackends) != 1 || get.Msg.DefaultBackend != "standard" {
		t.Fatalf("get: %#v %v", get, err)
	}
	if get.Msg.Capabilities[0].Id != "audio" || get.Msg.SessionBackends[0].Id != "tmux" {
		t.Fatalf("projection: %#v", get.Msg)
	}
	live, err := h.Liveness(context.Background(), connect.NewRequest(&capabilitiesv1.LivenessRequest{}))
	if err != nil || len(live.Msg.Capabilities) != 1 || live.Msg.Timestamp == "" {
		t.Fatalf("liveness: %#v %v", live, err)
	}
}

type projectionService struct{ *fakeService }

type updateService struct{ *fakeService }

type usageService struct{ *fakeService }

func (*usageService) AgentUsage(_ context.Context, target, agent string) (AgentUsageResult, error) {
	return AgentUsageResult{TargetID: target, AgentID: agent, Match: "likely", Reason: "account not confirmed", Source: "provider", ObservedAt: "2026-10-29T12:00:00Z", Windows: []AgentUsageWindow{{Pool: "codex", WindowMinutes: 10080, UsedPercent: 0, ObservedAt: "2026-10-29T11:00:00Z"}}}, nil
}

func TestConnectHandlerAgentUsagePreservesMatchAndWindows(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &usageService{fakeService: &fakeService{}}})
	response, err := h.AgentUsage(context.Background(), connect.NewRequest(&capabilitiesv1.AgentUsageRequest{TargetId: "local", AgentId: "codex"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TargetId != "local" || response.Msg.AgentId != "codex" || response.Msg.Match != "likely" || response.Msg.Reason == "" || len(response.Msg.Windows) != 1 || response.Msg.Windows[0].WindowMinutes != 10080 || response.Msg.Windows[0].UsedPercent != 0 || response.Msg.Windows[0].ObservedAt != "2026-10-29T11:00:00Z" {
		t.Fatalf("response lost usage evidence: %+v", response.Msg)
	}
	h = NewConnectHandler(Deps{Service: &fakeService{}})
	_, err = h.AgentUsage(context.Background(), connect.NewRequest(&capabilitiesv1.AgentUsageRequest{}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("without reader: %v", err)
	}
}

type harnessService struct{ *fakeService }

func (*harnessService) ListHarnesses(_ context.Context, target string) ([]HarnessDescriptor, error) {
	if target != "local" {
		return nil, fmt.Errorf("%w: %s", ErrTargetNotFound, target)
	}
	return []HarnessDescriptor{{
		AgentID: "claude", Resource: "claude-code", DisplayName: "Claude Code", Icon: "claude",
		UsageWindows:         []HarnessUsageWindow{{ID: "five_hour", WindowMinutes: 300}},
		SupportsContextLimit: true, SupportsUsageMode: true, SupportsModelSelection: true,
	}}, nil
}

func TestConnectHandlerListHarnessesProjectsDescriptors(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &harnessService{fakeService: &fakeService{}}})
	response, err := h.ListHarnesses(context.Background(), connect.NewRequest(&capabilitiesv1.ListHarnessesRequest{TargetId: "local"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TargetId != "local" || len(response.Msg.Harnesses) != 1 {
		t.Fatalf("response: %+v", response.Msg)
	}
	got := response.Msg.Harnesses[0]
	if got.AgentId != "claude" || got.Resource != "claude-code" || got.DisplayName != "Claude Code" || got.Icon != "claude" || !got.SupportsContextLimit || !got.SupportsUsageMode || !got.SupportsModelSelection || got.SupportsProviderSelection || len(got.UsageWindows) != 1 || got.UsageWindows[0].Id != "five_hour" || got.UsageWindows[0].WindowMinutes != 300 {
		t.Fatalf("descriptor lost a field: %+v", got)
	}
	if _, err := h.ListHarnesses(context.Background(), connect.NewRequest(&capabilitiesv1.ListHarnessesRequest{TargetId: "gone"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("an unknown target: %v", err)
	}
	h = NewConnectHandler(Deps{Service: &fakeService{}})
	if _, err := h.ListHarnesses(context.Background(), connect.NewRequest(&capabilitiesv1.ListHarnessesRequest{TargetId: "local"})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("without lister: %v", err)
	}
}

func (*updateService) AgentUpdate(_ context.Context, target, agent string) (AgentUpdateResult, error) {
	return AgentUpdateResult{TargetID: target, AgentID: agent, Installed: "1.0.0", Latest: "1.1.0", Status: "behind", CheckedAt: "2026-10-29T12:00:00Z", ProbeVerified: true}, nil
}

func TestConnectHandlerAgentUpdateProjectsTargetAndProbeEvidence(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &updateService{fakeService: &fakeService{}}})
	response, err := h.AgentUpdate(context.Background(), connect.NewRequest(&capabilitiesv1.AgentUpdateRequest{TargetId: "bridge-node:one", AgentId: "codex"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TargetId != "bridge-node:one" || response.Msg.AgentId != "codex" || response.Msg.Status != "behind" || !response.Msg.ProbeVerified || response.Msg.CheckedAt == "" {
		t.Fatalf("response lost target or observation evidence: %+v", response.Msg)
	}
}

func TestConnectHandlerAgentUpdateUnavailableWithoutReader(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{}})
	_, err := h.AgentUpdate(context.Background(), connect.NewRequest(&capabilitiesv1.AgentUpdateRequest{TargetId: "local", AgentId: "codex"}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v; want unavailable", connect.CodeOf(err))
	}
}

func (p *projectionService) Resolve(context.Context) Snapshot {
	return Snapshot{Timestamp: "now", DefaultBackend: "standard", Capabilities: []CapabilityState{{ID: "audio", Name: "Audio", Features: []string{"stt"}, Status: "available"}}, BackendOptions: []BackendOption{{ID: "tmux", DisplayName: "tmux", Available: true}}}
}

func (p *projectionService) Liveness(context.Context) Snapshot {
	return Snapshot{Timestamp: "live", Capabilities: []CapabilityState{{ID: "audio", Status: "available"}}}
}

func TestConnectHandlerRunActionInvalidArgument(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{actionErr: errors.New("unsupported action")}})

	_, err := h.RunAction(context.Background(), connect.NewRequest(&capabilitiesv1.RunActionRequest{
		CapabilityId: "audio-tools",
		ActionKind:   "operator_command",
	}))
	if err == nil {
		t.Fatal("RunAction error = nil, want invalid argument")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", got, connect.CodeInvalidArgument)
	}
}

type adapterResultChecker struct {
	result internalcaps.CheckResult
	calls  int
}

func (c *adapterResultChecker) Check(context.Context) (internalcaps.Status, string) {
	result := c.CheckResult(context.Background())
	return result.Status, result.Message
}

func (c *adapterResultChecker) CheckResult(context.Context) internalcaps.CheckResult {
	c.calls++
	return c.result
}

type adapterCommandRunner struct {
	calls int
}

func (r *adapterCommandRunner) Run(context.Context, string, ...string) (internalcaps.CommandResult, error) {
	r.calls++
	return internalcaps.CommandResult{
		Stdout:   []byte(`{"success":true,"verdict":"healthy","exit_code":0}`),
		ExitCode: 0,
	}, nil
}

func TestAdapterRunActionInvalidatesCacheAndReturnsFreshSnapshot(t *testing.T) {
	checker := &adapterResultChecker{result: internalcaps.CheckResult{
		Status:     internalcaps.StatusUnavailable,
		Message:    "scenario is installed but stopped",
		ReasonCode: "scenario_stopped",
		ActionKind: internalcaps.ActionKindScenarioStart,
	}}
	registry := internalcaps.NewRegistry(
		internalcaps.Known,
		map[string]internalcaps.Checker{"audio-tools": checker},
		time.Hour,
	)

	stale := registry.Resolve(context.Background())
	if len(stale) == 0 {
		t.Fatal("expected capability snapshot")
	}
	checker.result = internalcaps.CheckResult{
		Status:  internalcaps.StatusAvailable,
		Message: "scenario is healthy",
	}

	adapter := &Adapter{
		Registry:     registry,
		ActionRunner: &adapterCommandRunner{},
		CLIPath:      "vrooli",
	}
	result, err := adapter.RunAction(context.Background(), ActionRequest{
		CapabilityID: "audio-tools",
		ActionKind:   string(internalcaps.ActionKindScenarioStart),
	})
	if err != nil {
		t.Fatalf("RunAction returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %+v, want success", result)
	}
	for _, cap := range result.Snapshot.Capabilities {
		if cap.ID == "audio-tools" {
			if cap.Status != string(internalcaps.StatusAvailable) || cap.Message != "scenario is healthy" {
				t.Fatalf("audio-tools snapshot = %+v, want fresh available status", cap)
			}
			return
		}
	}
	t.Fatal("audio-tools missing from refreshed snapshot")
}

func TestAdapterRejectsLifecycleActionForHealthyCapability(t *testing.T) {
	checker := &adapterResultChecker{result: internalcaps.CheckResult{
		Status:     internalcaps.StatusAvailable,
		Message:    "scenario is healthy",
		ReasonCode: "scenario_healthy",
		ActionKind: internalcaps.ActionKindScenarioStart,
	}}
	registry := internalcaps.NewRegistry(
		internalcaps.Known,
		map[string]internalcaps.Checker{"audio-tools": checker},
		0,
	)
	runner := &adapterCommandRunner{}
	adapter := &Adapter{Registry: registry, ActionRunner: runner, CLIPath: "vrooli"}
	_, err := adapter.RunAction(context.Background(), ActionRequest{
		CapabilityID: "audio-tools",
		ActionKind:   string(internalcaps.ActionKindScenarioStart),
	})
	if err == nil || runner.calls != 0 {
		t.Fatalf("RunAction error=%v calls=%d, want rejection without lifecycle invocation", err, runner.calls)
	}
}

func TestAdapterProjectsBackendOptionsAndSupportsDescribeAndLiveness(t *testing.T) {
	registry := internalcaps.NewRegistry(
		[]internalcaps.Def{{ID: "audio", Name: "Audio", Description: "Audio capability", DependencyKind: internalcaps.DependencyScenario, DependencySlug: "audio"}},
		map[string]internalcaps.Checker{"audio": &adapterResultChecker{result: internalcaps.CheckResult{
			Status:  internalcaps.StatusAvailable,
			Message: "ready",
		}}},
		0,
	)
	backends := backend.New()
	backends.Register(backend.Descriptor{
		ID:              backend.Standard,
		DisplayName:     "Standard",
		Description:     "local PTY",
		SurvivesRestart: false,
		Available:       true,
		Reason:          "",
	}, nil)
	adapter := &Adapter{
		Registry:        registry,
		BackendRegistry: backends,
		DefaultBackend:  func() string { return string(backend.Standard) },
		Logger:          log.New(io.Discard, "", 0),
	}

	described, err := adapter.Describe(context.Background())
	if err != nil || len(described) == 0 {
		t.Fatalf("Describe = %d bytes, %v", len(described), err)
	}
	resolved := adapter.Resolve(context.Background())
	if resolved.DefaultBackend != string(backend.Standard) || len(resolved.BackendOptions) != 1 {
		t.Fatalf("Resolve = %+v", resolved)
	}
	live := adapter.Liveness(context.Background())
	if len(live.Capabilities) != 1 || live.Timestamp == "" {
		t.Fatalf("Liveness = %+v", live)
	}

	if _, err := (&Adapter{}).Describe(context.Background()); err == nil {
		t.Fatal("nil adapter registry should fail Describe")
	}
}

func TestModuleMountsDescribeRouteAndReportsUnavailableServices(t *testing.T) {
	checkers := make(map[string]internalcaps.Checker, len(internalcaps.Known))
	for _, def := range internalcaps.Known {
		checkers[def.ID] = &internalcaps.StaticChecker{Available: func() (bool, string) { return true, "available" }}
	}
	adapter := &Adapter{Registry: internalcaps.NewRegistry(internalcaps.Known, checkers, 0)}
	router := mux.NewRouter()
	Module(adapter, nil).Mount(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.Len() == 0 {
		t.Fatalf("describe response = %d, %d bytes", resp.Code, resp.Body.Len())
	}

	missing := mux.NewRouter()
	Module(&fakeService{}, nil).Mount(missing)
	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil)
	missingResp := httptest.NewRecorder()
	missing.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing describer status = %d", missingResp.Code)
	}
}

func TestIntegrationSurfaceProjectsTheRealKnownCatalogue(t *testing.T) {
	checkers := make(map[string]internalcaps.Checker, len(internalcaps.Known))
	for _, def := range internalcaps.Known {
		checkers[def.ID] = &internalcaps.StaticChecker{
			Available: func() (bool, string) { return true, "available" },
		}
	}
	registry := internalcaps.NewRegistry(internalcaps.Known, checkers, 0)
	adapter := &Adapter{Registry: registry}

	snapshot := adapter.Resolve(context.Background())
	got := make(map[string]bool, len(snapshot.Capabilities))
	for _, capability := range snapshot.Capabilities {
		got[capability.ID] = true
	}
	for _, id := range []string{"audio-tools", "vrooli-bridge", "ollama", "openrouter"} {
		if !got[id] {
			t.Fatalf("integration surface omitted %q; got %v", id, got)
		}
	}
	if len(snapshot.Capabilities) != len(internalcaps.Known) {
		t.Fatalf("integration surface projected %d capabilities, want %d", len(snapshot.Capabilities), len(internalcaps.Known))
	}
}

type updateActionService struct {
	*fakeService
	guardAdmits bool
	guardAction string
	denyAction  string
	calls       int
	err         error
}

func (s *updateActionService) GuardAgentUpdateAction(w http.ResponseWriter, _ *http.Request, action string) bool {
	s.guardAction = action
	if !s.guardAdmits || action == s.denyAction {
		http.Error(w, "operator only", http.StatusForbidden)
	}
	return s.guardAdmits && action != s.denyAction
}

func (s *updateActionService) AgentUpdate(_ context.Context, target, agent string) (AgentUpdateResult, error) {
	return AgentUpdateResult{TargetID: target, AgentID: agent, State: "update_ready", PrimaryAction: "update"}, nil
}

func (s *updateActionService) RunAgentUpdateAction(_ context.Context, target, agent, action, _ string) (AgentUpdateActionResult, error) {
	s.calls++
	if s.err != nil {
		return AgentUpdateActionResult{}, s.err
	}
	return AgentUpdateActionResult{Success: true, Message: action, OperationID: "op-1", Update: AgentUpdateResult{
		TargetID: target, AgentID: agent, State: "up_to_date", OperatorCommand: "cmd", Reason: "why", InstallKind: "managed",
		RollbackVersion: "1.0.0", NewerThanReviewed: true, Checks: []AgentUpdateCheck{{ID: "ownership", Passed: true, Message: "yours"}},
	}}, nil
}

func actionClientRequest(t *testing.T, service Service) *http.Response {
	return actionClientRequestBody(t, service, `{"targetId":"local","agentId":"codex","action":"update"}`)
}

func actionClientRequestBody(t *testing.T, service Service, body string) *http.Response {
	t.Helper()
	router := mux.NewRouter()
	Module(service, log.New(io.Discard, "", 0)).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	response, err := http.Post(server.URL+capabilitiesconnect.CapabilitiesServiceRunAgentUpdateActionProcedure, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func TestRunAgentUpdateActionIsGuardedBeforeItRuns(t *testing.T) {
	refused := &updateActionService{fakeService: &fakeService{}}
	if response := actionClientRequest(t, refused); response.StatusCode != http.StatusForbidden || refused.calls != 0 {
		t.Fatalf("status %d, calls %d", response.StatusCode, refused.calls)
	}
	if response := actionClientRequest(t, &fakeService{}); response.StatusCode != http.StatusForbidden {
		t.Fatalf("a service with no guard must refuse: %d", response.StatusCode)
	}
	admitted := &updateActionService{fakeService: &fakeService{}, guardAdmits: true}
	response := actionClientRequest(t, admitted)
	if response.StatusCode != http.StatusOK || admitted.calls != 1 {
		t.Fatalf("status %d, calls %d", response.StatusCode, admitted.calls)
	}
	if admitted.guardAction != "update" {
		t.Fatalf("guard saw action %q, want update", admitted.guardAction)
	}
}

func TestRunAgentUpdateActionGuardReceivesActionBeforeAdmission(t *testing.T) {
	service := &updateActionService{fakeService: &fakeService{}, guardAdmits: true, denyAction: "update"}
	response := actionClientRequestBody(t, service, `{"targetId":"local","agentId":"codex","action":"refresh_catalog"}`)
	if response.StatusCode != http.StatusOK || service.calls != 1 || service.guardAction != "refresh_catalog" {
		t.Fatalf("refresh status=%d calls=%d guardAction=%q", response.StatusCode, service.calls, service.guardAction)
	}
	response = actionClientRequest(t, service)
	if response.StatusCode != http.StatusForbidden || service.calls != 1 || service.guardAction != "update" {
		t.Fatalf("update status=%d calls=%d guardAction=%q", response.StatusCode, service.calls, service.guardAction)
	}
}

func TestAdapterRefusesActionsWithoutAConfiguredGuard(t *testing.T) {
	recorder := httptest.NewRecorder()
	if (&Adapter{}).GuardAgentUpdateAction(recorder, httptest.NewRequest(http.MethodPost, "/", nil), "update") || recorder.Code != http.StatusForbidden {
		t.Fatalf("an adapter with no guard admitted: %d", recorder.Code)
	}
}

func TestAgentReadsAreNotGuarded(t *testing.T) {
	refused := &updateActionService{fakeService: &fakeService{}}
	router := mux.NewRouter()
	Module(refused, log.New(io.Discard, "", 0)).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	response, err := http.Post(server.URL+capabilitiesconnect.CapabilitiesServiceAgentUpdateProcedure, "application/json", strings.NewReader(`{"targetId":"local","agentId":"codex"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestRunAgentUpdateActionProjectsEveryField(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &updateActionService{fakeService: &fakeService{}}})
	response, err := h.RunAgentUpdateAction(context.Background(), connect.NewRequest(&capabilitiesv1.RunAgentUpdateActionRequest{TargetId: "local", AgentId: "codex", Action: "update"}))
	if err != nil {
		t.Fatal(err)
	}
	update := response.Msg.GetUpdate()
	if !response.Msg.Success || response.Msg.Message != "update" || response.Msg.OperationId != "op-1" || update.State != "up_to_date" || update.OperatorCommand != "cmd" || update.Reason != "why" || update.InstallKind != "managed" || update.RollbackVersion != "1.0.0" || !update.NewerThanReviewed || len(update.Checks) != 1 || !update.Checks[0].Passed {
		t.Fatalf("response = %+v", response.Msg)
	}
}

func TestRunAgentUpdateActionErrorCodes(t *testing.T) {
	cases := map[string]struct {
		err  error
		want connect.Code
	}{
		"refused":  {fmt.Errorf("%w: operator only", ErrPermissionDenied), connect.CodePermissionDenied},
		"busy":     {fmt.Errorf("%w: running", ErrFailedPrecondition), connect.CodeFailedPrecondition},
		"gone":     {fmt.Errorf("%w: gone", ErrTargetNotFound), connect.CodeNotFound},
		"deadline": {context.DeadlineExceeded, connect.CodeDeadlineExceeded},
		"remote":   {connect.NewError(connect.CodePermissionDenied, errors.New("node write scope")), connect.CodePermissionDenied},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := NewConnectHandler(Deps{Service: &updateActionService{fakeService: &fakeService{}, err: tc.err}})
			_, err := h.RunAgentUpdateAction(context.Background(), connect.NewRequest(&capabilitiesv1.RunAgentUpdateActionRequest{}))
			if connect.CodeOf(err) != tc.want {
				t.Fatalf("code = %v, want %v", connect.CodeOf(err), tc.want)
			}
		})
	}
	h := NewConnectHandler(Deps{Service: &fakeService{}})
	if _, err := h.RunAgentUpdateAction(context.Background(), connect.NewRequest(&capabilitiesv1.RunAgentUpdateActionRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v", connect.CodeOf(err))
	}
}

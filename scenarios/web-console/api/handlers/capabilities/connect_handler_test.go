package capabilities

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	internalcaps "web-console/internal/capabilities"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
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

package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// stubOperatingModeHandler is a test double for the generated
// OperatingModeService Connect handler. It embeds the Unimplemented base so a
// test only wires the RPCs it exercises; each hook receives the decoded request
// (so tests can assert the CLI built the right proto) and returns the projection
// to serialize back over the wire.
type stubOperatingModeHandler struct {
	apiconnect.UnimplementedOperatingModeServiceHandler
	catalog       func(*apipb.OperatingModeCatalogRequest) (*apipb.OperatingModeCatalogResponse, error)
	getMode       func(*apipb.OperatingModeGetRequest) (*apipb.OperatingModeDetailResponse, error)
	getWorkspace  func(*apipb.OperatingModeWorkspaceRequest) (*apipb.OperatingModeWorkspace, error)
	switchMode    func(*apipb.OperatingModeSwitchRequest) (*apipb.OperatingModeSwitchResult, error)
	startPhase    func(*apipb.OperatingModeStartPhaseRequest) (*apipb.OperatingModeRoundEnvelope, error)
	refreshRound  func(*apipb.OperatingModeRoundActionRequest) (*apipb.OperatingModeRoundEnvelope, error)
	completeItems func(*apipb.OperatingModeCompleteItemsRequest) (*apipb.OperatingModeBacklogSyncResult, error)
	applyBacklog  func(*apipb.OperatingModeApplyBacklogSyncRequest) (*apipb.OperatingModeBacklogSyncResult, error)
}

func (s *stubOperatingModeHandler) Catalog(_ context.Context, req *connect.Request[apipb.OperatingModeCatalogRequest]) (*connect.Response[apipb.OperatingModeCatalogResponse], error) {
	msg, err := s.catalog(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) GetMode(_ context.Context, req *connect.Request[apipb.OperatingModeGetRequest]) (*connect.Response[apipb.OperatingModeDetailResponse], error) {
	msg, err := s.getMode(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) GetWorkspace(_ context.Context, req *connect.Request[apipb.OperatingModeWorkspaceRequest]) (*connect.Response[apipb.OperatingModeWorkspace], error) {
	msg, err := s.getWorkspace(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) SwitchMode(_ context.Context, req *connect.Request[apipb.OperatingModeSwitchRequest]) (*connect.Response[apipb.OperatingModeSwitchResult], error) {
	msg, err := s.switchMode(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) StartPhase(_ context.Context, req *connect.Request[apipb.OperatingModeStartPhaseRequest]) (*connect.Response[apipb.OperatingModeRoundEnvelope], error) {
	msg, err := s.startPhase(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) RefreshRound(_ context.Context, req *connect.Request[apipb.OperatingModeRoundActionRequest]) (*connect.Response[apipb.OperatingModeRoundEnvelope], error) {
	msg, err := s.refreshRound(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) CompleteItems(_ context.Context, req *connect.Request[apipb.OperatingModeCompleteItemsRequest]) (*connect.Response[apipb.OperatingModeBacklogSyncResult], error) {
	msg, err := s.completeItems(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubOperatingModeHandler) ApplyBacklogSync(_ context.Context, req *connect.Request[apipb.OperatingModeApplyBacklogSyncRequest]) (*connect.Response[apipb.OperatingModeBacklogSyncResult], error) {
	msg, err := s.applyBacklog(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

// newOperatingModeTestApp mounts the real generated OperatingModeService Connect
// handler over the provided stub on a test server and returns an App whose
// generated client talks to it — proving the full transport round-trip.
func newOperatingModeTestApp(t *testing.T, stub apiconnect.OperatingModeServiceHandler) *App {
	t.Helper()
	path, handler := apiconnect.NewOperatingModeServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	app, _ := newFeedbackTestApp(t, mux)
	return app
}

func TestCmdInitiativesModeList_ReadsCatalog(t *testing.T) {
	var called bool
	stub := &stubOperatingModeHandler{
		catalog: func(*apipb.OperatingModeCatalogRequest) (*apipb.OperatingModeCatalogResponse, error) {
			called = true
			return &apipb.OperatingModeCatalogResponse{
				Modes: []*apipb.OperatingModeCatalogEntry{{
					Mode:        "item-level",
					Label:       "Item Level",
					ScopeKind:   "backlog_item",
					RunStrategy: "existing_item_flow",
					Default:     true,
					Switchable:  true,
				}},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeList([]string{}); err != nil {
		t.Fatalf("cmdInitiativesModeList returned error: %v", err)
	}
	if !called {
		t.Error("Catalog was not called")
	}
}

func TestCmdInitiativesModeWorkspace_ReadsWorkspace(t *testing.T) {
	var gotName string
	stub := &stubOperatingModeHandler{
		getWorkspace: func(req *apipb.OperatingModeWorkspaceRequest) (*apipb.OperatingModeWorkspace, error) {
			gotName = req.GetInitiativeName()
			return &apipb.OperatingModeWorkspace{
				InitiativeName: "init",
				Mode:           "holistic-loop",
				Definition: &apipb.OperatingModeWorkspaceMode{
					Label:       "Holistic Loop",
					ScopeKind:   "initiative",
					RunStrategy: "operator_gated_loop",
					Phases: []*apipb.OperatingModeWorkspacePhase{
						{Phase: "investigate", ProfileKey: "swarm-manager/deep-work"},
					},
				},
				Artifacts: []*apipb.OperatingModeArtifactSnapshot{
					{Path: "modes/holistic-loop/findings.md", Required: true},
				},
				Rounds: []*apipb.OperatingModeRoundEnvelope{
					{Round: 1, Mode: "holistic-loop", Phase: "investigate", Status: "completed", RunId: "run-1", AgentProfileKey: "swarm-manager/deep-work"},
				},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeWorkspace([]string{"--name", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotName != "init" {
		t.Errorf("initiative name: %s", gotName)
	}
}

func TestCmdInitiativesModeSwitch_PostsCancellationConfirmation(t *testing.T) {
	var got *apipb.OperatingModeSwitchRequest
	stub := &stubOperatingModeHandler{
		switchMode: func(req *apipb.OperatingModeSwitchRequest) (*apipb.OperatingModeSwitchResult, error) {
			got = req
			return &apipb.OperatingModeSwitchResult{
				InitiativeName:         "init",
				FromMode:               "item-level",
				ToMode:                 "holistic-loop",
				CanceledItemExecutions: []*apipb.OperatingModeActiveItemExecution{{ItemRef: "execute/a"}},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeSwitch([]string{"--name", "init", "--mode", "holistic-loop", "--cancel-active-item-executions"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got.GetInitiativeName() != "init" {
		t.Errorf("initiative name: %s", got.GetInitiativeName())
	}
	if got.GetMode() != "holistic-loop" {
		t.Errorf("mode: %s", got.GetMode())
	}
	if !got.GetCancelActiveItemExecutions() {
		t.Errorf("cancel flag: %v", got.GetCancelActiveItemExecutions())
	}
}

// TestCmdInitiativesModeSwitch_SurfacesActiveExecutionsConflict proves the CLI
// decodes the structured OperatingModeActiveItemExecutionsConflict detail off a
// FailedPrecondition Connect error and reports the affected executions.
func TestCmdInitiativesModeSwitch_SurfacesActiveExecutionsConflict(t *testing.T) {
	stub := &stubOperatingModeHandler{
		switchMode: func(*apipb.OperatingModeSwitchRequest) (*apipb.OperatingModeSwitchResult, error) {
			cerr := connect.NewError(connect.CodeFailedPrecondition, errors.New("active item executions must be canceled"))
			detail, err := connect.NewErrorDetail(&apipb.OperatingModeActiveItemExecutionsConflict{
				InitiativeName: "init",
				FromMode:       "holistic-loop",
				ToMode:         "item-level",
				Executions: []*apipb.OperatingModeActiveItemExecution{
					{ItemRef: "execute/a", RunId: "run-9", Status: "running"},
				},
			})
			if err != nil {
				t.Fatalf("build detail: %v", err)
			}
			cerr.AddDetail(detail)
			return nil, cerr
		},
	}
	app := newOperatingModeTestApp(t, stub)
	err := app.cmdInitiativesModeSwitch([]string{"--name", "init", "--mode", "item-level"})
	if err == nil {
		t.Fatal("expected a conflict error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "execute/a") {
		t.Errorf("error should list the blocking execution ref, got: %s", msg)
	}
	if !strings.Contains(msg, "--cancel-active-item-executions") {
		t.Errorf("error should hint at the cancel flag, got: %s", msg)
	}
}

func TestCmdInitiativesModeStart_PostsPhaseStart(t *testing.T) {
	var got *apipb.OperatingModeStartPhaseRequest
	stub := &stubOperatingModeHandler{
		startPhase: func(req *apipb.OperatingModeStartPhaseRequest) (*apipb.OperatingModeRoundEnvelope, error) {
			got = req
			return &apipb.OperatingModeRoundEnvelope{
				Round: 2, Mode: "holistic-loop", Phase: "execute", Status: "agent_running",
				RunId: "run-2", AgentProfileKey: "swarm-manager/deep-work",
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeStart([]string{"--name", "init", "--phase", "execute", "--note", "go"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got.GetInitiativeName() != "init" || got.GetPhase() != "execute" {
		t.Errorf("start selectors: name=%s phase=%s", got.GetInitiativeName(), got.GetPhase())
	}
	if got.GetNote() != "go" {
		t.Errorf("note: %s", got.GetNote())
	}
}

func TestCmdInitiativesModeRefresh_PostsRoundRefresh(t *testing.T) {
	var got *apipb.OperatingModeRoundActionRequest
	stub := &stubOperatingModeHandler{
		refreshRound: func(req *apipb.OperatingModeRoundActionRequest) (*apipb.OperatingModeRoundEnvelope, error) {
			got = req
			return &apipb.OperatingModeRoundEnvelope{
				Round: 2, Mode: "phased-plan-drain", Phase: "execute_next", Status: "completed",
				AgentProfileKey: "swarm-manager/deep-work",
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeRefresh([]string{"--name", "init", "--mode", "phased-plan-drain", "--round", "2"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got.GetInitiativeName() != "init" || got.GetMode() != "phased-plan-drain" || got.GetRound() != 2 {
		t.Errorf("refresh selectors: %+v", got)
	}
}

func TestCmdInitiativesModeCompleteItems_PostsRunValidatedRefs(t *testing.T) {
	var got *apipb.OperatingModeCompleteItemsRequest
	stub := &stubOperatingModeHandler{
		completeItems: func(req *apipb.OperatingModeCompleteItemsRequest) (*apipb.OperatingModeBacklogSyncResult, error) {
			got = req
			return &apipb.OperatingModeBacklogSyncResult{
				InitiativeName: "init", Mode: "holistic-loop", Phase: "execute", Round: 3, RunId: "run-3",
				CompletedItems: []*apipb.OperatingModeBacklogCompletionResult{
					{ItemRef: "execute/a", FromStatus: "ready", ToStatus: "completed"},
				},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeComplete([]string{"--name", "init", "--mode", "holistic-loop", "--round", "3", "--run-id", "run-3", "--items", "execute/a"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got.GetRunId() != "run-3" {
		t.Errorf("run id: %s", got.GetRunId())
	}
	if refs := got.GetItemRefs(); len(refs) != 1 || refs[0] != "execute/a" {
		t.Errorf("item refs: %#v", refs)
	}
}

func TestCmdInitiativesModeApplyBacklogSync_PostsSelectedMutationIDs(t *testing.T) {
	var got *apipb.OperatingModeApplyBacklogSyncRequest
	stub := &stubOperatingModeHandler{
		applyBacklog: func(req *apipb.OperatingModeApplyBacklogSyncRequest) (*apipb.OperatingModeBacklogSyncResult, error) {
			got = req
			return &apipb.OperatingModeBacklogSyncResult{
				InitiativeName: "init", Mode: "phased-plan-drain", Phase: "classify_progress", Round: 4, RunId: "run-4",
				ProposalResult: &apipb.OperatingModeProposalApplyResult{
					Applied: 1, Failed: 0, Skipped: 1, Created: 1,
					Outcomes: []*apipb.OperatingModeProposalOutcome{
						{MutationId: "m1", Op: "add_item", Target: "fix/follow-up", Applied: true},
					},
				},
			}, nil
		},
	}
	app := newOperatingModeTestApp(t, stub)
	if err := app.cmdInitiativesModeApplyBacklogSync([]string{"--name", "init", "--mode", "phased-plan-drain", "--round", "4", "--run-id", "run-4", "--mutations", "m1,m3"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got.GetRunId() != "run-4" {
		t.Errorf("run id: %s", got.GetRunId())
	}
	if ids := got.GetAcceptedMutationIds(); len(ids) != 2 || ids[0] != "m1" || ids[1] != "m3" {
		t.Errorf("mutation ids: %#v", ids)
	}
}

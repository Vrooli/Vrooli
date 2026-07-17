package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	clitest "swarm-manager/cli/internal/testutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// stubAgentOperationsHandler is a test double for the generated
// AgentOperationsService Connect handler. It embeds the Unimplemented base so a
// test only wires the RPCs it exercises; each hook receives the decoded request
// (so tests can assert the CLI built the right proto) and returns the projection
// to serialize back over the wire.
type stubAgentOperationsHandler struct {
	apiconnect.UnimplementedAgentOperationsServiceHandler
	listCatalog     func(*apipb.AgentOpsListOperationCatalogRequest) (*apipb.AgentOpsListOperationCatalogResponse, error)
	compatibleModes func(*apipb.AgentOpsListCompatibleModesRequest) (*apipb.AgentOpsListCompatibleModesResponse, error)
	resolvedBinds   func(*apipb.AgentOpsGetResolvedBindingsRequest) (*apipb.AgentOpsGetResolvedBindingsResponse, error)
	listOverrides   func(*apipb.AgentOpsListBindingOverridesRequest) (*apipb.AgentOpsListBindingOverridesResponse, error)
	putOverride     func(*apipb.AgentOpsPutBindingOverrideRequest) (*apipb.AgentOpsPutBindingOverrideResponse, error)
	deleteOverride  func(*apipb.AgentOpsDeleteBindingOverrideRequest) (*apipb.AgentOpsDeleteBindingOverrideResponse, error)
	projection      func(*apipb.AgentOpsGetWorkflowProjectionRequest) (*apipb.AgentOpsGetWorkflowProjectionResponse, error)
	history         func(*apipb.AgentOpsListExecutionHistoryRequest) (*apipb.AgentOpsListExecutionHistoryResponse, error)
	migration       func(*apipb.AgentOpsGetMigrationStatusRequest) (*apipb.AgentOpsGetMigrationStatusResponse, error)
	reconcile       func(*apipb.AgentOpsRunReconciliationRequest) (*apipb.AgentOpsRunReconciliationResponse, error)
}

func (s *stubAgentOperationsHandler) ListOperationCatalog(_ context.Context, req *connect.Request[apipb.AgentOpsListOperationCatalogRequest]) (*connect.Response[apipb.AgentOpsListOperationCatalogResponse], error) {
	msg, err := s.listCatalog(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) ListCompatibleModes(_ context.Context, req *connect.Request[apipb.AgentOpsListCompatibleModesRequest]) (*connect.Response[apipb.AgentOpsListCompatibleModesResponse], error) {
	msg, err := s.compatibleModes(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) GetResolvedBindings(_ context.Context, req *connect.Request[apipb.AgentOpsGetResolvedBindingsRequest]) (*connect.Response[apipb.AgentOpsGetResolvedBindingsResponse], error) {
	msg, err := s.resolvedBinds(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) ListBindingOverrides(_ context.Context, req *connect.Request[apipb.AgentOpsListBindingOverridesRequest]) (*connect.Response[apipb.AgentOpsListBindingOverridesResponse], error) {
	msg, err := s.listOverrides(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) PutBindingOverride(_ context.Context, req *connect.Request[apipb.AgentOpsPutBindingOverrideRequest]) (*connect.Response[apipb.AgentOpsPutBindingOverrideResponse], error) {
	msg, err := s.putOverride(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) DeleteBindingOverride(_ context.Context, req *connect.Request[apipb.AgentOpsDeleteBindingOverrideRequest]) (*connect.Response[apipb.AgentOpsDeleteBindingOverrideResponse], error) {
	msg, err := s.deleteOverride(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) GetWorkflowProjection(_ context.Context, req *connect.Request[apipb.AgentOpsGetWorkflowProjectionRequest]) (*connect.Response[apipb.AgentOpsGetWorkflowProjectionResponse], error) {
	msg, err := s.projection(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) ListExecutionHistory(_ context.Context, req *connect.Request[apipb.AgentOpsListExecutionHistoryRequest]) (*connect.Response[apipb.AgentOpsListExecutionHistoryResponse], error) {
	msg, err := s.history(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) GetMigrationStatus(_ context.Context, req *connect.Request[apipb.AgentOpsGetMigrationStatusRequest]) (*connect.Response[apipb.AgentOpsGetMigrationStatusResponse], error) {
	msg, err := s.migration(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

func (s *stubAgentOperationsHandler) RunReconciliation(_ context.Context, req *connect.Request[apipb.AgentOpsRunReconciliationRequest]) (*connect.Response[apipb.AgentOpsRunReconciliationResponse], error) {
	msg, err := s.reconcile(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

// newAgentOpsTestApp mounts the real generated AgentOperationsService Connect
// handler over the provided stub on a test server and returns an App whose
// generated client talks to it — proving the full transport round-trip.
func newAgentOpsTestApp(t *testing.T, stub apiconnect.AgentOperationsServiceHandler) *App {
	t.Helper()
	path, handler := apiconnect.NewAgentOperationsServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	clitest.NewAPIServer(t, mux)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

func TestCmdAgentOpsCatalog_ListsContracts(t *testing.T) {
	stub := &stubAgentOperationsHandler{
		listCatalog: func(*apipb.AgentOpsListOperationCatalogRequest) (*apipb.AgentOpsListOperationCatalogResponse, error) {
			return &apipb.AgentOpsListOperationCatalogResponse{Entries: []*apipb.AgentOpsCatalogEntry{{
				Contract: &domainpb.AgentOpsOperationContract{Id: "review-round", Version: "1.0.0", Summary: "Run one review round."},
				Revision: "sha256:abc",
				CompatibleTargets: []domainpb.OperatingModeTargetKind{
					domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_INITIATIVE,
					domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_SCENARIO,
				},
			}}}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error { return app.cmdAgentOpsCatalog(nil) })
	if !containsAll(out, "review-round@1.0.0", "initiative, scenario", "sha256:abc") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdAgentOpsCompatibleModes_ScenarioSelectorRoundTrips(t *testing.T) {
	var got *apipb.AgentOpsTargetSelector
	stub := &stubAgentOperationsHandler{
		compatibleModes: func(req *apipb.AgentOpsListCompatibleModesRequest) (*apipb.AgentOpsListCompatibleModesResponse, error) {
			got = req.GetTarget()
			return &apipb.AgentOpsListCompatibleModesResponse{Modes: []*apipb.AgentOpsCompatibleMode{{
				Mode: "scenario-spec-sync", ModeRevision: "sha256:m", ModeDigest: "sha256:m",
				TargetKind: domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_SCENARIO,
				Verdicts:   []*apipb.AgentOpsModeOperationVerdict{{Operation: "spec-sync", OperationVersion: "1.0.0", Compatible: true}},
			}}}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsCompatibleModes([]string{"--target-kind", "scenario", "--target", "demo-scenario"})
	})
	if got.GetKind() != domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_SCENARIO || got.GetId() != "demo-scenario" {
		t.Fatalf("scenario selector must round-trip over the wire: %+v", got)
	}
	if !containsAll(out, "scenario-spec-sync", "(target: scenario)", "spec-sync@1.0.0") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdAgentOpsBindings_RendersWinnerAndContributions(t *testing.T) {
	stub := &stubAgentOperationsHandler{
		resolvedBinds: func(req *apipb.AgentOpsGetResolvedBindingsRequest) (*apipb.AgentOpsGetResolvedBindingsResponse, error) {
			return &apipb.AgentOpsGetResolvedBindingsResponse{Operations: []*apipb.AgentOpsResolvedOperationBinding{
				{
					Operation: "review-round", OperationVersion: "1.0.0", Resolved: true,
					Binding: &domainpb.AgentOpsOperationBinding{
						Operation: "review-round", Mode: "holistic-loop", ModeRevision: "sha256:r",
						Layer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE,
					},
					PolicyId: "initiative-default", PolicyRevision: "sha256:p",
					Contributions: []*apipb.AgentOpsBindingContribution{
						{Binding: &domainpb.AgentOpsOperationBinding{Mode: "synthetic-loop", ModeRevision: "sha256:d", Layer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_SYSTEM_DEFAULT}},
						{Binding: &domainpb.AgentOpsOperationBinding{Mode: "holistic-loop", ModeRevision: "sha256:r", Layer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE}, Winning: true},
					},
				},
				{Operation: "spec-sync", OperationVersion: "1.0.0", Error: "no-binding", ErrorMessage: "no binding in scope"},
			}}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsBindings([]string{"--target-kind", "initiative", "--target", "ship-x", "--verbose"})
	})
	if !containsAll(out,
		"review-round@1.0.0 -> holistic-loop@sha256:r [initiative-override]",
		"policy : initiative-default (sha256:p)",
		"* [initiative-override] holistic-loop@sha256:r",
		"[system-default] synthetic-loop@sha256:d",
		"spec-sync@1.0.0 -> UNRESOLVED (no-binding: no binding in scope)") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdAgentOpsOverrides_ListSetClear(t *testing.T) {
	var putReq *apipb.AgentOpsPutBindingOverrideRequest
	var delReq *apipb.AgentOpsDeleteBindingOverrideRequest
	found := true
	stub := &stubAgentOperationsHandler{
		listOverrides: func(req *apipb.AgentOpsListBindingOverridesRequest) (*apipb.AgentOpsListBindingOverridesResponse, error) {
			return &apipb.AgentOpsListBindingOverridesResponse{Overrides: []*apipb.AgentOpsBindingOverrideDocument{{
				Binding: &domainpb.AgentOpsOperationBinding{
					Operation: "review-round", Mode: "holistic-loop", ModeRevision: "sha256:r",
					Layer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE,
				},
				File: "review-round.json", Revision: "sha256:f", UpdatedAt: "2026-07-14T10:00:00Z",
			}}}, nil
		},
		putOverride: func(req *apipb.AgentOpsPutBindingOverrideRequest) (*apipb.AgentOpsPutBindingOverrideResponse, error) {
			putReq = req
			return &apipb.AgentOpsPutBindingOverrideResponse{
				Stored: &domainpb.AgentOpsOperationBinding{
					Operation: req.GetOperation(), Mode: req.GetMode(), ModeRevision: req.GetModeRevision(),
					Layer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE,
				},
				File: "review-round.json", Revision: "sha256:new",
			}, nil
		},
		deleteOverride: func(req *apipb.AgentOpsDeleteBindingOverrideRequest) (*apipb.AgentOpsDeleteBindingOverrideResponse, error) {
			delReq = req
			return &apipb.AgentOpsDeleteBindingOverrideResponse{Found: found}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsOverrides([]string{"list", "--owner-kind", "initiative", "--owner", "ship-x"})
	})
	if !containsAll(out, "review-round@* -> holistic-loop@sha256:r", "review-round.json") {
		t.Fatalf("unexpected list output: %s", out)
	}

	out = clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsOverrides([]string{
			"set", "--owner-kind", "initiative", "--owner", "ship-x",
			"--operation", "review-round", "--mode", "holistic-loop", "--mode-revision", "sha256:r",
		})
	})
	if putReq.GetOwner().GetId() != "ship-x" || putReq.GetOperation() != "review-round" ||
		putReq.GetMode() != "holistic-loop" || putReq.GetModeRevision() != "sha256:r" {
		t.Fatalf("put request mis-built: %+v", putReq)
	}
	if !containsAll(out, "Binding override stored", "initiative-override", "snapshot-at-invoke") {
		t.Fatalf("unexpected set output: %s", out)
	}

	out = clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsOverrides([]string{"clear", "--owner-kind", "initiative", "--owner", "ship-x", "--operation", "review-round"})
	})
	if delReq.GetOwner().GetId() != "ship-x" || delReq.GetOperation() != "review-round" {
		t.Fatalf("delete request mis-built: %+v", delReq)
	}
	if !strings.Contains(out, "Override for review-round cleared.") {
		t.Fatalf("unexpected clear output: %s", out)
	}

	// Deleting an absent override reports honestly rather than pretending success.
	found = false
	out = clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsOverrides([]string{"clear", "--owner-kind", "initiative", "--owner", "ship-x", "--operation", "review-round"})
	})
	if !strings.Contains(out, "No override for review-round was stored") {
		t.Fatalf("unexpected clear-absent output: %s", out)
	}

	if err := app.cmdAgentOpsOverrides([]string{"promote"}); err == nil || !strings.Contains(err.Error(), "unknown overrides action") {
		t.Fatalf("unknown action must be rejected, got %v", err)
	}
	if err := app.cmdAgentOpsOverrides([]string{"set", "--owner-kind", "initiative", "--owner", "ship-x", "--operation", "op"}); err == nil {
		t.Fatal("set without --mode/--mode-revision must fail")
	}
}

func TestCmdAgentOpsOverridesMutationsHonorGlobalDryRun(t *testing.T) {
	stub := &stubAgentOperationsHandler{
		putOverride: func(*apipb.AgentOpsPutBindingOverrideRequest) (*apipb.AgentOpsPutBindingOverrideResponse, error) {
			t.Fatal("dry-run must not reach the server")
			return nil, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	app.globalDry = true
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsOverrides([]string{
			"set", "--owner-kind", "initiative", "--owner", "ship-x",
			"--operation", "review-round", "--mode", "m", "--mode-revision", "sha256:r",
		})
	})
	if !strings.Contains(out, "[dry-run]") {
		t.Fatalf("unexpected dry-run output: %s", out)
	}
}

func TestCmdAgentOpsWorkflow_RendersProjection(t *testing.T) {
	stub := &stubAgentOperationsHandler{
		projection: func(req *apipb.AgentOpsGetWorkflowProjectionRequest) (*apipb.AgentOpsGetWorkflowProjectionResponse, error) {
			return &apipb.AgentOpsGetWorkflowProjectionResponse{
				Found: true,
				Workflow: &domainpb.AgentOpsWorkflowInstance{
					InstanceId: "wf-1", DomainKind: "initiative", DomainId: "ship-x",
					State:   domainpb.AgentOpsWorkflowState_AGENT_OPS_WORKFLOW_STATE_RUNNING,
					Version: 3,
					Decisions: []*domainpb.AgentOpsHumanDecision{
						{Decision: "retry-review", Actor: "operator", AtVersion: 2},
					},
					LegalActions: []domainpb.AgentOpsDomainAction{domainpb.AgentOpsDomainAction_AGENT_OPS_DOMAIN_ACTION_OPEN_REVIEW},
				},
				PolicyId: "initiative-default", PolicyRevision: "sha256:p",
				Operations: []*apipb.AgentOpsOperationProjection{
					{
						Operation: "review-round", ExecutionId: "exec-1", Attempt: 1, State: "failed", Outcome: "failed",
						SnapshotFound: true, Mode: "holistic-loop", ModeRevision: "sha256:r",
						BindingLayer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE,
					},
					{Operation: "review-round", ExecutionId: "exec-2", Attempt: 2, State: "running", PriorExecutionId: "exec-1"},
				},
			}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsWorkflow([]string{"--target-kind", "initiative", "--target", "ship-x"})
	})
	if !containsAll(out,
		"state     : running (v3)",
		"decisions : 1",
		"- open-review",
		"attempt=1 state=failed outcome=failed holistic-loop@sha256:r [initiative-override]",
		"attempt=2 state=running", "(snapshot missing)", "retry of exec-1") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdAgentOpsHistory_PassesLimit(t *testing.T) {
	var gotLimit int32
	stub := &stubAgentOperationsHandler{
		history: func(req *apipb.AgentOpsListExecutionHistoryRequest) (*apipb.AgentOpsListExecutionHistoryResponse, error) {
			gotLimit = req.GetLimit()
			return &apipb.AgentOpsListExecutionHistoryResponse{Executions: []*apipb.AgentOpsExecutionSummary{{
				ExecutionId: "exec-new", Operation: "review-round", OperationVersion: "1.0.0",
				Mode: "holistic-loop", ModeRevision: "sha256:r",
				BindingLayer: domainpb.AgentOpsBindingLayer_AGENT_OPS_BINDING_LAYER_SYSTEM_DEFAULT,
				Outcome:      "accepted", Reproducible: true, RecordedAt: "2026-07-14T10:00:00Z",
			}}}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentOpsHistory([]string{"--target-kind", "initiative", "--target", "ship-x", "--limit", "1"})
	})
	if gotLimit != 1 {
		t.Fatalf("limit not passed: %d", gotLimit)
	}
	if !containsAll(out, "exec=exec-new outcome=accepted (reproducible)", "[system-default]") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdAgentOpsMigrationStatus(t *testing.T) {
	stub := &stubAgentOperationsHandler{
		migration: func(*apipb.AgentOpsGetMigrationStatusRequest) (*apipb.AgentOpsGetMigrationStatusResponse, error) {
			return &apipb.AgentOpsGetMigrationStatusResponse{State: "not-started"}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error { return app.cmdAgentOpsMigrationStatus(nil) })
	if !containsAll(out, "state       : not-started", "Phase-8 migrator has not run") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdAgentOpsReconcile(t *testing.T) {
	stub := &stubAgentOperationsHandler{
		reconcile: func(*apipb.AgentOpsRunReconciliationRequest) (*apipb.AgentOpsRunReconciliationResponse, error) {
			return &apipb.AgentOpsRunReconciliationResponse{
				DirsScanned: 4, SnapshotsSeen: 7, SkippedTooRecent: 1,
				Reaped: []string{"/data/agentops/executions/exec-orphan.json"},
			}, nil
		},
	}
	app := newAgentOpsTestApp(t, stub)
	out := clitest.CaptureStdout(t, func() error { return app.cmdAgentOpsReconcile(nil) })
	if !containsAll(out, "dirs scanned       : 4", "reaped             : 1", "exec-orphan.json") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAgentOpsTargetKindVocabulary(t *testing.T) {
	for raw, want := range map[string]domainpb.OperatingModeTargetKind{
		"backlog-item":   domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_BACKLOG_ITEM,
		"initiative":     domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_INITIATIVE,
		"plan-execution": domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_PLAN_EXECUTION,
		"scenario":       domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_SCENARIO,
	} {
		got, err := agentOpsTargetKind(raw)
		if err != nil || got != want {
			t.Fatalf("agentOpsTargetKind(%q) = %v, %v; want %v", raw, got, err, want)
		}
	}
	if _, err := agentOpsTargetKind("nope"); err == nil {
		t.Fatal("unknown target kind must error")
	}
}

func TestAgentOperationsCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	for _, args := range [][]string{
		{"agent-operations", "catalog", "--help"},
		{"agent-operations", "compatible-modes", "--help"},
		{"agent-operations", "bindings", "--help"},
		{"agent-operations", "overrides", "--help"},
		{"agent-operations", "workflow", "--help"},
		{"agent-operations", "history", "--help"},
		{"agent-operations", "migration-status", "--help"},
		{"agent-operations", "reconcile", "--help"},
	} {
		if err := app.Run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
}

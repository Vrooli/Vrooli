package workflows

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
	workflowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows/workflows_v1connect"
)

// workflowService provides the exercised RPCs over a real Connect transport. The
// embedded interface keeps unrelated service methods intentionally unavailable.
type workflowService struct {
	workflowsconnect.WorkflowsServiceHandler
	start     *workflowsv1.StartWorkflowRequest
	list      *workflowsv1.ListWorkflowsRequest
	get       *workflowsv1.GetWorkflowRequest
	readiness *workflowsv1.GetPromotionReadinessRequest
	empty     bool
	rpcErr    error
}

func (s *workflowService) StartWorkflow(_ context.Context, req *connect.Request[workflowsv1.StartWorkflowRequest]) (*connect.Response[workflowsv1.StartWorkflowResponse], error) {
	s.start = req.Msg
	if s.rpcErr != nil {
		return nil, s.rpcErr
	}
	if s.empty {
		return connect.NewResponse(&workflowsv1.StartWorkflowResponse{}), nil
	}
	return connect.NewResponse(&workflowsv1.StartWorkflowResponse{Workflow: &workflowsv1.Workflow{Id: "wf-1", Status: workflowsv1.WorkflowStatus_WORKFLOW_STATUS_QUEUED, AssetId: "button", TargetScenario: "store", AgentManagerRunId: "run-1", Summary: "started"}, QueueDepth: 2}), nil
}

func (s *workflowService) ListWorkflows(_ context.Context, req *connect.Request[workflowsv1.ListWorkflowsRequest]) (*connect.Response[workflowsv1.ListWorkflowsResponse], error) {
	s.list = req.Msg
	if s.rpcErr != nil {
		return nil, s.rpcErr
	}
	if s.empty {
		return connect.NewResponse(&workflowsv1.ListWorkflowsResponse{}), nil
	}
	return connect.NewResponse(&workflowsv1.ListWorkflowsResponse{Workflows: []*workflowsv1.Workflow{{Id: "wf-1", Status: workflowsv1.WorkflowStatus_WORKFLOW_STATUS_SUCCEEDED, AssetId: "button", Summary: "ok"}}}), nil
}

func (s *workflowService) GetWorkflow(_ context.Context, req *connect.Request[workflowsv1.GetWorkflowRequest]) (*connect.Response[workflowsv1.GetWorkflowResponse], error) {
	s.get = req.Msg
	if s.rpcErr != nil {
		return nil, s.rpcErr
	}
	if s.empty {
		return connect.NewResponse(&workflowsv1.GetWorkflowResponse{}), nil
	}
	return connect.NewResponse(&workflowsv1.GetWorkflowResponse{Workflow: &workflowsv1.Workflow{Id: "wf-2", Status: workflowsv1.WorkflowStatus_WORKFLOW_STATUS_FAILED, Error: "broken"}}), nil
}

func (s *workflowService) GetPromotionReadiness(_ context.Context, req *connect.Request[workflowsv1.GetPromotionReadinessRequest]) (*connect.Response[workflowsv1.GetPromotionReadinessResponse], error) {
	s.readiness = req.Msg
	if s.rpcErr != nil {
		return nil, s.rpcErr
	}
	if s.empty {
		return connect.NewResponse(&workflowsv1.GetPromotionReadinessResponse{}), nil
	}
	return connect.NewResponse(&workflowsv1.GetPromotionReadinessResponse{Readiness: &workflowsv1.PromotionReadiness{LibraryId: "button", SelectedVersion: "1.2.0", Ready: false, Blockers: []string{"missing examples"}, AvailableExampleCount: 1, RequiredExampleCount: 3, ParityReportPresent: true, OriginReplacementPresent: true, OriginReplacementClean: false, NextValidationCommand: "go test ./..."}}), nil
}

func newTestHandlers(t *testing.T, svc *workflowService) *handlers {
	t.Helper()
	path, rpc := workflowsconnect.NewWorkflowsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, rpc)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &handlers{client: workflowsconnect.NewWorkflowsServiceClient(server.Client(), server.URL)}
}

func workflowContext(flags, positionals map[string]string) cliapp.RunContext {
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{
			Flags:       []cliapp.Flag{{Name: "kind"}, {Name: "asset-id"}, {Name: "source-scenario"}, {Name: "target-scenario"}, {Name: "source-path"}, {Name: "version"}, {Name: "idempotency-key"}, {Name: "confirm-overwrite"}, {Name: "override-validation"}, {Name: "active"}, {Name: "limit"}, {Name: "origin-scenario"}},
			Positionals: []cliapp.Positional{{Name: "id"}, {Name: "asset-id"}},
		}, Flags: flags, Positionals: positionals,
	})
}

func TestStartCallAndReport(t *testing.T) {
	svc := &workflowService{}
	h := newTestHandlers(t, svc)
	ctx := workflowContext(map[string]string{"kind": "adopt", "asset-id": "button", "source-scenario": "source", "target-scenario": "store", "source-path": "src/Button.tsx", "version": "1.2.0", "idempotency-key": "key", "confirm-overwrite": "true", "override-validation": "true"}, nil)
	msg, err := h.startCall(ctx)
	require.NoError(t, err)
	require.Equal(t, workflowsv1.WorkflowKind_WORKFLOW_KIND_ADOPT, svc.start.Kind)
	require.Equal(t, "button", svc.start.AssetId)
	require.Equal(t, "source", svc.start.SourceScenario)
	require.Equal(t, "store", svc.start.TargetScenario)
	require.Equal(t, "src/Button.tsx", svc.start.SourcePath)
	require.Equal(t, "1.2.0", svc.start.RequestedVersion)
	require.Equal(t, "key", svc.start.IdempotencyKey)
	require.True(t, svc.start.ConfirmOverwrite)
	require.True(t, svc.start.OverrideValidation)
	report := h.startReport(ctx, msg)
	require.Equal(t, []string{"Workflow queued with depth 2."}, report.Result)
	require.Contains(t, report.Changes[0], "wf-1 [WORKFLOW_STATUS_QUEUED]")
	require.ErrorContains(t, func() error {
		_, err := h.startCall(workflowContext(map[string]string{"kind": "bad"}, nil))
		return err
	}(), "--kind must be extract or adopt")
}

func TestListGetAndPromotionReadiness(t *testing.T) {
	svc := &workflowService{}
	h := newTestHandlers(t, svc)
	ctx := workflowContext(map[string]string{"asset-id": "button", "target-scenario": "store", "active": "true", "limit": "7"}, nil)
	list, err := h.listCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "button", svc.list.AssetId)
	require.Equal(t, "store", svc.list.TargetScenario)
	require.True(t, svc.list.ActiveOnly)
	require.Equal(t, int32(7), svc.list.Limit)
	require.Equal(t, []string{"Found 1 workflow(s)."}, h.listReport(ctx, list).Summary)
	require.Contains(t, h.listReport(ctx, list).Results[0], "button")
	badLimit := workflowContext(map[string]string{"limit": "many"}, nil)
	require.ErrorContains(t, func() error { _, err := h.listCall(badLimit); return err }(), "--limit must be an integer")

	getCtx := workflowContext(nil, map[string]string{"id": "wf-2"})
	got, err := h.getCall(getCtx)
	require.NoError(t, err)
	require.Equal(t, "wf-2", svc.get.Id)
	require.Contains(t, h.getReport(getCtx, got).Results[0], "broken")

	readyCtx := workflowContext(map[string]string{"origin-scenario": "source", "version": "1.2.0"}, map[string]string{"asset-id": "button"})
	ready, err := h.promotionReadinessCall(readyCtx)
	require.NoError(t, err)
	require.Equal(t, "button", svc.readiness.AssetId)
	require.Equal(t, "source", svc.readiness.OriginScenario)
	require.Equal(t, "1.2.0", svc.readiness.Version)
	report := h.promotionReadinessReport(readyCtx, ready)
	require.Equal(t, []string{"Promotion readiness for button@1.2.0: false."}, report.Summary)
	require.Equal(t, "missing examples", report.Results[0])
	require.Contains(t, report.Results[1], "examples 1/3; parity=true")
	require.Equal(t, []string{"go test ./..."}, report.RetrievalHints)
}

func TestWorkflowParsingFormattingAndEmptyResponses(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  workflowsv1.WorkflowKind
		ok    bool
	}{
		{"extract", "extract", workflowsv1.WorkflowKind_WORKFLOW_KIND_EXTRACT, true},
		{"adopt", "adopt", workflowsv1.WorkflowKind_WORKFLOW_KIND_ADOPT, true},
		{"invalid", "", workflowsv1.WorkflowKind_WORKFLOW_KIND_UNSPECIFIED, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseKind(tc.value)
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			} else {
				require.ErrorContains(t, err, "--kind must be extract or adopt")
			}
		})
	}
	require.Equal(t, "(nil)", format(nil))
	require.Equal(t, "id [WORKFLOW_STATUS_SUCCEEDED] asset=a target=t run=r summary", format(&workflowsv1.Workflow{Id: "id", Status: workflowsv1.WorkflowStatus_WORKFLOW_STATUS_SUCCEEDED, AssetId: "a", TargetScenario: "t", AgentManagerRunId: "r", Summary: "summary"}))
	require.True(t, strings.HasSuffix(format(&workflowsv1.Workflow{Id: "id", Error: "failure", Summary: "ignored"}), "failure"))
}

func TestWorkflowCallFailuresAndMissingPayloads(t *testing.T) {
	for _, tc := range []struct {
		name string
		rpc  func(*handlers, cliapp.RunContext) error
	}{
		{"start", func(h *handlers, c cliapp.RunContext) error { _, e := h.startCall(c); return e }},
		{"list", func(h *handlers, c cliapp.RunContext) error { _, e := h.listCall(c); return e }},
		{"get", func(h *handlers, c cliapp.RunContext) error { _, e := h.getCall(c); return e }},
		{"readiness", func(h *handlers, c cliapp.RunContext) error { _, e := h.promotionReadinessCall(c); return e }},
	} {
		t.Run(tc.name+" api error", func(t *testing.T) {
			svc := &workflowService{rpcErr: errors.New("backend down")}
			h := newTestHandlers(t, svc)
			err := tc.rpc(h, workflowContext(map[string]string{"kind": "extract"}, map[string]string{"id": "wf", "asset-id": "asset"}))
			require.Error(t, err)
			require.Contains(t, err.Error(), "backend down")
		})
	}
	for _, tc := range []struct {
		name string
		rpc  func(*handlers, cliapp.RunContext) error
		want string
	}{
		{"start", func(h *handlers, c cliapp.RunContext) error { _, e := h.startCall(c); return e }, "server returned no workflow"},
		{"get", func(h *handlers, c cliapp.RunContext) error { _, e := h.getCall(c); return e }, "server returned no workflow"},
		{"readiness", func(h *handlers, c cliapp.RunContext) error { _, e := h.promotionReadinessCall(c); return e }, "server returned no promotion readiness"},
	} {
		t.Run(tc.name+" missing payload", func(t *testing.T) {
			h := newTestHandlers(t, &workflowService{empty: true})
			err := tc.rpc(h, workflowContext(map[string]string{"kind": "extract"}, map[string]string{"id": "wf", "asset-id": "asset"}))
			require.ErrorContains(t, err, tc.want)
		})
	}
}

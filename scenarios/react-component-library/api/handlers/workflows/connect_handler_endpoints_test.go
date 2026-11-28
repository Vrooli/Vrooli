package workflows

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	workflowspb "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows"
	internal "react-component-library/internal/workflows"
)

type endpointService struct {
	workflow   internal.Workflow
	readiness  internal.PromotionReadiness
	err        error
	startInput internal.StartInput
	listArgs   struct {
		asset, target string
		active        bool
		limit         int
	}
	readinessInput internal.PromotionReadinessInput
}

func (s *endpointService) Start(_ context.Context, in internal.StartInput) (internal.Workflow, int, error) {
	s.startInput = in
	return s.workflow, 4, s.err
}

func (s *endpointService) List(_ context.Context, asset, target string, active bool, limit int) ([]internal.Workflow, error) {
	s.listArgs.asset, s.listArgs.target, s.listArgs.active, s.listArgs.limit = asset, target, active, limit
	return []internal.Workflow{s.workflow}, s.err
}

func (s *endpointService) Get(context.Context, string) (internal.Workflow, error) {
	return s.workflow, s.err
}

func (s *endpointService) Refresh(context.Context, string) (internal.Workflow, error) {
	return s.workflow, s.err
}

func (s *endpointService) Stop(context.Context, string) (internal.Workflow, error) {
	return s.workflow, s.err
}

func (s *endpointService) Retry(context.Context, string, string) (internal.Workflow, int, error) {
	return s.workflow, 2, s.err
}

func (s *endpointService) PromotionReadiness(_ context.Context, in internal.PromotionReadinessInput) (internal.PromotionReadiness, error) {
	s.readinessInput = in
	return s.readiness, s.err
}

func TestConnectWorkflowEndpointsReturnDomainResults(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	svc := &endpointService{
		workflow:  internal.Workflow{ID: "wf-7", Kind: internal.KindAdopt, Status: internal.StatusQueued, AssetID: "asset-1", CreatedAt: now, UpdatedAt: now},
		readiness: internal.PromotionReadiness{AssetID: "asset-1", LibraryID: "lib-1", Ready: true, DependencyLibraryIDs: []string{"lib-dep"}},
	}
	h := NewConnectHandler(svc, nil)
	ctx := context.Background()

	started, err := h.StartWorkflow(ctx, connect.NewRequest(&workflowspb.StartWorkflowRequest{Kind: workflowspb.WorkflowKind_WORKFLOW_KIND_ADOPT, AssetId: "asset-1", TargetScenario: "dashboard", IdempotencyKey: "request-1", ConfirmOverwrite: true, OverrideValidation: true}))
	if err != nil {
		t.Fatal(err)
	}
	if started.Msg.QueueDepth != 4 || started.Msg.Workflow.Id != "wf-7" || !started.Msg.Workflow.CanStop {
		t.Fatalf("unexpected start response: %#v", started.Msg)
	}
	if svc.startInput.Kind != internal.KindAdopt || svc.startInput.AssetID != "asset-1" || svc.startInput.TargetScenario != "dashboard" || !svc.startInput.ConfirmOverwrite || !svc.startInput.OverrideValidation {
		t.Fatalf("start request not forwarded: %#v", svc.startInput)
	}

	listed, err := h.ListWorkflows(ctx, connect.NewRequest(&workflowspb.ListWorkflowsRequest{AssetId: "asset-1", TargetScenario: "dashboard", ActiveOnly: true, Limit: 25}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.Workflows) != 1 || listed.Msg.Workflows[0].Id != "wf-7" || svc.listArgs != (struct {
		asset, target string
		active        bool
		limit         int
	}{"asset-1", "dashboard", true, 25}) {
		t.Fatalf("unexpected list response/arguments: %#v, %#v", listed.Msg, svc.listArgs)
	}

	for name, call := range map[string]func() (*workflowspb.Workflow, error){
		"get": func() (*workflowspb.Workflow, error) {
			r, e := h.GetWorkflow(ctx, connect.NewRequest(&workflowspb.GetWorkflowRequest{Id: "wf-7"}))
			if e != nil {
				return nil, e
			}
			return r.Msg.Workflow, nil
		},
		"refresh": func() (*workflowspb.Workflow, error) {
			r, e := h.RefreshWorkflow(ctx, connect.NewRequest(&workflowspb.RefreshWorkflowRequest{Id: "wf-7"}))
			if e != nil {
				return nil, e
			}
			return r.Msg.Workflow, nil
		},
		"stop": func() (*workflowspb.Workflow, error) {
			r, e := h.StopWorkflow(ctx, connect.NewRequest(&workflowspb.StopWorkflowRequest{Id: "wf-7"}))
			if e != nil {
				return nil, e
			}
			return r.Msg.Workflow, nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			w, e := call()
			if e != nil {
				t.Fatal(e)
			}
			if w.Id != "wf-7" || w.Kind != workflowspb.WorkflowKind_WORKFLOW_KIND_ADOPT {
				t.Fatalf("unexpected workflow: %#v", w)
			}
		})
	}

	retried, err := h.RetryWorkflow(ctx, connect.NewRequest(&workflowspb.RetryWorkflowRequest{Id: "wf-7", IdempotencyKey: "retry-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if retried.Msg.QueueDepth != 2 || retried.Msg.Workflow.Id != "wf-7" {
		t.Fatalf("unexpected retry response: %#v", retried.Msg)
	}

	ready, err := h.GetPromotionReadiness(ctx, connect.NewRequest(&workflowspb.GetPromotionReadinessRequest{AssetId: "asset-1", OriginScenario: "dashboard", Version: "1.2.0"}))
	if err != nil {
		t.Fatal(err)
	}
	if !ready.Msg.Readiness.Ready || ready.Msg.Readiness.LibraryId != "lib-1" || len(ready.Msg.Readiness.DependencyLibraryIds) != 1 {
		t.Fatalf("unexpected readiness response: %#v", ready.Msg.Readiness)
	}
	if svc.readinessInput != (internal.PromotionReadinessInput{AssetID: "asset-1", OriginScenario: "dashboard", Version: "1.2.0"}) {
		t.Fatalf("readiness request not forwarded: %#v", svc.readinessInput)
	}
}

func TestConnectWorkflowEndpointsMapServiceErrors(t *testing.T) {
	svc := &endpointService{err: errors.Join(errors.New("lookup failed"), internal.ErrNotFound)}
	h := NewConnectHandler(svc, nil)
	_, err := h.GetWorkflow(context.Background(), connect.NewRequest(&workflowspb.GetWorkflowRequest{Id: "missing"}))
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("GetWorkflow error code = %v, want not found", got)
	}
}

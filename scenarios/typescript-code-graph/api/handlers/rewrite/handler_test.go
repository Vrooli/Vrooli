package rewrite

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	intgraph "typescript-code-graph/internal/graph"
	rewritedom "typescript-code-graph/internal/rewrite"
	"typescript-code-graph/internal/sidecar"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	rewritepb "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/rewrite"
)

func newSvc(t *testing.T, fake *sidecarmocks.FakeSidecarClient) *rewritedom.Service {
	t.Helper()
	if fake == nil {
		fake = &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}
	}
	store := rewritedom.NewMemoryPlanStore()
	return rewritedom.NewService(store, fake, intgraph.NewPathMutex())
}

func TestRewritePlan_HappyPath(t *testing.T) {
	svc := newSvc(t, nil)
	req := connect.NewRequest(&graphv1.RewritePlanRequest{
		ProjectPath: "/abs/proj",
		Operations: []*rewritepb.Operation{
			{Op: &rewritepb.Operation_FileMove{FileMove: &rewritepb.FileMove{FromPath: "a.ts", ToPath: "b.ts"}}},
		},
	})
	resp, err := RewritePlan(context.Background(), req, svc)
	if err != nil {
		t.Fatalf("RewritePlan: %v", err)
	}
	if resp.Msg.GetPlanId() == "" {
		t.Errorf("plan_id missing")
	}
	if len(resp.Msg.GetNormalizedOperations()) != 1 {
		t.Errorf("normalized_operations missing")
	}
}

func TestRewritePlan_InvalidArgument(t *testing.T) {
	svc := newSvc(t, nil)
	req := connect.NewRequest(&graphv1.RewritePlanRequest{ProjectPath: ""})
	_, err := RewritePlan(context.Background(), req, svc)
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestRewriteApply_RejectsApplyFalse(t *testing.T) {
	svc := newSvc(t, nil)
	req := connect.NewRequest(&graphv1.RewriteApplyRequest{
		ProjectPath: "/abs/proj",
		PlanId:      "any",
		Apply:       false,
	})
	_, err := RewriteApply(context.Background(), req, svc, false)
	if err == nil {
		t.Fatal("expected InvalidArgument when apply=false")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestRewriteApply_PlanNotFound(t *testing.T) {
	svc := newSvc(t, nil)
	req := connect.NewRequest(&graphv1.RewriteApplyRequest{
		ProjectPath: "/abs/proj",
		PlanId:      "missing",
		Apply:       true,
	})
	_, err := RewriteApply(context.Background(), req, svc, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("want FailedPrecondition, got %v", connect.CodeOf(err))
	}
}

func TestRewriteApply_DryRunSkipsSidecar(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusUnhealthy} // unhealthy on purpose
	svc := newSvc(t, fake)

	// Plan first.
	planReq := connect.NewRequest(&graphv1.RewritePlanRequest{
		ProjectPath: "/abs/proj",
		Operations: []*rewritepb.Operation{
			{Op: &rewritepb.Operation_FileMove{FileMove: &rewritepb.FileMove{FromPath: "a.ts", ToPath: "b.ts"}}},
		},
	})
	planResp, err := RewritePlan(context.Background(), planReq, svc)
	if err != nil {
		t.Fatalf("RewritePlan: %v", err)
	}

	applyReq := connect.NewRequest(&graphv1.RewriteApplyRequest{
		ProjectPath: "/abs/proj",
		PlanId:      planResp.Msg.GetPlanId(),
		Apply:       true,
	})
	applyResp, err := RewriteApply(context.Background(), applyReq, svc, true)
	if err != nil {
		t.Fatalf("RewriteApply dry-run: %v", err)
	}
	if !applyResp.Msg.GetDryRun() {
		t.Errorf("dry_run should be true")
	}
	if len(applyResp.Msg.GetResults()) != 1 {
		t.Errorf("expected 1 synthetic result")
	}
	if fake.RewriteApplyCalls != 0 {
		t.Errorf("sidecar must not be called on dry-run; got %d", fake.RewriteApplyCalls)
	}
}

func TestRewriteApply_RealApplyHitsSidecar(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		RewriteApplyFn: func(ctx context.Context, p string, ops []sidecar.Operation) ([]sidecar.OperationResult, error) {
			out := make([]sidecar.OperationResult, len(ops))
			for i := range ops {
				out[i] = sidecar.OperationResult{Status: "OPERATION_STATUS_OK"}
			}
			return out, nil
		},
	}
	svc := newSvc(t, fake)

	planReq := connect.NewRequest(&graphv1.RewritePlanRequest{
		ProjectPath: "/abs/proj",
		Operations: []*rewritepb.Operation{
			{Op: &rewritepb.Operation_FileMove{FileMove: &rewritepb.FileMove{FromPath: "a.ts", ToPath: "b.ts"}}},
		},
	})
	planResp, err := RewritePlan(context.Background(), planReq, svc)
	if err != nil {
		t.Fatalf("RewritePlan: %v", err)
	}

	applyReq := connect.NewRequest(&graphv1.RewriteApplyRequest{
		ProjectPath: "/abs/proj",
		PlanId:      planResp.Msg.GetPlanId(),
		Apply:       true,
	})
	applyResp, err := RewriteApply(context.Background(), applyReq, svc, false)
	if err != nil {
		t.Fatalf("RewriteApply: %v", err)
	}
	if applyResp.Msg.GetDryRun() {
		t.Errorf("dry_run should be false")
	}
	if fake.RewriteApplyCalls != 1 {
		t.Errorf("expected 1 sidecar call, got %d", fake.RewriteApplyCalls)
	}
	if applyResp.Msg.GetResults()[0].GetStatus() != rewritepb.OperationStatus_OPERATION_STATUS_OK {
		t.Errorf("expected OK status")
	}
}

func TestEndpoints_StableIDs(t *testing.T) {
	ids := map[string]bool{}
	for _, ep := range Endpoints {
		if ep.Path == "" {
			t.Errorf("endpoint %s missing path", ep.ID)
		}
		if ids[ep.ID] {
			t.Errorf("duplicate endpoint id %s", ep.ID)
		}
		ids[ep.ID] = true
	}
	if !ids["rewrite_plan"] || !ids["rewrite_apply"] {
		t.Errorf("missing rewrite endpoints")
	}
}

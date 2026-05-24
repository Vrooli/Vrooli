package graph

import (
	"context"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	intgraph "go-code-graph/internal/graph"
	graphmocks "go-code-graph/internal/graph/mocks"
	intrewrite "go-code-graph/internal/rewrite"
	rewritemocks "go-code-graph/internal/rewrite/mocks"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
	rewrite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/rewrite"
)

// newTestClientWithRewrite spins up a Connect server with both
// services wired and returns a paired client.
func newTestClientWithRewrite(t *testing.T, rsvc *intrewrite.Service) graph_v1connect.GoCodeGraphServiceClient {
	t.Helper()
	gsvc := intgraph.NewService(&graphmocks.FakeLoader{}, intgraph.NewPathMutex())
	_, h := graph_v1connect.NewGoCodeGraphServiceHandler(NewConnectHandler(Deps{
		GraphService:   gsvc,
		RewriteService: rsvc,
		Logger:         log.New(io.Discard, "", 0),
	}))
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return graph_v1connect.NewGoCodeGraphServiceClient(server.Client(), server.URL)
}

// newRewriteService builds a rewrite service backed by an in-memory
// store and a fake executor that records the ops it sees.
func newRewriteService(t *testing.T, exec intrewrite.RewriteExecutor) (*intrewrite.Service, intrewrite.PlanStore) {
	t.Helper()
	store := intrewrite.NewMemoryStore()
	return intrewrite.NewService(store, exec, intgraph.NewPathMutex()), store
}

// fileMoveOp returns a FileMove proto operation.
func fileMoveOp(from, to string) *rewrite_v1.Operation {
	return &rewrite_v1.Operation{
		Op: &rewrite_v1.Operation_FileMove{
			FileMove: &rewrite_v1.FileMove{FromPath: from, ToPath: to},
		},
	}
}

func TestHandlerRewritePlanHappyPath(t *testing.T) {
	t.Parallel()
	svc, _ := newRewriteService(t, &rewritemocks.FakeExecutor{})
	client := newTestClientWithRewrite(t, svc)

	resp, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/tmp/x",
		Operations:   []*rewrite_v1.Operation{fileMoveOp("a/b.go", "c/d.go")},
	}))
	if err != nil {
		t.Fatalf("RewritePlan: %v", err)
	}
	if resp.Msg.GetPlanId() == "" {
		t.Fatalf("expected non-empty plan_id")
	}
	if got := len(resp.Msg.GetNormalizedOperations()); got != 1 {
		t.Fatalf("normalized_operations: want 1, got %d", got)
	}
}

func TestHandlerRewritePlanEmptyIsInvalidArgument(t *testing.T) {
	t.Parallel()
	svc, _ := newRewriteService(t, &rewritemocks.FakeExecutor{})
	client := newTestClientWithRewrite(t, svc)

	_, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/tmp/x",
	}))
	if err == nil {
		t.Fatal("expected error for empty operations")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestHandlerRewriteApplyHappyPath(t *testing.T) {
	t.Parallel()
	var seenOps []intrewrite.Operation
	exec := &rewritemocks.FakeExecutor{
		ExecuteFunc: func(_ context.Context, _ string, op intrewrite.Operation) error {
			seenOps = append(seenOps, op)
			return nil
		},
	}
	svc, _ := newRewriteService(t, exec)
	client := newTestClientWithRewrite(t, svc)

	planResp, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/tmp/x",
		Operations:   []*rewrite_v1.Operation{fileMoveOp("a.go", "b.go")},
	}))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applyResp, err := client.RewriteApply(context.Background(), connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: "/tmp/x",
		PlanId:       planResp.Msg.GetPlanId(),
		Apply:        true,
	}))
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applyResp.Msg.GetDryRun() {
		t.Fatalf("expected dry_run=false")
	}
	if got := len(applyResp.Msg.GetResults()); got != 1 {
		t.Fatalf("results: want 1, got %d", got)
	}
	if applyResp.Msg.GetResults()[0].GetStatus() != rewrite_v1.OperationStatus_OPERATION_STATUS_OK {
		t.Fatalf("expected OK status")
	}
	if len(seenOps) != 1 {
		t.Fatalf("executor saw %d ops, want 1", len(seenOps))
	}
}

func TestHandlerRewriteApplyApplyFalseIsInvalidArgument(t *testing.T) {
	t.Parallel()
	svc, _ := newRewriteService(t, &rewritemocks.FakeExecutor{})
	client := newTestClientWithRewrite(t, svc)

	planResp, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/tmp/x",
		Operations:   []*rewrite_v1.Operation{fileMoveOp("a.go", "b.go")},
	}))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	_, err = client.RewriteApply(context.Background(), connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: "/tmp/x",
		PlanId:       planResp.Msg.GetPlanId(),
		Apply:        false,
	}))
	if err == nil {
		t.Fatal("expected error when apply=false")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestHandlerRewriteApplyDryRunHeaderThreadsThrough(t *testing.T) {
	t.Parallel()
	called := false
	exec := &rewritemocks.FakeExecutor{
		ExecuteFunc: func(_ context.Context, _ string, _ intrewrite.Operation) error {
			called = true
			return nil
		},
	}
	svc, _ := newRewriteService(t, exec)
	client := newTestClientWithRewrite(t, svc)

	planResp, err := client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: "/tmp/x",
		Operations:   []*rewrite_v1.Operation{fileMoveOp("a.go", "b.go")},
	}))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	req := connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: "/tmp/x",
		PlanId:       planResp.Msg.GetPlanId(),
		Apply:        true,
	})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := client.RewriteApply(context.Background(), req)
	if err != nil {
		t.Fatalf("Apply dry-run: %v", err)
	}
	if !resp.Msg.GetDryRun() {
		t.Fatalf("expected dry_run=true on response")
	}
	if called {
		t.Fatalf("executor must not be invoked in dry-run")
	}
}

func TestHandlerRewriteApplyUnknownPlanIsInvalidArgument(t *testing.T) {
	t.Parallel()
	svc, _ := newRewriteService(t, &rewritemocks.FakeExecutor{})
	client := newTestClientWithRewrite(t, svc)

	_, err := client.RewriteApply(context.Background(), connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: "/tmp/x",
		PlanId:       "deadbeef",
		Apply:        true,
	}))
	if err == nil {
		t.Fatal("expected error for unknown plan_id")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

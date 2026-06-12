package rewrite

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
	rewritev1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/rewrite"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "typescript-code-graph/cli/internal/testutil"
)

type fakeService struct {
	graphconnect.UnimplementedTypeScriptCodeGraphServiceHandler

	mu        sync.Mutex
	planResp  *graphv1.RewritePlanResponse
	planReqs  []*graphv1.RewritePlanRequest
	planErr   error
	applyResp *graphv1.RewriteApplyResponse
	applyReqs []*graphv1.RewriteApplyRequest
	applyErr  error
}

func (s *fakeService) RewritePlan(_ context.Context, req *connect.Request[graphv1.RewritePlanRequest]) (*connect.Response[graphv1.RewritePlanResponse], error) {
	s.mu.Lock()
	s.planReqs = append(s.planReqs, req.Msg)
	s.mu.Unlock()
	if s.planErr != nil {
		return nil, s.planErr
	}
	resp := s.planResp
	if resp == nil {
		resp = &graphv1.RewritePlanResponse{}
	}
	return connect.NewResponse(resp), nil
}

func (s *fakeService) RewriteApply(_ context.Context, req *connect.Request[graphv1.RewriteApplyRequest]) (*connect.Response[graphv1.RewriteApplyResponse], error) {
	s.mu.Lock()
	s.applyReqs = append(s.applyReqs, req.Msg)
	s.mu.Unlock()
	if s.applyErr != nil {
		return nil, s.applyErr
	}
	resp := s.applyResp
	if resp == nil {
		resp = &graphv1.RewriteApplyResponse{}
	}
	return connect.NewResponse(resp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := graphconnect.NewTypeScriptCodeGraphServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func writeOpsFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ops.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestPlan_ParsesOpsAndRenders(t *testing.T) {
	svc := &fakeService{planResp: &graphv1.RewritePlanResponse{
		PlanId: "plan-123",
		NormalizedOperations: []*rewritev1.Operation{
			{Op: &rewritev1.Operation_FileMove{FileMove: &rewritev1.FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}}},
			{Op: &rewritev1.Operation_ImportRewrite{ImportRewrite: &rewritev1.ImportRewrite{OldPath: "./a", NewPath: "./b"}}},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)

	opsPath := writeOpsFile(t, `[
		{"kind": "file_move", "from_path": "src/a.ts", "to_path": "src/b.ts"},
		{"kind": "import_rewrite", "old_path": "./a", "new_path": "./b"}
	]`)

	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "ops-file", Required: true}},
		Flags:       []cliapp.Flag{{Name: "project-path", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"ops-file": opsPath},
		Flags:       map[string]string{"project-path": "/tmp/proj"},
	})

	require.NoError(t, h.plan(ctx))
	require.Len(t, svc.planReqs, 1)
	require.Equal(t, "/tmp/proj", svc.planReqs[0].GetProjectPath())
	require.Len(t, svc.planReqs[0].GetOperations(), 2)
	require.Equal(t, "src/a.ts", svc.planReqs[0].GetOperations()[0].GetFileMove().GetFromPath())
	require.Equal(t, "./a", svc.planReqs[0].GetOperations()[1].GetImportRewrite().GetOldPath())

	body := out.String()
	require.Contains(t, body, "Plan plan-123 ready (2 op(s))")
	require.Contains(t, body, "file_move: src/a.ts -> src/b.ts")
	require.Contains(t, body, "import_rewrite: ./a -> ./b")
	require.Contains(t, body, "rewrite apply plan-123 --project-path /tmp/proj")
}

func TestPlan_RejectsEmptyOpsFile(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	h := newHandlers(core)
	opsPath := writeOpsFile(t, `[]`)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "ops-file", Required: true}},
		Flags:       []cliapp.Flag{{Name: "project-path", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"ops-file": opsPath},
		Flags:       map[string]string{"project-path": "/tmp/proj"},
	})
	err := h.plan(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no operations")
}

func TestPlan_RejectsUnknownKind(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	h := newHandlers(core)
	opsPath := writeOpsFile(t, `[{"kind":"bogus"}]`)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "ops-file", Required: true}},
		Flags:       []cliapp.Flag{{Name: "project-path", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"ops-file": opsPath},
		Flags:       map[string]string{"project-path": "/tmp/proj"},
	})
	err := h.plan(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown kind")
}

func TestApply_RendersResultsAndDryRun(t *testing.T) {
	svc := &fakeService{applyResp: &graphv1.RewriteApplyResponse{
		PlanId: "plan-123",
		Results: []*rewritev1.OperationResult{
			{
				Operation: &rewritev1.Operation{Op: &rewritev1.Operation_FileMove{FileMove: &rewritev1.FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}}},
				Status:    rewritev1.OperationStatus_OPERATION_STATUS_OK,
			},
		},
		DryRun: true,
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "plan-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "project-path", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"plan-id": "plan-123"},
		Flags:       map[string]string{"project-path": "/tmp/proj"},
	})

	require.NoError(t, h.apply(ctx))
	require.Len(t, svc.applyReqs, 1)
	require.Equal(t, "plan-123", svc.applyReqs[0].GetPlanId())
	require.Equal(t, "/tmp/proj", svc.applyReqs[0].GetProjectPath())
	require.True(t, svc.applyReqs[0].GetApply(), "apply must be true; dry-run is threaded via X-Dry-Run header")

	body := out.String()
	require.Contains(t, body, "DRY RUN")
	require.Contains(t, body, "Applied plan plan-123")
	require.Contains(t, body, "OPERATION_STATUS_OK")
	require.Contains(t, body, "file_move: src/a.ts -> src/b.ts")
}

func TestApply_SurfacesFailures(t *testing.T) {
	svc := &fakeService{applyResp: &graphv1.RewriteApplyResponse{
		PlanId: "plan-9",
		Results: []*rewritev1.OperationResult{
			{
				Operation: &rewritev1.Operation{Op: &rewritev1.Operation_ImportRewrite{ImportRewrite: &rewritev1.ImportRewrite{OldPath: "./x", NewPath: "./y"}}},
				Status:    rewritev1.OperationStatus_OPERATION_STATUS_FAILED,
				Message:   "no files matched",
			},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "plan-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "project-path", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"plan-id": "plan-9"},
		Flags:       map[string]string{"project-path": "/tmp/proj"},
	})

	require.NoError(t, h.apply(ctx))
	body := out.String()
	require.Contains(t, body, "1 op(s), 1 failed")
	require.Contains(t, body, "no files matched")
	require.NotContains(t, body, "DRY RUN")
}

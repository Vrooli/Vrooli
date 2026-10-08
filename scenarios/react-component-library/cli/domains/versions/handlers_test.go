package versions

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

type versionsService struct {
	mu       sync.Mutex
	listResp *versionsv1.ListVersionsResponse
	getResp  *versionsv1.GetVersionResponse
	diffResp *versionsv1.DiffVersionsResponse
	listReqs []*versionsv1.ListVersionsRequest
	getReqs  []*versionsv1.GetVersionRequest
	diffReqs []*versionsv1.DiffVersionsRequest
}

func (s *versionsService) ListVersions(_ context.Context, req *connect.Request[versionsv1.ListVersionsRequest]) (*connect.Response[versionsv1.ListVersionsResponse], error) {
	s.mu.Lock()
	s.listReqs = append(s.listReqs, req.Msg)
	s.mu.Unlock()
	if s.listResp == nil {
		s.listResp = &versionsv1.ListVersionsResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *versionsService) GetVersion(_ context.Context, req *connect.Request[versionsv1.GetVersionRequest]) (*connect.Response[versionsv1.GetVersionResponse], error) {
	s.mu.Lock()
	s.getReqs = append(s.getReqs, req.Msg)
	s.mu.Unlock()
	if s.getResp == nil {
		s.getResp = &versionsv1.GetVersionResponse{Version: sampleVersion()}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *versionsService) DiffVersions(_ context.Context, req *connect.Request[versionsv1.DiffVersionsRequest]) (*connect.Response[versionsv1.DiffVersionsResponse], error) {
	s.mu.Lock()
	s.diffReqs = append(s.diffReqs, req.Msg)
	s.mu.Unlock()
	if s.diffResp == nil {
		s.diffResp = &versionsv1.DiffVersionsResponse{}
	}
	return connect.NewResponse(s.diffResp), nil
}

func connectAPI(t *testing.T, svc *versionsService) http.Handler {
	t.Helper()
	path, handler := versionsconnect.NewVersionsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

type lifecycleService struct {
	versionsconnect.UnimplementedVersionLifecycleServiceHandler
	planResp    *versionsv1.PlanCleanupResponse
	cleanupResp *versionsv1.CleanupVersionsResponse
	planCalls   int
	cleanupReq  *versionsv1.CleanupVersionsRequest
}

func (s *lifecycleService) PlanCleanup(_ context.Context, _ *connect.Request[versionsv1.PlanCleanupRequest]) (*connect.Response[versionsv1.PlanCleanupResponse], error) {
	s.planCalls++
	return connect.NewResponse(s.planResp), nil
}

func (s *lifecycleService) CleanupVersions(_ context.Context, req *connect.Request[versionsv1.CleanupVersionsRequest]) (*connect.Response[versionsv1.CleanupVersionsResponse], error) {
	s.cleanupReq = req.Msg
	return connect.NewResponse(s.cleanupResp), nil
}

func connectLifecycleAPI(t *testing.T, versionsSvc *versionsService, lifecycleSvc *lifecycleService) http.Handler {
	t.Helper()
	versionsPath, versionsHandler := versionsconnect.NewVersionsServiceHandler(versionsSvc)
	lifecyclePath, lifecycleHandler := versionsconnect.NewVersionLifecycleServiceHandler(lifecycleSvc)
	mux := http.NewServeMux()
	mux.Handle(versionsPath, versionsHandler)
	mux.Handle(lifecyclePath, lifecycleHandler)
	return mux
}

func sampleVersion() *versionsv1.Version {
	ts := timestamppb.New(time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC))
	return &versionsv1.Version{
		Id:            "ver-1",
		ComponentId:   "cmp-btn",
		Version:       "1.0.0",
		ContentSha256: "abc123def456",
		ChangelogMd:   "auto",
		RecordedAt:    ts,
	}
}

func TestVersionsList_ForwardsPositionalAndLimit(t *testing.T) {
	svc := &versionsService{listResp: &versionsv1.ListVersionsResponse{
		Versions: []*versionsv1.Version{sampleVersion()},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}},
		Flags:       []cliapp.Flag{{Name: "limit"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-btn"},
		Flags:       map[string]string{"limit": "25"},
	})
	msg, err := h.listCall(ctx)
	require.NoError(t, err)
	require.NoError(t, ctx.RenderList(h.listReport(ctx, msg)))
	require.Len(t, svc.listReqs, 1)
	require.Equal(t, "cmp-btn", svc.listReqs[0].ComponentId)
	require.Equal(t, int32(25), svc.listReqs[0].Limit)
	require.Contains(t, out.String(), "Found 1 version(s).")
}

func TestVersionsShow_PassesIncludeContent(t *testing.T) {
	svc := &versionsService{getResp: &versionsv1.GetVersionResponse{
		Version: sampleVersion(),
		Content: "actual file body here",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "version"}},
		Flags:       []cliapp.Flag{{Name: "with-content"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-btn", "version": "1.0.0"},
		Flags:       map[string]string{"with-content": "1"},
	})
	msg, err := h.showCall(ctx)
	require.NoError(t, err)
	require.NoError(t, ctx.RenderList(h.showReport(ctx, msg)))
	require.Len(t, svc.getReqs, 1)
	require.True(t, svc.getReqs[0].IncludeContent)
	require.Contains(t, out.String(), "actual file body here")
}

func TestVersionsDiff_RendersSummaryAndRows(t *testing.T) {
	svc := &versionsService{diffResp: &versionsv1.DiffVersionsResponse{
		FromLabel: "1.0.0", ToLabel: "1.0.1",
		Additions: 2, Removals: 1,
		Rows: []*versionsv1.DiffRow{
			{
				Left:  &versionsv1.DiffCell{LineNumber: 1, Text: "alpha", Op: versionsv1.DiffOp_DIFF_OP_EQUAL},
				Right: &versionsv1.DiffCell{LineNumber: 1, Text: "alpha", Op: versionsv1.DiffOp_DIFF_OP_EQUAL},
			},
			{
				Left:  &versionsv1.DiffCell{LineNumber: 2, Text: "beta", Op: versionsv1.DiffOp_DIFF_OP_REMOVE},
				Right: &versionsv1.DiffCell{Op: versionsv1.DiffOp_DIFF_OP_EMPTY},
			},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "from"}, {Name: "to"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-btn", "from": "1.0.0", "to": "1.0.1"},
	})
	msg, err := h.diffCall(ctx)
	require.NoError(t, err)
	require.NoError(t, ctx.RenderList(h.diffReport(ctx, msg)))
	require.Len(t, svc.diffReqs, 1)
	require.Equal(t, "1.0.0", svc.diffReqs[0].From)
	body := out.String()
	require.Contains(t, body, "1.0.0 → 1.0.1 : +2 / -1")
	require.Contains(t, body, "alpha")
}

func TestRetireSchemaDoesNotRequireCleanupPlanHash(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &versionsService{}))
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}, {Name: "version"}},
		Flags:       []cliapp.Flag{{Name: "confirm"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "cmp-btn", "version": "1.0.0"},
		Flags:       map[string]string{"confirm": "yes"},
	})
	require.False(t, ctx.FlagDeclared("plan-hash"))
}

func TestReapRequiresConfirmationForMutation(t *testing.T) {
	lifecycleSvc := &lifecycleService{
		planResp:    &versionsv1.PlanCleanupResponse{PlanHash: "hash"},
		cleanupResp: &versionsv1.CleanupVersionsResponse{PlanHash: "hash", Applied: true},
	}
	core := clitest.NewTestApp(t, connectLifecycleAPI(t, &versionsService{}, lifecycleSvc))
	h := newHandlers(core)
	flags := []cliapp.Flag{
		{Name: "component-id"},
		{Name: "library-id"},
		{Name: "older-than-days"},
		{Name: "confirm", Bool: true},
		{Name: "plan-hash"},
	}

	dryRun, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: flags,
	}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.reap(dryRun))
	require.Equal(t, 1, lifecycleSvc.planCalls)
	require.Nil(t, lifecycleSvc.cleanupReq)

	confirm, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: flags,
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"confirm": "true", "plan-hash": "hash"},
	})
	require.NoError(t, h.reap(confirm))
	require.NotNil(t, lifecycleSvc.cleanupReq)
	require.True(t, lifecycleSvc.cleanupReq.Confirm)
	require.Equal(t, "hash", lifecycleSvc.cleanupReq.PlanHash)
}

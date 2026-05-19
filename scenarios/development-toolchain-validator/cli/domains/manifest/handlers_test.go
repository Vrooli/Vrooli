package manifest

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest"
	manifestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest/manifest_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "development-toolchain-validator/cli/internal/testutil"
)

type fakeService struct {
	mu sync.Mutex

	listResp   *manifestv1.ListManifestsResponse
	getResp    *manifestv1.GetManifestResponse
	upsertResp *manifestv1.UpsertManifestResponse
	clearResp  *manifestv1.ClearStaleResponse

	listErr   error
	getErr    error
	upsertErr error
	clearErr  error

	getReq    *manifestv1.GetManifestRequest
	upsertReq *manifestv1.UpsertManifestRequest
	clearReq  *manifestv1.ClearStaleRequest
}

func (f *fakeService) ListManifests(_ context.Context, _ *connect.Request[manifestv1.ListManifestsRequest]) (*connect.Response[manifestv1.ListManifestsResponse], error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp == nil {
		f.listResp = &manifestv1.ListManifestsResponse{}
	}
	return connect.NewResponse(f.listResp), nil
}

func (f *fakeService) GetManifest(_ context.Context, req *connect.Request[manifestv1.GetManifestRequest]) (*connect.Response[manifestv1.GetManifestResponse], error) {
	f.mu.Lock()
	f.getReq = req.Msg
	f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	return connect.NewResponse(f.getResp), nil
}

func (f *fakeService) UpsertManifest(_ context.Context, req *connect.Request[manifestv1.UpsertManifestRequest]) (*connect.Response[manifestv1.UpsertManifestResponse], error) {
	f.mu.Lock()
	f.upsertReq = req.Msg
	f.mu.Unlock()
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	return connect.NewResponse(f.upsertResp), nil
}

func (f *fakeService) ClearStale(_ context.Context, req *connect.Request[manifestv1.ClearStaleRequest]) (*connect.Response[manifestv1.ClearStaleResponse], error) {
	f.mu.Lock()
	f.clearReq = req.Msg
	f.mu.Unlock()
	if f.clearErr != nil {
		return nil, f.clearErr
	}
	if f.clearResp == nil {
		f.clearResp = &manifestv1.ClearStaleResponse{}
	}
	return connect.NewResponse(f.clearResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := manifestconnect.NewManifestServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sample(skill, golden string) *manifestv1.Manifest {
	return &manifestv1.Manifest{
		SkillId:           skill,
		GoldenSlug:        golden,
		AllowedPaths:      []string{"src/**"},
		WildcardAllowed:   false,
		ConvergenceTarget: manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF,
		UpdatedAt:         timestamppb.New(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)),
	}
}

func TestList_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &manifestv1.ListManifestsResponse{
		Manifests: []*manifestv1.Manifest{sample("a", "g1"), sample("b", "g2")},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 2 manifest(s).")
	require.Contains(t, out.String(), "a/g1")
}

func TestGet_PassesPositionals(t *testing.T) {
	svc := &fakeService{getResp: &manifestv1.GetManifestResponse{Manifest: sample("plan-skill-discovery", "reference-react-vite")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "skill_id", Required: true},
			{Name: "golden_slug", Required: true},
		},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{
		"skill_id":    "plan-skill-discovery",
		"golden_slug": "reference-react-vite",
	}})
	require.NoError(t, h.get(ctx))
	require.Equal(t, "plan-skill-discovery", svc.getReq.SkillId)
	require.Equal(t, "reference-react-vite", svc.getReq.GoldenSlug)
	require.Contains(t, out.String(), "Fetched manifest")
}

func TestGet_NotFoundSurfaced(t *testing.T) {
	svc := &fakeService{getErr: connect.NewError(connect.CodeNotFound, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "skill_id", Required: true},
			{Name: "golden_slug", Required: true},
		},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{
		"skill_id":    "missing",
		"golden_slug": "missing",
	}})
	err := h.get(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
}

func TestUpsert_BuildsRequestFromFlags(t *testing.T) {
	svc := &fakeService{upsertResp: &manifestv1.UpsertManifestResponse{Manifest: sample("plan-skill-discovery", "reference-react-vite")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{
			{Name: "skill"}, {Name: "golden"},
			{Name: "allow"}, {Name: "wildcard-allowed", Bool: true},
			{Name: "convergence"}, {Name: "template-version"}, {Name: "skill-version"},
		},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"skill":            "plan-skill-discovery",
			"golden":           "reference-react-vite",
			"allow":            "src/**, docs/**",
			"wildcard-allowed": "true",
			"convergence":      "empty-diff",
			"template-version": "1.0.0",
			"skill-version":    "2026-05-01",
		},
	})
	require.NoError(t, h.upsert(ctx))
	require.NotNil(t, svc.upsertReq)
	require.Equal(t, "plan-skill-discovery", svc.upsertReq.Manifest.SkillId)
	require.True(t, svc.upsertReq.Manifest.WildcardAllowed)
	require.Equal(t, []string{"src/**", "docs/**"}, svc.upsertReq.Manifest.AllowedPaths)
	require.Equal(t, manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF, svc.upsertReq.Manifest.ConvergenceTarget)
	require.Equal(t, "1.0.0", svc.upsertReq.Manifest.TemplateVersionPinned)
}

func TestClearStale_PassesArgs(t *testing.T) {
	svc := &fakeService{clearResp: &manifestv1.ClearStaleResponse{
		ClearedAt: timestamppb.New(time.Date(2026, 5, 18, 13, 0, 0, 0, time.UTC)),
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "skill"}, {Name: "golden"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"skill":  "plan-skill-discovery",
		"golden": "reference-react-vite",
	}})
	require.NoError(t, h.clearStale(ctx))
	require.Equal(t, "plan-skill-discovery", svc.clearReq.SkillId)
	require.Equal(t, "reference-react-vite", svc.clearReq.GoldenSlug)
	require.Contains(t, out.String(), "Cleared staleness")
}

package staleness

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	stalenessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness"
	stalenessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness/staleness_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "development-toolchain-validator/cli/internal/testutil"
)

type fakeService struct {
	resp *stalenessv1.ListStaleResponse
	err  error
}

func (f *fakeService) ListStale(context.Context, *connect.Request[stalenessv1.ListStaleRequest]) (*connect.Response[stalenessv1.ListStaleResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.resp == nil {
		f.resp = &stalenessv1.ListStaleResponse{}
	}
	return connect.NewResponse(f.resp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := stalenessconnect.NewStalenessServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestList_RendersStaleEntries(t *testing.T) {
	svc := &fakeService{resp: &stalenessv1.ListStaleResponse{Entries: []*stalenessv1.StaleEntry{
		{
			SkillId: "plan-skill", GoldenSlug: "reference-react-vite", Kind: stalenessv1.StaleKind_STALE_KIND_TEMPLATE_DRIFT,
			ManifestTemplateVersionPinned: "1.0.0", GoldenTemplateVersionCurrent: "1.1.0",
			ManifestSkillVersionPinned: "v1", SkillVersionCurrent: "v1",
		},
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 1 stale tuple(s).")
	require.Contains(t, out.String(), "template_drift")
	require.Contains(t, out.String(), "plan-skill/reference-react-vite")
}

func TestList_EmptyReturnsZero(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 0 stale tuple(s).")
}

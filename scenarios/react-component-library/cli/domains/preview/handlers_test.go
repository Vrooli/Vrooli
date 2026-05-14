package preview

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "react-component-library/cli/internal/testutil"
)

type previewService struct {
	mu         sync.Mutex
	bundleResp *previewv1.GetPreviewBundleResponse
	bundleErr  error
	bundleReqs []*previewv1.GetPreviewBundleRequest
}

func (s *previewService) GetPreviewBundle(_ context.Context, req *connect.Request[previewv1.GetPreviewBundleRequest]) (*connect.Response[previewv1.GetPreviewBundleResponse], error) {
	s.mu.Lock()
	s.bundleReqs = append(s.bundleReqs, req.Msg)
	s.mu.Unlock()
	if s.bundleErr != nil {
		return nil, s.bundleErr
	}
	if s.bundleResp == nil {
		s.bundleResp = &previewv1.GetPreviewBundleResponse{}
	}
	return connect.NewResponse(s.bundleResp), nil
}

func connectAPI(t *testing.T, svc *previewService) http.Handler {
	t.Helper()
	path, handler := previewconnect.NewPreviewServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestPreviewBundle_PrintsJS(t *testing.T) {
	svc := &previewService{bundleResp: &previewv1.GetPreviewBundleResponse{
		Js:         "export const X = () => null;",
		SourcePath: "components/X.tsx",
		Sha256:     "cafef00d",
		Warnings:   []string{"unused import"},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "cmp-x"},
	})

	require.NoError(t, h.bundle(ctx))
	require.Len(t, svc.bundleReqs, 1)
	require.Equal(t, "cmp-x", svc.bundleReqs[0].Id)
	body := out.String()
	require.Contains(t, body, "Bundled components/X.tsx")
	require.Contains(t, body, "sha256=cafef00d")
	require.Contains(t, body, "export const X = () => null;")
	require.Contains(t, body, "1 warning(s) reported.")
	require.Contains(t, body, "unused import")
}

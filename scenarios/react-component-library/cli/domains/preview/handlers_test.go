package preview

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

type previewService struct {
	previewconnect.UnimplementedPreviewServiceHandler
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

	msg, err := h.bundleCall(ctx)
	require.NoError(t, err)
	require.NoError(t, cliapp.RenderProtoList(ctx, msg, h.bundleReport(ctx, msg)))
	require.Len(t, svc.bundleReqs, 1)
	require.Equal(t, "cmp-x", svc.bundleReqs[0].Id)
	body := out.String()
	require.Contains(t, body, "Bundled components/X.tsx")
	require.Contains(t, body, "sha256=cafef00d")
	require.Contains(t, body, "export const X = () => null;")
	require.Contains(t, body, "1 warning(s) reported.")
	require.Contains(t, body, "unused import")
}

func TestPreviewCaptureValidatesScopeAndPassesOptions(t *testing.T) {
	core := clitest.NewTestApp(t, http.NewServeMux())
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "asset"}, {Name: "version"}, {Name: "all", Bool: true}, {Name: "refresh", Bool: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{})
	err := (&handlers{}).capture(ctx)
	require.ErrorContains(t, err, "requires --asset <library-id> or --all")

	bin := t.TempDir()
	python := filepath.Join(bin, "python3")
	require.NoError(t, os.WriteFile(python, []byte("#!/bin/sh\nprintf 'captured one\\ncaptured two\\n'\n"), 0o700))
	t.Setenv("PATH", bin)
	ctx, output := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"asset": " button ", "version": " 1.2.0 ", "refresh": "true",
	}})
	require.NoError(t, (&handlers{}).capture(ctx))
	body := output.String()
	require.Contains(t, body, "Preview capture completed.")
	require.Contains(t, body, "captured one")
	require.Contains(t, body, "captured two")
}

func TestPreviewCaptureReturnsCommandOutputOnFailure(t *testing.T) {
	core := clitest.NewTestApp(t, http.NewServeMux())
	bin := t.TempDir()
	python := filepath.Join(bin, "python3")
	require.NoError(t, os.WriteFile(python, []byte("#!/bin/sh\necho capture-failed >&2\nexit 9\n"), 0o700))
	t.Setenv("PATH", bin)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "asset"}, {Name: "version"}, {Name: "all", Bool: true}, {Name: "refresh", Bool: true}}}, cliapptest.TestRunContextOptions{Flags: map[string]string{"all": "true"}})
	err := (&handlers{}).capture(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "capture previews")
	require.Contains(t, err.Error(), "capture-failed")
}

func TestBundleReportWithoutWarningsOnlyRendersBundleBody(t *testing.T) {
	report := (&handlers{}).bundleReport(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{}), &previewv1.GetPreviewBundleResponse{SourcePath: "src/Button.tsx", Sha256: "abc", Js: "export {}"})
	require.Equal(t, []string{"Bundled src/Button.tsx (sha256=abc, 9 byte(s))."}, report.Summary)
	require.Equal(t, []string{"export {}"}, report.Results)
	require.False(t, strings.Contains(strings.Join(report.Summary, " "), "warning"))
}

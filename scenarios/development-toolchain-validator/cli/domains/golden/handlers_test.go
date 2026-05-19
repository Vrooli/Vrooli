package golden

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	goldenv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden"
	goldenconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden/golden_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "development-toolchain-validator/cli/internal/testutil"
)

type fakeService struct {
	mu             sync.Mutex
	listResp       *goldenv1.ListGoldensResponse
	getResp        *goldenv1.GetGoldenResponse
	registerResp   *goldenv1.RegisterGoldenResponse
	updateResp     *goldenv1.UpdateGoldenResponse
	deleteResp     *goldenv1.DeleteGoldenResponse
	regenerateResp *goldenv1.RegenerateGoldenResponse

	listErr       error
	getErr        error
	registerErr   error
	updateErr     error
	deleteErr     error
	regenerateErr error

	getSlugs        []string
	registerInputs  []*goldenv1.RegisterGoldenRequest
	updateInputs    []*goldenv1.UpdateGoldenRequest
	deleteSlugs     []string
	regenerateSlugs []string
}

func (s *fakeService) ListGoldens(context.Context, *connect.Request[goldenv1.ListGoldensRequest]) (*connect.Response[goldenv1.ListGoldensResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &goldenv1.ListGoldensResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *fakeService) GetGolden(_ context.Context, req *connect.Request[goldenv1.GetGoldenRequest]) (*connect.Response[goldenv1.GetGoldenResponse], error) {
	s.mu.Lock()
	s.getSlugs = append(s.getSlugs, req.Msg.Slug)
	s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp == nil {
		s.getResp = &goldenv1.GetGoldenResponse{}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *fakeService) RegisterGolden(_ context.Context, req *connect.Request[goldenv1.RegisterGoldenRequest]) (*connect.Response[goldenv1.RegisterGoldenResponse], error) {
	s.mu.Lock()
	s.registerInputs = append(s.registerInputs, req.Msg)
	s.mu.Unlock()
	if s.registerErr != nil {
		return nil, s.registerErr
	}
	if s.registerResp == nil {
		s.registerResp = &goldenv1.RegisterGoldenResponse{}
	}
	return connect.NewResponse(s.registerResp), nil
}

func (s *fakeService) UpdateGolden(_ context.Context, req *connect.Request[goldenv1.UpdateGoldenRequest]) (*connect.Response[goldenv1.UpdateGoldenResponse], error) {
	s.mu.Lock()
	s.updateInputs = append(s.updateInputs, req.Msg)
	s.mu.Unlock()
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	if s.updateResp == nil {
		s.updateResp = &goldenv1.UpdateGoldenResponse{}
	}
	return connect.NewResponse(s.updateResp), nil
}

func (s *fakeService) DeleteGolden(_ context.Context, req *connect.Request[goldenv1.DeleteGoldenRequest]) (*connect.Response[goldenv1.DeleteGoldenResponse], error) {
	s.mu.Lock()
	s.deleteSlugs = append(s.deleteSlugs, req.Msg.Slug)
	s.mu.Unlock()
	if s.deleteErr != nil {
		return nil, s.deleteErr
	}
	if s.deleteResp == nil {
		s.deleteResp = &goldenv1.DeleteGoldenResponse{}
	}
	return connect.NewResponse(s.deleteResp), nil
}

func (s *fakeService) RegenerateGolden(_ context.Context, req *connect.Request[goldenv1.RegenerateGoldenRequest]) (*connect.Response[goldenv1.RegenerateGoldenResponse], error) {
	s.mu.Lock()
	s.regenerateSlugs = append(s.regenerateSlugs, req.Msg.Slug)
	s.mu.Unlock()
	if s.regenerateErr != nil {
		return nil, s.regenerateErr
	}
	if s.regenerateResp == nil {
		s.regenerateResp = &goldenv1.RegenerateGoldenResponse{}
	}
	return connect.NewResponse(s.regenerateResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := goldenconnect.NewGoldenServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleProto(slug string) *goldenv1.Golden {
	ts := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return &goldenv1.Golden{
		Id:                    "id-" + slug,
		Slug:                  slug,
		TemplateId:            "react-vite",
		TemplateVersionPinned: "1.0.1",
		Path:                  "scenarios/" + slug,
		CreatedAt:             ts,
		LastRegeneratedAt:     ts,
	}
}

func TestGoldensList_RendersResults(t *testing.T) {
	svc := &fakeService{listResp: &goldenv1.ListGoldensResponse{
		Goldens: []*goldenv1.Golden{sampleProto("alpha"), sampleProto("bravo")},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 2 golden(s).")
	require.Contains(t, out.String(), "alpha")
	require.Contains(t, out.String(), "bravo")
}

func TestGoldensList_SurfacesConnectErrors(t *testing.T) {
	svc := &fakeService{listErr: connect.NewError(connect.CodeInternal, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	err := h.list(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal")
}

func TestGoldensGet_PassesSlug(t *testing.T) {
	svc := &fakeService{getResp: &goldenv1.GetGoldenResponse{Golden: sampleProto("alpha")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"slug": "alpha"}})
	require.NoError(t, h.get(ctx))
	require.Equal(t, []string{"alpha"}, svc.getSlugs)
	require.Contains(t, out.String(), "Fetched golden alpha.")
}

func TestGoldensRegister_SendsAllFlags(t *testing.T) {
	svc := &fakeService{registerResp: &goldenv1.RegisterGoldenResponse{Golden: sampleProto("alpha")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "slug"}, {Name: "template"}, {Name: "version"}, {Name: "path"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"slug": "alpha", "template": "react-vite", "version": "1.0.1", "path": "scenarios/alpha"},
	})
	require.NoError(t, h.register(ctx))
	require.Len(t, svc.registerInputs, 1)
	require.Equal(t, "alpha", svc.registerInputs[0].Slug)
	require.Equal(t, "react-vite", svc.registerInputs[0].TemplateId)
	require.Equal(t, "1.0.1", svc.registerInputs[0].TemplateVersion)
	require.Equal(t, "scenarios/alpha", svc.registerInputs[0].Path)
	require.Contains(t, out.String(), "Registered golden alpha.")
}

func TestGoldensRegister_SurfacesAlreadyExists(t *testing.T) {
	svc := &fakeService{registerErr: connect.NewError(connect.CodeAlreadyExists, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "slug"}, {Name: "template"}, {Name: "version"}, {Name: "path"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"slug": "alpha", "template": "x", "version": "1", "path": "p"},
	})
	err := h.register(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already_exists")
}

func TestGoldensUpdate_PassesPatch(t *testing.T) {
	svc := &fakeService{updateResp: &goldenv1.UpdateGoldenResponse{Golden: sampleProto("alpha")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "path"}, {Name: "version"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "alpha"},
		Flags:       map[string]string{"version": "1.0.2"},
	})
	require.NoError(t, h.update(ctx))
	require.Equal(t, "alpha", svc.updateInputs[0].Slug)
	require.Equal(t, "1.0.2", svc.updateInputs[0].TemplateVersion)
}

func TestGoldensDelete_RefusesWithoutYes(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "yes", Bool: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"slug": "alpha"}})
	err := h.delete(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--yes")
	require.Empty(t, svc.deleteSlugs)
}

func TestGoldensDelete_CallsServiceWithYes(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "yes", Bool: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "alpha"},
		BoolFlags:   map[string]bool{"yes": true},
	})
	require.NoError(t, h.delete(ctx))
	require.Equal(t, []string{"alpha"}, svc.deleteSlugs)
	require.Contains(t, out.String(), "Deleted golden alpha.")
}

func TestGoldensRegenerate_RefusesWithoutYes(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "yes", Bool: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"slug": "alpha"}})
	err := h.regenerate(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--yes")
	require.Empty(t, svc.regenerateSlugs)
}

func TestGoldensRegenerate_HappyPath(t *testing.T) {
	svc := &fakeService{regenerateResp: &goldenv1.RegenerateGoldenResponse{Golden: sampleProto("alpha")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "slug", Required: true}},
		Flags:       []cliapp.Flag{{Name: "yes", Bool: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"slug": "alpha"},
		BoolFlags:   map[string]bool{"yes": true},
	})
	require.NoError(t, h.regenerate(ctx))
	require.Equal(t, []string{"alpha"}, svc.regenerateSlugs)
	require.Contains(t, out.String(), "Regenerated golden alpha")
}

func TestRegister_Wiring(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	require.Equal(t, "goldens", group.Name)
	require.True(t, group.NeedsAPI)
	names := make([]string, 0, len(group.Subcommands))
	for _, sc := range group.Subcommands {
		names = append(names, sc.Name)
	}
	require.ElementsMatch(t,
		[]string{"list", "get", "register", "update", "delete", "regenerate"},
		names)
	for _, sc := range group.Subcommands {
		require.NotNil(t, sc.RunCtx, "subcommand %s should use RunCtx", sc.Name)
	}
}

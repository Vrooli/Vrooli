package golden_test

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"development-toolchain-validator/handlers/golden"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	goldenv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden"
	goldenconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden/golden_v1connect"

	internalgolden "development-toolchain-validator/internal/golden"
)

// fakeService is a hand-rolled Service stub. Inlined here so the
// handler tests do not depend on an exported mocks package — the
// handler boundary is small enough that a per-file fake stays clearer.
type fakeService struct {
	ListOut []internalgolden.Golden
	ListErr error

	GetOut internalgolden.Golden
	GetErr error

	RegisterInputs []internalgolden.RegisterInput
	RegisterOut    *internalgolden.Golden
	RegisterErr    error

	UpdateInputs []internalgolden.UpdateInput
	UpdateOut    *internalgolden.Golden
	UpdateErr    error

	DeleteSlugs []string
	DeleteErr   error

	RegenerateSlugs []string
	RegenerateOut   *internalgolden.Golden
	RegenerateErr   error
}

func (f *fakeService) List(context.Context) ([]internalgolden.Golden, error) {
	return f.ListOut, f.ListErr
}

func (f *fakeService) Get(_ context.Context, slug string) (internalgolden.Golden, error) {
	if f.GetErr != nil {
		return internalgolden.Golden{}, f.GetErr
	}
	g := f.GetOut
	if g.Slug == "" {
		g.Slug = slug
	}
	return g, nil
}

func (f *fakeService) Register(_ context.Context, in internalgolden.RegisterInput) (internalgolden.Golden, error) {
	f.RegisterInputs = append(f.RegisterInputs, in)
	if f.RegisterErr != nil {
		return internalgolden.Golden{}, f.RegisterErr
	}
	if f.RegisterOut != nil {
		return *f.RegisterOut, nil
	}
	return internalgolden.Golden{Slug: in.Slug, TemplateID: in.TemplateID, TemplateVersionPinned: in.TemplateVersion, Path: in.Path}, nil
}

func (f *fakeService) Update(_ context.Context, in internalgolden.UpdateInput) (internalgolden.Golden, error) {
	f.UpdateInputs = append(f.UpdateInputs, in)
	if f.UpdateErr != nil {
		return internalgolden.Golden{}, f.UpdateErr
	}
	if f.UpdateOut != nil {
		return *f.UpdateOut, nil
	}
	return internalgolden.Golden{Slug: in.Slug}, nil
}

func (f *fakeService) Delete(_ context.Context, slug string) error {
	f.DeleteSlugs = append(f.DeleteSlugs, slug)
	return f.DeleteErr
}

func (f *fakeService) Regenerate(_ context.Context, slug string) (internalgolden.Golden, error) {
	f.RegenerateSlugs = append(f.RegenerateSlugs, slug)
	if f.RegenerateErr != nil {
		return internalgolden.Golden{}, f.RegenerateErr
	}
	if f.RegenerateOut != nil {
		return *f.RegenerateOut, nil
	}
	return internalgolden.Golden{Slug: slug}, nil
}

func newClient(t *testing.T, svc internalgolden.Service, logger *log.Logger) goldenconnect.GoldenServiceClient {
	t.Helper()
	if logger == nil {
		logger, _ = connectxtest.NewLogger(t)
	}
	path, handler := goldenconnect.NewGoldenServiceHandler(golden.NewConnectHandler(golden.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return goldenconnect.NewGoldenServiceClient(server.Client(), server.URL)
}

func TestConnectHandler_ListReturnsItems(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{ListOut: []internalgolden.Golden{
		{ID: "1", Slug: "alpha", TemplateID: "react-vite", TemplateVersionPinned: "1.0.1", Path: "p", CreatedAt: now, LastRegeneratedAt: now},
		{ID: "2", Slug: "bravo", TemplateID: "react-vite", TemplateVersionPinned: "1.0.1", Path: "q", CreatedAt: now, LastRegeneratedAt: now},
	}}, nil)
	resp, err := client.ListGoldens(context.Background(), connect.NewRequest(&goldenv1.ListGoldensRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Goldens, 2)
	require.Equal(t, "alpha", resp.Msg.Goldens[0].Slug)
}

func TestConnectHandler_RegisterSuccess(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	fake := &fakeService{RegisterOut: &internalgolden.Golden{ID: "abc", Slug: "g", TemplateID: "react-vite", TemplateVersionPinned: "1.0.1", Path: "p", CreatedAt: now, LastRegeneratedAt: now}}
	client := newClient(t, fake, nil)
	resp, err := client.RegisterGolden(context.Background(), connect.NewRequest(&goldenv1.RegisterGoldenRequest{
		Slug: "g", TemplateId: "react-vite", TemplateVersion: "1.0.1", Path: "p",
	}))
	require.NoError(t, err)
	require.Equal(t, "abc", resp.Msg.Golden.Id)
	require.Equal(t, "g", fake.RegisterInputs[0].Slug)
}

func TestConnectHandler_RegisterInvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{RegisterErr: internalgolden.ErrInvalidGolden{Field: "slug", Reason: "required"}}, nil)
	_, err := client.RegisterGolden(context.Background(), connect.NewRequest(&goldenv1.RegisterGoldenRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnectHandler_RegisterAlreadyExists(t *testing.T) {
	client := newClient(t, &fakeService{RegisterErr: internalgolden.ErrGoldenAlreadyExists{Slug: "g"}}, nil)
	_, err := client.RegisterGolden(context.Background(), connect.NewRequest(&goldenv1.RegisterGoldenRequest{Slug: "g", TemplateId: "x", TemplateVersion: "1", Path: "p"}))
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestConnectHandler_GetNotFound(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: internalgolden.ErrGoldenNotFound{Slug: "ghost"}}, nil)
	_, err := client.GetGolden(context.Background(), connect.NewRequest(&goldenv1.GetGoldenRequest{Slug: "ghost"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnectHandler_UpdatePassThrough(t *testing.T) {
	fake := &fakeService{UpdateOut: &internalgolden.Golden{Slug: "g", TemplateVersionPinned: "1.0.2"}}
	client := newClient(t, fake, nil)
	resp, err := client.UpdateGolden(context.Background(), connect.NewRequest(&goldenv1.UpdateGoldenRequest{Slug: "g", TemplateVersion: "1.0.2"}))
	require.NoError(t, err)
	require.Equal(t, "1.0.2", resp.Msg.Golden.TemplateVersionPinned)
	require.Equal(t, "g", fake.UpdateInputs[0].Slug)
}

func TestConnectHandler_DeleteSucceeds(t *testing.T) {
	fake := &fakeService{}
	client := newClient(t, fake, nil)
	_, err := client.DeleteGolden(context.Background(), connect.NewRequest(&goldenv1.DeleteGoldenRequest{Slug: "g"}))
	require.NoError(t, err)
	require.Equal(t, []string{"g"}, fake.DeleteSlugs)
}

func TestConnectHandler_RegenerateSurfacesInternalOnRunnerFailure(t *testing.T) {
	logger, logBuf := connectxtest.NewLogger(t)
	client := newClient(t, &fakeService{RegenerateErr: internalgolden.ErrRegenerateFailed{Slug: "g", Wrapped: errors.New("boom")}}, logger)
	_, err := client.RegenerateGolden(context.Background(), connect.NewRequest(&goldenv1.RegenerateGoldenRequest{Slug: "g"}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.Contains(t, logBuf.String(), "boom")
}

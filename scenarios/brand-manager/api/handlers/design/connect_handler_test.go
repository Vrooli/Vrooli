package design_test

import (
	"context"
	"strings"
	"testing"

	"brand-manager/handlers/design"
	internaldesign "brand-manager/internal/design"
	mocks "brand-manager/internal/design/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	designv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design"
	designconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design/design_v1connect"
)

// newClient wires the real internal design service over an in-memory brand store
// behind the generated Connect handler, exercising handler + adapter + service +
// renderer together.
func newClient(t *testing.T, store *mocks.FakeBrandStore) designconnect.DesignServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	svc := internaldesign.NewService(store, logger)
	path, handler := designconnect.NewDesignServiceHandler(design.NewConnectHandler(design.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return designconnect.NewDesignServiceClient(server.Client(), server.URL)
}

func seededStore(t *testing.T) *mocks.FakeBrandStore {
	t.Helper()
	store := &mocks.FakeBrandStore{}
	store.Seed(internaldesign.Brand{
		ID:      "brand-1",
		Name:    "Acme",
		Version: 2,
		Colors:  internaldesign.Colors{Primary: "#112233"},
	})
	return store
}

func TestConnect_GenerateReturnsMarkdown(t *testing.T) {
	client := newClient(t, seededStore(t))

	resp, err := client.GenerateDesignLanguage(context.Background(), connect.NewRequest(&designv1.GenerateDesignLanguageRequest{
		BrandId: "brand-1",
	}))
	require.NoError(t, err)
	require.Equal(t, "brand-1", resp.Msg.BrandId)
	require.True(t, strings.Contains(resp.Msg.Markdown, "# Acme DESIGN.md"))
	require.True(t, strings.Contains(resp.Msg.Markdown, "#112233"))
}

func TestConnect_GenerateMissingArgIsInvalidArgument(t *testing.T) {
	client := newClient(t, seededStore(t))

	_, err := client.GenerateDesignLanguage(context.Background(), connect.NewRequest(&designv1.GenerateDesignLanguageRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_GenerateUnknownBrandIsNotFound(t *testing.T) {
	client := newClient(t, &mocks.FakeBrandStore{})

	_, err := client.GenerateDesignLanguage(context.Background(), connect.NewRequest(&designv1.GenerateDesignLanguageRequest{
		BrandId: "ghost",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

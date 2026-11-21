package models_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	modelsH "audio-tools/handlers/models"
	"audio-tools/internal/modelcatalog"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models/models_v1connect"
)

// newServer mounts the real module on a mux-backed httptest server. The unit
// tests call the handler directly; this proves the Connect path, service name,
// and endpoint descriptors the CLI and UI actually use.
func newServer(t *testing.T) modelsconnect.ModelRegistryServiceClient {
	t.Helper()
	mod := modelsH.Module(modelsH.Deps{Catalog: modelcatalog.Default()})
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return modelsconnect.NewModelRegistryServiceClient(http.DefaultClient, srv.URL)
}

func TestModule_ListAndResolveOverConnect(t *testing.T) {
	client := newServer(t)
	ctx := context.Background()

	list, err := client.ListModels(ctx, connect.NewRequest(&modelsv1.ListModelsRequest{Capability: "stt", OnlyReady: true}))
	require.NoError(t, err)
	require.NotEmpty(t, list.Msg.GetModels())
	for _, record := range list.Msg.GetModels() {
		require.Equal(t, "stt", record.GetCapability())
		require.Equal(t, modelsv1.Readiness_READINESS_AVAILABLE, record.GetReadiness())
	}

	resolved, err := client.ResolveModel(ctx, connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: "stt",
		Commercial: true,
	}))
	require.NoError(t, err)
	require.Equal(t, "whisper-small", resolved.Msg.GetModel().GetId())
	require.Equal(t, modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED, resolved.Msg.GetModel().GetCommercialUse())
	require.Contains(t, resolved.Msg.GetExplanation(), "commercial-only")
}

func TestModule_InvalidCapabilityIsInvalidArgument(t *testing.T) {
	client := newServer(t)
	_, err := client.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{Capability: "nope"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

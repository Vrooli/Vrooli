package provider_lifecycle_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"audio-tools/handlers/provider_lifecycle"
	"audio-tools/internal/capabilities"
	capmocks "audio-tools/internal/capabilities/mocks"
	"audio-tools/internal/testutil/mocks"
	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
	plconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle/provider_lifecycle_v1connect"
)

func TestModuleMountsConnectServiceForLocalProviderRead(t *testing.T) {
	registry := capabilities.NewRegistry([]capabilities.Def{{
		ID: "whisper-stt", DependencyKind: capabilities.DependencyResource,
		DependencySlug: "whisper", Features: []string{"voice-input"},
	}}, nil, time.Minute)
	mod := provider_lifecycle.Module(provider_lifecycle.Deps{
		Registry: registry, Controller: &capmocks.FakeController{},
		Logger: mocks.NewFakeLogger(), Clock: scheduletest.New(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)),
	})
	require.Equal(t, "provider_lifecycle", mod.Name)
	require.Len(t, mod.Endpoints, 6)

	router := mux.NewRouter()
	mod.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := plconnect.NewProviderLifecycleServiceClient(http.DefaultClient, server.URL)

	resp, err := client.ListLocalProviders(context.Background(), connect.NewRequest(&plv1.ListLocalProvidersRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetProviders())
	require.Equal(t, "whisper-stt", resp.Msg.GetProviders()[0].GetProviderId())
}

package searchregistry

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestDescriptorPreservesSearchTransportAndTuning(t *testing.T) {
	descriptor, err := Descriptor(aisearch.ProviderConfig{
		ProviderID: "signal-inbox.signals", ProviderGroup: "signal-inbox", Bucket: "BUCKET_KNOW", Type: "signal", Description: "immutable captures", Scope: "SCOPE_PROJECT",
		Endpoint:            []byte(`{"http_json":{"scenario_id":"signal-inbox","path":"/search","method":"HTTP_METHOD_POST"}}`),
		StatusEndpoint:      []byte(`{"http_json":{"scenario_id":"signal-inbox","path":"/health","method":"HTTP_METHOD_GET"}}`),
		IndexTimestampField: "metrics.last_indexed_at",
		Tuning:              aisearch.TuningConfig{Engine: "dense", EmbedModel: "ai-gateway:embedding.default", EmbedTaskPrefix: true, RerankShortlist: 50},
	})
	require.NoError(t, err)
	require.Equal(t, "signal-inbox.signals", descriptor.GetProviderId())
	require.Equal(t, "/search", descriptor.GetEndpoint().GetHttpJson().GetPath())
	require.Equal(t, "metrics.last_indexed_at", descriptor.GetIndexTimestampField())
	require.Equal(t, "ai-gateway:embedding.default", descriptor.GetTuning().GetEmbedModel())
	require.True(t, descriptor.GetTuning().GetEmbedTaskPrefix())
}

type fakeRegistryClient struct{ calls int }

func (f *fakeRegistryClient) RegisterProvider(_ context.Context, req *connect.Request[registryv1.RegisterProviderRequest]) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	f.calls++
	if req.Msg.GetDescriptor_().GetProviderId() == "" {
		return nil, errors.New("missing provider")
	}
	return connect.NewResponse(&registryv1.RegisterProviderResponse{Created: true}), nil
}

func TestRegisterOneResolvesAndUpserts(t *testing.T) {
	client := &fakeRegistryClient{}
	err := registerOne(context.Background(), &registryv1.ProviderDescriptor{ProviderId: "signal-inbox.signals"}, func(context.Context) (string, error) {
		return "http://search-hub.test", nil
	}, func(string) RegistryClient { return client })
	require.NoError(t, err)
	require.Equal(t, 1, client.calls)
}

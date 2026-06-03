package registry_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"

	handler "search-hub/handlers/registry"
	internalregistry "search-hub/internal/registry"
)

// fakeStore is a hand-written Store fake: tests arrange via struct fields and
// assert against the recorded call inputs. Keeps the handler test off a real
// DB (that path is covered by internal/registry/store_test.go).
type fakeStore struct {
	upsertOut  bool
	upsertErr  error
	listOut    []*registryv1.ProviderDescriptor
	listErr    error
	deleteOut  bool
	deleteErr  error
	lastUpsert *registryv1.ProviderDescriptor
	lastFilter internalregistry.ListFilter
	lastDelete string
}

func (f *fakeStore) Upsert(_ context.Context, d *registryv1.ProviderDescriptor) (bool, error) {
	f.lastUpsert = d
	return f.upsertOut, f.upsertErr
}

func (f *fakeStore) List(_ context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	f.lastFilter = filter
	return f.listOut, f.listErr
}

func (f *fakeStore) Get(_ context.Context, id string) (*registryv1.ProviderDescriptor, error) {
	return nil, internalregistry.ErrProviderNotFound{ProviderID: id}
}

func (f *fakeStore) Delete(_ context.Context, id string) (bool, error) {
	f.lastDelete = id
	return f.deleteOut, f.deleteErr
}

func newClient(t *testing.T, store internalregistry.Store) registryconnect.RegistryServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := registryconnect.NewRegistryServiceHandler(handler.NewConnectHandler(handler.Deps{Store: store, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return registryconnect.NewRegistryServiceClient(server.Client(), server.URL)
}

func validDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "cli-health.commands",
		ProviderGroup: "cli-health",
		Bucket:        registryv1.Bucket_BUCKET_DO,
		Type:          "command",
		Description:   "CLI commands.",
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId: "cli-health", Path: "/x", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
		}}},
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results", IdField: "name", TitleField: "name", ScoreField: "score",
			ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
}

func TestRegisterProviderCreated(t *testing.T) {
	store := &fakeStore{upsertOut: true}
	client := newClient(t, store)

	resp, err := client.RegisterProvider(context.Background(), connect.NewRequest(&registryv1.RegisterProviderRequest{
		Descriptor_: validDescriptor(),
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetCreated())
	require.Equal(t, "cli-health.commands", resp.Msg.GetDescriptor_().GetProviderId())
	require.Equal(t, "cli-health.commands", store.lastUpsert.GetProviderId())
}

func TestRegisterProviderInvalidArgument(t *testing.T) {
	store := &fakeStore{upsertErr: internalregistry.ErrInvalidDescriptor{Field: "bucket", Reason: "required"}}
	client := newClient(t, store)

	_, err := client.RegisterProvider(context.Background(), connect.NewRequest(&registryv1.RegisterProviderRequest{
		Descriptor_: validDescriptor(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestRegisterProviderInternalErrorIsOpaque(t *testing.T) {
	store := &fakeStore{upsertErr: errors.New("disk on fire: /var/lib/secret")}
	client := newClient(t, store)

	_, err := client.RegisterProvider(context.Background(), connect.NewRequest(&registryv1.RegisterProviderRequest{
		Descriptor_: validDescriptor(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret", "internal details must not leak to the client")
}

func TestListProvidersPassesFilters(t *testing.T) {
	store := &fakeStore{listOut: []*registryv1.ProviderDescriptor{validDescriptor()}}
	client := newClient(t, store)

	resp, err := client.ListProviders(context.Background(), connect.NewRequest(&registryv1.ListProvidersRequest{
		Bucket: registryv1.Bucket_BUCKET_DO,
		Type:   "command",
		State:  registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetProviders(), 1)
	require.Equal(t, int32(registryv1.Bucket_BUCKET_DO), store.lastFilter.Bucket)
	require.Equal(t, "command", store.lastFilter.Type)
	require.Equal(t, int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE), store.lastFilter.State)
}

func TestDeregisterProvider(t *testing.T) {
	store := &fakeStore{deleteOut: true}
	client := newClient(t, store)

	resp, err := client.DeregisterProvider(context.Background(), connect.NewRequest(&registryv1.DeregisterProviderRequest{
		ProviderId: "cli-health.commands",
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetRemoved())
	require.Equal(t, "cli-health.commands", store.lastDelete)
}

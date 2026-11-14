package registry_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"

	handler "search-hub/handlers/registry"
	internalregistry "search-hub/internal/registry"
)

// fakeStore is a hand-written Store fake: tests arrange via struct fields and
// assert against the recorded call inputs. Keeps the handler test off a real
// DB (that path is covered by internal/registry/store_test.go).
type fakeStore struct {
	upsertOut   bool
	upsertToken string
	upsertErr   error
	listOut     []*registryv1.ProviderDescriptor
	listErr     error
	tokenOut    string
	tokenErr    error
	deleteOut   bool
	deleteErr   error
	lastUpsert  *registryv1.ProviderDescriptor
	lastToken   string
	lastFilter  internalregistry.ListFilter
	lastDelete  string
	descriptor  *registryv1.ProviderDescriptor
}

func (f *fakeStore) Upsert(_ context.Context, d *registryv1.ProviderDescriptor, presentedToken string) (bool, string, error) {
	f.lastUpsert = d
	f.lastToken = presentedToken
	return f.upsertOut, f.upsertToken, f.upsertErr
}

func (f *fakeStore) List(_ context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	f.lastFilter = filter
	return f.listOut, f.listErr
}

func (f *fakeStore) Get(_ context.Context, id string) (*registryv1.ProviderDescriptor, error) {
	if f.descriptor != nil && f.descriptor.GetProviderId() == id {
		return f.descriptor, nil
	}
	return nil, internalregistry.ErrProviderNotFound{ProviderID: id}
}

func (f *fakeStore) Token(_ context.Context, id string) (string, error) {
	return f.tokenOut, f.tokenErr
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

type fakeProviderControl struct {
	responses []*controlv1.ReindexResponse
	requests  []*controlv1.ReindexRequest
}

func (f *fakeProviderControl) ReindexRequest(_ context.Context, _ *registryv1.ProviderDescriptor, _ string, request *controlv1.ReindexRequest) (*controlv1.ReindexResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected provider control call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func (*fakeProviderControl) ReindexStatus(context.Context, *registryv1.ProviderDescriptor, string, string) (*controlv1.ReindexStatusResponse, error) {
	return &controlv1.ReindexStatusResponse{JobId: "job", State: "succeeded"}, nil
}

func newClientWithControl(t *testing.T, store internalregistry.Store, providerControl handler.ProviderControlClient) registryconnect.RegistryServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := registryconnect.NewRegistryServiceHandler(handler.NewConnectHandler(handler.Deps{Store: store, Logger: logger, Control: providerControl}))
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

func TestEmbeddingMigrationRejectsUnsupportedAndIncompleteRequestsAtAPIBoundary(t *testing.T) {
	client := newClient(t, &fakeStore{})
	tests := []struct {
		name    string
		request *registryv1.ExecuteEmbeddingMigrationRequest
	}{
		{name: "missing provider", request: &registryv1.ExecuteEmbeddingMigrationRequest{Action: "shadow"}},
		{name: "unknown action", request: &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: "provider", Action: "retarget"}},
		{name: "abort is unsupported", request: &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: "provider", Action: "abort"}},
		{name: "shadow recipe is incomplete", request: &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: "provider", Action: "shadow", ShadowCollection: "candidate"}},
		{name: "malformed target", request: &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: "provider", Action: "shadow", ShadowCollection: "../live", EmbeddingModel: "m", EmbeddingRole: "r", EmbeddingDimensions: 3, EmbeddingPolicySchemaVersion: "v1"}},
		{name: "status requires a job", request: &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: "provider", Action: "status"}},
		{name: "status cannot carry mutation targets", request: &registryv1.ExecuteEmbeddingMigrationRequest{ProviderId: "provider", Action: "status", JobId: "job", ShadowCollection: "candidate"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ExecuteEmbeddingMigration(context.Background(), connect.NewRequest(tt.request))
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}

func TestEmbeddingMigrationNegotiatesCapabilitiesBeforeMutationAndNormalizesPromote(t *testing.T) {
	store := &fakeStore{descriptor: validDescriptor(), tokenOut: "provider-token"}
	provider := &fakeProviderControl{responses: []*controlv1.ReindexResponse{
		{MigrationProtocolVersion: aisearch.ReindexMigrationProtocolVersion, SupportedMigrationActions: []string{"shadow", "cutover", "rollback"}},
		{JobId: "job-cutover"},
	}}
	client := newClientWithControl(t, store, provider)
	resp, err := client.ExecuteEmbeddingMigration(context.Background(), connect.NewRequest(&registryv1.ExecuteEmbeddingMigrationRequest{
		ProviderId: "cli-health.commands", Action: "promote", ShadowCollection: "candidate",
	}))
	require.NoError(t, err)
	require.Equal(t, "capabilities", provider.requests[0].GetAction(), "the read-only capability probe must precede mutation")
	require.Equal(t, "cutover", provider.requests[1].GetAction(), "promote alias must be normalized before provider dispatch")
	require.Equal(t, "cutover", resp.Msg.GetState())
}

func TestEmbeddingMigrationPassesEveryShadowFieldToProvider(t *testing.T) {
	store := &fakeStore{descriptor: validDescriptor(), tokenOut: "provider-token"}
	provider := &fakeProviderControl{responses: []*controlv1.ReindexResponse{
		{MigrationProtocolVersion: aisearch.ReindexMigrationProtocolVersion, SupportedMigrationActions: []string{"shadow", "cutover", "rollback"}},
		{JobId: "shadow-job", PlannedUpserts: 9, PlannedDeletes: 2},
	}}
	client := newClientWithControl(t, store, provider)
	resp, err := client.ExecuteEmbeddingMigration(context.Background(), connect.NewRequest(&registryv1.ExecuteEmbeddingMigrationRequest{
		ProviderId: "cli-health.commands", Action: "shadow", ShadowCollection: "candidate-recipe-b",
		EmbeddingModel: "nomic-embed-text:latest", EmbeddingRole: "embedding.default", EmbeddingDimensions: 768,
		EmbeddingPolicySchemaVersion: "policy/v4", Scope: "tenant-a", DryRun: true,
	}))
	require.NoError(t, err)
	require.Equal(t, "shadow-job", resp.Msg.GetJobId())
	require.Len(t, provider.requests, 2)
	shadow := provider.requests[1]
	require.Equal(t, "shadow", shadow.GetAction())
	require.Equal(t, "candidate-recipe-b", shadow.GetShadowCollection())
	require.Equal(t, "nomic-embed-text:latest", shadow.GetEmbeddingModel())
	require.Equal(t, "embedding.default", shadow.GetEmbeddingRole())
	require.Equal(t, int32(768), shadow.GetEmbeddingDimensions())
	require.Equal(t, "policy/v4", shadow.GetEmbeddingPolicySchemaVersion())
	require.Equal(t, "tenant-a", shadow.GetScope())
	require.True(t, shadow.GetDryRun())
}

func TestEmbeddingMigrationRefusesBeforeDispatchWhenNegotiatedActionUnsupported(t *testing.T) {
	store := &fakeStore{descriptor: validDescriptor(), tokenOut: "provider-token"}
	provider := &fakeProviderControl{responses: []*controlv1.ReindexResponse{{
		MigrationProtocolVersion:  aisearch.ReindexMigrationProtocolVersion,
		SupportedMigrationActions: []string{"shadow"},
	}}}
	client := newClientWithControl(t, store, provider)
	_, err := client.ExecuteEmbeddingMigration(context.Background(), connect.NewRequest(&registryv1.ExecuteEmbeddingMigrationRequest{
		ProviderId: "cli-health.commands", Action: "cutover", ShadowCollection: "candidate",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Len(t, provider.requests, 1, "only the read-only capability probe may reach provider")
	require.Equal(t, "capabilities", provider.requests[0].GetAction())
}

func TestEmbeddingMigrationReturnsNegotiatedCapabilities(t *testing.T) {
	store := &fakeStore{descriptor: validDescriptor(), tokenOut: "provider-token"}
	provider := &fakeProviderControl{responses: []*controlv1.ReindexResponse{{
		MigrationProtocolVersion:  aisearch.ReindexMigrationProtocolVersion,
		SupportedMigrationActions: []string{"shadow", "cutover", "rollback"},
	}}}
	client := newClientWithControl(t, store, provider)
	resp, err := client.ExecuteEmbeddingMigration(context.Background(), connect.NewRequest(&registryv1.ExecuteEmbeddingMigrationRequest{
		ProviderId: "cli-health.commands", Action: "capabilities",
	}))
	require.NoError(t, err)
	require.Equal(t, aisearch.ReindexMigrationProtocolVersion, resp.Msg.GetMigrationProtocolVersion())
	require.Equal(t, []string{"shadow", "cutover", "rollback"}, resp.Msg.GetSupportedActions())
}

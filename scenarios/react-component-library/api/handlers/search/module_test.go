package search

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search/search_v1connect"
	domain "react-component-library/internal/search"
)

func newHandlerService(t *testing.T) (*Handler, *domain.Service) {
	t.Helper()
	database := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	service := domain.NewService(database, t.TempDir(), nil)
	require.NoError(t, service.ReplaceDocuments(context.Background(), []domain.Document{
		{
			SubjectID:    "controls.button",
			Kind:         searchv1.SearchKind_SEARCH_KIND_ASSET,
			Title:        "Button",
			Body:         "Primary action control.",
			Href:         "/library?q=Button",
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
		},
		{
			SubjectID:    "workspace-harbor",
			Kind:         searchv1.SearchKind_SEARCH_KIND_WORKSPACE,
			Title:        "Harbor onboarding",
			Body:         "First-run onboarding workspace.",
			Href:         "/design/workspace-harbor",
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
		},
	}))
	return &Handler{Service: service}, service
}

func TestSearchUsesRequestRoutedDBPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	ensureSchema := func(ctx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(ctx, pool, apidb.SchemaProviderFunc(Schema))
	}
	require.NoError(t, ensureSchema(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(ensureSchema)
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "search-routing", time.Minute))

	service := domain.NewService(routed, t.TempDir(), nil)
	require.NoError(t, service.ReplaceDocuments(ctx, []domain.Document{{
		SubjectID: "primary-only", Kind: searchv1.SearchKind_SEARCH_KIND_WORKSPACE,
		Title: "Primary only", Body: "routing marker", Href: "/primary-only",
	}}))
	handler := NewHandler(service)
	primaryResponse, err := handler.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: "routing marker"}))
	require.NoError(t, err)
	require.Len(t, primaryResponse.Msg.GetResults(), 1)
	require.Equal(t, "workspace:primary-only", primaryResponse.Msg.Results[0].GetId())

	testResponse, err := handler.Search(apidb.WithTestMode(ctx), connect.NewRequest(&searchv1.SearchRequest{Query: "routing marker"}))
	require.NoError(t, err)
	require.Empty(t, testResponse.Msg.GetResults(), "test-mode request must read the isolated routed pool")
	status, err := handler.GetSearchStatus(apidb.WithTestMode(ctx), connect.NewRequest(&searchv1.GetSearchStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(0), status.Msg.GetDocumentCount())
}

func TestEndpointsCoverEveryProcedure(t *testing.T) {
	procedures := map[string]bool{
		searchconnect.SearchServiceSearchProcedure:          false,
		searchconnect.SearchServiceSuggestProcedure:         false,
		searchconnect.SearchServiceGetSearchStatusProcedure: false,
		searchconnect.SearchServiceReindexProcedure:         false,
	}
	require.Len(t, Endpoints, len(procedures))
	ids := map[string]bool{}
	for _, endpoint := range Endpoints {
		require.Contains(t, procedures, endpoint.Path)
		require.False(t, procedures[endpoint.Path], "procedure %s is described twice", endpoint.Path)
		procedures[endpoint.Path] = true
		require.False(t, ids[endpoint.ID], "endpoint id %s is duplicated", endpoint.ID)
		ids[endpoint.ID] = true
		require.Equal(t, "POST", endpoint.Method)
		require.Equal(t, "search", endpoint.Category)
	}
	for procedure, seen := range procedures {
		require.True(t, seen, "procedure %s has no endpoint descriptor", procedure)
	}
}

func TestSearchRPCsUseTheService(t *testing.T) {
	handler, _ := newHandlerService(t)
	ctx := context.Background()

	searchResponse, err := handler.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: "button"}))
	require.NoError(t, err)
	require.NotEmpty(t, searchResponse.Msg.Results)
	require.Equal(t, "Button", searchResponse.Msg.Results[0].GetTitle())
	require.True(t, searchResponse.Msg.Status.GetIndexed())

	suggestResponse, err := handler.Suggest(ctx, connect.NewRequest(&searchv1.SuggestRequest{Query: "harb", Limit: 5}))
	require.NoError(t, err)
	require.Contains(t, suggestResponse.Msg.Suggestions, "Harbor onboarding")

	statusResponse, err := handler.GetSearchStatus(ctx, connect.NewRequest(&searchv1.GetSearchStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(2), statusResponse.Msg.GetDocumentCount())
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED, statusResponse.Msg.GetSemanticLeg())
}

func TestAskRPCReportsUnavailableInsteadOfLexicalAnswer(t *testing.T) {
	handler, _ := newHandlerService(t)
	_, err := handler.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "button", Ask: true}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeUnavailable, connectErr.Code())
}

func TestAskProviderFailureMapsToUnavailable(t *testing.T) {
	err := searchConnectError(domain.ErrAskProviderUnavailable)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestAskInvalidModelAnswerMapsToUnavailableAndInvalidQuestionMapsToArgument(t *testing.T) {
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(searchConnectError(domain.ErrAskInvalidAnswer)))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(searchConnectError(domain.ErrAskInvalidQuery)))
}

func TestUninitializedIndexMapsToFailedPrecondition(t *testing.T) {
	database := databasetest.NewSQLite(t) // schema deliberately not applied
	handler := NewHandler(domain.NewService(database, t.TempDir(), nil))

	_, err := handler.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "button"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	// Status stays answerable before the schema exists so the UI can prompt a
	// rebuild instead of showing a fault.
	status, statusErr := handler.GetSearchStatus(context.Background(), connect.NewRequest(&searchv1.GetSearchStatusRequest{}))
	require.NoError(t, statusErr)
	require.False(t, status.Msg.GetIndexed())
}

func TestHandlerWithoutServiceIsUnavailable(t *testing.T) {
	handler := Handler{}
	_, err := handler.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{Query: "button"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

// TestModuleServesConnectOverHTTP proves the mount path and generated client
// agree for all four RPCs. A handler that compiles but is mounted at the wrong
// prefix fails here.
func TestModuleServesConnectOverHTTP(t *testing.T) {
	scenarioRoot := t.TempDir()
	assetDir := filepath.Join(scenarioRoot, "catalog", "assets", "controls")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "button.json"), []byte(`{
		"kind": "catalog-asset",
		"asset": {
			"id": "controls.button",
			"name": "Button",
			"kind": "component",
			"domain": "controls",
			"description": "Primary action control.",
			"targets": ["react-vite"]
		}
	}`), 0o644))

	database := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))

	router := mux.NewRouter()
	Module(database, scenarioRoot, nil).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := searchconnect.NewSearchServiceClient(server.Client(), server.URL)

	ctx := context.Background()
	reindexed, err := client.Reindex(ctx, connect.NewRequest(&searchv1.ReindexRequest{Force: true}))
	require.NoError(t, err)
	require.Equal(t, int32(10), reindexed.Msg.GetDocumentCount(), "one catalog asset plus nine commands")
	require.NotEmpty(t, reindexed.Msg.GetIndexedAt())

	response, err := client.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: "button"}))
	require.NoError(t, err)
	require.NotEmpty(t, response.Msg.Results)
	require.Equal(t, "Button", response.Msg.Results[0].GetTitle())
	require.Equal(t, "/library?q=Button", response.Msg.Results[0].GetHref())
	require.True(t, response.Msg.Status.GetIndexed())

	status, err := client.GetSearchStatus(ctx, connect.NewRequest(&searchv1.GetSearchStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(10), status.Msg.GetDocumentCount())
	require.False(t, status.Msg.GetStale())
	require.Equal(t, searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED, status.Msg.GetSemanticLeg())
}

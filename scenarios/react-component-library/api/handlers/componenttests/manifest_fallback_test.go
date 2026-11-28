package componenttests_test

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	testsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"
	"google.golang.org/protobuf/encoding/protojson"

	componenthandlers "react-component-library/handlers/components"
	componenttesthandlers "react-component-library/handlers/componenttests"
	internalcomponents "react-component-library/internal/components"
	localdb "react-component-library/internal/database"
)

// staleCatalogLookup simulates a registry index that has not yet caught up
// with the authored catalog manifest. Canonical library-id lookups continue
// to use the real registry service after the handler resolves the public id.
type staleCatalogLookup struct {
	internalcomponents.Service
	catalogGetCalls  atomic.Int32
	catalogListCalls atomic.Int32
}

func (s *staleCatalogLookup) Get(ctx context.Context, id string) (internalcomponents.Component, error) {
	if id == "controls.button" {
		s.catalogGetCalls.Add(1)
		return internalcomponents.Component{}, sql.ErrNoRows
	}
	return s.Service.Get(ctx, id)
}

func (s *staleCatalogLookup) GetByLibraryID(ctx context.Context, id string) (internalcomponents.Component, error) {
	if id == "controls.button" {
		return internalcomponents.Component{}, sql.ErrNoRows
	}
	return s.Service.GetByLibraryID(ctx, id)
}

func (s *staleCatalogLookup) List(ctx context.Context, query internalcomponents.SearchQuery) ([]internalcomponents.Component, error) {
	if query.Limit == 2000 {
		s.catalogListCalls.Add(1)
		return []internalcomponents.Component{}, nil
	}
	return s.Service.List(ctx, query)
}

func TestRunReportResolvesPublicCatalogIDFromManifestWhenRegistryIsStale(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internalcomponents.Schema)))

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "catalog", "assets"), 0o755))
	sourceRoot := filepath.Join(root, "library")
	versionRoot := filepath.Join(sourceRoot, "components", "Button", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(versionRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceRoot, "components", "Button", "component.json"), []byte(`{"libraryId":"rcl:Button","catalogId":"controls.button","displayName":"Button","latest":"1.0.0"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionRoot, "Button.tsx"), []byte("/**\n * @libraryId rcl:Button\n * @version 1.0.0\n */\nexport const Button = () => null;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionRoot, "story.json"), []byte(`{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"idle","name":"Idle","args":{},"expect":[{"kind":"role","role":"button","name":"Button"}]}]}`), 0o644))

	assets, repo := componenthandlers.BuildService(database, schedule.System(), sourceRoot)
	staleAssets := &staleCatalogLookup{Service: assets}
	router := mux.NewRouter()
	componenthandlers.ModuleFromService(assets, repo, sourceRoot, log.New(io.Discard, "", 0)).Mount(router)
	componenttesthandlers.ModuleWithExecutor(database, staleAssets, sourceRoot, passingExecutor{}, log.New(io.Discard, "", 0)).Mount(router)
	index := call(router, componentsconnect.ComponentsServiceIndexComponentsProcedure, `{}`)
	require.Equal(t, http.StatusOK, index.Code, index.Body.String())

	run := call(router, testsconnect.ComponentTestsServiceRunComponentTestProcedure, `{"componentId":"controls.button","version":"1.0.0"}`)
	require.Equal(t, http.StatusOK, run.Code, run.Body.String())
	var runResponse componenttestsv1.RunComponentTestResponse
	require.NoError(t, protojson.Unmarshal(run.Body.Bytes(), &runResponse), run.Body.String())
	require.NotNil(t, runResponse.GetReport())
	require.Equal(t, "passed", runResponse.GetReport().GetVerdict())
	require.NotEmpty(t, runResponse.GetReport().GetId())
	require.NotEmpty(t, runResponse.GetReport().GetResults())
	require.GreaterOrEqual(t, staleAssets.catalogGetCalls.Load(), int32(1), "the public catalog id should miss the stale registry before manifest fallback")
	require.GreaterOrEqual(t, staleAssets.catalogListCalls.Load(), int32(1), "the stale registry scan should be empty before manifest fallback")

	get := call(router, testsconnect.ComponentTestsServiceGetComponentTestReportProcedure, `{"id":"`+runResponse.GetReport().GetId()+`"}`)
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	var durable componenttestsv1.GetComponentTestReportResponse
	require.NoError(t, protojson.Unmarshal(get.Body.Bytes(), &durable), get.Body.String())
	require.Equal(t, runResponse.GetReport().GetId(), durable.GetReport().GetId())
	require.Equal(t, "rcl:Button", durable.GetReport().GetRootLibraryId())
	require.Equal(t, "passed", durable.GetReport().GetVerdict())
}

package deps_test

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	depsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/deps/deps_v1connect"

	"react-component-library/handlers/deps"
	localdb "react-component-library/internal/database"
	internaldeps "react-component-library/internal/deps"

	db "github.com/vrooli/api-core/databasetest"
)

type fakePackageReader struct {
	byScenario map[string][]byte
}

func (f fakePackageReader) Read(_ context.Context, scenario string) ([]byte, error) {
	return f.byScenario[scenario], nil
}

func setupModule(t *testing.T) (*mux.Router, internaldeps.Service) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaldeps.Schema),
	))
	pkgs := fakePackageReader{byScenario: map[string][]byte{
		"target": []byte(`{"dependencies":{"react":"^18.3.1"}}`),
	}}
	svc := deps.BuildService(d, pkgs)
	m := deps.ModuleFromService(svc, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)
	return r, svc
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, deps.Endpoints, 2)
}

func TestModule_ListAndValidate(t *testing.T) {
	r, svc := setupModule(t)
	require.NoError(t, svc.SyncForComponent(context.Background(), internaldeps.SyncInput{
		ComponentID: "cmp-1",
		LibraryID:   "react-component-library:Button",
		Declarations: []internaldeps.DeclarationFields{
			{Version: "1.0.0", DepName: "react", VersionRange: "^18.0.0"},
		},
	}))

	rw := callConnect(r, depsconnect.DepsServiceListDeclarationsProcedure, `{"component_id":"cmp-1"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"depName":"react"`)
	require.Contains(t, rw.Body.String(), `"versionRange":"^18.0.0"`)
	require.Contains(t, rw.Body.String(), `"version":"1.0.0"`)

	rw = callConnect(r, depsconnect.DepsServiceValidateAdoptionProcedure, `{"component_id":"cmp-1","scenario":"target","version":"1.0.0"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"kind":"VERDICT_KIND_OK"`)
}

func TestModule_ValidateRejectsMissingComponentID(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, depsconnect.DepsServiceValidateAdoptionProcedure, `{"scenario":"target"}`)
	require.Equal(t, http.StatusBadRequest, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"invalid_argument"`)
}

func TestModule_ValidateAdoptionReportsBlockingIncompatibleMajor(t *testing.T) {
	r, svc := setupModule(t)
	require.NoError(t, svc.SyncForComponent(context.Background(), internaldeps.SyncInput{
		ComponentID: "cmp-legacy",
		LibraryID:   "react-component-library:LegacyWidget",
		Declarations: []internaldeps.DeclarationFields{
			{Version: "1.0.0", DepName: "react", VersionRange: "^17.0.0"},
		},
	}))

	rw := callConnect(r, depsconnect.DepsServiceValidateAdoptionProcedure, `{"component_id":"cmp-legacy","scenario":"target","version":"1.0.0"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"kind":"VERDICT_KIND_BLOCK"`)
	require.Contains(t, rw.Body.String(), `"kind":"ISSUE_KIND_INCOMPATIBLE_MAJOR"`)
	require.Contains(t, rw.Body.String(), `"scenarioVersion":"^18.3.1"`)
	require.Contains(t, rw.Body.String(), `"detail":"declared ^17.x.x but target is 18.x.x"`)
}

func TestModule_RoutesReadsFromRequestContext(t *testing.T) {
	ctx := context.Background()
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	testPath := filepath.Join(t.TempDir(), "test.db")
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryPath})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, routed.Close()) })
	ensure := func(ctx context.Context, d *sql.DB) error {
		return apidb.EnsureSchemas(ctx, d,
			apidb.SchemaProviderFunc(localdb.SystemSchema),
			apidb.SchemaProviderFunc(internaldeps.Schema),
		)
	}
	require.NoError(t, ensure(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(ensure)
	require.NoError(t, routed.InstallTestPool(ctx, testPath, "deps-route", 0))

	svc := deps.BuildService(routed, fakePackageReader{})
	require.NoError(t, svc.SyncForComponent(apidb.WithTestMode(ctx), internaldeps.SyncInput{
		ComponentID:  "routed-component",
		LibraryID:    "react-component-library:Routed",
		Declarations: []internaldeps.DeclarationFields{{Version: "1.0.0", DepName: "react", VersionRange: "^18.0.0"}},
	}))
	m := deps.ModuleFromService(svc, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)

	primary := callConnect(r, depsconnect.DepsServiceListDeclarationsProcedure, `{"component_id":"routed-component"}`)
	require.Equal(t, http.StatusOK, primary.Code, primary.Body.String())
	require.NotContains(t, primary.Body.String(), `"depName":"react"`)
	testReq, err := http.NewRequest(http.MethodPost, depsconnect.DepsServiceListDeclarationsProcedure, strings.NewReader(`{"component_id":"routed-component"}`))
	require.NoError(t, err)
	testReq = testReq.WithContext(apidb.WithTestMode(testReq.Context()))
	testReq.Header.Set("Content-Type", "application/json")
	testResponse := httptest.NewRecorder()
	r.ServeHTTP(testResponse, testReq)
	require.Equal(t, http.StatusOK, testResponse.Code, testResponse.Body.String())
	require.Contains(t, testResponse.Body.String(), `"depName":"react"`)
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}

package adoptions_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	intakeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptionintake/adoptionintake_v1connect"
	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"react-component-library/handlers/adoptions"
	internaladoptions "react-component-library/internal/adoptions"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"

	"github.com/vrooli/api-core/schedule"
)

// stubLibrary is the minimal LibraryReader the handler needs for
// transport-edge tests. Pure-Go, no sqlite — keeps the handler suite
// orthogonal to the components repository.
type stubLibrary struct {
	component components.Component
	content   components.Content
}

func (s *stubLibrary) Get(_ context.Context, id string) (components.Component, error) {
	if id != s.component.ID {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return s.component, nil
}

func (s *stubLibrary) List(_ context.Context, _ components.SearchQuery) ([]components.Component, error) {
	return []components.Component{s.component}, nil
}

func (s *stubLibrary) GetContent(_ context.Context, id string) (components.Content, error) {
	if id != s.component.ID {
		return components.Content{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return s.content, nil
}

func (s *stubLibrary) GetVersion(_ context.Context, componentID, version string) (components.ComponentVersion, error) {
	if componentID != s.component.ID {
		return components.ComponentVersion{}, components.ErrComponentNotFound{IDOrLibraryID: componentID + "@" + version}
	}
	return components.ComponentVersion{
		ComponentID:   componentID,
		LibraryID:     s.component.LibraryID,
		Version:       version,
		Status:        components.VersionStatusReleased,
		Content:       s.content.Body,
		ContentSHA256: s.content.SHA256,
		SourcePath:    "components/Button/versions/" + version + "/Button.tsx",
	}, nil
}

func (s *stubLibrary) ListVersions(ctx context.Context, componentID string, _ int) ([]components.ComponentVersion, error) {
	if componentID != s.component.ID {
		return nil, nil
	}
	v, err := s.GetVersion(ctx, componentID, s.component.Version)
	if err != nil {
		return nil, err
	}
	return []components.ComponentVersion{v}, nil
}

func setupModule(t *testing.T) (*mux.Router, string, *stubLibrary) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaladoptions.Schema),
	))
	require.NoError(t, internaladoptions.EnsureMigrations(context.Background(), d))
	root := t.TempDir()
	lib := &stubLibrary{
		component: components.Component{ID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.0.0", LatestVersion: "1.0.0"},
		content:   components.Content{Body: "X", SHA256: sha256OfHandlerTests("X")},
	}
	m := adoptions.ModuleWithRoot(d, schedule.System(), lib, root, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)
	return r, root, lib
}

func sha256OfHandlerTests(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func TestModule_Shape(t *testing.T) {
	r, _, _ := setupModule(t)
	require.NotNil(t, r)
	ids := make(map[string]bool, len(adoptions.Endpoints))
	for _, endpoint := range adoptions.Endpoints {
		ids[endpoint.ID] = true
	}
	for _, id := range []string{
		"adoptions_list_scenarios", "adoptions_reconcile", "adoptions_reconverge",
		"adoptions_list_effective", "adoptions_sync_tokens", "adoptions_prune_tokens",
		"adoptions_resolve_path", "adoptions_discover", "adoptions_confirm_discovery",
	} {
		require.Truef(t, ids[id], "missing endpoint %q", id)
	}
}

func TestListAdoptionsUsesRequestRoutedDBPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "routed-primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	ensureSchema := func(ctx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(ctx, pool, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internaladoptions.Schema))
	}
	require.NoError(t, ensureSchema(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(ensureSchema)
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "adoptions-routing", time.Minute))
	_, err = internaladoptions.NewSQLiteRepository(routed.Primary(), schedule.System()).Create(ctx, internaladoptions.CreateInput{
		ID: "primary-only", ComponentID: "cmp-btn", LibraryID: "rcl:Button", Scenario: "demo", AdoptedPath: "components/Button/Button.tsx", AdoptedVersion: "1.0.0",
	})
	require.NoError(t, err)

	router := mux.NewRouter()
	adoptions.ModuleWithRoot(routed, schedule.System(), &stubLibrary{component: components.Component{ID: "cmp-btn", LibraryID: "rcl:Button"}}, t.TempDir(), log.New(io.Discard, "", 0)).Mount(router)
	list := func(requestContext context.Context) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, adoptionsconnect.AdoptionsServiceListAdoptionsProcedure, strings.NewReader(`{}`)).WithContext(requestContext)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	primaryResponse := list(ctx)
	require.Equal(t, http.StatusOK, primaryResponse.Code, primaryResponse.Body.String())
	require.Contains(t, primaryResponse.Body.String(), "primary-only")
	testResponse := list(apidb.WithTestMode(ctx))
	require.Equal(t, http.StatusOK, testResponse.Code, testResponse.Body.String())
	require.NotContains(t, testResponse.Body.String(), "primary-only", "test-mode request must read the isolated routed pool")
}

func TestModule_CreateListRefreshDelete(t *testing.T) {
	r, root, _ := setupModule(t)

	// Pre-seed the target scenario file so Create can hash the snapshot.
	scenarioDir := filepath.Join(root, "swarm-manager")
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "Button.tsx"), []byte("X"), 0o600))

	rw := callConnect(r, adoptionsconnect.AdoptionsServiceApplyAdoptionProcedure,
		`{"component_id":"cmp-btn","scenario":"swarm-manager","adopted_path":"Button.tsx","version":"1.0.0","replace_existing":true,"confirm_overwrite":true}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), "swarm-manager")

	rw = callConnect(r, adoptionsconnect.AdoptionsServiceListAdoptionsProcedure, `{"limit":10}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), "Button.tsx")

	rw = callConnect(r, adoptionsconnect.AdoptionsServiceRefreshAdoptionsProcedure, `{}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	// X matches the library content sha → current.
	require.Contains(t, rw.Body.String(), "LIBRARY_VERSION_STATUS_CURRENT")
	require.Contains(t, rw.Body.String(), "LOCAL_STATUS_CLEAN")
}

func TestModule_CreateRejectsMissingComponent(t *testing.T) {
	r, _, _ := setupModule(t)
	rw := callConnect(r, adoptionsconnect.AdoptionsServiceApplyAdoptionProcedure,
		`{"component_id":"nope","scenario":"x","adopted_path":"p.tsx"}`)
	require.Equal(t, http.StatusBadRequest, rw.Code, rw.Body.String())
}

func TestModule_DeleteRequiresConfirmationBeforeRemovingLastOwnedFile(t *testing.T) {
	r, root, _ := setupModule(t)
	scenarioDir := filepath.Join(root, "swarm-manager")
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	filePath := filepath.Join(scenarioDir, "Button.tsx")
	require.NoError(t, os.WriteFile(filePath, []byte("X"), 0o600))

	created := callConnect(r, adoptionsconnect.AdoptionsServiceApplyAdoptionProcedure,
		`{"component_id":"cmp-btn","scenario":"swarm-manager","adopted_path":"Button.tsx","version":"1.0.0","replace_existing":true,"confirm_overwrite":true}`)
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	var applyResponse struct {
		Adoption struct {
			ID string `json:"id"`
		} `json:"adoption"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &applyResponse))
	require.NotEmpty(t, applyResponse.Adoption.ID)

	deletePath := adoptionsconnect.AdoptionsServiceDeleteAdoptionProcedure
	preview := callConnect(r, deletePath, `{"id":"`+applyResponse.Adoption.ID+`"}`)
	require.Equal(t, http.StatusBadRequest, preview.Code, preview.Body.String())
	require.Contains(t, preview.Body.String(), "Button.tsx")
	require.FileExists(t, filePath, "unconfirmed deletion must leave the scenario file intact")

	deleted := callConnect(r, deletePath, `{"id":"`+applyResponse.Adoption.ID+`","confirm_remove_files":true}`)
	require.Equal(t, http.StatusOK, deleted.Code, deleted.Body.String())
	require.Contains(t, deleted.Body.String(), `"removedFiles":["Button.tsx"]`)
	require.NoFileExists(t, filePath)

	listed := callConnect(r, adoptionsconnect.AdoptionsServiceListAdoptionsProcedure, `{"limit":10}`)
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	require.NotContains(t, listed.Body.String(), applyResponse.Adoption.ID)
}

func TestModule_AdoptionIntakePathsHaveSingleOwner(t *testing.T) {
	r, _, _ := setupModule(t)

	intake := callConnect(r, intakeconnect.AdoptionIntakeServiceSuggestAdoptionsProcedure, `{}`)
	require.Equal(t, http.StatusNotImplemented, intake.Code, intake.Body.String())

	legacy := callConnect(r, "/vrooli.react_component_library.v1.adoptions.AdoptionsService/SuggestAdoptions", `{}`)
	require.Equal(t, http.StatusNotFound, legacy.Code, legacy.Body.String())
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}

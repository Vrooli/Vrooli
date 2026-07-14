package adoptions_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	adoptionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions/adoptions_v1connect"

	"react-component-library/handlers/adoptions"
	internaladoptions "react-component-library/internal/adoptions"
	"react-component-library/internal/clock"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/testutil/db"
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

func setupModule(t *testing.T) (*mux.Router, string, *stubLibrary) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaladoptions.Schema),
	))
	root := t.TempDir()
	lib := &stubLibrary{
		component: components.Component{ID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.0.0", LatestVersion: "1.0.0"},
		content:   components.Content{Body: "X", SHA256: sha256OfHandlerTests("X")},
	}
	m := adoptions.ModuleWithRoot(d, clock.System{}, lib, root, log.New(io.Discard, "", 0))
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
	require.Len(t, adoptions.Endpoints, 8, "adoptions ships reconcile, list, apply, reapply, delete, refresh, resolve-path, suggest")
	require.Equal(t, "adoptions_reconcile", adoptions.Endpoints[0].ID)
	require.Equal(t, "adoptions_resolve_path", adoptions.Endpoints[6].ID)
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

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}

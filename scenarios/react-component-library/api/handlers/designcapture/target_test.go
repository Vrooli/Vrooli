package designcapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	domain "react-component-library/internal/designcapture"
	"react-component-library/internal/preview"
)

func TestTargetServesExactIsolatedBytesAndRejectsCorruption(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(domain.Schema)))
	repo := domain.NewSQLiteRepository(db)
	html := "<!doctype html><html><body>Exact bytes</body></html>"
	digest := sha256.Sum256([]byte(html))
	hash := strings.Repeat("a", 64)
	request := domain.Request{Target: domain.Target{Scenario: "demo", DesignID: "home", Revision: hash, RenderHash: hash, InputsSHA256: hash, HTMLSHA256: hex.EncodeToString(digest[:]), Kind: "preview", Kit: "vrooli-default", Theme: "light", Direction: "ltr"}, HTML: html, Width: 390, Height: 844}
	op, err := repo.Create(ctx, "target-one", request)
	require.NoError(t, err)
	router := mux.NewRouter()
	MountTarget(router, func(context.Context) (domain.Repository, error) { return repo, nil })
	get := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/design-captures/"+op.ID+"/target.html", nil))
		return w
	}
	response := get()
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, html, response.Body.String())
	require.Equal(t, "sandbox allow-scripts", response.Header().Get("Content-Security-Policy"))
	require.Equal(t, hash, response.Header().Get("X-RCL-Render-Hash"))
	_, err = db.ExecContext(ctx, `UPDATE design_capture_operations SET request_hash=? WHERE id=?`, strings.Repeat("b", 64), op.ID)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, get().Code)
}
func TestTargetRefusesMissingRoutedRepository(t *testing.T) {
	router := mux.NewRouter()
	MountTarget(router, func(context.Context) (domain.Repository, error) { return nil, errors.New("missing test lease") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/design-captures/anything/target.html", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestTargetLocalFormPolicyRetainsNavigationProhibition(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(domain.Schema)))
	repo := domain.NewSQLiteRepository(db)
	html := "<!doctype html><html><head>" + preview.LocalSubmitPolicyMarker + "</head><body>Exact bytes</body></html>"
	digest := sha256.Sum256([]byte(html))
	hash := strings.Repeat("a", 64)
	request := domain.Request{Target: domain.Target{Scenario: "demo", DesignID: "home", Revision: hash, RenderHash: hash, InputsSHA256: hash, HTMLSHA256: hex.EncodeToString(digest[:]), Kind: "preview", Kit: "vrooli-default", Theme: "light", Direction: "ltr"}, HTML: html, Width: 390, Height: 844}
	op, err := repo.Create(ctx, "target-one", request)
	require.NoError(t, err)
	router := mux.NewRouter()
	MountTarget(router, func(context.Context) (domain.Repository, error) { return repo, nil })
	get := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/design-captures/"+op.ID+"/target.html", nil))
		return w
	}
	response := get()
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, html, response.Body.String())
	require.Equal(t, "sandbox allow-scripts allow-forms; form-action 'none'", response.Header().Get("Content-Security-Policy"))
	require.Equal(t, hash, response.Header().Get("X-RCL-Render-Hash"))
	_, err = db.ExecContext(ctx, `UPDATE design_capture_operations SET request_hash=? WHERE id=?`, strings.Repeat("b", 64), op.ID)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, get().Code)
}

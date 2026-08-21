package versions_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"react-component-library/handlers/versions"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	internalversions "react-component-library/internal/versions"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
)

type fakeAdoptions struct{ content map[string]string }

func (f *fakeAdoptions) ResolveAdoption(_ context.Context, id string) (string, error) {
	return f.content[id], nil
}

func setupModule(t *testing.T) (*mux.Router, internalversions.Service) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
	))
	resolver := &fakeAdoptions{content: map[string]string{
		"adp-1": "library shared\nadopted line",
	}}
	svc := versions.BuildService(d, schedule.System(), resolver)
	m := versions.Module(d, schedule.System(), resolver, log.New(io.Discard, "", 0))
	r := mux.NewRouter()
	m.Mount(r)
	return r, svc
}

func TestModule_Shape(t *testing.T) {
	r, _ := setupModule(t)
	require.NotNil(t, r)
	require.Len(t, versions.Endpoints, 11)
}

func TestModule_RecordListDiff(t *testing.T) {
	r, svc := setupModule(t)
	ctx := context.Background()

	// Seed two versions for cmp-1 via the service so we exercise the
	// production Record path (which also resolves the listener seam).
	_, _, err := svc.Record(ctx, internalversions.RecordInput{
		ComponentID: "cmp-1",
		Content:     "// @version 1.0.0\nshared\nbeta",
	})
	require.NoError(t, err)
	_, _, err = svc.Record(ctx, internalversions.RecordInput{
		ComponentID: "cmp-1",
		Content:     "// @version 1.0.1\nshared\nGAMMA",
	})
	require.NoError(t, err)

	rw := callConnect(r, versionsconnect.VersionsServiceListVersionsProcedure,
		`{"component_id":"cmp-1","limit":10}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"version":"1.0.1"`)
	require.Contains(t, rw.Body.String(), `"version":"1.0.0"`)

	rw = callConnect(r, versionsconnect.VersionsServiceGetVersionProcedure,
		`{"component_id":"cmp-1","version":"1.0.0","include_content":true}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), "shared")
	require.Contains(t, rw.Body.String(), "beta")

	rw = callConnect(r, versionsconnect.VersionsServiceDiffVersionsProcedure,
		`{"component_id":"cmp-1","from":"1.0.0","to":"1.0.1"}`)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	body := rw.Body.String()
	require.Contains(t, body, `"fromLabel":"1.0.0"`)
	require.Contains(t, body, `"toLabel":"1.0.1"`)
	// Header line text and beta/GAMMA pair produce mixed ops; counts
	// vary with the exact alignment, but the response must surface at
	// least one addition and one removal.
	require.Contains(t, body, `"additions"`)
	require.Contains(t, body, `"removals"`)
}

func TestModule_DiffAdoptionSide(t *testing.T) {
	r, svc := setupModule(t)
	ctx := context.Background()
	_, _, err := svc.Record(ctx, internalversions.RecordInput{
		ComponentID: "cmp-1",
		Content:     "library shared\nlibrary line",
	})
	require.NoError(t, err)

	rw := callConnect(r, versionsconnect.VersionsServiceDiffVersionsProcedure,
		`{"component_id":"cmp-1","from":"","to":"adoption:adp-1"}`)
	require.Equal(t, http.StatusBadRequest, rw.Code, "empty from must reject")

	rw = callConnect(r, versionsconnect.VersionsServiceDiffVersionsProcedure,
		`{"component_id":"cmp-1","from":"","to":""}`)
	require.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestModule_GetMissingReturnsNotFound(t *testing.T) {
	r, _ := setupModule(t)
	rw := callConnect(r, versionsconnect.VersionsServiceGetVersionProcedure,
		`{"component_id":"cmp-missing","version":"9.9.9"}`)
	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}

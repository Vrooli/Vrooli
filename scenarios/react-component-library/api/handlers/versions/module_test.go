package versions_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"react-component-library/handlers/versions"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/versionledger"
	internalversions "react-component-library/internal/versions"

	db "github.com/vrooli/api-core/databasetest"
	"google.golang.org/protobuf/reflect/protoreflect"

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
	require.Len(t, versions.Endpoints, 17)
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

type recoveryMaterializer struct{ calls int }

func (m *recoveryMaterializer) EnsureMaterialized(context.Context, string, string, string) (components.MaterializeResult, error) {
	m.calls++
	return components.MaterializeResult{}, nil
}

func TestDoctorTransportsOptionalOrphanLedgerMetadata(t *testing.T) {
	fields := (&versionsv1.DoctorIssue{}).ProtoReflect().Descriptor().Fields()
	wantNumbers := map[protoreflect.Name]protoreflect.FieldNumber{
		"ledger_created_at":  7,
		"ledger_released_at": 8,
		"ledger_retired_at":  9,
		"ledger_lifecycle":   10,
		"ledger_file_count":  11,
	}
	for name, number := range wantNumbers {
		field := fields.ByName(name)
		require.NotNil(t, field, "missing DoctorIssue field %s", name)
		require.Equal(t, number, field.Number())
		require.True(t, field.HasOptionalKeyword(), "field %s must preserve optional presence", name)
	}

	ctx := context.Background()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database,
		apidb.SchemaProviderFunc(components.Schema),
		apidb.SchemaProviderFunc(versionledger.Schema),
	))
	_, err := database.Exec(`INSERT INTO version_ledger (library_id, version, created_at, released_at, retired_at, lifecycle_state, file_count) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"react-component-library:Orphan", "1.2.3", "created-value", "released-value", "retired-value", "lifecycle-value", 5)
	require.NoError(t, err)
	_, err = database.Exec(`INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content_sha256, indexed_at, presence) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"evicted-id", "component-id", "react-component-library:Ordinary", "2.0.0", "released", "components/Ordinary/versions/2.0.0/Ordinary.tsx", "sha", "indexed-value", "evicted")
	require.NoError(t, err)

	handler := versions.NewConnectHandler(versions.Deps{Ledger: versionledger.NewRepository(database, t.TempDir())})
	path, service := versionsconnect.NewVersionLifecycleServiceHandler(handler)
	router := mux.NewRouter()
	router.PathPrefix(path).Handler(service)
	response := callConnect(router, versionsconnect.VersionLifecycleServiceDoctorProcedure, `{}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var body struct {
		Issues []map[string]any `json:"issues"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Len(t, body.Issues, 2)
	issues := make(map[string]map[string]any, len(body.Issues))
	for _, issue := range body.Issues {
		issues[issue["libraryId"].(string)] = issue
	}
	orphan := issues["react-component-library:Orphan"]
	require.Equal(t, "created-value", orphan["ledgerCreatedAt"])
	require.Equal(t, "released-value", orphan["ledgerReleasedAt"])
	require.Equal(t, "retired-value", orphan["ledgerRetiredAt"])
	require.Equal(t, "lifecycle-value", orphan["ledgerLifecycle"])
	require.Equal(t, float64(5), orphan["ledgerFileCount"])
	ordinary := issues["react-component-library:Ordinary"]
	require.NotContains(t, ordinary, "ledgerCreatedAt")
	require.NotContains(t, ordinary, "ledgerReleasedAt")
	require.NotContains(t, ordinary, "ledgerRetiredAt")
	require.NotContains(t, ordinary, "ledgerLifecycle")
	require.NotContains(t, ordinary, "ledgerFileCount")
}

func TestImportArchiveKeepsWholeLedgerContractForBothOverwriteValues(t *testing.T) {
	ctx := context.Background()
	newDatabase := func() *sql.DB {
		database := db.NewSQLite(t)
		require.NoError(t, apidb.EnsureSchemas(ctx, database,
			apidb.SchemaProviderFunc(components.Schema),
			apidb.SchemaProviderFunc(versionledger.Schema),
		))
		return database
	}
	sourceDB := newDatabase()
	sourceLedger := versionledger.NewRepository(sourceDB, t.TempDir())
	_, err := sourceDB.Exec(`INSERT INTO version_ledger (library_id, version, lifecycle_state) VALUES ('react-component-library:First', '1.0.0', 'released')`)
	require.NoError(t, err)
	firstArchive := filepath.Join(t.TempDir(), "first-ledger.json")
	_, err = sourceLedger.ExportArchive(ctx, firstArchive)
	require.NoError(t, err)
	destination := newDatabase()
	materializer := &recoveryMaterializer{}
	handler := versions.NewConnectHandler(versions.Deps{Ledger: versionledger.NewRepository(destination, t.TempDir()), Materializer: materializer})
	path, service := versionsconnect.NewVersionLifecycleServiceHandler(handler)
	router := mux.NewRouter()
	router.PathPrefix(path).Handler(service)
	postImport := func(archivePath string, overwrite bool) *httptest.ResponseRecorder {
		body := fmt.Sprintf(`{"path":%q,"overwrite":%t}`, archivePath, overwrite)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, versionsconnect.VersionLifecycleServiceImportArchiveProcedure, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		return recorder
	}
	require.Equal(t, http.StatusOK, postImport(firstArchive, false).Code)
	var firstState string
	require.NoError(t, destination.QueryRow(`SELECT lifecycle_state FROM version_ledger WHERE library_id='react-component-library:First' AND version='1.0.0'`).Scan(&firstState))
	require.Equal(t, "released", firstState)

	secondDB := newDatabase()
	_, err = secondDB.Exec(`INSERT INTO version_ledger (library_id, version, lifecycle_state) VALUES ('react-component-library:Second', '2.0.0', 'archived')`)
	require.NoError(t, err)
	secondArchive := filepath.Join(t.TempDir(), "second-ledger.json")
	_, err = versionledger.NewRepository(secondDB, t.TempDir()).ExportArchive(ctx, secondArchive)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postImport(secondArchive, true).Code)
	var finalRows int
	require.NoError(t, destination.QueryRow(`SELECT COUNT(*) FROM version_ledger`).Scan(&finalRows))
	require.Equal(t, 1, finalRows)
	var secondState string
	require.NoError(t, destination.QueryRow(`SELECT lifecycle_state FROM version_ledger WHERE library_id='react-component-library:Second' AND version='2.0.0'`).Scan(&secondState))
	require.Equal(t, "archived", secondState)
	require.Equal(t, 0, materializer.calls)
}

func TestRecoverButton224RejectsMalformedBundleWithoutMutation(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "malformed", data: `{"schema_version":`},
		{name: "wrong identity", data: `{"schema_version":1,"library_id":"react-component-library:Other","version":"2.2.4","status":"released","source_revision":"50f54d5a89ebd8da93099475dbd1263eeb01af0a","published_at":"2026-08-31T05:19:38Z","files":[]}`},
		{name: "wrong path", data: `{"schema_version":1,"library_id":"react-component-library:Button","version":"2.2.4","status":"released","source_revision":"50f54d5a89ebd8da93099475dbd1263eeb01af0a","published_at":"2026-08-31T05:19:38Z","files":[{"path":"../Button.tsx","content":"bad","sha256":"bad","entry":true},{"path":"story.tsx","content":"bad","sha256":"bad"}]}`},
		{name: "wrong hash", data: `{"schema_version":1,"library_id":"react-component-library:Button","version":"2.2.4","status":"released","source_revision":"50f54d5a89ebd8da93099475dbd1263eeb01af0a","published_at":"2026-08-31T05:19:38Z","files":[{"path":"Button.tsx","content":"bad","sha256":"bad","entry":true},{"path":"story.tsx","content":"bad","sha256":"bad"}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := db.NewSQLite(t)
			require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
				apidb.SchemaProviderFunc(components.Schema),
				apidb.SchemaProviderFunc(versionledger.Schema),
			))
			materializedRoot := t.TempDir()
			materializer := &recoveryMaterializer{}
			ledger := versionledger.NewRepository(database, materializedRoot)
			handler := versions.NewConnectHandler(versions.Deps{Ledger: ledger, Materializer: materializer})
			path, service := versionsconnect.NewVersionLifecycleServiceHandler(handler)
			router := mux.NewRouter()
			router.PathPrefix(path).Handler(service)
			bundle := filepath.Join(t.TempDir(), "invalid-bundle.json")
			require.NoError(t, os.WriteFile(bundle, []byte(test.data), 0o600))
			before := recoveryTableCounts(t, database)
			body := fmt.Sprintf(`{"path":%q}`, bundle)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, versionsconnect.VersionLifecycleServiceRecoverButton224Procedure, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), `"code":"failed_precondition"`)
			require.Equal(t, before, recoveryTableCounts(t, database))
			require.Zero(t, materializer.calls)
			entries, err := os.ReadDir(materializedRoot)
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
}

func recoveryTableCounts(t *testing.T, database *sql.DB) [3]int {
	t.Helper()
	var counts [3]int
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM component_versions`).Scan(&counts[0]))
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM component_version_files`).Scan(&counts[1]))
	require.NoError(t, database.QueryRow(`SELECT COUNT(*) FROM version_ledger`).Scan(&counts[2]))
	return counts
}

func callConnect(r *mux.Router, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	return rw
}

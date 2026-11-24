package workflows_test

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
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	workflowconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows/workflows_v1connect"

	"react-component-library/handlers/workflows"
	domain "react-component-library/internal/workflows"
)

func TestModuleReadsRequestRoutedDBPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	applySchema := func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, domain.Schema())
		return err
	}
	require.NoError(t, applySchema(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(applySchema)
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "workflows-routing", time.Minute))

	_, err = domain.NewSQLiteRepository(routed.Primary(), schedule.System()).Create(ctx, domain.Workflow{
		ID: "primary-only", Kind: domain.KindExtract, Status: domain.StatusQueued,
	})
	require.NoError(t, err)

	router := mux.NewRouter()
	workflows.Module(routed, schedule.System(), nil, log.New(io.Discard, "", 0)).Mount(router)
	list := func(requestContext context.Context) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, workflowconnect.WorkflowsServiceListWorkflowsProcedure, strings.NewReader(`{}`)).WithContext(requestContext)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	primaryResponse := list(ctx)
	require.Equal(t, http.StatusOK, primaryResponse.Code, primaryResponse.Body.String())
	require.Contains(t, primaryResponse.Body.String(), "primary-only")
	testResponse := list(apidb.WithTestMode(ctx))
	require.Equal(t, http.StatusOK, testResponse.Code, testResponse.Body.String())
	require.NotContains(t, testResponse.Body.String(), "primary-only", "request context must select isolated routed pool")
}

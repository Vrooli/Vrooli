package activity_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	db "github.com/vrooli/api-core/databasetest"
	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity/activity_v1connect"

	activityhandler "react-component-library/handlers/activity"
	domain "react-component-library/internal/activity"
)

func newRouter(t *testing.T) (*mux.Router, *domain.Store) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), domain.Schema())
	require.NoError(t, err)
	store := domain.NewSQLiteStore(sqlDB)
	router := mux.NewRouter()
	activityhandler.ModuleFromService(store).Mount(router)
	return router, store
}

func call(router *mux.Router, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestListActivityRPC(t *testing.T) {
	router, store := newRouter(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	require.NoError(t, store.Record(ctx, domain.Event{Actor: "ada", Action: "asset.publish", SubjectKind: "asset", SubjectID: "button", At: base.Format(time.RFC3339)}))
	require.NoError(t, store.Record(ctx, domain.Event{Actor: "lin", Action: "workspace.save", SubjectKind: "workspace", SubjectID: "ws-1", At: base.Add(time.Hour).Format(time.RFC3339)}))

	response := call(router, activityconnect.ActivityServiceListActivityProcedure, `{"subject_kind":"asset","limit":10}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var got activityv1.ListActivityResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &got))
	require.Len(t, got.Events, 1)
	require.Equal(t, "button", got.Events[0].SubjectId)
	require.Equal(t, "ada", got.Events[0].Actor)
}

func TestGetActivitySummaryRPC(t *testing.T) {
	router, store := newRouter(t)
	ctx := context.Background()
	at := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, store.Record(ctx, domain.Event{Actor: "ada", Action: "asset.publish", SubjectKind: "asset", SubjectID: "button", At: at.Format(time.RFC3339)}))

	response := call(router, activityconnect.ActivityServiceGetActivitySummaryProcedure, `{"days":30}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var got activityv1.GetActivitySummaryResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &got))
	require.Equal(t, int32(1), got.Total)
	require.False(t, got.Empty)
	require.Len(t, got.ByActor, 1)
	require.Equal(t, "ada", got.ByActor[0].Actor)
}

func TestEmptySummaryIsMeasuredZero(t *testing.T) {
	router, _ := newRouter(t)
	response := call(router, activityconnect.ActivityServiceGetActivitySummaryProcedure, `{}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var got activityv1.GetActivitySummaryResponse
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &got))
	require.Zero(t, got.Total)
	require.True(t, got.Empty)
}

func TestInvalidSinceIsInvalidArgument(t *testing.T) {
	router, _ := newRouter(t)
	response := call(router, activityconnect.ActivityServiceListActivityProcedure, `{"since":"yesterday"}`)
	require.Contains(t, response.Body.String(), `"code":"invalid_argument"`, response.Body.String())
}

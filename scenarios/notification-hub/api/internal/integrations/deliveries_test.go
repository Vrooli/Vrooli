package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestDeliveryProjectionBoundsExactIncidentFilterAndReportsTruncation(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	handler := DeliveryProjectionHandler(service)

	query := url.Values{"prefix": {"incident."}, "limit": {"200"}, "incident_id": {"inc-1"}}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/deliveries?"+query.Encode(), nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response struct {
		Notifications []hub.DeliveryProjection `json:"notifications"`
		Truncated     bool                     `json:"truncated"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Empty(t, response.Notifications)
	require.False(t, response.Truncated)

	for i := 0; i < 201; i++ {
		query.Add("incident_id", "incident-"+strconv.Itoa(i))
	}
	tooMany := httptest.NewRecorder()
	handler.ServeHTTP(tooMany, httptest.NewRequest(http.MethodGet, "/api/v1/integrations/deliveries?"+query.Encode(), nil))
	require.Equal(t, http.StatusBadRequest, tooMany.Code, tooMany.Body.String())
}

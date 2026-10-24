package handlers

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	handlermocks "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
)

func TestHandleGetPressureHistory(t *testing.T) {
	now := time.Now().UTC()
	h := NewMetricsHandler(&config.Config{}, handlermocks.NewMonitorQuerier().WithPressureHistory(&models.PressureHistory{
		Start: now.Add(-time.Hour), End: now,
		SomeAvg10: []models.GPUHistoryPoint{{Timestamp: now, Value: 12.5}},
	}), slog.Default())
	w := httptest.NewRecorder()
	h.HandleGetPressureHistory(w, httptest.NewRequest(http.MethodGet, "/api/v1/forensics/pressure?window=1h", nil))
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if points, ok := body["memory_psi_some_avg10"].([]interface{}); !ok || len(points) != 1 {
		t.Fatalf("pressure history = %#v", body)
	}
}

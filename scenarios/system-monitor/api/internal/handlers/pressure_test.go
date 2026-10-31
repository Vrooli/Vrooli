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

func TestHandleGetPressureSnapshotReturnsTypedDegradedState(t *testing.T) {
	h := NewMetricsHandler(&config.Config{}, handlermocks.NewMonitorQuerier().WithPressureSnapshot(&models.PressureSnapshot{
		Available: false, DegradedReason: "memory PSI unavailable", Timestamp: time.Now().UTC(),
	}), slog.Default())
	w := httptest.NewRecorder()
	h.HandleGetPressureSnapshot(w, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/pressure", nil))
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["available"] != false || body["degraded_reason"] != "memory PSI unavailable" {
		t.Fatalf("pressure response = %#v", body)
	}
}

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

func TestHandleGetGPUHistory(t *testing.T) {
	now := time.Now().UTC()
	h := NewMetricsHandler(&config.Config{}, handlermocks.NewMonitorQuerier().WithGPUHistory(&models.GPUHistory{
		Start: now.Add(-time.Hour), End: now,
		VRAMUsedMB: []models.GPUHistoryPoint{{Timestamp: now, Value: 2048}},
	}), slog.Default())
	w := httptest.NewRecorder()
	h.HandleGetGPUHistory(w, httptest.NewRequest(http.MethodGet, "/api/v1/forensics/gpu?window=1h", nil))
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if points, ok := body["vram_used_mb"].([]interface{}); !ok || len(points) != 1 {
		t.Fatalf("gpu history = %#v", body)
	}
}

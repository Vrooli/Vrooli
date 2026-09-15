package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/devicegraph"
)

type restDeviceGraphProvider struct{}

func (restDeviceGraphProvider) DeviceGraph(context.Context) devicegraph.Graph {
	rungs := make(map[devicegraph.Rung]devicegraph.RungState, len(devicegraph.Rungs))
	for _, rung := range devicegraph.Rungs {
		rungs[rung] = devicegraph.RungState{Rung: rung, State: devicegraph.StateMeasured, ObservedAt: time.Now()}
	}
	return devicegraph.Graph{
		CollectedAt: time.Date(2026, 8, 26, 22, 0, 0, 0, time.UTC),
		Platform:    "darwin",
		Devices: []devicegraph.Device{{
			ID: "block:disk0", Class: devicegraph.ClassBlockDevice, Model: "APPLE SSD",
			Readings: map[string]float64{"capacity_bytes": 251000193024},
			Rungs:    rungs,
		}},
	}
}

func TestHandleGetDeviceGraphUsesLowerCamelRESTContract(t *testing.T) {
	handler := NewDeviceGraphHandler(restDeviceGraphProvider{}, nil)
	recorder := httptest.NewRecorder()
	handler.HandleGetDeviceGraph(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/devices", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"collectedAt":"2026-08-26T22:00:00Z"`) {
		t.Fatalf("response omitted lowerCamel collectedAt: %s", body)
	}
	if !strings.Contains(body, `"capacity_bytes":251000193024`) {
		t.Fatalf("response omitted device reading: %s", body)
	}
	if strings.Contains(body, `"collected_at"`) {
		t.Fatalf("response used proto field names: %s", body)
	}
}

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
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

func TestProtoGraphPreservesCompositeFunctionsInStableDeviceAttributes(t *testing.T) {
	rungs := make(map[devicegraph.Rung]devicegraph.RungState, len(devicegraph.Rungs))
	for _, rung := range devicegraph.Rungs {
		rungs[rung] = devicegraph.RungState{Rung: rung, State: devicegraph.StateMeasured, ObservedAt: time.Now().UTC()}
	}
	graph := devicegraph.Graph{Platform: "linux", Devices: []devicegraph.Device{{
		ID: "usb:1-6", Class: devicegraph.ClassUSBDevice, Rungs: rungs,
		Attributes: map[string]string{
			"readiness_firmware_state":  hostinventory.DeviceReadinessUnknown,
			"readiness_interface_state": hostinventory.DeviceReadinessNotApplicable,
			"readiness_function_state":  hostinventory.DeviceReadinessUnverified,
		},
		Functions: []hostinventory.DeviceFunction{
			{ID: "usb:1-6:1.1", Class: "storage", BoundDriver: "usb-storage", BindingState: hostinventory.DeviceBindingBound},
			{ID: "usb:1-6:1.0", Class: "wireless", BindingState: hostinventory.DeviceBindingUnbound},
		},
	}}}
	wire := protoGraph(graph)
	if !wire.GetAvailable() || len(wire.GetDevices()) != 1 {
		t.Fatalf("device graph was not published: %+v", wire)
	}
	want := `[{"id":"usb:1-6:1.0","class":"wireless","bindingState":"unbound"},{"id":"usb:1-6:1.1","class":"storage","boundDriver":"usb-storage","bindingState":"bound"}]`
	if got := wire.GetDevices()[0].GetAttributes()["native_functions_json"]; got != want {
		t.Fatalf("published composite functions = %q, want stable ordered JSON %q", got, want)
	}
	for key, expected := range graph.Devices[0].Attributes {
		if got := wire.GetDevices()[0].GetAttributes()[key]; got != expected {
			t.Errorf("published %s = %q, want %q", key, got, expected)
		}
	}
	if graph.Devices[0].Functions[0].ID != "usb:1-6:1.1" {
		t.Fatalf("serialization mutated source function order: %+v", graph.Devices[0].Functions)
	}
}

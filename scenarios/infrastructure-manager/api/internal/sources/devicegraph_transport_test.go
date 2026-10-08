package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	devicegraphpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph"
	devicegraphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph/devicegraphconnect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeOwner struct {
	graph *devicegraphpb.Graph
	err   error
}

func (f fakeOwner) GetDeviceGraph(context.Context, *connect.Request[devicegraphpb.GetDeviceGraphRequest]) (*connect.Response[devicegraphpb.GetDeviceGraphResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&devicegraphpb.GetDeviceGraphResponse{Graph: f.graph}), nil
}

func ownerServer(t *testing.T, owner fakeOwner) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := devicegraphconnect.NewDeviceGraphServiceHandler(owner)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// TestConnectTransportPreservesEveryRungGrade is the contract that makes the
// SB9-SB13 join trustworthy. All four grades must survive the wire as
// themselves: `unavailable` (no mechanism on this host), `unmeasurable` (the
// host refused), `not_applicable` (meaningless for the class) and `measured`
// are four different facts, and collapsing any pair reports hardware nobody
// can read as hardware that is fine.
func TestConnectTransportPreservesEveryRungGrade(t *testing.T) {
	observed := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	server := ownerServer(t, fakeOwner{graph: &devicegraphpb.Graph{
		Available:   true,
		Platform:    "linux",
		CollectedAt: timestamppb.New(observed),
		Devices: []*devicegraphpb.Device{{
			Id: "block:nvme0n1", Class: "block-device", ParentId: "pci:0000:01:00.0",
			Vendor: "ACME", Model: "SSD", Driver: "nvme", SysPath: "/sys/block/nvme0n1",
			Attributes: map[string]string{"transport": "nvme", "kernel_name": "nvme0n1"},
			Readings:   map[string]float64{"capacity_bytes": 1024},
			Rungs: []*devicegraphpb.RungState{
				{Rung: devicegraphpb.Rung_RUNG_IDENTITY, Grade: devicegraphpb.RungGrade_RUNG_GRADE_MEASURED, Mechanism: "sysfs", ObservedAt: timestamppb.New(observed)},
				{Rung: devicegraphpb.Rung_RUNG_TELEMETRY, Grade: devicegraphpb.RungGrade_RUNG_GRADE_MEASURED, Mechanism: "sysfs", ObservedAt: timestamppb.New(observed)},
				{Rung: devicegraphpb.Rung_RUNG_EVIDENCE, Grade: devicegraphpb.RungGrade_RUNG_GRADE_NOT_APPLICABLE, Reason: "nothing to retain", ObservedAt: timestamppb.New(observed)},
				{Rung: devicegraphpb.Rung_RUNG_CONTROL, Grade: devicegraphpb.RungGrade_RUNG_GRADE_UNMEASURABLE, Reason: "permission denied", Mechanism: "smartctl", Remediation: "grant CAP_SYS_RAWIO", ObservedAt: timestamppb.New(observed)},
				{Rung: devicegraphpb.Rung_RUNG_ANTICIPATION, Grade: devicegraphpb.RungGrade_RUNG_GRADE_UNAVAILABLE, Reason: "smartctl is not installed", Mechanism: "smartctl", ObservedAt: timestamppb.New(observed)},
			},
		}},
		Subsystems: []*devicegraphpb.Subsystem{{
			Name:       "memory",
			Attributes: map[string]string{"registered_controllers": "0"},
			Rungs: []*devicegraphpb.RungState{
				{Rung: devicegraphpb.Rung_RUNG_IDENTITY, Grade: devicegraphpb.RungGrade_RUNG_GRADE_UNMEASURABLE, Reason: "no EDAC controller registered", ObservedAt: timestamppb.New(observed)},
			},
		}},
		VirtualNetworkInterfaces: []string{"lo", "docker0"},
	}})

	graph, err := ConnectDeviceGraphTransport{HTTP: server.Client()}.DeviceGraph(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(graph.Devices))
	}
	device := graph.Devices[0]
	for _, field := range []struct{ name, got, want string }{
		{"id", device.ID, "block:nvme0n1"},
		{"class", device.Class, "block-device"},
		{"parent", device.ParentID, "pci:0000:01:00.0"},
		{"vendor", device.Vendor, "ACME"},
		{"model", device.Model, "SSD"},
		{"driver", device.Driver, "nvme"},
		{"sys_path", device.SysPath, "/sys/block/nvme0n1"},
	} {
		if field.got != field.want {
			t.Errorf("device %s = %q, want %q", field.name, field.got, field.want)
		}
	}
	if device.Attributes["kernel_name"] != "nvme0n1" {
		t.Errorf("kernel node name was dropped: %v", device.Attributes)
	}
	if device.Readings["capacity_bytes"] != 1024 {
		t.Errorf("readings were dropped: %v", device.Readings)
	}

	wantGrades := map[string]string{
		"identity": "measured", "telemetry": "measured", "evidence": "not_applicable",
		"control": "unmeasurable", "anticipation": "unavailable",
	}
	if len(device.Rungs) != len(wantGrades) {
		t.Fatalf("got %d rungs, want %d", len(device.Rungs), len(wantGrades))
	}
	for rung, want := range wantGrades {
		state, ok := device.Rungs[rung]
		if !ok {
			t.Errorf("rung %s was dropped in transit", rung)
			continue
		}
		if state.State != want {
			t.Errorf("rung %s = %q, want %q", rung, state.State, want)
		}
	}
	if device.Rungs["control"].Reason != "permission denied" {
		t.Error("an unmeasurable rung arrived without its reason; the gap is now unexplained")
	}
	if device.Rungs["control"].Remediation != "grant CAP_SYS_RAWIO" {
		t.Error("the remediation was dropped; the gap now names no fix")
	}
	if len(graph.Subsystems) != 1 || graph.Subsystems[0].Attributes["registered_controllers"] != "0" {
		t.Errorf("subsystem attributes were dropped: %+v", graph.Subsystems)
	}
	if len(graph.VirtualNetworkInterfaces) != 2 {
		t.Error("virtual interfaces were dropped; their exclusion is now silent instead of visible")
	}
}

// TestConnectTransportRefusesAnUnavailableGraph pins that the owner's own
// failure-to-observe is carried through as a failure, not as a graph. An empty
// graph served OK is a claim that the host has no hardware.
func TestConnectTransportRefusesAnUnavailableGraph(t *testing.T) {
	server := ownerServer(t, fakeOwner{graph: &devicegraphpb.Graph{
		Available:         false,
		UnavailableReason: "the device-graph walk produced neither a device nor a graded subsystem",
	}})
	graph, err := ConnectDeviceGraphTransport{HTTP: server.Client()}.DeviceGraph(context.Background(), server.URL)
	if err == nil {
		t.Fatal("an unavailable graph was returned as a graph")
	}
	if len(graph.Devices) != 0 {
		t.Fatal("a failed read returned device content")
	}
	if !strings.Contains(err.Error(), "neither a device nor a graded subsystem") {
		t.Errorf("the error %q loses the owner's reason", err)
	}
}

// TestConnectTransportNeverReadsAnUnsetGradeAsMeasured guards the wire's zero
// value. A grade nobody set must not become a healthy reading.
func TestConnectTransportNeverReadsAnUnsetGradeAsMeasured(t *testing.T) {
	server := ownerServer(t, fakeOwner{graph: &devicegraphpb.Graph{
		Available: true,
		Devices: []*devicegraphpb.Device{{
			Id: "pci:0000:02:00.0", Class: "pci-device",
			Rungs: []*devicegraphpb.RungState{
				{Rung: devicegraphpb.Rung_RUNG_IDENTITY},
			},
		}},
	}})
	graph, err := ConnectDeviceGraphTransport{HTTP: server.Client()}.DeviceGraph(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got := graph.Devices[0].Rungs["identity"].State; got != "unmeasurable" {
		t.Fatalf("an unset grade arrived as %q, want unmeasurable", got)
	}
}

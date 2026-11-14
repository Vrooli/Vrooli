package devicegraph

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestNativeBusProjectionPreservesSharedIdentityBindingAndCoverage(t *testing.T) {
	observedAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	env := Env{NativeDevices: func(context.Context) hostinventory.NativeDeviceObservation {
		return hostinventory.NativeDeviceObservation{
			Mechanism: "fixture SetupAPI",
			ProbeStatus: map[string]hostinventory.IntegrityProbeState{
				"pciDevices": hostinventory.IntegrityProbeOK,
				"usbDevices": hostinventory.IntegrityProbeOK,
				"socDevices": hostinventory.IntegrityProbeOK,
			},
			Devices: []hostinventory.DeviceInfo{
				{
					ID: "pnp:PCI\\VEN_14C3&DEV_7927", BusType: "pci", Address: "PCI\\VEN_14C3&DEV_7927",
					Class: "network controller", VendorID: "14c3", DeviceID: "7927", BindingState: hostinventory.DeviceBindingUnbound,
					Readiness: hostinventory.DeviceReadiness{
						FirmwareState: hostinventory.DeviceReadinessUnknown, FirmwareReason: "no direct firmware observation",
						InterfaceState: hostinventory.DeviceReadinessAbsent, InterfaceReason: "no interface exposed",
						FunctionState: hostinventory.DeviceReadinessUnverified, FunctionReason: "function not verified",
						InterfaceNames: []string{"wlan0"}, InterfaceStatus: map[string]string{"wlan0": "down"},
					},
					Functions: []hostinventory.DeviceFunction{
						{ID: "usb:1-6:1.1", Class: "storage", BoundDriver: "usb-storage", BindingState: hostinventory.DeviceBindingBound},
						{ID: "usb:1-6:1.0", Class: "wireless", Modalias: "usb:v0B05p1A2B", BindingState: hostinventory.DeviceBindingUnbound},
					},
				},
				{ID: "pnp:ACPI\\ACME0001", BusType: "acpi", Class: "system peripheral", BindingState: hostinventory.DeviceBindingBound},
			},
		}
	}}
	graph := Graph{CollectedAt: observedAt}
	b := newBuilder(env.normalized(), &graph, grader{at: observedAt})
	collectNativeBusDevices(context.Background(), b)

	device, ok := graph.DeviceByID("pnp:PCI\\VEN_14C3&DEV_7927")
	if !ok {
		t.Fatal("shared native identity was not projected")
	}
	if device.Class != ClassPCIDevice {
		t.Fatalf("class = %q, want PCI controller (interface telemetry is a separate graph row)", device.Class)
	}
	if device.Attributes["binding_state"] != hostinventory.DeviceBindingUnbound {
		t.Fatalf("binding state = %q, want shared unbound fact", device.Attributes["binding_state"])
	}
	for key, want := range map[string]string{
		"readiness_firmware_state":   hostinventory.DeviceReadinessUnknown,
		"readiness_firmware_reason":  "no direct firmware observation",
		"readiness_interface_state":  hostinventory.DeviceReadinessAbsent,
		"readiness_interface_reason": "no interface exposed",
		"readiness_interface_names":  `["wlan0"]`,
		"readiness_interface_status": `{"wlan0":"down"}`,
		"readiness_function_state":   hostinventory.DeviceReadinessUnverified,
		"readiness_function_reason":  "function not verified",
	} {
		if got := device.Attributes[key]; got != want {
			t.Errorf("shared readiness attribute %s = %q, want %q", key, got, want)
		}
	}
	if len(device.Functions) != 2 || device.Functions[0].ID != "usb:1-6:1.1" || device.Functions[1].ID != "usb:1-6:1.0" {
		t.Fatalf("shared composite functions = %+v, want both source records preserved", device.Functions)
	}
	platformDevice, ok := graph.DeviceByID("pnp:ACPI\\ACME0001")
	if !ok || platformDevice.Class != ClassPlatformDevice {
		t.Fatalf("ACPI device = %+v, found=%t; want a platform device, not a PCI device", platformDevice, ok)
	}
	if state := device.Rungs[RungControl]; state.State != StateUnmeasurable || state.Remediation == "" {
		t.Fatalf("control rung = %+v, want actionable unmeasurable state", state)
	}
	for _, name := range []string{"pci-bus", "usb-bus"} {
		subsystem, ok := graph.SubsystemByName(name)
		if !ok || subsystem.Rungs[RungIdentity].State != StateMeasured {
			t.Fatalf("successful %s coverage = %+v, want measured including successful-empty", name, subsystem)
		}
	}
}

func TestNativeBusProjectionDoesNotTurnFailedOrUnsupportedEnumerationIntoAbsence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state hostinventory.IntegrityProbeState
		want  State
	}{
		{name: "failed", state: hostinventory.IntegrityProbeFailed, want: StateUnmeasurable},
		{name: "partial", state: hostinventory.IntegrityProbeDegraded, want: StateUnmeasurable},
		{name: "unsupported", state: hostinventory.IntegrityProbeUnsupported, want: StateUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := time.Now().UTC()
			env := Env{NativeDevices: func(context.Context) hostinventory.NativeDeviceObservation {
				return hostinventory.NativeDeviceObservation{ProbeStatus: map[string]hostinventory.IntegrityProbeState{
					"pciDevices": tc.state, "usbDevices": hostinventory.IntegrityProbeOK,
				}, ProbeErrors: map[string]string{"pciDevices": "fixture failure"}}
			}}
			graph := Graph{}
			b := newBuilder(env.normalized(), &graph, grader{at: at})
			collectNativeBusDevices(context.Background(), b)
			pci, _ := graph.SubsystemByName("pci-bus")
			if got := pci.Rungs[RungIdentity].State; got != tc.want {
				t.Fatalf("PCI identity state = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPlatformJSONCommandsFailClosedWhenCommandReturnsValidJSONAndError(t *testing.T) {
	runErr := context.DeadlineExceeded
	env := Env{Run: func(context.Context, time.Duration, string, ...string) ([]byte, error) {
		return []byte(`{"SPPCIDataType":[]}`), runErr
	}}
	if _, err := runDarwinProfiler(context.Background(), env.normalized(), "SPPCIDataType"); err == nil {
		t.Fatal("macOS accepted valid JSON from a failed command")
	}
	if _, err := runWindowsJSON(context.Background(), env.normalized(), "powershell", `Get-PnpDevice`); err == nil {
		t.Fatal("Windows accepted valid JSON from a failed command")
	}
}

func TestDarwinNetworkProbeRetainsButGradesPartialPortOutput(t *testing.T) {
	at := time.Now().UTC()
	env := Env{
		LookPath: func(name string) (string, error) { return name, nil },
		Run: func(_ context.Context, _ time.Duration, name string, _ ...string) ([]byte, error) {
			if name == darwinNetworkTool {
				return []byte("Hardware Port: Wi-Fi\nDevice: en0\n"), context.DeadlineExceeded
			}
			return []byte("Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll\nen0 1500 <Link> xx 10 0 1000 20 0 2000 0\n"), nil
		},
	}
	graph := Graph{}
	b := newBuilder(env.normalized(), &graph, grader{at: at})
	collectDarwinNetwork(context.Background(), b)
	if len(graph.Devices) != 1 || graph.Devices[0].ID != "net:en0" {
		t.Fatalf("partial interface output was not retained: %+v", graph.Devices)
	}
	subsystem, ok := graph.SubsystemByName("network-interfaces")
	if !ok || subsystem.Rungs[RungIdentity].State != StateUnmeasurable || subsystem.Attributes["enumeration"] != "partial" {
		t.Fatalf("partial interface enumeration = %+v, found=%t; want explicit partial coverage", subsystem, ok)
	}
}

func TestDarwinNetworkCountersRejectOutputFromFailedCommand(t *testing.T) {
	at := time.Now().UTC()
	env := Env{
		LookPath: func(name string) (string, error) { return name, nil },
		Run: func(_ context.Context, _ time.Duration, name string, _ ...string) ([]byte, error) {
			if name == darwinNetworkTool {
				return []byte("Hardware Port: Wi-Fi\nDevice: en0\n"), nil
			}
			return []byte("Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll\nen0 1500 <Link> xx 10 0 1000 20 0 2000 0\n"), context.DeadlineExceeded
		},
	}
	graph := Graph{}
	b := newBuilder(env.normalized(), &graph, grader{at: at})
	collectDarwinNetwork(context.Background(), b)
	device, ok := graph.DeviceByID("net:en0")
	if !ok {
		t.Fatal("hardware port was not retained")
	}
	if state := device.Rungs[RungTelemetry]; state.State != StateUnmeasurable || len(device.Readings) != 0 {
		t.Fatalf("failed netstat output was accepted as telemetry: state=%+v readings=%v", state, device.Readings)
	}
}

func TestDarwinStorageDistinguishesSuccessfulEmptyFromFailedProbe(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want State
	}{
		{name: "successful empty", want: StateMeasured},
		{name: "failed", err: context.DeadlineExceeded, want: StateUnmeasurable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := time.Now().UTC()
			env := Env{
				LookPath: func(name string) (string, error) { return name, nil },
				Run: func(_ context.Context, _ time.Duration, _ string, args ...string) ([]byte, error) {
					return []byte(`{"` + args[len(args)-1] + `":[]}`), tc.err
				},
			}
			graph := Graph{}
			b := newBuilder(env.normalized(), &graph, grader{at: at})
			collectDarwinStorage(context.Background(), b)
			subsystem, ok := graph.SubsystemByName("block-storage")
			if !ok || subsystem.Rungs[RungIdentity].State != tc.want {
				t.Fatalf("storage inventory = %+v, found=%t; want identity %s", subsystem, ok, tc.want)
			}
		})
	}
}

func TestWindowsStorageProbeFailureIsUnmeasurable(t *testing.T) {
	at := time.Now().UTC()
	env := Env{Run: func(context.Context, time.Duration, string, ...string) ([]byte, error) {
		return []byte(`{"DeviceId":"0"}`), context.DeadlineExceeded
	}}
	graph := Graph{}
	b := newBuilder(env.normalized(), &graph, grader{at: at})
	collectWindowsStorage(context.Background(), b, "powershell")
	subsystem, ok := graph.SubsystemByName("block-storage")
	if !ok || subsystem.Rungs[RungIdentity].State != StateUnmeasurable {
		t.Fatalf("failed storage command = %+v, found=%t; want unmeasurable", subsystem, ok)
	}
}

func TestWindowsSuccessfulEmptyStorageAndNetworkEnumerationsAreMeasured(t *testing.T) {
	for _, tc := range []struct {
		name  string
		probe func(context.Context, *builder)
		want  string
	}{
		{name: "storage", probe: func(ctx context.Context, b *builder) { collectWindowsStorage(ctx, b, "powershell") }, want: "block-storage"},
		{name: "network", probe: func(ctx context.Context, b *builder) { collectWindowsNetwork(ctx, b, "powershell") }, want: "network-interfaces"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := time.Now().UTC()
			env := Env{Run: func(context.Context, time.Duration, string, ...string) ([]byte, error) {
				return []byte("[]"), nil
			}}
			graph := Graph{}
			b := newBuilder(env.normalized(), &graph, grader{at: at})
			tc.probe(context.Background(), b)
			subsystem, ok := graph.SubsystemByName(tc.want)
			if !ok || subsystem.Rungs[RungIdentity].State != StateMeasured {
				t.Fatalf("successful empty Windows %s enumeration = %+v, found=%t; want measured", tc.name, subsystem, ok)
			}
		})
	}
}

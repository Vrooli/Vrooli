package hostinventory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxBluetoothReadinessSeparatesHCIFromOSController(t *testing.T) {
	for _, tc := range []struct{ name, payload, commandErr, wantStatus, wantState string }{
		{"missing OS controller", `{"type":"a{oa{sa{sv}}}","data":[{}]}`, "", "controller_absent", DeviceReadinessInitialized},
		{"powered OS controller", `{"type":"a{oa{sa{sv}}}","data":[{"/org/bluez/hci0":{"org.bluez.Adapter1":{"Powered":{"type":"b","data":true}}}}]}`, "", "controller_powered", DeviceReadinessInitialized},
		{"disabled OS controller", `{"type":"a{oa{sa{sv}}}","data":[{"/org/bluez/hci0":{"org.bluez.Adapter1":{"Powered":{"type":"b","data":false}}}}]}`, "", "controller_disabled", DeviceReadinessDisabled},
		{"failed probe", "", "service unavailable", "controller_unknown", DeviceReadinessInitialized},
		{"malformed probe", `{"type":"a{oa{sa{sv}}}","data":[null]}`, "", "controller_unknown", DeviceReadinessInitialized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeIntegrityFixture(t, filepath.Join(root, "1-6:1.0", "bluetooth", "hci0", "rfkill0"), map[string]string{"soft": "0", "hard": "0"})
			devices := []DeviceInfo{{SysPath: root, Functions: []DeviceFunction{{ID: "1-6:1.0", Modalias: "usb:v0489pE13AicE0isc01ip01in00", BindingState: DeviceBindingUnbound}}}}
			if err := observeLinuxDeviceInterfaces(nativeIntegrityFiles{}, &devices[0]); err != nil {
				t.Fatal(err)
			}
			calls := 0
			observeLinuxBluetoothControllers(context.Background(), devices, func(ctx context.Context, name string, args ...string) ([]byte, error) {
				calls++
				if name != "/usr/bin/busctl" || !strings.Contains(strings.Join(args, " "), "--auto-start=no --allow-interactive-authorization=no") || args[len(args)-1] != "GetManagedObjects" {
					t.Fatalf("unexpected passive call: %s %v", name, args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("probe has no deadline")
				}
				if tc.commandErr != "" {
					return nil, errors.New(tc.commandErr)
				}
				return []byte(tc.payload), nil
			})
			r := devices[0].Readiness
			if calls != 1 || r.InterfaceState != tc.wantState || !strings.Contains(r.InterfaceStatus["hci0"], tc.wantStatus) || r.FunctionState != DeviceReadinessUnverified || r.FirmwareState != DeviceReadinessUnknown {
				t.Fatalf("readiness=%+v calls=%d", r, calls)
			}
		})
	}
}

func TestLinuxBluetoothSysfsAbsenceAndErrorsStayDistinct(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(*testing.T, string)
		want    string
		wantErr bool
	}{
		{name: "no HCI", want: DeviceReadinessAbsent},
		{name: "empty HCI directory", setup: func(t *testing.T, p string) {
			if err := os.MkdirAll(filepath.Join(p, "1-6:1.0", "bluetooth"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, want: DeviceReadinessAbsent},
		{name: "failed HCI enumeration", setup: func(t *testing.T, p string) {
			writeIntegrityFixture(t, filepath.Join(p, "1-6:1.0"), map[string]string{"bluetooth": "not a directory"})
		}, want: DeviceReadinessUnknown, wantErr: true},
		{name: "soft blocked", setup: func(t *testing.T, p string) {
			writeIntegrityFixture(t, filepath.Join(p, "1-6:1.0", "bluetooth", "hci0", "rfkill0"), map[string]string{"soft": "1", "hard": "0"})
		}, want: DeviceReadinessDisabled},
		{name: "failed rfkill read", setup: func(t *testing.T, p string) {
			writeIntegrityFixture(t, filepath.Join(p, "1-6:1.0", "bluetooth", "hci0", "rfkill0"), map[string]string{"soft": "0"})
		}, want: DeviceReadinessUnknown, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, p)
			}
			device := DeviceInfo{SysPath: p, Functions: []DeviceFunction{{ID: "1-6:1.0", BoundDriver: "btusb"}}}
			err := observeLinuxDeviceInterfaces(nativeIntegrityFiles{}, &device)
			if (err != nil) != tc.wantErr || device.Readiness.InterfaceState != tc.want {
				t.Fatalf("readiness=%+v error=%v", device.Readiness, err)
			}
		})
	}
}

func TestBluetoothSnapshotRejectsAmbiguityAndRetainsPeerProof(t *testing.T) {
	for _, raw := range []string{`{"type":"a{oa{sa{sv}}}","data":[{"/org/bluez/hci0":{},"/org/bluez/hci0":{}}]}`, `{"type":"a{oa{sa{sv}}}","data":[["/org/bluez/hci0",{},"/org/bluez/hci0",{}]]}`, `{"type":"wrong","data":[{}]}`, strings.Repeat("x", (64<<10)+1)} {
		if _, err := ParseLinuxBluetoothControllers([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid payload: %.80s", raw)
		}
	}
	raw := `{"type":"a{oa{sa{sv}}}","data":[["/org/bluez/hci0",["org.bluez.Adapter1",["Powered",{"type":"b","data":true}]],"/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF",["org.bluez.Device1",["Paired",{"type":"b","data":true},"Connected",{"type":"b","data":true}]]]]}`
	controllers, err := ParseLinuxBluetoothControllers([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got := controllers["hci0"]; !got.PowerKnown || !got.Powered || !got.PeerConnected {
		t.Fatalf("lost peer evidence: %+v", got)
	}
}

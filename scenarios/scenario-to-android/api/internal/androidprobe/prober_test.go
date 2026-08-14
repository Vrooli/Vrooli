package androidprobe

import (
	"context"
	"errors"
	"testing"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type fakeDevices struct {
	items []DeviceObservation
}

func (f fakeDevices) List(context.Context) ([]DeviceObservation, error) { return f.items, nil }

func TestProbeReportsLocalEmulatorAndPhysicalDevice(t *testing.T) {
	lookPath := func(name string) (string, error) { return "/bin/" + name, nil }
	p := Prober{
		LookPath: lookPath,
		Getenv: func(name string) string {
			if name == "ANDROID_SDK_ROOT" {
				return "/sdk"
			}
			return ""
		},
		Run: func(context.Context, string, ...string) ([]byte, error) { return []byte("Pixel_API_36\n"), nil },
		KVM: func() (bool, bool, string) { return true, true, "" },
		Devices: fakeDevices{items: []DeviceObservation{{
			ID: "SM_A037U", Label: "Galaxy A03s", NodeID: "phone-node", Serial: "SM_A037U",
			OS: "Android 14", Architecture: "arm64", Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge, ID: "usb", Available: true},
			Capabilities: []string{deliveryramp.CapabilityAndroidWebView, deliveryramp.CapabilityScreenRecording, deliveryramp.CapabilityDeviceControl}, Available: true,
		}}},
		Now: func() time.Time { return time.Unix(10, 0) },
	}
	inventory, err := p.Probe(context.Background(), deliveryramp.ProbeRequest{RequiredCapability: []string{deliveryramp.CapabilityDeviceControl}})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Targets) != 2 {
		t.Fatalf("targets = %d, want local emulator and physical device", len(inventory.Targets))
	}
	if !inventory.Targets[0].Available || inventory.Targets[0].Transport.Kind != deliveryramp.TransportLocal {
		t.Fatalf("local target = %#v", inventory.Targets[0])
	}
	if inventory.Targets[1].ID != "SM_A037U" || inventory.Targets[1].Transport.Kind != deliveryramp.TransportBridge {
		t.Fatalf("physical target = %#v", inventory.Targets[1])
	}
}

func TestProbeFailsClosedWithoutKVMAndNamesNextAction(t *testing.T) {
	p := Prober{
		LookPath: func(name string) (string, error) { return "/bin/" + name, nil },
		Run:      func(context.Context, string, ...string) ([]byte, error) { return []byte("Pixel_API_36\n"), nil },
		KVM:      func() (bool, bool, string) { return true, false, "permission denied" },
		Devices:  fakeDevices{items: nil},
	}
	inventory, err := p.Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Targets[0].Available {
		t.Fatal("unaccelerated emulator was reported available")
	}
	if inventory.Targets[0].MissingCapability == "" || inventory.Targets[0].NextAction == "" {
		t.Fatalf("unavailable target lacks remediation: %#v", inventory.Targets[0])
	}
}

func TestProbePropagatesDeviceInventoryFailure(t *testing.T) {
	p := Prober{
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
		Devices:  failingDevices{},
	}
	_, err := p.Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err == nil {
		t.Fatal("expected inventory error")
	}
}

type failingDevices struct{}

func (failingDevices) List(context.Context) ([]DeviceObservation, error) {
	return nil, errors.New("device-control unavailable")
}

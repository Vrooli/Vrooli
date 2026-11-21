package targets

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices/devices_v1connect"
)

type listedDeviceService struct {
	devicesconnect.UnimplementedDeviceServiceHandler
	devices []*devicesv1.Device
}

func (s listedDeviceService) ListDevices(context.Context, *connect.Request[devicesv1.ListDevicesRequest]) (*connect.Response[devicesv1.ListDevicesResponse], error) {
	return connect.NewResponse(&devicesv1.ListDevicesResponse{Devices: s.devices}), nil
}

func TestDeviceControlInventoryUsesActionableProfileAndPreservesIdentity(t *testing.T) {
	_, handler := devicesconnect.NewDeviceServiceHandler(listedDeviceService{devices: []*devicesv1.Device{{
		Id: "android-6affcac615a281a2", Name: "Moto", Kind: "physical", Serial: "serial-moto",
		Transport: "wireless", Status: "available", Health: "available",
		Capabilities: []*devicesv1.CapabilitySnapshot{{Name: "device-control", Status: "available"}},
		Transports: []*devicesv1.TransportProfile{
			{StrategyId: "android-adb", Name: "wireless", Health: "available"},
			{StrategyId: "android-adb", Name: "usb", Health: "available"},
		},
	}}})
	server := httptest.NewServer(handler)
	defer server.Close()

	observations, err := (DeviceControlInventory{Resolve: func(context.Context) (string, error) { return server.URL, nil }, Client: server.Client()}).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(observations))
	}
	got := observations[0]
	if got.ID != "android-6affcac615a281a2" || got.Serial != "serial-moto" {
		t.Fatalf("identity = id %q serial %q, want stable device and serial identities", got.ID, got.Serial)
	}
	if got.ADBTransport != "usb" {
		t.Fatalf("execution transport = %q, want observed USB profile", got.ADBTransport)
	}
	if got.Transport.ID != "serial-moto" {
		t.Fatalf("reported transport identity = %q, want serial-moto", got.Transport.ID)
	}
	if !strings.Contains(got.Reason, `selected transport "wireless"`) {
		t.Fatalf("reason %q does not retain the selected transport context", got.Reason)
	}
	if len(got.Capabilities) != 1 || got.Capabilities[0] != "device-control" {
		t.Fatalf("capabilities = %#v, want retained device-control capability", got.Capabilities)
	}
}

func TestActionableADBTransportPrefersObservedUSBForWirelessSelection(t *testing.T) {
	profiles := []*devicesv1.TransportProfile{
		{StrategyId: "android-adb", Name: "wireless", Health: "available"},
		{StrategyId: "android-adb", Name: "usb", Health: "available"},
	}

	if got := actionableADBTransport("wireless", profiles); got != "usb" {
		t.Fatalf("actionableADBTransport(wireless) = %q, want usb", got)
	}
}

func TestActionableADBTransportRetainsWirelessWithoutObservedUSB(t *testing.T) {
	profiles := []*devicesv1.TransportProfile{
		{StrategyId: "android-adb", Name: "wireless", Health: "available"},
	}

	if got := actionableADBTransport("wireless", profiles); got != "wireless" {
		t.Fatalf("actionableADBTransport(wireless) = %q, want wireless", got)
	}
}

func TestActionableADBTransportRetainsWirelessWhenUSBIsUnavailable(t *testing.T) {
	profiles := []*devicesv1.TransportProfile{
		{StrategyId: "android-adb", Name: "wireless", Health: "available"},
		{StrategyId: "android-adb", Name: "usb", Health: "unavailable"},
	}

	if got := actionableADBTransport("wireless", profiles); got != "wireless" {
		t.Fatalf("actionableADBTransport(wireless) = %q, want wireless", got)
	}
}

func TestActionableADBTransportRequiresAndroidADBStrategy(t *testing.T) {
	profiles := []*devicesv1.TransportProfile{
		{StrategyId: "other-strategy", Name: "usb", Health: "available"},
	}

	if got := actionableADBTransport("wireless", profiles); got != "wireless" {
		t.Fatalf("actionableADBTransport(wireless) = %q, want wireless", got)
	}
}

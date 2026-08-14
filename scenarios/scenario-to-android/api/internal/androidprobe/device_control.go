package androidprobe

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices/devices_v1connect"
)

// DeviceControlInventory is the provider-neutral read adapter used by the
// Android ramp. Leases and execution remain device-control responsibilities;
// this client only projects its typed inventory into ramp observations.
type DeviceControlInventory struct {
	Resolve func(context.Context) (string, error)
	Client  *http.Client
}

func NewDeviceControlInventory() DeviceControlInventory {
	return DeviceControlInventory{Resolve: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "device-control")
	}, Client: http.DefaultClient}
}

func (c DeviceControlInventory) List(ctx context.Context) ([]DeviceObservation, error) {
	resolve := c.Resolve
	if resolve == nil {
		return nil, fmt.Errorf("device-control URL resolver is not configured")
	}
	baseURL, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve device-control URL: %w", err)
	}
	httpClient := c.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := devicesconnect.NewDeviceServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
	response, err := client.ListDevices(ctx, connect.NewRequest(&devicesv1.ListDevicesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list device-control devices: %w", err)
	}
	observations := make([]DeviceObservation, 0, len(response.Msg.Devices))
	for _, device := range response.Msg.Devices {
		if device == nil {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(device.Kind))
		if kind != "android" && kind != "physical" && kind != "emulator" {
			continue
		}
		capabilities := make([]string, 0, len(device.Capabilities))
		seenCapabilities := make(map[string]struct{}, len(device.Capabilities))
		available := strings.EqualFold(device.Status, "ready") || strings.EqualFold(device.Status, "available") || strings.EqualFold(device.Health, "healthy") || strings.EqualFold(device.Health, "available")
		for _, capability := range device.Capabilities {
			if strings.EqualFold(capability.Status, "available") {
				name := normalizeCapability(capability.Name)
				if name != "" {
					if _, seen := seenCapabilities[name]; !seen {
						seenCapabilities[name] = struct{}{}
						capabilities = append(capabilities, name)
					}
				}
			}
		}
		// This client talks to the device-control service on the serving host.
		// USB and wireless ADB are therefore both local execution transports;
		// bridge dispatch is represented by the validation matrix's bridge
		// executor, not inferred from a phone's wireless link.
		transport := deliveryramp.TransportLocal
		if strings.EqualFold(device.Transport, "bridge") || strings.EqualFold(device.Transport, "remote") {
			transport = deliveryramp.TransportBridge
		}
		observations = append(observations, DeviceObservation{
			ID: device.Id, Label: device.Name, NodeID: device.HostNodeId, Serial: device.Serial, ADBTransport: device.Transport,
			OS: "Android", Architecture: device.Model, Transport: deliveryramp.Transport{Kind: transport, ID: device.Serial, Available: available},
			Capabilities: capabilities, Available: available, Reason: firstReason(device.HealthReason, device.Status),
		})
	}
	return observations, nil
}

func normalizeCapability(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "webview-attach", "android-webview":
		return deliveryramp.CapabilityAndroidWebView
	case "screen-recording", "native-recording":
		return deliveryramp.CapabilityScreenRecording
	case "device-control", "input", "semantic-tree":
		return deliveryramp.CapabilityDeviceControl
	default:
		return strings.TrimSpace(raw)
	}
}

func firstReason(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "device-control Android target observed"
}

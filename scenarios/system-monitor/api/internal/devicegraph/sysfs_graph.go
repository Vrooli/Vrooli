package devicegraph

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// collectSysfsGraph is the complete Linux device-graph walk. It lives in an
// untagged file so every parser it drives can be exercised against a fixture
// sysfs root from any build target, and the platform dispatch file merely
// selects it.
//
// Order matters: buses are enumerated first so the devices that hang off them
// find a parent already in the index.
func collectSysfsGraph(ctx context.Context, b *builder) {
	collectPCIDevices(b)
	collectUSBDevices(b)
	projectLinuxBusReadiness(ctx, b)
	collectBlockDevices(ctx, b)
	collectNetworkInterfaces(b)
	collectThermalSensors(b)
	collectMemoryErrors(b)
}

func projectLinuxBusReadiness(ctx context.Context, b *builder) {
	devices := make([]hostinventory.DeviceInfo, 0, len(b.graph.Devices))
	indexes := make([]int, 0, len(b.graph.Devices))
	for i, device := range b.graph.Devices {
		bus, address, ok := strings.Cut(device.ID, ":")
		if !ok || address == "" || device.SysPath == "" || (bus != "pci" && bus != "usb") {
			continue
		}
		devices = append(devices, hostinventory.DeviceInfo{
			Address: address, SysPath: device.SysPath, Class: device.Attributes["pci_class_name"], Functions: device.Functions,
		})
		indexes = append(indexes, i)
	}
	hostinventory.ObserveLinuxDeviceReadiness(ctx, devices, func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return b.env.Run(ctx, time.Second, name, args...)
	})
	for i, index := range indexes {
		setNativeReadinessAttributes(&b.graph.Devices[index], devices[i].Readiness)
	}
}

package devicegraph

import "context"

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
	collectBlockDevices(ctx, b)
	collectNetworkInterfaces(b)
	collectThermalSensors(b)
	collectMemoryErrors(b)
}

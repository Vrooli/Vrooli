//go:build !linux && !darwin && !windows

package devicegraph

import "context"

// collectPlatformDevices reports the absence of a backend for this operating
// system. It deliberately publishes graded subsystems rather than an empty
// graph so no consumer can read silence as "no hardware problems".
func collectPlatformDevices(_ context.Context, b *builder) {
	unsupportedPlatform(b, "no device-graph backend is implemented for this operating system")
}

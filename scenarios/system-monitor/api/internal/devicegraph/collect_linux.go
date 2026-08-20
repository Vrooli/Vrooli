//go:build linux

package devicegraph

import "context"

// collectPlatformDevices drives the sysfs walk. Linux is the fully implemented
// backend: every rung in this package has a real Linux mechanism behind it.
func collectPlatformDevices(ctx context.Context, b *builder) {
	collectSysfsGraph(ctx, b)
}

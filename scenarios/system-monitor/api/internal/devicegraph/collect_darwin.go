//go:build darwin

package devicegraph

import "context"

// collectPlatformDevices drives the macOS command probes.
func collectPlatformDevices(ctx context.Context, b *builder) {
	collectDarwinGraph(ctx, b)
}

//go:build windows

package devicegraph

import "context"

// collectPlatformDevices drives the Windows PowerShell probes.
func collectPlatformDevices(ctx context.Context, b *builder) {
	collectWindowsGraph(ctx, b)
}

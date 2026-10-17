//go:build !linux && !darwin && !windows

package collectors

import "context"

func collectPlatformNetwork(context.Context, *NetworkCollector) platformNetworkReading {
	return networkUnsupported("network backend is unavailable on this operating system")
}

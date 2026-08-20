//go:build linux

package collectors

import "context"

func collectPlatformNetwork(ctx context.Context, c *NetworkCollector) platformNetworkReading {
	stats := c.getNetworkStats(ctx)
	return networkReading(map[string]interface{}{
		"tcp_connections": c.getTCPConnections(ctx),
		"tcp_states":      c.getTCPConnectionStates(ctx),
		"network_stats":   stats,
		"port_usage":      c.getPortUsage(ctx),
		"bandwidth":       c.calculateBandwidth(),
	}, "linux procfs and netlink")
}

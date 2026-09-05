//go:build linux

package collectors

import "context"

func collectPlatformNetwork(ctx context.Context, c *NetworkCollector) platformNetworkReading {
	stats := c.getNetworkStats(ctx)
	established := c.getTCPConnections(ctx)
	values := map[string]interface{}{
		"tcp_connections": established,
		"tcp_states":      c.getTCPConnectionStates(ctx),
		"network_stats":   stats,
		"port_usage":      c.getPortUsage(ctx),
		"bandwidth":       c.calculateBandwidth(),
	}
	// An aggregate connection count says something is wrong but never what. The
	// /proc walk that names the owner is too expensive to run every cycle, so it
	// runs only once the count is already alarming.
	if shouldAttributeSockets(established) {
		values["socket_owners"] = attributeSocketOwners(ctx, established, 10)
	}
	return networkReading(values, "linux procfs and netlink")
}

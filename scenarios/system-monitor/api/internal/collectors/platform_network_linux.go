//go:build linux

package collectors

import (
	"context"
	"time"
)

func collectPlatformNetwork(ctx context.Context, c *NetworkCollector) platformNetworkReading {
	stats := c.getNetworkStats(ctx)
	states, stateErr := readTCPStates(ctx)
	established := states["established"]
	if stateErr != nil {
		states = c.getTCPConnectionStates(ctx)
	}
	values := map[string]interface{}{
		"tcp_connections": established,
		"tcp_states":      states,
		"network_stats":   stats,
		"port_usage":      c.getPortUsage(ctx),
		"bandwidth":       c.calculateBandwidth(),
	}
	if interfaces, err := readNetDevInterfaces(ctx, c.rates); err == nil {
		values["interfaces"] = interfaces
		values["interface_counters_status"] = "measured"
	} else {
		values["interface_counters_status"] = "failed"
		values["interface_counters_reason"] = err.Error()
	}
	now := time.Now()
	for _, state := range []string{"established", "time_wait", "close_wait"} {
		if rate, measured := c.rates.observeGauge("tcp:"+state, uint64(maxInt(states[state], 0)), now); measured {
			values[state+"_rate_per_second"] = rate
			values[state+"_rate_status"] = "measured"
		} else {
			values[state+"_rate_status"] = "not_yet_sampled"
		}
	}
	// An aggregate connection count says something is wrong but never what. The
	// /proc walk that names the owner is too expensive to run every cycle, so it
	// runs only once the count is already alarming.
	if shouldAttributeSockets(established) {
		values["socket_owners"] = attributeSocketOwners(ctx, established, 10)
	}
	return networkReading(values, "linux procfs and netlink")
}

func maxInt(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

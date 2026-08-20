//go:build darwin

package collectors

import (
	"context"
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/unix"
)

func collectPlatformNetwork(_ context.Context, c *NetworkCollector) platformNetworkReading {
	interfaces, err := net.Interfaces()
	if err != nil {
		return platformNetworkReading{status: "failed", reason: fmt.Sprintf("enumerate interfaces: %v", err), provenance: "darwin getifaddrs"}
	}
	stats := emptyNetworkStats()
	seen := 0
	for _, iface := range interfaces {
		if iface.Index <= 0 {
			continue
		}
		raw, err := unix.SysctlRaw(fmt.Sprintf("net.link.generic.system.ifdata.%d", iface.Index))
		if err != nil {
			continue
		}
		data, ok := darwinIfData64(raw)
		if !ok {
			continue
		}
		seen++
		stats["bytes_recv"] = stats["bytes_recv"].(int64) + int64(data.Ibytes)
		stats["packets_recv"] = stats["packets_recv"].(int64) + int64(data.Ipackets)
		stats["errors_in"] = stats["errors_in"].(int64) + int64(data.Ierrors)
		stats["bytes_sent"] = stats["bytes_sent"].(int64) + int64(data.Obytes)
		stats["packets_sent"] = stats["packets_sent"].(int64) + int64(data.Opackets)
		stats["errors_out"] = stats["errors_out"].(int64) + int64(data.Oerrors)
		stats["dropped_in"] = stats["dropped_in"].(int64) + int64(data.Iqdrops)
	}
	if seen == 0 {
		return platformNetworkReading{status: "failed", reason: "darwin interface counters unavailable", provenance: "darwin sysctl if_data"}
	}
	stats["dropped_out"] = int64(0)
	return networkReading(map[string]interface{}{
		"network_stats": stats,
		"bandwidth":     bandwidthFromCounters(c, stats["bytes_recv"].(int64), stats["bytes_sent"].(int64)),
		"tcp_states": map[string]string{
			"status": "unsupported",
			"reason": "Darwin TCP PCB state enumeration is not enabled in this build",
		},
	}, "darwin sysctl if_data")
}

func darwinIfData64(raw []byte) (unix.IfData64, bool) {
	var data unix.IfData64
	size := int(unsafe.Sizeof(data))
	if len(raw) < size {
		return data, false
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(&data)), size), raw[:size])
	return data, true
}

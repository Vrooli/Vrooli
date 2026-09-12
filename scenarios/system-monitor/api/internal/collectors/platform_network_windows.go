//go:build windows

package collectors

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func collectPlatformNetwork(_ context.Context, c *NetworkCollector) platformNetworkReading {
	var table *windows.MibIfTable2
	if errCode := windows.GetIfTable2Ex(windows.MibIfTableNormal, &table); errCode != nil {
		return platformNetworkReading{status: "failed", reason: fmt.Sprintf("GetIfTable2Ex: %v", errCode), provenance: "Windows GetIfTable2Ex"}
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))
	if table == nil || table.NumEntries == 0 {
		return platformNetworkReading{status: "failed", reason: "Windows returned no interface counters", provenance: "Windows GetIfTable2Ex"}
	}
	rows := unsafe.Slice(&table.Table[0], table.NumEntries)
	stats := emptyNetworkStats()
	for _, row := range rows {
		stats["bytes_recv"] = stats["bytes_recv"].(int64) + int64(row.InOctets)
		stats["packets_recv"] = stats["packets_recv"].(int64) + int64(row.InUcastPkts)
		stats["errors_in"] = stats["errors_in"].(int64) + int64(row.InErrors)
		stats["dropped_in"] = stats["dropped_in"].(int64) + int64(row.InDiscards)
		stats["bytes_sent"] = stats["bytes_sent"].(int64) + int64(row.OutOctets)
		stats["packets_sent"] = stats["packets_sent"].(int64) + int64(row.OutUcastPkts)
		stats["errors_out"] = stats["errors_out"].(int64) + int64(row.OutErrors)
		stats["dropped_out"] = stats["dropped_out"].(int64) + int64(row.OutDiscards)
	}
	return networkReading(map[string]interface{}{
		"network_stats": stats,
		"bandwidth":     bandwidthFromCounters(c, stats["bytes_recv"].(int64), stats["bytes_sent"].(int64)),
		"tcp_states": map[string]string{
			"status": "unsupported",
			"reason": "Windows TCP table is not enabled in this build",
		},
	}, "Windows GetIfTable2Ex")
}

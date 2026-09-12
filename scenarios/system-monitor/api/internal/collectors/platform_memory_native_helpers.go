//go:build windows

package collectors

func memoryReadingFromBytes(total, available, buffers, cached, swapTotal, swapFree uint64, source string) platformMemoryReading {
	if total == 0 || available > total {
		return platformMemoryReading{status: "failed", reason: "native memory backend returned an invalid capacity", provenance: source}
	}
	used := total - available
	usage := float64(used) / float64(total) * 100
	swapUsed := uint64(0)
	if swapTotal >= swapFree {
		swapUsed = swapTotal - swapFree
	}
	swapPercent := float64(0)
	if swapTotal > 0 {
		swapPercent = float64(swapUsed) / float64(swapTotal) * 100
	}
	return platformMemoryReading{
		usage:      usage,
		status:     "measured",
		provenance: source,
		details: map[string]int64{
			"total":     bytesToInt64(total),
			"used":      bytesToInt64(used),
			"available": bytesToInt64(available),
			"cached":    bytesToInt64(cached),
			"buffers":   bytesToInt64(buffers),
		},
		swap: map[string]interface{}{
			"total":   bytesToInt64(swapTotal),
			"used":    bytesToInt64(swapUsed),
			"percent": swapPercent,
		},
	}
}

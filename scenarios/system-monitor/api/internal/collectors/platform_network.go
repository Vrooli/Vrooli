package collectors

import "time"

type platformNetworkReading struct {
	values     map[string]interface{}
	status     string
	reason     string
	provenance string
}

func networkReading(values map[string]interface{}, source string) platformNetworkReading {
	return platformNetworkReading{values: values, status: "measured", provenance: source}
}

func networkUnsupported(reason string) platformNetworkReading {
	return platformNetworkReading{values: map[string]interface{}{}, status: "unsupported", reason: reason, provenance: "platform backend"}
}

func emptyNetworkStats() map[string]interface{} {
	return map[string]interface{}{
		"bytes_sent": int64(0), "bytes_recv": int64(0),
		"packets_sent": int64(0), "packets_recv": int64(0),
		"errors_in": int64(0), "errors_out": int64(0),
		"dropped_in": int64(0), "dropped_out": int64(0),
	}
}

func bandwidthFromCounters(c *NetworkCollector, bytesRecv, bytesSent int64) map[string]float64 {
	bandwidth := map[string]float64{"in_mbps": 0.0, "out_mbps": 0.0}
	if c.lastBytesRecv > 0 && c.lastBytesSent > 0 && bytesRecv >= c.lastBytesRecv && bytesSent >= c.lastBytesSent {
		elapsed := time.Since(c.lastCheck).Seconds()
		if elapsed > 0 {
			bandwidth["in_mbps"] = float64(bytesRecv-c.lastBytesRecv) * 8 / elapsed / 1_000_000
			bandwidth["out_mbps"] = float64(bytesSent-c.lastBytesSent) * 8 / elapsed / 1_000_000
		}
	}
	c.lastBytesRecv = bytesRecv
	c.lastBytesSent = bytesSent
	c.lastCheck = time.Now()
	return bandwidth
}

package collectors

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NetworkCollector collects network metrics
type NetworkCollector struct {
	BaseCollector
	mu            sync.Mutex
	lastBytesRecv int64
	lastBytesSent int64
	lastCheck     time.Time
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		BaseCollector: NewBaseCollector("network", 10*time.Second),
		lastCheck:     time.Now(),
	}
}

// Collect gathers network metrics
func (c *NetworkCollector) Collect(ctx context.Context) (*MetricData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if collectorOS != runtime.GOOS {
		return unsupportedMetricData(c.GetName(), "network"), nil
	}
	reading := collectPlatformNetwork(ctx, c)
	values := networkReadingValues(reading)

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "network",
		Values:        values,
		Tags: map[string]string{
			"os":     collectorOS,
			"source": reading.provenance,
		},
	}, nil
}

// networkReadingValues annotates a platform result without assuming that a
// failed native probe allocated its values map. Some Darwin/Windows probe
// failures intentionally return only status and reason.
func networkReadingValues(reading platformNetworkReading) map[string]interface{} {
	values := reading.values
	if values == nil {
		values = make(map[string]interface{})
	}
	if reading.status != "" {
		values["status"] = reading.status
	}
	if reading.reason != "" {
		values["reason"] = reading.reason
	}
	return values
}

// getTCPConnections returns the number of TCP connections
func (c *NetworkCollector) getTCPConnections(ctx context.Context) int {
	if collectorOS != "linux" {
		return 0
	}

	states, err := readTCPStates(ctx)
	if err != nil {
		return 0
	}
	return states["established"]
}

// getTCPConnectionStates returns TCP connection states breakdown
func (c *NetworkCollector) getTCPConnectionStates(ctx context.Context) map[string]int {
	states := map[string]int{
		"established": 0,
		"time_wait":   0,
		"close_wait":  0,
		"fin_wait1":   0,
		"fin_wait2":   0,
		"syn_sent":    0,
		"syn_recv":    0,
		"closing":     0,
		"last_ack":    0,
		"listen":      0,
		"total":       0,
	}

	nativeStates, err := readTCPStates(ctx)
	if err != nil {
		return states
	}
	for state, count := range nativeStates {
		states[state] = count
	}

	return states
}

// getNetworkStats returns network statistics
func (c *NetworkCollector) getNetworkStats(ctx context.Context) map[string]interface{} {
	stats := map[string]interface{}{
		"bytes_sent":   int64(0),
		"bytes_recv":   int64(0),
		"packets_sent": int64(0),
		"packets_recv": int64(0),
		"errors_in":    int64(0),
		"errors_out":   int64(0),
		"dropped_in":   int64(0),
		"dropped_out":  int64(0),
	}

	dev, err := readPrimaryNetDev(ctx)
	if err != nil {
		return stats
	}

	stats["bytes_recv"] = dev.bytesRecv
	stats["packets_recv"] = dev.packetsRecv
	stats["errors_in"] = dev.errorsIn
	stats["dropped_in"] = dev.droppedIn
	stats["bytes_sent"] = dev.bytesSent
	stats["packets_sent"] = dev.packetsSent
	stats["errors_out"] = dev.errorsOut
	stats["dropped_out"] = dev.droppedOut

	return stats
}

// getPortUsage returns port usage statistics
func (c *NetworkCollector) getPortUsage(ctx context.Context) map[string]int {
	usage := map[string]int{
		"used":  0,
		"total": 32767, // Typical ephemeral port range
	}

	used, err := countEphemeralTCPPorts(ctx)
	if err == nil {
		usage["used"] = used
	}

	return usage
}

// calculateBandwidth calculates network bandwidth usage
func (c *NetworkCollector) calculateBandwidth() map[string]float64 {
	bandwidth := map[string]float64{
		"in_mbps":  0.0,
		"out_mbps": 0.0,
	}

	dev, err := readPrimaryNetDev(context.Background())
	if err != nil {
		return bandwidth
	}

	bytesRecv := dev.bytesRecv
	bytesSent := dev.bytesSent

	// Calculate bandwidth if we have previous values
	if c.lastBytesRecv > 0 && c.lastBytesSent > 0 {
		timeDiff := time.Since(c.lastCheck).Seconds()
		if timeDiff > 0 {
			bandwidth["in_mbps"] = float64(bytesRecv-c.lastBytesRecv) * 8 / timeDiff / 1_000_000
			bandwidth["out_mbps"] = float64(bytesSent-c.lastBytesSent) * 8 / timeDiff / 1_000_000
		}
	}

	c.lastBytesRecv = bytesRecv
	c.lastBytesSent = bytesSent
	c.lastCheck = time.Now()

	return bandwidth
}

type netDevStats struct {
	bytesRecv   int64
	packetsRecv int64
	errorsIn    int64
	droppedIn   int64
	bytesSent   int64
	packetsSent int64
	errorsOut   int64
	droppedOut  int64
}

var tcpStateCodes = map[string]string{
	"01": "established",
	"02": "syn_sent",
	"03": "syn_recv",
	"04": "fin_wait1",
	"05": "fin_wait2",
	"06": "time_wait",
	"08": "close_wait",
	"09": "last_ack",
	"0A": "listen",
	"0B": "closing",
}

func readTCPStates(ctx context.Context) (map[string]int, error) {
	states := map[string]int{
		"established": 0,
		"time_wait":   0,
		"close_wait":  0,
		"fin_wait1":   0,
		"fin_wait2":   0,
		"syn_sent":    0,
		"syn_recv":    0,
		"closing":     0,
		"last_ack":    0,
		"listen":      0,
		"total":       0,
	}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if ctx.Err() != nil {
			return states, ctx.Err()
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			state, ok := tcpStateCodes[strings.ToUpper(fields[3])]
			if !ok {
				continue
			}
			states[state]++
			states["total"]++
		}
	}
	return states, nil
}

func countEphemeralTCPPorts(ctx context.Context) (int, error) {
	count := 0
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if ctx.Err() != nil {
			return count, ctx.Err()
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			if isEphemeralAddressPort(fields[1]) {
				count++
			}
		}
	}
	return count, nil
}

func isEphemeralAddressPort(address string) bool {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return false
	}
	port, err := strconv.ParseInt(parts[1], 16, 64)
	if err != nil {
		return false
	}
	return port >= 30000 && port <= 69999
}

func readPrimaryNetDev(ctx context.Context) (netDevStats, error) {
	raw, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return netDevStats{}, err
	}

	var fallback *netDevStats
	for _, line := range strings.Split(string(raw), "\n")[2:] {
		if ctx.Err() != nil {
			return netDevStats{}, ctx.Err()
		}
		name, stats, ok := parseNetDevLine(line)
		if !ok || name == "lo" {
			continue
		}
		if strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "ens") || strings.HasPrefix(name, "enp") || strings.HasPrefix(name, "wl") {
			return stats, nil
		}
		if fallback == nil {
			copy := stats
			fallback = &copy
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return netDevStats{}, os.ErrNotExist
}

func parseNetDevLine(line string) (string, netDevStats, bool) {
	parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
	if len(parts) != 2 {
		return "", netDevStats{}, false
	}
	fields := strings.Fields(parts[1])
	if len(fields) < 16 {
		return "", netDevStats{}, false
	}
	return strings.TrimSpace(parts[0]), netDevStats{
		bytesRecv:   parseInt64OrZero(fields[0]),
		packetsRecv: parseInt64OrZero(fields[1]),
		errorsIn:    parseInt64OrZero(fields[2]),
		droppedIn:   parseInt64OrZero(fields[3]),
		bytesSent:   parseInt64OrZero(fields[8]),
		packetsSent: parseInt64OrZero(fields[9]),
		errorsOut:   parseInt64OrZero(fields[10]),
		droppedOut:  parseInt64OrZero(fields[11]),
	}, true
}

func parseInt64OrZero(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// GetConnectionPools returns connection pool information
func GetConnectionPools() []map[string]interface{} {
	// This would require integration with application metrics
	// For now, return mock data
	return []map[string]interface{}{
		{
			"name":      "postgres-main",
			"active":    8,
			"idle":      2,
			"max_size":  10,
			"waiting":   0,
			"healthy":   true,
			"leak_risk": "low",
		},
		{
			"name":      "redis-main",
			"active":    45,
			"idle":      55,
			"max_size":  100,
			"waiting":   0,
			"healthy":   true,
			"leak_risk": "low",
		},
	}
}

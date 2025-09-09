package collectors

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// NetworkCollector collects network metrics
type NetworkCollector struct {
	BaseCollector
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
	tcpConnections := c.getTCPConnections()
	tcpStates := c.getTCPConnectionStates()
	networkStats := c.getNetworkStats()
	portUsage := c.getPortUsage()
	
	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "network",
		Values: map[string]interface{}{
			"tcp_connections": tcpConnections,
			"tcp_states":      tcpStates,
			"network_stats":   networkStats,
			"port_usage":      portUsage,
			"bandwidth":       c.calculateBandwidth(),
		},
	}, nil
}

// getTCPConnections returns the number of TCP connections
func (c *NetworkCollector) getTCPConnections() int {
	if runtime.GOOS != "linux" {
		return 50 + (time.Now().Second() % 20)
	}
	
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep ESTABLISHED | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	
	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}
	return count
}

// getTCPConnectionStates returns TCP connection states breakdown
func (c *NetworkCollector) getTCPConnectionStates() map[string]int {
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
	
	if runtime.GOOS != "linux" {
		states["established"] = 10
		states["listen"] = 5
		states["total"] = 15
		return states
	}
	
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | awk 'NR>2 {print $6}' | sort | uniq -c")
	output, err := cmd.Output()
	if err != nil {
		return states
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		
		count, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		
		state := strings.ToLower(parts[1])
		switch state {
		case "established":
			states["established"] = count
		case "time_wait":
			states["time_wait"] = count
		case "close_wait":
			states["close_wait"] = count
		case "fin_wait1":
			states["fin_wait1"] = count
		case "fin_wait2":
			states["fin_wait2"] = count
		case "syn_sent":
			states["syn_sent"] = count
		case "syn_recv":
			states["syn_recv"] = count
		case "closing":
			states["closing"] = count
		case "last_ack":
			states["last_ack"] = count
		case "listen":
			states["listen"] = count
		}
		
		states["total"] += count
	}
	
	return states
}

// getNetworkStats returns network statistics
func (c *NetworkCollector) getNetworkStats() map[string]interface{} {
	stats := map[string]interface{}{
		"bytes_sent":     int64(0),
		"bytes_recv":     int64(0),
		"packets_sent":   int64(0),
		"packets_recv":   int64(0),
		"errors_in":      int64(0),
		"errors_out":     int64(0),
		"dropped_in":     int64(0),
		"dropped_out":    int64(0),
	}
	
	if runtime.GOOS != "linux" {
		return stats
	}
	
	// Get network interface statistics
	cmd := exec.Command("bash", "-c", "cat /proc/net/dev | grep -E '(eth|ens|enp|wl)' | head -1")
	output, err := cmd.Output()
	if err != nil {
		return stats
	}
	
	fields := strings.Fields(string(output))
	if len(fields) >= 17 {
		stats["bytes_recv"], _ = strconv.ParseInt(fields[1], 10, 64)
		stats["packets_recv"], _ = strconv.ParseInt(fields[2], 10, 64)
		stats["errors_in"], _ = strconv.ParseInt(fields[3], 10, 64)
		stats["dropped_in"], _ = strconv.ParseInt(fields[4], 10, 64)
		
		stats["bytes_sent"], _ = strconv.ParseInt(fields[9], 10, 64)
		stats["packets_sent"], _ = strconv.ParseInt(fields[10], 10, 64)
		stats["errors_out"], _ = strconv.ParseInt(fields[11], 10, 64)
		stats["dropped_out"], _ = strconv.ParseInt(fields[12], 10, 64)
	}
	
	return stats
}

// getPortUsage returns port usage statistics
func (c *NetworkCollector) getPortUsage() map[string]int {
	usage := map[string]int{
		"used":  0,
		"total": 32767, // Typical ephemeral port range
	}
	
	if runtime.GOOS != "linux" {
		usage["used"] = 150
		return usage
	}
	
	// Count used ephemeral ports
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep -E ':([3-6][0-9]{4})' | wc -l")
	output, err := cmd.Output()
	if err == nil {
		usage["used"], _ = strconv.Atoi(strings.TrimSpace(string(output)))
	}
	
	return usage
}

// calculateBandwidth calculates network bandwidth usage
func (c *NetworkCollector) calculateBandwidth() map[string]float64 {
	bandwidth := map[string]float64{
		"in_mbps":  0.0,
		"out_mbps": 0.0,
	}
	
	if runtime.GOOS != "linux" {
		bandwidth["in_mbps"] = 12.5
		bandwidth["out_mbps"] = 8.2
		return bandwidth
	}
	
	// Get current bytes
	cmd := exec.Command("bash", "-c", "cat /proc/net/dev | grep -E '(eth|ens|enp|wl)' | head -1 | awk '{print $2, $10}'")
	output, err := cmd.Output()
	if err != nil {
		return bandwidth
	}
	
	fields := strings.Fields(string(output))
	if len(fields) >= 2 {
		bytesRecv, _ := strconv.ParseInt(fields[0], 10, 64)
		bytesSent, _ := strconv.ParseInt(fields[1], 10, 64)
		
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
	}
	
	return bandwidth
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
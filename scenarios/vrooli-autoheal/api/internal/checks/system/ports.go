// Package system provides system-level health checks
// [REQ:SYSTEM-PORTS-001]
package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// PortCheck monitors ephemeral port exhaustion.
type PortCheck struct {
	warningThreshold  int // percentage
	criticalThreshold int // percentage
}

// PortCheckOption configures a PortCheck.
type PortCheckOption func(*PortCheck)

// WithPortThresholds sets warning and critical thresholds (percentages of ephemeral port range).
func WithPortThresholds(warning, critical int) PortCheckOption {
	return func(c *PortCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// NewPortCheck creates an ephemeral port exhaustion check.
// Default thresholds: warning at 70%, critical at 85%
func NewPortCheck(opts ...PortCheckOption) *PortCheck {
	c := &PortCheck{
		warningThreshold:  70,
		criticalThreshold: 85,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PortCheck) ID() string          { return "system-ports" }
func (c *PortCheck) Title() string       { return "Ephemeral Port Usage" }
func (c *PortCheck) Description() string { return "Monitors ephemeral port usage to prevent connection failures" }
func (c *PortCheck) Importance() string {
	return "Port exhaustion causes 'cannot assign requested address' errors for new connections"
}
func (c *PortCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *PortCheck) IntervalSeconds() int       { return 300 }
func (c *PortCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *PortCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if runtime.GOOS == "windows" {
		result.Status = checks.StatusWarning
		result.Message = "Port exhaustion check not yet implemented for Windows"
		result.Details["platform"] = "windows"
		return result
	}

	// Read ephemeral port range
	portMin, portMax, err := readEphemeralPortRange()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read ephemeral port range"
		result.Details["error"] = err.Error()
		return result
	}

	totalPorts := portMax - portMin + 1
	result.Details["portMin"] = portMin
	result.Details["portMax"] = portMax
	result.Details["totalPorts"] = totalPorts

	// Count connections using ephemeral ports
	usedPorts, topConsumers, err := countEphemeralPortUsage(portMin, portMax)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to count port usage"
		result.Details["error"] = err.Error()
		return result
	}

	usedPercent := 0
	if totalPorts > 0 {
		usedPercent = int((usedPorts * 100) / totalPorts)
	}

	result.Details["usedPorts"] = usedPorts
	result.Details["usedPercent"] = usedPercent
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold
	result.Details["topConsumers"] = topConsumers

	// Calculate score
	score := 100 - usedPercent
	if score < 0 {
		score = 0
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "ephemeral-ports",
				Passed: usedPercent < c.criticalThreshold,
				Detail: fmt.Sprintf("%d%% used (%d / %d ports)", usedPercent, usedPorts, totalPorts),
			},
		},
	}

	switch {
	case usedPercent >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Ephemeral port exhaustion critical: %d%% used", usedPercent)
	case usedPercent >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Ephemeral port usage warning: %d%% used", usedPercent)
	default:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Ephemeral port usage healthy: %d%% used", usedPercent)
	}

	return result
}

// readEphemeralPortRange reads the ephemeral port range from /proc/sys/net/ipv4/ip_local_port_range
func readEphemeralPortRange() (min, max int, err error) {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_local_port_range")
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected format in ip_local_port_range: %s", string(data))
	}

	min, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, err
	}
	max, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, err
	}

	return min, max, nil
}

// portConsumer tracks port usage by local address
type portConsumer struct {
	LocalAddr string `json:"localAddr"`
	Count     int    `json:"count"`
}

// countEphemeralPortUsage counts connections using ephemeral ports
func countEphemeralPortUsage(portMin, portMax int) (int, []portConsumer, error) {
	usedPorts := 0
	consumerCounts := make(map[string]int)

	// Check both TCP and TCP6
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, nil, err
		}

		scanner := bufio.NewScanner(file)
		// Skip header line
		scanner.Scan()

		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			// Local address is field 1, format: "XXXXXXXX:PPPP" (hex IP:hex port)
			localAddr := fields[1]
			colonIdx := strings.LastIndex(localAddr, ":")
			if colonIdx == -1 {
				continue
			}

			portHex := localAddr[colonIdx+1:]
			port, err := strconv.ParseInt(portHex, 16, 32)
			if err != nil {
				continue
			}

			if int(port) >= portMin && int(port) <= portMax {
				usedPorts++

				// Track by IP for consumer stats
				ip := localAddr[:colonIdx]
				consumerCounts[ip]++
			}
		}
		file.Close()

		if err := scanner.Err(); err != nil {
			return 0, nil, err
		}
	}

	// Convert to sorted slice (top consumers first)
	var consumers []portConsumer
	for addr, count := range consumerCounts {
		consumers = append(consumers, portConsumer{
			LocalAddr: addr,
			Count:     count,
		})
	}

	// Sort by count descending (simple bubble sort for small lists)
	for i := 0; i < len(consumers); i++ {
		for j := i + 1; j < len(consumers); j++ {
			if consumers[j].Count > consumers[i].Count {
				consumers[i], consumers[j] = consumers[j], consumers[i]
			}
		}
	}

	// Return top 5
	limit := 5
	if len(consumers) < limit {
		limit = len(consumers)
	}

	return usedPorts, consumers[:limit], nil
}

// Package system provides system-level health checks
// [REQ:SYSTEM-PORTS-001] [REQ:HEAL-ACTION-001]
package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

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

// RecoveryActions returns available recovery actions for port management
// [REQ:HEAL-ACTION-001]
func (c *PortCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isHighUsage := false
	if lastResult != nil {
		if pct, ok := lastResult.Details["usedPercent"].(int); ok {
			isHighUsage = pct >= c.warningThreshold
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "analyze",
			Name:        "Analyze Connections",
			Description: "Show detailed breakdown of port usage by process",
			Dangerous:   false,
			Available:   true,
		},
		{
			ID:          "time-wait",
			Name:        "Show TIME_WAIT",
			Description: "List connections in TIME_WAIT state that are holding ports",
			Dangerous:   false,
			Available:   isHighUsage,
		},
		{
			ID:          "kill-port",
			Name:        "Kill Port Listener",
			Description: "Terminate process listening on a specific port (requires port parameter)",
			Dangerous:   true,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *PortCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "analyze":
		return c.executeAnalyze(ctx, start)
	case "time-wait":
		return c.executeTimeWait(ctx, start)
	case "kill-port":
		// This action needs a port parameter but we return helpful info
		result.Success = true
		result.Message = "Use lsof -i :<port> to identify and kill specific port listeners"
		result.Output = "To kill a specific port listener:\n" +
			"  lsof -t -iTCP:<port> -sTCP:LISTEN | xargs kill\n\n" +
			"This action is available via API with port parameter"
		result.Duration = time.Since(start)
		return result
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeAnalyze provides detailed port usage breakdown by process
func (c *PortCheck) executeAnalyze(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "analyze",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Use ss to get connection stats by process
	cmd := exec.CommandContext(ctx, "ss", "-tunap")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to netstat
		cmd = exec.CommandContext(ctx, "netstat", "-tunp")
		output, err = cmd.CombinedOutput()
		if err != nil {
			result.Success = false
			result.Error = "Failed to get connection info: " + err.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Duration = time.Since(start)
	result.Success = true
	result.Message = "Connection analysis complete"
	result.Output = string(output)
	return result
}

// executeTimeWait shows connections in TIME_WAIT state
func (c *PortCheck) executeTimeWait(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "time-wait",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Use ss to filter for TIME-WAIT connections
	cmd := exec.CommandContext(ctx, "ss", "-tan", "state", "time-wait")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = "Failed to get TIME_WAIT connections: " + err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Count TIME_WAIT connections
	lines := strings.Split(string(output), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "State") {
			count++
		}
	}

	result.Duration = time.Since(start)
	result.Success = true
	result.Message = fmt.Sprintf("Found %d connections in TIME_WAIT state", count)
	result.Output = string(output)
	return result
}

// Ensure PortCheck implements HealableCheck
var _ checks.HealableCheck = (*PortCheck)(nil)

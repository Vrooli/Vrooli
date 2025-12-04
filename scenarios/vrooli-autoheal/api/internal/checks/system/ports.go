// Package system provides system-level health checks
// [REQ:SYSTEM-PORTS-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// PortCheck monitors ephemeral port exhaustion.
type PortCheck struct {
	warningThreshold  int // percentage
	criticalThreshold int // percentage
	portReader        checks.PortReader
	executor          checks.CommandExecutor
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

// WithPortReader sets the port reader (for testing).
// [REQ:TEST-SEAM-001]
func WithPortReader(reader checks.PortReader) PortCheckOption {
	return func(c *PortCheck) {
		c.portReader = reader
	}
}

// WithPortExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithPortExecutor(executor checks.CommandExecutor) PortCheckOption {
	return func(c *PortCheck) {
		c.executor = executor
	}
}

// NewPortCheck creates an ephemeral port exhaustion check.
// Default thresholds: warning at 70%, critical at 85%
func NewPortCheck(opts ...PortCheckOption) *PortCheck {
	c := &PortCheck{
		warningThreshold:  70,
		criticalThreshold: 85,
		portReader:        checks.DefaultPortReader,
		executor:          checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PortCheck) ID() string    { return "system-ports" }
func (c *PortCheck) Title() string { return "Ephemeral Port Usage" }
func (c *PortCheck) Description() string {
	return "Monitors ephemeral port usage to prevent connection failures"
}
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

	// Read port statistics via injected reader
	portInfo, err := c.portReader.ReadPortStats()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read port statistics"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["totalPorts"] = portInfo.TotalPorts
	result.Details["usedPorts"] = portInfo.UsedPorts
	result.Details["usedPercent"] = portInfo.UsedPercent
	result.Details["timeWait"] = portInfo.TimeWait
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	usedPercent := portInfo.UsedPercent

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
				Detail: fmt.Sprintf("%d%% used (%d / %d ports)", usedPercent, portInfo.UsedPorts, portInfo.TotalPorts),
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

	// Use ss to get connection stats by process via executor
	output, err := c.executor.CombinedOutput(ctx, "ss", "-tunap")
	if err != nil {
		// Fallback to netstat
		output, err = c.executor.CombinedOutput(ctx, "netstat", "-tunp")
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

	// Use ss to filter for TIME-WAIT connections via executor
	output, err := c.executor.CombinedOutput(ctx, "ss", "-tan", "state", "time-wait")
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

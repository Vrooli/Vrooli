// Package system provides system-level health checks
// [REQ:SYSTEM-SWAP-001]
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

// SwapCheck monitors swap usage as an indicator of memory pressure.
type SwapCheck struct {
	warningThreshold  int // percentage
	criticalThreshold int // percentage
}

// SwapCheckOption configures a SwapCheck.
type SwapCheckOption func(*SwapCheck)

// WithSwapThresholds sets warning and critical thresholds (percentages).
func WithSwapThresholds(warning, critical int) SwapCheckOption {
	return func(c *SwapCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// NewSwapCheck creates a swap usage check.
// Default thresholds: warning at 50%, critical at 80%
func NewSwapCheck(opts ...SwapCheckOption) *SwapCheck {
	c := &SwapCheck{
		warningThreshold:  50,
		criticalThreshold: 80,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *SwapCheck) ID() string          { return "system-swap" }
func (c *SwapCheck) Title() string       { return "Swap Usage" }
func (c *SwapCheck) Description() string { return "Monitors swap usage as an indicator of memory pressure" }
func (c *SwapCheck) Importance() string {
	return "High swap usage indicates memory pressure and can cause severe performance degradation"
}
func (c *SwapCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *SwapCheck) IntervalSeconds() int       { return 300 }
func (c *SwapCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *SwapCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if runtime.GOOS == "windows" {
		result.Status = checks.StatusWarning
		result.Message = "Swap check not yet implemented for Windows"
		result.Details["platform"] = "windows"
		return result
	}

	// Read /proc/meminfo to get swap information
	swapTotal, swapFree, err := readSwapInfo()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read swap information"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["swapTotalKB"] = swapTotal
	result.Details["swapFreeKB"] = swapFree
	result.Details["swapTotalBytes"] = swapTotal * 1024
	result.Details["swapFreeBytes"] = swapFree * 1024

	// Handle no swap configured
	if swapTotal == 0 {
		result.Status = checks.StatusWarning
		result.Message = "No swap configured - system may OOM kill processes under memory pressure"
		result.Details["swapConfigured"] = false
		return result
	}

	swapUsed := swapTotal - swapFree
	usedPercent := int((swapUsed * 100) / swapTotal)

	result.Details["swapUsedKB"] = swapUsed
	result.Details["swapUsedBytes"] = swapUsed * 1024
	result.Details["usedPercent"] = usedPercent
	result.Details["swapConfigured"] = true
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Calculate score (inverse of usage)
	score := 100 - usedPercent
	if score < 0 {
		score = 0
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "swap",
				Passed: usedPercent < c.criticalThreshold,
				Detail: fmt.Sprintf("%d%% used (%s / %s)", usedPercent, formatBytes(uint64(swapUsed*1024)), formatBytes(uint64(swapTotal*1024))),
			},
		},
	}

	switch {
	case usedPercent >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Swap usage critical: %d%% used", usedPercent)
	case usedPercent >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Swap usage warning: %d%% used - system under memory pressure", usedPercent)
	default:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Swap usage healthy: %d%% used", usedPercent)
	}

	return result
}

// readSwapInfo reads swap total and free from /proc/meminfo
func readSwapInfo() (total, free uint64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "SwapTotal:":
			total, _ = strconv.ParseUint(fields[1], 10, 64)
		case "SwapFree:":
			free, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}

	return total, free, scanner.Err()
}

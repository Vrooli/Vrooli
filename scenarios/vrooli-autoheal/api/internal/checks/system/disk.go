// Package system provides system-level health checks for disk, memory, and processes
// [REQ:SYSTEM-DISK-001]
package system

import (
	"context"
	"fmt"
	"runtime"

	"syscall"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DiskCheck monitors disk space usage on specified partitions.
type DiskCheck struct {
	partitions       []string
	warningThreshold int // percentage
	criticalThreshold int // percentage
}

// DiskCheckOption configures a DiskCheck.
type DiskCheckOption func(*DiskCheck)

// WithPartitions sets the partitions to check.
func WithPartitions(partitions []string) DiskCheckOption {
	return func(c *DiskCheck) {
		c.partitions = partitions
	}
}

// WithDiskThresholds sets warning and critical thresholds (percentages).
func WithDiskThresholds(warning, critical int) DiskCheckOption {
	return func(c *DiskCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// NewDiskCheck creates a disk space check.
// Default partitions: "/" and "/home"
// Default thresholds: warning at 80%, critical at 90%
func NewDiskCheck(opts ...DiskCheckOption) *DiskCheck {
	c := &DiskCheck{
		partitions:       []string{"/"},
		warningThreshold: 80,
		criticalThreshold: 90,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *DiskCheck) ID() string          { return "system-disk" }
func (c *DiskCheck) Title() string       { return "Disk Space" }
func (c *DiskCheck) Description() string { return "Monitors disk space usage on configured partitions" }
func (c *DiskCheck) Importance() string {
	return "Low disk space can cause service failures, database corruption, and log loss"
}
func (c *DiskCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *DiskCheck) IntervalSeconds() int       { return 300 }
func (c *DiskCheck) Platforms() []platform.Type { return nil } // all platforms

func (c *DiskCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if runtime.GOOS == "windows" {
		// Windows uses different filesystem APIs
		return c.runWindows(ctx, result)
	}

	return c.runUnix(ctx, result)
}

func (c *DiskCheck) runUnix(ctx context.Context, result checks.Result) checks.Result {
	var subChecks []checks.SubCheck
	worstStatus := checks.StatusOK
	partitionDetails := make([]map[string]interface{}, 0)

	for _, partition := range c.partitions {
		var stat syscall.Statfs_t
		err := syscall.Statfs(partition, &stat)
		if err != nil {
			subChecks = append(subChecks, checks.SubCheck{
				Name:   partition,
				Passed: false,
				Detail: fmt.Sprintf("failed to stat: %v", err),
			})
			worstStatus = checks.WorstStatus(worstStatus, checks.StatusCritical)
			continue
		}

		// Calculate usage percentage
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		used := total - free
		usedPercent := 0
		if total > 0 {
			usedPercent = int((used * 100) / total)
		}

		// Determine status for this partition
		var partStatus checks.Status
		var passed bool
		switch {
		case usedPercent >= c.criticalThreshold:
			partStatus = checks.StatusCritical
			passed = false
		case usedPercent >= c.warningThreshold:
			partStatus = checks.StatusWarning
			passed = true // warning is still passing
		default:
			partStatus = checks.StatusOK
			passed = true
		}

		worstStatus = checks.WorstStatus(worstStatus, partStatus)

		subChecks = append(subChecks, checks.SubCheck{
			Name:   partition,
			Passed: passed,
			Detail: fmt.Sprintf("%d%% used (%s / %s)", usedPercent, formatBytes(used), formatBytes(total)),
		})

		partitionDetails = append(partitionDetails, map[string]interface{}{
			"partition":   partition,
			"usedPercent": usedPercent,
			"usedBytes":   used,
			"totalBytes":  total,
			"freeBytes":   free,
			"status":      string(partStatus),
		})
	}

	result.Status = worstStatus
	result.Details["partitions"] = partitionDetails
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Calculate overall score (inverse of worst usage)
	score := 100
	for _, p := range partitionDetails {
		used := p["usedPercent"].(int)
		remaining := 100 - used
		if remaining < score {
			score = remaining
		}
	}

	result.Metrics = &checks.HealthMetrics{
		Score:     &score,
		SubChecks: subChecks,
	}

	switch worstStatus {
	case checks.StatusOK:
		result.Message = "Disk space healthy on all partitions"
	case checks.StatusWarning:
		result.Message = "Disk space warning - some partitions above " + fmt.Sprintf("%d%%", c.warningThreshold)
	case checks.StatusCritical:
		result.Message = "Disk space critical - some partitions above " + fmt.Sprintf("%d%%", c.criticalThreshold)
	}

	return result
}

func (c *DiskCheck) runWindows(ctx context.Context, result checks.Result) checks.Result {
	// Windows implementation would use GetDiskFreeSpaceEx
	// For now, return a warning that this is not implemented
	result.Status = checks.StatusWarning
	result.Message = "Disk check not yet implemented for Windows"
	result.Details["platform"] = "windows"
	return result
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Package system provides system-level health checks
// [REQ:SYSTEM-INODE-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"fmt"
	"runtime"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// InodeCheck monitors inode usage on specified partitions.
// Inode exhaustion can cause "no space left on device" errors even with free disk space.
type InodeCheck struct {
	partitions        []string
	warningThreshold  int // percentage
	criticalThreshold int // percentage
	fsReader          checks.FileSystemReader
}

// InodeCheckOption configures an InodeCheck.
type InodeCheckOption func(*InodeCheck)

// WithInodePartitions sets the partitions to check.
func WithInodePartitions(partitions []string) InodeCheckOption {
	return func(c *InodeCheck) {
		c.partitions = partitions
	}
}

// WithInodeThresholds sets warning and critical thresholds (percentages).
func WithInodeThresholds(warning, critical int) InodeCheckOption {
	return func(c *InodeCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithInodeFileSystemReader sets the filesystem reader (for testing).
// [REQ:TEST-SEAM-001]
func WithInodeFileSystemReader(reader checks.FileSystemReader) InodeCheckOption {
	return func(c *InodeCheck) {
		c.fsReader = reader
	}
}

// NewInodeCheck creates an inode usage check.
// Default partitions: "/"
// Default thresholds: warning at 80%, critical at 90%
func NewInodeCheck(opts ...InodeCheckOption) *InodeCheck {
	c := &InodeCheck{
		partitions:        []string{"/"},
		warningThreshold:  80,
		criticalThreshold: 90,
		fsReader:          checks.DefaultFileSystemReader,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *InodeCheck) ID() string    { return "system-inode" }
func (c *InodeCheck) Title() string { return "Inode Usage" }
func (c *InodeCheck) Description() string {
	return "Monitors inode usage to prevent file creation failures"
}
func (c *InodeCheck) Importance() string {
	return "Inode exhaustion causes 'no space left on device' errors even with free disk space"
}
func (c *InodeCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *InodeCheck) IntervalSeconds() int       { return 300 }
func (c *InodeCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *InodeCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if runtime.GOOS == "windows" {
		// Windows doesn't have inodes
		result.Status = checks.StatusOK
		result.Message = "Inode check not applicable on Windows"
		result.Details["platform"] = "windows"
		return result
	}

	var subChecks []checks.SubCheck
	worstStatus := checks.StatusOK
	partitionDetails := make([]map[string]interface{}, 0)

	for _, partition := range c.partitions {
		stat, err := c.fsReader.Statfs(partition)
		if err != nil {
			subChecks = append(subChecks, checks.SubCheck{
				Name:   partition,
				Passed: false,
				Detail: fmt.Sprintf("failed to stat: %v", err),
			})
			worstStatus = checks.WorstStatus(worstStatus, checks.StatusCritical)
			continue
		}

		// Calculate inode usage percentage
		totalInodes := stat.Files
		freeInodes := stat.Ffree
		usedInodes := totalInodes - freeInodes
		usedPercent := 0
		if totalInodes > 0 {
			usedPercent = int((usedInodes * 100) / totalInodes)
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
			passed = true
		default:
			partStatus = checks.StatusOK
			passed = true
		}

		worstStatus = checks.WorstStatus(worstStatus, partStatus)

		subChecks = append(subChecks, checks.SubCheck{
			Name:   partition,
			Passed: passed,
			Detail: fmt.Sprintf("%d%% used (%d / %d inodes)", usedPercent, usedInodes, totalInodes),
		})

		partitionDetails = append(partitionDetails, map[string]interface{}{
			"partition":   partition,
			"usedPercent": usedPercent,
			"usedInodes":  usedInodes,
			"totalInodes": totalInodes,
			"freeInodes":  freeInodes,
			"status":      string(partStatus),
		})
	}

	result.Status = worstStatus
	result.Details["partitions"] = partitionDetails
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Calculate overall score
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
		result.Message = "Inode usage healthy on all partitions"
	case checks.StatusWarning:
		result.Message = "Inode usage warning - some partitions above " + fmt.Sprintf("%d%%", c.warningThreshold)
	case checks.StatusCritical:
		result.Message = "Inode usage critical - some partitions above " + fmt.Sprintf("%d%%", c.criticalThreshold)
	}

	return result
}

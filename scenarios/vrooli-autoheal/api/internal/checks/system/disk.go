// Package system provides system-level health checks for disk, memory, and processes
// [REQ:SYSTEM-DISK-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
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

// DiskCheck monitors disk space usage on specified partitions.
type DiskCheck struct {
	partitions        []string
	warningThreshold  int // percentage
	criticalThreshold int // percentage
	fsReader          checks.FileSystemReader
	executor          checks.CommandExecutor
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

// WithFileSystemReader sets the filesystem reader (for testing).
// [REQ:TEST-SEAM-001]
func WithFileSystemReader(reader checks.FileSystemReader) DiskCheckOption {
	return func(c *DiskCheck) {
		c.fsReader = reader
	}
}

// WithDiskExecutor sets the command executor (for testing and recovery actions).
// [REQ:TEST-SEAM-001]
func WithDiskExecutor(executor checks.CommandExecutor) DiskCheckOption {
	return func(c *DiskCheck) {
		c.executor = executor
	}
}

// NewDiskCheck creates a disk space check.
// Default partitions: "/" and "/home"
// Default thresholds: warning at 80%, critical at 90%
func NewDiskCheck(opts ...DiskCheckOption) *DiskCheck {
	c := &DiskCheck{
		partitions:        []string{"/"},
		warningThreshold:  80,
		criticalThreshold: 90,
		fsReader:          checks.DefaultFileSystemReader,
		executor:          checks.DefaultExecutor,
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
		if used, ok := p["usedPercent"].(int); ok {
			remaining := 100 - used
			if remaining < score {
				score = remaining
			}
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
	// Use WMIC to get disk information on Windows
	// WMIC outputs: "DeviceID  FreeSpace      Size"
	output, err := c.executor.Output(ctx, "wmic", "logicaldisk", "get", "DeviceID,FreeSpace,Size", "/format:csv")
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Failed to query disk information on Windows"
		result.Details["error"] = err.Error()
		return result
	}

	var subChecks []checks.SubCheck
	worstStatus := checks.StatusOK
	partitionDetails := make([]map[string]interface{}, 0)

	// Parse WMIC CSV output
	// Format: Node,DeviceID,FreeSpace,Size
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue // Skip header and empty lines
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		deviceID := strings.TrimSpace(parts[1])
		freeSpaceStr := strings.TrimSpace(parts[2])
		sizeStr := strings.TrimSpace(parts[3])

		if deviceID == "" || sizeStr == "" || sizeStr == "0" {
			continue // Skip drives with no size (CD-ROM, etc.)
		}

		// Check if this partition is in the configured list (or check all if not specified)
		shouldCheck := len(c.partitions) == 0
		for _, p := range c.partitions {
			// Windows partitions are like "C:", "D:", etc.
			if strings.EqualFold(p, deviceID) || strings.EqualFold(p, strings.TrimSuffix(deviceID, ":")) {
				shouldCheck = true
				break
			}
		}
		if !shouldCheck {
			continue
		}

		// Parse sizes
		var total, free uint64
		fmt.Sscanf(sizeStr, "%d", &total)
		fmt.Sscanf(freeSpaceStr, "%d", &free)

		if total == 0 {
			continue
		}

		used := total - free
		usedPercent := int((used * 100) / total)

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
			Name:   deviceID,
			Passed: passed,
			Detail: fmt.Sprintf("%d%% used (%s / %s)", usedPercent, formatBytes(used), formatBytes(total)),
		})

		partitionDetails = append(partitionDetails, map[string]interface{}{
			"partition":   deviceID,
			"usedPercent": usedPercent,
			"usedBytes":   used,
			"totalBytes":  total,
			"freeBytes":   free,
			"status":      string(partStatus),
		})
	}

	if len(partitionDetails) == 0 {
		result.Status = checks.StatusWarning
		result.Message = "No disk partitions found to check"
		result.Details["platform"] = "windows"
		return result
	}

	result.Status = worstStatus
	result.Details["partitions"] = partitionDetails
	result.Details["platform"] = "windows"
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Calculate overall score (inverse of worst usage)
	score := 100
	for _, p := range partitionDetails {
		if used, ok := p["usedPercent"].(int); ok {
			remaining := 100 - used
			if remaining < score {
				score = remaining
			}
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

// RecoveryActions returns available recovery actions for disk space issues
// [REQ:HEAL-ACTION-001]
func (c *DiskCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"

	return []checks.RecoveryAction{
		{
			ID:          "clean-apt-cache",
			Name:        "Clean APT Cache",
			Description: "Remove cached package files from apt",
			Dangerous:   false,
			Available:   isLinux,
		},
		{
			ID:          "clean-journal",
			Name:        "Clean System Journals",
			Description: "Vacuum old journal logs (keeps last 100MB)",
			Dangerous:   false,
			Available:   isLinux,
		},
		{
			ID:          "clean-docker",
			Name:        "Clean Docker Resources",
			Description: "Remove unused Docker images, containers, and volumes",
			Dangerous:   true, // Could remove containers/images in use
			Available:   isLinux,
		},
		{
			ID:          "find-large-files",
			Name:        "Find Large Files",
			Description: "List the largest files consuming disk space",
			Dangerous:   false,
			Available:   isLinux,
		},
		{
			ID:          "analyze-usage",
			Name:        "Analyze Disk Usage",
			Description: "Show disk usage breakdown by directory",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *DiskCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "clean-apt-cache":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "apt-get", "clean")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to clean APT cache"
			return result
		}

		result.Success = true
		result.Message = "APT cache cleaned successfully"
		return result

	case "clean-journal":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "journalctl", "--vacuum-size=100M")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to vacuum journal logs"
			return result
		}

		result.Success = true
		result.Message = "Journal logs vacuumed successfully"
		return result

	case "clean-docker":
		output, err := c.executor.CombinedOutput(ctx, "docker", "system", "prune", "-af", "--volumes")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to clean Docker resources"
			return result
		}

		result.Success = true
		result.Message = "Docker resources cleaned successfully"
		return result

	case "find-large-files":
		return c.executeFindLargeFiles(ctx, start)

	case "analyze-usage":
		return c.executeAnalyzeUsage(ctx, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeFindLargeFiles finds the largest files on the system
func (c *DiskCheck) executeFindLargeFiles(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "find-large-files",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Largest Files (>100MB) ===\n\n")

	// Find large files in each monitored partition
	for _, partition := range c.partitions {
		outputBuilder.WriteString(fmt.Sprintf("--- %s ---\n", partition))
		output, _ := c.executor.CombinedOutput(ctx, "find", partition, "-type", "f", "-size", "+100M",
			"-exec", "ls", "-lh", "{}", ";", "-maxdepth", "5")
		if len(output) > 0 {
			outputBuilder.Write(output)
		} else {
			outputBuilder.WriteString("No files >100MB found\n")
		}
		outputBuilder.WriteString("\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Large file search completed"
	return result
}

// executeAnalyzeUsage shows disk usage breakdown
func (c *DiskCheck) executeAnalyzeUsage(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "analyze-usage",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var outputBuilder strings.Builder
	outputBuilder.WriteString("=== Disk Usage Analysis ===\n\n")

	// Show overall disk usage
	outputBuilder.WriteString("--- Partition Summary ---\n")
	dfOutput, _ := c.executor.CombinedOutput(ctx, "df", "-h")
	outputBuilder.Write(dfOutput)
	outputBuilder.WriteString("\n\n")

	// Show top-level directory sizes for each partition
	for _, partition := range c.partitions {
		outputBuilder.WriteString(fmt.Sprintf("--- Top directories in %s ---\n", partition))
		duOutput, _ := c.executor.CombinedOutput(ctx, "du", "-h", "--max-depth=1", partition)
		outputBuilder.Write(duOutput)
		outputBuilder.WriteString("\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Disk usage analysis completed"
	return result
}

// Ensure DiskCheck implements HealableCheck
var _ checks.HealableCheck = (*DiskCheck)(nil)

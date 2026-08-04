// Package system provides system-level health checks for disk, memory, and processes
// [REQ:SYSTEM-DISK-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package system

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/cleanupmanager"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

const (
	diskCheckID = "system-disk"

	// Used only when the check configuration declares no threshold at all.
	fallbackWarningPercent  = 80
	fallbackCriticalPercent = 90
)

// DiskCheck monitors disk space usage on specified partitions.
type DiskCheck struct {
	partitions        []string
	warningThreshold  int // percentage
	criticalThreshold int // percentage
	intervalSeconds   int
	fsReader          checks.FileSystemReader
	executor          checks.CommandExecutor
	cleanup           cleanupmanager.Reporter
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

// WithDiskInterval sets how often the scheduler runs this check.
// Non-positive values are ignored so a partially populated config cannot
// silently disable the check.
func WithDiskInterval(seconds int) DiskCheckOption {
	return func(c *DiskCheck) {
		if seconds > 0 {
			c.intervalSeconds = seconds
		}
	}
}

// WithCleanupReporter sets the storage-manager client used by the
// request-cleanup heal action.
// [REQ:TEST-SEAM-001]
func WithCleanupReporter(reporter cleanupmanager.Reporter) DiskCheckOption {
	return func(c *DiskCheck) {
		if reporter != nil {
			c.cleanup = reporter
		}
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
//
// Every default — partitions, thresholds, and interval — is read from the
// user-facing check configuration in the userconfig package. That package is
// the single source: there is no second copy here for the scheduler and the
// configuration surface to disagree about.
func NewDiskCheck(opts ...DiskCheckOption) *DiskCheck {
	defaults := userconfig.GetCheckDefaults(diskCheckID)
	c := &DiskCheck{
		partitions:        defaultDiskPartitions(defaults),
		warningThreshold:  defaultThresholdPercent(defaults, thresholdWarning),
		criticalThreshold: defaultThresholdPercent(defaults, thresholdCritical),
		intervalSeconds:   defaults.IntervalSeconds,
		fsReader:          checks.DefaultFileSystemReader,
		executor:          checks.DefaultExecutor,
		cleanup:           cleanupmanager.NewClient(cleanupmanager.Config{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type thresholdKind int

const (
	thresholdWarning thresholdKind = iota
	thresholdCritical
)

// defaultDiskPartitions resolves the partitions to watch, falling back to the
// platform root when the configuration names none.
func defaultDiskPartitions(defaults userconfig.CheckDefaults) []string {
	if defaults.Thresholds != nil && len(defaults.Thresholds.Partitions) > 0 {
		return append([]string(nil), defaults.Thresholds.Partitions...)
	}
	if runtime.GOOS == "windows" {
		return []string{`C:\`}
	}
	return []string{"/"}
}

func defaultThresholdPercent(defaults userconfig.CheckDefaults, kind thresholdKind) int {
	if defaults.Thresholds != nil {
		switch kind {
		case thresholdWarning:
			if defaults.Thresholds.WarningPercent != nil {
				return int(*defaults.Thresholds.WarningPercent)
			}
		case thresholdCritical:
			if defaults.Thresholds.CriticalPercent != nil {
				return int(*defaults.Thresholds.CriticalPercent)
			}
		}
	}
	if kind == thresholdCritical {
		return fallbackCriticalPercent
	}
	return fallbackWarningPercent
}

func (c *DiskCheck) ID() string          { return diskCheckID }
func (c *DiskCheck) Title() string       { return "Disk Space" }
func (c *DiskCheck) Description() string { return "Monitors disk space usage on configured partitions" }
func (c *DiskCheck) Importance() string {
	return "Low disk space can cause service failures, database corruption, and log loss"
}
func (c *DiskCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *DiskCheck) IntervalSeconds() int       { return c.intervalSeconds }
func (c *DiskCheck) Platforms() []platform.Type { return nil } // all platforms

// DiskUsage is the result of measuring one partition. It is derived purely
// from a StatfsResult so the arithmetic can be tested without a filesystem.
type DiskUsage struct {
	TotalBytes     uint64 // Size of the filesystem
	UsedBytes      uint64 // Bytes consumed by files
	AvailableBytes uint64 // Bytes an unprivileged writer can still consume
	UsedPercent    int    // Capacity as `df` reports it
}

// measureUsage converts filesystem statistics into the same capacity figure
// `df` prints.
//
// `df` does not compute used/total. It computes used/(used+available), where
// used counts every allocated block and available excludes the superuser
// reserve. The reserve therefore disappears from both the numerator and the
// denominator instead of being silently counted as free space.
//
// This distinction is the reason for this function's existence. On the
// incident host — 1831.7 GB total, 221.3 GB free, 128.2 GB available — the old
// Bfree-based formula reported 87 percent while `df` reported 93 percent. The
// check sat in its warning band for an entire filesystem that was critically
// full for every unprivileged writer, including the runtime supervisor.
func measureUsage(stat *checks.StatfsResult) DiskUsage {
	if stat == nil || stat.Bsize <= 0 {
		return DiskUsage{}
	}
	blockSize := uint64(stat.Bsize)

	usage := DiskUsage{
		TotalBytes:     stat.Blocks * blockSize,
		AvailableBytes: stat.Bavail * blockSize,
	}
	if stat.Blocks >= stat.Bfree {
		usage.UsedBytes = (stat.Blocks - stat.Bfree) * blockSize
	}

	// Capacity visible to an unprivileged writer. `df` rounds the percentage
	// up, so a filesystem with any bytes in use never reports 0 percent and a
	// nearly-full one never rounds down to a comfortable-looking number.
	capacity := usage.UsedBytes + usage.AvailableBytes
	if capacity > 0 {
		usage.UsedPercent = int((usage.UsedBytes*100 + capacity - 1) / capacity)
	}
	return usage
}

// Run measures every configured partition through the injected filesystem
// reader. There is one implementation for all platforms: the platform
// difference lives entirely in RealFileSystemReader.Statfs.
func (c *DiskCheck) Run(_ context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if len(c.partitions) == 0 {
		result.Status = checks.StatusWarning
		result.Message = "No disk partitions configured to check"
		return result
	}

	var subChecks []checks.SubCheck
	worstStatus := checks.StatusOK
	partitionDetails := make([]map[string]interface{}, 0, len(c.partitions))

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

		usage := measureUsage(stat)
		partStatus, passed := c.classify(usage.UsedPercent)
		worstStatus = checks.WorstStatus(worstStatus, partStatus)

		subChecks = append(subChecks, checks.SubCheck{
			Name:   partition,
			Passed: passed,
			Detail: fmt.Sprintf("%d%% used (%s / %s, %s available)",
				usage.UsedPercent, formatBytes(usage.UsedBytes), formatBytes(usage.TotalBytes), formatBytes(usage.AvailableBytes)),
		})

		partitionDetails = append(partitionDetails, map[string]interface{}{
			"partition":      partition,
			"usedPercent":    usage.UsedPercent,
			"usedBytes":      usage.UsedBytes,
			"totalBytes":     usage.TotalBytes,
			"availableBytes": usage.AvailableBytes,
			"status":         string(partStatus),
		})
	}

	result.Status = worstStatus
	result.Details["partitions"] = partitionDetails
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	// Score is the headroom left on the fullest partition.
	score := 100
	for _, p := range partitionDetails {
		if used, ok := p["usedPercent"].(int); ok {
			if remaining := 100 - used; remaining < score {
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
		result.Message = fmt.Sprintf("Disk space warning - some partitions above %d%%", c.warningThreshold)
	case checks.StatusCritical:
		result.Message = fmt.Sprintf("Disk space critical - some partitions above %d%%", c.criticalThreshold)
	}

	return result
}

// classify maps a usage percentage to a status. Warning still counts as
// passing; only critical fails the check.
func (c *DiskCheck) classify(usedPercent int) (checks.Status, bool) {
	switch {
	case usedPercent >= c.criticalThreshold:
		return checks.StatusCritical, false
	case usedPercent >= c.warningThreshold:
		return checks.StatusWarning, true
	default:
		return checks.StatusOK, true
	}
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
			ID:   requestCleanupActionID,
			Name: "Request storage-manager reclamation",
			Description: "Reports disk pressure to storage-manager, which reclaims safe-tier space " +
				"without an operator present. This is the action that makes the disk check able to heal.",
			Dangerous: false,
			Available: true,
		},
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
	case requestCleanupActionID:
		return c.executeRequestCleanup(ctx, start)

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

// requestCleanupActionID is the heal action that makes disk pressure
// self-remediating.
//
// It replaces the old assumption, recorded in userconfig as
// "Can't auto-heal disk space", that nothing could be done automatically.
// storage-manager exists precisely to reclaim space, and it enforces its own
// safety boundary: only safe-tier providers run unattended.
const requestCleanupActionID = "request-cleanup"

// executeRequestCleanup reports current pressure to storage-manager and returns
// what it reclaimed.
//
// The band reported is derived from the same thresholds the check classifies
// with, so the heal action escalates exactly as far as the observation
// justifies: a critical partition authorises unattended reclamation, a warning
// one only asks for a preview.
func (c *DiskCheck) executeRequestCleanup(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  requestCleanupActionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	if c.cleanup == nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "storage-manager client not configured"
		result.Message = "Cannot request cleanup"
		return result
	}

	worst, worstPartition, err := c.worstPartition()
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Could not measure disk pressure"
		return result
	}

	band, ok := c.bandFor(worst.UsedPercent)
	if !ok {
		result.Duration = time.Since(start)
		result.Success = true
		result.Message = fmt.Sprintf("No cleanup requested: %s is at %d%%, below the warning threshold of %d%%",
			worstPartition, worst.UsedPercent, c.warningThreshold)
		return result
	}

	outcome, err := c.cleanup.ReportPressure(ctx, cleanupmanager.Report{
		SourceScenario: "vrooli-autoheal",
		Partition:      worstPartition,
		UsedPercent:    float64(worst.UsedPercent),
		Band:           band,
		AvailableBytes: int64(worst.AvailableBytes),
	})
	result.Duration = time.Since(start)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to reach storage-manager"
		return result
	}

	result.Success = true
	result.Output = fmt.Sprintf("partition=%s used=%d%% band=%s action=%s reclaimed=%s withheld=%v",
		worstPartition, worst.UsedPercent, band, outcome.Action, formatBytes(uint64(outcome.ReclaimedBytes)), outcome.ProvidersWithheld)
	result.Message = fmt.Sprintf("storage-manager %s; reclaimed %s", outcome.Action, formatBytes(uint64(outcome.ReclaimedBytes)))
	return result
}

// worstPartition measures every configured partition and returns the fullest.
// Remediation should target the partition under the most pressure, not
// whichever happens to be listed first.
func (c *DiskCheck) worstPartition() (DiskUsage, string, error) {
	var (
		worst     DiskUsage
		worstName string
		measured  bool
		lastErr   error
	)
	for _, partition := range c.partitions {
		stat, err := c.fsReader.Statfs(partition)
		if err != nil {
			lastErr = err
			continue
		}
		usage := measureUsage(stat)
		if !measured || usage.UsedPercent > worst.UsedPercent {
			worst, worstName, measured = usage, partition, true
		}
	}
	if !measured {
		if lastErr != nil {
			return DiskUsage{}, "", fmt.Errorf("no partition could be measured: %w", lastErr)
		}
		return DiskUsage{}, "", fmt.Errorf("no partitions configured")
	}
	return worst, worstName, nil
}

// bandFor maps a usage percentage onto the band to report, using the same
// thresholds the check classifies with. Usage below the warning threshold
// reports no band at all, so healthy disks never request cleanup.
func (c *DiskCheck) bandFor(usedPercent int) (cleanupmanager.Band, bool) {
	switch {
	case usedPercent >= c.criticalThreshold:
		return cleanupmanager.BandCritical, true
	case usedPercent >= c.warningThreshold:
		return cleanupmanager.BandHigh, true
	default:
		return "", false
	}
}

// Package system provides system-level health checks
// [REQ:SYSTEM-CLAUDE-CACHE-001] [REQ:HEAL-ACTION-001]
package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ClaudeCacheCheck monitors the Claude Code cache directory for excessive files
// that can cause ENOSPC (file watcher) errors.
type ClaudeCacheCheck struct {
	warningFileCount  int // File count to trigger warning
	criticalFileCount int // File count to trigger critical
	todoMaxAgeDays    int // Age in days for todo cleanup
	historyMaxAgeDays int // Age in days for history cleanup
	shellMaxAgeDays   int // Age in days for shell snapshot cleanup
}

// ClaudeCacheCheckOption configures a ClaudeCacheCheck.
type ClaudeCacheCheckOption func(*ClaudeCacheCheck)

// WithClaudeCacheThresholds sets warning and critical thresholds.
func WithClaudeCacheThresholds(warning, critical int) ClaudeCacheCheckOption {
	return func(c *ClaudeCacheCheck) {
		c.warningFileCount = warning
		c.criticalFileCount = critical
	}
}

// NewClaudeCacheCheck creates a Claude Code cache check.
// Default thresholds: warning at 5000 files, critical at 10000 files.
func NewClaudeCacheCheck(opts ...ClaudeCacheCheckOption) *ClaudeCacheCheck {
	c := &ClaudeCacheCheck{
		warningFileCount:  5000,
		criticalFileCount: 10000,
		todoMaxAgeDays:    7,
		historyMaxAgeDays: 30,
		shellMaxAgeDays:   7,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ClaudeCacheCheck) ID() string    { return "system-claude-cache" }
func (c *ClaudeCacheCheck) Title() string { return "Claude Code Cache" }
func (c *ClaudeCacheCheck) Description() string {
	return "Monitors Claude Code cache directory for excessive files that cause file watcher errors"
}
func (c *ClaudeCacheCheck) Importance() string {
	return "Excessive cache files exhaust inotify watchers, causing ENOSPC errors and Claude Code startup failures"
}
func (c *ClaudeCacheCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *ClaudeCacheCheck) IntervalSeconds() int       { return 3600 } // Check hourly
func (c *ClaudeCacheCheck) Platforms() []platform.Type { return nil }  // All platforms

func (c *ClaudeCacheCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Get Claude cache directory
	home, err := os.UserHomeDir()
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Could not determine home directory"
		result.Details["error"] = err.Error()
		return result
	}

	claudeDir := filepath.Join(home, ".claude")
	result.Details["claudeDir"] = claudeDir

	// Check if directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		result.Status = checks.StatusOK
		result.Message = "Claude Code cache directory does not exist (Claude Code not used)"
		result.Details["exists"] = false
		return result
	}
	result.Details["exists"] = true

	// Count files and analyze by category
	stats, err := c.analyzeCache(claudeDir)
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Failed to analyze Claude Code cache"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["totalFiles"] = stats.totalFiles
	result.Details["totalSize"] = stats.totalSize
	result.Details["totalSizeHuman"] = formatCacheBytes(stats.totalSize)
	result.Details["todoFiles"] = stats.todoFiles
	result.Details["historyFiles"] = stats.historyFiles
	result.Details["shellFiles"] = stats.shellFiles
	result.Details["staleFiles"] = stats.staleFiles
	result.Details["warningThreshold"] = c.warningFileCount
	result.Details["criticalThreshold"] = c.criticalFileCount

	// Calculate score
	score := 100
	if stats.totalFiles > c.criticalFileCount {
		score = 0
	} else if stats.totalFiles > c.warningFileCount {
		// Scale score between 50 and 0 based on proximity to critical
		excess := stats.totalFiles - c.warningFileCount
		range_ := c.criticalFileCount - c.warningFileCount
		score = 50 - (excess * 50 / range_)
	} else if stats.totalFiles > c.warningFileCount/2 {
		// Scale score between 100 and 50 as we approach warning
		score = 100 - (stats.totalFiles * 50 / c.warningFileCount)
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "file-count",
				Passed: stats.totalFiles < c.criticalFileCount,
				Detail: fmt.Sprintf("%d files in cache (%s)", stats.totalFiles, formatCacheBytes(stats.totalSize)),
			},
			{
				Name:   "stale-files",
				Passed: stats.staleFiles < 1000,
				Detail: fmt.Sprintf("%d stale files eligible for cleanup", stats.staleFiles),
			},
		},
	}

	// Determine status
	switch {
	case stats.totalFiles >= c.criticalFileCount:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Claude Code cache has %d files (critical threshold: %d) - may cause ENOSPC errors",
			stats.totalFiles, c.criticalFileCount)
	case stats.totalFiles >= c.warningFileCount:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Claude Code cache has %d files (warning threshold: %d) - cleanup recommended",
			stats.totalFiles, c.warningFileCount)
	default:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Claude Code cache is healthy (%d files, %s)",
			stats.totalFiles, formatCacheBytes(stats.totalSize))
	}

	return result
}

// cacheStats holds statistics about the Claude cache
type cacheStats struct {
	totalFiles   int
	totalSize    int64
	todoFiles    int
	historyFiles int
	shellFiles   int
	staleFiles   int // Files older than their respective thresholds
}

// analyzeCache walks the Claude cache directory and gathers statistics
func (c *ClaudeCacheCheck) analyzeCache(dir string) (*cacheStats, error) {
	stats := &cacheStats{}
	now := time.Now()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors (permission denied, etc.)
		}
		if info.IsDir() {
			return nil
		}

		stats.totalFiles++
		stats.totalSize += info.Size()

		// Categorize by directory
		relPath, _ := filepath.Rel(dir, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		if len(parts) > 0 {
			switch parts[0] {
			case "todos":
				stats.todoFiles++
				if info.ModTime().Before(now.AddDate(0, 0, -c.todoMaxAgeDays)) {
					stats.staleFiles++
				}
			case "file-history":
				stats.historyFiles++
				if info.ModTime().Before(now.AddDate(0, 0, -c.historyMaxAgeDays)) {
					stats.staleFiles++
				}
			case "shell-snapshots":
				stats.shellFiles++
				if info.ModTime().Before(now.AddDate(0, 0, -c.shellMaxAgeDays)) {
					stats.staleFiles++
				}
			}
		}

		return nil
	})

	return stats, err
}

// formatCacheBytes converts bytes to human-readable format
func formatCacheBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// RecoveryActions returns available recovery actions for Claude cache
// [REQ:HEAL-ACTION-001]
func (c *ClaudeCacheCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasStaleFiles := false
	if lastResult != nil {
		if stale, ok := lastResult.Details["staleFiles"].(int); ok && stale > 0 {
			hasStaleFiles = true
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "cleanup-stale",
			Name:        "Cleanup Stale Files",
			Description: fmt.Sprintf("Remove stale files (todos >%dd, history >%dd, shells >%dd)",
				c.todoMaxAgeDays, c.historyMaxAgeDays, c.shellMaxAgeDays),
			Dangerous:   false,
			Available:   hasStaleFiles,
		},
		{
			ID:          "cleanup-all",
			Name:        "Full Cleanup",
			Description: "Remove all cleanable cache files (keeps recent data)",
			Dangerous:   true,
			Available:   true,
		},
		{
			ID:          "analyze",
			Name:        "Analyze Cache",
			Description: "Get detailed breakdown of cache contents",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *ClaudeCacheCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Could not determine home directory"
		result.Duration = time.Since(start)
		return result
	}

	claudeDir := filepath.Join(home, ".claude")

	switch actionID {
	case "cleanup-stale":
		return c.executeCleanupStale(claudeDir, start)

	case "cleanup-all":
		return c.executeCleanupAll(claudeDir, start)

	case "analyze":
		return c.executeAnalyze(claudeDir, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeCleanupStale removes only stale files
func (c *ClaudeCacheCheck) executeCleanupStale(claudeDir string, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "cleanup-stale",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var output strings.Builder
	var removed, failed int
	now := time.Now()

	// Define cleanup targets
	targets := []struct {
		dir    string
		maxAge time.Duration
	}{
		{"todos", time.Duration(c.todoMaxAgeDays) * 24 * time.Hour},
		{"file-history", time.Duration(c.historyMaxAgeDays) * 24 * time.Hour},
		{"shell-snapshots", time.Duration(c.shellMaxAgeDays) * 24 * time.Hour},
	}

	for _, target := range targets {
		targetDir := filepath.Join(claudeDir, target.dir)
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			continue
		}

		output.WriteString(fmt.Sprintf("=== Cleaning %s (older than %d days) ===\n", target.dir, int(target.maxAge.Hours()/24)))

		cutoff := now.Add(-target.maxAge)
		err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			if info.ModTime().Before(cutoff) {
				if err := os.Remove(path); err != nil {
					failed++
					output.WriteString(fmt.Sprintf("FAILED: %s - %v\n", path, err))
				} else {
					removed++
				}
			}
			return nil
		})

		if err != nil {
			output.WriteString(fmt.Sprintf("Error walking %s: %v\n", target.dir, err))
		}
	}

	// Clean empty directories
	c.cleanEmptyDirs(claudeDir)

	result.Duration = time.Since(start)
	result.Output = output.String()
	result.Success = true
	result.Message = fmt.Sprintf("Cleanup complete: removed %d files, %d failed", removed, failed)
	return result
}

// executeCleanupAll removes all cleanable cache files (keeps recent)
func (c *ClaudeCacheCheck) executeCleanupAll(claudeDir string, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "cleanup-all",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var output strings.Builder
	var removed, failed int

	// Clean all files in these directories (but keep recent 1 day)
	targets := []string{"todos", "file-history", "shell-snapshots"}
	cutoff := time.Now().Add(-24 * time.Hour)

	for _, target := range targets {
		targetDir := filepath.Join(claudeDir, target)
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			continue
		}

		output.WriteString(fmt.Sprintf("=== Cleaning %s ===\n", target))

		err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Keep files from the last day
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(path); err != nil {
					failed++
					output.WriteString(fmt.Sprintf("FAILED: %s - %v\n", path, err))
				} else {
					removed++
				}
			}
			return nil
		})

		if err != nil {
			output.WriteString(fmt.Sprintf("Error walking %s: %v\n", target, err))
		}
	}

	// Clean empty directories
	c.cleanEmptyDirs(claudeDir)

	result.Duration = time.Since(start)
	result.Output = output.String()
	result.Success = true
	result.Message = fmt.Sprintf("Full cleanup complete: removed %d files, %d failed", removed, failed)
	return result
}

// executeAnalyze returns detailed cache breakdown
func (c *ClaudeCacheCheck) executeAnalyze(claudeDir string, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "analyze",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("=== Claude Code Cache Analysis ===\n"))
	output.WriteString(fmt.Sprintf("Location: %s\n\n", claudeDir))

	// Analyze each subdirectory
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to read cache directory"
		result.Duration = time.Since(start)
		return result
	}

	totalFiles := 0
	var totalSize int64

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subdir := filepath.Join(claudeDir, entry.Name())
		files := 0
		var size int64

		filepath.Walk(subdir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			files++
			size += info.Size()
			return nil
		})

		totalFiles += files
		totalSize += size
		output.WriteString(fmt.Sprintf("%-20s: %6d files (%s)\n", entry.Name(), files, formatCacheBytes(size)))
	}

	output.WriteString(fmt.Sprintf("\n%-20s: %6d files (%s)\n", "TOTAL", totalFiles, formatCacheBytes(totalSize)))
	output.WriteString(fmt.Sprintf("\nWarning threshold: %d files\n", c.warningFileCount))
	output.WriteString(fmt.Sprintf("Critical threshold: %d files\n", c.criticalFileCount))

	result.Duration = time.Since(start)
	result.Output = output.String()
	result.Success = true
	result.Message = fmt.Sprintf("Analysis complete: %d files, %s", totalFiles, formatCacheBytes(totalSize))
	return result
}

// cleanEmptyDirs removes empty directories
func (c *ClaudeCacheCheck) cleanEmptyDirs(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == dir {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err == nil && len(entries) == 0 {
			os.Remove(path)
		}
		return nil
	})
}

// Ensure ClaudeCacheCheck implements HealableCheck
var _ checks.HealableCheck = (*ClaudeCacheCheck)(nil)

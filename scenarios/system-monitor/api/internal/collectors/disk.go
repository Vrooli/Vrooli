package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"system-monitor-api/internal/models"
)

// DiskCollector collects disk metrics
type DiskCollector struct {
	BaseCollector
}

// NewDiskCollector creates a new disk collector
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{
		BaseCollector: NewBaseCollector("disk", 30*time.Second),
	}
}

// Collect gathers disk metrics
func (c *DiskCollector) Collect(ctx context.Context) (*MetricData, error) {
	diskUsage := c.getDiskUsage()
	ioStats := c.getIOStats()
	fileDescriptors := c.getFileDescriptors()

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "disk",
		Values: map[string]interface{}{
			"usage":            diskUsage,
			"io_stats":         ioStats,
			"file_descriptors": fileDescriptors,
		},
	}, nil
}

// getDiskUsage returns disk usage information
func (c *DiskCollector) getDiskUsage() map[string]interface{} {
	usage := map[string]interface{}{
		"used":    int64(0),
		"total":   int64(0),
		"free":    int64(0),
		"percent": float64(0),
	}

	if runtime.GOOS != "linux" {
		usage["percent"] = 67.8
		return usage
	}

	cmd := exec.Command("bash", "-c", "df -B1 / | tail -1 | awk '{print $2, $3, $4}'")
	output, err := cmd.Output()
	if err != nil {
		return usage
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) >= 3 {
		total, _ := strconv.ParseInt(fields[0], 10, 64)
		used, _ := strconv.ParseInt(fields[1], 10, 64)
		free, _ := strconv.ParseInt(fields[2], 10, 64)

		usage["total"] = total
		usage["used"] = used
		usage["free"] = free
		if total > 0 {
			usage["percent"] = float64(used) / float64(total) * 100
		}
	}

	return usage
}

// getIOStats returns disk I/O statistics
func (c *DiskCollector) getIOStats() map[string]interface{} {
	ioStats := map[string]interface{}{
		"read_mb_per_sec":  float64(0),
		"write_mb_per_sec": float64(0),
		"io_wait_percent":  float64(0),
		"queue_depth":      float64(0),
		"reads_per_sec":    float64(0),
		"writes_per_sec":   float64(0),
	}

	if runtime.GOOS != "linux" {
		ioStats["read_mb_per_sec"] = 15.0
		ioStats["write_mb_per_sec"] = 8.0
		ioStats["io_wait_percent"] = 2.5
		return ioStats
	}

	// Get I/O wait percentage from /proc/stat
	cmd := exec.Command("bash", "-c", "grep '^cpu ' /proc/stat | awk '{print ($5/($2+$3+$4+$5+$6+$7+$8))*100}'")
	output, err := cmd.Output()
	if err == nil {
		ioStats["io_wait_percent"], _ = strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	}

	// Get iostat data if available
	cmd = exec.Command("bash", "-c", "iostat -x 1 2 | grep -A1 avg-cpu | tail -1")
	output, err = cmd.Output()
	if err == nil {
		// Parse iostat output
		fields := strings.Fields(string(output))
		if len(fields) >= 6 {
			ioStats["io_wait_percent"], _ = strconv.ParseFloat(fields[3], 64)
		}
	}

	// Mock additional stats for now
	ioStats["read_mb_per_sec"] = 15.0
	ioStats["write_mb_per_sec"] = 8.0
	ioStats["queue_depth"] = 0.2

	return ioStats
}

// getFileDescriptors returns file descriptor usage
func (c *DiskCollector) getFileDescriptors() map[string]interface{} {
	fdInfo := map[string]interface{}{
		"used":    0,
		"max":     65536,
		"percent": float64(0),
	}

	if runtime.GOOS != "linux" {
		fdInfo["used"] = 1024
		fdInfo["percent"] = 1.56
		return fdInfo
	}

	// Get current FD count from /proc/sys/fs/file-nr (much faster than lsof)
	data, err := os.ReadFile("/proc/sys/fs/file-nr")
	if err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			used, _ := strconv.Atoi(parts[0])
			fdInfo["used"] = used
		}
	}

	// Get max FD limit
	data, err = os.ReadFile("/proc/sys/fs/file-max")
	if err == nil {
		max, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		if max > 0 {
			fdInfo["max"] = max
		}
	}

	// Calculate percentage
	if used, ok := fdInfo["used"].(int); ok {
		if max, ok := fdInfo["max"].(int); ok && max > 0 {
			fdInfo["percent"] = float64(used) / float64(max) * 100
		}
	}

	return fdInfo
}

// GetDiskPartitions returns information about disk partitions
func GetDiskPartitions() ([]map[string]interface{}, error) {
	if runtime.GOOS != "linux" {
		return []map[string]interface{}{}, nil
	}

	cmd := exec.Command("bash", "-c", "df -B1 --output=source,size,used,avail,pcent,target | tail -n +2")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var partitions []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		sizeBytes, _ := strconv.ParseInt(fields[1], 10, 64)
		usedBytes, _ := strconv.ParseInt(fields[2], 10, 64)
		availBytes, _ := strconv.ParseInt(fields[3], 10, 64)
		percent := strings.TrimSuffix(fields[4], "%")
		percentVal, _ := strconv.ParseFloat(percent, 64)
		mountPoint := strings.Join(fields[5:], " ")

		partitions = append(partitions, map[string]interface{}{
			"device":          fields[0],
			"size_bytes":      sizeBytes,
			"size_human":      formatBytesHuman(sizeBytes),
			"used_bytes":      usedBytes,
			"used_human":      formatBytesHuman(usedBytes),
			"available_bytes": availBytes,
			"available_human": formatBytesHuman(availBytes),
			"use_percent":     percentVal,
			"mount_point":     mountPoint,
		})
	}

	return partitions, nil
}

func formatBytesHuman(bytesValue int64) string {
	if bytesValue <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(bytesValue)
	index := 0
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}
	if index == 0 {
		return fmt.Sprintf("%d %s", bytesValue, units[index])
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f %s", value, units[index])
	}
	return fmt.Sprintf("%.1f %s", value, units[index])
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// GetLargestDirectories returns the heaviest directories inside mount up to the specified depth.
func GetLargestDirectories(mount string, depth, limit int) ([]models.DiskUsageEntry, error) {
	entries := []models.DiskUsageEntry{}
	if runtime.GOOS != "linux" {
		return entries, nil
	}
	if mount == "" {
		mount = "/"
	}
	if depth <= 0 {
		depth = 2
	}
	if limit <= 0 {
		limit = 8
	}
	cmdStr := fmt.Sprintf("du -x -B1 --max-depth=%d %s 2>/dev/null | sort -nr | head -n %d", depth, shellQuote(mount), limit+1)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return entries, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sizeBytes, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		path := strings.Join(fields[1:], " ")
		cleanPath := filepath.Clean(path)
		if cleanPath == filepath.Clean(mount) {
			continue
		}
		entries = append(entries, models.DiskUsageEntry{
			Path:      cleanPath,
			SizeBytes: sizeBytes,
			SizeHuman: formatBytesHuman(sizeBytes),
			Category:  "directory",
		})
		if len(entries) >= limit {
			break
		}
	}
	return entries, nil
}

// GetLargestFiles returns the largest files within a mount point (size threshold 50MB).
func GetLargestFiles(mount string, limit int) ([]models.DiskUsageEntry, error) {
	entries := []models.DiskUsageEntry{}
	if runtime.GOOS != "linux" {
		return entries, nil
	}
	if mount == "" {
		mount = "/"
	}
	if limit <= 0 {
		limit = 8
	}
	cmdStr := fmt.Sprintf("find %s -xdev -type f -size +52428800c -printf '%%s\\t%%p\\n' 2>/dev/null | sort -nr | head -n %d", shellQuote(mount), limit)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return entries, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		sizeBytes, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			continue
		}
		path := filepath.Clean(strings.TrimSpace(parts[1]))
		entries = append(entries, models.DiskUsageEntry{
			Path:      path,
			SizeBytes: sizeBytes,
			SizeHuman: formatBytesHuman(sizeBytes),
			Category:  "file",
		})
		if len(entries) >= limit {
			break
		}
	}
	return entries, nil
}

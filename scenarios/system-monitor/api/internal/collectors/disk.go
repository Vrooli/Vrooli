package collectors

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// DiskCollector collects disk metrics
type DiskCollector struct {
	BaseCollector
	lastInotifySample time.Time
	lastInotifyStats  map[string]interface{}
	inotifyInterval   time.Duration
}

// NewDiskCollector creates a new disk collector
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{
		BaseCollector:   NewBaseCollector("disk", 30*time.Second),
		inotifyInterval: 5 * time.Minute,
	}
}

// Collect gathers disk metrics
func (c *DiskCollector) Collect(ctx context.Context) (*MetricData, error) {
	diskUsage := c.getDiskUsage()
	ioStats := c.getIOStats()
	fileDescriptors := c.getFileDescriptors()
	inotifyStats := c.getInotifyStats()

	return &MetricData{
		CollectorName: c.GetName(),
		Timestamp:     time.Now(),
		Type:          "disk",
		Values: map[string]interface{}{
			"usage":            diskUsage,
			"io_stats":         ioStats,
			"file_descriptors": fileDescriptors,
			"inotify_watchers": inotifyStats,
		},
		Tags: map[string]string{
			"os":     collectorOS,
			"source": "native filesystem statistics",
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

	measured, err := ReadDiskUsage(RootMountPath())
	if err != nil || measured.TotalBytes <= 0 {
		usage["status"] = "failed"
		usage["reason"] = "disk usage could not be measured"
		if err != nil {
			usage["reason"] = err.Error()
		}
		return usage
	}
	usage["total"] = measured.TotalBytes
	usage["used"] = measured.UsedBytes
	// "free" is the space a writer can actually use, which is what every
	// consumer of this metric cares about.
	usage["free"] = measured.AvailableBytes
	usage["percent"] = measured.UsedPercent

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
	if collectorOS != "linux" {
		ioStats["status"] = "unsupported"
		ioStats["reason"] = "disk I/O counters are not implemented for this platform"
		return ioStats
	}

	if wait, ok := readIOWaitPercent(); ok {
		ioStats["io_wait_percent"] = wait
	}

	return ioStats
}

// getFileDescriptors returns file descriptor usage
func (c *DiskCollector) getFileDescriptors() map[string]interface{} {
	fdInfo := map[string]interface{}{
		"used":    0,
		"max":     65536,
		"percent": float64(0),
	}
	if collectorOS != "linux" {
		fdInfo["status"] = "unsupported"
		fdInfo["reason"] = "system-wide file descriptor counters are not implemented for this platform"
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

// getInotifyStats returns inotify watcher and instance utilisation
func (c *DiskCollector) getInotifyStats() map[string]interface{} {
	if !c.lastInotifySample.IsZero() && time.Since(c.lastInotifySample) < c.inotifyInterval {
		return cloneMap(c.lastInotifyStats)
	}

	stats := map[string]interface{}{
		"supported":         collectorOS == "linux",
		"watches_used":      0,
		"watches_max":       0,
		"watches_percent":   float64(0),
		"instances_used":    0,
		"instances_max":     0,
		"instances_percent": float64(0),
	}
	if collectorOS != "linux" {
		stats["reason"] = "inotify is Linux-specific"
		return stats
	}

	watchesMax, err := readIntFromFile("/proc/sys/fs/inotify/max_user_watches")
	if err == nil {
		stats["watches_max"] = watchesMax
	}

	instancesMax, err := readIntFromFile("/proc/sys/fs/inotify/max_user_instances")
	if err == nil {
		stats["instances_max"] = instancesMax
	}

	watchesUsed, instancesUsed := countInotifyUsage(watchesMax, instancesMax)
	stats["watches_used"] = watchesUsed
	stats["instances_used"] = instancesUsed

	if watchesMax > 0 {
		stats["watches_percent"] = math.Min(100, (float64(watchesUsed)/float64(watchesMax))*100)
	}

	if instancesMax > 0 {
		stats["instances_percent"] = math.Min(100, (float64(instancesUsed)/float64(instancesMax))*100)
	}

	c.lastInotifySample = time.Now()
	c.lastInotifyStats = cloneMap(stats)
	return stats
}

func readIntFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func countInotifyUsage(watchLimit, instanceLimit int) (int, int) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, 0
	}

	watchesUsed := 0
	instancesUsed := 0

	for _, entry := range entries {
		pid, ok := procPID(entry)
		if !ok {
			continue
		}

		if countPIDInotifyUsage(pid, watchLimit, instanceLimit, &watchesUsed, &instancesUsed) {
			return watchLimit, instanceLimit
		}
	}

	return clampToLimit(watchesUsed, watchLimit), clampToLimit(instancesUsed, instanceLimit)
}

// procPID returns the numeric PID name for a /proc directory entry, reporting
// ok=false for non-directory or non-numeric entries.
func procPID(entry os.DirEntry) (string, bool) {
	if !entry.IsDir() {
		return "", false
	}
	pid := entry.Name()
	if _, err := strconv.Atoi(pid); err != nil {
		return "", false
	}
	return pid, true
}

// countPIDInotifyUsage accumulates inotify instance/watch usage for a single
// PID into the supplied totals. It returns true when both limits have been
// reached, signalling the caller to short-circuit.
func countPIDInotifyUsage(pid string, watchLimit, instanceLimit int, watchesUsed, instancesUsed *int) bool {
	fdinfoPath := filepath.Join("/proc", pid, "fdinfo")
	fdEntries, err := os.ReadDir(fdinfoPath)
	if err != nil {
		return false
	}

	for _, fdEntry := range fdEntries {
		watchers, ok := scanInotifyFD(filepath.Join(fdinfoPath, fdEntry.Name()))
		if !ok {
			continue
		}

		*instancesUsed++
		*watchesUsed += watchers

		if watchLimit > 0 && *watchesUsed >= watchLimit && instanceLimit > 0 && *instancesUsed >= instanceLimit {
			return true
		}
	}

	return false
}

// scanInotifyFD inspects a single fdinfo file, returning the number of watchers
// it represents and ok=true when the descriptor is an inotify instance. A
// descriptor with no explicit watches counts as a single watcher.
func scanInotifyFD(filePath string) (int, bool) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasInotify := false
	watchersInFile := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "inotify") {
			hasInotify = true
			if strings.HasPrefix(line, "inotify wd:") {
				watchersInFile++
			}
		}
	}

	if !hasInotify {
		return 0, false
	}
	if watchersInFile == 0 {
		watchersInFile = 1
	}
	return watchersInFile, true
}

// clampToLimit caps value at limit when limit is positive.
func clampToLimit(value, limit int) int {
	if limit > 0 && value > limit {
		return limit
	}
	return value
}

func cloneMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return map[string]interface{}{}
	}
	clone := make(map[string]interface{}, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func readIOWaitPercent() (float64, bool) {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			return 0, false
		}
		var total uint64
		for _, field := range fields[1:] {
			value, _ := strconv.ParseUint(field, 10, 64)
			total += value
		}
		iowait, _ := strconv.ParseUint(fields[5], 10, 64)
		if total == 0 {
			return 0, false
		}
		return float64(iowait) / float64(total) * 100, true
	}
	return 0, false
}

// GetDiskPartitions returns information about disk partitions
func GetDiskPartitions() ([]map[string]interface{}, error) {
	if collectorOS != "linux" {
		return []map[string]interface{}{}, nil
	}

	mounts, err := readProcMounts()
	if err != nil {
		return nil, err
	}

	partitions := make([]map[string]interface{}, 0, len(mounts))
	for _, mount := range mounts {
		measured, err := ReadDiskUsage(mount.mountPoint)
		if err != nil || measured.TotalBytes <= 0 {
			continue
		}

		partitions = append(partitions, map[string]interface{}{
			"device":          mount.device,
			"size_bytes":      measured.TotalBytes,
			"size_human":      formatBytesHuman(measured.TotalBytes),
			"used_bytes":      measured.UsedBytes,
			"used_human":      formatBytesHuman(measured.UsedBytes),
			"available_bytes": measured.AvailableBytes,
			"available_human": formatBytesHuman(measured.AvailableBytes),
			"use_percent":     measured.UsedPercent,
			"mount_point":     mount.mountPoint,
		})
	}

	return partitions, nil
}

type procMount struct {
	device     string
	mountPoint string
}

func readProcMounts() ([]procMount, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mounts []procMount
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		mounts = append(mounts, procMount{
			device:     unescapeProcMount(fields[0]),
			mountPoint: unescapeProcMount(fields[1]),
		})
	}
	return mounts, scanner.Err()
}

func unescapeProcMount(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(value)
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
	if collectorOS != "linux" {
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
	output, err := commandOutput(context.Background(), 2*time.Second, "bash", "-c", cmdStr)
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
	if collectorOS != "linux" {
		return entries, nil
	}
	if mount == "" {
		mount = "/"
	}
	if limit <= 0 {
		limit = 8
	}
	cmdStr := fmt.Sprintf("find %s -xdev -type f -size +52428800c -printf '%%s\\t%%p\\n' 2>/dev/null | sort -nr | head -n %d", shellQuote(mount), limit)
	output, err := commandOutput(context.Background(), 2*time.Second, "bash", "-c", cmdStr)
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

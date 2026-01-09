package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

// Bundle size thresholds for warnings
const (
	BundleSizeWarningThreshold  = 500 * 1024 * 1024  // 500MB - warn
	BundleSizeCriticalThreshold = 1024 * 1024 * 1024 // 1GB - critical warning
)

// BundleSizeWarning represents a warning about bundle size.
type BundleSizeWarning struct {
	Level      string          `json:"level"` // "warning" or "critical"
	Message    string          `json:"message"`
	TotalBytes int64           `json:"total_bytes"`
	TotalHuman string          `json:"total_human"`
	LargeFiles []LargeFileInfo `json:"large_files,omitempty"`
}

// LargeFileInfo describes a file contributing significantly to bundle size.
type LargeFileInfo struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
}

// calculateBundleSize walks the bundle directory and returns total size and large files.
// Large files are those over 10MB.
func calculateBundleSize(bundleDir string) (int64, []LargeFileInfo) {
	const largeFileThreshold = 10 * 1024 * 1024 // 10MB

	var totalSize int64
	var largeFiles []LargeFileInfo

	_ = filepath.WalkDir(bundleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		size := info.Size()
		totalSize += size

		if size >= largeFileThreshold {
			relPath, _ := filepath.Rel(bundleDir, path)
			largeFiles = append(largeFiles, LargeFileInfo{
				Path:      relPath,
				SizeBytes: size,
				SizeHuman: humanReadableSize(size),
			})
		}

		return nil
	})

	// Sort large files by size descending
	sort.Slice(largeFiles, func(i, j int) bool {
		return largeFiles[i].SizeBytes > largeFiles[j].SizeBytes
	})

	// Keep only top 10 largest files
	if len(largeFiles) > 10 {
		largeFiles = largeFiles[:10]
	}

	return totalSize, largeFiles
}

// checkBundleSizeWarning returns a warning if the bundle exceeds size thresholds.
func checkBundleSizeWarning(totalSize int64, largeFiles []LargeFileInfo) *BundleSizeWarning {
	if totalSize < BundleSizeWarningThreshold {
		return nil
	}

	var level, message string
	if totalSize >= BundleSizeCriticalThreshold {
		level = "critical"
		message = fmt.Sprintf("Bundle size (%s) exceeds 1GB. This will result in very large installers and slow downloads. Consider removing large assets or using delta updates.",
			humanReadableSize(totalSize))
	} else {
		level = "warning"
		message = fmt.Sprintf("Bundle size (%s) exceeds 500MB. Consider optimizing assets to reduce download size.",
			humanReadableSize(totalSize))
	}

	return &BundleSizeWarning{
		Level:      level,
		Message:    message,
		TotalBytes: totalSize,
		TotalHuman: humanReadableSize(totalSize),
		LargeFiles: largeFiles,
	}
}

// humanReadableSize converts bytes to human-readable format.
func humanReadableSize(bytes int64) string {
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

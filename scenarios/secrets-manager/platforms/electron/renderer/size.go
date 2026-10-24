package bundle

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

// Bundle size thresholds for warnings
const (
	SizeWarningThreshold  = 500 * 1024 * 1024  // 500MB - warn
	SizeCriticalThreshold = 1024 * 1024 * 1024 // 1GB - critical warning
	LargeFileThreshold    = 10 * 1024 * 1024   // 10MB
)

// defaultSizeCalculator is the default implementation of SizeCalculator.
type defaultSizeCalculator struct{}

// Calculate walks the bundle directory and returns total size and large files.
func (c *defaultSizeCalculator) Calculate(bundleDir string) (int64, []LargeFileInfo) {
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

		if size >= LargeFileThreshold {
			relPath, _ := filepath.Rel(bundleDir, path)
			largeFiles = append(largeFiles, LargeFileInfo{
				Path:      relPath,
				SizeBytes: size,
				SizeHuman: HumanReadableSize(size),
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

// CheckWarning returns a warning if the bundle exceeds size thresholds.
func (c *defaultSizeCalculator) CheckWarning(totalSize int64, largeFiles []LargeFileInfo) *SizeWarning {
	if totalSize < SizeWarningThreshold {
		return nil
	}

	var level, message string
	if totalSize >= SizeCriticalThreshold {
		level = "critical"
		message = fmt.Sprintf("Bundle size (%s) exceeds 1GB. This will result in very large installers and slow downloads. Consider removing large assets or using delta updates.",
			HumanReadableSize(totalSize))
	} else {
		level = "warning"
		message = fmt.Sprintf("Bundle size (%s) exceeds 500MB. Consider optimizing assets to reduce download size.",
			HumanReadableSize(totalSize))
	}

	return &SizeWarning{
		Level:      level,
		Message:    message,
		TotalBytes: totalSize,
		TotalHuman: HumanReadableSize(totalSize),
		LargeFiles: largeFiles,
	}
}

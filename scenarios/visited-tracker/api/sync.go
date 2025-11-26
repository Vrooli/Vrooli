package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/google/uuid"
)

const (
	defaultPriorityWeight = 1.0
)

// expandBraces expands brace patterns like "*.{ts,tsx}" into ["*.ts", "*.tsx"]
func expandBraces(pattern string) []string {
	// Simple brace expansion for patterns like "*.{a,b,c}"
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")

	// No braces found, return pattern as-is
	if start == -1 || end == -1 || end <= start {
		return []string{pattern}
	}

	// Extract prefix, options, and suffix
	prefix := pattern[:start]
	suffix := pattern[end+1:]
	optionsStr := pattern[start+1 : end]
	options := strings.Split(optionsStr, ",")

	// Build expanded patterns
	var expanded []string
	for _, opt := range options {
		expanded = append(expanded, prefix+strings.TrimSpace(opt)+suffix)
	}

	return expanded
}

// getCampaignBaseDir returns the absolute base directory for a campaign
func getCampaignBaseDir(campaign *Campaign) string {
	baseDir := "."
	if campaign.Location != nil && *campaign.Location != "" {
		baseDir = *campaign.Location
	}

	// Make baseDir absolute if it's not already
	if !filepath.IsAbs(baseDir) {
		cwd, _ := os.Getwd()
		baseDir = filepath.Join(cwd, baseDir)
	}

	return baseDir
}

// normalizeFilePath normalizes a file path relative to the campaign's base directory
// Returns both the relative path (for storage) and absolute path (for comparison)
func normalizeFilePath(campaign *Campaign, filePath string) (relativePath string, absolutePath string) {
	baseDir := getCampaignBaseDir(campaign)

	// Determine absolute path
	if filepath.IsAbs(filePath) {
		absolutePath = filepath.Clean(filePath)
	} else {
		// Join with baseDir, not cwd
		absolutePath = filepath.Clean(filepath.Join(baseDir, filePath))
	}

	// Calculate relative path from baseDir
	relPath, err := filepath.Rel(baseDir, absolutePath)
	if err != nil {
		// If we can't make it relative, use the cleaned file path
		relativePath = filepath.Clean(filePath)
		logger.Printf("⚠️ Could not calculate relative path for %s from %s: %v", absolutePath, baseDir, err)
	} else {
		relativePath = relPath
	}

	return relativePath, absolutePath
}

// syncCampaignFiles finds files matching patterns and adds them to the campaign
func syncCampaignFiles(campaign *Campaign, patterns []string) (*SyncResult, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns specified")
	}

	baseDir := getCampaignBaseDir(campaign)

	// Verify the directory exists
	if info, err := os.Stat(baseDir); err != nil {
		return nil, fmt.Errorf("campaign location does not exist: %s (%v)", baseDir, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("campaign location is not a directory: %s", baseDir)
	}

	logger.Printf("🔍 Syncing files for campaign '%s' from directory: %s", campaign.Name, baseDir)
	logger.Printf("🔍 Using patterns: %v", patterns)

	// Expand brace patterns
	var expandedPatterns []string
	for _, pattern := range patterns {
		expanded := expandBraces(pattern)
		expandedPatterns = append(expandedPatterns, expanded...)
	}
	logger.Printf("🔍 Expanded patterns: %v", expandedPatterns)

	// Find files matching patterns
	var foundFiles []string

	for _, pattern := range expandedPatterns {
		logger.Printf("🔍 Processing pattern: %s", pattern)

		// Use doublestar for globstar (**) pattern support from the campaign's location
		fsys := os.DirFS(baseDir)
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			logger.Printf("⚠️ Pattern glob failed for %s: %v", pattern, err)
			continue
		}

		logger.Printf("🔍 Pattern '%s' found %d matches: %v", pattern, len(matches), matches)

		for _, match := range matches {
			// Convert match to absolute path (it's relative to baseDir)
			fullPath := filepath.Join(baseDir, match)

			// Skip directories
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				foundFiles = append(foundFiles, fullPath)
				logger.Printf("✅ Added file: %s", fullPath)
			} else if err == nil && info.IsDir() {
				logger.Printf("⏭️ Skipped directory: %s", fullPath)
			} else {
				logger.Printf("⚠️ Could not stat file %s: %v", fullPath, err)
			}
		}
	}

	logger.Printf("🔍 Total files found before deduplication: %d", len(foundFiles))

	// Deduplicate
	fileSet := make(map[string]bool)
	var uniqueFiles []string
	for _, file := range foundFiles {
		abs, _ := filepath.Abs(file)
		if !fileSet[abs] {
			fileSet[abs] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}

	logger.Printf("🔍 Unique files after deduplication: %d", len(uniqueFiles))

	// Apply exclusion patterns
	var filteredFiles []string
	for _, file := range uniqueFiles {
		excluded := false
		absPath, _ := filepath.Abs(file)

		for _, excludePattern := range campaign.ExcludePatterns {
			matched, err := filepath.Match(excludePattern, absPath)
			if err == nil && matched {
				excluded = true
				logger.Printf("⏭️ Excluded by pattern '%s': %s", excludePattern, file)
				break
			}
			// Also check if any parent directory matches the pattern
			pathParts := strings.Split(absPath, string(filepath.Separator))
			for _, part := range pathParts {
				if matched, _ := filepath.Match(strings.Trim(excludePattern, "**/"), part); matched {
					excluded = true
					logger.Printf("⏭️ Excluded by directory pattern '%s': %s", excludePattern, file)
					break
				}
			}
			if excluded {
				break
			}
		}

		if !excluded {
			filteredFiles = append(filteredFiles, file)
		}
	}

	logger.Printf("🔍 Files after exclusion filtering: %d", len(filteredFiles))

	// Check campaign size limit
	if campaign.MaxFiles > 0 && len(filteredFiles) > campaign.MaxFiles {
		return nil, fmt.Errorf("pattern matches %d files but campaign limit is %d. Refine patterns or increase max_files", len(filteredFiles), campaign.MaxFiles)
	}

	addedCount := 0

	// Add new files to tracked files
	for _, filePath := range filteredFiles {
		absolutePath, _ := filepath.Abs(filePath)

		// Check if already tracked
		found := false
		for _, tracked := range campaign.TrackedFiles {
			if tracked.AbsolutePath == absolutePath {
				found = true
				break
			}
		}

		if !found {
			// Get file info
			fileInfo, err := os.Stat(absolutePath)
			var size int64
			var modTime time.Time
			if err == nil {
				size = fileInfo.Size()
				modTime = fileInfo.ModTime()
			} else {
				modTime = time.Now()
				logger.Printf("⚠️ Could not get file info for %s: %v", absolutePath, err)
			}

			// Calculate relative path from campaign location
			relPath, err := filepath.Rel(baseDir, absolutePath)
			if err != nil {
				relPath = filePath
				logger.Printf("⚠️ Could not calculate relative path for %s: %v", absolutePath, err)
			}

			newFile := TrackedFile{
				ID:             uuid.New(),
				FilePath:       relPath,
				AbsolutePath:   absolutePath,
				VisitCount:     0,
				FirstSeen:      time.Now().UTC(),
				LastModified:   modTime.UTC(),
				SizeBytes:      size,
				Deleted:        false,
				PriorityWeight: defaultPriorityWeight,
				Excluded:       false,
				Metadata:       make(map[string]interface{}),
			}

			campaign.TrackedFiles = append(campaign.TrackedFiles, newFile)
			addedCount++
			logger.Printf("➕ Added tracked file: %s (rel: %s)", absolutePath, relPath)
		} else {
			logger.Printf("⏭️ File already tracked: %s", absolutePath)
		}
	}

	logger.Printf("📊 Sync results: %d files added, %d total tracked files", addedCount, len(campaign.TrackedFiles))

	// Create structure snapshot
	snapshot := StructureSnapshot{
		ID:           uuid.New(),
		Timestamp:    time.Now().UTC(),
		TotalFiles:   len(campaign.TrackedFiles),
		NewFiles:     []string{}, // Could be enhanced to track what was added
		DeletedFiles: []string{}, // Could be enhanced to track what was removed
		MovedFiles:   make(map[string]string),
		SnapshotData: make(map[string]interface{}),
	}

	campaign.StructureSnapshots = append(campaign.StructureSnapshots, snapshot)

	// Update staleness scores
	updateStalenessScores(campaign)

	return &SyncResult{
		Added:      addedCount,
		Moved:      0,
		Removed:    0,
		SnapshotID: snapshot.ID,
		Total:      len(campaign.TrackedFiles),
	}, nil
}

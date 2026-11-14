package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	if len(patterns) == 0 || len(patterns) > 32 {
		return nil, fmt.Errorf("require 1..32 patterns")
	}

	baseDir := getCampaignBaseDir(campaign)

	// Verify the directory exists
	if info, err := os.Stat(baseDir); err != nil {
		return nil, fmt.Errorf("campaign location does not exist: %s (%v)", baseDir, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("campaign location is not a directory: %s", baseDir)
	}

	// Expand brace patterns
	var expandedPatterns []string
	for _, pattern := range patterns {
		expanded := expandBraces(pattern)
		expandedPatterns = append(expandedPatterns, expanded...)
	}

	// Find files matching patterns
	var foundFiles []string

	for _, pattern := range expandedPatterns {

		// Use doublestar for globstar (**) pattern support from the campaign's location
		fsys := os.DirFS(baseDir)
		matches, err := doublestar.Glob(fsys, pattern, doublestar.WithFailOnIOErrors())
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			// Convert match to absolute path (it's relative to baseDir)
			fullPath := filepath.Join(baseDir, match)

			// Skip directories
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				foundFiles = append(foundFiles, fullPath)
			} else if err == nil && info.IsDir() {
			} else {
				return nil, fmt.Errorf("stat matched file %s: %w", fullPath, err)
			}
		}
	}

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

	// Apply exclusion patterns
	var filteredFiles []string
	for _, file := range uniqueFiles {
		excluded := false
		absPath, _ := filepath.Abs(file)

		for _, excludePattern := range campaign.ExcludePatterns {
			matched, err := filepath.Match(excludePattern, absPath)
			if err == nil && matched {
				excluded = true
				break
			}
			// Also check if any parent directory matches the pattern
			pathParts := strings.Split(absPath, string(filepath.Separator))
			for _, part := range pathParts {
				if matched, _ := filepath.Match(strings.Trim(excludePattern, "*/"), part); matched {
					excluded = true
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

	// Check campaign size limit
	if campaign.MaxFiles > 0 && len(filteredFiles) > campaign.MaxFiles {
		return nil, fmt.Errorf("pattern matches %d files but campaign limit is %d. Refine patterns or increase max_files", len(filteredFiles), campaign.MaxFiles)
	}

	if len(filteredFiles) > 10000 {
		return nil, fmt.Errorf("scan exceeds 10000 files")
	}
	// Build a complete scan before changing state. Partial scans must never mark
	// unobserved files deleted or accept a review against stale revision data.
	sort.Strings(filteredFiles)
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	scanned := make(map[string]TrackedFile, len(filteredFiles))
	var scannedBytes int64
	for _, path := range filteredFiles {
		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		realBase, err := filepath.EvalSymlinks(baseDir)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(realBase, real)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("file escapes campaign location: %s", path)
		}
		relativePath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil, err
		}
		revision, info, err := fileRevision(root, relativePath)
		if err != nil {
			return nil, err
		}
		scannedBytes += info.Size()
		if scannedBytes > 128<<20 {
			return nil, fmt.Errorf("scan exceeds 128 MiB content budget")
		}
		rel, _ = filepath.Rel(baseDir, path)
		scanned[path] = TrackedFile{
			ID: uuid.New(), FilePath: rel, AbsolutePath: path,
			FirstSeen: time.Now().UTC(), LastModified: info.ModTime().UTC(), ContentHash: &revision,
			SizeBytes: info.Size(), PriorityWeight: defaultPriorityWeight, Metadata: map[string]interface{}{},
		}
	}
	snapshot := StructureSnapshot{ID: uuid.New(), Timestamp: time.Now().UTC(), MovedFiles: map[string]string{}}
	existing := map[string]int{}
	oldByHash := map[string][]int{}
	newByHash := map[string][]string{}
	for i, file := range campaign.TrackedFiles {
		if file.Deleted {
			continue
		}
		existing[file.AbsolutePath] = i
		if _, found := scanned[file.AbsolutePath]; !found && file.ContentHash != nil {
			oldByHash[*file.ContentHash] = append(oldByHash[*file.ContentHash], i)
		}
	}
	for path, file := range scanned {
		if _, found := existing[path]; !found {
			newByHash[*file.ContentHash] = append(newByHash[*file.ContentHash], path)
		}
	}
	for _, path := range filteredFiles {
		fresh := scanned[path]
		i, found := existing[path]
		if !found {
			old := oldByHash[*fresh.ContentHash]
			if len(old) == 1 && len(newByHash[*fresh.ContentHash]) == 1 {
				i, found = old[0], true
				snapshot.MovedFiles[campaign.TrackedFiles[i].FilePath] = fresh.FilePath
			}
		}
		if found {
			file := &campaign.TrackedFiles[i]
			file.FilePath, file.AbsolutePath = fresh.FilePath, path
			file.ContentHash, file.LastModified, file.SizeBytes = fresh.ContentHash, fresh.LastModified, fresh.SizeBytes
		} else {
			campaign.TrackedFiles = append(campaign.TrackedFiles, fresh)
			snapshot.NewFiles = append(snapshot.NewFiles, fresh.FilePath)
		}
	}
	for i := range campaign.TrackedFiles {
		file := &campaign.TrackedFiles[i]
		if !file.Deleted {
			if _, found := scanned[file.AbsolutePath]; !found {
				file.Deleted = true
				snapshot.DeletedFiles = append(snapshot.DeletedFiles, file.FilePath)
			}
		}
	}
	snapshot.TotalFiles = len(scanned)
	campaign.StructureSnapshots = append(campaign.StructureSnapshots, snapshot)
	if len(campaign.StructureSnapshots) > 100 {
		campaign.StructureSnapshots = campaign.StructureSnapshots[len(campaign.StructureSnapshots)-100:]
	}
	updateStalenessScores(campaign)
	return &SyncResult{Added: len(snapshot.NewFiles), Moved: len(snapshot.MovedFiles), Removed: len(snapshot.DeletedFiles), SnapshotID: snapshot.ID, Total: len(scanned)}, nil
}

// A failed or changing read cannot establish a content revision.
func fileRevision(root *os.Root, path string) (string, os.FileInfo, error) {
	f, err := root.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return "", nil, err
	}
	if !before.Mode().IsRegular() {
		return "", nil, fmt.Errorf("not a regular file: %s", path)
	}
	const maxBytes = 64 << 20
	if before.Size() > maxBytes {
		return "", nil, fmt.Errorf("file exceeds revision read limit: %s", path)
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxBytes+1))
	if err != nil {
		return "", nil, err
	}
	if n > maxBytes {
		return "", nil, fmt.Errorf("file exceeds revision read limit: %s", path)
	}
	after, err := f.Stat()
	if err != nil {
		return "", nil, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return "", nil, fmt.Errorf("file changed during scan: %s", path)
	}
	return hex.EncodeToString(h.Sum(nil)), after, nil
}

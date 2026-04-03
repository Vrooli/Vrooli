package scenarios

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// archivePresets defines named file patterns for archive preservation.
var archivePresets = map[string][]string{
	"documentation": {"PRD.md", "README.md", "docs/**", "*.md"},
	"requirements":  {"PRD.md", "requirements/**", "specs/**", "REQUIREMENTS.md"},
	"planning":      {"PRD.md", ".vrooli/**", "planning/**", "design/**"},
	"all-planning":  {"PRD.md", "README.md", "docs/**", "requirements/**", "specs/**", "planning/**", "design/**", ".vrooli/**", "*.md"},
}

var archiveIgnoredDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"dist":         {},
	"build":        {},
	"coverage":     {},
	".next":        {},
	".turbo":       {},
	"target":       {},
	"vendor":       {},
}

func isIgnoredArchivePath(path string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if _, ignored := archiveIgnoredDirs[part]; ignored {
			return true
		}
	}
	return false
}

// copyPreservedFiles copies files matching the specified patterns from scenario to idea directory.
func copyPreservedFiles(scenarioPath, ideaDir string, preserveFiles *apipb.PreserveFilesRequest) ([]string, error) {
	explicitPatterns := append([]string{}, preserveFiles.Paths...)
	presetPatterns := []string{}
	if preserveFiles.Preset != nil && *preserveFiles.Preset != "" {
		presetMatches, ok := archivePresets[*preserveFiles.Preset]
		if ok {
			presetPatterns = append(presetPatterns, presetMatches...)
		}
	}

	patterns := append([]string{}, explicitPatterns...)
	patterns = append(patterns, presetPatterns...)
	if len(patterns) == 0 {
		return nil, nil
	}

	// Deduplicate patterns
	seen := make(map[string]bool)
	uniquePatterns := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		normalized, err := normalizeArchiveRelativePath(pattern)
		if err != nil {
			slog.Warn("skipping invalid preserve path", "pattern", pattern, "error", err)
			continue
		}
		if !seen[normalized] {
			seen[normalized] = true
			uniquePatterns = append(uniquePatterns, normalized)
		}
	}

	// Collect files matching patterns. Preset matches exclude generated/vendor dirs.
	matchedFiles := make(map[string]bool)
	for _, pattern := range uniquePatterns {
		matches, err := resolveGlobPattern(scenarioPath, pattern)
		if err != nil {
			slog.Warn("failed to resolve pattern", "pattern", pattern, "error", err)
			continue
		}
		isPresetPattern := false
		for _, presetPattern := range presetPatterns {
			if presetPattern == pattern {
				isPresetPattern = true
				break
			}
		}
		for _, match := range matches {
			if isPresetPattern && isIgnoredArchivePath(match) {
				continue
			}
			matchedFiles[match] = true
		}
	}

	// Copy matched files
	var preserved []string
	for relPath := range matchedFiles {
		srcPath := filepath.Join(scenarioPath, relPath)
		dstPath := filepath.Join(ideaDir, relPath)

		if err := copyFile(srcPath, dstPath); err != nil {
			slog.Warn("failed to copy preserved file", "path", relPath, "error", err)
			continue
		}
		preserved = append(preserved, relPath)
	}

	sort.Strings(preserved)
	return preserved, nil
}

// resolveGlobPattern expands a glob pattern relative to a base directory.
func resolveGlobPattern(baseDir, pattern string) ([]string, error) {
	normalizedPattern, err := normalizeArchiveRelativePath(pattern)
	if err != nil {
		return nil, err
	}

	// Handle exact file matches first
	exactPath := filepath.Join(baseDir, normalizedPattern)
	if info, err := os.Stat(exactPath); err == nil && !info.IsDir() {
		return []string{normalizedPattern}, nil
	}

	// Use doublestar for ** glob support
	fullPattern := filepath.Join(baseDir, normalizedPattern)
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return nil, err
	}

	// Convert to relative paths and filter directories
	var result []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		relPath, err := filepath.Rel(baseDir, match)
		if err != nil {
			continue
		}
		normalizedRelPath, err := normalizeArchiveRelativePath(relPath)
		if err != nil {
			continue
		}
		result = append(result, normalizedRelPath)
	}

	return result, nil
}

func normalizeArchiveRelativePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	normalized := filepath.Clean(filepath.FromSlash(trimmed))
	if normalized == "." {
		return "", errors.New("path must reference a file")
	}
	if filepath.IsAbs(normalized) {
		return "", errors.New("path must be relative")
	}
	if normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}
	return normalized, nil
}

// copyFile copies a file from src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("cannot copy directory: %s", src)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Chmod(srcInfo.Mode())
}

func (h *Handler) deriveBacklogIdeasRoot(scenarioPath string) (string, error) {
	trimmedScenarioPath := strings.TrimSpace(scenarioPath)
	if trimmedScenarioPath != "" {
		cleanScenarioPath := filepath.Clean(trimmedScenarioPath)
		if strings.EqualFold(filepath.Base(cleanScenarioPath), "swarm-manager") {
			return "", errProtectedScenarioDelete
		}
	}

	baseDir := strings.TrimSpace(h.scenariosDir)
	if baseDir == "" {
		baseDir = "scenarios"
	}
	if !filepath.IsAbs(baseDir) {
		if absBaseDir, err := filepath.Abs(baseDir); err == nil {
			baseDir = absBaseDir
		}
	}
	return filepath.Join(baseDir, "swarm-manager", "ideas"), nil
}

func preservePresetOrCustom(preserveFiles *apipb.PreserveFilesRequest) string {
	if preserveFiles == nil {
		return "none"
	}
	if len(preserveFiles.Paths) > 0 {
		return "custom"
	}
	if preserveFiles.Preset != nil && strings.TrimSpace(*preserveFiles.Preset) != "" {
		return "preset:" + strings.ToLower(strings.TrimSpace(*preserveFiles.Preset))
	}
	return "none"
}

func archiveActor() string {
	actor := strings.TrimSpace(os.Getenv("USER"))
	if actor == "" {
		return "swarm-manager-api"
	}
	return actor
}

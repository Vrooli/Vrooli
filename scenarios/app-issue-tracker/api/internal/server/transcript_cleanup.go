package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"app-issue-tracker-api/internal/logging"
)

// TranscriptCleanupConfig controls transcript file retention
type TranscriptCleanupConfig struct {
	// MaxAge is the maximum age of transcripts to keep (older ones are deleted)
	MaxAge time.Duration
	// MaxCount is the maximum number of transcript pairs to keep (oldest deleted first)
	MaxCount int
	// Enabled controls whether cleanup runs
	Enabled bool
}

// DefaultTranscriptCleanupConfig returns sensible defaults for transcript cleanup
// Retention policy: keep last 50 investigations OR 3 days (whichever is more restrictive)
func DefaultTranscriptCleanupConfig() TranscriptCleanupConfig {
	return TranscriptCleanupConfig{
		MaxAge:   3 * 24 * time.Hour, // 3 days (reduced from 7)
		MaxCount: 50,                 // Keep last 50 investigations (reduced from 100)
		Enabled:  true,
	}
}

type transcriptFile struct {
	path    string
	modTime time.Time
}

// CleanupOldTranscripts removes old transcript files based on age and count limits
func CleanupOldTranscripts(scenarioRoot string, config TranscriptCleanupConfig) error {
	if !config.Enabled {
		return nil
	}

	transcriptDir := filepath.Join(scenarioRoot, "tmp", "codex")
	if _, err := os.Stat(transcriptDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	// Collect all transcript and related files
	var files []transcriptFile
	err := filepath.Walk(transcriptDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, transcriptFile{
			path:    path,
			modTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	now := time.Now()
	deleted := 0
	preserved := 0

	for i, file := range files {
		age := now.Sub(file.modTime)
		shouldDelete := false

		// Delete if too old
		if age > config.MaxAge {
			shouldDelete = true
		}

		// Delete if beyond count limit (keeping newest MaxCount files)
		if i >= config.MaxCount {
			shouldDelete = true
		}

		if shouldDelete {
			if err := os.Remove(file.path); err != nil {
				logging.LogWarn("Failed to delete old transcript file", "path", file.path, "error", err)
			} else {
				deleted++
			}
		} else {
			preserved++
		}
	}

	if deleted > 0 {
		logging.LogInfo(
			"Transcript cleanup completed",
			"deleted", deleted,
			"preserved", preserved,
			"max_age_days", int(config.MaxAge.Hours()/24),
			"max_count", config.MaxCount,
		)
	}

	return nil
}

// CleanupCodexMarkerFiles removes .codex-run-*.marker and codex-prompt-*.txt files
// from the scenario root directory. These are temporary files created by Claude Code CLI
// during investigation execution that should be cleaned up periodically.
func CleanupCodexMarkerFiles(scenarioRoot string) error {
	// Pattern matching for marker and prompt files
	markerPattern := filepath.Join(scenarioRoot, ".codex-run-*.marker")
	promptPattern := filepath.Join(scenarioRoot, "codex-prompt-*.txt")

	deleted := 0

	// Clean marker files
	markerFiles, err := filepath.Glob(markerPattern)
	if err != nil {
		return fmt.Errorf("failed to glob marker files: %w", err)
	}
	for _, file := range markerFiles {
		if err := os.Remove(file); err != nil {
			logging.LogWarn("Failed to delete marker file", "path", file, "error", err)
		} else {
			deleted++
		}
	}

	// Clean prompt files
	promptFiles, err := filepath.Glob(promptPattern)
	if err != nil {
		return fmt.Errorf("failed to glob prompt files: %w", err)
	}
	for _, file := range promptFiles {
		if err := os.Remove(file); err != nil {
			logging.LogWarn("Failed to delete prompt file", "path", file, "error", err)
		} else {
			deleted++
		}
	}

	if deleted > 0 {
		logging.LogDebug("Cleaned up Codex temporary files", "deleted", deleted)
	}

	return nil
}

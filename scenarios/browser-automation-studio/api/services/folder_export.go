package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/storage"
)

// allowedOutputDirs defines safe base directories for folder exports.
var allowedOutputDirs = []string{
	"coverage",
	"data",
	"tmp",
	"temp",
	"log",
	"logs",
	"artifact",
	"artifacts",
}

// ExportToFolder exports execution timeline and all related artifacts to a structured folder.
func (s *WorkflowService) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	// Validate and prepare output directory
	if err := validateAndPrepareOutputDir(outputDir); err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}

	// Fetch execution data
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	workflow, err := s.repo.GetWorkflow(ctx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	timeline, err := s.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to get timeline: %w", err)
	}

	// Export timeline JSON
	timelineJSON, err := json.MarshalIndent(timeline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal timeline: %w", err)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "timeline.json"), timelineJSON, 0644); err != nil {
		return fmt.Errorf("failed to write timeline.json: %w", err)
	}

	// Generate markdown files
	workflowName := workflow.Name
	if workflowName == "" {
		workflowName = "Unnamed Workflow"
	}

	readmeContent := GenerateTimelineMarkdown(timeline, workflowName)
	if err := os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	summaryContent := GenerateExecutionSummaryMarkdown(timeline)
	if err := os.WriteFile(filepath.Join(outputDir, "execution-summary.md"), []byte(summaryContent), 0644); err != nil {
		return fmt.Errorf("failed to write execution-summary.md: %w", err)
	}

	consoleContent := GenerateConsoleLogsMarkdown(timeline)
	if err := os.WriteFile(filepath.Join(outputDir, "console-logs.md"), []byte(consoleContent), 0644); err != nil {
		return fmt.Errorf("failed to write console-logs.md: %w", err)
	}

	networkContent := GenerateNetworkActivityMarkdown(timeline)
	if err := os.WriteFile(filepath.Join(outputDir, "network-activity.md"), []byte(networkContent), 0644); err != nil {
		return fmt.Errorf("failed to write network-activity.md: %w", err)
	}

	assertionsContent := GenerateAssertionsMarkdown(timeline)
	if err := os.WriteFile(filepath.Join(outputDir, "assertions.md"), []byte(assertionsContent), 0644); err != nil {
		return fmt.Errorf("failed to write assertions.md: %w", err)
	}

	// Export screenshots
	screenshotCount := 0
	for _, frame := range timeline.Frames {
		if frame.Screenshot != nil {
			screenshotCount++
		}
	}

	if screenshotCount > 0 && storageClient != nil {
		screenshotsDir := filepath.Join(outputDir, "screenshots")
		if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
			return fmt.Errorf("failed to create screenshots directory: %w", err)
		}

		// Export each screenshot by downloading from storage
		for i, frame := range timeline.Frames {
			if frame.Screenshot == nil || frame.Screenshot.URL == "" {
				continue
			}

			// Extract object name from URL
			// URL format: /screenshots/{execution-id}/{filename}
			objectName := frame.Screenshot.URL
			if strings.HasPrefix(objectName, "/") {
				objectName = objectName[1:]
			}

			// Download screenshot from MinIO
			reader, info, err := storageClient.GetScreenshot(ctx, objectName)
			if err != nil {
				// Log but don't fail - screenshot might be missing
				continue
			}

			// Generate filename: step-{index}-{node-id}.{ext}
			ext := "png"
			if frame.Screenshot.ContentType == "image/jpeg" {
				ext = "jpg"
			} else if info != nil && strings.Contains(info.ContentType, "jpeg") {
				ext = "jpg"
			}

			// Sanitize node ID for filename (use function from replay_renderer.go)
			nodeID := sanitizeFilename(frame.NodeID)
			filename := fmt.Sprintf("step-%02d-%s.%s", i+1, nodeID, ext)
			screenshotPath := filepath.Join(screenshotsDir, filename)

			// Write screenshot to file
			outFile, err := os.Create(screenshotPath)
			if err != nil {
				reader.Close()
				return fmt.Errorf("failed to create screenshot file %s: %w", filename, err)
			}

			_, err = io.Copy(outFile, reader)
			reader.Close()
			outFile.Close()

			if err != nil {
				return fmt.Errorf("failed to write screenshot %s: %w", filename, err)
			}
		}
	}

	return nil
}

// validateAndPrepareOutputDir validates the output directory path and prepares it for writing.
func validateAndPrepareOutputDir(outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Security check: prevent directory traversal attacks
	cleanPath := filepath.Clean(absPath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("directory traversal not allowed")
	}

	// Check if path is under one of the allowed base directories
	allowed := false
	for _, allowedDir := range allowedOutputDirs {
		// Check if path contains the allowed directory
		if strings.Contains(cleanPath, string(filepath.Separator)+allowedDir+string(filepath.Separator)) ||
			strings.HasSuffix(cleanPath, string(filepath.Separator)+allowedDir) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("output directory must be within one of: %s", strings.Join(allowedOutputDirs, ", "))
	}

	// Check for dangerous paths
	dangerousPaths := []string{
		"/",
		"/bin",
		"/boot",
		"/dev",
		"/etc",
		"/lib",
		"/proc",
		"/root",
		"/sbin",
		"/sys",
		"/usr",
		"/var",
		filepath.Join(os.Getenv("HOME"), ".ssh"),
	}

	for _, dangerous := range dangerousPaths {
		if cleanPath == dangerous || strings.HasPrefix(cleanPath, dangerous+string(filepath.Separator)) {
			return fmt.Errorf("cannot write to system directory: %s", dangerous)
		}
	}

	// Create directory if it doesn't exist (including parents)
	if err := os.MkdirAll(cleanPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clear existing files in the directory (but not subdirectories for safety)
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip subdirectories for safety
			continue
		}

		entryPath := filepath.Join(cleanPath, entry.Name())
		if err := os.Remove(entryPath); err != nil {
			return fmt.Errorf("failed to remove existing file %s: %w", entry.Name(), err)
		}
	}

	return nil
}

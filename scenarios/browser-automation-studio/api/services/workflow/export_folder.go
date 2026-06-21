package workflow

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/storage"
)

// ExportToFolder exports execution timeline and all related artifacts to a
// structured folder. It fetches the timeline, builds a pure ExportPlan via
// BuildExportPlan, and materializes it via WriteExportPlan — keeping
// content generation (pure) cleanly separated from I/O.
func (s *WorkflowService) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
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

	plan, err := BuildExportPlan(timeline, workflow.Name)
	if err != nil {
		return err
	}

	if err := WriteExportPlan(ctx, plan, outputDir, storageClient); err != nil {
		return err
	}

	// Performance traces are written by the driver into the execution's
	// artifact root (not the timeline). Copy them into the export folder so
	// the capture handler's PerformanceProducer can surface them, and
	// best-effort upload to object storage alongside other artifacts.
	if err := s.exportPerformanceArtifacts(ctx, executionID, outputDir, storageClient); err != nil {
		return fmt.Errorf("export performance artifacts: %w", err)
	}

	return nil
}

// performanceArtifactFiles are the canonical filenames the driver emits into
// the execution's artifacts/performance directory.
var performanceArtifactFiles = []string{"performance.json", "performance.web-vitals.json"}

// exportPerformanceArtifacts copies any driver-written performance trace +
// web-vitals files from the execution artifact root into outputDir/performance
// and uploads them to object storage. A missing source directory is not an
// error (the run did not request a perf trace).
func (s *WorkflowService) exportPerformanceArtifacts(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	if strings.TrimSpace(s.executionDataRoot) == "" {
		return nil
	}
	srcDir := filepath.Join(s.executionDataRoot, executionID.String(), "artifacts", "performance")
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat performance dir: %w", err)
	}

	destDir := filepath.Join(outputDir, "performance")
	for _, name := range performanceArtifactFiles {
		srcPath := filepath.Join(srcDir, name)
		info, err := os.Stat(srcPath)
		if err != nil {
			// Trace is mandatory; web-vitals is best-effort (absent when the
			// page emitted no observable metrics). Skip whatever is missing.
			continue
		}
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("create performance dir: %w", err)
		}
		if err := copyFile(srcPath, filepath.Join(destDir, name)); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
		if storageClient != nil {
			label := strings.TrimSuffix(name, filepath.Ext(name))
			if _, err := storageClient.StoreArtifactFromFile(ctx, executionID, "performance/"+label, srcPath, "application/json"); err != nil && s.log != nil {
				s.log.WithError(err).WithField("execution_id", executionID).Warn("failed to upload performance artifact to storage")
			}
		}
		_ = info
	}
	return nil
}

// copyFile copies src to dst, truncating dst if it exists.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// WriteExportPlan is the I/O half of ExportToFolder: it validates and
// prepares the output directory, writes every PlannedFile, and best-effort
// downloads each PlannedScreenshot from storage. A missing screenshot
// object is skipped (not fatal); a write error fails the export.
func WriteExportPlan(ctx context.Context, plan *ExportPlan, outputDir string, storageClient storage.StorageInterface) error {
	if err := validateAndPrepareOutputDir(outputDir); err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}

	for _, f := range plan.Files {
		full := filepath.Join(outputDir, f.RelPath)
		if dir := filepath.Dir(full); dir != outputDir {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", f.RelPath, err)
			}
		}
		if err := os.WriteFile(full, f.Content, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", f.RelPath, err)
		}
	}

	if len(plan.Screenshots) == 0 || storageClient == nil {
		return nil
	}

	screenshotsDir := filepath.Join(outputDir, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create screenshots directory: %w", err)
	}

	for _, shot := range plan.Screenshots {
		// Download from storage (skip if fails - screenshot might be missing).
		reader, info, err := storageClient.GetScreenshot(ctx, shot.ObjectName)
		if err != nil {
			continue
		}

		ext := shot.FallbackExt
		if info != nil && strings.Contains(info.ContentType, "jpeg") {
			ext = "jpg"
		}
		destPath := filepath.Join(outputDir, shot.RelPathNoExt+"."+ext)

		outFile, err := os.Create(destPath)
		if err != nil {
			reader.Close()
			return fmt.Errorf("failed to create screenshot file %s: %w", destPath, err)
		}

		_, err = io.Copy(outFile, reader)
		reader.Close()
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write screenshot %s: %w", destPath, err)
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

	// Check for dangerous system paths
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
	if err := os.MkdirAll(cleanPath, 0o755); err != nil {
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

// ResultSummary provides a quick pass/fail summary for programmatic access.
type ResultSummary struct {
	Status         string `json:"status"`
	Success        bool   `json:"success"`
	ExecutionID    string `json:"execution_id"`
	DurationMs     int64  `json:"duration_ms"`
	StepsTotal     int    `json:"steps_total"`
	StepsCompleted int    `json:"steps_completed"`
	StepsFailed    int    `json:"steps_failed"`
	Error          string `json:"error,omitempty"`
}

// buildResultSummary creates a simple summary from the execution timeline.
func buildResultSummary(timeline *export.ExecutionTimeline) *ResultSummary {
	result := &ResultSummary{
		Status:      timeline.Status,
		ExecutionID: timeline.ExecutionID.String(),
		StepsTotal:  len(timeline.Frames),
	}

	// Determine success from status
	normalizedStatus := strings.ToLower(strings.TrimPrefix(timeline.Status, "EXECUTION_STATUS_"))
	result.Success = normalizedStatus == "completed"

	// Calculate duration
	if timeline.CompletedAt != nil && !timeline.StartedAt.IsZero() {
		result.DurationMs = timeline.CompletedAt.Sub(timeline.StartedAt).Milliseconds()
	}

	// Count completed and failed steps, capture first error
	for _, frame := range timeline.Frames {
		normalizedFrameStatus := strings.ToLower(strings.TrimPrefix(frame.Status, "STEP_STATUS_"))
		if normalizedFrameStatus == "completed" {
			result.StepsCompleted++
		} else if normalizedFrameStatus == "failed" {
			result.StepsFailed++
			if result.Error == "" && frame.Error != "" {
				result.Error = frame.Error
			}
		}
	}

	return result
}

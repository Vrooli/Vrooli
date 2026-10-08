package records

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Handler owns record behavior shared by generated Connect handlers and the
// pipeline; records no longer expose a parallel REST administration surface.
type Handler struct {
	records        Store
	builds         BuildStoreAdapter
	smokeTests     SmokeTestStoreAdapter
	logger         *slog.Logger
	scenarioRoot   string
	outputPathFunc func(scenarioName string) string
}

// BuildStoreAdapter adapts the build store interface for record operations.
type BuildStoreAdapter interface {
	Get(id string) (*BuildStatusView, bool)
	Update(id string, fn func(status *BuildStatusView)) error
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithScenarioRoot sets the scenario root path.
func WithScenarioRoot(root string) HandlerOption {
	return func(h *Handler) {
		h.scenarioRoot = root
	}
}

// WithOutputPathFunc sets the function for computing desktop output paths.
func WithOutputPathFunc(fn func(scenarioName string) string) HandlerOption {
	return func(h *Handler) {
		h.outputPathFunc = fn
	}
}

// WithSmokeTestStore sets the smoke test store for enriching records with video data.
func WithSmokeTestStore(store SmokeTestStoreAdapter) HandlerOption {
	return func(h *Handler) {
		h.smokeTests = store
	}
}

// SetSmokeTestStore sets the smoke test store after construction.
// This is used when the smoke test store is created after the handler.
func (h *Handler) SetSmokeTestStore(store SmokeTestStoreAdapter) {
	h.smokeTests = store
}

// NewHandler creates a new records handler.
func NewHandler(records Store, builds BuildStoreAdapter, logger *slog.Logger, opts ...HandlerOption) *Handler {
	h := &Handler{
		records: records,
		builds:  builds,
		logger:  logger,
	}
	for _, opt := range opts {
		opt(h)
	}
	// Default output path function
	if h.outputPathFunc == nil && h.scenarioRoot != "" {
		h.outputPathFunc = func(scenarioName string) string {
			return filepath.Join(h.scenarioRoot, scenarioName, "platforms", "electron")
		}
	}
	return h
}

// resolveMovePaths validates and resolves absolute source and destination paths for a move.
func (h *Handler) resolveMovePaths(rec *DesktopAppRecord, req *MoveRequest) (string, string, error) {
	if req.Target == "" {
		req.Target = "destination"
	}

	dest := rec.DestinationPath
	if req.Target == "custom" {
		if req.DestinationPath == "" {
			return "", "", fmt.Errorf("destination_path required for custom target")
		}
		dest = req.DestinationPath
	}
	if dest == "" {
		return "", "", fmt.Errorf("no destination path recorded")
	}

	absSrc, err := filepath.Abs(rec.OutputPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve source: %v", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", "", fmt.Errorf("resolve destination: %v", err)
	}
	if absSrc == absDest {
		return "", "", fmt.Errorf("source and destination are the same")
	}
	if _, err := os.Stat(absSrc); err != nil {
		return "", "", fmt.Errorf("source missing: %v", err)
	}
	return absSrc, absDest, nil
}

// performMove creates the destination directory and renames the source.
func performMove(absSrc, absDest string) error {
	if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
		return fmt.Errorf("prepare destination: %v", err)
	}
	if err := os.Rename(absSrc, absDest); err != nil {
		return fmt.Errorf("move failed: %v", err)
	}
	return nil
}

// updateRecordAfterMove persists path changes to the record and build stores.
func (h *Handler) updateRecordAfterMove(rec *DesktopAppRecord, req *MoveRequest, absDest string) {
	rec.OutputPath = absDest
	if req.Target != "custom" {
		rec.LocationMode = "proper"
	}
	if err := h.records.Upsert(rec); err != nil {
		h.logger.Warn("failed to update record after move", "error", err)
	}

	if rec.BuildID != "" && h.builds != nil {
		_ = h.builds.Update(rec.BuildID, func(status *BuildStatusView) {
			status.OutputPath = absDest
			if status.Metadata == nil {
				status.Metadata = map[string]interface{}{}
			}
			status.Metadata["moved_to"] = absDest
		})
	}
}

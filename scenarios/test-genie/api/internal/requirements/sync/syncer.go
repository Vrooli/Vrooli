// Package sync synchronizes requirement files with test results.
package sync

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// Reader abstracts file reading operations.
type Reader interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Exists(path string) bool
}

// Writer abstracts file writing operations.
type Writer interface {
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// osReader implements Reader using the os package.
type osReader struct{}

func (r *osReader) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (r *osReader) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (r *osReader) Exists(path string) bool                    { _, err := os.Stat(path); return err == nil }

// osWriter implements Writer using the os package.
type osWriter struct{}

func (w *osWriter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (w *osWriter) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Syncer synchronizes requirement files with test results.
type Syncer interface {
	// Sync updates requirement files based on test evidence.
	Sync(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle, opts Options) (*Result, error)

	// Preview shows what changes would be made without writing.
	Preview(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle, opts Options) (*Result, error)
}

// Options configures sync behavior.
type Options struct {
	// PruneOrphans removes validations for deleted test files.
	PruneOrphans bool

	// DiscoverNew adds validations for new test files.
	DiscoverNew bool

	// UpdateStatuses updates validation statuses from evidence.
	UpdateStatuses bool

	// AllowPartial allows sync without full test suite completion.
	AllowPartial bool

	// TestCommands contains commands that were executed.
	TestCommands []string

	// DryRun previews changes without writing files.
	DryRun bool

	// ScenarioRoot is the path to the scenario directory.
	ScenarioRoot string
}

// DefaultOptions returns default sync options.
func DefaultOptions() Options {
	return Options{
		PruneOrphans:   false, // Conservative default
		DiscoverNew:    true,
		UpdateStatuses: true,
		AllowPartial:   true,
		DryRun:         false,
	}
}

// Result contains sync operation results.
type Result struct {
	// Statistics
	FilesUpdated       int
	ValidationsAdded   int
	ValidationsRemoved int
	StatusesChanged    int

	// Details
	Changes []Change
	Errors  []error

	// Metadata
	SyncedAt     time.Time
	TestCommands []string
}

// Change represents a single change made during sync.
type Change struct {
	Type          ChangeType
	FilePath      string
	RequirementID string
	Field         string
	OldValue      string
	NewValue      string
}

// ChangeType categorizes sync changes.
type ChangeType string

const (
	ChangeTypeStatusUpdate      ChangeType = "status_update"
	ChangeTypeValidationAdded   ChangeType = "validation_added"
	ChangeTypeValidationRemoved ChangeType = "validation_removed"
	ChangeTypeMetadataUpdate    ChangeType = "metadata_update"
)

// syncer implements Syncer.
type syncer struct {
	reader         Reader
	writer         Writer
	statusUpdater  *StatusUpdater
	orphanDetector *OrphanDetector
	fileWriter     *FileWriter
}

// New creates a Syncer with the provided dependencies.
func New(reader Reader, writer Writer) Syncer {
	return &syncer{
		reader:         reader,
		writer:         writer,
		statusUpdater:  NewStatusUpdater(),
		orphanDetector: NewOrphanDetector(reader),
		fileWriter:     NewFileWriter(writer),
	}
}

// NewDefault creates a Syncer using the real file system.
func NewDefault() Syncer {
	reader := &osReader{}
	writer := &osWriter{}
	return New(reader, writer)
}

// Sync updates requirement files based on test evidence.
func (s *syncer) Sync(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle, opts Options) (*Result, error) {
	result := &Result{
		SyncedAt:     time.Now(),
		TestCommands: opts.TestCommands,
	}

	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	// Phase 1: Update validation statuses
	if opts.UpdateStatuses {
		changes, err := s.statusUpdater.UpdateStatuses(ctx, index, evidence)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
		result.Changes = append(result.Changes, changes...)
		result.StatusesChanged = len(changes)
	}

	// Phase 2: Detect and optionally remove orphaned validations
	if opts.PruneOrphans && opts.ScenarioRoot != "" {
		orphans, err := s.orphanDetector.DetectOrphans(ctx, index, opts.ScenarioRoot)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}

		for _, orphan := range orphans {
			change := Change{
				Type:          ChangeTypeValidationRemoved,
				FilePath:      orphan.FilePath,
				RequirementID: orphan.RequirementID,
				Field:         "validation",
				OldValue:      orphan.ValidationRef,
			}
			result.Changes = append(result.Changes, change)
			result.ValidationsRemoved++
		}
	}

	// Phase 3: Write updated files
	if !opts.DryRun {
		filesWritten, err := s.writeUpdatedFiles(ctx, index, result.Changes)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
		result.FilesUpdated = filesWritten

		// Write sync metadata
		if opts.ScenarioRoot != "" {
			s.writeSyncMetadata(ctx, opts.ScenarioRoot, result)
		}
	}

	return result, nil
}

// Preview shows what changes would be made without writing.
func (s *syncer) Preview(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle, opts Options) (*Result, error) {
	opts.DryRun = true
	return s.Sync(ctx, index, evidence, opts)
}

// writeUpdatedFiles writes all modified modules to disk.
func (s *syncer) writeUpdatedFiles(ctx context.Context, index *parsing.ModuleIndex, changes []Change) (int, error) {
	// Group changes by file
	fileChanges := make(map[string][]Change)
	for _, change := range changes {
		fileChanges[change.FilePath] = append(fileChanges[change.FilePath], change)
	}

	written := 0
	for filePath := range fileChanges {
		module := index.GetModule(filePath)
		if module == nil {
			continue
		}

		err := s.fileWriter.WriteModule(ctx, module)
		if err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}

// writeSyncMetadata writes sync operation metadata to coverage directory.
func (s *syncer) writeSyncMetadata(ctx context.Context, scenarioRoot string, result *Result) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create sync directory
	syncDir := filepath.Join(scenarioRoot, sharedartifacts.SyncDir)
	if err := s.writer.MkdirAll(syncDir, 0755); err != nil {
		return err
	}

	// Write latest.json
	metadata := SyncMetadata{
		SyncedAt:           result.SyncedAt,
		TestCommands:       result.TestCommands,
		FilesUpdated:       result.FilesUpdated,
		ValidationsAdded:   result.ValidationsAdded,
		ValidationsRemoved: result.ValidationsRemoved,
		StatusesChanged:    result.StatusesChanged,
		ErrorCount:         len(result.Errors),
	}

	return s.fileWriter.WriteJSON(sharedartifacts.SyncMetadataPath(scenarioRoot), metadata)
}

// SyncMetadata contains information about a sync operation.
type SyncMetadata struct {
	SyncedAt           time.Time `json:"synced_at"`
	TestCommands       []string  `json:"test_commands,omitempty"`
	FilesUpdated       int       `json:"files_updated"`
	ValidationsAdded   int       `json:"validations_added"`
	ValidationsRemoved int       `json:"validations_removed"`
	StatusesChanged    int       `json:"statuses_changed"`
	ErrorCount         int       `json:"error_count"`
}

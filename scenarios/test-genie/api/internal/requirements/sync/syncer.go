// Package sync synchronizes requirement files with test results.
package sync

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/artifactpaths"

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

	// ArtifactRoot is test-genie's governed storage root for this scenario.
	// It is deliberately distinct from ScenarioRoot, which contains source.
	ArtifactRoot string
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
	artifactRoot   func(string) (string, error)
}

// New creates a Syncer with the provided dependencies.
func New(reader Reader, writer Writer) Syncer {
	return &syncer{
		reader:         reader,
		writer:         writer,
		statusUpdater:  NewStatusUpdater(),
		orphanDetector: NewOrphanDetector(reader),
		fileWriter:     NewFileWriter(writer),
		artifactRoot:   func(root string) (string, error) { return root, nil },
	}
}

// NewDefault creates a Syncer using the real file system.
func NewDefault() Syncer {
	reader := &osReader{}
	writer := &osWriter{}
	result := New(reader, writer).(*syncer)
	result.artifactRoot = artifactpaths.ScenarioRootForDir
	return result
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
			artifactRoot := opts.ArtifactRoot
			if artifactRoot == "" {
				artifactRoot, err = s.artifactRoot(opts.ScenarioRoot)
				if err != nil {
					result.Errors = append(result.Errors, err)
				}
			}
			if artifactRoot != "" {
				if err := s.writeSyncMetadata(ctx, artifactRoot, result); err != nil {
					result.Errors = append(result.Errors, err)
				}
			}
			if err := s.updatePRDOperationalTargets(ctx, index, opts.ScenarioRoot, opts.ArtifactRoot, result.SyncedAt); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}

	return result, nil
}

func (s *syncer) updatePRDOperationalTargets(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot, artifactRoot string, ts time.Time) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if s == nil || s.reader == nil || s.writer == nil {
		return nil
	}
	if index == nil || scenarioRoot == "" {
		return nil
	}

	prdPath := filepath.Join(scenarioRoot, "PRD.md")
	if !s.reader.Exists(prdPath) {
		return nil
	}

	desired := desiredOperationalTargetCheckboxes(index)
	if len(desired) == 0 {
		return nil
	}

	raw, err := s.reader.ReadFile(prdPath)
	if err != nil {
		return err
	}
	original := string(raw)
	updated, changed := updateOperationalTargetChecklistLines(original, desired)
	if !changed {
		return nil
	}

	if artifactRoot == "" {
		var err error
		artifactRoot, err = s.artifactRoot(scenarioRoot)
		if err != nil {
			return err
		}
	}
	backupDir := artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot, "requirements-sync", "prd-backups")
	if err := s.writer.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, "PRD.md."+ts.Format("20060102-150405"))
	if err := s.writer.WriteFile(backupPath, raw, 0o644); err != nil {
		return err
	}
	return s.writer.WriteFile(prdPath, []byte(updated), 0o644)
}

// otIDPattern matches operational-target identifiers (e.g. OT-P0-001) inside a
// requirement's PRDRef field. It is the single regex source for OT extraction.
var otIDPattern = regexp.MustCompile(`OT-[Pp][0-2]-\d{3}`)

// OTPriorityCount holds the complete/total requirement tallies for one priority
// band (P0/P1/P2) of operational targets.
type OTPriorityCount struct {
	Complete int
	Total    int
}

// OTSummary aggregates operational-target completion across a module index.
// An operational target is "complete" only when every requirement that links to
// it (via PRDRef) is itself complete — matching the PRD checkbox semantics.
type OTSummary struct {
	// Complete is the number of operational targets whose requirements are all complete.
	Complete int
	// Total is the number of distinct operational targets referenced by requirements.
	Total int
	// ByPriority breaks the tallies down by priority band ("P0"/"P1"/"P2").
	ByPriority map[string]OTPriorityCount
}

// otRequirementCounts walks the index once and returns, per operational-target
// ID, how many of its linked requirements exist and how many are complete.
func otRequirementCounts(index *parsing.ModuleIndex) map[string]*OTPriorityCount {
	if index == nil {
		return nil
	}
	byOT := make(map[string]*OTPriorityCount)
	for _, module := range index.Modules {
		if module == nil {
			continue
		}
		for i := range module.Requirements {
			req := &module.Requirements[i]
			if req == nil {
				continue
			}
			m := otIDPattern.FindString(req.PRDRef)
			if m == "" {
				continue
			}
			ot := strings.ToUpper(m)
			c := byOT[ot]
			if c == nil {
				c = &OTPriorityCount{}
				byOT[ot] = c
			}
			c.Total++
			if req.Status == types.StatusComplete {
				c.Complete++
			}
		}
	}
	return byOT
}

// OperationalTargetSummary aggregates OT completion for the whole index. It is
// the read-side counterpart to desiredOperationalTargetCheckboxes and is reused
// by the execute report so partial runs can show cached OT counts.
func OperationalTargetSummary(index *parsing.ModuleIndex) OTSummary {
	byOT := otRequirementCounts(index)
	summary := OTSummary{ByPriority: map[string]OTPriorityCount{}}
	for ot, c := range byOT {
		if c == nil || c.Total == 0 {
			continue
		}
		summary.Total++
		complete := c.Complete == c.Total
		if complete {
			summary.Complete++
		}
		// Priority band is the middle segment of the ID: OT-P0-001 -> P0.
		band := "P2"
		if parts := strings.Split(ot, "-"); len(parts) >= 2 {
			band = strings.ToUpper(parts[1])
		}
		pc := summary.ByPriority[band]
		pc.Total++
		if complete {
			pc.Complete++
		}
		summary.ByPriority[band] = pc
	}
	return summary
}

func desiredOperationalTargetCheckboxes(index *parsing.ModuleIndex) map[string]bool {
	byOT := otRequirementCounts(index)
	if byOT == nil {
		return nil
	}
	desired := make(map[string]bool, len(byOT))
	for ot, c := range byOT {
		if c == nil || c.Total == 0 {
			continue
		}
		desired[ot] = c.Complete == c.Total
	}
	return desired
}

func updateOperationalTargetChecklistLines(content string, desired map[string]bool) (string, bool) {
	if strings.TrimSpace(content) == "" || len(desired) == 0 {
		return content, false
	}

	lineRe := regexp.MustCompile(`^(\s*-\s*\[)([ xX])(\]\s*)(OT-[Pp][0-2]-\d{3})(\b.*)$`)
	lines := strings.Split(content, "\n")
	changed := false

	for i, line := range lines {
		m := lineRe.FindStringSubmatch(line)
		if len(m) != 6 {
			continue
		}
		otKey := strings.ToUpper(m[4])
		wantChecked, ok := desired[otKey]
		if !ok {
			continue
		}
		currentChecked := m[2] == "x" || m[2] == "X"
		if currentChecked == wantChecked {
			continue
		}
		mark := " "
		if wantChecked {
			mark = "x"
		}
		lines[i] = m[1] + mark + m[3] + m[4] + m[5]
		changed = true
	}

	if !changed {
		return content, false
	}
	return strings.Join(lines, "\n"), true
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
	if err := s.writer.MkdirAll(syncDir, 0o755); err != nil {
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

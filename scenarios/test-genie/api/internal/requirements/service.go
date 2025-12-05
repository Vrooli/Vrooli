package requirements

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/requirements/discovery"
	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/evidence"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/reporting"
	"test-genie/internal/requirements/snapshot"
	syncpkg "test-genie/internal/requirements/sync"
	"test-genie/internal/requirements/types"
	"test-genie/internal/requirements/validation"
)

// Service orchestrates requirement operations.
type Service struct {
	reader          Reader
	writer          Writer
	discoverer      discovery.Discoverer
	parser          parsing.Parser
	loader          evidence.Loader
	enricher        enrichment.Enricher
	syncer          syncpkg.Syncer
	reporter        reporting.Reporter
	validator       validation.Validator
	snapshotBuilder snapshot.Builder
}

// NewService creates a Service with production dependencies.
func NewService() *Service {
	reader := NewOSReader()
	writer := NewOSWriter()

	return &Service{
		reader:          reader,
		writer:          writer,
		discoverer:      discovery.NewDefault(),
		parser:          parsing.NewDefault(),
		loader:          evidence.NewDefault(),
		enricher:        enrichment.New(),
		syncer:          syncpkg.NewDefault(),
		reporter:        reporting.New(),
		validator:       validation.NewDefault(),
		snapshotBuilder: snapshot.New(),
	}
}

// NewServiceWithDeps creates a Service with provided dependencies.
func NewServiceWithDeps(reader Reader, writer Writer) *Service {
	// Create sync-compatible reader/writer adapters
	syncReader := &syncReaderAdapter{reader: reader}
	syncWriter := &syncWriterAdapter{writer: writer}

	return &Service{
		reader:          reader,
		writer:          writer,
		discoverer:      discovery.New(reader),
		parser:          parsing.New(reader),
		loader:          evidence.New(reader),
		enricher:        enrichment.New(),
		syncer:          syncpkg.New(syncReader, syncWriter),
		reporter:        reporting.New(),
		validator:       validation.New(reader),
		snapshotBuilder: snapshot.New(),
	}
}

// syncReaderAdapter adapts Reader to syncpkg.Reader
type syncReaderAdapter struct {
	reader Reader
}

func (a *syncReaderAdapter) ReadFile(path string) ([]byte, error) { return a.reader.ReadFile(path) }
func (a *syncReaderAdapter) ReadDir(path string) ([]fs.DirEntry, error) {
	return a.reader.ReadDir(path)
}
func (a *syncReaderAdapter) Exists(path string) bool { return a.reader.Exists(path) }

// syncWriterAdapter adapts Writer to syncpkg.Writer
type syncWriterAdapter struct {
	writer Writer
}

func (a *syncWriterAdapter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return a.writer.WriteFile(path, data, perm)
}
func (a *syncWriterAdapter) MkdirAll(path string, perm fs.FileMode) error {
	return a.writer.MkdirAll(path, perm)
}

// SyncInput matches the existing orchestrator interface.
type SyncInput struct {
	ScenarioName     string
	ScenarioDir      string
	PhaseDefinitions []phases.Definition
	PhaseResults     []phases.ExecutionResult
	CommandHistory   []string
}

// SyncOutput contains sync operation results.
type SyncOutput struct {
	FilesUpdated       int
	ValidationsAdded   int
	ValidationsRemoved int
	StatusesChanged    int
	Errors             []error
}

// Sync performs full requirements synchronization.
func (s *Service) Sync(ctx context.Context, input SyncInput) error {
	// 1. Discover requirement files
	files, err := s.discoverer.Discover(ctx, input.ScenarioDir)
	if err != nil {
		if errors.Is(err, types.ErrNoRequirementsDir) || errors.Is(err, discovery.ErrNoRequirementsDir) {
			// No requirements directory - nothing to sync
			return nil
		}
		return fmt.Errorf("discovery: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	// 2. Parse all requirement files
	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	// 3. Load test evidence
	evidenceBundle, err := s.loader.LoadAll(ctx, input.ScenarioDir)
	if err != nil {
		return fmt.Errorf("loading evidence: %w", err)
	}

	// 4. Convert phase results to evidence
	if len(input.PhaseResults) > 0 {
		phaseEvidence := convertPhaseResults(input.PhaseResults)
		evidenceBundle.PhaseResults.Merge(phaseEvidence)
	}

	// 5. Enrich requirements with live status
	if err := s.enricher.Enrich(ctx, index, evidenceBundle); err != nil {
		return fmt.Errorf("enrichment: %w", err)
	}

	// 6. Sync files
	opts := syncpkg.Options{
		ScenarioRoot:   input.ScenarioDir,
		TestCommands:   input.CommandHistory,
		UpdateStatuses: true,
		DiscoverNew:    true,
	}
	result, err := s.syncer.Sync(ctx, index, evidenceBundle, opts)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	log.Printf("Sync complete: %d files updated, %d statuses changed",
		result.FilesUpdated, result.StatusesChanged)

	// 7. Write snapshot
	summary := s.enricher.ComputeSummary(index.Modules)
	snap, err := s.snapshotBuilder.Build(ctx, index, summary)
	if err == nil && snap != nil {
		snapshotPath := filepath.Join(input.ScenarioDir, "coverage", "requirements-sync", "latest.json")
		s.writer.MkdirAll(filepath.Dir(snapshotPath), 0755)
		snapshot.WriteSnapshot(s.writer, snapshotPath, snap)
	}

	return nil
}

// Report generates a requirements report.
func (s *Service) Report(ctx context.Context, scenarioDir string, opts reporting.Options, w io.Writer) error {
	// 1. Discover requirement files
	files, err := s.discoverer.Discover(ctx, scenarioDir)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}

	// 2. Parse all requirement files
	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	// 3. Load test evidence
	evidenceBundle, err := s.loader.LoadAll(ctx, scenarioDir)
	if err != nil {
		return fmt.Errorf("loading evidence: %w", err)
	}

	// 4. Enrich requirements
	if err := s.enricher.Enrich(ctx, index, evidenceBundle); err != nil {
		return fmt.Errorf("enrichment: %w", err)
	}

	// 5. Compute summary
	summary := s.enricher.ComputeSummary(index.Modules)

	// 6. Generate report
	return s.reporter.Generate(ctx, index, summary, opts, w)
}

// Validate checks requirement structure.
func (s *Service) Validate(ctx context.Context, scenarioDir string) (*types.ValidationResult, error) {
	// 1. Discover requirement files
	files, err := s.discoverer.Discover(ctx, scenarioDir)
	if err != nil {
		if errors.Is(err, types.ErrNoRequirementsDir) || errors.Is(err, discovery.ErrNoRequirementsDir) {
			return types.NewValidationResult(), nil
		}
		return nil, fmt.Errorf("discovery: %w", err)
	}

	// 2. Parse all requirement files
	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	// 3. Run validation rules
	result := s.validator.Validate(ctx, index, scenarioDir)

	return result, nil
}

// GetSummary returns a summary of the requirements.
func (s *Service) GetSummary(ctx context.Context, scenarioDir string) (enrichment.Summary, error) {
	// 1. Discover requirement files
	files, err := s.discoverer.Discover(ctx, scenarioDir)
	if err != nil {
		return enrichment.Summary{}, fmt.Errorf("discovery: %w", err)
	}

	// 2. Parse all requirement files
	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return enrichment.Summary{}, fmt.Errorf("parsing: %w", err)
	}

	// 3. Load test evidence
	evidenceBundle, err := s.loader.LoadAll(ctx, scenarioDir)
	if err != nil {
		return enrichment.Summary{}, fmt.Errorf("loading evidence: %w", err)
	}

	// 4. Enrich requirements
	if err := s.enricher.Enrich(ctx, index, evidenceBundle); err != nil {
		return enrichment.Summary{}, fmt.Errorf("enrichment: %w", err)
	}

	// 5. Compute and return summary
	return s.enricher.ComputeSummary(index.Modules), nil
}

// convertPhaseResults converts orchestrator phase results to evidence.
func convertPhaseResults(results []phases.ExecutionResult) types.EvidenceMap {
	evidenceMap := make(types.EvidenceMap)

	for _, result := range results {
		record := types.EvidenceRecord{
			Phase:           result.Name,
			Status:          types.NormalizeLiveStatus(result.Status),
			DurationSeconds: float64(result.DurationSeconds),
			SourcePath:      result.LogPath,
			Evidence:        result.Error,
		}

		// Store as phase-level result
		key := "__phase__" + result.Name
		evidenceMap[key] = append(evidenceMap[key], record)
	}

	return evidenceMap
}

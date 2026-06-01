package requirements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/requirements/discovery"
	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/evidence"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/reporting"
	"test-genie/internal/requirements/snapshot"
	"test-genie/internal/requirements/types"
	"test-genie/internal/requirements/validation"
	sharedartifacts "test-genie/internal/shared/artifacts"

	syncpkg "test-genie/internal/requirements/sync"
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

// StatusChange records a single requirement-level declared-status transition
// produced by a sync run. Kind classification (promotion/regression) is left to
// the caller so this layer stays free of presentation semantics.
type StatusChange struct {
	ID     string
	PRDRef string
	From   string
	To     string
}

// SyncReport is the structured outcome of a sync or a read-only snapshot. It
// carries the counts the execute report renders so callers never recompute
// status semantics. On a skipped/cached read, Synced is false and Changes is
// empty, but Summary/OT/LastSyncedAt still reflect the last persisted state.
type SyncReport struct {
	// Synced is true when this report reflects a fresh write this run.
	Synced bool
	// LastSyncedAt is the timestamp of the most recent persisted sync, if any.
	LastSyncedAt *time.Time
	// Summary is the enrichment rollup of requirement counts.
	Summary enrichment.Summary
	// OT aggregates operational-target completion.
	OT syncpkg.OTSummary
	// Changes lists requirement-level status transitions made this run.
	Changes []StatusChange
}

// Sync performs full requirements synchronization and returns a structured
// report of the resulting counts and status changes. A nil report (with nil
// error) means there is no requirements directory to sync.
func (s *Service) Sync(ctx context.Context, input SyncInput) (*SyncReport, error) {
	// 1. Discover requirement files
	files, err := s.discoverer.Discover(ctx, input.ScenarioDir)
	if err != nil {
		if errors.Is(err, types.ErrNoRequirementsDir) || errors.Is(err, discovery.ErrNoRequirementsDir) {
			// No requirements directory - nothing to sync
			return nil, nil
		}
		return nil, fmt.Errorf("discovery: %w", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	// 2. Parse all requirement files
	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	// 3. Load test evidence
	evidenceBundle, err := s.loader.LoadAll(ctx, input.ScenarioDir)
	if err != nil {
		return nil, fmt.Errorf("loading evidence: %w", err)
	}

	// 4. Convert phase results to evidence
	if len(input.PhaseResults) > 0 {
		phaseEvidence := convertPhaseResults(input.PhaseResults)
		evidenceBundle.PhaseResults.Merge(phaseEvidence)
	}

	// 5. Enrich requirements with live status
	if err := s.enricher.Enrich(ctx, index, evidenceBundle); err != nil {
		return nil, fmt.Errorf("enrichment: %w", err)
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
		return nil, fmt.Errorf("sync: %w", err)
	}

	log.Printf("Sync complete: %d files updated, %d statuses changed",
		result.FilesUpdated, result.StatusesChanged)

	// 7. Write snapshot
	summary := s.enricher.ComputeSummary(index.Modules)
	snap, err := s.snapshotBuilder.Build(ctx, index, summary)
	if err == nil && snap != nil {
		snapshotPath := filepath.Join(input.ScenarioDir, "coverage", "requirements-sync", "latest.json")
		if err := s.writer.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
			return nil, fmt.Errorf("create snapshot dir: %w", err)
		}
		if err := snapshot.WriteSnapshot(s.writer, snapshotPath, snap); err != nil {
			return nil, fmt.Errorf("write snapshot: %w", err)
		}
	}

	report := &SyncReport{
		Synced:       true,
		LastSyncedAt: s.readLastSyncedAt(input.ScenarioDir),
		Summary:      summary,
		OT:           syncpkg.OperationalTargetSummary(index),
		Changes:      collectStatusChanges(index, result.Changes),
	}
	return report, nil
}

// Snapshot reads the last persisted requirement state without writing anything.
// It powers the execute report on partial/skipped runs: counts reflect disk,
// Synced is false, and Changes is empty. A nil report means no requirements dir.
func (s *Service) Snapshot(ctx context.Context, input SyncInput) (*SyncReport, error) {
	files, err := s.discoverer.Discover(ctx, input.ScenarioDir)
	if err != nil {
		if errors.Is(err, types.ErrNoRequirementsDir) || errors.Is(err, discovery.ErrNoRequirementsDir) {
			return nil, nil
		}
		return nil, fmt.Errorf("discovery: %w", err)
	}
	if len(files) == 0 {
		return nil, nil
	}

	index, err := s.parser.ParseAll(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	// Enrich with whatever evidence is already on disk so live status reflects
	// the last full run, but never write — this is a read-only view.
	evidenceBundle, err := s.loader.LoadAll(ctx, input.ScenarioDir)
	if err != nil {
		return nil, fmt.Errorf("loading evidence: %w", err)
	}
	if err := s.enricher.Enrich(ctx, index, evidenceBundle); err != nil {
		return nil, fmt.Errorf("enrichment: %w", err)
	}

	report := &SyncReport{
		Synced:       false,
		LastSyncedAt: s.readLastSyncedAt(input.ScenarioDir),
		Summary:      s.enricher.ComputeSummary(index.Modules),
		OT:           syncpkg.OperationalTargetSummary(index),
	}
	return report, nil
}

// readLastSyncedAt loads the timestamp of the most recent persisted sync from
// the sync metadata file. Returns nil when no sync has ever run.
func (s *Service) readLastSyncedAt(scenarioDir string) *time.Time {
	if scenarioDir == "" {
		return nil
	}
	raw, err := s.reader.ReadFile(sharedartifacts.SyncMetadataPath(scenarioDir))
	if err != nil {
		return nil
	}
	var meta struct {
		SyncedAt time.Time `json:"synced_at"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil || meta.SyncedAt.IsZero() {
		return nil
	}
	t := meta.SyncedAt
	return &t
}

// collectStatusChanges maps the syncer's raw change list down to
// requirement-level declared-status transitions, attaching each requirement's
// PRDRef so callers can group changes by operational target.
func collectStatusChanges(index *parsing.ModuleIndex, changes []syncpkg.Change) []StatusChange {
	if len(changes) == 0 {
		return nil
	}
	prdRefByID := make(map[string]string)
	if index != nil {
		for _, module := range index.Modules {
			if module == nil {
				continue
			}
			for i := range module.Requirements {
				req := &module.Requirements[i]
				prdRefByID[req.ID] = req.PRDRef
			}
		}
	}
	var out []StatusChange
	for _, c := range changes {
		// Only requirement-level status transitions are user-facing; validation
		// status churn is an implementation detail.
		if c.Type != syncpkg.ChangeTypeStatusUpdate || c.Field != "status" {
			continue
		}
		out = append(out, StatusChange{
			ID:     c.RequirementID,
			PRDRef: prdRefByID[c.RequirementID],
			From:   c.OldValue,
			To:     c.NewValue,
		})
	}
	return out
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

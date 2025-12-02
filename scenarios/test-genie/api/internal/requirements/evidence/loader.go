// Package evidence loads test evidence from various sources.
package evidence

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"test-genie/internal/requirements/types"
)

// Reader abstracts file reading for evidence loading.
type Reader interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Exists(path string) bool
}

// osReader implements Reader using the os package.
type osReader struct{}

func (r *osReader) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (r *osReader) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (r *osReader) Exists(path string) bool                    { _, err := os.Stat(path); return err == nil }

// Loader loads test evidence from various sources.
type Loader interface {
	// LoadAll loads evidence from all sources in a scenario.
	LoadAll(ctx context.Context, scenarioRoot string) (*types.EvidenceBundle, error)

	// LoadPhaseResults loads evidence from phase result files.
	LoadPhaseResults(ctx context.Context, scenarioRoot string) (types.EvidenceMap, error)

	// LoadVitestEvidence loads evidence from vitest coverage files.
	LoadVitestEvidence(ctx context.Context, scenarioRoot string) (map[string][]types.VitestResult, error)

	// LoadManualValidations loads evidence from manual validation logs.
	LoadManualValidations(ctx context.Context, scenarioRoot string) (*types.ManualManifest, error)
}

// loader implements Loader using file system operations.
type loader struct {
	reader Reader
}

// New creates a Loader with the provided Reader.
func New(reader Reader) Loader {
	return &loader{reader: reader}
}

// NewDefault creates a Loader using the real file system.
func NewDefault() Loader {
	return &loader{reader: &osReader{}}
}

// LoadAll loads evidence from all sources.
func (l *loader) LoadAll(ctx context.Context, scenarioRoot string) (*types.EvidenceBundle, error) {
	bundle := types.NewEvidenceBundle()

	// Load phase results
	phaseResults, err := l.LoadPhaseResults(ctx, scenarioRoot)
	if err == nil && len(phaseResults) > 0 {
		bundle.PhaseResults = phaseResults
	}

	// Load vitest evidence
	vitestEvidence, err := l.LoadVitestEvidence(ctx, scenarioRoot)
	if err == nil && len(vitestEvidence) > 0 {
		bundle.VitestEvidence = vitestEvidence
	}

	// Load manual validations
	manualValidations, err := l.LoadManualValidations(ctx, scenarioRoot)
	if err == nil && manualValidations != nil {
		bundle.ManualValidations = manualValidations
	}

	bundle.LoadedAt = time.Now()
	return bundle, nil
}

// LoadPhaseResults loads evidence from phase result files.
func (l *loader) LoadPhaseResults(ctx context.Context, scenarioRoot string) (types.EvidenceMap, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	evidenceMap := make(types.EvidenceMap)

	// Check multiple possible locations for phase results
	possibleDirs := []string{
		filepath.Join(scenarioRoot, "coverage", "phase-results"),
		filepath.Join(scenarioRoot, "test", "coverage", "phase-results"),
		filepath.Join(scenarioRoot, "test", "artifacts", "phase-results"),
	}

	for _, dir := range possibleDirs {
		if !l.reader.Exists(dir) {
			continue
		}

		results, err := loadPhaseResultsFromDir(ctx, l.reader, dir)
		if err != nil {
			continue
		}

		evidenceMap.Merge(results)
	}

	return evidenceMap, nil
}

// LoadVitestEvidence loads evidence from vitest coverage files.
func (l *loader) LoadVitestEvidence(ctx context.Context, scenarioRoot string) (map[string][]types.VitestResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	evidenceMap := make(map[string][]types.VitestResult)

	// Check multiple possible locations
	possiblePaths := []string{
		filepath.Join(scenarioRoot, "ui", "coverage", "vitest-requirements.json"),
		filepath.Join(scenarioRoot, "coverage", "vitest-requirements.json"),
		filepath.Join(scenarioRoot, "test", "coverage", "vitest.json"),
	}

	for _, path := range possiblePaths {
		if !l.reader.Exists(path) {
			continue
		}

		results, err := loadVitestFromFile(ctx, l.reader, path)
		if err != nil {
			continue
		}

		// Merge results
		for reqID, res := range results {
			evidenceMap[reqID] = append(evidenceMap[reqID], res...)
		}
	}

	return evidenceMap, nil
}

// LoadManualValidations loads evidence from manual validation logs.
func (l *loader) LoadManualValidations(ctx context.Context, scenarioRoot string) (*types.ManualManifest, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Check multiple possible locations
	possiblePaths := []string{
		filepath.Join(scenarioRoot, "coverage", "manual-validations", "log.jsonl"),
		filepath.Join(scenarioRoot, "test", "coverage", "manual-validations", "log.jsonl"),
		filepath.Join(scenarioRoot, "coverage", "manual", "log.jsonl"),
	}

	for _, path := range possiblePaths {
		if !l.reader.Exists(path) {
			continue
		}

		manifest, err := loadManualFromFile(ctx, l.reader, path)
		if err != nil {
			continue
		}

		return manifest, nil
	}

	return nil, nil
}

// LoadFromPhaseExecution creates evidence from phase execution results.
// This is used to convert test-genie's internal execution results to evidence.
func LoadFromPhaseExecution(phaseResults []types.PhaseResult) types.EvidenceMap {
	evidenceMap := make(types.EvidenceMap)

	for _, result := range phaseResults {
		for _, reqID := range result.RequirementIDs {
			record := types.EvidenceRecord{
				RequirementID:   reqID,
				Status:          result.ToLiveStatus(),
				Phase:           result.Phase,
				Evidence:        result.LogPath,
				UpdatedAt:       result.ExecutedAt,
				DurationSeconds: result.DurationSeconds,
				SourcePath:      result.LogPath,
			}
			evidenceMap.Add(record)
		}
	}

	return evidenceMap
}

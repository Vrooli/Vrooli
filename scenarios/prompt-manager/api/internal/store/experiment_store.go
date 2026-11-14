package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileExperimentStore implements ExperimentStore using the file system.
// Experiments are runtime-class data (mutated by execution, not authored)
// and live under the RuntimeData root:
//
//	<RuntimeData>/experiments/{experiment-id}/experiment.json + outcomes.jsonl
type FileExperimentStore struct {
	runtimeDataDir string
}

// NewFileExperimentStore creates a new file-based experiment store rooted at
// the given RuntimeData class directory.
func NewFileExperimentStore(runtimeDataDir string) *FileExperimentStore {
	return &FileExperimentStore{runtimeDataDir: runtimeDataDir}
}

// experimentsDir returns the path to the experiments directory.
func (s *FileExperimentStore) experimentsDir() string {
	return filepath.Join(s.runtimeDataDir, "experiments")
}

// experimentDir returns the directory for a specific experiment.
func (s *FileExperimentStore) experimentDir(experimentID string) string {
	return filepath.Join(s.experimentsDir(), experimentID)
}

// List returns all experiments.
func (s *FileExperimentStore) List(ctx context.Context) ([]Experiment, error) {
	dirs, err := ListDirectories(s.experimentsDir())
	if err != nil {
		return nil, nil // No experiments directory = no experiments
	}

	var experiments []Experiment
	for _, eid := range dirs {
		exp, err := s.loadExperiment(eid)
		if err != nil {
			continue // Skip malformed experiments
		}
		experiments = append(experiments, *exp)
	}

	return experiments, nil
}

// ListBySkill returns experiments for a specific skill.
func (s *FileExperimentStore) ListBySkill(ctx context.Context, skillID string) ([]Experiment, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []Experiment
	for _, exp := range all {
		if exp.SkillID == skillID {
			filtered = append(filtered, exp)
		}
	}
	return filtered, nil
}

// Get retrieves an experiment by ID.
func (s *FileExperimentStore) Get(ctx context.Context, experimentID string) (*Experiment, error) {
	exp, err := s.loadExperiment(experimentID)
	if err != nil {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}
	return exp, nil
}

// Create creates a new experiment.
func (s *FileExperimentStore) Create(ctx context.Context, experiment *Experiment) error {
	expDir := s.experimentDir(experiment.ID)
	if FileExists(filepath.Join(expDir, "experiment.json")) {
		return fmt.Errorf("experiment already exists: %s", experiment.ID)
	}

	experiment.Kind = KindExperiment
	experiment.SchemaVersion = CurrentSchemaVersion
	if experiment.Status == "" {
		experiment.Status = ExperimentStatusDraft
	}
	experiment.Timestamps = NewTimestamps()

	if err := os.MkdirAll(expDir, 0o755); err != nil {
		return fmt.Errorf("creating experiment directory: %w", err)
	}

	if err := SaveJSON(filepath.Join(expDir, "experiment.json"), experiment); err != nil {
		return fmt.Errorf("writing experiment.json: %w", err)
	}

	return nil
}

// Update updates an existing experiment.
func (s *FileExperimentStore) Update(ctx context.Context, experimentID string, experiment *Experiment) error {
	existing, err := s.loadExperiment(experimentID)
	if err != nil {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}

	// Carry forward immutable fields
	experiment.Kind = existing.Kind
	experiment.SchemaVersion = existing.SchemaVersion
	experiment.ID = existing.ID
	experiment.Timestamps = existing.Timestamps
	experiment.UpdateTimestamp()

	expDir := s.experimentDir(experimentID)
	if err := SaveJSON(filepath.Join(expDir, "experiment.json"), experiment); err != nil {
		return fmt.Errorf("writing experiment.json: %w", err)
	}

	return nil
}

// Delete removes an experiment and its outcomes.
func (s *FileExperimentStore) Delete(ctx context.Context, experimentID string) error {
	expDir := s.experimentDir(experimentID)
	if !FileExists(filepath.Join(expDir, "experiment.json")) {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	return DeleteDirectory(expDir)
}

// RecordOutcome appends an opaque outcome to the experiment's JSONL log.
func (s *FileExperimentStore) RecordOutcome(ctx context.Context, experimentID string, outcome ExperimentOutcome) error {
	if _, err := s.loadExperiment(experimentID); err != nil {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}

	path := filepath.Join(s.experimentDir(experimentID), "outcomes.jsonl")
	return AppendJSONL(path, outcome)
}

// RecordServe appends a serve event to the experiment's serve JSONL log.
func (s *FileExperimentStore) RecordServe(ctx context.Context, serve ExperimentServe) error {
	if _, err := s.loadExperiment(serve.ExperimentID); err != nil {
		return fmt.Errorf("experiment not found: %s", serve.ExperimentID)
	}
	if serve.ServedAt == "" {
		serve.ServedAt = time.Now().UTC().Format(time.RFC3339)
	}
	path := filepath.Join(s.experimentDir(serve.ExperimentID), "serve.jsonl")
	return AppendJSONL(path, serve)
}

// ListServes returns all raw serve events for an experiment.
func (s *FileExperimentStore) ListServes(ctx context.Context, experimentID string) ([]ExperimentServe, error) {
	if _, err := s.loadExperiment(experimentID); err != nil {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}

	path := filepath.Join(s.experimentDir(experimentID), "serve.jsonl")
	return readServesJSONL(path)
}

// CountServesByVariant returns serve counts grouped by variant ID.
// Only parses the variantId field from each line — the rest of the record is not touched.
func (s *FileExperimentStore) CountServesByVariant(ctx context.Context, experimentID string) (map[string]int, error) {
	if _, err := s.loadExperiment(experimentID); err != nil {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}

	path := filepath.Join(s.experimentDir(experimentID), "serve.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, fmt.Errorf("opening serves: %w", err)
	}
	defer f.Close()

	counts := make(map[string]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Only parse the variantId field
		var envelope struct {
			VariantID string `json:"variantId"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue // Skip malformed lines
		}
		counts[envelope.VariantID]++
	}

	return counts, scanner.Err()
}

// ListOutcomes returns all raw outcomes for an experiment.
func (s *FileExperimentStore) ListOutcomes(ctx context.Context, experimentID string) ([]ExperimentOutcome, error) {
	if _, err := s.loadExperiment(experimentID); err != nil {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}

	path := filepath.Join(s.experimentDir(experimentID), "outcomes.jsonl")
	return readOutcomesJSONL(path)
}

// CountOutcomesByVariant returns outcome counts grouped by variant ID.
// Only parses the variantId field from each line — the data blob is not touched.
func (s *FileExperimentStore) CountOutcomesByVariant(ctx context.Context, experimentID string) (map[string]int, error) {
	if _, err := s.loadExperiment(experimentID); err != nil {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}

	path := filepath.Join(s.experimentDir(experimentID), "outcomes.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, fmt.Errorf("opening outcomes: %w", err)
	}
	defer f.Close()

	counts := make(map[string]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Only parse the variantId field
		var envelope struct {
			VariantID string `json:"variantId"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue // Skip malformed lines
		}
		counts[envelope.VariantID]++
	}

	return counts, scanner.Err()
}

// loadExperiment loads an experiment from its JSON file.
func (s *FileExperimentStore) loadExperiment(experimentID string) (*Experiment, error) {
	path := filepath.Join(s.experimentDir(experimentID), "experiment.json")
	return LoadJSON[Experiment](path)
}

// readServesJSONL reads all serve events from a JSONL file.
func readServesJSONL(path string) ([]ExperimentServe, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening serves: %w", err)
	}
	defer f.Close()

	var serves []ExperimentServe
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var serve ExperimentServe
		if err := json.Unmarshal(line, &serve); err != nil {
			continue // Skip malformed lines
		}
		serves = append(serves, serve)
	}

	if err := scanner.Err(); err != nil {
		return serves, fmt.Errorf("scanning serves: %w", err)
	}

	return serves, nil
}

// readOutcomesJSONL reads all outcomes from a JSONL file.
func readOutcomesJSONL(path string) ([]ExperimentOutcome, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening outcomes: %w", err)
	}
	defer f.Close()

	var outcomes []ExperimentOutcome
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var outcome ExperimentOutcome
		if err := json.Unmarshal(line, &outcome); err != nil {
			continue // Skip malformed lines
		}
		outcomes = append(outcomes, outcome)
	}

	if err := scanner.Err(); err != nil {
		return outcomes, fmt.Errorf("scanning outcomes: %w", err)
	}

	return outcomes, nil
}

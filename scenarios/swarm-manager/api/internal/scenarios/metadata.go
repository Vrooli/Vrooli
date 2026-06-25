package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// metadataPath returns the path to the metadata file for a scenario.
func metadataPath(scenarioPath string) string {
	return filepath.Join(scenarioPath, ".vrooli", "metadata.json")
}

// loadMetadata reads the metadata for a scenario.
// [REQ:REQ-P0-007] Load editable metadata from .vrooli/metadata.json
func (h *Handler) loadMetadata(scenarioPath string) (ScenarioMetadata, bool, error) {
	metaPath := metadataPath(scenarioPath)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if no metadata file exists
			return ScenarioMetadata{
				IsGreenfield: false,
			}, false, nil
		}
		return ScenarioMetadata{}, false, err
	}

	var metadata ScenarioMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return ScenarioMetadata{}, true, err
	}
	return metadata, true, nil
}

// saveMetadata writes the metadata for a scenario.
// [REQ:REQ-P0-007] Persist editable metadata to .vrooli/metadata.json
func (h *Handler) saveMetadata(scenarioPath string, metadata ScenarioMetadata) error {
	metaPath := metadataPath(scenarioPath)

	// Ensure .vrooli directory exists
	dir := filepath.Dir(metaPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0o600)
}

// loadScenarioFromSource maps CLI metadata into a Scenario enriched with local data.
// [REQ:REQ-P0-007] Includes metadata for greenfield settings
func (h *Handler) loadScenarioFromSource(source ScenarioSource) (Scenario, error) {
	name := strings.TrimSpace(source.Name)
	if name == "" {
		return Scenario{}, errors.New("scenario name missing")
	}
	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		return Scenario{}, errors.New("scenario path missing")
	}

	if _, err := os.Stat(scenarioPath); err != nil {
		return Scenario{}, err
	}

	displayName := name
	description := strings.TrimSpace(source.Description)

	status := normalizeScenarioStatus(source.Status)

	// Read priority from lighthouse.json if available
	priority := loadPriorityFromLighthouse(scenarioPath)

	tags := source.Tags
	if tags == nil {
		tags = []string{}
	}

	// Determine default greenfield status (check for PRD.md)
	defaultGreenfield := true
	prdPath := filepath.Join(scenarioPath, "PRD.md")
	if _, err := os.Stat(prdPath); err == nil {
		defaultGreenfield = false
	}

	// Load metadata for editable fields
	metadata, metaExists, err := h.loadMetadata(scenarioPath)
	if err != nil {
		slog.Warn("failed to load metadata", "scenario", name, "error", err)
		metadata = ScenarioMetadata{
			IsGreenfield: false,
		}
		metaExists = false
	}

	// Use metadata values; metadata file stores explicit user choices
	isGreenfield := metadata.IsGreenfield
	if !metaExists {
		isGreenfield = defaultGreenfield
	}

	return Scenario{
		Name:         name,
		DisplayName:  displayName,
		Description:  description,
		Status:       status,
		Priority:     priority,
		IsGreenfield: isGreenfield,
		Tags:         tags,
	}, nil
}

// loadScenarioByPath builds a Scenario from a name and filesystem path.
// Used by the Archiver when the scenario is already located.
func (h *Handler) loadScenarioByPath(name, scenarioPath string) (Scenario, error) {
	return h.loadScenarioFromSource(ScenarioSource{
		Name: name,
		Path: scenarioPath,
	})
}

// loadAllScenarios reads all scenarios from the CLI source.
func (h *Handler) loadAllScenarios(ctx context.Context) ([]Scenario, error) {
	sources, err := h.source.List(ctx)
	if err != nil {
		return nil, err
	}

	var scores map[string]int
	if len(sources) > 0 {
		scores = h.getCompletenessScores(ctx)
	}
	scenarios := make([]Scenario, 0, len(sources))
	for _, source := range sources {
		scenario, err := h.loadScenarioFromSource(source)
		if err != nil {
			slog.Warn("skipping scenario due to load error", "scenario", source.Name, "error", err)
			continue
		}
		applyCompletenessScore(&scenario, scores)
		scenarios = append(scenarios, scenario)
	}
	return scenarios, nil
}

// loadScenario reads a single scenario by name.
func (h *Handler) loadScenario(ctx context.Context, name string) (Scenario, error) {
	source, found, err := h.findScenarioSource(ctx, name)
	if err != nil {
		return Scenario{}, err
	}
	if !found {
		return Scenario{}, os.ErrNotExist
	}
	scenario, err := h.loadScenarioFromSource(source)
	if err != nil {
		return Scenario{}, err
	}
	applyCompletenessScore(&scenario, h.getCompletenessScores(ctx))
	return scenario, nil
}

func (h *Handler) findScenarioSource(ctx context.Context, name string) (ScenarioSource, bool, error) {
	sources, err := h.source.List(ctx)
	if err != nil {
		return ScenarioSource{}, false, err
	}
	trimmed := strings.TrimSpace(name)
	for _, source := range sources {
		if source.Name == trimmed {
			return source, true, nil
		}
	}
	return ScenarioSource{}, false, nil
}

func (h *Handler) getCompletenessScores(ctx context.Context) map[string]int {
	if h.completeness == nil {
		return nil
	}
	scores, err := h.completeness.Scores(ctx)
	if err != nil {
		slog.Warn("failed to load completeness scores", "error", err)
		return nil
	}
	return scores
}

func normalizeScenarioStatus(status string) ScenarioStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return StatusRunning
	case "stopped":
		return StatusStopped
	case "error":
		return StatusError
	default:
		return StatusUnknown
	}
}

func loadPriorityFromLighthouse(scenarioPath string) int {
	priority := 5 // Default priority
	lighthousePath := filepath.Join(scenarioPath, ".vrooli", "lighthouse.json")
	if lighthouseData, err := os.ReadFile(lighthousePath); err == nil {
		var lighthouse struct {
			Priority int `json:"priority"`
		}
		if err := json.Unmarshal(lighthouseData, &lighthouse); err == nil && lighthouse.Priority > 0 {
			priority = lighthouse.Priority
		}
	}
	return priority
}

func applyCompletenessScore(scenario *Scenario, scores map[string]int) {
	if scenario == nil || scores == nil {
		return
	}
	score, ok := scores[scenario.Name]
	if !ok {
		return
	}
	scenario.CompletenessScore = &score
}

package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/backlog"
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
	h.catalogMu.Lock()
	defer h.catalogMu.Unlock()
	if len(h.catalog) > 0 && time.Since(h.catalogCachedAt) < catalogCacheTTL {
		return append([]Scenario(nil), h.catalog...), nil
	}

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
	h.attachCatalogHealth(ctx, scenarios)
	h.catalog = append([]Scenario(nil), scenarios...)
	h.catalogCachedAt = time.Now()
	return append([]Scenario(nil), scenarios...), nil
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
	h.attachHealth(ctx, &scenario)
	return scenario, nil
}

func (h *Handler) attachHealth(ctx context.Context, scenario *Scenario) {
	h.attachHealthWithFixes(ctx, scenario, nil)
}

func (h *Handler) attachHealthWithFixes(ctx context.Context, scenario *Scenario, fixes []backlog.BacklogItem) {
	if h.health == nil || scenario == nil {
		return
	}
	snapshot := h.health.Snapshot(ctx, scenario.Name)
	if fixes == nil && h.backlogLister != nil {
		loaded, err := h.backlogLister.LoadAll([]backlog.BacklogKind{backlog.KindFix})
		if err != nil {
			slog.Warn("failed to load remediation reconciliation state", "error", err)
		} else {
			fixes = loaded
		}
	}
	h.attachRemediationItems(&snapshot, scenario.Name, fixes)
	scenario.Health = &snapshot
}

// attachCatalogHealth keeps the catalog path bounded. Health is independent per
// scenario, so serial provider calls needlessly made a 113-row catalog take seconds.
func (h *Handler) attachCatalogHealth(ctx context.Context, scenarios []Scenario) {
	if h.health == nil || len(scenarios) == 0 {
		return
	}
	workers := catalogHealthWorkers
	if workers > len(scenarios) {
		workers = len(scenarios)
	}
	var fixes []backlog.BacklogItem
	if h.backlogLister != nil {
		loaded, err := h.backlogLister.LoadAll([]backlog.BacklogKind{backlog.KindFix})
		if err != nil {
			slog.Warn("failed to load remediation reconciliation state", "error", err)
		} else {
			fixes = loaded
		}
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				h.attachHealthWithFixes(ctx, &scenarios[index], fixes)
			}
		}()
	}
	for index := range scenarios {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
}

func (h *Handler) attachRemediationItems(snapshot *ScenarioHealthSnapshot, scenario string, items []backlog.BacklogItem) {
	if snapshot == nil {
		return
	}
	known := map[string]struct{}{}
	for _, phase := range snapshot.Phases {
		if phase.PriorityCapabilityID == "" {
			continue
		}
		fingerprint, fingerprintErr := (RemediationTarget{Scenario: scenario, ProviderPhase: phase.Phase, CapabilityID: phase.PriorityCapabilityID}).Fingerprint()
		if fingerprintErr == nil {
			known[fingerprint] = struct{}{}
		}
	}
	for _, item := range items {
		if _, ok := known[item.FindingRef]; !ok {
			continue
		}
		snapshot.Remediation = append(snapshot.Remediation, ScenarioRemediationState{Fingerprint: item.FindingRef, State: string(item.Status), WorkRef: string(item.Kind) + "/" + item.Name, UpdatedAt: item.Updated})
	}
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// VrooliScenarioLister shells out to the Vrooli CLI to discover every scenario on disk.
type VrooliScenarioLister struct{}

// NewVrooliScenarioLister constructs the default CLI-backed lister.
func NewVrooliScenarioLister() *VrooliScenarioLister {
	return &VrooliScenarioLister{}
}

// ListScenarios runs `vrooli scenario list --json` and returns every scenario with metadata.
func (l *VrooliScenarioLister) ListScenarios(ctx context.Context) ([]ScenarioMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario list failed: %w", err)
	}

	var payload scenarioListResponse
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("parse scenario list failed: %w", err)
	}

	scenarios := make([]ScenarioMetadata, 0, len(payload.Scenarios))
	for _, item := range payload.Scenarios {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		scenarios = append(scenarios, ScenarioMetadata{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
			Status:      strings.TrimSpace(item.Status),
			Tags:        append([]string(nil), item.Tags...),
		})
	}
	return scenarios, nil
}

type scenarioListResponse struct {
	Success   bool                 `json:"success"`
	Scenarios []scenarioListRecord `json:"scenarios"`
}

type scenarioListRecord struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
}

var _ ScenarioLister = (*VrooliScenarioLister)(nil)

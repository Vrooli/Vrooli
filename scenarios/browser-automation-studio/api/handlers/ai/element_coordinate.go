package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// getElementAtCoordinate uses browserless to get element candidates at specific coordinates.
// It returns a selection result containing the element at the coordinate and nearby ancestor candidates.
func (h *ElementAnalysisHandler) getElementAtCoordinate(ctx context.Context, url string, x, y int) (*ElementSelectionResult, error) {
	if h.runner == nil {
		return nil, fmt.Errorf("automation runner not configured")
	}

	instructions := []autocontracts.CompiledInstruction{
		{
			Index:  0,
			NodeID: "probe.navigate",
			Type:   "navigate",
			Params: map[string]any{
				"url":       url,
				"waitUntil": "networkidle2",
				"timeoutMs": 45000,
			},
		},
		{
			Index:  1,
			NodeID: "probe.element",
			Type:   "probeElements",
			Params: map[string]any{
				"probeX":       x,
				"probeY":       y,
				"probeRadius":  8,
				"probeSamples": 36,
			},
		},
	}

	outcomes, _, err := h.runner.run(ctx, 0, 0, instructions)
	if err != nil {
		return nil, fmt.Errorf("automation probe failed: %w", err)
	}
	if len(outcomes) < 2 {
		return nil, fmt.Errorf("probe did not return any outcomes")
	}

	probe := outcomes[1]
	if !probe.Success {
		if probe.Failure != nil && strings.TrimSpace(probe.Failure.Message) != "" {
			return nil, errors.New(strings.TrimSpace(probe.Failure.Message))
		}
		return nil, fmt.Errorf("element probe unsuccessful")
	}

	if probe.ProbeResult == nil {
		return nil, fmt.Errorf("element probe did not return data")
	}

	data, err := json.Marshal(probe.ProbeResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal probe result: %w", err)
	}

	var selection ElementSelectionResult
	if err := json.Unmarshal(data, &selection); err != nil {
		return nil, fmt.Errorf("failed to decode probe result: %w", err)
	}

	if len(selection.Candidates) == 0 {
		return nil, fmt.Errorf("no qualifying elements found at coordinates")
	}

	if selection.SelectedIndex < 0 || selection.SelectedIndex >= len(selection.Candidates) {
		selection.SelectedIndex = 0
	}

	if selection.Element == nil {
		selection.Element = selection.Candidates[selection.SelectedIndex].Element
	}

	return &selection, nil
}

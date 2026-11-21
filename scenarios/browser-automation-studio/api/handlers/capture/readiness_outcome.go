package capture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// declaredReadinessOutcome reads the semantic lifecycle observed by the
// driver's declared-profile wait nodes. It is deliberately best-effort: the
// capture itself remains valid when old drivers or external pages do not emit
// this telemetry.
func declaredReadinessOutcome(outDir string) (string, error) {
	timing, err := readinessTiming(outDir)
	return timing.outcome, err
}

type readinessTimelineTiming struct {
	outcome         string
	navigationMS    int64
	readinessWaitMS int64
}

func readinessTiming(outDir string) (readinessTimelineTiming, error) {
	timing := readinessTimelineTiming{}
	raw, err := os.ReadFile(filepath.Join(outDir, "timeline.json"))
	if err != nil {
		return readinessTimelineTiming{}, fmt.Errorf("read timeline.json: %w", err)
	}
	var timeline struct {
		Frames []struct {
			StepType             string         `json:"step_type"`
			DurationMS           int64          `json:"duration_ms"`
			ExtractedDataPreview map[string]any `json:"extracted_data_preview"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(raw, &timeline); err != nil {
		return readinessTimelineTiming{}, fmt.Errorf("decode timeline.json: %w", err)
	}

	// A required region error is the page-level terminal failure. Partial is
	// surfaced next so callers can distinguish a degraded-but-usable capture;
	// an all-empty page remains a valid, explicit terminal state.
	seenReady := false
	seenEmpty := false
	for _, frame := range timeline.Frames {
		if frame.StepType == "navigate" {
			timing.navigationMS += frame.DurationMS
		}
		if frame.StepType != "wait" {
			continue
		}
		timing.readinessWaitMS += frame.DurationMS
		state, _ := frame.ExtractedDataPreview["experience_surface_state"].(string)
		switch state {
		case "error":
			timing.outcome = "error"
			return timing, nil
		case "partial":
			timing.outcome = "partial"
			return timing, nil
		case "ready":
			seenReady = true
		case "empty":
			seenEmpty = true
		}
	}
	if seenReady {
		timing.outcome = "ready"
		return timing, nil
	}
	if seenEmpty {
		timing.outcome = "empty"
		return timing, nil
	}
	return timing, nil
}

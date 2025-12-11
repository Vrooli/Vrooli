package analyzer

import (
	"fmt"
	"math"

	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// detectExcessiveTime checks if a step took too long.
func (a *Analyzer) detectExcessiveTime(stepIndex int, durationMs int64) []contracts.FrictionSignal {
	if durationMs <= a.config.ExcessiveStepDurationMs {
		return nil
	}

	severity := contracts.SeverityLow
	score := float64(durationMs-a.config.ExcessiveStepDurationMs) / float64(a.config.ExcessiveStepDurationMs) * 50
	if score > 75 {
		severity = contracts.SeverityHigh
	} else if score > 50 {
		severity = contracts.SeverityMedium
	}

	return []contracts.FrictionSignal{{
		Type:        contracts.FrictionExcessiveTime,
		StepIndex:   stepIndex,
		Severity:    severity,
		Score:       math.Min(score, 100),
		Description: fmt.Sprintf("Step took %dms, expected <%dms", durationMs, a.config.ExcessiveStepDurationMs),
		Evidence: map[string]any{
			"actual_ms":   durationMs,
			"expected_ms": a.config.ExcessiveStepDurationMs,
		},
	}}
}

// detectZigZag checks for erratic cursor movement.
func (a *Analyzer) detectZigZag(stepIndex int, path *contracts.CursorPath) []contracts.FrictionSignal {
	if path == nil || len(path.Points) < 2 {
		return nil
	}

	// A directness < 0.7 means path is > 30% longer than direct route
	if path.Directness >= (1.0 - a.config.ZigZagThreshold) {
		return nil
	}

	score := (1.0 - path.Directness) * 100
	severity := contracts.SeverityLow
	if score > 60 {
		severity = contracts.SeverityHigh
	} else if score > 40 {
		severity = contracts.SeverityMedium
	}

	return []contracts.FrictionSignal{{
		Type:      contracts.FrictionZigZagPath,
		StepIndex: stepIndex,
		Severity:  severity,
		Score:     score,
		Description: fmt.Sprintf("Cursor path efficiency %.0f%% (%.0fpx traveled, %.0fpx direct)",
			path.Directness*100, path.TotalDistancePx, path.DirectDistance),
		Evidence: map[string]any{
			"directness":      path.Directness,
			"total_distance":  path.TotalDistancePx,
			"direct_distance": path.DirectDistance,
			"zigzag_score":    path.ZigZagScore,
		},
	}}
}

// detectHesitations checks for pauses in cursor movement.
func (a *Analyzer) detectHesitations(stepIndex int, path *contracts.CursorPath) []contracts.FrictionSignal {
	if path == nil || path.Hesitations == 0 {
		return nil
	}

	score := float64(path.Hesitations) * 15 // 15 points per hesitation
	severity := contracts.SeverityLow
	if path.Hesitations >= 3 {
		severity = contracts.SeverityHigh
	} else if path.Hesitations >= 2 {
		severity = contracts.SeverityMedium
	}

	return []contracts.FrictionSignal{{
		Type:        contracts.FrictionLongHesitation,
		StepIndex:   stepIndex,
		Severity:    severity,
		Score:       math.Min(score, 100),
		Description: fmt.Sprintf("%d hesitation(s) detected (>200ms pauses)", path.Hesitations),
		Evidence: map[string]any{
			"hesitation_count": path.Hesitations,
		},
	}}
}

// detectRapidClicks checks for frustrated clicking.
func (a *Analyzer) detectRapidClicks(stepIndex int, traces []contracts.InteractionTrace) []contracts.FrictionSignal {
	clicks := filterClicks(traces)
	if len(clicks) < a.config.RapidClickCountThreshold {
		return nil
	}

	// Check for rapid succession
	rapidCount := 0
	for i := 1; i < len(clicks); i++ {
		gap := clicks[i].Timestamp.Sub(clicks[i-1].Timestamp).Milliseconds()
		if gap < a.config.RapidClickWindowMs {
			rapidCount++
		}
	}

	if rapidCount < a.config.RapidClickCountThreshold-1 {
		return nil
	}

	score := float64(rapidCount) * 25
	severity := contracts.SeverityMedium
	if rapidCount >= 4 {
		severity = contracts.SeverityHigh
	}

	return []contracts.FrictionSignal{{
		Type:        contracts.FrictionRapidClicks,
		StepIndex:   stepIndex,
		Severity:    severity,
		Score:       math.Min(score, 100),
		Description: fmt.Sprintf("%d rapid clicks detected (possible frustration)", rapidCount+1),
		Evidence: map[string]any{
			"rapid_click_count": rapidCount + 1,
			"window_ms":         a.config.RapidClickWindowMs,
		},
	}}
}

// detectMultipleRetries checks for steps that required multiple attempts.
func (a *Analyzer) detectMultipleRetries(stepIndex int, retryCount int) []contracts.FrictionSignal {
	if retryCount == 0 {
		return nil
	}

	score := float64(retryCount) * 30 // 30 points per retry
	severity := contracts.SeverityLow
	if retryCount >= 3 {
		severity = contracts.SeverityHigh
	} else if retryCount >= 2 {
		severity = contracts.SeverityMedium
	}

	return []contracts.FrictionSignal{{
		Type:        contracts.FrictionMultipleRetries,
		StepIndex:   stepIndex,
		Severity:    severity,
		Score:       math.Min(score, 100),
		Description: fmt.Sprintf("Step required %d retry attempt(s)", retryCount),
		Evidence: map[string]any{
			"retry_count": retryCount,
		},
	}}
}

// calculateFrictionScore computes a weighted average of signal scores.
func (a *Analyzer) calculateFrictionScore(signals []contracts.FrictionSignal) float64 {
	if len(signals) == 0 {
		return 0
	}

	// Weighted by severity
	weights := map[contracts.Severity]float64{
		contracts.SeverityLow:    0.5,
		contracts.SeverityMedium: 1.0,
		contracts.SeverityHigh:   2.0,
	}

	totalWeight := 0.0
	weightedSum := 0.0
	for _, s := range signals {
		w := weights[s.Severity]
		totalWeight += w
		weightedSum += s.Score * w
	}

	if totalWeight == 0 {
		return 0
	}

	return math.Min(weightedSum/totalWeight, 100)
}

func filterClicks(traces []contracts.InteractionTrace) []contracts.InteractionTrace {
	clicks := make([]contracts.InteractionTrace, 0)
	for _, t := range traces {
		if t.ActionType == contracts.ActionClick {
			clicks = append(clicks, t)
		}
	}
	return clicks
}

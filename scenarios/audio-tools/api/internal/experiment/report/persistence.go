//go:build audio_lab

package report

import (
	"encoding/json"
	"fmt"
	"time"

	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	exprecipe "audio-tools/internal/experiment/recipe"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func experimentRunsForReport(report inteval.EvalReport, reportProto *evalv1.EvalReport, realized map[string]any) ([]intexp.Run, error) {
	protoRows := reportProto.GetPerStrategy()
	if len(report.PerStrategy) != len(protoRows) {
		return nil, fmt.Errorf("experiment report row mismatch: internal=%d proto=%d", len(report.PerStrategy), len(protoRows))
	}
	runs := make([]intexp.Run, 0, len(protoRows))
	for i, strategyReport := range protoRows {
		condition, err := experimentRunConditionJSON(report.PerStrategy[i], realized)
		if err != nil {
			return nil, fmt.Errorf("marshal strategy condition: %w", err)
		}
		runs = append(runs, intexp.Run{
			Strategy:      strategyReport.GetStrategy(),
			ConditionJSON: condition,
		})
	}
	return runs, nil
}

// ExperimentRunsForReport produces persistence rows from a realized report.

func ExperimentRunsForReport(report inteval.EvalReport, reportProto *evalv1.EvalReport, realized map[string]any) ([]intexp.Run, error) {
	return experimentRunsForReport(report, reportProto, realized)
}

// ExperimentRunsForMetricsOnlyReport stores only strategy and route identity
// alongside the metrics-only report artifact. It deliberately omits labels,
// realized input details, and diagnostics that could contain user text.
func ExperimentRunsForMetricsOnlyReport(report inteval.EvalReport, reportProto *evalv1.EvalReport) ([]intexp.Run, error) {
	protoRows := reportProto.GetPerStrategy()
	if len(report.PerStrategy) != len(protoRows) {
		return nil, fmt.Errorf("experiment report row mismatch: internal=%d proto=%d", len(report.PerStrategy), len(protoRows))
	}
	runs := make([]intexp.Run, 0, len(protoRows))
	for i, row := range report.PerStrategy {
		condition, err := json.Marshal(map[string]any{
			"strategy": string(row.Strategy), "engine_id": row.EngineID,
			"model_id": row.ModelID, "policy_profile": row.PolicyProfile,
			"replay_lane": row.ReplayLane, "fault_profile": row.FaultProfile,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal metrics-only strategy condition: %w", err)
		}
		runs = append(runs, intexp.Run{Strategy: protoRows[i].GetStrategy(), ConditionJSON: condition})
	}
	return runs, nil
}

func experimentRunConditionJSON(row inteval.StrategyReport, realized map[string]any) ([]byte, error) {
	baseStrategy := row.BaseStrategy
	if baseStrategy == "" {
		baseStrategy = row.Strategy
	}
	return json.Marshal(map[string]any{
		"strategy":        string(row.Strategy),
		"base_strategy":   string(baseStrategy),
		"label":           row.Label,
		"engine_id":       row.EngineID,
		"policy_profile":  row.PolicyProfile,
		"replay_lane":     row.ReplayLane,
		"fault_profile":   row.FaultProfile,
		"condition_group": row.ConditionGroup,
		"speaker": map[string]any{
			"extraction_enabled":   row.ExtractionEnabled,
			"verification_enabled": row.VerificationEnabled,
		},
		"realized": realized,
	})
}

func longFormSourceDurationWarning(longForm *experimentv1.LongFormRecipe, clips []exprecipe.Clip) (inteval.ReportWarning, bool) {
	if longForm == nil {
		return inteval.ReportWarning{}, false
	}
	targets := longForm.GetSweepDurationsSeconds()
	if len(targets) == 0 && longForm.GetTargetDurationSeconds() > 0 {
		targets = []int32{longForm.GetTargetDurationSeconds()}
	}
	var maxTargetSec int32
	for _, target := range targets {
		if target > maxTargetSec {
			maxTargetSec = target
		}
	}
	if maxTargetSec <= 0 {
		return inteval.ReportWarning{}, false
	}
	sourceMs := sourceDurationMs(clips)
	targetMs := int64(maxTargetSec) * int64(time.Second/time.Millisecond)
	if sourceMs <= 0 || sourceMs >= targetMs {
		return inteval.ReportWarning{}, false
	}
	return inteval.ReportWarning{
		Code:     "source_audio_under_target",
		Severity: "warning",
		Message:  fmt.Sprintf("The selected corpus provides %.1f seconds of unique source audio, below the requested long-form target of %d seconds; generated inputs may reuse clips.", float64(sourceMs)/1000, maxTargetSec),
	}, true
}

func sourceDurationMs(clips []exprecipe.Clip) int64 {
	var total int64
	for _, clip := range clips {
		sampleRate := clip.SampleRate
		if sampleRate <= 0 {
			sampleRate = exprecipe.CanonicalSampleRate
		}
		if sampleRate <= 0 {
			continue
		}
		total += int64(float64(len(clip.PCM)) / float64(sampleRate*2) * float64(time.Second/time.Millisecond))
	}
	return total
}

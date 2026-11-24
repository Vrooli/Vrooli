//go:build audio_lab

package eval

import (
	"fmt"
	"math"
)

func chooseWinner(rows []StrategyReport, latencyMeasured bool) int {
	return chooseWinnerWithScaling(rows, latencyMeasured, true)
}

func chooseAggregateWinner(rows []StrategyReport, latencyMeasured bool) int {
	return chooseWinnerWithScaling(rows, latencyMeasured, false)
}

func chooseWinnerWithScaling(rows []StrategyReport, latencyMeasured bool, scalingAware bool) int {
	best := 0
	for i := 1; i < len(rows); i++ {
		if strategyLess(rows[i], rows[best], latencyMeasured, scalingAware) {
			best = i
		}
	}
	return best
}

func strategyLess(a, b StrategyReport, latencyMeasured bool, scalingAware bool) bool {
	if strategySafetyPassed(a) != strategySafetyPassed(b) {
		return strategySafetyPassed(a)
	}
	if math.Abs(a.WER-b.WER) > reportWERTieThreshold {
		return a.WER < b.WER
	}
	if scalingAware {
		aRisk, bRisk := longFormScalingRisk(a.Scaling), longFormScalingRisk(b.Scaling)
		if aRisk != bRisk {
			return aRisk < bRisk
		}
	}
	if latencyMeasured && math.Abs(a.FinalizationLatencyP95Ms-b.FinalizationLatencyP95Ms) > 25 {
		return a.FinalizationLatencyP95Ms < b.FinalizationLatencyP95Ms
	}
	if a.WhisperCalls != b.WhisperCalls {
		return a.WhisperCalls < b.WhisperCalls
	}
	return string(a.Strategy) < string(b.Strategy)
}

func corpusConfidence(clips int, audioSeconds float64, latencyMeasured bool) (string, []string) {
	notes := []string{}
	if clips < 10 {
		notes = append(notes, "Fewer than 10 clips makes per-strategy differences easy to overfit.")
	}
	if audioSeconds > 0 && audioSeconds < 120 {
		notes = append(notes, "Less than 120 seconds of audio makes latency and WER less stable.")
	}
	if !latencyMeasured {
		notes = append(notes, "Latency was not measured; the recommendation is quality/cost-only.")
	}
	if len(notes) > 0 {
		return "low", notes
	}
	if clips >= 25 && audioSeconds >= 300 && latencyMeasured {
		return "high", []string{"Corpus size and latency repeats are sufficient for a stronger local recommendation."}
	}
	return "medium", []string{"Recommendation is suitable for local tuning; confirm with a broader corpus before promotion."}
}

func explainStrategy(row, winner StrategyReport, isWinner bool, latencyMeasured bool) (string, []string, []ReportWarning) {
	scalingWarnings := append([]ReportWarning(nil), row.Scaling.Warnings...)
	scalingRisk := longFormScalingRisk(row.Scaling)
	if isWinner {
		if !strategySafetyPassed(row) {
			return "recommended-despite-safety-failure", []string{
				"All evaluated strategies failed the safety gate; this is the least-bad diagnostic result, not a promotion recommendation.",
			}, append(scalingWarnings, safetyFailureWarning(false))
		}
		reasons := []string{"Lowest WER after deterministic normalization."}
		if latencyMeasured {
			reasons = append(reasons, "Best tie-break balance of p95 finalization latency and compute cost.")
		} else {
			reasons = append(reasons, "Latency was not part of this run.")
		}
		if scalingRisk == 0 {
			reasons = append(reasons, "Long-form scaling is acceptable over the measured sweep.")
		}
		return "winner", reasons, scalingWarnings
	}
	reasons := []string{}
	warnings := scalingWarnings
	verdict := "competitive"
	if !strategySafetyPassed(row) {
		verdict = "loser"
		reasons = append(reasons, "Safety gate failed while at least one evaluated strategy passed.")
		warnings = append(warnings, safetyFailureWarning(true))
	}
	if row.WERDeltaVsWinner > 0.005 {
		verdict = "loser"
		reasons = append(reasons, fmt.Sprintf("WER is %.1f percentage points worse than the winner.", row.WERDeltaVsWinner*100))
		warnings = append(warnings, ReportWarning{Code: "higher_wer", Severity: "warning", Message: "WER is materially higher than the winning strategy."})
	}
	if row.CallMultiplierVsWinner > 1.5 {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, fmt.Sprintf("Uses %.1fx as many Whisper calls as the winner.", row.CallMultiplierVsWinner))
		warnings = append(warnings, ReportWarning{Code: "higher_compute", Severity: "warning", Message: "Compute cost is materially higher than the winning strategy."})
	}
	if latencyMeasured && row.P95DeltaMsVsWinner > 250 {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, fmt.Sprintf("p95 finalization is %.0f ms slower than the winner.", row.P95DeltaMsVsWinner))
		warnings = append(warnings, ReportWarning{Code: "higher_p95", Severity: "warning", Message: "Finalization latency is materially slower than the winning strategy."})
	}
	if row.PartialRevisions > winner.PartialRevisions+3 {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, "Shows more partial transcript revisions, so live text is less stable.")
		warnings = append(warnings, ReportWarning{Code: "higher_revisions", Severity: "warning", Message: "Partial transcript revisions are higher than the winning strategy."})
	}
	if scalingRisk > longFormScalingRisk(winner.Scaling) {
		if verdict == "competitive" {
			verdict = "tradeoff"
		}
		reasons = append(reasons, scalingRiskReason(row.Scaling))
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "Close to the winner on measured metrics, but loses the deterministic tie-breaks.")
	}
	return verdict, reasons, warnings
}

func winnerReasons(winner StrategyReport, rows []StrategyReport, latencyMeasured bool, aggregateWinnerIndex int) []string {
	reasons := []string{fmt.Sprintf("%s has %.1f%% WER on this corpus.", displayLabel(winner), winner.WER*100)}
	if !strategySafetyPassed(winner) {
		reasons = append(reasons, "Every evaluated strategy failed the safety gate; the winner is recommended despite safety failure.")
	}
	if latencyMeasured {
		reasons = append(reasons, fmt.Sprintf("p95 finalization is %.0f ms with %d Whisper calls.", winner.FinalizationLatencyP95Ms, winner.WhisperCalls))
	} else {
		reasons = append(reasons, fmt.Sprintf("Uses %d Whisper calls over %.1f seconds of audio.", winner.WhisperCalls, winner.WhisperAudioSeconds))
	}
	if aggregateWinnerIndex >= 0 && aggregateWinnerIndex < len(rows) && rows[aggregateWinnerIndex].Strategy != winner.Strategy {
		reasons = append(reasons, fmt.Sprintf("Short-form aggregate tie-breaks favored %s, but %s is safer for long dictation over the measured sweep.", displayLabel(rows[aggregateWinnerIndex]), displayLabel(winner)))
	}
	if longFormScalingRisk(winner.Scaling) == 0 {
		reasons = append(reasons, fmt.Sprintf("Measured scaling is latency=%s and compute=%s.", winner.Scaling.LatencyClassification, winner.Scaling.ComputeClassification))
	}
	for _, row := range rows {
		if row.Strategy == "overlap_agree" && row.Strategy != winner.Strategy && row.WhisperCalls > winner.WhisperCalls {
			reasons = append(reasons, "Overlap-agree is not worth its extra calls on this corpus unless a later run shows better WER or stability.")
			break
		}
	}
	return reasons
}

func strategySafetyPassed(row StrategyReport) bool {
	if row.Safety.Passed {
		return true
	}
	return len(row.Safety.Reasons) == 0 &&
		len(row.Safety.RetractionEvents) == 0 &&
		row.Safety.MaxDroppedSpanWords == 0 &&
		row.Safety.DroppedSpanThresholdWords == 0
}

func safetyFailureWarning(safeAlternativeExists bool) ReportWarning {
	message := "Safety gate failed; this strategy is not eligible while any safe strategy exists."
	if !safeAlternativeExists {
		message = "Safety gate failed for every evaluated strategy; this winner is diagnostic only."
	}
	return ReportWarning{
		Code:     "safety_failed",
		Severity: "error",
		Message:  message,
	}
}

func longFormScalingRisk(s ScalingAnalysis) int {
	if distinctPointDurations(s.Points) < minScalingDistinctDurations {
		return 1
	}
	if s.LatencyClassification == "superlinear" || s.ComputeClassification == "superlinear" {
		return 2
	}
	if s.LatencyClassification == "inconclusive" || s.ComputeClassification == "inconclusive" {
		return 1
	}
	return 0
}

func scalingRiskReason(s ScalingAnalysis) string {
	if s.LatencyClassification == "superlinear" && s.ComputeClassification == "superlinear" {
		return "Shows superlinear latency and compute growth over the measured duration sweep."
	}
	if s.LatencyClassification == "superlinear" {
		return "Shows superlinear finalization-latency growth over the measured duration sweep."
	}
	if s.ComputeClassification == "superlinear" {
		return "Shows superlinear compute growth over the measured duration sweep."
	}
	return "Scaling evidence is inconclusive for long-form dictation."
}

func callMultiplier(calls, winnerCalls int) float64 {
	if winnerCalls <= 0 {
		if calls <= 0 {
			return 1
		}
		return 0
	}
	return float64(calls) / float64(winnerCalls)
}

func displayLabel(row StrategyReport) string {
	if row.Label != "" {
		return row.Label
	}
	return string(row.Strategy)
}

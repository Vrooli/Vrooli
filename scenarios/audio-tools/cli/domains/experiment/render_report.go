package experiment

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func renderReport(ctx cliapp.RunContext, msg *experimentv1.GetExperimentReportResponse) error {
	if msg == nil || msg.GetExperiment() == nil {
		return fmt.Errorf("server returned no experiment")
	}
	exp := msg.GetExperiment()
	fmt.Fprintf(ctx.Stdout(), "Experiment %s (%s)\n", exp.GetId(), statusLabel(exp.GetStatus()))
	if errMsg := strings.TrimSpace(exp.GetError()); errMsg != "" {
		fmt.Fprintf(ctx.Stdout(), "Error: %s\n", errMsg)
	}
	printReportTable(ctx, msg.GetReport())
	return nil
}

func printReportTable(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	if report == nil || len(report.GetPerStrategy()) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No report rows available.")
		return
	}
	printReportSummary(ctx, report)
	fmt.Fprintf(ctx.Stdout(), "%-34s  %6s  %6s  %6s  %8s  %9s  %9s  %8s  %8s  %8s  %12s\n",
		"STRATEGY", "WER%", "CALLS", "RTF", "AUDIO_S", "LAT_P50", "LAT_P95", "REVISES", "SAFE", "MAXDROP", "VERDICT")
	for _, s := range report.GetPerStrategy() {
		lat50, lat95 := "-", "-"
		if report.GetLatencyMeasured() {
			lat50 = fmt.Sprintf("%.0fms", s.GetFinalizationLatencyP50Ms())
			lat95 = fmt.Sprintf("%.0fms", s.GetFinalizationLatencyP95Ms())
		}
		safe := "pass"
		if s.GetSafety() != nil && !s.GetSafety().GetPassed() {
			safe = "fail"
		}
		maxDrop := int32(0)
		if s.GetSafety() != nil {
			maxDrop = s.GetSafety().GetMaxDroppedSpanWords()
		}
		verdict := s.GetVerdict()
		if verdict == "" {
			verdict = "-"
		}
		fmt.Fprintf(ctx.Stdout(), "%-34s  %6.1f  %6d  %6.2f  %8.2f  %9s  %9s  %8d  %8s  %8d  %12s\n",
			s.GetLabel(), s.GetWer()*100, s.GetWhisperCalls(), s.GetRtf(),
			s.GetWhisperAudioSeconds(), lat50, lat95, s.GetPartialRevisions(), safe, maxDrop, verdict)
	}
	printReportWarnings(ctx, report)
	printLengthCurves(ctx, report)
	printScalingAnalysis(ctx, report)
}

func estimatedSecondsLabel(seconds int32) string {
	if seconds <= 0 {
		return ""
	}
	if seconds < 60 {
		return fmt.Sprintf("estimated_seconds=%d", seconds)
	}
	return fmt.Sprintf("estimated_seconds=%d (~%dm%02ds)", seconds, seconds/60, seconds%60)
}

func printReportSummary(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	summary := report.GetSummary()
	if summary == nil || summary.GetRecommendation() == "" {
		return
	}
	confidence := summary.GetConfidence()
	if confidence == "" {
		confidence = "unknown"
	}
	fmt.Fprintf(ctx.Stdout(), "Recommendation: %s (confidence: %s)\n", summary.GetRecommendation(), confidence)
	for _, reason := range summary.GetReasons() {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", reason)
	}
	for _, note := range summary.GetConfidenceNotes() {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", note)
	}
	fmt.Fprintln(ctx.Stdout())
}

func printReportWarnings(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	warnings := report.GetWarnings()
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nWarnings:")
	for _, w := range warnings {
		severity := w.GetSeverity()
		if severity == "" {
			severity = "info"
		}
		code := w.GetCode()
		if code == "" {
			code = "warning"
		}
		fmt.Fprintf(ctx.Stdout(), "  - %s/%s: %s\n", severity, code, w.GetMessage())
	}
}

func printLengthCurves(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	hasCurves := false
	for _, s := range report.GetPerStrategy() {
		if len(s.GetLengthCurves()) > 0 {
			hasCurves = true
			break
		}
	}
	if !hasCurves {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nLength curves:")
	for _, s := range report.GetPerStrategy() {
		if len(s.GetLengthCurves()) == 0 {
			continue
		}
		fmt.Fprintf(ctx.Stdout(), "  %s\n", s.GetLabel())
		fmt.Fprintf(ctx.Stdout(), "    %-12s  %5s  %6s  %9s  %9s  %8s\n", "BUCKET", "CLIPS", "WER%", "P95", "TTFC", "MAXDROP")
		for _, curve := range s.GetLengthCurves() {
			p95, ttfc := "-", "-"
			if report.GetLatencyMeasured() {
				p95 = fmt.Sprintf("%.0fms", curve.GetFinalizationLatencyP95Ms())
				ttfc = fmt.Sprintf("%.0fms", curve.GetMeanTimeToFirstCommitMs())
			}
			fmt.Fprintf(ctx.Stdout(), "    %-12s  %5d  %6.1f  %9s  %9s  %8d\n",
				curve.GetBucket(), curve.GetClipCount(), curve.GetWer()*100, p95, ttfc, curve.GetMaxDroppedSpanWords())
		}
	}
}

func printScalingAnalysis(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	hasScaling := false
	for _, s := range report.GetPerStrategy() {
		if scalingHasSignal(s.GetScaling()) {
			hasScaling = true
			break
		}
	}
	if !hasScaling {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nScaling analysis:")
	fmt.Fprintf(ctx.Stdout(), "  %-22s  %-12s  %-12s  %-10s  %-8s  %-18s  %s\n", "STRATEGY", "LATENCY", "COMPUTE", "CONF", "POINTS", "FIT", "WARNING")
	for _, s := range report.GetPerStrategy() {
		scaling := s.GetScaling()
		if !scalingHasSignal(scaling) {
			continue
		}
		fmt.Fprintf(ctx.Stdout(), "  %-22s  %-12s  %-12s  %-10s  %-8d  %-18s  %s\n",
			truncate(strategyLabel(s), 22),
			scaling.GetLatencyClassification(),
			scaling.GetComputeClassification(),
			scaling.GetConfidence(),
			len(scaling.GetPoints()),
			scalingFitLabel(scaling.GetLatencyFit()),
			scalingWarningLabel(scaling))
	}
}

func scalingHasSignal(s *evalv1.ScalingAnalysis) bool {
	return s != nil && (len(s.GetPoints()) > 0 || s.GetLatencyClassification() != "" || s.GetComputeClassification() != "")
}

func scalingWarningLabel(s *evalv1.ScalingAnalysis) string {
	for _, warning := range s.GetWarnings() {
		if warning.GetCode() != "" {
			return warning.GetCode()
		}
	}
	return "-"
}

func scalingFitLabel(fit *evalv1.ScalingModelFit) string {
	if fit == nil || fit.GetModel() == "" || fit.GetModel() == "none" {
		return "-"
	}
	if fit.GetExponent() > 0 {
		return fmt.Sprintf("%s R2=%.2f exp=%.2f", fit.GetModel(), fit.GetRSquared(), fit.GetExponent())
	}
	return fmt.Sprintf("%s R2=%.2f", fit.GetModel(), fit.GetRSquared())
}

func strategyLabel(s *evalv1.StrategyReport) string {
	if s.GetLabel() != "" {
		return s.GetLabel()
	}
	return s.GetStrategy()
}

func formatRunStatus(run *experimentv1.ExperimentRun) string {
	if run == nil {
		return "(nil run)"
	}
	condition := strings.TrimSpace(run.GetConditionJson())
	if condition != "" && condition != "{}" {
		return fmt.Sprintf("%s - condition=%s", run.GetStrategy(), condition)
	}
	return fmt.Sprintf("%s - completed", run.GetStrategy())
}

func experimentTerminal(exp *experimentv1.Experiment) bool {
	if exp == nil {
		return false
	}
	switch exp.GetStatus() {
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED,
		experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED,
		experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED:
		return true
	default:
		return false
	}
}

// winnerStrategyRow returns the report's winning strategy row, preferring the
// summary's declared winner, then a verdict=="winner" row, then lowest WER.
func winnerStrategyRow(report *evalv1.EvalReport) *evalv1.StrategyReport {
	if report == nil {
		return nil
	}
	rows := report.GetPerStrategy()
	if len(rows) == 0 {
		return nil
	}
	if summary := report.GetSummary(); summary != nil {
		if ws := summary.GetWinnerStrategy(); ws != "" {
			for _, r := range rows {
				if r.GetStrategy() == ws {
					return r
				}
			}
		}
		if wl := summary.GetWinnerLabel(); wl != "" {
			for _, r := range rows {
				if r.GetLabel() == wl {
					return r
				}
			}
		}
	}
	for _, r := range rows {
		if r.GetVerdict() == "winner" {
			return r
		}
	}
	best := rows[0]
	for _, r := range rows[1:] {
		if r.GetWer() < best.GetWer() {
			best = r
		}
	}
	return best
}

func safetyLabel(safety *evalv1.SafetyGateReport) string {
	if safety == nil {
		return "-"
	}
	if safety.GetPassed() {
		return "SAFE"
	}
	return "UNSAFE"
}

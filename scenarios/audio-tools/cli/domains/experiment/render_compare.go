package experiment

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

// comparisonRow is the per-experiment projection the comparison table renders:
// the experiment, its report (may be nil), and the winning strategy row.
type comparisonRow struct {
	exp    *experimentv1.Experiment
	report *evalv1.EvalReport
	winner *evalv1.StrategyReport
}

func printComparison(ctx cliapp.RunContext, experiments []*experimentv1.ComparedExperiment) {
	if len(experiments) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No experiments returned.")
		return
	}
	rows := make([]comparisonRow, 0, len(experiments))
	bestIdx := -1
	anyMissingReport := false
	for i, ce := range experiments {
		row := comparisonRow{exp: ce.GetExperiment(), report: ce.GetReport()}
		if row.report != nil {
			row.winner = winnerStrategyRow(row.report)
		} else {
			anyMissingReport = true
		}
		rows = append(rows, row)
		if row.winner == nil {
			continue
		}
		if bestIdx == -1 {
			bestIdx = i
			continue
		}
		best := rows[bestIdx].winner
		if row.winner.GetWer() < best.GetWer() ||
			(row.winner.GetWer() == best.GetWer() && row.winner.GetRtf() < best.GetRtf()) {
			bestIdx = i
		}
	}
	var bestWinner *evalv1.StrategyReport
	if bestIdx >= 0 {
		bestWinner = rows[bestIdx].winner
	}

	fmt.Fprintf(ctx.Stdout(), "%-2s %-18s %-38s %-10s %-22s %7s %8s %12s %6s %6s %7s %7s %7s\n",
		"", "NAME", "ID", "STATUS", "WINNER", "WER%", "P95", "SCALE", "CALLS", "RTF", "SAFE", "dWER%", "dRTF")
	for i, row := range rows {
		mark := ""
		if i == bestIdx {
			mark = "*"
		}
		name := row.exp.GetName()
		if name == "" {
			name = "-"
		}
		winnerLabel, werCol, p95Col, scaleCol, callsCol, rtfCol, safeCol, dWerCol, dRtfCol := "-", "-", "-", "-", "-", "-", "-", "-", "-"
		if row.winner != nil {
			winnerLabel = strategyLabel(row.winner)
			werCol = fmt.Sprintf("%.1f", row.winner.GetWer()*100)
			callsCol = fmt.Sprintf("%d", row.winner.GetWhisperCalls())
			rtfCol = fmt.Sprintf("%.2f", row.winner.GetRtf())
			if scalingHasSignal(row.winner.GetScaling()) {
				scaleCol = fmt.Sprintf("%s/%s", row.winner.GetScaling().GetLatencyClassification(), row.winner.GetScaling().GetComputeClassification())
			}
			if row.report.GetLatencyMeasured() {
				p95Col = fmt.Sprintf("%.0fms", row.winner.GetFinalizationLatencyP95Ms())
			}
			safeCol = safetyLabel(row.winner.GetSafety())
			if bestWinner != nil {
				dWerCol = fmt.Sprintf("%+.1f", (row.winner.GetWer()-bestWinner.GetWer())*100)
				dRtfCol = fmt.Sprintf("%+.2f", row.winner.GetRtf()-bestWinner.GetRtf())
			}
		}
		fmt.Fprintf(ctx.Stdout(), "%-2s %-18s %-38s %-10s %-22s %7s %8s %12s %6s %6s %7s %7s %7s\n",
			mark, truncate(name, 18), row.exp.GetId(), statusLabel(row.exp.GetStatus()),
			truncate(winnerLabel, 22), werCol, p95Col, truncate(scaleCol, 12), callsCol, rtfCol, safeCol, dWerCol, dRtfCol)
	}
	if bestIdx >= 0 {
		fmt.Fprintf(ctx.Stdout(), "\n* best = lowest winner WER (tie-break lower RTF); deltas are vs that experiment.\n")
	}
	if anyMissingReport {
		fmt.Fprintln(ctx.Stdout(), "Rows without a winner have no stored report yet (still running, or failed before reporting).")
	}
	printRecipeDiff(ctx, rows)
	printStrategyAlignment(ctx, rows)
	exps := make([]*experimentv1.Experiment, 0, len(rows))
	for _, row := range rows {
		exps = append(exps, row.exp)
	}
	printExperimentErrors(ctx, exps)
	fmt.Fprintln(ctx.Stdout(), "Use --json for full recipes and per-experiment report payloads.")
}

func printRecipeDiff(ctx cliapp.RunContext, rows []comparisonRow) {
	diffs := recipeDiffLines(rows)
	if len(diffs) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nRecipe differences:")
	for _, line := range diffs {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", line)
	}
}

func recipeDiffLines(rows []comparisonRow) []string {
	if len(rows) < 2 {
		return nil
	}
	perExperiment := make([]map[string]string, 0, len(rows))
	fieldSet := map[string]struct{}{}
	for _, row := range rows {
		fields := recipeFields(row.exp.GetRecipe())
		perExperiment = append(perExperiment, fields)
		for field := range fields {
			fieldSet[field] = struct{}{}
		}
	}
	fields := make([]string, 0, len(fieldSet))
	for field := range fieldSet {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	var out []string
	for _, field := range fields {
		first := perExperiment[0][field]
		changed := false
		for _, values := range perExperiment[1:] {
			if values[field] != first {
				changed = true
				break
			}
		}
		if !changed {
			continue
		}
		if len(rows) == 2 {
			out = append(out, fmt.Sprintf("%s: %s -> %s", field, valueOrDash(perExperiment[0][field]), valueOrDash(perExperiment[1][field])))
			continue
		}
		parts := make([]string, 0, len(rows))
		for i, row := range rows {
			parts = append(parts, fmt.Sprintf("%s=%s", experimentShortID(row.exp), valueOrDash(perExperiment[i][field])))
		}
		out = append(out, fmt.Sprintf("%s: %s", field, strings.Join(parts, ", ")))
	}
	return out
}

func recipeFields(recipe *experimentv1.ExperimentRecipe) map[string]string {
	fields := map[string]string{}
	if recipe == nil {
		return fields
	}
	fields["clip_ids"] = strings.Join(recipe.GetClipIds(), ",")
	fields["realtime_repeats"] = strconv.Itoa(int(recipe.GetRealtimeRepeats()))
	fields["chunk_ms"] = strconv.Itoa(int(recipe.GetChunkMs()))
	fields["seed"] = strconv.FormatInt(recipe.GetSeed(), 10)
	fields["dropped_span_threshold_words"] = strconv.Itoa(int(recipe.GetDroppedSpanThresholdWords()))
	fields["latency_tail_seconds"] = strconv.Itoa(int(recipe.GetLatencyTailSeconds()))
	if lf := recipe.GetLongForm(); lf != nil {
		fields["long_form.enabled"] = strconv.FormatBool(lf.GetEnabled())
		fields["long_form.target_duration_seconds"] = strconv.Itoa(int(lf.GetTargetDurationSeconds()))
		fields["long_form.gap_ms"] = strconv.Itoa(int(lf.GetGapMs()))
		fields["long_form.tag_contains"] = lf.GetTagContains()
		fields["long_form.sweep_durations_seconds"] = int32sCSV(lf.GetSweepDurationsSeconds())
	}
	if aug := recipe.GetAugmentation(); aug != nil {
		fields["augmentation.noise_types"] = strings.Join(aug.GetNoiseTypes(), ",")
		fields["augmentation.snr_db"] = float64sCSV(aug.GetSnrDb())
		fields["augmentation.competing_voice_ids"] = strings.Join(aug.GetCompetingVoiceIds(), ",")
		fields["augmentation.competing_text"] = aug.GetCompetingText()
	}
	if speaker := recipe.GetSpeaker(); speaker != nil {
		fields["speaker.target_profile_id"] = speaker.GetTargetProfileId()
		fields["speaker.extraction_enabled"] = strconv.FormatBool(speaker.GetExtractionEnabled())
		fields["speaker.verification_enabled"] = strconv.FormatBool(speaker.GetVerificationEnabled())
		fields["speaker.verification_mode"] = speaker.GetVerificationMode().String()
		fields["speaker.threshold"] = fmt.Sprintf("%.3g", speaker.GetThreshold())
		fields["speaker.fallback_without_verification"] = strconv.FormatBool(speaker.GetFallbackWithoutVerification())
		fields["speaker.ablation_enabled"] = strconv.FormatBool(speaker.GetAblationEnabled())
	}
	for _, strategy := range recipe.GetStrategies() {
		key := strategyDiffKey(strategy, fields)
		fields[key+".kind"] = strategy.GetKind()
		fields[key+".overlap_max_window_ms"] = strconv.Itoa(int(strategy.GetOverlapMaxWindowMs()))
		fields[key+".overlap_max_stall_rejects"] = strconv.Itoa(int(strategy.GetOverlapMaxStallRejects()))
		fields[key+".overlap_window_ms"] = strconv.Itoa(int(strategy.GetOverlapWindowMs()))
		fields[key+".overlap_commit_runs"] = strconv.Itoa(int(strategy.GetOverlapCommitRuns()))
		fields[key+".vad_silence_ms"] = strconv.Itoa(int(strategy.GetVadSilenceMs()))
	}
	return fields
}

func strategyDiffKey(strategy *evalv1.EvalStrategy, fields map[string]string) string {
	base := "strategy." + valueOrDash(strategy.GetKind())
	key := base
	for i := 2; ; i++ {
		if _, exists := fields[key+".kind"]; !exists {
			return key
		}
		key = fmt.Sprintf("%s[%d]", base, i)
	}
}

func printStrategyAlignment(ctx cliapp.RunContext, rows []comparisonRow) {
	keys := alignedStrategyKeys(rows)
	if len(keys) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nBy-strategy alignment:")
	fmt.Fprintf(ctx.Stdout(), "  %-28s", "STRATEGY")
	for _, row := range rows {
		fmt.Fprintf(ctx.Stdout(), "  %-24s", experimentShortID(row.exp))
	}
	fmt.Fprintln(ctx.Stdout())
	for _, key := range keys {
		fmt.Fprintf(ctx.Stdout(), "  %-28s", truncate(key, 28))
		for _, row := range rows {
			fmt.Fprintf(ctx.Stdout(), "  %-24s", strategyMetricCell(row, key))
		}
		fmt.Fprintln(ctx.Stdout())
	}
}

func alignedStrategyKeys(rows []comparisonRow) []string {
	seen := map[string]struct{}{}
	for _, row := range rows {
		if row.report == nil {
			continue
		}
		for _, strategy := range row.report.GetPerStrategy() {
			seen[strategyAlignmentKey(strategy)] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func strategyMetricCell(row comparisonRow, key string) string {
	if row.report == nil {
		return "-"
	}
	for _, strategy := range row.report.GetPerStrategy() {
		if strategyAlignmentKey(strategy) != key {
			continue
		}
		p95 := "-"
		if row.report.GetLatencyMeasured() {
			p95 = fmt.Sprintf("%.0fms", strategy.GetFinalizationLatencyP95Ms())
		}
		return fmt.Sprintf("wer %.1f p95 %s", strategy.GetWer()*100, p95)
	}
	return "-"
}

func strategyAlignmentKey(strategy *evalv1.StrategyReport) string {
	if strategy.GetStrategy() != "" {
		return strings.TrimSpace(strings.SplitN(strategy.GetStrategy(), "/", 2)[0])
	}
	if strategy.GetLabel() != "" {
		return strings.TrimSpace(strings.SplitN(strategy.GetLabel(), "/", 2)[0])
	}
	return "(unknown)"
}

func experimentShortID(exp *experimentv1.Experiment) string {
	if exp == nil {
		return "(nil)"
	}
	if name := strings.TrimSpace(exp.GetName()); name != "" {
		return truncate(name, 18)
	}
	return truncate(exp.GetId(), 18)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func printExperimentErrors(ctx cliapp.RunContext, experiments []*experimentv1.Experiment) {
	var lines []string
	for _, exp := range experiments {
		if exp == nil {
			continue
		}
		if errMsg := strings.TrimSpace(exp.GetError()); errMsg != "" {
			lines = append(lines, fmt.Sprintf("  - %s: %s", exp.GetId(), errMsg))
		}
	}
	if len(lines) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nExperiment errors:")
	for _, line := range lines {
		fmt.Fprintln(ctx.Stdout(), line)
	}
}

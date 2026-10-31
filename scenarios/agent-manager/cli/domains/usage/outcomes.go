package usage

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

var windowFlag = cliapp.Flag{Name: "window", Description: "last_7d (default), last_30d, this_week, this_month, last_month, this_quarter"}

func windowToken(op cliapp.OperationContext) (*sharedmeasurepb.TimeWindow, error) {
	token := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(op.Flag("window")), "-", "_"))
	if token == "" {
		return nil, nil
	}
	value, ok := sharedmeasurepb.TimeWindowToken_value["TIME_WINDOW_TOKEN_"+token]
	if !ok {
		return nil, fmt.Errorf("invalid --window %q; use last_7d, last_30d, this_week, this_month, last_month or this_quarter", op.Flag("window"))
	}
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: sharedmeasurepb.TimeWindowToken(value)}}, nil
}

func parityCommand(newClient clientFor) cliapp.Command {
	return usageCommand("parity", "Does every session get the same measures, however it started? Coverage by start path and harness [--window TOKEN] [--json]",
		"One row per start path and harness: counts and shares for usage, tool facts, known tool outcomes, operator message signals, model, usage mode and outcome estimate, each with a reason when below 90%. A run with no commands counts as no commands, never as a gap. The window is by transcript activity, so old sessions imported this week stay out. Attached rows that duplicate a transcript or were helper processes stay out of denominators and are counted. Gaps lists every known difference between paths and why it remains.",
		[]cliapp.Flag{windowFlag},
		func(op cliapp.OperationContext) (proto.Message, error) {
			window, err := windowToken(op)
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageParity(context.Background(), connect.NewRequest(&measurepb.UsageParityRequest{Window: window}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageParityResponse)
			report := cliapp.ListReport{ResultsHeading: "Coverage by start path", Summary: []string{
				fmt.Sprintf("%s → %s · %d old sessions imported in this window are excluded · ledger behind by %d runs", stamp(r.GetWindowStart()), stamp(r.GetWindowEnd()), r.GetBackfillRuns(), r.GetLedgerPending()),
			}}
			for _, row := range r.GetRows() {
				report.Results = append(report.Results, parityRowLines(row)...)
			}
			report.Results = append(report.Results, "Gaps:")
			for _, gap := range r.GetGaps() {
				report.Results = append(report.Results, fmt.Sprintf("  [%s] %s: %d. %s", gap.GetStatus(), gap.GetSubject(), gap.GetCount(), gap.GetReason()))
			}
			return report
		})
}

func parityRowLines(row *measurepb.UsageParityRow) []string {
	head := fmt.Sprintf("%s / %s: %d runs, %d in denominators", row.GetPath(), row.GetHarness(), row.GetRuns(), row.GetEligibleRuns())
	if row.GetNoRequests() > 0 {
		head += fmt.Sprintf(", %d with no requests", row.GetNoRequests())
	}
	for _, excluded := range row.GetExcluded() {
		head += fmt.Sprintf(", %d %s", excluded.GetCount(), excluded.GetName())
	}
	lines := []string{head}
	for _, m := range row.GetMeasures() {
		if m.GetDenominator() == 0 && m.GetReason() == "" {
			continue
		}
		line := fmt.Sprintf("  %-16s %d/%d (%.1f%%)", m.GetName(), m.GetNumerator(), m.GetDenominator(), m.GetShare()*100)
		if m.GetNotApplicable() > 0 {
			line += fmt.Sprintf(", %d not applicable", m.GetNotApplicable())
		}
		if m.GetReason() != "" {
			line += ": " + m.GetReason()
		}
		lines = append(lines, line)
	}
	return lines
}

func outcomesCommand(newClient clientFor) cliapp.Command {
	flags := []cliapp.Flag{windowFlag,
		{Name: "by", Description: "model (default), role or mode"},
		{Name: "baseline", Description: "Row key to compare with; default the priciest row with enough data"},
	}
	return usageCommand("outcomes", "What does an accepted outcome cost? Estimates by model, role or mode, with ranges [--by model|role|mode] [--window TOKEN] [--json]",
		"Estimates per work unit (linked retries and sub-agents together) whether its outcome was accepted (independent evidence: an operator approval, a passing validation or the agent's own complete verdict), completed (a headless run finished and nothing checked it; never counted as accepted), corrected, failed, abandoned or unknown, each with its evidence source. Rows are split by path (headless or operator) and never pooled. Points per accepted outcome count failed attempts and rework; unknown units are excluded and reported. Each row also gives consumption per run, active time, run length, rework, unknown rate and break-even against a baseline of the same runner, with medians and 90% bootstrap ranges. Rows below 5 accepted outcomes read not_enough_data. These are observational estimates, not proof. Line counts are never outcomes. Definitions: docs/reference/operational-stats.md.",
		flags,
		func(op cliapp.OperationContext) (proto.Message, error) {
			window, err := windowToken(op)
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{Window: window, By: strings.TrimSpace(op.Flag("by")), Baseline: strings.TrimSpace(op.Flag("baseline"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageOutcomesResponse)
			report := cliapp.ListReport{ResultsHeading: "Outcome estimates by " + r.GetBy(), Summary: []string{
				fmt.Sprintf("%s → %s · %d work units, %d unknown (%.0f%%) · %d linked units, mean link confidence %.2f", stamp(r.GetWindowStart()), stamp(r.GetWindowEnd()), r.GetUnits(), r.GetUnknownUnits(), shareOf(r.GetUnknownUnits(), r.GetUnits())*100, r.GetLinkedUnits(), r.GetLinkConfidence()),
				"Evidence: " + evidenceText(r.GetEvidence()),
				"Paths: " + evidenceText(r.GetPaths()),
			}}
			for _, row := range r.GetRows() {
				report.Results = append(report.Results, outcomeRowLines(row)...)
			}
			for _, note := range r.GetNotes() {
				report.Results = append(report.Results, "Note: "+note)
			}
			if len(r.GetRefit()) > 0 {
				report.Results = append(report.Results, "Weights from blocks one model dominated:")
				for _, row := range r.GetRefit() {
					line := fmt.Sprintf("  %s/%s: %s. %s", row.GetRunner(), row.GetModel(), row.GetStatus(), row.GetReason())
					if implied := row.GetImpliedPointsPerMillion(); implied.GetAvailable() {
						line += fmt.Sprintf(" Implied %.2f pts/M (%.2f to %.2f) vs attributed %.2f.", implied.GetValue(), implied.GetLow(), implied.GetHigh(), row.GetPriorPointsPerMillion())
					}
					report.Results = append(report.Results, line)
				}
			}
			return report
		})
}

// pathText names the path a row belongs to, so a model on both paths prints two
// distinguishable rows.
func pathText(stratum string) string {
	if stratum == "" {
		return ""
	}
	return " [" + stratum + " path]"
}

func shareOf(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func evidenceText(rows []*measurepb.UsageEvidenceCount) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, fmt.Sprintf("%s %d", row.GetSource(), row.GetUnits()))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func rangeText(r *measurepb.UsageRange, format string) string {
	if !r.GetAvailable() {
		return "not enough data"
	}
	return fmt.Sprintf(format+" ("+format+" to "+format+")", r.GetValue(), r.GetLow(), r.GetHigh())
}

func outcomeRowLines(row *measurepb.UsageOutcomeRow) []string {
	verdict := row.GetVerdict()
	if row.GetStatusReason() != "" {
		verdict += ": " + row.GetStatusReason()
	}
	lines := []string{
		fmt.Sprintf("%s / %s%s: %s", row.GetRunner(), row.GetKey(), pathText(row.GetStratum()), verdict),
		fmt.Sprintf("  units %d: accepted %d, completed %d (finished, nothing checked it), corrected %d, failed %d, abandoned %d, unknown %d (%.0f%%) · rework %s · ≈%.2f pts", row.GetUnits(), row.GetAccepted(), row.GetCompleted(), row.GetCorrected(), row.GetFailed(), row.GetAbandoned(), row.GetUnknown(), row.GetUnknownRate()*100, rangeText(row.GetReworkRate(), "%.2f"), row.GetPointsTotal()),
		fmt.Sprintf("  pts per accepted %s · consumption per accepted %s", rangeText(row.GetPointsPerAccepted(), "%.2f"), rangeText(row.GetConsumptionPerAccepted(), "%.0f")),
		fmt.Sprintf("  per run (%d, median): pts %s · consumption %s · active min %s", row.GetRuns(), rangeText(row.GetPointsPerRun(), "%.3f"), rangeText(row.GetConsumptionPerRun(), "%.0f"), rangeText(row.GetActiveMinutesPerRun(), "%.1f")),
		fmt.Sprintf("  wall hours mean %.2f, median %.2f, %d runs over 24 h, idle %.0f%%", row.GetWallHoursMean(), row.GetWallHoursMedian(), row.GetRunsOver_24H(), row.GetIdleShare()*100),
		"  evidence: " + evidenceText(row.GetEvidence()),
	}
	if row.GetCompleted() > 0 {
		lines = append(lines, fmt.Sprintf("  weaker unit, never a verdict: pts per completed %s (completed units are counted as outcomes here only)", rangeText(row.GetPointsPerCompleted(), "%.2f")))
	}
	if row.GetBaseline() != "" && row.GetBaseline() != row.GetKey() {
		lines = append(lines, fmt.Sprintf("  vs %s: cheaper by %s · consumption %s · run length %s · break-even %.2fx (%s)", row.GetBaseline(), rangeText(row.GetCheaperBy(), "%.2fx"), rangeText(row.GetConsumptionRatio(), "%.2fx"), rangeText(row.GetRunLengthRatio(), "%.2fx"), row.GetBreakEvenRatio(), row.GetPriceSource()))
	}
	return lines
}

package usage

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	"agent-manager/cli/domains/measures"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

var (
	providerFlag = cliapp.Flag{Name: "provider", Description: "Provider filter (e.g. openai, anthropic)"}
	limitFlag    = cliapp.Flag{Name: "limit", Description: "Maximum rows (default 20)"}
	runFlag      = cliapp.Flag{Name: "run", Description: "Only rows for this run id (the run usage panel)"}
)

func parseLimit(op cliapp.OperationContext) (int32, error) {
	raw := strings.TrimSpace(op.Flag("limit"))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("--limit must be a positive integer no larger than %d", math.MaxInt32)
	}
	return int32(value), nil
}

// usageCommand is one read of a Measures usage procedure, rendered as text or
// emitted as the wire shape with --json.
func usageCommand(name, description, help string, flags []cliapp.Flag, call func(cliapp.OperationContext) (proto.Message, error), render func(proto.Message) cliapp.ListReport) cliapp.Command {
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: description, Usage: "agent-manager usage " + name + " [flags] [--json]", HelpText: help, Args: cliapp.ArgSchema{Flags: flags}}
	return cmd.WithPrimitive(cliapp.ProtoList(call, func(_ cliapp.OperationContext, response proto.Message) cliapp.ListReport { return render(response) }))
}

func burnersCommand(newClient clientFor) cliapp.Command {
	return usageCommand("burners", "What is using each subscription window? [--by session|run|profile|workload|model|runner|modality|tool_kind] [--json]",
		"Splits each current window's observed used-percent increase across the usage that ran in it, recalibrating tokens to points per window, with an explicit unattributed (off-disk) remainder. Points are estimates.",
		[]cliapp.Flag{providerFlag, {Name: "by", Description: "Group by dimension (default session)"}, {Name: "since", Description: "RFC 3339: attribute only the increase after it"}, runFlag, limitFlag},
		func(op cliapp.OperationContext) (proto.Message, error) {
			limit, err := parseLimit(op)
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{
				Provider: strings.TrimSpace(op.Flag("provider")), GroupBy: strings.TrimSpace(op.Flag("by")), Since: strings.TrimSpace(op.Flag("since")), RunId: strings.TrimSpace(op.Flag("run")), Limit: limit,
			}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageBurnersResponse)
			report := cliapp.ListReport{ResultsHeading: "Windows", Summary: []string{fmt.Sprintf("%d window(s) as of %s; points are estimates", len(r.GetWindows()), stamp(r.GetGeneratedAt()))}}
			if len(r.GetWindows()) == 0 {
				report.Summary = append(report.Summary, "No current provider window has used-percent readings.")
			}
			scopes := make([]windowScope, len(r.GetWindows()))
			for i, w := range r.GetWindows() {
				scopes[i] = windowScope{w.GetProvider(), w.GetPool(), w.GetWindow(), w.GetWindowMinutes()}
			}
			titles := windowTitles(scopes)
			for i, w := range r.GetWindows() {
				report.Results = append(report.Results, attributionLines(w, titles[i])...)
			}
			report.RetrievalHints = []string{"agent-manager usage burners --by model  # or profile, workload, runner, modality, tool_kind", "agent-manager usage recommendations  # what to change"}
			return report
		})
}

func attributionLines(w *measurepb.UsageWindowAttribution, title string) []string {
	lines := []string{fmt.Sprintf("%s  %s → %s  observed +%.0f pts (now %.0f%%)", title, stamp(w.GetPeriodStart()), stamp(w.GetPeriodEnd()), w.GetObservedIncreasePoints(), w.GetUsedPercent())}
	for _, b := range w.GetBurners() {
		detail := []string{}
		for _, value := range []string{b.GetModel(), b.GetRunner()} {
			if value != "" {
				detail = append(detail, value)
			}
		}
		if b.GetRuns() > 1 {
			detail = append(detail, fmt.Sprintf("%d runs", b.GetRuns()))
		}
		detail = append(detail, fmt.Sprintf("%s tokens (%s cached)", compact(b.GetTokens().GetTotal()), compact(b.GetTokens().GetCachedInput())))
		lines = append(lines, fmt.Sprintf("  ≈%5.1f pts %4.0f%%  %s  [%s]", b.GetPoints(), b.GetShare()*100, truncateLabel(b.GetLabel(), 70), strings.Join(detail, ", ")))
	}
	if w.GetOtherBurners() > 0 {
		lines = append(lines, fmt.Sprintf("  ≈%5.1f pts        %d smaller items", w.GetOtherPoints(), w.GetOtherBurners()))
	}
	lines = append(lines, fmt.Sprintf("  ≈%5.1f pts %4.0f%%  unattributed / off-disk", w.GetUnattributedPoints(), share(w.GetUnattributedPoints(), w.GetObservedIncreasePoints())*100))
	c := w.GetCalibration()
	calibration := fmt.Sprintf("  calibration %s over %d block(s)", c.GetMethod(), c.GetSlices())
	if c.GetMethod() == "fitted" {
		calibration += fmt.Sprintf(", R² %.2f", c.GetRSquared())
	}
	for _, weight := range c.GetWeights() {
		if weight.GetCachedInputTokensPerPoint() > 0 {
			calibration += fmt.Sprintf("; %s %s cached/pt", weight.GetModel(), compact(int64(weight.GetCachedInputTokensPerPoint())))
		}
	}
	if c.GetNote() != "" {
		calibration += " — " + c.GetNote()
	}
	lines = append(lines, calibration)
	if pending := w.GetPendingTokens().GetTotal(); pending > 0 {
		lines = append(lines, fmt.Sprintf("  pending: %s tokens after the newest reading", compact(pending)))
	}
	for _, m := range w.GetMeteredLines() {
		line := fmt.Sprintf("  metered outside tokens: %s ×%d (%s)", m.GetKind(), m.GetCount(), m.GetState())
		if m.GetAmount() != "" {
			line += " " + m.GetAmount()
		}
		lines = append(lines, line+" — "+m.GetNote())
	}
	return lines
}

func recommendationsCommand(newClient clientFor) cliapp.Command {
	return usageCommand("recommendations", "What should I change? One ranked list of owner-level actions [--provider NAME] [--pool P --window W] [--run ID] [--json]",
		"Merges the quota levers (R-model, R-model-mix, R-context, R-lifetime, R-floor, R-idle, R-image, R-skill), tool friction findings, opportunities (repeated sequences, bypasses) and operator interventions the accuracy gate admits into one list. Each action names its owner (skill, scenario, command, program, harness setting, session or workload), the change, its effect in its own unit (points, calls, tokens, interventions; finished sessions 'would have saved', only running ones are 'projected'), a score on its named scale with the absolute cost it is a share of (quota scale for levers: share of the window's points; friction scale for the rest: the geometric mean of harm share and run breadth, or operator-message share for interventions), evidence and the measure that verifies it. Quota levers are listed first, then friction actions: the two scales are never interleaved. Harm kinds weigh equally. Actions under 1% of their kind are counted; a lever family that did not fire says why; coverage says what each source could see. --pool and --window evaluate one window's levers alone.",
		[]cliapp.Flag{providerFlag, {Name: "pool", Description: "With --provider and --window: evaluate this window alone"}, {Name: "window", Description: "With --provider and --pool: evaluate this window alone"}, runFlag, limitFlag},
		func(op cliapp.OperationContext) (proto.Message, error) {
			limit, err := parseLimit(op)
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{
				Provider: strings.TrimSpace(op.Flag("provider")), RunId: strings.TrimSpace(op.Flag("run")), Limit: limit, Pool: strings.TrimSpace(op.Flag("pool")), Window: strings.TrimSpace(op.Flag("window")),
			}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageRecommendationsResponse)
			summary := fmt.Sprintf("%d action(s) as of %s: quota levers ranked by their share of a window's points, then friction actions ranked by √(harm share × runs touched); the scales are not comparable", len(r.GetActions()), stamp(r.GetGeneratedAt()))
			if r.GetBelowMateriality() > 0 {
				summary += fmt.Sprintf("; %d more scoring under %.0f%%", r.GetBelowMateriality(), 100*r.GetMaterialityScore())
			}
			report := cliapp.ListReport{ResultsHeading: "Actions", Summary: []string{summary}}
			group := ""
			for _, action := range r.GetActions() {
				if heading := actionGroup(action); heading != group {
					group = heading
					report.Results = append(report.Results, heading)
				}
				report.Results = append(report.Results, actionLines(action)...)
			}
			for _, rule := range r.GetRules() {
				if !rule.GetFired() {
					report.Results = append(report.Results, fmt.Sprintf("   %s did not fire: %s", rule.GetRule(), rule.GetReason()))
				}
			}
			for _, line := range r.GetCoverage() {
				report.Results = append(report.Results, "   coverage: "+line)
			}
			for _, line := range r.GetScoring() {
				report.Results = append(report.Results, "   "+line)
			}
			report.RetrievalHints = []string{"agent-manager usage recommendations --json  # evidence ids and drill-down per action", "agent-manager usage friction  # report or dismiss a tool finding by fingerprint", "agent-manager usage opportunities  # every repeated sequence and bypass"}
			return report
		})
}

// actionLines renders one action: rank, title, score and confidence; owner;
// the change; its effect in its own unit; evidence and verification; the
// largest drill-down sessions.
func actionLines(a *measurepb.UsageAction) []string {
	lines := []string{
		fmt.Sprintf("%d. %s  %s  [%s %s, %s]", a.GetRank(), a.GetTitle(), scoreText(a), a.GetSource(), a.GetKind(), a.GetConfidence()),
		fmt.Sprintf("   owner: %s %s", a.GetOwnerKind(), truncateLabel(a.GetOwner(), 90)),
		"   " + a.GetAction(),
		"   effect: " + effectText(a.GetEffect()) + " · " + a.GetRationale(),
		"   score: " + a.GetScoreBasis(),
	}
	lines = append(lines, frictionDetailLines(a.GetRootCause(), a.GetSubs(), a.GetGuesses(), 0, 0)...)
	if a.GetRecoverablePoints() > 0 {
		lines = append(lines, fmt.Sprintf("   ≈%.1f pts of it still recoverable before the window resets", a.GetRecoverablePoints()))
	}
	for _, d := range a.GetDrilldown()[:min(3, len(a.GetDrilldown()))] {
		lines = append(lines, fmt.Sprintf("   · %s: %s", truncateLabel(d.GetLabel(), 60), d.GetDetail()))
	}
	return append(lines, "   "+evidenceSummary(a.GetEvidenceRefs())+" · verify: "+a.GetVerify())
}

// actionGroup is the heading of the group the API ranks an action in: quota
// levers, friction actions, then a live session's own advice.
func actionGroup(a *measurepb.UsageAction) string {
	switch {
	case a.GetOwnerKind() == "session":
		return "Live sessions (quota scale)"
	case a.GetSource() == "lever":
		return "Quota levers (quota scale: share of a window's observed points)"
	}
	return "Friction actions (friction scale: √(harm share × runs touched))"
}

func scoreText(a *measurepb.UsageAction) string {
	if a.GetEffect().GetBasis() == "range" {
		return "unproven range"
	}
	return fmt.Sprintf("%.1f%%", 100*a.GetScore())
}

func effectText(e *measurepb.UsageActionEffect) string {
	switch e.GetUnit() {
	case "points":
		if e.GetBasis() == "range" {
			return fmt.Sprintf("0 to %.1f pts (unproven range)", e.GetUpper())
		}
		return fmt.Sprintf("%.1f pts (%s)", e.GetValue(), e.GetBasis())
	case "ms":
		return fmt.Sprintf("%.0f min (%s)", e.GetValue()/60000, e.GetBasis())
	case "tokens":
		return fmt.Sprintf("%s tokens (%s)", compact(int64(e.GetValue())), e.GetBasis())
	}
	return fmt.Sprintf("%.0f %s (%s)", e.GetValue(), e.GetUnit(), e.GetBasis())
}

func evidenceSummary(refs []*measurepb.UsageEvidenceRef) string {
	counts := map[string]int{}
	order := []string{}
	for _, ref := range refs {
		if counts[ref.GetKind()] == 0 {
			order = append(order, ref.GetKind())
		}
		counts[ref.GetKind()]++
	}
	parts := make([]string, 0, len(order))
	for _, kind := range order {
		parts = append(parts, fmt.Sprintf("%d %s", counts[kind], kind))
	}
	if len(parts) == 0 {
		return "evidence: drill-down"
	}
	return "evidence: " + strings.Join(parts, ", ")
}

func coverageCommand(newClient clientFor) cliapp.Command {
	return usageCommand("coverage", "Can I trust this? Transcript capture coverage against the disk [--since RFC3339] [--json]",
		"Walks transcript roots and compares files modified since --since (default 7 days) with index checkpoints, listing up to 50 issues per root.",
		[]cliapp.Flag{{Name: "since", Description: "RFC 3339 (default 7 days ago)"}},
		func(op cliapp.OperationContext) (proto.Message, error) {
			response, err := newClient(op.Core()).UsageCoverage(context.Background(), connect.NewRequest(&measurepb.UsageCoverageRequest{Since: strings.TrimSpace(op.Flag("since"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageCoverageResponse)
			t := r.GetTotals()
			report := cliapp.ListReport{ResultsHeading: "Roots", Summary: []string{fmt.Sprintf("captured %s of transcript bytes since %s · %d files · %d pending · %d unreadable", coverageShare(t), stamp(r.GetSince()), t.GetFiles(), t.GetPending(), t.GetUnreadable())}}
			for _, root := range r.GetRoots() {
				report.Results = append(report.Results, fmt.Sprintf("%-24s %-11s %6s  files %d (indexed %d, owned %d, excluded %d, pending %d, unreadable %d)  turns %d, usage reports %d",
					root.GetKind(), root.GetHarness(), coverageShare(root), root.GetFiles(), root.GetIndexed(), root.GetOwned(), root.GetExcluded(), root.GetPending(), root.GetUnreadable(), root.GetTurns(), root.GetUsageReports()))
				for _, issue := range root.GetIssues() {
					report.Results = append(report.Results, "    "+issue.GetReason()+": "+issue.GetPath())
				}
			}
			report.RetrievalHints = []string{"Method: " + r.GetMethod()}
			return report
		})
}

func coverageShare(root *measurepb.UsageCoverageRoot) string {
	if root.GetBytes() <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", root.GetCapturedShare()*100)
}

func compareCommand(newClient clientFor) cliapp.Command {
	flags := append(measures.SelectorFlags("left-"), measures.SelectorFlags("right-")...)
	flags = append(flags, providerFlag, cliapp.Flag{Name: "change-binding", Description: "Label of the change between the cohorts (echoed)"})
	return usageCommand("compare", "Did the change work? Quota per accepted work unit between two cohorts [--left-* --right-*] [--json]",
		"Prices each cohort's usage as its attributed share of the observed window increase and compares points per accepted work unit. Accepted is an outcome estimate with a stated source; a completed run, or an imported session whose transcript ended, is not accepted, and unknown outcomes stay out of the ratio. Select each side with --left-/--right- cohort, run-ids, filter-json and window, the same selector as `measures`.",
		flags,
		func(op cliapp.OperationContext) (proto.Message, error) {
			left, err := measures.ReadSelector(op, "left-")
			if err != nil {
				return nil, err
			}
			right, err := measures.ReadSelector(op, "right-")
			if err != nil {
				return nil, err
			}
			response, err := newClient(op.Core()).UsageCompare(context.Background(), connect.NewRequest(&measurepb.UsageCompareRequest{Left: left, Right: right, Provider: strings.TrimSpace(op.Flag("provider")), ChangeBinding: strings.TrimSpace(op.Flag("change-binding"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(response proto.Message) cliapp.ListReport {
			r := response.(*measurepb.UsageCompareResponse)
			report := cliapp.ListReport{ResultsHeading: "Cohorts", Summary: []string{"Verdict: " + r.GetVerdict()}}
			if r.GetVerdict() != "insufficient" {
				report.Summary[0] += fmt.Sprintf(" (points per accepted unit %+.0f%%)", r.GetPointsPerAcceptedChange()*100)
			}
			for _, side := range []struct {
				name string
				s    *measurepb.UsageCompareSide
			}{{"left", r.GetLeft()}, {"right", r.GetRight()}} {
				report.Results = append(report.Results, fmt.Sprintf("%-5s %d runs (%d completed), %d accepted units, %d unknown, %d priced · ≈%.2f pts · ≈%.3f pts per accepted · %s tokens per accepted · %s → %s",
					side.name, side.s.GetRuns(), side.s.GetCompletedRuns(), side.s.GetAcceptedUnits(), side.s.GetUnknownUnits(), side.s.GetPricedRuns(), side.s.GetPoints(), side.s.GetPointsPerAccepted(), compact(int64(side.s.GetTokensPerAccepted())), stamp(side.s.GetWindowStart()), stamp(side.s.GetWindowEnd())))
			}
			report.Results = append(report.Results, r.GetCaveats()...)
			return report
		})
}

func share(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return part / whole
}

func compact(value int64) string {
	switch {
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(value)/1e9)
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1e6)
	case value >= 1_000:
		return fmt.Sprintf("%.0fk", float64(value)/1e3)
	}
	return strconv.FormatInt(value, 10)
}

func truncateLabel(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) <= limit {
		return value
	}
	return string([]rune(value)[:limit-1]) + "…"
}

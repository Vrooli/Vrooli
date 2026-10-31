// This file holds `usage experiment plan|start|status|end|propose`: Learn
// comparisons that send a role's new managed runs at random to the model it
// runs on today or one lower model (docs/configuration/usage-modes.md, "Learn
// mode"). The API owns assignment, analysis, the verdict and authority; this
// command only carries the request and renders the answer, including the
// expected time to a verdict before anything starts.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"agent-manager/cli/internal/support"

	"connectrpc.com/connect"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const usageExperimentUsage = `agent-manager usage experiment plan --runner claude|codex --role ROLE [--split 50] [--json]
  agent-manager usage experiment start --runner claude|codex --role ROLE --until <duration like 90d>|off [--model X] [--split 50] [--sessions [--block-days N]] [--label text] --confirm [--json]
  agent-manager usage experiment status [<id>] [--json]
  agent-manager usage experiment end <id> [--json]
  agent-manager usage experiment propose <id> [--json]`

const usageExperimentHelp = `A Learn comparison tests whether a lower model earns its keep for one role. New managed runs of the role are assigned at random, by the run id, to the incumbent or the challenger; the arm is recorded in the run's policy snapshot (canary_arm). Analysis is by assigned arm (intention to treat) with a permutation test and 90% ranges, and ends in one of four verdicts: cheaper looks better, about the same, higher looks better, not enough data.

plan      shows the expected time to a verdict from the role's own recent pace and variance, with a range. It starts nothing.
start     shows that plan, and starts the comparison only with --confirm. Verified operator only; an agent run is refused with 403. Exempt roles (code.fallback, code.review, judgment.supervision, code.supervision and the resource's own) are never randomized. --sessions also opts operator sessions in as randomized day blocks (labelled at launch; one full-power launch is allowed and recorded as an exception that stays in its assigned arm).
status    lists every comparison with its design (Randomized), arms, verdict or the data still needed, and the role recommendation. Agents may read it.
end       ends one comparison. Verified operator only.
propose   raises a decision that asks a verified human to approve making the challenger the role's default. It applies nothing: Agent Manager does not edit resource files, so the exact policy change is printed for the operator to apply.

Learn pauses while a quota window is forecast to run out, and a Save mode that applies takes priority. At about 200 managed runs a month only large differences are detectable; read the plan before starting.`

func (a *App) usageExperimentCommand() cliapp.Command {
	command := support.Command("experiment", "Compare a lower model against a role's default at random: plan|start|status|end|propose [--json]", a.usageExperiment)
	command.Usage = usageExperimentUsage
	command.HelpText = usageExperimentHelp
	return command
}

func (a *App) usageExperiment(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("agent-manager usage experiment " + strings.TrimPrefix(usageExperimentUsage, "agent-manager usage experiment "))
		fmt.Println()
		fmt.Println(usageExperimentHelp)
		return nil
	}
	switch args[0] {
	case "plan":
		return a.usageExperimentPlan(args[1:])
	case "start":
		return a.usageExperimentStart(args[1:])
	case "status", "list":
		return a.usageExperimentStatus(args[1:])
	case "end":
		return a.usageExperimentEnd(args[1:])
	case "propose":
		return a.usageExperimentPropose(args[1:])
	}
	return fmt.Errorf("unknown usage experiment subcommand %q; expected plan, start, status, end or propose", args[0])
}

// humanDays renders a duration in days the way an operator plans: days, weeks
// or months, and "over N years" past two.
func humanDays(days float64) string {
	switch {
	case days <= 0 || math.IsNaN(days):
		return "not reached"
	case days < 14:
		return fmt.Sprintf("%.0f days", math.Ceil(days))
	case days < 90:
		return fmt.Sprintf("%.0f weeks", math.Ceil(days/7))
	case days < 730:
		return fmt.Sprintf("%.0f months", math.Ceil(days/30))
	}
	return fmt.Sprintf("over %.0f years", math.Floor(days/365))
}

// approxDays is humanDays with "about" in front, except where it already says "over".
func approxDays(days float64) string {
	text := humanDays(days)
	if strings.HasPrefix(text, "over ") || text == "not reached" {
		return text
	}
	return "about " + text
}

func horizonText(h *measurepb.UsageExperimentHorizon) string {
	if h == nil || !h.GetReached() {
		return "not reached within the simulated range (more than 700 units per arm)"
	}
	text := fmt.Sprintf("%s (%d units per arm)", approxDays(h.GetDays()), h.GetPerArm())
	if h.GetHighDays() > 0 {
		text += fmt.Sprintf("; range %s to %s", humanDays(h.GetLowDays()), humanDays(h.GetHighDays()))
	} else {
		text += fmt.Sprintf("; at least %s, upper end beyond the simulated range", strings.TrimPrefix(humanDays(h.GetLowDays()), "over "))
	}
	return text
}

func printPlan(plan *measurepb.UsageExperimentPlan, indent string) {
	if plan == nil {
		fmt.Println(indent + "expected time to a verdict: unavailable")
		return
	}
	variance := fmt.Sprintf("measured from %d finished units", plan.GetPilotUnits())
	if plan.GetVarianceAssumed() {
		variance = fmt.Sprintf("assumed (only %d finished units, too few to measure)", plan.GetPilotUnits())
	}
	fmt.Printf("%sRole pace: %.2f finished units a day over the last 30 days; cost spread (CV) %.2f, %s.\n", indent, plan.GetUnitsPerDay(), plan.GetCostCv(), variance)
	fmt.Printf("%sA verdict needs %d accepted outcomes in each arm, an effect of at least %.0f%% on points per accepted outcome, and rework up no more than %.0f points.\n",
		indent, plan.GetMinAcceptedPerArm(), plan.GetMinEffect()*100, plan.GetReworkMargin()*100)
	fmt.Printf("%sTo detect a %.0f%% cost reduction with %.0f%% power: %s.\n", indent, plan.GetEffect()*100, plan.GetPower()*100, horizonText(plan.GetDetect()))
	fmt.Printf("%sTo show \"about the same\" (within %.0f%%) if there is no difference: %s.\n", indent, plan.GetMinEffect()*100, horizonText(plan.GetSame()))
	if plan.GetDetectableIn_90Days() > 0 {
		fmt.Printf("%sIn 90 days only a reduction of %.0f%% or more could be detected.\n", indent, plan.GetDetectableIn_90Days()*100)
	} else {
		fmt.Printf("%sIn 90 days no reduction up to 90%% could be detected at this pace.\n", indent)
	}
	for _, note := range plan.GetNotes() {
		fmt.Println(indent + "Note: " + note)
	}
}

func (a *App) usageExperimentPlan(args []string) error {
	fs := flag.NewFlagSet("usage experiment plan", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	runner := fs.String("runner", "", "claude or codex")
	role := fs.String("role", "", "the role to compare")
	split := fs.Int("split", 0, "challenger share, 1 to 99 (default 50)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*runner) == "" || strings.TrimSpace(*role) == "" {
		return errors.New("usage: agent-manager usage experiment plan --runner claude|codex --role ROLE [--split 50] [--json]")
	}
	response, err := a.modeClient(false).PlanUsageExperiment(context.Background(), connect.NewRequest(&measurepb.PlanUsageExperimentRequest{
		Runner: strings.TrimSpace(*runner), Role: strings.TrimSpace(*role), SplitPercent: int32(*split),
	}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	if response.Msg.GetRefusal() != "" {
		fmt.Println("Not comparable: " + response.Msg.GetRefusal())
		return nil
	}
	printPlan(response.Msg.GetPlan(), "")
	return nil
}

func (a *App) usageExperimentStart(args []string) error {
	fs := flag.NewFlagSet("usage experiment start", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	runner := fs.String("runner", "", "claude or codex")
	role := fs.String("role", "", "the role to compare")
	profile := fs.String("profile", "", "profile key to narrow to")
	model := fs.String("model", "", "challenger model (default: the resource's lower tier)")
	until := fs.String("until", "", "<duration like 90d>|off|<RFC3339>")
	label := fs.String("label", "", "operator label")
	split := fs.Int("split", 0, "challenger share, 1 to 99 (default 50)")
	sessions := fs.Bool("sessions", false, "also opt operator sessions in as randomized day blocks")
	blockDays := fs.Int("block-days", 0, "days per session block (default 1)")
	confirm := fs.Bool("confirm", false, "start after reading the expected time to a verdict")
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*runner) == "" || strings.TrimSpace(*role) == "" || strings.TrimSpace(*until) == "" {
		return errors.New("usage: agent-manager usage experiment start --runner claude|codex --role ROLE --until <duration like 90d>|off [--model X] [--split 50] [--sessions] --confirm [--json]")
	}
	ctx := context.Background()
	plan, err := a.modeClient(false).PlanUsageExperiment(ctx, connect.NewRequest(&measurepb.PlanUsageExperimentRequest{
		Runner: strings.TrimSpace(*runner), Role: strings.TrimSpace(*role), SplitPercent: int32(*split),
	}))
	if err != nil {
		return modeRPCError(err)
	}
	if plan.Msg.GetRefusal() != "" {
		return errors.New("not started: " + plan.Msg.GetRefusal())
	}
	if !*confirm {
		if *jsonOut {
			return printProtoJSON(plan.Msg)
		}
		fmt.Println("Expected time to a verdict (nothing started yet):")
		printPlan(plan.Msg.GetPlan(), "  ")
		fmt.Println("Re-run with --confirm to start the comparison.")
		return nil
	}
	response, err := a.modeClient(true).StartUsageExperiment(ctx, connect.NewRequest(&measurepb.StartUsageExperimentRequest{
		Runner: strings.TrimSpace(*runner), Role: strings.TrimSpace(*role), Profile: strings.TrimSpace(*profile), Model: strings.TrimSpace(*model),
		Until: strings.TrimSpace(*until), Label: strings.TrimSpace(*label), SplitPercent: int32(*split), Sessions: *sessions, BlockDays: int32(*blockDays),
	}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	mode := response.Msg.GetMode()
	fmt.Printf("Started Learn comparison %s (started by %s) for %s on %s.\n", mode.GetId(), mode.GetStartedBy(), mode.GetScope().GetRole(), mode.GetScope().GetRunner())
	for _, s := range mode.GetLearn().GetChallenger() {
		for from, to := range s.GetModelMap() {
			fmt.Printf("  %d%% of new runs go to %s instead of %s, chosen at random.\n", mode.GetLearn().GetSplitPercent(), to, from)
		}
	}
	printPlan(response.Msg.GetPlan(), "  ")
	for _, note := range response.Msg.GetNotes() {
		fmt.Println("- " + note)
	}
	return nil
}

func (a *App) usageExperimentEnd(args []string) error {
	fs := flag.NewFlagSet("usage experiment end", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return errors.New("usage: agent-manager usage experiment end <id> [--json]")
	}
	response, err := a.modeClient(true).EndUsageMode(context.Background(), connect.NewRequest(&measurepb.EndUsageModeRequest{Ids: fs.Args()}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	if len(response.Msg.GetEndedIds()) == 0 {
		fmt.Println("No active comparison ended.")
		return nil
	}
	fmt.Printf("Ended comparison %s. Its arms stay recorded and the analysis stays readable with `usage experiment status`.\n", strings.Join(response.Msg.GetEndedIds(), ", "))
	return nil
}

var verdictWords = map[string]string{
	"cheaper_looks_better": "cheaper looks better",
	"about_the_same":       "about the same",
	"higher_looks_better":  "higher looks better",
	"not_enough_data":      "not enough data",
}

func (a *App) usageExperimentStatus(args []string) error {
	fs := flag.NewFlagSet("usage experiment status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id := ""
	if len(fs.Args()) > 0 {
		id = fs.Args()[0]
	}
	response, err := a.modeClient(false).UsageExperiment(context.Background(), connect.NewRequest(&measurepb.UsageExperimentRequest{ModeId: id}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	if len(response.Msg.GetExperiments()) == 0 {
		fmt.Println("No Learn comparison has run. Plan one with `agent-manager usage experiment plan --runner codex --role ROLE`.")
		return nil
	}
	for _, e := range response.Msg.GetExperiments() {
		printExperiment(e)
	}
	return nil
}

func printExperiment(e *measurepb.UsageExperiment) {
	state := "ended"
	switch {
	case e.GetPaused():
		state = "paused (" + e.GetPausedReason() + ")"
	case e.GetActive():
		state = "running"
	}
	fmt.Printf("\n%s  %s  [%s]\n", e.GetModeId(), valueOr(e.GetLabel(), e.GetRole()), state)
	fmt.Printf("  design: Randomized, %d%% to the challenger %s against %s, role %s on %s", e.GetSplitPercent(), valueOr(e.GetChallengerModel(), "?"), valueOr(e.GetIncumbentModel(), "?"), e.GetRole(), e.GetRunner())
	if e.GetSessionsOptIn() {
		fmt.Printf("; operator sessions included as %d-day blocks", max(int(e.GetBlockDays()), 1))
	}
	fmt.Println()
	for _, arm := range []*measurepb.UsageExperimentArm{e.GetIncumbent(), e.GetChallenger()} {
		fmt.Printf("  %-10s %d units, %d known, %d accepted, %d exceptions; points per accepted %s; rework %s\n", arm.GetArm(), arm.GetUnits(), arm.GetKnown(), arm.GetAccepted(), arm.GetExceptions(),
			rangeText(arm.GetPointsPerAccepted(), 1), rangeText(arm.GetReworkRate(), 100))
	}
	fmt.Printf("  verdict: %s\n", verdictWords[e.GetVerdict()])
	fmt.Printf("  %s\n", e.GetReason())
	if e.GetCostRatio().GetAvailable() {
		fmt.Printf("  cost ratio challenger/incumbent %s, permutation p=%.3f; rework difference %s points, p=%.3f\n", rangeText(e.GetCostRatio(), 1), e.GetCostPValue(), rangeText(e.GetReworkDifference(), 100), e.GetReworkPValue())
	}
	if e.GetNeedAccepted() > 0 {
		fmt.Printf("  still needed: %d more accepted outcomes in the thinner arm\n", e.GetNeedAccepted())
	}
	if e.GetVerdict() == "not_enough_data" {
		printPlan(e.GetPlan(), "  ")
	}
	r := e.GetRecommendation()
	if r.GetAvailable() {
		fmt.Printf("  recommendation: make %s the default for %s instead of %s. This is an ask; nothing is applied.\n", r.GetToModel(), r.GetRole(), r.GetFromModel())
		if len(r.GetPolicyOps()) > 0 {
			fmt.Printf("  exact policy change in %s (apply it yourself; Agent Manager does not edit resource files):\n", r.GetPolicyFile())
			for _, op := range r.GetPolicyOps() {
				fmt.Println("    " + op)
			}
		}
		if r.GetPolicyNote() != "" {
			fmt.Println("  note: " + r.GetPolicyNote())
		}
		if r.GetDecisionState() != "" {
			fmt.Printf("  decision %s: %s %s\n", r.GetDecisionId(), r.GetDecisionState(), r.GetDecisionNote())
		} else {
			fmt.Printf("  to ask for approval: agent-manager usage experiment propose %s\n", e.GetModeId())
		}
	}
}

func rangeText(r *measurepb.UsageRange, scale float64) string {
	if !r.GetAvailable() {
		return "unknown"
	}
	return fmt.Sprintf("%.2f (%.2f to %.2f)", r.GetValue()*scale, r.GetLow()*scale, r.GetHigh()*scale)
}

func (a *App) usageExperimentPropose(args []string) error {
	fs := flag.NewFlagSet("usage experiment propose", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return errors.New("usage: agent-manager usage experiment propose <id> [--json]")
	}
	response, err := a.modeClient(true).ProposeRoleDefault(context.Background(), connect.NewRequest(&measurepb.ProposeRoleDefaultRequest{ModeId: fs.Args()[0]}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	r := response.Msg.GetRecommendation()
	fmt.Printf("Raised decision %s: make %s the default for %s. A verified human must approve it; nothing is applied either way.\n", r.GetDecisionId(), r.GetToModel(), r.GetRole())
	if len(r.GetPolicyOps()) > 0 {
		fmt.Printf("Exact policy change in %s:\n", r.GetPolicyFile())
		for _, op := range r.GetPolicyOps() {
			fmt.Println("  " + op)
		}
	}
	if r.GetPolicyNote() != "" {
		fmt.Println("Note: " + r.GetPolicyNote())
	}
	return nil
}

// This file implements the generic epoch step-back check (large-effort-orchestration §4.4).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// exitStepBack is the distinct status the epoch check returns when a
// step-back trigger fires, so a worker or orchestrator script can branch on it.
const exitStepBack = 3

// exitRefused is the status of an orchestrator check that refuses: an unmet
// gate or unanswered diminishing returns under epoch-check --acceptance, a
// failing goal-home lint, a refused handoff or a refused park.
const exitRefused = 4

// epochCheckReport is the shared goal-home report plus what the CLI adds:
// goal-level findings, the acceptance verdict and the runs a step-back woke.
type epochCheckReport struct {
	*goalhome.EpochReport
	Findings          []goalhome.Finding     `json:"findings,omitempty"`
	AcceptanceBlocked []string               `json:"acceptance_blocked,omitempty"`
	Woken             []string               `json:"woken,omitempty"`
	Signal            *domainpb.EffortSignal `json:"signal,omitempty"`
}

// shouldWakeOnStepBack suppresses only a server-confirmed duplicate. Errors and
// older servers omit the optional result, so their existing wake behavior remains.
func shouldWakeOnStepBack(signal *domainpb.EffortSignal, recordErr error) bool {
	return recordErr != nil || signal == nil || signal.NewlyRecorded == nil || *signal.NewlyRecorded
}

// effortEpochCheck is `agent-manager effort epoch-check <epoch-file>`.
func (a *App) effortEpochCheck(args []string) error {
	fs := flag.NewFlagSet("effort epoch-check", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	k := fs.Int("k", 10, "Flatline: log lines over which an unchanged exit metric fires")
	overrun := fs.Float64("overrun", 0, "Optional observation: work units above this multiple of the estimate (0 = disabled; never stops work)")
	growth := fs.Int("growth", 0, "Optional observation: refactor net runtime lines above this value (0 = disabled; never stops work)")
	featureFactor := fs.Float64("feature-factor", 0, "Optional observation: feature net runtime lines above this multiple of expected size (0 = disabled; never stops work)")
	wallClock := fs.Duration("wall-clock", 0, "Optional elapsed-time observation (0 = disabled; never a stop or evidence of poor performance)")
	spendThreshold := fs.Float64("spend-threshold", 0, "Optimization only: worker weighted non-cache tokens above which review is prompted (0 = not evaluated; never stops work)")
	runs := fs.String("runs", "", "Comma-separated worker run IDs for the spend trigger (default: the brief's Workers field)")
	wakeKey := fs.String("wake-key", "", "On a fired step-back, wake the run parked on this key (the orchestrator's run ID)")
	effortRef := fs.String("effort", "", "On a fired step-back, record a step_back signal only; this does not dispatch supervision. To request a supervisor heartbeat, use `prompt-manager team heartbeat-trigger effort-supervision effort-supervisor` when authorized")
	acceptance := fs.Bool("acceptance", false, "Orchestrator acceptance check: exit 4 on an unmet typed gate or an unanswered diminishing-returns finding; never exits 3 or wakes")
	scenario := fs.String("scenario", "", "Scenario for receipt-backed journey gates (default: the brief's Scenario field)")
	var path string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		path, args = args[0], args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("usage: agent-manager effort epoch-check <epoch-file> [--acceptance] [--wake-key <run-id>] [--effort <ref>] [--runs ids] [--spend-threshold n] [--json]")
	}
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	file, err := goalhome.ParseEpoch(handle)
	handle.Close()
	if err != nil {
		return err
	}
	workerIDs := *runs
	if workerIDs == "" {
		workerIDs = file.Fields["workers"]
	}
	var weighted *float64
	var spendWarning string
	if ids := goalhome.WorkerRunIDs(workerIDs); *spendThreshold > 0 && len(ids) > 0 {
		spend, err := a.weightedTokens(ids, defaultTierWeights)
		if err != nil {
			spendWarning = fmt.Sprintf("worker spend unavailable: %v; continue work without a spend verdict", err)
		} else if !spend.Complete {
			spendWarning = "worker spend is unqualified for one or more runs; continue work without a spend verdict"
		} else {
			weighted = &spend.TotalWeighted
		}
	}
	limits := goalhome.Thresholds{FlatlineK: *k, OverrunFactor: *overrun, RefactorGrowth: *growth, FeatureFactor: *featureFactor, WallClock: *wallClock, SpendThreshold: *spendThreshold}
	report := &epochCheckReport{EpochReport: goalhome.EvaluateEpoch(path, file, limits, time.Now().UTC(), weighted)}
	if *scenario == "" {
		*scenario = file.Fields["scenario"]
	}
	read := a.journeyReceiptReader
	if read == nil {
		read = readWorkflowJourneyReceipt
	}
	goalhome.ResolveJourneyReceipts(report.EpochReport, *scenario, read)
	if spendWarning != "" {
		report.Warnings = append(report.Warnings, goalhome.Finding{Code: "spend-unavailable", Detail: spendWarning})
	}
	report.Findings = goalFindings(path)
	if *acceptance {
		report.AcceptanceBlocked = acceptanceBlocks(report)
	} else if len(report.StepBack) > 0 {
		newSignal := true
		if *effortRef != "" {
			signal, err := a.recordStepBack(*effortRef, path, report.StepBack, len(file.Log))
			newSignal = shouldWakeOnStepBack(signal, err)
			if err != nil {
				fmt.Fprintf(os.Stderr, "step_back signal failed: %v\n", err)
			} else {
				report.Signal = signal
			}
		}
		if *wakeKey != "" && newSignal {
			woken, err := a.services.Runs.WakeByKey(&domainpb.WakeParkedRunsRequest{Producer: "children", Key: *wakeKey, Result: "STEP_BACK " + strings.Join(report.StepBack, "; ") + " (epoch " + path + ")"})
			if err != nil {
				fmt.Fprintf(os.Stderr, "friction wake failed: %v\n", err)
			}
			report.Woken = woken
		}
	}
	if *jsonOut {
		data, err := json.Marshal(report)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
	} else {
		printEpochReport(report, *acceptance)
	}
	switch {
	case *acceptance && len(report.AcceptanceBlocked) > 0:
		return exitCodeError{code: exitRefused}
	case !*acceptance && len(report.StepBack) > 0:
		return exitCodeError{code: exitStepBack}
	}
	return nil
}

// goalFindings evaluates diminishing returns for the goal home that holds the
// epoch file (<home>/epochs/E<n>.md). Workers see the finding; only the
// orchestrator's --acceptance (and effort lint) fails on it.
func goalFindings(epochPath string) []goalhome.Finding {
	dir, ok := goalhome.EpochHome(epochPath)
	if !ok {
		return nil
	}
	// Ordinary checks use the canonical v2 home. Legacy imports are explicit.
	home, err := loadGoalHome(dir)
	if err != nil {
		return []goalhome.Finding{{Code: "goal-home-unreadable", Detail: err.Error()}}
	}
	for _, missing := range home.Missing {
		if missing == goalhome.CharterFile || missing == goalhome.StatusFile {
			return nil
		}
	}
	if finding := goalhome.ForecastOf(home).Finding(); finding != nil {
		return []goalhome.Finding{*finding}
	}
	return nil
}

// acceptanceBlocks lists what stops the orchestrator from writing ACCEPTED:
// typed gates that are not met, and blocking goal findings. A legacy exit
// gate is reported as a warning and never blocks.
func acceptanceBlocks(report *epochCheckReport) []string {
	var blocks []string
	for _, gate := range report.Gates {
		if gate.State == goalhome.GateUnmet {
			blocks = append(blocks, gate.ID+" "+gate.Type+": "+gate.Detail)
		}
	}
	for _, finding := range report.Findings {
		if finding.Blocking {
			blocks = append(blocks, finding.Code+": "+finding.Detail)
		}
	}
	return blocks
}

func printEpochReport(report *epochCheckReport, acceptance bool) {
	fmt.Printf("Epoch %s (%s): %d work units, estimate %d\n", report.File, report.Kind, report.WorkUnits, report.Estimate)
	fmt.Printf("Net lines: runtime %+d, test %+d\n", report.NetRuntimeLines, report.NetTestLines)
	fmt.Printf("Metric history: %s\n", strings.Join(report.MetricHistory, " → "))
	if len(report.OpenDirectives) > 0 {
		fmt.Printf("Open directives: %s\n", strings.Join(report.OpenDirectives, ", "))
	} else {
		fmt.Println("Open directives: none")
	}
	if report.WeightedTokens != nil {
		fmt.Printf("Weighted non-cache tokens: %.0f\n", *report.WeightedTokens)
	}
	for _, trigger := range report.Triggers {
		state := "not fired"
		if trigger.Fired {
			state = "FIRED"
		}
		fmt.Printf("  %-10s %-9s %s\n", trigger.Name, state, trigger.Detail)
	}
	if report.GateStatus != goalhome.GatesLegacy {
		fmt.Printf("Gates: %s\n", report.GateStatus)
		for _, gate := range report.Gates {
			fmt.Printf("  %-4s %-9s %-7s %s\n", gate.ID, gate.Type, gate.State, gate.Detail)
		}
	}
	if report.YieldEstimate != nil {
		fmt.Printf("Yield estimate: %s\n", report.YieldEstimate)
	}
	if report.Yield != nil {
		fmt.Printf("Yield: %s (%s)\n", report.Yield, report.YieldSource)
	} else if report.AcceptedAt != nil || report.YieldEstimate != nil {
		fmt.Println("Yield: unknown (no yield= on the ACCEPTED line and no inventory gate measured from a baseline; slice-log sums are not a yield)")
	}
	for _, warning := range report.Warnings {
		location := ""
		if warning.Line > 0 {
			location = fmt.Sprintf(" (line %d)", warning.Line)
		}
		fmt.Printf("WARNING %s%s: %s\n", warning.Code, location, warning.Detail)
	}
	for _, line := range report.MalformedLogLines {
		fmt.Printf("Malformed slice-log line (not counted): %s\n", line)
	}
	if report.Accepted != "" {
		fmt.Println(report.Accepted)
	}
	if len(report.ReopenedBy) > 0 {
		fmt.Printf("REOPENED by %s, dated after the last ACCEPTED line\n", strings.Join(report.ReopenedBy, ", "))
	}
	for _, reason := range report.AfterAcceptance {
		fmt.Printf("Accepted epoch, reported only: %s\n", reason)
	}
	for _, finding := range report.Findings {
		fmt.Printf("FINDING %s: %s\n", finding.Code, finding.Detail)
	}
	for _, reason := range report.Review {
		fmt.Printf("OBSERVE (continue) %s\n", reason)
	}
	if acceptance {
		if len(report.StepBack) > 0 {
			fmt.Printf("Step-back triggers (reported only under --acceptance): %s\n", strings.Join(report.StepBack, "; "))
		}
		if len(report.AcceptanceBlocked) > 0 {
			fmt.Printf("ACCEPTANCE_BLOCKED %s\n", strings.Join(report.AcceptanceBlocked, "; "))
		} else if report.GateStatus == goalhome.GatesLegacy {
			fmt.Println("ACCEPTANCE not checked: legacy exit gate; rerun its commands yourself")
		} else {
			fmt.Println("ACCEPTANCE ok: every admitted gate is met")
		}
		return
	}
	if len(report.StepBack) > 0 {
		fmt.Printf("STEP_BACK %s\n", strings.Join(report.StepBack, "; "))
		if len(report.Woken) > 0 {
			fmt.Printf("Woke %s\n", strings.Join(report.Woken, ", "))
		}
	}
}

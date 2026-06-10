package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// MeasuresSummary mirrors the measures-health validation summary so the phase
// pointer keeps a stable JSON shape across runs.
type MeasuresSummary struct {
	Scenario string `json:"scenario"`
	Passed   bool   `json:"passed"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Infos    int    `json:"infos"`
	Skipped  bool   `json:"skipped,omitempty"`
}

func (s MeasuresSummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%s passed=%v errors=%d warnings=%d infos=%d",
		s.Scenario, s.Passed, s.Errors, s.Warnings, s.Infos)
}

// measuresFinding is the structured shape `measures-health validate scenario
// <name> --json` emits per finding. Field names are proto wire names
// (snake_case) and severity is the proto enum string ("SEVERITY_ERROR" /
// "SEVERITY_WARNING" / "SEVERITY_INFO") — identical to the security producer so
// the mapping is uniform.
type measuresFinding struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
	FilePath    string `json:"file_path"`
	Scanner     string `json:"scanner"`
}

type measuresReport struct {
	Scenario string            `json:"scenario"`
	Passed   bool              `json:"passed"`
	Findings []measuresFinding `json:"findings"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Infos    int `json:"infos"`
	} `json:"summary"`
	SkippedScanners []string `json:"skipped_scanners"`
}

// runMeasuresValidate executes `measures-health validate scenario <name> --json`
// (adding `--probe` when probe is true) and returns the raw JSON output. A seam
// for tests, mirroring runSecurityValidate.
var runMeasuresValidate = func(ctx context.Context, scenario string, probe bool) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("measures-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate measures-health CLI: %w (install via `vrooli scenario start measures-health`)", lookErr)
	}
	args := []string{"validate", "scenario", scenario, "--json"}
	if probe {
		args = append(args, "--probe")
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	out, runErr := cmd.Output()
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			// measures-health exits non-zero when it finds ERROR findings; the
			// JSON report is still on stdout. Surface the code, not an error.
			return out, ee.ExitCode(), nil
		}
		return out, 0, runErr
	}
	return out, 0, nil
}

// measuresTargetReachable reports whether the target scenario's API is reachable,
// so the producer can behaviorally probe (`--probe`) its declared measures rather
// than just grading them statically. A seam for tests. When the target is down
// the producer falls back to the static path — an unreachable target is a skip,
// never a phase failure (test-genie sweeps non-running scenarios). Probing only
// when reachable also avoids the probe's per-measure HTTP timeouts against a dead
// endpoint; if the target races down between this check and the probe, the
// measures-health prober self-degrades (skipped, no hollow-declaration).
var measuresTargetReachable = func(ctx context.Context, scenario string) bool {
	url, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	return err == nil && strings.TrimSpace(url) != ""
}

type measuresRunResult struct {
	shared.RunResult[MeasuresSummary]
}

// measuresSkip builds a non-failing skip result for when the measures-health
// producer is unavailable (CLI absent or API unreachable). The optional measures
// phase must never fail the suite on the producer's availability — only on real
// findings it actually returns.
func measuresSkip(reason string) *measuresRunResult {
	return &measuresRunResult{
		RunResult: shared.RunResult[MeasuresSummary]{
			Success:      true,
			Observations: []shared.Observation{shared.NewSkipObservation(reason)},
			Summary:      MeasuresSummary{Passed: true, Skipped: true},
		},
	}
}

// runMeasuresPhase shells out to `measures-health validate scenario <name>
// --json` and translates the JSON findings into Observations + Summary +
// FINDING_SOURCE_MEASURES findings. Like the security phase this is a CLI
// subprocess (not raw HTTP) so the test-genie → measures-health integration
// always rides the public CLI surface. A missing CLI is a skip-class miss,
// never a hard failure.
func runMeasuresPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_MEASURES") == "1" {
		summary := MeasuresSummary{Scenario: env.ScenarioName, Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("measures phase disabled via TEST_GENIE_SKIP_MEASURES"),
			},
		}
		writePhasePointer(env, "measures", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary MeasuresSummary
	summary.Scenario = env.ScenarioName
	var archFindings []*architecturev1.ArchitectureFinding

	// Probe behaviorally when the target scenario is reachable, so EM's `measures`
	// rung reflects "actually answers" (a declared-but-unserved measure surfaces as
	// a hollow-declaration ERROR), not merely "declared". Otherwise grade statically.
	probe := measuresTargetReachable(ctx, env.ScenarioName)

	report := RunPhase(ctx, logWriter, "measures",
		func() (*measuresRunResult, error) {
			stdout, exitCode, runErr := runMeasuresValidate(ctx, env.ScenarioName, probe)
			if runErr != nil {
				return measuresSkip(fmt.Sprintf("measures-health CLI unavailable (%v) — start it via `vrooli scenario start measures-health`", runErr)), nil
			}
			if len(strings.TrimSpace(string(stdout))) == 0 {
				return measuresSkip("measures-health returned no report (is the measures-health API running?)"), nil
			}

			rep, parseErr := parseMeasuresOutput(stdout)
			if parseErr != nil {
				return &measuresRunResult{
					RunResult: shared.RunResult[MeasuresSummary]{
						Success:      false,
						Error:        fmt.Errorf("parse measures-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: shared.FailureClassSystem,
						Remediation:  "Run `measures-health validate scenario " + env.ScenarioName + " --json` manually and inspect output.",
					},
				}, nil
			}
			archFindings = measuresArchFindings(env.ScenarioName, rep)
			return translateMeasuresReport(rep, exitCode), nil
		},
		func(r *measuresRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[MeasuresSummary]
			summaryText := ""
			if r != nil {
				result = r.RunResult
				summary = r.Summary
				if summary.Scenario == "" {
					summary.Scenario = env.ScenarioName
				}
				summaryText = summary.String()
			}
			return ExtractWithSummary(
				result.Success,
				result.Error,
				result.FailureClass,
				result.Remediation,
				result.Observations,
				"📐",
				fmt.Sprintf("Measures validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings
	writePhasePointer(env, "measures", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Measures summary: %s", summary.String())
	return report
}

// measuresArchFindings maps measures-health findings into the shared
// ArchitectureFinding contract (source=MEASURES). The finding's file_path
// becomes the single location.
func measuresArchFindings(scenario string, rep *measuresReport) []*architecturev1.ArchitectureFinding {
	if rep == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
			f.RuleID, f.Severity, measuresMessage(f), f.Remediation,
			nonEmptyLocations(f.FilePath), nil,
		))
	}
	return out
}

func parseMeasuresOutput(raw []byte) (*measuresReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("measures-health produced empty output")
	}
	var rep measuresReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func translateMeasuresReport(rep *measuresReport, exitCode int) *measuresRunResult {
	out := &measuresRunResult{}
	out.Summary = MeasuresSummary{
		Scenario: rep.Scenario,
		Passed:   rep.Passed,
		Errors:   rep.Summary.Errors,
		Warnings: rep.Summary.Warnings,
		Infos:    rep.Summary.Infos,
	}

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatMeasuresFinding(f)
		switch normalizeFindingSeverity(f.Severity) {
		case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
			architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
			out.Observations = append(out.Observations, shared.NewErrorObservation(msg))
			failureCount++
		case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
			out.Observations = append(out.Observations, shared.NewWarningObservation(msg))
		default:
			out.Observations = append(out.Observations, shared.NewInfoObservation(msg))
		}
	}

	switch {
	case failureCount > 0:
		out.Success = false
		out.FailureClass = shared.FailureClassTestFailure
		out.Error = fmt.Errorf("%d measures finding(s) at ERROR severity", failureCount)
		out.Remediation = "Run `measures-health validate scenario " + rep.Scenario + "` for details; declare a measure for each uncovered stateful domain or waive it in measures.omitted[]."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("measures-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a measures-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

// measuresMessage is the human message stored on the ArchitectureFinding.
func measuresMessage(f measuresFinding) string {
	msg := strings.TrimSpace(f.Title)
	if desc := strings.TrimSpace(f.Description); desc != "" && desc != msg {
		msg += " — " + desc
	}
	if scanner := strings.TrimSpace(f.Scanner); scanner != "" {
		msg = fmt.Sprintf("[%s] %s", scanner, msg)
	}
	return msg
}

func formatMeasuresFinding(f measuresFinding) string {
	parts := []string{strings.TrimSpace(f.RuleID)}
	if t := strings.TrimSpace(f.Title); t != "" {
		parts = append(parts, t)
	}
	line := strings.Join(parts, ": ")
	if loc := strings.TrimSpace(f.FilePath); loc != "" {
		line += fmt.Sprintf(" [%s]", loc)
	}
	if rem := strings.TrimSpace(f.Remediation); rem != "" {
		line += "\n    remediation: " + rem
	}
	return line
}

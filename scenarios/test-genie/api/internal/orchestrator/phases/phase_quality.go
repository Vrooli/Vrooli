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

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// QualitySummary mirrors quality-health's audit summary so phase pointers keep
// a stable JSON shape across runs.
type QualitySummary struct {
	Scenario         string `json:"scenario"`
	Status           string `json:"status"`
	Passed           bool   `json:"passed"`
	Errors           int    `json:"errors"`
	Warnings         int    `json:"warnings"`
	Infos            int    `json:"infos"`
	Surfaces         int    `json:"surfaces"`
	Contracts        int    `json:"contracts"`
	AutofixableCount int    `json:"autofixable_count"`
	Skipped          bool   `json:"skipped,omitempty"`
}

func (s QualitySummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%s status=%s passed=%v errors=%d warnings=%d infos=%d surfaces=%d contracts=%d autofixable=%d",
		s.Scenario, s.Status, s.Passed, s.Errors, s.Warnings, s.Infos, s.Surfaces, s.Contracts, s.AutofixableCount)
}

// qualityFinding is the structured shape `quality-health audit run <scenario>
// --json` emits for each static-quality contract finding.
type qualityFinding struct {
	ID               string `json:"id"`
	RuleID           string `json:"rule_id"`
	Severity         string `json:"severity"`
	Message          string `json:"message"`
	FilePath         string `json:"file_path"`
	Remediation      string `json:"remediation"`
	Evidence         string `json:"evidence"`
	SurfaceID        string `json:"surface_id"`
	AutofixAvailable bool   `json:"autofix_available"`
	AutofixCommand   string `json:"autofix_command"`
	FixClass         string `json:"fix_class"`
}

type qualityReport struct {
	RunID    string           `json:"run_id"`
	Scenario string           `json:"scenario"`
	Status   string           `json:"status"`
	Summary  string           `json:"summary"`
	Findings []qualityFinding `json:"findings"`
	Counts   struct {
		Errors           int `json:"errors"`
		Warnings         int `json:"warnings"`
		Infos            int `json:"infos"`
		Surfaces         int `json:"surfaces"`
		Contracts        int `json:"contracts"`
		AutofixableCount int `json:"autofixable_count"`
	} `json:"counts"`
	NextSteps []string `json:"next_steps"`
}

// runQualityValidate executes `quality-health audit run <scenario> --json` and
// returns the raw JSON output. Tests replace this seam.
var runQualityValidate = func(ctx context.Context, scenario string) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("quality-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate quality-health CLI: %w (install via `vrooli scenario start quality-health`)", lookErr)
	}
	cmd := exec.CommandContext(ctx, bin, "audit", "run", scenario, "--json")
	out, runErr := cmd.Output()
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			return out, ee.ExitCode(), nil
		}
		return out, 0, runErr
	}
	return out, 0, nil
}

type qualityRunResult struct {
	shared.RunResult[QualitySummary]
}

// runQualityPhase shells out to quality-health and maps its findings into the
// standards channel. This is a mandatory phase: provider absence is a real
// missing dependency because Test Genie no longer owns a native lint/type
// fallback after the hard cutover.
func runQualityPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_QUALITY") == "1" {
		summary := QualitySummary{Scenario: env.ScenarioName, Status: "skipped", Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("quality phase disabled via TEST_GENIE_SKIP_QUALITY"),
			},
		}
		writePhasePointer(env, "quality", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary QualitySummary
	summary.Scenario = env.ScenarioName
	var archFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "quality",
		func() (*qualityRunResult, error) {
			stdout, exitCode, runErr := runQualityValidate(ctx, env.ScenarioName)
			if runErr != nil {
				return &qualityRunResult{
					RunResult: shared.RunResult[QualitySummary]{
						Success:      false,
						Error:        fmt.Errorf("invoke quality-health audit: %w", runErr),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Ensure the quality-health CLI and API are reachable (run `vrooli scenario start quality-health`).",
					},
				}, nil
			}

			rep, parseErr := parseQualityOutput(stdout)
			if parseErr != nil {
				return &qualityRunResult{
					RunResult: shared.RunResult[QualitySummary]{
						Success:      false,
						Error:        fmt.Errorf("parse quality-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: shared.FailureClassSystem,
						Remediation:  "Run `quality-health audit run " + env.ScenarioName + " --json` manually and inspect output.",
					},
				}, nil
			}
			archFindings = qualityArchFindings(env.ScenarioName, rep)
			return translateQualityReport(rep, exitCode), nil
		},
		func(r *qualityRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[QualitySummary]
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
				"🧭",
				fmt.Sprintf("Quality validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings
	writePhasePointer(env, "quality", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Quality summary: %s", summary.String())
	return report
}

func qualityArchFindings(scenario string, rep *qualityReport) []*architecturev1.ArchitectureFinding {
	if rep == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
			f.RuleID, f.Severity, qualityMessage(f), f.Remediation,
			nonEmptyLocations(f.FilePath), nil,
		))
	}
	return out
}

func parseQualityOutput(raw []byte) (*qualityReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("quality-health produced empty output")
	}
	var rep qualityReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func translateQualityReport(rep *qualityReport, exitCode int) *qualityRunResult {
	out := &qualityRunResult{}
	out.Summary = QualitySummary{
		Scenario:         rep.Scenario,
		Status:           rep.Status,
		Passed:           rep.Status == "passed" || rep.Status == "degraded",
		Errors:           rep.Counts.Errors,
		Warnings:         rep.Counts.Warnings,
		Infos:            rep.Counts.Infos,
		Surfaces:         rep.Counts.Surfaces,
		Contracts:        rep.Counts.Contracts,
		AutofixableCount: rep.Counts.AutofixableCount,
	}

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatQualityFinding(f)
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
	for _, step := range rep.NextSteps {
		if strings.TrimSpace(step) != "" {
			out.Observations = append(out.Observations, shared.NewInfoObservation("next step: "+strings.TrimSpace(step)))
		}
	}
	if rep.Counts.AutofixableCount > 0 {
		out.Observations = append(out.Observations, shared.NewInfoObservation(fmt.Sprintf("%d quality finding(s) autofixable; run `quality-health fix-config run %s --dry-run`.", rep.Counts.AutofixableCount, rep.Scenario)))
	}

	switch {
	case failureCount > 0 || rep.Counts.Errors > 0 || strings.EqualFold(rep.Status, "failed"):
		out.Success = false
		out.FailureClass = shared.FailureClassTestFailure
		out.Error = fmt.Errorf("%d quality finding(s) at ERROR severity", maxInt(failureCount, rep.Counts.Errors))
		out.Remediation = "Run `quality-health audit run " + rep.Scenario + " --json` for details and fix strict lint/type quality contract violations."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("quality-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a quality-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

func qualityMessage(f qualityFinding) string {
	msg := strings.TrimSpace(f.Message)
	if msg == "" {
		msg = strings.TrimSpace(f.RuleID)
	}
	if evidence := strings.TrimSpace(f.Evidence); evidence != "" {
		msg += " — " + evidence
	}
	return msg
}

func formatQualityFinding(f qualityFinding) string {
	parts := []string{strings.TrimSpace(f.RuleID)}
	if msg := strings.TrimSpace(f.Message); msg != "" {
		parts = append(parts, msg)
	}
	line := strings.Join(parts, ": ")
	if loc := strings.TrimSpace(f.FilePath); loc != "" {
		line += fmt.Sprintf(" [%s]", loc)
	}
	if rem := strings.TrimSpace(f.Remediation); rem != "" {
		line += "\n    remediation: " + rem
	}
	if f.AutofixAvailable {
		command := strings.TrimSpace(f.AutofixCommand)
		if command == "" {
			command = "quality-health fix-config run <scenario> --dry-run"
		}
		line += "\n    autofix: " + command
	}
	return line
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

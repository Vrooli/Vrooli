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
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// UnitSummary mirrors unit-health's validation summary so the unit phase
// pointer keeps a stable JSON shape across runs. The unit phase is a thin
// delegate over `unit-health validate scenario <name>` after the hard cutover —
// Test Genie no longer owns a native Go/Node/Python/shell test runner or a
// separate coverage parser.
type UnitSummary struct {
	Scenario          string `json:"scenario"`
	Status            string `json:"status"`
	Passed            bool   `json:"passed"`
	Errors            int    `json:"errors"`
	Warnings          int    `json:"warnings"`
	Infos             int    `json:"infos"`
	Surfaces          int    `json:"surfaces"`
	Workspaces        int    `json:"workspaces"`
	CoverageTargets   int    `json:"coverage_targets"`
	LocalCurrentLevel string `json:"local_current_level,omitempty"`
	LocalNextLevel    string `json:"local_next_level,omitempty"`
	Skipped           bool   `json:"skipped,omitempty"`
}

func (s UnitSummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	text := fmt.Sprintf("%s status=%s passed=%v errors=%d warnings=%d infos=%d surfaces=%d workspaces=%d coverage_targets=%d",
		s.Scenario, s.Status, s.Passed, s.Errors, s.Warnings, s.Infos, s.Surfaces, s.Workspaces, s.CoverageTargets)
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		text += fmt.Sprintf(" local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
	}
	return text
}

// unitFinding is the structured shape `unit-health validate scenario <name>
// --json` emits for each test-maturity finding (test execution, coverage,
// architecture, quality, diagnostics).
type unitFinding struct {
	ID           string `json:"id"`
	WorkspaceID  string `json:"workspace_id"`
	SurfaceID    string `json:"surface_id"`
	Language     string `json:"language"`
	Code         string `json:"code"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	FilePath     string `json:"file_path"`
	Symbol       string `json:"symbol"`
	Message      string `json:"message"`
	Evidence     string `json:"evidence"`
	Remediation  string `json:"remediation"`
	WhyItMatters string `json:"why_it_matters"`
}

type unitReport struct {
	RunID          string                       `json:"run_id"`
	Scenario       string                       `json:"scenario"`
	Status         string                       `json:"status"`
	Summary        string                       `json:"summary"`
	DegradedReason string                       `json:"degraded_reason"`
	Findings       []unitFinding                `json:"findings"`
	AssessmentRaw  json.RawMessage              `json:"assessment"`
	Assessment     *commonv1.MaturityAssessment `json:"-"`
	Counts         struct {
		Errors          int `json:"errors"`
		Warnings        int `json:"warnings"`
		Infos           int `json:"infos"`
		Surfaces        int `json:"surfaces"`
		Workspaces      int `json:"workspaces"`
		CoverageTargets int `json:"coverage_targets"`
	} `json:"counts"`
	NextSteps []string `json:"next_steps"`
}

// runUnitValidate executes `unit-health validate scenario <name> --execution
// --json` and returns the raw JSON output. The `--execution` flag is required:
// the unit phase must actually run the planned test commands, not just plan
// them. Tests replace this seam.
var runUnitValidate = func(ctx context.Context, scenario string) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("unit-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate unit-health CLI: %w (install via `vrooli scenario start unit-health`)", lookErr)
	}
	cmd := exec.CommandContext(ctx, bin, "validate", "scenario", scenario, "--execution", "--json")
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

type unitRunResult struct {
	shared.RunResult[UnitSummary]
}

// runUnitPhase shells out to unit-health and maps its findings into the shared
// finding contract. Coverage-category findings are emitted into the COVERAGE
// finding channel (preserving the dimension semantics of the retired separate
// `coverage` phase); all findings additionally surface as observations. This is
// a mandatory phase: provider absence is a real missing dependency because Test
// Genie no longer owns a native unit-test runner after the hard cutover.
func runUnitPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_UNIT") == "1" {
		summary := UnitSummary{Scenario: env.ScenarioName, Status: "skipped", Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("unit phase disabled via TEST_GENIE_SKIP_UNIT"),
			},
		}
		writePhasePointer(env, "unit", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary UnitSummary
	summary.Scenario = env.ScenarioName
	var coverageFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "unit",
		func() (*unitRunResult, error) {
			stdout, exitCode, runErr := runUnitValidate(ctx, env.ScenarioName)
			if runErr != nil {
				return &unitRunResult{
					RunResult: shared.RunResult[UnitSummary]{
						Success:      false,
						Error:        fmt.Errorf("invoke unit-health validate: %w", runErr),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Ensure the unit-health CLI and API are reachable (run `vrooli scenario start unit-health`).",
					},
				}, nil
			}

			rep, parseErr := parseUnitOutput(stdout)
			if parseErr != nil {
				return &unitRunResult{
					RunResult: shared.RunResult[UnitSummary]{
						Success:      false,
						Error:        fmt.Errorf("parse unit-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: classifyProviderParseFailure(parseErr),
						Remediation:  "Run `unit-health validate scenario " + env.ScenarioName + " --execution --json` manually and inspect output.",
					},
				}, nil
			}
			coverageFindings = unitCoverageFindings(env.ScenarioName, rep)
			return translateUnitReport(rep, exitCode), nil
		},
		func(r *unitRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[UnitSummary]
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
				"🧪",
				fmt.Sprintf("Unit validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = coverageFindings
	writePhasePointer(env, "unit", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Unit summary: %s", summary.String())
	return report
}

// unitCoverageFindings maps unit-health coverage-category findings into the
// shared ArchitectureFinding contract on the COVERAGE channel. After the hard
// cutover the unit phase owns coverage analysis (the separate `coverage` phase
// is gone), so its coverage findings must still land in the `coverage`
// dimension via the FINDING_SOURCE_COVERAGE token. Non-coverage findings (test
// architecture, quality, execution, diagnostics) are surfaced as observations
// only — there is no architecture FindingSource for the `tests` dimension; the
// `unit` phase's PhaseMap entry attributes those to `tests`.
func unitCoverageFindings(scenario string, rep *unitReport) []*architecturev1.ArchitectureFinding {
	if rep == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		if !strings.EqualFold(strings.TrimSpace(f.Category), "coverage") {
			continue
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
			f.Code, f.Severity, unitMessage(f), f.Remediation,
			nonEmptyLocations(f.FilePath), nil,
		))
	}
	return out
}

func parseUnitOutput(raw []byte) (*unitReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("unit-health produced empty output")
	}
	var rep unitReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	assessment, err := requireProviderAssessmentJSON("unit-health", "unit", rep.AssessmentRaw)
	if err != nil {
		return nil, err
	}
	rep.Assessment = assessment
	return &rep, nil
}

func translateUnitReport(rep *unitReport, exitCode int) *unitRunResult {
	out := &unitRunResult{}
	out.Summary = UnitSummary{
		Scenario:        rep.Scenario,
		Status:          rep.Status,
		Passed:          rep.Status == "passed" || rep.Status == "degraded",
		Errors:          rep.Counts.Errors,
		Warnings:        rep.Counts.Warnings,
		Infos:           rep.Counts.Infos,
		Surfaces:        rep.Counts.Surfaces,
		Workspaces:      rep.Counts.Workspaces,
		CoverageTargets: rep.Counts.CoverageTargets,
	}
	out.Summary.LocalCurrentLevel, out.Summary.LocalNextLevel = localMaturitySummary(rep.Assessment)

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatUnitFinding(f)
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
	if strings.TrimSpace(rep.DegradedReason) != "" {
		out.Observations = append(out.Observations, shared.NewWarningObservation("degraded: "+strings.TrimSpace(rep.DegradedReason)))
	}
	for _, step := range rep.NextSteps {
		if strings.TrimSpace(step) != "" {
			out.Observations = append(out.Observations, shared.NewInfoObservation("next step: "+strings.TrimSpace(step)))
		}
	}

	switch {
	case failureCount > 0 || rep.Counts.Errors > 0 || strings.EqualFold(rep.Status, "failed"):
		out.Success = false
		out.FailureClass = shared.FailureClassTestFailure
		out.Error = fmt.Errorf("%d unit finding(s) at ERROR severity", maxInt(failureCount, rep.Counts.Errors))
		out.Remediation = "Run `unit-health validate scenario " + rep.Scenario + " --execution --json` for details and fix the failing/uncovered tests."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("unit-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a unit-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

func unitMessage(f unitFinding) string {
	msg := strings.TrimSpace(f.Message)
	if msg == "" {
		msg = strings.TrimSpace(f.Code)
	}
	if evidence := strings.TrimSpace(f.Evidence); evidence != "" {
		msg += " — " + evidence
	}
	return msg
}

func formatUnitFinding(f unitFinding) string {
	parts := []string{strings.TrimSpace(f.Code)}
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
	return line
}

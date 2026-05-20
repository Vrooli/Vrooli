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
)

// UIHealthSummary mirrors the ui-health validation summary so the phase
// pointer keeps stable JSON shape across runs.
type UIHealthSummary struct {
	Scenario string `json:"scenario"`
	Passed   bool   `json:"passed"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Infos    int    `json:"infos"`
	Skipped  bool   `json:"skipped,omitempty"`
}

func (s UIHealthSummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%s passed=%v errors=%d warnings=%d infos=%d",
		s.Scenario, s.Passed, s.Errors, s.Warnings, s.Infos)
}

type uiHealthFinding struct {
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	Location   string `json:"location"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

type uiHealthReport struct {
	Scenario string            `json:"scenario"`
	Passed   bool              `json:"passed"`
	Findings []uiHealthFinding `json:"findings"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Infos    int `json:"infos"`
	} `json:"summary"`
}

// runUIHealthValidate is the seam tests substitute.
var runUIHealthValidate = func(ctx context.Context, scenario string) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("ui-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate ui-health CLI: %w (install via `vrooli scenario start ui-health`)", lookErr)
	}
	cmd := exec.CommandContext(ctx, bin, "validate", "scenario", scenario, "--json")
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

// runUIHealthPhase shells out to `ui-health validate scenario <name> --json`
// and translates the JSON findings into Observations + Summary.
func runUIHealthPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_UI_HEALTH") == "1" {
		summary := UIHealthSummary{Scenario: env.ScenarioName, Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("ui-health phase disabled via TEST_GENIE_SKIP_UI_HEALTH"),
			},
		}
		writePhasePointer(env, "ui-health", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary UIHealthSummary
	summary.Scenario = env.ScenarioName

	report := RunPhase(ctx, logWriter, "ui-health",
		func() (*uiHealthRunResult, error) {
			stdout, exitCode, runErr := runUIHealthValidate(ctx, env.ScenarioName)
			if runErr != nil {
				return &uiHealthRunResult{
					RunResult: shared.RunResult[UIHealthSummary]{
						Success:      false,
						Error:        fmt.Errorf("invoke ui-health validate: %w", runErr),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Ensure the ui-health CLI is installed and reachable (run `vrooli scenario start ui-health`).",
					},
				}, nil
			}

			rep, parseErr := parseUIHealthOutput(stdout)
			if parseErr != nil {
				return &uiHealthRunResult{
					RunResult: shared.RunResult[UIHealthSummary]{
						Success:      false,
						Error:        fmt.Errorf("parse ui-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: shared.FailureClassSystem,
						Remediation:  "Run `ui-health validate scenario " + env.ScenarioName + " --json` manually and inspect output.",
					},
				}, nil
			}
			return translateUIHealthReport(rep, exitCode), nil
		},
		func(r *uiHealthRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[UIHealthSummary]
			summaryText := ""
			if r != nil {
				result = r.RunResult
				summary = r.Summary
				summaryText = r.Summary.String()
			}
			return ExtractWithSummary(
				result.Success,
				result.Error,
				result.FailureClass,
				result.Remediation,
				result.Observations,
				"🎨",
				fmt.Sprintf("UI-health validation completed (%s)", summaryText),
			)
		},
	)

	writePhasePointer(env, "ui-health", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "UI-health summary: %s", summary.String())
	return report
}

type uiHealthRunResult struct {
	shared.RunResult[UIHealthSummary]
}

func parseUIHealthOutput(raw []byte) (*uiHealthReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("ui-health produced empty output")
	}
	var rep uiHealthReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func translateUIHealthReport(rep *uiHealthReport, exitCode int) *uiHealthRunResult {
	out := &uiHealthRunResult{}
	out.Summary = UIHealthSummary{
		Scenario: rep.Scenario,
		Passed:   rep.Passed,
		Errors:   rep.Summary.Errors,
		Warnings: rep.Summary.Warnings,
		Infos:    rep.Summary.Infos,
	}

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatUIHealthFinding(f)
		switch f.Severity {
		case "SEVERITY_ERROR":
			out.Observations = append(out.Observations, shared.NewErrorObservation(msg))
			failureCount++
		case "SEVERITY_WARNING":
			out.Observations = append(out.Observations, shared.NewWarningObservation(msg))
		default:
			out.Observations = append(out.Observations, shared.NewInfoObservation(msg))
		}
	}

	switch {
	case failureCount > 0:
		out.Success = false
		out.FailureClass = shared.FailureClassTestFailure
		out.Error = fmt.Errorf("%d ui-health finding(s) at ERROR severity", failureCount)
		out.Remediation = "Run `ui-health validate scenario " + rep.Scenario + "` for details and fix the manifest/slot drift."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("ui-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a ui-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

func formatUIHealthFinding(f uiHealthFinding) string {
	parts := []string{strings.TrimSpace(f.Code)}
	if msg := strings.TrimSpace(f.Message); msg != "" {
		parts = append(parts, msg)
	}
	line := strings.Join(parts, ": ")
	if loc := strings.TrimSpace(f.Location); loc != "" {
		line += fmt.Sprintf(" [%s]", loc)
	}
	if sug := strings.TrimSpace(f.Suggestion); sug != "" {
		line += "\n    suggestion: " + sug
	}
	return line
}

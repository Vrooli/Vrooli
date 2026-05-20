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

// ContractsSummary mirrors the cli-health validation summary so the phase
// pointer keeps stable JSON shape across runs.
type ContractsSummary struct {
	Scenario string `json:"scenario"`
	Passed   bool   `json:"passed"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Infos    int    `json:"infos"`
	Skipped  bool   `json:"skipped,omitempty"`
}

func (s ContractsSummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%s passed=%v errors=%d warnings=%d infos=%d",
		s.Scenario, s.Passed, s.Errors, s.Warnings, s.Infos)
}

// cliHealthFinding is the structured shape `cli-health validate --json` emits
// for each finding. The shape mirrors validation_pb proto JSON.
type cliHealthFinding struct {
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	Location   string `json:"location"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

type cliHealthReport struct {
	Scenario string             `json:"scenario"`
	Passed   bool               `json:"passed"`
	Findings []cliHealthFinding `json:"findings"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Infos    int `json:"infos"`
	} `json:"summary"`
}

// Seam for tests: runCLIHealthValidate executes `cli-health validate scenario
// <name> --json` and returns the raw JSON output. Tests substitute a fake.
var runCLIHealthValidate = func(ctx context.Context, scenario string) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("cli-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate cli-health CLI: %w (install via `vrooli scenario start cli-health`)", lookErr)
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

// runContractsPhase shells out to `cli-health validate scenario <name>
// --json` and translates the JSON findings into Observations + Summary.
// Per the cli-health-scenario-v1 plan, this is a CLI subprocess (not raw
// HTTP) so the test-genie → cli-health integration always rides the
// public CLI surface.
func runContractsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_CONTRACTS") == "1" {
		summary := ContractsSummary{Scenario: env.ScenarioName, Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("contracts phase disabled via TEST_GENIE_SKIP_CONTRACTS"),
			},
		}
		writePhasePointer(env, "contracts", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary ContractsSummary
	summary.Scenario = env.ScenarioName

	report := RunPhase(ctx, logWriter, "contracts",
		func() (*contractsRunResult, error) {
			stdout, exitCode, runErr := runCLIHealthValidate(ctx, env.ScenarioName)
			if runErr != nil {
				return &contractsRunResult{
					RunResult: shared.RunResult[ContractsSummary]{
						Success:      false,
						Error:        fmt.Errorf("invoke cli-health validate: %w", runErr),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Ensure the cli-health CLI is installed and reachable (run `vrooli scenario start cli-health`).",
					},
				}, nil
			}

			rep, parseErr := parseCLIHealthOutput(stdout)
			if parseErr != nil {
				return &contractsRunResult{
					RunResult: shared.RunResult[ContractsSummary]{
						Success:      false,
						Error:        fmt.Errorf("parse cli-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: shared.FailureClassSystem,
						Remediation:  "Run `cli-health validate scenario " + env.ScenarioName + " --json` manually and inspect output.",
					},
				}, nil
			}
			return translateContractsReport(rep, exitCode), nil
		},
		func(r *contractsRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[ContractsSummary]
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
				"📑",
				fmt.Sprintf("Contracts validation completed (%s)", summaryText),
			)
		},
	)

	writePhasePointer(env, "contracts", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Contracts summary: %s", summary.String())
	return report
}

type contractsRunResult struct {
	shared.RunResult[ContractsSummary]
}

func parseCLIHealthOutput(raw []byte) (*cliHealthReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("cli-health produced empty output")
	}
	var rep cliHealthReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func translateContractsReport(rep *cliHealthReport, exitCode int) *contractsRunResult {
	out := &contractsRunResult{}
	out.Summary = ContractsSummary{
		Scenario: rep.Scenario,
		Passed:   rep.Passed,
		Errors:   rep.Summary.Errors,
		Warnings: rep.Summary.Warnings,
		Infos:    rep.Summary.Infos,
	}

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatContractFinding(f)
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
		out.Error = fmt.Errorf("%d contracts finding(s) at ERROR severity", failureCount)
		out.Remediation = "Run `cli-health validate scenario " + rep.Scenario + "` for details and fix the manifest/proto drift."
	case exitCode != 0:
		// Non-zero exit with no error findings would indicate a bug in
		// cli-health; surface it as a system failure rather than silently passing.
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("cli-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a cli-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

func formatContractFinding(f cliHealthFinding) string {
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

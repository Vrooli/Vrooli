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

// ProtoSummary mirrors the proto-health validation summary so phase pointers
// keep a stable JSON shape across runs.
type ProtoSummary struct {
	Scenario string `json:"scenario"`
	Passed   bool   `json:"passed"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Infos    int    `json:"infos"`
	Skipped  bool   `json:"skipped,omitempty"`
}

func (s ProtoSummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%s passed=%v errors=%d warnings=%d infos=%d",
		s.Scenario, s.Passed, s.Errors, s.Warnings, s.Infos)
}

// protoFinding is the structured shape `proto-health validate scenario <name>
// --json` emits per finding.
type protoFinding struct {
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	Location   string `json:"location"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

type protoReport struct {
	Scenario string         `json:"scenario"`
	Passed   bool           `json:"passed"`
	Findings []protoFinding `json:"findings"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Infos    int `json:"infos"`
	} `json:"summary"`
}

// runProtoValidate executes `proto-health validate scenario <name> --json` and
// returns the raw JSON output. Tests replace this seam.
var runProtoValidate = func(ctx context.Context, scenario string) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("proto-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate proto-health CLI: %w (install via `vrooli scenario start proto-health`)", lookErr)
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

type protoRunResult struct {
	shared.RunResult[ProtoSummary]
}

func protoSkip(reason string) *protoRunResult {
	return &protoRunResult{
		RunResult: shared.RunResult[ProtoSummary]{
			Success:      true,
			Observations: []shared.Observation{shared.NewSkipObservation(reason)},
			Summary:      ProtoSummary{Passed: true, Skipped: true},
		},
	}
}

// runProtoPhase shells out to `proto-health validate scenario <name> --json`
// and translates findings into the shared ArchitectureFinding seam. The phase
// is optional, so producer absence is a skip; real ERROR findings fail.
func runProtoPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_PROTO") == "1" {
		summary := ProtoSummary{Scenario: env.ScenarioName, Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("proto phase disabled via TEST_GENIE_SKIP_PROTO"),
			},
		}
		writePhasePointer(env, "proto", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary ProtoSummary
	summary.Scenario = env.ScenarioName
	var archFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "proto",
		func() (*protoRunResult, error) {
			stdout, exitCode, runErr := runProtoValidate(ctx, env.ScenarioName)
			if runErr != nil {
				return protoSkip(fmt.Sprintf("proto-health CLI unavailable (%v) — start it via `vrooli scenario start proto-health`", runErr)), nil
			}
			if len(strings.TrimSpace(string(stdout))) == 0 {
				return protoSkip("proto-health returned no report (is the proto-health API running?)"), nil
			}

			rep, parseErr := parseProtoOutput(stdout)
			if parseErr != nil {
				return &protoRunResult{
					RunResult: shared.RunResult[ProtoSummary]{
						Success:      false,
						Error:        fmt.Errorf("parse proto-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: shared.FailureClassSystem,
						Remediation:  "Run `proto-health validate scenario " + env.ScenarioName + " --json` manually and inspect output.",
					},
				}, nil
			}
			archFindings = protoArchFindings(env.ScenarioName, rep)
			return translateProtoReport(rep, exitCode), nil
		},
		func(r *protoRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[ProtoSummary]
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
				"🧬",
				fmt.Sprintf("Proto validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings
	writePhasePointer(env, "proto", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Proto summary: %s", summary.String())
	return report
}

func protoArchFindings(scenario string, rep *protoReport) []*architecturev1.ArchitectureFinding {
	if rep == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_PROTO,
			f.Code, f.Severity, f.Message, f.Suggestion,
			nonEmptyLocations(f.Location), nil,
		))
	}
	return out
}

func parseProtoOutput(raw []byte) (*protoReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("proto-health produced empty output")
	}
	var rep protoReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

func translateProtoReport(rep *protoReport, exitCode int) *protoRunResult {
	out := &protoRunResult{}
	out.Summary = ProtoSummary{
		Scenario: rep.Scenario,
		Passed:   rep.Passed,
		Errors:   rep.Summary.Errors,
		Warnings: rep.Summary.Warnings,
		Infos:    rep.Summary.Infos,
	}

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatProtoFinding(f)
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
		out.Error = fmt.Errorf("%d proto finding(s) at ERROR severity", failureCount)
		out.Remediation = "Run `proto-health validate scenario " + rep.Scenario + "` for details and fix proto contract drift."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("proto-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a proto-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

func formatProtoFinding(f protoFinding) string {
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

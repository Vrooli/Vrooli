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

// SecuritySummary mirrors the security-health validation summary so the phase
// pointer keeps a stable JSON shape across runs.
type SecuritySummary struct {
	Scenario          string `json:"scenario"`
	Passed            bool   `json:"passed"`
	Errors            int    `json:"errors"`
	Warnings          int    `json:"warnings"`
	Infos             int    `json:"infos"`
	LocalCurrentLevel string `json:"local_current_level,omitempty"`
	LocalNextLevel    string `json:"local_next_level,omitempty"`
	Skipped           bool   `json:"skipped,omitempty"`
}

func (s SecuritySummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	text := fmt.Sprintf("%s passed=%v errors=%d warnings=%d infos=%d",
		s.Scenario, s.Passed, s.Errors, s.Warnings, s.Infos)
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		text += fmt.Sprintf(" local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
	}
	return text
}

// securityFinding is the structured shape `security-health validate scenario
// <name> --json` emits per finding. Field names are proto wire names
// (UseProtoNames: snake_case) and severity is the proto enum string
// ("SEVERITY_ERROR" / "SEVERITY_WARNING" / "SEVERITY_INFO").
type securityFinding struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
	FilePath    string `json:"file_path"`
	Scanner     string `json:"scanner"`
}

type securityReport struct {
	Scenario      string                       `json:"scenario"`
	Passed        bool                         `json:"passed"`
	Findings      []securityFinding            `json:"findings"`
	AssessmentRaw json.RawMessage              `json:"assessment"`
	Assessment    *commonv1.MaturityAssessment `json:"-"`
	Summary       struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Infos    int `json:"infos"`
	} `json:"summary"`
	SkippedScanners []string `json:"skipped_scanners"`
}

// runSecurityValidate executes `security-health validate scenario <name>
// --json` and returns the raw JSON output. A seam for tests, mirroring
// runCLIHealthValidate.
var runSecurityValidate = func(ctx context.Context, scenario string) (stdout []byte, exitCode int, err error) {
	bin, lookErr := exec.LookPath("security-health")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate security-health CLI: %w (install via `vrooli scenario start security-health`)", lookErr)
	}
	cmd := exec.CommandContext(ctx, bin, "validate", "scenario", scenario, "--json")
	out, runErr := cmd.Output()
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			// security-health exits non-zero when it finds ERROR findings; the
			// JSON report is still on stdout. Surface the code, not an error.
			return out, ee.ExitCode(), nil
		}
		return out, 0, runErr
	}
	return out, 0, nil
}

type securityRunResult struct {
	shared.RunResult[SecuritySummary]
}

// securitySkip builds a non-failing skip result for when the security-health
// producer is unavailable (CLI absent or API unreachable). The optional
// security phase must never fail the suite on the producer's availability —
// only on real findings it actually returns.
func securitySkip(reason string) *securityRunResult {
	return &securityRunResult{
		RunResult: shared.RunResult[SecuritySummary]{
			Success:      true,
			Observations: []shared.Observation{shared.NewSkipObservation(reason)},
			Summary:      SecuritySummary{Passed: true, Skipped: true},
		},
	}
}

// runSecurityPhase shells out to `security-health validate scenario <name>
// --json` and translates the JSON findings into Observations + Summary +
// FINDING_SOURCE_SECURITY findings. Like the contracts/ui-health phases this
// is a CLI subprocess (not raw HTTP) so the test-genie → security-health
// integration always rides the public CLI surface. A missing CLI is a
// skip-class miss (FailureClassMissingDependency), never a hard failure.
func runSecurityPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_SECURITY") == "1" {
		summary := SecuritySummary{Scenario: env.ScenarioName, Passed: true, Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation("security phase disabled via TEST_GENIE_SKIP_SECURITY"),
			},
		}
		writePhasePointer(env, "security", report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary SecuritySummary
	summary.Scenario = env.ScenarioName
	var archFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "security",
		func() (*securityRunResult, error) {
			stdout, exitCode, runErr := runSecurityValidate(ctx, env.ScenarioName)
			if runErr != nil {
				// CLI binary not on PATH — the producer scenario isn't
				// installed. This is an OPTIONAL phase, so degrade to a
				// non-failing skip rather than crashing the suite.
				return securitySkip(fmt.Sprintf("security-health CLI unavailable (%v) — start it via `vrooli scenario start security-health`", runErr)), nil
			}
			if len(strings.TrimSpace(string(stdout))) == 0 {
				// CLI present but produced no report (commonly: the
				// security-health API server isn't running, so the Connect
				// call was refused). Treat as a skip, not a failure — the
				// gate only acts on real findings, never on the producer's
				// availability.
				return securitySkip("security-health returned no report (is the security-health API running?)"), nil
			}

			rep, parseErr := parseSecurityOutput(stdout)
			if parseErr != nil {
				return &securityRunResult{
					RunResult: shared.RunResult[SecuritySummary]{
						Success:      false,
						Error:        fmt.Errorf("parse security-health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: classifyProviderParseFailure(parseErr),
						Remediation:  "Run `security-health validate scenario " + env.ScenarioName + "` for the human report, then retry with `--json` if the structural output still looks malformed.",
					},
				}, nil
			}
			archFindings = securityArchFindings(env.ScenarioName, rep)
			return translateSecurityReport(rep, exitCode), nil
		},
		func(r *securityRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[SecuritySummary]
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
				"🔐",
				fmt.Sprintf("Security validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings
	writePhasePointer(env, "security", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Security summary: %s", summary.String())
	return report
}

// securityArchFindings maps security-health findings into the shared
// ArchitectureFinding contract (source=SECURITY). The finding's file_path
// becomes the single location; security findings carry no domains.
func securityArchFindings(scenario string, rep *securityReport) []*architecturev1.ArchitectureFinding {
	if rep == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
			f.RuleID, f.Severity, securityMessage(f), f.Remediation,
			nonEmptyLocations(f.FilePath), nil,
		))
	}
	return out
}

func parseSecurityOutput(raw []byte) (*securityReport, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("security-health produced empty output")
	}
	var rep securityReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		return nil, err
	}
	assessment, err := requireProviderAssessmentJSON("security-health", "security", rep.AssessmentRaw)
	if err != nil {
		return nil, err
	}
	rep.Assessment = assessment
	return &rep, nil
}

func translateSecurityReport(rep *securityReport, exitCode int) *securityRunResult {
	out := &securityRunResult{}
	out.Summary = SecuritySummary{
		Scenario: rep.Scenario,
		Passed:   rep.Passed,
		Errors:   rep.Summary.Errors,
		Warnings: rep.Summary.Warnings,
		Infos:    rep.Summary.Infos,
	}
	out.Summary.LocalCurrentLevel, out.Summary.LocalNextLevel = localMaturitySummary(rep.Assessment)

	failureCount := 0
	for _, f := range rep.Findings {
		msg := formatSecurityFinding(f)
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
		out.Error = fmt.Errorf("%d security finding(s) at ERROR severity", failureCount)
		out.Remediation = "Run `security-health validate scenario " + rep.Scenario + "` for details; rotate exposed secrets and bump vulnerable dependencies."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("security-health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a security-health issue; the validator should always emit findings when it fails."
	default:
		out.Success = true
	}
	return out
}

// securityMessage is the human message stored on the ArchitectureFinding.
func securityMessage(f securityFinding) string {
	msg := strings.TrimSpace(f.Title)
	if desc := strings.TrimSpace(f.Description); desc != "" && desc != msg {
		msg += " — " + desc
	}
	if scanner := strings.TrimSpace(f.Scanner); scanner != "" {
		msg = fmt.Sprintf("[%s] %s", scanner, msg)
	}
	return msg
}

func formatSecurityFinding(f securityFinding) string {
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

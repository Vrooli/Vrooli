package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	"google.golang.org/protobuf/encoding/protojson"
)

// DependencySummary mirrors the scenario-dependency-analyzer health summary so
// phase pointers keep a stable JSON shape while SDA owns dependency semantics.
type DependencySummary struct {
	Scenario             string `json:"scenario"`
	Passed               bool   `json:"passed"`
	Sections             int    `json:"sections"`
	Surfaces             int    `json:"surfaces"`
	Findings             int    `json:"findings"`
	Errors               int    `json:"errors"`
	Warnings             int    `json:"warnings"`
	Infos                int    `json:"infos"`
	DegradedDependencies int    `json:"degraded_dependencies"`
	GovernanceStatus     string `json:"governance_status,omitempty"`
	PolicyStatus         string `json:"policy_status,omitempty"`
	LocalCurrentLevel    string `json:"local_current_level,omitempty"`
	LocalNextLevel       string `json:"local_next_level,omitempty"`
}

func (s DependencySummary) String() string {
	text := fmt.Sprintf("%s passed=%t sections=%d surfaces=%d findings=%d errors=%d warnings=%d infos=%d degraded=%d governance=%s policy=%s",
		s.Scenario, s.Passed, s.Sections, s.Surfaces, s.Findings, s.Errors, s.Warnings, s.Infos, s.DegradedDependencies, s.GovernanceStatus, s.PolicyStatus)
	if s.LocalCurrentLevel != "" || s.LocalNextLevel != "" {
		text += fmt.Sprintf(" local=%s next=%s", s.LocalCurrentLevel, s.LocalNextLevel)
	}
	return text
}

// runDependencyHealth executes the single SDA-owned producer for Test Genie's
// dependencies phase. It is a seam for tests and intentionally uses SDA health,
// not SDA drift plus native Test Genie checks.
var runDependencyHealth = func(ctx context.Context, scenario string) ([]byte, int, error) {
	out, err := phaseCommandCapture(ctx, "", nil, "scenario-dependency-analyzer", "health", scenario, "--json")
	if err != nil {
		var ee interface{ ExitCode() int }
		if errors.As(err, &ee) {
			return []byte(out), ee.ExitCode(), nil
		}
		return []byte(out), 0, err
	}
	return []byte(out), 0, nil
}

type dependencyHealthRunResult struct {
	shared.RunResult[DependencySummary]
}

// runDependenciesPhase delegates dependency validation to
// scenario-dependency-analyzer health <scenario> --json and maps the returned
// SDA findings into Test Genie's dependency finding channel.
func runDependenciesPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary DependencySummary
	summary.Scenario = env.ScenarioName
	var archFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "dependencies",
		func() (*dependencyHealthRunResult, error) {
			stdout, exitCode, runErr := runDependencyHealth(ctx, env.ScenarioName)
			if runErr != nil {
				msg := fmt.Sprintf("scenario-dependency-analyzer health unavailable: %v", runErr)
				archFindings = []*architecturev1.ArchitectureFinding{
					newFinding(
						env.ScenarioName,
						architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
						"dependency.producer_unavailable",
						"ERROR",
						msg,
						"Start scenario-dependency-analyzer with `vrooli scenario start scenario-dependency-analyzer`, then rerun the dependencies phase.",
						nil,
						[]string{"dependencies"},
					),
				}
				return &dependencyHealthRunResult{
					RunResult: shared.RunResult[DependencySummary]{
						Success:      false,
						Error:        fmt.Errorf("%s", msg),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Start scenario-dependency-analyzer with `vrooli scenario start scenario-dependency-analyzer`, then rerun the dependencies phase.",
						Summary:      summary,
					},
				}, nil
			}
			if len(strings.TrimSpace(string(stdout))) == 0 {
				msg := "scenario-dependency-analyzer health returned no report"
				archFindings = []*architecturev1.ArchitectureFinding{
					newFinding(
						env.ScenarioName,
						architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
						"dependency.producer_empty_report",
						"ERROR",
						msg,
						"Verify the scenario-dependency-analyzer API is running and rerun `scenario-dependency-analyzer health "+env.ScenarioName+" --json`.",
						nil,
						[]string{"dependencies"},
					),
				}
				return &dependencyHealthRunResult{
					RunResult: shared.RunResult[DependencySummary]{
						Success:      false,
						Error:        errors.New(msg),
						FailureClass: shared.FailureClassMissingDependency,
						Remediation:  "Verify the scenario-dependency-analyzer API is running, then rerun the dependencies phase.",
						Summary:      summary,
					},
				}, nil
			}
			resp, parseErr := parseDependencyHealthOutput(stdout)
			if parseErr != nil {
				return &dependencyHealthRunResult{
					RunResult: shared.RunResult[DependencySummary]{
						Success:      false,
						Error:        fmt.Errorf("parse scenario-dependency-analyzer health output: %w (exit=%d)", parseErr, exitCode),
						FailureClass: classifyProviderParseFailure(parseErr),
						Remediation:  "Run `scenario-dependency-analyzer health " + env.ScenarioName + " --json` directly and fix malformed producer output.",
						Summary:      summary,
					},
				}, nil
			}
			if resp.GetScenario() == "" {
				resp.Scenario = env.ScenarioName
			}
			archFindings = dependencyHealthArchFindings(env.ScenarioName, resp)
			return translateDependencyHealthReport(resp, exitCode), nil
		},
		func(r *dependencyHealthRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[DependencySummary]
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
				"📦",
				fmt.Sprintf("Dependency validation completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings

	writePhasePointer(env, "dependencies", report, map[string]any{"summary": summary}, logWriter)
	return report
}

func parseDependencyHealthOutput(raw []byte) (*healthv1.DependencyHealthResponse, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return nil, errors.New("scenario-dependency-analyzer produced empty output")
	}
	var report healthv1.DependencyHealthResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &report); err != nil {
		return nil, err
	}
	if err := requireProtoProviderAssessment("scenario-dependency-analyzer", "dependencies", report.GetAssessment()); err != nil {
		return nil, err
	}
	return &report, nil
}

func translateDependencyHealthReport(report *healthv1.DependencyHealthResponse, exitCode int) *dependencyHealthRunResult {
	out := &dependencyHealthRunResult{}
	out.Summary = dependencySummary(report)
	failureCount := 0
	for _, section := range report.GetSections() {
		out.Observations = append(out.Observations, shared.NewInfoObservation(formatDependencyHealthSection(section)))
	}
	for _, degraded := range report.GetDegradedDependencies() {
		out.Observations = append(out.Observations, shared.NewWarningObservation(formatDependencyHealthDegraded(degraded)))
	}
	for _, finding := range report.GetFindings() {
		msg := formatDependencyHealthFinding(finding)
		switch normalizeFindingSeverity(finding.GetSeverity()) {
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
		out.Error = fmt.Errorf("%d dependency health finding(s) at ERROR severity", failureCount)
		out.Remediation = "Run `scenario-dependency-analyzer health " + report.GetScenario() + " --json` for full dependency health details."
	case exitCode != 0:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = fmt.Errorf("scenario-dependency-analyzer health exited with code %d but reported no errors", exitCode)
		out.Remediation = "File a scenario-dependency-analyzer issue; dependency health should emit ERROR findings when it exits non-zero."
	default:
		out.Success = true
	}
	return out
}

func dependencySummary(report *healthv1.DependencyHealthResponse) DependencySummary {
	if report == nil {
		return DependencySummary{}
	}
	s := report.GetSummary()
	out := DependencySummary{
		Scenario:             report.GetScenario(),
		Passed:               report.GetPassed(),
		Sections:             int(s.GetSections()),
		Surfaces:             int(s.GetSurfaces()),
		Findings:             int(s.GetFindings()),
		Errors:               int(s.GetErrors()),
		Warnings:             int(s.GetWarnings()),
		Infos:                int(s.GetInfos()),
		DegradedDependencies: int(s.GetDegradedDependencies()),
	}
	if gov := report.GetGovernanceSummary(); gov != nil {
		out.GovernanceStatus = gov.GetStatus()
	}
	if policy := report.GetPolicySummary(); policy != nil {
		out.PolicyStatus = policy.GetStatus()
	}
	out.LocalCurrentLevel, out.LocalNextLevel = protoLocalMaturitySummary(report.GetAssessment())
	return out
}

func dependencyHealthArchFindings(scenario string, report *healthv1.DependencyHealthResponse) []*architecturev1.ArchitectureFinding {
	if report == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(report.GetFindings()))
	for _, finding := range report.GetFindings() {
		targetScenario := strings.TrimSpace(report.GetScenario())
		if targetScenario == "" {
			targetScenario = scenario
		}
		code := firstNonEmptyString(finding.GetRuleId(), finding.GetId(), "dependency.health")
		out = append(out, newFinding(
			targetScenario,
			architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
			code,
			finding.GetSeverity(),
			dependencyHealthMessage(finding),
			finding.GetRemediation(),
			nonEmptyLocations(finding.GetFilePath()),
			[]string{"dependencies"},
		))
	}
	return out
}

func dependencyHealthMessage(f *healthv1.DependencyHealthFinding) string {
	if f == nil {
		return ""
	}
	msg := strings.TrimSpace(f.GetTitle())
	if desc := strings.TrimSpace(f.GetDescription()); desc != "" && desc != msg {
		if msg == "" {
			msg = desc
		} else {
			msg += " — " + desc
		}
	}
	if domain := strings.TrimSpace(f.GetSourceDomain()); domain != "" {
		msg = fmt.Sprintf("[%s] %s", domain, msg)
	}
	return strings.TrimSpace(msg)
}

func formatDependencyHealthFinding(f *healthv1.DependencyHealthFinding) string {
	if f == nil {
		return ""
	}
	parts := []string{firstNonEmptyString(f.GetRuleId(), f.GetId())}
	if title := strings.TrimSpace(f.GetTitle()); title != "" {
		parts = append(parts, title)
	}
	line := strings.Join(nonEmptyStrings(parts...), ": ")
	if loc := strings.TrimSpace(f.GetFilePath()); loc != "" {
		line += fmt.Sprintf(" [%s]", loc)
	}
	if observed := strings.TrimSpace(f.GetObserved()); observed != "" {
		line += "\n    observed: " + observed
	}
	if expected := strings.TrimSpace(f.GetExpected()); expected != "" {
		line += "\n    expected: " + expected
	}
	if rem := strings.TrimSpace(f.GetRemediation()); rem != "" {
		line += "\n    remediation: " + rem
	}
	return strings.TrimSpace(line)
}

func formatDependencyHealthSection(s *healthv1.DependencyHealthSection) string {
	return strings.TrimSpace(fmt.Sprintf("%s: %s - %s", s.GetStatus(), s.GetTitle(), s.GetSummary()))
}

func formatDependencyHealthDegraded(d *healthv1.DegradedDependency) string {
	return strings.TrimSpace(fmt.Sprintf("degraded %s (%s): %s", d.GetDependency(), d.GetDomain(), d.GetReason()))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

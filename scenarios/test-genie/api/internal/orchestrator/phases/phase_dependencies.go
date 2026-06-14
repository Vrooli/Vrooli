package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"

	"test-genie/internal/dependencies"
	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// cliClient is the shared typed Vrooli CLI client used to read live resource
// health for the dependency phase.
var cliClient = vroolicli.New()

type dependencyDriftReport struct {
	Findings []dependencyDriftFinding `json:"findings"`
}

type dependencyDriftFinding struct {
	Scenario   string                    `json:"scenario"`
	Dependency string                    `json:"dependency"`
	Kind       string                    `json:"kind"`
	Severity   string                    `json:"severity"`
	Message    string                    `json:"message"`
	Evidence   []dependencyDriftEvidence `json:"evidence"`
}

type dependencyDriftEvidence struct {
	Source     string `json:"source"`
	ImportPath string `json:"import_path"`
	FromFile   string `json:"from_file"`
	ToFile     string `json:"to_file"`
	Path       string `json:"path"`
	Analyzer   string `json:"analyzer"`
}

var runDependencyDrift = func(ctx context.Context, scenario string) ([]byte, int, error) {
	bin, lookErr := exec.LookPath("scenario-dependency-analyzer")
	if lookErr != nil {
		return nil, 0, fmt.Errorf("locate scenario-dependency-analyzer CLI: %w (install via `vrooli scenario start scenario-dependency-analyzer`)", lookErr)
	}
	cmd := exec.CommandContext(ctx, bin, "drift", scenario, "--json")
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

// runDependenciesPhase validates runtime/tool requirements using the dependencies package.
// This includes baseline commands, language runtimes, package managers, and resources.
func runDependenciesPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string
	var archFindings []*architecturev1.ArchitectureFinding

	report := RunPhase(ctx, logWriter, "dependencies",
		func() (*dependencies.RunResult, error) {
			config := dependencies.Config{
				ScenarioDir:                      env.ScenarioDir,
				ScenarioName:                     env.ScenarioName,
				AppRoot:                          env.AppRoot,
				CommandLookup:                    commandLookup,
				SkipResourceHealthWhenNoRequired: env.Mapping.HasLogicalPlacement(),
				ScenarioStatusFetcher:            cliClient,
				ResourceStatusFetcher:            cliClient,
			}

			opts := []dependencies.Option{
				dependencies.WithLogger(logWriter),
			}

			runner := dependencies.New(config, opts...)
			return runner.Run(ctx), nil
		},
		func(r *dependencies.RunResult) PhaseResult[dependencies.Observation] {
			if r != nil {
				summary = fmt.Sprintf("%d checks", r.Summary.TotalChecks())
			}
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"",
				fmt.Sprintf("Dependency validation completed (%d checks)", r.Summary.TotalChecks()),
			)
		},
	)

	driftFindings, driftObs := collectDependencyDriftFindings(ctx, env.ScenarioName)
	archFindings = append(archFindings, driftFindings...)
	report.Observations = append(report.Observations, driftObs...)
	report.Findings = archFindings

	writePhasePointer(env, "dependencies", report, map[string]any{"summary": summary}, logWriter)
	return report
}

func collectDependencyDriftFindings(ctx context.Context, scenario string) ([]*architecturev1.ArchitectureFinding, []Observation) {
	stdout, exitCode, err := runDependencyDrift(ctx, scenario)
	if err != nil {
		return nil, []Observation{NewSkipObservation(fmt.Sprintf("dependency drift skipped: %v", err))}
	}
	if strings.TrimSpace(string(stdout)) == "" {
		return nil, []Observation{NewSkipObservation("dependency drift skipped: scenario-dependency-analyzer returned no report")}
	}
	report, parseErr := parseDependencyDriftOutput(stdout)
	if parseErr != nil {
		return nil, []Observation{NewWarningObservation(fmt.Sprintf("dependency drift parse failed (exit=%d): %v", exitCode, parseErr))}
	}
	findings := dependencyDriftArchFindings(scenario, report)
	if len(findings) == 0 {
		return nil, []Observation{NewSuccessObservation("dependency drift clean: declared scenario dependencies match import evidence")}
	}
	return findings, []Observation{NewWarningObservation(fmt.Sprintf("dependency drift findings: %d", len(findings)))}
}

func parseDependencyDriftOutput(raw []byte) (*dependencyDriftReport, error) {
	if strings.TrimSpace(string(raw)) == "" {
		return nil, fmt.Errorf("empty dependency drift report")
	}
	var report dependencyDriftReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func dependencyDriftArchFindings(scenario string, report *dependencyDriftReport) []*architecturev1.ArchitectureFinding {
	if report == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		targetScenario := strings.TrimSpace(finding.Scenario)
		if targetScenario == "" {
			targetScenario = scenario
		}
		codeKind := strings.TrimSpace(finding.Kind)
		if codeKind == "" {
			codeKind = "dependency-drift"
		}
		f := newFinding(
			targetScenario,
			architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
			"dependency."+codeKind,
			finding.Severity,
			dependencyDriftMessage(finding),
			dependencyDriftSuggestion(finding),
			nonEmptyLocations("scenarios/"+targetScenario+"/.vrooli/service.json"),
			[]string{"dependencies"},
		)
		f.Evidence = dependencyDriftEvidenceToProto(finding.Evidence)
		out = append(out, f)
	}
	return out
}

func dependencyDriftMessage(f dependencyDriftFinding) string {
	if msg := strings.TrimSpace(f.Message); msg != "" {
		return msg
	}
	return fmt.Sprintf("%s dependency drift: %s -> %s", strings.TrimSpace(f.Kind), strings.TrimSpace(f.Scenario), strings.TrimSpace(f.Dependency))
}

func dependencyDriftSuggestion(f dependencyDriftFinding) string {
	switch strings.TrimSpace(f.Kind) {
	case "undeclared-but-used":
		return "Declare the scenario dependency in .vrooli/service.json or remove the import-level usage."
	case "declared-without-import-evidence":
		return "Confirm the dependency is runtime-only, or remove the stale declaration if it is no longer used."
	default:
		return "Review declared scenario dependencies against the actual interface graph."
	}
}

func dependencyDriftEvidenceToProto(in []dependencyDriftEvidence) []*architecturev1.Evidence {
	out := make([]*architecturev1.Evidence, 0, len(in))
	for _, ev := range in {
		locator := firstNonEmptyString(ev.ImportPath, ev.Path, ev.FromFile, ev.ToFile)
		summary := strings.TrimSpace(ev.Source)
		if ev.Analyzer != "" {
			summary = strings.TrimSpace(summary + " via " + ev.Analyzer)
		}
		out = append(out, &architecturev1.Evidence{
			Kind:    firstNonEmptyString(ev.Source, "dependency_drift"),
			Summary: summary,
			Locator: locator,
		})
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

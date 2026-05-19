package lint

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/lint/execution"
	"test-genie/internal/shared"
)

// Runner orchestrates lint validation across discovered top-level components.
type Runner struct {
	config    Config
	settings  *Settings
	registry  *handlerRegistry
	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new lint validation runner.
func New(config Config, opts ...Option) *Runner {
	settings := config.Settings
	if settings == nil {
		settings = DefaultSettings()
	}
	if config.CommandRunner == nil {
		config.CommandRunner = execution.ProductionRunner{}
	}

	r := &Runner{
		config:    config,
		settings:  settings,
		registry:  newHandlerRegistry(config),
		logWriter: io.Discard,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// Run executes lint validation across discovered components.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	components, err := discoverComponents(r.config.ScenarioDir, r.settings)
	if err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	var (
		observations   []Observation
		componentRuns  []ComponentResult
		policyFindings []PolicyFinding
		summary        = LintSummary{ComponentsDiscovered: len(components)}
		failed         bool
	)

	shared.LogInfo(r.logWriter, "Starting lint validation for %s", r.config.ScenarioName)

	for _, component := range components {
		if component.IsRoot && len(component.CodeEvidence) == 0 {
			continue
		}

		override := r.settings.Components[component.Name]
		if !override.ComponentEnabled() {
			componentRuns = append(componentRuns, ComponentResult{
				Component:    component,
				Skipped:      true,
				SkipReason:   "disabled via lint.components override",
				Observations: []Observation{NewSkipObservation(fmt.Sprintf("%s: linting disabled via configuration", component.RelativePath))},
			})
			summary.ComponentsSkipped++
			continue
		}

		h, evidence, resolveErr := resolveHandler(component, r.settings, r.registry)
		if resolveErr != nil {
			failed = true
			summary.PolicyErrors++
			finding := PolicyFinding{
				Component: component.Name,
				Path:      component.RelativePath,
				Severity:  PolicySeverityError,
				Message:   resolveErr.Error(),
			}
			policyFindings = append(policyFindings, finding)
			componentRuns = append(componentRuns, ComponentResult{
				Component:      component,
				Matched:        false,
				Success:        false,
				PolicyFindings: []PolicyFinding{finding},
				Observations:   []Observation{observationForPolicyFinding(finding)},
			})
			continue
		}

		if h == nil {
			findings := evaluatePolicy(component, false, r.settings)
			componentRuns = append(componentRuns, ComponentResult{
				Component:      component,
				Matched:        false,
				Success:        len(findings) == 0,
				PolicyFindings: findings,
			})
			if len(findings) > 0 {
				summary.ComponentsUnmatched++
				for _, finding := range findings {
					policyFindings = append(policyFindings, finding)
					switch finding.Severity {
					case PolicySeverityWarning:
						summary.PolicyWarnings++
					case PolicySeverityError:
						summary.PolicyErrors++
						failed = true
					}
				}
			}
			continue
		}

		component.DetectionReason = evidence
		result := h.Run(ctx, component)
		result.HandlerID = h.ID()
		result.Strict = r.settings.Handlers[h.ID()].StrictForComponent(override)
		componentRuns = append(componentRuns, result)
		summary.ComponentsLinted++
		summary.TypeErrors += result.TypeErrors
		summary.LintWarnings += result.LintWarnings
		observations = append(observations, NewSectionObservation("🧪", fmt.Sprintf("Linting %s (%s)", component.RelativePath, h.ID())))
		observations = append(observations, result.Observations...)

		if result.Skipped {
			summary.ComponentsSkipped++
		}
		if result.TypeErrors > 0 {
			failed = true
		}
		if result.Strict && result.LintWarnings > 0 {
			failed = true
			summary.TypeErrors += result.LintWarnings
		}
	}

	for idx := range componentRuns {
		if len(componentRuns[idx].PolicyFindings) == 0 {
			continue
		}
		for _, finding := range componentRuns[idx].PolicyFindings {
			if finding.Severity == PolicySeverityIgnore {
				continue
			}
			obs := observationForPolicyFinding(finding)
			if obs.Message != "" {
				componentRuns[idx].Observations = append(componentRuns[idx].Observations, obs)
				observations = append(observations, obs)
			}
		}
	}

	if summary.ComponentsLinted == 0 && len(policyFindings) == 0 {
		observations = append(observations, NewInfoObservation("No lintable top-level components detected"))
	}

	success := !failed && summary.TypeErrors == 0 && summary.PolicyErrors == 0
	var remediation string
	var runErr error
	if !success {
		runErr = fmt.Errorf("lint validation failed")
		remediation = "Fix lint/type issues and configure lint handlers for unmatched components before proceeding."
	}

	if success {
		if summary.TotalIssues() > 0 {
			observations = append(observations, NewWarningObservation(
				fmt.Sprintf("Lint completed with %d issue(s) across %d component(s)", summary.TotalIssues(), summary.ComponentsLinted),
			))
		} else {
			observations = append(observations, NewSuccessObservation(
				fmt.Sprintf("Lint validation passed (%d component(s) linted)", summary.ComponentsLinted),
			))
		}
	} else {
		observations = append(observations, NewErrorObservation(
			fmt.Sprintf("Lint validation failed: %d type error(s), %d policy error(s)", summary.TypeErrors, summary.PolicyErrors),
		))
	}

	shared.LogInfo(r.logWriter, "Lint validation complete: %s", summary.String())

	return &RunResult{
		Success:        success,
		Error:          runErr,
		FailureClass:   failureClassForLint(summary, success),
		Remediation:    remediation,
		Observations:   observations,
		Summary:        summary,
		Components:     componentRuns,
		PolicyFindings: policyFindings,
	}
}

func failureClassForLint(summary LintSummary, success bool) FailureClass {
	if success {
		return FailureClassNone
	}
	if summary.TypeErrors > 0 || summary.PolicyErrors > 0 {
		return FailureClassMisconfiguration
	}
	return FailureClassSystem
}

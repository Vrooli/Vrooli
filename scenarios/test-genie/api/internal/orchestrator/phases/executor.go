package phases

import (
	"context"
	"fmt"
	"io"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/metrics"
)

// seam: StandardRunResult is the contract native phase runners satisfy so the
// orchestrator can assemble reports, summaries, and phase pointers without
// per-phase extraction boilerplate.
type StandardRunResult interface {
	Succeeded() bool
	Err() error
	Failure() shared.FailureClass
	RemediationText() string
	ObservationList() []shared.Observation
	SummaryText() string
}

type nativePhaseConfig struct {
	summaryIcon    string
	summaryMessage func(Name, string) string
	onReport       func(*RunReport, StandardRunResult)
}

// NativePhaseOption customizes RunNativePhase for phase-specific report hooks.
type NativePhaseOption func(*nativePhaseConfig)

// WithNativePhaseSummaryIcon changes the success-summary observation icon.
func WithNativePhaseSummaryIcon(icon string) NativePhaseOption {
	return func(cfg *nativePhaseConfig) {
		cfg.summaryIcon = icon
	}
}

// WithNativePhaseSummaryMessage changes the success-summary observation text.
func WithNativePhaseSummaryMessage(format func(Name, string) string) NativePhaseOption {
	return func(cfg *nativePhaseConfig) {
		cfg.summaryMessage = format
	}
}

// WithNativePhaseReportHook lets a native phase attach typed findings before
// the phase pointer is written.
func WithNativePhaseReportHook(hook func(*RunReport, StandardRunResult)) NativePhaseOption {
	return func(cfg *nativePhaseConfig) {
		cfg.onReport = hook
	}
}

// RunNativePhase executes the native phase pattern: optional expectation
// loading, runner execution, observation conversion, summary emission, and
// phase-pointer persistence.
func RunNativePhase[TExpect any](
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	name Name,
	loadExpectations func(string) (TExpect, error),
	execute func(TExpect) (StandardRunResult, error),
	opts ...NativePhaseOption,
) RunReport {
	cfg := nativePhaseConfig{
		summaryIcon:    "✅",
		summaryMessage: defaultNativePhaseSummary,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	// seam: every native phase that flows through RunNativePhase is measured at
	// this single chokepoint, mirroring the delegated path's report.Metrics wiring
	// (phase_validationprovider.go). The collector's default baseline environment
	// (stdlib os/arch/num_cpu) is sufficient — no host-inventory shell per phase.
	m := metrics.Start()
	finish := func(report RunReport, result StandardRunResult, summary string) RunReport {
		report.Metrics = m.Stop()
		if cfg.onReport != nil {
			cfg.onReport(&report, result)
		}
		writePhasePointer(env, name.String(), report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	if report := CheckContext(ctx); report != nil {
		return finish(*report, nil, "")
	}

	var expectations TExpect
	if loadExpectations != nil {
		loadResult := LoadExpectationsOrFail(logWriter, env.ScenarioDir, loadExpectations, name.String())
		if loadResult.FailReport != nil {
			return finish(*loadResult.FailReport, nil, "")
		}
		expectations = loadResult.Expectations
	}

	result, err := execute(expectations)
	if err != nil {
		shared.LogError(logWriter, "%s execution failed: %v", name.String(), err)
		return finish(RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           fmt.Sprintf("Check %s configuration and try again.", name.String()),
		}, result, "")
	}

	summary := ""
	if result != nil {
		summary = result.SummaryText()
	}
	adapter := RunnerResultAdapter{
		Success:      result != nil && result.Succeeded(),
		Error:        nil,
		FailureClass: shared.FailureClassSystem,
	}
	if result != nil {
		adapter.Error = result.Err()
		adapter.FailureClass = result.Failure()
		adapter.Remediation = result.RemediationText()
		adapter.Observations = ConvertObservationsGeneric(result.ObservationList(), ExtractStandardObservation[shared.Observation])
	}

	report := ResultToReportWithSummary(
		adapter,
		cfg.summaryIcon,
		cfg.summaryMessage(name, summary),
		name.String()+" complete",
		logWriter,
	)
	return finish(report, result, summary)
}

func defaultNativePhaseSummary(name Name, summary string) string {
	return fmt.Sprintf("%s validation completed (%s)", titlePhaseName(name), summary)
}

func titlePhaseName(name Name) string {
	value := name.String()
	if value == "" {
		return "Phase"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

// RunPhase executes a standard phase pattern with observation conversion.
// This is the simplest helper for phases that don't need expectations.
//
// Usage:
//
//	return RunPhase(ctx, logWriter, "unit", func() (*unit.RunResult, error) {
//	    runner := unit.New(config, unit.WithLogger(logWriter))
//	    return runner.Run(ctx), nil
//	}, func(r *unit.RunResult) PhaseResult[unit.Observation] {
//	    return PhaseResult[unit.Observation]{...}
//	})
func RunPhase[TResult any, TObs StandardObservation](
	ctx context.Context,
	logWriter io.Writer,
	phaseName string,
	execute func() (*TResult, error),
	extract func(*TResult) PhaseResult[TObs],
) RunReport {
	// Check context first
	if report := CheckContext(ctx); report != nil {
		return *report
	}

	// Execute the runner
	result, err := execute()
	if err != nil {
		shared.LogError(logWriter, "%s execution failed: %v", phaseName, err)
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
			Remediation:           fmt.Sprintf("Check %s configuration and try again.", phaseName),
		}
	}

	// Extract and convert
	phaseResult := extract(result)
	observations := ConvertObservationsGeneric(phaseResult.Observations, ExtractStandardObservation[TObs])

	adapter := RunnerResultAdapter{
		Success:      phaseResult.Success,
		Error:        phaseResult.Error,
		FailureClass: phaseResult.FailureClass,
		Remediation:  phaseResult.Remediation,
		Observations: observations,
	}

	// Return with or without summary
	if phaseResult.Summary != "" {
		return ResultToReportWithSummary(
			adapter,
			phaseResult.SummaryIcon,
			phaseResult.Summary,
			phaseName+" complete",
			logWriter,
		)
	}
	return ResultToReport(adapter, phaseName+" complete", logWriter)
}

// PhaseResult holds the extracted data from a runner result.
// This provides a uniform structure for the extraction function.
type PhaseResult[TObs any] struct {
	Success      bool
	Error        error
	FailureClass shared.FailureClass
	Remediation  string
	Observations []TObs

	// Optional summary (if set, ResultToReportWithSummary is used)
	Summary     string
	SummaryIcon string
}

// ExtractSimple creates a PhaseResult from common RunResult fields.
// Use this for types that have the standard Success, Error, FailureClass, etc. fields.
func ExtractSimple[TObs any](
	success bool,
	err error,
	failureClass shared.FailureClass,
	remediation string,
	observations []TObs,
) PhaseResult[TObs] {
	return PhaseResult[TObs]{
		Success:      success,
		Error:        err,
		FailureClass: failureClass,
		Remediation:  remediation,
		Observations: observations,
	}
}

// ExtractWithSummary creates a PhaseResult with a summary observation.
func ExtractWithSummary[TObs any](
	success bool,
	err error,
	failureClass shared.FailureClass,
	remediation string,
	observations []TObs,
	summaryIcon string,
	summary string,
) PhaseResult[TObs] {
	return PhaseResult[TObs]{
		Success:      success,
		Error:        err,
		FailureClass: failureClass,
		Remediation:  remediation,
		Observations: observations,
		Summary:      summary,
		SummaryIcon:  summaryIcon,
	}
}

package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

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

// RunPhaseWithExpectations executes a phase that requires loading expectations.
//
// Usage:
//
//	return RunPhaseWithExpectations(ctx, env, logWriter, "structure",
//	    structure.LoadExpectations,
//	    func(exp *structure.Expectations) (*structure.RunResult, error) {
//	        runner := structure.New(config, structure.WithLogger(logWriter))
//	        return runner.Run(ctx), nil
//	    },
//	    func(r *structure.RunResult) PhaseResult[structure.Observation] {...},
//	)
func RunPhaseWithExpectations[TExpect any, TResult any, TObs StandardObservation](
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	phaseName string,
	loadExpectations func(string) (TExpect, error),
	execute func(expectations TExpect) (*TResult, error),
	extract func(*TResult) PhaseResult[TObs],
) RunReport {
	// Check context first
	if report := CheckContext(ctx); report != nil {
		return *report
	}

	// Load expectations
	loadResult := LoadExpectationsOrFail(logWriter, env.ScenarioDir, loadExpectations, phaseName)
	if loadResult.FailReport != nil {
		return *loadResult.FailReport
	}

	// Execute with expectations
	result, err := execute(loadResult.Expectations)
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

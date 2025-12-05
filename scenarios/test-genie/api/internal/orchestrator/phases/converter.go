package phases

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/shared"
)

// RunnerResultAdapter provides a common interface for all package-specific RunResult types.
// This allows generic handling of runner results without copying code across phases.
type RunnerResultAdapter struct {
	Success      bool
	Error        error
	FailureClass shared.FailureClass
	Remediation  string
	Observations []Observation
}

// ResultToReport converts a RunnerResultAdapter to a RunReport.
// This consolidates the common pattern used across all phases.
func ResultToReport(result RunnerResultAdapter, successMessage string, logWriter io.Writer) RunReport {
	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: string(shared.StandardizeFailureClass(result.FailureClass)),
			Remediation:           result.Remediation,
			Observations:          result.Observations,
		}
	}

	if successMessage != "" {
		shared.LogSuccess(logWriter, successMessage)
	}
	return RunReport{Observations: result.Observations}
}

// ResultToReportWithSummary converts a RunnerResultAdapter to a RunReport, adding a summary observation.
// Use this when the phase should append a final summary observation on success.
func ResultToReportWithSummary(result RunnerResultAdapter, summaryIcon, summaryText, successMessage string, logWriter io.Writer) RunReport {
	if !result.Success {
		return RunReport{
			Err:                   result.Error,
			FailureClassification: string(shared.StandardizeFailureClass(result.FailureClass)),
			Remediation:           result.Remediation,
			Observations:          result.Observations,
		}
	}

	observations := result.Observations
	if summaryText != "" {
		observations = append(observations, Observation{
			Icon: summaryIcon,
			Text: summaryText,
		})
	}

	if successMessage != "" {
		shared.LogSuccess(logWriter, successMessage)
	}
	return RunReport{Observations: observations}
}

// StandardObservation is an interface for observation types that follow the standard pattern.
// Most packages define Observation types with Type, Icon, and Message fields.
type StandardObservation interface {
	GetType() shared.ObservationType
	GetIcon() string
	GetMessage() string
}

// ExtractStandardObservation extracts observation data from any type implementing StandardObservation.
// This eliminates the need for package-specific extractors when the observation type follows the standard pattern.
func ExtractStandardObservation[T StandardObservation](o T) shared.ObservationData {
	return shared.ObservationData{
		Type:    o.GetType(),
		Icon:    o.GetIcon(),
		Message: o.GetMessage(),
	}
}

// ConvertObservationsGeneric converts a slice of observations from any package
// to phases.Observation using a type-specific extractor function.
func ConvertObservationsGeneric[T any](obs []T, extractor func(T) shared.ObservationData) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = convertObservationData(extractor(o))
	}
	return result
}

// convertObservationData converts shared.ObservationData to phases.Observation.
func convertObservationData(o shared.ObservationData) Observation {
	switch o.Type {
	case shared.ObservationSection:
		return NewSectionObservation(o.Icon, o.Message)
	case shared.ObservationSuccess:
		return NewSuccessObservation(o.Message)
	case shared.ObservationWarning:
		return NewWarningObservation(o.Message)
	case shared.ObservationError:
		return NewErrorObservation(o.Message)
	case shared.ObservationInfo:
		return NewInfoObservation(o.Message)
	case shared.ObservationSkip:
		return NewSkipObservation(o.Message)
	default:
		return NewObservation(o.Message)
	}
}

// CheckContext checks if the context has been cancelled and returns a failure report if so.
// Returns nil if context is OK.
func CheckContext(ctx context.Context) *RunReport {
	if err := ctx.Err(); err != nil {
		return &RunReport{Err: err, FailureClassification: FailureClassSystem}
	}
	return nil
}

// LoadExpectationsResult holds the result of loading expectations.
type LoadExpectationsResult[T any] struct {
	Expectations T
	FailReport   *RunReport
}

// LoadExpectationsOrFail attempts to load expectations and returns a failure report on error.
// The phaseName is used in error messages.
func LoadExpectationsOrFail[T any](
	logWriter io.Writer,
	scenarioDir string,
	loader func(string) (T, error),
	phaseName string,
) LoadExpectationsResult[T] {
	expectations, err := loader(scenarioDir)
	if err != nil {
		shared.LogError(logWriter, "Failed to load %s expectations: %v", phaseName, err)
		var zero T
		return LoadExpectationsResult[T]{
			Expectations: zero,
			FailReport: &RunReport{
				Err:                   err,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           fmt.Sprintf("Fix .vrooli/testing.json so %s settings can be parsed.", phaseName),
			},
		}
	}
	return LoadExpectationsResult[T]{Expectations: expectations}
}

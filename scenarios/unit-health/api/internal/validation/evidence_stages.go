package validation

import (
	"unit-health/internal/executor"
	"unit-health/internal/testquality"
)

// EvidenceStages describes observations, not four interchangeable pass verdicts.
// Review evidence has no producer in this run and must never be inferred.
type EvidenceStages struct {
	Configured  string
	Analyzed    string
	Executed    string
	Reviewed    string
	SourceRunID string
}

func (stages EvidenceStages) Normalized() EvidenceStages {
	known := func(value string, allowed ...string) string {
		for _, candidate := range allowed {
			if value == candidate {
				return value
			}
		}
		return "unknown"
	}
	stages.Configured = known(stages.Configured, "observed", "cached")
	stages.Analyzed = known(stages.Analyzed, "observed", "partial", "cached")
	stages.Executed = known(stages.Executed, "not_requested", "not_executed", "passed", "failed", "refused", "cached")
	stages.Reviewed = known(stages.Reviewed, "not_supplied")
	return stages
}

func summarizeEvidenceStages(response Response, executionRequested bool) *EvidenceStages {
	stages := &EvidenceStages{Configured: "unknown", Analyzed: "unknown", Executed: "not_requested", Reviewed: "not_supplied", SourceRunID: response.RunID}
	if len(response.ProjectionChecks) > 0 {
		stages.Configured = "observed"
	}
	if response.TestQuality != nil {
		report := response.TestQuality.Normalized()
		if report.UnavailableReason == "" || report.UnavailableReason == testquality.ReasonNone {
			assessed, unknown := 0, 0
			for _, coverage := range report.Coverage {
				assessed += coverage.Assessed
				unknown += coverage.Unknown
			}
			if assessed > 0 {
				stages.Analyzed = "observed"
			}
			if unknown > 0 || len(report.IncompleteReasons) > 0 {
				stages.Analyzed = "partial"
			}
		}
	}
	if !executionRequested {
		return stages
	}
	stages.Executed = "not_executed"
	if len(response.CommandResults) == 0 {
		return stages
	}
	stages.Executed = "passed"
	for _, result := range response.CommandResults {
		if result.FailureClass == executor.ClassUnsupported {
			stages.Executed = "refused"
			return stages
		}
		switch result.Status {
		case executor.StatusPassed:
		case executor.StatusFailed, executor.StatusTimeout, executor.StatusError:
			stages.Executed = "failed"
		default:
			stages.Executed = "unknown"
			return stages
		}
	}
	return stages
}

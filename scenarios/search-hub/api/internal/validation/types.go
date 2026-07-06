package validation

import (
	"fmt"
	"strings"
)

const (
	CodeConfigMissing           = "SEARCH_CONFIG_MISSING"
	CodeConfigInvalid           = "SEARCH_CONFIG_INVALID"
	CodeProviderInvalid         = "SEARCH_PROVIDER_INVALID"
	CodeProviderGroupMismatch   = "SEARCH_PROVIDER_GROUP_MISMATCH"
	CodeStatusEndpointMissing   = "SEARCH_STATUS_ENDPOINT_MISSING"
	CodeControlEndpointMissing  = "SEARCH_CONTROL_ENDPOINT_MISSING"
	CodeEvalCorpusMissing       = "SEARCH_EVAL_CORPUS_MISSING"
	CodeEvalCorpusInvalid       = "SEARCH_EVAL_CORPUS_INVALID"
	CodeEvalRunMissing          = "SEARCH_EVAL_RUN_MISSING"
	CodeEvalRunStale            = "SEARCH_EVAL_RUN_STALE"
	CodeEvalAssertFailed        = "SEARCH_EVAL_ASSERT_FAILED"
	CodeEvalLabelsStale         = "SEARCH_EVAL_LABELS_STALE"
	CodeEvalProviderUnavailable = "SEARCH_EVAL_PROVIDER_UNAVAILABLE"
	CodeTuningBudgetInvalid     = "SEARCH_TUNING_BUDGET_INVALID"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Finding struct {
	Code        string
	Severity    Severity
	Title       string
	Message     string
	Location    string
	Remediation string
}

type EvalEvidence struct {
	SuiteID       string
	LastRunID     string
	LastRunAt     string
	Freshness     string
	CorpusStatus  string
	FailureReason string
}

type Report struct {
	Scenario     string
	Path         string
	Findings     []Finding
	Summary      Summary
	EvalEvidence []EvalEvidence
}

type Summary struct {
	Providers int
	Errors    int
	Warnings  int
}

func (s Summary) Status() string {
	if s.Errors > 0 {
		return "failed"
	}
	return "passed"
}

func (s Summary) String() string {
	return fmt.Sprintf("providers=%d errors=%d warnings=%d", s.Providers, s.Errors, s.Warnings)
}

func severityToAssessment(severity Severity) string {
	switch severity {
	case SeverityError:
		return "SEVERITY_ERROR"
	case SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_WARNING"
	}
}

func normalizeScenario(value string) string {
	return strings.TrimSpace(value)
}

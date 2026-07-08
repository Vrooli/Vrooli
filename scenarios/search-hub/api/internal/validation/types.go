package validation

import (
	"fmt"
	"strings"
)

const (
	CodeConfigMissing           = "SEARCH_CONFIG_MISSING"
	CodeConfigInvalid           = "SEARCH_CONFIG_INVALID"
	CodeProviderInvalid         = "SEARCH_PROVIDER_INVALID"
	CodeProviderClassMissing    = "SEARCH_PROVIDER_CLASS_MISSING"
	CodeProviderGroupMismatch   = "SEARCH_PROVIDER_GROUP_MISMATCH"
	CodeStatusEndpointMissing   = "SEARCH_STATUS_ENDPOINT_MISSING"
	CodeReindexEndpointMissing  = "SEARCH_REINDEX_ENDPOINT_MISSING"
	CodeControlEndpointMissing  = "SEARCH_CONTROL_ENDPOINT_MISSING"
	CodeEvalCorpusMissing       = "SEARCH_EVAL_CORPUS_MISSING"
	CodeEvalCorpusInvalid       = "SEARCH_EVAL_CORPUS_INVALID"
	CodeEvalCorpusInadequate    = "SEARCH_EVAL_CORPUS_INADEQUATE"
	CodeEvalCorpusThin          = "SEARCH_EVAL_CORPUS_THIN"
	CodeEvalCorpusCoverage      = "SEARCH_EVAL_CORPUS_COVERAGE"
	CodeEvalRunMissing          = "SEARCH_EVAL_RUN_MISSING"
	CodeEvalRunStale            = "SEARCH_EVAL_RUN_STALE"
	CodeEvalRunOutdated         = "SEARCH_EVAL_RUN_OUTDATED"
	CodeEvalRecallBelowTarget   = "SEARCH_EVAL_RECALL_BELOW_TARGET"
	CodeEvalAssertFailed        = "SEARCH_EVAL_ASSERT_FAILED"
	CodeEvalLabelsStale         = "SEARCH_EVAL_LABELS_STALE"
	CodeEvalProviderUnavailable = "SEARCH_EVAL_PROVIDER_UNAVAILABLE"
	CodeTuningBudgetInvalid     = "SEARCH_TUNING_BUDGET_INVALID"
	CodePerfBudgetBreach        = "SEARCH_PERF_BUDGET_BREACH"
	CodePerfBudgetUnproven      = "SEARCH_PERF_BUDGET_UNPROVEN"
	CodePerfDegraded            = "SEARCH_PERF_DEGRADED"
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
	SuiteID            string
	LastRunID          string
	LastRunAt          string
	Freshness          string
	CorpusStatus       string
	FailureReason      string
	Recall             float64
	RecallTarget       float64
	GradeablePositives int
	MetCases           int
	BelowCases         int
	LatencyP95Ms       int32
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

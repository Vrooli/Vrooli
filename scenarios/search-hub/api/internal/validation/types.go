package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// StatusProbe is an optional live seam used by maturity validation. A zero
// timestamp is an honest "not reported" result; validation never synthesizes
// an index age from Search Hub's own observations.
type StatusProbe interface {
	ProbeIndexTimestamp(context.Context, *registryv1.ProviderDescriptor) (time.Time, error)
}

const (
	CodeConfigMissing             = "SEARCH_CONFIG_MISSING"
	CodeConfigInvalid             = "SEARCH_CONFIG_INVALID"
	CodeProviderDescriptorMissing = "SEARCH_PROVIDER_DESCRIPTOR_MISSING"
	// These evidence-floor findings keep a missing descriptor from allowing
	// unrelated capability ladders to default to their highest rung. They are
	// emitted only when Search Hub has no descriptor to inspect.
	CodeGovernanceEvidenceMissing  = "SEARCH_GOVERNANCE_EVIDENCE_MISSING"
	CodeEvalEvidenceMissing        = "SEARCH_EVAL_EVIDENCE_MISSING"
	CodeOperabilityEvidenceMissing = "SEARCH_OPERABILITY_EVIDENCE_MISSING"
	CodeProviderInvalid            = "SEARCH_PROVIDER_INVALID"
	CodeProviderClassMissing       = "SEARCH_PROVIDER_CLASS_MISSING"
	CodeProviderGroupMismatch      = "SEARCH_PROVIDER_GROUP_MISMATCH"
	CodeStatusEndpointMissing      = "SEARCH_STATUS_ENDPOINT_MISSING"
	CodeIndexAgeUnreported         = "SEARCH_INDEX_AGE_UNREPORTED"
	CodeReindexEndpointMissing     = "SEARCH_REINDEX_ENDPOINT_MISSING"
	CodeControlEndpointMissing     = "SEARCH_CONTROL_ENDPOINT_MISSING"
	CodeEvalCorpusMissing          = "SEARCH_EVAL_CORPUS_MISSING"
	CodeEvalCorpusInvalid          = "SEARCH_EVAL_CORPUS_INVALID"
	CodeEvalCorpusInadequate       = "SEARCH_EVAL_CORPUS_INADEQUATE"
	CodeEvalCorpusThin             = "SEARCH_EVAL_CORPUS_THIN"
	CodeEvalCorpusDegenerate       = "SEARCH_EVAL_CORPUS_DEGENERATE"
	CodeEvalCorpusCoverage         = "SEARCH_EVAL_CORPUS_COVERAGE"
	CodeEvalRunMissing             = "SEARCH_EVAL_RUN_MISSING"
	CodeEvalRunStale               = "SEARCH_EVAL_RUN_STALE"
	CodeEvalRunOutdated            = "SEARCH_EVAL_RUN_OUTDATED"
	CodeEvalRecallBelowTarget      = "SEARCH_EVAL_RECALL_BELOW_TARGET"
	CodeEvalAssertFailed           = "SEARCH_EVAL_ASSERT_FAILED"
	CodeEvalLabelsStale            = "SEARCH_EVAL_LABELS_STALE"
	CodeEvalProviderUnavailable    = "SEARCH_EVAL_PROVIDER_UNAVAILABLE"
	CodeEvalUnavailable            = "SEARCH_EVAL_UNAVAILABLE"
	CodeEvalSuiteOrphaned          = "SEARCH_EVAL_SUITE_ORPHANED"
	CodeTuningBudgetInvalid        = "SEARCH_TUNING_BUDGET_INVALID"
	CodePerfBudgetBreach           = "SEARCH_PERF_BUDGET_BREACH"
	CodePerfBudgetUnproven         = "SEARCH_PERF_BUDGET_UNPROVEN"
	CodePerfSamplesUnproven        = "SEARCH_PERF_SAMPLES_UNPROVEN"
	CodePerfDegraded               = "SEARCH_PERF_DEGRADED"
	CodeLiveDegraded               = "SEARCH_LIVE_DEGRADED"
	CodeLiveZeroYield              = "SEARCH_LIVE_ZERO_YIELD"
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

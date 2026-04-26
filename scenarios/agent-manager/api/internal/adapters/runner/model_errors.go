package runner

import (
	"regexp"
	"strings"

	"agent-manager/internal/domain"
)

// ModelErrorKind classifies the outcome of a runner invocation with respect to the
// model selection. The executor uses this to decide whether to walk to the next
// entry in the preset chain before surfacing a failure.
type ModelErrorKind int

const (
	// ModelErrorNone means the output does not look like a model-related failure.
	// The executor should treat the run outcome as it normally would.
	ModelErrorNone ModelErrorKind = iota
	// ModelErrorUnavailable means the runner rejected the requested model (deprecated,
	// unknown, invalid, or not available to this account). Another model from the
	// chain should be tried.
	ModelErrorUnavailable
	// ModelErrorTransient means a retryable condition (rate limit, network hiccup).
	// Callers may choose to wait, but chain walking is not appropriate.
	ModelErrorTransient
)

// modelErrorPatterns are per-runner regex patterns for ModelErrorUnavailable.
// Stored beside the runner adapters so a vendor phrasing change touches only one place.
var modelErrorPatterns = map[domain.RunnerType][]*regexp.Regexp{
	domain.RunnerTypeClaudeCode: {
		regexp.MustCompile(`(?i)unknown model`),
		regexp.MustCompile(`(?i)model not found`),
		regexp.MustCompile(`(?i)invalid model`),
		regexp.MustCompile(`(?i)model .* is not available`),
	},
	domain.RunnerTypeCodex: {
		regexp.MustCompile(`(?i)unknown model`),
		regexp.MustCompile(`(?i)model .* not found`),
		regexp.MustCompile(`(?i)invalid model`),
		regexp.MustCompile(`(?i)model .* (is )?(deprecated|retired|not available|no longer)`),
		regexp.MustCompile(`(?i)unsupported model`),
	},
	domain.RunnerTypeOpenCode: {
		regexp.MustCompile(`(?i)unknown model`),
		regexp.MustCompile(`(?i)model not found`),
		regexp.MustCompile(`(?i)invalid model`),
		regexp.MustCompile(`(?i)model .* is not available`),
	},
}

var transientPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)rate limit`),
	regexp.MustCompile(`(?i)temporarily unavailable`),
	regexp.MustCompile(`(?i)timed? out`),
	regexp.MustCompile(`(?i)connection (reset|refused)`),
}

// ClassifyModelError inspects the runner's error text and exit status and reports
// whether it indicates a model-availability failure. The classification is conservative:
// only strings that clearly indicate "the runner did not accept the model" return
// ModelErrorUnavailable. Everything else defaults to ModelErrorNone.
func ClassifyModelError(runnerType domain.RunnerType, stderr string, exitCode int) ModelErrorKind {
	text := strings.TrimSpace(stderr)
	if text == "" {
		return ModelErrorNone
	}

	for _, pattern := range transientPatterns {
		if pattern.MatchString(text) {
			return ModelErrorTransient
		}
	}

	patterns, ok := modelErrorPatterns[runnerType]
	if !ok {
		return ModelErrorNone
	}
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return ModelErrorUnavailable
		}
	}
	return ModelErrorNone
}

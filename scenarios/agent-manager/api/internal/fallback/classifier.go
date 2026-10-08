package fallback

import (
	"regexp"
	"strings"
)

// Classifier converts raw runner/probe failure signals into a typed
// *ClassifiedError. Each codec (claude, codex, opencode) implements its
// own Classifier that inspects native structured signals (HTTP status,
// exit code, JSON event fields) before falling back to the residual
// TextClassifier defined here.
//
// Implementations MUST return nil to indicate "no failure detected"
// (i.e. the result was successful or the signal was empty). Returning
// a non-nil ClassifiedError with ReasonUnknown is the explicit "I saw
// a failure but could not classify it" signal.
type Classifier interface {
	// Classify inspects the input and returns a typed error or nil when
	// the input does not represent a failure.
	Classify(in ClassifyInput) *ClassifiedError
}

// ClassifyInput is the union of signals a Classifier can consume.
// Codecs populate the fields they have; absent fields are zero values.
type ClassifyInput struct {
	// RunnerType is the runner whose output produced this signal. Used
	// by per-codec classifiers as a sanity check; the residual text
	// classifier does not require it.
	RunnerType string

	// Stderr is the captured stderr buffer (or representative slice).
	Stderr string

	// ExitCode is the runner process exit code (0 = success).
	ExitCode int

	// HTTPStatus is the HTTP status code from a structured codec event,
	// when the codec emitted one. 0 when not applicable.
	HTTPStatus int

	// StructuredCode is a codec-native error code (e.g. claude's
	// is_error type, codex's CodexError.Code). Free-form string;
	// per-codec classifiers interpret it; the residual text classifier
	// ignores it.
	StructuredCode string

	// Cause is the underlying Go error, when classification is being
	// driven from an error return rather than a structured event.
	Cause error
}

// TextClassifier is the residual Classifier that inspects stderr text
// patterns. It is the safety net for cases where codecs have no
// structured signal (e.g. a runner crashed before emitting JSON).
//
// It is intentionally narrower than the per-codec classifiers: it only
// emits Reasons that can be confidently identified from text alone.
// Codec-specific classifiers should run BEFORE this one, not instead of
// it.
type TextClassifier struct{}

// NewTextClassifier returns the singleton-shaped residual classifier.
func NewTextClassifier() *TextClassifier { return &TextClassifier{} }

// Patterns. Kept package-private so callers cannot accumulate ad-hoc
// regex spread across the codebase — the canonical text patterns live
// here only. Per-codec classifiers may add codec-specific patterns
// inside their own packages.
//
// Order matters: more-specific patterns must precede their generic
// supersets so the first match wins.
var (
	rateLimitPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)rate[\s-]?limit`),
		regexp.MustCompile(`(?i)429\s+too\s+many\s+requests`),
		regexp.MustCompile(`(?i)throttl(ed|ing)`),
	}

	authFailurePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)invalid\s+api\s+key`),
		regexp.MustCompile(`(?i)authentication\s+(failed|error)`),
		regexp.MustCompile(`(?i)unauthorized`),
		regexp.MustCompile(`(?i)\b401\b`),
		regexp.MustCompile(`(?i)\b403\b`),
	}

	quotaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)quota\s+(exceeded|exhausted)`),
		regexp.MustCompile(`(?i)insufficient\s+(credit|quota|balance)`),
		regexp.MustCompile(`(?i)billing`),
		regexp.MustCompile(`(?i)\b402\b`),
	}

	modelDeprecatedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)deprecated`),
		regexp.MustCompile(`(?i)retired`),
		regexp.MustCompile(`(?i)no\s+longer\s+(available|supported)`),
		regexp.MustCompile(`(?i)sunset`),
	}

	modelUnknownPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)unknown\s+model`),
		regexp.MustCompile(`(?i)model\s+not\s+found`),
		regexp.MustCompile(`(?i)model\s+\S+\s+(was\s+)?not\s+found`),
		regexp.MustCompile(`(?i)invalid\s+model`),
		regexp.MustCompile(`(?i)unsupported\s+model`),
	}

	modelUnavailablePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)model\s+.*\s+is\s+not\s+available`),
		regexp.MustCompile(`(?i)model\s+.*\s+(is\s+)?unavailable`),
	}

	contextLengthPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)context[\s_-]?length`),
		regexp.MustCompile(`(?i)max(imum)?\s+(context|tokens)`),
		regexp.MustCompile(`(?i)token\s+limit`),
		regexp.MustCompile(`(?i)too\s+many\s+tokens`),
	}

	networkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)connection\s+(reset|refused|aborted)`),
		regexp.MustCompile(`(?i)timed?\s+out`),
		regexp.MustCompile(`(?i)temporarily\s+unavailable`),
		regexp.MustCompile(`(?i)\b(502|503|504)\b`),
		regexp.MustCompile(`(?i)network\s+(error|unreachable)`),
	}

	binaryMissingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)command\s+not\s+found`),
		regexp.MustCompile(`(?i)no\s+such\s+file\s+or\s+directory`),
		regexp.MustCompile(`(?i)executable\s+file\s+not\s+found`),
	}

	invalidFlagPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)unknown\s+(flag|option|argument)`),
		regexp.MustCompile(`(?i)unrecognized\s+(flag|option)`),
	}

	sessionExpiredPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)session\s+expired`),
		regexp.MustCompile(`(?i)token\s+expired`),
	}

	sessionStateLostPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)session\s+(state\s+)?(lost|missing|truncat)`),
		regexp.MustCompile(`(?i)rollout\s+file\s+(missing|truncat|invalid)`),
	}
)

// Classify implements Classifier. Returns nil when stderr is empty (no
// signal of failure). For non-empty input it always returns non-nil:
// either a specific Reason or ClassifiedError with ReasonUnknown.
func (TextClassifier) Classify(in ClassifyInput) *ClassifiedError {
	text := strings.TrimSpace(in.Stderr)
	if text == "" {
		// Fall through to Cause-only classification when stderr is empty.
		if in.Cause != nil {
			text = in.Cause.Error()
		}
	}
	if text == "" {
		// No text to classify. If the caller observed a non-zero exit
		// code, we still saw a failure — surface it as ReasonUnknown so
		// callers don't silently treat it as success.
		if in.ExitCode != 0 {
			return classified(ReasonUnknown, "", in)
		}
		return nil
	}

	// Order: most-specific signals first. Rate limit and auth before
	// quota (overlapping wording). Model-specific before context-length
	// (a "context length" message is sometimes phrased as "model too
	// large"). Network last among recoverable signals.
	switch {
	case anyMatch(text, rateLimitPatterns):
		return classified(ReasonRateLimit, text, in)
	case anyMatch(text, authFailurePatterns):
		return classified(ReasonAuthFailure, text, in)
	case anyMatch(text, quotaPatterns):
		return classified(ReasonQuotaExhausted, text, in)
	case anyMatch(text, modelDeprecatedPatterns):
		return classified(ReasonModelDeprecated, text, in)
	case anyMatch(text, modelUnknownPatterns):
		return classified(ReasonModelUnknown, text, in)
	case anyMatch(text, modelUnavailablePatterns):
		return classified(ReasonModelUnavailable, text, in)
	case anyMatch(text, contextLengthPatterns):
		return classified(ReasonContextLengthExceeded, text, in)
	case anyMatch(text, networkPatterns):
		return classified(ReasonNetworkTransient, text, in)
	case anyMatch(text, binaryMissingPatterns):
		return classified(ReasonBinaryMissing, text, in)
	case anyMatch(text, invalidFlagPatterns):
		return classified(ReasonInvalidFlag, text, in)
	case anyMatch(text, sessionStateLostPatterns):
		return classified(ReasonSessionStateLost, text, in)
	case anyMatch(text, sessionExpiredPatterns):
		return classified(ReasonSessionExpired, text, in)
	}
	return classified(ReasonUnknown, text, in)
}

func anyMatch(text string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

func classified(r Reason, text string, in ClassifyInput) *ClassifiedError {
	return &ClassifiedError{
		Reason:     r,
		Message:    truncateForMessage(text),
		Cause:      in.Cause,
		HTTPStatus: in.HTTPStatus,
		ExitCode:   in.ExitCode,
	}
}

// truncateForMessage trims excessive stderr down to a single short line
// suitable for an event payload Message field. Keeps the first line and
// caps total length at 256 chars.
func truncateForMessage(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	const maxLen = 256
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

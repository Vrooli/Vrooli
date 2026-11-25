package safety

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Action is the reviewer's verdict for one image. allow means nothing to
// disclose; flag means sexual (the host marks it sensitive); refuse means the
// one non-configurable refusal.
type Action string

const (
	ActionAllow  Action = "allow"
	ActionFlag   Action = "flag"
	ActionRefuse Action = "refuse"
)

// ReasonCode is the machine-readable why. It is recorded on the job for both
// passes and refusals, so a refusal can be triaged without parsing prose.
type ReasonCode string

const (
	ReasonClean                ReasonCode = "clean"
	ReasonSexualAdult          ReasonCode = "sexual_adult"
	ReasonSexualMinorFace      ReasonCode = "sexual_minor_face"
	ReasonSexualUnassessable   ReasonCode = "sexual_unassessable"
	ReasonReviewerUnavailable  ReasonCode = "reviewer_unavailable"
	ReasonReviewerUnconfigured ReasonCode = "reviewer_unconfigured"
	ReasonWeightsMissing       ReasonCode = "weights_missing"
	ReasonReviewerInvalid      ReasonCode = "reviewer_invalid_output"
	ReasonConsentRequired      ReasonCode = "consent_required"
)

// ReviewError is a technical failure of the reviewer (timeout, crash, bad
// output). It is not a verdict: the boundary fails closed, but the failure does
// not count against the model and is reported as unavailable rather than
// forbidden.
type ReviewError struct {
	Kind  ReasonCode
	Cause error
}

func (e *ReviewError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return string(e.Kind)
	}
	return string(e.Kind) + ": " + e.Cause.Error()
}

func (e *ReviewError) Unwrap() error { return e.Cause }

// IsReviewUnavailable reports whether err is a technical reviewer failure.
func IsReviewUnavailable(err error) bool {
	var reviewErr *ReviewError
	return errors.As(err, &reviewErr)
}

// ReviewHealth is the runtime state of the configured reviewer. It is reported
// by discovery so a deployment cannot claim a boundary that is not running.
type ReviewHealth struct {
	Configured   bool
	Available    bool
	Detail       string
	Reviews      int64
	Refusals     int64
	Errors       int64
	LastReviewAt time.Time
}

// Summary is a one-line health description for discovery/logs.
func (h ReviewHealth) Summary() string {
	switch {
	case !h.Configured:
		return "reviewer: not configured"
	case !h.Available:
		return "reviewer: unavailable (" + h.Detail + ")"
	case h.Reviews == 0:
		return "reviewer: configured (no review has run yet)"
	default:
		return fmt.Sprintf("reviewer: available (%d reviews, %d refusals, %d errors%s)",
			h.Reviews, h.Refusals, h.Errors, lastReviewSuffix(h.LastReviewAt))
	}
}

func lastReviewSuffix(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return ", last " + t.UTC().Format(time.RFC3339)
}

// healthTracker is the shared, concurrency-safe health/counter state used by the
// command reviewer and the second-opinion wrapper.
type healthTracker struct {
	mu           sync.Mutex
	configured   bool
	available    bool
	detail       string
	reviews      int64
	refusals     int64
	errors       int64
	lastReviewAt time.Time
}

func (t *healthTracker) setConfigured(detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.configured = true
	t.available = true
	t.detail = detail
}

func (t *healthTracker) setUnavailable(detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.configured = true
	t.available = false
	t.detail = detail
}

func (t *healthTracker) record(err error, refused bool, at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reviews++
	if err != nil {
		t.errors++
		t.available = false
		t.detail = firstLine(err.Error())
	} else {
		t.available = true
		t.detail = ""
	}
	if refused {
		t.refusals++
	}
	t.lastReviewAt = at
}

func (t *healthTracker) snapshot() ReviewHealth {
	t.mu.Lock()
	defer t.mu.Unlock()
	return ReviewHealth{
		Configured: t.configured, Available: t.available, Detail: t.detail,
		Reviews: t.reviews, Refusals: t.refusals, Errors: t.errors, LastReviewAt: t.lastReviewAt,
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

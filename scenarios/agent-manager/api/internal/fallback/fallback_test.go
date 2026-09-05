package fallback_test

import (
	"errors"
	"testing"

	"agent-manager/internal/fallback"
)

// TestReasonRecoveryActionExhaustive enforces the contract that every
// Reason in AllReasons() has a recovery-action mapping. Adding a Reason
// without updating reasonRecovery is a CI failure here.
func TestReasonRecoveryActionExhaustive(t *testing.T) {
	for _, r := range fallback.AllReasons() {
		got := fallback.Recovery(r)
		if got == "" {
			t.Errorf("reason %q has empty RecoveryAction (missing from reasonRecovery map)", r)
			continue
		}
		// Sanity-check the action is a defined constant.
		switch got {
		case fallback.RecoveryRetryImmediate,
			fallback.RecoveryRetryBackoff,
			fallback.RecoveryFallbackToNext,
			fallback.RecoveryAbort,
			fallback.RecoveryEscalateOperator:
		default:
			t.Errorf("reason %q maps to unknown RecoveryAction %q", r, got)
		}
	}
}

// TestRecoveryUnknownReasonSafelyAborts ensures that a Reason value
// constructed outside this package via casting (defensive: should never
// happen in practice) returns RecoveryAbort, not an empty action.
func TestRecoveryUnknownReasonSafelyAborts(t *testing.T) {
	got := fallback.Recovery(fallback.Reason("not-a-real-reason"))
	if got != fallback.RecoveryAbort {
		t.Errorf("Recovery on unknown Reason = %q, want %q", got, fallback.RecoveryAbort)
	}
}

func TestReasonHelpers(t *testing.T) {
	tests := []struct {
		reason           fallback.Reason
		modelUnavailable bool
		transient        bool
	}{
		{fallback.ReasonModelDeprecated, true, false},
		{fallback.ReasonModelUnknown, true, false},
		{fallback.ReasonModelUnavailable, true, false},
		{fallback.ReasonContextLengthExceeded, true, false},
		{fallback.ReasonRateLimit, false, true},
		{fallback.ReasonNetworkTransient, false, true},
		{fallback.ReasonProbeTimeout, false, true},
		{fallback.ReasonAuthFailure, false, false},
		{fallback.ReasonQuotaExhausted, false, false},
		{fallback.ReasonBinaryMissing, false, false},
		{fallback.ReasonInvalidFlag, false, false},
		{fallback.ReasonSessionExpired, false, false},
		{fallback.ReasonSessionStateLost, false, false},
		{fallback.ReasonUnknown, false, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.reason), func(t *testing.T) {
			if got := tc.reason.IsModelUnavailable(); got != tc.modelUnavailable {
				t.Errorf("%q.IsModelUnavailable() = %v, want %v", tc.reason, got, tc.modelUnavailable)
			}
			if got := tc.reason.IsTransient(); got != tc.transient {
				t.Errorf("%q.IsTransient() = %v, want %v", tc.reason, got, tc.transient)
			}
		})
	}
}

func TestClassifiedError_ErrorAndUnwrap(t *testing.T) {
	cause := errors.New("boom")
	ce := fallback.New(fallback.ReasonRateLimit, "hit limit", cause)
	if ce.Error() != "rate_limit: hit limit" {
		t.Errorf("Error() = %q", ce.Error())
	}
	if !errors.Is(ce, cause) {
		t.Errorf("errors.Is should find cause through Unwrap")
	}
	if ce.Recovery() != fallback.RecoveryRetryBackoff {
		t.Errorf("Recovery() = %q", ce.Recovery())
	}
	if !ce.IsTransient() {
		t.Error("IsTransient() should be true for rate_limit")
	}
}

func TestClassifiedError_NilSafe(t *testing.T) {
	var ce *fallback.ClassifiedError
	if ce.Error() != "" {
		t.Error("nil.Error() should be empty")
	}
	if ce.IsTransient() {
		t.Error("nil.IsTransient() should be false")
	}
	if ce.IsModelUnavailable() {
		t.Error("nil.IsModelUnavailable() should be false")
	}
	if ce.Recovery() != fallback.RecoveryAbort {
		t.Error("nil.Recovery() should be RecoveryAbort")
	}
}

func TestAsClassified(t *testing.T) {
	original := fallback.New(fallback.ReasonAuthFailure, "401", nil)
	wrapped := errors.Join(errors.New("context"), original)
	got := fallback.AsClassified(wrapped)
	if got == nil {
		t.Fatal("AsClassified should unwrap")
	}
	if got.Reason != fallback.ReasonAuthFailure {
		t.Errorf("Reason = %q", got.Reason)
	}
	if fallback.AsClassified(nil) != nil {
		t.Error("AsClassified(nil) should be nil")
	}
	if fallback.AsClassified(errors.New("plain")) != nil {
		t.Error("AsClassified(plain) should be nil")
	}
}

func TestTextClassifier_EmptyReturnsNil(t *testing.T) {
	tc := fallback.NewTextClassifier()
	if got := tc.Classify(fallback.ClassifyInput{}); got != nil {
		t.Errorf("empty input should produce nil, got %+v", got)
	}
	if got := tc.Classify(fallback.ClassifyInput{Stderr: "   \n\t  "}); got != nil {
		t.Errorf("whitespace stderr should produce nil, got %+v", got)
	}
}

func TestTextClassifier_DispatchTable(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  fallback.Reason
	}{
		{"rate_limit phrasing", "rate limit exceeded", fallback.ReasonRateLimit},
		{"429 status", "HTTP 429 too many requests", fallback.ReasonRateLimit},
		{"throttled", "request throttled, retry later", fallback.ReasonRateLimit},
		{"invalid api key", "invalid api key supplied", fallback.ReasonAuthFailure},
		{"401", "got 401 unauthorized", fallback.ReasonAuthFailure},
		{"quota exhausted", "quota exceeded for org", fallback.ReasonQuotaExhausted},
		{"deprecated model", "model gpt-5-codex is deprecated", fallback.ReasonModelDeprecated},
		{"no longer available", "model x is no longer available", fallback.ReasonModelDeprecated},
		{"unknown model", "error: unknown model 'gpt-5-codex'", fallback.ReasonModelUnknown},
		{"model not found", "the model gpt-5-codex was not found", fallback.ReasonModelUnknown},
		{"invalid model", "invalid model identifier", fallback.ReasonModelUnknown},
		{"unsupported model", "unsupported model 'gpt-5.2-codex'", fallback.ReasonModelUnknown},
		{"context length", "context length exceeded by 500 tokens", fallback.ReasonContextLengthExceeded},
		{"max tokens", "exceeded maximum tokens for model", fallback.ReasonContextLengthExceeded},
		{"connection reset", "connection reset by peer", fallback.ReasonNetworkTransient},
		{"timed out", "request timed out", fallback.ReasonNetworkTransient},
		{"503", "HTTP 503 service unavailable", fallback.ReasonNetworkTransient},
		{"command not found", "claude: command not found", fallback.ReasonBinaryMissing},
		{"ENOENT", "exec: no such file or directory", fallback.ReasonBinaryMissing},
		{"unknown flag", "unknown flag --frobnitz", fallback.ReasonInvalidFlag},
		{"session expired", "session expired, please reauthenticate", fallback.ReasonSessionExpired},
		{"session state lost", "session state lost; rollout missing", fallback.ReasonSessionStateLost},
		{"rollout truncated", "rollout file truncated", fallback.ReasonSessionStateLost},
		{"unclassifiable", "the agent exited unexpectedly", fallback.ReasonUnknown},
	}

	tc := fallback.NewTextClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tc.Classify(fallback.ClassifyInput{Stderr: tt.input})
			if got == nil {
				t.Fatalf("expected non-nil for %q", tt.input)
			}
			if got.Reason != tt.want {
				t.Errorf("Classify(%q).Reason = %q, want %q", tt.input, got.Reason, tt.want)
			}
		})
	}
}

// TestTextClassifier_FallbackToCause verifies that when stderr is empty
// but Cause is set, classification runs against Cause.Error().
func TestTextClassifier_FallbackToCause(t *testing.T) {
	tc := fallback.NewTextClassifier()
	got := tc.Classify(fallback.ClassifyInput{
		Cause: errors.New("rate limit hit"),
	})
	if got == nil || got.Reason != fallback.ReasonRateLimit {
		t.Errorf("expected ReasonRateLimit from Cause, got %+v", got)
	}
}

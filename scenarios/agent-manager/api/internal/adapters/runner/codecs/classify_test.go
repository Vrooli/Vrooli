package codecs

import (
	"testing"

	"agent-manager/internal/fallback"
)

// Tests for Codec.Classify. Each codec is asserted on:
//   - Codec-specific shapes (rate-limit, session-expired, session-state-lost)
//     which override the residual TextClassifier.
//   - Pass-through to TextClassifier for shapes the codec doesn't own
//     (model unknown, network, auth, etc.).
//   - Nil sentinel ONLY when stderr is empty AND exitCode == 0.

func TestClaude_Classify(t *testing.T) {
	c := NewClaudeForTest()
	cases := []struct {
		name     string
		stderr   string
		exitCode int
		wantNil  bool
		want     fallback.Reason
	}{
		{name: "empty success", wantNil: true},
		{name: "empty failure exit code", exitCode: 1, want: fallback.ReasonUnknown},
		{
			name:   "claude rate-limit body",
			stderr: `Claude AI usage limit reached|1700000000`, want: fallback.ReasonRateLimit,
		},
		{name: "session-expired", stderr: "session abc not found", want: fallback.ReasonSessionExpired},
		{name: "model unknown delegated", stderr: "Unknown model 'sonnet-99'", want: fallback.ReasonModelUnknown},
		{name: "auth-failure delegated", stderr: "401 unauthorized", want: fallback.ReasonAuthFailure},
		{name: "network delegated", stderr: "connection reset by peer", want: fallback.ReasonNetworkTransient},
		{name: "unclassified", stderr: "the moon is full", want: fallback.ReasonUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Classify(tc.stderr, tc.exitCode)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("want nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("want %s, got nil", tc.want)
			}
			if got.Reason != tc.want {
				t.Fatalf("want %s, got %s", tc.want, got.Reason)
			}
		})
	}
}

func TestCodex_Classify(t *testing.T) {
	c := NewCodexForTest()
	cases := []struct {
		name     string
		stderr   string
		exitCode int
		wantNil  bool
		want     fallback.Reason
	}{
		{name: "empty success", wantNil: true},
		{name: "thread not found expired", stderr: "thread abc not found", want: fallback.ReasonSessionExpired},
		{
			name:   "rollout writer state-lost",
			stderr: "thread xyz not found: failed to record rollout items", want: fallback.ReasonSessionStateLost,
		},
		{
			name:   "rollout writer state-lost fn-name",
			stderr: "thread abc not found in record_rollout_items", want: fallback.ReasonSessionStateLost,
		},
		{
			name:   "model deprecated delegated",
			stderr: "model gpt-5-codex is deprecated", want: fallback.ReasonModelDeprecated,
		},
		{
			name:   "model unknown delegated",
			stderr: "unknown model 'gpt-5-codex'", want: fallback.ReasonModelUnknown,
		},
		{name: "rate limit delegated", stderr: "rate limit exceeded", want: fallback.ReasonRateLimit},
		{name: "unclassified", stderr: "the agent exited unexpectedly", want: fallback.ReasonUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Classify(tc.stderr, tc.exitCode)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("want nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("want %s, got nil", tc.want)
			}
			if got.Reason != tc.want {
				t.Fatalf("want %s, got %s", tc.want, got.Reason)
			}
		})
	}
}

func TestOpenCode_Classify(t *testing.T) {
	c := NewOpenCodeForTest()
	cases := []struct {
		name     string
		stderr   string
		exitCode int
		wantNil  bool
		want     fallback.Reason
	}{
		{name: "empty success", wantNil: true},
		{name: "session not found", stderr: "session abc not found", want: fallback.ReasonSessionExpired},
		{name: "session expired", stderr: "session expired", want: fallback.ReasonSessionExpired},
		{name: "session invalid", stderr: "session is invalid", want: fallback.ReasonSessionExpired},
		{name: "model invalid delegated", stderr: "invalid model 'claude-bogus'", want: fallback.ReasonModelUnknown},
		{name: "unrelated", stderr: "something unrelated went wrong", want: fallback.ReasonUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Classify(tc.stderr, tc.exitCode)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("want nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("want %s, got nil", tc.want)
			}
			if got.Reason != tc.want {
				t.Fatalf("want %s, got %s", tc.want, got.Reason)
			}
		})
	}
}

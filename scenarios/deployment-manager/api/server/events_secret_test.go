package server

import (
	"errors"
	"testing"
)

func TestResolveEventsWebhookSecretPrefersEnvOverride(t *testing.T) {
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", "  explicit-secret  ")
	got, err := resolveEventsWebhookSecret()
	if err != nil {
		t.Fatalf("resolveEventsWebhookSecret() error = %v", err)
	}
	if got != "explicit-secret" {
		t.Fatalf("resolveEventsWebhookSecret() = %q, want the trimmed env override", got)
	}
}

func TestShouldMountDecisionWebhook(t *testing.T) {
	cases := []struct {
		name   string
		secret string
		err    error
		want   bool
	}{
		{name: "resolved secret mounts", secret: "a-real-secret", err: nil, want: true},
		{name: "resolver error does not mount", secret: "", err: errors.New("credential authority unavailable"), want: false},
		{name: "resolver error with stale secret does not mount", secret: "leftover", err: errors.New("boom"), want: false},
		{name: "empty secret does not mount", secret: "", err: nil, want: false},
		{name: "whitespace secret does not mount", secret: "   ", err: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldMountDecisionWebhook(tc.secret, tc.err); got != tc.want {
				t.Fatalf("shouldMountDecisionWebhook(%q, %v) = %v, want %v", tc.secret, tc.err, got, tc.want)
			}
		})
	}
}

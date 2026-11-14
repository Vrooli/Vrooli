package conversations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestParseOptionsKeepsLabelsAndDefaultsThemToKeys(t *testing.T) {
	options, err := parseOptions([]string{"close=Close the goal", "re-aim"})
	require.NoError(t, err)
	require.Equal(t, []askOption{{Key: "close", Label: "Close the goal"}, {Key: "re-aim", Label: "re-aim"}}, options)
}

func TestParseOptionsRejectsTooFewOrRepeatedKeys(t *testing.T) {
	_, err := parseOptions([]string{"approve"})
	require.Error(t, err)
	_, err = parseOptions([]string{"approve", "approve=Yes"})
	require.Error(t, err)
	_, err = parseOptions([]string{"=label", "reject"})
	require.Error(t, err)
}

func TestParseDeadlineAcceptsTimeOrDuration(t *testing.T) {
	now := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	got, err := parseDeadline("24h", now)
	require.NoError(t, err)
	require.Equal(t, "2026-10-08T12:00:00Z", got)
	got, err = parseDeadline("2026-10-09T08:30:00+02:00", now)
	require.NoError(t, err)
	require.Equal(t, "2026-10-09T06:30:00Z", got)
	_, err = parseDeadline("tomorrow", now)
	require.Error(t, err)
}

func TestWaitContextOutlivesDefaultTimeoutUntilDeadline(t *testing.T) {
	deadline := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	ctx, cancel := waitContext(deadline)
	defer cancel()
	got, ok := ctx.Deadline()
	want, _ := time.Parse(time.RFC3339, deadline)
	if !ok || !got.Equal(want.Add(waitGrace)) {
		t.Fatalf("wait context deadline = %v (set=%v), want %v", got, ok, want.Add(waitGrace))
	}
}

func TestParseReassertWithinNeedsASubjectKeyAndStaysInRange(t *testing.T) {
	got, err := parseReassertWithin("15m", "disk-pressure")
	require.NoError(t, err)
	require.Equal(t, int32(900), got)
	got, err = parseReassertWithin("", "")
	require.NoError(t, err)
	require.Zero(t, got, "no flag, no lapse")
	for name, value := range map[string]string{"words": "soon", "zero": "0s", "negative": "-5m", "fraction": "1500ms", "over thirty days": "721h"} {
		_, err = parseReassertWithin(value, "disk-pressure")
		require.Error(t, err, name)
	}
	_, err = parseReassertWithin("15m", "")
	require.Error(t, err, "a window without a subject key has nothing to apply to")
}

func TestAskOutcomeLineSaysWhatTheHubDid(t *testing.T) {
	for outcome, want := range map[string]string{
		"":           "Created ask a-1.",
		"created":    "Created ask a-1.",
		"reasserted": "Ask a-1 is already open with these terms; nothing new was sent.",
		"updated":    "Updated open ask a-1 in place; no new notification was sent.",
		"replaced":   "Created ask a-1; it replaced the open ask for this subject.",
		"suppressed": "Ask a-1 was dismissed with these exact terms; no new ask was created.",
		"returned":   "Ask a-1 was just answered; returning it instead of creating another.",
	} {
		require.Equal(t, want, askOutcomeLine(&v1.AskResponse{AskId: "a-1", Outcome: outcome}), outcome)
	}
}

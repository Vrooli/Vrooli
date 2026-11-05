package conversations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

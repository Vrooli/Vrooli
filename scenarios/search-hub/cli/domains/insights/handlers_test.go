package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWindowPreservesDayCompatibility(t *testing.T) {
	got, err := parseWindow("7")
	require.NoError(t, err)
	require.Equal(t, int32(7), got)

	got, err = parseWindow("0")
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestParseWindowAcceptsSubDayDurations(t *testing.T) {
	got, err := parseWindow("15m")
	require.NoError(t, err)
	require.Zero(t, got)

	got, err = parseWindow("2h")
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestParseWindowRejectsThinOrInvalidDurationSyntax(t *testing.T) {
	_, err := parseWindow("30s")
	require.Error(t, err)
	_, err = parseWindow("yesterday")
	require.Error(t, err)
}

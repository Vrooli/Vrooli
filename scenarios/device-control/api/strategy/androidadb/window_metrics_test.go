package androidadb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWindowMetricsReportsBarsIMEAndCutoutFromSourceFrames(t *testing.T) {
	dump := stringsJoinLines(
		"InsetsSource: {mType=statusBars, mFrame=[0,0][1080,84], mVisible=true}",
		"InsetsSource: {mType=navigationBars, mFrame=[0,2328][1080,2400], mVisible=true}",
		"InsetsSource: {mType=ime, mFrame=[0,1200][1080,2400], mVisible=true}",
		"mDisplayCutout=DisplayCutout{insets=Rect(0, 96 - 0, 0)}",
	)

	metrics, ok := parseWindowMetrics(dump, 1080, 2400)
	require.True(t, ok)
	require.True(t, metrics.StatusBarSourcePresent)
	require.True(t, metrics.StatusBarVisible)
	require.Equal(t, 84, metrics.StatusBarHeightPX)
	require.True(t, metrics.NavigationSourcePresent)
	require.Equal(t, 72, metrics.NavigationBarHeightPX)
	require.True(t, metrics.IMESourcePresent)
	require.True(t, metrics.IMEVisible)
	require.Equal(t, 1200, metrics.IMEHeightPX)
	require.Equal(t, 96, metrics.SafeInsetTopPX)
	require.Equal(t, 72, metrics.SafeInsetBottomPX)
	require.True(t, metrics.CutoutSourcePresent)
}

func TestParseWindowMetricsPreservesHiddenBarInsetsAndSideNavigation(t *testing.T) {
	dump := stringsJoinLines(
		"InsetsSource: {mType=statusBars, mFrame=[0,0][2400,84], mVisible=false}",
		"InsetsSource: {mType=navigationBars, mFrame=[0,0][84,1080], mVisible=true}",
		"InsetsSource: {mType=ime, mFrame=[0,1080][2400,1080], mVisible=false}",
		"mDisplayCutout=null",
	)

	metrics, ok := parseWindowMetrics(dump, 2400, 1080)
	require.True(t, ok)
	require.False(t, metrics.StatusBarVisible)
	require.Equal(t, 84, metrics.StatusBarHeightPX)
	require.Equal(t, 84, metrics.NavigationBarHeightPX)
	require.False(t, metrics.IMEVisible)
	require.Zero(t, metrics.IMEHeightPX)
	require.True(t, metrics.CutoutSourcePresent)
	require.Equal(t, 84, metrics.SafeInsetTopPX)
	require.Equal(t, 84, metrics.SafeInsetBottomPX)

	values := metrics.Values()
	require.Equal(t, false, values["android.status_bar.visible"])
	require.Equal(t, false, values["android.ime.visible"])
	require.Equal(t, 0, metrics.SafeInsetLeftPX)
	require.Equal(t, 0, metrics.SafeInsetRightPX)
}

func TestParseWindowMetricsFailsClosedWhenSourcesAreAbsent(t *testing.T) {
	metrics, ok := parseWindowMetrics("mShowingLockscreen=false", 1080, 2400)
	require.False(t, ok)
	require.Empty(t, metrics.Values())

	metrics, ok = parseWindowMetrics("InsetsSource: {mType=statusBars, mFrame=[0,0][1080,84], mVisible=true}", 1080, 2400)
	require.True(t, ok)
	require.NotContains(t, metrics.Values(), "android.ime.visible")
	require.NotContains(t, metrics.Values(), "android.safe_inset.left_px")
}

func stringsJoinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

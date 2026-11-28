package androidadb

import (
	"context"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestForegroundFactsPreserveWindowDisagreementAndPreferTopResumed(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell dumpsys activity activities":    []byte("mResumedActivity: ActivityRecord{a u0 old.home/.Home t1}\ntopResumedActivity=ActivityRecord{b u0 com.example/.Home t2}"),
		"adb -s tv shell dumpsys window":                 []byte("mCurrentFocus=Window{a u0 com.android.tv.settings/.Settings}\nprivate window text"),
		"adb -s tv shell settings get global boot_count": []byte("42"),
		"adb -s tv shell cat /proc/uptime":               []byte("15.20 30.00"),
	}}
	got, err := NewWithRunner(r, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "foreground-state", Package: "com.example"})
	require.NoError(t, err)
	require.Equal(t, "com.example/com.example.Home", got.ResumedActivity)
	require.Equal(t, "com.android.tv.settings/com.android.tv.settings.Settings", got.FocusedWindowActivity)
	require.Contains(t, got.Reason, "disagree")
	require.Empty(t, got.ForegroundPackage)
	require.Equal(t, "42", got.BootCount)
	require.Equal(t, 15.2, *got.UptimeSeconds)
	require.Empty(t, got.Diagnostic)
}

func TestForegroundFactsUnknownIsNotInventedSuccess(t *testing.T) {
	got, err := NewWithRunner(&scriptedRunner{}, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "foreground-state", Package: "com.example"})
	require.Error(t, err)
	require.Empty(t, got.ResumedActivity)
	require.Nil(t, got.UptimeSeconds)
}

func TestTopResumedActivityCannotBeReplacedByStaleWindowPointer(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell dumpsys activity activities": []byte("topResumedActivity=ActivityRecord{a u0 media.player/.Player t2}"),
		"adb -s tv shell dumpsys window":              []byte("mCurrentFocus=Window{b u0 com.android.tv.settings/.MainSettings}\nmShowingLockscreen=false"),
	}}
	state, err := NewWithRunner(r, "tv").ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "media.player", state.ForegroundPackage, "Android top-resumed activity must remain authoritative over an older independent focus pointer")
}

func TestSingleLegacyResumedActivityCannotBeReplacedByStaleWindowPointer(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell dumpsys activity activities": []byte("mResumedActivity: ActivityRecord{a u0 media.player/.Player t2}"),
		"adb -s tv shell dumpsys window":              []byte("mCurrentFocus=Window{b u0 com.android.tv.settings/.MainSettings}\nmShowingLockscreen=false"),
	}}
	state, err := NewWithRunner(r, "tv").ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "media.player", state.ForegroundPackage)
}

func TestAccessibilityBindingIsActiveUserScopedAndUnknownWithoutRecognizedDump(t *testing.T) {
	dump := "User state[\nattributes:{id=10,}\nBound services:{Service[id=com.example/.Service]}\nEnabled services:{}\n}]\nUser state[\nattributes:{id=0,}\nBound services:{}\nEnabled services:{com.example/.Service}\n}]"
	require.True(t, *boundAccessibilityService(dump, "10", "com.example/com.example.Service"))
	require.False(t, *boundAccessibilityService(dump, "0", "com.example/com.example.Service"))
	require.Nil(t, boundAccessibilityService(dump, "99", "com.example/com.example.Service"))
	require.Nil(t, boundAccessibilityService("unknown output", "0", "com.example/com.example.Service"))
}

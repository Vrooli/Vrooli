package androidadb

import (
	"context"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

const testStockHome = "com.google.android.apps.tv.launcherx/com.google.android.apps.tv.launcherx.home.HomeActivity"
const testDailyHome = "com.example.daily/com.example.code.HomeActivity"
const homeResolve = "adb -s tv shell cmd package resolve-activity --brief --user 0 -a android.intent.action.MAIN -c android.intent.category.HOME"

type homeChangeRunner struct {
	*scriptedRunner
	apply bool
}

func (r *homeChangeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	output, err := r.scriptedRunner.Run(ctx, name, args...)
	if err == nil && r.apply && strings.Contains(strings.Join(args, " "), "set-home-activity") {
		r.responses[homeResolve] = []byte(testDailyHome)
	}
	return output, err
}
func homeTestRunner() *homeChangeRunner {
	return &homeChangeRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell am get-current-user": []byte("0"),
		homeResolve:                           []byte("com.google.android.apps.tv.launcherx/.home.HomeActivity"),
		homeResolve + " -p com.example.daily": []byte(testDailyHome),
		"adb -s tv shell cmd package set-home-activity --user 0 " + testDailyHome: []byte("Success\n"),
	}}, apply: true}
}
func TestHomeSelectionVerifiesCandidatePriorUserAndActualReadback(t *testing.T) {
	runner := homeTestRunner()
	result, err := NewWithRunner(runner, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "home-set", Package: "com.example.daily", Value: "com.google.android.apps.tv.launcherx/.home.HomeActivity"})
	require.NoError(t, err)
	require.Equal(t, "home_verified", result.Outcome)
	require.Equal(t, testStockHome, result.PreviousHomeComponent)
	require.Equal(t, testDailyHome, result.HomeComponent)
	require.Equal(t, "0", result.AndroidUser)
	calls := strings.Join(runner.calls, "\n")
	require.Equal(t, 1, strings.Count(calls, "set-home-activity"))
	require.NotContains(t, calls, "disable")
	require.NotContains(t, calls, "settings put")
}
func TestHomeSelectionRejectsStaleOrIneligibleRequestsBeforeMutation(t *testing.T) {
	for _, name := range []string{"stale", "not-home", "invalid-package", "unknown-user"} {
		t.Run(name, func(t *testing.T) {
			runner := homeTestRunner()
			request := strategy.AppLifecycleRequest{Operation: "home-set", Package: "com.example.daily", Value: testStockHome}
			switch name {
			case "stale":
				request.Value = "com.other.home/.Home"
			case "not-home":
				delete(runner.responses, homeResolve+" -p com.example.daily")
			case "invalid-package":
				request.Package = "com.example.daily;reboot"
			case "unknown-user":
				runner.responses["adb -s tv shell am get-current-user"] = []byte("unknown")
			}
			_, err := NewWithRunner(runner, "tv").AppLifecycle(context.Background(), request)
			require.Error(t, err)
			require.NotContains(t, strings.Join(runner.calls, "\n"), "set-home-activity")
		})
	}
}
func TestAcceptedHomeChangeWithoutMatchingReadbackIsNotSuccessOrRetried(t *testing.T) {
	runner := homeTestRunner()
	runner.apply = false
	result, err := NewWithRunner(runner, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "home-set", Package: "com.example.daily", Value: testStockHome})
	require.ErrorContains(t, err, "reconcile")
	require.Equal(t, "accepted", result.Outcome)
	require.Equal(t, testStockHome, result.HomeComponent)
	require.Equal(t, 1, strings.Count(strings.Join(runner.calls, "\n"), "set-home-activity"))
}

func TestHomeFactsKeepRolePriorityAndEnabledSettingDistinct(t *testing.T) {
	data := "3 activities found:\n  priority=2 preferredOrder=0 isDefault=true\n  com.google.android.apps.tv.launcherx/.Home\n  priority=1 preferredOrder=0\n  com.google.android.tungsten.setupwraith/.RecoveryActivity\n  priority=0 preferredOrder=0\n  com.example.daily/com.example.code.HomeActivity\n"
	candidates := parseHomeCandidates(data)
	require.Len(t, candidates, 3)
	require.Equal(t, 2, *candidates[0].Priority)
	require.Equal(t, 1, *candidates[1].Priority)
	require.Equal(t, 0, *candidates[2].Priority)
	roles := "user_states={\n user_id=10\n roles={\n name=android.app.role.HOME\n holders=com.other.home\n }\n}\nuser_states={\n user_id=0\n roles={\n name=android.app.role.HOME\n holders=com.example.daily\n }\n}\n"
	holders, ok := parseHomeRoleHolders(roles, "0")
	require.True(t, ok)
	require.Equal(t, []string{"com.example.daily"}, holders)
	_, ok = parseHomeRoleHolders(roles, "99")
	require.False(t, ok)
	runner := homeTestRunner()
	runner.responses["adb -s tv shell cmd package query-activities --brief --user 0 -a android.intent.action.MAIN -c android.intent.category.HOME"] = []byte(data)
	runner.responses["adb -s tv shell dumpsys role"] = []byte(roles)
	runner.responses["adb -s tv shell dumpsys package com.google.android.apps.tv.launcherx"] = []byte("User 0: installed=true hidden=false enabled=0\nUser 10: installed=true enabled=3")
	result, err := NewWithRunner(runner, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "home-state", Package: "com.example.daily"})
	require.NoError(t, err)
	require.Equal(t, testStockHome, result.HomeComponent)
	require.Equal(t, []string{"com.example.daily"}, *result.HomeRoleHolders)
	require.Equal(t, 0, *result.HomeCandidates[0].EnabledSetting)
	for _, call := range runner.calls {
		require.NotContains(t, call, "set-home-activity")
		require.NotContains(t, call, "disable")
	}
}

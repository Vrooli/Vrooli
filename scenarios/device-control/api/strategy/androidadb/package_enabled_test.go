package androidadb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

type enabledPackageRunner struct {
	*scriptedRunner
	setting int
	apply   bool
}

func (r *enabledPackageRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := r.scriptedRunner.Run(ctx, name, args...)
	command := strings.Join(args, " ")
	if command == "-s tv shell dumpsys package com.example.home" {
		return []byte(fmt.Sprintf("User 0: installed=true hidden=false enabled=%d\nUser 10: installed=true enabled=1", r.setting)), nil
	}
	if r.apply {
		switch command {
		case "-s tv shell pm disable-user --user 0 com.example.home":
			r.setting = 3
		case "-s tv shell pm default-state --user 0 com.example.home":
			r.setting = 0
		case "-s tv shell pm enable --user 0 com.example.home":
			r.setting = 1
		}
	}
	return out, err
}
func enabledTestRunner() *enabledPackageRunner {
	return &enabledPackageRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{"adb -s tv shell am get-current-user": []byte("0"), "adb -s tv shell pm list packages --user 0 com.example.home": []byte("package:com.example.home")}}, apply: true}
}
func TestPackageEnabledChangeAndExactDefaultRestoreRequireReadback(t *testing.T) {
	runner := enabledTestRunner()
	adapter := NewWithRunner(runner, "tv")
	disabled, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-enabled-set", Package: "com.example.home", Value: "disabled-user", ExpectedValue: "default"})
	require.NoError(t, err)
	require.Equal(t, "enabled_setting_verified", disabled.Outcome)
	require.Equal(t, "default", disabled.PreviousEnabledSetting)
	require.Equal(t, "disabled-user", disabled.EnabledSetting)
	restored, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-enabled-set", Package: "com.example.home", Value: "default", ExpectedValue: "disabled-user"})
	require.NoError(t, err)
	require.Equal(t, "default", restored.EnabledSetting)
	require.Equal(t, 0, runner.setting, "restoration must restore DEFAULT, not invent explicit ENABLED")
	calls := strings.Join(runner.calls, "\n")
	require.Equal(t, 1, strings.Count(calls, "pm disable-user"))
	require.Equal(t, 1, strings.Count(calls, "pm default-state"))
	require.NotContains(t, calls, "pm enable --user")
}
func TestPackageEnabledMutationRefusesStaleUnknownAndUnrelatedPackages(t *testing.T) {
	for _, name := range []string{"stale", "unsupported-value", "prefix-collision", "shell-syntax"} {
		t.Run(name, func(t *testing.T) {
			runner := enabledTestRunner()
			request := strategy.AppLifecycleRequest{Operation: "package-enabled-set", Package: "com.example.home", Value: "disabled-user", ExpectedValue: "default"}
			switch name {
			case "stale":
				runner.setting = 1
			case "unsupported-value":
				request.Value = "bootloader"
			case "prefix-collision":
				runner.responses["adb -s tv shell pm list packages --user 0 com.example.home"] = []byte("package:com.example.homeextra")
			case "shell-syntax":
				request.Package = "com.example.home;reboot"
			}
			_, err := NewWithRunner(runner, "tv").AppLifecycle(context.Background(), request)
			require.Error(t, err)
			require.NotContains(t, strings.Join(runner.calls, "\n"), "pm disable-user")
		})
	}
}
func TestPackageEnabledAcknowledgementWithoutStateChangeCannotPassOrRetry(t *testing.T) {
	runner := enabledTestRunner()
	runner.apply = false
	result, err := NewWithRunner(runner, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-enabled-set", Package: "com.example.home", Value: "disabled-user", ExpectedValue: "default"})
	require.ErrorContains(t, err, "reconciliation")
	require.NotEqual(t, "enabled_setting_verified", result.Outcome)
	require.Empty(t, result.EnabledSetting)
	require.Equal(t, 1, strings.Count(strings.Join(runner.calls, "\n"), "pm disable-user"))
}
func TestLaunchRejectsShellSyntaxBeforeAnyCommand(t *testing.T) {
	runner := &scriptedRunner{}
	_, err := NewWithRunner(runner, "tv").launchPackage(context.Background(), "com.example.home;reboot")
	require.Error(t, err)
	require.Empty(t, runner.calls)
}

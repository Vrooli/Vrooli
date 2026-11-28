package androidadb

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestInstallReusesPackageDataForUpdateMigration(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "install", Value: "/tmp/hello-mobile.apk"}))
	require.Contains(t, runner.calls[0], "install -r /tmp/hello-mobile.apk")
}

type installSideEffectTimeoutRunner struct {
	*scriptedRunner
	packageName string
	installed   bool
	timeoutErr  error
}

func (r *installSideEffectTimeoutRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := strings.Join(append([]string{name}, args...), " ")
	switch {
	case strings.Contains(command, " install -r "):
		r.installed = true
		_, _ = r.scriptedRunner.Run(ctx, name, args...)
		return nil, r.timeoutErr
	case strings.HasSuffix(command, "shell pm list packages "+r.packageName):
		if r.installed {
			r.responses[command] = []byte("package:" + r.packageName + "\n")
		} else {
			r.responses[command] = nil
		}
	case strings.HasSuffix(command, "shell pm uninstall "+r.packageName):
		r.installed = false
	}
	return r.scriptedRunner.Run(ctx, name, args...)
}

func TestInstallTimeoutAfterSideEffectRemainsObservableAndCleanable(t *testing.T) {
	const packageName = "com.example.generated"
	timeoutErr := context.DeadlineExceeded
	runner := &installSideEffectTimeoutRunner{
		scriptedRunner: &scriptedRunner{responses: map[string][]byte{}},
		packageName:    packageName,
		timeoutErr:     timeoutErr,
	}
	adapter := NewWithRunner(runner, "serial-1")

	err := adapter.Actuate(context.Background(), strategy.Actuation{Action: "install", Value: "/tmp/generated.apk"})
	require.ErrorIs(t, err, timeoutErr)
	require.True(t, runner.installed, "the timed-out command may have installed the package")

	state, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: packageName})
	require.NoError(t, err)
	require.True(t, state.Installed, "package-state must expose the install side effect after timeout")

	cleanup, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "uninstall", Package: packageName})
	require.NoError(t, err)
	require.Equal(t, "accepted", cleanup.Outcome)

	state, err = adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: packageName})
	require.NoError(t, err)
	require.False(t, state.Installed, "exact-package cleanup must make absence observable")
	require.Contains(t, strings.Join(runner.calls, "\n"), "shell pm uninstall "+packageName)
}

func TestPackageStateReportsExpectedInstallationState(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pm list packages com.example.hello": []byte("package:com.example.hello\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "package-state", Package: "com.example.hello", Value: "present"}))
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "package-state", Package: "com.example.hello", Value: "absent"}), "still installed")

	runner.responses["adb -s serial-1 shell pm list packages com.example.hello"] = nil
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "package-state", Package: "com.example.hello", Value: "absent"}))
}

func TestAppLifecyclePackageStateVerifiesInstallation(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pm list packages org.smarttube.stable": []byte("package:org.smarttube.stable\n"),
		"adb -s serial-1 shell dumpsys package org.smarttube.stable":  []byte("Packages:\n  Package [org.smarttube.stable]\n    versionCode=2510 minSdk=23 targetSdk=35\n    versionName=25.10\n"),
		"adb -s serial-1 shell pm list packages com.example.absent":   []byte(""),
		"adb -s serial-1 shell pm list packages org.smart":            []byte("package:org.smarttube.stable\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")

	present, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: "org.smarttube.stable"})
	require.NoError(t, err)
	require.True(t, present.Installed)
	require.Equal(t, "25.10", present.Version)
	require.EqualValues(t, 2510, present.VersionCode)
	require.Equal(t, strategy.StatusAvailable, present.Status)

	absent, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: "com.example.absent"})
	require.NoError(t, err)
	require.False(t, absent.Installed)
	require.Equal(t, strategy.StatusAvailable, absent.Status)

	// A prefix collision must not report a shorter package present.
	prefix, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-state", Package: "org.smart"})
	require.NoError(t, err)
	require.False(t, prefix.Installed)
}

func TestAppLifecycleNetworkStateReadsAllRadioSettings(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell dumpsys wifi":                         []byte("Wi-Fi is enabled\n"),
		"adb -s serial-1 shell settings get global mobile_data":      []byte("1\n"),
		"adb -s serial-1 shell settings get global airplane_mode_on": []byte("0\n"),
	}}
	result, err := NewWithRunner(runner, "serial-1").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "network-state"})
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, result.Status)
	require.NotNil(t, result.NetworkState)
	require.True(t, *result.NetworkState.WifiEnabled)
	require.True(t, *result.NetworkState.MobileDataEnabled)
	require.False(t, *result.NetworkState.AirplaneModeEnabled)
	require.Len(t, runner.calls, 3)
}

func TestAppLifecycleNetworkStateFailsClosedWhenAnySettingIsUnknown(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell dumpsys wifi":                         []byte("Wi-Fi state is not reported\n"),
		"adb -s serial-1 shell settings get global mobile_data":      []byte("1\n"),
		"adb -s serial-1 shell settings get global airplane_mode_on": []byte("0\n"),
	}}
	result, err := NewWithRunner(runner, "serial-1").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "network-state"})
	require.ErrorContains(t, err, "Wi-Fi")
	require.Equal(t, strategy.StatusUnavailable, result.Status)
	require.Nil(t, result.NetworkState)
}

type fixtureSigningInspector struct{ digest string }

func (f fixtureSigningInspector) InspectSigningCertificate(context.Context, string) (string, error) {
	return f.digest, nil
}

type packageSigningRunner struct{ *scriptedRunner }

func (r *packageSigningRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if len(args) >= 4 && args[len(args)-3] == "pull" {
		if err := os.WriteFile(args[len(args)-1], []byte("installed APK fixture"), 0o600); err != nil {
			return nil, err
		}
	}
	return r.scriptedRunner.Run(ctx, name, args...)
}

func TestAppLifecyclePackageSigningStateReadsExactInstalledBaseAPK(t *testing.T) {
	runner := &packageSigningRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pm list packages com.example.portal": []byte("package:com.example.portal\n"),
		"adb -s serial-1 shell dumpsys package com.example.portal":  []byte("versionName=1.4.0\n"),
		"adb -s serial-1 shell pm path com.example.portal":          []byte("package:/data/app/com.example.portal-1/base.apk\npackage:/data/app/com.example.portal-1/split_config.arm64_v8a.apk\n"),
	}}}
	adapter := NewWithRunner(runner, "serial-1")
	adapter.signingInspector = fixtureSigningInspector{digest: strings.Repeat("a", 64)}
	result, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-signing-state", Package: "com.example.portal"})
	require.NoError(t, err)
	require.True(t, result.Installed)
	require.Equal(t, strings.Repeat("a", 64), result.SigningCertificateSHA256)
	serialized, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(serialized), `"installed":true`)
	require.Contains(t, string(serialized), `"signing_certificate_sha256":"`+strings.Repeat("a", 64)+`"`)
	foundPull := false
	for _, call := range runner.calls {
		if strings.HasPrefix(call, "adb -s serial-1 pull /data/app/com.example.portal-1/base.apk ") {
			foundPull = true
		}
	}
	require.True(t, foundPull, "signer inspection must pull the exact base.apk, not a split APK")
}

func TestAppLifecyclePackageSigningStateRejectsUnsafeInstalledPath(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pm list packages com.example.portal": []byte("package:com.example.portal\n"),
		"adb -s serial-1 shell dumpsys package com.example.portal":  []byte("versionName=1.4.0\n"),
		"adb -s serial-1 shell pm path com.example.portal":          []byte("package:/data/app/../../tmp/base.apk\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	adapter.signingInspector = fixtureSigningInspector{digest: strings.Repeat("a", 64)}
	_, err := adapter.AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "package-signing-state", Package: "com.example.portal"})
	require.ErrorContains(t, err, "safe package-manager path")
	for _, call := range runner.calls {
		require.NotContains(t, call, " pull ", "unsafe remote paths must not be pulled")
	}
}

func TestLaunchUsesTVActivityAndReportsObservedForeground(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER -p org.smarttube.stable": []byte("org.smarttube.stable/.MainActivity\n"),
		"adb -s serial-1 shell am start -W -n org.smarttube.stable/.MainActivity":                                                                                       []byte("Status: ok\nActivity: org.smarttube.stable/.MainActivity\n"),
		"adb -s serial-1 shell dumpsys window": []byte("mCurrentFocus=Window{abc u0 org.smarttube.stable/.MainActivity}"),
	}}
	result, err := NewWithRunner(runner, "serial-1").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "launch", Package: "org.smarttube.stable"})
	require.NoError(t, err)
	require.Equal(t, "foreground_observed", result.Outcome)
	require.Equal(t, "org.smarttube.stable", result.ForegroundPackage)
	require.Contains(t, result.Diagnostic, "Status: ok")
	require.False(t, result.ObservedAt.IsZero())
	require.NotContains(t, strings.Join(runner.calls, "\n"), "monkey")
}

func TestLaunchFallsBackToOrdinaryLauncherAndRejectsZeroExitError(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER -p com.android.tv.settings": []byte("No activity found"),
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LAUNCHER -p com.android.tv.settings":          []byte("com.android.tv.settings/.MainSettings"),
		"adb -s serial-1 shell am start -W -n com.android.tv.settings/.MainSettings":                                                                                       []byte("Error: Activity not started, permission denied"),
	}}
	err := NewWithRunner(runner, "serial-1").Actuate(context.Background(), strategy.Actuation{Action: "launch", Package: "com.android.tv.settings"})
	require.ErrorContains(t, err, "permission denied")
}

func TestLaunchDoesNotClaimForegroundFromAcceptanceAlone(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell cmd package resolve-activity --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER -p org.smarttube.stable": []byte("org.smarttube.stable/.MainActivity"),
		"adb -s serial-1 shell am start -W -n org.smarttube.stable/.MainActivity":                                                                                       []byte("Status: ok"),
		"adb -s serial-1 shell dumpsys window": []byte("mCurrentFocus=Window{abc u0 com.google.android.apps.tv.dreamx/.DreamActivity}"),
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := NewWithRunner(runner, "serial-1").AppLifecycle(ctx, strategy.AppLifecycleRequest{Operation: "launch", Package: "org.smarttube.stable"})
	require.NoError(t, err)
	require.Equal(t, "accepted", result.Outcome)
	require.Equal(t, "com.google.android.apps.tv.dreamx", result.ForegroundPackage)
}

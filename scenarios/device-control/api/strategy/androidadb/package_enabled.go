package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

var applicationIdentityPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
var enabledSettings = map[string]int{"default": 0, "enabled": 1, "disabled-user": 3}

func enabledSettingFromDump(data, user string) (int, bool) {
	match := regexp.MustCompile(`(?m)^\s*User\s+` + regexp.QuoteMeta(user) + `:\s+[^\n]*\benabled=([0-4])\b`).FindStringSubmatch(data)
	if len(match) != 2 {
		return 0, false
	}
	value, _ := strconv.Atoi(match[1])
	return value, true
}
func (a *Adapter) readPackageEnabled(ctx context.Context, user, pkg string) (int, error) {
	installed, err := a.runner.Run(ctx, "adb", a.args("shell", "pm", "list", "packages", "--user", user, pkg)...)
	if err != nil {
		return 0, err
	}
	found := false
	for _, line := range strings.Split(string(installed), "\n") {
		if strings.TrimSpace(line) == "package:"+pkg {
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("exact package is not installed for the selected user")
	}
	data, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "package", pkg)...)
	if err != nil || len(data) > 128*1024 {
		return 0, fmt.Errorf("package enabled state is unavailable")
	}
	state, ok := enabledSettingFromDump(string(data), user)
	if !ok {
		return 0, fmt.Errorf("package enabled state is unknown for the selected user")
	}
	return state, nil
}
func (a *Adapter) setPackageEnabled(ctx context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	result := strategy.AppLifecycleResult{Operation: "package-enabled-set", Package: request.Package, Status: strategy.StatusUnavailable}
	desired, desiredOK := enabledSettings[request.Value]
	expected, expectedOK := enabledSettings[request.ExpectedValue]
	if !desiredOK || !expectedOK || request.Permission != "" {
		return result, fmt.Errorf("package-enabled-set requires desired and expected settings: default, enabled, or disabled-user")
	}
	user, err := a.currentAndroidUser(ctx)
	if err != nil {
		return result, err
	}
	result.AndroidUser = user
	previous, err := a.readPackageEnabled(ctx, user, request.Package)
	if err != nil {
		return result, err
	}
	if previous != expected {
		return result, fmt.Errorf("package enabled state changed since inspection; reconcile before changing it")
	}
	result.PreviousEnabledSetting = request.ExpectedValue
	result.EnabledSetting = request.ExpectedValue
	if previous == desired {
		result.Status = strategy.StatusAvailable
		result.Outcome = "enabled_setting_verified"
		result.ObservedAt = time.Now().UTC()
		return result, nil
	}
	currentUser, err := a.currentAndroidUser(ctx)
	if err != nil || currentUser != user {
		return result, fmt.Errorf("Android user changed before package configuration")
	}
	command := map[int]string{0: "default-state", 1: "enable", 3: "disable-user"}[desired]
	output, commandErr := a.runner.Run(ctx, "adb", a.args("shell", "pm", command, "--user", user, request.Package)...)
	result.Diagnostic = boundedDiagnostic(output)
	result.Outcome = "accepted"
	if commandErr != nil {
		result.Outcome = "unknown"
	}
	observed, readErr := a.readPackageEnabled(ctx, user, request.Package)
	if readErr != nil || observed != desired {
		result.EnabledSetting = ""
		return result, fmt.Errorf("package configuration outcome requires reconciliation before retry")
	}
	if commandErr != nil {
		result.Reason = "command returned an error; desired setting independently observed"
	}
	result.Status = strategy.StatusAvailable
	result.EnabledSetting = request.Value
	result.Outcome = "enabled_setting_verified"
	result.ObservedAt = time.Now().UTC()
	return result, nil
}

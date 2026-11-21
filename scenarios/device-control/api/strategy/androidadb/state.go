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

func (a *Adapter) ReadState(ctx context.Context) (strategy.DeviceState, error) {
	// A stale ADB transport can remain listed as connected while a shell
	// command blocks indefinitely. Bound the whole observation so another
	// transport can be tried and caller cancellation remains effective.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	state := strategy.DeviceState{Unavailable: map[string]string{}}
	probe := func(field string, args ...string) string {
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil {
			state.Unavailable[field] = "adb " + strings.Join(a.args(args...), " ")
			return ""
		}
		return string(out)
	}
	state.WirelessDebugging = a.readWirelessDebuggingState(ctx)
	if state.WirelessDebugging == nil {
		state.Unavailable["wireless_debugging"] = "Android version and wireless debugging state could not be determined"
	}
	activity := probe("foreground_package", "shell", "dumpsys", "activity", "activities")
	state.ForegroundPackage = foregroundPackage(activity)
	if state.ForegroundPackage == "" && activity != "" {
		state.Unavailable["foreground_package"] = "adb " + strings.Join(a.args("shell", "dumpsys", "activity", "activities"), " ")
	}
	power := probe("screen_state", "shell", "dumpsys", "power")
	state.ScreenState = screenState(power)
	if state.ScreenState == "" && power != "" {
		state.Unavailable["screen_state"] = "adb " + strings.Join(a.args("shell", "dumpsys", "power"), " ")
	}
	state.LockState = a.readLockState(ctx, &state)
	input := probe("orientation", "shell", "dumpsys", "input")
	state.Orientation = orientation(input)
	if state.Orientation == "" && input != "" {
		state.Unavailable["orientation"] = "adb " + strings.Join(a.args("shell", "dumpsys", "input"), " ")
	}
	rotation := strings.TrimSpace(probe("rotation_setting", "shell", "settings", "get", "system", "user_rotation"))
	state.RotationSettings = map[string]string{}
	if rotation == "null" || rotation == "0" || rotation == "1" || rotation == "2" || rotation == "3" {
		state.RotationSettings["user_rotation"] = rotation
	}
	if state.Orientation == "" {
		state.Orientation = orientationValue(rotation)
		if state.Orientation != "" {
			delete(state.Unavailable, "orientation")
		}
	}
	autoRotate := probe("auto_rotate", "shell", "settings", "get", "system", "accelerometer_rotation")
	if value := strings.TrimSpace(autoRotate); value == "0" || value == "1" || value == "null" {
		state.RotationSettings["accelerometer_rotation"] = value
	}
	if strings.TrimSpace(autoRotate) == "0" || strings.TrimSpace(autoRotate) == "1" {
		state.AutoRotate = strings.TrimSpace(autoRotate) == "1"
		state.AutoRotateKnown = true
	}
	battery := probe("battery", "shell", "dumpsys", "battery")
	state.BatteryLevel, state.Charging = batteryState(battery)
	if state.BatteryLevel < 0 {
		if battery != "" {
			state.Unavailable["battery"] = "adb " + strings.Join(a.args("shell", "dumpsys", "battery"), " ")
		}
	} else {
		state.ChargingKnown = true
	}
	thermal := probe("thermal_status", "shell", "dumpsys", "thermalservice")
	state.ThermalStatus = thermalState(thermal)
	if state.ThermalStatus == "" && thermal != "" {
		state.Unavailable["thermal_status"] = "adb " + strings.Join(a.args("shell", "dumpsys", "thermalservice"), " ")
	}
	size := probe("display_metrics", "shell", "wm", "size")
	state.DisplayWidth, state.DisplayHeight = displaySize(size)
	density := probe("display_density", "shell", "wm", "density")
	state.DisplayDensity = displayDensity(density)
	if state.DisplayWidth <= 0 || state.DisplayHeight <= 0 {
		state.Unavailable["display_metrics"] = "adb " + strings.Join(a.args("shell", "wm", "size"), " ")
	}
	if state.DisplayDensity <= 0 {
		state.Unavailable["display_density"] = "adb " + strings.Join(a.args("shell", "wm", "density"), " ")
	}
	// Keep the bounded, additive dreams probe last so it cannot consume the
	// caller's remaining budget before any existing state fields are read.
	dreamStateValue, dreamComponent, unavailableDreamState := probeDreamState(ctx, a.runner, a.args("shell", "dumpsys", "dreams"), dreamStateProbeTimeout)
	state.DreamState, state.DreamComponent = dreamStateValue, dreamComponent
	if unavailableDreamState != "" {
		state.Unavailable["dream_state"] = unavailableDreamState
	}
	if len(state.Unavailable) == 0 {
		state.Unavailable = nil
	}
	if err := ctx.Err(); err != nil {
		return state, fmt.Errorf("android device state observation: %w", err)
	}
	return state, nil
}

func foregroundPackage(output string) string {
	if top := topResumedComponent(output); top != "" {
		return strings.SplitN(top, "/", 2)[0]
	}
	matches := regexp.MustCompile(`(?i)(?:mResumedActivity|ResumedActivity):.*?\s(?:u\d+\s+)?([A-Za-z0-9._]+)/(?:[A-Za-z0-9.$_]+)`).FindAllStringSubmatch(output, -1)
	pkg := ""
	for _, match := range matches {
		if pkg != "" && pkg != match[1] {
			return ""
		}
		pkg = match[1]
	}
	return pkg
}

func screenState(output string) string {
	if regexp.MustCompile(`(?i)(Display Power|mWakefulness).*?(ON|Awake)`).MatchString(output) {
		return "on"
	}
	if regexp.MustCompile(`(?i)(Display Power|mWakefulness).*?(OFF|Asleep|Dozing)`).MatchString(output) {
		return "off"
	}
	return ""
}

func orientation(output string) string {
	match := regexp.MustCompile(`(?i)(?:mCurrentRotation|SurfaceOrientation)\s*[:=]\s*(\d)`).FindStringSubmatch(output)
	if len(match) != 2 {
		return ""
	}
	values := map[string]string{"0": "portrait", "1": "landscape", "2": "reverse-portrait", "3": "reverse-landscape"}
	return values[match[1]]
}

func orientationValue(output string) string {
	value := strings.TrimSpace(output)
	if match := regexp.MustCompile(`\d`).FindString(value); match != "" {
		return map[string]string{"0": "portrait", "1": "landscape", "2": "reverse-portrait", "3": "reverse-landscape"}[match]
	}
	return ""
}

func batteryState(output string) (int, bool) {
	level := -1
	if match := regexp.MustCompile(`(?m)^\s*level:\s*(\d+)`).FindStringSubmatch(output); len(match) == 2 {
		level, _ = strconv.Atoi(match[1])
	}
	charging := regexp.MustCompile(`(?mi)^\s*(?:AC powered|USB powered|Wireless powered):\s*true`).MatchString(output)
	return level, charging
}

func thermalState(output string) string {
	match := regexp.MustCompile(`(?i)(?:status|thermal status)\s*[:=]\s*([A-Za-z0-9_-]+)`).FindStringSubmatch(output)
	if len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return ""
}

func displaySize(output string) (int, int) {
	match := regexp.MustCompile(`(\d+)x(\d+)`).FindStringSubmatch(output)
	if len(match) != 3 {
		return 0, 0
	}
	width, _ := strconv.Atoi(match[1])
	height, _ := strconv.Atoi(match[2])
	return width, height
}

func displayDensity(output string) int {
	match := regexp.MustCompile(`(\d+)`).FindStringSubmatch(output)
	if len(match) != 2 {
		return 0
	}
	density, _ := strconv.Atoi(match[1])
	return density
}

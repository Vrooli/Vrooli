package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"device-control/strategy"
)

const maxAndroidWindowDumpBytes = 512 * 1024

var (
	windowInsetSourcePattern = regexp.MustCompile(`(?i)type=(statusBars|navigationBars|ime).*?frame=\[(-?\d+),\s*(-?\d+)\]\[(-?\d+),\s*(-?\d+)\].*?visible=(true|false)`)
	windowCutoutPattern      = regexp.MustCompile(`(?i)DisplayCutout\{[^}]*insets=Rect\(\s*(-?\d+)\s*,\s*(-?\d+)\s*-\s*(-?\d+)\s*,\s*(-?\d+)\s*\)`)
	windowNoCutoutPattern    = regexp.MustCompile(`(?i)mDisplayCutout\s*=\s*null|DisplayCutout\{insets=Rect\(0,\s*0\s*-\s*0,\s*0\)`)
)

type AndroidWindowMetrics struct {
	StatusBarVisible        bool
	StatusBarHeightPX       int
	NavigationBarVisible    bool
	NavigationBarHeightPX   int
	IMEVisible              bool
	IMEHeightPX             int
	CutoutTopPX             int
	CutoutBottomPX          int
	CutoutLeftPX            int
	CutoutRightPX           int
	CutoutSourcePresent     bool
	SafeInsetTopPX          int
	SafeInsetBottomPX       int
	SafeInsetLeftPX         int
	SafeInsetRightPX        int
	StatusBarSourcePresent  bool
	NavigationSourcePresent bool
	IMESourcePresent        bool
}

func (m AndroidWindowMetrics) Values() map[string]any {
	values := map[string]any{}
	if m.StatusBarSourcePresent {
		values["android.status_bar.visible"] = m.StatusBarVisible
		values["android.status_bar.height_px"] = m.StatusBarHeightPX
	}
	if m.NavigationSourcePresent {
		values["android.navigation_bar.visible"] = m.NavigationBarVisible
		values["android.navigation_bar.height_px"] = m.NavigationBarHeightPX
	}
	if m.IMESourcePresent {
		values["android.ime.visible"] = m.IMEVisible
		values["android.ime.height_px"] = m.IMEHeightPX
	}
	if m.CutoutSourcePresent {
		values["android.cutout.top_px"] = m.CutoutTopPX
		values["android.cutout.bottom_px"] = m.CutoutBottomPX
		values["android.cutout.left_px"] = m.CutoutLeftPX
		values["android.cutout.right_px"] = m.CutoutRightPX
		values["android.safe_inset.left_px"] = m.SafeInsetLeftPX
		values["android.safe_inset.right_px"] = m.SafeInsetRightPX
		if m.StatusBarSourcePresent {
			values["android.safe_inset.top_px"] = m.SafeInsetTopPX
		}
		if m.NavigationSourcePresent {
			values["android.safe_inset.bottom_px"] = m.SafeInsetBottomPX
		}
	}
	return values
}

func parseWindowMetrics(dump string, displayWidth, displayHeight int) (AndroidWindowMetrics, bool) {
	var metrics AndroidWindowMetrics
	for _, line := range strings.Split(dump, "\n") {
		match := windowInsetSourcePattern.FindStringSubmatch(line)
		if len(match) != 7 {
			continue
		}
		left, okLeft := parseWindowInt(match[2])
		top, okTop := parseWindowInt(match[3])
		right, okRight := parseWindowInt(match[4])
		bottom, okBottom := parseWindowInt(match[5])
		if !okLeft || !okTop || !okRight || !okBottom || right < left || bottom < top {
			continue
		}
		visible := strings.EqualFold(match[6], "true")
		switch strings.ToLower(match[1]) {
		case "statusbars":
			metrics.StatusBarSourcePresent = true
			metrics.StatusBarVisible = visible
			metrics.StatusBarHeightPX = edgeInsetHeight(left, top, right, bottom, displayWidth, displayHeight)
		case "navigationbars":
			metrics.NavigationSourcePresent = true
			metrics.NavigationBarVisible = visible
			metrics.NavigationBarHeightPX = edgeInsetHeight(left, top, right, bottom, displayWidth, displayHeight)
		case "ime":
			metrics.IMESourcePresent = true
			metrics.IMEVisible = visible
			if visible {
				metrics.IMEHeightPX = edgeInsetHeight(left, top, right, bottom, displayWidth, displayHeight)
			}
		}
	}
	for _, match := range windowCutoutPattern.FindAllStringSubmatch(dump, -1) {
		if len(match) != 5 {
			continue
		}
		values := [4]int{}
		valid := true
		for i := range values {
			value, ok := parseWindowInt(match[i+1])
			if !ok || value < 0 || value > max(displayWidth, displayHeight) {
				valid = false
				break
			}
			values[i] = value
		}
		if valid {
			metrics.CutoutSourcePresent = true
			metrics.CutoutLeftPX, metrics.CutoutTopPX, metrics.CutoutRightPX, metrics.CutoutBottomPX = values[0], values[1], values[2], values[3]
			break
		}
	}
	if !metrics.CutoutSourcePresent && windowNoCutoutPattern.MatchString(dump) {
		metrics.CutoutSourcePresent = true
	}
	if !metrics.StatusBarSourcePresent && !metrics.NavigationSourcePresent && !metrics.IMESourcePresent && !metrics.CutoutSourcePresent {
		return AndroidWindowMetrics{}, false
	}
	metrics.SafeInsetTopPX = max(metrics.StatusBarHeightPX, metrics.CutoutTopPX)
	metrics.SafeInsetBottomPX = max(metrics.NavigationBarHeightPX, metrics.CutoutBottomPX)
	metrics.SafeInsetLeftPX = metrics.CutoutLeftPX
	metrics.SafeInsetRightPX = metrics.CutoutRightPX
	return metrics, true
}

func edgeInsetHeight(left, top, right, bottom, displayWidth, displayHeight int) int {
	if displayWidth > 0 && right-left >= displayWidth/2 {
		return bottom - top
	}
	if displayHeight > 0 && bottom-top >= displayHeight/2 {
		return right - left
	}
	return 0
}

func parseWindowInt(raw string) (int, bool) {
	value, err := strconv.Atoi(raw)
	return value, err == nil
}

func (a *Adapter) readWindowMetrics(ctx context.Context) (string, error) {
	args := a.args("shell", "dumpsys", "window")
	var output []byte
	var err error
	if bounded, ok := a.runner.(boundedRunner); ok {
		output, err = bounded.RunBounded(ctx, "adb", maxAndroidWindowDumpBytes, args...)
	} else {
		output, err = a.runner.Run(ctx, "adb", args...)
		if err == nil && len(output) > maxAndroidWindowDumpBytes {
			err = fmt.Errorf("Android window output exceeds %d bytes", maxAndroidWindowDumpBytes)
		}
	}
	if err != nil {
		return "", fmt.Errorf("read Android window insets: %w", err)
	}
	return string(output), nil
}

func androidWindowPropertyDescriptors() []strategy.PropertyDescriptor {
	descriptors := make([]strategy.PropertyDescriptor, 0, 14)
	for _, name := range []string{
		"android.status_bar.visible", "android.navigation_bar.visible", "android.ime.visible",
	} {
		descriptors = append(descriptors, strategy.PropertyDescriptor{Name: name, ValueType: "boolean", Writable: false, StateClass: strategy.StateBearing, StateDomain: "android_window." + name})
	}
	for _, name := range []string{
		"android.status_bar.height_px", "android.navigation_bar.height_px", "android.ime.height_px",
		"android.cutout.top_px", "android.cutout.bottom_px", "android.cutout.left_px", "android.cutout.right_px",
		"android.safe_inset.top_px", "android.safe_inset.bottom_px", "android.safe_inset.left_px", "android.safe_inset.right_px",
	} {
		descriptors = append(descriptors, strategy.PropertyDescriptor{Name: name, ValueType: "number", Writable: false, Minimum: intPtr(0), StateClass: strategy.StateBearing, StateDomain: "android_window." + name})
	}
	return descriptors
}

func intPtr(value float64) *float64 { return &value }

// Package usagelabel is the one owner of how a provider window is named (SPEC
// P2-D12): "Claude · weekly", "Codex · 5-hour", "daily"; never "5h". It has no
// dependencies so the API and the CLI can share it.
package usagelabel

import "fmt"

// NamedWindowMinutes is the length of the windows providers name instead of
// measuring.
var NamedWindowMinutes = map[string]int64{"five_hour": 300, "seven_day": 10080}

// Provider names a provider by the plan the operator subscribes to.
func Provider(provider string) string {
	switch provider {
	case "openai":
		return "Codex"
	case "anthropic":
		return "Claude"
	}
	return provider
}

// roundingMinutes is how far a reported window length may sit from a whole
// hour and still be that length: providers report 299m and 10079m for the
// 5-hour and weekly windows.
const roundingMinutes = 2

// Minutes is the window's length in minutes: the reported minutes (or those
// parsed from window, or named by it), rounded to the nearest whole hour when
// within roundingMinutes of it.
func Minutes(minutes int64, window string) int64 {
	if minutes == 0 {
		if _, err := fmt.Sscanf(window, "%dm", &minutes); err != nil {
			minutes = NamedWindowMinutes[window]
		}
	}
	if nearest := (minutes + 30) / 60 * 60; nearest > 0 && minutes-nearest <= roundingMinutes && nearest-minutes <= roundingMinutes {
		return nearest
	}
	return minutes
}

// Window is "weekly", "daily" or "5-hour" (P2-D12, never "5h"); an
// unrecognised window is returned as-is.
func Window(minutes int64, window string) string {
	switch minutes = Minutes(minutes, window); {
	case minutes == 10080:
		return "weekly"
	case minutes == 1440:
		return "daily"
	case minutes > 0 && minutes%60 == 0:
		return fmt.Sprintf("%d-hour", minutes/60)
	}
	return window
}

// Title is "Claude · weekly" or "Codex · 5-hour".
func Title(provider, window string, minutes int64) string {
	return Provider(provider) + " · " + Window(minutes, window)
}

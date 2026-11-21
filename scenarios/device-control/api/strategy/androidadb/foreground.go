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

func activityComponent(data string, patterns ...string) string {
	for _, pattern := range patterns {
		match := regexp.MustCompile(pattern).FindStringSubmatch(data)
		if len(match) == 2 {
			if component := canonicalHome(match[1]); component != "" {
				return component
			}
		}
	}
	return ""
}

func topResumedComponent(data string) string {
	return activityComponent(data, `(?m)\btopResumedActivity\s*[:=][^\n]*?\s([A-Za-z0-9_.$]+/[A-Za-z0-9_.$]+)`)
}

// foregroundState preserves independent Android signals instead of hiding a
// disagreement behind one package name. No raw task/window contents leave here.
func (a *Adapter) foregroundState(ctx context.Context, pkg string) (strategy.AppLifecycleResult, error) {
	r := strategy.AppLifecycleResult{Operation: "foreground-state", Package: pkg, Status: strategy.StatusUnavailable}
	read := func(args ...string) string {
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil || len(out) > 256*1024 {
			return ""
		}
		return string(out)
	}
	activity := read("shell", "dumpsys", "activity", "activities")
	r.ResumedActivity = topResumedComponent(activity)
	if r.ResumedActivity != "" {
		r.ResumedActivitySource = "top-resumed"
	} else {
		r.ResumedActivity = activityComponent(activity, `(?m)\b(?:mResumedActivity|ResumedActivity)\s*[:=][^\n]*?\s([A-Za-z0-9_.$]+/[A-Za-z0-9_.$]+)`)
		if r.ResumedActivity != "" {
			r.ResumedActivitySource = "legacy-resumed"
		}
	}
	r.FocusedWindowActivity = activityComponent(read("shell", "dumpsys", "window"), `(?m)\bmCurrentFocus=Window\{[^\n]*?\s([A-Za-z0-9_.$]+/[A-Za-z0-9_.$]+)`)
	boot := strings.TrimSpace(read("shell", "settings", "get", "global", "boot_count"))
	if regexp.MustCompile(`^[0-9]{1,10}$`).MatchString(boot) {
		r.BootCount = boot
	}
	if fields := strings.Fields(read("shell", "cat", "/proc/uptime")); len(fields) == 2 {
		if v, err := strconv.ParseFloat(fields[0], 64); err == nil && v >= 0 && v < 1e12 {
			r.UptimeSeconds = &v
		}
	}
	if r.ResumedActivity == "" && r.FocusedWindowActivity == "" {
		return r, fmt.Errorf("foreground activity and window signals are unavailable")
	}
	r.Status = strategy.StatusAvailable
	r.Outcome = "foreground_facts"
	r.ObservedAt = time.Now().UTC()
	if r.ResumedActivity != "" && r.FocusedWindowActivity != "" && strings.SplitN(r.ResumedActivity, "/", 2)[0] != strings.SplitN(r.FocusedWindowActivity, "/", 2)[0] {
		r.Reason = "resumed activity and focused window disagree; neither proves the physical screen alone"
	}
	return r, nil
}

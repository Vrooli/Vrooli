package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"device-control/strategy"
)

// accessibilityState reads only one requested service's configuration. It
// neither enables services nor returns other services or screen contents.
func (a *Adapter) accessibilityState(ctx context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	result := strategy.AppLifecycleResult{Operation: "accessibility-state", Package: request.Package, Status: strategy.StatusUnavailable}
	component := canonicalHome(request.Package + "/" + request.Value)
	if component == "" || request.Permission != "" {
		return result, fmt.Errorf("accessibility-state requires one service class in value")
	}
	user, err := a.currentAndroidUser(ctx)
	if err != nil {
		return result, err
	}
	result.AndroidUser = user
	read := func(key string) (string, error) {
		out, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "--user", user, "get", "secure", key)...)
		if err != nil || len(out) > 64*1024 {
			return "", fmt.Errorf("accessibility configuration is unavailable")
		}
		return strings.TrimSpace(string(out)), nil
	}
	master, err := read("accessibility_enabled")
	if err != nil || (master != "0" && master != "1") {
		return result, fmt.Errorf("accessibility enabled state is unknown")
	}
	list, err := read("enabled_accessibility_services")
	if err != nil {
		return result, err
	}
	configured := false
	if list != "" && list != "null" {
		for _, entry := range strings.Split(list, ":") {
			observed := canonicalHome(entry)
			if observed == "" {
				return result, fmt.Errorf("accessibility service configuration is unrecognized")
			}
			configured = configured || observed == component
		}
	}
	enabled := master == "1" && configured
	result.ServiceComponent = component
	result.ServiceConfigured = &configured
	result.ServiceEnabled = &enabled
	if data, dumpErr := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "accessibility")...); dumpErr == nil && len(data) <= 256*1024 {
		result.ServiceBound = boundAccessibilityService(string(data), user, component)
		result.ServiceCrashed = accessibilityServiceSet(string(data), user, component, "Crashed")
		result.ServiceBinding = accessibilityServiceSet(string(data), user, component, "Binding")
		if strings.Contains(string(data), "ACCESSIBILITY MANAGER (dumpsys accessibility)") {
			active := strings.Contains(string(data), "Ui Automation[")
			result.UIAutomationActive = &active
		}
	}
	result.Status = strategy.StatusAvailable
	result.Outcome = "accessibility_observed"
	result.Reason = "configuration observed; runtime binding is unknown when absent, and Home routing requires independent verification"
	result.ObservedAt = time.Now().UTC()
	return result, nil
}

func accessibilityServiceSet(data, user, component, kind string) *bool {
	for _, block := range strings.Split(data, "User state[")[1:] {
		id := regexp.MustCompile(`attributes:\{id=([0-9]+)[,}]`).FindStringSubmatch(block)
		if len(id) != 2 || id[1] != user {
			continue
		}
		marker := kind + " services:{"
		start := strings.Index(block, marker)
		if start < 0 {
			return nil
		}
		section := block[start+len(marker):]
		end := strings.Index(section, "\n")
		if end < 0 {
			return nil
		}
		section = section[:end]
		matches := regexp.MustCompile(`[A-Za-z][A-Za-z0-9_.$]*/[A-Za-z0-9_.$]+`).FindAllString(section, -1)
		if strings.TrimSpace(section) != "}" && len(matches) == 0 {
			return nil
		}
		found := false
		for _, match := range matches {
			found = found || canonicalHome(match) == component
		}
		return &found
	}
	return nil
}

func boundAccessibilityService(data, user, component string) *bool {
	for _, block := range strings.Split(data, "User state[")[1:] {
		id := regexp.MustCompile(`attributes:\{id=([0-9]+)[,}]`).FindStringSubmatch(block)
		if len(id) != 2 || id[1] != user {
			continue
		}
		start, end := strings.Index(block, "Bound services:{"), strings.Index(block, "Enabled services:{")
		if start < 0 || end <= start {
			return nil
		}
		bound := block[start+len("Bound services:{") : end]
		ids := regexp.MustCompile(`\bid=([A-Za-z0-9_.$]+/[A-Za-z0-9_.$]+)`).FindAllStringSubmatch(bound, -1)
		if strings.TrimSpace(bound) != "}" && len(ids) == 0 {
			return nil
		}
		found := false
		for _, id := range ids {
			found = found || canonicalHome(id[1]) == component
		}
		return &found
	}
	return nil
}

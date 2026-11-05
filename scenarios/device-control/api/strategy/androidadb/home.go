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

func canonicalHome(component string) string {
	if len(component) > 512 || !launchComponentPattern.MatchString(component) {
		return ""
	}
	parts := strings.SplitN(component, "/", 2)
	if strings.HasPrefix(parts[1], ".") {
		parts[1] = parts[0] + parts[1]
	}
	return parts[0] + "/" + parts[1]
}

func (a *Adapter) currentAndroidUser(ctx context.Context) (string, error) {
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "am", "get-current-user")...)
	user := strings.TrimSpace(string(out))
	if err != nil || !regexp.MustCompile(`^[0-9]{1,8}$`).MatchString(user) {
		return "", fmt.Errorf("current Android user is not observable")
	}
	return user, nil
}

func (a *Adapter) resolveHome(ctx context.Context, user, pkg string) (string, error) {
	args := []string{"shell", "cmd", "package", "resolve-activity", "--brief", "--user", user, "-a", "android.intent.action.MAIN", "-c", "android.intent.category.HOME"}
	if pkg != "" {
		args = append(args, "-p", pkg)
	}
	out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if component := canonicalHome(strings.TrimSpace(line)); component != "" && (pkg == "" || strings.HasPrefix(component, pkg+"/")) {
			return component, nil
		}
	}
	return "", fmt.Errorf("Home activity is not resolvable for the selected user/package")
}

// setHome is a compare-and-set default-Home operation, not arbitrary shell.
// Value is the expected previous component, obtained from current inspection.
func (a *Adapter) setHome(ctx context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	result := strategy.AppLifecycleResult{Operation: "home-set", Package: request.Package, Status: strategy.StatusUnavailable}
	expected := canonicalHome(request.Value)
	if expected == "" || !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`).MatchString(request.Package) {
		return result, fmt.Errorf("home-set requires a package and expected prior Home component")
	}
	user, err := a.currentAndroidUser(ctx)
	if err != nil {
		return result, err
	}
	result.AndroidUser = user
	previous, err := a.resolveHome(ctx, user, "")
	if err != nil {
		return result, err
	}
	result.PreviousHomeComponent = previous
	result.HomeComponent = previous
	if previous != expected {
		return result, fmt.Errorf("Home changed since inspection; reconcile the current default before changing it")
	}
	candidate, err := a.resolveHome(ctx, user, request.Package)
	if err != nil {
		return result, err
	}
	if candidate == previous {
		result.Status = strategy.StatusAvailable
		result.Outcome = "home_verified"
		result.ObservedAt = time.Now().UTC()
		return result, nil
	}
	currentUser, err := a.currentAndroidUser(ctx)
	if err != nil || currentUser != user {
		return result, fmt.Errorf("Android user changed before Home selection")
	}
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "package", "set-home-activity", "--user", user, candidate)...)
	result.Diagnostic = boundedDiagnostic(out)
	if err != nil || strings.TrimSpace(string(out)) != "Success" {
		return result, fmt.Errorf("Android did not accept Home selection; inspect current Home before any retry")
	}
	result.Outcome = "accepted"
	observed, err := a.resolveHome(ctx, user, "")
	result.HomeComponent = observed
	if err != nil || observed != candidate {
		return result, fmt.Errorf("Home selection was accepted but readback did not match; reconcile before any retry")
	}
	result.Status = strategy.StatusAvailable
	result.Outcome = "home_verified"
	result.ObservedAt = time.Now().UTC()
	return result, nil
}

// homeState distinguishes the chosen role from effective HOME resolution and
// reports only bounded Home-related facts, never the private raw system dumps.
func (a *Adapter) homeState(ctx context.Context, pkg string) (strategy.AppLifecycleResult, error) {
	result := strategy.AppLifecycleResult{Operation: "home-state", Package: pkg, Status: strategy.StatusAvailable, Outcome: "home_observed", HomeUnavailable: map[string]string{}}
	user, err := a.currentAndroidUser(ctx)
	if err != nil {
		return result, err
	}
	result.AndroidUser = user
	result.HomeComponent, err = a.resolveHome(ctx, user, "")
	if err != nil {
		result.HomeUnavailable["resolved_home"] = "current Home was not resolvable"
	}
	data, err := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "package", "query-activities", "--brief", "--user", user, "-a", "android.intent.action.MAIN", "-c", "android.intent.category.HOME")...)
	if err == nil && len(data) <= 128*1024 {
		result.HomeCandidates = parseHomeCandidates(string(data))
	}
	if len(result.HomeCandidates) == 0 {
		result.HomeUnavailable["candidates"] = "Home candidate list was unavailable or unrecognized"
	}
	for i, candidate := range result.HomeCandidates {
		name := strings.SplitN(candidate.Component, "/", 2)[0]
		data, err = a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "package", name)...)
		if err == nil && len(data) <= 128*1024 {
			if value, ok := enabledSettingFromDump(string(data), user); ok {
				result.HomeCandidates[i].EnabledSetting = &value
			}
		}
	}
	data, err = a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "role")...)
	if err == nil && len(data) <= 128*1024 {
		if holders, ok := parseHomeRoleHolders(string(data), user); ok {
			result.HomeRoleHolders = &holders
		}
	}
	if result.HomeRoleHolders == nil {
		result.HomeUnavailable["role_holders"] = "Home role holder was unavailable or unrecognized"
	}
	result.ObservedAt = time.Now().UTC()
	if len(result.HomeUnavailable) == 0 {
		result.HomeUnavailable = nil
	}
	return result, nil
}

func parseHomeCandidates(data string) []strategy.HomeCandidate {
	var result []strategy.HomeCandidate
	var priority *int
	pattern := regexp.MustCompile(`\bpriority=(-?[0-9]+)\b`)
	for _, line := range strings.Split(data, "\n") {
		if match := pattern.FindStringSubmatch(line); len(match) == 2 {
			if value, err := strconv.Atoi(match[1]); err == nil {
				priority = &value
			}
		}
		if component := canonicalHome(strings.TrimSpace(line)); component != "" {
			result = append(result, strategy.HomeCandidate{Component: component, Priority: priority})
			priority = nil
			if len(result) > 32 {
				return nil
			}
		}
	}
	return result
}

func parseHomeRoleHolders(data, user string) ([]string, bool) {
	depth, userDepth, roleDepth := 0, -1, -1
	currentUser, role := "", ""
	holders := []string{}
	for _, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasSuffix(line, "{") {
			depth++
			if strings.HasPrefix(line, "user_states=") {
				userDepth = depth
				currentUser = ""
			}
			if strings.HasPrefix(line, "roles=") && userDepth > 0 {
				roleDepth = depth
				role = ""
				holders = []string{}
			}
			continue
		}
		if line == "}" {
			if roleDepth == depth {
				if currentUser == user && role == "android.app.role.HOME" {
					return holders, true
				}
				roleDepth = -1
			}
			if userDepth == depth {
				userDepth = -1
				currentUser = ""
			}
			depth--
			continue
		}
		if userDepth > 0 && strings.HasPrefix(line, "user_id=") {
			currentUser = strings.TrimPrefix(line, "user_id=")
		}
		if roleDepth > 0 && strings.HasPrefix(line, "name=") {
			role = strings.TrimPrefix(line, "name=")
		}
		if roleDepth > 0 && strings.HasPrefix(line, "holders=") {
			name := strings.TrimPrefix(line, "holders=")
			if !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`).MatchString(name) || len(holders) >= 16 {
				return nil, false
			}
			holders = append(holders, name)
		}
	}
	return nil, false
}

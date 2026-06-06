package cliutil

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	lookPathFn           = exec.LookPath
	execCommandContextFn = exec.CommandContext
)

// DetectPortFromVrooli returns a detector that asks vrooli for the port of a
// scenario. The detector is instance-aware: it resolves the shadow-aware target
// (explicit --instance override or ambient VROOLI_SHADOW_SCENARIOS) at call time,
// so when the named scenario is shadowed the lookup addresses "<name>@shadow".
// If that non-live lookup yields nothing, it warns once and falls back to the
// live instance — never silent. For an unshadowed scenario the target is the
// bare name and behavior is unchanged.
func DetectPortFromVrooli(scenarioName, portVar string) func() string {
	return func() string {
		target := ResolveShadowTarget(scenarioName)
		port := detectPortForTarget(target, portVar)
		if port == "" && IsNonLiveTarget(target) {
			WarnShadowFallback(scenarioName)
			port = detectPortForTarget(BareScenarioName(scenarioName), portVar)
		}
		return port
	}
}

func detectPortForTarget(target, portVar string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	argv0 := "vrooli"
	if resolved, err := lookPathFn("vrooli"); err == nil && strings.TrimSpace(resolved) != "" {
		argv0 = resolved
	}
	cmd := execCommandContextFn(ctx, argv0, "--no-stale-check", "--json", "scenario", "port", target, portVar)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return sanitizePortOutput(string(output))
}

func sanitizePortOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}
	if port := portFromJSON(trimmed); port != "" {
		return port
	}
	re := regexp.MustCompile(`\b(\d{2,5})\b`)
	match := re.FindString(trimmed)
	return strings.TrimSpace(match)
}

func portFromJSON(output string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ""
	}
	switch v := payload["port"].(type) {
	case float64:
		if v > 0 {
			return strconv.Itoa(int(v))
		}
	case string:
		return strings.TrimSpace(v)
	}
	return ""
}

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

// DetectPortFromVrooli returns a detector that asks vrooli for the port of a scenario.
func DetectPortFromVrooli(scenarioName, portVar string) func() string {
	return func() string {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		argv0 := "vrooli"
		if resolved, err := lookPathFn("vrooli"); err == nil && strings.TrimSpace(resolved) != "" {
			argv0 = resolved
		}
		cmd := execCommandContextFn(ctx, argv0, "--no-stale-check", "--json", "scenario", "port", scenarioName, portVar)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return ""
		}
		return sanitizePortOutput(string(output))
	}
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

package cliutil

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// DetectPortFromVrooli returns a detector that asks vrooli for the port of a scenario.
func DetectPortFromVrooli(scenarioName, portVar string) func() string {
	return func() string {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, portVar)
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
	re := regexp.MustCompile(`\b(\d{2,5})\b`)
	match := re.FindString(trimmed)
	return strings.TrimSpace(match)
}

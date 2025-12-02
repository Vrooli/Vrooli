package cliutil

import (
	"context"
	"os/exec"
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
		return strings.TrimSpace(string(output))
	}
}

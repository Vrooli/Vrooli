package main

import (
	"context"
	"fmt"
	"strings"
)

// StopScenarioResult represents the result of stopping a scenario
type StopScenarioResult struct {
	OK           bool   `json:"ok"`
	ScenarioStop bool   `json:"scenario_stop"` // vrooli scenario stop succeeded
	OrphanKills  int    `json:"orphan_kills"`  // number of orphaned processes killed
	PortKills    int    `json:"port_kills"`    // number of processes killed on ports
	Message      string `json:"message"`
	Error        string `json:"error,omitempty"`
}

// StopExistingScenario gracefully stops any running instance of the scenario.
// This should be called before starting a new deployment to avoid port conflicts.
//
// Strategy:
// 1. Try vrooli scenario stop <id> (graceful lifecycle-aware stop)
// 2. Kill any orphaned processes matching the scenario ID
// 3. Kill any processes on the target ports (UI, API)
func StopExistingScenario(
	ctx context.Context,
	sshRunner SSHRunner,
	cfg SSHConfig,
	workdir string,
	scenarioID string,
	targetPorts []int, // Ports to forcefully clear (e.g., [35000, 15000])
) StopScenarioResult {
	result := StopScenarioResult{OK: true}

	// Step 1: Try vrooli scenario stop (if vrooli CLI exists)
	checkCliResult, _ := sshRunner.Run(ctx, cfg, vrooliCommand(workdir, "which vrooli || echo notfound"))
	if !strings.Contains(checkCliResult.Stdout, "notfound") {
		stopCmd := vrooliCommand(workdir, "vrooli scenario stop "+shellQuoteSingle(scenarioID))
		stopResult, err := sshRunner.Run(ctx, cfg, stopCmd)
		if err == nil && stopResult.ExitCode == 0 {
			result.ScenarioStop = true
		}
		// Don't fail on error - continue with fallback cleanup
	}

	// Step 2: Kill orphaned processes matching scenario ID
	// This catches processes started outside the lifecycle (manual starts, debug sessions)
	killOrphansCmd := fmt.Sprintf("pkill -f %s 2>/dev/null; true", shellQuoteSingle(scenarioID))
	orphanResult, _ := sshRunner.Run(ctx, cfg, killOrphansCmd)
	if orphanResult.ExitCode == 0 {
		result.OrphanKills++ // pkill returns 0 if it killed something
	}

	// Step 3: Kill any remaining processes on target ports
	// This is the nuclear option for port conflicts
	for _, port := range targetPorts {
		// Get PIDs on this port using ss
		ssCmd := fmt.Sprintf("ss -tlnpH 'sport = :%d' 2>/dev/null || true", port)
		ssResult, _ := sshRunner.Run(ctx, cfg, ssCmd)
		pids := extractPIDsFromSS(ssResult.Stdout)

		for _, pid := range pids {
			// Try graceful kill first, then SIGKILL as fallback
			killCmd := fmt.Sprintf("kill %s 2>/dev/null && sleep 1 && ! kill -0 %s 2>/dev/null || kill -9 %s 2>/dev/null || true", pid, pid, pid)
			sshRunner.Run(ctx, cfg, killCmd)
			result.PortKills++
		}
	}

	// Build message
	var parts []string
	if result.ScenarioStop {
		parts = append(parts, "scenario stopped via vrooli")
	}
	if result.OrphanKills > 0 {
		parts = append(parts, "killed orphaned processes")
	}
	if result.PortKills > 0 {
		parts = append(parts, fmt.Sprintf("cleared %d port processes", result.PortKills))
	}
	if len(parts) == 0 {
		result.Message = "no existing processes found"
	} else {
		result.Message = strings.Join(parts, ", ")
	}

	return result
}

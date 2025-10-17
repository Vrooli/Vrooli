package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// getScenarioResourceUsage fetches resource usage for a scenario by checking its running processes
func getScenarioResourceUsage(scenarioName string, port int) map[string]float64 {
	if port == 0 {
		return nil
	}

	// Use ps to get CPU and memory usage for process using this port
	// This is a basic implementation - could be enhanced with /proc parsing for more accuracy
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Find PID using the port
	cmd := exec.CommandContext(ctx, "lsof", "-ti", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return nil
	}

	// Get CPU and memory stats
	cmd = exec.CommandContext(ctx, "ps", "-p", pid, "-o", "%cpu,%mem", "--no-headers")
	output, err = cmd.Output()
	if err != nil {
		return nil
	}

	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return nil
	}

	var cpu, mem float64
	fmt.Sscanf(fields[0], "%f", &cpu)
	fmt.Sscanf(fields[1], "%f", &mem)

	return map[string]float64{
		"cpu_percent":    cpu,
		"memory_percent": mem,
	}
}

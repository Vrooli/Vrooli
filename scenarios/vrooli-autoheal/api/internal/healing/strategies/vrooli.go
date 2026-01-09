// Package strategies provides reusable healing action implementations.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package strategies

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"vrooli-autoheal/internal/checks"
)

// VrooliEntityType represents the type of Vrooli entity (resource or scenario).
type VrooliEntityType string

const (
	VrooliResource VrooliEntityType = "resource"
	VrooliScenario VrooliEntityType = "scenario"
)

// VrooliStrategy provides common Vrooli CLI actions for resources and scenarios.
// [REQ:TEST-SEAM-001]
type VrooliStrategy struct {
	entityType VrooliEntityType
	entityName string
	executor   checks.CommandExecutor
}

// NewVrooliStrategy creates a new Vrooli CLI strategy.
func NewVrooliStrategy(entityType VrooliEntityType, entityName string, executor checks.CommandExecutor) *VrooliStrategy {
	if executor == nil {
		executor = checks.DefaultExecutor
	}
	return &VrooliStrategy{
		entityType: entityType,
		entityName: entityName,
		executor:   executor,
	}
}

// Start starts the resource or scenario.
func (v *VrooliStrategy) Start(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "start",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "start", v.entityName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Failed to start %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("%s %s started successfully", v.entityType, v.entityName)
	return result
}

// Stop stops the resource or scenario.
func (v *VrooliStrategy) Stop(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "stop",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "stop", v.entityName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Failed to stop %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("%s %s stopped successfully", v.entityType, v.entityName)
	return result
}

// Restart restarts the resource or scenario.
func (v *VrooliStrategy) Restart(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "restart",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "restart", v.entityName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Failed to restart %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("%s %s restarted successfully", v.entityType, v.entityName)
	return result
}

// Status gets the status of the resource or scenario.
func (v *VrooliStrategy) Status(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "status",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "status", v.entityName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Failed to get status for %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Retrieved status for %s %s", v.entityType, v.entityName)
	return result
}

// Logs retrieves recent logs.
func (v *VrooliStrategy) Logs(ctx context.Context, checkID string, lines int) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "logs",
		CheckID:   checkID,
		Timestamp: start,
	}

	if lines <= 0 {
		lines = 100
	}

	output, err := v.executor.CombinedOutput(ctx,
		"vrooli", string(v.entityType), "logs", v.entityName, "--tail", fmt.Sprintf("%d", lines))
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Failed to retrieve logs for %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Retrieved logs for %s %s", v.entityType, v.entityName)
	return result
}

// GetPorts retrieves the ports used by the entity.
func (v *VrooliStrategy) GetPorts(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "ports",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "port", v.entityName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Failed to get ports for %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Retrieved ports for %s %s", v.entityType, v.entityName)
	return result
}

// CleanupPorts kills processes on the entity's ports.
func (v *VrooliStrategy) CleanupPorts(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "cleanup-ports",
		CheckID:   checkID,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Step 1: Get entity ports
	outputBuilder.WriteString("=== Getting ports ===\n")
	portOutput, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "port", v.entityName)
	outputBuilder.Write(portOutput)
	outputBuilder.WriteString("\n")

	if err != nil {
		result.Output = outputBuilder.String()
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "Failed to get ports: " + err.Error()
		result.Message = fmt.Sprintf("Could not determine ports for %s %s", v.entityType, v.entityName)
		return result
	}

	// Step 2: Parse ports
	ports := ExtractPorts(string(portOutput))
	if len(ports) == 0 {
		result.Output = outputBuilder.String()
		result.Duration = time.Since(start)
		result.Success = true
		result.Message = fmt.Sprintf("No ports found to cleanup for %s %s", v.entityType, v.entityName)
		return result
	}

	outputBuilder.WriteString(fmt.Sprintf("Found ports: %v\n\n", ports))

	// Step 3: Kill processes on each port
	killedCount := 0
	for _, port := range ports {
		outputBuilder.WriteString(fmt.Sprintf("=== Cleaning port %d ===\n", port))

		// Find process on port using lsof
		pidOutput, err := v.executor.Output(ctx, "lsof", "-ti", fmt.Sprintf(":%d", port))

		if err != nil || len(strings.TrimSpace(string(pidOutput))) == 0 {
			outputBuilder.WriteString("No process found on port\n")
			continue
		}

		// Kill each PID
		pids := strings.Fields(strings.TrimSpace(string(pidOutput)))
		for _, pidStr := range pids {
			outputBuilder.WriteString(fmt.Sprintf("Killing PID %s... ", pidStr))

			// First try SIGTERM
			if err := v.executor.Run(ctx, "kill", pidStr); err != nil {
				// If SIGTERM fails, try SIGKILL
				if err := v.executor.Run(ctx, "kill", "-9", pidStr); err != nil {
					outputBuilder.WriteString(fmt.Sprintf("FAILED: %v\n", err))
					continue
				}
			}
			outputBuilder.WriteString("OK\n")
			killedCount++
		}
	}

	result.Output = outputBuilder.String()
	result.Duration = time.Since(start)
	result.Success = true
	result.Message = fmt.Sprintf("Port cleanup complete: killed %d processes on %d ports", killedCount, len(ports))
	return result
}

// Diagnose gathers diagnostic information about the entity.
func (v *VrooliStrategy) Diagnose(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "diagnose",
		CheckID:   checkID,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	titleCaser := cases.Title(language.English)

	// Status
	outputBuilder.WriteString(fmt.Sprintf("=== %s Status ===\n", titleCaser.String(string(v.entityType))))
	statusOutput, _ := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "status", v.entityName)
	outputBuilder.Write(statusOutput)
	outputBuilder.WriteString("\n\n")

	// Ports
	outputBuilder.WriteString(fmt.Sprintf("=== %s Ports ===\n", titleCaser.String(string(v.entityType))))
	portOutput, _ := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "port", v.entityName)
	outputBuilder.Write(portOutput)
	outputBuilder.WriteString("\n\n")

	// Recent logs
	outputBuilder.WriteString("=== Recent Logs (last 50 lines) ===\n")
	logsOutput, _ := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "logs", v.entityName, "--tail", "50")
	outputBuilder.Write(logsOutput)
	outputBuilder.WriteString("\n")

	result.Output = outputBuilder.String()
	result.Duration = time.Since(start)
	result.Success = true
	result.Message = fmt.Sprintf("Diagnostic information gathered for %s %s", v.entityType, v.entityName)
	return result
}

// CleanRestart performs a stop, port cleanup, and restart.
func (v *VrooliStrategy) CleanRestart(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "restart-clean",
		CheckID:   checkID,
		Timestamp: start,
	}

	var outputBuilder strings.Builder

	// Step 1: Stop
	outputBuilder.WriteString(fmt.Sprintf("=== Stopping %s ===\n", v.entityType))
	stopOutput, _ := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "stop", v.entityName)
	outputBuilder.Write(stopOutput)
	outputBuilder.WriteString("\n")

	// Step 2: Cleanup ports
	outputBuilder.WriteString("=== Cleaning up ports ===\n")
	portResult := v.CleanupPorts(ctx, checkID)
	outputBuilder.WriteString(portResult.Output)
	outputBuilder.WriteString("\n")

	// Step 3: Start
	outputBuilder.WriteString(fmt.Sprintf("=== Starting %s ===\n", v.entityType))
	startOutput, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "start", v.entityName)
	outputBuilder.Write(startOutput)
	result.Output = outputBuilder.String()
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Clean restart failed for %s %s", v.entityType, v.entityName)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Clean restart completed for %s %s", v.entityType, v.entityName)
	return result
}

// IsRunning checks if the entity is currently running by checking status.
func (v *VrooliStrategy) IsRunning(ctx context.Context) bool {
	output, err := v.executor.CombinedOutput(ctx, "vrooli", string(v.entityType), "status", v.entityName)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(output))

	// Check for negative patterns first
	if strings.Contains(lower, "not running") ||
		strings.Contains(lower, "stopped") ||
		strings.Contains(lower, "exited") ||
		strings.Contains(lower, "failed") {
		return false
	}

	// Check for positive patterns
	return strings.Contains(lower, "running") ||
		strings.Contains(lower, "healthy") ||
		strings.Contains(lower, "started")
}

// ExtractPorts extracts port numbers from CLI output.
// This is a shared utility function.
func ExtractPorts(output string) []int {
	var ports []int
	seen := make(map[int]bool)

	// Look for common port patterns in the output
	// Patterns: "port: 8080", "PORT=8080", ":8080", "8080/tcp"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for _, field := range fields {
			// Handle KEY=VALUE patterns
			if idx := strings.Index(field, "="); idx != -1 {
				field = field[idx+1:]
			}

			// Remove common prefixes/suffixes
			field = strings.TrimPrefix(field, ":")
			field = strings.TrimSuffix(field, "/tcp")
			field = strings.TrimSuffix(field, "/udp")

			// Try to parse as port number (valid range: 1-65535)
			if port, err := strconv.Atoi(field); err == nil && port > 0 && port <= 65535 {
				// Filter out likely non-port numbers (too small or reserved)
				if port >= 1024 && !seen[port] {
					ports = append(ports, port)
					seen[port] = true
				}
			}
		}
	}

	return ports
}

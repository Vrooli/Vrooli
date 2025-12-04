// Package strategies provides reusable healing action implementations.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package strategies

import (
	"context"
	"fmt"
	"time"

	"vrooli-autoheal/internal/checks"
)

// SystemdStrategy provides common systemd service management actions.
// It supports restart, start, stop, and logs commands.
// [REQ:TEST-SEAM-001]
type SystemdStrategy struct {
	serviceName string
	executor    checks.CommandExecutor
}

// NewSystemdStrategy creates a new systemd strategy for the given service.
func NewSystemdStrategy(serviceName string, executor checks.CommandExecutor) *SystemdStrategy {
	if executor == nil {
		executor = checks.DefaultExecutor
	}
	return &SystemdStrategy{
		serviceName: serviceName,
		executor:    executor,
	}
}

// Restart restarts the systemd service.
func (s *SystemdStrategy) Restart(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "restart",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := s.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", s.serviceName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to restart " + s.serviceName
		return result
	}

	result.Success = true
	result.Message = s.serviceName + " restarted successfully"
	return result
}

// Start starts the systemd service.
func (s *SystemdStrategy) Start(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "start",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := s.executor.CombinedOutput(ctx, "sudo", "systemctl", "start", s.serviceName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to start " + s.serviceName
		return result
	}

	result.Success = true
	result.Message = s.serviceName + " started successfully"
	return result
}

// Stop stops the systemd service.
func (s *SystemdStrategy) Stop(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "stop",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := s.executor.CombinedOutput(ctx, "sudo", "systemctl", "stop", s.serviceName)
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to stop " + s.serviceName
		return result
	}

	result.Success = true
	result.Message = s.serviceName + " stopped successfully"
	return result
}

// Status checks the systemd service status.
func (s *SystemdStrategy) Status(ctx context.Context, checkID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "status",
		CheckID:   checkID,
		Timestamp: start,
	}

	output, err := s.executor.CombinedOutput(ctx, "systemctl", "status", s.serviceName, "--no-pager")
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		// systemctl status returns non-zero for stopped services, but still useful output
		result.Success = true
		result.Message = s.serviceName + " status retrieved"
		return result
	}

	result.Success = true
	result.Message = s.serviceName + " status retrieved"
	return result
}

// Logs retrieves recent logs for the systemd service.
func (s *SystemdStrategy) Logs(ctx context.Context, checkID string, lines int) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  "logs",
		CheckID:   checkID,
		Timestamp: start,
	}

	if lines <= 0 {
		lines = 100
	}

	output, err := s.executor.CombinedOutput(ctx,
		"journalctl", "-u", s.serviceName, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to retrieve logs for " + s.serviceName
		return result
	}

	result.Success = true
	result.Message = "Retrieved logs for " + s.serviceName
	return result
}

// IsActive checks if the systemd service is active.
func (s *SystemdStrategy) IsActive(ctx context.Context) bool {
	err := s.executor.Run(ctx, "systemctl", "is-active", "--quiet", s.serviceName)
	return err == nil
}

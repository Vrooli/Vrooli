// Package runner provides runner adapter implementations.
//
// This file implements the OpenCode runner adapter for executing
// OpenCode via the resource-opencode wrapper within agent-manager.
package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// OpenCodeResourceCommand is the Vrooli resource wrapper command
const OpenCodeResourceCommand = "resource-opencode"

// =============================================================================
// OpenCode Runner Implementation
// =============================================================================

// OpenCodeRunner implements the Runner interface for OpenCode CLI.
type OpenCodeRunner struct {
	binaryPath  string
	available   bool
	message     string
	installHint string
	mu          sync.Mutex
	runs        map[uuid.UUID]*exec.Cmd
}

// NewOpenCodeRunner creates a new OpenCode runner.
func NewOpenCodeRunner() (*OpenCodeRunner, error) {
	// Look for resource-opencode in PATH (the Vrooli wrapper)
	binaryPath, err := exec.LookPath(OpenCodeResourceCommand)
	if err != nil {
		return &OpenCodeRunner{
			available:   false,
			message:     "resource-opencode not found in PATH",
			installHint: "Run: vrooli resource install opencode",
			runs:        make(map[uuid.UUID]*exec.Cmd),
		}, nil
	}

	// Verify the resource is healthy by checking status
	runner := &OpenCodeRunner{
		binaryPath: binaryPath,
		available:  true,
		message:    "resource-opencode available",
		runs:       make(map[uuid.UUID]*exec.Cmd),
	}

	// Quick health check via status command
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "status", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		runner.available = false
		runner.message = fmt.Sprintf("resource-opencode status check failed: %v", err)
		runner.installHint = "Run: resource-opencode manage install"
		return runner, nil
	}

	// Parse JSON status to check health
	var statusData map[string]interface{}
	if err := json.Unmarshal(output, &statusData); err == nil {
		if healthy, ok := statusData["healthy"].(bool); ok && !healthy {
			runner.available = false
			if msg, ok := statusData["message"].(string); ok {
				runner.message = msg
			} else {
				runner.message = "resource-opencode is not healthy"
			}
			runner.installHint = "Run: resource-opencode manage install"
		}
	}

	return runner, nil
}

// Type returns the runner type identifier.
func (r *OpenCodeRunner) Type() domain.RunnerType {
	return domain.RunnerTypeOpenCode
}

// Capabilities returns what this runner supports.
func (r *OpenCodeRunner) Capabilities() Capabilities {
	return Capabilities{
		SupportsMessages:     true,
		SupportsToolEvents:   true,
		SupportsCostTracking: false, // OpenCode may not track costs the same way
		SupportsStreaming:    false,
		SupportsCancellation: true,
		MaxTurns:             0, // unlimited
		SupportedModels: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"claude-3-5-sonnet",
			"gemini-2.0-flash",
		},
	}
}

// Execute runs OpenCode with the given configuration.
func (r *OpenCodeRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, fmt.Errorf("opencode runner is not available: %s", r.message)
	}

	startTime := time.Now()

	// Build command arguments - use "run" subcommand
	// OpenCode supports: resource-opencode run <args>
	args := []string{"run", "--prompt", req.Prompt}

	// Add working directory if specified
	if req.WorkingDir != "" {
		args = append(args, "--cwd", req.WorkingDir)
	}

	// Create command using resource-opencode
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}

	// Set environment
	cmd.Env = r.buildEnv(req)

	// Track the running command for cancellation
	r.mu.Lock()
	r.runs[req.RunID] = cmd
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.runs, req.RunID)
		r.mu.Unlock()
	}()

	// Create pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start resource-opencode: %w", err)
	}

	// Emit starting event
	if req.EventSink != nil {
		req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"OpenCode execution started",
		))
	}

	// Process output
	metrics := ExecutionMetrics{}
	var outputBuilder strings.Builder
	var errorOutput strings.Builder

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Read stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")

		// Emit log event for each line
		if req.EventSink != nil {
			req.EventSink.Emit(domain.NewLogEvent(
				req.RunID,
				"info",
				line,
			))
		}
	}

	// Wait for command to complete
	err = cmd.Wait()
	duration := time.Since(startTime)

	// Determine result
	result := &ExecuteResult{
		Duration: duration,
		Metrics:  metrics,
	}

	if err != nil {
		// Check if it was cancelled
		if ctx.Err() == context.Canceled {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = "execution cancelled"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.Success = false
			result.ExitCode = exitErr.ExitCode()
			result.ErrorMessage = errorOutput.String()
		} else {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = err.Error()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
		result.Summary = &domain.RunSummary{
			Description: outputBuilder.String(),
		}
	}

	// Emit completion event
	if req.EventSink != nil {
		finalStatus := string(domain.RunStatusComplete)
		if !result.Success {
			finalStatus = string(domain.RunStatusFailed)
		}
		req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusRunning),
			finalStatus,
			"OpenCode execution completed",
		))
		req.EventSink.Close()
	}

	return result, nil
}

// Stop attempts to gracefully stop a running OpenCode instance.
func (r *OpenCodeRunner) Stop(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	cmd, exists := r.runs[runID]
	r.mu.Unlock()

	if !exists {
		return fmt.Errorf("run %s not found", runID)
	}

	// Try graceful termination first (SIGTERM)
	if cmd.Process != nil {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			// If SIGTERM fails, force kill
			return cmd.Process.Kill()
		}
	}

	return nil
}

// IsAvailable checks if OpenCode is currently available.
func (r *OpenCodeRunner) IsAvailable(ctx context.Context) (bool, string) {
	if !r.available {
		msg := r.message
		if r.installHint != "" {
			msg += ". " + r.installHint
		}
		return false, msg
	}

	// Verify the binary still exists
	if _, err := os.Stat(r.binaryPath); os.IsNotExist(err) {
		return false, "resource-opencode binary not found. Run: vrooli resource install opencode"
	}

	return true, "resource-opencode is available"
}

// InstallHint returns instructions for installing this runner.
func (r *OpenCodeRunner) InstallHint() string {
	return r.installHint
}

// buildEnv constructs environment variables for resource-opencode run.
func (r *OpenCodeRunner) buildEnv(req ExecuteRequest) []string {
	env := os.Environ()

	// Non-interactive mode
	env = append(env, "OPENCODE_NON_INTERACTIVE=true")

	// Model selection via environment
	if req.Profile != nil && req.Profile.Model != "" {
		env = append(env, fmt.Sprintf("OPENCODE_MODEL=%s", req.Profile.Model))
	}

	// Max turns
	if req.Profile != nil && req.Profile.MaxTurns > 0 {
		env = append(env, fmt.Sprintf("MAX_TURNS=%d", req.Profile.MaxTurns))
	}

	// Timeout in seconds
	if req.Profile != nil && req.Profile.Timeout > 0 {
		env = append(env, fmt.Sprintf("TIMEOUT=%d", int(req.Profile.Timeout.Seconds())))
	}

	// Allowed tools
	if req.Profile != nil && len(req.Profile.AllowedTools) > 0 {
		env = append(env, fmt.Sprintf("ALLOWED_TOOLS=%s", strings.Join(req.Profile.AllowedTools, ",")))
	}

	// Add any custom environment from the request
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

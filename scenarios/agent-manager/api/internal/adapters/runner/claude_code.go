// Package runner provides runner adapter implementations.
//
// This file implements the Claude Code runner adapter for executing
// Claude Code CLI as an agent within agent-manager.
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

// =============================================================================
// Claude Code Runner Implementation
// =============================================================================

// ClaudeCodeRunner implements the Runner interface for Claude Code CLI.
type ClaudeCodeRunner struct {
	binaryPath string
	available  bool
	message    string
	mu         sync.Mutex
	runs       map[uuid.UUID]*exec.Cmd
}

// NewClaudeCodeRunner creates a new Claude Code runner.
func NewClaudeCodeRunner() (*ClaudeCodeRunner, error) {
	// Look for claude in PATH
	binaryPath, err := exec.LookPath("claude")
	if err != nil {
		return &ClaudeCodeRunner{
			available: false,
			message:   "claude CLI not found in PATH",
			runs:      make(map[uuid.UUID]*exec.Cmd),
		}, nil
	}

	return &ClaudeCodeRunner{
		binaryPath: binaryPath,
		available:  true,
		message:    "claude CLI available",
		runs:       make(map[uuid.UUID]*exec.Cmd),
	}, nil
}

// Type returns the runner type identifier.
func (r *ClaudeCodeRunner) Type() domain.RunnerType {
	return domain.RunnerTypeClaudeCode
}

// Capabilities returns what this runner supports.
func (r *ClaudeCodeRunner) Capabilities() Capabilities {
	return Capabilities{
		SupportsMessages:     true,
		SupportsToolEvents:   true,
		SupportsCostTracking: true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		MaxTurns:             0, // unlimited
		SupportedModels: []string{
			"sonnet",
			"opus",
			"haiku",
			"claude-sonnet-4-5-20250929",
			"claude-opus-4-5-20251101",
		},
	}
}

// Execute runs Claude Code with the given configuration.
func (r *ClaudeCodeRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, fmt.Errorf("claude-code runner is not available: %s", r.message)
	}

	startTime := time.Now()

	// Build command arguments
	args := r.buildArgs(req)

	// Create command
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Dir = req.WorkingDir

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range req.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

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

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Emit starting event
	if req.EventSink != nil {
		req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"Claude Code execution started",
		))
	}

	// Write prompt and close stdin
	if _, err := stdin.Write([]byte(req.Prompt)); err != nil {
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

	// Process streaming output
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Parse streaming JSON output
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse the streaming event
		event, err := r.parseStreamEvent(req.RunID, line)
		if err != nil {
			// Log parsing error but continue
			if req.EventSink != nil {
				req.EventSink.Emit(domain.NewLogEvent(
					req.RunID,
					"warn",
					fmt.Sprintf("Failed to parse event: %v", err),
				))
			}
			continue
		}

		// Update metrics based on event
		r.updateMetrics(event, &metrics, &lastAssistantMessage)

		// Emit to sink
		if req.EventSink != nil && event != nil {
			req.EventSink.Emit(event)
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
			Description: lastAssistantMessage,
			TurnsUsed:   metrics.TurnsUsed,
			TokensUsed:  metrics.TokensInput + metrics.TokensOutput,
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
			"Claude Code execution completed",
		))
		req.EventSink.Close()
	}

	return result, nil
}

// Stop attempts to gracefully stop a running Claude Code instance.
func (r *ClaudeCodeRunner) Stop(ctx context.Context, runID uuid.UUID) error {
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

// IsAvailable checks if Claude Code is currently available.
func (r *ClaudeCodeRunner) IsAvailable(ctx context.Context) (bool, string) {
	if !r.available {
		return false, r.message
	}

	// Verify the binary still exists
	if _, err := os.Stat(r.binaryPath); os.IsNotExist(err) {
		return false, "claude CLI binary not found"
	}

	// Could add additional health checks here (API availability, etc.)
	return true, "claude CLI is available"
}

// buildArgs constructs command-line arguments for claude CLI.
func (r *ClaudeCodeRunner) buildArgs(req ExecuteRequest) []string {
	args := []string{
		"--print",
		"--output-format", "stream-json",
	}

	// Model selection
	if req.Profile != nil && req.Profile.Model != "" {
		args = append(args, "--model", req.Profile.Model)
	}

	// Allowed tools
	if req.Profile != nil && len(req.Profile.AllowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(req.Profile.AllowedTools, " "))
	}

	// Denied tools
	if req.Profile != nil && len(req.Profile.DeniedTools) > 0 {
		args = append(args, "--disallowed-tools", strings.Join(req.Profile.DeniedTools, " "))
	}

	// Skip permission prompts if configured (for sandboxed environments)
	if req.Profile != nil && req.Profile.SkipPermissionPrompt {
		args = append(args, "--dangerously-skip-permissions")
	}

	return args
}

// ClaudeStreamEvent represents a single event from Claude Code's stream-json output.
type ClaudeStreamEvent struct {
	Type      string          `json:"type"`
	Message   *ClaudeMessage  `json:"message,omitempty"`
	Usage     *ClaudeUsage    `json:"usage,omitempty"`
	ToolUse   *ClaudeToolUse  `json:"tool_use,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *ClaudeError    `json:"error,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
}

// ClaudeMessage represents a message in the Claude stream.
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeUsage represents token usage information.
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeToolUse represents a tool call in the stream.
type ClaudeToolUse struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ClaudeError represents an error in the stream.
type ClaudeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// parseStreamEvent parses a single line from Claude's stream-json output.
func (r *ClaudeCodeRunner) parseStreamEvent(runID uuid.UUID, line string) (*domain.RunEvent, error) {
	var streamEvent ClaudeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	switch streamEvent.Type {
	case "message":
		if streamEvent.Message != nil {
			return domain.NewMessageEvent(
				runID,
				streamEvent.Message.Role,
				streamEvent.Message.Content,
			), nil
		}

	case "assistant":
		// Assistant text content
		if streamEvent.Message != nil {
			return domain.NewMessageEvent(
				runID,
				"assistant",
				streamEvent.Message.Content,
			), nil
		}

	case "tool_use":
		if streamEvent.ToolUse != nil {
			var input map[string]interface{}
			if streamEvent.ToolUse.Input != nil {
				json.Unmarshal(streamEvent.ToolUse.Input, &input)
			}
			return domain.NewToolCallEvent(
				runID,
				streamEvent.ToolUse.Name,
				input,
			), nil
		}

	case "tool_result":
		// Parse tool result
		var resultStr string
		if streamEvent.Result != nil {
			json.Unmarshal(streamEvent.Result, &resultStr)
		}
		return domain.NewToolResultEvent(
			runID,
			"", // tool name not always available in result
			resultStr,
			nil,
		), nil

	case "error":
		if streamEvent.Error != nil {
			return domain.NewErrorEvent(
				runID,
				streamEvent.Error.Code,
				streamEvent.Error.Message,
				false,
			), nil
		}

	case "usage":
		if streamEvent.Usage != nil {
			// Emit as metric event
			return domain.NewMetricEvent(
				runID,
				"tokens",
				float64(streamEvent.Usage.InputTokens+streamEvent.Usage.OutputTokens),
				"tokens",
			), nil
		}
	}

	// Unknown or unhandled event type - log it
	return domain.NewLogEvent(
		runID,
		"debug",
		fmt.Sprintf("Unhandled event type: %s", streamEvent.Type),
	), nil
}

// updateMetrics updates execution metrics based on parsed events.
func (r *ClaudeCodeRunner) updateMetrics(event *domain.RunEvent, metrics *ExecutionMetrics, lastAssistant *string) {
	if event == nil {
		return
	}

	switch data := event.Data.(type) {
	case *domain.MessageEventData:
		if data.Role == "assistant" {
			*lastAssistant = data.Content
			metrics.TurnsUsed++
		}
	case *domain.ToolCallEventData:
		metrics.ToolCallCount++
	case *domain.MetricEventData:
		if data.Name == "tokens" {
			// This is cumulative usage
			totalTokens := int(data.Value)
			if totalTokens > metrics.TokensInput+metrics.TokensOutput {
				metrics.TokensOutput = totalTokens - metrics.TokensInput
			}
		}
	}
}

// Verify interface compliance
var _ Runner = (*ClaudeCodeRunner)(nil)

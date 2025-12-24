// Package runner provides runner adapter implementations.
//
// This file implements the OpenCode runner adapter for executing
// OpenCode via the resource-opencode wrapper within agent-manager.
package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
			"anthropic/claude-sonnet-4-5",
			"anthropic/claude-opus-4-5",
			"openai/gpt-4o",
			"openai/o4-mini",
			"google/gemini-2.0-flash",
			"deepseek/deepseek-chat",
		},
	}
}

// Execute runs OpenCode with the given configuration.
func (r *OpenCodeRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, &domain.RunnerError{
			RunnerType:  domain.RunnerTypeOpenCode,
			Operation:   "availability",
			Cause:       errors.New(r.message),
			IsTransient: false,
		}
	}

	startTime := time.Now()

	// Build command arguments
	// resource-opencode run passes through to opencode CLI
	// Correct syntax: resource-opencode run run <message> --format json
	args := r.buildArgs(req)

	// Create command using resource-opencode.
	// Prefix with env to surface the tag in the process command line for reconciler detection.
	tag := req.GetTag()
	envArgs := append([]string{fmt.Sprintf("OPENCODE_AGENT_TAG=%s", tag), r.binaryPath}, args...)
	cmd := exec.CommandContext(ctx, "env", envArgs...)
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
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeOpenCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeOpenCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Create stdin pipe and close it immediately after start
	// This signals to OpenCode that there's no additional input coming
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeOpenCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeOpenCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Close stdin immediately - we pass prompt via command line args
	stdin.Close()

	// Emit starting event
	if req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"OpenCode execution started",
		))
	}

	// Process streaming JSON output
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder
	stepFinished := false

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
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Check for step_finish event before parsing
		// OpenCode doesn't exit after step_finish in JSON mode, so we need to detect it
		if strings.Contains(line, `"type":"step_finish"`) {
			stepFinished = true
			// Handle step_finish specially - it may contain both message and cost data
			r.handleStepFinish(req.RunID, line, &metrics, &lastAssistantMessage, req.EventSink)
			break
		}

		// Parse the streaming event(s)
		events, err := r.parseStreamEvents(req.RunID, line)
		if err != nil {
			// Log parsing error but continue
			if req.EventSink != nil {
				_ = req.EventSink.Emit(domain.NewLogEvent(
					req.RunID,
					"warn",
					fmt.Sprintf("Failed to parse event: %v", err),
				))
			}
			continue
		}

		// Skip silently if parseStreamEvents returned no events (non-JSON lines)
		if len(events) == 0 {
			continue
		}

		for _, event := range events {
			if event == nil {
				continue
			}
			// Update metrics based on event
			r.updateMetrics(event, &metrics, &lastAssistantMessage)

			// Emit to sink
			if req.EventSink != nil {
				_ = req.EventSink.Emit(event)
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewLogEvent(
			req.RunID,
			"warn",
			fmt.Sprintf("OpenCode output scan error: %v", scanErr),
		))
	}

	// If step finished but process is still running, terminate it gracefully
	if stepFinished && cmd.Process != nil {
		// Give a brief moment for cleanup, then terminate
		time.Sleep(100 * time.Millisecond)
		_ = cmd.Process.Signal(os.Interrupt)
		// Wait briefly for graceful shutdown
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill if it doesn't exit gracefully
			_ = cmd.Process.Kill()
			<-done
		}
	}

	// Wait for command to complete (if not already done above)
	if !stepFinished {
		err = cmd.Wait()
	} else {
		err = nil // Step finished successfully
	}
	duration := time.Since(startTime)

	// Determine result
	result := &ExecuteResult{
		Duration: duration,
		Metrics:  metrics,
	}

	if err != nil {
		errorMessage := strings.TrimSpace(errorOutput.String())
		// Check if it was cancelled
		if ctx.Err() == context.Canceled {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = "execution cancelled"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.Success = false
			result.ExitCode = exitErr.ExitCode()
			if errorMessage == "" {
				errorMessage = exitErr.Error()
			}
			result.ErrorMessage = errorMessage
			if result.ExitCode != 0 && errorMessage != "" {
				result.ErrorMessage = fmt.Sprintf("exit code %d: %s", result.ExitCode, errorMessage)
			}
		} else {
			result.Success = false
			result.ExitCode = -1
			if errorMessage == "" {
				errorMessage = err.Error()
			}
			result.ErrorMessage = errorMessage
		}
		if req.EventSink != nil && strings.TrimSpace(result.ErrorMessage) != "" {
			_ = req.EventSink.Emit(domain.NewErrorEvent(
				req.RunID,
				"execution_error",
				result.ErrorMessage,
				false,
			))
		}
	} else {
		result.Success = true
		result.ExitCode = 0
		if lastAssistantMessage == "" {
			lastAssistantMessage = "OpenCode completed without assistant message."
			if req.EventSink != nil {
				_ = req.EventSink.Emit(domain.NewMessageEvent(req.RunID, "assistant", lastAssistantMessage))
			}
		}
		result.Summary = &domain.RunSummary{
			Description:  lastAssistantMessage,
			TurnsUsed:    metrics.TurnsUsed,
			TokensUsed:   TotalTokens(metrics),
			CostEstimate: metrics.CostEstimateUSD,
		}
	}

	// Emit completion event
	if req.EventSink != nil {
		finalStatus := string(domain.RunStatusComplete)
		if !result.Success {
			finalStatus = string(domain.RunStatusFailed)
		}
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusRunning),
			finalStatus,
			"OpenCode execution completed",
		))
		_ = req.EventSink.Close()
	}

	return result, nil
}

// Stop attempts to gracefully stop a running OpenCode instance.
func (r *OpenCodeRunner) Stop(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	cmd, exists := r.runs[runID]
	r.mu.Unlock()

	if !exists {
		return domain.NewNotFoundErrorWithID("Run", runID.String())
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

// buildArgs constructs command-line arguments for resource-opencode run.
func (r *OpenCodeRunner) buildArgs(req ExecuteRequest) []string {
	// resource-opencode run passes through to opencode CLI
	// Syntax: resource-opencode run run <message> [options]
	args := []string{
		"run", // resource-opencode subcommand
		"run", // opencode subcommand
		req.Prompt,
		"--format", "json", // Enable JSON output for event parsing
	}

	// Get the resolved config (handles profile + inline overrides)
	cfg := req.GetConfig()

	// Model selection via CLI flag
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	return args
}

// buildEnv constructs environment variables for resource-opencode run.
func (r *OpenCodeRunner) buildEnv(req ExecuteRequest) []string {
	env := os.Environ()

	// Non-interactive mode
	env = append(env, "OPENCODE_NON_INTERACTIVE=true")

	// Get the resolved config (handles profile + inline overrides)
	cfg := req.GetConfig()

	// Model selection via environment (backup to CLI flag)
	if cfg.Model != "" {
		env = append(env, fmt.Sprintf("OPENCODE_MODEL=%s", cfg.Model))
	}

	// Max turns
	if cfg.MaxTurns > 0 {
		env = append(env, fmt.Sprintf("MAX_TURNS=%d", cfg.MaxTurns))
	}

	// Timeout in seconds
	if cfg.Timeout > 0 {
		env = append(env, fmt.Sprintf("TIMEOUT=%d", int(cfg.Timeout.Seconds())))
	}

	// Allowed tools
	if len(cfg.AllowedTools) > 0 {
		env = append(env, fmt.Sprintf("ALLOWED_TOOLS=%s", strings.Join(cfg.AllowedTools, ",")))
	}

	// Add any custom environment from the request
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// =============================================================================
// OpenCode Stream Event Types
// =============================================================================

// OpenCodeStreamEvent represents a single event from OpenCode's JSON output.
// OpenCode format: {"type":"...", "timestamp":..., "sessionID":"...", "part":{...}}
type OpenCodeStreamEvent struct {
	Type      string         `json:"type"`
	Timestamp int64          `json:"timestamp,omitempty"`
	SessionID string         `json:"sessionID,omitempty"`
	Part      *OpenCodePart  `json:"part,omitempty"`
	Error     *OpenCodeError `json:"error,omitempty"`
}

// OpenCodePart represents the part field in OpenCode events.
// Contains the actual content/data for each event type.
type OpenCodePart struct {
	ID        string          `json:"id,omitempty"`
	SessionID string          `json:"sessionID,omitempty"`
	MessageID string          `json:"messageID,omitempty"`
	Type      string          `json:"type"` // "text", "step-start", "step-finish", "tool"
	Text      string          `json:"text,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	Snapshot  string          `json:"snapshot,omitempty"`
	Cost      float64         `json:"cost,omitempty"`
	Tokens    *OpenCodeTokens `json:"tokens,omitempty"`
	Name      string          `json:"name,omitempty"`  // Legacy tool name field
	Input     json.RawMessage `json:"input,omitempty"` // Legacy tool input field
	Output    string          `json:"output,omitempty"`
	IsError   bool            `json:"isError,omitempty"`
	Time      *OpenCodeTime   `json:"time,omitempty"`
	// New fields for actual OpenCode tool_use format
	Tool   string         `json:"tool,omitempty"`   // Actual tool name (e.g., "write", "bash")
	CallID string         `json:"callID,omitempty"` // Tool call ID
	State  *OpenCodeState `json:"state,omitempty"`  // Tool state with input/output
}

// OpenCodeState represents the state field in tool_use events.
type OpenCodeState struct {
	Status   string                 `json:"status,omitempty"`   // "pending", "completed", etc.
	Input    map[string]interface{} `json:"input,omitempty"`    // Tool input arguments
	Output   string                 `json:"output,omitempty"`   // Tool output
	Title    string                 `json:"title,omitempty"`    // Display title
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}

// OpenCodeTokens represents token usage in step_finish events.
type OpenCodeTokens struct {
	Input     int            `json:"input"`
	Output    int            `json:"output"`
	Reasoning int            `json:"reasoning,omitempty"`
	Cache     *OpenCodeCache `json:"cache,omitempty"`
}

// OpenCodeCache represents cache token usage.
type OpenCodeCache struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

// OpenCodeTime represents timing information.
type OpenCodeTime struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// OpenCodeError represents an error in the stream.
type OpenCodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// parseStreamEvent parses a single line from OpenCode's JSON output.
// OpenCode format: {"type":"...", "timestamp":..., "sessionID":"...", "part":{...}}
// Returns nil, nil for lines that should be silently skipped (non-JSON startup output).
// This preserves legacy behavior for tests by selecting a primary event.
func (r *OpenCodeRunner) parseStreamEvent(runID uuid.UUID, line string) (*domain.RunEvent, error) {
	events, err := r.parseStreamEvents(runID, line)
	if err != nil || len(events) == 0 {
		return nil, err
	}
	// Prefer tool results (completed tool_use) and metrics (step_finish) to preserve test expectations.
	var fallback *domain.RunEvent
	for _, event := range events {
		if event == nil {
			continue
		}
		if event.EventType == domain.EventTypeToolResult {
			return event, nil
		}
		if event.EventType == domain.EventTypeMetric {
			fallback = event
		}
		if fallback == nil {
			fallback = event
		}
	}
	return fallback, nil
}

// parseStreamEvents parses a single line from OpenCode's JSON output into one or more events.
// Returns an empty slice for lines that should be silently skipped (non-JSON startup output).
func (r *OpenCodeRunner) parseStreamEvents(runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	// Skip empty lines
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// Quick check: valid JSON must start with '{' or '['
	// This avoids logging warnings for non-JSON startup output
	if len(line) > 0 && line[0] != '{' && line[0] != '[' {
		// Silently skip non-JSON lines (startup messages, etc.)
		return nil, nil
	}

	if line[0] == '[' {
		var streamEvents []OpenCodeStreamEvent
		if err := json.Unmarshal([]byte(line), &streamEvents); err != nil {
			return nil, domain.NewInternalError("invalid opencode JSON", err)
		}
		events := []*domain.RunEvent{}
		for _, streamEvent := range streamEvents {
			parsed, err := r.parseOpenCodeStreamEvent(runID, streamEvent)
			if err != nil {
				return nil, err
			}
			events = append(events, parsed...)
		}
		return events, nil
	}

	var streamEvent OpenCodeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil, domain.NewInternalError("invalid opencode JSON", err)
	}

	return r.parseOpenCodeStreamEvent(runID, streamEvent)
}

func (r *OpenCodeRunner) parseOpenCodeStreamEvent(runID uuid.UUID, streamEvent OpenCodeStreamEvent) ([]*domain.RunEvent, error) {
	// Handle error events
	if streamEvent.Error != nil {
		code := streamEvent.Error.Code
		if code == "" {
			code = streamEvent.Error.Type
		}
		if code == "" {
			code = "execution_error"
		}
		return []*domain.RunEvent{domain.NewErrorEvent(runID, code, streamEvent.Error.Message, false)}, nil
	}

	// Handle events based on top-level type
	switch streamEvent.Type {
	case "step_start":
		// Session/step started
		return []*domain.RunEvent{domain.NewLogEvent(runID, "info", "OpenCode step started")}, nil

	case "text":
		// Text response from assistant
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", streamEvent.Part.Text)}, nil
		}

	case "tool_call", "tool_use", "tool-call":
		// Tool invocation - OpenCode uses "tool_use" with part.tool and part.state.input
		if streamEvent.Part != nil {
			// Get tool name - prefer part.tool (actual OpenCode format)
			toolName := streamEvent.Part.Tool
			if toolName == "" {
				toolName = streamEvent.Part.Name // Legacy fallback
			}
			if toolName == "" {
				toolName = "unknown_tool"
			}

			// Get input - prefer part.state.input (actual OpenCode format)
			input := make(map[string]interface{})
			if streamEvent.Part.State != nil && streamEvent.Part.State.Input != nil {
				input = streamEvent.Part.State.Input
			} else if streamEvent.Part.Input != nil {
				// Legacy fallback
				_ = json.Unmarshal(streamEvent.Part.Input, &input)
			}

			// Check if this is a completed tool call (OpenCode sometimes bundles result in same event)
			if streamEvent.Part.State != nil && streamEvent.Part.State.Status == "completed" {
				events := []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, input)}
				output := streamEvent.Part.State.Output
				if output == "" {
					output = streamEvent.Part.Output
				}
				toolCallID := streamEvent.Part.CallID
				var errMsg error
				if streamEvent.Part.IsError {
					errMsg = fmt.Errorf("%s", output)
				}
				events = append(events, domain.NewToolResultEvent(runID, toolName, toolCallID, output, errMsg))
				return events, nil
			}

			return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, input)}, nil
		}

	case "tool_result", "tool-result":
		// Tool execution result
		if streamEvent.Part != nil {
			// Get tool name - prefer part.tool (actual OpenCode format)
			toolName := streamEvent.Part.Tool
			if toolName == "" {
				toolName = streamEvent.Part.Name // Legacy fallback
			}

			// Get output - prefer part.state.output (actual OpenCode format)
			output := streamEvent.Part.Output
			if output == "" && streamEvent.Part.State != nil {
				output = streamEvent.Part.State.Output
			}

			toolCallID := streamEvent.Part.CallID
			var errMsg error
			events := []*domain.RunEvent{}
			if streamEvent.Part.State != nil && streamEvent.Part.State.Input != nil {
				events = append(events, domain.NewToolCallEvent(runID, toolName, streamEvent.Part.State.Input))
			}
			if streamEvent.Part.IsError {
				errMsg = fmt.Errorf("%s", output)
				events = append(events, domain.NewToolResultEvent(runID, toolName, toolCallID, "", errMsg))
				return events, nil
			}
			events = append(events, domain.NewToolResultEvent(runID, toolName, toolCallID, output, nil))
			return events, nil
		}

	case "step_finish":
		// Step completed - contains cost, token usage, and possibly the final message
		if streamEvent.Part != nil {
			// First, try to extract assistant message from Snapshot or Text
			// This will be returned and the cost event will be handled in updateMetrics
			// via the CostEventData that we embed
			costEvent, err := r.parseStepFinishEvent(runID, streamEvent.Part)
			if err != nil {
				return nil, err
			}
			events := []*domain.RunEvent{}
			if costEvent != nil {
				events = append(events, costEvent)
			}
			if msgEvent := r.extractAssistantMessage(runID, streamEvent.Part); msgEvent != nil {
				events = append(events, msgEvent)
			}
			return events, nil
		}

	case "error":
		// Error event
		if streamEvent.Part != nil && streamEvent.Part.IsError {
			return []*domain.RunEvent{domain.NewErrorEvent(runID, "execution_error", streamEvent.Part.Output, false)}, nil
		}

	case "user_message":
		// User message echo
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "user", streamEvent.Part.Text)}, nil
		}

	case "thinking":
		// Model reasoning/thinking
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", fmt.Sprintf("Thinking: %s", streamEvent.Part.Text))}, nil
		}

	case "assistant", "response", "message", "assistant_message":
		// Alternative event types for assistant messages (OpenCode may vary)
		if streamEvent.Part != nil {
			text := streamEvent.Part.Text
			if text == "" {
				text = streamEvent.Part.Output // Fallback to output field
			}
			if text != "" {
				return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", text)}, nil
			}
		}

	case "content", "content_block":
		// Content block events (similar to Claude Code format)
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			// Check if it's assistant content based on Part.Type
			role := "assistant"
			if streamEvent.Part.Type == "user" {
				role = "user"
			}
			return []*domain.RunEvent{domain.NewMessageEvent(runID, role, streamEvent.Part.Text)}, nil
		}
	}

	// Check Part.Type as secondary classification (nested type info)
	// This handles cases where streamEvent.Type is generic but Part.Type is specific
	// OpenCode uses part.type="tool" for tool invocations
	if streamEvent.Part != nil && streamEvent.Part.Type != "" {
		switch streamEvent.Part.Type {
		case "text", "assistant":
			if streamEvent.Part.Text != "" {
				return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", streamEvent.Part.Text)}, nil
			}
		case "tool", "tool-call", "tool_call", "tool_use":
			// OpenCode uses part.type="tool" with part.tool for the tool name
			toolName := streamEvent.Part.Tool
			if toolName == "" {
				toolName = streamEvent.Part.Name // Legacy fallback
			}
			if toolName == "" {
				toolName = "unknown_tool"
			}

			// Check if this is a completed tool (has result)
			if streamEvent.Part.State != nil && streamEvent.Part.State.Status == "completed" {
				events := []*domain.RunEvent{}
				if streamEvent.Part.State.Input != nil {
					events = append(events, domain.NewToolCallEvent(runID, toolName, streamEvent.Part.State.Input))
				}
				output := streamEvent.Part.State.Output
				if output == "" {
					output = streamEvent.Part.Output
				}
				toolCallID := streamEvent.Part.CallID
				var errMsg error
				if streamEvent.Part.IsError {
					errMsg = fmt.Errorf("%s", output)
				}
				events = append(events, domain.NewToolResultEvent(runID, toolName, toolCallID, output, errMsg))
				return events, nil
			}

			// Get input from state.input (actual OpenCode format)
			input := make(map[string]interface{})
			if streamEvent.Part.State != nil && streamEvent.Part.State.Input != nil {
				input = streamEvent.Part.State.Input
			} else if streamEvent.Part.Input != nil {
				_ = json.Unmarshal(streamEvent.Part.Input, &input)
			}
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, input)}, nil
		case "tool-result", "tool_result":
			toolName := streamEvent.Part.Tool
			if toolName == "" {
				toolName = streamEvent.Part.Name
			}
			output := streamEvent.Part.Output
			if output == "" && streamEvent.Part.State != nil {
				output = streamEvent.Part.State.Output
			}
			toolCallID := streamEvent.Part.CallID
			var errMsg error
			if streamEvent.Part.IsError {
				errMsg = fmt.Errorf("%s", output)
			}
			events := []*domain.RunEvent{}
			if streamEvent.Part.State != nil && streamEvent.Part.State.Input != nil {
				events = append(events, domain.NewToolCallEvent(runID, toolName, streamEvent.Part.State.Input))
			}
			events = append(events, domain.NewToolResultEvent(runID, toolName, toolCallID, output, errMsg))
			return events, nil
		}
	}

	// Unknown or unhandled event type - log it for debugging
	return []*domain.RunEvent{domain.NewLogEvent(
		runID,
		"debug",
		fmt.Sprintf("OpenCode event [%s]", streamEvent.Type),
	)}, nil
}

// parseStepFinishEvent handles the step_finish event with cost/token data.
// Returns multiple events: a message event (if snapshot available) and a cost event.
func (r *OpenCodeRunner) parseStepFinishEvent(runID uuid.UUID, part *OpenCodePart) (*domain.RunEvent, error) {
	// Extract token and cost information
	var inputTokens, outputTokens, cacheRead, cacheWrite int
	if part.Tokens != nil {
		inputTokens = part.Tokens.Input
		outputTokens = part.Tokens.Output
		if part.Tokens.Cache != nil {
			cacheRead = part.Tokens.Cache.Read
			cacheWrite = part.Tokens.Cache.Write
		}
	}

	// Create cost event with all available data
	costEvent := &domain.RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: domain.EventTypeMetric,
		Timestamp: time.Now(),
		Data: &domain.CostEventData{
			InputTokens:         inputTokens,
			OutputTokens:        outputTokens,
			CacheCreationTokens: cacheWrite,
			CacheReadTokens:     cacheRead,
			TotalCostUSD:        part.Cost,
		},
	}
	return costEvent, nil
}

// extractAssistantMessage tries to extract the final assistant message from step_finish.
// OpenCode stores the complete assistant response in the Snapshot field.
func (r *OpenCodeRunner) extractAssistantMessage(runID uuid.UUID, part *OpenCodePart) *domain.RunEvent {
	// Prefer text or output if available.
	if part.Text != "" {
		return domain.NewMessageEvent(runID, "assistant", part.Text)
	}
	if part.Output != "" {
		return domain.NewMessageEvent(runID, "assistant", part.Output)
	}

	// Snapshot sometimes contains a hash instead of content.
	if part.Snapshot != "" && !isLikelyHash(part.Snapshot) {
		return domain.NewMessageEvent(runID, "assistant", part.Snapshot)
	}
	return nil
}

// handleStepFinish processes a step_finish event, emitting both message and cost events.
// OpenCode's step_finish contains the final assistant message (in Snapshot) and token/cost data.
func (r *OpenCodeRunner) handleStepFinish(runID uuid.UUID, line string, metrics *ExecutionMetrics, lastAssistant *string, sink EventSink) {
	// Parse the step_finish event
	var streamEvent OpenCodeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return
	}

	if streamEvent.Part == nil {
		return
	}

	part := streamEvent.Part

	// 1. Extract and emit assistant message if available
	if msgEvent := r.extractAssistantMessage(runID, part); msgEvent != nil {
		r.updateMetrics(msgEvent, metrics, lastAssistant)
		if sink != nil {
			_ = sink.Emit(msgEvent)
		}
	}

	// 2. Extract and emit cost/token metrics
	costEvent, err := r.parseStepFinishEvent(runID, part)
	if err == nil && costEvent != nil {
		r.updateMetrics(costEvent, metrics, lastAssistant)
		if sink != nil {
			_ = sink.Emit(costEvent)
		}
	}
}

// updateMetrics updates execution metrics based on parsed events.
func (r *OpenCodeRunner) updateMetrics(event *domain.RunEvent, metrics *ExecutionMetrics, lastAssistant *string) {
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
	case *domain.ToolResultEventData:
		// OpenCode bundles tool call and result in one event with status:"completed"
		// Count these as tool calls for metrics purposes
		metrics.ToolCallCount++
	case *domain.MetricEventData:
		if data.Name == "tokens" {
			totalTokens := int(data.Value)
			if totalTokens > metrics.TokensInput+metrics.TokensOutput {
				metrics.TokensOutput = totalTokens - metrics.TokensInput
			}
		} else if data.Name == "cost" {
			metrics.CostEstimateUSD = data.Value
		}
	case *domain.CostEventData:
		metrics.TokensInput = data.InputTokens
		metrics.TokensOutput = data.OutputTokens
		metrics.CacheReadTokens = data.CacheReadTokens
		metrics.CacheCreationTokens = data.CacheCreationTokens
		metrics.CostEstimateUSD = data.TotalCostUSD
	}
}

func isLikelyHash(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if len(trimmed) != 40 && len(trimmed) != 64 {
		return false
	}
	for _, r := range trimmed {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

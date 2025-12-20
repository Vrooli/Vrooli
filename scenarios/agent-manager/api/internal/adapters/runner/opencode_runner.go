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
		return nil, fmt.Errorf("opencode runner is not available: %s", r.message)
	}

	startTime := time.Now()

	// Build command arguments
	// resource-opencode run passes through to opencode CLI
	// Correct syntax: resource-opencode run run <message> --format json
	args := r.buildArgs(req)

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

	// Create stdin pipe and close it immediately after start
	// This signals to OpenCode that there's no additional input coming
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start resource-opencode: %w", err)
	}

	// Close stdin immediately - we pass prompt via command line args
	stdin.Close()

	// Emit starting event
	if req.EventSink != nil {
		req.EventSink.Emit(domain.NewStatusEvent(
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

		// Skip silently if parseStreamEvent returned nil, nil (non-JSON lines)
		if event == nil {
			continue
		}

		// Update metrics based on event
		r.updateMetrics(event, &metrics, &lastAssistantMessage)

		// Emit to sink
		if req.EventSink != nil {
			req.EventSink.Emit(event)
		}
	}

	// If step finished but process is still running, terminate it gracefully
	if stepFinished && cmd.Process != nil {
		// Give a brief moment for cleanup, then terminate
		time.Sleep(100 * time.Millisecond)
		cmd.Process.Signal(os.Interrupt)
		// Wait briefly for graceful shutdown
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill if it doesn't exit gracefully
			cmd.Process.Kill()
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
			Description:  lastAssistantMessage,
			TurnsUsed:    metrics.TurnsUsed,
			TokensUsed:   metrics.TokensInput + metrics.TokensOutput,
			CostEstimate: metrics.CostEstimateUSD,
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

// buildArgs constructs command-line arguments for resource-opencode run.
func (r *OpenCodeRunner) buildArgs(req ExecuteRequest) []string {
	// resource-opencode run passes through to opencode CLI
	// Syntax: resource-opencode run run <message> [options]
	args := []string{
		"run",  // resource-opencode subcommand
		"run",  // opencode subcommand
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
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp,omitempty"`
	SessionID string          `json:"sessionID,omitempty"`
	Part      *OpenCodePart   `json:"part,omitempty"`
	Error     *OpenCodeError  `json:"error,omitempty"`
}

// OpenCodePart represents the part field in OpenCode events.
// Contains the actual content/data for each event type.
type OpenCodePart struct {
	ID        string            `json:"id,omitempty"`
	SessionID string            `json:"sessionID,omitempty"`
	MessageID string            `json:"messageID,omitempty"`
	Type      string            `json:"type"` // "text", "step-start", "step-finish", "tool"
	Text      string            `json:"text,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Snapshot  string            `json:"snapshot,omitempty"`
	Cost      float64           `json:"cost,omitempty"`
	Tokens    *OpenCodeTokens   `json:"tokens,omitempty"`
	Name      string            `json:"name,omitempty"`  // Legacy tool name field
	Input     json.RawMessage   `json:"input,omitempty"` // Legacy tool input field
	Output    string            `json:"output,omitempty"`
	IsError   bool              `json:"isError,omitempty"`
	Time      *OpenCodeTime     `json:"time,omitempty"`
	// New fields for actual OpenCode tool_use format
	Tool   string          `json:"tool,omitempty"`   // Actual tool name (e.g., "write", "bash")
	CallID string          `json:"callID,omitempty"` // Tool call ID
	State  *OpenCodeState  `json:"state,omitempty"`  // Tool state with input/output
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
	Input     int             `json:"input"`
	Output    int             `json:"output"`
	Reasoning int             `json:"reasoning,omitempty"`
	Cache     *OpenCodeCache  `json:"cache,omitempty"`
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
func (r *OpenCodeRunner) parseStreamEvent(runID uuid.UUID, line string) (*domain.RunEvent, error) {
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

	var streamEvent OpenCodeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Handle error events
	if streamEvent.Error != nil {
		code := streamEvent.Error.Code
		if code == "" {
			code = streamEvent.Error.Type
		}
		if code == "" {
			code = "execution_error"
		}
		return domain.NewErrorEvent(runID, code, streamEvent.Error.Message, false), nil
	}

	// Handle events based on top-level type
	switch streamEvent.Type {
	case "step_start":
		// Session/step started
		return domain.NewLogEvent(runID, "info", "OpenCode step started"), nil

	case "text":
		// Text response from assistant
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return domain.NewMessageEvent(runID, "assistant", streamEvent.Part.Text), nil
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
				json.Unmarshal(streamEvent.Part.Input, &input)
			}

			// Check if this is a completed tool call (OpenCode sometimes bundles result in same event)
			if streamEvent.Part.State != nil && streamEvent.Part.State.Status == "completed" {
				// This tool_call includes the result - emit as tool_result instead
				output := streamEvent.Part.State.Output
				if output == "" {
					output = streamEvent.Part.Output
				}
				toolCallID := streamEvent.Part.CallID
				var errMsg error
				if streamEvent.Part.IsError {
					errMsg = fmt.Errorf("%s", output)
				}
				return domain.NewToolResultEvent(runID, toolName, toolCallID, output, errMsg), nil
			}

			return domain.NewToolCallEvent(runID, toolName, input), nil
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
			if streamEvent.Part.IsError {
				errMsg = fmt.Errorf("%s", output)
				return domain.NewToolResultEvent(runID, toolName, toolCallID, "", errMsg), nil
			}
			return domain.NewToolResultEvent(runID, toolName, toolCallID, output, nil), nil
		}

	case "step_finish":
		// Step completed - contains cost, token usage, and possibly the final message
		if streamEvent.Part != nil {
			// First, try to extract assistant message from Snapshot or Text
			// This will be returned and the cost event will be handled in updateMetrics
			// via the CostEventData that we embed
			if msgEvent := r.extractAssistantMessage(runID, streamEvent.Part); msgEvent != nil {
				// Return message event - cost data is already in metrics from step_finish
				// We'll handle cost separately via parseStepFinishEvent called after
				return msgEvent, nil
			}
			// If no message, return the cost event
			return r.parseStepFinishEvent(runID, streamEvent.Part)
		}

	case "error":
		// Error event
		if streamEvent.Part != nil && streamEvent.Part.IsError {
			return domain.NewErrorEvent(runID, "execution_error", streamEvent.Part.Output, false), nil
		}

	case "user_message":
		// User message echo
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return domain.NewMessageEvent(runID, "user", streamEvent.Part.Text), nil
		}

	case "thinking":
		// Model reasoning/thinking
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return domain.NewLogEvent(runID, "debug", fmt.Sprintf("Thinking: %s", streamEvent.Part.Text)), nil
		}

	case "assistant", "response", "message", "assistant_message":
		// Alternative event types for assistant messages (OpenCode may vary)
		if streamEvent.Part != nil {
			text := streamEvent.Part.Text
			if text == "" {
				text = streamEvent.Part.Output // Fallback to output field
			}
			if text != "" {
				return domain.NewMessageEvent(runID, "assistant", text), nil
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
			return domain.NewMessageEvent(runID, role, streamEvent.Part.Text), nil
		}
	}

	// Check Part.Type as secondary classification (nested type info)
	// This handles cases where streamEvent.Type is generic but Part.Type is specific
	// OpenCode uses part.type="tool" for tool invocations
	if streamEvent.Part != nil && streamEvent.Part.Type != "" {
		switch streamEvent.Part.Type {
		case "text", "assistant":
			if streamEvent.Part.Text != "" {
				return domain.NewMessageEvent(runID, "assistant", streamEvent.Part.Text), nil
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
				output := streamEvent.Part.State.Output
				if output == "" {
					output = streamEvent.Part.Output
				}
				toolCallID := streamEvent.Part.CallID
				var errMsg error
				if streamEvent.Part.IsError {
					errMsg = fmt.Errorf("%s", output)
				}
				return domain.NewToolResultEvent(runID, toolName, toolCallID, output, errMsg), nil
			}

			// Get input from state.input (actual OpenCode format)
			input := make(map[string]interface{})
			if streamEvent.Part.State != nil && streamEvent.Part.State.Input != nil {
				input = streamEvent.Part.State.Input
			} else if streamEvent.Part.Input != nil {
				json.Unmarshal(streamEvent.Part.Input, &input)
			}
			return domain.NewToolCallEvent(runID, toolName, input), nil
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
			return domain.NewToolResultEvent(runID, toolName, toolCallID, output, errMsg), nil
		}
	}

	// Unknown or unhandled event type - log it for debugging
	return domain.NewLogEvent(
		runID,
		"debug",
		fmt.Sprintf("OpenCode event [%s]", streamEvent.Type),
	), nil
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
	// Check Snapshot field - contains the full assistant response
	if part.Snapshot != "" {
		return domain.NewMessageEvent(runID, "assistant", part.Snapshot)
	}
	// Fallback to Text field
	if part.Text != "" {
		return domain.NewMessageEvent(runID, "assistant", part.Text)
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
			sink.Emit(msgEvent)
		}
	}

	// 2. Extract and emit cost/token metrics
	costEvent, err := r.parseStepFinishEvent(runID, part)
	if err == nil && costEvent != nil {
		r.updateMetrics(costEvent, metrics, lastAssistant)
		if sink != nil {
			sink.Emit(costEvent)
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
		metrics.CostEstimateUSD = data.TotalCostUSD
	}
}

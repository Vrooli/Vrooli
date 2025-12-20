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

		// If we received step_finish, break out of the loop
		// OpenCode has completed but doesn't exit in JSON mode
		if stepFinished {
			break
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
	Type      string            `json:"type"` // "text", "step-start", "step-finish", "tool-call", "tool-result"
	Text      string            `json:"text,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Snapshot  string            `json:"snapshot,omitempty"`
	Cost      float64           `json:"cost,omitempty"`
	Tokens    *OpenCodeTokens   `json:"tokens,omitempty"`
	Name      string            `json:"name,omitempty"`  // Tool name
	Input     json.RawMessage   `json:"input,omitempty"` // Tool input
	Output    string            `json:"output,omitempty"`
	IsError   bool              `json:"isError,omitempty"`
	Time      *OpenCodeTime     `json:"time,omitempty"`
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

	case "tool_call", "tool_use":
		// Tool invocation (OpenCode may use either "tool_call" or "tool_use")
		if streamEvent.Part != nil {
			var input map[string]interface{}
			if streamEvent.Part.Input != nil {
				json.Unmarshal(streamEvent.Part.Input, &input)
			}
			return domain.NewToolCallEvent(runID, streamEvent.Part.Name, input), nil
		}

	case "tool_result", "tool-result":
		// Tool execution result
		if streamEvent.Part != nil {
			var errMsg error
			if streamEvent.Part.IsError {
				errMsg = fmt.Errorf("%s", streamEvent.Part.Output)
				return domain.NewToolResultEvent(runID, streamEvent.Part.Name, "", errMsg), nil
			}
			return domain.NewToolResultEvent(runID, streamEvent.Part.Name, streamEvent.Part.Output, nil), nil
		}

	case "step_finish":
		// Step completed - contains cost and token usage
		if streamEvent.Part != nil {
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
	}

	// Unknown or unhandled event type - log it for debugging
	return domain.NewLogEvent(
		runID,
		"debug",
		fmt.Sprintf("OpenCode event [%s]", streamEvent.Type),
	), nil
}

// parseStepFinishEvent handles the step_finish event with cost/token data.
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

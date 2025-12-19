// Package runner provides runner adapter implementations.
//
// This file implements the Claude Code runner adapter for executing
// Claude Code via the resource-claude-code wrapper within agent-manager.
package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ResourceCommand is the Vrooli resource wrapper command
const ClaudeCodeResourceCommand = "resource-claude-code"

// =============================================================================
// Claude Code Runner Implementation
// =============================================================================

// ClaudeCodeRunner implements the Runner interface for Claude Code CLI.
type ClaudeCodeRunner struct {
	binaryPath  string
	available   bool
	message     string
	installHint string
	mu          sync.Mutex
	runs        map[uuid.UUID]*exec.Cmd
}

// NewClaudeCodeRunner creates a new Claude Code runner.
func NewClaudeCodeRunner() (*ClaudeCodeRunner, error) {
	// Look for resource-claude-code in PATH (the Vrooli wrapper)
	binaryPath, err := exec.LookPath(ClaudeCodeResourceCommand)
	if err != nil {
		return &ClaudeCodeRunner{
			available:   false,
			message:     "resource-claude-code not found in PATH",
			installHint: "Run: vrooli resource install claude-code",
			runs:        make(map[uuid.UUID]*exec.Cmd),
		}, nil
	}

	// Verify the resource is healthy by checking status
	runner := &ClaudeCodeRunner{
		binaryPath: binaryPath,
		available:  true,
		message:    "resource-claude-code available",
		runs:       make(map[uuid.UUID]*exec.Cmd),
	}

	// Quick health check via status command
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "status", "--format", "json", "--fast")
	output, err := cmd.Output()
	if err != nil {
		runner.available = false
		runner.message = fmt.Sprintf("resource-claude-code status check failed: %v", err)
		runner.installHint = "Run: resource-claude-code manage install"
		return runner, nil
	}

	// Parse JSON status to check health
	var statusData map[string]interface{}
	if err := json.Unmarshal(output, &statusData); err == nil {
		if healthy, ok := statusData["healthy"].(string); ok && healthy != "true" {
			runner.available = false
			if msg, ok := statusData["health_message"].(string); ok {
				runner.message = msg
			} else {
				runner.message = "resource-claude-code is not healthy"
			}
			runner.installHint = "Run: resource-claude-code manage install"
		}
	}

	return runner, nil
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

	// Create command using resource-claude-code
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Dir = req.WorkingDir

	// Set environment using buildEnv (handles all configuration via env vars)
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

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start resource-claude-code: %w", err)
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
		msg := r.message
		if r.installHint != "" {
			msg += ". " + r.installHint
		}
		return false, msg
	}

	// Verify the binary still exists
	if _, err := os.Stat(r.binaryPath); os.IsNotExist(err) {
		return false, "resource-claude-code binary not found. Run: vrooli resource install claude-code"
	}

	return true, "resource-claude-code is available"
}

// InstallHint returns instructions for installing this runner.
func (r *ClaudeCodeRunner) InstallHint() string {
	return r.installHint
}

// buildArgs constructs command-line arguments for resource-claude-code run.
func (r *ClaudeCodeRunner) buildArgs(req ExecuteRequest) []string {
	// Use the "run" subcommand with --tag for agent tracking
	// Tag defaults to RunID if not set, but can be customized for readability
	args := []string{
		"run",
		"--tag", req.GetTag(),
		"-", // Read prompt from stdin
	}
	return args
}

// buildEnv constructs environment variables for resource-claude-code run.
func (r *ClaudeCodeRunner) buildEnv(req ExecuteRequest) []string {
	env := os.Environ()

	// Output format - use stream-json for event streaming
	env = append(env, "OUTPUT_FORMAT=stream-json")

	// Non-interactive mode for autonomous execution
	env = append(env, "CLAUDE_NON_INTERACTIVE=true")

	// Get the resolved config (handles profile + inline overrides)
	cfg := req.GetConfig()

	// Model selection via environment
	if cfg.Model != "" {
		env = append(env, fmt.Sprintf("CLAUDE_MODEL=%s", cfg.Model))
	}

	// Max turns
	if cfg.MaxTurns > 0 {
		env = append(env, fmt.Sprintf("MAX_TURNS=%d", cfg.MaxTurns))
	} else {
		env = append(env, "MAX_TURNS=30") // Default
	}

	// Timeout in seconds
	if cfg.Timeout > 0 {
		env = append(env, fmt.Sprintf("TIMEOUT=%d", int(cfg.Timeout.Seconds())))
	}

	// Allowed tools
	if len(cfg.AllowedTools) > 0 {
		env = append(env, fmt.Sprintf("ALLOWED_TOOLS=%s", strings.Join(cfg.AllowedTools, ",")))
	}

	// Skip permission prompts if configured (for sandboxed environments)
	if cfg.SkipPermissionPrompt {
		env = append(env, "SKIP_PERMISSIONS=yes")
	}

	// Add any custom environment from the request
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// ClaudeStreamEvent represents a single event from Claude Code's stream-json output.
type ClaudeStreamEvent struct {
	Type         string              `json:"type"`
	Subtype      string              `json:"subtype,omitempty"` // e.g., "success", "error"
	Message      *ClaudeMessage      `json:"message,omitempty"`
	Usage        *ClaudeUsage        `json:"usage,omitempty"`
	ToolUse      *ClaudeToolUse      `json:"tool_use,omitempty"`
	Result       json.RawMessage     `json:"result,omitempty"`
	Error        *ClaudeError        `json:"error,omitempty"`
	SessionID    string              `json:"session_id,omitempty"`
	IsError      bool                `json:"is_error,omitempty"`
	DurationMs   int                 `json:"duration_ms,omitempty"`
	DurationAPI  int                 `json:"duration_api_ms,omitempty"`
	NumTurns     int                 `json:"num_turns,omitempty"`
	TotalCostUSD float64             `json:"total_cost_usd,omitempty"`
	ServiceTier  string              `json:"service_tier,omitempty"` // e.g., "standard"
	ContentBlock *ClaudeContentBlock `json:"content_block,omitempty"`
	Delta        *ClaudeDelta        `json:"delta,omitempty"`
}

// ClaudeMessage represents a message in the Claude stream.
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeUsage represents detailed token usage information.
type ClaudeUsage struct {
	InputTokens              int               `json:"input_tokens"`
	OutputTokens             int               `json:"output_tokens"`
	CacheCreationInputTokens int               `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int               `json:"cache_read_input_tokens,omitempty"`
	ServerToolUse            *ClaudeServerTool `json:"server_tool_use,omitempty"`
}

// ClaudeServerTool represents server-side tool usage.
type ClaudeServerTool struct {
	WebSearchRequests int `json:"web_search_requests,omitempty"`
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

// ClaudeContentBlock represents a content block in streaming.
type ClaudeContentBlock struct {
	Type string `json:"type"` // "text", "tool_use"
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Text string `json:"text,omitempty"`
}

// ClaudeDelta represents incremental updates in streaming.
type ClaudeDelta struct {
	Type        string `json:"type"` // "text_delta", "input_json_delta"
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// RateLimitInfo contains parsed rate limit information.
type RateLimitInfo struct {
	Detected   bool
	LimitType  string // "5_hour", "daily", "weekly", "token"
	ResetTime  *time.Time
	RetryAfter int // seconds
	Message    string
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

	case "result":
		// Final result event - contains cost, usage, and potential rate limits
		return r.parseResultEvent(runID, &streamEvent)

	case "content_block_start":
		// Start of a content block (text or tool use)
		if streamEvent.ContentBlock != nil {
			if streamEvent.ContentBlock.Type == "tool_use" {
				return domain.NewLogEvent(
					runID,
					"debug",
					fmt.Sprintf("Starting tool: %s", streamEvent.ContentBlock.Name),
				), nil
			}
		}

	case "content_block_delta":
		// Incremental content update (for streaming)
		if streamEvent.Delta != nil && streamEvent.Delta.Text != "" {
			return domain.NewLogEvent(
				runID,
				"trace",
				streamEvent.Delta.Text,
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

// parseResultEvent handles the final "result" event which contains cost and rate limit info.
func (r *ClaudeCodeRunner) parseResultEvent(runID uuid.UUID, event *ClaudeStreamEvent) (*domain.RunEvent, error) {
	// Check for rate limit error in result
	if event.IsError {
		var resultStr string
		if event.Result != nil {
			json.Unmarshal(event.Result, &resultStr)
		}

		// Check for rate limit pattern: "Claude AI usage limit reached|timestamp"
		rateLimitInfo := r.detectRateLimit(resultStr)
		if rateLimitInfo.Detected {
			return domain.NewRateLimitEvent(
				runID,
				rateLimitInfo.LimitType,
				rateLimitInfo.Message,
				rateLimitInfo.ResetTime,
				rateLimitInfo.RetryAfter,
			), nil
		}

		// Generic error
		return domain.NewErrorEvent(
			runID,
			"execution_error",
			resultStr,
			false,
		), nil
	}

	// Successful result - emit cost event if we have usage data
	if event.Usage != nil || event.TotalCostUSD > 0 {
		costEvent := &domain.RunEvent{
			ID:        uuid.New(),
			RunID:     runID,
			EventType: domain.EventTypeMetric,
			Timestamp: time.Now(),
			Data: &domain.CostEventData{
				InputTokens:         event.Usage.InputTokens,
				OutputTokens:        event.Usage.OutputTokens,
				CacheCreationTokens: event.Usage.CacheCreationInputTokens,
				CacheReadTokens:     event.Usage.CacheReadInputTokens,
				TotalCostUSD:        event.TotalCostUSD,
				ServiceTier:         event.ServiceTier,
			},
		}
		if event.Usage.ServerToolUse != nil {
			if data, ok := costEvent.Data.(*domain.CostEventData); ok {
				data.WebSearchRequests = event.Usage.ServerToolUse.WebSearchRequests
			}
		}
		return costEvent, nil
	}

	// Result with no special data
	return domain.NewLogEvent(
		runID,
		"info",
		fmt.Sprintf("Execution completed in %d turns", event.NumTurns),
	), nil
}

// detectRateLimit parses rate limit information from error messages.
func (r *ClaudeCodeRunner) detectRateLimit(resultStr string) RateLimitInfo {
	info := RateLimitInfo{
		Detected: false,
		Message:  resultStr,
	}

	// Pattern 1: "Claude AI usage limit reached|timestamp"
	if strings.Contains(resultStr, "usage limit reached") || strings.Contains(resultStr, "rate limit") {
		info.Detected = true
		info.LimitType = "5_hour" // Most common limit type

		// Try to parse reset timestamp from "limit reached|1755806400" format
		parts := strings.Split(resultStr, "|")
		if len(parts) >= 2 {
			if timestamp, err := strconv.ParseInt(strings.TrimSpace(parts[len(parts)-1]), 10, 64); err == nil {
				resetTime := time.Unix(timestamp, 0)
				info.ResetTime = &resetTime
				info.RetryAfter = int(time.Until(resetTime).Seconds())
				if info.RetryAfter < 0 {
					info.RetryAfter = 0
				}
			}
		}

		// Determine limit type from message content
		lowerMsg := strings.ToLower(resultStr)
		if strings.Contains(lowerMsg, "daily") {
			info.LimitType = "daily"
		} else if strings.Contains(lowerMsg, "weekly") {
			info.LimitType = "weekly"
		} else if strings.Contains(lowerMsg, "token") {
			info.LimitType = "token"
		}
	}

	return info
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
	case *domain.CostEventData:
		// Update detailed token counts and cost
		metrics.TokensInput = data.InputTokens
		metrics.TokensOutput = data.OutputTokens
		metrics.CostEstimateUSD = data.TotalCostUSD
	case *domain.RateLimitEventData:
		// Rate limit detected - this will cause execution to fail
		// The error is handled in the Execute function
	}
}

// Verify interface compliance
var _ Runner = (*ClaudeCodeRunner)(nil)

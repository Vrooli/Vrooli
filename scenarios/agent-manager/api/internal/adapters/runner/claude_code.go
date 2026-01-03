// Package runner provides runner adapter implementations.
//
// This file implements the Claude Code runner adapter for executing
// Claude Code via the resource-claude-code wrapper within agent-manager.
package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
	streamState map[uuid.UUID]*claudeStreamState
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
			streamState: make(map[uuid.UUID]*claudeStreamState),
		}, nil
	}

	// Verify the resource is healthy by checking status
	runner := &ClaudeCodeRunner{
		binaryPath:  binaryPath,
		available:   true,
		message:     "resource-claude-code available",
		runs:        make(map[uuid.UUID]*exec.Cmd),
		streamState: make(map[uuid.UUID]*claudeStreamState),
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
		// Check health status - handle both boolean and string formats
		isHealthy := true
		if healthy, ok := statusData["healthy"].(bool); ok {
			isHealthy = healthy
		} else if healthyStr, ok := statusData["healthy"].(string); ok {
			isHealthy = healthyStr == "true"
		}

		if !isHealthy {
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
			"claude-haiku-4-5-20251001",
		},
	}
}

// Execute runs Claude Code with the given configuration.
func (r *ClaudeCodeRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, &domain.RunnerError{
			RunnerType:  domain.RunnerTypeClaudeCode,
			Operation:   "availability",
			Cause:       errors.New(r.message),
			IsTransient: false,
		}
	}

	startTime := time.Now()
	r.initStreamState(req.RunID)
	defer r.clearStreamState(req.RunID)

	// Build command arguments
	args := r.buildArgs(req)

	// Create command using resource-claude-code
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Dir = req.WorkingDir

	// Set environment using buildEnv (handles all configuration via env vars)
	cmd.Env = r.buildEnv(req)

	// Create a new process group so we can kill the entire subprocess tree
	// This is needed because resource-claude-code is a bash wrapper that may have
	// child processes (tee, cleanup handlers) that outlive the main Claude process
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
			RunnerType: domain.RunnerTypeClaudeCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeClaudeCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeClaudeCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeClaudeCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Emit starting event
	if req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"Claude Code execution started",
		))
	}

	// Write prompt and close stdin
	if _, err := stdin.Write([]byte(req.Prompt)); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeClaudeCode,
			Operation:  "execute",
			Cause:      err,
		}
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
	// Use a larger buffer for the scanner - Claude's stream-json can output very long lines
	// when reading large files or returning tool results (default is 64KB, we use 10MB)
	const maxScannerBuffer = 10 * 1024 * 1024 // 10MB
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), maxScannerBuffer)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

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

	// Check if scanner exited due to an error (vs clean EOF)
	scannerErr := scanner.Err()
	if scannerErr != nil {
		// Scanner hit an error - log it but continue to wait for process
		if req.EventSink != nil {
			_ = req.EventSink.Emit(domain.NewLogEvent(
				req.RunID,
				"warn",
				fmt.Sprintf("Scanner error (possible buffer overflow or I/O error): %v", scannerErr),
			))
		}
	}

	// Wait for command to complete with timeout
	// The bash wrapper script may hang after Claude exits (cleanup handlers, tee, etc.)
	// so we wait with a timeout and kill the process group if it doesn't exit cleanly
	const wrapperCleanupTimeout = 30 * time.Second

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case err = <-waitDone:
		// Normal completion - wrapper script exited cleanly
	case <-time.After(wrapperCleanupTimeout):
		// Wrapper script is stuck - kill the entire process group
		if cmd.Process != nil {
			// Kill the process group (negative PID kills the group)
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		// Wait for the killed process to be reaped
		err = <-waitDone
		// Log that we had to force kill
		if req.EventSink != nil {
			_ = req.EventSink.Emit(domain.NewLogEvent(
				req.RunID,
				"warn",
				"Wrapper script did not exit cleanly after stdout closed; killed process group",
			))
		}
	case <-ctx.Done():
		// Context cancelled - kill the process group
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		err = <-waitDone
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
			"Claude Code execution completed",
		))
		_ = req.EventSink.Close()
	}

	return result, nil
}

// Stop attempts to gracefully stop a running Claude Code instance.
// Uses process group signals to ensure the entire subprocess tree is stopped.
func (r *ClaudeCodeRunner) Stop(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	cmd, exists := r.runs[runID]
	r.mu.Unlock()

	if !exists {
		return domain.NewNotFoundErrorWithID("Run", runID.String())
	}

	if cmd.Process == nil {
		return nil
	}

	// Try graceful termination first (SIGTERM to process group)
	// Negative PID sends signal to the entire process group
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill the process group
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
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

type claudeStreamState struct {
	textBuffer     strings.Builder
	toolUseActive  bool
	toolUseID      string
	toolUseName    string
	toolUsePayload strings.Builder
	lastAssistant  string
}

// ClaudeMessage represents a message in the Claude stream.
// Content can be either a string or an array of content blocks.
type ClaudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // Can be string or []ContentBlock
}

// ClaudeContentItem represents a single item in a content array.
type ClaudeContentItem struct {
	Type      string          `json:"type"`                  // "text", "tool_use", "tool_result"
	Text      string          `json:"text,omitempty"`        // For text blocks
	ID        string          `json:"id,omitempty"`          // For tool_use blocks
	Name      string          `json:"name,omitempty"`        // For tool_use blocks
	Input     json.RawMessage `json:"input,omitempty"`       // For tool_use blocks
	ToolUseID string          `json:"tool_use_id,omitempty"` // For tool_result blocks
	Content   string          `json:"content,omitempty"`     // For tool_result blocks
}

// ExtractTextContent extracts text content from a ClaudeMessage.
// Handles both string content and array of content blocks.
func (m *ClaudeMessage) ExtractTextContent() string {
	if len(m.Content) == 0 {
		return ""
	}

	// Try parsing as a simple string first
	var simpleString string
	if err := json.Unmarshal(m.Content, &simpleString); err == nil {
		return simpleString
	}

	// Try parsing as an array of content blocks
	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err == nil {
		var textParts []string
		for _, block := range contentBlocks {
			if block.Type == "text" && block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		}
		return strings.Join(textParts, "\n")
	}

	return ""
}

// ExtractToolUses extracts tool use blocks from a ClaudeMessage content array.
func (m *ClaudeMessage) ExtractToolUses() []ClaudeContentItem {
	if len(m.Content) == 0 {
		return nil
	}

	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err != nil {
		return nil
	}

	var toolUses []ClaudeContentItem
	for _, block := range contentBlocks {
		if block.Type == "tool_use" {
			toolUses = append(toolUses, block)
		}
	}
	return toolUses
}

// ExtractToolResults extracts tool result blocks from a ClaudeMessage content array.
// These appear in user messages as responses to tool_use blocks from the assistant.
func (m *ClaudeMessage) ExtractToolResults() []ClaudeContentItem {
	if len(m.Content) == 0 {
		return nil
	}

	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err != nil {
		return nil
	}

	var toolResults []ClaudeContentItem
	for _, block := range contentBlocks {
		if block.Type == "tool_result" {
			toolResults = append(toolResults, block)
		}
	}
	return toolResults
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
// Returns nil, nil for lines that should be silently skipped (non-JSON startup output).
func (r *ClaudeCodeRunner) parseStreamEvent(runID uuid.UUID, line string) (*domain.RunEvent, error) {
	events, err := r.parseStreamEvents(runID, line)
	if err != nil || len(events) == 0 {
		return nil, err
	}
	for _, event := range events {
		if event != nil {
			return event, nil
		}
	}
	return nil, nil
}

func (r *ClaudeCodeRunner) initStreamState(runID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streamState[runID] = &claudeStreamState{}
}

func (r *ClaudeCodeRunner) clearStreamState(runID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streamState, runID)
}

func (r *ClaudeCodeRunner) streamStateFor(runID uuid.UUID) *claudeStreamState {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.streamState == nil {
		r.streamState = make(map[uuid.UUID]*claudeStreamState)
	}
	state, ok := r.streamState[runID]
	if !ok {
		state = &claudeStreamState{}
		r.streamState[runID] = state
	}
	return state
}

func (r *ClaudeCodeRunner) resetToolUseState(state *claudeStreamState) {
	state.toolUseActive = false
	state.toolUseID = ""
	state.toolUseName = ""
	state.toolUsePayload.Reset()
}

func (r *ClaudeCodeRunner) flushStreamMessage(runID uuid.UUID, state *claudeStreamState) []*domain.RunEvent {
	if state == nil {
		return nil
	}
	if state.textBuffer.Len() == 0 {
		return nil
	}
	message := state.textBuffer.String()
	state.textBuffer.Reset()
	state.lastAssistant = message
	return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", message)}
}

func (r *ClaudeCodeRunner) toolCallFromState(runID uuid.UUID, state *claudeStreamState) *domain.RunEvent {
	if state == nil || !state.toolUseActive {
		return nil
	}
	raw := strings.TrimSpace(state.toolUsePayload.String())
	var input map[string]interface{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			input = map[string]interface{}{"raw": raw}
		}
	}
	return domain.NewToolCallEvent(runID, state.toolUseName, input)
}

// parseStreamEvents parses a single line from Claude's stream-json output.
// Returns multiple events to preserve tool calls/results emitted in one message.
func (r *ClaudeCodeRunner) parseStreamEvents(runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	// Skip empty lines
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// Quick check: valid JSON objects start with '{', arrays with '['
	// Skip non-JSON lines like "Initializing...", "[Info] ...", etc.
	if len(line) == 0 {
		return nil, nil
	}
	firstChar := line[0]
	if firstChar != '{' && firstChar != '[' {
		return nil, nil
	}
	// Lines starting with '[' followed by a letter are likely log prefixes like "[Info]"
	// Valid JSON arrays start with '[' followed by whitespace, '{', '[', '"', digit, or ']'
	if firstChar == '[' && len(line) > 1 {
		secondChar := line[1]
		// Check if it looks like a log prefix rather than JSON array
		if (secondChar >= 'A' && secondChar <= 'Z') || (secondChar >= 'a' && secondChar <= 'z') {
			return nil, nil
		}
	}

	var streamEvent ClaudeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		// Silently skip malformed JSON from startup/debug output
		// Real streaming events from Claude Code are always well-formed
		return nil, nil
	}

	state := r.streamStateFor(runID)

	switch streamEvent.Type {
	case "message":
		var events []*domain.RunEvent
		if streamEvent.Message != nil {
			// Extract text content (handles both string and array formats)
			textContent := streamEvent.Message.ExtractTextContent()
			if textContent != "" {
				if streamEvent.Message.Role == "assistant" {
					state.lastAssistant = textContent
				}
				events = append(events, domain.NewMessageEvent(
					runID,
					streamEvent.Message.Role,
					textContent,
				))
				state.textBuffer.Reset()
			}
			toolUses := streamEvent.Message.ExtractToolUses()
			for _, tool := range toolUses {
				var input map[string]interface{}
				if tool.Input != nil {
					_ = json.Unmarshal(tool.Input, &input)
				}
				events = append(events, domain.NewToolCallEvent(runID, tool.Name, input))
			}
		}
		return events, nil

	case "assistant":
		// Assistant turn event - may contain content or just be a turn marker
		if streamEvent.Message != nil {
			var events []*domain.RunEvent
			textContent := streamEvent.Message.ExtractTextContent()
			if textContent != "" {
				state.lastAssistant = textContent
				events = append(events, domain.NewMessageEvent(
					runID,
					"assistant",
					textContent,
				))
				state.textBuffer.Reset()
			}
			// Also check for tool uses in the message content
			toolUses := streamEvent.Message.ExtractToolUses()
			if len(toolUses) > 0 {
				for _, tool := range toolUses {
					var input map[string]interface{}
					if tool.Input != nil {
						_ = json.Unmarshal(tool.Input, &input)
					}
					events = append(events, domain.NewToolCallEvent(runID, tool.Name, input))
				}
			}
			if len(events) > 0 {
				return events, nil
			}
		}
		// Turn marker without content - log for debugging
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Assistant turn started")}, nil

	case "user":
		// User turn event - may contain tool results or the user's prompt
		if streamEvent.Message != nil {
			var events []*domain.RunEvent
			// Check for tool results first (responses to tool_use from assistant)
			toolResults := streamEvent.Message.ExtractToolResults()
			if len(toolResults) > 0 {
				for _, result := range toolResults {
					// Use toolUseID to correlate with the originating tool_call event
					events = append(events, domain.NewToolResultEvent(
						runID,
						"",               // tool name not available from result
						result.ToolUseID, // Tool call ID for correlation
						result.Content,
						nil, // No error for successful tool results
					))
				}
			}

			textContent := streamEvent.Message.ExtractTextContent()
			if textContent != "" {
				events = append(events, domain.NewMessageEvent(runID, "user", textContent))
			}
			if len(events) > 0 {
				return events, nil
			}
		}
		// Turn marker without content
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "User turn marker")}, nil

	case "tool_use":
		if streamEvent.ToolUse != nil {
			var input map[string]interface{}
			if streamEvent.ToolUse.Input != nil {
				_ = json.Unmarshal(streamEvent.ToolUse.Input, &input)
			}
			return []*domain.RunEvent{domain.NewToolCallEvent(
				runID,
				streamEvent.ToolUse.Name,
				input,
			)}, nil
		}

	case "tool_result":
		// Parse tool result
		var resultStr string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &resultStr)
		}
		return []*domain.RunEvent{domain.NewToolResultEvent(
			runID,
			"", // tool name not always available in result
			"", // tool call ID not available in this event type
			resultStr,
			nil,
		)}, nil

	case "error":
		if streamEvent.Error != nil {
			return []*domain.RunEvent{domain.NewErrorEvent(
				runID,
				streamEvent.Error.Code,
				streamEvent.Error.Message,
				false,
			)}, nil
		}

	case "usage":
		if streamEvent.Usage != nil {
			// Emit as metric event
			return []*domain.RunEvent{domain.NewMetricEvent(
				runID,
				"tokens",
				float64(streamEvent.Usage.InputTokens+streamEvent.Usage.OutputTokens),
				"tokens",
			)}, nil
		}

	case "result":
		var events []*domain.RunEvent
		var resultStr string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &resultStr)
		}
		if !streamEvent.IsError && resultStr != "" && state.lastAssistant == "" {
			state.lastAssistant = resultStr
			events = append(events, domain.NewMessageEvent(runID, "assistant", resultStr))
		}
		// Final result event - contains cost, usage, and potential rate limits
		event, err := r.parseResultEvent(runID, &streamEvent)
		if err != nil || event == nil {
			return nil, err
		}
		events = append(events, event)
		return events, nil

	case "system":
		// System context/prompt - log for debugging but don't emit as user-visible event
		return []*domain.RunEvent{domain.NewLogEvent(
			runID,
			"debug",
			"System context received",
		)}, nil

	case "content_block_start":
		// Start of a content block (text or tool use)
		if streamEvent.ContentBlock != nil {
			if streamEvent.ContentBlock.Type == "tool_use" {
				state.toolUseActive = true
				state.toolUseID = streamEvent.ContentBlock.ID
				state.toolUseName = streamEvent.ContentBlock.Name
				state.toolUsePayload.Reset()
				return nil, nil
			}
		}

	case "content_block_delta":
		// Incremental content update (for streaming)
		if streamEvent.Delta != nil {
			switch streamEvent.Delta.Type {
			case "text_delta":
				if streamEvent.Delta.Text != "" {
					state.textBuffer.WriteString(streamEvent.Delta.Text)
				}
				return nil, nil
			case "input_json_delta":
				if state.toolUseActive && streamEvent.Delta.PartialJSON != "" {
					state.toolUsePayload.WriteString(streamEvent.Delta.PartialJSON)
				}
				return nil, nil
			}
		}
		return nil, nil

	case "content_block_stop":
		if state.toolUseActive {
			toolEvent := r.toolCallFromState(runID, state)
			r.resetToolUseState(state)
			if toolEvent != nil {
				return []*domain.RunEvent{toolEvent}, nil
			}
		}
		return nil, nil

	case "message_start":
		// Message lifecycle events - silently skip (content comes via other events)
		return nil, nil
	case "message_delta":
		if streamEvent.Delta != nil && streamEvent.Delta.Text != "" {
			state.textBuffer.WriteString(streamEvent.Delta.Text)
			return nil, nil
		}
		return []*domain.RunEvent{domain.NewLogEvent(
			runID,
			"debug",
			"message_delta received without text payload",
		)}, nil
	case "message_stop":
		events := r.flushStreamMessage(runID, state)
		if state.toolUseActive {
			toolEvent := r.toolCallFromState(runID, state)
			r.resetToolUseState(state)
			if toolEvent != nil {
				events = append(events, toolEvent)
			}
		}
		if len(events) > 0 {
			return events, nil
		}
		return nil, nil

	case "init", "start", "ping", "heartbeat":
		// Initialization and keep-alive events - silently skip
		return nil, nil

	case "":
		// Empty event type - silently skip
		return nil, nil
	}

	// Unknown event type - log it for debugging but don't spam
	if streamEvent.Type != "" {
		return []*domain.RunEvent{domain.NewLogEvent(
			runID,
			"debug",
			fmt.Sprintf("Unhandled event type: %s", streamEvent.Type),
		)}, nil
	}
	return nil, nil
}

// parseResultEvent handles the final "result" event which contains cost and rate limit info.
func (r *ClaudeCodeRunner) parseResultEvent(runID uuid.UUID, event *ClaudeStreamEvent) (*domain.RunEvent, error) {
	// Check for rate limit error in result
	if event.IsError {
		var resultStr string
		if event.Result != nil {
			_ = json.Unmarshal(event.Result, &resultStr)
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
		metrics.CacheReadTokens = data.CacheReadTokens
		metrics.CacheCreationTokens = data.CacheCreationTokens
		metrics.CostEstimateUSD = data.TotalCostUSD
	case *domain.RateLimitEventData:
		// Rate limit detected - this will cause execution to fail
		// The error is handled in the Execute function
	}
}

// Verify interface compliance
var _ Runner = (*ClaudeCodeRunner)(nil)

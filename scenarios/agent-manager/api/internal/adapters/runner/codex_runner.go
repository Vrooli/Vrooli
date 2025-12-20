// Package runner provides runner adapter implementations.
//
// This file implements the Codex runner adapter for executing
// Codex via the resource-codex wrapper within agent-manager.
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

// CodexResourceCommand is the Vrooli resource wrapper command
const CodexResourceCommand = "resource-codex"

// CodexCLICommand is the direct Codex CLI command for JSON streaming
const CodexCLICommand = "codex"

// =============================================================================
// Codex Stream Event Types
// =============================================================================

// CodexStreamEvent represents a single event from Codex's --json output.
type CodexStreamEvent struct {
	Type     string          `json:"type"`
	ThreadID string          `json:"thread_id,omitempty"`
	Item     *CodexItem      `json:"item,omitempty"`
	Usage    *CodexUsage     `json:"usage,omitempty"`
	Error    *CodexError     `json:"error,omitempty"`
	Tool     *CodexToolEvent `json:"tool,omitempty"`
}

// CodexItem represents an item in the Codex stream.
type CodexItem struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"` // agent_message, reasoning, file_change, tool_call, tool_result
	Text     string            `json:"text,omitempty"`
	Name     string            `json:"name,omitempty"`     // tool name
	Input    json.RawMessage   `json:"input,omitempty"`    // tool input
	Output   string            `json:"output,omitempty"`   // tool output
	ExitCode *int              `json:"exit_code,omitempty"`
	Changes  []CodexFileChange `json:"changes,omitempty"` // for file_change items
	Status   string            `json:"status,omitempty"`  // for file_change items (e.g., "completed")
}

// CodexFileChange represents a file modification in Codex's file_change event.
type CodexFileChange struct {
	Path string `json:"path"`           // absolute path to the file
	Kind string `json:"kind,omitempty"` // add, modify, delete
}

// CodexUsage represents token usage in turn.completed events.
type CodexUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens,omitempty"`
	OutputTokens      int `json:"output_tokens"`
}

// CodexError represents an error in the stream.
type CodexError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CodexToolEvent represents a tool-related event.
type CodexToolEvent struct {
	Name   string          `json:"name"`
	Input  json.RawMessage `json:"input,omitempty"`
	Output string          `json:"output,omitempty"`
}

// =============================================================================
// Codex Runner Implementation
// =============================================================================

// CodexRunner implements the Runner interface for OpenAI Codex CLI.
type CodexRunner struct {
	binaryPath    string // resource-codex wrapper path
	codexCLIPath  string // direct codex CLI path (for JSON streaming)
	available     bool
	message       string
	installHint   string
	mu            sync.Mutex
	runs          map[uuid.UUID]*exec.Cmd
	useJSONStream bool // whether to use direct codex CLI with --json
}

// NewCodexRunner creates a new Codex runner.
func NewCodexRunner() (*CodexRunner, error) {
	// Look for resource-codex in PATH (the Vrooli wrapper)
	binaryPath, err := exec.LookPath(CodexResourceCommand)
	if err != nil {
		return &CodexRunner{
			available:   false,
			message:     "resource-codex not found in PATH",
			installHint: "Run: vrooli resource install codex",
			runs:        make(map[uuid.UUID]*exec.Cmd),
		}, nil
	}

	// Also check for direct codex CLI for JSON streaming
	codexCLIPath, _ := exec.LookPath(CodexCLICommand)

	// Verify the resource is healthy by checking status
	runner := &CodexRunner{
		binaryPath:    binaryPath,
		codexCLIPath:  codexCLIPath,
		available:     true,
		message:       "resource-codex available",
		runs:          make(map[uuid.UUID]*exec.Cmd),
		useJSONStream: codexCLIPath != "", // Enable JSON streaming if codex CLI is available
	}

	// Quick health check via status command
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "status", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// Exit code 1 = running but not healthy, 2 = stopped
		// Check if we can still parse the JSON for more details
		var statusData map[string]interface{}
		if jsonErr := json.Unmarshal(output, &statusData); jsonErr == nil {
			// Got JSON, check the healthy field
			if healthyStr, ok := statusData["healthy"].(string); ok && healthyStr == "unknown" {
				// Unknown status - still allow usage but note the uncertainty
				runner.available = true
				if msg, ok := statusData["message"].(string); ok {
					runner.message = msg
				} else {
					runner.message = "Codex CLI installed, login status unknown"
				}
				return runner, nil
			}
		}
		runner.available = false
		runner.message = fmt.Sprintf("resource-codex status check failed: %v", err)
		runner.installHint = "Run: resource-codex manage install"
		return runner, nil
	}

	// Parse JSON status to check health
	var statusData map[string]interface{}
	if err := json.Unmarshal(output, &statusData); err == nil {
		// Handle different healthy values
		switch healthy := statusData["healthy"].(type) {
		case bool:
			if !healthy {
				runner.available = false
				if msg, ok := statusData["message"].(string); ok {
					runner.message = msg
				} else {
					runner.message = "resource-codex is not healthy"
				}
				runner.installHint = "Run: resource-codex manage install"
			}
		case string:
			// "unknown" or other string value - allow usage but note uncertainty
			if healthy == "unknown" {
				runner.available = true
				if msg, ok := statusData["message"].(string); ok {
					runner.message = msg
				} else {
					runner.message = "Codex CLI installed, login status unknown"
				}
			}
		}
	}

	return runner, nil
}

// Type returns the runner type identifier.
func (r *CodexRunner) Type() domain.RunnerType {
	return domain.RunnerTypeCodex
}

// Capabilities returns what this runner supports.
func (r *CodexRunner) Capabilities() Capabilities {
	return Capabilities{
		SupportsMessages:     true,
		SupportsToolEvents:   true,
		SupportsCostTracking: true,
		SupportsStreaming:    r.useJSONStream, // JSON streaming if codex CLI available
		SupportsCancellation: true,
		MaxTurns:             0, // unlimited
		SupportedModels: []string{
			"o4-mini",
			"o3",
			"codex-mini-latest",
			"gpt-5-codex",
		},
	}
}

// Execute runs Codex with the given configuration.
func (r *CodexRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, fmt.Errorf("codex runner is not available: %s", r.message)
	}

	// Use JSON streaming if available, otherwise fall back to wrapper
	if r.useJSONStream {
		return r.executeWithJSONStream(ctx, req)
	}
	return r.executeWithWrapper(ctx, req)
}

// executeWithJSONStream uses the direct codex CLI with --json for structured event streaming.
func (r *CodexRunner) executeWithJSONStream(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	startTime := time.Now()

	// Build command arguments for codex exec --json
	args := r.buildJSONArgs(req)

	// Create command using direct codex CLI
	cmd := exec.CommandContext(ctx, r.codexCLIPath, args...)
	cmd.Dir = req.WorkingDir

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

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

	// Emit starting event
	if req.EventSink != nil {
		req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"Codex execution started",
		))
	}

	// Write prompt and close stdin
	if _, err := stdin.Write([]byte(req.Prompt)); err != nil {
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

	// Process streaming JSON output
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
		event := r.parseCodexStreamEvent(req.RunID, line)
		if event == nil {
			continue
		}

		// Update metrics based on event
		r.updateCodexMetrics(event, &metrics, &lastAssistantMessage)

		// Emit to sink
		if req.EventSink != nil {
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
			"Codex execution completed",
		))
		req.EventSink.Close()
	}

	return result, nil
}

// executeWithWrapper uses the resource-codex wrapper (fallback without JSON streaming).
func (r *CodexRunner) executeWithWrapper(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	startTime := time.Now()

	// Build command arguments - use "run" subcommand with stdin
	args := []string{"run", "-"}

	// Create command using resource-codex
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Dir = req.WorkingDir

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

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start resource-codex: %w", err)
	}

	// Emit starting event
	if req.EventSink != nil {
		req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"Codex execution started",
		))
	}

	// Write prompt and close stdin
	if _, err := stdin.Write([]byte(req.Prompt)); err != nil {
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	stdin.Close()

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
			"Codex execution completed",
		))
		req.EventSink.Close()
	}

	return result, nil
}

// Stop attempts to gracefully stop a running Codex instance.
func (r *CodexRunner) Stop(ctx context.Context, runID uuid.UUID) error {
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

// IsAvailable checks if Codex is currently available.
func (r *CodexRunner) IsAvailable(ctx context.Context) (bool, string) {
	if !r.available {
		msg := r.message
		if r.installHint != "" {
			msg += ". " + r.installHint
		}
		return false, msg
	}

	// Verify the binary still exists
	if _, err := os.Stat(r.binaryPath); os.IsNotExist(err) {
		return false, "resource-codex binary not found. Run: vrooli resource install codex"
	}

	return true, "resource-codex is available"
}

// InstallHint returns instructions for installing this runner.
func (r *CodexRunner) InstallHint() string {
	return r.installHint
}

// buildEnv constructs environment variables for resource-codex run.
func (r *CodexRunner) buildEnv(req ExecuteRequest) []string {
	env := os.Environ()

	// Non-interactive mode
	env = append(env, "CODEX_NON_INTERACTIVE=true")

	// Get the resolved config (handles profile + inline overrides)
	cfg := req.GetConfig()

	// Model selection via environment
	if cfg.Model != "" {
		env = append(env, fmt.Sprintf("CODEX_MODEL=%s", cfg.Model))
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

// buildJSONArgs constructs command-line arguments for codex exec --json.
func (r *CodexRunner) buildJSONArgs(req ExecuteRequest) []string {
	cfg := req.GetConfig()

	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--full-auto",
	}

	// Model selection
	if cfg.Model != "" {
		args = append(args, "-m", cfg.Model)
	}

	// Working directory
	if req.WorkingDir != "" {
		args = append(args, "-C", req.WorkingDir)
	}

	// Read prompt from stdin
	args = append(args, "-")

	return args
}

// parseCodexStreamEvent parses a single line from Codex's --json output.
func (r *CodexRunner) parseCodexStreamEvent(runID uuid.UUID, line string) *domain.RunEvent {
	// Skip empty lines
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '{' {
		return nil
	}

	var streamEvent CodexStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil
	}

	switch streamEvent.Type {
	case "thread.started":
		return domain.NewLogEvent(runID, "debug", "Thread started: "+streamEvent.ThreadID)

	case "turn.started":
		return domain.NewLogEvent(runID, "debug", "Turn started")

	case "item.completed":
		if streamEvent.Item != nil {
			switch streamEvent.Item.Type {
			case "agent_message":
				if streamEvent.Item.Text != "" {
					return domain.NewMessageEvent(runID, "assistant", streamEvent.Item.Text)
				}
			case "reasoning":
				// Codex outputs reasoning/thinking as a separate item type
				if streamEvent.Item.Text != "" {
					return domain.NewLogEvent(runID, "debug", "Reasoning: "+streamEvent.Item.Text)
				}
			case "file_change":
				// Codex uses file_change instead of tool_call for file operations
				// Map this to a tool_call event for consistency
				if len(streamEvent.Item.Changes) > 0 {
					// Build input map with file change details
					input := map[string]interface{}{
						"status": streamEvent.Item.Status,
					}
					// Collect all file paths and their change kinds
					files := make([]map[string]string, 0, len(streamEvent.Item.Changes))
					for _, change := range streamEvent.Item.Changes {
						files = append(files, map[string]string{
							"path": change.Path,
							"kind": change.Kind,
						})
					}
					input["files"] = files
					return domain.NewToolCallEvent(runID, "file_change", input)
				}
			case "tool_call":
				var input map[string]interface{}
				if streamEvent.Item.Input != nil {
					json.Unmarshal(streamEvent.Item.Input, &input)
				}
				return domain.NewToolCallEvent(runID, streamEvent.Item.Name, input)
			case "tool_result":
				return domain.NewToolResultEvent(
					runID,
					streamEvent.Item.Name,
					streamEvent.Item.Output,
					nil,
				)
			}
		}

	case "turn.completed":
		if streamEvent.Usage != nil {
			return domain.NewMetricEvent(
				runID,
				"tokens",
				float64(streamEvent.Usage.InputTokens+streamEvent.Usage.OutputTokens),
				"tokens",
			)
		}

	case "error":
		if streamEvent.Error != nil {
			return domain.NewErrorEvent(
				runID,
				streamEvent.Error.Code,
				streamEvent.Error.Message,
				false,
			)
		}
	}

	return nil
}

// updateCodexMetrics updates execution metrics based on parsed events.
func (r *CodexRunner) updateCodexMetrics(event *domain.RunEvent, metrics *ExecutionMetrics, lastAssistant *string) {
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
			metrics.TokensOutput = totalTokens
		}
	}
}

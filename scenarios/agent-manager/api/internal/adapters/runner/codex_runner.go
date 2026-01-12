// Package runner provides runner adapter implementations.
//
// This file implements the Codex runner adapter for executing
// Codex via the resource-codex wrapper within agent-manager.
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
	ID               string            `json:"id"`
	Type             string            `json:"type"` // agent_message, reasoning, file_change, tool_call, tool_result
	Text             string            `json:"text,omitempty"`
	Name             string            `json:"name,omitempty"`   // tool name
	Input            json.RawMessage   `json:"input,omitempty"`  // tool input
	Output           string            `json:"output,omitempty"` // tool output
	ExitCode         *int              `json:"exit_code,omitempty"`
	Command          string            `json:"command,omitempty"`
	AggregatedOutput string            `json:"aggregated_output,omitempty"` // for command_execution items
	Changes          []CodexFileChange `json:"changes,omitempty"`           // for file_change items
	Status           string            `json:"status,omitempty"`            // for file_change items (e.g., "completed")
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
	binaryPath     string // resource-codex wrapper path
	codexCLIPath   string // direct codex CLI path (for JSON streaming)
	available      bool
	message        string
	installHint    string
	mu             sync.Mutex
	runs           map[uuid.UUID]*exec.Cmd
	useJSONStream  bool // whether to use direct codex CLI with --json
	pricingService PricingService
	runModels      map[uuid.UUID]string
	runThreadIDs   map[uuid.UUID]string // Thread IDs for session continuation
}

// PricingService defines the interface for pricing calculations.
// This allows decoupling from the concrete pricing.Service implementation.
type PricingService interface {
	CalculateCost(ctx context.Context, req PricingCostRequest) (*PricingCostCalculation, error)
}

// PricingCostRequest contains inputs for cost calculation.
type PricingCostRequest struct {
	Model               string
	RunnerType          string
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
}

// PricingCostCalculation contains calculated costs with provenance.
type PricingCostCalculation struct {
	InputCostUSD         float64
	OutputCostUSD        float64
	CacheReadCostUSD     float64
	CacheCreationCostUSD float64
	TotalCostUSD         float64
	CostSource           string
	Provider             string
	CanonicalModel       string
	PricingFetchedAt     time.Time
	PricingVersion       string
}

// CodexRunnerOption configures a CodexRunner.
type CodexRunnerOption func(*CodexRunner)

// WithPricingService sets the pricing service for cost calculations.
func WithPricingService(svc PricingService) CodexRunnerOption {
	return func(r *CodexRunner) {
		r.pricingService = svc
	}
}

// NewCodexRunner creates a new Codex runner.
func NewCodexRunner(opts ...CodexRunnerOption) (*CodexRunner, error) {
	// Look for resource-codex in PATH (the Vrooli wrapper)
	binaryPath, err := exec.LookPath(CodexResourceCommand)
	if err != nil {
		runner := &CodexRunner{
			available:   false,
			message:     "resource-codex not found in PATH",
			installHint: "Run: vrooli resource install codex",
			runs:        make(map[uuid.UUID]*exec.Cmd),
		}
		for _, opt := range opts {
			opt(runner)
		}
		return runner, nil
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
		runModels:     make(map[uuid.UUID]string),
		runThreadIDs:  make(map[uuid.UUID]string),
	}
	for _, opt := range opts {
		opt(runner)
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
		SupportsContinuation: true, // Codex supports "codex resume <thread_id>"
		MaxTurns:             0,    // unlimited
		SupportedModels: []string{
			"gpt-5.2-codex",
			"gpt-5.1-codex-max",
			"gpt-5.1-codex-mini",
			"gpt-5.2",
		},
	}
}

// Execute runs Codex with the given configuration.
func (r *CodexRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, &domain.RunnerError{
			RunnerType:  domain.RunnerTypeCodex,
			Operation:   "availability",
			Cause:       errors.New(r.message),
			IsTransient: false,
		}
	}
	r.trackRunModel(req.RunID, req.GetConfig().Model)
	defer r.clearRunModel(req.RunID)

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

	// Create command using direct codex CLI.
	// Prefix with env to surface the tag in the process command line for reconciler detection.
	tag := req.GetTag()
	envArgs := append([]string{fmt.Sprintf("CODEX_AGENT_TAG=%s", tag), r.codexCLIPath}, args...)
	cmd := exec.CommandContext(ctx, "env", envArgs...)
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
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
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
			"Codex execution started",
		))
	}

	// Write prompt and close stdin
	if _, err := stdin.Write([]byte(req.Prompt)); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
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

	// Parse streaming JSON output (and capture thread_id for session continuation)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse the streaming event(s) and capture thread_id
		events := r.parseCodexStreamEventsWithThreadID(req.RunID, line)
		if len(events) == 0 {
			continue
		}

		for _, event := range events {
			if event == nil {
				continue
			}
			// Update metrics based on event
			r.updateCodexMetrics(event, &metrics, &lastAssistantMessage)

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
			fmt.Sprintf("Codex output scan error: %v", scanErr),
		))
	}

	// Wait for command to complete
	err = cmd.Wait()
	duration := time.Since(startTime)

	// Capture thread ID before clearing
	sessionID := r.threadIDForRun(req.RunID)
	defer r.clearThreadID(req.RunID)

	// Determine result
	result := &ExecuteResult{
		Duration:  duration,
		Metrics:   metrics,
		SessionID: sessionID,
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
			"Codex execution completed",
		))
		_ = req.EventSink.Close()
	}

	return result, nil
}

// executeWithWrapper uses the resource-codex wrapper (fallback without JSON streaming).
func (r *CodexRunner) executeWithWrapper(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	startTime := time.Now()

	// Build command arguments - use "run" subcommand with stdin and tag for process tracking
	args := []string{"run", "--tag", req.GetTag(), "-"}

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
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Provide prompt via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
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
			"Codex execution started",
		))
	}

	// Write prompt and close stdin
	if _, err := stdin.Write([]byte(req.Prompt)); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
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
			_ = req.EventSink.Emit(domain.NewLogEvent(
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
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusRunning),
			finalStatus,
			"Codex execution completed",
		))
		_ = req.EventSink.Close()
	}

	return result, nil
}

// Stop attempts to gracefully stop a running Codex instance.
func (r *CodexRunner) Stop(ctx context.Context, runID uuid.UUID) error {
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
	}

	// Respect the profile's sandbox setting.
	// When RequiresSandbox is false, we use --dangerously-bypass-approvals-and-sandbox
	// which is designed for "environments that are externally sandboxed" (per Codex docs).
	// This enables network access (e.g., SSH) which is blocked by --full-auto.
	if cfg.RequiresSandbox {
		args = append(args, "--full-auto")
	} else {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
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
// It returns the primary event for compatibility with existing tests.
func (r *CodexRunner) parseCodexStreamEvent(runID uuid.UUID, line string) *domain.RunEvent {
	events := r.parseCodexStreamEvents(runID, line)
	if len(events) == 0 {
		return nil
	}
	return events[0]
}

// parseCodexStreamEvents parses a single line from Codex's --json output.
// Some lines can map to multiple events (e.g., tool call + tool result).
func (r *CodexRunner) parseCodexStreamEvents(runID uuid.UUID, line string) []*domain.RunEvent {
	// Skip empty lines
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "" || line[0] != '{' {
		return nil
	}

	var streamEvent CodexStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil
	}

	events := []*domain.RunEvent{}

	// Handle top-level tool payloads (some Codex builds emit tool data outside item.completed).
	if streamEvent.Tool != nil && streamEvent.Item == nil {
		toolName := streamEvent.Tool.Name
		var input map[string]interface{}
		if streamEvent.Tool.Input != nil {
			_ = json.Unmarshal(streamEvent.Tool.Input, &input)
		}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, toolName, input))
		}
		if streamEvent.Tool.Output != "" {
			events = append(events, domain.NewToolResultEvent(runID, toolName, "", streamEvent.Tool.Output, nil))
		}
		if len(events) > 0 {
			return events
		}
	}

	if strings.HasPrefix(streamEvent.Type, "item.") && streamEvent.Item != nil {
		return r.parseCodexItemEvents(runID, streamEvent.Item)
	}

	switch streamEvent.Type {
	case "thread.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Thread started: "+streamEvent.ThreadID)}

	case "turn.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Turn started")}

	case "turn.completed":
		if streamEvent.Usage != nil {
			costEvent := r.buildCodexCostEvent(runID, streamEvent.Usage)
			return []*domain.RunEvent{costEvent}
		}

	case "error":
		if streamEvent.Error != nil {
			return []*domain.RunEvent{domain.NewErrorEvent(
				runID,
				streamEvent.Error.Code,
				streamEvent.Error.Message,
				false,
			)}
		}
	}

	return nil
}

func (r *CodexRunner) parseCodexItemEvents(runID uuid.UUID, item *CodexItem) []*domain.RunEvent {
	if item == nil {
		return nil
	}

	switch item.Type {
	case "agent_message":
		if item.Text != "" {
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", item.Text)}
		}
	case "reasoning":
		// Codex outputs reasoning/thinking as a separate item type
		if item.Text != "" {
			return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Reasoning: "+item.Text)}
		}
	case "file_change":
		// Codex uses file_change instead of tool_call for file operations
		// Map this to a tool_call event for consistency
		if len(item.Changes) > 0 {
			// Build input map with file change details
			input := map[string]interface{}{
				"status": item.Status,
			}
			// Collect all file paths and their change kinds
			files := make([]map[string]string, 0, len(item.Changes))
			for _, change := range item.Changes {
				files = append(files, map[string]string{
					"path": change.Path,
					"kind": change.Kind,
				})
			}
			input["files"] = files
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, "file_change", input)}
		}
	case "tool_call":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, item.Name, input)}
	case "tool_result":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		events := []*domain.RunEvent{}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, item.Name, input))
		}
		events = append(events, domain.NewToolResultEvent(
			runID,
			item.Name,
			"", // Codex doesn't provide tool call IDs
			item.Output,
			nil,
		))
		return events
	case "command_execution":
		// Codex emits shell commands as command_execution items; map to bash tool events.
		toolName := "bash"
		if item.Status == "completed" {
			var errMsg error
			if item.ExitCode != nil && *item.ExitCode != 0 {
				errMsg = fmt.Errorf("command exited with code %d", *item.ExitCode)
			}
			return []*domain.RunEvent{domain.NewToolResultEvent(
				runID,
				toolName,
				"",
				item.AggregatedOutput,
				errMsg,
			)}
		}
		if item.Command != "" {
			input := map[string]interface{}{
				"command":     item.Command,
				"status":      item.Status,
				"runner_tool": "command_execution",
			}
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, input)}
		}
	}

	return nil
}

func (r *CodexRunner) buildCodexCostEvent(runID uuid.UUID, usage *CodexUsage) *domain.RunEvent {
	if usage == nil {
		return nil
	}
	model := r.modelForRun(runID)
	costData := &domain.CostEventData{
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
		CacheReadTokens: usage.CachedInputTokens,
		Model:           model,
		CostSource:      domain.CostSourceUnknown,
	}

	// Use pricing service if available
	if r.pricingService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		calc, err := r.pricingService.CalculateCost(ctx, PricingCostRequest{
			Model:           model,
			RunnerType:      string(domain.RunnerTypeCodex),
			InputTokens:     usage.InputTokens,
			OutputTokens:    usage.OutputTokens,
			CacheReadTokens: usage.CachedInputTokens,
		})
		if err == nil && calc != nil {
			costData.InputCostUSD = calc.InputCostUSD
			costData.OutputCostUSD = calc.OutputCostUSD
			costData.CacheReadCostUSD = calc.CacheReadCostUSD
			costData.TotalCostUSD = calc.TotalCostUSD
			costData.CostSource = calc.CostSource
			costData.PricingProvider = calc.Provider
			costData.PricingModel = calc.CanonicalModel
			if !calc.PricingFetchedAt.IsZero() {
				costData.PricingFetchedAt = &calc.PricingFetchedAt
			}
			costData.PricingVersion = calc.PricingVersion
		}
	}

	return &domain.RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: domain.EventTypeMetric,
		Timestamp: time.Now(),
		Data:      costData,
	}
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
	case *domain.CostEventData:
		// Track token breakdown and cost from CostEventData
		metrics.TokensInput += data.InputTokens
		metrics.TokensOutput += data.OutputTokens
		metrics.CacheReadTokens += data.CacheReadTokens
		metrics.CacheCreationTokens += data.CacheCreationTokens
		metrics.CostEstimateUSD += data.TotalCostUSD
	case *domain.MetricEventData:
		// Legacy fallback for any MetricEventData that might still come through
		if data.Name == "tokens" {
			totalTokens := int(data.Value)
			metrics.TokensOutput = totalTokens
		}
	}
}

func (r *CodexRunner) trackRunModel(runID uuid.UUID, model string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runModels == nil {
		r.runModels = make(map[uuid.UUID]string)
	}
	r.runModels[runID] = strings.TrimSpace(model)
}

func (r *CodexRunner) clearRunModel(runID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runModels, runID)
}

func (r *CodexRunner) modelForRun(runID uuid.UUID) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runModels[runID]
}

func (r *CodexRunner) trackThreadID(runID uuid.UUID, threadID string) {
	if threadID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runThreadIDs == nil {
		r.runThreadIDs = make(map[uuid.UUID]string)
	}
	// Only set once - first thread ID captured is the session ID
	if _, exists := r.runThreadIDs[runID]; !exists {
		r.runThreadIDs[runID] = threadID
	}
}

func (r *CodexRunner) clearThreadID(runID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runThreadIDs, runID)
}

func (r *CodexRunner) threadIDForRun(runID uuid.UUID) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runThreadIDs[runID]
}

// Continue resumes an existing session with a follow-up message.
// Uses Codex's "resume" command to continue the conversation.
func (r *CodexRunner) Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, &domain.RunnerError{
			RunnerType:  domain.RunnerTypeCodex,
			Operation:   "availability",
			Cause:       errors.New(r.message),
			IsTransient: false,
		}
	}

	if req.SessionID == "" {
		return nil, ErrSessionExpired
	}

	if !r.useJSONStream {
		// Resume requires direct codex CLI
		return nil, ErrContinuationNotSupported
	}

	startTime := time.Now()

	// Build command arguments for codex resume.
	// codex resume expects the prompt as an argument (stdin is ignored).
	codexArgs := []string{"resume"}
	if req.WorkingDir != "" {
		codexArgs = append(codexArgs, "-C", req.WorkingDir)
	}
	codexArgs = append(codexArgs, "--full-auto", req.SessionID)
	if strings.TrimSpace(req.Prompt) != "" {
		codexArgs = append(codexArgs, req.Prompt)
	}

	// codex resume requires a TTY; wrap it with `script` to allocate a pseudo-terminal.
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "continue",
			Cause:      fmt.Errorf("script not found for TTY allocation: %w", err),
		}
	}
	cmd := exec.CommandContext(ctx, scriptPath, "-q", "/dev/null", "-c", shellEscapeCommand(r.codexCLIPath, codexArgs...))
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}

	// Set environment
	env := os.Environ()
	env = append(env, "CODEX_NON_INTERACTIVE=true")
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

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
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "continue",
			Cause:      err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "continue",
			Cause:      err,
		}
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "continue",
			Cause:      err,
		}
	}

	// Emit starting event
	if req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusRunning),
			string(domain.RunStatusRunning),
			"Codex continuation started",
		))
	}

	// Process streaming output
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
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

	// Read stdout (codex resume does not emit JSON)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")
		if req.EventSink != nil {
			_ = req.EventSink.Emit(domain.NewLogEvent(
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
		Duration:  duration,
		Metrics:   metrics,
		SessionID: req.SessionID, // Preserve the session ID for further continuations
	}

	if err != nil {
		if ctx.Err() == context.Canceled {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = "continuation cancelled"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.Success = false
			result.ExitCode = exitErr.ExitCode()
			result.ErrorMessage = errorOutput.String()
			// Check if session expired
			if strings.Contains(result.ErrorMessage, "thread") && strings.Contains(result.ErrorMessage, "not found") {
				return nil, ErrSessionExpired
			}
		} else {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = err.Error()
		}
	} else {
		result.Success = true
		result.ExitCode = 0
		output := strings.TrimSpace(outputBuilder.String())
		if output != "" {
			lastAssistantMessage = output
		}
		if lastAssistantMessage != "" {
			result.Summary = &domain.RunSummary{
				Description:  lastAssistantMessage,
				TurnsUsed:    metrics.TurnsUsed,
				TokensUsed:   TotalTokens(metrics),
				CostEstimate: metrics.CostEstimateUSD,
			}
		}
		if req.EventSink != nil && output != "" {
			_ = req.EventSink.Emit(domain.NewMessageEvent(req.RunID, "assistant", output))
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
			"Codex continuation completed",
		))
		_ = req.EventSink.Close()
	}

	return result, nil
}

func shellEscapeCommand(command string, args ...string) string {
	escaped := make([]string, 0, len(args)+1)
	escaped = append(escaped, shellEscapeArg(command))
	for _, arg := range args {
		escaped = append(escaped, shellEscapeArg(arg))
	}
	return strings.Join(escaped, " ")
}

func shellEscapeArg(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.ContainsAny(arg, " \t\n'\"\\$`") {
		return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
	}
	return arg
}

// parseCodexStreamEventsWithThreadID is like parseCodexStreamEvents but also captures thread_id.
func (r *CodexRunner) parseCodexStreamEventsWithThreadID(runID uuid.UUID, line string) []*domain.RunEvent {
	// Skip empty lines
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "" || line[0] != '{' {
		return nil
	}

	var streamEvent CodexStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil
	}

	// Capture thread_id for session continuation
	if streamEvent.ThreadID != "" {
		r.trackThreadID(runID, streamEvent.ThreadID)
	}

	return r.parseCodexStreamEventsInternal(runID, &streamEvent)
}

// parseCodexStreamEventsInternal processes a parsed CodexStreamEvent.
func (r *CodexRunner) parseCodexStreamEventsInternal(runID uuid.UUID, streamEvent *CodexStreamEvent) []*domain.RunEvent {
	events := []*domain.RunEvent{}

	// Handle top-level tool payloads
	if streamEvent.Tool != nil && streamEvent.Item == nil {
		toolName := streamEvent.Tool.Name
		var input map[string]interface{}
		if streamEvent.Tool.Input != nil {
			_ = json.Unmarshal(streamEvent.Tool.Input, &input)
		}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, toolName, input))
		}
		if streamEvent.Tool.Output != "" {
			events = append(events, domain.NewToolResultEvent(runID, toolName, "", streamEvent.Tool.Output, nil))
		}
		if len(events) > 0 {
			return events
		}
	}

	if strings.HasPrefix(streamEvent.Type, "item.") && streamEvent.Item != nil {
		return r.parseCodexItemEvents(runID, streamEvent.Item)
	}

	switch streamEvent.Type {
	case "thread.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Thread started: "+streamEvent.ThreadID)}

	case "turn.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Turn started")}

	case "turn.completed":
		if streamEvent.Usage != nil {
			costEvent := r.buildCodexCostEvent(runID, streamEvent.Usage)
			return []*domain.RunEvent{costEvent}
		}

	case "error":
		if streamEvent.Error != nil {
			return []*domain.RunEvent{domain.NewErrorEvent(
				runID,
				streamEvent.Error.Code,
				streamEvent.Error.Message,
				false,
			)}
		}
	}

	return nil
}

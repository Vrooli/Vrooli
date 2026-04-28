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
	"io"
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
//
// All process launches flow through [launcherSelector] and the resulting
// [LaunchedProcess] is registered in the launched map; there is no direct
// *exec.Cmd path. Tracking-mode requests still resolve to the HostLauncher
// under the hood, so behavior is unchanged for non-protected runs while
// protected mode and any future Launcher implementation get uniform
// tracking, cancellation, and stop semantics.
type CodexRunner struct {
	binaryPath     string // resource-codex wrapper path
	codexCLIPath   string // direct codex CLI path (for JSON streaming)
	available      bool
	message        string
	installHint    string
	mu             sync.Mutex
	launched       map[uuid.UUID]LaunchedProcess
	useJSONStream  bool // whether to use direct codex CLI with --json
	pricingService PricingService
	runModels      map[uuid.UUID]string
	runThreadIDs   map[uuid.UUID]string // Thread IDs for session continuation

	// selector picks host vs sandbox launcher per Execute call. Routing
	// rules and warn-event semantics live in launcherSelector.Pick. Wired
	// at construction time; main.go can swap the sandbox factory later via
	// SetSandboxLauncherFactory once the workspace-sandbox provider is up.
	selector *launcherSelector
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

// NewCodexRunner creates a new Codex runner with a default HostLauncher
// and no sandbox factory; protected-mode requests fall back to host
// execution. Use NewCodexRunnerWithLaunchers for protected mode in
// production (main.go wires the workspace-sandbox provider).
func NewCodexRunner(opts ...CodexRunnerOption) (*CodexRunner, error) {
	return NewCodexRunnerWithLaunchers(NewHostLauncher(), nil, opts...)
}

// NewCodexRunnerWithLaunchers wires the runner with an explicit
// HostLauncher and (optionally) a SandboxLauncherFactory. The factory is
// consulted by launcherSelector.Pick when a streaming-path Execute call
// arrives with SandboxConfig.Mode == Protected and a non-nil SandboxID.
func NewCodexRunnerWithLaunchers(host Launcher, sandboxFactory SandboxLauncherFactory, opts ...CodexRunnerOption) (*CodexRunner, error) {
	selector := newLauncherSelector(host, sandboxFactory)
	// Look for resource-codex in PATH (the Vrooli wrapper)
	binaryPath, err := exec.LookPath(CodexResourceCommand)
	if err != nil {
		runner := &CodexRunner{
			available:   false,
			message:     "resource-codex not found in PATH",
			installHint: "Run: vrooli resource install codex",
			launched:    make(map[uuid.UUID]LaunchedProcess),
			selector:    selector,
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
		launched:      make(map[uuid.UUID]LaunchedProcess),
		useJSONStream: codexCLIPath != "", // Enable JSON streaming if codex CLI is available
		runModels:     make(map[uuid.UUID]string),
		runThreadIDs:  make(map[uuid.UUID]string),
		selector:      selector,
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

// SetSandboxLauncherFactory wires (or replaces) the protected-mode
// factory used by streaming-path Execute calls. main.go invokes this
// after constructing the workspace-sandbox provider; tests can use it to
// inject a mock factory.
func (r *CodexRunner) SetSandboxLauncherFactory(factory SandboxLauncherFactory) {
	r.selector.SetSandboxLauncherFactory(factory)
}

// Type returns the runner type identifier.
func (r *CodexRunner) Type() domain.RunnerType {
	return domain.RunnerTypeCodex
}

// Capabilities returns what this runner supports.
func (r *CodexRunner) Capabilities() Capabilities {
	return Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsStreaming:        r.useJSONStream, // JSON streaming if codex CLI available
		SupportsCancellation:     true,
		SupportsContinuation:     true, // Codex supports "codex resume <thread_id>"
		SupportsImageAttachments: false,
		MaxTurns:                 0, // unlimited
		SupportedModels: []string{
			"gpt-5.2-codex",
			"gpt-5.1-codex-max",
			"gpt-5.1-codex-mini",
			"gpt-5.2",
		},
		SupportedFeatures: []string{},
		AllowedExtraFlags: []string{"--verbose"},
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
	if req.Transcript != nil {
		return r.executeWithJSONTranscript(ctx, req, startTime)
	}

	// Build command arguments for codex exec --json. Codex has no native
	// system prompt mechanism, so EffectivePrompt() prepends SystemPrompt
	// with <system-instructions> tags if present and the result is fed
	// over stdin by the launcher.
	args := r.buildJSONArgs(req)

	// Pick host vs sandbox launcher (tracking → host; protected → sandbox
	// when a factory is wired and a sandbox ID is present, else host with
	// a warn event); see launcherSelector.Pick for the routing rules.
	launcher := r.selector.Pick(ctx, req)
	launchReq := buildEnvWrappedLaunchRequest(
		"CODEX_AGENT_TAG", r.codexCLIPath, args,
		req.GetTag(), req.EffectivePrompt(), r.buildEnv(req), req.WorkingDir,
	)
	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Track the launched process for cancellation and Stop().
	r.mu.Lock()
	r.launched[req.RunID] = proc
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.launched, req.RunID)
		r.mu.Unlock()
		_ = proc.Wait()
	}()

	// Emit starting event
	if req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"Codex execution started",
		))
	}

	// Process streaming JSON output
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Parse streaming JSON output. The launched process owns the
	// idle-timeout watchdog and grandchild cleanup.
	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		proc.ResetIdleTimer()
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

	if proc.TimedOut() && req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewLogEvent(
			req.RunID, "warn",
			fmt.Sprintf("Process idle for %v without output; killed process group", DefaultStreamIdleTimeout),
		))
	}
	if scanErr := scanner.Err(); scanErr != nil && req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewLogEvent(
			req.RunID,
			"warn",
			fmt.Sprintf("Codex output scan error: %v", scanErr),
		))
	}

	// Wait for process cleanup (grandchildren killed, exit status collected).
	err = proc.Wait()
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
		} else if code, ok := extractExitCode(err); ok {
			result.Success = false
			result.ExitCode = code
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
			Description:   lastAssistantMessage,
			TurnsUsed:     metrics.TurnsUsed,
			TokensUsed:    TotalTokens(metrics),
			CostEstimate:  metrics.CostEstimateUSD,
			ContextTokens: metrics.TokensInput,
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
	if req.Transcript != nil {
		return r.executeWithWrapperTranscript(ctx, req, startTime)
	}

	// Build command arguments - use "run" subcommand with stdin and tag
	// for process tracking. The wrapper itself surfaces the tag via its
	// `--tag` flag (visible in /proc/<pid>/cmdline through resource-codex
	// itself), so unlike the JSON-stream path we don't need the env shim.
	args := []string{"run", "--tag", req.GetTag(), "-"}

	// Pick host vs sandbox launcher (tracking → host; protected → sandbox
	// when a factory is wired and a sandbox ID is present, else host with
	// a warn event); see launcherSelector.Pick for the routing rules.
	launcher := r.selector.Pick(ctx, req)
	launchReq := LaunchRequest{
		Command:     r.binaryPath,
		Args:        args,
		Env:         r.buildEnv(req),
		WorkingDir:  req.WorkingDir,
		Stdin:       strings.NewReader(req.EffectivePrompt()),
		IdleTimeout: DefaultStreamIdleTimeout,
	}
	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Track the launched process for cancellation and Stop().
	r.mu.Lock()
	r.launched[req.RunID] = proc
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.launched, req.RunID)
		r.mu.Unlock()
		_ = proc.Wait()
	}()

	// Emit starting event
	if req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusStarting),
			string(domain.RunStatusRunning),
			"Codex execution started",
		))
	}

	// Process output
	metrics := ExecutionMetrics{}
	var outputBuilder strings.Builder
	var errorOutput strings.Builder

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Read stdout — strip ANSI and skip pure-formatting lines
	scanner := bufio.NewScanner(proc.Stdout())
	for scanner.Scan() {
		proc.ResetIdleTimer()
		line := scanner.Text()
		if isOnlyANSI(line) {
			continue
		}
		cleaned := stripANSI(line)
		outputBuilder.WriteString(cleaned)
		outputBuilder.WriteString("\n")

		// Emit log event for each meaningful line
		if req.EventSink != nil && strings.TrimSpace(cleaned) != "" {
			_ = req.EventSink.Emit(domain.NewLogEvent(
				req.RunID,
				"info",
				cleaned,
			))
		}
	}

	// Wait for process cleanup (grandchildren killed, exit status collected).
	err = proc.Wait()
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
		} else if code, ok := extractExitCode(err); ok {
			result.Success = false
			result.ExitCode = code
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
//
// Stop checks both the new launched-process registry (streaming Execute
// paths, post-launcher-refactor) and the legacy *exec.Cmd registry
// (durable-transcript and Continue paths). Either may hold the live run.
func (r *CodexRunner) Stop(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	proc, ok := r.launched[runID]
	r.mu.Unlock()
	if !ok {
		return domain.NewNotFoundErrorWithID("Run", runID.String())
	}

	// Graceful: 5s grace period before SIGKILL escalation.
	proc.Signal(5 * time.Second)
	// Honor parent ctx cancellation as an immediate-kill escalation.
	if ctx != nil {
		go func() {
			<-ctx.Done()
			proc.Kill()
		}()
	}
	return nil
}

func (r *CodexRunner) executeWithJSONTranscript(ctx context.Context, req ExecuteRequest, startTime time.Time) (*ExecuteResult, error) {
	launcher := r.selector.Pick(ctx, req)
	launchReq := buildEnvWrappedLaunchRequest(
		"CODEX_AGENT_TAG", r.codexCLIPath, r.buildJSONArgs(req),
		req.GetTag(), req.EffectivePrompt(), r.buildEnv(req), req.WorkingDir,
	)
	return r.runTranscriptCommand(ctx, req.RunID, req.EventSink, req.Transcript, startTime, codexTranscriptSpec{
		launcher:      launcher,
		request:       launchReq,
		startMessage:  "Codex execution started",
		endMessage:    "Codex execution completed",
		parseFn:       r.ParseTranscriptLine,
		updateMetrics: r.updateCodexMetrics,
	})
}

func (r *CodexRunner) continueWithJSONTranscript(ctx context.Context, req ContinueRequest, startTime time.Time) (*ExecuteResult, error) {
	codexArgs := []string{"exec", "resume", "--json", "--skip-git-repo-check", "--full-auto", req.SessionID}
	if strings.TrimSpace(req.Prompt) != "" {
		codexArgs = append(codexArgs, req.Prompt)
	}
	tag := fmt.Sprintf("codex-continue-%s", req.RunID.String()[:8])
	env := sanitizedBaseEnv()
	env = append(env, "CODEX_NON_INTERACTIVE=true")
	env = appendEnvMap(env, req.Environment)

	launcher := r.selector.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)
	launchReq := buildEnvWrappedLaunchRequest(
		"CODEX_AGENT_TAG", r.codexCLIPath, codexArgs,
		tag, "", env, req.WorkingDir,
	)
	result, err := r.runTranscriptCommand(ctx, req.RunID, req.EventSink, req.Transcript, startTime, codexTranscriptSpec{
		launcher:      launcher,
		request:       launchReq,
		startMessage:  "Codex continuation started",
		endMessage:    "Codex continuation completed",
		parseFn:       r.ParseTranscriptLine,
		updateMetrics: r.updateCodexMetrics,
	})
	if result != nil && result.SessionID == "" {
		result.SessionID = req.SessionID
	}
	return result, err
}

func (r *CodexRunner) executeWithWrapperTranscript(ctx context.Context, req ExecuteRequest, startTime time.Time) (*ExecuteResult, error) {
	// Wrapper path uses resource-codex's own --tag flag (visible in
	// /proc/<pid>/cmdline through the wrapper itself), so unlike the JSON
	// path we don't need the env shim — call the binary directly.
	launcher := r.selector.Pick(ctx, req)
	launchReq := LaunchRequest{
		Command:     r.binaryPath,
		Args:        []string{"run", "--tag", req.GetTag(), "-"},
		Env:         r.buildEnv(req),
		WorkingDir:  req.WorkingDir,
		Stdin:       strings.NewReader(req.EffectivePrompt()),
		IdleTimeout: DefaultStreamIdleTimeout,
	}
	return r.runTranscriptCommand(ctx, req.RunID, req.EventSink, req.Transcript, startTime, codexTranscriptSpec{
		launcher:      launcher,
		request:       launchReq,
		startMessage:  "Codex execution started",
		endMessage:    "Codex execution completed",
		parseFn:       r.parseWrapperTranscriptLine,
		updateMetrics: r.updateCodexMetrics,
	})
}

// codexTranscriptSpec carries the per-call launcher wiring + parse hooks
// into runTranscriptCommand. Mirror of claude_code's durableLaunch.
type codexTranscriptSpec struct {
	launcher      Launcher
	request       LaunchRequest
	startMessage  string
	endMessage    string
	parseFn       func(uuid.UUID, string) TranscriptParseResult
	updateMetrics func(*domain.RunEvent, *ExecutionMetrics, *string)
}

func (r *CodexRunner) runTranscriptCommand(
	ctx context.Context,
	runID uuid.UUID,
	sink EventSink,
	transcript *TranscriptConfig,
	startTime time.Time,
	spec codexTranscriptSpec,
) (*ExecuteResult, error) {
	if transcript == nil || transcript.StdoutFile == nil {
		return nil, &domain.RunnerError{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: errors.New("durable transcript stdout file is required")}
	}

	proc, err := spec.launcher.Launch(ctx, spec.request)
	if err != nil {
		return nil, &domain.RunnerError{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: err}
	}

	r.mu.Lock()
	r.launched[runID] = proc
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.launched, runID)
		r.mu.Unlock()
	}()

	if transcript.OnProcessStart != nil {
		if err := transcript.OnProcessStart(proc.PID(), proc.PID()); err != nil {
			proc.Kill()
			_ = proc.Wait()
			return nil, err
		}
	}
	if sink != nil {
		_ = sink.Emit(domain.NewStatusEvent(runID, string(domain.RunStatusStarting), string(domain.RunStatusRunning), spec.startMessage))
	}

	// Pipe stdout straight into the transcript file. The launcher's stdout
	// reader closes on process exit, so the goroutine drains and returns.
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		_, _ = io.Copy(transcript.StdoutFile, proc.Stdout())
	}()

	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder
	var terminal *TranscriptTerminal

	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			line := scanner.Text()
			errorOutput.WriteString(line)
			errorOutput.WriteString("\n")
			if transcript.StderrFile != nil {
				_, _ = io.WriteString(transcript.StderrFile, line+"\n")
			}
		}
	}()

	consumeCtx, cancelConsume := context.WithCancel(context.Background())
	liveDone := make(chan struct{})
	var liveCursor int64
	go func() {
		defer close(liveDone)
		cursor, liveTerminal, _ := Consume(consumeCtx, ConsumeArgs{
			RunID:       runID,
			Transcript:  transcript.TranscriptPath,
			Live:        true,
			ParseFn:     spec.parseFn,
			EventSink:   sink,
			OnAdvance:   transcript.OnAdvance,
			OnSessionID: transcript.OnSessionID,
			OnEvents: func(events []*domain.RunEvent) {
				for _, evt := range events {
					spec.updateMetrics(evt, &metrics, &lastAssistantMessage)
				}
			},
		})
		liveCursor = cursor
		if liveTerminal != nil {
			terminal = liveTerminal
		}
	}()

	waitErr := proc.Wait()
	cancelConsume()
	<-liveDone
	<-stdoutDone

	_, finalTerminal, drainErr := Consume(context.Background(), ConsumeArgs{
		RunID:       runID,
		Transcript:  transcript.TranscriptPath,
		StartAt:     liveCursor,
		ParseFn:     spec.parseFn,
		EventSink:   sink,
		OnAdvance:   transcript.OnAdvance,
		OnSessionID: transcript.OnSessionID,
		OnEvents: func(events []*domain.RunEvent) {
			for _, evt := range events {
				spec.updateMetrics(evt, &metrics, &lastAssistantMessage)
			}
		},
	})
	if finalTerminal != nil {
		terminal = finalTerminal
	}

	result := &ExecuteResult{
		Duration:  time.Since(startTime),
		Metrics:   metrics,
		SessionID: r.threadIDForRun(runID),
	}
	defer r.clearThreadID(runID)

	if waitErr != nil {
		if ctx.Err() == context.Canceled {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = "execution cancelled"
		} else if code, ok := extractExitCode(waitErr); ok {
			result.Success = false
			result.ExitCode = code
			result.ErrorMessage = errorOutput.String()
		} else {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = waitErr.Error()
		}
	} else if terminal != nil {
		result.Success = terminal.Success
		result.ExitCode = terminal.ExitCode
		result.ErrorMessage = terminal.ErrorMessage
	} else {
		result.Success = true
		result.ExitCode = 0
	}
	if drainErr != nil && result.Success {
		result.Success = false
		result.ExitCode = -1
		result.ErrorMessage = drainErr.Error()
	}
	if result.Success {
		result.Summary = terminalSummaryFromMessage(lastAssistantMessage, metrics)
	}

	if sink != nil {
		finalStatus := string(domain.RunStatusComplete)
		if !result.Success {
			finalStatus = string(domain.RunStatusFailed)
		}
		_ = sink.Emit(domain.NewStatusEvent(runID, string(domain.RunStatusRunning), finalStatus, spec.endMessage))
		_ = sink.Close()
	}
	return result, nil
}

func (r *CodexRunner) parseWrapperTranscriptLine(runID uuid.UUID, line string) TranscriptParseResult {
	text := strings.TrimSpace(line)
	if text == "" {
		return TranscriptParseResult{}
	}
	return TranscriptParseResult{
		Events: []*domain.RunEvent{domain.NewLogEvent(runID, "info", text)},
	}
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

// ProbeModel checks that the Codex binary is available. A deep check (validating
// the model ID via a live request) is intentionally avoided — each probe would
// cost vendor quota. Authoritative "model is gone" signal comes from runtime
// classification. Empty modelID is the runner-default sentinel and always accepted.
func (r *CodexRunner) ProbeModel(ctx context.Context, modelID string) error {
	if available, msg := r.IsAvailable(ctx); !available {
		return fmt.Errorf("codex unavailable: %s", msg)
	}
	return nil
}

// buildEnv constructs environment variables for resource-codex run.
func (r *CodexRunner) buildEnv(req ExecuteRequest) []string {
	env := sanitizedBaseEnv()

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
	return appendEnvMap(env, req.Environment)
}

// buildJSONArgs constructs command-line arguments for codex exec --json.
func (r *CodexRunner) buildJSONArgs(req ExecuteRequest) []string {
	cfg := req.GetConfig()

	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
	}

	// Network access policy determines Codex's sandbox mode.
	// RequiresSandbox only controls overlayfs file isolation (run mode decision).
	switch cfg.NetworkAccess.Effective() {
	case domain.NetworkAccessNone:
		args = append(args, "--full-auto")
	default:
		// localhost or full: bypass Codex's internal sandbox to allow network access.
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

	// Validated extra flags for this runner
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeCodex]; ok {
		args = append(args, extras...)
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
			events = append(events, domain.NewToolCallEvent(runID, toolName, "", input))
		}
		if streamEvent.Tool.Output != "" {
			events = append(events, domain.NewToolResultEvent(runID, toolName, "", stripANSI(streamEvent.Tool.Output), nil))
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
				stripANSI(streamEvent.Error.Code),
				stripANSI(streamEvent.Error.Message),
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
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", stripANSI(item.Text))}
		}
	case "reasoning":
		// Codex outputs reasoning/thinking as a separate item type
		if item.Text != "" {
			return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Reasoning: "+stripANSI(item.Text))}
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
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, "file_change", "", input)}
		}
	case "tool_call":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, item.Name, "", input)}
	case "tool_result":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		events := []*domain.RunEvent{}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, item.Name, "", input))
		}
		events = append(events, domain.NewToolResultEvent(
			runID,
			item.Name,
			"", // Codex doesn't provide tool call IDs
			stripANSI(item.Output),
			nil,
		))
		return events
	case "command_execution":
		// Codex emits shell commands as command_execution items; map to bash tool events.
		toolName := "bash"
		isTerminal := item.Status == "completed" || item.Status == "failed" || item.Status == "error" || item.Status == "cancelled" || item.Status == "timed_out"
		if isTerminal {
			events := make([]*domain.RunEvent, 0, 2)
			// Keep backward-compatible behavior for successful command completion:
			// only emit tool_result. For non-success terminal states, emit tool_call
			// + tool_result so failed commands retain command/status context.
			if item.Command != "" && item.Status != "completed" {
				input := map[string]interface{}{
					"command":     item.Command,
					"status":      item.Status,
					"runner_tool": "command_execution",
				}
				events = append(events, domain.NewToolCallEvent(runID, toolName, "", input))
			}

			var errMsg error
			if item.ExitCode != nil && *item.ExitCode != 0 {
				errMsg = fmt.Errorf("command exited with code %d", *item.ExitCode)
			} else if item.Status != "completed" {
				errMsg = fmt.Errorf("command status: %s", item.Status)
			}

			output := item.AggregatedOutput
			if strings.TrimSpace(output) == "" {
				output = item.Output
			}
			events = append(events, domain.NewToolResultEvent(
				runID,
				toolName,
				"",
				stripANSI(output),
				errMsg,
			))
			return events
		}
		if item.Command != "" {
			input := map[string]interface{}{
				"command":     item.Command,
				"status":      item.Status,
				"runner_tool": "command_execution",
			}
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, "", input)}
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
// Uses "codex exec resume --json" for structured JSONL output, matching the
// Execute path. This avoids the PTY/script wrapper that caused character-by-
// character event spam when using the interactive "codex resume" command.
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
		// exec resume --json requires direct codex CLI
		return nil, ErrContinuationNotSupported
	}

	startTime := time.Now()
	if req.Transcript != nil {
		return r.continueWithJSONTranscript(ctx, req, startTime)
	}

	// Build command arguments for "codex exec resume --json".
	// This is the non-interactive equivalent of "codex resume" and emits
	// structured JSONL events on stdout — no PTY wrapper needed.
	codexArgs := []string{
		"exec", "resume",
		"--json",
		"--skip-git-repo-check",
		"--full-auto",
	}
	// Note: "codex exec resume" does not support -C/--cd; the working
	// directory is set via LaunchRequest.WorkingDir below instead.
	codexArgs = append(codexArgs, req.SessionID)
	if strings.TrimSpace(req.Prompt) != "" {
		codexArgs = append(codexArgs, req.Prompt)
	}

	// Continue uses a synthesized tag (vs req.GetTag() on Execute) so log
	// queries can distinguish continuation runs from initial runs of the
	// same RunID.
	tag := fmt.Sprintf("codex-continue-%s", req.RunID.String()[:8])
	env := sanitizedBaseEnv()
	env = append(env, "CODEX_NON_INTERACTIVE=true")
	env = appendEnvMap(env, req.Environment)

	// Pick host vs sandbox launcher (tracking → host; protected → sandbox
	// when the run was originally sandboxed); see launcherSelector.PickFor
	// for the routing rules.
	launcher := r.selector.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)
	launchReq := buildEnvWrappedLaunchRequest(
		"CODEX_AGENT_TAG", r.codexCLIPath, codexArgs,
		tag, "", env, req.WorkingDir,
	)
	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeCodex,
			Operation:  "continue",
			Cause:      err,
		}
	}

	r.mu.Lock()
	r.launched[req.RunID] = proc
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.launched, req.RunID)
		r.mu.Unlock()
		_ = proc.Wait()
	}()

	// Emit starting event
	if req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewStatusEvent(
			req.RunID,
			string(domain.RunStatusRunning),
			string(domain.RunStatusRunning),
			"Codex continuation started",
		))
	}

	// Process streaming JSON output (same as executeWithJSONStream)
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Parse streaming JSONL output. Launcher handles process lifecycle.
	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		proc.ResetIdleTimer()
		line := scanner.Text()
		if line == "" {
			continue
		}

		events := r.parseCodexStreamEventsWithThreadID(req.RunID, line)
		if len(events) == 0 {
			continue
		}

		for _, event := range events {
			if event == nil {
				continue
			}
			r.updateCodexMetrics(event, &metrics, &lastAssistantMessage)
			if req.EventSink != nil {
				_ = req.EventSink.Emit(event)
			}
		}
	}

	if proc.TimedOut() && req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewLogEvent(
			req.RunID, "warn",
			fmt.Sprintf("Process idle for %v without output; killed process group", DefaultStreamIdleTimeout),
		))
	}
	if scanErr := scanner.Err(); scanErr != nil && req.EventSink != nil {
		_ = req.EventSink.Emit(domain.NewLogEvent(
			req.RunID,
			"warn",
			fmt.Sprintf("Codex continuation scan error: %v", scanErr),
		))
	}

	// Wait for process cleanup (grandchildren killed, exit status collected).
	err = proc.Wait()
	duration := time.Since(startTime)

	// Capture thread ID (may have been updated during continuation)
	sessionID := r.threadIDForRun(req.RunID)
	defer r.clearThreadID(req.RunID)
	if sessionID == "" {
		sessionID = req.SessionID // Preserve the original if no new one was emitted
	}

	// Determine result
	result := &ExecuteResult{
		Duration:  duration,
		Metrics:   metrics,
		SessionID: sessionID,
	}

	if err != nil {
		if ctx.Err() == context.Canceled {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = "continuation cancelled"
		} else if code, ok := extractExitCode(err); ok {
			result.Success = false
			result.ExitCode = code
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
		result.Summary = &domain.RunSummary{
			Description:   lastAssistantMessage,
			TurnsUsed:     metrics.TurnsUsed,
			TokensUsed:    TotalTokens(metrics),
			CostEstimate:  metrics.CostEstimateUSD,
			ContextTokens: metrics.TokensInput,
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

func (r *CodexRunner) ParseTranscriptLine(runID uuid.UUID, line string) TranscriptParseResult {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "" || line[0] != '{' {
		return TranscriptParseResult{}
	}

	var streamEvent CodexStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return TranscriptParseResult{}
	}

	result := TranscriptParseResult{
		Events:    r.parseCodexStreamEventsInternal(runID, &streamEvent),
		SessionID: streamEvent.ThreadID,
	}
	if streamEvent.ThreadID != "" {
		r.trackThreadID(runID, streamEvent.ThreadID)
	}

	switch streamEvent.Type {
	case "turn.completed":
		result.Terminal = &TranscriptTerminal{Success: true, ExitCode: 0}
	case "error":
		if streamEvent.Error != nil {
			result.Terminal = &TranscriptTerminal{
				Success:      false,
				ExitCode:     1,
				ErrorMessage: stripANSI(streamEvent.Error.Message),
			}
		}
	}

	return result
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
			events = append(events, domain.NewToolCallEvent(runID, toolName, "", input))
		}
		if streamEvent.Tool.Output != "" {
			events = append(events, domain.NewToolResultEvent(runID, toolName, "", stripANSI(streamEvent.Tool.Output), nil))
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
				stripANSI(streamEvent.Error.Code),
				stripANSI(streamEvent.Error.Message),
				false,
			)}
		}
	}

	return nil
}

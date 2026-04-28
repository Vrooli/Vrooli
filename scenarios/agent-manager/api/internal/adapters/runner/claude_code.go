// Package runner provides runner adapter implementations.
//
// This file implements the Claude Code runner adapter for executing
// Claude Code via direct CLI invocation within agent-manager.
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
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ClaudeCodeCLICommand is the direct Claude CLI binary name.
const ClaudeCodeCLICommand = "claude"

// ClaudeCodeResourceCommand is the legacy Vrooli resource wrapper (kept for
// transition-period process detection in the reconciler/terminator).
const ClaudeCodeResourceCommand = "resource-claude-code"

// =============================================================================
// Claude Code Runner Implementation
// =============================================================================

// ClaudeCodeRunner implements the Runner interface for Claude Code CLI.
//
// All process launches flow through [launcherSelector.Pick] and the
// resulting [LaunchedProcess] is registered in the launched map; there is
// no direct *exec.Cmd path. Tracking-mode requests still resolve to the
// HostLauncher under the hood, so behavior is unchanged for non-protected
// runs while protected mode and any future Launcher implementation get
// uniform tracking, cancellation, and stop semantics.
type ClaudeCodeRunner struct {
	binaryPath  string
	available   bool
	message     string
	installHint string
	mu          sync.Mutex
	launched    map[uuid.UUID]LaunchedProcess
	streamState map[uuid.UUID]*claudeStreamState

	// selector picks host vs sandbox launcher per Execute call. Routing
	// rules and warn-event semantics live in launcherSelector.Pick.
	selector *launcherSelector
}

// NewClaudeCodeRunner creates a new Claude Code runner with a default
// HostLauncher and no sandbox factory; protected-mode requests fall back
// to host execution. Use NewClaudeCodeRunnerWithLaunchers for protected
// mode in production.
func NewClaudeCodeRunner() (*ClaudeCodeRunner, error) {
	return NewClaudeCodeRunnerWithLaunchers(NewHostLauncher(), nil)
}

// NewClaudeCodeRunnerWithLaunchers wires the runner with an explicit
// HostLauncher and (optionally) a SandboxLauncherFactory. The factory is
// consulted when ExecuteRequest's resolved SandboxConfig.Mode is
// Protected and SandboxID is non-nil.
func NewClaudeCodeRunnerWithLaunchers(host Launcher, sandboxFactory SandboxLauncherFactory) (*ClaudeCodeRunner, error) {
	selector := newLauncherSelector(host, sandboxFactory)
	binaryPath, err := exec.LookPath(ClaudeCodeCLICommand)
	if err != nil {
		return &ClaudeCodeRunner{
			available:   false,
			message:     "claude CLI not found in PATH",
			installHint: "Install: npm install -g @anthropic-ai/claude-code",
			launched:    make(map[uuid.UUID]LaunchedProcess),
			streamState: make(map[uuid.UUID]*claudeStreamState),
			selector:    selector,
		}, nil
	}

	return &ClaudeCodeRunner{
		binaryPath:  binaryPath,
		available:   true,
		message:     "claude CLI available",
		launched:    make(map[uuid.UUID]LaunchedProcess),
		streamState: make(map[uuid.UUID]*claudeStreamState),
		selector:    selector,
	}, nil
}

// SetSandboxLauncherFactory wires (or replaces) the protected-mode factory.
// Used by main.go where the sandbox provider is constructed after the
// runner; tests can also use this to inject a mock factory.
func (r *ClaudeCodeRunner) SetSandboxLauncherFactory(factory SandboxLauncherFactory) {
	r.selector.SetSandboxLauncherFactory(factory)
}

// Type returns the runner type identifier.
func (r *ClaudeCodeRunner) Type() domain.RunnerType {
	return domain.RunnerTypeClaudeCode
}

// Capabilities returns what this runner supports.
func (r *ClaudeCodeRunner) Capabilities() Capabilities {
	return Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsStreaming:        true,
		SupportsCancellation:     true,
		SupportsContinuation:     true, // Claude Code supports --resume for session continuation
		SupportsImageAttachments: true,
		MaxTurns:                 0, // unlimited
		SupportedModels: []string{
			"sonnet",
			"opus",
			"haiku",
			"claude-sonnet-4-5-20250929",
			"claude-opus-4-5-20251101",
			"claude-haiku-4-5-20251001",
		},
		SupportedFeatures: []string{"EnableBrowser"},
		AllowedExtraFlags: []string{"--disallowedTools"},
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

	if req.Transcript != nil {
		return r.executeWithDurableTranscript(ctx, req, startTime)
	}

	// Build command arguments and prompt.
	args := r.buildArgs(req)
	prompt := buildPromptWithAttachments(req.Prompt, req.Attachments)

	// Pick host vs sandbox launcher (tracking → host; protected → sandbox
	// when a factory is wired and a sandbox ID is present, else host with
	// a warn event); see launcherSelector.Pick for the routing rules.
	launcher := r.selector.Pick(ctx, req)

	launchReq := buildEnvWrappedLaunchRequest(
		"CLAUDE_CODE_AGENT_TAG", r.binaryPath, args,
		req.GetTag(), prompt, r.buildEnv(req), req.WorkingDir,
	)
	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeClaudeCode,
			Operation:  "execute",
			Cause:      err,
		}
	}

	// Track the running process for cancellation
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
			"Claude Code execution started",
		))
	}

	// Process streaming output
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder
	var rateLimitEvent *domain.RateLimitEventData

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Parse streaming JSON output. The launched process handles lifecycle:
	// when the runner process exits, grandchildren are killed and the
	// scanner gets EOF automatically.
	const maxScannerBuffer = 10 * 1024 * 1024 // 10MB
	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 64*1024), maxScannerBuffer)

	for scanner.Scan() {
		proc.ResetIdleTimer()
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
			// Invariant: RateLimitEventData is only produced by parseResultEvent
			// when the terminal `result` event has is_error=true. Mid-stream
			// system/api_retry events emit a log, not a RateLimitEvent, so they
			// will not flip the run outcome here.
			if data, ok := event.Data.(*domain.RateLimitEventData); ok {
				rateLimitEvent = data
			}

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

	// Check if scanner exited due to an error (vs clean EOF).
	if scannerErr := scanner.Err(); scannerErr != nil {
		if req.EventSink != nil {
			_ = req.EventSink.Emit(domain.NewLogEvent(
				req.RunID,
				"warn",
				fmt.Sprintf("Scanner error (possible buffer overflow or I/O error): %v", scannerErr),
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
	} else if rateLimitEvent != nil {
		result.Success = false
		result.ExitCode = 429
		result.ErrorMessage = strings.TrimSpace(rateLimitEvent.Message)
		if result.ErrorMessage == "" {
			result.ErrorMessage = "rate limit reached"
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
		emitStderrAsWarnOnSuccess(req.RunID, req.EventSink, errorOutput.String())
	}

	// Capture session ID for conversation continuation (before stream state is cleared)
	if state := r.streamStateFor(req.RunID); state != nil && state.sessionID != "" {
		result.SessionID = state.sessionID
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
// Sends SIGTERM to the process group first, then escalates to SIGKILL
// after a grace period. The actual process reaping is handled by the
// Execute/Continue method's wait goroutine.
func (r *ClaudeCodeRunner) Stop(ctx context.Context, runID uuid.UUID) error {
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

func (r *ClaudeCodeRunner) executeWithDurableTranscript(ctx context.Context, req ExecuteRequest, startTime time.Time) (*ExecuteResult, error) {
	launcher := r.selector.Pick(ctx, req)
	launchReq := buildEnvWrappedLaunchRequest(
		"CLAUDE_CODE_AGENT_TAG", r.binaryPath, r.buildArgs(req),
		req.GetTag(), buildPromptWithAttachments(req.Prompt, req.Attachments),
		r.buildEnv(req), req.WorkingDir,
	)
	return r.runDurableCommand(ctx, req.RunID, req.EventSink, req.Transcript, startTime, durableLaunch{
		launcher:     launcher,
		request:      launchReq,
		startFrom:    string(domain.RunStatusStarting),
		startTo:      string(domain.RunStatusRunning),
		startMessage: "Claude Code execution started",
		endMessage:   "Claude Code execution completed",
	})
}

func (r *ClaudeCodeRunner) continueWithDurableTranscript(ctx context.Context, req ContinueRequest, startTime time.Time) (*ExecuteResult, error) {
	tag := fmt.Sprintf("claude-continue-%s", req.RunID.String()[:8])
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--resume", req.SessionID,
		"--dangerously-skip-permissions",
		"-",
	}
	env := sanitizedBaseEnv()
	env = append(env, fmt.Sprintf("CLAUDE_CODE_AGENT_TAG=%s", tag))
	env = appendEnvMap(env, req.Environment)

	launcher := r.selector.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)
	launchReq := buildEnvWrappedLaunchRequest(
		"CLAUDE_CODE_AGENT_TAG", r.binaryPath, args,
		tag, buildPromptWithAttachments(req.Prompt, req.Attachments),
		env, req.WorkingDir,
	)
	result, err := r.runDurableCommand(ctx, req.RunID, req.EventSink, req.Transcript, startTime, durableLaunch{
		launcher:     launcher,
		request:      launchReq,
		startFrom:    string(domain.RunStatusRunning),
		startTo:      string(domain.RunStatusRunning),
		startMessage: "Claude Code continuation started",
		endMessage:   "Claude Code continuation completed",
	})
	if result != nil && result.SessionID == "" {
		result.SessionID = req.SessionID
	}
	return result, err
}

// durableLaunch carries the per-call launcher wiring + status messaging
// into runDurableCommand. The runner builds it from selector.Pick (or
// PickContinue) plus a LaunchRequest from the launch-request builder.
type durableLaunch struct {
	launcher     Launcher
	request      LaunchRequest
	startFrom    string
	startTo      string
	startMessage string
	endMessage   string
}

// runDurableCommand routes a coding-agent run through the [Launcher] seam
// and writes its stdout to the durable transcript file. Stderr is mirrored
// into transcript.StderrFile when present, scanned for the error-message
// summary, and a live [Consume] goroutine streams parsed events to the
// sink while the process runs. Stdin is delivered via LaunchRequest.Stdin
// (the launcher copies it once and closes the pipe).
func (r *ClaudeCodeRunner) runDurableCommand(ctx context.Context, runID uuid.UUID, sink EventSink, transcript *TranscriptConfig, startTime time.Time, spec durableLaunch) (*ExecuteResult, error) {
	if transcript == nil || transcript.StdoutFile == nil {
		return nil, &domain.RunnerError{RunnerType: domain.RunnerTypeClaudeCode, Operation: "execute", Cause: errors.New("durable transcript stdout file is required")}
	}

	proc, err := spec.launcher.Launch(ctx, spec.request)
	if err != nil {
		return nil, &domain.RunnerError{RunnerType: domain.RunnerTypeClaudeCode, Operation: "execute", Cause: err}
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
		_ = sink.Emit(domain.NewStatusEvent(runID, spec.startFrom, spec.startTo, spec.startMessage))
	}

	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder
	var terminal *TranscriptTerminal

	// Pipe stdout straight into the transcript file. The launcher's
	// Stdout reader closes on process exit, so the goroutine drains and
	// returns naturally.
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		_, _ = io.Copy(transcript.StdoutFile, proc.Stdout())
	}()

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
			ParseFn:     r.ParseTranscriptLine,
			EventSink:   sink,
			OnAdvance:   transcript.OnAdvance,
			OnSessionID: transcript.OnSessionID,
			OnEvents: func(events []*domain.RunEvent) {
				for _, evt := range events {
					r.updateMetrics(evt, &metrics, &lastAssistantMessage)
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

	finalCursor, finalTerminal, drainErr := Consume(context.Background(), ConsumeArgs{
		RunID:       runID,
		Transcript:  transcript.TranscriptPath,
		StartAt:     liveCursor,
		ParseFn:     r.ParseTranscriptLine,
		EventSink:   sink,
		OnAdvance:   transcript.OnAdvance,
		OnSessionID: transcript.OnSessionID,
		OnEvents: func(events []*domain.RunEvent) {
			for _, evt := range events {
				r.updateMetrics(evt, &metrics, &lastAssistantMessage)
			}
		},
	})
	if finalTerminal != nil {
		terminal = finalTerminal
	}
	if transcript.OnAdvance != nil {
		_ = transcript.OnAdvance(finalCursor, 0)
	}

	duration := time.Since(startTime)
	result := &ExecuteResult{Duration: duration, Metrics: metrics}
	if state := r.streamStateFor(runID); state != nil && state.sessionID != "" {
		result.SessionID = state.sessionID
	}

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
	} else if strings.TrimSpace(result.ErrorMessage) == "" {
		result.ErrorMessage = strings.TrimSpace(errorOutput.String())
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

// Continue resumes an existing session with a follow-up message.
// Uses Claude Code's --resume flag to continue the conversation.
func (r *ClaudeCodeRunner) Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error) {
	if !r.available {
		return nil, &domain.RunnerError{
			RunnerType:  domain.RunnerTypeClaudeCode,
			Operation:   "availability",
			Cause:       errors.New(r.message),
			IsTransient: false,
		}
	}

	if req.SessionID == "" {
		return nil, ErrSessionExpired
	}

	startTime := time.Now()
	r.initStreamState(req.RunID)
	defer r.clearStreamState(req.RunID)

	if req.Transcript != nil {
		return r.continueWithDurableTranscript(ctx, req, startTime)
	}

	// Build command arguments for continuation
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--resume", req.SessionID,
		"--dangerously-skip-permissions",
		"-", // Read prompt from stdin
	}

	// Continue uses a synthesized tag (vs req.GetTag() on Execute) so log
	// queries can distinguish continuation runs from initial runs of the
	// same RunID.
	tag := fmt.Sprintf("claude-continue-%s", req.RunID.String()[:8])
	env := sanitizedBaseEnv()
	env = append(env, fmt.Sprintf("CLAUDE_CODE_AGENT_TAG=%s", tag))
	env = appendEnvMap(env, req.Environment)

	// Pick host vs sandbox launcher (tracking → host; protected → sandbox
	// when the run was originally sandboxed); see launcherSelector.PickFor
	// for the routing rules.
	launcher := r.selector.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)
	prompt := buildPromptWithAttachments(req.Prompt, req.Attachments)
	launchReq := buildEnvWrappedLaunchRequest(
		"CLAUDE_CODE_AGENT_TAG", r.binaryPath, args,
		tag, prompt, env, req.WorkingDir,
	)
	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: domain.RunnerTypeClaudeCode,
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
			"Claude Code continuation started",
		))
	}

	// Process streaming output
	metrics := ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder
	var rateLimitEvent *domain.RateLimitEventData

	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			errorOutput.WriteString(scanner.Text())
			errorOutput.WriteString("\n")
		}
	}()

	// Parse streaming JSON output. Launcher handles process lifecycle:
	// when the launched process exits, the stdout reader returns EOF.
	const maxScannerBuffer = 10 * 1024 * 1024 // 10MB
	scanner := bufio.NewScanner(proc.Stdout())
	scanner.Buffer(make([]byte, 64*1024), maxScannerBuffer)

	for scanner.Scan() {
		proc.ResetIdleTimer()
		line := scanner.Text()
		if line == "" {
			continue
		}

		events, err := r.parseStreamEvents(req.RunID, line)
		if err != nil {
			if req.EventSink != nil {
				_ = req.EventSink.Emit(domain.NewLogEvent(
					req.RunID,
					"warn",
					fmt.Sprintf("Failed to parse event: %v", err),
				))
			}
			continue
		}

		if len(events) == 0 {
			continue
		}

		for _, event := range events {
			if event == nil {
				continue
			}
			r.updateMetrics(event, &metrics, &lastAssistantMessage)
			if data, ok := event.Data.(*domain.RateLimitEventData); ok {
				rateLimitEvent = data
			}
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

	// Wait for process cleanup.
	err = proc.Wait()

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
		} else if code, ok := extractExitCode(err); ok {
			result.Success = false
			result.ExitCode = code
			result.ErrorMessage = errorOutput.String()
			// Detect session-expiry signal in stderr (claude exits with a
			// generic non-zero code; the message is what tells us why).
			if strings.Contains(result.ErrorMessage, "session") && strings.Contains(result.ErrorMessage, "not found") {
				return nil, ErrSessionExpired
			}
		} else {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = err.Error()
		}
	} else if rateLimitEvent != nil {
		result.Success = false
		result.ExitCode = 429
		result.ErrorMessage = strings.TrimSpace(rateLimitEvent.Message)
		if result.ErrorMessage == "" {
			result.ErrorMessage = "rate limit reached"
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
		emitStderrAsWarnOnSuccess(req.RunID, req.EventSink, errorOutput.String())
	}

	// Update session ID from stream if a new one was provided
	if state := r.streamStateFor(req.RunID); state != nil && state.sessionID != "" {
		result.SessionID = state.sessionID
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
			"Claude Code continuation completed",
		))
		_ = req.EventSink.Close()
	}

	return result, nil
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
		return false, "claude CLI not found. Install: npm install -g @anthropic-ai/claude-code"
	}

	return true, "claude CLI is available"
}

// InstallHint returns instructions for installing this runner.
func (r *ClaudeCodeRunner) InstallHint() string {
	return r.installHint
}

// ProbeModel is intentionally lenient for Claude Code: the canonical presets are
// vendor aliases (opus/sonnet/haiku) that resolve server-side to whatever build is
// current, so there is no meaningful local check to run. Pinned version IDs are
// also accepted; if they have been retired, the runtime classifier surfaces the
// failure on the first real invocation.
func (r *ClaudeCodeRunner) ProbeModel(ctx context.Context, modelID string) error {
	if available, msg := r.IsAvailable(ctx); !available {
		return fmt.Errorf("claude-code unavailable: %s", msg)
	}
	return nil
}

// buildArgs constructs command-line arguments for direct claude CLI invocation.
func (r *ClaudeCodeRunner) buildArgs(req ExecuteRequest) []string {
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose", // Required with --print --output-format stream-json for full event stream
	}

	cfg := req.GetConfig()

	// Max turns
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(cfg.MaxTurns))
	} else {
		args = append(args, "--max-turns", "30")
	}

	// Model selection
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	// Skip permission prompts for autonomous execution
	if cfg.SkipPermissionPrompt {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Allowed tools
	if len(cfg.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(cfg.AllowedTools, ","))
	}

	// System prompt
	if req.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", req.SystemPrompt)
	}

	// Feature flags
	if cfg.Features.EnableBrowser {
		args = append(args, "--chrome")
	}

	// Validated extra flags for this runner
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeClaudeCode]; ok {
		args = append(args, extras...)
	}

	args = append(args, "-") // Read prompt from stdin
	return args
}

// buildEnv constructs environment variables for direct claude CLI invocation.
// All configuration is passed via CLI args; only the agent tag and custom env remain.
func (r *ClaudeCodeRunner) buildEnv(req ExecuteRequest) []string {
	env := sanitizedBaseEnv()

	// Tag for reconciler process detection via /proc/<pid>/environ.
	env = append(env, fmt.Sprintf("CLAUDE_CODE_AGENT_TAG=%s", req.GetTag()))

	// Add any custom environment from the request
	return appendEnvMap(env, req.Environment)
}

// ClaudeStreamEvent represents a single event from Claude Code's stream-json output.
type ClaudeStreamEvent struct {
	Type         string              `json:"type"`
	Subtype      string              `json:"subtype,omitempty"` // e.g., "success", "error", "api_retry"
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

	// Fields emitted by system/api_retry events. See:
	// https://code.claude.com/docs/en/errors and
	// https://backgroundclaude.com/blog/stream-json
	//
	// Note: the api_retry event also carries an "error" field holding a short
	// string like "rate_limit", but we cannot add a second Go field with the
	// "error" JSON tag without colliding with the existing *ClaudeError object
	// decoded from type:"error" events. ErrorStatus (the HTTP status) is
	// sufficient to describe the retry for logging purposes.
	ErrorStatus  int `json:"error_status,omitempty"` // e.g., 429
	Attempt      int `json:"attempt,omitempty"`      // retry attempt number
	MaxRetries   int `json:"max_retries,omitempty"`  // configured retry cap
	RetryDelayMs int `json:"retry_delay_ms,omitempty"`
}

type claudeStreamState struct {
	textBuffer     strings.Builder
	toolUseActive  bool
	toolUseID      string
	toolUseName    string
	toolUsePayload strings.Builder
	lastAssistant  string
	sessionID      string // Captured from stream for conversation continuation
	gotResult      bool   // True when the final "result" event has been received
	resultIsError  bool   // True if the result event indicated an error

	// Compaction tracking
	pendingCompact bool   // True if we just saw a /compact command
	compactCommand string // The full /compact command text
	compactFocus   string // Extracted focus instruction
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
// ANSI escape sequences are stripped as defense-in-depth: even if the
// resource-claude-code wrapper correctly separates stderr from stdout,
// tool results (e.g., Bash output) may still embed terminal formatting.
func (m *ClaudeMessage) ExtractTextContent() string {
	if len(m.Content) == 0 {
		return ""
	}

	// Try parsing as a simple string first
	var simpleString string
	if err := json.Unmarshal(m.Content, &simpleString); err == nil {
		return stripANSI(simpleString)
	}

	// Try parsing as an array of content blocks
	var contentBlocks []ClaudeContentItem
	if err := json.Unmarshal(m.Content, &contentBlocks); err == nil {
		var textParts []string
		for _, block := range contentBlocks {
			if block.Type == "text" && block.Text != "" {
				textParts = append(textParts, stripANSI(block.Text))
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
			// Strip ANSI from tool result content (Bash output often has terminal formatting)
			block.Content = stripANSI(block.Content)
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

// ClaudeError represents an error in the stream. The Claude Code CLI uses
// the "error" JSON field with two different shapes: an object
// {"code": ..., "message": ...} for type:"error" events, and a bare string
// like "rate_limit" for system/api_retry events. UnmarshalJSON accepts both
// so a single struct field can decode either form without collision.
type ClaudeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UnmarshalJSON accepts either an object or a bare string. A bare string is
// stored in Code and Message is left empty.
func (e *ClaudeError) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		e.Code = s
		return nil
	}
	type alias ClaudeError
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = ClaudeError(a)
	return nil
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
	message := stripANSI(state.textBuffer.String())
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
	return domain.NewToolCallEvent(runID, state.toolUseName, state.toolUseID, input)
}

// =============================================================================
// Prompt Helpers
// =============================================================================

// buildPromptWithAttachments prepends attachment file paths to a prompt.
// Claude Code reads image files when paths are provided in the input.
func buildPromptWithAttachments(prompt string, attachments []Attachment) string {
	if len(attachments) == 0 {
		return prompt
	}
	var sb strings.Builder
	for _, att := range attachments {
		sb.WriteString(att.FilePath)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(prompt)
	return sb.String()
}

// =============================================================================
// Compaction Detection Helpers
// =============================================================================

// parseCompactCommand extracts focus from "/compact focus on auth" -> "auth".
func parseCompactCommand(content string) (isCompact bool, focus string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/compact") {
		return false, ""
	}
	// Ensure it's actually "/compact" and not "/compacting" etc.
	rest := strings.TrimPrefix(content, "/compact")
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' && rest[0] != '\n' {
		return false, ""
	}

	remainder := strings.TrimSpace(rest)
	if strings.HasPrefix(remainder, "focus on ") {
		focus = strings.TrimPrefix(remainder, "focus on ")
	} else if remainder != "" {
		focus = remainder
	}

	return true, strings.TrimSpace(focus)
}

// isCompactionSummary checks if content looks like a compaction summary.
func isCompactionSummary(content string) bool {
	return strings.Contains(content, "<summary>") ||
		strings.HasPrefix(strings.TrimSpace(content), "Summary of")
}

// extractSummaryContent extracts content from <summary>...</summary> tags.
func extractSummaryContent(content string) string {
	start := strings.Index(content, "<summary>")
	end := strings.Index(content, "</summary>")

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(content[start+len("<summary>") : end])
	}

	// No tags, return as-is (some runners don't use tags)
	return content
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

	// Capture session_id for conversation continuation if present
	if streamEvent.SessionID != "" && state.sessionID == "" {
		state.sessionID = streamEvent.SessionID
	}

	switch streamEvent.Type {
	case "message":
		var events []*domain.RunEvent
		if streamEvent.Message != nil {
			// Extract text content (handles both string and array formats)
			textContent := streamEvent.Message.ExtractTextContent()
			if textContent != "" {
				if streamEvent.Message.Role == "user" {
					// Check for /compact command in user messages
					if isCompact, focus := parseCompactCommand(textContent); isCompact {
						state.pendingCompact = true
						state.compactCommand = textContent
						state.compactFocus = focus
						// Don't emit the /compact as a regular message
						return nil, nil
					}
					// Don't emit user messages from the stream — the orchestrator
					// already creates message events for both the initial prompt
					// and follow-up messages. Emitting them here produces duplicates
					// that show as spurious "You" entries in the timeline. This also
					// suppresses subagent prompts (Agent tool internal messages) that
					// Claude Code echoes through the parent stream.
				} else {
					// Check for compaction summary in assistant response
					if state.pendingCompact && streamEvent.Message.Role == "assistant" {
						if isCompactionSummary(textContent) {
							state.pendingCompact = false
							summary := extractSummaryContent(textContent)
							return []*domain.RunEvent{
								domain.NewCompactionEvent(
									runID,
									summary,
									"manual",
									state.compactFocus,
									0, // messagesCompacted (not available from stream)
									0, // tokensBefore
									0, // tokensAfter
									state.compactCommand,
								),
							}, nil
						}
						// Not a summary, reset state and fall through to normal handling
						state.pendingCompact = false
					}

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
			}
			// Extract tool results from user messages (tool_result blocks)
			if streamEvent.Message.Role == "user" {
				toolResults := streamEvent.Message.ExtractToolResults()
				for _, result := range toolResults {
					events = append(events, domain.NewToolResultEvent(
						runID,
						"",               // tool name not available from result
						result.ToolUseID, // Tool call ID for correlation
						result.Content,
						nil, // No error for successful tool results
					))
				}
			}
			toolUses := streamEvent.Message.ExtractToolUses()
			for _, tool := range toolUses {
				var input map[string]interface{}
				if tool.Input != nil {
					_ = json.Unmarshal(tool.Input, &input)
				}
				events = append(events, domain.NewToolCallEvent(runID, tool.Name, tool.ID, input))
			}
		}
		return events, nil

	case "assistant":
		// Assistant turn event - may contain content or just be a turn marker
		if streamEvent.Message != nil {
			var events []*domain.RunEvent
			textContent := streamEvent.Message.ExtractTextContent()
			if textContent != "" {
				// Check for compaction summary if we're expecting one
				if state.pendingCompact && isCompactionSummary(textContent) {
					state.pendingCompact = false
					summary := extractSummaryContent(textContent)
					return []*domain.RunEvent{
						domain.NewCompactionEvent(
							runID,
							summary,
							"manual",
							state.compactFocus,
							0, 0, 0,
							state.compactCommand,
						),
					}, nil
				}
				if state.pendingCompact {
					state.pendingCompact = false
				}

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
					events = append(events, domain.NewToolCallEvent(runID, tool.Name, tool.ID, input))
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
				// Check for /compact command
				if isCompact, focus := parseCompactCommand(textContent); isCompact {
					state.pendingCompact = true
					state.compactCommand = textContent
					state.compactFocus = focus
					// Don't emit the /compact as a regular message
				}
				// Don't emit user text as a message — the orchestrator already
				// creates message events for both the initial prompt and follow-ups.
				// Emitting here produces duplicate "You" entries in the timeline.
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
				streamEvent.ToolUse.ID,
				input,
			)}, nil
		}

	case "tool_result":
		// Parse tool result
		var resultStr string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &resultStr)
		}
		resultStr = stripANSI(resultStr)
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
		// Mark that we received the final result event — this means Claude Code
		// has finished and we have all the data we need from the stream.
		state.gotResult = true
		state.resultIsError = streamEvent.IsError

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
		// Detect automatic compaction signals; Claude Code surfaces these
		// via `subtype: "auto-compacting"` or via text in `result`.
		var sysResult string
		if streamEvent.Result != nil {
			_ = json.Unmarshal(streamEvent.Result, &sysResult)
		}
		if strings.Contains(strings.ToLower(streamEvent.Subtype), "auto-compact") || isAutoCompactMarker(sysResult) {
			return []*domain.RunEvent{domain.NewCompactionEvent(
				runID,
				strings.TrimSpace(sysResult),
				"auto",
				"",
				0, 0, 0,
				"",
			)}, nil
		}
		// api_retry means the CLI hit a transient failure (often 429) and is
		// retrying automatically. This is informational — it is NOT a terminal
		// failure and must not produce a RateLimitEvent. Only the final `result`
		// event with `is_error: true` determines run outcome.
		if streamEvent.Subtype == "api_retry" {
			return []*domain.RunEvent{domain.NewLogEvent(
				runID,
				"warn",
				fmt.Sprintf("Claude CLI auto-retry: HTTP %d, attempt %d/%d, next in %dms",
					streamEvent.ErrorStatus,
					streamEvent.Attempt, streamEvent.MaxRetries,
					streamEvent.RetryDelayMs),
			)}, nil
		}
		// Otherwise, log for debugging but don't emit as user-visible event.
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

func (r *ClaudeCodeRunner) ParseTranscriptLine(runID uuid.UUID, line string) TranscriptParseResult {
	events, err := r.parseStreamEvents(runID, line)
	result := TranscriptParseResult{
		Events: events,
		Err:    err,
	}

	var streamEvent ClaudeStreamEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &streamEvent); err == nil {
		result.SessionID = streamEvent.SessionID
		if strings.EqualFold(streamEvent.Type, "result") {
			terminal := &TranscriptTerminal{
				Success:  !streamEvent.IsError,
				ExitCode: 0,
			}
			if streamEvent.IsError {
				terminal.Success = false
				terminal.ExitCode = 1
				terminal.ErrorMessage = strings.TrimSpace(decodeClaudeResultString(streamEvent.Result))
				if streamEvent.Subtype == "error" && strings.Contains(strings.ToLower(terminal.ErrorMessage), "rate limit") {
					terminal.ExitCode = 429
				}
				if terminal.ErrorMessage == "" {
					terminal.ErrorMessage = "runner reported terminal error"
				}
			}
			result.Terminal = terminal
		}
	}

	return result
}

// parseResultEvent handles the final "result" event which contains cost and rate limit info.
func (r *ClaudeCodeRunner) parseResultEvent(runID uuid.UUID, event *ClaudeStreamEvent) (*domain.RunEvent, error) {
	resultStr := decodeClaudeResultString(event.Result)

	// Rate-limit classification is gated on the CLI's own `is_error` flag. The
	// `result` field of a successful run contains the agent's final assistant
	// message, which can legitimately mention rate limits (e.g., an agent
	// writing about rate-limit detection). Scanning it would produce false
	// positives that misclassify successful runs as 429 failures. Only treat
	// the result as a rate limit when the CLI itself flagged the run as
	// errored — this is the only authoritative signal.
	if event.IsError {
		if rl := r.detectRateLimit(resultStr); rl.Detected {
			return domain.NewRateLimitEvent(
				runID,
				rl.LimitType,
				rl.Message,
				rl.ResetTime,
				rl.RetryAfter,
			), nil
		}
		msg := formatErrorMessage(event.Subtype, event.NumTurns, event.DurationMs, resultStr)
		errEvent := domain.NewErrorEvent(runID, "execution_error", msg, false)
		if data, ok := errEvent.Data.(*domain.ErrorEventData); ok {
			data.Details = buildErrorDetails(event.Subtype, event.NumTurns, event.DurationMs, event.SessionID, resultStr, "")
		}
		return errEvent, nil
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
				CostSource:          domain.CostSourceRunnerReported,
				PricingProvider:     "claude-code",
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

func decodeClaudeResultString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var result string
	if err := json.Unmarshal(raw, &result); err == nil {
		return result
	}
	return strings.TrimSpace(string(raw))
}

// maxRateLimitMessageLen caps the size of messages eligible for rate-limit
// classification. All documented Claude Code rate-limit banners fit in under
// ~100 chars (see https://code.claude.com/docs/en/errors). A much longer
// payload is almost certainly something else (e.g., an error dump or tool
// output that happens to include a trigger word), so we refuse to classify
// it as a rate limit even if it contains a matching phrase.
const maxRateLimitMessageLen = 512

// detectRateLimit parses rate limit information from error messages. Callers
// MUST gate this on the CLI's `is_error` flag — feeding successful-run output
// into this function will produce false positives when the agent's own text
// mentions rate limits.
func (r *ClaudeCodeRunner) detectRateLimit(resultStr string) RateLimitInfo {
	info := RateLimitInfo{
		Detected: false,
		Message:  resultStr,
	}

	if len(resultStr) > maxRateLimitMessageLen {
		return info
	}

	lowerMsg := strings.ToLower(resultStr)

	// Anchored phrases documented by Anthropic in
	// https://code.claude.com/docs/en/errors. Bare "rate limit" is NOT matched
	// because it appears in ordinary prose that discusses rate limiting.
	matched := strings.Contains(lowerMsg, "usage limit reached") ||
		strings.Contains(lowerMsg, "rate limit reached") ||
		strings.Contains(lowerMsg, "rate limit exceeded") ||
		strings.Contains(lowerMsg, "request rejected (429)") ||
		strings.Contains(lowerMsg, "server is temporarily limiting requests") ||
		(strings.Contains(lowerMsg, "hit your") && strings.Contains(lowerMsg, "limit")) ||
		(strings.Contains(lowerMsg, "reached your") && strings.Contains(lowerMsg, "limit"))

	if !matched {
		return info
	}

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
	if strings.Contains(lowerMsg, "daily") {
		info.LimitType = "daily"
	} else if strings.Contains(lowerMsg, "weekly") {
		info.LimitType = "weekly"
	} else if strings.Contains(lowerMsg, "token") {
		info.LimitType = "token"
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

// emitStderrAsWarnOnSuccess publishes captured stderr as a warn-level
// run-log event when the process exited cleanly but produced diagnostic
// output. Without this, launch-time diagnostics (e.g. bwrap warnings or
// the chdir failure that reproduced the swarm-manager 134ms-no-output
// regression) are silently dropped on the success path because
// errorOutput is otherwise only consulted when err != nil.
//
// Truncates at 4 KB to keep run-event payload sizes bounded; operators
// reading the raw process logs on disk get the unabridged output.
func emitStderrAsWarnOnSuccess(runID uuid.UUID, sink EventSink, stderr string) {
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" || sink == nil {
		return
	}
	_ = sink.Emit(domain.NewLogEvent(
		runID,
		"warn",
		fmt.Sprintf("Runner stderr (process exited cleanly):\n%s", truncateForLog(trimmed, 4096)),
	))
}

// truncateForLog returns s capped at max bytes, suffixing with a marker
// when truncation occurred. Callers use this for run-event payloads that
// must not balloon the event store.
func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…[truncated]"
}

// Verify interface compliance
var _ Runner = (*ClaudeCodeRunner)(nil)

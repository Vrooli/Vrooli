// Package core provides the generic [runner.Runner] implementation that
// drives any agent CLI through a [codecs.Codec].
//
// The agent-manager refactor that introduced this package consolidated
// three near-identical runner implementations (Claude Code, Codex,
// OpenCode — each ~1.7k LOC, ~80% duplicated plumbing) into one shared
// machinery + thin per-runner codec files. Adding a new agent now means
// implementing [codecs.Codec] in one ~250 LOC file; no changes to core/
// are required.
//
// What core owns:
//   - process launch + lifecycle (selecting host vs sandbox launcher,
//     registering the [runner.LaunchedProcess] for cancellation)
//   - stdout scanning (buffered line reader sized via tunable levers)
//   - stderr accumulation (background goroutine into a strings.Builder)
//   - durable-transcript path (stdout tee + live [runner.Consume] tail
//     plus a final drain after process exit)
//   - status / log / completion event emission (the only places that
//     touch [runner.EventSink] in the runner layer)
//   - result classification (exit code, ctx-cancelled detection, typed
//     terminal error propagation, rate-limit-flip via codec hook)
//
// What codecs own (see [codecs.Codec]):
//   - CLI args, env, stdin prompt
//   - per-line stdout JSON decoding (DecodeStreamLine)
//   - transcript-line decoding for replay (ParseTranscriptLine)
//   - per-run state (text buffers, tool accumulators, session ID)
//   - capabilities, available, probe-model
//   - status-message labels
package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// runnerLog returns the runner-core's component-tagged logger,
// pre-tagged with codec/runner type so every log line is filterable
// without per-call boilerplate.
func (r *Runner) runnerLog() *slog.Logger {
	return obs.Component("runner-core").With(
		obs.KeyRunnerType, string(r.codec.Type()),
	)
}

// launcherLabel returns "sandbox" or "host" so structured logs and
// lifecycle events can distinguish the launch path. Identification is
// by Go type name to avoid coupling core to the parent runner package's
// concrete types via an exported interface.
func launcherLabel(l runner.Launcher) string {
	if l == nil {
		return ""
	}
	name := fmt.Sprintf("%T", l)
	switch {
	case strings.Contains(name, "SandboxLauncher"), strings.Contains(name, "sandboxLauncher"):
		return "sandbox"
	case strings.Contains(name, "HostLauncher"), strings.Contains(name, "hostLauncher"):
		return "host"
	default:
		return name
	}
}

// Runner is the generic [runner.Runner] implementation parameterised by
// a [codecs.Codec]. One Runner instance per codec is created at
// startup; it is safe for concurrent Execute/Continue/Stop calls across
// different RunIDs.
type Runner struct {
	codec    codecs.Codec
	selector launcherSelector

	mu       sync.Mutex
	launched map[uuid.UUID]runner.LaunchedProcess
}

// launcherSelector is the host-vs-sandbox launcher picker. It mirrors
// the shape of [runner.launcherSelector] (which lives in the parent
// package and is internal to it) so core can hold a swappable selector
// behind an interface without forcing the parent package to export its
// concrete type.
type launcherSelector interface {
	Pick(ctx context.Context, req runner.ExecuteRequest) runner.Launcher
	PickFor(ctx context.Context, runID uuid.UUID, cfg *domain.RunConfig, sandboxID *uuid.UUID, sink runner.EventSink) runner.Launcher
	SetSandboxLauncherFactory(factory runner.SandboxLauncherFactory)
}

// NewRunner constructs a generic Runner around the supplied Codec. The
// host launcher and (optional) sandbox factory follow the same wiring
// pattern as the legacy per-runner constructors: the host launcher is
// used for tracking-mode runs; the factory is consulted by the selector
// when a request resolves to a protected-mode sandbox.
func NewRunner(codec codecs.Codec, host runner.Launcher, sandboxFactory runner.SandboxLauncherFactory) *Runner {
	return &Runner{
		codec:    codec,
		selector: runner.NewLauncherSelector(host, sandboxFactory),
		launched: make(map[uuid.UUID]runner.LaunchedProcess),
	}
}

// SetSandboxLauncherFactory swaps in (or removes) the protected-mode
// factory. Used by main.go where the sandbox provider is constructed
// after the runner registry; tests can also use it to inject a mock.
func (r *Runner) SetSandboxLauncherFactory(factory runner.SandboxLauncherFactory) {
	r.selector.SetSandboxLauncherFactory(factory)
}

// Type satisfies [runner.Runner] by delegating to the codec.
func (r *Runner) Type() domain.RunnerType { return r.codec.Type() }

// Capabilities satisfies [runner.Runner] by delegating to the codec.
func (r *Runner) Capabilities() runner.Capabilities { return r.codec.Capabilities() }

// IsAvailable satisfies [runner.Runner] by consulting the codec.
func (r *Runner) IsAvailable(ctx context.Context) (bool, string) {
	return r.codec.Available(ctx)
}

// ProbeModel satisfies [runner.Runner] by consulting the codec.
func (r *Runner) ProbeModel(ctx context.Context, modelID string) error {
	return r.codec.ProbeModel(ctx, modelID)
}

// Classify satisfies [runner.Runner] by consulting the codec's
// structured-signal classifier.
func (r *Runner) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	return r.codec.Classify(stderr, exitCode)
}

// ParseTranscriptLine satisfies [runner.TranscriptParser] for single-line
// callers. Multi-line transcript consumers should call NewTranscriptParser
// and reuse the returned parser for the whole stream.
func (r *Runner) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	return r.codec.ParseTranscriptLine(runID, line)
}

// NewTranscriptParser satisfies [runner.TranscriptParserFactory].
func (r *Runner) NewTranscriptParser() runner.TranscriptParser {
	return r.codec.NewTranscriptParser()
}

// TagEnvKey satisfies [runner.AgentLaunchInfo], exposing the codec's per-run
// tag env key so the interactive substrate can prepend "<key>=<tag>" to the
// launch command it hands web-console (the reconciler reads it from /proc).
func (r *Runner) TagEnvKey() string { return r.codec.TagEnvKey() }

// BinaryPath satisfies [runner.AgentLaunchInfo], exposing the codec's resolved
// CLI binary path for the interactive launch command.
func (r *Runner) BinaryPath() string { return r.codec.BinaryPath() }

// Stop attempts a graceful shutdown of a running agent. SIGTERM is sent
// to the process group with a bounded grace period before SIGKILL
// escalation; ctx cancellation is honoured as immediate kill.
func (r *Runner) Stop(ctx context.Context, runID uuid.UUID) error {
	r.mu.Lock()
	proc, ok := r.launched[runID]
	r.mu.Unlock()
	if !ok {
		return domain.NewNotFoundErrorWithID("Run", runID.String())
	}

	proc.Signal(config.DefaultLevers().Heartbeat.RunnerSignalGracePeriod)
	if ctx != nil {
		go func() {
			<-ctx.Done()
			proc.Kill()
		}()
	}
	return nil
}

// Execute runs the agent with the given configuration. Routes to the
// durable-transcript path when [runner.ExecuteRequest.Transcript] is
// supplied; otherwise uses the legacy in-memory streaming path.
func (r *Runner) Execute(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
	if avail, msg := r.codec.Available(ctx); !avail {
		return nil, &domain.RunnerError{
			RunnerType:  r.codec.Type(),
			Operation:   "availability",
			Cause:       errors.New(msg),
			IsTransient: false,
		}
	}

	startTime := time.Now()
	state := r.codec.NewState()

	if req.Transcript != nil {
		return r.executeWithDurableTranscript(ctx, req, state, startTime)
	}

	args := r.codec.BuildArgs(state, req)
	prompt := r.codec.BuildPrompt(req.Prompt, req.Attachments)
	env := r.codec.BuildEnv(req.GetTag(), req.Environment)
	launcher := r.selector.Pick(ctx, req)
	launcherType := launcherLabel(launcher)
	sandboxIDStr := ""
	if req.SandboxID != nil {
		sandboxIDStr = req.SandboxID.String()
	}

	launchReq := runner.BuildEnvWrappedLaunchRequest(
		r.codec.TagEnvKey(), r.codec.BinaryPath(), args,
		req.GetTag(), prompt, env, req.WorkingDir,
	)

	r.runnerLog().Info("agent launch",
		obs.KeyRunID, req.RunID.String(),
		obs.KeyLauncherType, launcherType,
		obs.KeySandboxID, sandboxIDStr,
		"binary", filepath.Base(r.codec.BinaryPath()),
		"argCount", len(args),
		"envKeys", redactedEnvKeys(env),
		"workdir", req.WorkingDir,
	)

	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		r.runnerLog().Error("agent launch failed",
			obs.KeyRunID, req.RunID.String(),
			obs.KeyLauncherType, launcherType,
			obs.KeyError, err.Error(),
		)
		return nil, &domain.RunnerError{
			RunnerType: r.codec.Type(),
			Operation:  "execute",
			Cause:      err,
		}
	}

	r.runnerLog().Info("agent launched",
		obs.KeyRunID, req.RunID.String(),
		"pid", proc.PID(),
		obs.KeyLauncherType, launcherType,
		obs.KeySandboxID, sandboxIDStr,
	)
	obs.EmitRunnerAcquired(req.EventSink, req.RunID, obs.RunnerAcquiredFields{
		RunnerType:   r.codec.Type(),
		LauncherType: launcherType,
		SandboxID:    req.SandboxID,
	})

	r.registerProcess(req.RunID, proc)
	defer r.deregisterProcess(req.RunID, proc)

	r.emitStart(req.EventSink, req.RunID, r.codec.Labels().StartMessage, false)

	metrics := runner.ExecutionMetrics{}
	var lastAssistantMessage string
	errorOutput := r.spawnStderrAccumulator(proc)

	r.scanStream(ctx, scanInputs{
		runID:         req.RunID,
		state:         state,
		proc:          proc,
		sink:          req.EventSink,
		metrics:       &metrics,
		lastAssistant: &lastAssistantMessage,
	})

	waitErr := proc.Wait()
	errorOutput.wait()
	stderr := errorOutput.String()

	duration := time.Since(startTime)
	result := r.classifyResult(ctx, waitErr, stderr, state, &metrics, lastAssistantMessage, duration, false)

	if result.Success {
		runner.EmitStderrAsWarnOnSuccess(req.RunID, req.EventSink, stderr)
	}

	r.logRunnerExited(req.RunID, req.EventSink, result, duration)

	r.emitCompletion(req.EventSink, req.RunID, result.Success, r.codec.Labels().EndMessage, false)

	return result, nil
}

// logRunnerExited emits the runner-exited lifecycle event + a structured
// log line. Centralised so both Execute and Continue funnel into the
// same recorder; future adjustments (e.g. capturing stderr length) live
// in one place.
func (r *Runner) logRunnerExited(runID uuid.UUID, sink runner.EventSink, result *runner.ExecuteResult, duration time.Duration) {
	if result == nil {
		return
	}
	exitCode := result.ExitCode
	terminalCode := ""
	if result.TerminalError != nil {
		if dom, ok := result.TerminalError.(domain.DomainError); ok {
			terminalCode = string(dom.Code())
		}
	}
	r.runnerLog().Info("agent exited",
		obs.KeyRunID, runID.String(),
		obs.KeyExitCode, exitCode,
		obs.KeyDuration, duration.Milliseconds(),
		obs.KeyTerminalCode, terminalCode,
		"success", result.Success,
	)
	obs.EmitRunnerExited(sink, runID, obs.RunnerExitedFields{
		RunnerType:   r.codec.Type(),
		ExitCode:     &exitCode,
		Duration:     duration,
		TerminalCode: terminalCode,
		Success:      result.Success,
	})
}

// redactedEnvKeys returns a sorted list of env-var keys (values
// elided). Argv values can carry secrets; env values often do; keys are
// always safe and useful for diagnostic comparison ("did this run get
// the same env as the last successful one?"). Keys are passed as-is
// without further filtering — callers should not put secrets into the
// key namespace.
//
// Accepts the os.Environ()-shaped slice ("KEY=value") that codec
// BuildEnv emits.
func redactedEnvKeys(env []string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for _, kv := range env {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		keys = append(keys, kv[:eq])
	}
	// Cheap order: caller wants stability, not lexical correctness.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// Continue resumes an existing session with a follow-up message.
// Returns a typed *domain.RunnerError (ErrCodeRunnerSessionExpired or
// ErrCodeRunnerSessionStateLost) when the codec recognises a session
// or rollout-state failure shape — see [codecs.Codec.ClassifyTerminalError].
func (r *Runner) Continue(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
	if avail, msg := r.codec.Available(ctx); !avail {
		return nil, &domain.RunnerError{
			RunnerType:  r.codec.Type(),
			Operation:   "availability",
			Cause:       errors.New(msg),
			IsTransient: false,
		}
	}
	if req.SessionID == "" {
		return nil, domain.NewRunnerSessionExpiredError(r.codec.Type(),
			errors.New("continue called with empty session id"))
	}

	startTime := time.Now()
	state := r.codec.NewState()
	tag := r.codec.ContinueTag(req)

	if req.Transcript != nil {
		return r.continueWithDurableTranscript(ctx, req, state, tag, startTime)
	}

	args := r.codec.BuildContinueArgs(state, req)
	prompt := r.codec.BuildPrompt(req.Prompt, req.Attachments)
	env := r.codec.BuildEnv(tag, req.Environment)
	launcher := r.selector.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)

	launchReq := runner.BuildEnvWrappedLaunchRequest(
		r.codec.TagEnvKey(), r.codec.BinaryPath(), args,
		tag, prompt, env, req.WorkingDir,
	)
	proc, err := launcher.Launch(ctx, launchReq)
	if err != nil {
		return nil, &domain.RunnerError{
			RunnerType: r.codec.Type(),
			Operation:  "continue",
			Cause:      err,
		}
	}

	r.registerProcess(req.RunID, proc)
	defer r.deregisterProcess(req.RunID, proc)

	r.emitStart(req.EventSink, req.RunID, r.codec.Labels().ContinueStartMessage, true)

	metrics := runner.ExecutionMetrics{}
	var lastAssistantMessage string
	errorOutput := r.spawnStderrAccumulator(proc)

	r.scanStream(ctx, scanInputs{
		runID:         req.RunID,
		state:         state,
		proc:          proc,
		sink:          req.EventSink,
		metrics:       &metrics,
		lastAssistant: &lastAssistantMessage,
	})

	waitErr := proc.Wait()
	errorOutput.wait()
	stderr := errorOutput.String()

	duration := time.Since(startTime)
	result := r.classifyResult(ctx, waitErr, stderr, state, &metrics, lastAssistantMessage, duration, true)

	if result.Success {
		runner.EmitStderrAsWarnOnSuccess(req.RunID, req.EventSink, stderr)
	}

	// Preserve the session ID for further continuations: codecs may not
	// emit a fresh session_id on every continuation, so keep the input
	// when the stream did not reveal a new one.
	if result.SessionID == "" {
		result.SessionID = req.SessionID
	}

	// Codec-specific terminal-error classification. The codec recognises
	// "session/thread gone" vs "rollout-writer dropped the thread" vs
	// other failure shapes and returns a typed RunnerError so the
	// orchestration timeline shows a stable code (RUNNER_SESSION_EXPIRED
	// / RUNNER_SESSION_STATE_LOST) rather than a bare INTERNAL.
	if !result.Success {
		if classified := r.codec.ClassifyTerminalError(result.ErrorMessage, result.ExitCode); classified != nil {
			r.runnerLog().Warn("classified continue failure",
				obs.KeyRunID, req.RunID.String(),
				obs.KeyDuration, duration.Milliseconds(),
				obs.KeyTerminalCode, string(classified.Code()),
			)
			return nil, classified
		}
	}

	r.logRunnerExited(req.RunID, req.EventSink, result, duration)

	r.emitCompletion(req.EventSink, req.RunID, result.Success, r.codec.Labels().ContinueEndMessage, true)

	return result, nil
}

// scanInputs bundles the per-call state passed into scanStream. Avoids
// a long parameter list and keeps each scan-loop iteration focused on
// one line.
type scanInputs struct {
	runID         uuid.UUID
	state         codecs.State
	proc          runner.LaunchedProcess
	sink          runner.EventSink
	metrics       *runner.ExecutionMetrics
	lastAssistant *string
}

// scanStream consumes proc.Stdout() line by line, dispatching to the
// codec for decoding and emitting parsed events through sink. Honours
// codec.OnEarlyTerminate for runners that signal completion via an
// in-stream sentinel (e.g. OpenCode's step_finish).
func (r *Runner) scanStream(ctx context.Context, in scanInputs) {
	scanner := bufio.NewScanner(in.proc.Stdout())
	scanner.Buffer(make([]byte, 0, 64*1024), config.DefaultLevers().Scanner.StdoutMaxLineBytes)

	for scanner.Scan() {
		in.proc.ResetIdleTimer()
		line := scanner.Text()
		if line == "" {
			continue
		}

		events, err := r.codec.DecodeStreamLine(in.state, in.runID, line)
		if err != nil {
			if in.sink != nil {
				_ = in.sink.Emit(domain.NewLogEvent(
					in.runID, "warn",
					fmt.Sprintf("Failed to parse event: %v", err),
				))
			}
			continue
		}
		for _, event := range events {
			if event == nil {
				continue
			}
			r.codec.UpdateMetrics(event, in.metrics, in.lastAssistant)
			if in.sink != nil {
				_ = in.sink.Emit(event)
			}
		}

		// OnEarlyTerminate runs *after* the line's events are decoded
		// and emitted, so terminal sentinel lines (e.g. OpenCode's
		// terminal step_finish) still surface their cost/message events
		// before the scanner loop exits. The codec stashes "did we just
		// see a terminator?" on state during DecodeStreamLine and reads
		// it back here.
		if r.codec.OnEarlyTerminate(in.state, line) {
			break
		}
	}

	if in.proc.TimedOut() && in.sink != nil {
		_ = in.sink.Emit(domain.NewLogEvent(
			in.runID, "warn",
			fmt.Sprintf("Process idle for %v without output; killed process group", runner.DefaultStreamIdleTimeout),
		))
	}
	if scannerErr := scanner.Err(); scannerErr != nil && in.sink != nil {
		_ = in.sink.Emit(domain.NewLogEvent(
			in.runID, "warn",
			fmt.Sprintf("Scanner error (possible buffer overflow or I/O error): %v", scannerErr),
		))
	}
}

// classifyResult produces the final ExecuteResult from a wait error,
// captured stderr, and codec-managed state. Handles ctx cancellation,
// typed exit-code extraction, typed-terminal-error propagation, and
// invokes the codec's PostClassify hook.
func (r *Runner) classifyResult(
	ctx context.Context,
	waitErr error,
	errorOutput string,
	state codecs.State,
	metrics *runner.ExecutionMetrics,
	lastAssistant string,
	duration time.Duration,
	isContinue bool,
) *runner.ExecuteResult {
	result := &runner.ExecuteResult{
		Duration: duration,
		Metrics:  *metrics,
	}

	if waitErr != nil {
		switch {
		case ctx.Err() == context.Canceled:
			result.Success = false
			result.ExitCode = -1
			if isContinue {
				result.ErrorMessage = "continuation cancelled"
			} else {
				result.ErrorMessage = "execution cancelled"
			}
		default:
			if code, ok := runner.ExtractExitCode(waitErr); ok {
				result.Success = false
				result.ExitCode = code
				result.ErrorMessage = errorOutput
			} else {
				result.Success = false
				result.ExitCode = -1
				result.ErrorMessage = waitErr.Error()
				if _, ok := waitErr.(domain.DomainError); ok {
					result.TerminalError = waitErr
				}
			}
		}
	} else {
		result.Success = true
		result.ExitCode = 0
		result.Summary = &domain.RunSummary{
			Description:   lastAssistant,
			TurnsUsed:     metrics.TurnsUsed,
			TokensUsed:    runner.TotalTokens(*metrics),
			CostEstimate:  metrics.CostEstimateUSD,
			ContextTokens: metrics.TokensInput,
		}
	}

	r.codec.PostClassify(state, result)

	if sid := state.SessionID(); sid != "" {
		result.SessionID = sid
	}

	// Apply codec-specific terminal-error classification on the failure
	// path. Sets result.TerminalError so the orchestration layer's
	// promotion-into-execErr (phases/execute.go) emits a typed event
	// with the right ErrorCode rather than ErrCodeInternal.
	//
	// Skipped when waitErr already produced a domain-typed
	// TerminalError (e.g. SANDBOX_NO_EXIT_INFO from the launcher) so
	// we don't overwrite a more-specific upstream classification.
	if !result.Success && result.TerminalError == nil {
		if classified := r.codec.ClassifyTerminalError(result.ErrorMessage, result.ExitCode); classified != nil {
			result.TerminalError = classified
		}
	}

	return result
}

// registerProcess records a launched process so Stop() can reach it
// and the deferred Wait() chain can clean it up.
func (r *Runner) registerProcess(runID uuid.UUID, proc runner.LaunchedProcess) {
	r.mu.Lock()
	r.launched[runID] = proc
	r.mu.Unlock()
}

// deregisterProcess removes the process from the launched map and waits
// for it to fully exit (idempotent — Wait can be called multiple times).
func (r *Runner) deregisterProcess(runID uuid.UUID, proc runner.LaunchedProcess) {
	r.mu.Lock()
	delete(r.launched, runID)
	r.mu.Unlock()
	_ = proc.Wait()
}

// emitStart sends the "execution started" status event when sink != nil.
// On Execute, the previous status is "starting"; on Continue both ends
// are "running" because Continue takes an already-running run-row from
// the orchestrator.
func (r *Runner) emitStart(sink runner.EventSink, runID uuid.UUID, message string, isContinue bool) {
	if sink == nil {
		return
	}
	from := string(domain.RunStatusStarting)
	if isContinue {
		from = string(domain.RunStatusRunning)
	}
	_ = sink.Emit(domain.NewStatusEvent(
		runID, from, string(domain.RunStatusRunning), message,
	))
}

// emitCompletion sends the final status event and closes the sink.
// Closing the sink signals downstream consumers (broadcaster, recovery
// drainer) that no more events will arrive for this run.
func (r *Runner) emitCompletion(sink runner.EventSink, runID uuid.UUID, success bool, message string, _ bool) {
	if sink == nil {
		return
	}
	finalStatus := string(domain.RunStatusComplete)
	if !success {
		finalStatus = string(domain.RunStatusFailed)
	}
	_ = sink.Emit(domain.NewStatusEvent(
		runID, string(domain.RunStatusRunning), finalStatus, message,
	))
	_ = sink.Close()
}

// stderrAccumulator wraps the stderr-drainer goroutine with a done
// channel so callers can synchronise on full drain rather than racing
// against proc.Wait()'s return.
type stderrAccumulator struct {
	buf  strings.Builder
	mu   sync.Mutex
	done chan struct{}
}

// String reports the captured stderr. Safe to call after [stderrAccumulator.wait].
func (a *stderrAccumulator) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buf.String()
}

// wait blocks until the drainer goroutine has consumed everything from
// proc.Stderr() (which happens when the producer closes the pipe). The
// caller MUST invoke this after proc.Wait() returns and before reading
// String(); otherwise the captured stderr may be incomplete.
func (a *stderrAccumulator) wait() { <-a.done }

// spawnStderrAccumulator launches a goroutine that drains proc.Stderr()
// into an internal buffer. Use [stderrAccumulator.wait] after
// proc.Wait() to ensure the drainer has finished before reading.
func (r *Runner) spawnStderrAccumulator(proc runner.LaunchedProcess) *stderrAccumulator {
	a := &stderrAccumulator{done: make(chan struct{})}
	go func() {
		defer close(a.done)
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			a.mu.Lock()
			a.buf.WriteString(scanner.Text())
			a.buf.WriteString("\n")
			a.mu.Unlock()
		}
	}()
	return a
}

// =============================================================================
// Durable-transcript path
// =============================================================================

// executeWithDurableTranscript is the Execute() variant that writes
// stdout to a durable file and runs a live [runner.Consume] tail so
// agent-manager can resume after a restart.
func (r *Runner) executeWithDurableTranscript(
	ctx context.Context,
	req runner.ExecuteRequest,
	state codecs.State,
	startTime time.Time,
) (*runner.ExecuteResult, error) {
	args := r.codec.BuildArgs(state, req)
	prompt := r.codec.BuildPrompt(req.Prompt, req.Attachments)
	env := r.codec.BuildEnv(req.GetTag(), req.Environment)
	launcher := r.selector.Pick(ctx, req)

	launchReq := runner.BuildEnvWrappedLaunchRequest(
		r.codec.TagEnvKey(), r.codec.BinaryPath(), args,
		req.GetTag(), prompt, env, req.WorkingDir,
	)
	return r.runDurable(ctx, durableInputs{
		runID:        req.RunID,
		sandboxID:    req.SandboxID,
		sink:         req.EventSink,
		transcript:   req.Transcript,
		state:        state,
		startTime:    startTime,
		launcher:     launcher,
		request:      launchReq,
		startMessage: r.codec.Labels().StartMessage,
		endMessage:   r.codec.Labels().EndMessage,
		isContinue:   false,
	})
}

// continueWithDurableTranscript is the Continue() variant that writes
// stdout to a durable file. Behaves like executeWithDurableTranscript
// but uses the codec's continue-shape (different args, different tag).
func (r *Runner) continueWithDurableTranscript(
	ctx context.Context,
	req runner.ContinueRequest,
	state codecs.State,
	tag string,
	startTime time.Time,
) (*runner.ExecuteResult, error) {
	args := r.codec.BuildContinueArgs(state, req)
	prompt := r.codec.BuildPrompt(req.Prompt, req.Attachments)
	env := r.codec.BuildEnv(tag, req.Environment)
	launcher := r.selector.PickFor(ctx, req.RunID, req.GetConfig(), req.SandboxID, req.EventSink)

	launchReq := runner.BuildEnvWrappedLaunchRequest(
		r.codec.TagEnvKey(), r.codec.BinaryPath(), args,
		tag, prompt, env, req.WorkingDir,
	)
	result, err := r.runDurable(ctx, durableInputs{
		runID:        req.RunID,
		sandboxID:    req.SandboxID,
		sink:         req.EventSink,
		transcript:   req.Transcript,
		state:        state,
		startTime:    startTime,
		launcher:     launcher,
		request:      launchReq,
		startMessage: r.codec.Labels().ContinueStartMessage,
		endMessage:   r.codec.Labels().ContinueEndMessage,
		isContinue:   true,
	})
	if result != nil && result.SessionID == "" {
		result.SessionID = req.SessionID
	}
	return result, err
}

type durableInputs struct {
	runID        uuid.UUID
	sandboxID    *uuid.UUID
	sink         runner.EventSink
	transcript   *runner.TranscriptConfig
	state        codecs.State
	startTime    time.Time
	launcher     runner.Launcher
	request      runner.LaunchRequest
	startMessage string
	endMessage   string
	isContinue   bool
}

// runDurable runs a launched agent through the durable-transcript path:
// stdout is tee'd to the on-disk transcript while a background
// [runner.Consume] tail parses it back into events for the live sink.
// After the process exits, a final [runner.Consume] drain catches any
// trailing transcript bytes the live tail might have raced past.
//
// This is the path that makes restart-resume work. When agent-manager
// dies mid-run, the agent process keeps writing transcript.ndjson, and
// the orchestrator's Recovery sweep on next boot finds the live process,
// reattaches via the same Consume loop, and continues from
// transcript.OnAdvance's last cursor.
func (r *Runner) runDurable(ctx context.Context, in durableInputs) (*runner.ExecuteResult, error) {
	if in.transcript == nil || in.transcript.StdoutFile == nil {
		return nil, &domain.RunnerError{
			RunnerType: r.codec.Type(),
			Operation:  "execute",
			Cause:      errors.New("durable transcript stdout file is required"),
		}
	}

	launcherType := launcherLabel(in.launcher)
	sandboxIDStr := ""
	if in.sandboxID != nil && *in.sandboxID != uuid.Nil {
		sandboxIDStr = in.sandboxID.String()
	}
	r.runnerLog().Info("agent launch (durable)",
		obs.KeyRunID, in.runID.String(),
		obs.KeyLauncherType, launcherType,
		obs.KeySandboxID, sandboxIDStr,
		"binary", filepath.Base(r.codec.BinaryPath()),
		"argCount", len(in.request.Args),
		"workdir", in.request.WorkingDir,
		"transcript", in.transcript.TranscriptPath,
	)

	proc, err := in.launcher.Launch(ctx, in.request)
	if err != nil {
		r.runnerLog().Error("agent launch failed (durable)",
			obs.KeyRunID, in.runID.String(),
			obs.KeyLauncherType, launcherType,
			obs.KeySandboxID, sandboxIDStr,
			obs.KeyError, err.Error(),
		)
		return nil, &domain.RunnerError{
			RunnerType: r.codec.Type(),
			Operation:  "execute",
			Cause:      err,
		}
	}

	r.runnerLog().Info("agent launched (durable)",
		obs.KeyRunID, in.runID.String(),
		"pid", proc.PID(),
		obs.KeyLauncherType, launcherType,
		obs.KeySandboxID, sandboxIDStr,
	)
	obs.EmitRunnerAcquired(in.sink, in.runID, obs.RunnerAcquiredFields{
		RunnerType:   r.codec.Type(),
		LauncherType: launcherType,
		SandboxID:    in.sandboxID,
	})

	r.registerProcess(in.runID, proc)
	defer func() {
		r.mu.Lock()
		delete(r.launched, in.runID)
		r.mu.Unlock()
	}()

	if in.transcript.OnProcessStart != nil {
		if err := in.transcript.OnProcessStart(proc.PID(), proc.PID()); err != nil {
			proc.Kill()
			_ = proc.Wait()
			return nil, err
		}
	}

	r.emitStart(in.sink, in.runID, in.startMessage, in.isContinue)

	metrics := runner.ExecutionMetrics{}
	var lastAssistantMessage string
	var errorOutput strings.Builder
	var terminal *runner.TranscriptTerminal

	// Tee stdout to the transcript file. The launcher's stdout reader
	// closes on process exit, so this goroutine drains and returns.
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		_, _ = io.Copy(in.transcript.StdoutFile, proc.Stdout())
	}()

	// Stderr drainer mirrors into transcript.StderrFile when present.
	go func() {
		scanner := bufio.NewScanner(proc.Stderr())
		for scanner.Scan() {
			line := scanner.Text()
			errorOutput.WriteString(line)
			errorOutput.WriteString("\n")
			if in.transcript.StderrFile != nil {
				_, _ = io.WriteString(in.transcript.StderrFile, line+"\n")
			}
		}
	}()

	// Live tail: parses the transcript as it grows, dispatching events.
	consumeCtx, cancelConsume := context.WithCancel(context.Background())
	liveDone := make(chan struct{})
	var liveCursor int64
	transcriptParser := r.codec.NewTranscriptParser()
	go func() {
		defer close(liveDone)
		cursor, liveTerminal, _ := runner.Consume(consumeCtx, runner.ConsumeArgs{
			RunID:       in.runID,
			Transcript:  in.transcript.TranscriptPath,
			Live:        true,
			ParseFn:     transcriptParser.ParseTranscriptLine,
			EventSink:   in.sink,
			OnAdvance:   in.transcript.OnAdvance,
			OnSessionID: in.transcript.OnSessionID,
			OnEvents: func(events []*domain.RunEvent) {
				for _, evt := range events {
					r.codec.UpdateMetrics(evt, &metrics, &lastAssistantMessage)
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

	// Final drain: catch up on any trailing bytes the live tail raced past.
	finalCursor, finalTerminal, drainErr := runner.Consume(context.Background(), runner.ConsumeArgs{
		RunID:       in.runID,
		Transcript:  in.transcript.TranscriptPath,
		StartAt:     liveCursor,
		ParseFn:     transcriptParser.ParseTranscriptLine,
		EventSink:   in.sink,
		OnAdvance:   in.transcript.OnAdvance,
		OnSessionID: in.transcript.OnSessionID,
		OnEvents: func(events []*domain.RunEvent) {
			for _, evt := range events {
				r.codec.UpdateMetrics(evt, &metrics, &lastAssistantMessage)
			}
		},
	})
	if finalTerminal != nil {
		terminal = finalTerminal
	}
	if in.transcript.OnAdvance != nil {
		_ = in.transcript.OnAdvance(finalCursor, 0)
	}

	result := &runner.ExecuteResult{
		Duration: time.Since(in.startTime),
		Metrics:  metrics,
	}

	switch {
	case waitErr != nil && ctx.Err() == context.Canceled:
		result.Success = false
		result.ExitCode = -1
		if in.isContinue {
			result.ErrorMessage = "continuation cancelled"
		} else {
			result.ErrorMessage = "execution cancelled"
		}
	case waitErr != nil:
		if code, ok := runner.ExtractExitCode(waitErr); ok {
			result.Success = false
			result.ExitCode = code
			result.ErrorMessage = errorOutput.String()
		} else {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = waitErr.Error()
			if _, ok := waitErr.(domain.DomainError); ok {
				result.TerminalError = waitErr
			}
		}
	case terminal != nil:
		result.Success = terminal.Success
		result.ExitCode = terminal.ExitCode
		result.ErrorMessage = terminal.ErrorMessage
		if terminal.Summary != nil {
			result.Summary = terminal.Summary
		}
	default:
		result.Success = true
		result.ExitCode = 0
	}
	if drainErr != nil && result.Success {
		result.Success = false
		result.ExitCode = -1
		result.ErrorMessage = drainErr.Error()
	}
	if result.Success && result.Summary == nil {
		result.Summary = runner.TerminalSummaryFromMessage(lastAssistantMessage, metrics)
	} else if !result.Success && strings.TrimSpace(result.ErrorMessage) == "" {
		result.ErrorMessage = strings.TrimSpace(errorOutput.String())
	}

	r.codec.PostClassify(in.state, result)

	if result.Success {
		runner.EmitStderrAsWarnOnSuccess(in.runID, in.sink, errorOutput.String())
	}

	if sid := in.state.SessionID(); sid != "" {
		result.SessionID = sid
	}

	r.logRunnerExited(in.runID, in.sink, result, time.Since(in.startTime))

	r.emitCompletion(in.sink, in.runID, result.Success, in.endMessage, in.isContinue)

	return result, nil
}

// PID returns the host-visible process ID for a registered run, or 0 if
// the run is not currently registered. Used by the reconciler to spot
// stranded children.
func (r *Runner) PID(runID uuid.UUID) int {
	r.mu.Lock()
	proc, ok := r.launched[runID]
	r.mu.Unlock()
	if !ok || proc == nil {
		return 0
	}
	return proc.PID()
}

// LaunchedProcess returns the registered LaunchedProcess for runID, or
// nil if no process is currently registered. Exposed for tests that
// need to manipulate the underlying process directly.
func (r *Runner) LaunchedProcess(runID uuid.UUID) runner.LaunchedProcess {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.launched[runID]
}

// Compile-time interface checks.
var (
	_ runner.Runner           = (*Runner)(nil)
	_ runner.TranscriptParser = (*Runner)(nil)
)

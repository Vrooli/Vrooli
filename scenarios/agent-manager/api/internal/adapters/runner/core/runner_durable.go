// Responsibility: the durable-transcript execution path (runDurable and its
// phase functions).
package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// runDurable runs a launched agent through the durable-transcript path:
// host launchers write stdout/stderr directly to the run's durable files,
// while a background [runner.Consume] tail parses stdout for the live sink.
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
	d := &durableRun{r: r, in: in, toolLimitTriggered: make(chan struct{})}
	if err := d.markInvocationBoundary(); err != nil {
		return nil, err
	}
	if err := d.launch(ctx); err != nil {
		return nil, err
	}
	defer d.unregister()

	if in.transcript.OnProcessStart != nil {
		if err := in.transcript.OnProcessStart(d.proc.PID(), d.proc.PID()); err != nil {
			d.proc.Kill()
			_ = d.proc.Wait()
			return nil, err
		}
	}

	r.emitStart(in.sink, in.runID, in.startMessage, in.isContinue)

	d.teeOutput()
	d.startLiveTail()
	waitErr := d.awaitExit(ctx)
	drainErr := d.drainTranscript()
	waitErr = d.applyToolLimit(waitErr)
	d.recoverUsage()
	result := d.buildResult(ctx, waitErr, drainErr)
	d.finish(ctx, result)
	return result, nil
}

// durableRun is the mutable state of one durable invocation. The live tail
// goroutine owns metrics, observedEvents, terminal and liveCursor until
// liveDone closes; the runner reads them only after that.
type durableRun struct {
	r  *Runner
	in durableInputs

	proc          runner.LaunchedProcess
	directOutput  bool
	launcherType  string
	sandboxIDStr  string
	stderrPath    string
	stderrStart   int64
	transcriptEnd int64 // transcript size when this invocation began

	metrics              runner.ExecutionMetrics
	lastAssistantMessage string
	observedEvents       []*domain.RunEvent
	errorOutput          strings.Builder
	terminal             *runner.TranscriptTerminal

	toolLimitTriggered chan struct{}
	toolLimitOnce      sync.Once

	// Durable replay parses with a fresh transcript parser rather than the
	// live stdout state. Preserve its discovered session id explicitly so a
	// recovered/persisted transcript has the same continuation identity as the
	// non-durable path.
	sessionMu           sync.Mutex
	transcriptSessionID string

	parser        runner.TranscriptParser
	liveCursor    int64
	liveDone      chan struct{}
	cancelConsume context.CancelFunc
	stdoutDone    chan struct{}
	stderrDone    chan struct{}
}

// markInvocationBoundary records where this invocation's bytes begin.
// Continuations append to the same durable transcript. The new parser owns
// only bytes written by this invocation, never earlier terminal evidence that
// happens to share the Run ID and file.
func (d *durableRun) markInvocationBoundary() error {
	in := d.in
	info, err := in.transcript.StdoutFile.Stat()
	if err != nil {
		return fmt.Errorf("inspect invocation transcript boundary: %w", err)
	}
	d.transcriptEnd = info.Size()
	d.liveCursor = d.transcriptEnd
	d.stderrPath = in.transcript.StderrPath
	if in.transcript.StderrFile != nil {
		if d.stderrPath == "" {
			d.stderrPath = in.transcript.StderrFile.Name()
		}
		stderrInfo, err := in.transcript.StderrFile.Stat()
		if err != nil {
			return fmt.Errorf("inspect invocation stderr boundary: %w", err)
		}
		d.stderrStart = stderrInfo.Size()
	}
	if d.transcriptEnd > 0 && in.transcript.OnAdvance != nil {
		if err := in.transcript.OnAdvance(d.transcriptEnd, 0); err != nil {
			return fmt.Errorf("persist invocation transcript boundary: %w", err)
		}
	}
	return nil
}

// launch starts the process and registers it with the runner.
func (d *durableRun) launch(ctx context.Context) error {
	r, in := d.r, d.in
	d.launcherType = launcherLabel(in.launcher)
	if in.sandboxID != nil && *in.sandboxID != uuid.Nil {
		d.sandboxIDStr = in.sandboxID.String()
	}
	r.runnerLog().Info("agent launch (durable)",
		obs.KeyRunID, in.runID.String(),
		obs.KeyLauncherType, d.launcherType,
		obs.KeySandboxID, d.sandboxIDStr,
		"binary", filepath.Base(r.codec.BinaryPath()),
		"argCount", len(in.request.Args),
		"workdir", in.request.WorkingDir,
		"transcript", in.transcript.TranscriptPath,
	)

	var err error
	if durableLauncher, ok := in.launcher.(runner.DurableOutputLauncher); ok && in.transcript.StderrFile != nil && d.stderrPath != "" {
		d.proc, err = durableLauncher.LaunchDurable(ctx, in.request, in.transcript.StdoutFile, in.transcript.StderrFile)
		d.directOutput = true
	} else {
		d.proc, err = in.launcher.Launch(ctx, in.request)
	}
	if err != nil {
		r.runnerLog().Error("agent launch failed (durable)",
			obs.KeyRunID, in.runID.String(),
			obs.KeyLauncherType, d.launcherType,
			obs.KeySandboxID, d.sandboxIDStr,
			obs.KeyError, err.Error(),
		)
		return &domain.RunnerError{
			RunnerType:            r.codec.Type(),
			Operation:             "execute",
			Cause:                 err,
			ExecutionStartedKnown: true,
			ExecutionStarted:      false,
		}
	}

	r.runnerLog().Info("agent launched (durable)",
		obs.KeyRunID, in.runID.String(),
		"pid", d.proc.PID(),
		obs.KeyLauncherType, d.launcherType,
		obs.KeySandboxID, d.sandboxIDStr,
	)
	obs.EmitRunnerAcquired(in.sink, in.runID, obs.RunnerAcquiredFields{
		RunnerType:   r.codec.Type(),
		LauncherType: d.launcherType,
		SandboxID:    in.sandboxID,
	})

	r.registerProcess(in.runID, d.proc)
	return nil
}

func (d *durableRun) unregister() {
	d.r.mu.Lock()
	delete(d.r.launched, d.in.runID)
	d.r.mu.Unlock()
}

// stopForToolLimit kills the process once the run exceeds MaxToolCalls.
func (d *durableRun) stopForToolLimit() {
	in := d.in
	if in.config == nil || in.config.MaxToolCalls <= 0 || d.metrics.ToolCallCount < in.config.MaxToolCalls {
		return
	}
	d.toolLimitOnce.Do(func() {
		close(d.toolLimitTriggered)
		if in.sink != nil {
			_ = in.sink.Emit(domain.NewLogEvent(in.runID, "warn", toolCallLimitError(d.r.codec.Type(), in.config.MaxToolCalls).Error()))
		}
		d.proc.Kill()
	})
}

// applyToolLimit reports the tool-call limit as the process error when it
// fired.
func (d *durableRun) applyToolLimit(waitErr error) error {
	select {
	case <-d.toolLimitTriggered:
		return toolCallLimitError(d.r.codec.Type(), d.in.config.MaxToolCalls)
	default:
		return waitErr
	}
}

func (d *durableRun) onSessionID(sessionID string) error {
	if sessionID != "" {
		d.sessionMu.Lock()
		d.transcriptSessionID = sessionID
		d.sessionMu.Unlock()
	}
	if d.in.transcript.OnSessionID != nil {
		return d.in.transcript.OnSessionID(sessionID)
	}
	return nil
}

func (d *durableRun) onEvents(events []*domain.RunEvent) {
	for _, evt := range events {
		d.r.codec.UpdateMetrics(evt, &d.metrics, &d.lastAssistantMessage)
		d.observedEvents = append(d.observedEvents, evt)
		d.stopForToolLimit()
	}
}

// consumeArgs is the transcript consumer configuration shared by the live
// tail and the final drain.
func (d *durableRun) consumeArgs(startAt int64, live bool) runner.ConsumeArgs {
	in := d.in
	return runner.ConsumeArgs{
		RunID:       in.runID,
		Transcript:  in.transcript.TranscriptPath,
		StartAt:     startAt,
		Live:        live,
		ParseFn:     d.parser.ParseTranscriptLine,
		EventSink:   in.sink,
		OnAdvance:   in.transcript.OnAdvance,
		OnSessionID: d.onSessionID,
		OnEvents:    d.onEvents,
	}
}

// teeOutput copies a non-host launcher's stdout and stderr pipes to the
// durable files. Host durable output bypasses these API-owned pipes and is
// parsed from the same transcript.
func (d *durableRun) teeOutput() {
	if d.directOutput {
		return
	}
	r, in := d.r, d.in
	d.stdoutDone = make(chan struct{})
	go func() {
		defer close(d.stdoutDone)
		defer obs.RecoverToFailure("stdout transcript tee", nil)
		if _, err := io.Copy(in.transcript.StdoutFile, d.proc.Stdout()); err != nil {
			r.reportTranscriptError(in.runID, in.sink, "stdout transcript tee", err)
		}
	}()

	d.stderrDone = make(chan struct{})
	go func() {
		defer close(d.stderrDone)
		defer obs.RecoverToFailure("stderr transcript drainer", nil)
		scanner := bufio.NewScanner(d.proc.Stderr())
		for scanner.Scan() {
			line := scanner.Text()
			d.errorOutput.WriteString(line)
			d.errorOutput.WriteString("\n")
			if in.transcript.StderrFile != nil {
				if _, err := io.WriteString(in.transcript.StderrFile, line+"\n"); err != nil {
					r.reportTranscriptError(in.runID, in.sink, "stderr transcript tee", err)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			r.reportTranscriptError(in.runID, in.sink, "stderr transcript scan", err)
		}
	}()
}

// startLiveTail parses the transcript as it grows, dispatching events.
func (d *durableRun) startLiveTail() {
	r, in := d.r, d.in
	var consumeCtx context.Context
	consumeCtx, d.cancelConsume = context.WithCancel(context.Background())
	d.liveDone = make(chan struct{})
	d.parser = r.codec.NewTranscriptParser()
	if setter, ok := d.parser.(runner.TranscriptSessionSetter); ok {
		// A fresh execution's transcript starts with its session; a
		// continuation's resumes a session that already has usage.
		setter.SetTranscriptSession(in.sessionID, !in.isContinue)
	}
	if in.config != nil {
		if setter, ok := d.parser.(runner.TranscriptModelSetter); ok {
			setter.SetTranscriptModel(in.config.Model)
		}
		if setter, ok := d.parser.(runner.TranscriptBillingSetter); ok {
			setter.SetTranscriptBilling(in.config.Billing)
		}
	}
	go func() {
		defer close(d.liveDone)
		// The live consumer parses untrusted agent transcript output — the
		// most realistic panic source in the process.
		defer obs.RecoverToFailure("transcript live consumer", nil)
		cursor, liveTerminal, consumeErr := runner.Consume(consumeCtx, d.consumeArgs(d.liveCursor, true))
		if consumeErr != nil && !errors.Is(consumeErr, context.Canceled) {
			r.reportTranscriptError(in.runID, in.sink, "live transcript consume", consumeErr)
		}
		d.liveCursor = cursor
		if liveTerminal != nil {
			d.terminal = liveTerminal
		}
	}()
}

// awaitExit waits for the process and the output pipes, then stops the live
// tail. It returns the process's wait error.
func (d *durableRun) awaitExit(ctx context.Context) error {
	in := d.in
	if !d.directOutput {
		<-d.stdoutDone
		<-d.stderrDone
	}
	waitErr := d.proc.Wait()
	if d.directOutput {
		output, err := readFileRange(d.stderrPath, d.stderrStart)
		if err != nil {
			d.r.reportTranscriptError(in.runID, in.sink, "stderr transcript read", err)
		} else {
			d.errorOutput.WriteString(output)
		}
	}
	d.cancelConsume()
	<-d.liveDone
	return d.applyToolLimit(waitErr)
}

// drainTranscript catches up on any trailing bytes the live tail raced past
// and persists the final cursor. It returns the drain error.
func (d *durableRun) drainTranscript() error {
	in := d.in
	finalCursor, finalTerminal, drainErr := runner.Consume(context.Background(), d.consumeArgs(d.liveCursor, false))
	if finalTerminal != nil {
		d.terminal = finalTerminal
	}
	if drainErr != nil {
		d.r.reportTranscriptError(in.runID, in.sink, "final transcript drain", drainErr)
	}
	if in.transcript.OnAdvance != nil {
		if err := in.transcript.OnAdvance(finalCursor, 0); err != nil {
			d.r.reportTranscriptError(in.runID, in.sink, "transcript cursor persistence", err)
		}
	}
	return drainErr
}

// recoverUsage recovers token usage the interrupted stream never reported.
func (d *durableRun) recoverUsage() {
	in := d.in
	d.sessionMu.Lock()
	recoverySessionID := d.transcriptSessionID
	d.sessionMu.Unlock()
	if sid := in.state.SessionID(); sid != "" {
		recoverySessionID = sid
	}
	if recoverySessionID == "" {
		recoverySessionID = in.sessionID
	}
	d.r.recoverInterruptedUsage(interruptedUsageInputs{
		runID: in.runID, sessionID: recoverySessionID, env: in.request.Env,
		config: in.config, startedAt: in.startTime, sink: in.sink,
	}, &d.observedEvents, &d.metrics, &d.lastAssistantMessage)
}

// buildResult turns the process exit, transcript terminal and drain outcome
// into the invocation's ExecuteResult.
func (d *durableRun) buildResult(ctx context.Context, waitErr, drainErr error) *runner.ExecuteResult {
	in := d.in
	result := &runner.ExecuteResult{
		Duration: time.Since(in.startTime),
		Metrics:  d.metrics,
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
			result.ErrorMessage = d.errorOutput.String()
		} else {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMessage = waitErr.Error()
			if _, ok := waitErr.(domain.DomainError); ok {
				result.TerminalError = waitErr
			}
		}
	case d.terminal != nil:
		result.Success = d.terminal.Success
		result.ExitCode = d.terminal.ExitCode
		result.ErrorMessage = d.terminal.ErrorMessage
		if d.terminal.Summary != nil {
			result.Summary = d.terminal.Summary
		}
	default:
		result.Success = true
		result.ExitCode = 0
	}
	var cursorAdvanceErr *runner.TranscriptCursorAdvanceError
	if drainErr != nil && result.Success && !errors.As(drainErr, &cursorAdvanceErr) {
		result.Success = false
		result.ExitCode = -1
		result.ErrorMessage = drainErr.Error()
	}
	if !result.Success && strings.TrimSpace(result.ErrorMessage) == "" {
		result.ErrorMessage = strings.TrimSpace(d.errorOutput.String())
	}
	return result
}

// finish classifies the result with the codec, resolves the run result and
// summary, and closes the event sink.
func (d *durableRun) finish(ctx context.Context, result *runner.ExecuteResult) {
	r, in := d.r, d.in
	r.codec.PostClassify(in.state, result)
	terminalReason := "completed"
	if ctx.Err() == context.Canceled {
		terminalReason = "cancelled"
	} else if !result.Success {
		terminalReason = "failed"
	}
	result.Result = domain.ResolveRunResult(d.observedEvents, result.Success, result.ExitCode, terminalReason)
	result.Summary = domain.SummaryFromRunResult(
		result.Result,
		d.metrics.TurnsUsed,
		int64(runner.TotalTokens(d.metrics)),
		d.metrics.TokensInput,
		d.metrics.CostEstimateUSD,
	)

	if result.Success {
		runner.EmitStderrAsWarnOnSuccess(in.runID, in.sink, d.errorOutput.String())
	}

	if sid := in.state.SessionID(); sid != "" {
		result.SessionID = sid
	} else {
		d.sessionMu.Lock()
		result.SessionID = d.transcriptSessionID
		d.sessionMu.Unlock()
	}

	r.logRunnerExited(in.runID, in.sink, result, time.Since(in.startTime))

	closeEventSink(in.sink)
}

package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

type RecoverResult struct {
	Run        *domain.Run
	Recovered  bool
	Idempotent bool
	Message    string
}

func (r *Reconciler) RecoverInFlightRuns(ctx context.Context) error {
	running := domain.RunStatusRunning
	runs, err := r.runs.List(ctx, repository.RunListFilter{Status: &running})
	if err != nil {
		return err
	}
	recoveryLog := obs.Component("recovery")
	for _, run := range runs {
		// List() returns runs with a pruned column set that omits
		// ResolvedConfig — re-fetch with Get so recoveryParser has the
		// runner type. Skipping this re-fetch silently no-ops recovery
		// (recoveryParser bails when ResolvedConfig is nil), which was
		// the original 2026-04-29 production bug surfaced by the
		// restart-resume integration test.
		full, err := r.runs.Get(ctx, run.ID)
		if err != nil || full == nil {
			recoveryLog.Warn("run hydrate failed", obs.KeyRunID, run.ID.String(), obs.KeyError, errString(err))
			continue
		}
		if _, err := r.recoverRun(ctx, full, true); err != nil {
			recoveryLog.Warn("run recovery failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}

	pending := domain.RunStatusPending
	pendingRuns, err := r.runs.List(ctx, repository.RunListFilter{Status: &pending})
	if err != nil {
		return err
	}
	for _, run := range pendingRuns {
		if r.pendingRunRecovery == nil {
			recoveryLog.Warn("pending run cannot be re-enqueued; recovery owner is not wired", obs.KeyRunID, run.ID.String())
			continue
		}
		if _, err := r.pendingRunRecovery.ResumeRun(ctx, run.ID); err != nil {
			recoveryLog.Warn("pending run re-enqueue failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}
	return nil
}

// errString returns the error message or "<nil>" when err is nil. Used
// by structured logging sites where err is nullable but we still want a
// stable string representation.
func errString(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func (r *Reconciler) RecoverRun(ctx context.Context, runID uuid.UUID) (*RecoverResult, error) {
	run, err := r.runs.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundErrorWithID("Run", runID.String())
	}
	return r.recoverRun(ctx, run, false)
}

func (r *Reconciler) recoverRun(ctx context.Context, run *domain.Run, allowTail bool) (*RecoverResult, error) {
	// Interactive runs use a parallel recovery path: liveness is the web-console
	// session (GetSession), not a local process, and completion is driven by the
	// reattached transcript tailer's turn-boundary debounce, not a bare drain.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return r.recoverInteractiveRun(ctx, run, allowTail)
	}

	parser, transcriptPath, state, err := r.recoveryParser(run)
	if err != nil {
		return nil, err
	}
	if parser == nil {
		return &RecoverResult{Run: run, Idempotent: true, Message: "no transcript parser available"}, nil
	}

	terminal, err := r.drainTranscript(ctx, run, transcriptPath, state, parser)
	if err != nil {
		return nil, err
	}

	if terminal != nil {
		return r.finalizeRecoveredRun(ctx, run, terminal)
	}

	if r.isProcessAlive(ctx, run) {
		if allowTail {
			r.startTailer(run, transcriptPath, state, parser)
		}
		return &RecoverResult{Run: run, Recovered: true, Message: fmt.Sprintf("resumed run %s from transcript at offset %d", run.ID, run.TranscriptCursor)}, nil
	}

	return r.failRecoveredRun(ctx, run, "runner exited before terminal event")
}

func (r *Reconciler) finalizeRecoveredRun(ctx context.Context, run *domain.Run, terminal *runner.TranscriptTerminal) (*RecoverResult, error) {
	now := r.now()
	if terminal != nil && terminal.Success {
		run.Status = domain.RunStatusComplete
		run.ErrorMsg = ""
		run.EndedAt = &now
		run.UpdatedAt = now
		run.ExitCode = intPtr(terminal.ExitCode)
		result, summary, resultErr := r.buildRecoveredResult(ctx, run.ID, true, terminal.ExitCode, "completed")
		if resultErr == nil {
			if result.Selection.Status == domain.FinalOutputSelectionUnavailable && terminal.Summary != nil && terminal.Summary.Description != "" {
				evidence := domain.NewProviderMessageEvent(run.ID, "assistant", terminal.Summary.Description, domain.MessageEventData{
					ProviderOrigin:    "recovery",
					CompletionReason:  "terminal_summary",
					Terminal:          true,
					ProviderEventType: "transcript_terminal",
					RawEvidenceRef:    "transcript:terminal.summary",
				})
				result = domain.ResolveRunResult([]*domain.RunEvent{evidence}, true, terminal.ExitCode, "completed")
				summary = domain.SummaryFromRunResult(result, terminal.Summary.TurnsUsed, terminal.Summary.TokensUsed, terminal.Summary.ContextTokens, terminal.Summary.CostEstimate)
			}
			if r.structuredResults != nil && run.ResolvedConfig != nil {
				result.Structured = r.structuredResults.Resolve(ctx, run.ResolvedConfig.ResultSpec, result)
			}
			run.Result = result
			run.Summary = summary
		}
		if err := r.runs.Update(ctx, run); err != nil {
			return nil, err
		}
		if r.broadcaster != nil {
			r.broadcaster.BroadcastRunStatus(run)
		}
		return &RecoverResult{Run: run, Recovered: true, Message: "recovered terminal success from transcript"}, nil
	}

	if terminal != nil && !terminal.Success {
		run.Status = domain.RunStatusFailed
		run.ErrorMsg = terminal.ErrorMessage
		run.EndedAt = &now
		run.UpdatedAt = now
		run.ExitCode = intPtr(terminal.ExitCode)
		result, summary, resultErr := r.buildRecoveredResult(ctx, run.ID, false, terminal.ExitCode, "failed")
		if resultErr == nil {
			if r.structuredResults != nil && run.ResolvedConfig != nil {
				result.Structured = r.structuredResults.Resolve(ctx, run.ResolvedConfig.ResultSpec, result)
			}
			run.Result = result
			run.Summary = summary
		}
		if err := r.runs.Update(ctx, run); err != nil {
			return nil, err
		}
		if r.broadcaster != nil {
			r.broadcaster.BroadcastRunStatus(run)
		}
		return &RecoverResult{Run: run, Recovered: true, Message: "recovered terminal failure from transcript"}, nil
	}

	return r.failRecoveredRun(ctx, run, "runner exited before terminal event")
}

func (r *Reconciler) failRecoveredRun(ctx context.Context, run *domain.Run, message string) (*RecoverResult, error) {
	now := r.now()
	run.Status = domain.RunStatusFailed
	run.ErrorMsg = message
	run.EndedAt = &now
	run.UpdatedAt = now
	if err := r.runs.Update(ctx, run); err != nil {
		return nil, err
	}
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
	return &RecoverResult{Run: run, Recovered: true, Message: message}, nil
}

func (r *Reconciler) drainTranscript(ctx context.Context, run *domain.Run, transcriptPath string, state *runstate.Snapshot, parser runner.TranscriptParser) (*runner.TranscriptTerminal, error) {
	sink := r.recoveryEventSink(run.ID)
	transcriptParser := parser
	if factory, ok := parser.(runner.TranscriptParserFactory); ok {
		transcriptParser = factory.NewTranscriptParser()
	}
	_, terminal, err := runner.Consume(ctx, runner.ConsumeArgs{
		RunID:      run.ID,
		Transcript: transcriptPath,
		StartAt:    run.TranscriptCursor,
		ParseFn: func(runID uuid.UUID, line string) runner.TranscriptParseResult {
			return transcriptParser.ParseTranscriptLine(runID, line)
		},
		EventSink: sink,
		OnAdvance: func(cursor, lastSeq int64) error {
			if cursor > run.TranscriptCursor {
				run.TranscriptCursor = cursor
			}
			if lastSeq > run.TranscriptLastSeq {
				run.TranscriptLastSeq = lastSeq
			}
			if state != nil {
				s, err := runstate.Open(run.ID, runstate.OpenOptions{
					RunnerType: run.ResolvedConfig.RunnerType,
					WorkingDir: state.Meta.WorkingDir,
					StartedAt:  state.Meta.StartedAt,
				})
				if err == nil {
					_ = s.PersistCursor(run.TranscriptCursor, run.TranscriptLastSeq)
					_ = s.Close()
				}
			}
			return r.runs.Update(context.Background(), run)
		},
		OnSessionID: func(sessionID string) error {
			if sessionID == "" || run.SessionID == sessionID {
				return nil
			}
			run.SessionID = sessionID
			return r.runs.Update(context.Background(), run)
		},
	})
	return terminal, err
}

func (r *Reconciler) recoveryParser(run *domain.Run) (runner.TranscriptParser, string, *runstate.Snapshot, error) {
	if run.ResolvedConfig == nil {
		return nil, "", nil, nil
	}
	rr, err := r.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, "", nil, err
	}
	parser, ok := rr.(runner.TranscriptParser)
	if !ok {
		return nil, "", nil, nil
	}

	transcriptPath := run.TranscriptPath
	var state *runstate.Snapshot
	if transcriptPath != "" {
		if snapshot, err := runstate.Load(run.ID, ""); err == nil {
			state = snapshot
		}
		return parser, transcriptPath, state, nil
	}

	if run.ResolvedConfig.RunnerType == domain.RunnerTypeClaudeCode && run.SessionID != "" {
		fallback, err := findClaudeNativeTranscript(run.SessionID)
		if err == nil {
			return parser, fallback, state, nil
		}
	}

	return parser, "", state, fmt.Errorf("transcript missing for run %s", run.ID)
}

func (r *Reconciler) recoveryEventSink(runID uuid.UUID) runner.EventSink {
	if r.events != nil {
		if r.broadcaster != nil {
			return &broadcastingEventSink{store: r.events, runID: runID, broadcaster: r.broadcaster}
		}
		return &eventStoreAdapter{store: r.events, runID: runID}
	}
	return &noOpEventSink{}
}

func (r *Reconciler) startTailer(run *domain.Run, transcriptPath string, state *runstate.Snapshot, parser runner.TranscriptParser) {
	if transcriptPath == "" {
		return
	}

	r.recoveryMu.Lock()
	if cancel, ok := r.tailers[run.ID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.tailers[run.ID] = cancel
	r.recoveryMu.Unlock()

	go func() {
		defer func() {
			r.recoveryMu.Lock()
			delete(r.tailers, run.ID)
			r.recoveryMu.Unlock()
		}()
		// Log-only containment: the process may still be healthy, so the run
		// must not be failed here; the stale sweep is the backstop.
		defer obs.RecoverToFailure("recovery transcript tailer", nil)

		ticker := time.NewTicker(r.levers.Recovery.TranscriptTailInterval)
		defer ticker.Stop()
		for {
			if _, err := r.recoverRun(ctx, run, false); err == nil && !r.isProcessAlive(ctx, run) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			_, _ = transcriptPath, state
			_, _ = parser, run
		}
	}()
}

func (r *Reconciler) buildRecoveredResult(ctx context.Context, runID uuid.UUID, success bool, exitCode int, terminalReason string) (*domain.RunResult, *domain.RunSummary, error) {
	return resolvePersistedRunResult(ctx, r.events, runID, success, exitCode, terminalReason)
}

func findClaudeNativeTranscript(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	pattern := filepath.Join(home, ".claude", "projects", "*", sessionID+".jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("native claude transcript not found for session %s", sessionID)
	}
	best := matches[0]
	bestInfo, _ := os.Stat(best)
	for _, match := range matches[1:] {
		info, err := os.Stat(match)
		if err == nil && bestInfo != nil && info.ModTime().After(bestInfo.ModTime()) {
			best = match
			bestInfo = info
		}
	}
	return best, nil
}

func intPtr(v int) *int {
	return &v
}

func (r *Reconciler) cleanupRunStateDirs(ctx context.Context) {
	cutoff := r.now().Add(-time.Duration(r.levers.Storage.RunStateRetentionDays) * 24 * time.Hour)
	statuses := []domain.RunStatus{
		domain.RunStatusComplete,
		domain.RunStatusFailed,
		domain.RunStatusCancelled,
	}
	for _, status := range statuses {
		runs, err := r.runs.List(ctx, repository.RunListFilter{Status: &status})
		if err != nil {
			continue
		}
		for _, run := range runs {
			if run == nil {
				continue
			}
			terminalAt := run.UpdatedAt
			if run.EndedAt != nil && !run.EndedAt.IsZero() {
				terminalAt = *run.EndedAt
			}
			if terminalAt.After(cutoff) {
				continue
			}
			_ = os.RemoveAll(runstate.RunDir("", run.ID))
		}
	}
}

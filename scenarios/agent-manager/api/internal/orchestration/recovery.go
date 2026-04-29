package orchestration

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
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
	for _, run := range runs {
		if _, err := r.recoverRun(ctx, run, true); err != nil {
			log.Printf("[recovery] run %s: %v", run.ID, err)
		}
	}
	return nil
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
	now := time.Now()
	if terminal != nil && terminal.Success {
		run.Status = domain.RunStatusComplete
		run.ErrorMsg = ""
		run.EndedAt = &now
		run.UpdatedAt = now
		run.ExitCode = intPtr(terminal.ExitCode)
		summary, summaryErr := r.buildRecoveredSummary(ctx, run.ID)
		if summaryErr == nil && recoveredSummaryHasContent(summary) {
			run.Summary = summary
		} else if terminal.Summary != nil {
			run.Summary = terminal.Summary
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
	now := time.Now()
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
	_, terminal, err := runner.Consume(ctx, runner.ConsumeArgs{
		RunID:      run.ID,
		Transcript: transcriptPath,
		StartAt:    run.TranscriptCursor,
		ParseFn: func(runID uuid.UUID, line string) runner.TranscriptParseResult {
			return parser.ParseTranscriptLine(runID, line)
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

func (r *Reconciler) buildRecoveredSummary(ctx context.Context, runID uuid.UUID) (*domain.RunSummary, error) {
	if r.events == nil {
		return nil, nil
	}
	events, err := r.events.Get(ctx, runID, event.GetOptions{})
	if err != nil {
		return nil, err
	}
	var summary domain.RunSummary
	for _, evt := range events {
		switch data := evt.Data.(type) {
		case *domain.MessageEventData:
			if data.Role == "assistant" && data.Content != "" {
				summary.Description = data.Content
				summary.TurnsUsed++
			}
		case *domain.CostEventData:
			summary.TokensUsed = data.InputTokens + data.OutputTokens + data.CacheReadTokens + data.CacheCreationTokens
			summary.ContextTokens = data.InputTokens
			summary.CostEstimate = data.TotalCostUSD
		}
	}
	return &summary, nil
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

func recoveredSummaryHasContent(summary *domain.RunSummary) bool {
	if summary == nil {
		return false
	}
	return summary.Description != "" ||
		len(summary.FilesModified) > 0 ||
		len(summary.FilesCreated) > 0 ||
		len(summary.FilesDeleted) > 0 ||
		summary.TokensUsed > 0 ||
		summary.TurnsUsed > 0 ||
		summary.CostEstimate > 0 ||
		summary.ContextTokens > 0
}

func (r *Reconciler) cleanupRunStateDirs(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(r.levers.Storage.RunStateRetentionDays) * 24 * time.Hour)
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

// Package mocks provides controllable collaborators for orchestration tests.
package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	adapterrunner "agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

var (
	_ adapterrunner.TranscriptParser        = (*TranscriptReplayRunner)(nil)
	_ adapterrunner.TranscriptParserFactory = (*TranscriptReplayRunner)(nil)
)

// TranscriptReplayRunner is a runner.MockRunner with a compact transcript
// grammar for recovery tests:
//   - session:<id> records a session ID
//   - message:<text> emits an assistant message
//   - done:<text> emits a successful terminal summary
//   - fail:<text> emits a failed terminal state
//   - any other non-empty line emits an info log event
type TranscriptReplayRunner struct {
	*adapterrunner.MockRunner
	mu    sync.Mutex
	turns []ReplayTurn
	next  int
}

// ReplayTurn is one deterministic Execute or Continue response. A non-nil
// Gate models a turn that must wait for an external continue/approval signal;
// closing it releases the runner without a wall-clock sleep.
type ReplayTurn struct {
	Result *adapterrunner.ExecuteResult
	Err    error
	Gate   <-chan struct{}
	// Started receives one notification immediately before Gate is awaited.
	// It lets tests synchronize without wall-clock sleeps.
	Started chan<- struct{}
}

func NewTranscriptReplayRunner(rt domain.RunnerType) *TranscriptReplayRunner {
	r := &TranscriptReplayRunner{MockRunner: adapterrunner.NewMockRunner(rt)}
	r.ExecuteFunc = r.replayExecute
	r.ContinueFunc = r.replayContinue
	return r
}

// SetReplayTurns installs an ordered Execute/Continue script and resets its
// cursor. It is intended for a single test-owned runner before execution.
func (r *TranscriptReplayRunner) SetReplayTurns(turns ...ReplayTurn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.turns = append([]ReplayTurn(nil), turns...)
	r.next = 0
}

func (r *TranscriptReplayRunner) replayExecute(ctx context.Context, _ adapterrunner.ExecuteRequest) (*adapterrunner.ExecuteResult, error) {
	return r.nextTurn(ctx, "execute")
}

func (r *TranscriptReplayRunner) replayContinue(ctx context.Context, _ adapterrunner.ContinueRequest) (*adapterrunner.ExecuteResult, error) {
	return r.nextTurn(ctx, "continue")
}

func (r *TranscriptReplayRunner) nextTurn(ctx context.Context, operation string) (*adapterrunner.ExecuteResult, error) {
	r.mu.Lock()
	if r.next >= len(r.turns) {
		r.mu.Unlock()
		return nil, fmt.Errorf("transcript replay %s has no scripted turn remaining", operation)
	}
	turn := r.turns[r.next]
	r.next++
	r.mu.Unlock()
	if turn.Gate != nil {
		if turn.Started != nil {
			turn.Started <- struct{}{}
		}
		select {
		case <-turn.Gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if turn.Result == nil && turn.Err == nil {
		return &adapterrunner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	return turn.Result, turn.Err
}

// NewTranscriptParser returns a fresh parser for one logical transcript. The
// replay grammar is stateless, so the same value serves every consumption.
func (r *TranscriptReplayRunner) NewTranscriptParser() adapterrunner.TranscriptParser {
	return r
}

func (r *TranscriptReplayRunner) ParseTranscriptLine(runID uuid.UUID, line string) adapterrunner.TranscriptParseResult {
	line = strings.TrimSpace(line)
	switch {
	case line == "":
		return adapterrunner.TranscriptParseResult{}
	case strings.HasPrefix(line, "session:"):
		return adapterrunner.TranscriptParseResult{SessionID: strings.TrimPrefix(line, "session:")}
	case strings.HasPrefix(line, "message:"):
		return adapterrunner.TranscriptParseResult{
			Events: []*domain.RunEvent{
				domain.NewMessageEvent(runID, "assistant", strings.TrimPrefix(line, "message:")),
			},
		}
	case strings.HasPrefix(line, "done:"):
		return adapterrunner.TranscriptParseResult{
			Terminal: &adapterrunner.TranscriptTerminal{
				Success:  true,
				ExitCode: 0,
				Summary: &domain.RunSummary{
					Description: strings.TrimPrefix(line, "done:"),
				},
			},
		}
	case strings.HasPrefix(line, "fail:"):
		return adapterrunner.TranscriptParseResult{
			Terminal: &adapterrunner.TranscriptTerminal{
				Success:      false,
				ExitCode:     1,
				ErrorMessage: strings.TrimPrefix(line, "fail:"),
			},
		}
	default:
		return adapterrunner.TranscriptParseResult{
			Events: []*domain.RunEvent{
				domain.NewLogEvent(runID, "info", line),
			},
		}
	}
}

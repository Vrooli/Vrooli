package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Default poll interval for transcript consumption is sourced from
// config.DefaultLevers().Recovery.TranscriptPollInterval. Callers (the
// recovery tailer, codex/opencode resume drains) override via
// ConsumeArgs.PollInterval when they want a different cadence.
var transcriptPollInterval = config.DefaultLevers().Recovery.TranscriptPollInterval

type ConsumeArgs struct {
	RunID      uuid.UUID
	Transcript string
	StartAt    int64
	// EndAt bounds a non-live inspection to an immutable observed file extent.
	// Zero preserves ordinary live/replay behavior.
	EndAt        int64
	Live         bool
	ParseFn      func(uuid.UUID, string) TranscriptParseResult
	EventSink    EventSink
	OnAdvance    func(cursor, lastSeq int64) error
	OnEvents     func(events []*domain.RunEvent)
	OnSessionID  func(sessionID string) error
	OnGoalStatus func(marker GoalMarker) error
	OnLabel      func(label string, source domain.RunLabelSource) error
	OnTerminal   func(terminal *TranscriptTerminal) error
	PollInterval time.Duration
}

// TranscriptCursorAdvanceError identifies a failure to persist replay
// progress after evidence was already consumed. The caller must report it so
// recovery remains observable, but a successfully exited agent process does
// not become a failed run solely because its resumability cursor could not be
// saved.
type TranscriptCursorAdvanceError struct{ Err error }

func (e *TranscriptCursorAdvanceError) Error() string { return e.Err.Error() }
func (e *TranscriptCursorAdvanceError) Unwrap() error { return e.Err }

func Consume(ctx context.Context, args ConsumeArgs) (int64, *TranscriptTerminal, error) {
	if args.ParseFn == nil {
		return args.StartAt, nil, fmt.Errorf("parse function is required")
	}
	file, err := os.Open(args.Transcript)
	if err != nil {
		return args.StartAt, nil, err
	}
	defer file.Close()

	if args.StartAt > 0 {
		if _, err := file.Seek(args.StartAt, io.SeekStart); err != nil {
			return args.StartAt, nil, err
		}
	}

	poll := args.PollInterval
	if poll <= 0 {
		poll = transcriptPollInterval
	}

	var source io.Reader = file
	if args.EndAt > 0 {
		if args.Live || args.EndAt < args.StartAt {
			return args.StartAt, nil, fmt.Errorf("bounded inspection requires a non-live valid extent")
		}
		source = io.LimitReader(file, args.EndAt-args.StartAt)
	}
	reader := bufio.NewReader(source)
	cursor := args.StartAt
	var terminal *TranscriptTerminal

	for {
		select {
		case <-ctx.Done():
			return cursor, terminal, ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if !args.Live {
					return cursor, terminal, nil
				}
				// A non-empty EOF fragment is a half-written line whose
				// terminating newline has not been flushed yet. Rewind to
				// the last complete-line boundary and rebuild the reader so
				// the fragment is re-read once completed, rather than
				// consumed-and-lost (bufio advances past it) and later
				// emitted as a corrupt partial JSON line.
				if len(line) > 0 {
					if _, serr := file.Seek(cursor, io.SeekStart); serr != nil {
						return cursor, terminal, serr
					}
					reader.Reset(file)
				}
				time.Sleep(poll)
				continue
			}
			return cursor, terminal, err
		}

		cursor += int64(len(line))
		result := args.ParseFn(args.RunID, line)
		if result.Err != nil {
			return cursor, terminal, result.Err
		}
		if result.UserTurnStarted {
			// Boundary evidence does not require retaining or emitting private
			// user content. Invalidate before callbacks can return an error.
			terminal = nil
		}
		if result.SessionID != "" && args.OnSessionID != nil {
			if err := args.OnSessionID(result.SessionID); err != nil {
				return cursor, terminal, err
			}
		}
		if result.Label != "" && args.OnLabel != nil {
			if err := args.OnLabel(result.Label, result.LabelSource); err != nil {
				return cursor, terminal, err
			}
		}
		if !result.Timestamp.IsZero() {
			for _, event := range result.Events {
				if event != nil {
					event.Timestamp = result.Timestamp
				}
			}
		}
		if len(result.Events) > 0 && args.OnEvents != nil {
			args.OnEvents(result.Events)
		}
		for _, event := range result.Events {
			if event != nil {
				if message, ok := event.Data.(*domain.MessageEventData); ok && message.Role == "user" {
					terminal = nil // a later user turn supersedes the prior boundary
				}
			}
		}
		if result.Goal != nil && (result.Goal.Status == GoalStatusActive || result.Goal.Status == GoalStatusPaused) {
			terminal = nil
		}
		// Provider turn completion follows a native goal terminal. It must not
		// downgrade the accepted goal verdict to an ordinary idle boundary.
		if result.Terminal != nil && !(result.Terminal.Success && terminal != nil && strings.HasPrefix(terminal.TerminalReason, "goal_")) {
			terminal = result.Terminal
			if args.OnTerminal != nil {
				if err := args.OnTerminal(result.Terminal); err != nil {
					return cursor, terminal, err
				}
			}
		}
		if result.Goal != nil {
			if args.OnGoalStatus != nil {
				if err := args.OnGoalStatus(*result.Goal); err != nil {
					return cursor, terminal, err
				}
			}
		}
		if result.Goal != nil && result.Goal.Status != GoalStatusActive && result.Goal.Status != GoalStatusPaused {
			// A non-active goal status is a deliberate harness terminal. A
			// complete/blocked goal is a verdict; a usage/budget limit is an
			// involuntary interruption and must never be reported as success.
			terminal = &TranscriptTerminal{
				Success:        true,
				ExitCode:       0,
				TerminalReason: "goal_" + string(result.Goal.Status),
			}
			switch result.Goal.Status {
			case GoalStatusComplete:
				terminal.TerminalClass = domain.RunTerminalClassVerdict
				terminal.StopReason = domain.RunStopReasonComplete
			case GoalStatusBlocked:
				terminal.TerminalClass = domain.RunTerminalClassVerdict
				terminal.StopReason = domain.RunStopReasonBlocked
			case GoalStatusUsageLimited, GoalStatusBudgetLimited:
				terminal.Success = false
				terminal.TerminalClass = domain.RunTerminalClassInterruption
				terminal.StopReason = domain.RunStopReasonUsageWindow
			default:
				terminal.Success = false
				terminal.TerminalClass = domain.RunTerminalClassInterruption
				terminal.StopReason = domain.RunStopReasonCrash
			}
			if args.OnTerminal != nil {
				if err := args.OnTerminal(terminal); err != nil {
					return cursor, terminal, err
				}
			}
		}

		lastSeq := int64(0)
		for _, evt := range result.Events {
			if evt == nil || args.EventSink == nil {
				continue
			}
			if err := args.EventSink.Emit(evt); err != nil {
				return cursor, terminal, err
			}
			lastSeq = evt.Sequence
			if seqSink, ok := args.EventSink.(SequencedEventSink); ok {
				lastSeq = seqSink.LastSequence()
			}
		}

		if args.OnAdvance != nil {
			if err := args.OnAdvance(cursor, lastSeq); err != nil {
				return cursor, terminal, &TranscriptCursorAdvanceError{Err: err}
			}
		}
	}
}

// TerminalSummaryFromMessage builds a RunSummary from the captured last
// assistant message and rolling metrics. Used on the success path of the
// durable-transcript flow when the codec did not surface an explicit
// terminal summary.
func TerminalSummaryFromMessage(message string, metrics ExecutionMetrics) *domain.RunSummary {
	if message == "" && metrics == (ExecutionMetrics{}) {
		return nil
	}
	return &domain.RunSummary{
		Description:   message,
		TurnsUsed:     metrics.TurnsUsed,
		TokensUsed:    TotalTokens(metrics),
		CostEstimate:  metrics.CostEstimateUSD,
		ContextTokens: metrics.TokensInput,
	}
}

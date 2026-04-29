package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
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
	RunID        uuid.UUID
	Transcript   string
	StartAt      int64
	Live         bool
	ParseFn      func(uuid.UUID, string) TranscriptParseResult
	EventSink    EventSink
	OnAdvance    func(cursor, lastSeq int64) error
	OnEvents     func(events []*domain.RunEvent)
	OnSessionID  func(sessionID string) error
	OnTerminal   func(terminal *TranscriptTerminal) error
	PollInterval time.Duration
}

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

	reader := bufio.NewReader(file)
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
		if result.SessionID != "" && args.OnSessionID != nil {
			if err := args.OnSessionID(result.SessionID); err != nil {
				return cursor, terminal, err
			}
		}
		if len(result.Events) > 0 && args.OnEvents != nil {
			args.OnEvents(result.Events)
		}
		if result.Terminal != nil {
			terminal = result.Terminal
			if args.OnTerminal != nil {
				if err := args.OnTerminal(result.Terminal); err != nil {
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
				return cursor, terminal, err
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

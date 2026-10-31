package runner

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	// Snapshot supplies an already bounded immutable source for a non-live
	// read beginning at zero. Transcript remains its logical path.
	Snapshot io.Reader
	StartAt  int64
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
	// StopBeforeTranscriptResult can stop consumption before a boundary record
	// is emitted or persisted. It is used by interactive goal draining to keep
	// the next user turn out of the completed run's result.
	StopBeforeTranscriptResult func(TranscriptParseResult) bool
	// StopAfterTranscriptResult can stop consumption after the current record
	// and cursor have been durably advanced. It is used to end a bounded drain
	// at the provider turn boundary.
	StopAfterTranscriptResult func(TranscriptParseResult) bool
	PollInterval              time.Duration
}

// TranscriptEventID is the stable identity of the index-th event parsed from
// line, found at the absolute file offset in the transcript at source. The
// file path separates invocations that start a new transcript, the absolute
// offset separates appended continuations of one file, and the line hash keeps
// a rewritten file's different content from reusing a stored identity.
func TranscriptEventID(runID uuid.UUID, source string, offset int64, index int, line string) uuid.UUID {
	sum := sha256.Sum256([]byte(line))
	return uuid.NewSHA1(runID, []byte(fmt.Sprintf("transcript:%s:offset:%d:event:%d:%x", source, offset, index, sum[:8])))
}

// AssignTranscriptEventIDs gives the events one line produced their stable
// identities and repoints evidence events, created before the ids changed, at
// the final id of the event they vouch for.
func AssignTranscriptEventIDs(runID uuid.UUID, source string, offset int64, line string, events []*domain.RunEvent) {
	renamed := map[string]string{}
	for index, event := range events {
		if event == nil {
			continue
		}
		previous := event.ID.String()
		event.ID = TranscriptEventID(runID, source, offset, index, line)
		renamed[previous] = event.ID.String()
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		if message, ok := event.Data.(*domain.MessageEventData); ok && message.EvidenceForEventID != "" {
			if id, ok := renamed[message.EvidenceForEventID]; ok {
				message.EvidenceForEventID = id
			}
		}
	}
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
	var source io.Reader
	var file *os.File
	if args.Snapshot != nil {
		if args.Live || args.StartAt != 0 {
			return args.StartAt, nil, fmt.Errorf("snapshot requires a non-live read starting at zero")
		}
		source = args.Snapshot
	} else {
		var err error
		file, err = os.Open(args.Transcript)
		if err != nil {
			return args.StartAt, nil, err
		}
		defer file.Close()
		if args.StartAt > 0 {
			if _, err := file.Seek(args.StartAt, io.SeekStart); err != nil {
				return args.StartAt, nil, err
			}
		}
		source = file
	}

	poll := args.PollInterval
	if poll <= 0 {
		poll = transcriptPollInterval
	}

	if args.EndAt > 0 {
		if args.Live || args.EndAt < args.StartAt {
			return args.StartAt, nil, fmt.Errorf("bounded inspection requires a non-live valid extent")
		}
		source = io.LimitReader(source, args.EndAt-args.StartAt)
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

		occurrenceOffset := cursor
		cursor += int64(len(line))
		result := args.ParseFn(args.RunID, line)
		if args.StopBeforeTranscriptResult != nil && args.StopBeforeTranscriptResult(result) {
			return occurrenceOffset, terminal, nil
		}
		// Occurrence identity includes the real source location, so re-reading
		// a range (a restart that lost its cursor, a replaced file) replays the
		// same rows instead of appending a second copy, while identical records
		// at distinct offsets stay distinct evidence.
		source, err := filepath.Abs(args.Transcript)
		if err != nil {
			return cursor, terminal, err
		}
		AssignTranscriptEventIDs(args.RunID, source, occurrenceOffset, line, result.Events)
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
		if args.StopAfterTranscriptResult != nil && args.StopAfterTranscriptResult(result) {
			return cursor, terminal, nil
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
		TokensUsed:    int64(TotalTokens(metrics)),
		CostEstimate:  metrics.CostEstimateUSD,
		ContextTokens: metrics.TokensInput,
	}
}

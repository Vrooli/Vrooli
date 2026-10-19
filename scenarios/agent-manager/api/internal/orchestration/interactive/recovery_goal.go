// Responsibility: inspect retained transcript goals without replaying tool effects.
package interactive

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// RetainedGoalEvidence couples the accepted native goal verdict with the exact
// marker that justified it. Recovery can persist the same observation without
// replaying transcript messages or tool effects.
type RetainedGoalEvidence struct {
	Terminal *runner.TranscriptTerminal
	Marker   runner.GoalMarker
	At       time.Time
}

// RetainedContinuationEvidence is a terminal observed only in the exact
// transcript segment admitted for one interactive continuation.
type RetainedContinuationEvidence struct {
	Terminal *runner.TranscriptTerminal
	Cursor   int64
}

// InspectRetainedContinuation repairs the crash window after a successful
// continuation terminal and cursor were persisted but before Finalize. It
// replays only the owner-recorded invocation segment, with the same parser,
// bounded file extent, and idle debounce as live completion. Any later turn
// boundary supersedes an earlier terminal; unknown or changing evidence is
// never treated as completion.
func (c *Coordinator) InspectRetainedContinuation(ctx context.Context, run *domain.Run) (*RetainedContinuationEvidence, error) {
	if run == nil || run.Status != domain.RunStatusRunning || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive ||
		run.ResolvedConfig == nil || run.SessionID == "" || run.TranscriptPath == "" || run.InteractiveInvocationStartedAt == nil || c.deps.Tailer == nil {
		return nil, nil
	}
	policy := run.ResolvedConfig.InteractiveContinuation
	if policy == nil || !policy.CompleteAfterFirstSuccessfulTurnWithoutGoalMarker ||
		!policy.InvocationStartedAt.Equal(*run.InteractiveInvocationStartedAt) || policy.TranscriptStartCursor == nil || *policy.TranscriptStartCursor < 0 {
		return nil, nil
	}
	start := *policy.TranscriptStartCursor
	if run.TranscriptCursor < start {
		return nil, nil
	}
	settle := c.debounce
	if settle <= 0 {
		settle = defaultDebounceWindow
	}
	ctx, cancel := context.WithTimeout(ctx, settle+4*time.Second)
	defer cancel()
	before, err := os.Stat(run.TranscriptPath)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() < start || before.Size() > 16<<20 {
		return nil, fmt.Errorf("retained continuation extent does not match its invocation cursor")
	}
	if before.Size() == start {
		return nil, nil
	}
	if before.Size()-start > 16<<20 {
		return nil, fmt.Errorf("retained continuation segment exceeds bounded replay")
	}
	// A cursor behind EOF means the normal reattached tailer still owns unseen
	// bytes. Only recover a terminal after its cursor was already durably saved.
	if run.TranscriptCursor != before.Size() {
		return nil, nil
	}
	timer := time.NewTimer(settle)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	afterSettle, err := os.Stat(run.TranscriptPath)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, afterSettle) || before.Size() != afterSettle.Size() || !before.ModTime().Equal(afterSettle.ModTime()) || run.TranscriptCursor != afterSettle.Size() {
		return nil, fmt.Errorf("retained continuation changed during completion debounce")
	}
	parser, err := c.deps.Tailer.resolveParser(run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, err
	}
	if setter, ok := parser.(interface {
		SetGoalContext(uuid.UUID, string, string)
	}); ok {
		setter.SetGoalContext(run.ID, run.SessionID, run.ResolvedConfig.Until)
	}
	// The invocation segment often starts after the original session header.
	// Prove the transcript's native session binding from its bounded prefix
	// before trusting a terminal in that segment; a path or persisted SessionID
	// alone is not transcript evidence.
	sessionMatched := false
	prefixCursor, _, err := runner.Consume(ctx, runner.ConsumeArgs{
		RunID: run.ID, Transcript: run.TranscriptPath, EndAt: before.Size(),
		ParseFn: func(id uuid.UUID, line string) runner.TranscriptParseResult {
			result := parser.ParseTranscriptLine(id, line)
			if result.SessionID != "" {
				if result.SessionID != run.SessionID {
					result.Err = fmt.Errorf("retained continuation session does not match owner run")
				} else {
					sessionMatched = true
				}
			}
			return result
		},
		StopAfterTranscriptResult: func(result runner.TranscriptParseResult) bool { return result.SessionID != "" },
	})
	if err != nil {
		return nil, err
	}
	if !sessionMatched || prefixCursor > before.Size() {
		return nil, nil
	}
	var latest *runner.TranscriptTerminal
	var sawTurnBoundary, sawGoalMarker bool
	var sessionMismatch bool
	cursor, _, err := runner.Consume(ctx, runner.ConsumeArgs{
		RunID: run.ID, Transcript: run.TranscriptPath, StartAt: start, EndAt: before.Size(),
		ParseFn: func(id uuid.UUID, line string) runner.TranscriptParseResult {
			result := parser.ParseTranscriptLine(id, line)
			if result.SessionID != "" && result.SessionID != run.SessionID {
				sessionMismatch = true
				result.Err = fmt.Errorf("retained continuation session does not match owner run")
				return result
			}
			if result.UserTurnStarted || result.ProviderTurnStarted {
				sawTurnBoundary = true
				latest = nil
			}
			for _, event := range result.Events {
				if event != nil {
					if message, ok := event.Data.(*domain.MessageEventData); ok && message.Role == "user" {
						sawTurnBoundary = true
						latest = nil
					}
				}
			}
			if result.Goal != nil {
				sawGoalMarker = true
				latest = nil
			}
			if result.Terminal != nil {
				if strings.HasPrefix(result.Terminal.TerminalReason, "goal_") {
					sawGoalMarker = true
					latest = nil
				} else {
					latest = result.Terminal
				}
			}
			return result
		},
	})
	if err != nil {
		return nil, err
	}
	afterRead, err := os.Stat(run.TranscriptPath)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, afterRead) || before.Size() != afterRead.Size() || !before.ModTime().Equal(afterRead.ModTime()) || cursor != before.Size() || run.TranscriptCursor != before.Size() {
		return nil, fmt.Errorf("retained continuation changed or has an incomplete tail; no finalization")
	}
	if !sessionMatched || sessionMismatch || !sawTurnBoundary || sawGoalMarker || latest == nil {
		return nil, nil
	}
	return &RetainedContinuationEvidence{Terminal: latest, Cursor: cursor}, nil
}

// InspectRetainedGoal repairs the cursor-before-finalization crash gap without
// replaying events, costs, messages or tool effects. It uses the canonical codec,
// not a second parser or a textual completion heuristic. Unknown is not failure.
func (c *Coordinator) InspectRetainedGoal(ctx context.Context, run *domain.Run) (*RetainedGoalEvidence, error) {
	if run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.Until) == "" || run.SessionID == "" || run.TranscriptPath == "" || run.StartedAt == nil || c.deps.Tailer == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	before, err := os.Stat(run.TranscriptPath)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 16<<20 {
		return nil, fmt.Errorf("retained goal evidence exceeds bounded regular-file replay")
	}
	parser, err := c.deps.Tailer.resolveParser(run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, err
	}
	// Retained replay starts after the live tailer's original parser context was
	// discarded. Seed the same run/session/objective pins before consuming so a
	// correlated get_goal read is validated against this run, not an empty parser.
	if setter, ok := parser.(interface {
		SetGoalContext(uuid.UUID, string, string)
	}); ok {
		setter.SetGoalContext(run.ID, run.SessionID, run.ResolvedConfig.Until)
	}
	// Turn-boundary metadata works with the parser's default privacy policy.
	// No user message retention is needed for this read-only goal inspection.
	attemptStart := run.StartedAt
	if run.InteractiveInvocationStartedAt != nil {
		attemptStart = run.InteractiveInvocationStartedAt
	}
	var lastGoal *runner.GoalMarker
	var goalTime time.Time
	sessionMatched := false
	cursor, terminal, err := runner.Consume(ctx, runner.ConsumeArgs{
		RunID: run.ID, Transcript: run.TranscriptPath, EndAt: before.Size(),
		ParseFn: func(id uuid.UUID, line string) runner.TranscriptParseResult {
			result := parser.ParseTranscriptLine(id, line)
			if result.SessionID != "" {
				if result.SessionID != run.SessionID {
					result.Err = fmt.Errorf("retained goal session does not match owner run")
				} else {
					sessionMatched = true
				}
			}
			if result.UserTurnStarted {
				lastGoal = nil
			}
			for _, event := range result.Events {
				if event != nil {
					if message, ok := event.Data.(*domain.MessageEventData); ok && message.Role == "user" {
						lastGoal = nil
					}
				}
			}
			if result.Goal != nil {
				copy := *result.Goal
				lastGoal = &copy
				goalTime = result.Timestamp
			}
			return result
		},
	})
	if err != nil {
		return nil, err
	}
	after, err := os.Stat(run.TranscriptPath)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || cursor != before.Size() {
		return nil, fmt.Errorf("retained goal transcript changed or has an incomplete tail; no finalization")
	}
	if !sessionMatched || lastGoal == nil || terminal == nil || !strings.HasPrefix(terminal.TerminalReason, "goal_") || terminal.TerminalReason != "goal_"+string(lastGoal.Status) || goalTime.IsZero() || goalTime.Before(*attemptStart) || !runner.NativeGoalObjectiveMatches(lastGoal.Objective, run.ResolvedConfig.Until) {
		return nil, nil
	}
	return &RetainedGoalEvidence{Terminal: terminal, Marker: *lastGoal, At: goalTime}, nil
}

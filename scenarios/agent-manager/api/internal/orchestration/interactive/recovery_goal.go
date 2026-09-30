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

// InspectRetainedGoal repairs the cursor-before-finalization crash gap without
// replaying events, costs, messages or tool effects. It uses the canonical codec,
// not a second parser or a textual completion heuristic. Unknown is not failure.
func (c *Coordinator) InspectRetainedGoal(ctx context.Context, run *domain.Run) (*runner.TranscriptTerminal, error) {
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
	if !sessionMatched || lastGoal == nil || terminal == nil || !strings.HasPrefix(terminal.TerminalReason, "goal_") || goalTime.IsZero() || goalTime.Before(*attemptStart) || strings.TrimSpace(lastGoal.Objective) != strings.TrimSpace(run.ResolvedConfig.Until) {
		return nil, nil
	}
	return terminal, nil
}

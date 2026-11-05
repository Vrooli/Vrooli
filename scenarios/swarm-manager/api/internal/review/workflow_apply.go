package review

import (
	"context"
	"fmt"
)

// ApplyWorkflowRound collects and applies one terminal independent-review
// result. It is the explicit local mutation boundary for declared reviews.
func (s *Service) ApplyWorkflowRound(ctx context.Context, kind, name string, roundNum int) (Round, bool, error) {
	itemDir := s.resolveItemDir(kind, name)
	round, err := readRound(itemDir, roundNum)
	if err != nil {
		return Round{}, false, fmt.Errorf("load review round: %w", err)
	}
	if round == nil {
		return Round{}, false, fmt.Errorf("review round %d does not exist", roundNum)
	}
	if s.transitionRunner == nil {
		return Round{}, false, fmt.Errorf("transition runner is not configured")
	}
	correlation, err := s.transitionRunner.FindCorrelation("work.review", reviewSubject(kind, name, roundNum))
	if err != nil {
		return Round{}, false, fmt.Errorf("find review transition correlation: %w", err)
	}
	alreadyApplied := round.Status != RoundStatusGathering
	if _, err := s.transitionRunner.ApplyExecution(ctx, correlation.ExecutionID); err != nil {
		return Round{}, false, err
	}
	applied, err := readRound(itemDir, roundNum)
	if err != nil || applied == nil {
		return Round{}, false, fmt.Errorf("reload applied review round: %w", err)
	}
	return *applied, alreadyApplied, nil
}

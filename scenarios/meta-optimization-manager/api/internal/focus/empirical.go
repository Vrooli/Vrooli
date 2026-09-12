package focus

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"meta-optimization-manager/internal/trials"
)

// TrialHistoryReader is the consumer-owned read seam for recorded trial
// evidence. It intentionally exposes only the repository's read operation;
// deriving focus gaps can never dispatch a trial.
type TrialHistoryReader interface {
	Runs(ctx context.Context, filter trials.RunFilter, limit int, desc bool) ([]trials.TrialRun, error)
}

type empiricalGapSource struct {
	reader TrialHistoryReader
}

// NewEmpiricalGapSource constructs the trials-backed empirical source.
func NewEmpiricalGapSource(reader TrialHistoryReader) GapSource {
	return &empiricalGapSource{reader: reader}
}

var _ GapSource = (*empiricalGapSource)(nil)

func (*empiricalGapSource) Axis() Axis { return AxisEmpirical }

// trialMinimumObservations and trialMinimumFailures prevent one transient
// verdict from becoming a readiness-board gap. Revisit these thresholds when
// the fixed trial corpus has enough historical volume to distinguish a
// transient from a sustained task-level problem without the conservative floor.
const (
	trialMinimumObservations = 2
	trialMinimumFailures     = 2
	trialFailureRate         = 0.5
)

func (s *empiricalGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s.reader == nil {
		return nil, fmt.Errorf("trials history reader is not configured")
	}
	runs, err := s.reader.Runs(ctx, trials.RunFilter{}, 0, true)
	if err != nil {
		return nil, fmt.Errorf("read trial history: %w", err)
	}
	byTask := make(map[string][]trials.TrialRun)
	for _, run := range runs {
		taskID := strings.TrimSpace(run.TaskID)
		if taskID == "" {
			continue
		}
		byTask[taskID] = append(byTask[taskID], run)
	}
	taskIDs := make([]string, 0, len(byTask))
	for taskID := range byTask {
		taskIDs = append(taskIDs, taskID)
	}
	sort.Strings(taskIDs)

	out := make([]Gap, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		taskRuns := byTask[taskID]
		sort.SliceStable(taskRuns, func(i, j int) bool {
			if !taskRuns[i].At.Equal(taskRuns[j].At) {
				return taskRuns[i].At.After(taskRuns[j].At)
			}
			return taskRuns[i].ID > taskRuns[j].ID
		})
		failures := 0
		for _, run := range taskRuns {
			if run.Verdict == trials.VerdictFail || run.Verdict == trials.VerdictError {
				failures++
			}
		}
		if len(taskRuns) < trialMinimumObservations || failures < trialMinimumFailures || float64(failures)/float64(len(taskRuns)) < trialFailureRate {
			continue
		}
		newestRunID := strings.TrimSpace(taskRuns[0].ID)
		out = append(out, Gap{
			ID:              "empirical/trials/" + taskID,
			Axis:            AxisEmpirical,
			Title:           fmt.Sprintf("trial task %s has sustained failures", taskID),
			Global:          true,
			EvidenceSource:  "trials",
			EvidenceLocator: fmt.Sprintf("trial-task:%s/run:%s", taskID, newestRunID),
			Recurrence:      failures,
			Notes:           []string{fmt.Sprintf("%d of %d recorded verdicts failed or errored (%.0f%%)", failures, len(taskRuns), float64(failures)/float64(len(taskRuns))*100)},
		})
	}
	return out, nil
}

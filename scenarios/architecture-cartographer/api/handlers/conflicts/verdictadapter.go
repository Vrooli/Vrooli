// verdictadapter.go bridges the signals service to the conflicts
// VerdictProvider seam used by the mislocated_file detector.
//
// The conflicts package intentionally declares a narrow local Verdict
// type so detectors don't depend on the signals package's full
// interface; this adapter is the one production caller that translates
// signals.Verdict → conflicts.Verdict. It is batch-only by design (see
// the VerdictProvider docstring): the snapshot, domain map, and
// GraphContext are expensive to build, and a per-chunk caller made
// DetectConflicts O(F²×D×S).

package conflicts

import (
	"context"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// signalsVerdictAdapter satisfies conflicts.VerdictProvider by calling
// signals.ScoreBatch (single snapshot/domain-map/context build,
// concurrent per-chunk aggregation).
type signalsVerdictAdapter struct {
	signals signals.Service
}

func NewSignalsVerdictAdapter(s signals.Service) conflicts.VerdictProvider {
	return signalsVerdictAdapter{signals: s}
}

func (a signalsVerdictAdapter) VerdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk) ([]conflicts.Verdict, error) {
	return a.verdictsFor(ctx, scenario, chunks, false)
}

func (a signalsVerdictAdapter) ContentVerdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk) ([]conflicts.Verdict, error) {
	return a.verdictsFor(ctx, scenario, chunks, true)
}

func (a signalsVerdictAdapter) verdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk, contentOnly bool) ([]conflicts.Verdict, error) {
	if len(chunks) == 0 {
		return nil, nil
	}
	input := signals.ScoreBatchInput{
		Scenario: scenario,
		Chunks:   chunks,
	}
	scoreBatch := a.signals.ScoreBatch
	if contentOnly {
		scoreBatch = a.signals.ContentScoreBatch
	}
	verdicts, err := scoreBatch(ctx, input)
	if err != nil {
		return nil, err
	}
	out := make([]conflicts.Verdict, len(verdicts))
	for i, v := range verdicts {
		out[i] = conflicts.Verdict{
			ChunkID:        v.ChunkID,
			ChunkPath:      v.ChunkPath,
			Tier:           string(v.Tier),
			TopDomain:      v.TopDomain,
			TopValue:       v.TopValue,
			RunnerUpDomain: v.RunnerUpDomain,
			RunnerUpValue:  v.RunnerUpValue,
			Tied:           v.Tied,
			AllAbstained:   len(v.Scores) == 0 && len(v.Abstentions) > 0,
		}
	}
	return out, nil
}

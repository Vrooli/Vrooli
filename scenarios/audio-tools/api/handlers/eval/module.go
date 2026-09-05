// Package eval contains only the protobuf transport mapping for evaluation
// reports. Replay execution belongs to internal/eval.
package eval

import (
	"context"

	inteval "audio-tools/internal/eval"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

// Deps is a compatibility alias for callers that historically reached the
// runner through this transport package.
type Deps = inteval.RunnerDeps

func RunReport(ctx context.Context, deps Deps, clipIDs []string, strategies []*evalv1.EvalStrategy, repeats, chunkMs int32) (inteval.EvalReport, error) {
	return inteval.RunReportWithOptions(ctx, deps, clipIDs, strategies, repeats, chunkMs, inteval.EvalOptions{})
}

func RunReportWithOptions(ctx context.Context, deps Deps, clipIDs []string, strategies []*evalv1.EvalStrategy, repeats, chunkMs int32, opts inteval.EvalOptions) (inteval.EvalReport, error) {
	return inteval.RunReportWithOptions(ctx, deps, clipIDs, strategies, repeats, chunkMs, opts)
}

func RunReportForClips(ctx context.Context, deps Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, repeats, chunkMs int32) (inteval.EvalReport, error) {
	return inteval.RunReportForClipsWithOptions(ctx, deps, clips, strategies, repeats, chunkMs, inteval.EvalOptions{})
}

func RunReportForClipsWithOptions(ctx context.Context, deps Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, repeats, chunkMs int32, opts inteval.EvalOptions) (inteval.EvalReport, error) {
	return inteval.RunReportForClipsWithOptions(ctx, deps, clips, strategies, repeats, chunkMs, opts)
}

func RunReportForCells(ctx context.Context, deps Deps, clips []inteval.Clip, cells []*experimentv1.EvaluationCell, chunkMs int32, opts inteval.EvalOptions) (inteval.EvalReport, error) {
	return inteval.RunReportForCells(ctx, deps, clips, cells, chunkMs, opts)
}

func RunReportCellsWithOptions(ctx context.Context, deps Deps, clipIDs []string, cells []*experimentv1.EvaluationCell, chunkMs int32, opts inteval.EvalOptions) (inteval.EvalReport, error) {
	return inteval.RunReportCellsWithOptions(ctx, deps, clipIDs, cells, chunkMs, opts)
}

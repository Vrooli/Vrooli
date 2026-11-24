//go:build audio_lab

package report

import (
	"context"
	"fmt"
	"strings"

	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	exprecipe "audio-tools/internal/experiment/recipe"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func newExperimentProgress(total int64, emit func(int, string)) *experimentProgress {
	if total <= 0 || emit == nil {
		return nil
	}
	return &experimentProgress{total: total, emit: emit}
}

func (p *experimentProgress) options(scope string, opts inteval.EvalOptions) inteval.EvalOptions {
	opts.ProgressScope = scope
	opts.Progress = p.step
	return opts
}

func (p *experimentProgress) step(update inteval.EvalProgress) {
	done := p.completed.Add(1)
	if done > p.total {
		done = p.total
	}
	progress := 5 + int(done*85/p.total)
	if progress > 94 {
		progress = 94
	}
	p.emit(progress, experimentProgressMessage(done, p.total, update))
}

func experimentProgressMessage(done, total int64, update inteval.EvalProgress) string {
	scope := strings.TrimSpace(update.Scope)
	if scope == "" {
		scope = "condition default"
	}
	phase := update.Phase
	if phase == "" {
		phase = "eval"
	}
	strategy := update.Strategy
	if strategy == "" {
		strategy = "strategy"
	}
	clip := update.ClipID
	if clip == "" {
		clip = "clip"
	}
	message := fmt.Sprintf("running %d/%d %s %s: strategy %s %d/%d, clip %s %d/%d",
		done, total, scope, phase, strategy, update.StrategyIndex, update.StrategyTotal, clip, update.ClipIndex, update.ClipTotal)
	if update.RepeatTotal > 0 {
		message += fmt.Sprintf(", repeat %d/%d", update.RepeatIndex, update.RepeatTotal)
	}
	return message
}

func experimentWorkUnits(clips []inteval.Clip, augGroups []exprecipe.ConditionGroup, strategies []*evalv1.EvalStrategy, realtimeRepeats int32, speakerConditions []speakerEvalCondition, speakerEnabled bool) int64 {
	if len(augGroups) == 0 {
		return evalWorkUnits(len(clips), strategies, realtimeRepeats) * int64(evaluatedSpeakerConditionCount(speakerConditions, speakerEnabled))
	}
	var total int64
	for _, group := range augGroups {
		total += evalWorkUnits(len(group.Clips), strategies, realtimeRepeats) * int64(evaluatedSpeakerConditionCount(speakerConditions, speakerEnabled))
	}
	return total
}

func cellExperimentWorkUnits(clips []inteval.Clip, cells []*experimentv1.EvaluationCell) int64 {
	if len(clips) == 0 || len(cells) == 0 {
		return 0
	}
	var total int64
	for _, cell := range cells {
		if cell == nil || cell.GetRepeatCount() < 1 {
			continue
		}
		total += int64(len(clips)) * int64(cell.GetRepeatCount())
	}
	return total
}

func cellRecipeWorkUnits(clipCount int, cells []*experimentv1.EvaluationCell) int64 {
	if clipCount <= 0 || len(cells) == 0 {
		return 0
	}
	var total int64
	for _, cell := range cells {
		if cell == nil || cell.GetRepeatCount() < 1 {
			continue
		}
		units := int64(cell.GetRepeatCount())
		if cell.GetReplayLane() == experimentv1.ReplayLane_REPLAY_LANE_REALTIME {
			units++ // deterministic quality pass plus paced latency repeats
		}
		total += int64(clipCount) * units
	}
	return total
}

func evaluatedSpeakerConditionCount(conditions []speakerEvalCondition, speakerEnabled bool) int {
	if !speakerEnabled || len(conditions) == 0 {
		return 1
	}
	count := 0
	for _, cond := range conditions {
		if !cond.Skipped {
			count++
		}
	}
	return count
}

func evalWorkUnits(clipCount int, strategies []*evalv1.EvalStrategy, realtimeRepeats int32) int64 {
	if clipCount <= 0 {
		return 0
	}
	strategyCount := len(strategies)
	if strategyCount == 0 {
		strategyCount = 3
	}
	repeats := int(realtimeRepeats)
	if repeats < 0 {
		repeats = 0
	}
	return int64(clipCount * strategyCount * (1 + repeats))
}

func estimateClipSeconds(corpusSvc *intcorpus.Service) func(context.Context, []string) (int32, error) {
	if corpusSvc == nil {
		return nil
	}
	return func(ctx context.Context, clipIDs []string) (int32, error) {
		var clips []intcorpus.Clip
		var err error
		if len(clipIDs) == 0 {
			clips, err = corpusSvc.ListClips(ctx, intcorpus.ListFilter{})
			if err != nil {
				return 0, err
			}
		} else {
			clips = make([]intcorpus.Clip, 0, len(clipIDs))
			for _, id := range clipIDs {
				clip, err := corpusSvc.GetClip(ctx, id)
				if err != nil {
					return 0, err
				}
				clips = append(clips, clip)
			}
		}
		var totalMs int64
		for _, clip := range clips {
			if clip.DurationMs > 0 {
				totalMs += clip.DurationMs
			}
		}
		if totalMs <= 0 {
			return 0, nil
		}
		return int32((totalMs + 999) / 1000), nil
	}
}

// EstimateClipSeconds exposes the report queue-budget estimator to composition.

func EstimateClipSeconds(corpusSvc *intcorpus.Service) func(context.Context, []string) (int32, error) {
	return estimateClipSeconds(corpusSvc)
}

package measures

import (
	"sort"
	"time"

	"agent-manager/internal/outcomeledger"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// protoModePeriods lists the recorded mode periods that overlap the window.
// labels names the periods that are still active.
func protoModePeriods(periods []outcomeledger.ModePeriod, labels map[string]string, from, to time.Time) []*measurepb.UsageModePeriod {
	var out []*measurepb.UsageModePeriod
	for _, mode := range periods {
		end := mode.EndedAt
		if end.IsZero() {
			end = mode.ExpiresAt
		}
		if mode.StartedAt.After(to) || (!end.IsZero() && end.Before(from)) {
			continue
		}
		label := labels[mode.ID]
		if label == "" {
			label = mode.Label
		}
		out = append(out, &measurepb.UsageModePeriod{
			Id: mode.ID, Label: label, Runner: mode.Runner,
			StartedAt: rfc3339(mode.StartedAt), EndedAt: rfc3339(mode.EndedAt), ExpiresAt: rfc3339(mode.ExpiresAt),
		})
	}
	return out
}

// modelSeries sums each model's attributed points by the UTC day its run
// started. Runs without a start time cannot be placed and are left out.
func modelSeries(runs []outcomeledger.RunFacts, use map[string]map[string]outcomeledger.Use, from, to time.Time) []*measurepb.UsageModelSeries {
	type cell struct {
		points float64
		runs   int64
	}
	byModel := map[string]map[time.Time]*cell{}
	runners := map[string]string{}
	for _, run := range runs {
		at := run.FirstActivity
		if at.IsZero() {
			at = run.SessionStart
		}
		if at.IsZero() {
			at = run.CreatedAt
		}
		if at.IsZero() || at.Before(from) || at.After(to) {
			continue
		}
		day := at.UTC().Truncate(24 * time.Hour)
		for model, modelUse := range use[run.RunID] {
			if byModel[model] == nil {
				byModel[model] = map[time.Time]*cell{}
				runners[model] = run.Harness
			}
			c := byModel[model][day]
			if c == nil {
				c = &cell{}
				byModel[model][day] = c
			}
			c.points += modelUse.Points
			c.runs++
		}
	}
	models := make([]string, 0, len(byModel))
	for model := range byModel {
		models = append(models, model)
	}
	sort.Strings(models)
	out := make([]*measurepb.UsageModelSeries, 0, len(models))
	for _, model := range models {
		days := make([]time.Time, 0, len(byModel[model]))
		for day := range byModel[model] {
			days = append(days, day)
		}
		sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
		series := &measurepb.UsageModelSeries{Model: model, Runner: runners[model]}
		for _, day := range days {
			c := byModel[model][day]
			series.Buckets = append(series.Buckets, &measurepb.UsageSeriesBucket{At: rfc3339(day), Points: c.points, Runs: c.runs})
		}
		out = append(out, series)
	}
	return out
}

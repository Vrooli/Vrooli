package usageattribution

import (
	"fmt"
	"sort"
	"time"
)

// Dominance and sample floors for reading a model's weight from quota blocks.
const (
	dominantShare  = 0.9
	minRefitBlocks = 5
)

// Span is one recorded usage mode period.
type Span struct{ From, To time.Time }

// RefitRow says whether quota blocks one model dominated could fit its weight.
type RefitRow struct {
	Runner, Model, PriceSource string
	DominantBlocks, ModeBlocks int
	ImpliedPerMillion          float64
	ImpliedLow, ImpliedHigh    float64
	PriorPerMillion            float64
	Usable                     bool
	Reason                     string
}

// Refit reads each model's points per million consumption units from the quota
// blocks of the period's windows in which that model was at least dominantShare
// of priced use. Calibration fits one scale per window, so a shared block cannot
// tell models apart; only dominated blocks can. It is reported beside the
// weights and is not fed back into attribution.
func Refit(period PeriodUse, modes []Span) []RefitRow {
	type sample struct{ implied []float64 }
	dominant := map[string]*sample{}
	modeBlocks := map[string]int{}
	points, consumption := map[string]float64{}, map[string]float64{}
	for _, models := range period.Runs {
		for model, use := range models {
			points[model] += use.Points
			consumption[model] += Consumption(use.Tokens)
		}
	}
	runnerOf := map[string]string{}
	for _, window := range period.Windows {
		runner := RunnerFor(window.Provider)
		blockList := blocks(Slices(window, window.Start))
		if len(blockList) == 0 {
			continue
		}
		perBlock := make([]map[string]float64, len(blockList))
		usage := make([]map[string]float64, len(blockList))
		for i := range perBlock {
			perBlock[i], usage[i] = map[string]float64{}, map[string]float64{}
		}
		last := blockList[len(blockList)-1].To
		for _, turn := range period.Turns {
			if turn.Runner != runner || turn.At.Before(blockList[0].From) || turn.At.After(last) {
				continue
			}
			index := sort.Search(len(blockList), func(i int) bool { return !blockList[i].To.Before(turn.At) })
			if index >= len(blockList) {
				continue
			}
			runnerOf[turn.Model] = runner
			perBlock[index][turn.Model] += period.Prices(turn.Runner, turn.Model).normalized().cost(turn.Tokens)
			usage[index][turn.Model] += Consumption(turn.Tokens)
		}
		for i, block := range blockList {
			var total float64
			for _, cost := range perBlock[i] {
				total += cost
			}
			if total <= 0 {
				continue
			}
			for model, cost := range perBlock[i] {
				if cost/total < dominantShare || usage[i][model] <= 0 {
					continue
				}
				if dominant[model] == nil {
					dominant[model] = &sample{}
				}
				dominant[model].implied = append(dominant[model].implied, block.Points/usage[i][model]*1e6)
				for _, mode := range modes {
					if !block.To.Before(mode.From) && !block.From.After(mode.To) {
						modeBlocks[model]++
						break
					}
				}
			}
		}
	}
	names := map[string]bool{}
	for model := range consumption {
		if runnerOf[model] != "" {
			names[model] = true
		}
	}
	rows := make([]RefitRow, 0, len(names))
	for model := range names {
		row := RefitRow{Runner: runnerOf[model], Model: model, PriceSource: period.Sources[model], ModeBlocks: modeBlocks[model]}
		if consumption[model] > 0 {
			row.PriorPerMillion = points[model] / consumption[model] * 1e6
		}
		if s := dominant[model]; s != nil {
			sort.Float64s(s.implied)
			row.DominantBlocks = len(s.implied)
			row.ImpliedPerMillion = s.implied[len(s.implied)/2]
			row.ImpliedLow = s.implied[len(s.implied)/20]
			row.ImpliedHigh = s.implied[len(s.implied)-1-len(s.implied)/20]
		}
		switch {
		case row.DominantBlocks >= minRefitBlocks:
			row.Usable = true
			row.Reason = fmt.Sprintf("%d blocks dominated, %d in a mode period; implied weight is reported, not applied: calibration fits one scale per window", row.DominantBlocks, row.ModeBlocks)
		default:
			row.Reason = fmt.Sprintf("%d blocks dominated, %d in a mode period; need %d, so the weight stays %s", row.DominantBlocks, row.ModeBlocks, minRefitBlocks, firstNonEmptySource(row.PriceSource))
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Runner != rows[j].Runner {
			return rows[i].Runner < rows[j].Runner
		}
		return rows[i].Model < rows[j].Model
	})
	return rows
}

func firstNonEmptySource(source string) string {
	if source == "" {
		return "unfitted"
	}
	return "a " + source
}

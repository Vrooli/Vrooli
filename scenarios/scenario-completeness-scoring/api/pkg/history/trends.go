package history

import (
	"math"
	"time"
)

// TrendDirection indicates the direction of score change
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendDeclining TrendDirection = "declining"
	TrendStalled   TrendDirection = "stalled"
	TrendStable    TrendDirection = "stable"
	TrendUnknown   TrendDirection = "unknown"
)

// TrendAnalysis contains trend analysis for a scenario
// [REQ:SCS-HIST-003] Trend detection
type TrendAnalysis struct {
	Scenario           string         `json:"scenario"`
	CurrentScore       int            `json:"current_score"`
	PreviousScore      int            `json:"previous_score"`
	Delta              int            `json:"delta"`
	Direction          TrendDirection `json:"direction"`
	Classification     string         `json:"classification"`
	StallCount         int            `json:"stall_count"`
	IsStalled          bool           `json:"is_stalled"`
	SnapshotsAnalyzed  int            `json:"snapshots_analyzed"`
	OldestSnapshot     *time.Time     `json:"oldest_snapshot,omitempty"`
	NewestSnapshot     *time.Time     `json:"newest_snapshot,omitempty"`
	AverageScore       float64        `json:"average_score"`
	MinScore           int            `json:"min_score"`
	MaxScore           int            `json:"max_score"`
	Volatility         float64        `json:"volatility"`
}

// TrendAnalyzer analyzes score trends
// [REQ:SCS-HIST-003] Trend detection
type TrendAnalyzer struct {
	repo          *Repository
	stallThreshold int // Number of unchanged snapshots to consider stalled
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer(repo *Repository, stallThreshold int) *TrendAnalyzer {
	if stallThreshold <= 0 {
		stallThreshold = 5 // Default: 5 unchanged snapshots = stalled
	}
	return &TrendAnalyzer{
		repo:           repo,
		stallThreshold: stallThreshold,
	}
}

// Analyze performs trend analysis for a scenario
// [REQ:SCS-HIST-003] Detect score improvements/regressions and stalls
func (a *TrendAnalyzer) Analyze(scenario string, snapshotLimit int) (*TrendAnalysis, error) {
	if snapshotLimit <= 0 {
		snapshotLimit = 30
	}

	snapshots, err := a.repo.GetHistory(scenario, snapshotLimit)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return &TrendAnalysis{
			Scenario:          scenario,
			Direction:         TrendUnknown,
			SnapshotsAnalyzed: 0,
		}, nil
	}

	analysis := &TrendAnalysis{
		Scenario:          scenario,
		CurrentScore:      snapshots[0].Score,
		Classification:    snapshots[0].Classification,
		SnapshotsAnalyzed: len(snapshots),
	}

	// Set newest/oldest timestamps
	newestTime := snapshots[0].CreatedAt
	oldestTime := snapshots[len(snapshots)-1].CreatedAt
	analysis.NewestSnapshot = &newestTime
	analysis.OldestSnapshot = &oldestTime

	if len(snapshots) == 1 {
		analysis.Direction = TrendUnknown
		analysis.AverageScore = float64(snapshots[0].Score)
		analysis.MinScore = snapshots[0].Score
		analysis.MaxScore = snapshots[0].Score
		return analysis, nil
	}

	// Calculate delta from previous
	analysis.PreviousScore = snapshots[1].Score
	analysis.Delta = analysis.CurrentScore - analysis.PreviousScore

	// Calculate statistics
	sum := 0
	analysis.MinScore = snapshots[0].Score
	analysis.MaxScore = snapshots[0].Score
	for _, s := range snapshots {
		sum += s.Score
		if s.Score < analysis.MinScore {
			analysis.MinScore = s.Score
		}
		if s.Score > analysis.MaxScore {
			analysis.MaxScore = s.Score
		}
	}
	analysis.AverageScore = float64(sum) / float64(len(snapshots))

	// Calculate volatility (standard deviation)
	var sumSquaredDiff float64
	for _, s := range snapshots {
		diff := float64(s.Score) - analysis.AverageScore
		sumSquaredDiff += diff * diff
	}
	analysis.Volatility = math.Sqrt(sumSquaredDiff / float64(len(snapshots)))

	// Detect stalls (count consecutive unchanged scores from most recent)
	stallCount := 0
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i].Score == analysis.CurrentScore {
			stallCount++
		} else {
			break
		}
	}
	analysis.StallCount = stallCount
	analysis.IsStalled = stallCount >= a.stallThreshold

	// Determine direction
	analysis.Direction = a.determineDirection(analysis)

	return analysis, nil
}

// determineDirection calculates the overall trend direction
func (a *TrendAnalyzer) determineDirection(analysis *TrendAnalysis) TrendDirection {
	// If stalled, report as stalled regardless of delta
	if analysis.IsStalled {
		return TrendStalled
	}

	// Use delta to determine direction
	if analysis.Delta > 0 {
		return TrendImproving
	} else if analysis.Delta < 0 {
		return TrendDeclining
	}

	// Delta is 0 but not stalled yet - consider stable
	return TrendStable
}

// AnalyzeAll performs trend analysis for all scenarios with history
func (a *TrendAnalyzer) AnalyzeAll(snapshotLimit int) ([]*TrendAnalysis, error) {
	scenarios, err := a.repo.GetAllScenarios()
	if err != nil {
		return nil, err
	}

	var results []*TrendAnalysis
	for _, scenario := range scenarios {
		analysis, err := a.Analyze(scenario, snapshotLimit)
		if err != nil {
			// Log and continue with other scenarios
			continue
		}
		results = append(results, analysis)
	}

	return results, nil
}

// GetTrendSummary returns a summary of trends across all scenarios
type TrendSummary struct {
	TotalScenarios int            `json:"total_scenarios"`
	Improving      int            `json:"improving"`
	Declining      int            `json:"declining"`
	Stalled        int            `json:"stalled"`
	Stable         int            `json:"stable"`
	Unknown        int            `json:"unknown"`
	AverageScore   float64        `json:"average_score"`
	TopImprovers   []*TrendAnalysis `json:"top_improvers,omitempty"`
	TopDecliners   []*TrendAnalysis `json:"top_decliners,omitempty"`
}

// GetTrendSummary returns an aggregate summary of all scenario trends
func (a *TrendAnalyzer) GetTrendSummary(snapshotLimit int) (*TrendSummary, error) {
	analyses, err := a.AnalyzeAll(snapshotLimit)
	if err != nil {
		return nil, err
	}

	summary := &TrendSummary{
		TotalScenarios: len(analyses),
	}

	if len(analyses) == 0 {
		return summary, nil
	}

	var scoreSum float64
	var improvers, decliners []*TrendAnalysis

	for _, analysis := range analyses {
		scoreSum += float64(analysis.CurrentScore)

		switch analysis.Direction {
		case TrendImproving:
			summary.Improving++
			improvers = append(improvers, analysis)
		case TrendDeclining:
			summary.Declining++
			decliners = append(decliners, analysis)
		case TrendStalled:
			summary.Stalled++
		case TrendStable:
			summary.Stable++
		default:
			summary.Unknown++
		}
	}

	summary.AverageScore = scoreSum / float64(len(analyses))

	// Sort and take top 5 improvers and decliners
	// Simple bubble sort for small lists
	for i := 0; i < len(improvers)-1; i++ {
		for j := 0; j < len(improvers)-i-1; j++ {
			if improvers[j].Delta < improvers[j+1].Delta {
				improvers[j], improvers[j+1] = improvers[j+1], improvers[j]
			}
		}
	}
	for i := 0; i < len(decliners)-1; i++ {
		for j := 0; j < len(decliners)-i-1; j++ {
			if decliners[j].Delta > decliners[j+1].Delta {
				decliners[j], decliners[j+1] = decliners[j+1], decliners[j]
			}
		}
	}

	if len(improvers) > 5 {
		improvers = improvers[:5]
	}
	if len(decliners) > 5 {
		decliners = decliners[:5]
	}

	summary.TopImprovers = improvers
	summary.TopDecliners = decliners

	return summary, nil
}

package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/history"
	"scenario-completeness-scoring/pkg/scoring"
)

// BulkRefreshResult represents the result of refreshing all scenario scores
// [REQ:SCS-ANALYSIS-003] Bulk score refresh result structure
type BulkRefreshResult struct {
	Total        int                   `json:"total"`
	Successful   int                   `json:"successful"`
	Failed       int                   `json:"failed"`
	Scenarios    []ScenarioRefreshInfo `json:"scenarios"`
	Duration     time.Duration         `json:"duration_ms"`
	RefreshedAt  time.Time             `json:"refreshed_at"`
}

// ScenarioRefreshInfo represents refresh result for a single scenario
type ScenarioRefreshInfo struct {
	Scenario       string `json:"scenario"`
	Category       string `json:"category,omitempty"`
	Score          int    `json:"score,omitempty"`
	Classification string `json:"classification,omitempty"`
	PreviousScore  int    `json:"previous_score,omitempty"`
	Delta          int    `json:"delta,omitempty"`
	Error          string `json:"error,omitempty"`
	Success        bool   `json:"success"`
}

// BulkRefresher handles bulk score refresh operations
type BulkRefresher struct {
	vrooliRoot  string
	collector   *collectors.MetricsCollector
	historyRepo *history.Repository
}

// NewBulkRefresher creates a new bulk refresher
func NewBulkRefresher(vrooliRoot string, collector *collectors.MetricsCollector, historyRepo *history.Repository) *BulkRefresher {
	return &BulkRefresher{
		vrooliRoot:  vrooliRoot,
		collector:   collector,
		historyRepo: historyRepo,
	}
}

// RefreshAll recalculates scores for all scenarios
// [REQ:SCS-ANALYSIS-003] Bulk score refresh implementation
func (b *BulkRefresher) RefreshAll() (*BulkRefreshResult, error) {
	start := time.Now()

	scenariosDir := filepath.Join(b.vrooliRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, err
	}

	// Collect scenario names
	var scenarioNames []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		scenarioNames = append(scenarioNames, entry.Name())
	}

	// Process scenarios concurrently with limited parallelism
	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []ScenarioRefreshInfo

	for _, name := range scenarioNames {
		wg.Add(1)
		go func(scenarioName string) {
			defer wg.Done()
			sem <- struct{}{} // Acquire
			defer func() { <-sem }() // Release

			info := b.refreshScenario(scenarioName)
			mu.Lock()
			results = append(results, info)
			mu.Unlock()
		}(name)
	}

	wg.Wait()

	// Count successes and failures
	successful := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			successful++
		} else {
			failed++
		}
	}

	return &BulkRefreshResult{
		Total:       len(results),
		Successful:  successful,
		Failed:      failed,
		Scenarios:   results,
		Duration:    time.Since(start),
		RefreshedAt: time.Now().UTC(),
	}, nil
}

// refreshScenario refreshes a single scenario
func (b *BulkRefresher) refreshScenario(scenarioName string) ScenarioRefreshInfo {
	metrics, err := b.collector.Collect(scenarioName)
	if err != nil {
		return ScenarioRefreshInfo{
			Scenario: scenarioName,
			Error:    err.Error(),
			Success:  false,
		}
	}

	thresholds := scoring.GetThresholds(metrics.Category)
	breakdown := scoring.CalculateCompletenessScore(*metrics, thresholds, 0)

	// Get previous score from history if available
	previousScore := 0
	if b.historyRepo != nil {
		latest, err := b.historyRepo.GetLatest(scenarioName)
		if err == nil && latest != nil {
			previousScore = latest.Score
		}

		// Save new snapshot
		_, _ = b.historyRepo.Save(scenarioName, &breakdown, nil)
	}

	return ScenarioRefreshInfo{
		Scenario:       scenarioName,
		Category:       metrics.Category,
		Score:          breakdown.Score,
		Classification: breakdown.Classification,
		PreviousScore:  previousScore,
		Delta:          breakdown.Score - previousScore,
		Success:        true,
	}
}

// RefreshSelected recalculates scores for selected scenarios
func (b *BulkRefresher) RefreshSelected(scenarioNames []string) (*BulkRefreshResult, error) {
	start := time.Now()

	// Process scenarios concurrently with limited parallelism
	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []ScenarioRefreshInfo

	for _, name := range scenarioNames {
		wg.Add(1)
		go func(scenarioName string) {
			defer wg.Done()
			sem <- struct{}{} // Acquire
			defer func() { <-sem }() // Release

			info := b.refreshScenario(scenarioName)
			mu.Lock()
			results = append(results, info)
			mu.Unlock()
		}(name)
	}

	wg.Wait()

	// Count successes and failures
	successful := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			successful++
		} else {
			failed++
		}
	}

	return &BulkRefreshResult{
		Total:       len(results),
		Successful:  successful,
		Failed:      failed,
		Scenarios:   results,
		Duration:    time.Since(start),
		RefreshedAt: time.Now().UTC(),
	}, nil
}

// CompareScenarios compares scores across multiple scenarios
// [REQ:SCS-ANALYSIS-004] Cross-scenario comparison
type ComparisonResult struct {
	Scenarios   []ScenarioComparison `json:"scenarios"`
	BestScore   int                  `json:"best_score"`
	WorstScore  int                  `json:"worst_score"`
	AverageScore float64             `json:"average_score"`
	ComparedAt  time.Time            `json:"compared_at"`
}

// ScenarioComparison represents a scenario in a comparison
type ScenarioComparison struct {
	Scenario       string                 `json:"scenario"`
	Category       string                 `json:"category"`
	Score          int                    `json:"score"`
	Classification string                 `json:"classification"`
	Breakdown      scoring.ScoreBreakdown `json:"breakdown"`
	Rank           int                    `json:"rank"`
}

// Compare compares multiple scenarios
func (b *BulkRefresher) Compare(scenarioNames []string) (*ComparisonResult, error) {
	var comparisons []ScenarioComparison

	for _, name := range scenarioNames {
		metrics, err := b.collector.Collect(name)
		if err != nil {
			continue // Skip scenarios that fail to load
		}

		thresholds := scoring.GetThresholds(metrics.Category)
		breakdown := scoring.CalculateCompletenessScore(*metrics, thresholds, 0)

		comparisons = append(comparisons, ScenarioComparison{
			Scenario:       name,
			Category:       metrics.Category,
			Score:          breakdown.Score,
			Classification: breakdown.Classification,
			Breakdown:      breakdown,
		})
	}

	if len(comparisons) == 0 {
		return &ComparisonResult{
			Scenarios:    comparisons,
			ComparedAt:   time.Now().UTC(),
		}, nil
	}

	// Sort by score (descending) and assign ranks
	// Simple bubble sort for small lists
	for i := 0; i < len(comparisons); i++ {
		for j := i + 1; j < len(comparisons); j++ {
			if comparisons[j].Score > comparisons[i].Score {
				comparisons[i], comparisons[j] = comparisons[j], comparisons[i]
			}
		}
	}

	// Assign ranks
	for i := range comparisons {
		comparisons[i].Rank = i + 1
	}

	// Calculate stats
	bestScore := comparisons[0].Score
	worstScore := comparisons[len(comparisons)-1].Score
	totalScore := 0
	for _, c := range comparisons {
		totalScore += c.Score
	}
	averageScore := float64(totalScore) / float64(len(comparisons))

	return &ComparisonResult{
		Scenarios:    comparisons,
		BestScore:    bestScore,
		WorstScore:   worstScore,
		AverageScore: averageScore,
		ComparedAt:   time.Now().UTC(),
	}, nil
}

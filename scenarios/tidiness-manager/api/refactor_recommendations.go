package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// RefactorRecommendation combines visited-tracker data with detailed file metrics
type RefactorRecommendation struct {
	FilePath      string `json:"file_path"`
	Language      string `json:"language"`
	FileExtension string `json:"file_extension"`

	// visited-tracker data
	VisitCount     int        `json:"visit_count"`
	LastVisited    *time.Time `json:"last_visited,omitempty"`
	StalenessScore float64    `json:"staleness_score"`

	// Code quality metrics
	LineCount      int      `json:"line_count"`
	TodoCount      int      `json:"todo_count"`
	FixmeCount     int      `json:"fixme_count"`
	HackCount      int      `json:"hack_count"`
	ImportCount    int      `json:"import_count"`
	FunctionCount  int      `json:"function_count"`
	CodeLines      int      `json:"code_lines"`
	CommentLines   int      `json:"comment_lines"`
	CommentRatio   float64  `json:"comment_to_code_ratio"`
	HasTestFile    bool     `json:"has_test_file"`
	ComplexityAvg  *float64 `json:"complexity_avg,omitempty"`
	ComplexityMax  *int     `json:"complexity_max,omitempty"`
	DuplicationPct *float64 `json:"duplication_pct,omitempty"`

	// Computed priority
	RefactorPriority float64 `json:"refactor_priority"`
}

// RefactorRecommender provides refactor recommendations combining visited-tracker + file metrics
type RefactorRecommender struct {
	db          *sql.DB
	campaignMgr *CampaignManager
	prioritizer *FilePrioritizer
}

// toFloatPtr converts sql.NullFloat64 to *float64
func toFloatPtr(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	val := nf.Float64
	return &val
}

// toIntPtr converts sql.NullInt64 to *int
func toIntPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	val := int(ni.Int64)
	return &val
}

// getMetricFloat extracts float from nullable pointer, defaults to 0
func getMetricFloat(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}

// getMetricInt extracts int from nullable pointer, defaults to 0
func getMetricInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// NewRefactorRecommender creates a RefactorRecommender instance
func NewRefactorRecommender(db *sql.DB, campaignMgr *CampaignManager) *RefactorRecommender {
	return &RefactorRecommender{
		db:          db,
		campaignMgr: campaignMgr,
		prioritizer: NewFilePrioritizer(3), // default max 3 visits
	}
}

// GetRecommendations returns refactor recommendations for a scenario
func (rr *RefactorRecommender) GetRecommendations(ctx context.Context, scenario string, limit int, sortBy string, minLines int, maxVisits int) ([]RefactorRecommendation, error) {
	// Get campaign from visited-tracker (if available)
	var campaign *Campaign
	var trackedFiles []TrackedFile

	if rr.campaignMgr != nil {
		campaign, _ = rr.campaignMgr.GetOrCreateCampaign(scenario)
		if campaign != nil {
			trackedFiles, _ = rr.campaignMgr.GetCampaignFiles(campaign.ID)
		}
	}

	// Create map of tracked files for quick lookup
	trackedMap := make(map[string]*TrackedFile)
	for i := range trackedFiles {
		trackedMap[trackedFiles[i].FilePath] = &trackedFiles[i]
	}

	// Get file metrics from database
	fileMetrics, err := rr.getFileMetricsFromDB(ctx, scenario, minLines)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metrics: %w", err)
	}

	// Combine data
	recommendations := make([]RefactorRecommendation, 0, len(fileMetrics))

	for _, metrics := range fileMetrics {
		rec := RefactorRecommendation{
			FilePath:       metrics.FilePath,
			Language:       metrics.Language,
			FileExtension:  metrics.FileExtension,
			LineCount:      metrics.LineCount,
			TodoCount:      metrics.TodoCount,
			FixmeCount:     metrics.FixmeCount,
			HackCount:      metrics.HackCount,
			ImportCount:    metrics.ImportCount,
			FunctionCount:  metrics.FunctionCount,
			CodeLines:      metrics.CodeLines,
			CommentLines:   metrics.CommentLines,
			CommentRatio:   metrics.CommentRatio,
			HasTestFile:    metrics.HasTestFile,
			ComplexityAvg:  metrics.ComplexityAvg,
			ComplexityMax:  metrics.ComplexityMax,
			DuplicationPct: metrics.DuplicationPct,
		}

		// Add visit tracking data if available
		if tracked, exists := trackedMap[metrics.FilePath]; exists {
			rec.VisitCount = tracked.VisitCount
			if !tracked.LastVisited.IsZero() {
				rec.LastVisited = &tracked.LastVisited
			}
			rec.StalenessScore = tracked.StalenessScore

			// Apply max visits filter
			if maxVisits > 0 && tracked.VisitCount >= maxVisits {
				continue
			}
		} else {
			// File not in visited-tracker yet - high priority for unvisited files
			rec.VisitCount = 0
			rec.StalenessScore = 1000.0
		}

		// Calculate composite priority score
		rec.RefactorPriority = rr.calculatePriority(rec)

		recommendations = append(recommendations, rec)
	}

	// Sort by requested criteria
	rr.sortRecommendations(recommendations, sortBy)

	// Limit results
	if limit > 0 && len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// getFileMetricsFromDB retrieves file metrics from database
func (rr *RefactorRecommender) getFileMetricsFromDB(ctx context.Context, scenario string, minLines int) ([]DetailedFileMetrics, error) {
	query := `
		SELECT
			file_path, language, file_extension, line_count,
			todo_count, fixme_count, hack_count,
			import_count, function_count, code_lines, comment_lines,
			comment_to_code_ratio, has_test_file,
			complexity_avg, complexity_max, duplication_pct
		FROM file_metrics
		WHERE scenario = $1
	`

	args := []interface{}{scenario}

	if minLines > 0 {
		query += " AND line_count >= $2"
		args = append(args, minLines)
	}

	rows, err := rr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []DetailedFileMetrics
	for rows.Next() {
		var m DetailedFileMetrics
		var lang, ext sql.NullString
		var complexityAvg sql.NullFloat64
		var complexityMax sql.NullInt64
		var duplicationPct sql.NullFloat64

		err := rows.Scan(
			&m.FilePath, &lang, &ext, &m.LineCount,
			&m.TodoCount, &m.FixmeCount, &m.HackCount,
			&m.ImportCount, &m.FunctionCount, &m.CodeLines, &m.CommentLines,
			&m.CommentRatio, &m.HasTestFile,
			&complexityAvg, &complexityMax, &duplicationPct,
		)
		if err != nil {
			continue
		}

		if lang.Valid {
			m.Language = lang.String
		}
		if ext.Valid {
			m.FileExtension = ext.String
		}
		m.ComplexityAvg = toFloatPtr(complexityAvg)
		m.ComplexityMax = toIntPtr(complexityMax)
		m.DuplicationPct = toFloatPtr(duplicationPct)

		metrics = append(metrics, m)
	}

	return metrics, nil
}

// calculatePriority computes a composite refactor priority score
func (rr *RefactorRecommender) calculatePriority(rec RefactorRecommendation) float64 {
	score := rec.StalenessScore // Base: visited-tracker staleness (0-1000+)

	// Length penalty (0-100 points) - files over 500 lines
	if rec.LineCount > 500 {
		score += float64(rec.LineCount-500) / 10.0
	}

	// Complexity penalty (0-50 points) - complexity over 10
	complexity := getMetricInt(rec.ComplexityMax)
	if complexity > 10 {
		score += float64(complexity-10) * 2.0
	}

	// Duplication penalty (0-30 points) - over 5% duplication
	duplication := getMetricFloat(rec.DuplicationPct)
	if duplication > 0.05 {
		score += (duplication - 0.05) * 300.0
	}

	// Technical debt penalty (0-20 points)
	debtMarkers := rec.TodoCount + rec.FixmeCount*2 + rec.HackCount*3
	score += float64(debtMarkers)

	// Missing tests penalty (0-10 points)
	if !rec.HasTestFile {
		score += 10.0
	}

	// Poor comment ratio penalty (0-10 points) - less than 5% comments
	if rec.CommentRatio < 0.05 {
		score += 10.0
	}

	return score
}

// sortRecommendations sorts by the requested criteria
func (rr *RefactorRecommender) sortRecommendations(recs []RefactorRecommendation, sortBy string) {
	switch sortBy {
	case "complexity":
		sort.Slice(recs, func(i, j int) bool {
			return getMetricInt(recs[i].ComplexityMax) > getMetricInt(recs[j].ComplexityMax)
		})
	case "length":
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].LineCount > recs[j].LineCount
		})
	case "duplication":
		sort.Slice(recs, func(i, j int) bool {
			return getMetricFloat(recs[i].DuplicationPct) > getMetricFloat(recs[j].DuplicationPct)
		})
	case "priority":
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].RefactorPriority > recs[j].RefactorPriority
		})
	case "staleness":
		fallthrough
	default:
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].StalenessScore > recs[j].StalenessScore
		})
	}
}

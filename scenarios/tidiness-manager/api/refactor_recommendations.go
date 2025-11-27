package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// RefactorRecommendation combines visited-tracker data with detailed file metrics
type RefactorRecommendation struct {
	FilePath         string     `json:"file_path"`
	Language         string     `json:"language"`
	FileExtension    string     `json:"file_extension"`

	// visited-tracker data
	VisitCount       int        `json:"visit_count"`
	LastVisited      *time.Time `json:"last_visited,omitempty"`
	StalenessScore   float64    `json:"staleness_score"`

	// Code quality metrics
	LineCount        int        `json:"line_count"`
	TodoCount        int        `json:"todo_count"`
	FixmeCount       int        `json:"fixme_count"`
	HackCount        int        `json:"hack_count"`
	ImportCount      int        `json:"import_count"`
	FunctionCount    int        `json:"function_count"`
	CodeLines        int        `json:"code_lines"`
	CommentLines     int        `json:"comment_lines"`
	CommentRatio     float64    `json:"comment_to_code_ratio"`
	HasTestFile      bool       `json:"has_test_file"`
	ComplexityAvg    *float64   `json:"complexity_avg,omitempty"`
	ComplexityMax    *int       `json:"complexity_max,omitempty"`
	DuplicationPct   *float64   `json:"duplication_pct,omitempty"`

	// Computed priority
	RefactorPriority float64    `json:"refactor_priority"`
}

// RefactorRecommender provides refactor recommendations combining visited-tracker + file metrics
type RefactorRecommender struct {
	db          *sql.DB
	campaignMgr *CampaignManager
	prioritizer *FilePrioritizer
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
			FilePath:      metrics.FilePath,
			Language:      metrics.Language,
			FileExtension: metrics.FileExtension,
			LineCount:     metrics.LineCount,
			TodoCount:     metrics.TodoCount,
			FixmeCount:    metrics.FixmeCount,
			HackCount:     metrics.HackCount,
			ImportCount:   metrics.ImportCount,
			FunctionCount: metrics.FunctionCount,
			CodeLines:     metrics.CodeLines,
			CommentLines:  metrics.CommentLines,
			CommentRatio:  metrics.CommentRatio,
			HasTestFile:   metrics.HasTestFile,
			ComplexityAvg: metrics.ComplexityAvg,
			ComplexityMax: metrics.ComplexityMax,
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
		if complexityAvg.Valid {
			val := complexityAvg.Float64
			m.ComplexityAvg = &val
		}
		if complexityMax.Valid {
			val := int(complexityMax.Int64)
			m.ComplexityMax = &val
		}
		if duplicationPct.Valid {
			val := duplicationPct.Float64
			m.DuplicationPct = &val
		}

		metrics = append(metrics, m)
	}

	return metrics, nil
}

// calculatePriority computes a composite refactor priority score
func (rr *RefactorRecommender) calculatePriority(rec RefactorRecommendation) float64 {
	score := 0.0

	// Base: visited-tracker staleness (0-1000+)
	score += rec.StalenessScore

	// Length penalty (0-100 points) - files over 500 lines
	if rec.LineCount > 500 {
		score += float64(rec.LineCount-500) / 10.0
	}

	// Complexity penalty (0-50 points) - complexity over 10
	if rec.ComplexityMax != nil && *rec.ComplexityMax > 10 {
		score += float64(*rec.ComplexityMax-10) * 2.0
	}

	// Duplication penalty (0-30 points) - over 5% duplication
	if rec.DuplicationPct != nil && *rec.DuplicationPct > 0.05 {
		score += (*rec.DuplicationPct - 0.05) * 300.0
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
			ci := 0
			if recs[i].ComplexityMax != nil {
				ci = *recs[i].ComplexityMax
			}
			cj := 0
			if recs[j].ComplexityMax != nil {
				cj = *recs[j].ComplexityMax
			}
			return ci > cj
		})
	case "length":
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].LineCount > recs[j].LineCount
		})
	case "duplication":
		sort.Slice(recs, func(i, j int) bool {
			di := 0.0
			if recs[i].DuplicationPct != nil {
				di = *recs[i].DuplicationPct
			}
			dj := 0.0
			if recs[j].DuplicationPct != nil {
				dj = *recs[j].DuplicationPct
			}
			return di > dj
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

// HTTP handler for refactor recommendations endpoint
func (s *Server) handleRefactorRecommendations(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")
	if scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario parameter is required")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		sortBy = "priority" // default to composite priority
	}

	minLines := 0
	if ml := r.URL.Query().Get("min_lines"); ml != "" {
		if v, err := strconv.Atoi(ml); err == nil && v > 0 {
			minLines = v
		}
	}

	maxVisits := 0
	if mv := r.URL.Query().Get("max_visits"); mv != "" {
		if v, err := strconv.Atoi(mv); err == nil && v > 0 {
			maxVisits = v
		}
	}

	recommender := NewRefactorRecommender(s.db, s.campaignMgr)
	recommendations, err := recommender.GetRecommendations(
		r.Context(),
		scenario,
		limit,
		sortBy,
		minLines,
		maxVisits,
	)
	if err != nil {
		s.log("failed to get refactor recommendations", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenario,
		})
		respondError(w, http.StatusInternalServerError, "failed to get recommendations")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scenario":        scenario,
		"recommendations": recommendations,
		"count":           len(recommendations),
	})
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Performance history handlers
func getPerformanceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	algorithmID := vars["id"]

	// Parse query parameters
	language := r.URL.Query().Get("language")
	inputSize := r.URL.Query().Get("input_size")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	// Build query
	query := `
		SELECT 
			ph.id,
			ph.algorithm_id,
			ph.implementation_id,
			ph.language,
			ph.avg_execution_time_ms,
			ph.min_execution_time_ms,
			ph.max_execution_time_ms,
			ph.avg_memory_mb,
			ph.input_size,
			ph.sample_count,
			ph.performance_score,
			ph.recorded_at,
			i.version as implementation_version
		FROM performance_history ph
		LEFT JOIN implementations i ON i.id = ph.implementation_id
		WHERE ph.algorithm_id = $1
	`

	args := []interface{}{algorithmID}
	argCount := 2

	if language != "" {
		query += fmt.Sprintf(" AND ph.language = $%d", argCount)
		args = append(args, language)
		argCount++
	}

	if inputSize != "" {
		query += fmt.Sprintf(" AND ph.input_size = $%d", argCount)
		args = append(args, inputSize)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY ph.recorded_at DESC LIMIT $%d", argCount)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("Error fetching performance history:", err)
		http.Error(w, "Error fetching performance history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var entry struct {
			ID                    string
			AlgorithmID           string
			ImplementationID      *string
			Language              string
			AvgExecutionTimeMs    float64
			MinExecutionTimeMs    float64
			MaxExecutionTimeMs    float64
			AvgMemoryMb           float64
			InputSize             int
			SampleCount           int
			PerformanceScore      *float64
			RecordedAt            time.Time
			ImplementationVersion *string
		}

		err := rows.Scan(
			&entry.ID,
			&entry.AlgorithmID,
			&entry.ImplementationID,
			&entry.Language,
			&entry.AvgExecutionTimeMs,
			&entry.MinExecutionTimeMs,
			&entry.MaxExecutionTimeMs,
			&entry.AvgMemoryMb,
			&entry.InputSize,
			&entry.SampleCount,
			&entry.PerformanceScore,
			&entry.RecordedAt,
			&entry.ImplementationVersion,
		)
		if err != nil {
			log.Println("Error scanning performance history row:", err)
			continue
		}

		historyEntry := map[string]interface{}{
			"id":                    entry.ID,
			"algorithm_id":          entry.AlgorithmID,
			"language":              entry.Language,
			"avg_execution_time_ms": entry.AvgExecutionTimeMs,
			"min_execution_time_ms": entry.MinExecutionTimeMs,
			"max_execution_time_ms": entry.MaxExecutionTimeMs,
			"avg_memory_mb":         entry.AvgMemoryMb,
			"input_size":            entry.InputSize,
			"sample_count":          entry.SampleCount,
			"recorded_at":           entry.RecordedAt.Format(time.RFC3339),
		}

		if entry.ImplementationID != nil {
			historyEntry["implementation_id"] = *entry.ImplementationID
		}
		if entry.PerformanceScore != nil {
			historyEntry["performance_score"] = *entry.PerformanceScore
		}
		if entry.ImplementationVersion != nil {
			historyEntry["implementation_version"] = *entry.ImplementationVersion
		}

		history = append(history, historyEntry)
	}

	response := map[string]interface{}{
		"algorithm_id": algorithmID,
		"history":      history,
		"count":        len(history),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getPerformanceTrendsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	algorithmID := vars["id"]

	// Get trends from the view
	query := `
		SELECT 
			algorithm_name,
			display_name,
			language,
			week,
			avg_weekly_time_ms,
			avg_weekly_memory_mb,
			avg_weekly_score,
			sample_count
		FROM performance_trends
		WHERE algorithm_id = $1
		ORDER BY week DESC
		LIMIT 52  -- Last year of data
	`

	rows, err := db.Query(query, algorithmID)
	if err != nil {
		log.Println("Error fetching performance trends:", err)
		http.Error(w, "Error fetching performance trends", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var trends []map[string]interface{}
	var algorithmName, displayName string

	for rows.Next() {
		var trend struct {
			AlgorithmName     string
			DisplayName       string
			Language          string
			Week              time.Time
			AvgWeeklyTimeMs   float64
			AvgWeeklyMemoryMb float64
			AvgWeeklyScore    *float64
			SampleCount       int
		}

		err := rows.Scan(
			&trend.AlgorithmName,
			&trend.DisplayName,
			&trend.Language,
			&trend.Week,
			&trend.AvgWeeklyTimeMs,
			&trend.AvgWeeklyMemoryMb,
			&trend.AvgWeeklyScore,
			&trend.SampleCount,
		)
		if err != nil {
			log.Println("Error scanning trend row:", err)
			continue
		}

		algorithmName = trend.AlgorithmName
		displayName = trend.DisplayName

		trendEntry := map[string]interface{}{
			"language":           trend.Language,
			"week":               trend.Week.Format("2006-01-02"),
			"avg_execution_time": trend.AvgWeeklyTimeMs,
			"avg_memory_mb":      trend.AvgWeeklyMemoryMb,
			"sample_count":       trend.SampleCount,
		}

		if trend.AvgWeeklyScore != nil {
			trendEntry["avg_performance_score"] = *trend.AvgWeeklyScore
		}

		trends = append(trends, trendEntry)
	}

	response := map[string]interface{}{
		"algorithm_id":   algorithmID,
		"algorithm_name": algorithmName,
		"display_name":   displayName,
		"trends":         trends,
		"count":          len(trends),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func recordPerformanceHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AlgorithmID      string `json:"algorithm_id"`
		ImplementationID string `json:"implementation_id,omitempty"`
		Language         string `json:"language"`
		ExecutionResults []struct {
			ExecutionTimeMs float64 `json:"execution_time_ms"`
			MemoryMb        float64 `json:"memory_mb"`
		} `json:"execution_results"`
		InputSize    int                    `json:"input_size"`
		TestCategory string                 `json:"test_category,omitempty"`
		Environment  map[string]interface{} `json:"environment,omitempty"`
		Notes        string                 `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Calculate statistics from execution results
	var totalTime, totalMemory float64
	minTime, minMemory := math.MaxFloat64, math.MaxFloat64
	var maxTime, maxMemory float64

	for _, result := range request.ExecutionResults {
		totalTime += result.ExecutionTimeMs
		totalMemory += result.MemoryMb

		if result.ExecutionTimeMs < minTime {
			minTime = result.ExecutionTimeMs
		}
		if result.ExecutionTimeMs > maxTime {
			maxTime = result.ExecutionTimeMs
		}
		if result.MemoryMb < minMemory {
			minMemory = result.MemoryMb
		}
		if result.MemoryMb > maxMemory {
			maxMemory = result.MemoryMb
		}
	}

	sampleCount := len(request.ExecutionResults)
	avgTime := totalTime / float64(sampleCount)
	avgMemory := totalMemory / float64(sampleCount)

	// Calculate standard deviation for time
	var sumSquaredDiff float64
	for _, result := range request.ExecutionResults {
		diff := result.ExecutionTimeMs - avgTime
		sumSquaredDiff += diff * diff
	}
	stdDevTime := math.Sqrt(sumSquaredDiff / float64(sampleCount))

	// Convert environment to JSON
	envJSON, _ := json.Marshal(request.Environment)

	// Insert into performance_history table
	query := `
		INSERT INTO performance_history (
			algorithm_id, implementation_id, language,
			avg_execution_time_ms, min_execution_time_ms, max_execution_time_ms, std_dev_time_ms,
			avg_memory_mb, min_memory_mb, max_memory_mb,
			input_size, sample_count, test_category,
			environment_info, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, performance_score, recorded_at
	`

	var historyID string
	var performanceScore *float64
	var recordedAt time.Time

	// Handle optional implementation_id
	var implID interface{}
	if request.ImplementationID != "" {
		implID = request.ImplementationID
	} else {
		implID = nil
	}

	err := db.QueryRow(
		query,
		request.AlgorithmID, implID, request.Language,
		avgTime, minTime, maxTime, stdDevTime,
		avgMemory, minMemory, maxMemory,
		request.InputSize, sampleCount, request.TestCategory,
		envJSON, request.Notes,
	).Scan(&historyID, &performanceScore, &recordedAt)

	if err != nil {
		log.Println("Error recording performance history:", err)
		http.Error(w, "Error recording performance", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":            historyID,
		"recorded_at":   recordedAt.Format(time.RFC3339),
		"avg_time_ms":   avgTime,
		"avg_memory_mb": avgMemory,
		"sample_count":  sampleCount,
	}

	if performanceScore != nil {
		response["performance_score"] = *performanceScore
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

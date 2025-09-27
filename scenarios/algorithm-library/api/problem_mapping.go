package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// ProblemMapping represents a mapping to a coding platform problem
type ProblemMapping struct {
	ID          string   `json:"id,omitempty"`
	AlgorithmID string   `json:"algorithm_id"`
	Platform    string   `json:"platform"`
	ProblemID   string   `json:"problem_id"`
	ProblemName string   `json:"problem_name"`
	ProblemURL  string   `json:"problem_url,omitempty"`
	Difficulty  string   `json:"difficulty,omitempty"`
	Topics      []string `json:"topics,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

// ProblemMappingResponse represents the response for problem mapping queries
type ProblemMappingResponse struct {
	AlgorithmName string           `json:"algorithm_name"`
	Category      string           `json:"category"`
	Problems      []ProblemMapping `json:"problems"`
	TotalCount    int              `json:"total_count"`
}

// PlatformStats represents statistics for a coding platform
type PlatformStats struct {
	Platform       string `json:"platform"`
	TotalProblems  int    `json:"total_problems"`
	EasyCount      int    `json:"easy_count"`
	MediumCount    int    `json:"medium_count"`
	HardCount      int    `json:"hard_count"`
}

// getAlgorithmProblemsHandler returns all problems mapped to an algorithm
func getAlgorithmProblemsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	algorithmID := vars["id"]
	
	// Support filtering by platform
	platform := r.URL.Query().Get("platform")
	difficulty := r.URL.Query().Get("difficulty")
	
	// First, get algorithm info
	var algoName, algoCategory string
	err := db.QueryRow(`
		SELECT display_name, category 
		FROM algorithms 
		WHERE id = $1 OR name = $1
	`, algorithmID).Scan(&algoName, &algoCategory)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Algorithm not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	// Build query for problems
	query := `
		SELECT pm.id, pm.algorithm_id, pm.platform, pm.problem_id, 
		       pm.problem_name, COALESCE(pm.problem_url, ''), 
		       COALESCE(pm.difficulty, ''), pm.topics, COALESCE(pm.notes, '')
		FROM problem_mappings pm
		WHERE pm.algorithm_id = (SELECT id FROM algorithms WHERE id = $1 OR name = $1)
	`
	
	args := []interface{}{algorithmID}
	argCount := 1
	
	if platform != "" {
		argCount++
		query += fmt.Sprintf(" AND LOWER(pm.platform) = LOWER($%d)", argCount)
		args = append(args, platform)
	}
	
	if difficulty != "" {
		argCount++
		query += fmt.Sprintf(" AND LOWER(pm.difficulty) = LOWER($%d)", argCount)
		args = append(args, difficulty)
	}
	
	query += ` ORDER BY 
		CASE pm.difficulty 
			WHEN 'easy' THEN 1
			WHEN 'medium' THEN 2
			WHEN 'hard' THEN 3
			ELSE 4
		END, pm.platform, pm.problem_name`
	
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	problems := []ProblemMapping{}
	for rows.Next() {
		var pm ProblemMapping
		var topics pq.StringArray
		
		err := rows.Scan(
			&pm.ID, &pm.AlgorithmID, &pm.Platform, &pm.ProblemID,
			&pm.ProblemName, &pm.ProblemURL, &pm.Difficulty,
			&topics, &pm.Notes,
		)
		if err != nil {
			continue
		}
		
		pm.Topics = []string(topics)
		problems = append(problems, pm)
	}
	
	response := ProblemMappingResponse{
		AlgorithmName: algoName,
		Category:      algoCategory,
		Problems:      problems,
		TotalCount:    len(problems),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// searchProblemMappingsHandler searches for problems across all algorithms
func searchProblemMappingsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	platform := r.URL.Query().Get("platform")
	difficulty := r.URL.Query().Get("difficulty")
	
	sqlQuery := `
		SELECT DISTINCT 
			a.id, a.display_name, a.category,
			pm.id, pm.platform, pm.problem_id, pm.problem_name,
			COALESCE(pm.problem_url, ''), COALESCE(pm.difficulty, ''),
			pm.topics, COALESCE(pm.notes, '')
		FROM problem_mappings pm
		JOIN algorithms a ON pm.algorithm_id = a.id
		WHERE 1=1
	`
	
	args := []interface{}{}
	argCount := 0
	
	if query != "" {
		argCount++
		sqlQuery += fmt.Sprintf(` AND (
			LOWER(pm.problem_name) LIKE LOWER($%d) OR
			LOWER(pm.notes) LIKE LOWER($%d) OR
			LOWER(a.display_name) LIKE LOWER($%d)
		)`, argCount, argCount, argCount)
		args = append(args, "%"+query+"%")
	}
	
	if platform != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND LOWER(pm.platform) = LOWER($%d)", argCount)
		args = append(args, platform)
	}
	
	if difficulty != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND LOWER(pm.difficulty) = LOWER($%d)", argCount)
		args = append(args, difficulty)
	}
	
	sqlQuery += " ORDER BY a.display_name, pm.platform, pm.problem_name LIMIT 50"
	
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	type SearchResult struct {
		AlgorithmID   string         `json:"algorithm_id"`
		AlgorithmName string         `json:"algorithm_name"`
		Category      string         `json:"category"`
		Problem       ProblemMapping `json:"problem"`
	}
	
	results := []SearchResult{}
	for rows.Next() {
		var result SearchResult
		var pm ProblemMapping
		var topics pq.StringArray
		
		err := rows.Scan(
			&result.AlgorithmID, &result.AlgorithmName, &result.Category,
			&pm.ID, &pm.Platform, &pm.ProblemID, &pm.ProblemName,
			&pm.ProblemURL, &pm.Difficulty, &topics, &pm.Notes,
		)
		if err != nil {
			continue
		}
		
		pm.Topics = []string(topics)
		pm.AlgorithmID = result.AlgorithmID
		result.Problem = pm
		results = append(results, result)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

// getPlatformStatsHandler returns statistics about problem mappings by platform
func getPlatformStatsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			platform,
			COUNT(*) as total,
			SUM(CASE WHEN difficulty = 'easy' THEN 1 ELSE 0 END) as easy_count,
			SUM(CASE WHEN difficulty = 'medium' THEN 1 ELSE 0 END) as medium_count,
			SUM(CASE WHEN difficulty = 'hard' THEN 1 ELSE 0 END) as hard_count
		FROM problem_mappings
		GROUP BY platform
		ORDER BY total DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	stats := []PlatformStats{}
	for rows.Next() {
		var s PlatformStats
		err := rows.Scan(
			&s.Platform, &s.TotalProblems,
			&s.EasyCount, &s.MediumCount, &s.HardCount,
		)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}
	
	// Also get total algorithm coverage
	var totalAlgorithms, algorithmsWithProblems int
	db.QueryRow("SELECT COUNT(*) FROM algorithms").Scan(&totalAlgorithms)
	db.QueryRow(`
		SELECT COUNT(DISTINCT algorithm_id) 
		FROM problem_mappings
	`).Scan(&algorithmsWithProblems)
	
	response := map[string]interface{}{
		"platforms":               stats,
		"total_algorithms":        totalAlgorithms,
		"algorithms_with_problems": algorithmsWithProblems,
		"coverage_percentage":     float64(algorithmsWithProblems) / float64(totalAlgorithms) * 100,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// addProblemMappingHandler adds a new problem mapping
func addProblemMappingHandler(w http.ResponseWriter, r *http.Request) {
	var pm ProblemMapping
	if err := json.NewDecoder(r.Body).Decode(&pm); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if pm.AlgorithmID == "" || pm.Platform == "" || pm.ProblemID == "" || pm.ProblemName == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Normalize platform name
	pm.Platform = strings.ToLower(pm.Platform)
	
	// Insert the mapping
	var id string
	err := db.QueryRow(`
		INSERT INTO problem_mappings 
		(algorithm_id, platform, problem_id, problem_name, problem_url, difficulty, topics, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (platform, problem_id) 
		DO UPDATE SET 
			problem_name = EXCLUDED.problem_name,
			problem_url = EXCLUDED.problem_url,
			difficulty = EXCLUDED.difficulty,
			topics = EXCLUDED.topics,
			notes = EXCLUDED.notes,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`, pm.AlgorithmID, pm.Platform, pm.ProblemID, pm.ProblemName,
		pm.ProblemURL, pm.Difficulty, pq.Array(pm.Topics), pm.Notes).Scan(&id)
	
	if err != nil {
		http.Error(w, "Failed to add problem mapping", http.StatusInternalServerError)
		return
	}
	
	pm.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pm)
}

// getRecommendedProblemsHandler recommends practice problems based on difficulty
func getRecommendedProblemsHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	difficulty := r.URL.Query().Get("difficulty")
	limit := r.URL.Query().Get("limit")
	
	if limit == "" {
		limit = "10"
	}
	
	query := `
		SELECT DISTINCT
			a.id, a.display_name, a.category,
			pm.platform, pm.problem_id, pm.problem_name,
			pm.problem_url, pm.difficulty, pm.topics
		FROM problem_mappings pm
		JOIN algorithms a ON pm.algorithm_id = a.id
		WHERE pm.problem_url IS NOT NULL AND pm.problem_url != ''
	`
	
	args := []interface{}{}
	argCount := 0
	
	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND a.category = $%d", argCount)
		args = append(args, category)
	}
	
	if difficulty != "" {
		argCount++
		query += fmt.Sprintf(" AND pm.difficulty = $%d", argCount)
		args = append(args, difficulty)
	}
	
	// Order by difficulty and randomize within each difficulty level
	query += ` ORDER BY 
		CASE pm.difficulty 
			WHEN 'easy' THEN 1
			WHEN 'medium' THEN 2
			WHEN 'hard' THEN 3
			ELSE 4
		END, RANDOM()
		LIMIT ` + limit
	
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	type RecommendedProblem struct {
		AlgorithmID   string   `json:"algorithm_id"`
		AlgorithmName string   `json:"algorithm_name"`
		Category      string   `json:"category"`
		Platform      string   `json:"platform"`
		ProblemID     string   `json:"problem_id"`
		ProblemName   string   `json:"problem_name"`
		ProblemURL    string   `json:"problem_url"`
		Difficulty    string   `json:"difficulty"`
		Topics        []string `json:"topics"`
	}
	
	recommendations := []RecommendedProblem{}
	for rows.Next() {
		var rec RecommendedProblem
		var topics pq.StringArray
		
		err := rows.Scan(
			&rec.AlgorithmID, &rec.AlgorithmName, &rec.Category,
			&rec.Platform, &rec.ProblemID, &rec.ProblemName,
			&rec.ProblemURL, &rec.Difficulty, &topics,
		)
		if err != nil {
			continue
		}
		
		rec.Topics = []string(topics)
		recommendations = append(recommendations, rec)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
		"count":           len(recommendations),
	})
}
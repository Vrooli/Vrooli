package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Algorithm struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	DisplayName      string   `json:"display_name"`
	Category         string   `json:"category"`
	Description      string   `json:"description"`
	ComplexityTime   string   `json:"complexity_time"`
	ComplexitySpace  string   `json:"complexity_space"`
	Difficulty       string   `json:"difficulty"`
	Tags             []string `json:"tags"`
	LanguageCount    int      `json:"language_count"`
	TestCaseCount    int      `json:"test_case_count"`
	HasValidatedImpl bool     `json:"has_validated_impl"`
}

type Implementation struct {
	ID               string  `json:"id"`
	AlgorithmID      string  `json:"algorithm_id"`
	Language         string  `json:"language"`
	Code             string  `json:"code"`
	Version          string  `json:"version"`
	IsPrimary        bool    `json:"is_primary"`
	Validated        bool    `json:"validated"`
	ValidationCount  int     `json:"validation_count"`
	PerformanceScore float64 `json:"performance_score"`
}

type ValidationRequest struct {
	AlgorithmID string `json:"algorithm_id"`
	Language    string `json:"language"`
	Code        string `json:"code"`
}

type ValidationResult struct {
	Valid          bool                   `json:"valid"`
	TestResults    []TestResult          `json:"test_results"`
	Performance    map[string]interface{} `json:"performance"`
}

type TestResult struct {
	TestCaseID     string `json:"test_case_id"`
	Passed         bool   `json:"passed"`
	ExecutionTime  int    `json:"execution_time_ms"`
	ActualOutput   string `json:"actual_output"`
	ExpectedOutput string `json:"expected_output"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

var db *sql.DB

func main() {
	// Initialize database connection
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DB", "algorithm_library")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Set up routes
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Algorithm routes
	router.HandleFunc("/api/v1/algorithms/search", searchAlgorithmsHandler).Methods("GET")
	router.HandleFunc("/api/v1/algorithms/{id}", getAlgorithmHandler).Methods("GET")
	router.HandleFunc("/api/v1/algorithms/{id}/implementations", getImplementationsHandler).Methods("GET")
	router.HandleFunc("/api/v1/algorithms/validate", validateAlgorithmHandler).Methods("POST")
	router.HandleFunc("/api/v1/algorithms/categories", getCategoriesHandler).Methods("GET")
	router.HandleFunc("/api/v1/algorithms/stats", getStatsHandler).Methods("GET")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment or use default
	port := getEnv("API_PORT", "3250")
	
	log.Printf("Algorithm Library API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	err := db.Ping()
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	response := map[string]interface{}{
		"status":   status,
		"service":  "algorithm-library-api",
		"database": err == nil,
	}

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

func searchAlgorithmsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	category := r.URL.Query().Get("category")
	complexity := r.URL.Query().Get("complexity")
	language := r.URL.Query().Get("language")
	difficulty := r.URL.Query().Get("difficulty")

	// Build SQL query
	sqlQuery := `
		SELECT 
			a.id, a.name, a.display_name, a.category, a.description,
			a.complexity_time, a.complexity_space, a.difficulty,
			array_to_json(a.tags) as tags,
			COUNT(DISTINCT i.language) as language_count,
			COUNT(DISTINCT tc.id) as test_case_count,
			BOOL_OR(i.validated) as has_validated_impl
		FROM algorithms a
		LEFT JOIN implementations i ON a.id = i.algorithm_id
		LEFT JOIN test_cases tc ON a.id = tc.algorithm_id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if query != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND (LOWER(a.name) LIKE LOWER($%d) OR LOWER(a.description) LIKE LOWER($%d))", argCount, argCount)
		args = append(args, "%"+query+"%")
	}

	if category != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND a.category = $%d", argCount)
		args = append(args, category)
	}

	if complexity != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND (a.complexity_time = $%d OR a.complexity_space = $%d)", argCount, argCount)
		args = append(args, complexity)
	}

	if difficulty != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND a.difficulty = $%d", argCount)
		args = append(args, difficulty)
	}

	sqlQuery += " GROUP BY a.id ORDER BY a.name"

	// Execute query
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Search query error: %v", err)
		return
	}
	defer rows.Close()

	algorithms := []Algorithm{}
	for rows.Next() {
		var algo Algorithm
		var tagsJSON []byte
		
		err := rows.Scan(
			&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
			&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
			&algo.Difficulty, &tagsJSON, &algo.LanguageCount,
			&algo.TestCaseCount, &algo.HasValidatedImpl,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}

		// Parse tags JSON
		if tagsJSON != nil {
			json.Unmarshal(tagsJSON, &algo.Tags)
		}

		// Filter by language if specified
		if language != "" {
			var hasLanguage bool
			err = db.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM implementations 
					WHERE algorithm_id = $1 AND language = $2
				)`, algo.ID, language).Scan(&hasLanguage)
			
			if err != nil || !hasLanguage {
				continue
			}
		}

		algorithms = append(algorithms, algo)
	}

	response := map[string]interface{}{
		"algorithms": algorithms,
		"total":      len(algorithms),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getAlgorithmHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var algo Algorithm
	var tagsJSON []byte

	err := db.QueryRow(`
		SELECT 
			a.id, a.name, a.display_name, a.category, a.description,
			a.complexity_time, a.complexity_space, a.difficulty,
			array_to_json(a.tags) as tags
		FROM algorithms a
		WHERE a.id = $1 OR a.name = $1
	`, id).Scan(
		&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
		&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
		&algo.Difficulty, &tagsJSON,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Algorithm not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Get algorithm error: %v", err)
		return
	}

	// Parse tags
	if tagsJSON != nil {
		json.Unmarshal(tagsJSON, &algo.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(algo)
}

func getImplementationsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	language := r.URL.Query().Get("language")

	// Get algorithm info
	var algo Algorithm
	var tagsJSON []byte
	
	err := db.QueryRow(`
		SELECT 
			a.id, a.name, a.display_name, a.category, a.description,
			a.complexity_time, a.complexity_space, a.difficulty,
			array_to_json(a.tags) as tags
		FROM algorithms a
		WHERE a.id = $1 OR a.name = $1
	`, id).Scan(
		&algo.ID, &algo.Name, &algo.DisplayName, &algo.Category,
		&algo.Description, &algo.ComplexityTime, &algo.ComplexitySpace,
		&algo.Difficulty, &tagsJSON,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Algorithm not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Parse tags
	if tagsJSON != nil {
		json.Unmarshal(tagsJSON, &algo.Tags)
	}

	// Get implementations
	query := `
		SELECT 
			id, algorithm_id, language, code, version, 
			is_primary, validated, validation_count, 
			COALESCE(performance_score, 0)
		FROM implementations
		WHERE algorithm_id = $1
	`
	
	args := []interface{}{algo.ID}
	if language != "" {
		query += " AND language = $2"
		args = append(args, language)
	}
	query += " ORDER BY is_primary DESC, validated DESC, language"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Get implementations error: %v", err)
		return
	}
	defer rows.Close()

	implementations := []Implementation{}
	for rows.Next() {
		var impl Implementation
		err := rows.Scan(
			&impl.ID, &impl.AlgorithmID, &impl.Language, &impl.Code,
			&impl.Version, &impl.IsPrimary, &impl.Validated,
			&impl.ValidationCount, &impl.PerformanceScore,
		)
		if err != nil {
			log.Printf("Implementation scan error: %v", err)
			continue
		}
		implementations = append(implementations, impl)
	}

	response := map[string]interface{}{
		"algorithm":       algo,
		"implementations": implementations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateAlgorithmHandler(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Call Judge0 through n8n workflow for actual validation
	// For now, return mock response
	result := ValidationResult{
		Valid: true,
		TestResults: []TestResult{
			{
				TestCaseID:     "test1",
				Passed:         true,
				ExecutionTime:  45,
				ActualOutput:   "[1, 2, 3, 4, 5]",
				ExpectedOutput: "[1, 2, 3, 4, 5]",
			},
		},
		Performance: map[string]interface{}{
			"execution_time_ms": 45,
			"memory_used_kb":    1024,
		},
	}

	// Log usage
	_, err := db.Exec(`
		INSERT INTO usage_stats (algorithm_id, action, caller, metadata)
		VALUES ($1, 'validate', 'api', $2)
	`, req.AlgorithmID, map[string]string{"language": req.Language})
	
	if err != nil {
		log.Printf("Failed to log usage: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT DISTINCT category, COUNT(*) as count
		FROM algorithms
		GROUP BY category
		ORDER BY category
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := []map[string]interface{}{}
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		categories = append(categories, map[string]interface{}{
			"name":  category,
			"count": count,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func getStatsHandler(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalAlgorithms      int `json:"total_algorithms"`
		TotalImplementations int `json:"total_implementations"`
		TotalTestCases       int `json:"total_test_cases"`
		ValidatedCount       int `json:"validated_implementations"`
	}

	// Get statistics
	db.QueryRow("SELECT COUNT(*) FROM algorithms").Scan(&stats.TotalAlgorithms)
	db.QueryRow("SELECT COUNT(*) FROM implementations").Scan(&stats.TotalImplementations)
	db.QueryRow("SELECT COUNT(*) FROM test_cases").Scan(&stats.TotalTestCases)
	db.QueryRow("SELECT COUNT(*) FROM implementations WHERE validated = true").Scan(&stats.ValidatedCount)

	// Get language distribution
	languageRows, _ := db.Query(`
		SELECT language, COUNT(*) as count
		FROM implementations
		GROUP BY language
		ORDER BY count DESC
	`)
	defer languageRows.Close()

	languages := map[string]int{}
	for languageRows.Next() {
		var lang string
		var count int
		if err := languageRows.Scan(&lang, &count); err == nil {
			languages[lang] = count
		}
	}

	response := map[string]interface{}{
		"statistics": stats,
		"languages":  languages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
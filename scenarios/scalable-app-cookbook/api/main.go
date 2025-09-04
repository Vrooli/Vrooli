package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Pattern struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Chapter        string   `json:"chapter"`
	Section        string   `json:"section"`
	MaturityLevel  string   `json:"maturity_level"`
	Tags           []string `json:"tags"`
	WhatAndWhy     string   `json:"what_and_why"`
	WhenToUse      string   `json:"when_to_use"`
	Tradeoffs      string   `json:"tradeoffs"`
	RefPatterns    []string `json:"reference_patterns"`
	FailureModes   string   `json:"failure_modes"`
	CostLevers     string   `json:"cost_levers"`
	RecipeCount    int      `json:"recipe_count"`
	ImplCount      int      `json:"implementation_count"`
	Languages      []string `json:"languages"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

type Recipe struct {
	ID                string                 `json:"id"`
	PatternID         string                 `json:"pattern_id"`
	Title             string                 `json:"title"`
	Type              string                 `json:"type"` // greenfield, brownfield, migration
	Prerequisites     []string               `json:"prerequisites"`
	Steps             []map[string]interface{} `json:"steps"`
	ConfigSnippets    map[string]interface{} `json:"config_snippets"`
	ValidationChecks  []string               `json:"validation_checks"`
	Artifacts         []string               `json:"artifacts"`
	Metrics           []string               `json:"metrics"`
	Rollbacks         []string               `json:"rollbacks"`
	Prompts           []string               `json:"prompts"`
	TimeoutSec        int                    `json:"timeout_sec"`
}

type Implementation struct {
	ID           string   `json:"id"`
	RecipeID     string   `json:"recipe_id"`
	Language     string   `json:"language"`
	Code         string   `json:"code"`
	FilePath     string   `json:"file_path"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	TestCode     string   `json:"test_code"`
}

type GenerationRequest struct {
	RecipeID       string                 `json:"recipe_id"`
	Language       string                 `json:"language"`
	Parameters     map[string]interface{} `json:"parameters"`
	TargetPlatform string                 `json:"target_platform"`
}

type GenerationResult struct {
	GeneratedCode      string   `json:"generated_code"`
	FileStructure      map[string]interface{} `json:"file_structure"`
	Dependencies       []string `json:"dependencies"`
	SetupInstructions  []string `json:"setup_instructions"`
}

var db *sql.DB

func main() {
	// Initialize database connection
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5433")
	dbUser := getEnv("POSTGRES_USER", "vrooli")
	dbPassword := getEnv("POSTGRES_PASSWORD", "")
	dbName := getEnv("POSTGRES_DB", "vrooli")

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

	// Pattern routes
	router.HandleFunc("/api/v1/patterns/search", searchPatternsHandler).Methods("GET")
	router.HandleFunc("/api/v1/patterns/{id}", getPatternHandler).Methods("GET")
	router.HandleFunc("/api/v1/patterns/{id}/recipes", getRecipesHandler).Methods("GET")
	router.HandleFunc("/api/v1/patterns/chapters", getChaptersHandler).Methods("GET")
	router.HandleFunc("/api/v1/patterns/stats", getStatsHandler).Methods("GET")

	// Recipe routes
	router.HandleFunc("/api/v1/recipes/{id}", getRecipeHandler).Methods("GET")
	router.HandleFunc("/api/v1/recipes/generate", generateCodeHandler).Methods("POST")

	// Implementation routes
	router.HandleFunc("/api/v1/implementations", getImplementationsHandler).Methods("GET")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment or use default
	port := getEnv("API_PORT", "3300")
	
	log.Printf("Scalable App Cookbook API starting on port %s", port)
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
		"service":  "scalable-app-cookbook-api",
		"database": err == nil,
	}

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

func searchPatternsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	chapter := r.URL.Query().Get("chapter")
	section := r.URL.Query().Get("section")
	maturityLevel := r.URL.Query().Get("maturity_level")
	tags := r.URL.Query().Get("tags")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Parse limit and offset
	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build SQL query using the pattern_summary view
	sqlQuery := `
		SELECT 
			id, title, chapter, section, maturity_level,
			array_to_json(tags) as tags,
			recipe_count, implementation_count, languages,
			created_at, updated_at
		FROM scalable_app_cookbook.pattern_summary
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if query != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND (LOWER(title) LIKE LOWER($%d) OR LOWER(section) LIKE LOWER($%d))", argCount, argCount)
		args = append(args, "%"+query+"%")
	}

	if chapter != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND chapter = $%d", argCount)
		args = append(args, chapter)
	}

	if section != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND section = $%d", argCount)
		args = append(args, section)
	}

	if maturityLevel != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND maturity_level = $%d", argCount)
		args = append(args, maturityLevel)
	}

	if tags != "" {
		argCount++
		tagArray := strings.Split(tags, ",")
		sqlQuery += fmt.Sprintf(" AND tags && $%d", argCount)
		args = append(args, tagArray)
	}

	// Get total count for pagination
	countQuery := strings.Replace(sqlQuery, `SELECT 
			id, title, chapter, section, maturity_level,
			array_to_json(tags) as tags,
			recipe_count, implementation_count, languages,
			created_at, updated_at
		FROM scalable_app_cookbook.pattern_summary`, "SELECT COUNT(*) FROM pattern_summary", 1)

	var total int
	db.QueryRow(countQuery, args...).Scan(&total)

	// Add ordering, limit and offset
	sqlQuery += " ORDER BY chapter, section, title"
	argCount++
	sqlQuery += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit)
	
	argCount++
	sqlQuery += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	// Execute query
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Search query error: %v", err)
		return
	}
	defer rows.Close()

	patterns := []Pattern{}
	for rows.Next() {
		var pattern Pattern
		var tagsJSON []byte
		var languagesJSON []byte
		
		err := rows.Scan(
			&pattern.ID, &pattern.Title, &pattern.Chapter, &pattern.Section,
			&pattern.MaturityLevel, &tagsJSON, &pattern.RecipeCount,
			&pattern.ImplCount, &languagesJSON, &pattern.CreatedAt, &pattern.UpdatedAt,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}

		// Parse JSON arrays
		if tagsJSON != nil {
			json.Unmarshal(tagsJSON, &pattern.Tags)
		}
		if languagesJSON != nil {
			json.Unmarshal(languagesJSON, &pattern.Languages)
		}

		patterns = append(patterns, pattern)
	}

	// Get facets for filtering
	facets := getFacets()

	response := map[string]interface{}{
		"patterns": patterns,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"facets":   facets,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getPatternHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var pattern Pattern
	var tagsJSON []byte
	var refPatternsJSON []byte

	err := db.QueryRow(`
		SELECT 
			id, title, chapter, section, maturity_level,
			array_to_json(tags) as tags, what_and_why, when_to_use,
			tradeoffs, array_to_json(reference_patterns) as ref_patterns,
			failure_modes, cost_levers, created_at, updated_at
		FROM scalable_app_cookbook.patterns
		WHERE id = $1 OR LOWER(title) = LOWER($1)
	`, id).Scan(
		&pattern.ID, &pattern.Title, &pattern.Chapter, &pattern.Section,
		&pattern.MaturityLevel, &tagsJSON, &pattern.WhatAndWhy,
		&pattern.WhenToUse, &pattern.Tradeoffs, &refPatternsJSON,
		&pattern.FailureModes, &pattern.CostLevers, &pattern.CreatedAt, &pattern.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Pattern not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Get pattern error: %v", err)
		return
	}

	// Parse JSON arrays
	if tagsJSON != nil {
		json.Unmarshal(tagsJSON, &pattern.Tags)
	}
	if refPatternsJSON != nil {
		json.Unmarshal(refPatternsJSON, &pattern.RefPatterns)
	}

	// Log usage
	db.Exec(`
		INSERT INTO scalable_app_cookbook.pattern_usage (pattern_id, access_type, success, metadata)
		VALUES ($1, 'retrieve', true, $2)
	`, pattern.ID, map[string]string{"user_agent": r.UserAgent()})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pattern)
}

func getRecipesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patternID := vars["id"]
	recipeType := r.URL.Query().Get("type")

	// Get pattern info first
	var pattern Pattern
	var tagsJSON []byte
	
	err := db.QueryRow(`
		SELECT id, title, chapter, section, maturity_level, array_to_json(tags) as tags
		FROM scalable_app_cookbook.patterns
		WHERE id = $1 OR LOWER(title) = LOWER($1)
	`, patternID).Scan(
		&pattern.ID, &pattern.Title, &pattern.Chapter, &pattern.Section,
		&pattern.MaturityLevel, &tagsJSON,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Pattern not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if tagsJSON != nil {
		json.Unmarshal(tagsJSON, &pattern.Tags)
	}

	// Get recipes
	query := `
		SELECT 
			id, pattern_id, title, type, prerequisites,
			steps, config_snippets, validation_checks,
			artifacts, metrics, rollbacks, prompts, timeout_sec
		FROM scalable_app_cookbook.recipes
		WHERE pattern_id = $1
	`
	
	args := []interface{}{pattern.ID}
	if recipeType != "" {
		query += " AND type = $2"
		args = append(args, recipeType)
	}
	query += " ORDER BY type, title"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Get recipes error: %v", err)
		return
	}
	defer rows.Close()

	recipes := []Recipe{}
	for rows.Next() {
		var recipe Recipe
		var prereqJSON, stepsJSON, configJSON, checksJSON []byte
		var artifactsJSON, metricsJSON, rollbacksJSON, promptsJSON []byte
		
		err := rows.Scan(
			&recipe.ID, &recipe.PatternID, &recipe.Title, &recipe.Type,
			&prereqJSON, &stepsJSON, &configJSON, &checksJSON,
			&artifactsJSON, &metricsJSON, &rollbacksJSON, &promptsJSON,
			&recipe.TimeoutSec,
		)
		if err != nil {
			log.Printf("Recipe scan error: %v", err)
			continue
		}

		// Parse JSON fields
		if prereqJSON != nil {
			json.Unmarshal(prereqJSON, &recipe.Prerequisites)
		}
		if stepsJSON != nil {
			json.Unmarshal(stepsJSON, &recipe.Steps)
		}
		if configJSON != nil {
			json.Unmarshal(configJSON, &recipe.ConfigSnippets)
		}
		if checksJSON != nil {
			json.Unmarshal(checksJSON, &recipe.ValidationChecks)
		}
		if artifactsJSON != nil {
			json.Unmarshal(artifactsJSON, &recipe.Artifacts)
		}
		if metricsJSON != nil {
			json.Unmarshal(metricsJSON, &recipe.Metrics)
		}
		if rollbacksJSON != nil {
			json.Unmarshal(rollbacksJSON, &recipe.Rollbacks)
		}
		if promptsJSON != nil {
			json.Unmarshal(promptsJSON, &recipe.Prompts)
		}

		recipes = append(recipes, recipe)
	}

	response := map[string]interface{}{
		"pattern": pattern,
		"recipes": recipes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var recipe Recipe
	var prereqJSON, stepsJSON, configJSON, checksJSON []byte
	var artifactsJSON, metricsJSON, rollbacksJSON, promptsJSON []byte

	err := db.QueryRow(`
		SELECT 
			id, pattern_id, title, type, prerequisites,
			steps, config_snippets, validation_checks,
			artifacts, metrics, rollbacks, prompts, timeout_sec
		FROM scalable_app_cookbook.recipes
		WHERE id = $1 OR LOWER(title) = LOWER($1)
	`, id).Scan(
		&recipe.ID, &recipe.PatternID, &recipe.Title, &recipe.Type,
		&prereqJSON, &stepsJSON, &configJSON, &checksJSON,
		&artifactsJSON, &metricsJSON, &rollbacksJSON, &promptsJSON,
		&recipe.TimeoutSec,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Get recipe error: %v", err)
		return
	}

	// Parse JSON fields
	if prereqJSON != nil {
		json.Unmarshal(prereqJSON, &recipe.Prerequisites)
	}
	if stepsJSON != nil {
		json.Unmarshal(stepsJSON, &recipe.Steps)
	}
	if configJSON != nil {
		json.Unmarshal(configJSON, &recipe.ConfigSnippets)
	}
	if checksJSON != nil {
		json.Unmarshal(checksJSON, &recipe.ValidationChecks)
	}
	if artifactsJSON != nil {
		json.Unmarshal(artifactsJSON, &recipe.Artifacts)
	}
	if metricsJSON != nil {
		json.Unmarshal(metricsJSON, &recipe.Metrics)
	}
	if rollbacksJSON != nil {
		json.Unmarshal(rollbacksJSON, &recipe.Rollbacks)
	}
	if promptsJSON != nil {
		json.Unmarshal(promptsJSON, &recipe.Prompts)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func generateCodeHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get recipe and implementations
	var recipe Recipe
	var stepsJSON, configJSON []byte
	
	err := db.QueryRow(`
		SELECT id, title, steps, config_snippets
		FROM scalable_app_cookbook.recipes
		WHERE id = $1 OR LOWER(title) = LOWER($1)
	`, req.RecipeID).Scan(&recipe.ID, &recipe.Title, &stepsJSON, &configJSON)

	if err == sql.ErrNoRows {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Parse JSON fields
	if stepsJSON != nil {
		json.Unmarshal(stepsJSON, &recipe.Steps)
	}
	if configJSON != nil {
		json.Unmarshal(configJSON, &recipe.ConfigSnippets)
	}

	// Get implementation for the requested language
	var impl Implementation
	var depsJSON []byte
	
	err = db.QueryRow(`
		SELECT code, file_path, description, dependencies
		FROM scalable_app_cookbook.implementations
		WHERE recipe_id = $1 AND language = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, recipe.ID, req.Language).Scan(&impl.Code, &impl.FilePath, &impl.Description, &depsJSON)

	if err == sql.ErrNoRows {
		http.Error(w, fmt.Sprintf("No implementation found for language: %s", req.Language), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Get implementation error: %v", err)
		return
	}

	if depsJSON != nil {
		json.Unmarshal(depsJSON, &impl.Dependencies)
	}

	// Generate result (basic template replacement for now)
	result := GenerationResult{
		GeneratedCode: impl.Code,
		FileStructure: map[string]interface{}{
			"main_file": impl.FilePath,
			"dependencies": impl.Dependencies,
		},
		Dependencies: impl.Dependencies,
		SetupInstructions: []string{
			fmt.Sprintf("Create file: %s", impl.FilePath),
			"Install dependencies as specified",
			"Run tests to verify implementation",
		},
	}

	// Log usage
	db.Exec(`
		INSERT INTO scalable_app_cookbook.pattern_usage (pattern_id, access_type, success, metadata)
		VALUES ((SELECT pattern_id FROM recipes WHERE id = $1), 'generate', true, $2)
	`, recipe.ID, map[string]string{
		"language": req.Language,
		"recipe": recipe.Title,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getImplementationsHandler(w http.ResponseWriter, r *http.Request) {
	recipeID := r.URL.Query().Get("recipe_id")
	language := r.URL.Query().Get("language")

	if recipeID == "" {
		http.Error(w, "recipe_id parameter required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			id, recipe_id, language, code, file_path,
			description, dependencies, test_code
		FROM scalable_app_cookbook.implementations
		WHERE recipe_id = $1
	`
	
	args := []interface{}{recipeID}
	if language != "" {
		query += " AND language = $2"
		args = append(args, language)
	}
	query += " ORDER BY language"

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
		var depsJSON []byte
		
		err := rows.Scan(
			&impl.ID, &impl.RecipeID, &impl.Language, &impl.Code,
			&impl.FilePath, &impl.Description, &depsJSON, &impl.TestCode,
		)
		if err != nil {
			log.Printf("Implementation scan error: %v", err)
			continue
		}

		if depsJSON != nil {
			json.Unmarshal(depsJSON, &impl.Dependencies)
		}

		implementations = append(implementations, impl)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(implementations)
}

func getChaptersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT DISTINCT chapter, COUNT(*) as pattern_count
		FROM scalable_app_cookbook.patterns
		GROUP BY chapter
		ORDER BY chapter
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	chapters := []map[string]interface{}{}
	for rows.Next() {
		var chapter string
		var count int
		if err := rows.Scan(&chapter, &count); err != nil {
			continue
		}
		chapters = append(chapters, map[string]interface{}{
			"name":          chapter,
			"pattern_count": count,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chapters)
}

func getStatsHandler(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalPatterns        int `json:"total_patterns"`
		TotalRecipes         int `json:"total_recipes"`
		TotalImplementations int `json:"total_implementations"`
		TotalChapters        int `json:"total_chapters"`
	}

	// Get statistics
	db.QueryRow("SELECT COUNT(*) FROM patterns").Scan(&stats.TotalPatterns)
	db.QueryRow("SELECT COUNT(*) FROM recipes").Scan(&stats.TotalRecipes)
	db.QueryRow("SELECT COUNT(*) FROM implementations").Scan(&stats.TotalImplementations)
	db.QueryRow("SELECT COUNT(DISTINCT chapter) FROM patterns").Scan(&stats.TotalChapters)

	// Get maturity level distribution
	maturityRows, _ := db.Query(`
		SELECT maturity_level, COUNT(*) as count
		FROM scalable_app_cookbook.patterns
		GROUP BY maturity_level
		ORDER BY maturity_level
	`)
	defer maturityRows.Close()

	maturityLevels := map[string]int{}
	for maturityRows.Next() {
		var level string
		var count int
		if err := maturityRows.Scan(&level, &count); err == nil {
			maturityLevels[level] = count
		}
	}

	// Get language distribution
	langRows, _ := db.Query(`
		SELECT language, COUNT(*) as count
		FROM scalable_app_cookbook.implementations
		GROUP BY language
		ORDER BY count DESC
	`)
	defer langRows.Close()

	languages := map[string]int{}
	for langRows.Next() {
		var lang string
		var count int
		if err := langRows.Scan(&lang, &count); err == nil {
			languages[lang] = count
		}
	}

	response := map[string]interface{}{
		"statistics":      stats,
		"maturity_levels": maturityLevels,
		"languages":       languages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getFacets() map[string]interface{} {
	facets := map[string]interface{}{}

	// Get chapters
	chapters := []string{}
	rows, _ := db.Query("SELECT DISTINCT chapter FROM patterns ORDER BY chapter")
	for rows.Next() {
		var chapter string
		if err := rows.Scan(&chapter); err == nil {
			chapters = append(chapters, chapter)
		}
	}
	rows.Close()
	facets["chapters"] = chapters

	// Get maturity levels
	levels := []string{}
	rows, _ = db.Query("SELECT DISTINCT maturity_level FROM patterns ORDER BY maturity_level")
	for rows.Next() {
		var level string
		if err := rows.Scan(&level); err == nil {
			levels = append(levels, level)
		}
	}
	rows.Close()
	facets["maturity_levels"] = levels

	// Get all unique tags
	tags := []string{}
	rows, _ = db.Query("SELECT DISTINCT unnest(tags) as tag FROM patterns ORDER BY tag")
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			tags = append(tags, tag)
		}
	}
	rows.Close()
	facets["tags"] = tags

	return facets
}
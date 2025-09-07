package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	
	"scenario-generator-api/pipeline"
)

type Scenario struct {
	ID               string            `json:"id" db:"id"`
	Name             string            `json:"name" db:"name"`
	Description      string            `json:"description" db:"description"`
	Prompt           string            `json:"prompt" db:"prompt"`
	FilesGenerated   map[string]string `json:"files_generated" db:"files_generated"`
	ResourcesUsed    []string          `json:"resources_used" db:"resources_used"`
	Status           string            `json:"status" db:"status"`
	GenerationID     *string           `json:"generation_id" db:"generation_id"`
	Complexity       string            `json:"complexity" db:"complexity"`
	Category         string            `json:"category" db:"category"`
	EstimatedRevenue int               `json:"estimated_revenue" db:"estimated_revenue"`
	ClaudePrompt     *string           `json:"claude_prompt" db:"claude_prompt"`
	ClaudeResponse   *string           `json:"claude_response" db:"claude_response"`
	GenerationError  *string           `json:"generation_error" db:"generation_error"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time        `json:"completed_at" db:"completed_at"`
}

type ScenarioTemplate struct {
	ID                   string    `json:"id" db:"id"`
	Name                 string    `json:"name" db:"name"`
	Description          string    `json:"description" db:"description"`
	Category             string    `json:"category" db:"category"`
	PromptTemplate       string    `json:"prompt_template" db:"prompt_template"`
	ResourcesSuggested   []string  `json:"resources_suggested" db:"resources_suggested"`
	ComplexityLevel      string    `json:"complexity_level" db:"complexity_level"`
	EstimatedRevenueMin  int       `json:"estimated_revenue_min" db:"estimated_revenue_min"`
	EstimatedRevenueMax  int       `json:"estimated_revenue_max" db:"estimated_revenue_max"`
	UsageCount           int       `json:"usage_count" db:"usage_count"`
	SuccessRate          *float64  `json:"success_rate" db:"success_rate"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
	IsActive             bool      `json:"is_active" db:"is_active"`
}

type GenerationRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	Complexity  string   `json:"complexity"`
	Category    string   `json:"category"`
	Resources   []string `json:"resources"`
}

type GenerationLog struct {
	ID               string     `json:"id" db:"id"`
	ScenarioID       string     `json:"scenario_id" db:"scenario_id"`
	Step             string     `json:"step" db:"step"`
	Prompt           string     `json:"prompt" db:"prompt"`
	Response         *string    `json:"response" db:"response"`
	Success          bool       `json:"success" db:"success"`
	ErrorMessage     *string    `json:"error_message" db:"error_message"`
	StartedAt        time.Time  `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at" db:"completed_at"`
	DurationSeconds  *int       `json:"duration_seconds" db:"duration_seconds"`
}

type APIServer struct {
	db         *sql.DB
	pipeline   *pipeline.Pipeline
}

func main() {
	port := getEnv("API_PORT", getEnv("PORT", ""))

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresHost := getEnv("POSTGRES_HOST", "localhost")
		postgresPort := getEnv("POSTGRES_PORT", "5432")
		postgresUser := getEnv("POSTGRES_USER", "postgres")
		postgresPassword := getEnv("POSTGRES_PASSWORD", "postgres")
		postgresDB := getEnv("POSTGRES_DB", "postgres")
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", 
			postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Initialize generation pipeline
	generationPipeline := pipeline.NewPipeline(db)

	server := &APIServer{
		db:       db,
		pipeline: generationPipeline,
	}

	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health check
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	
	// Scenarios endpoints
	api.HandleFunc("/scenarios", server.getScenarios).Methods("GET")
	api.HandleFunc("/scenarios", server.createScenario).Methods("POST")
	api.HandleFunc("/scenarios/{id}", server.getScenario).Methods("GET")
	api.HandleFunc("/scenarios/{id}", server.updateScenario).Methods("PUT")
	api.HandleFunc("/scenarios/{id}", server.deleteScenario).Methods("DELETE")
	
	// Generation endpoints
	api.HandleFunc("/generate", server.generateScenario).Methods("POST")
	api.HandleFunc("/generate/status/{id}", server.getGenerationStatus).Methods("GET")
	
	// Templates endpoints
	api.HandleFunc("/templates", server.getTemplates).Methods("GET")
	api.HandleFunc("/templates/{id}", server.getTemplate).Methods("GET")
	
	// Search and filtering
	api.HandleFunc("/search/scenarios", server.searchScenarios).Methods("GET")
	api.HandleFunc("/featured", server.getFeaturedScenarios).Methods("GET")
	
	// Logs endpoints
	api.HandleFunc("/scenarios/{id}/logs", server.getScenarioLogs).Methods("GET")
	
	// Backlog endpoints
	api.HandleFunc("/backlog", server.handleGetBacklog).Methods("GET")
	api.HandleFunc("/backlog", server.handleCreateBacklogItem).Methods("POST")
	api.HandleFunc("/backlog/{id}", server.handleGetBacklogItem).Methods("GET")
	api.HandleFunc("/backlog/{id}", server.handleUpdateBacklogItem).Methods("PUT")
	api.HandleFunc("/backlog/{id}", server.handleDeleteBacklogItem).Methods("DELETE")
	api.HandleFunc("/backlog/{id}/generate", server.handleGenerateFromBacklog).Methods("POST")
	api.HandleFunc("/backlog/{id}/move", server.handleMoveBacklogItem).Methods("POST")
	
	// Backlog statistics and metadata endpoints
	api.HandleFunc("/backlog/stats", server.getBacklogStats).Methods("GET")
	api.HandleFunc("/backlog/metadata", server.getBacklogMetadata).Methods("GET")
	
	// Validation endpoints
	api.HandleFunc("/validate/scenario", server.validateScenario).Methods("POST")
	api.HandleFunc("/validate/backlog-item", server.validateBacklogItem).Methods("POST")

	// Start backlog file watcher
	watcher := NewBacklogWatcher(server)
	watcher.Start()
	defer watcher.Stop()

	log.Printf("🚀 Scenario Generator API starting on port %s", port)
	log.Printf("🗄️  Database: %s", postgresURL)
	log.Printf("🤖 Pipeline: Active (using resource-claude-code)")
	log.Printf("📁 Backlog watcher: Active")

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}


func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]interface{}{
			"database": s.checkDatabase(),
			"pipeline": "active",
			"vrooli_resource": "claude",
		},
	}

	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) checkDatabase() string {
	if err := s.db.Ping(); err != nil {
		return "unhealthy"
	}
	return "healthy"
}

func (s *APIServer) getScenarios(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, prompt, files_generated, resources_used, 
		       status, generation_id, complexity, category, estimated_revenue,
		       claude_prompt, claude_response, generation_error,
		       created_at, updated_at, completed_at
		FROM scenarios 
		ORDER BY created_at DESC 
		LIMIT 50`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scenarios []Scenario
	for rows.Next() {
		var scenario Scenario
		var filesJSON []byte
		var resourcesJSON []byte
		
		err := rows.Scan(
			&scenario.ID, &scenario.Name, &scenario.Description, &scenario.Prompt,
			&filesJSON, &resourcesJSON, &scenario.Status, &scenario.GenerationID,
			&scenario.Complexity, &scenario.Category, &scenario.EstimatedRevenue,
			&scenario.ClaudePrompt, &scenario.ClaudeResponse, &scenario.GenerationError,
			&scenario.CreatedAt, &scenario.UpdatedAt, &scenario.CompletedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse JSON arrays
		if len(filesJSON) > 0 {
			json.Unmarshal(filesJSON, &scenario.FilesGenerated)
		}
		if len(resourcesJSON) > 0 {
			json.Unmarshal(resourcesJSON, &scenario.ResourcesUsed)
		}

		scenarios = append(scenarios, scenario)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

func (s *APIServer) createScenario(w http.ResponseWriter, r *http.Request) {
	var scenario Scenario
	if err := json.NewDecoder(r.Body).Decode(&scenario); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate UUID for new scenario
	scenario.ID = uuid.New().String()
	scenario.CreatedAt = time.Now()
	scenario.UpdatedAt = time.Now()
	scenario.Status = "requested"

	filesJSON, _ := json.Marshal(scenario.FilesGenerated)
	resourcesJSON, _ := json.Marshal(scenario.ResourcesUsed)

	query := `
		INSERT INTO scenarios (id, name, description, prompt, files_generated, 
		                      resources_used, status, complexity, category, 
		                      estimated_revenue, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	err := s.db.QueryRow(query,
		scenario.ID, scenario.Name, scenario.Description, scenario.Prompt,
		filesJSON, resourcesJSON, scenario.Status, scenario.Complexity,
		scenario.Category, scenario.EstimatedRevenue, scenario.CreatedAt, scenario.UpdatedAt,
	).Scan(&scenario.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(scenario)
}

func (s *APIServer) getScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]

	query := `
		SELECT id, name, description, prompt, files_generated, resources_used, 
		       status, generation_id, complexity, category, estimated_revenue,
		       claude_prompt, claude_response, generation_error,
		       created_at, updated_at, completed_at
		FROM scenarios 
		WHERE id = $1`

	var scenario Scenario
	var filesJSON []byte
	var resourcesJSON []byte

	err := s.db.QueryRow(query, scenarioID).Scan(
		&scenario.ID, &scenario.Name, &scenario.Description, &scenario.Prompt,
		&filesJSON, &resourcesJSON, &scenario.Status, &scenario.GenerationID,
		&scenario.Complexity, &scenario.Category, &scenario.EstimatedRevenue,
		&scenario.ClaudePrompt, &scenario.ClaudeResponse, &scenario.GenerationError,
		&scenario.CreatedAt, &scenario.UpdatedAt, &scenario.CompletedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse JSON arrays
	if len(filesJSON) > 0 {
		json.Unmarshal(filesJSON, &scenario.FilesGenerated)
	}
	if len(resourcesJSON) > 0 {
		json.Unmarshal(resourcesJSON, &scenario.ResourcesUsed)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenario)
}

func (s *APIServer) generateScenario(w http.ResponseWriter, r *http.Request) {
	var req GenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to pipeline request
	pipelineReq := pipeline.GenerationRequest{
		Name:        req.Name,
		Description: req.Description,
		Prompt:      req.Prompt,
		Complexity:  req.Complexity,
		Category:    req.Category,
		Resources:   req.Resources,
		Iterations: pipeline.IterationLimits{
			Planning:       3,
			Implementation: 2,
			Validation:     5,
		},
	}

	// Start async generation
	scenarioID, err := s.pipeline.GenerateAsync(pipelineReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start generation: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario_id":    scenarioID,
		"status":         "generating",
		"prompt":         req.Prompt,
		"estimated_time": "3-8 minutes",
		"message":        "Scenario generation started using AI pipeline",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}









func (s *APIServer) getTemplates(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, category, prompt_template, resources_suggested,
		       complexity_level, estimated_revenue_min, estimated_revenue_max,
		       usage_count, success_rate, created_at, updated_at, is_active
		FROM scenario_templates 
		WHERE is_active = true
		ORDER BY usage_count DESC, name ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var templates []ScenarioTemplate
	for rows.Next() {
		var template ScenarioTemplate
		var resourcesJSON []byte
		
		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &template.Category,
			&template.PromptTemplate, &resourcesJSON, &template.ComplexityLevel,
			&template.EstimatedRevenueMin, &template.EstimatedRevenueMax,
			&template.UsageCount, &template.SuccessRate, &template.CreatedAt,
			&template.UpdatedAt, &template.IsActive,
		)
		if err != nil {
			continue
		}

		// Parse resources JSON array
		if len(resourcesJSON) > 0 {
			json.Unmarshal(resourcesJSON, &template.ResourcesSuggested)
		}

		templates = append(templates, template)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (s *APIServer) searchScenarios(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		http.Error(w, "Search term required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, name, description, prompt, files_generated, resources_used, 
		       status, generation_id, complexity, category, estimated_revenue,
		       claude_prompt, claude_response, generation_error,
		       created_at, updated_at, completed_at
		FROM scenarios 
		WHERE (
			name ILIKE $1 OR 
			description ILIKE $1 OR 
			prompt ILIKE $1 OR
			category ILIKE $1
		)
		ORDER BY 
			CASE WHEN name ILIKE $1 THEN 1 ELSE 2 END,
			created_at DESC
		LIMIT 20`

	searchPattern := "%" + searchTerm + "%"
	rows, err := s.db.Query(query, searchPattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scenarios []Scenario
	for rows.Next() {
		var scenario Scenario
		var filesJSON []byte
		var resourcesJSON []byte
		
		err := rows.Scan(
			&scenario.ID, &scenario.Name, &scenario.Description, &scenario.Prompt,
			&filesJSON, &resourcesJSON, &scenario.Status, &scenario.GenerationID,
			&scenario.Complexity, &scenario.Category, &scenario.EstimatedRevenue,
			&scenario.ClaudePrompt, &scenario.ClaudeResponse, &scenario.GenerationError,
			&scenario.CreatedAt, &scenario.UpdatedAt, &scenario.CompletedAt,
		)
		if err != nil {
			continue
		}

		// Parse JSON arrays
		if len(filesJSON) > 0 {
			json.Unmarshal(filesJSON, &scenario.FilesGenerated)
		}
		if len(resourcesJSON) > 0 {
			json.Unmarshal(resourcesJSON, &scenario.ResourcesUsed)
		}

		scenarios = append(scenarios, scenario)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

// updateScenario updates an existing scenario's metadata
func (s *APIServer) updateScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]
	
	// Parse request body
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Build dynamic UPDATE query based on provided fields
	var setClause []string
	var args []interface{}
	argCount := 1
	
	// Allowed fields for update
	allowedFields := map[string]bool{
		"name": true, "description": true, "prompt": true,
		"complexity": true, "category": true, "estimated_revenue": true,
		"status": true, "generation_error": true, "notes": true,
	}
	
	for field, value := range updates {
		if !allowedFields[field] {
			continue // Skip non-allowed fields
		}
		setClause = append(setClause, fmt.Sprintf("%s = $%d", field, argCount))
		args = append(args, value)
		argCount++
	}
	
	if len(setClause) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}
	
	// Add updated_at timestamp
	setClause = append(setClause, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++
	
	// Add scenario ID as last parameter
	args = append(args, scenarioID)
	
	// Execute update query
	query := fmt.Sprintf(`
		UPDATE scenarios 
		SET %s 
		WHERE id = $%d
		RETURNING id, name, description, prompt, status, complexity, category, 
		          estimated_revenue, created_at, updated_at
	`, strings.Join(setClause, ", "), argCount)
	
	var scenario Scenario
	err := s.db.QueryRow(query, args...).Scan(
		&scenario.ID, &scenario.Name, &scenario.Description, &scenario.Prompt,
		&scenario.Status, &scenario.Complexity, &scenario.Category,
		&scenario.EstimatedRevenue, &scenario.CreatedAt, &scenario.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to update scenario: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Log the update
	log.Printf("Updated scenario %s: %v", scenarioID, updates)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenario)
}

// deleteScenario removes a scenario and its associated data
func (s *APIServer) deleteScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]
	
	// Start transaction for cascade delete
	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	
	// Check if scenario exists
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM scenarios WHERE id = $1)", scenarioID).Scan(&exists)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	if !exists {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}
	
	// Delete associated generation logs (cascade should handle this, but being explicit)
	_, err = tx.Exec("DELETE FROM generation_logs WHERE scenario_id = $1", scenarioID)
	if err != nil {
		http.Error(w, "Failed to delete generation logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Delete the scenario
	result, err := tx.Exec("DELETE FROM scenarios WHERE id = $1", scenarioID)
	if err != nil {
		http.Error(w, "Failed to delete scenario: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Check if deletion actually occurred
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to verify deletion: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	if rowsAffected == 0 {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Log the deletion
	log.Printf("Deleted scenario %s and its associated data", scenarioID)
	
	// Return 204 No Content on successful deletion
	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) getGenerationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]
	
	// Use pipeline to get status
	status, err := s.pipeline.GetStatus(scenarioID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Scenario not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) getTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]

	query := `
		SELECT id, name, description, category, prompt_template, resources_suggested,
		       complexity_level, estimated_revenue_min, estimated_revenue_max,
		       usage_count, success_rate, created_at, updated_at, is_active
		FROM scenario_templates 
		WHERE id = $1`

	var template ScenarioTemplate
	var resourcesJSON []byte

	err := s.db.QueryRow(query, templateID).Scan(
		&template.ID, &template.Name, &template.Description, &template.Category,
		&template.PromptTemplate, &resourcesJSON, &template.ComplexityLevel,
		&template.EstimatedRevenueMin, &template.EstimatedRevenueMax,
		&template.UsageCount, &template.SuccessRate, &template.CreatedAt,
		&template.UpdatedAt, &template.IsActive,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse resources JSON array
	if len(resourcesJSON) > 0 {
		json.Unmarshal(resourcesJSON, &template.ResourcesSuggested)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func (s *APIServer) getFeaturedScenarios(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, prompt, files_generated, resources_used, 
		       status, generation_id, complexity, category, estimated_revenue,
		       claude_prompt, claude_response, generation_error,
		       created_at, updated_at, completed_at
		FROM scenarios 
		WHERE status = 'completed' AND estimated_revenue > 20000
		ORDER BY estimated_revenue DESC, created_at DESC
		LIMIT 12`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scenarios []Scenario
	for rows.Next() {
		var scenario Scenario
		var filesJSON []byte
		var resourcesJSON []byte
		
		err := rows.Scan(
			&scenario.ID, &scenario.Name, &scenario.Description, &scenario.Prompt,
			&filesJSON, &resourcesJSON, &scenario.Status, &scenario.GenerationID,
			&scenario.Complexity, &scenario.Category, &scenario.EstimatedRevenue,
			&scenario.ClaudePrompt, &scenario.ClaudeResponse, &scenario.GenerationError,
			&scenario.CreatedAt, &scenario.UpdatedAt, &scenario.CompletedAt,
		)
		if err != nil {
			continue
		}

		// Parse JSON arrays
		if len(filesJSON) > 0 {
			json.Unmarshal(filesJSON, &scenario.FilesGenerated)
		}
		if len(resourcesJSON) > 0 {
			json.Unmarshal(resourcesJSON, &scenario.ResourcesUsed)
		}

		scenarios = append(scenarios, scenario)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

func (s *APIServer) getScenarioLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["id"]

	query := `
		SELECT id, scenario_id, step, prompt, response, success, error_message,
		       started_at, completed_at, duration_seconds
		FROM generation_logs 
		WHERE scenario_id = $1
		ORDER BY started_at ASC`

	rows, err := s.db.Query(query, scenarioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []GenerationLog
	for rows.Next() {
		var log GenerationLog
		
		err := rows.Scan(
			&log.ID, &log.ScenarioID, &log.Step, &log.Prompt,
			&log.Response, &log.Success, &log.ErrorMessage,
			&log.StartedAt, &log.CompletedAt, &log.DurationSeconds,
		)
		if err != nil {
			continue
		}

		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// Statistics and metadata endpoints
func (s *APIServer) getBacklogStats(w http.ResponseWriter, r *http.Request) {
	// Load all backlog items
	pending, _ := s.getBacklogItems("./backlog/pending")
	inProgress, _ := s.getBacklogItems("./backlog/in-progress")
	completed, _ := s.getBacklogItems("./backlog/completed")
	failed, _ := s.getBacklogItems("./backlog/failed")
	
	// Calculate statistics
	totalRevenue := 0
	allItems := append(append(append(pending, inProgress...), completed...), failed...)
	for _, item := range allItems {
		totalRevenue += item.EstimatedRevenue
	}
	
	// Category breakdown
	categoryStats := make(map[string]int)
	priorityStats := make(map[string]int)
	for _, item := range allItems {
		categoryStats[item.Category]++
		priorityStats[item.Priority]++
	}
	
	stats := map[string]interface{}{
		"pending":           len(pending),
		"in_progress":       len(inProgress),
		"completed":         len(completed),
		"failed":            len(failed),
		"total_revenue":     totalRevenue,
		"total_revenue_k":   totalRevenue / 1000,
		"category_breakdown": categoryStats,
		"priority_breakdown": priorityStats,
		"timestamp":         time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) getBacklogMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]interface{}{
		"status_options": []map[string]string{
			{"id": "pending", "name": "Pending", "color": "text-yellow-500", "icon": "clock"},
			{"id": "in_progress", "name": "In Progress", "color": "text-blue-500", "icon": "loader"},
			{"id": "completed", "name": "Completed", "color": "text-green-500", "icon": "check-circle"},
			{"id": "failed", "name": "Failed", "color": "text-red-500", "icon": "alert-circle"},
		},
		"priority_options": []map[string]string{
			{"id": "high", "name": "High", "color": "text-red-500"},
			{"id": "medium", "name": "Medium", "color": "text-yellow-500"},
			{"id": "low", "name": "Low", "color": "text-green-500"},
		},
		"category_options": []string{
			"business-tool", "ai-automation", "content-marketing", "customer-service",
			"e-commerce", "analytics", "document-processing", "financial", "healthcare",
			"education", "productivity", "social-media", "project-management",
		},
		"complexity_options": []map[string]interface{}{
			{"id": "simple", "name": "Simple", "color": "text-green-500", "revenue": "$10K-20K"},
			{"id": "intermediate", "name": "Intermediate", "color": "text-blue-500", "revenue": "$15K-35K"},
			{"id": "advanced", "name": "Advanced", "color": "text-purple-500", "revenue": "$25K-50K"},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

// Validation endpoints
func (s *APIServer) validateScenario(w http.ResponseWriter, r *http.Request) {
	var scenario GenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&scenario); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	validation := map[string]interface{}{
		"valid": true,
		"errors": []string{},
		"warnings": []string{},
	}
	
	// Validate required fields
	if strings.TrimSpace(scenario.Name) == "" {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Name is required")
	}
	
	if strings.TrimSpace(scenario.Description) == "" {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Description is required")
	}
	
	if strings.TrimSpace(scenario.Prompt) == "" {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Prompt is required")
	}
	
	// Validate field lengths
	if len(scenario.Name) > 200 {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Name must be less than 200 characters")
	}
	
	if len(scenario.Description) > 1000 {
		validation["warnings"] = append(validation["warnings"].([]string), "Description is quite long - consider shortening")
	}
	
	// Validate complexity and category
	validComplexities := map[string]bool{"simple": true, "intermediate": true, "advanced": true}
	if !validComplexities[scenario.Complexity] {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Invalid complexity level")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validation)
}

func (s *APIServer) validateBacklogItem(w http.ResponseWriter, r *http.Request) {
	var item BacklogItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	validation := map[string]interface{}{
		"valid": true,
		"errors": []string{},
		"warnings": []string{},
	}
	
	// Validate required fields
	if strings.TrimSpace(item.Name) == "" {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Name is required")
	}
	
	if strings.TrimSpace(item.Description) == "" {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Description is required")
	}
	
	if item.EstimatedRevenue < 0 {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Estimated revenue cannot be negative")
	}
	
	// Validate priority
	validPriorities := map[string]bool{"high": true, "medium": true, "low": true}
	if item.Priority != "" && !validPriorities[strings.ToLower(item.Priority)] {
		validation["valid"] = false
		validation["errors"] = append(validation["errors"].([]string), "Invalid priority level")
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validation)
}

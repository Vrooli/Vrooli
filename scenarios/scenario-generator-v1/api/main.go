package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
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
	claudeAvailable bool
}

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresPort := os.Getenv("POSTGRES_PORT")
		if postgresPort == "" {
			postgresPort = "5432"
		}
		postgresURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%s/postgres?sslmode=disable", postgresPort)
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

	// Check Claude Code availability
	claudeAvailable := checkClaudeCode()

	server := &APIServer{
		db:              db,
		claudeAvailable: claudeAvailable,
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
	api.HandleFunc("/generate/n8n", server.generateScenarioN8N).Methods("POST")
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

	// Start backlog file watcher
	watcher := NewBacklogWatcher(server)
	watcher.Start()
	defer watcher.Stop()

	log.Printf("🚀 Scenario Generator API starting on port %s", port)
	log.Printf("🗄️  Database: %s", postgresURL)
	log.Printf("🤖 Claude Code: %v", claudeAvailable)
	log.Printf("📁 Backlog watcher: Active")

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func checkClaudeCode() bool {
	cmd := exec.Command("claude", "--version")
	err := cmd.Run()
	return err == nil
}

func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]interface{}{
			"database": s.checkDatabase(),
			"claude_code": s.checkClaudeCodeStatus(),
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

func (s *APIServer) checkClaudeCodeStatus() string {
	if s.claudeAvailable {
		return "available"
	}
	return "unavailable"
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

	if !s.claudeAvailable {
		http.Error(w, "Claude Code is not available", http.StatusServiceUnavailable)
		return
	}

	// Generate unique generation ID
	generationID := uuid.New().String()

	// Create scenario record
	scenarioID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO scenarios (id, name, description, prompt, status, generation_id,
		                      complexity, category, estimated_revenue, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := s.db.Exec(query,
		scenarioID, req.Name, req.Description, req.Prompt, "generating", generationID,
		req.Complexity, req.Category, estimateRevenue(req.Complexity), now, now,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start generation process in background
	go s.runScenarioGeneration(scenarioID, req)

	response := map[string]interface{}{
		"generation_id": generationID,
		"scenario_id":   scenarioID,
		"status":        "generating",
		"prompt":        req.Prompt,
		"estimated_time": "2-5 minutes",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) generateScenarioN8N(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	scenarioName, ok := req["scenario_name"].(string)
	if !ok || scenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}
	
	description, ok := req["description"].(string)
	if !ok || description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}

	// Call n8n workflow
	n8nURL := os.Getenv("N8N_WEBHOOK_URL")
	if n8nURL == "" {
		n8nPort := os.Getenv("N8N_PORT")
		if n8nPort == "" {
			n8nPort = "5678"
		}
		n8nURL = fmt.Sprintf("http://localhost:%s/webhook/generate-scenario", n8nPort)
	}

	// Prepare request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to prepare request", http.StatusInternalServerError)
		return
	}

	// Call n8n workflow
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Post(n8nURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("Error calling n8n workflow: %v", err)
		http.Error(w, "Failed to call generation workflow", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read workflow response", http.StatusInternalServerError)
		return
	}

	// Parse response
	var n8nResp map[string]interface{}
	if err := json.Unmarshal(body, &n8nResp); err != nil {
		http.Error(w, "Invalid workflow response", http.StatusInternalServerError)
		return
	}

	// Store scenario in database if successful
	if summary, ok := n8nResp["summary"].(map[string]interface{}); ok {
		if success, ok := summary["success"].(bool); ok && success {
			// Create database record
			scenarioID := uuid.New().String()
			generationID := uuid.New().String()
			if genID, ok := summary["generation_id"].(string); ok {
				generationID = genID
			}
			
			now := time.Now()
			complexity := "intermediate"
			if c, ok := req["complexity"].(string); ok {
				complexity = c
			}
			
			category := "saas-applications"
			if cat, ok := req["category"].(string); ok {
				category = cat
			}

			query := `
				INSERT INTO scenarios (id, name, description, prompt, status, generation_id,
				                      complexity, category, estimated_revenue, created_at, updated_at, completed_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

			_, err = s.db.Exec(query,
				scenarioID, scenarioName, description, "", "completed", generationID,
				complexity, category, estimateRevenue(complexity), now, now, now,
			)
			
			if err != nil {
				log.Printf("Failed to store scenario in database: %v", err)
			}
		}
	}

	// Return n8n response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (s *APIServer) runScenarioGeneration(scenarioID string, req GenerationRequest) {
	log.Printf("Starting scenario generation for %s", scenarioID)
	
	// Build Claude Code prompt
	promptFile := "prompts/scenario-generation-prompt.md"
	claudePrompt := buildClaudePrompt(req, promptFile)
	
	// Log generation start
	s.logGenerationStep(scenarioID, "generation_start", claudePrompt, nil, true, nil)
	
	// Run Claude Code
	response, err := s.runClaudeCode(claudePrompt)
	
	if err != nil {
		// Log error and update scenario
		errorMsg := err.Error()
		s.logGenerationStep(scenarioID, "generation_error", claudePrompt, &response, false, &errorMsg)
		s.updateScenarioStatus(scenarioID, "failed", nil, &claudePrompt, &response, &errorMsg)
		return
	}
	
	// Log success and update scenario
	s.logGenerationStep(scenarioID, "generation_complete", claudePrompt, &response, true, nil)
	
	// Parse files from response (simplified - just store the response)
	files := map[string]string{
		"claude_output": response,
	}
	
	now := time.Now()
	s.updateScenarioStatus(scenarioID, "completed", &now, &claudePrompt, &response, nil)
	s.updateScenarioFiles(scenarioID, files)
	
	log.Printf("Completed scenario generation for %s", scenarioID)
}

func (s *APIServer) runClaudeCode(prompt string) (string, error) {
	// Create temporary file for prompt
	tmpFile := fmt.Sprintf("/tmp/claude_prompt_%d.md", time.Now().Unix())
	
	// Write prompt to file
	err := os.WriteFile(tmpFile, []byte(prompt), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write prompt file: %v", err)
	}
	defer os.Remove(tmpFile)
	
	// Run Claude Code CLI
	cmd := exec.Command("claude", "chat", "--file", tmpFile)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return "", fmt.Errorf("claude code failed: %v - %s", err, string(output))
	}
	
	return string(output), nil
}

func buildClaudePrompt(req GenerationRequest, promptFile string) string {
	// Read prompt template if it exists, otherwise use default
	template := ""
	if data, err := os.ReadFile(promptFile); err == nil {
		template = string(data)
	} else {
		template = `# Vrooli Scenario Generation

You are an expert at creating Vrooli scenarios. Generate a complete scenario based on this request:

**Name:** {{NAME}}
**Description:** {{DESCRIPTION}}
**Prompt:** {{PROMPT}}
**Complexity:** {{COMPLEXITY}}
**Category:** {{CATEGORY}}

Create a complete scenario including:
1. Service configuration (service.json)
2. Database schema if needed
3. API endpoints
4. CLI commands
5. UI components
6. Deployment scripts
7. Documentation

Focus on practical, deployable solutions that provide real business value.`
	}
	
	// Replace placeholders
	prompt := strings.ReplaceAll(template, "{{NAME}}", req.Name)
	prompt = strings.ReplaceAll(prompt, "{{DESCRIPTION}}", req.Description)
	prompt = strings.ReplaceAll(prompt, "{{PROMPT}}", req.Prompt)
	prompt = strings.ReplaceAll(prompt, "{{COMPLEXITY}}", req.Complexity)
	prompt = strings.ReplaceAll(prompt, "{{CATEGORY}}", req.Category)
	
	return prompt
}

func estimateRevenue(complexity string) int {
	switch complexity {
	case "simple":
		return 15000
	case "intermediate":
		return 25000
	case "advanced":
		return 40000
	default:
		return 20000
	}
}

func (s *APIServer) logGenerationStep(scenarioID, step, prompt string, response *string, success bool, errorMsg *string) {
	query := `
		INSERT INTO generation_logs (scenario_id, step, prompt, response, success, error_message, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	s.db.Exec(query, scenarioID, step, prompt, response, success, errorMsg, time.Now())
}

func (s *APIServer) updateScenarioStatus(scenarioID, status string, completedAt *time.Time, claudePrompt, claudeResponse, errorMsg *string) {
	query := `
		UPDATE scenarios 
		SET status = $1, updated_at = $2, completed_at = $3, claude_prompt = $4, claude_response = $5, generation_error = $6
		WHERE id = $7`
	
	s.db.Exec(query, status, time.Now(), completedAt, claudePrompt, claudeResponse, errorMsg, scenarioID)
}

func (s *APIServer) updateScenarioFiles(scenarioID string, files map[string]string) {
	filesJSON, _ := json.Marshal(files)
	query := `UPDATE scenarios SET files_generated = $1, updated_at = $2 WHERE id = $3`
	s.db.Exec(query, filesJSON, time.Now(), scenarioID)
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

// Stub implementations for remaining endpoints
func (s *APIServer) updateScenario(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) deleteScenario(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getGenerationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	generationID := vars["id"]
	
	query := `SELECT status, updated_at FROM scenarios WHERE generation_id = $1`
	
	var status string
	var updatedAt time.Time
	err := s.db.QueryRow(query, generationID).Scan(&status, &updatedAt)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Generation not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"generation_id": generationID,
		"status": status,
		"updated_at": updatedAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
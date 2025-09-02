package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
)

type Experiment struct {
	ID                 string            `json:"id" db:"id"`
	Name               string            `json:"name" db:"name"`
	Description        string            `json:"description" db:"description"`
	Prompt             string            `json:"prompt" db:"prompt"`
	TargetScenario     string            `json:"target_scenario" db:"target_scenario"`
	NewResource        string            `json:"new_resource" db:"new_resource"`
	ExistingResources  []string          `json:"existing_resources" db:"existing_resources"`
	FilesGenerated     map[string]string `json:"files_generated" db:"files_generated"`
	ModificationsMade  map[string]string `json:"modifications_made" db:"modifications_made"`
	Status             string            `json:"status" db:"status"`
	ExperimentID       *string           `json:"experiment_id" db:"experiment_id"`
	ClaudePrompt       *string           `json:"claude_prompt" db:"claude_prompt"`
	ClaudeResponse     *string           `json:"claude_response" db:"claude_response"`
	GenerationError    *string           `json:"generation_error" db:"generation_error"`
	CreatedAt          time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at" db:"updated_at"`
	CompletedAt        *time.Time        `json:"completed_at" db:"completed_at"`
}

type ExperimentTemplate struct {
	ID                     string    `json:"id" db:"id"`
	Name                   string    `json:"name" db:"name"`
	Description            string    `json:"description" db:"description"`
	PromptTemplate         string    `json:"prompt_template" db:"prompt_template"`
	TargetScenarioPattern  *string   `json:"target_scenario_pattern" db:"target_scenario_pattern"`
	ResourceCategory       *string   `json:"resource_category" db:"resource_category"`
	UsageCount             int       `json:"usage_count" db:"usage_count"`
	SuccessRate            *float64  `json:"success_rate" db:"success_rate"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
	IsActive               bool      `json:"is_active" db:"is_active"`
}

type AvailableScenario struct {
	ID                        string    `json:"id" db:"id"`
	Name                      string    `json:"name" db:"name"`
	DisplayName               *string   `json:"display_name" db:"display_name"`
	Description               *string   `json:"description" db:"description"`
	Path                      string    `json:"path" db:"path"`
	CurrentResources          []string  `json:"current_resources" db:"current_resources"`
	ResourceCategories        []string  `json:"resource_categories" db:"resource_categories"`
	ExperimentationFriendly   bool      `json:"experimentation_friendly" db:"experimentation_friendly"`
	ComplexityLevel           string    `json:"complexity_level" db:"complexity_level"`
	LastExperimentDate        *time.Time `json:"last_experiment_date" db:"last_experiment_date"`
	CreatedAt                 time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at" db:"updated_at"`
}

type ExperimentRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Prompt         string `json:"prompt"`
	TargetScenario string `json:"target_scenario"`
	NewResource    string `json:"new_resource"`
}

type ExperimentLog struct {
	ID               string     `json:"id" db:"id"`
	ExperimentID     string     `json:"experiment_id" db:"experiment_id"`
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
	db     *sql.DB
	router *mux.Router
}

func NewAPIServer() *APIServer {
	return &APIServer{}
}

func (s *APIServer) InitDB() error {
	// Get database connection details
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5433")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DB", "postgres")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	s.db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	if err = s.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	log.Printf("Connected to database at %s:%s", dbHost, dbPort)
	return nil
}

func (s *APIServer) InitRoutes() {
	s.router = mux.NewRouter()

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/experiments", s.ListExperiments).Methods("GET")
	api.HandleFunc("/experiments", s.CreateExperiment).Methods("POST")
	api.HandleFunc("/experiments/{id}", s.GetExperiment).Methods("GET")
	api.HandleFunc("/experiments/{id}", s.UpdateExperiment).Methods("PUT")
	api.HandleFunc("/experiments/{id}", s.DeleteExperiment).Methods("DELETE")
	api.HandleFunc("/experiments/{id}/logs", s.GetExperimentLogs).Methods("GET")
	
	api.HandleFunc("/templates", s.ListTemplates).Methods("GET")
	api.HandleFunc("/scenarios", s.ListScenarios).Methods("GET")
	
	// Health check
	s.router.HandleFunc("/health", s.HealthCheck).Methods("GET")
}

func (s *APIServer) ListExperiments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")
	
	query := `SELECT id, name, description, prompt, target_scenario, new_resource, 
	          existing_resources, files_generated, modifications_made, status, 
	          experiment_id, claude_prompt, claude_response, generation_error,
	          created_at, updated_at, completed_at 
	          FROM experiments`
	
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}
	
	query += " ORDER BY created_at DESC"
	
	if limit != "" {
		query += " LIMIT $" + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	experiments := []Experiment{}
	for rows.Next() {
		var exp Experiment
		var existingResourcesJSON, filesGeneratedJSON, modificationsMadeJSON []byte
		
		err := rows.Scan(
			&exp.ID, &exp.Name, &exp.Description, &exp.Prompt,
			&exp.TargetScenario, &exp.NewResource, &existingResourcesJSON,
			&filesGeneratedJSON, &modificationsMadeJSON, &exp.Status,
			&exp.ExperimentID, &exp.ClaudePrompt, &exp.ClaudeResponse,
			&exp.GenerationError, &exp.CreatedAt, &exp.UpdatedAt, &exp.CompletedAt,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Parse JSON fields
		json.Unmarshal(existingResourcesJSON, &exp.ExistingResources)
		json.Unmarshal(filesGeneratedJSON, &exp.FilesGenerated)
		json.Unmarshal(modificationsMadeJSON, &exp.ModificationsMade)

		experiments = append(experiments, exp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(experiments)
}

func (s *APIServer) CreateExperiment(w http.ResponseWriter, r *http.Request) {
	var req ExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.TargetScenario == "" || req.NewResource == "" {
		http.Error(w, "Missing required fields: name, target_scenario, new_resource", http.StatusBadRequest)
		return
	}

	// Create experiment record
	experimentID := uuid.New().String()
	query := `INSERT INTO experiments (id, name, description, prompt, target_scenario, new_resource, status)
	          VALUES ($1, $2, $3, $4, $5, $6, 'requested') RETURNING id, created_at`
	
	var exp Experiment
	err := s.db.QueryRow(query, experimentID, req.Name, req.Description, req.Prompt, 
		req.TargetScenario, req.NewResource).Scan(&exp.ID, &exp.CreatedAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create experiment: %v", err), http.StatusInternalServerError)
		return
	}

	exp.Name = req.Name
	exp.Description = req.Description
	exp.Prompt = req.Prompt
	exp.TargetScenario = req.TargetScenario
	exp.NewResource = req.NewResource
	exp.Status = "requested"

	// Start Claude Code generation asynchronously
	go s.processExperiment(exp.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exp)
}

func (s *APIServer) processExperiment(experimentID string) {
	// Update status to running
	s.db.Exec("UPDATE experiments SET status = 'running' WHERE id = $1", experimentID)

	// Load experiment details
	var exp Experiment
	query := `SELECT name, description, prompt, target_scenario, new_resource FROM experiments WHERE id = $1`
	err := s.db.QueryRow(query, experimentID).Scan(
		&exp.Name, &exp.Description, &exp.Prompt, &exp.TargetScenario, &exp.NewResource)
	if err != nil {
		s.db.Exec("UPDATE experiments SET status = 'failed', generation_error = $2 WHERE id = $1", 
			experimentID, "Failed to load experiment details")
		return
	}

	// Load experiment prompt template
	promptTemplate, err := s.loadPromptTemplate("experiment-prompt.md", exp)
	if err != nil {
		s.db.Exec("UPDATE experiments SET status = 'failed', generation_error = $2 WHERE id = $1", 
			experimentID, fmt.Sprintf("Failed to load prompt template: %v", err))
		return
	}

	// Call Claude Code
	response, err := s.callClaudeCode(promptTemplate)
	if err != nil {
		s.db.Exec("UPDATE experiments SET status = 'failed', generation_error = $2 WHERE id = $1", 
			experimentID, fmt.Sprintf("Claude Code call failed: %v", err))
		return
	}

	// Update experiment with results
	completedAt := time.Now()
	_, err = s.db.Exec(`UPDATE experiments SET 
		status = 'completed', 
		claude_prompt = $2, 
		claude_response = $3, 
		completed_at = $4 
		WHERE id = $1`, 
		experimentID, promptTemplate, response, completedAt)
	
	if err != nil {
		log.Printf("Failed to update experiment %s: %v", experimentID, err)
	}
}

func (s *APIServer) loadPromptTemplate(templateFile string, exp Experiment) (string, error) {
	// Read prompt template
	promptPath := fmt.Sprintf("prompts/%s", templateFile)
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt template: %v", err)
	}

	// Replace template variables
	template := string(content)
	template = strings.ReplaceAll(template, "{{NAME}}", exp.Name)
	template = strings.ReplaceAll(template, "{{DESCRIPTION}}", exp.Description)
	template = strings.ReplaceAll(template, "{{PROMPT}}", exp.Prompt)
	template = strings.ReplaceAll(template, "{{TARGET_SCENARIO}}", exp.TargetScenario)
	template = strings.ReplaceAll(template, "{{NEW_RESOURCE}}", exp.NewResource)

	return template, nil
}

func (s *APIServer) callClaudeCode(prompt string) (string, error) {
	// Create temporary prompt file
	tempFile := fmt.Sprintf("/tmp/claude_prompt_%d.md", time.Now().UnixNano())
	err := os.WriteFile(tempFile, []byte(prompt), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write temporary prompt file: %v", err)
	}
	defer os.Remove(tempFile)

	// Call Claude Code CLI (if available)
	cmd := exec.Command("claude-code", "--file", tempFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude-code execution failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

func (s *APIServer) GetExperiment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `SELECT id, name, description, prompt, target_scenario, new_resource, 
	          existing_resources, files_generated, modifications_made, status, 
	          experiment_id, claude_prompt, claude_response, generation_error,
	          created_at, updated_at, completed_at 
	          FROM experiments WHERE id = $1`

	var exp Experiment
	var existingResourcesJSON, filesGeneratedJSON, modificationsMadeJSON []byte
	
	err := s.db.QueryRow(query, id).Scan(
		&exp.ID, &exp.Name, &exp.Description, &exp.Prompt,
		&exp.TargetScenario, &exp.NewResource, &existingResourcesJSON,
		&filesGeneratedJSON, &modificationsMadeJSON, &exp.Status,
		&exp.ExperimentID, &exp.ClaudePrompt, &exp.ClaudeResponse,
		&exp.GenerationError, &exp.CreatedAt, &exp.UpdatedAt, &exp.CompletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Experiment not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Parse JSON fields
	json.Unmarshal(existingResourcesJSON, &exp.ExistingResources)
	json.Unmarshal(filesGeneratedJSON, &exp.FilesGenerated)
	json.Unmarshal(modificationsMadeJSON, &exp.ModificationsMade)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exp)
}

func (s *APIServer) UpdateExperiment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	for key, value := range updates {
		if key == "status" || key == "files_generated" || key == "modifications_made" {
			setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argCount))
			args = append(args, value)
			argCount++
		}
	}

	if len(setParts) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	query := fmt.Sprintf("UPDATE experiments SET %s WHERE id = $%d", 
		strings.Join(setParts, ", "), argCount)
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Update failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "updated"}`))
}

func (s *APIServer) DeleteExperiment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := s.db.Exec("DELETE FROM experiments WHERE id = $1", id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Delete failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "deleted"}`))
}

func (s *APIServer) GetExperimentLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	experimentID := vars["id"]

	query := `SELECT id, experiment_id, step, prompt, response, success, error_message,
	          started_at, completed_at, duration_seconds 
	          FROM experiment_logs WHERE experiment_id = $1 ORDER BY started_at`

	rows, err := s.db.Query(query, experimentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []ExperimentLog{}
	for rows.Next() {
		var log ExperimentLog
		err := rows.Scan(
			&log.ID, &log.ExperimentID, &log.Step, &log.Prompt,
			&log.Response, &log.Success, &log.ErrorMessage,
			&log.StartedAt, &log.CompletedAt, &log.DurationSeconds,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan failed: %v", err), http.StatusInternalServerError)
			return
		}
		logs = append(logs, log)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *APIServer) ListTemplates(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, description, prompt_template, target_scenario_pattern,
	          resource_category, usage_count, success_rate, created_at, updated_at, is_active
	          FROM experiment_templates WHERE is_active = true ORDER BY name`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	templates := []ExperimentTemplate{}
	for rows.Next() {
		var tmpl ExperimentTemplate
		err := rows.Scan(
			&tmpl.ID, &tmpl.Name, &tmpl.Description, &tmpl.PromptTemplate,
			&tmpl.TargetScenarioPattern, &tmpl.ResourceCategory, &tmpl.UsageCount,
			&tmpl.SuccessRate, &tmpl.CreatedAt, &tmpl.UpdatedAt, &tmpl.IsActive,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan failed: %v", err), http.StatusInternalServerError)
			return
		}
		templates = append(templates, tmpl)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (s *APIServer) ListScenarios(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, display_name, description, path, current_resources, 
	          resource_categories, experimentation_friendly, complexity_level, 
	          last_experiment_date, created_at, updated_at
	          FROM available_scenarios WHERE experimentation_friendly = true ORDER BY name`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	scenarios := []AvailableScenario{}
	for rows.Next() {
		var scenario AvailableScenario
		var currentResourcesJSON, resourceCategoriesJSON []byte
		
		err := rows.Scan(
			&scenario.ID, &scenario.Name, &scenario.DisplayName, &scenario.Description,
			&scenario.Path, &currentResourcesJSON, &resourceCategoriesJSON,
			&scenario.ExperimentationFriendly, &scenario.ComplexityLevel,
			&scenario.LastExperimentDate, &scenario.CreatedAt, &scenario.UpdatedAt,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Parse JSON arrays
		json.Unmarshal(currentResourcesJSON, &scenario.CurrentResources)
		json.Unmarshal(resourceCategoriesJSON, &scenario.ResourceCategories)

		scenarios = append(scenarios, scenario)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

func (s *APIServer) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "resource-experimenter-api",
	})
}

func (s *APIServer) Start() error {
	port := getEnv("API_PORT", "8092")
	
	// Enable CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)(s.router)

	log.Printf("Starting Resource Experimenter API on port %s", port)
	return http.ListenAndServe(":"+port, corsHandler)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	server := NewAPIServer()

	// Initialize database connection
	if err := server.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer server.db.Close()

	// Initialize routes
	server.InitRoutes()

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
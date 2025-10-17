package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Config struct {
	APIPort      int
	RegistryPort int
	DatabaseURL  string
	ScenariosPath string
}

type MCPEndpoint struct {
	ID           string    `json:"id"`
	ScenarioName string    `json:"name"`
	MCPPort      int       `json:"port"`
	Status       string    `json:"status"`
	HasMCP       bool      `json:"hasMCP"`
	Tools        []string  `json:"tools"`
	Confidence   string    `json:"confidence"`
	LastHealthCheck time.Time `json:"lastHealthCheck,omitempty"`
}

type MCPRegistry struct {
	Version   string                 `json:"version"`
	Endpoints []MCPRegistryEndpoint  `json:"endpoints"`
}

type MCPRegistryEndpoint struct {
	Name        string `json:"name"`
	Transport   string `json:"transport"`
	URL         string `json:"url"`
	ManifestURL string `json:"manifest_url"`
}

type AddMCPRequest struct {
	ScenarioName string       `json:"scenario_name"`
	AgentConfig  AgentConfig  `json:"agent_config"`
}

type AgentConfig struct {
	Template   string `json:"template,omitempty"`
	AutoDetect bool   `json:"auto_detect"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
}

func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		router: mux.NewRouter(),
	}
}

func (s *Server) Initialize() error {
	// Connect to database
	var err error
	s.db, err = sql.Open("postgres", s.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set up routes
	s.setupRoutes()

	return nil
}

func (s *Server) setupRoutes() {
	// Health check at root level (required for lifecycle system)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Health check (also available under API prefix)
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	// MCP endpoints
	api.HandleFunc("/mcp/endpoints", s.handleGetEndpoints).Methods("GET")
	api.HandleFunc("/mcp/add", s.handleAddMCP).Methods("POST")
	api.HandleFunc("/mcp/registry", s.handleGetRegistry).Methods("GET")
	api.HandleFunc("/mcp/scenarios/{name}", s.handleGetScenarioDetails).Methods("GET")
	api.HandleFunc("/mcp/sessions/{id}", s.handleGetSession).Methods("GET")
	
	// Enable CORS
	s.router.Use(corsMiddleware)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "scenario-to-mcp",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetEndpoints(w http.ResponseWriter, r *http.Request) {
	// Run detector to get current status
	detectorPath := filepath.Join(s.config.ScenariosPath, "scenario-to-mcp", "lib", "detector.js")
	cmd := exec.Command("node", detectorPath, "scan")
	output, err := cmd.Output()
	if err != nil {
		s.sendError(w, "Failed to scan scenarios", http.StatusInternalServerError)
		return
	}

	var scenarios []MCPEndpoint
	if err := json.Unmarshal(output, &scenarios); err != nil {
		s.sendError(w, "Failed to parse scan results", http.StatusInternalServerError)
		return
	}

	// Query database for additional status info
	rows, err := s.db.Query(`
		SELECT scenario_name, mcp_port, status, last_health_check 
		FROM mcp.endpoints 
		WHERE status = 'active'
	`)
	if err == nil {
		defer rows.Close()
		
		statusMap := make(map[string]MCPEndpoint)
		for rows.Next() {
			var e MCPEndpoint
			rows.Scan(&e.ScenarioName, &e.MCPPort, &e.Status, &e.LastHealthCheck)
			statusMap[e.ScenarioName] = e
		}
		
		// Merge database status with scan results
		for i, scenario := range scenarios {
			if dbStatus, exists := statusMap[scenario.ScenarioName]; exists {
				scenarios[i].Status = dbStatus.Status
				scenarios[i].MCPPort = dbStatus.MCPPort
				scenarios[i].LastHealthCheck = dbStatus.LastHealthCheck
			}
		}
	}

	// Calculate statistics
	stats := map[string]int{
		"total":    len(scenarios),
		"withMCP":  0,
		"active":   0,
		"inactive": 0,
	}
	
	for _, s := range scenarios {
		if s.HasMCP {
			stats["withMCP"]++
		}
		if s.Status == "active" {
			stats["active"]++
		} else if s.Status == "inactive" {
			stats["inactive"]++
		}
	}

	response := map[string]interface{}{
		"scenarios": scenarios,
		"summary":   stats,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAddMCP(w http.ResponseWriter, r *http.Request) {
	var req AddMCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ScenarioName == "" {
		s.sendError(w, "Scenario name is required", http.StatusBadRequest)
		return
	}

	// Generate session ID (database will use gen_random_uuid())
	var sessionID string
	err := s.db.QueryRow(`
		INSERT INTO mcp.agent_sessions (scenario_name, agent_type, status, start_time)
		VALUES ($1, 'claude-code', 'pending', NOW())
		RETURNING id::text
	`, req.ScenarioName).Scan(&sessionID)
	
	if err != nil {
		s.sendError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// In a real implementation, this would spawn a Claude-code agent
	// For now, we'll simulate the process
	go s.simulateMCPAddition(sessionID, req.ScenarioName, req.AgentConfig)

	response := map[string]interface{}{
		"success":          true,
		"agent_session_id": sessionID,
		"estimated_time":   300, // 5 minutes
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetRegistry(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT scenario_name, mcp_port, capabilities, mcp_version, metadata
		FROM mcp.endpoints
		WHERE status = 'active' AND mcp_port IS NOT NULL
		ORDER BY scenario_name
	`)
	
	if err != nil {
		s.sendError(w, "Failed to query registry", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	registry := MCPRegistry{
		Version:   "1.0",
		Endpoints: []MCPRegistryEndpoint{},
	}

	for rows.Next() {
		var name string
		var port int
		var capabilities, metadata sql.NullString
		var version sql.NullString
		
		if err := rows.Scan(&name, &port, &capabilities, &version, &metadata); err != nil {
			continue
		}
		
		endpoint := MCPRegistryEndpoint{
			Name:        name,
			Transport:   "stdio",
			URL:         fmt.Sprintf("http://localhost:%d", port),
			ManifestURL: fmt.Sprintf("http://localhost:%d/manifest", port),
		}
		
		registry.Endpoints = append(registry.Endpoints, endpoint)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registry)
}

func (s *Server) handleGetScenarioDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	
	// Run detector for detailed info
	detectorPath := filepath.Join(s.config.ScenariosPath, "scenario-to-mcp", "lib", "detector.js")
	cmd := exec.Command("node", detectorPath, "check", scenarioName)
	output, err := cmd.Output()
	if err != nil {
		s.sendError(w, "Failed to get scenario details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	
	var session struct {
		ID           string    `json:"id"`
		ScenarioName string    `json:"scenario_name"`
		Status       string    `json:"status"`
		StartTime    time.Time `json:"start_time"`
		EndTime      *time.Time `json:"end_time,omitempty"`
		Logs         string    `json:"logs,omitempty"`
	}
	
	err := s.db.QueryRow(`
		SELECT id, scenario_name, status, start_time, end_time, logs
		FROM mcp.agent_sessions
		WHERE id = $1
	`, sessionID).Scan(&session.ID, &session.ScenarioName, &session.Status, 
		&session.StartTime, &session.EndTime, &session.Logs)
	
	if err != nil {
		s.sendError(w, "Session not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) simulateMCPAddition(sessionID, scenarioName string, config AgentConfig) {
	// Update session status to running
	s.db.Exec(`
		UPDATE mcp.agent_sessions 
		SET status = 'running', logs = 'Starting MCP addition process...\n'
		WHERE id = $1
	`, sessionID)
	
	// Simulate processing time
	time.Sleep(2 * time.Second)
	
	// Run code generator
	generatorPath := filepath.Join(s.config.ScenariosPath, "scenario-to-mcp", "lib", "code-generator.js")
	cmd := exec.Command("node", generatorPath, "generate", scenarioName)
	output, err := cmd.Output()
	
	if err != nil {
		s.db.Exec(`
			UPDATE mcp.agent_sessions 
			SET status = 'failed', end_time = NOW(), 
			    logs = logs || 'Failed to generate MCP implementation\n'
			WHERE id = $1
		`, sessionID)
		return
	}
	
	// Allocate port for the scenario
	var port int
	s.db.QueryRow(`SELECT mcp.allocate_mcp_port($1)`, scenarioName).Scan(&port)
	
	// Store in database
	_, err = s.db.Exec(`
		INSERT INTO mcp.endpoints (scenario_name, scenario_path, mcp_port, status, created_at)
		VALUES ($1, $2, $3, 'pending', NOW())
		ON CONFLICT (scenario_name) 
		DO UPDATE SET mcp_port = $3, status = 'pending', updated_at = NOW()
	`, scenarioName, filepath.Join(s.config.ScenariosPath, scenarioName), port)
	
	if err != nil {
		s.db.Exec(`
			UPDATE mcp.agent_sessions 
			SET status = 'failed', end_time = NOW(),
			    logs = logs || 'Failed to store MCP configuration\n'
			WHERE id = $1
		`, sessionID)
		return
	}
	
	// Update session as completed
	s.db.Exec(`
		UPDATE mcp.agent_sessions 
		SET status = 'completed', end_time = NOW(),
		    logs = logs || 'MCP implementation added successfully\n',
		    result = $2
		WHERE id = $1
	`, sessionID, string(output))
}

func (s *Server) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

func (s *Server) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-to-mcp

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	var registryMode bool
	flag.BoolVar(&registryMode, "registry-mode", false, "Run in registry mode")
	flag.Parse()

	// Get scenarios path - default to Vrooli root if not set
	defaultScenariosPath := filepath.Join("..", "..")
	if homeDir := os.Getenv("HOME"); homeDir != "" {
		defaultScenariosPath = filepath.Join(homeDir, "Vrooli", "scenarios")
	}

	config := &Config{
		APIPort:      getEnvInt("API_PORT", 3290),
		RegistryPort: getEnvInt("REGISTRY_PORT", 3292),
		DatabaseURL:  getEnvString("DATABASE_URL", ""),
		ScenariosPath: getEnvString("SCENARIOS_PATH", defaultScenariosPath),
	}

	// Validate required configuration
	if config.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	server := NewServer(config)
	
	if err := server.Initialize(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	port := config.APIPort
	if registryMode {
		port = config.RegistryPort
		log.Printf("Starting MCP Registry on port %d", port)
	} else {
		log.Printf("Starting Scenario to MCP API on port %d", port)
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), server.router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
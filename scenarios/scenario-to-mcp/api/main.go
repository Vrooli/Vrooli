package main

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Config struct {
	APIPort       int
	RegistryPort  int
	DatabaseURL   string
	ScenariosPath string
}

type MCPEndpoint struct {
	ID              string    `json:"id"`
	ScenarioName    string    `json:"name"`
	MCPPort         int       `json:"port"`
	Status          string    `json:"status"`
	HasMCP          bool      `json:"hasMCP"`
	Tools           []string  `json:"tools"`
	Confidence      string    `json:"confidence"`
	LastHealthCheck time.Time `json:"lastHealthCheck,omitempty"`
}

type MCPRegistry struct {
	Version   string                `json:"version"`
	Endpoints []MCPRegistryEndpoint `json:"endpoints"`
}

type MCPRegistryEndpoint struct {
	Name        string `json:"name"`
	Transport   string `json:"transport"`
	URL         string `json:"url"`
	ManifestURL string `json:"manifest_url"`
}

type AddMCPRequest struct {
	ScenarioName string      `json:"scenario_name"`
	AgentConfig  AgentConfig `json:"agent_config"`
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

const scenarioName = "scenario-to-mcp"

type DocMetadata struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Category     string    `json:"category"`
	Summary      string    `json:"summary,omitempty"`
	RelativePath string    `json:"relativePath"`
	Source       string    `json:"source"`
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
	absolutePath string    `json:"-"`
}

var (
	errDocNotFound = errors.New("document not found")
	curatedDocs    = []struct {
		relativePath string
		title        string
		category     string
		summary      string
		source       string
	}{
		{
			relativePath: "README.md",
			title:        "Scenario Overview",
			category:     "Overview",
			summary:      "High-level description of the Scenario to MCP application and core workflows.",
			source:       "scenario",
		},
		{
			relativePath: "PRD.md",
			title:        "Product Requirements",
			category:     "Product",
			summary:      "Detailed requirements and objectives that shape the Scenario to MCP experience.",
			source:       "scenario",
		},
		{
			relativePath: "PROBLEMS.md",
			title:        "Known Limitations",
			category:     "Reference",
			summary:      "Documented issues and design gaps that influence MCP onboarding.",
			source:       "scenario",
		},
		{
			relativePath: "TEST_IMPLEMENTATION_SUMMARY.md",
			title:        "Testing Summary",
			category:     "Reference",
			summary:      "Testing approach and coverage notes for the scenario API and UI.",
			source:       "scenario",
		},
	}
)

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

	// Documentation routes
	api.HandleFunc("/docs", s.handleListDocs).Methods("GET")
	api.HandleFunc("/docs/content", s.handleGetDocContent).Methods("GET")

	// Enable CORS
	s.router.Use(corsMiddleware)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Only allow localhost origins for security
		if origin == "http://localhost:3291" || origin == "http://localhost:36111" ||
			origin == "http://localhost:39942" || origin == "http://127.0.0.1:3291" ||
			origin == "http://127.0.0.1:36111" || origin == "http://127.0.0.1:39942" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
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
		"readiness": true,
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
		ID           string     `json:"id"`
		ScenarioName string     `json:"scenario_name"`
		Status       string     `json:"status"`
		StartTime    time.Time  `json:"start_time"`
		EndTime      *time.Time `json:"end_time,omitempty"`
		Logs         string     `json:"logs,omitempty"`
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

func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	docs, err := s.collectDocumentation()
	if err != nil {
		s.sendError(w, "Failed to load documentation index", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"docs":  docs,
		"count": len(docs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetDocContent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		s.sendError(w, "Document id is required", http.StatusBadRequest)
		return
	}

	doc, err := s.findDocumentationByID(id)
	if err != nil {
		if errors.Is(err, errDocNotFound) {
			s.sendError(w, "Document not found", http.StatusNotFound)
			return
		}
		s.sendError(w, "Failed to resolve document", http.StatusInternalServerError)
		return
	}

	data, err := os.ReadFile(doc.absolutePath)
	if err != nil {
		s.sendError(w, "Failed to read document", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":           doc.ID,
		"title":        doc.Title,
		"category":     doc.Category,
		"summary":      doc.Summary,
		"relativePath": doc.RelativePath,
		"source":       doc.Source,
		"lastModified": doc.LastModified,
		"size":         doc.Size,
		"content":      string(data),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

func (s *Server) collectDocumentation() ([]DocMetadata, error) {
	scenarioRoot := filepath.Join(s.config.ScenariosPath, scenarioName)
	docsRoot := filepath.Join(scenarioRoot, "docs")
	entries := make([]DocMetadata, 0)

	if info, err := os.Stat(docsRoot); err == nil && info.IsDir() {
		if err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			relativePath, err := filepath.Rel(scenarioRoot, path)
			if err != nil {
				return err
			}
			relativePath = filepath.ToSlash(relativePath)

			doc := DocMetadata{
				ID:           buildDocID("docs", relativePath),
				Title:        deriveDocTitle(d.Name()),
				Category:     deriveDocCategory(relativePath, "Documentation"),
				Summary:      extractDocSummary(path),
				RelativePath: relativePath,
				Source:       "docs",
				LastModified: info.ModTime(),
				Size:         info.Size(),
				absolutePath: path,
			}

			entries = append(entries, doc)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	for _, spec := range curatedDocs {
		absolutePath := filepath.Join(scenarioRoot, spec.relativePath)
		info, err := os.Stat(absolutePath)
		if err != nil || info.IsDir() {
			continue
		}

		relative := filepath.ToSlash(spec.relativePath)
		doc := DocMetadata{
			ID:           buildDocID(spec.source, relative),
			Title:        spec.title,
			Category:     spec.category,
			Summary:      firstNonEmpty(spec.summary, extractDocSummary(absolutePath)),
			RelativePath: relative,
			Source:       spec.source,
			LastModified: info.ModTime(),
			Size:         info.Size(),
			absolutePath: absolutePath,
		}

		entries = append(entries, doc)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Category == entries[j].Category {
			return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
		}
		return strings.ToLower(entries[i].Category) < strings.ToLower(entries[j].Category)
	})

	return entries, nil
}

func (s *Server) findDocumentationByID(id string) (*DocMetadata, error) {
	source, relative, parseErr := parseDocID(id)
	if parseErr != nil {
		return nil, errDocNotFound
	}

	entries, err := s.collectDocumentation()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.ID == id || (entry.Source == source && entry.RelativePath == relative) {
			return &entry, nil
		}
	}

	return nil, errDocNotFound
}

func buildDocID(source, relative string) string {
	key := strings.Join([]string{source, filepath.ToSlash(relative)}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(key))
}

func parseDocID(id string) (string, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid document id")
	}

	return parts[0], parts[1], nil
}

func deriveDocTitle(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		return name
	}

	segments := strings.FieldsFunc(base, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, segment := range segments {
		segments[i] = toTitleWord(segment)
	}

	if len(segments) == 0 {
		return toTitleWord(base)
	}

	return strings.Join(segments, " ")
}

func deriveDocCategory(relativePath, fallback string) string {
	relativePath = filepath.ToSlash(relativePath)
	parts := strings.Split(relativePath, "/")
	if len(parts) == 0 {
		return fallback
	}

	if parts[0] == "docs" {
		if len(parts) > 2 {
			return toTitleWord(strings.ReplaceAll(parts[1], "-", " "))
		}
		return "Documentation"
	}

	if len(parts) > 1 {
		return toTitleWord(strings.ReplaceAll(parts[0], "-", " "))
	}

	return toTitleWord(strings.ReplaceAll(parts[0], "-", " "))
}

func extractDocSummary(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		return line
	}

	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func toTitleWord(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	runes := []rune(strings.ToLower(value))
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
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
		APIPort:       getEnvInt("API_PORT", 3290),
		RegistryPort:  getEnvInt("REGISTRY_PORT", 3292),
		DatabaseURL:   getEnvString("DATABASE_URL", ""),
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
		fmt.Fprintf(os.Stdout, `{"level":"info","component":"registry","message":"Starting MCP Registry","port":%d,"timestamp":"%s"}`+"\n",
			port, time.Now().Format(time.RFC3339))
	} else {
		fmt.Fprintf(os.Stdout, `{"level":"info","component":"api","message":"Starting Scenario to MCP API","port":%d,"timestamp":"%s"}`+"\n",
			port, time.Now().Format(time.RFC3339))
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
// Test change

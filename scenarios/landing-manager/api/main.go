package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config          *Config
	db              *sql.DB
	router          *mux.Router
	templateService *TemplateService
	httpClient      *http.Client
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := seedDefaultData(db); err != nil {
		return nil, fmt.Errorf("failed to seed default data: %w", err)
	}

	srv := &Server{
		config:          cfg,
		db:              db,
		router:          mux.NewRouter(),
		templateService: NewTemplateService(),
		httpClient:      &http.Client{Timeout: 15 * time.Second},
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")

	// Template management endpoints
	s.router.HandleFunc("/api/v1/templates", s.handleTemplateList).Methods("GET")
	s.router.HandleFunc("/api/v1/templates/{id}", s.handleTemplateShow).Methods("GET")
	s.router.HandleFunc("/api/v1/generate", s.handleGenerate).Methods("POST")
	s.router.HandleFunc("/api/v1/customize", s.handleCustomize).Methods("POST")
	s.router.HandleFunc("/api/v1/generated", s.handleGeneratedList).Methods("GET")
	s.router.HandleFunc("/api/v1/preview/{scenario_id}", s.handlePreviewLinks).Methods("GET")
	s.router.HandleFunc("/api/v1/personas", s.handlePersonaList).Methods("GET")
	s.router.HandleFunc("/api/v1/personas/{id}", s.handlePersonaShow).Methods("GET")

	// Admin authentication endpoints (OT-P0-008)
	s.router.HandleFunc("/api/v1/admin/login", handleTemplateOnly("admin authentication")).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/logout", handleTemplateOnly("admin authentication")).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/session", handleTemplateOnly("admin authentication")).Methods("GET")

	// A/B Testing variant endpoints (OT-P0-014 through OT-P0-018)
	s.router.HandleFunc("/api/v1/variants/select", handleTemplateOnly("variant selection")).Methods("GET")
	s.router.HandleFunc("/api/v1/variants", handleTemplateOnly("variant management")).Methods("GET")
	s.router.HandleFunc("/api/v1/variants", handleTemplateOnly("variant management")).Methods("POST")
	s.router.HandleFunc("/api/v1/variants/{slug}", handleTemplateOnly("variant management")).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", handleTemplateOnly("variant management")).Methods("PATCH")
	s.router.HandleFunc("/api/v1/variants/{slug}/archive", handleTemplateOnly("variant management")).Methods("POST")
	s.router.HandleFunc("/api/v1/variants/{slug}", handleTemplateOnly("variant management")).Methods("DELETE")

	// Metrics & Analytics endpoints (OT-P0-019 through OT-P0-024)
	s.router.HandleFunc("/api/v1/metrics/track", handleTemplateOnly("metrics tracking")).Methods("POST")
	s.router.HandleFunc("/api/v1/metrics/summary", handleTemplateOnly("metrics summary")).Methods("GET")
	s.router.HandleFunc("/api/v1/metrics/variants", handleTemplateOnly("metrics variant stats")).Methods("GET")

	// Stripe Payment endpoints (OT-P0-025 through OT-P0-030)
	s.router.HandleFunc("/api/v1/checkout/create", handleTemplateOnly("checkout")).Methods("POST")
	s.router.HandleFunc("/api/v1/webhooks/stripe", handleTemplateOnly("Stripe webhooks")).Methods("POST")
	s.router.HandleFunc("/api/v1/subscription/verify", handleTemplateOnly("subscription verification")).Methods("GET")
	s.router.HandleFunc("/api/v1/subscription/cancel", handleTemplateOnly("subscription cancel")).Methods("POST")

	// Content Customization endpoints (OT-P0-012, OT-P0-013: CUSTOM-SPLIT, CUSTOM-LIVE)
	s.router.HandleFunc("/api/v1/variants/{variant_id}/sections", handleTemplateOnly("section listing")).Methods("GET")
	s.router.HandleFunc("/api/v1/sections/{id}", handleTemplateOnly("section detail")).Methods("GET")
	s.router.HandleFunc("/api/v1/sections/{id}", handleTemplateOnly("section update")).Methods("PATCH")
	s.router.HandleFunc("/api/v1/sections", handleTemplateOnly("section creation")).Methods("POST")
	s.router.HandleFunc("/api/v1/sections/{id}", handleTemplateOnly("section delete")).Methods("DELETE")

	// Lifecycle management endpoints for generated scenarios
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", s.handleScenarioStart).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/stop", s.handleScenarioStop).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/restart", s.handleScenarioRestart).Methods("POST")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/status", s.handleScenarioStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/logs", s.handleScenarioLogs).Methods("GET")
	s.router.HandleFunc("/api/v1/lifecycle/{scenario_id}/promote", s.handleScenarioPromote).Methods("POST")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "landing-manager-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Landing Manager API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// seedDefaultData is a no-op in factory mode; template data lives in generated scenarios.
func seedDefaultData(db *sql.DB) error {
	logStructured("seed_default_data_skipped", map[string]interface{}{
		"reason": "factory_mode",
	})
	return nil
}

func (s *Server) handleTemplateList(w http.ResponseWriter, r *http.Request) {
	templates, err := s.templateService.ListTemplates()
	if err != nil {
		s.log("failed to list templates", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Failed to list templates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (s *Server) handleTemplateShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	template, err := s.templateService.GetTemplate(id)
	if err != nil {
		s.log("failed to get template", map[string]interface{}{"id": id, "error": err.Error()})
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TemplateID string                 `json:"template_id"`
		Name       string                 `json:"name"`
		Slug       string                 `json:"slug"`
		Options    map[string]interface{} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := s.templateService.GenerateScenario(req.TemplateID, req.Name, req.Slug, req.Options)
	if err != nil {
		s.log("failed to generate scenario", map[string]interface{}{
			"template_id": req.TemplateID,
			"name":        req.Name,
			"error":       err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to generate scenario: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGeneratedList(w http.ResponseWriter, r *http.Request) {
	scenarios, err := s.templateService.ListGeneratedScenarios()
	if err != nil {
		s.log("failed to list generated scenarios", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Failed to list generated scenarios", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

func (s *Server) handlePreviewLinks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["scenario_id"]

	preview, err := s.templateService.GetPreviewLinks(scenarioID)
	if err != nil {
		s.log("failed to get preview links", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to get preview links: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preview)
}

func (s *Server) handleCustomize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string   `json:"scenario_id"`
		Brief      string   `json:"brief"`
		Assets     []string `json:"assets"`
		Preview    bool     `json:"preview"`
		PersonaID  string   `json:"persona_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	issueTrackerBase, err := s.resolveIssueTrackerBase()
	if err != nil {
		http.Error(w, fmt.Sprintf("Issue tracker unavailable: %v", err), http.StatusBadGateway)
		return
	}

	issueTitle := fmt.Sprintf("Customize landing page scenario: %s", strings.TrimSpace(req.ScenarioID))
	if strings.TrimSpace(req.ScenarioID) == "" {
		issueTitle = "Customize landing page scenario (unnamed)"
	}

	// Build description with persona prompt if provided
	personaPrompt := ""
	if req.PersonaID != "" {
		persona, err := s.templateService.GetPersona(req.PersonaID)
		if err == nil {
			personaPrompt = fmt.Sprintf("\n\nPersona: %s\nGuidance:\n%s", persona.Name, persona.Prompt)
		}
	}

	description := fmt.Sprintf(
		"Requested customization for landing page scenario.\n\nScenario: %s\nBrief:\n%s\nAssets: %v\nPreview: %t%s\nSource: landing-manager factory\nTimestamp: %s\n\nExpected deliverables:\n- Apply brief to template-safe areas (content, design tokens, imagery)\n- Run A/B variant setup if applicable\n- Regenerate preview links and summarize changes\n- Return next steps and validation status.",
		req.ScenarioID, req.Brief, req.Assets, req.Preview, personaPrompt, time.Now().UTC().Format(time.RFC3339),
	)

	issuePayload := map[string]interface{}{
		"title":       issueTitle,
		"description": description,
		"type":        "feature",
		"priority":    "high",
		"app_id":      "landing-manager",
		"tags":        []string{"landing-manager", "landing-page", "customization", "automation"},
		"environment": map[string]interface{}{
			"scenario_id":  req.ScenarioID,
			"template_id":  "saas-landing-page",
			"requested_by": "landing-manager",
			"preview_mode": req.Preview,
			"asset_hints":  req.Assets,
		},
	}

	issueResp := struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}{}

	if err := s.postJSON(issueTrackerBase+"/issues", issuePayload, &issueResp); err != nil {
		s.log("issue_tracker_create_failed", map[string]interface{}{"error": err.Error()})
		http.Error(w, fmt.Sprintf("Failed to file issue: %v", err), http.StatusBadGateway)
		return
	}

	issueID := ""
	if issueResp.Data != nil {
		if v, ok := issueResp.Data["issue_id"].(string); ok {
			issueID = v
		}
		if issueID == "" {
			if nested, ok := issueResp.Data["issue"].(map[string]interface{}); ok {
				if v, ok := nested["id"].(string); ok {
					issueID = v
				}
			}
		}
	}

	// Trigger investigation to kick off the agent workflow
	investigation := struct {
		RunID   string `json:"run_id,omitempty"`
		Status  string `json:"status,omitempty"`
		AgentID string `json:"agent_id,omitempty"`
	}{}

	if issueID != "" {
		investigatePayload := map[string]interface{}{
			"issue_id": issueID,
			"priority": "high",
		}
		investigateResp := struct {
			Success bool                   `json:"success"`
			Data    map[string]interface{} `json:"data"`
		}{}
		if err := s.postJSON(issueTrackerBase+"/investigate", investigatePayload, &investigateResp); err == nil && investigateResp.Data != nil {
			if v, ok := investigateResp.Data["run_id"].(string); ok {
				investigation.RunID = v
			}
			if v, ok := investigateResp.Data["status"].(string); ok {
				investigation.Status = v
			}
			if v, ok := investigateResp.Data["agent_id"].(string); ok {
				investigation.AgentID = v
			}
		} else if err != nil {
			s.log("issue_tracker_investigate_failed", map[string]interface{}{"error": err.Error(), "issue_id": issueID})
		}
	}

	response := map[string]interface{}{
		"status":      "queued",
		"issue_id":    issueID,
		"tracker_url": issueTrackerBase,
		"agent":       investigation.AgentID,
		"run_id":      investigation.RunID,
		"message":     issueResp.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handlePersonaList(w http.ResponseWriter, r *http.Request) {
	personas, err := s.templateService.GetPersonas()
	if err != nil {
		s.log("failed to list personas", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Failed to list personas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(personas)
}

func (s *Server) handlePersonaShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	persona, err := s.templateService.GetPersona(id)
	if err != nil {
		s.log("failed to get persona", map[string]interface{}{"id": id, "error": err.Error()})
		http.Error(w, "Persona not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(persona)
}

// loggingMiddleware prints structured request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fields := map[string]interface{}{
			"method":   r.Method,
			"path":     r.RequestURI,
			"duration": time.Since(start).String(),
		}
		logStructured("request_completed", fields)
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	logStructured(msg, fields)
}

func logStructured(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Printf(`{"level":"info","message":"%s","timestamp":"%s"}`, msg, time.Now().UTC().Format(time.RFC3339))
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf(`{"level":"info","message":"%s","fields":%s,"timestamp":"%s"}`, msg, fieldsJSON, time.Now().UTC().Format(time.RFC3339))
}

// handleTemplateOnly makes clear that specific capabilities belong to generated landing scenarios, not the factory.
func handleTemplateOnly(feature string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		response := map[string]string{
			"status":  "template_only",
			"feature": feature,
			"message": "Use a generated landing scenario to access this capability; the landing-manager factory only creates templates and scenarios.",
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}

func logStructuredError(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Printf(`{"level":"error","message":"%s","timestamp":"%s"}`, msg, time.Now().UTC().Format(time.RFC3339))
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf(`{"level":"error","message":"%s","fields":%s,"timestamp":"%s"}`, msg, fieldsJSON, time.Now().UTC().Format(time.RFC3339))
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func (s *Server) resolveIssueTrackerBase() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("APP_ISSUE_TRACKER_API_BASE")); raw != "" {
		return strings.TrimSuffix(raw, "/"), nil
	}

	if port := strings.TrimSpace(os.Getenv("APP_ISSUE_TRACKER_API_PORT")); port != "" {
		return fmt.Sprintf("http://localhost:%s/api/v1", port), nil
	}

	// Fallback: attempt to discover via vrooli CLI
	cmd := execCommandContext(context.Background(), "vrooli", "scenario", "port", "app-issue-tracker", "API_PORT")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("unable to resolve app-issue-tracker API port")
	}
	port := strings.TrimSpace(string(out))
	if port == "" {
		return "", fmt.Errorf("app-issue-tracker API port not available")
	}
	return fmt.Sprintf("http://localhost:%s/api/v1", port), nil
}

// execCommandContext is wrapped for testability
var execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func (s *Server) postJSON(url string, payload interface{}, out interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("issue tracker request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("issue tracker responded %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start landing-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

// Lifecycle management handlers

// isScenarioNotFound checks if a command error indicates a missing scenario
func isScenarioNotFound(output string) bool {
	return strings.Contains(output, "not found") ||
		strings.Contains(output, "does not exist") ||
		strings.Contains(output, "No such scenario") ||
		strings.Contains(output, "No lifecycle log found")
}

func (s *Server) handleScenarioStart(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}

	// Resolve scenario path (check staging area first, then production)
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Try generated/staging area first
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	var cmd *exec.Cmd
	if _, err := os.Stat(stagingPath); err == nil {
		// Scenario is in staging area - use --path parameter
		cmd = exec.Command("vrooli", "scenario", "start", scenarioID, "--path", stagingPath)
	} else if _, err := os.Stat(productionPath); err == nil {
		// Scenario is in production location - use standard start
		cmd = exec.Command("vrooli", "scenario", "start", scenarioID)
	} else {
		// Scenario not found anywhere
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found in staging (%s) or production (%s)", scenarioID, stagingPath, productionPath),
		})
		return
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if isScenarioNotFound(outputStr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
			})
			return
		}

		s.log("scenario_start_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
			"output":      outputStr,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to start scenario: %v", err),
			"output":  outputStr,
		})
		return
	}

	s.log("scenario_started", map[string]interface{}{"scenario_id": scenarioID})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Scenario started successfully",
		"scenario_id": scenarioID,
		"output":      string(output),
	})
}

func (s *Server) handleScenarioStop(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}

	// Resolve scenario path (check staging area first, then production)
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Try generated/staging area first
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	var cmd *exec.Cmd
	if _, err := os.Stat(stagingPath); err == nil {
		// Scenario is in staging area - use --path parameter
		cmd = exec.Command("vrooli", "scenario", "stop", scenarioID, "--path", stagingPath)
	} else if _, err := os.Stat(productionPath); err == nil {
		// Scenario is in production location - use standard stop
		cmd = exec.Command("vrooli", "scenario", "stop", scenarioID)
	} else {
		// Scenario directory not found, but stop is idempotent - try standard stop anyway
		// The CLI will handle non-running scenarios gracefully
		cmd = exec.Command("vrooli", "scenario", "stop", scenarioID)
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		// Check if this is a "scenario not found" or "utilities not found" error
		// Stop is idempotent - these errors mean the scenario can't be running anyway
		if isScenarioNotFound(outputStr) || strings.Contains(outputStr, "Cannot find Vrooli utilities") {
			// Stop is idempotent - return success even if scenario doesn't exist
			// This allows for safe cleanup operations
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Scenario '%s' already stopped or not found", scenarioID),
				"output":  outputStr,
			})
			return
		}

		s.log("scenario_stop_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
			"output":      outputStr,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to stop scenario: %v", err),
			"output":  outputStr,
		})
		return
	}

	s.log("scenario_stopped", map[string]interface{}{"scenario_id": scenarioID})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Scenario stopped successfully",
		"scenario_id": scenarioID,
		"output":      string(output),
	})
}

func (s *Server) handleScenarioRestart(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}

	// Resolve scenario path (check staging area first)
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	var cmd *exec.Cmd
	if _, err := os.Stat(stagingPath); err == nil {
		// Scenario is in staging area - use --path parameter
		cmd = exec.Command("vrooli", "scenario", "restart", scenarioID, "--path", stagingPath)
	} else if _, err := os.Stat(productionPath); err == nil {
		// Scenario is in production location
		cmd = exec.Command("vrooli", "scenario", "restart", scenarioID)
	} else {
		// Scenario not found anywhere
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found in staging or production", scenarioID),
		})
		return
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if isScenarioNotFound(outputStr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
			})
			return
		}

		s.log("scenario_restart_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
			"output":      outputStr,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to restart scenario: %v", err),
			"output":  outputStr,
		})
		return
	}

	s.log("scenario_restarted", map[string]interface{}{"scenario_id": scenarioID})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Scenario restarted successfully",
		"scenario_id": scenarioID,
		"output":      string(output),
	})
}

func (s *Server) handleScenarioStatus(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}

	// Resolve scenario path (check staging area first, then production)
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Try generated/staging area first
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	// Check if scenario is in staging area
	if _, err := os.Stat(stagingPath); err == nil {
		// Scenario is in staging area - check process metadata directory directly
		// since vrooli scenario status command uses the API (which doesn't know about staging)
		processDir := filepath.Join(os.Getenv("HOME"), ".vrooli", "processes", "scenarios", scenarioID)
		running := false
		statusText := fmt.Sprintf("Scenario '%s' is in staging area (generated/)", scenarioID)

		// Check if any processes are running for this scenario
		if entries, err := os.ReadDir(processDir); err == nil && len(entries) > 0 {
			// Count active processes by checking PIDs
			activeCount := 0
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pid" {
					pidFile := filepath.Join(processDir, entry.Name())
					if pidBytes, err := os.ReadFile(pidFile); err == nil {
						pidStr := strings.TrimSpace(string(pidBytes))
						// Check if process is still running
						checkCmd := exec.Command("kill", "-0", pidStr)
						if checkCmd.Run() == nil {
							activeCount++
						}
					}
				}
			}
			if activeCount > 0 {
				running = true
				statusText = fmt.Sprintf("Scenario '%s' is running in staging area (%d active process(es))", scenarioID, activeCount)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"scenario_id": scenarioID,
			"running":     running,
			"status_text": statusText,
			"location":    "staging",
		})
		return
	}

	// Not in staging, check production
	var cmd *exec.Cmd
	if _, err := os.Stat(productionPath); err == nil {
		// Scenario is in production location - use standard status
		cmd = exec.Command("vrooli", "scenario", "status", scenarioID)
	} else {
		// Scenario not found anywhere
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
		})
		return
	}

	output, err := cmd.CombinedOutput()
	statusText := string(output)

	if err != nil {
		// Check if scenario doesn't exist (exit status 1 with "not found" message)
		if isScenarioNotFound(statusText) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
			})
			return
		}

		// Other errors (permissions, CLI issues, etc.)
		s.log("scenario_status_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get scenario status: %v", err),
		})
		return
	}

	// Parse vrooli scenario status output
	running := strings.Contains(statusText, "🟢 RUNNING") || strings.Contains(statusText, "Status:        🟢 RUNNING")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"scenario_id": scenarioID,
		"running":     running,
		"status_text": statusText,
		"location":    "production",
	})
}

func (s *Server) handleScenarioLogs(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}

	// Get tail parameter (default 50 lines)
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "50"
	}

	// Resolve scenario path (check staging area first, then production)
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	// Try generated/staging area first
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	var cmd *exec.Cmd
	if _, err := os.Stat(stagingPath); err == nil {
		// Scenario is in staging area - use --path parameter
		cmd = exec.Command("vrooli", "scenario", "logs", scenarioID, "--path", stagingPath, "--tail", tail)
	} else if _, err := os.Stat(productionPath); err == nil {
		// Scenario is in production location - use standard logs
		cmd = exec.Command("vrooli", "scenario", "logs", scenarioID, "--tail", tail)
	} else {
		// Scenario not found anywhere
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
		})
		return
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if isScenarioNotFound(outputStr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
			})
			return
		}

		s.log("scenario_logs_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get scenario logs: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"scenario_id": scenarioID,
		"logs":        string(output),
	})
}

func (s *Server) handleScenarioPromote(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if scenarioID == "" {
		http.Error(w, "scenario_id is required", http.StatusBadRequest)
		return
	}

	// Verify scenario exists in generated/ folder
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	generatedPath := filepath.Join(vrooliRoot, "scenarios", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	// Check if scenario exists in generated/
	if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
		s.log("promote_failed_not_found", map[string]interface{}{
			"scenario_id": scenarioID,
			"path":        generatedPath,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found in staging area (generated/)", scenarioID),
		})
		return
	}

	// Check if production path already exists
	if _, err := os.Stat(productionPath); err == nil {
		s.log("promote_failed_conflict", map[string]interface{}{
			"scenario_id": scenarioID,
			"path":        productionPath,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' already exists in production. Delete or rename it first.", scenarioID),
		})
		return
	}

	// Stop the scenario before moving
	stopCmd := exec.Command("vrooli", "scenario", "stop", scenarioID)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		s.log("promote_stop_warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"output":      string(output),
			"error":       err.Error(),
		})
		// Continue anyway - scenario might not have been running
	}

	// Move the scenario directory
	if err := os.Rename(generatedPath, productionPath); err != nil {
		s.log("promote_failed_move", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to move scenario to production: %v", err),
		})
		return
	}

	s.log("scenario_promoted", map[string]interface{}{
		"scenario_id": scenarioID,
		"from":        generatedPath,
		"to":          productionPath,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"message":        "Scenario promoted to production successfully",
		"scenario_id":    scenarioID,
		"production_path": productionPath,
	})
}

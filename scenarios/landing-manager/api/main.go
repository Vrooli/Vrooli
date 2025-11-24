package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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

func (s *Server) handleCustomize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string   `json:"scenario_id"`
		Brief      string   `json:"brief"`
		Assets     []string `json:"assets"`
		Preview    bool     `json:"preview"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Stub implementation
	response := map[string]interface{}{
		"job_id":   fmt.Sprintf("job-%d", time.Now().Unix()),
		"status":   "queued",
		"agent_id": "agent-claude-code-1",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
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

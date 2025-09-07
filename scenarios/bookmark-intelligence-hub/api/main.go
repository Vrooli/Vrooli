package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "github.com/lib/pq"
)

// Configuration
type Config struct {
	Port           int    `json:"port"`
	DatabaseURL    string `json:"database_url"`
	HuginnURL      string `json:"huginn_url"`
	BrowserlessURL string `json:"browserless_url"`
	APIToken       string `json:"api_token"`
}

// Server represents the API server
type Server struct {
	config     *Config
	db         *sql.DB
	router     *mux.Router
	httpServer *http.Server
}

// Response structures
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Database  string    `json:"database"`
	Huginn    string    `json:"huginn"`
}

type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type ProfileResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
}

type StatsResponse struct {
	TotalBookmarks   int     `json:"total_bookmarks"`
	CategoriesCount  int     `json:"categories_count"`
	PendingActions   int     `json:"pending_actions"`
	AccuracyRate     float64 `json:"accuracy_rate"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty"`
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create server
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// Start server
	log.Printf("🔖 Bookmark Intelligence Hub API starting on port %d", config.Port)
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// NewServer creates a new API server instance
func NewServer(config *Config) (*Server, error) {
	// Connect to database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	server := &Server{
		config: config,
		db:     db,
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()
	
	// API versioning
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	// Profile management
	api.HandleFunc("/profiles", s.handleGetProfiles).Methods("GET")
	api.HandleFunc("/profiles", s.handleCreateProfile).Methods("POST")
	api.HandleFunc("/profiles/{id}", s.handleGetProfile).Methods("GET")
	api.HandleFunc("/profiles/{id}", s.handleUpdateProfile).Methods("PUT")
	api.HandleFunc("/profiles/{id}/stats", s.handleGetProfileStats).Methods("GET")
	
	// Bookmark management
	api.HandleFunc("/bookmarks/process", s.handleProcessBookmarks).Methods("POST")
	api.HandleFunc("/bookmarks/query", s.handleQueryBookmarks).Methods("GET")
	api.HandleFunc("/bookmarks/sync", s.handleSyncBookmarks).Methods("POST")
	
	// Category management
	api.HandleFunc("/categories", s.handleGetCategories).Methods("GET")
	api.HandleFunc("/categories", s.handleCreateCategory).Methods("POST")
	api.HandleFunc("/categories/{id}", s.handleUpdateCategory).Methods("PUT")
	api.HandleFunc("/categories/{id}", s.handleDeleteCategory).Methods("DELETE")
	
	// Action management
	api.HandleFunc("/actions", s.handleGetActions).Methods("GET")
	api.HandleFunc("/actions/approve", s.handleApproveActions).Methods("POST")
	api.HandleFunc("/actions/reject", s.handleRejectActions).Methods("POST")
	
	// Platform management
	api.HandleFunc("/platforms", s.handleGetPlatforms).Methods("GET")
	api.HandleFunc("/platforms/status", s.handleGetPlatformStatus).Methods("GET")
	api.HandleFunc("/platforms/{platform}/sync", s.handleSyncPlatform).Methods("POST")
	
	// Analytics
	api.HandleFunc("/analytics/metrics", s.handleGetMetrics).Methods("GET")
	
	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	
	s.router.Use(c.Handler)
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.authMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.router,
	}

	// Start server in a goroutine
	serverError := make(chan error, 1)
	go func() {
		serverError <- s.httpServer.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-serverError:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// Close cleans up server resources
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbStatus := "healthy"
	if err := s.db.Ping(); err != nil {
		dbStatus = "unhealthy: " + err.Error()
	}

	// Check Huginn connection (placeholder)
	huginnStatus := "unknown"
	
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Database:  dbStatus,
		Huginn:    huginnStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get all profiles
func (s *Server) handleGetProfiles(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement profile fetching from database
	profiles := []ProfileResponse{
		{
			ID:          "demo-profile-id",
			Name:        "Demo Profile",
			Description: "Default demo profile",
			Settings:    map[string]interface{}{"auto_approve": true},
			CreatedAt:   time.Now(),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

// Create new profile
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement profile creation
	s.sendError(w, http.StatusNotImplemented, "Profile creation not implemented yet")
}

// Get specific profile
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	
	// TODO: Implement profile fetching by ID
	profile := ProfileResponse{
		ID:          profileID,
		Name:        "Demo Profile",
		Description: "Default demo profile",
		Settings:    map[string]interface{}{"auto_approve": true},
		CreatedAt:   time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// Update profile
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement profile updating
	s.sendError(w, http.StatusNotImplemented, "Profile updating not implemented yet")
}

// Get profile stats
func (s *Server) handleGetProfileStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]
	
	// TODO: Implement actual stats calculation from database
	_ = profileID // Use profileID to query database
	
	stats := StatsResponse{
		TotalBookmarks:  1247,
		CategoriesCount: 8,
		PendingActions:  3,
		AccuracyRate:    92.5,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Process bookmarks
func (s *Server) handleProcessBookmarks(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement bookmark processing logic
	response := map[string]interface{}{
		"success":         true,
		"processed_count": 1,
		"message":        "Bookmark processing endpoint - implementation pending",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Query bookmarks
func (s *Server) handleQueryBookmarks(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement bookmark querying with filters
	bookmarks := []map[string]interface{}{
		{
			"id":          "1",
			"title":       "Sample Bookmark",
			"platform":    "reddit",
			"category":    "Programming",
			"created_at":  time.Now(),
		},
	}
	
	response := map[string]interface{}{
		"bookmarks":   bookmarks,
		"total_count": len(bookmarks),
		"categories":  []string{"Programming", "Recipes", "Fitness"},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Sync bookmarks
func (s *Server) handleSyncBookmarks(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement bookmark synchronization
	response := map[string]interface{}{
		"success":         true,
		"processed_count": 5,
		"message":        "Bookmark sync endpoint - implementation pending",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get categories
func (s *Server) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category fetching
	categories := []map[string]interface{}{
		{"name": "Programming", "count": 342},
		{"name": "Recipes", "count": 156},
		{"name": "Fitness", "count": 89},
		{"name": "Travel", "count": 67},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// Create category
func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category creation
	s.sendError(w, http.StatusNotImplemented, "Category creation not implemented yet")
}

// Update category
func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category updating
	s.sendError(w, http.StatusNotImplemented, "Category updating not implemented yet")
}

// Delete category
func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category deletion
	s.sendError(w, http.StatusNotImplemented, "Category deletion not implemented yet")
}

// Get actions
func (s *Server) handleGetActions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement action fetching
	actions := []map[string]interface{}{
		{
			"id":          "1",
			"title":       "Add Recipe to Recipe Book",
			"description": "Chocolate chip cookies from @BakeWithJoy",
			"status":      "pending",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

// Approve actions
func (s *Server) handleApproveActions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement action approval
	response := map[string]interface{}{
		"success":        true,
		"approved_count": 1,
		"message":       "Action approval endpoint - implementation pending",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Reject actions
func (s *Server) handleRejectActions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement action rejection
	response := map[string]interface{}{
		"success":       true,
		"rejected_count": 1,
		"message":      "Action rejection endpoint - implementation pending",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get platforms
func (s *Server) handleGetPlatforms(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement platform information fetching
	platforms := []map[string]interface{}{
		{
			"name":         "reddit",
			"display_name": "Reddit",
			"icon":         "fab fa-reddit",
			"color":        "#FF4500",
			"supported":    true,
		},
		{
			"name":         "twitter",
			"display_name": "X (Twitter)",
			"icon":         "fab fa-twitter",
			"color":        "#1DA1F2",
			"supported":    true,
		},
		{
			"name":         "tiktok",
			"display_name": "TikTok",
			"icon":         "fab fa-tiktok",
			"color":        "#FE2C55",
			"supported":    true,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(platforms)
}

// Get platform status
func (s *Server) handleGetPlatformStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement platform status checking
	statuses := []map[string]interface{}{
		{
			"name":      "reddit",
			"connected": false,
			"last_sync": nil,
		},
		{
			"name":      "twitter",
			"connected": false,
			"last_sync": nil,
		},
		{
			"name":      "tiktok",
			"connected": false,
			"last_sync": nil,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// Sync specific platform
func (s *Server) handleSyncPlatform(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	platform := vars["platform"]
	
	// TODO: Implement platform-specific sync
	response := map[string]interface{}{
		"success":         true,
		"platform":        platform,
		"processed_count": 10,
		"message":        fmt.Sprintf("Platform %s sync endpoint - implementation pending", platform),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get analytics metrics
func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement metrics collection
	metrics := map[string]interface{}{
		"total_bookmarks":     1247,
		"processing_accuracy": 92.5,
		"actions_executed":    156,
		"platform_breakdown": map[string]int{
			"reddit":  680,
			"twitter": 412,
			"tiktok":  155,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Middleware functions

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		
		// TODO: Implement proper authentication
		// For now, allow all requests
		next.ServeHTTP(w, r)
	})
}

// Helper functions

func (s *Server) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := ErrorResponse{
		Error:     http.StatusText(statusCode),
		Message:   message,
		Timestamp: time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// Configuration loading
func loadConfig() (*Config, error) {
	config := &Config{
		Port:           15200, // Default port
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/vrooli?sslmode=disable"),
		HuginnURL:      getEnv("HUGINN_URL", "http://localhost:3000"),
		BrowserlessURL: getEnv("BROWSERLESS_URL", "http://localhost:3001"),
		APIToken:       getEnv("BOOKMARK_API_TOKEN", ""),
	}
	
	// Override port if specified
	if portStr := os.Getenv("API_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		}
	}
	
	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
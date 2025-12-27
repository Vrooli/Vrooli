package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "vrooli-orchestrator"
	
	// Timeouts
	httpTimeout = 30 * time.Second
	
	// Database limits
	maxDBConnections = 10
	maxIdleConnections = 2
	connMaxLifetime = 5 * time.Minute
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[vrooli-orchestrator-api] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// HTTPError sends structured error response
func HTTPError(w http.ResponseWriter, message string, statusCode int, err error) {
	logger := NewLogger()
	logger.Error(fmt.Sprintf("HTTP %d: %s", statusCode, message), err)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResp := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().UTC(),
	}
	
	json.NewEncoder(w).Encode(errorResp)
}

// Profile represents a startup profile
type Profile struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	DisplayName      string                 `json:"display_name"`
	Description      string                 `json:"description"`
	Metadata         map[string]interface{} `json:"metadata"`
	Resources        []string               `json:"resources"`
	Scenarios        []string               `json:"scenarios"`
	AutoBrowser      []string               `json:"auto_browser"`
	EnvironmentVars  map[string]string      `json:"environment_vars"`
	IdleShutdown     *int                   `json:"idle_shutdown_minutes,omitempty"`
	Dependencies     []string               `json:"dependencies"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// OrchestratorService handles profile management and orchestration
type OrchestratorService struct {
	db                 *sql.DB
	logger             *Logger
	profileManager     *ProfileManager
	orchestratorManager *OrchestratorManager
}

// NewOrchestratorService creates a new orchestrator service
func NewOrchestratorService(db *sql.DB) *OrchestratorService {
	logger := NewLogger()
	profileManager := NewProfileManager(db, logger)
	orchestratorManager := NewOrchestratorManager(profileManager, logger)
	
	return &OrchestratorService{
		db:                  db,
		logger:             logger,
		profileManager:     profileManager,
		orchestratorManager: orchestratorManager,
	}
}

// ListProfiles returns all available profiles
func (o *OrchestratorService) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := o.profileManager.ListProfiles()
	if err != nil {
		HTTPError(w, "Failed to list profiles", http.StatusInternalServerError, err)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

// GetProfile returns a specific profile
func (o *OrchestratorService) GetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileName := vars["profileName"]
	
	if profileName == "" {
		HTTPError(w, "Profile name required", http.StatusBadRequest, nil)
		return
	}
	
	profile, err := o.profileManager.GetProfile(profileName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			HTTPError(w, "Profile not found", http.StatusNotFound, err)
		} else {
			HTTPError(w, "Failed to get profile", http.StatusInternalServerError, err)
		}
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// CreateProfile creates a new profile
func (o *OrchestratorService) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var profileData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&profileData); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	profile, err := o.profileManager.CreateProfile(profileData)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			HTTPError(w, "Profile already exists", http.StatusConflict, err)
		} else if strings.Contains(err.Error(), "required") {
			HTTPError(w, "Invalid profile data", http.StatusBadRequest, err)
		} else {
			HTTPError(w, "Failed to create profile", http.StatusInternalServerError, err)
		}
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// UpdateProfile updates an existing profile
func (o *OrchestratorService) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileName := vars["profileName"]
	
	if profileName == "" {
		HTTPError(w, "Profile name required", http.StatusBadRequest, nil)
		return
	}
	
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	profile, err := o.profileManager.UpdateProfile(profileName, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			HTTPError(w, "Profile not found", http.StatusNotFound, err)
		} else {
			HTTPError(w, "Failed to update profile", http.StatusInternalServerError, err)
		}
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// DeleteProfile deletes a profile
func (o *OrchestratorService) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileName := vars["profileName"]
	
	if profileName == "" {
		HTTPError(w, "Profile name required", http.StatusBadRequest, nil)
		return
	}
	
	err := o.profileManager.DeleteProfile(profileName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			HTTPError(w, "Profile not found", http.StatusNotFound, err)
		} else if strings.Contains(err.Error(), "cannot delete active") {
			HTTPError(w, "Cannot delete active profile", http.StatusConflict, err)
		} else {
			HTTPError(w, "Failed to delete profile", http.StatusInternalServerError, err)
		}
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Profile '%s' deleted successfully", profileName),
	})
}

// ActivateProfile activates a profile (starts resources and scenarios)
func (o *OrchestratorService) ActivateProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileName := vars["profileName"]
	
	if profileName == "" {
		HTTPError(w, "Profile name required", http.StatusBadRequest, nil)
		return
	}
	
	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		// If no JSON body, create empty request
		requestData = make(map[string]interface{})
	}
	
	o.logger.Info(fmt.Sprintf("Activating profile: %s", profileName))
	
	result, err := o.orchestratorManager.ActivateProfile(profileName, requestData)
	if err != nil {
		HTTPError(w, "Failed to activate profile", http.StatusInternalServerError, err)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeactivateProfile deactivates the current profile
func (o *OrchestratorService) DeactivateProfile(w http.ResponseWriter, r *http.Request) {
	o.logger.Info("Deactivating current profile")
	
	result, err := o.orchestratorManager.GetDeactivationResult()
	if err != nil {
		HTTPError(w, "Failed to deactivate profile", http.StatusInternalServerError, err)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetStatus returns current orchestrator status
func (o *OrchestratorService) GetStatus(w http.ResponseWriter, r *http.Request) {
	// Get active profile
	activeProfile, err := o.profileManager.GetActiveProfile()
	if err != nil {
		o.logger.Error("Failed to get active profile for status", err)
		// Continue anyway - non-critical error
	}
	
	// Get system stats
	resourceCount := 0
	scenarioCount := 0
	
	if activeProfile != nil {
		resourceCount = len(activeProfile.Resources)
		scenarioCount = len(activeProfile.Scenarios)
	}
	
	status := map[string]interface{}{
		"service":        serviceName,
		"version":        apiVersion,
		"status":         "healthy",
		"timestamp":      time.Now().UTC(),
		"active_profile": activeProfile,
		"resource_count": resourceCount,
		"scenario_count": scenarioCount,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getResourcePort queries the port registry for a resource's port
func getResourcePort(resourceName string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"source ${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh && ports::get_resource_port %s",
		resourceName,
	))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to get port for %s, using default: %v", resourceName, err)
		// Fallback to defaults
		defaults := map[string]string{
			"n8n":        "5678",
			"postgres":   "5433",
			"browserless": "3000",
		}
		if port, ok := defaults[resourceName]; ok {
			return port
		}
		return "8080" // Generic fallback
	}
	return strings.TrimSpace(string(output))
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-orchestrator",
	}) {
		return // Process was re-exec'd after rebuild
	}

	logger := NewLogger()
	
	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		logger.Error("Failed to connect to database", err)
		os.Exit(1)
	}

	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	logger.Info("✅ Database connected successfully")
	
	// Initialize orchestrator service
	orchestrator := NewOrchestratorService(db)
	
	// Setup routes
	r := mux.NewRouter()
	
	// Health endpoint
	r.HandleFunc("/health", health.Handler()).Methods("GET")
	
	// Profile management endpoints
	r.HandleFunc("/api/v1/profiles", orchestrator.ListProfiles).Methods("GET")
	r.HandleFunc("/api/v1/profiles", orchestrator.CreateProfile).Methods("POST")
	r.HandleFunc("/api/v1/profiles/{profileName}", orchestrator.GetProfile).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{profileName}", orchestrator.UpdateProfile).Methods("PUT")
	r.HandleFunc("/api/v1/profiles/{profileName}", orchestrator.DeleteProfile).Methods("DELETE")
	
	// Profile activation endpoints
	r.HandleFunc("/api/v1/profiles/{profileName}/activate", orchestrator.ActivateProfile).Methods("POST")
	r.HandleFunc("/api/v1/profiles/current/deactivate", orchestrator.DeactivateProfile).Methods("POST")
	
	// System status endpoint
	r.HandleFunc("/api/v1/status", orchestrator.GetStatus).Methods("GET")
	
	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: r,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}
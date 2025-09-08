package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type App struct {
	DB                 *sql.DB
	DeviceController   *DeviceController
	SafetyValidator    *SafetyValidator
	CalendarScheduler  *CalendarScheduler
}

func main() {
	app := &App{}
	
	// Database connection
	dbHost := getEnv("POSTGRES_HOST", "")
	if dbHost == "" {
		log.Fatal("POSTGRES_HOST environment variable is required")
	}
	dbPort := getEnv("POSTGRES_PORT", "")
	if dbPort == "" {
		log.Fatal("POSTGRES_PORT environment variable is required")
	}
	dbUser := getEnv("POSTGRES_USER", "")
	if dbUser == "" {
		log.Fatal("POSTGRES_USER environment variable is required")
	}
	dbPassword := getEnv("POSTGRES_PASSWORD", "")
	if dbPassword == "" {
		log.Fatal("POSTGRES_PASSWORD environment variable is required")
	}
	dbName := getEnv("POSTGRES_DB", "")
	if dbName == "" {
		log.Fatal("POSTGRES_DB environment variable is required")
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	app.DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer app.DB.Close()
	
	// Test connection
	if err := app.DB.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	// Initialize components
	app.DeviceController = NewDeviceController(app.DB)
	app.SafetyValidator = NewSafetyValidator(app.DB)
	app.CalendarScheduler = NewCalendarScheduler(app.DB)
	app.CalendarScheduler.SetDeviceController(app.DeviceController)
	
	// Setup routes
	router := mux.NewRouter()
	
	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", app.HealthCheck).Methods("GET")
	
	// Device Control routes
	api.HandleFunc("/devices/control", app.ControlDevice).Methods("POST")
	api.HandleFunc("/devices/{id}/status", app.GetDeviceStatus).Methods("GET")
	api.HandleFunc("/devices", app.ListDevices).Methods("GET")
	
	// Safety Validation routes
	api.HandleFunc("/automations/validate", app.ValidateAutomation).Methods("POST")
	api.HandleFunc("/automations/{id}/safety-check", app.GetSafetyStatus).Methods("GET")
	
	// Calendar Automation routes
	api.HandleFunc("/calendar/trigger", app.HandleCalendarEvent).Methods("POST")
	api.HandleFunc("/contexts/{context}/activate", app.ActivateContext).Methods("POST")
	api.HandleFunc("/contexts/{context}/deactivate", app.DeactivateContext).Methods("POST")
	api.HandleFunc("/contexts/current", app.GetCurrentContext).Methods("GET")
	
	// Home Profiles routes
	api.HandleFunc("/profiles", app.GetProfiles).Methods("GET")
	api.HandleFunc("/profiles/{id}/permissions", app.GetProfilePermissions).Methods("GET")
	
	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	// Start server
	port := getEnv("API_PORT", "30500")
	log.Printf("Home Automation API server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (app *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":             "healthy",
		"device_controller":  "ready",
		"safety_validator":   "ready", 
		"calendar_scheduler": "ready",
	}
	json.NewEncoder(w).Encode(status)
}

// Device Control handlers
func (app *App) ControlDevice(w http.ResponseWriter, r *http.Request) {
	var req DeviceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := app.DeviceController.ControlDevice(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (app *App) GetDeviceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	status, err := app.DeviceController.GetDeviceStatus(r.Context(), deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(status)
}

func (app *App) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := app.DeviceController.ListDevices(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// Safety Validation handlers
func (app *App) ValidateAutomation(w http.ResponseWriter, r *http.Request) {
	var req AutomationValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := app.SafetyValidator.ValidateAutomation(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (app *App) GetSafetyStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	automationID := vars["id"]

	status, err := app.SafetyValidator.GetSafetyStatus(r.Context(), automationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(status)
}

// Calendar Automation handlers
func (app *App) HandleCalendarEvent(w http.ResponseWriter, r *http.Request) {
	var req CalendarEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := app.CalendarScheduler.HandleCalendarEvent(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (app *App) ActivateContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	context := vars["context"]

	err := app.CalendarScheduler.ActivateContext(r.Context(), context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"context": context,
		"status":  "activated",
	})
}

func (app *App) DeactivateContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	context := vars["context"]

	err := app.CalendarScheduler.DeactivateContext(r.Context(), context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"context": context,
		"status":  "deactivated",
	})
}

func (app *App) GetCurrentContext(w http.ResponseWriter, r *http.Request) {
	context, err := app.CalendarScheduler.GetCurrentContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(context)
}

// Profile handlers
func (app *App) GetProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := app.DeviceController.GetProfiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

func (app *App) GetProfilePermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	permissions, err := app.DeviceController.GetProfilePermissions(r.Context(), profileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(permissions)
}
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	Port                   string
	PostgresURL           string
	QdrantURL             string
	AuthServiceURL        string
	NotificationServiceURL string
	OllamaURL             string
	NotificationProfileID  string
	NotificationAPIKey     string
	JWTSecret             string
}

// Database connection
var db *sql.DB

// Models
type Event struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time"`
	Timezone         string                 `json:"timezone"`
	Location         string                 `json:"location"`
	EventType        string                 `json:"event_type"`
	Status           string                 `json:"status"`
	Metadata         map[string]interface{} `json:"metadata"`
	AutomationConfig map[string]interface{} `json:"automation_config"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type CreateEventRequest struct {
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	StartTime        string                 `json:"start_time"`
	EndTime          string                 `json:"end_time"`
	Timezone         string                 `json:"timezone"`
	Location         string                 `json:"location"`
	EventType        string                 `json:"event_type"`
	Metadata         map[string]interface{} `json:"metadata"`
	AutomationConfig map[string]interface{} `json:"automation_config"`
	Recurrence       *RecurrenceRequest     `json:"recurrence"`
	Reminders        []ReminderRequest      `json:"reminders"`
}

type RecurrenceRequest struct {
	Pattern      string     `json:"pattern"`
	Interval     int        `json:"interval"`
	DaysOfWeek   []int      `json:"days_of_week"`
	EndDate      *time.Time `json:"end_date"`
	MaxOccurrences *int     `json:"max_occurrences"`
}

type ReminderRequest struct {
	MinutesBefore    int    `json:"minutes_before"`
	NotificationType string `json:"type"`
}

type ChatRequest struct {
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context"`
}

type ChatResponse struct {
	Response            string                   `json:"response"`
	SuggestedActions    []SuggestedAction       `json:"suggested_actions"`
	RequiresConfirmation bool                    `json:"requires_confirmation"`
	Context             map[string]interface{}   `json:"context"`
}

type SuggestedAction struct {
	Action     string                 `json:"action"`
	Confidence float64                `json:"confidence"`
	Parameters map[string]interface{} `json:"parameters"`
}

type User struct {
	ID           string `json:"id"`
	AuthUserID   string `json:"auth_user_id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	Timezone     string `json:"timezone"`
}

// Initialize configuration
func initConfig() *Config {
	// Load .env file if it exists
	godotenv.Load()

	return &Config{
		Port:                   getEnvOrDefault("PORT", "3300"),
		PostgresURL:           getEnvOrDefault("POSTGRES_URL", "postgres://vrooli:password@localhost:5433/calendar_system?sslmode=disable"),
		QdrantURL:             getEnvOrDefault("QDRANT_URL", "http://localhost:6333"),
		AuthServiceURL:        getEnvOrDefault("AUTH_SERVICE_URL", "http://localhost:3250"),
		NotificationServiceURL: getEnvOrDefault("NOTIFICATION_SERVICE_URL", "http://localhost:28100"),
		OllamaURL:             getEnvOrDefault("OLLAMA_URL", "http://localhost:11434"),
		NotificationProfileID:  getEnvOrDefault("NOTIFICATION_PROFILE_ID", "calendar-system-prod"),
		NotificationAPIKey:     getEnvOrDefault("NOTIFICATION_API_KEY", ""),
		JWTSecret:             getEnvOrDefault("JWT_SECRET", "calendar-secret-key"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Initialize database connection
func initDatabase(config *Config) error {
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection established")
	return nil
}

// Middleware for authentication
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Validate token with scenario-authenticator
		user, err := validateToken(token)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add user to request context
		r.Header.Set("X-User-ID", user.ID)
		r.Header.Set("X-Auth-User-ID", user.AuthUserID)
		r.Header.Set("X-User-Email", user.Email)

		next.ServeHTTP(w, r)
	})
}

// Validate token with scenario-authenticator service
func validateToken(token string) (*User, error) {
	// This would make an HTTP request to scenario-authenticator
	// For now, return a mock user for development
	return &User{
		ID:          "test-user-id",
		AuthUserID:  "auth-user-123",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Timezone:    "UTC",
	}, nil
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services": map[string]string{
			"database": "connected",
			"api":      "running",
		},
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["services"].(map[string]string)["database"] = "error: " + err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Event handlers
func createEventHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Parse timestamps
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Invalid start_time format: "+err.Error(), http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Invalid end_time format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create event in database
	eventID := uuid.New().String()
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	metadataJSON, _ := json.Marshal(req.Metadata)
	automationJSON, _ := json.Marshal(req.AutomationConfig)

	query := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, timezone, location, event_type, metadata, automation_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.Exec(query, eventID, userID, req.Title, req.Description, startTime, endTime, timezone, req.Location, req.EventType, metadataJSON, automationJSON)
	if err != nil {
		log.Printf("Error creating event: %v", err)
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	// Create reminders if specified
	reminderCount := 0
	for _, reminder := range req.Reminders {
		reminderID := uuid.New().String()
		reminderTime := startTime.Add(time.Duration(-reminder.MinutesBefore) * time.Minute)
		
		reminderQuery := `
			INSERT INTO event_reminders (id, event_id, minutes_before, notification_type, scheduled_time)
			VALUES ($1, $2, $3, $4, $5)
		`
		
		_, err = db.Exec(reminderQuery, reminderID, eventID, reminder.MinutesBefore, reminder.NotificationType, reminderTime)
		if err != nil {
			log.Printf("Error creating reminder: %v", err)
			// Continue with other reminders even if one fails
		} else {
			reminderCount++
		}
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"event": map[string]interface{}{
			"id":         eventID,
			"title":      req.Title,
			"start_time": startTime,
			"end_time":   endTime,
		},
		"reminders_scheduled": reminderCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func listEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	eventType := r.URL.Query().Get("event_type")
	search := r.URL.Query().Get("search")
	
	// Build query
	query := `
		SELECT id, user_id, title, description, start_time, end_time, timezone, location, event_type, status, created_at, updated_at
		FROM events 
		WHERE user_id = $1 AND status = 'active'
	`
	args := []interface{}{userID}
	argCount := 1

	if startDate != "" {
		argCount++
		query += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, startDate)
	}

	if endDate != "" {
		argCount++
		query += fmt.Sprintf(" AND start_time <= $%d", argCount)
		args = append(args, endDate)
	}

	if eventType != "" {
		argCount++
		query += fmt.Sprintf(" AND event_type = $%d", argCount)
		args = append(args, eventType)
	}

	if search != "" {
		argCount++
		query += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY start_time ASC LIMIT 100"

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying events: %v", err)
		http.Error(w, "Failed to query events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(
			&event.ID, &event.UserID, &event.Title, &event.Description,
			&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
			&event.EventType, &event.Status, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}
		events = append(events, event)
	}

	response := map[string]interface{}{
		"events":      events,
		"total_count": len(events),
		"has_more":    len(events) >= 100,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func scheduleChatHandler(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// For now, return a simple response
	// In a full implementation, this would use Ollama/Claude for NLP
	response := ChatResponse{
		Response: "I understand you want to schedule something. Can you provide more details about what you'd like to schedule?",
		SuggestedActions: []SuggestedAction{
			{
				Action:     "create_event",
				Confidence: 0.7,
				Parameters: map[string]interface{}{
					"title":      "New Event",
					"start_time": time.Now().Add(time.Hour).Format(time.RFC3339),
				},
			},
		},
		RequiresConfirmation: true,
		Context: map[string]interface{}{
			"conversation_id": uuid.New().String(),
			"user_message":    req.Message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func scheduleOptimizeHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for AI-powered schedule optimization
	response := map[string]interface{}{
		"suggestions": []map[string]interface{}{
			{
				"description":      "Move morning meeting to afternoon to create a 2-hour block",
				"affected_events":  []string{},
				"proposed_changes": []map[string]interface{}{},
				"confidence_score": 0.8,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	log.Println("Starting Calendar API...")

	// Initialize configuration
	config := initConfig()

	// Initialize database
	if err := initDatabase(config); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Setup routes
	router := mux.NewRouter()

	// Health check (no auth required)
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API routes (with auth middleware)
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(authMiddleware)

	// Event management
	api.HandleFunc("/events", createEventHandler).Methods("POST")
	api.HandleFunc("/events", listEventsHandler).Methods("GET")
	
	// Schedule management
	api.HandleFunc("/schedule/chat", scheduleChatHandler).Methods("POST")
	api.HandleFunc("/schedule/optimize", scheduleOptimizeHandler).Methods("POST")

	// Setup CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Setup logging middleware
	loggedRouter := handlers.LoggingHandler(os.Stdout, corsHandler.Handler(router))

	// Start server
	address := ":" + config.Port
	log.Printf("Calendar API server starting on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)
	
	if err := http.ListenAndServe(address, loggedRouter); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
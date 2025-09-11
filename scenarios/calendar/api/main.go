package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
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
var calendarProcessor *CalendarProcessor
var nlpProcessor *NLPProcessor

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

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT
		if port == "" {
			log.Fatal("❌ API_PORT or PORT environment variable is required")
		}
	}

	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// External service URLs - REQUIRED, no defaults
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		log.Fatal("❌ QDRANT_URL environment variable is required")
	}

	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		log.Fatal("❌ AUTH_SERVICE_URL environment variable is required")
	}

	notificationServiceURL := os.Getenv("NOTIFICATION_SERVICE_URL")
	if notificationServiceURL == "" {
		log.Fatal("❌ NOTIFICATION_SERVICE_URL environment variable is required")
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("❌ JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:                   port,
		PostgresURL:           postgresURL,
		QdrantURL:             qdrantURL,
		AuthServiceURL:        authServiceURL,
		NotificationServiceURL: notificationServiceURL,
		OllamaURL:             ollamaURL,
		NotificationProfileID:  os.Getenv("NOTIFICATION_PROFILE_ID"), // Optional
		NotificationAPIKey:     os.Getenv("NOTIFICATION_API_KEY"), // Optional
		JWTSecret:             jwtSecret,
	}
}

// getEnvOrDefault and requireEnv removed to prevent hardcoded defaults

// Initialize database connection
func initDatabase(config *Config) error {
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		return fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
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
	// Get auth service URL from environment (required)
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		return nil, fmt.Errorf("AUTH_SERVICE_URL environment variable is required")
	}

	// Create validation request
	req, err := http.NewRequest("GET", authServiceURL+"/api/v1/auth/validate", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Make request to scenario-authenticator
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token")
	}

	// Parse response
	var authResponse struct {
		UserID      string `json:"user_id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Create user from auth response
	return &User{
		ID:          authResponse.UserID,
		AuthUserID:  authResponse.UserID,
		Email:       authResponse.Email,
		DisplayName: authResponse.DisplayName,
		Timezone:    "UTC", // Default timezone, could be fetched from user profile
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

	// Use NLP processor to handle natural language scheduling
	ctx := r.Context()
	response, err := nlpProcessor.ProcessSchedulingRequest(ctx, userID, req.Message)
	if err != nil {
		log.Printf("Error processing chat request: %v", err)
		// Return a fallback response on error
		response = &ChatResponse{
			Response: "I'm having trouble understanding your request. Could you try rephrasing it?",
			SuggestedActions: []SuggestedAction{
				{
					Action:     "create_event",
					Confidence: 0.5,
				},
				{
					Action:     "list_events",
					Confidence: 0.5,
				},
			},
		}
	}

	// Add conversation context
	if response.Context == nil {
		response.Context = make(map[string]interface{})
	}
	response.Context["conversation_id"] = uuid.New().String()
	response.Context["user_message"] = req.Message
	response.Context["timestamp"] = time.Now().UTC()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func scheduleOptimizeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OptimizationGoal string    `json:"optimization_goal"` // "minimize_gaps", "maximize_focus_time", "balance_workload"
		StartDate        time.Time `json:"start_date"`
		EndDate          time.Time `json:"end_date"`
		Constraints      map[string]interface{} `json:"constraints"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Fetch user's events in the date range
	query := `
		SELECT id, title, start_time, end_time, event_type, metadata
		FROM events
		WHERE user_id = $1 AND start_time >= $2 AND end_time <= $3
		  AND status != 'cancelled'
		ORDER BY start_time`

	rows, err := db.Query(query, userID, req.StartDate, req.EndDate)
	if err != nil {
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var metadataJSON []byte
		if err := rows.Scan(&e.ID, &e.Title, &e.StartTime, &e.EndTime, &e.EventType, &metadataJSON); err != nil {
			continue
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}
		events = append(events, e)
	}

	// Analyze schedule and generate optimization suggestions
	suggestions := analyzeSchedule(events, req.OptimizationGoal)

	response := map[string]interface{}{
		"optimization_goal": req.OptimizationGoal,
		"current_efficiency": calculateScheduleEfficiency(events),
		"suggestions": suggestions,
		"potential_time_saved": calculateTimeSaved(suggestions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions for schedule optimization
func analyzeSchedule(events []Event, goal string) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	// Identify gaps and overlaps
	for i := 0; i < len(events)-1; i++ {
		gap := events[i+1].StartTime.Sub(events[i].EndTime)
		
		if gap > 30*time.Minute && gap < 2*time.Hour {
			// Suggest consolidating small gaps
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "consolidate_gap",
				"description": fmt.Sprintf("Move %s to eliminate %v gap", events[i+1].Title, gap),
				"event_id":    events[i+1].ID,
				"proposed_start": events[i].EndTime,
				"confidence":  0.7,
			})
		}
	}

	// Suggest batching similar events
	eventTypes := make(map[string][]Event)
	for _, e := range events {
		eventTypes[e.EventType] = append(eventTypes[e.EventType], e)
	}

	for eventType, typeEvents := range eventTypes {
		if len(typeEvents) > 2 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "batch_similar",
				"description": fmt.Sprintf("Batch %d %s events together", len(typeEvents), eventType),
				"event_ids":   getEventIDs(typeEvents),
				"confidence":  0.8,
			})
		}
	}

	return suggestions
}

func calculateScheduleEfficiency(events []Event) float64 {
	if len(events) == 0 {
		return 1.0
	}

	totalTime := events[len(events)-1].EndTime.Sub(events[0].StartTime)
	busyTime := time.Duration(0)
	for _, e := range events {
		busyTime += e.EndTime.Sub(e.StartTime)
	}

	if totalTime > 0 {
		return float64(busyTime) / float64(totalTime)
	}
	return 0
}

func calculateTimeSaved(suggestions []map[string]interface{}) int {
	// Estimate time saved in minutes
	totalMinutes := 0
	for _, s := range suggestions {
		if s["type"] == "consolidate_gap" {
			totalMinutes += 15 // Estimate 15 minutes saved per gap consolidation
		} else if s["type"] == "batch_similar" {
			totalMinutes += 30 // Estimate 30 minutes saved per batching
		}
	}
	return totalMinutes
}

func getEventIDs(events []Event) []string {
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}

// processRemindersHandler manually triggers reminder processing
func processRemindersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	if err := calendarProcessor.ProcessReminders(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to process reminders: %v", err), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"status": "success",
		"message": "Reminders processed successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start calendar

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	log.Println("Starting Calendar API...")

	// Initialize configuration
	config := initConfig()

	// Initialize database
	if err := initDatabase(config); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Initialize calendar processor
	calendarProcessor = NewCalendarProcessor(db, config)
	
	// Initialize NLP processor
	nlpProcessor = NewNLPProcessor(config.OllamaURL, db, config)
	
	// Start reminder processor in background
	ctx := context.Background()
	calendarProcessor.StartReminderProcessor(ctx)

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
	
	// Reminder management (replacing n8n workflows)
	api.HandleFunc("/reminders/process", processRemindersHandler).Methods("POST")

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
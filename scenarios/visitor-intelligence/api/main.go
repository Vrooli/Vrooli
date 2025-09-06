package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	Port        string
	PostgresURL string
	RedisAddr   string
	RedisDB     int
}

// Database connections
var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

// Data structures
type VisitorEvent struct {
	Fingerprint string                 `json:"fingerprint"`
	SessionID   string                 `json:"session_id"`
	Scenario    string                 `json:"scenario"`
	EventType   string                 `json:"event_type"`
	PageURL     string                 `json:"page_url"`
	Timestamp   time.Time              `json:"timestamp"`
	Properties  map[string]interface{} `json:"properties"`
	IPAddress   string                 `json:"-"`
}

type Visitor struct {
	ID                    string    `json:"id"`
	Fingerprint           string    `json:"fingerprint"`
	FirstSeen             time.Time `json:"first_seen"`
	LastSeen              time.Time `json:"last_seen"`
	SessionCount          int       `json:"session_count"`
	Identified            bool      `json:"identified"`
	Email                 *string   `json:"email"`
	UserAgent             *string   `json:"user_agent"`
	IPAddress             *string   `json:"ip_address"`
	Timezone              *string   `json:"timezone"`
	Language              *string   `json:"language"`
	ScreenResolution      *string   `json:"screen_resolution"`
	DeviceType            *string   `json:"device_type"`
	TotalPageViews        int       `json:"total_page_views"`
	TotalSessionDuration  int       `json:"total_session_duration"`
	Tags                  []string  `json:"tags"`
}

type VisitorSession struct {
	ID           string     `json:"id"`
	VisitorID    string     `json:"visitor_id"`
	Scenario     string     `json:"scenario"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	Duration     *int       `json:"duration"`
	PageViews    int        `json:"page_views"`
	EntryPage    *string    `json:"entry_page"`
	ExitPage     *string    `json:"exit_page"`
	Referrer     *string    `json:"referrer"`
	UTMSource    *string    `json:"utm_source"`
	UTMMedium    *string    `json:"utm_medium"`
	UTMCampaign  *string    `json:"utm_campaign"`
	IPAddress    *string    `json:"ip_address"`
	DeviceType   *string    `json:"device_type"`
	Bounce       bool       `json:"bounce"`
	Engaged      bool       `json:"engaged"`
}

type ScenarioAnalytics struct {
	Scenario          string  `json:"scenario"`
	UniqueVisitors    int     `json:"unique_visitors"`
	TotalSessions     int     `json:"total_sessions"`
	AvgSessionDuration float64 `json:"avg_session_duration"`
	TotalPageViews    int     `json:"total_page_views"`
	BounceRate        float64 `json:"bounce_rate"`
	IdentifiedVisitors int     `json:"identified_visitors"`
}

type TrackingResponse struct {
	Success   bool   `json:"success"`
	VisitorID string `json:"visitor_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// Initialize configuration
func initConfig() *Config {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	pgPort := os.Getenv("POSTGRES_PORT")
	if pgPort == "" {
		pgPort = "5432"
	}
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	if pgPassword == "" {
		pgPassword = "postgres"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	return &Config{
		Port:        port,
		PostgresURL: fmt.Sprintf("postgres://postgres:%s@localhost:%s/postgres?sslmode=disable", pgPassword, pgPort),
		RedisAddr:   fmt.Sprintf("localhost:%s", redisPort),
		RedisDB:     0,
	}
}

// Initialize database connections
func initDatabase(config *Config) error {
	var err error

	// PostgreSQL connection
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Test PostgreSQL connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Redis connection
	rdb = redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
		DB:   config.RedisDB,
	})

	// Test Redis connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Println("Database connections initialized successfully")
	return nil
}

// Get client IP address
func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// Get or create visitor
func getOrCreateVisitor(fingerprint, userAgent, ipAddress string) (*Visitor, error) {
	// Check Redis cache first
	cacheKey := "visitor:" + fingerprint
	cached, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var visitor Visitor
		if json.Unmarshal([]byte(cached), &visitor) == nil {
			return &visitor, nil
		}
	}

	// Check database
	var visitor Visitor
	query := `
		SELECT id, fingerprint, first_seen, last_seen, session_count, identified,
			   email, user_agent, ip_address, timezone, language, 
			   screen_resolution, device_type, total_page_views, 
			   total_session_duration, tags
		FROM visitors 
		WHERE fingerprint = $1
	`
	
	row := db.QueryRow(query, fingerprint)
	err = row.Scan(
		&visitor.ID, &visitor.Fingerprint, &visitor.FirstSeen, &visitor.LastSeen,
		&visitor.SessionCount, &visitor.Identified, &visitor.Email, &visitor.UserAgent,
		&visitor.IPAddress, &visitor.Timezone, &visitor.Language, &visitor.ScreenResolution,
		&visitor.DeviceType, &visitor.TotalPageViews, &visitor.TotalSessionDuration, &visitor.Tags,
	)

	if err == sql.ErrNoRows {
		// Create new visitor
		visitor = Visitor{
			ID:           uuid.New().String(),
			Fingerprint:  fingerprint,
			FirstSeen:    time.Now(),
			LastSeen:     time.Now(),
			SessionCount: 1,
			Identified:   false,
		}

		if userAgent != "" {
			visitor.UserAgent = &userAgent
		}
		if ipAddress != "" {
			visitor.IPAddress = &ipAddress
		}

		insertQuery := `
			INSERT INTO visitors (id, fingerprint, first_seen, last_seen, session_count, 
								  identified, user_agent, ip_address)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = db.Exec(insertQuery, visitor.ID, visitor.Fingerprint, visitor.FirstSeen,
			visitor.LastSeen, visitor.SessionCount, visitor.Identified, visitor.UserAgent, visitor.IPAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to create visitor: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query visitor: %v", err)
	}

	// Cache visitor data for 1 hour
	visitorJSON, _ := json.Marshal(visitor)
	rdb.Set(ctx, cacheKey, visitorJSON, time.Hour)

	return &visitor, nil
}

// Get or create session
func getOrCreateSession(visitorID, scenario, ipAddress string, properties map[string]interface{}) (string, error) {
	// Check for active session in Redis
	sessionKey := fmt.Sprintf("session:%s:%s", visitorID, scenario)
	sessionID, err := rdb.Get(ctx, sessionKey).Result()
	if err == nil {
		// Extend session timeout
		rdb.Expire(ctx, sessionKey, 30*time.Minute)
		return sessionID, nil
	}

	// Create new session
	sessionID = uuid.New().String()
	session := VisitorSession{
		ID:        sessionID,
		VisitorID: visitorID,
		Scenario:  scenario,
		StartTime: time.Now(),
	}

	if ipAddress != "" {
		session.IPAddress = &ipAddress
	}

	// Extract UTM parameters from properties
	if utm, ok := properties["utm_source"].(string); ok {
		session.UTMSource = &utm
	}
	if utm, ok := properties["utm_medium"].(string); ok {
		session.UTMMedium = &utm
	}
	if utm, ok := properties["utm_campaign"].(string); ok {
		session.UTMCampaign = &utm
	}
	if ref, ok := properties["referrer"].(string); ok {
		session.Referrer = &ref
	}

	// Insert session
	insertQuery := `
		INSERT INTO visitor_sessions (id, visitor_id, scenario, start_time, ip_address, 
									  referrer, utm_source, utm_medium, utm_campaign)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = db.Exec(insertQuery, session.ID, session.VisitorID, session.Scenario,
		session.StartTime, session.IPAddress, session.Referrer, session.UTMSource,
		session.UTMMedium, session.UTMCampaign)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}

	// Cache session ID for 30 minutes
	rdb.Set(ctx, sessionKey, sessionID, 30*time.Minute)

	return sessionID, nil
}

// Track visitor event
func trackEvent(event *VisitorEvent) error {
	// Insert event into database
	propertiesJSON, _ := json.Marshal(event.Properties)
	
	insertQuery := `
		INSERT INTO visitor_events (visitor_id, session_id, scenario, event_type, 
									page_url, timestamp, properties)
		VALUES ((SELECT id FROM visitors WHERE fingerprint = $1), $2, $3, $4, $5, $6, $7)
	`
	
	_, err := db.Exec(insertQuery, event.Fingerprint, event.SessionID, event.Scenario,
		event.EventType, event.PageURL, event.Timestamp, string(propertiesJSON))
	
	return err
}

// HTTP Handlers

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	// Check PostgreSQL
	if err := db.Ping(); err != nil {
		status.Services["postgres"] = "unhealthy: " + err.Error()
		status.Status = "degraded"
	} else {
		status.Services["postgres"] = "healthy"
	}

	// Check Redis
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		status.Services["redis"] = "unhealthy: " + err.Error()
		status.Status = "degraded"
	} else {
		status.Services["redis"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	if status.Status != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(status)
}

// Visitor tracking endpoint
func trackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event VisitorEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		response := TrackingResponse{Success: false, Message: "Invalid JSON"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set timestamp and IP
	event.Timestamp = time.Now()
	event.IPAddress = getClientIP(r)
	
	// Validate required fields
	if event.Fingerprint == "" || event.EventType == "" || event.Scenario == "" {
		response := TrackingResponse{Success: false, Message: "Missing required fields"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get or create visitor
	visitor, err := getOrCreateVisitor(event.Fingerprint, r.Header.Get("User-Agent"), event.IPAddress)
	if err != nil {
		log.Printf("Error getting/creating visitor: %v", err)
		response := TrackingResponse{Success: false, Message: "Internal error"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get or create session
	if event.SessionID == "" {
		event.SessionID, err = getOrCreateSession(visitor.ID, event.Scenario, event.IPAddress, event.Properties)
		if err != nil {
			log.Printf("Error getting/creating session: %v", err)
			response := TrackingResponse{Success: false, Message: "Internal error"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Track the event
	if err := trackEvent(&event); err != nil {
		log.Printf("Error tracking event: %v", err)
		response := TrackingResponse{Success: false, Message: "Internal error"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success response
	response := TrackingResponse{Success: true, VisitorID: visitor.ID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Get visitor profile
func getVisitorHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	visitorID := vars["id"]

	query := `
		SELECT id, fingerprint, first_seen, last_seen, session_count, identified,
			   email, user_agent, ip_address, timezone, language, 
			   screen_resolution, device_type, total_page_views, 
			   total_session_duration, COALESCE(tags, '{}')
		FROM visitors 
		WHERE id = $1
	`
	
	var visitor Visitor
	row := db.QueryRow(query, visitorID)
	err := row.Scan(
		&visitor.ID, &visitor.Fingerprint, &visitor.FirstSeen, &visitor.LastSeen,
		&visitor.SessionCount, &visitor.Identified, &visitor.Email, &visitor.UserAgent,
		&visitor.IPAddress, &visitor.Timezone, &visitor.Language, &visitor.ScreenResolution,
		&visitor.DeviceType, &visitor.TotalPageViews, &visitor.TotalSessionDuration, &visitor.Tags,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Visitor not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying visitor: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(visitor)
}

// Get scenario analytics
func getAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenario := vars["scenario"]

	// Parse timeframe parameter
	timeframe := r.URL.Query().Get("timeframe")
	var since time.Time
	switch timeframe {
	case "1d":
		since = time.Now().AddDate(0, 0, -1)
	case "7d":
		since = time.Now().AddDate(0, 0, -7)
	case "30d":
		since = time.Now().AddDate(0, 0, -30)
	case "90d":
		since = time.Now().AddDate(0, 0, -90)
	default:
		since = time.Now().AddDate(0, 0, -7) // Default to 7 days
	}

	query := `
		SELECT 
			COUNT(DISTINCT vs.visitor_id) as unique_visitors,
			COUNT(vs.id) as total_sessions,
			COALESCE(AVG(vs.duration), 0) as avg_session_duration,
			COALESCE(SUM(vs.page_views), 0) as total_page_views,
			COALESCE(AVG(CASE WHEN vs.bounce THEN 1.0 ELSE 0.0 END), 0) as bounce_rate,
			COUNT(CASE WHEN v.identified THEN 1 END) as identified_visitors
		FROM visitor_sessions vs
		LEFT JOIN visitors v ON vs.visitor_id = v.id
		WHERE vs.scenario = $1 AND vs.start_time >= $2
	`

	var analytics ScenarioAnalytics
	row := db.QueryRow(query, scenario, since)
	err := row.Scan(
		&analytics.UniqueVisitors, &analytics.TotalSessions, &analytics.AvgSessionDuration,
		&analytics.TotalPageViews, &analytics.BounceRate, &analytics.IdentifiedVisitors,
	)

	if err != nil {
		log.Printf("Error querying analytics: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	analytics.Scenario = scenario
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// Serve tracking script
func trackerScriptHandler(w http.ResponseWriter, r *http.Request) {
	// Read the tracking script file
	scriptPath := filepath.Join("..", "ui", "tracker.js")
	scriptContent, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		log.Printf("Error reading tracker script: %v", err)
		http.Error(w, "Tracker script not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Write(scriptContent)
}

// Main function
func main() {
	config := initConfig()

	// Initialize database connections
	if err := initDatabase(config); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	defer rdb.Close()

	// Setup routes
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/visitor/track", trackHandler).Methods("POST")
	api.HandleFunc("/visitor/{id}", getVisitorHandler).Methods("GET")
	api.HandleFunc("/analytics/scenario/{scenario}", getAnalyticsHandler).Methods("GET")

	// Utility routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/tracker.js", trackerScriptHandler).Methods("GET")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	log.Printf("Visitor Intelligence API starting on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)
	log.Printf("Tracking endpoint: http://localhost:%s/api/v1/visitor/track", config.Port)
	log.Printf("Tracking script: http://localhost:%s/tracker.js", config.Port)

	log.Fatal(http.ListenAndServe(":"+config.Port, handler))
}
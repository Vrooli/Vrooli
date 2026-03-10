package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Server wires the HTTP router and database connection
type Server struct {
	db     *sql.DB
	router *mux.Router
}

// NewServer initializes database and routes
func NewServer(db *sql.DB) *Server {
	srv := &Server{
		db:     db,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().
		Version("1.0.0").
		Check(&sqliteChecker{db: s.db}, health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Events API - P0-001, P0-003
	s.router.HandleFunc("/api/v1/events", s.createEvent).Methods("POST")
	s.router.HandleFunc("/api/v1/events", s.queryEvents).Methods("GET")
	s.router.HandleFunc("/api/v1/events/{id}", s.getEvent).Methods("GET")

	// Domains API - P0-002
	s.router.HandleFunc("/api/v1/domains", s.registerDomain).Methods("POST")
	s.router.HandleFunc("/api/v1/domains", s.listDomains).Methods("GET")
	s.router.HandleFunc("/api/v1/domains/{name}", s.getDomain).Methods("GET")
	s.router.HandleFunc("/api/v1/domains/{name}", s.updateDomain).Methods("PATCH")
	s.router.HandleFunc("/api/v1/domains/{name}/health", s.getDomainHealth).Methods("GET")

	// Statistics API - P0-003, P0-004
	s.router.HandleFunc("/api/v1/stats/timeline", s.getTimeline).Methods("GET")
	s.router.HandleFunc("/api/v1/stats/summary", s.getSummary).Methods("GET")
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// =============================================================================
// SQLite Health Checker
// =============================================================================

type sqliteChecker struct {
	db *sql.DB
}

func (c *sqliteChecker) Check(ctx context.Context) health.CheckResult {
	if c.db == nil {
		return health.CheckResult{
			Name:      "database",
			Connected: false,
			Error:     fmt.Errorf("not configured"),
		}
	}
	start := time.Now()
	err := c.db.PingContext(ctx)
	latency := time.Since(start)

	if err != nil {
		return health.CheckResult{
			Name:      "database",
			Connected: false,
			Latency:   latency,
			Error:     err,
		}
	}

	return health.CheckResult{
		Name:      "database",
		Connected: true,
		Latency:   latency,
		Database:  "lifestyle.db",
	}
}

// =============================================================================
// Event Types (P0-001)
// =============================================================================

// Event represents a cross-domain event with JSON payload
type Event struct {
	ID            string          `json:"id"`
	Timestamp     string          `json:"timestamp"`
	Domain        string          `json:"domain"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	IsIntervention bool           `json:"is_intervention"`
	HypothesisID  *string         `json:"hypothesis_id,omitempty"`
	CreatedAt     string          `json:"created_at"`
}

// CreateEventRequest is the request body for creating an event
type CreateEventRequest struct {
	Domain        string          `json:"domain"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Timestamp     *string         `json:"timestamp,omitempty"`
	IsIntervention bool           `json:"is_intervention"`
	HypothesisID  *string         `json:"hypothesis_id,omitempty"`
}

// =============================================================================
// Domain Types (P0-002)
// =============================================================================

// Domain represents a registered domain scenario
type Domain struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Status       string   `json:"status"` // active, inactive, unhealthy
	HealthURL    string   `json:"health_url,omitempty"`
	LastHealthAt *string  `json:"last_health_at,omitempty"`
	RegisteredAt string   `json:"registered_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// RegisterDomainRequest is the request body for registering a domain
type RegisterDomainRequest struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	HealthURL    string   `json:"health_url,omitempty"`
}

// =============================================================================
// Event Handlers
// =============================================================================

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Domain == "" || req.EventType == "" {
		writeError(w, http.StatusBadRequest, "domain and event_type are required")
		return
	}

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	timestamp := now
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	payload := []byte("{}")
	if req.Payload != nil {
		payload = req.Payload
	}

	_, err := s.db.Exec(`
		INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, timestamp, req.Domain, req.EventType, string(payload), req.IsIntervention, req.HypothesisID, now)
	if err != nil {
		log.Printf("Error inserting event: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	event := Event{
		ID:            id,
		Timestamp:     timestamp,
		Domain:        req.Domain,
		EventType:     req.EventType,
		Payload:       payload,
		IsIntervention: req.IsIntervention,
		HypothesisID:  req.HypothesisID,
		CreatedAt:     now,
	}

	writeJSON(w, http.StatusCreated, event)
}

func (s *Server) queryEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	domain := query.Get("domain")
	eventType := query.Get("event_type")
	startTime := query.Get("start")
	endTime := query.Get("end")
	limit := query.Get("limit")

	// Build query with filters
	sqlQuery := `SELECT id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at FROM events WHERE 1=1`
	args := []interface{}{}

	if domain != "" {
		sqlQuery += " AND domain = ?"
		args = append(args, domain)
	}
	if eventType != "" {
		sqlQuery += " AND event_type = ?"
		args = append(args, eventType)
	}
	if startTime != "" {
		sqlQuery += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		sqlQuery += " AND timestamp <= ?"
		args = append(args, endTime)
	}

	sqlQuery += " ORDER BY timestamp DESC"

	if limit != "" {
		sqlQuery += " LIMIT ?"
		args = append(args, limit)
	} else {
		sqlQuery += " LIMIT 100"
	}

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		log.Printf("Error querying events: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to query events")
		return
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var e Event
		var payload string
		var hypothesisID sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Domain, &e.EventType, &payload, &e.IsIntervention, &hypothesisID, &e.CreatedAt); err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}
		e.Payload = json.RawMessage(payload)
		if hypothesisID.Valid {
			e.HypothesisID = &hypothesisID.String
		}
		events = append(events, e)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var e Event
	var payload string
	var hypothesisID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at
		FROM events WHERE id = ?
	`, id).Scan(&e.ID, &e.Timestamp, &e.Domain, &e.EventType, &payload, &e.IsIntervention, &hypothesisID, &e.CreatedAt)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		log.Printf("Error getting event: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get event")
		return
	}

	e.Payload = json.RawMessage(payload)
	if hypothesisID.Valid {
		e.HypothesisID = &hypothesisID.String
	}

	writeJSON(w, http.StatusOK, e)
}

// =============================================================================
// Domain Handlers
// =============================================================================

func (s *Server) registerDomain(w http.ResponseWriter, r *http.Request) {
	var req RegisterDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "name and display_name are required")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	capabilities := "[]"
	if len(req.Capabilities) > 0 {
		capsJSON, _ := json.Marshal(req.Capabilities)
		capabilities = string(capsJSON)
	}

	// Upsert the domain
	_, err := s.db.Exec(`
		INSERT INTO domains (name, display_name, description, capabilities, status, health_url, registered_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			display_name = excluded.display_name,
			description = excluded.description,
			capabilities = excluded.capabilities,
			health_url = excluded.health_url,
			updated_at = excluded.updated_at
	`, req.Name, req.DisplayName, req.Description, capabilities, req.HealthURL, now, now)
	if err != nil {
		log.Printf("Error registering domain: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to register domain")
		return
	}

	domain := Domain{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Capabilities: req.Capabilities,
		Status:       "active",
		HealthURL:    req.HealthURL,
		RegisteredAt: now,
		UpdatedAt:    now,
	}

	writeJSON(w, http.StatusCreated, domain)
}

func (s *Server) listDomains(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT name, display_name, description, capabilities, status, health_url, last_health_at, registered_at, updated_at
		FROM domains ORDER BY name
	`)
	if err != nil {
		log.Printf("Error listing domains: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list domains")
		return
	}
	defer rows.Close()

	domains := []Domain{}
	for rows.Next() {
		var d Domain
		var capabilities string
		var lastHealthAt sql.NullString
		if err := rows.Scan(&d.Name, &d.DisplayName, &d.Description, &capabilities, &d.Status, &d.HealthURL, &lastHealthAt, &d.RegisteredAt, &d.UpdatedAt); err != nil {
			log.Printf("Error scanning domain: %v", err)
			continue
		}
		json.Unmarshal([]byte(capabilities), &d.Capabilities)
		if lastHealthAt.Valid {
			d.LastHealthAt = &lastHealthAt.String
		}
		domains = append(domains, d)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": domains,
		"count":   len(domains),
	})
}

func (s *Server) getDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var d Domain
	var capabilities string
	var lastHealthAt sql.NullString
	err := s.db.QueryRow(`
		SELECT name, display_name, description, capabilities, status, health_url, last_health_at, registered_at, updated_at
		FROM domains WHERE name = ?
	`, name).Scan(&d.Name, &d.DisplayName, &d.Description, &capabilities, &d.Status, &d.HealthURL, &lastHealthAt, &d.RegisteredAt, &d.UpdatedAt)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "domain not found")
		return
	}
	if err != nil {
		log.Printf("Error getting domain: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get domain")
		return
	}

	json.Unmarshal([]byte(capabilities), &d.Capabilities)
	if lastHealthAt.Valid {
		d.LastHealthAt = &lastHealthAt.String
	}

	writeJSON(w, http.StatusOK, d)
}

func (s *Server) updateDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Build update query dynamically
	setClause := "updated_at = ?"
	args := []interface{}{now}

	if status, ok := updates["status"].(string); ok {
		setClause += ", status = ?"
		args = append(args, status)
	}
	if displayName, ok := updates["display_name"].(string); ok {
		setClause += ", display_name = ?"
		args = append(args, displayName)
	}
	if description, ok := updates["description"].(string); ok {
		setClause += ", description = ?"
		args = append(args, description)
	}

	args = append(args, name)
	result, err := s.db.Exec(fmt.Sprintf("UPDATE domains SET %s WHERE name = ?", setClause), args...)
	if err != nil {
		log.Printf("Error updating domain: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update domain")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "domain not found")
		return
	}

	// Fetch and return updated domain
	s.getDomain(w, r)
}

func (s *Server) getDomainHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var d Domain
	var capabilities string
	var lastHealthAt sql.NullString
	err := s.db.QueryRow(`
		SELECT name, display_name, description, capabilities, status, health_url, last_health_at, registered_at, updated_at
		FROM domains WHERE name = ?
	`, name).Scan(&d.Name, &d.DisplayName, &d.Description, &capabilities, &d.Status, &d.HealthURL, &lastHealthAt, &d.RegisteredAt, &d.UpdatedAt)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "domain not found")
		return
	}
	if err != nil {
		log.Printf("Error getting domain health: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get domain")
		return
	}

	// If no health URL, just return current status
	if d.HealthURL == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domain":    name,
			"status":    d.Status,
			"last_check": lastHealthAt.String,
			"message":   "no health URL configured",
		})
		return
	}

	// Check health URL
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", d.HealthURL, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	now := time.Now().UTC().Format(time.RFC3339)
	status := "healthy"
	if err != nil || resp.StatusCode >= 300 {
		status = "unhealthy"
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Update domain status
	s.db.Exec("UPDATE domains SET status = ?, last_health_at = ?, updated_at = ? WHERE name = ?", status, now, now, name)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":    name,
		"status":    status,
		"last_check": now,
	})
}

// =============================================================================
// Statistics Handlers (P0-003, P0-004)
// =============================================================================

func (s *Server) getTimeline(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	days := query.Get("days")
	if days == "" {
		days = "7"
	}

	rows, err := s.db.Query(`
		SELECT
			date(timestamp) as day,
			domain,
			count(*) as event_count
		FROM events
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		GROUP BY date(timestamp), domain
		ORDER BY day DESC, domain
	`, days)
	if err != nil {
		log.Printf("Error getting timeline: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get timeline")
		return
	}
	defer rows.Close()

	timeline := []map[string]interface{}{}
	for rows.Next() {
		var day, domain string
		var count int
		if err := rows.Scan(&day, &domain, &count); err != nil {
			continue
		}
		timeline = append(timeline, map[string]interface{}{
			"day":    day,
			"domain": domain,
			"count":  count,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"timeline": timeline,
		"days":     days,
	})
}

func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {
	// Get counts by domain
	domainRows, err := s.db.Query(`
		SELECT domain, count(*) as event_count
		FROM events
		GROUP BY domain
		ORDER BY event_count DESC
	`)
	if err != nil {
		log.Printf("Error getting summary: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get summary")
		return
	}
	defer domainRows.Close()

	domains := []map[string]interface{}{}
	for domainRows.Next() {
		var domain string
		var count int
		if err := domainRows.Scan(&domain, &count); err != nil {
			continue
		}
		domains = append(domains, map[string]interface{}{
			"domain": domain,
			"count":  count,
		})
	}

	// Get total counts
	var totalEvents int
	s.db.QueryRow("SELECT count(*) FROM events").Scan(&totalEvents)

	var activeDomains int
	s.db.QueryRow("SELECT count(*) FROM domains WHERE status = 'active'").Scan(&activeDomains)

	// Get recent activity
	var lastEventTime sql.NullString
	s.db.QueryRow("SELECT max(timestamp) FROM events").Scan(&lastEventTime)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_events":    totalEvents,
		"active_domains":  activeDomains,
		"events_by_domain": domains,
		"last_event_at":   lastEventTime.String,
	})
}

// =============================================================================
// Helpers
// =============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow CORS from the UI server (local development and production)
		origin := r.Header.Get("Origin")
		if origin != "" {
			// Accept requests from localhost and common development origins
			// In production, this would be restricted to the specific UI domain
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initSchema(db *sql.DB) error {
	schema := `
	-- Events table (P0-001): Common envelope with JSON payloads
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		timestamp TEXT NOT NULL,
		domain TEXT NOT NULL,
		event_type TEXT NOT NULL,
		payload TEXT DEFAULT '{}',
		is_intervention INTEGER DEFAULT 0,
		hypothesis_id TEXT,
		created_at TEXT NOT NULL
	);

	-- Indexes for efficient cross-domain queries (P0-003)
	CREATE INDEX IF NOT EXISTS idx_events_domain_timestamp ON events(domain, timestamp);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
	CREATE INDEX IF NOT EXISTS idx_events_hypothesis ON events(hypothesis_id) WHERE hypothesis_id IS NOT NULL;

	-- Domains table (P0-002): Domain registration and discovery
	CREATE TABLE IF NOT EXISTS domains (
		name TEXT PRIMARY KEY,
		display_name TEXT NOT NULL,
		description TEXT DEFAULT '',
		capabilities TEXT DEFAULT '[]',
		status TEXT DEFAULT 'active',
		health_url TEXT DEFAULT '',
		last_health_at TEXT,
		registered_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_domains_status ON domains(status);
	`

	_, err := db.Exec(schema)
	return err
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "lifestyle-dashboard",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Get SQLite path from environment or use default
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = os.Getenv("SQLITE_DB")
	}
	if dbPath == "" {
		// Default to data directory
		dataDir := os.Getenv("SCENARIO_DATA_DIR")
		if dataDir == "" {
			dataDir = "."
		}
		dbPath = filepath.Join(dataDir, "lifestyle.db")
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	log.Printf("Opening SQLite database at: %s", dbPath)

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// SQLite-specific settings
	db.SetMaxOpenConns(1) // SQLite single-writer constraint
	db.SetMaxIdleConns(1)

	// Initialize schema
	if err := initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Competitor structure
type Competitor struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Importance  string          `json:"importance"`
	IsActive    bool            `json:"is_active"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// MonitoringTarget structure
type MonitoringTarget struct {
	ID              string          `json:"id"`
	CompetitorID    string          `json:"competitor_id"`
	URL             string          `json:"url"`
	TargetType      string          `json:"target_type"`
	Selector        string          `json:"selector,omitempty"`
	CheckFrequency  int             `json:"check_frequency"`
	LastChecked     *time.Time      `json:"last_checked,omitempty"`
	LastContentHash string          `json:"last_content_hash,omitempty"`
	IsActive        bool            `json:"is_active"`
	Config          json.RawMessage `json:"config,omitempty"`
}

// Alert structure
type Alert struct {
	ID             string          `json:"id"`
	CompetitorID   string          `json:"competitor_id"`
	Title          string          `json:"title"`
	Priority       string          `json:"priority"`
	URL            string          `json:"url"`
	Category       string          `json:"category"`
	Summary        string          `json:"summary"`
	Insights       json.RawMessage `json:"insights"`
	Actions        json.RawMessage `json:"actions"`
	RelevanceScore int             `json:"relevance_score"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ChangeAnalysis structure
type ChangeAnalysis struct {
	ID                 string          `json:"id"`
	CompetitorID       string          `json:"competitor_id"`
	TargetURL          string          `json:"target_url"`
	ChangeType         string          `json:"change_type"`
	RelevanceScore     int             `json:"relevance_score"`
	ChangeCategory     string          `json:"change_category"`
	ImpactLevel        string          `json:"impact_level"`
	KeyInsights        json.RawMessage `json:"key_insights"`
	RecommendedActions json.RawMessage `json:"recommended_actions"`
	Summary            string          `json:"summary"`
	CreatedAt          time.Time       `json:"created_at"`
}

var db *sql.DB

func initDB() {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
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

	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(randSource.Float64() * jitterRange)
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "competitor-change-monitor",
	})
}

// Get all competitors
func getCompetitorsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, description, category, importance, is_active, metadata, created_at, updated_at
		FROM competitors
		WHERE is_active = true
		ORDER BY importance DESC, name ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var competitors []Competitor
	for rows.Next() {
		var c Competitor
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Category, &c.Importance,
			&c.IsActive, &c.Metadata, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}
		competitors = append(competitors, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(competitors)
}

// Add a new competitor
func addCompetitorHandler(w http.ResponseWriter, r *http.Request) {
	var c Competitor
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var id string
	err := db.QueryRow(`
		INSERT INTO competitors (name, description, category, importance, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, c.Name, c.Description, c.Category, c.Importance, c.Metadata).Scan(&id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

// Get monitoring targets for a competitor
func getTargetsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitorID := vars["id"]

	rows, err := db.Query(`
		SELECT id, competitor_id, url, target_type, selector, check_frequency, 
		       last_checked, last_content_hash, is_active, config
		FROM monitoring_targets
		WHERE competitor_id = $1 AND is_active = true
	`, competitorID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var targets []MonitoringTarget
	for rows.Next() {
		var t MonitoringTarget
		err := rows.Scan(&t.ID, &t.CompetitorID, &t.URL, &t.TargetType, &t.Selector,
			&t.CheckFrequency, &t.LastChecked, &t.LastContentHash, &t.IsActive, &t.Config)
		if err != nil {
			continue
		}
		targets = append(targets, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}

// Add monitoring target
func addTargetHandler(w http.ResponseWriter, r *http.Request) {
	var t MonitoringTarget
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var id string
	err := db.QueryRow(`
		INSERT INTO monitoring_targets (competitor_id, url, target_type, selector, check_frequency, config)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, t.CompetitorID, t.URL, t.TargetType, t.Selector, t.CheckFrequency, t.Config).Scan(&id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// Get recent alerts
func getAlertsHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")

	query := `
		SELECT id, competitor_id, title, priority, url, category, summary, 
		       insights, actions, relevance_score, status, created_at
		FROM alerts
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if priority != "" {
		argCount++
		query += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, priority)
	}

	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		err := rows.Scan(&a.ID, &a.CompetitorID, &a.Title, &a.Priority, &a.URL,
			&a.Category, &a.Summary, &a.Insights, &a.Actions, &a.RelevanceScore,
			&a.Status, &a.CreatedAt)
		if err != nil {
			continue
		}
		alerts = append(alerts, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// Update alert status
func updateAlertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	var update struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE alerts SET status = $1 WHERE id = $2", update.Status, alertID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// Get recent changes/analyses
func getAnalysesHandler(w http.ResponseWriter, r *http.Request) {
	competitorID := r.URL.Query().Get("competitor_id")

	query := `
		SELECT id, competitor_id, target_url, change_type, relevance_score,
		       change_category, impact_level, key_insights, recommended_actions,
		       summary, created_at
		FROM change_analyses
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if competitorID != "" {
		argCount++
		query += fmt.Sprintf(" AND competitor_id = $%d", argCount)
		args = append(args, competitorID)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var analyses []ChangeAnalysis
	for rows.Next() {
		var a ChangeAnalysis
		err := rows.Scan(&a.ID, &a.CompetitorID, &a.TargetURL, &a.ChangeType,
			&a.RelevanceScore, &a.ChangeCategory, &a.ImpactLevel, &a.KeyInsights,
			&a.RecommendedActions, &a.Summary, &a.CreatedAt)
		if err != nil {
			continue
		}
		analyses = append(analyses, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analyses)
}

// Trigger manual scan
func triggerScanHandler(w http.ResponseWriter, r *http.Request) {
	// N8N URL is optional for trigger workflow
	n8nURL := os.Getenv("N8N_BASE_URL")

	// Call the scheduled scanner workflow
	resp, err := http.Post(n8nURL+"/webhook/competitor-monitor/manual-scan", "application/json", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start competitor-change-monitor

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	initDB()
	defer db.Close()

	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Competitor routes
	router.HandleFunc("/api/competitors", getCompetitorsHandler).Methods("GET")
	router.HandleFunc("/api/competitors", addCompetitorHandler).Methods("POST")
	router.HandleFunc("/api/competitors/{id}/targets", getTargetsHandler).Methods("GET")

	// Target routes
	router.HandleFunc("/api/targets", addTargetHandler).Methods("POST")

	// Alert routes
	router.HandleFunc("/api/alerts", getAlertsHandler).Methods("GET")
	router.HandleFunc("/api/alerts/{id}", updateAlertHandler).Methods("PATCH")

	// Analysis routes
	router.HandleFunc("/api/analyses", getAnalysesHandler).Methods("GET")

	// Scan routes
	router.HandleFunc("/api/scan", triggerScanHandler).Methods("POST")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("Competitor Monitor API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

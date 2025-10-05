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
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	DatabaseURL string
	Port        string
	ScriptsPath string
}

// Data structures
type AnalysisRequest struct {
	ScenarioPath string `json:"scenario_path,omitempty"`
	URL          string `json:"url,omitempty"`
	Mode         string `json:"mode"` // "local" or "deployed"
}

type AnalysisResult struct {
	ScenarioPath      string    `json:"scenario_path"`
	AnalysisTimestamp time.Time `json:"analysis_timestamp"`
	Mode              string    `json:"mode"`
	Branding          Branding  `json:"branding"`
	Pricing           Pricing   `json:"pricing"`
	Structure         Structure `json:"structure"`
}

type Branding struct {
	Colors    BrandColors `json:"colors"`
	Fonts     []string    `json:"fonts"`
	LogoPath  string      `json:"logo_path"`
	BrandName string      `json:"brand_name"`
}

type BrandColors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
}

type Pricing struct {
	Model        string        `json:"model"`
	Tiers        []PricingTier `json:"tiers"`
	Currency     string        `json:"currency"`
	BillingCycle string        `json:"billing_cycle"`
}

type PricingTier struct {
	Name   string `json:"name"`
	Price  string `json:"price"`
	Period string `json:"period"`
}

type Structure struct {
	HasAPI              bool   `json:"has_api"`
	HasUI               bool   `json:"has_ui"`
	HasCLI              bool   `json:"has_cli"`
	HasDatabase         bool   `json:"has_database"`
	HasExistingReferral bool   `json:"has_existing_referral"`
	APIFramework        string `json:"api_framework"`
	UIFramework         string `json:"ui_framework"`
}

type GenerateRequest struct {
	AnalysisData    AnalysisResult `json:"analysis_data"`
	CommissionRate  *float64       `json:"commission_rate,omitempty"`
	CustomBranding  *Branding      `json:"custom_branding,omitempty"`
	OutputDirectory string         `json:"output_directory,omitempty"`
}

type GenerateResponse struct {
	ProgramID             string   `json:"program_id"`
	TrackingCode          string   `json:"tracking_code"`
	LandingPageHTML       string   `json:"landing_page_html"`
	EmailTemplates        []string `json:"email_templates"`
	AnalyticsDashboardURL string   `json:"analytics_dashboard_url"`
	GeneratedFiles        []string `json:"generated_files"`
}

type ImplementRequest struct {
	ProgramID    string `json:"program_id"`
	ScenarioPath string `json:"scenario_path"`
	AutoMode     bool   `json:"auto_mode"`
}

type ImplementResponse struct {
	ImplementationStatus string   `json:"implementation_status"`
	FilesModified        []string `json:"files_modified"`
	ValidationResults    string   `json:"validation_results"`
}

// Global variables
var db *sql.DB
var config Config

// Initialize configuration
func initConfig() {
	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Database configuration - support both POSTGRES_URL and individual components
	databaseURL := os.Getenv("POSTGRES_URL")
	if databaseURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	config = Config{
		DatabaseURL: databaseURL,
		Port:        port,
		ScriptsPath: "./scripts",
	}
}

// getEnvWithDefault removed - no hardcoded defaults for configuration

// Database initialization
func initDatabase() error {
	var err error
	db, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
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
	log.Printf("📊 Database URL configured")

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
		jitter := time.Duration(jitterRange * rand.Float64())
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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"database":  "connected",
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "disconnected"
		health["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Analysis handler
func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	var req AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Mode != "local" && req.Mode != "deployed" {
		http.Error(w, "Mode must be 'local' or 'deployed'", http.StatusBadRequest)
		return
	}

	if req.Mode == "local" && req.ScenarioPath == "" {
		http.Error(w, "scenario_path is required for local mode", http.StatusBadRequest)
		return
	}

	if req.Mode == "deployed" && req.URL == "" {
		http.Error(w, "url is required for deployed mode", http.StatusBadRequest)
		return
	}

	// Run analysis script
	scriptPath := filepath.Join(config.ScriptsPath, "analyze-scenario.sh")
	var cmd *exec.Cmd

	if req.Mode == "local" {
		cmd = exec.Command(scriptPath, "--mode", "local", "--output", "json", req.ScenarioPath)
	} else {
		cmd = exec.Command(scriptPath, "--mode", "deployed", "--output", "json", req.URL)
	}

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Analysis script error: %v", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse and return results
	var result AnalysisResult
	if err := json.Unmarshal(output, &result); err != nil {
		log.Printf("Failed to parse analysis results: %v", err)
		http.Error(w, "Failed to parse analysis results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Generate referral program handler
func generateHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate unique program ID and tracking code
	programID := uuid.New().String()
	trackingCode := generateTrackingCode()

	// Prepare output directory
	outputDir := req.OutputDirectory
	if outputDir == "" {
		outputDir = fmt.Sprintf("/tmp/referral-program-%s", trackingCode)
	}

	// Write analysis data to temporary file
	tempFile := fmt.Sprintf("/tmp/analysis-%s.json", programID)
	analysisJSON, _ := json.MarshalIndent(req.AnalysisData, "", "  ")
	if err := os.WriteFile(tempFile, analysisJSON, 0644); err != nil {
		http.Error(w, "Failed to create temporary analysis file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile)

	// Run generation script
	scriptPath := filepath.Join(config.ScriptsPath, "generate-referral-pattern.sh")
	args := []string{"--output-dir", outputDir}

	if req.CommissionRate != nil {
		args = append(args, "--commission-rate", fmt.Sprintf("%.4f", *req.CommissionRate))
	}

	args = append(args, tempFile)
	cmd := exec.Command(scriptPath, args...)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Generation script error: %v", err)
		http.Error(w, fmt.Sprintf("Generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Log generation script output for debugging
	log.Printf("Generation script output: %s", string(output))

	// Store program in database
	commissionRate := 0.20 // Default
	if req.CommissionRate != nil {
		commissionRate = *req.CommissionRate
	}

	brandingJSON, _ := json.Marshal(req.AnalysisData.Branding)
	query := `
		INSERT INTO referral_programs (id, scenario_name, commission_rate, tracking_code, branding_config)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = db.Exec(query, programID, req.AnalysisData.Branding.BrandName, commissionRate, trackingCode, string(brandingJSON))
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to store program in database", http.StatusInternalServerError)
		return
	}

	// Generate response
	response := GenerateResponse{
		ProgramID:             programID,
		TrackingCode:          trackingCode,
		LandingPageHTML:       "<html>Landing page generated</html>", // Placeholder
		EmailTemplates:        []string{"welcome-email", "conversion-email"},
		AnalyticsDashboardURL: fmt.Sprintf("http://localhost:%s/dashboard/referral/%s", config.Port, programID),
		GeneratedFiles:        []string{outputDir + "/config.json", outputDir + "/api/referral.go"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Implementation handler - integrates with resource-claude-code
func implementHandler(w http.ResponseWriter, r *http.Request) {
	var req ImplementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get program details from database
	var program struct {
		ScenarioName   string
		CommissionRate float64
		TrackingCode   string
		BrandingConfig string
	}

	query := `SELECT scenario_name, commission_rate, tracking_code, branding_config FROM referral_programs WHERE id = $1`
	err := db.QueryRow(query, req.ProgramID).Scan(&program.ScenarioName, &program.CommissionRate, &program.TrackingCode, &program.BrandingConfig)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	// Placeholder for claude-code integration
	response := ImplementResponse{
		ImplementationStatus: "pending",
		FilesModified:        []string{},
		ValidationResults:    "Implementation queued - resource-claude-code integration pending",
	}

	if req.AutoMode {
		// TODO: Integrate with resource-claude-code
		// For now, return simulation
		response.ImplementationStatus = "simulated"
		response.FilesModified = []string{
			req.ScenarioPath + "/src/referral/config.json",
			req.ScenarioPath + "/src/referral/api/referral.go",
			req.ScenarioPath + "/src/referral/ui/ReferralDashboard.jsx",
		}
		response.ValidationResults = "Simulated implementation completed successfully"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// List programs handler
func listProgramsHandler(w http.ResponseWriter, r *http.Request) {
	scenarioFilter := r.URL.Query().Get("scenario")

	var query string
	var args []interface{}

	if scenarioFilter != "" {
		query = `SELECT id, scenario_name, commission_rate, tracking_code, created_at FROM referral_programs WHERE scenario_name = $1 ORDER BY created_at DESC`
		args = append(args, scenarioFilter)
	} else {
		query = `SELECT id, scenario_name, commission_rate, tracking_code, created_at FROM referral_programs ORDER BY created_at DESC LIMIT 50`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var programs []map[string]interface{}
	for rows.Next() {
		var id, scenarioName, trackingCode string
		var commissionRate float64
		var createdAt time.Time

		if err := rows.Scan(&id, &scenarioName, &commissionRate, &trackingCode, &createdAt); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}

		programs = append(programs, map[string]interface{}{
			"id":              id,
			"scenario_name":   scenarioName,
			"commission_rate": commissionRate,
			"tracking_code":   trackingCode,
			"created_at":      createdAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"programs": programs,
		"count":    len(programs),
	})
}

// Generate tracking code
func generateTrackingCode() string {
	return fmt.Sprintf("REF%d%s", time.Now().Unix()%10000, uuid.New().String()[:6])
}

// Setup routes
func setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1/referral").Subrouter()
	api.HandleFunc("/analyze", analyzeHandler).Methods("POST")
	api.HandleFunc("/generate", generateHandler).Methods("POST")
	api.HandleFunc("/implement", implementHandler).Methods("POST")
	api.HandleFunc("/programs", listProgramsHandler).Methods("GET")

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// CORS setup
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	return c.Handler(router).(*mux.Router)
}

// Main function
func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start referral-program-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	log.Println("Starting Referral Program Generator API...")

	// Initialize configuration
	initConfig()
	log.Printf("Configuration loaded - Port: %s", config.Port)

	// Initialize database
	if err := initDatabase(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Setup routes
	router := setupRoutes()

	// Start server
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 Referral Program Generator API running on port %s", config.Port)
	log.Printf("📊 Health check: http://localhost:%s/health", config.Port)
	log.Printf("🔍 Analysis endpoint: http://localhost:%s/api/v1/referral/analyze", config.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

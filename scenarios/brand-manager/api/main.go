package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	// API version
	apiVersion  = "2.0.0"
	serviceName = "brand-manager"

	// Defaults
	defaultPort = "8090"

	// Timeouts
	httpTimeout        = 30 * time.Second
	discoveryDelay     = 5 * time.Second
	brandGenTimeout    = 120 * time.Second
	integrationTimeout = 300 * time.Second

	// Database limits
	maxDBConnections   = 25
	maxIdleConnections = 5
	connMaxLifetime    = 5 * time.Minute

	// Defaults
	defaultLimit = 20
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[brand-manager-api] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func (l *Logger) Warn(msg string, err error) {
	l.Printf("WARN: %s: %v", msg, err)
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

// Brand represents a brand entity
type Brand struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	ShortName   string                 `json:"short_name,omitempty"`
	Slogan      string                 `json:"slogan,omitempty"`
	AdCopy      string                 `json:"ad_copy,omitempty"`
	Description string                 `json:"description,omitempty"`
	BrandColors map[string]interface{} `json:"brand_colors"`
	LogoURL     string                 `json:"logo_url,omitempty"`
	FaviconURL  string                 `json:"favicon_url,omitempty"`
	Assets      []interface{}          `json:"assets"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// IntegrationRequest represents a Claude Code integration request
type IntegrationRequest struct {
	ID              uuid.UUID              `json:"id"`
	BrandID         uuid.UUID              `json:"brand_id"`
	TargetAppPath   string                 `json:"target_app_path"`
	IntegrationType string                 `json:"integration_type"`
	ClaudeSessionID string                 `json:"claude_session_id,omitempty"`
	Status          string                 `json:"status"`
	RequestPayload  map[string]interface{} `json:"request_payload"`
	ResponsePayload map[string]interface{} `json:"response_payload,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
}

// BrandManagerService handles brand management operations
type BrandManagerService struct {
	db            *sql.DB
	n8nBaseURL    string
	windmillURL   string
	comfyUIURL    string
	minioEndpoint string
	vaultAddr     string
	httpClient    *http.Client
	logger        *Logger
}

// NewBrandManagerService creates a new brand manager service
func NewBrandManagerService(db *sql.DB, n8nURL, windmillURL, comfyUIURL, minioEndpoint, vaultAddr string) *BrandManagerService {
	return &BrandManagerService{
		db:            db,
		n8nBaseURL:    n8nURL,
		windmillURL:   windmillURL,
		comfyUIURL:    comfyUIURL,
		minioEndpoint: minioEndpoint,
		vaultAddr:     vaultAddr,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger: NewLogger(),
	}
}

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": serviceName,
		"version": apiVersion,
	})
}

// ListBrands returns all brands with pagination
func (bm *BrandManagerService) ListBrands(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	rows, err := bm.db.Query(`
		SELECT id, name, short_name, slogan, ad_copy, description, 
		       brand_colors, logo_url, favicon_url, assets, metadata, 
		       created_at, updated_at
		FROM brands
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		HTTPError(w, "Failed to query brands", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		var brand Brand
		var brandColorsJSON, assetsJSON, metadataJSON []byte

		err := rows.Scan(&brand.ID, &brand.Name, &brand.ShortName, &brand.Slogan,
			&brand.AdCopy, &brand.Description, &brandColorsJSON, &brand.LogoURL,
			&brand.FaviconURL, &assetsJSON, &metadataJSON, &brand.CreatedAt, &brand.UpdatedAt)
		if err != nil {
			bm.logger.Warn("Failed to scan brand row", err)
			continue
		}

		// Parse JSON fields
		json.Unmarshal(brandColorsJSON, &brand.BrandColors)
		json.Unmarshal(assetsJSON, &brand.Assets)
		json.Unmarshal(metadataJSON, &brand.Metadata)

		brands = append(brands, brand)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(brands)
}

// CreateBrand generates a new brand using n8n workflows
func (bm *BrandManagerService) CreateBrand(w http.ResponseWriter, r *http.Request) {
	var request struct {
		BrandName   string `json:"brand_name"`
		ShortName   string `json:"short_name,omitempty"`
		Industry    string `json:"industry"`
		Template    string `json:"template,omitempty"`
		LogoStyle   string `json:"logo_style,omitempty"`
		ColorScheme string `json:"color_scheme,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if request.BrandName == "" || request.Industry == "" {
		HTTPError(w, "Missing required fields: brand_name and industry", http.StatusBadRequest, nil)
		return
	}

	// Set defaults
	if request.Template == "" {
		request.Template = "modern-tech"
	}
	if request.LogoStyle == "" {
		request.LogoStyle = "minimalist"
	}
	if request.ColorScheme == "" {
		request.ColorScheme = "primary"
	}

	// Trigger brand generation via n8n webhook
	payload := map[string]interface{}{
		"brandName":   request.BrandName,
		"shortName":   request.ShortName,
		"industry":    request.Industry,
		"template":    request.Template,
		"logoStyle":   request.LogoStyle,
		"colorScheme": request.ColorScheme,
	}

	payloadBytes, _ := json.Marshal(payload)
	webhookURL := fmt.Sprintf("%s/webhook/generate-brand", bm.n8nBaseURL)

	resp, err := bm.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		HTTPError(w, "Failed to trigger brand generation", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Brand generation workflow failed", http.StatusInternalServerError,
			fmt.Errorf("n8n returned status %d", resp.StatusCode))
		return
	}

	var workflowResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workflowResult); err != nil {
		HTTPError(w, "Failed to decode workflow response", http.StatusInternalServerError, err)
		return
	}

	// Return workflow result with polling endpoint for status
	response := map[string]interface{}{
		"message":          "Brand generation started",
		"workflow_result":  workflowResult,
		"polling_endpoint": fmt.Sprintf("/api/brands/status/%s", request.BrandName),
		"estimated_time":   "45-60 seconds",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBrandByID retrieves a specific brand
func (bm *BrandManagerService) GetBrandByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	brandID := vars["id"]

	var brand Brand
	var brandColorsJSON, assetsJSON, metadataJSON []byte

	err := bm.db.QueryRow(`
		SELECT id, name, short_name, slogan, ad_copy, description, 
		       brand_colors, logo_url, favicon_url, assets, metadata, 
		       created_at, updated_at
		FROM brands WHERE id = $1`, brandID).Scan(
		&brand.ID, &brand.Name, &brand.ShortName, &brand.Slogan,
		&brand.AdCopy, &brand.Description, &brandColorsJSON, &brand.LogoURL,
		&brand.FaviconURL, &assetsJSON, &metadataJSON, &brand.CreatedAt, &brand.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			HTTPError(w, "Brand not found", http.StatusNotFound, err)
		} else {
			HTTPError(w, "Failed to query brand", http.StatusInternalServerError, err)
		}
		return
	}

	// Parse JSON fields
	json.Unmarshal(brandColorsJSON, &brand.BrandColors)
	json.Unmarshal(assetsJSON, &brand.Assets)
	json.Unmarshal(metadataJSON, &brand.Metadata)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(brand)
}

// GetBrandStatus checks generation status by brand name
func (bm *BrandManagerService) GetBrandStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	brandName := vars["name"]

	var brand Brand
	var brandColorsJSON, assetsJSON, metadataJSON []byte

	err := bm.db.QueryRow(`
		SELECT id, name, short_name, slogan, ad_copy, description, 
		       brand_colors, logo_url, favicon_url, assets, metadata, 
		       created_at, updated_at
		FROM brands WHERE name = $1
		ORDER BY created_at DESC LIMIT 1`, brandName).Scan(
		&brand.ID, &brand.Name, &brand.ShortName, &brand.Slogan,
		&brand.AdCopy, &brand.Description, &brandColorsJSON, &brand.LogoURL,
		&brand.FaviconURL, &assetsJSON, &metadataJSON, &brand.CreatedAt, &brand.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// Brand not found yet - generation might still be in progress
			response := map[string]interface{}{
				"status":     "in_progress",
				"message":    "Brand generation in progress",
				"brand_name": brandName,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			HTTPError(w, "Failed to check brand status", http.StatusInternalServerError, err)
		}
		return
	}

	// Parse JSON fields
	json.Unmarshal(brandColorsJSON, &brand.BrandColors)
	json.Unmarshal(assetsJSON, &brand.Assets)
	json.Unmarshal(metadataJSON, &brand.Metadata)

	response := map[string]interface{}{
		"status": "completed",
		"brand":  brand,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListIntegrations returns all integration requests with pagination
func (bm *BrandManagerService) ListIntegrations(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	rows, err := bm.db.Query(`
		SELECT id, brand_id, target_app_path, integration_type, claude_session_id,
		       status, request_payload, response_payload, created_at, completed_at
		FROM integration_requests
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		HTTPError(w, "Failed to query integrations", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var integrations []IntegrationRequest
	for rows.Next() {
		var integration IntegrationRequest
		var requestPayloadJSON, responsePayloadJSON []byte

		err := rows.Scan(&integration.ID, &integration.BrandID, &integration.TargetAppPath,
			&integration.IntegrationType, &integration.ClaudeSessionID, &integration.Status,
			&requestPayloadJSON, &responsePayloadJSON, &integration.CreatedAt, &integration.CompletedAt)
		if err != nil {
			bm.logger.Warn("Failed to scan integration row", err)
			continue
		}

		// Parse JSON fields
		json.Unmarshal(requestPayloadJSON, &integration.RequestPayload)
		json.Unmarshal(responsePayloadJSON, &integration.ResponsePayload)

		integrations = append(integrations, integration)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(integrations)
}

// CreateIntegration starts a Claude Code integration
func (bm *BrandManagerService) CreateIntegration(w http.ResponseWriter, r *http.Request) {
	var request struct {
		BrandID         string `json:"brand_id"`
		TargetAppPath   string `json:"target_app_path"`
		IntegrationType string `json:"integration_type"`
		CreateBackup    bool   `json:"create_backup"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if request.BrandID == "" || request.TargetAppPath == "" {
		HTTPError(w, "Missing required fields: brand_id and target_app_path", http.StatusBadRequest, nil)
		return
	}

	// Set defaults
	if request.IntegrationType == "" {
		request.IntegrationType = "full"
	}

	// Trigger Claude Code integration via n8n webhook
	payload := map[string]interface{}{
		"brandId":         request.BrandID,
		"targetAppPath":   request.TargetAppPath,
		"integrationType": request.IntegrationType,
		"createBackup":    request.CreateBackup,
	}

	payloadBytes, _ := json.Marshal(payload)
	webhookURL := fmt.Sprintf("%s/webhook/spawn-claude-integration", bm.n8nBaseURL)

	resp, err := bm.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		HTTPError(w, "Failed to trigger integration", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Integration workflow failed", http.StatusInternalServerError,
			fmt.Errorf("n8n returned status %d", resp.StatusCode))
		return
	}

	var workflowResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workflowResult); err != nil {
		HTTPError(w, "Failed to decode workflow response", http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"message":         "Integration started",
		"workflow_result": workflowResult,
		"estimated_time":  "3-5 minutes",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServiceURLs returns URLs for all connected services
func (bm *BrandManagerService) GetServiceURLs(w http.ResponseWriter, r *http.Request) {
	urls := map[string]interface{}{
		"services": map[string]string{
			"n8n":      bm.n8nBaseURL,
			"windmill": bm.windmillURL,
			"comfyui":  bm.comfyUIURL,
			"minio":    fmt.Sprintf("http://%s", bm.minioEndpoint),
			"vault":    bm.vaultAddr,
		},
		"dashboards": map[string]string{
			"brand_manager":       fmt.Sprintf("%s/apps/f/brand-manager/dashboard", bm.windmillURL),
			"integration_monitor": fmt.Sprintf("%s/apps/f/brand-manager/integration-dashboard", bm.windmillURL),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(urls)
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
			"n8n":      "5678",
			"windmill": "8000",
			"postgres": "5432",
			"comfyui":  "8188",
			"minio":    "9000",
			"vault":    "8200",
		}
		if port, ok := defaults[resourceName]; ok {
			return port
		}
		return "8080" // Generic fallback
	}
	return strings.TrimSpace(string(output))
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start brand-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}

	// Use port registry for resource ports
	n8nPort := getResourcePort("n8n")
	windmillPort := getResourcePort("windmill")
	_ = getResourcePort("postgres") // postgres port retrieved but connection uses URL from env
	comfyUIPort := getResourcePort("comfyui")
	minioPort := getResourcePort("minio")
	vaultPort := getResourcePort("vault")

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
	}

	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillURL = fmt.Sprintf("http://localhost:%s", windmillPort)
	}

	comfyUIURL := os.Getenv("COMFYUI_BASE_URL")
	if comfyUIURL == "" {
		comfyUIURL = fmt.Sprintf("http://localhost:%s", comfyUIPort)
	}

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = fmt.Sprintf("localhost:%s", minioPort)
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = fmt.Sprintf("http://localhost:%s", vaultPort)
	}

	// Database configuration - support both POSTGRES_URL and individual components
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger := NewLogger()
		logger.Error("Failed to connect to database", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("🎨 Database URL configured")

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
		logger := NewLogger()
		logger.Error(fmt.Sprintf("❌ Database connection failed after %d attempts", maxRetries), pingErr)
		os.Exit(1)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Initialize brand manager service
	brandManager := NewBrandManagerService(db, n8nURL, windmillURL, comfyUIURL, minioEndpoint, vaultAddr)

	// Setup routes
	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/brands", brandManager.ListBrands).Methods("GET")
	r.HandleFunc("/api/brands", brandManager.CreateBrand).Methods("POST")
	r.HandleFunc("/api/brands/{id}", brandManager.GetBrandByID).Methods("GET")
	r.HandleFunc("/api/brands/status/{name}", brandManager.GetBrandStatus).Methods("GET")
	r.HandleFunc("/api/integrations", brandManager.ListIntegrations).Methods("GET")
	r.HandleFunc("/api/integrations", brandManager.CreateIntegration).Methods("POST")
	r.HandleFunc("/api/services", brandManager.GetServiceURLs).Methods("GET")

	// Start server
	log.Printf("Starting Brand Manager API on port %s", port)
	log.Printf("  n8n URL: %s", n8nURL)
	log.Printf("  Windmill URL: %s", windmillURL)
	log.Printf("  ComfyUI URL: %s", comfyUIURL)
	log.Printf("  MinIO Endpoint: %s", minioEndpoint)
	log.Printf("  Vault URL: %s", vaultAddr)
	log.Printf("  Database: %s", dbURL)

	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}

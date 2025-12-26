package main

import (
	"bytes"
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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
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
	comfyUIURL    string
	minioEndpoint string
	vaultAddr     string
	httpClient    *http.Client
	logger        *Logger
}

// NewBrandManagerService creates a new brand manager service
func NewBrandManagerService(db *sql.DB, n8nURL, comfyUIURL, minioEndpoint, vaultAddr string) *BrandManagerService {
	return &BrandManagerService{
		db:            db,
		n8nBaseURL:    n8nURL,
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
			"n8n":     bm.n8nBaseURL,
			"comfyui": bm.comfyUIURL,
			"minio":   fmt.Sprintf("http://%s", bm.minioEndpoint),
			"vault":   bm.vaultAddr,
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "brand-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Use port registry for resource ports
	n8nPort := getResourcePort("n8n")
	_ = getResourcePort("postgres") // postgres port retrieved but connection uses URL from env
	comfyUIPort := getResourcePort("comfyui")
	minioPort := getResourcePort("minio")
	vaultPort := getResourcePort("vault")

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
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

	// Connect to database
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverPostgres,
		MaxOpenConns: maxDBConnections,
		MaxIdleConns: maxIdleConnections,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	log.Println("Database connection pool established successfully!")

	// Initialize brand manager service
	brandManager := NewBrandManagerService(db, n8nURL, comfyUIURL, minioEndpoint, vaultAddr)

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/brands", brandManager.ListBrands).Methods("GET")
	r.HandleFunc("/api/brands", brandManager.CreateBrand).Methods("POST")
	r.HandleFunc("/api/brands/{id}", brandManager.GetBrandByID).Methods("GET")
	r.HandleFunc("/api/brands/status/{name}", brandManager.GetBrandStatus).Methods("GET")
	r.HandleFunc("/api/integrations", brandManager.ListIntegrations).Methods("GET")
	r.HandleFunc("/api/integrations", brandManager.CreateIntegration).Methods("POST")
	r.HandleFunc("/api/services", brandManager.GetServiceURLs).Methods("GET")

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: r,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

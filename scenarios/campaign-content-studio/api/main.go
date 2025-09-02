package main

import (
	"bytes"
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
)

const (
	// API version
	apiVersion = "2.0.0"
	serviceName = "campaign-content-studio"
	
	// Defaults
	defaultPort = "8090"
	
	// Timeouts
	httpTimeout = 30 * time.Second
	discoveryDelay = 5 * time.Second
	
	// Database limits
	maxDBConnections = 25
	maxIdleConnections = 5
	connMaxLifetime = 5 * time.Minute
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[campaign-content-studio-api] ", log.LstdFlags|log.Lshortfile),
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

// Campaign represents a content campaign
type Campaign struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Document represents a campaign document
type Document struct {
	ID           uuid.UUID `json:"id"`
	CampaignID   uuid.UUID `json:"campaign_id"`
	Filename     string    `json:"filename"`
	FilePath     string    `json:"file_path"`
	ContentType  string    `json:"content_type"`
	ProcessedText string   `json:"processed_text,omitempty"`
	EmbeddingID  string    `json:"embedding_id,omitempty"`
	UploadDate   time.Time `json:"upload_date"`
}

// GeneratedContent represents generated content
type GeneratedContent struct {
	ID              uuid.UUID              `json:"id"`
	CampaignID      uuid.UUID              `json:"campaign_id"`
	ContentType     string                 `json:"content_type"`
	Prompt          string                 `json:"prompt"`
	GeneratedText   string                 `json:"generated_content"`
	UsedDocuments   []string               `json:"used_documents"`
	CreatedAt       time.Time              `json:"created_at"`
}

// CampaignService handles campaign content operations
type CampaignService struct {
	db          *sql.DB
	n8nBaseURL  string
	windmillURL string
	postgresURL string
	qdrantURL   string
	minioURL    string
	httpClient  *http.Client
	logger      *Logger
}

// NewCampaignService creates a new campaign service
func NewCampaignService(db *sql.DB, n8nURL, windmillURL, postgresURL, qdrantURL, minioURL string) *CampaignService {
	return &CampaignService{
		db:          db,
		n8nBaseURL:  n8nURL,
		windmillURL: windmillURL,
		postgresURL: postgresURL,
		qdrantURL:   qdrantURL,
		minioURL:    minioURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:      NewLogger(),
	}
}

// ListCampaigns returns all campaigns
func (c *CampaignService) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := c.db.Query(`
		SELECT id, name, description, settings, created_at, updated_at
		FROM campaigns
		ORDER BY updated_at DESC`)
	if err != nil {
		HTTPError(w, "Failed to query campaigns", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	
	var campaigns []Campaign
	for rows.Next() {
		var campaign Campaign
		var settingsJSON []byte
		
		err := rows.Scan(&campaign.ID, &campaign.Name, &campaign.Description, 
			&settingsJSON, &campaign.CreatedAt, &campaign.UpdatedAt)
		if err != nil {
			c.logger.Error("Failed to scan campaign row", err)
			continue
		}
		
		// Parse settings JSON
		if len(settingsJSON) > 0 {
			json.Unmarshal(settingsJSON, &campaign.Settings)
		}
		
		campaigns = append(campaigns, campaign)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaigns)
}

// CreateCampaign creates a new campaign
func (c *CampaignService) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign Campaign
	if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	if campaign.Name == "" {
		HTTPError(w, "Campaign name is required", http.StatusBadRequest, nil)
		return
	}
	
	campaign.ID = uuid.New()
	campaign.CreatedAt = time.Now().UTC()
	campaign.UpdatedAt = campaign.CreatedAt
	
	settingsJSON, _ := json.Marshal(campaign.Settings)
	
	_, err := c.db.Exec(`
		INSERT INTO campaigns (id, name, description, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		campaign.ID, campaign.Name, campaign.Description, settingsJSON,
		campaign.CreatedAt, campaign.UpdatedAt)
	
	if err != nil {
		HTTPError(w, "Failed to create campaign", http.StatusInternalServerError, err)
		return
	}
	
	// Trigger campaign management workflow
	go c.triggerWorkflow("campaign-management", map[string]interface{}{
		"operation": "create",
		"campaign_id": campaign.ID.String(),
		"name": campaign.Name,
		"description": campaign.Description,
		"settings": campaign.Settings,
	})
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(campaign)
}

// ListDocuments returns documents for a campaign
func (c *CampaignService) ListDocuments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["campaignId"]
	
	if campaignID == "" {
		HTTPError(w, "Campaign ID is required", http.StatusBadRequest, nil)
		return
	}
	
	rows, err := c.db.Query(`
		SELECT id, campaign_id, filename, file_path, content_type, upload_date
		FROM campaign_documents
		WHERE campaign_id = $1
		ORDER BY upload_date DESC`, campaignID)
	if err != nil {
		HTTPError(w, "Failed to query documents", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	
	var documents []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(&doc.ID, &doc.CampaignID, &doc.Filename, 
			&doc.FilePath, &doc.ContentType, &doc.UploadDate)
		if err != nil {
			c.logger.Error("Failed to scan document row", err)
			continue
		}
		documents = append(documents, doc)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

// GenerateContent generates content for a campaign
func (c *CampaignService) GenerateContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CampaignID    string `json:"campaign_id"`
		ContentType   string `json:"content_type"`
		Prompt        string `json:"prompt"`
		IncludeImages bool   `json:"include_images"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	if req.CampaignID == "" || req.ContentType == "" || req.Prompt == "" {
		HTTPError(w, "Campaign ID, content type, and prompt are required", http.StatusBadRequest, nil)
		return
	}
	
	// Trigger content generation workflow
	workflowReq := map[string]interface{}{
		"operation": "generate",
		"campaign_id": req.CampaignID,
		"content_type": req.ContentType,
		"prompt": req.Prompt,
		"include_images": req.IncludeImages,
	}
	
	result, err := c.triggerWorkflowSync("content-generation", workflowReq)
	if err != nil {
		HTTPError(w, "Content generation failed", http.StatusInternalServerError, err)
		return
	}
	
	// Store generated content in database
	contentID := uuid.New()
	generatedText, _ := result["generated_content"].(string)
	usedDocsInterface, _ := result["used_documents"].([]interface{})
	var usedDocs []string
	for _, doc := range usedDocsInterface {
		if docStr, ok := doc.(string); ok {
			usedDocs = append(usedDocs, docStr)
		}
	}
	
	_, err = c.db.Exec(`
		INSERT INTO generated_content (id, campaign_id, content_type, prompt, generated_content, used_documents, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		contentID, req.CampaignID, req.ContentType, req.Prompt, generatedText, 
		strings.Join(usedDocs, ","), time.Now().UTC())
	
	if err != nil {
		c.logger.Error("Failed to store generated content", err)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SearchDocuments performs semantic search on campaign documents
func (c *CampaignService) SearchDocuments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["campaignId"]
	
	var searchReq struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	if campaignID == "" || searchReq.Query == "" {
		HTTPError(w, "Campaign ID and search query are required", http.StatusBadRequest, nil)
		return
	}
	
	if searchReq.Limit == 0 {
		searchReq.Limit = 10
	}
	
	// Trigger search retrieval workflow
	result, err := c.triggerWorkflowSync("search-retrieval", map[string]interface{}{
		"operation": "search",
		"campaign_id": campaignID,
		"query": searchReq.Query,
		"limit": searchReq.Limit,
	})
	
	if err != nil {
		HTTPError(w, "Document search failed", http.StatusInternalServerError, err)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// triggerWorkflow triggers an n8n workflow asynchronously
func (c *CampaignService) triggerWorkflow(workflowName string, payload map[string]interface{}) {
	reqBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/webhook/%s", c.n8nBaseURL, workflowName)
	
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to trigger workflow %s", workflowName), err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		c.logger.Error(fmt.Sprintf("Workflow %s returned error status", workflowName), 
			fmt.Errorf("status: %d", resp.StatusCode))
	}
}

// triggerWorkflowSync triggers an n8n workflow synchronously and returns result
func (c *CampaignService) triggerWorkflowSync(workflowName string, payload map[string]interface{}) (map[string]interface{}, error) {
	reqBody, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/webhook/%s", c.n8nBaseURL, workflowName)
	
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("workflow returned status %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": serviceName,
		"version": apiVersion,
	})
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
			"n8n": "5678",
			"windmill": "5681",
			"postgres": "5433",
			"qdrant": "6333",
			"minio": "9000",
		}
		if port, ok := defaults[resourceName]; ok {
			return port
		}
		return "8080" // Generic fallback
	}
	return strings.TrimSpace(string(output))
}

func main() {
	// Load configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("SERVICE_PORT")
		if port == "" {
			port = defaultPort
		}
	}
	
	// Use port registry for resource ports
	n8nPort := getResourcePort("n8n")
	windmillPort := getResourcePort("windmill")
	postgresPort := getResourcePort("postgres")
	qdrantPort := getResourcePort("qdrant")
	minioPort := getResourcePort("minio")
	
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
	}
	
	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillURL = fmt.Sprintf("http://localhost:%s", windmillPort)
	}
	
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%s/campaign_content_studio?sslmode=disable", postgresPort)
	}
	
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = fmt.Sprintf("http://localhost:%s", qdrantPort)
	}
	
	minioURL := os.Getenv("MINIO_URL")
	if minioURL == "" {
		minioURL = fmt.Sprintf("http://localhost:%s", minioPort)
	}
	
	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
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
	
	// Test database connection
	if err := db.Ping(); err != nil {
		logger := NewLogger()
		logger.Error("Failed to ping database", err)
		os.Exit(1)
	}
	
	log.Println("Connected to database")
	
	// Initialize campaign service
	service := NewCampaignService(db, n8nURL, windmillURL, postgresURL, qdrantURL, minioURL)
	
	// Setup routes
	r := mux.NewRouter()
	
	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/campaigns", service.ListCampaigns).Methods("GET")
	r.HandleFunc("/campaigns", service.CreateCampaign).Methods("POST")
	r.HandleFunc("/campaigns/{campaignId}/documents", service.ListDocuments).Methods("GET")
	r.HandleFunc("/campaigns/{campaignId}/search", service.SearchDocuments).Methods("POST")
	r.HandleFunc("/generate", service.GenerateContent).Methods("POST")
	
	// Start server
	log.Printf("Starting Campaign Content Studio API on port %s", port)
	log.Printf("  n8n URL: %s", n8nURL)
	log.Printf("  Windmill URL: %s", windmillURL)
	log.Printf("  Postgres URL: %s", postgresURL)
	log.Printf("  Qdrant URL: %s", qdrantURL)
	log.Printf("  MinIO URL: %s", minioURL)
	
	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}
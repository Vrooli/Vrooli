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
	serviceName = "metareasoning-coordinator"
	
	// Defaults
	defaultEmbeddingModel = "nomic-embed-text"
	defaultWorkspace = "demo"
	defaultPort = "8090"
	
	// Timeouts
	httpTimeout = 30 * time.Second
	discoveryDelay = 5 * time.Second
	rediscoveryInterval = 5 * time.Minute
	
	// Database limits
	maxDBConnections = 25
	maxIdleConnections = 5
	connMaxLifetime = 5 * time.Minute
	
	// Search
	defaultSearchLimit = 10
	minSimilarityScore = 0.3
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[metareasoning-api] ", log.LstdFlags|log.Lshortfile),
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

// WorkflowMetadata represents lightweight workflow reference
type WorkflowMetadata struct {
	ID           uuid.UUID              `json:"id"`
	Platform     string                 `json:"platform"`
	PlatformID   string                 `json:"platform_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Capabilities map[string]interface{} `json:"capabilities"`
	UsageCount   int                    `json:"usage_count"`
	LastUsed     *time.Time             `json:"last_used,omitempty"`
}

// DiscoveryService handles workflow discovery from platforms
type DiscoveryService struct {
	db          *sql.DB
	n8nBaseURL  string
	windmillURL string
	qdrantURL   string
	httpClient  *http.Client
	logger      *Logger
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(db *sql.DB, n8nURL, windmillURL, qdrantURL string) *DiscoveryService {
	return &DiscoveryService{
		db:          db,
		n8nBaseURL:  n8nURL,
		windmillURL: windmillURL,
		qdrantURL:   qdrantURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:      NewLogger(),
	}
}

// DiscoverAndRegister queries platforms and registers workflows
func (d *DiscoveryService) DiscoverAndRegister() error {
	d.logger.Info("Starting workflow discovery...")
	
	// Initialize Qdrant collection if needed
	if err := d.initializeQdrantCollection(); err != nil {
		d.logger.Error("Failed to initialize Qdrant collection", err)
		// Continue anyway - semantic search will be disabled
	}
	
	// Discover workflows from all platforms
	platforms := []string{"n8n", "windmill"}
	for _, platform := range platforms {
		if err := d.discoverPlatformWorkflows(platform); err != nil {
			d.logger.Warn(fmt.Sprintf("Failed to discover %s workflows", platform), err)
		}
	}
	
	d.logger.Info("Workflow discovery completed")
	return nil
}

// initializeQdrantCollection ensures the workflow embeddings collection exists
func (d *DiscoveryService) initializeQdrantCollection() error {
	// Check if collection exists
	resp, err := d.httpClient.Get(fmt.Sprintf("%s/collections/workflow_embeddings", d.qdrantURL))
	if err != nil {
		return fmt.Errorf("could not check Qdrant collection: %w", err)
	}
	defer resp.Body.Close()
	
	// If collection exists, we're done
	if resp.StatusCode == 200 {
		d.logger.Info("Qdrant collection 'workflow_embeddings' already exists")
		return nil
	}
	
	// Get embedding dimensions from the model
	// nomic-embed-text produces 768-dimensional embeddings
	// mxbai-embed-large produces 1024-dimensional embeddings
	// all-minilm produces 384-dimensional embeddings
	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = defaultEmbeddingModel
	}
	
	embeddingSize := 768 // Default for nomic-embed-text
	switch embeddingModel {
	case "mxbai-embed-large":
		embeddingSize = 1024
	case "all-minilm":
		embeddingSize = 384
	case "nomic-embed-text":
		embeddingSize = 768
	default:
		// Try to detect by generating a test embedding
		testEmbedding := d.generateEmbedding("test")
		if len(testEmbedding) > 0 {
			embeddingSize = len(testEmbedding)
			d.logger.Info(fmt.Sprintf("Detected embedding size: %d for model %s", embeddingSize, embeddingModel))
		}
	}
	
	// Create collection
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size": embeddingSize,  // Match actual embedding dimension
			"distance": "Cosine",
		},
	}
	
	configBody, err := json.Marshal(collectionConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal collection config: %w", err)
	}
	
	createReq, err := http.NewRequest("PUT",
		fmt.Sprintf("%s/collections/workflow_embeddings", d.qdrantURL),
		bytes.NewBuffer(configBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := d.httpClient.Do(createReq)
	if err != nil {
		return fmt.Errorf("failed to create Qdrant collection: %w", err)
	}
	defer createResp.Body.Close()
	
	if createResp.StatusCode < 400 {
		d.logger.Info("Created Qdrant collection 'workflow_embeddings'")
		return nil
	} else {
		return fmt.Errorf("Qdrant returned status %d when creating collection", createResp.StatusCode)
	}
}

// discoverPlatformWorkflows is a consolidated discovery method
func (d *DiscoveryService) discoverPlatformWorkflows(platform string) error {
	var url string
	var idField, nameField string
	
	switch platform {
	case "n8n":
		url = fmt.Sprintf("%s/rest/workflows", d.n8nBaseURL)
		idField = "id"
		nameField = "name"
	case "windmill":
		url = fmt.Sprintf("%s/api/w/demo/apps", d.windmillURL)
		idField = "path"
		nameField = "summary"
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
	
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return err
	}
	
	// Register metareasoning workflows
	for _, item := range items {
		id, _ := item[idField].(string)
		name, _ := item[nameField].(string)
		
		// For n8n, also check the actual name field
		if platform == "n8n" {
			actualName, _ := item["name"].(string)
			if actualName != "" {
				name = actualName
			}
		}
		
		if d.isMetareasoningWorkflow(name) {
			tags := []string{}
			if platform == "n8n" {
				tags = extractTags(item)
			}
			d.registerWorkflow(platform, id, name, tags)
		}
	}
	
	return nil
}

func (d *DiscoveryService) isMetareasoningWorkflow(name string) bool {
	patterns := []string{"pros-cons", "swot", "risk", "decision", "self-review", "reasoning", "metareasoning"}
	nameLower := strings.ToLower(name)
	for _, pattern := range patterns {
		if strings.Contains(nameLower, pattern) {
			return true
		}
	}
	return false
}

func (d *DiscoveryService) registerWorkflow(platform, platformID, name string, tags []string) error {
	// Generate a unique embedding ID for this workflow
	embeddingID := fmt.Sprintf("%s_%s_%s", platform, platformID, uuid.New().String()[:8])
	
	// Insert workflow metadata into database
	_, err := d.db.Exec(`
		INSERT INTO workflow_registry (platform, platform_id, name, tags, embedding_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (platform, platform_id) 
		DO UPDATE SET name = $3, embedding_id = $5, updated_at = CURRENT_TIMESTAMP`,
		platform, platformID, name, tags, embeddingID)
	
	if err != nil {
		return err
	}
	
	// Create and store embedding in Qdrant (async to not block discovery)
	go d.createAndStoreEmbedding(embeddingID, name, tags)
	
	return nil
}

func extractTags(workflow map[string]interface{}) []string {
	// Extract tags from workflow metadata if available
	if tags, ok := workflow["tags"].([]interface{}); ok {
		result := make([]string, len(tags))
		for i, tag := range tags {
			result[i] = fmt.Sprintf("%v", tag)
		}
		return result
	}
	return []string{}
}

// scanWorkflowRow is a helper to scan workflow rows and parse tags
func scanWorkflowRow(rows *sql.Rows, includeLast bool) (*WorkflowMetadata, error) {
	var wf WorkflowMetadata
	var tagsArray sql.NullString
	
	if includeLast {
		err := rows.Scan(&wf.ID, &wf.Platform, &wf.PlatformID, &wf.Name,
			&wf.Description, &wf.Category, &tagsArray, &wf.UsageCount, &wf.LastUsed)
		if err != nil {
			return nil, err
		}
	} else {
		err := rows.Scan(&wf.ID, &wf.Platform, &wf.PlatformID, &wf.Name,
			&wf.Description, &wf.Category, &tagsArray, &wf.UsageCount)
		if err != nil {
			return nil, err
		}
	}
	
	// Parse tags
	if tagsArray.Valid {
		wf.Tags = strings.Split(strings.Trim(tagsArray.String, "{}"), ",")
	}
	
	return &wf, nil
}

// API Handlers

// ListWorkflows returns all discovered workflows
func (d *DiscoveryService) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.Query(`
		SELECT id, platform, platform_id, name, description, category, tags, usage_count, last_used
		FROM workflow_registry
		ORDER BY usage_count DESC, name`)
	if err != nil {
		HTTPError(w, "Failed to query workflows", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	
	var workflows []WorkflowMetadata
	for rows.Next() {
		wf, err := scanWorkflowRow(rows, true)
		if err != nil {
			continue
		}
		workflows = append(workflows, *wf)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

// ExecuteWorkflow proxies execution to the appropriate platform
func (d *DiscoveryService) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	platform := vars["platform"]
	workflowID := vars["workflowId"]
	
	// Log execution start
	execID := uuid.New()
	_, err := d.db.Exec(`
		INSERT INTO execution_log (id, workflow_id, platform, status, started_at)
		SELECT $1, id, $2, 'running', NOW()
		FROM workflow_registry
		WHERE platform = $2 AND platform_id = $3`,
		execID, platform, workflowID)
	if err != nil {
		log.Printf("Failed to log execution start: %v", err)
	}
	
	startTime := time.Now()
	
	// Proxy to appropriate platform
	result, execErr := d.executePlatformWorkflow(platform, workflowID, r)
	if execErr != nil && strings.Contains(execErr.Error(), "unsupported platform") {
		HTTPError(w, "Unknown platform", http.StatusBadRequest, execErr)
		return
	}
	
	// Log execution completion
	status := "success"
	if execErr != nil {
		status = "failed"
	}
	
	executionTime := int(time.Since(startTime).Milliseconds())
	_, err = d.db.Exec(`
		UPDATE execution_log 
		SET status = $1, execution_time_ms = $2, completed_at = NOW()
		WHERE id = $3`,
		status, executionTime, execID)
	if err != nil {
		log.Printf("Failed to log execution completion: %v", err)
	}
	
	// Update workflow usage stats
	_, err = d.db.Exec(`
		UPDATE workflow_registry 
		SET usage_count = usage_count + 1, last_used = NOW()
		WHERE platform = $1 AND platform_id = $2`,
		platform, workflowID)
	if err != nil {
		d.logger.Error("Failed to update workflow usage stats", err)
	}
	
	// Return result
	if execErr != nil {
		HTTPError(w, "Workflow execution failed", http.StatusInternalServerError, execErr)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// executePlatformWorkflow is a consolidated method to execute workflows on any platform
func (d *DiscoveryService) executePlatformWorkflow(platform, workflowID string, r *http.Request) (map[string]interface{}, error) {
	var targetURL string
	var method string
	
	switch platform {
	case "n8n":
		targetURL = fmt.Sprintf("%s/webhook/%s", d.n8nBaseURL, workflowID)
		method = r.Method // n8n webhooks support various methods
	case "windmill":
		targetURL = fmt.Sprintf("%s/api/w/demo/jobs/run/p/%s", d.windmillURL, workflowID)
		method = "POST" // Windmill always uses POST
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	
	// Create proxy request
	proxyReq, err := http.NewRequest(method, targetURL, r.Body)
	if err != nil {
		return nil, err
	}
	proxyReq.Header = r.Header
	
	// Execute
	resp, err := d.httpClient.Do(proxyReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// createAndStoreEmbedding creates a text embedding and stores it in Qdrant
func (d *DiscoveryService) createAndStoreEmbedding(embeddingID, text string, tags []string) {
	// Create search text from name and tags
	searchText := text + " " + strings.Join(tags, " ")
	
	// Use Ollama to generate embeddings (using a small model like nomic-embed-text)
	// This is a simplified approach - in production you'd use a proper embedding model
	embedding := d.generateEmbedding(searchText)
	if len(embedding) == 0 {
		log.Printf("Failed to generate embedding for %s", embeddingID)
		return
	}
	
	// Store in Qdrant
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id": embeddingID,
				"vector": embedding,
				"payload": map[string]interface{}{
					"text": text,
					"tags": tags,
				},
			},
		},
	}
	
	reqBody, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/collections/workflow_embeddings/points", d.qdrantURL),
		bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to store embedding in Qdrant: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		log.Printf("Qdrant returned error status %d for embedding %s", resp.StatusCode, embeddingID)
	}
}

// generateEmbedding creates a text embedding using Ollama's embedding models
func (d *DiscoveryService) generateEmbedding(text string) []float32 {
	// Use Ollama's embedding endpoint with a proper embedding model
	// Common embedding models: nomic-embed-text, mxbai-embed-large, all-minilm
	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = defaultEmbeddingModel
	}
	
	// Get Ollama port from port registry
	ollamaPort := getResourcePort("ollama")
	ollamaURL := fmt.Sprintf("http://localhost:%s", ollamaPort)
	
	// Prepare the embedding request
	payload := map[string]interface{}{
		"model":  embeddingModel,
		"prompt": text,
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		d.logger.Error("Failed to marshal embedding request", err)
		return nil
	}
	
	// Call Ollama's embedding endpoint
	resp, err := d.httpClient.Post(
		fmt.Sprintf("%s/api/embeddings", ollamaURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		d.logger.Error("Failed to get embedding from Ollama", err)
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		d.logger.Error("Ollama returned non-200 status for embedding", fmt.Errorf("status: %d", resp.StatusCode))
		return nil
	}
	
	// Parse the response
	var embeddingResp struct {
		Embedding []float64 `json:"embedding"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		d.logger.Error("Failed to decode Ollama embedding response", err)
		return nil
	}
	
	// Convert float64 to float32 for Qdrant compatibility
	embedding := make([]float32, len(embeddingResp.Embedding))
	for i, v := range embeddingResp.Embedding {
		embedding[i] = float32(v)
	}
	
	// Log successful embedding generation
	d.logger.Info(fmt.Sprintf("Generated embedding with %d dimensions for text: %.50s...", len(embedding), text))
	
	return embedding
}

// SearchWorkflows performs semantic search using Qdrant
func (d *DiscoveryService) SearchWorkflows(w http.ResponseWriter, r *http.Request) {
	var searchReq struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	if searchReq.Limit == 0 {
		searchReq.Limit = defaultSearchLimit
	}
	
	// Try semantic search first with Qdrant
	semanticResults := d.semanticSearch(searchReq.Query, searchReq.Limit)
	
	// If semantic search returns results, use them
	if len(semanticResults) > 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(semanticResults)
		return
	}
	
	// Fallback to keyword search if Qdrant is unavailable or returns no results
	query := "%" + strings.ToLower(searchReq.Query) + "%"
	rows, err := d.db.Query(`
		SELECT id, platform, platform_id, name, description, category, tags, usage_count
		FROM workflow_registry
		WHERE LOWER(name) LIKE $1 
		   OR LOWER(description) LIKE $1
		   OR LOWER(category) LIKE $1
		ORDER BY usage_count DESC
		LIMIT $2`, query, searchReq.Limit)
	if err != nil {
		HTTPError(w, "Failed to search workflows", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	
	var workflows []WorkflowMetadata
	for rows.Next() {
		wf, err := scanWorkflowRow(rows, false)
		if err != nil {
			continue
		}
		workflows = append(workflows, *wf)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

// semanticSearch performs vector similarity search using Qdrant
func (d *DiscoveryService) semanticSearch(query string, limit int) []WorkflowMetadata {
	// Generate embedding for the query
	queryEmbedding := d.generateEmbedding(query)
	if len(queryEmbedding) == 0 {
		d.logger.Warn("Failed to generate embedding for semantic search", fmt.Errorf("query: %s", query))
		return nil
	}
	
	d.logger.Info(fmt.Sprintf("Performing semantic search with %d-dimensional embedding", len(queryEmbedding)))
	
	// Search in Qdrant
	searchPayload := map[string]interface{}{
		"vector": queryEmbedding,
		"limit": limit,
		"with_payload": true,
		"score_threshold": minSimilarityScore,
	}
	
	reqBody, _ := json.Marshal(searchPayload)
	resp, err := d.httpClient.Post(
		fmt.Sprintf("%s/collections/workflow_embeddings/points/search", d.qdrantURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		log.Printf("Qdrant search failed: %v", err)
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		log.Printf("Qdrant search returned status %d", resp.StatusCode)
		return nil
	}
	
	// Parse Qdrant response
	var qdrantResp struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32               `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&qdrantResp); err != nil {
		log.Printf("Failed to decode Qdrant response: %v", err)
		return nil
	}
	
	// Fetch workflow details from database based on embedding IDs
	if len(qdrantResp.Result) == 0 {
		return nil
	}
	
	var embeddingIDs []string
	for _, result := range qdrantResp.Result {
		embeddingIDs = append(embeddingIDs, result.ID)
	}
	
	// Build query with placeholders
	placeholders := make([]string, len(embeddingIDs))
	args := make([]interface{}, len(embeddingIDs))
	for i, id := range embeddingIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	
	sqlQuery := fmt.Sprintf(`
		SELECT id, platform, platform_id, name, description, category, tags, usage_count
		FROM workflow_registry
		WHERE embedding_id IN (%s)
		ORDER BY array_position(ARRAY[%s]::text[], embedding_id)
	`, strings.Join(placeholders, ","), strings.Join(placeholders, ","))
	
	// Double the args for the two placeholders in the query
	doubledArgs := append(args, args...)
	rows, err := d.db.Query(sqlQuery, doubledArgs...)
	if err != nil {
		log.Printf("Failed to fetch workflows: %v", err)
		return nil
	}
	defer rows.Close()
	
	var workflows []WorkflowMetadata
	for rows.Next() {
		wf, err := scanWorkflowRow(rows, false)
		if err != nil {
			continue
		}
		workflows = append(workflows, *wf)
	}
	
	return workflows
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
		"source /home/matthalloran8/Vrooli/scripts/resources/port-registry.sh && resources::get_default_port %s",
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
	
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
	}
	
	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillURL = fmt.Sprintf("http://localhost:%s", windmillPort)
	}
	
	qdrantURL := os.Getenv("QDRANT_BASE_URL")
	if qdrantURL == "" {
		qdrantURL = fmt.Sprintf("http://localhost:%s", qdrantPort)
	}
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%s/metareasoning?sslmode=disable", postgresPort)
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
	
	// Test database connection
	if err := db.Ping(); err != nil {
		logger := NewLogger()
		logger.Error("Failed to ping database", err)
		os.Exit(1)
	}
	
	log.Println("Connected to database")
	
	// Initialize discovery service
	discovery := NewDiscoveryService(db, n8nURL, windmillURL, qdrantURL)
	
	// Start discovery in background
	go func() {
		time.Sleep(discoveryDelay) // Let platforms initialize
		if err := discovery.DiscoverAndRegister(); err != nil {
			log.Printf("Discovery failed: %v", err)
		}
		
		// Periodic re-discovery
		ticker := time.NewTicker(rediscoveryInterval)
		for range ticker.C {
			if err := discovery.DiscoverAndRegister(); err != nil {
				log.Printf("Re-discovery failed: %v", err)
			}
		}
	}()
	
	// Setup routes
	r := mux.NewRouter()
	
	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/workflows", discovery.ListWorkflows).Methods("GET")
	r.HandleFunc("/workflows/search", discovery.SearchWorkflows).Methods("POST")
	r.HandleFunc("/execute/{platform}/{workflowId}", discovery.ExecuteWorkflow).Methods("POST")
	
	// Start server
	log.Printf("Starting Metareasoning Coordinator API on port %s", port)
	log.Printf("  n8n URL: %s", n8nURL)
	log.Printf("  Windmill URL: %s", windmillURL)
	log.Printf("  Qdrant URL: %s", qdrantURL)
	log.Printf("  Database: %s", dbURL)
	
	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}
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
	serviceName = "agent-metareasoning-manager"
	
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
		Logger: log.New(os.Stdout, "[agent-metareasoning-manager-api] ", log.LstdFlags|log.Lshortfile),
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
	
	embeddingSize := 384 // Default for most models (vector workflow uses 384)
	switch embeddingModel {
	case "mxbai-embed-large":
		embeddingSize = 1024
	case "all-minilm":
		embeddingSize = 384
	case "nomic-embed-text":
		embeddingSize = 768
	case "llama3.2":
		embeddingSize = 384 // Default for the vector workflow
	default:
		// Try to detect by generating a test embedding via workflow
		testMetadata := map[string]interface{}{
			"point_id": "test_embedding_size_detection",
			"pattern_type": "test",
			"pattern_name": "size detection",
		}
		testResp, err := d.generateEmbeddingWithWorkflow("test", "workflow_embeddings", testMetadata)
		if err == nil && testResp.EmbeddingDimension > 0 {
			embeddingSize = testResp.EmbeddingDimension
			d.logger.Info(fmt.Sprintf("Detected embedding size: %d for model %s via workflow", embeddingSize, embeddingModel))
			
			// Clean up test embedding
			deleteURL := fmt.Sprintf("%s/collections/workflow_embeddings/points/%s", d.qdrantURL, testResp.PointID)
			deleteReq, _ := http.NewRequest("DELETE", deleteURL, nil)
			d.httpClient.Do(deleteReq) // Best effort cleanup
		} else {
			d.logger.Warn("Failed to detect embedding size via workflow, using default 384", err)
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

// createAndStoreEmbedding creates a text embedding and stores it in Qdrant using vector conversion workflow
func (d *DiscoveryService) createAndStoreEmbedding(embeddingID, text string, tags []string) {
	// Create search text from name and tags
	searchText := text + " " + strings.Join(tags, " ")
	
	// Prepare metadata for workflow_embeddings collection
	metadata := map[string]interface{}{
		"point_id": embeddingID,
		"pattern_type": "discovered_workflow",
		"pattern_name": text,
		"description": searchText,
		"tags": tags,
		"usage_count": 0,
		"effectiveness_score": 0.0,
		"workflow_reference": embeddingID, // Reference to the workflow
	}
	
	// Delegate to vector conversion workflow
	resp, err := d.generateEmbeddingWithWorkflow(searchText, "workflow_embeddings", metadata)
	if err != nil {
		d.logger.Error(fmt.Sprintf("Failed to generate embedding for %s via workflow", embeddingID), err)
		return
	}
	
	d.logger.Info(fmt.Sprintf("Successfully created and stored embedding for %s (ID: %s) in %dms", 
		text, resp.PointID, resp.ExecutionTimeMS))
}

// VectorWorkflowResponse represents the response from vector conversion workflow
type VectorWorkflowResponse struct {
	Status            string `json:"status"`
	PointID           string `json:"point_id"`
	Collection        string `json:"collection"`
	EmbeddingDimension int   `json:"embedding_dimension"`
	ExecutionTimeMS   int    `json:"execution_time_ms"`
	ModelUsed         string `json:"model_used"`
	Error            string `json:"error,omitempty"`
}

// generateEmbeddingWithWorkflow delegates embedding generation to vector conversion workflow
func (d *DiscoveryService) generateEmbeddingWithWorkflow(text string, collection string, metadata map[string]interface{}) (*VectorWorkflowResponse, error) {
	// Delegate to vector conversion workflow instead of direct Ollama calls
	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = defaultEmbeddingModel
	}
	
	// Get n8n port for workflow delegation
	n8nPort := getResourcePort("n8n")
	workflowURL := fmt.Sprintf("http://localhost:%s/webhook/vector-conversion", n8nPort)
	
	// Prepare workflow request
	workflowReq := map[string]interface{}{
		"text":       text,
		"collection": collection,
		"embedding_model": embeddingModel,
	}
	
	// Add metadata based on collection type
	for key, value := range metadata {
		workflowReq[key] = value
	}
	
	reqBody, err := json.Marshal(workflowReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow request: %w", err)
	}
	
	// Call vector conversion workflow
	resp, err := d.httpClient.Post(workflowURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call vector conversion workflow: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vector conversion workflow returned status %d", resp.StatusCode)
	}
	
	// Parse workflow response
	var workflowResp VectorWorkflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&workflowResp); err != nil {
		return nil, fmt.Errorf("failed to decode workflow response: %w", err)
	}
	
	// Check if workflow succeeded
	if workflowResp.Status != "success" {
		return nil, fmt.Errorf("vector conversion workflow failed: %s", workflowResp.Error)
	}
	
	d.logger.Info(fmt.Sprintf("Generated %d-dimensional embedding via workflow in %dms", 
		workflowResp.EmbeddingDimension, workflowResp.ExecutionTimeMS))
	
	return &workflowResp, nil
}

// generateEmbedding creates a text embedding using vector conversion workflow (legacy API)
func (d *DiscoveryService) generateEmbedding(text string) []float32 {
	// This function maintained for backward compatibility with search functionality
	// For new code, use generateEmbeddingWithWorkflow directly
	
	resp, err := d.generateEmbeddingWithWorkflow(text, "execution_embeddings", map[string]interface{}{
		"execution_id": fmt.Sprintf("search_%d", time.Now().Unix()),
		"workflow_type": "search_query",
		"status": "search",
	})
	
	if err != nil {
		d.logger.Error("Failed to generate embedding via workflow", err)
		return nil
	}
	
	// Note: We can't return the actual embedding vector from the workflow response
	// because the workflow stores it directly in Qdrant. For search, we'll modify
	// the search function to use the stored embedding directly.
	// This legacy function now just validates the workflow succeeded.
	if resp.EmbeddingDimension > 0 {
		// Return a placeholder to indicate success - actual embedding is in Qdrant
		placeholder := make([]float32, resp.EmbeddingDimension)
		return placeholder
	}
	
	return nil
}

// AnalyzeRequest represents incoming analysis request
type AnalyzeRequest struct {
	Type    string `json:"type"`
	Input   string `json:"input"`
	Context string `json:"context,omitempty"`
	Model   string `json:"model,omitempty"`
}

// AnalyzeWorkflow handles intelligent analysis type routing to appropriate workflows
func (d *DiscoveryService) AnalyzeWorkflow(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}
	
	if req.Type == "" || req.Input == "" {
		HTTPError(w, "Missing required fields: type and input", http.StatusBadRequest, nil)
		return
	}
	
	// Map analysis types to workflow IDs and platforms
	workflowMapping := map[string]struct {
		Platform   string
		WorkflowID string
	}{
		"pros-cons":       {"n8n", "pros-cons-analyzer"},
		"swot":           {"n8n", "swot-analysis"},
		"risk":           {"n8n", "risk-assessment"},
		"risk-assessment": {"n8n", "risk-assessment"},
		"self-review":    {"n8n", "self-review"},
		"reasoning-chain": {"n8n", "reasoning-chain"},
		"decision":       {"windmill", "decision-analyzer"},
	}
	
	mapping, exists := workflowMapping[req.Type]
	if !exists {
		// Try to find workflow by name pattern matching
		rows, err := d.db.Query(`
			SELECT platform, platform_id 
			FROM workflow_registry 
			WHERE LOWER(name) LIKE $1 
			ORDER BY usage_count DESC 
			LIMIT 1`, "%"+strings.ToLower(req.Type)+"%")
		if err != nil {
			HTTPError(w, "Failed to resolve analysis type", http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()
		
		if !rows.Next() {
			HTTPError(w, fmt.Sprintf("Unknown analysis type: %s", req.Type), http.StatusBadRequest, nil)
			return
		}
		
		if err := rows.Scan(&mapping.Platform, &mapping.WorkflowID); err != nil {
			HTTPError(w, "Failed to resolve workflow mapping", http.StatusInternalServerError, err)
			return
		}
	}
	
	// Build execution payload
	payload := map[string]interface{}{
		"input":   req.Input,
		"context": req.Context,
		"model":   req.Model,
	}
	
	if req.Model == "" {
		payload["model"] = "llama3.2" // Default model
	}
	
	// Create a mock HTTP request for execution
	payloadBytes, _ := json.Marshal(payload)
	mockReq, _ := http.NewRequest("POST", "", bytes.NewReader(payloadBytes))
	mockReq.Header.Set("Content-Type", "application/json")
	
	// Execute the workflow
	result, err := d.executePlatformWorkflow(mapping.Platform, mapping.WorkflowID, mockReq)
	if err != nil {
		HTTPError(w, "Workflow execution failed", http.StatusInternalServerError, err)
		return
	}
	
	// Return enriched result with metadata
	response := map[string]interface{}{
		"analysis_type": req.Type,
		"platform":     mapping.Platform,
		"workflow_id":  mapping.WorkflowID,
		"result":       result,
		"metadata": map[string]interface{}{
			"model_used": req.Model,
			"timestamp": time.Now().UTC(),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

// semanticSearchWithWorkflow performs vector similarity search using workflow delegation and Qdrant
func (d *DiscoveryService) semanticSearchWithWorkflow(query string, limit int) []WorkflowMetadata {
	// Generate query embedding using vector conversion workflow  
	queryMetadata := map[string]interface{}{
		"execution_id": fmt.Sprintf("search_%d", time.Now().Unix()),
		"workflow_type": "semantic_search",
		"status": "search_query",
	}
	
	// Store query embedding in execution_embeddings collection
	queryResp, err := d.generateEmbeddingWithWorkflow(query, "execution_embeddings", queryMetadata)
	if err != nil {
		d.logger.Error("Failed to generate search embedding via workflow", err)
		return nil
	}
	
	d.logger.Info(fmt.Sprintf("Generated %d-dimensional search embedding in %dms", 
		queryResp.EmbeddingDimension, queryResp.ExecutionTimeMS))
	
	// Now we need to retrieve the embedding from Qdrant to perform the search
	// Get the stored embedding point
	getPointURL := fmt.Sprintf("%s/collections/execution_embeddings/points/%s", d.qdrantURL, queryResp.PointID)
	pointResp, err := d.httpClient.Get(getPointURL)
	if err != nil {
		d.logger.Error("Failed to retrieve stored query embedding", err)
		return nil
	}
	defer pointResp.Body.Close()
	
	if pointResp.StatusCode >= 400 {
		d.logger.Error("Failed to retrieve query embedding from Qdrant", fmt.Errorf("status: %d", pointResp.StatusCode))
		return nil
	}
	
	// Parse the embedding point response
	var pointData struct {
		Result struct {
			Vector []float32 `json:"vector"`
		} `json:"result"`
	}
	
	if err := json.NewDecoder(pointResp.Body).Decode(&pointData); err != nil {
		d.logger.Error("Failed to decode embedding point response", err)
		return nil
	}
	
	// Perform the actual search using the retrieved embedding
	return d.performQdrantSearch(pointData.Result.Vector, limit)
}

// performQdrantSearch performs the actual vector search in Qdrant
func (d *DiscoveryService) performQdrantSearch(queryEmbedding []float32, limit int) []WorkflowMetadata {
	// Search in workflow_embeddings collection
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
		d.logger.Error("Qdrant search failed", err)
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		d.logger.Error("Qdrant search failed", fmt.Errorf("status: %d", resp.StatusCode))
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
		d.logger.Error("Failed to decode Qdrant search response", err)
		return nil
	}
	
	// Fetch workflow details from database based on embedding IDs
	if len(qdrantResp.Result) == 0 {
		d.logger.Info("No similar workflows found")
		return nil
	}
	
	var embeddingIDs []string
	for _, result := range qdrantResp.Result {
		embeddingIDs = append(embeddingIDs, result.ID)
	}
	
	// Build query with placeholders for IN clause
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
		d.logger.Error("Failed to fetch workflows from database", err)
		return nil
	}
	defer rows.Close()
	
	var workflows []WorkflowMetadata
	for rows.Next() {
		wf, err := scanWorkflowRow(rows, false)
		if err != nil {
			d.logger.Warn("Failed to scan workflow row", err)
			continue
		}
		workflows = append(workflows, *wf)
	}
	
	d.logger.Info(fmt.Sprintf("Found %d similar workflows", len(workflows)))
	return workflows
}

// semanticSearch performs vector similarity search (legacy wrapper)
func (d *DiscoveryService) semanticSearch(query string, limit int) []WorkflowMetadata {
	// Delegate to new workflow-based search implementation
	return d.semanticSearchWithWorkflow(query, limit)
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
	r.HandleFunc("/analyze", discovery.AnalyzeWorkflow).Methods("POST")
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
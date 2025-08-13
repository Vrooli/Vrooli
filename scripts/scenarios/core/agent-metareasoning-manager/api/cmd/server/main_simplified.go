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
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(db *sql.DB, n8nURL, windmillURL, qdrantURL string) *DiscoveryService {
	return &DiscoveryService{
		db:          db,
		n8nBaseURL:  n8nURL,
		windmillURL: windmillURL,
		qdrantURL:   qdrantURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DiscoverAndRegister queries platforms and registers workflows
func (d *DiscoveryService) DiscoverAndRegister() error {
	log.Println("Starting workflow discovery...")
	
	// Initialize Qdrant collection if needed
	d.initializeQdrantCollection()
	
	// Discover n8n workflows
	if err := d.discoverN8nWorkflows(); err != nil {
		log.Printf("Warning: Failed to discover n8n workflows: %v", err)
	}
	
	// Discover Windmill apps
	if err := d.discoverWindmillApps(); err != nil {
		log.Printf("Warning: Failed to discover Windmill apps: %v", err)
	}
	
	log.Println("Workflow discovery completed")
	return nil
}

// initializeQdrantCollection ensures the workflow embeddings collection exists
func (d *DiscoveryService) initializeQdrantCollection() {
	// Check if collection exists
	resp, err := d.httpClient.Get(fmt.Sprintf("%s/collections/workflow_embeddings", d.qdrantURL))
	if err != nil {
		log.Printf("Warning: Could not check Qdrant collection: %v", err)
		return
	}
	defer resp.Body.Close()
	
	// If collection exists, we're done
	if resp.StatusCode == 200 {
		log.Println("Qdrant collection 'workflow_embeddings' already exists")
		return
	}
	
	// Create collection
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size": 384,  // Match our embedding dimension
			"distance": "Cosine",
		},
	}
	
	configBody, _ := json.Marshal(collectionConfig)
	createReq, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/collections/workflow_embeddings", d.qdrantURL),
		bytes.NewBuffer(configBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := d.httpClient.Do(createReq)
	if err != nil {
		log.Printf("Warning: Failed to create Qdrant collection: %v", err)
		return
	}
	defer createResp.Body.Close()
	
	if createResp.StatusCode < 400 {
		log.Println("Created Qdrant collection 'workflow_embeddings'")
	} else {
		log.Printf("Warning: Qdrant returned status %d when creating collection", createResp.StatusCode)
	}
}

func (d *DiscoveryService) discoverN8nWorkflows() error {
	// Query n8n for workflows
	resp, err := d.httpClient.Get(fmt.Sprintf("%s/rest/workflows", d.n8nBaseURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var workflows []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workflows); err != nil {
		return err
	}
	
	// Register metareasoning workflows
	for _, wf := range workflows {
		name, _ := wf["name"].(string)
		if d.isMetareasoningWorkflow(name) {
			d.registerWorkflow("n8n", wf["id"].(string), name, extractTags(wf))
		}
	}
	
	return nil
}

func (d *DiscoveryService) discoverWindmillApps() error {
	// Query Windmill for apps
	resp, err := d.httpClient.Get(fmt.Sprintf("%s/api/w/demo/apps", d.windmillURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var apps []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return err
	}
	
	// Register metareasoning apps
	for _, app := range apps {
		path, _ := app["path"].(string)
		summary, _ := app["summary"].(string)
		if d.isMetareasoningWorkflow(path) || d.isMetareasoningWorkflow(summary) {
			d.registerWorkflow("windmill", path, summary, []string{})
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

// API Handlers

// ListWorkflows returns all discovered workflows
func (d *DiscoveryService) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.Query(`
		SELECT id, platform, platform_id, name, description, category, tags, usage_count, last_used
		FROM workflow_registry
		ORDER BY usage_count DESC, name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var workflows []WorkflowMetadata
	for rows.Next() {
		var wf WorkflowMetadata
		var tagsArray sql.NullString
		err := rows.Scan(&wf.ID, &wf.Platform, &wf.PlatformID, &wf.Name,
			&wf.Description, &wf.Category, &tagsArray, &wf.UsageCount, &wf.LastUsed)
		if err != nil {
			continue
		}
		// Parse tags
		if tagsArray.Valid {
			// Simple parsing of PostgreSQL array format
			wf.Tags = strings.Split(strings.Trim(tagsArray.String, "{}"), ",")
		}
		workflows = append(workflows, wf)
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
	var result map[string]interface{}
	var execErr error
	
	switch platform {
	case "n8n":
		result, execErr = d.executeN8nWorkflow(workflowID, r)
	case "windmill":
		result, execErr = d.executeWindmillJob(workflowID, r)
	default:
		http.Error(w, "Unknown platform", http.StatusBadRequest)
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
	_, _ = d.db.Exec(`
		UPDATE workflow_registry 
		SET usage_count = usage_count + 1, last_used = NOW()
		WHERE platform = $1 AND platform_id = $2`,
		platform, workflowID)
	
	// Return result
	if execErr != nil {
		http.Error(w, execErr.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (d *DiscoveryService) executeN8nWorkflow(workflowID string, r *http.Request) (map[string]interface{}, error) {
	// Forward request to n8n webhook
	webhookURL := fmt.Sprintf("%s/webhook/%s", d.n8nBaseURL, workflowID)
	
	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, webhookURL, r.Body)
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

func (d *DiscoveryService) executeWindmillJob(jobID string, r *http.Request) (map[string]interface{}, error) {
	// Forward request to Windmill
	jobURL := fmt.Sprintf("%s/api/w/demo/jobs/run/p/%s", d.windmillURL, jobID)
	
	// Create proxy request
	proxyReq, err := http.NewRequest("POST", jobURL, r.Body)
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

// generateEmbedding creates a simple text embedding using Ollama
func (d *DiscoveryService) generateEmbedding(text string) []float32 {
	// For simplicity, using a basic hash-based approach
	// In production, you'd call Ollama's embedding model or use a dedicated service
	// This creates a deterministic 384-dimensional vector from the text
	const dimensions = 384
	embedding := make([]float32, dimensions)
	
	// Simple hash-based embedding (replace with actual embedding model)
	for i, char := range text {
		idx := i % dimensions
		embedding[idx] += float32(char) / 1000.0
	}
	
	// Normalize
	var sum float32
	for _, v := range embedding {
		sum += v * v
	}
	if sum > 0 {
		norm := float32(1.0 / sqrt(float64(sum)))
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	
	return embedding
}

// sqrt helper function
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// SearchWorkflows performs semantic search using Qdrant
func (d *DiscoveryService) SearchWorkflows(w http.ResponseWriter, r *http.Request) {
	var searchReq struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if searchReq.Limit == 0 {
		searchReq.Limit = 10
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var workflows []WorkflowMetadata
	for rows.Next() {
		var wf WorkflowMetadata
		var tagsArray sql.NullString
		err := rows.Scan(&wf.ID, &wf.Platform, &wf.PlatformID, &wf.Name,
			&wf.Description, &wf.Category, &tagsArray, &wf.UsageCount)
		if err != nil {
			continue
		}
		if tagsArray.Valid {
			wf.Tags = strings.Split(strings.Trim(tagsArray.String, "{}"), ",")
		}
		workflows = append(workflows, wf)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

// semanticSearch performs vector similarity search using Qdrant
func (d *DiscoveryService) semanticSearch(query string, limit int) []WorkflowMetadata {
	// Generate embedding for the query
	queryEmbedding := d.generateEmbedding(query)
	if len(queryEmbedding) == 0 {
		return nil
	}
	
	// Search in Qdrant
	searchPayload := map[string]interface{}{
		"vector": queryEmbedding,
		"limit": limit,
		"with_payload": true,
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
		var wf WorkflowMetadata
		var tagsArray sql.NullString
		err := rows.Scan(&wf.ID, &wf.Platform, &wf.PlatformID, &wf.Name,
			&wf.Description, &wf.Category, &tagsArray, &wf.UsageCount)
		if err != nil {
			continue
		}
		if tagsArray.Valid {
			wf.Tags = strings.Split(strings.Trim(tagsArray.String, "{}"), ",")
		}
		workflows = append(workflows, wf)
	}
	
	return workflows
}

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "metareasoning-coordinator",
		"version": "2.0.0",
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
			port = "8090"
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
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	log.Println("Connected to database")
	
	// Initialize discovery service
	discovery := NewDiscoveryService(db, n8nURL, windmillURL, qdrantURL)
	
	// Start discovery in background
	go func() {
		time.Sleep(5 * time.Second) // Let platforms initialize
		if err := discovery.DiscoverAndRegister(); err != nil {
			log.Printf("Discovery failed: %v", err)
		}
		
		// Periodic re-discovery
		ticker := time.NewTicker(5 * time.Minute)
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
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed:", err)
	}
}
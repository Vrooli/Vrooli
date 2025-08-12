package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	httpClient  *http.Client
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(db *sql.DB, n8nURL, windmillURL string) *DiscoveryService {
	return &DiscoveryService{
		db:          db,
		n8nBaseURL:  n8nURL,
		windmillURL: windmillURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DiscoverAndRegister queries platforms and registers workflows
func (d *DiscoveryService) DiscoverAndRegister() error {
	log.Println("Starting workflow discovery...")
	
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
	_, err := d.db.Exec(`
		INSERT INTO workflow_registry (platform, platform_id, name, tags)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (platform, platform_id) 
		DO UPDATE SET name = $3, updated_at = CURRENT_TIMESTAMP`,
		platform, platformID, name, tags)
	return err
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

// SearchWorkflows performs semantic search (simplified version)
func (d *DiscoveryService) SearchWorkflows(w http.ResponseWriter, r *http.Request) {
	var searchReq struct {
		Query string `json:"query"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Simple keyword search for now (Qdrant integration would go here)
	query := "%" + strings.ToLower(searchReq.Query) + "%"
	rows, err := d.db.Query(`
		SELECT id, platform, platform_id, name, description, category, tags
		FROM workflow_registry
		WHERE LOWER(name) LIKE $1 
		   OR LOWER(description) LIKE $1
		   OR LOWER(category) LIKE $1
		ORDER BY usage_count DESC
		LIMIT 10`, query)
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
			&wf.Description, &wf.Category, &tagsArray)
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

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "metareasoning-coordinator",
		"version": "2.0.0",
	})
}

func main() {
	// Load configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}
	
	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillURL = "http://localhost:5681"
	}
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5433/metareasoning?sslmode=disable"
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
	discovery := NewDiscoveryService(db, n8nURL, windmillURL)
	
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
	
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed:", err)
	}
}
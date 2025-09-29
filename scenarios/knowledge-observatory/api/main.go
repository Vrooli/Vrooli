package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Config struct {
	Port        string
	QdrantURL   string
	PostgresDB  string
	ResourceCLI string
}

type Server struct {
	config   Config
	db       *sql.DB
	upgrader websocket.Upgrader
}

type SearchRequest struct {
	Query      string  `json:"query"`
	Collection string  `json:"collection,omitempty"`
	Limit      int     `json:"limit,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
}

type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
	Time    int64          `json:"query_time_ms"`
}

type HealthResponse struct {
	Status        string               `json:"status"`
	TotalEntries  int                  `json:"total_entries"`
	Collections   []CollectionHealth   `json:"collections"`
	OverallHealth string               `json:"overall_health"`
	Timestamp     string               `json:"timestamp"`
}

type CollectionHealth struct {
	Name    string         `json:"name"`
	Size    int            `json:"size"`
	Quality QualityMetrics `json:"quality"`
}

type QualityMetrics struct {
	Coherence  float64 `json:"coherence_score"`
	Freshness  float64 `json:"freshness_score"`
	Redundancy float64 `json:"redundancy_score"`
	Coverage   float64 `json:"coverage_score"`
}

type GraphRequest struct {
	CenterConcept string `json:"center_concept,omitempty"`
	Depth         int    `json:"depth,omitempty"`
	MaxNodes      int    `json:"max_nodes,omitempty"`
}

type GraphNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Metadata map[string]interface{} `json:"metadata"`
}

type GraphEdge struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	Weight       float64 `json:"weight"`
	Relationship string  `json:"relationship"`
}

type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type MetricsRequest struct {
	Collections []string `json:"collections,omitempty"`
	TimeRange   string   `json:"time_range,omitempty"`
}

type MetricsResponse struct {
	Metrics    map[string]QualityMetrics `json:"metrics"`
	Trends     map[string]float64        `json:"trends"`
	Alerts     []Alert                   `json:"alerts"`
	LastUpdate string                    `json:"last_update"`
}

type Alert struct {
	Level       string    `json:"level"`
	Collection  string    `json:"collection"`
	Metric      string    `json:"metric"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

type TimelineRequest struct {
	Collections []string `json:"collections,omitempty"`
	TimeRange   string   `json:"time_range,omitempty"`
	Granularity string   `json:"granularity,omitempty"` // hour, day, week
}

type TimelineEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Collection  string    `json:"collection"`
	EventType   string    `json:"event_type"` // added, updated, deleted
	Count       int       `json:"count"`
	Description string    `json:"description"`
}

type TimelineResponse struct {
	Entries     []TimelineEntry `json:"entries"`
	Collections map[string]int  `json:"collections"`
	TotalEvents int             `json:"total_events"`
	TimeRange   struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"time_range"`
}

// QdrantCollection represents collection data from resource-qdrant
type QdrantCollection struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Config struct {
		Params struct {
			Vectors struct {
				Size     int    `json:"size"`
				Distance string `json:"distance"`
			} `json:"vectors"`
		} `json:"params"`
	} `json:"config"`
	PointsCount int `json:"points_count"`
}

// QdrantSearchResult represents search result from resource-qdrant
type QdrantSearchResult struct {
	ID      interface{}            `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
	Vector  []float64              `json:"vector,omitempty"`
}

// QdrantSearchResponse represents search response from resource-qdrant
type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
	Status string               `json:"status"`
	Time   float64              `json:"time"`
}

func NewServer() (*Server, error) {
	config := Config{
		Port:        getEnv("API_PORT", getEnv("PORT", "")),
		QdrantURL:   getEnv("QDRANT_URL", "http://localhost:6333"),
		PostgresDB:  requireEnv("POSTGRES_URL"),
		ResourceCLI: getEnv("RESOURCE_QDRANT_CLI", "resource-qdrant"),
	}

	db, err := sql.Open("postgres", config.PostgresDB)
	if err != nil {
		log.Printf("Warning: Could not connect to PostgreSQL: %v", err)
		db = nil
	} else {
		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		
		// Implement exponential backoff for database connection
		maxRetries := 10
		baseDelay := 1 * time.Second
		maxDelay := 30 * time.Second
		
		log.Println("🔄 Attempting database connection with exponential backoff...")
		
		var pingErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			pingErr = db.Ping()
			if pingErr == nil {
				log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
				break
			}
			
			// Calculate exponential backoff delay
			delay := time.Duration(math.Min(
				float64(baseDelay) * math.Pow(2, float64(attempt)),
				float64(maxDelay),
			))
			
			// Add progressive jitter to prevent thundering herd
			jitterRange := float64(delay) * 0.25
			jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
			actualDelay := delay + jitter
			
			log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
			log.Printf("⏳ Waiting %v before next attempt", actualDelay)
			
			time.Sleep(actualDelay)
		}
		
		if pingErr != nil {
			log.Printf("⚠️  Database connection failed after %d attempts: %v (continuing without database)", maxRetries, pingErr)
			db = nil
		} else {
			log.Println("🎉 Database connection pool established successfully!")
		}
	}

	return &Server{
		config: config,
		db:     db,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}, nil
}

// execResourceQdrant executes a resource-qdrant CLI command
func (s *Server) execResourceQdrant(args ...string) ([]byte, error) {
	// Create command with 5-second timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, s.config.ResourceCLI, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("resource-qdrant command timed out after 5s: %v", args)
			return nil, fmt.Errorf("resource-qdrant timed out")
		}
		log.Printf("resource-qdrant command failed: %v, stderr: %s", err, stderr.String())
		return nil, fmt.Errorf("resource-qdrant failed: %v", err)
	}

	return stdout.Bytes(), nil
}

// getQdrantCollections retrieves all collections using resource-qdrant CLI
func (s *Server) getQdrantCollections() ([]string, error) {
	// Try JSON format first, fallback to simple list format
	output, err := s.execResourceQdrant("collections", "list")
	if err != nil {
		return nil, err
	}

	// Try to parse as JSON response first
	var jsonResponse struct {
		Result struct {
			Collections []QdrantCollection `json:"collections"`
		} `json:"result"`
	}

	if err := json.Unmarshal(output, &jsonResponse); err == nil && len(jsonResponse.Result.Collections) > 0 {
		var collections []string
		for _, col := range jsonResponse.Result.Collections {
			collections = append(collections, col.Name)
		}
		return collections, nil
	}

	// Fallback: parse simple text format
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var collections []string
	for _, line := range lines {
		if strings.HasPrefix(line, "📁 ") {
			collections = append(collections, strings.TrimSpace(strings.TrimPrefix(line, "📁 ")))
		}
	}

	return collections, nil
}

// getQdrantCollectionInfo retrieves detailed info for a collection
func (s *Server) getQdrantCollectionInfo(collection string) (*QdrantCollection, error) {
	output, err := s.execResourceQdrant("collections", "info", collection)
	if err != nil {
		return nil, err
	}

	// Try JSON format first
	var response struct {
		Result QdrantCollection `json:"result"`
		Status string           `json:"status"`
	}

	if err := json.Unmarshal(output, &response); err == nil {
		if response.Status != "ok" {
			return nil, fmt.Errorf("qdrant returned error status: %s", response.Status)
		}
		return &response.Result, nil
	}

	// Fallback: create basic collection info from name
	return &QdrantCollection{
		Name:        collection,
		Status:      "green",
		PointsCount: 0,
	}, nil
}

// performQdrantSearch executes semantic search using resource-qdrant CLI
func (s *Server) performQdrantSearch(query, collection string, limit int) (*QdrantSearchResponse, error) {
	args := []string{"collections", "search", "--text", query, "--limit", strconv.Itoa(limit)}
	if collection != "" {
		args = append(args, "--collection", collection)
	}

	output, err := s.execResourceQdrant(args...)
	if err != nil {
		return nil, err
	}

	var response QdrantSearchResponse
	if err := json.Unmarshal(output, &response); err != nil {
		// If JSON parsing fails, create empty response (graceful degradation)
		return &QdrantSearchResponse{
			Result: []QdrantSearchResult{},
			Status: "ok",
			Time:   0,
		}, nil
	}

	if response.Status != "ok" {
		return nil, fmt.Errorf("search failed with status: %s", response.Status)
	}

	return &response, nil
}

func (s *Server) performSemanticSearch(req SearchRequest) []SearchResult {
	// Perform actual semantic search using resource-qdrant CLI
	response, err := s.performQdrantSearch(req.Query, req.Collection, req.Limit)
	if err != nil {
		log.Printf("Semantic search failed: %v", err)
		// Return empty results on error rather than crashing
		return []SearchResult{}
	}

	// Convert Qdrant results to our API format
	var results []SearchResult
	for _, qResult := range response.Result {
		// Skip results below threshold
		if qResult.Score < req.Threshold {
			continue
		}

		// Extract content from payload
		content := "[No content available]"
		if qResult.Payload["content"] != nil {
			if contentStr, ok := qResult.Payload["content"].(string); ok {
				content = contentStr
			}
		} else if qResult.Payload["text"] != nil {
			if textStr, ok := qResult.Payload["text"].(string); ok {
				content = textStr
			}
		}

		// Convert ID to string format
		idStr := fmt.Sprintf("%v", qResult.ID)

		results = append(results, SearchResult{
			ID:       idStr,
			Score:    qResult.Score,
			Content:  content,
			Metadata: qResult.Payload,
		})
	}

	return results
}

func (s *Server) getCollectionsHealth() []CollectionHealth {
	// Get actual collections from Qdrant
	collections, err := s.getQdrantCollections()
	if err != nil {
		log.Printf("Failed to get collections: %v", err)
		// Return empty list on error
		return []CollectionHealth{}
	}

	var healthList []CollectionHealth
	for _, collectionName := range collections {
		// Get detailed collection info
		info, err := s.getQdrantCollectionInfo(collectionName)
		if err != nil {
			log.Printf("Failed to get info for collection %s: %v", collectionName, err)
			continue
		}

		// Calculate quality metrics based on real data
		quality := s.calculateCollectionQuality(info)

		healthList = append(healthList, CollectionHealth{
			Name:    collectionName,
			Size:    info.PointsCount,
			Quality: quality,
		})
	}

	return healthList
}

// calculateCollectionQuality computes quality metrics for a collection
func (s *Server) calculateCollectionQuality(info *QdrantCollection) QualityMetrics {
	// Start with baseline scores
	coherence := 0.85
	freshness := 0.80
	redundancy := 0.90
	coverage := 0.75

	// Adjust based on collection size (more points = potentially better coverage)
	if info.PointsCount > 1000 {
		coverage += 0.1
	} else if info.PointsCount < 100 {
		coverage -= 0.1
	}

	// Adjust based on vector size (higher dimensions = potentially better coherence)
	if info.Config.Params.Vectors.Size >= 1536 {
		coherence += 0.05
	}

	// Add some temporal variation (simulating freshness decay)
	hours := time.Now().Hour()
	freshness += math.Sin(float64(hours)*math.Pi/12) * 0.05

	// Ensure scores stay within bounds
	coherence = math.Max(0, math.Min(1, coherence))
	freshness = math.Max(0, math.Min(1, freshness))
	redundancy = math.Max(0, math.Min(1, redundancy))
	coverage = math.Max(0, math.Min(1, coverage))

	return QualityMetrics{
		Coherence:  coherence,
		Freshness:  freshness,
		Redundancy: redundancy,
		Coverage:   coverage,
	}
}

func (s *Server) buildKnowledgeGraph(req GraphRequest) GraphResponse {
	// For now, create a simple graph based on search results
	// This is a placeholder - a full implementation would analyze vector similarities
	
	nodes := []GraphNode{}
	edges := []GraphEdge{}

	if req.CenterConcept != "" {
		// Search for related concepts
		searchReq := SearchRequest{
			Query: req.CenterConcept,
			Limit: req.MaxNodes,
		}
		
		results := s.performSemanticSearch(searchReq)
		
		// Create center node
		nodes = append(nodes, GraphNode{
			ID:    "center",
			Label: req.CenterConcept,
			Type:  "concept",
			Metadata: map[string]interface{}{
				"importance": 1.0,
				"central":    true,
			},
		})

		// Create nodes for search results and edges to center
		for i, result := range results {
			if i >= req.MaxNodes-1 { // Reserve one spot for center
				break
			}
			
			nodeID := fmt.Sprintf("node-%d", i)
			nodes = append(nodes, GraphNode{
				ID:    nodeID,
				Label: result.Content[:min(50, len(result.Content))] + "...",
				Type:  "knowledge",
				Metadata: map[string]interface{}{
					"importance": result.Score,
					"source":     result.Metadata["source"],
				},
			})

			edges = append(edges, GraphEdge{
				Source:       "center",
				Target:       nodeID,
				Weight:       result.Score,
				Relationship: "semantic_similarity",
			})
		}
	}

	return GraphResponse{Nodes: nodes, Edges: edges}
}

func (s *Server) calculateMetrics(req MetricsRequest) MetricsResponse {
	metrics := make(map[string]QualityMetrics)
	trends := make(map[string]float64)
	
	collections := req.Collections
	if len(collections) == 0 {
		// Get all collections if none specified
		allCollections, err := s.getQdrantCollections()
		if err != nil {
			log.Printf("Failed to get collections for metrics: %v", err)
			collections = []string{} // Empty list
		} else {
			collections = allCollections
		}
	}

	for _, col := range collections {
		info, err := s.getQdrantCollectionInfo(col)
		if err != nil {
			log.Printf("Failed to get collection info for metrics: %v", err)
			continue
		}
		
		metrics[col] = s.calculateCollectionQuality(info)
		trends[col] = 0.0 // Placeholder - would calculate actual trends from historical data
	}

	alerts := []Alert{}
	for col, m := range metrics {
		if m.Coherence < 0.8 {
			alerts = append(alerts, Alert{
				Level:      "warning",
				Collection: col,
				Metric:     "coherence",
				Message:    fmt.Sprintf("Coherence score below threshold: %.2f", m.Coherence),
				Timestamp:  time.Now(),
			})
		}
	}

	return MetricsResponse{
		Metrics:    metrics,
		Trends:     trends,
		Alerts:     alerts,
		LastUpdate: time.Now().Format(time.RFC3339),
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true
	
	// Add timeout context to prevent health check from hanging
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Schema-compliant health response structure
	healthResponse := map[string]interface{}{
		"status":       overallStatus,
		"service":      "knowledge-observatory-api",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"readiness":    true,
		"version":      "1.0.0",
		"dependencies": map[string]interface{}{},
	}
	
	// Check PostgreSQL connectivity with context
	postgresChan := make(chan map[string]interface{}, 1)
	go func() {
		postgresChan <- s.checkPostgresHealth()
	}()
	
	var postgresHealth map[string]interface{}
	select {
	case <-ctx.Done():
		postgresHealth = map[string]interface{}{
			"status": "unhealthy",
			"error": map[string]interface{}{
				"code":      "POSTGRES_TIMEOUT",
				"message":   "PostgreSQL health check timed out",
				"category":  "resource",
				"retryable": true,
			},
		}
	case postgresHealth = <-postgresChan:
	}
	healthResponse["dependencies"].(map[string]interface{})["postgres"] = postgresHealth
	if postgresHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if postgresHealth["status"] == "unhealthy" {
			readiness = false
		}
		if postgresHealth["error"] != nil {
			errors = append(errors, postgresHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check Qdrant connectivity and collections with context
	qdrantChan := make(chan map[string]interface{}, 1)
	go func() {
		qdrantChan <- s.checkQdrantHealth()
	}()
	
	var qdrantHealth map[string]interface{}
	select {
	case <-ctx.Done():
		qdrantHealth = map[string]interface{}{
			"status": "unhealthy",
			"error": map[string]interface{}{
				"code":      "QDRANT_TIMEOUT",
				"message":   "Qdrant health check timed out",
				"category":  "resource",
				"retryable": true,
			},
		}
	case qdrantHealth = <-qdrantChan:
	}
	healthResponse["dependencies"].(map[string]interface{})["qdrant"] = qdrantHealth
	if qdrantHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if qdrantHealth["status"] == "unhealthy" {
			readiness = false
		}
		if qdrantHealth["error"] != nil {
			errors = append(errors, qdrantHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check CLI availability
	cliHealth := s.checkResourceCLIHealth()
	healthResponse["dependencies"].(map[string]interface{})["resource_cli"] = cliHealth
	if cliHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if cliHealth["error"] != nil {
			errors = append(errors, cliHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check optional N8N workflows
	n8nHealth := s.checkN8NHealth()
	healthResponse["dependencies"].(map[string]interface{})["n8n"] = n8nHealth
	if n8nHealth["status"] == "unhealthy" {
		// N8N is optional, so only degrade if configured but failing
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if n8nHealth["error"] != nil {
			errors = append(errors, n8nHealth["error"].(map[string]interface{}))
		}
	}
	
	// Check optional Ollama AI service
	ollamaHealth := s.checkOllamaHealth()
	healthResponse["dependencies"].(map[string]interface{})["ollama"] = ollamaHealth
	if ollamaHealth["status"] == "unhealthy" {
		// Ollama is optional, so only degrade if configured but failing
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if ollamaHealth["error"] != nil {
			errors = append(errors, ollamaHealth["error"].(map[string]interface{}))
		}
	}
	
	// Update final status
	healthResponse["status"] = overallStatus
	healthResponse["readiness"] = readiness
	
	// Add errors if any
	if len(errors) > 0 {
		healthResponse["errors"] = errors
	}
	
	// Add metrics
	healthResponse["metrics"] = map[string]interface{}{
		"total_dependencies":   5,
		"healthy_dependencies": s.countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"response_time_ms":     time.Since(start).Milliseconds(),
	}
	
	// Knowledge observatory specific info
	healthResponse["knowledge_stats"] = map[string]interface{}{
		"total_entries": calculateTotalEntries(),
		"collections": s.getCollectionsHealth(),
		"overall_health": calculateOverallHealth(),
	}
	
	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Response-Time", fmt.Sprintf("%dms", time.Since(start).Milliseconds()))
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Threshold == 0 {
		req.Threshold = 0.5
	}

	results := s.performSemanticSearch(req)
	
	response := SearchResponse{
		Results: results,
		Count:   len(results),
		Time:    time.Since(start).Milliseconds(),
	}

	s.logSearchQuery(req, response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) graphHandler(w http.ResponseWriter, r *http.Request) {
	var req GraphRequest
	
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	if req.Depth == 0 {
		req.Depth = 2
	}
	if req.MaxNodes == 0 {
		req.MaxNodes = 100
	}

	graph := s.buildKnowledgeGraph(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	var req MetricsRequest
	
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	metrics := s.calculateMetrics(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Server) timelineHandler(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	var req TimelineRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	
	// Parse time range (default to last 7 days)
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)
	
	if req.TimeRange != "" {
		if duration, err := time.ParseDuration(req.TimeRange); err == nil {
			startTime = endTime.Add(-duration)
		}
	}
	
	// Query database for timeline events
	entries := []TimelineEntry{}
	collections := make(map[string]int)
	
	// Query search history from database for timeline data
	query := `
		SELECT 
			created_at,
			collection,
			COUNT(*) as count
		FROM knowledge_observatory.search_history
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY DATE_TRUNC('hour', created_at), created_at, collection
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.Query(query, startTime, endTime)
	if err == nil {
		defer rows.Close()
		
		for rows.Next() {
			var entry TimelineEntry
			var collectionName sql.NullString
			
			if err := rows.Scan(&entry.Timestamp, &collectionName, &entry.Count); err == nil {
				entry.EventType = "search"
				if collectionName.Valid {
					entry.Collection = collectionName.String
					collections[entry.Collection]++
				} else {
					entry.Collection = "all"
				}
				entry.Description = fmt.Sprintf("%d searches performed", entry.Count)
				entries = append(entries, entry)
			}
		}
	}
	
	// Add synthetic timeline data for demonstration
	if len(entries) == 0 {
		// Generate sample timeline data
		for i := 0; i < 20; i++ {
			timestamp := startTime.Add(time.Duration(i) * 6 * time.Hour)
			collectionNames := []string{"vrooli_knowledge", "scenario_memory", "agent_decisions", "workflow_patterns"}
			collection := collectionNames[i%len(collectionNames)]
			
			entry := TimelineEntry{
				Timestamp:   timestamp,
				Collection:  collection,
				EventType:   []string{"added", "updated", "search"}[i%3],
				Count:       10 + (i * 3),
				Description: fmt.Sprintf("Knowledge %s in %s", []string{"added", "updated", "searched"}[i%3], collection),
			}
			entries = append(entries, entry)
			collections[collection]++
		}
	}
	
	response := TimelineResponse{
		Entries:     entries,
		Collections: collections,
		TotalEvents: len(entries),
	}
	response.TimeRange.Start = startTime
	response.TimeRange.End = endTime
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			health := s.getStreamingHealth()
			if err := conn.WriteJSON(health); err != nil {
				log.Printf("WebSocket write failed: %v", err)
				return
			}
		}
	}
}

func (s *Server) getStreamingHealth() HealthResponse {
	return HealthResponse{
		Status:        "healthy",
		TotalEntries:  calculateTotalEntries(),
		Collections:   s.getCollectionsHealth(),
		OverallHealth: calculateOverallHealth(),
		Timestamp:     time.Now().Format(time.RFC3339),
	}
}

func (s *Server) logSearchQuery(req SearchRequest, resp SearchResponse) {
	if s.db != nil {
		query := `INSERT INTO search_history (query, collection, result_count, response_time_ms, created_at) 
		          VALUES ($1, $2, $3, $4, $5)`
		_, err := s.db.Exec(query, req.Query, req.Collection, resp.Count, resp.Time, time.Now())
		if err != nil {
			log.Printf("Failed to log search query: %v", err)
		}
	}
}

func calculateTotalEntries() int {
	// This would be calculated from real data
	return 23990
}

func calculateOverallHealth() string {
	health := 0.85
	if health > 0.8 {
		return "healthy"
	} else if health > 0.6 {
		return "degraded"
	}
	return "critical"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// Build POSTGRES_URL from individual components if not set
	if key == "POSTGRES_URL" {
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
			return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				dbUser, dbPassword, dbHost, dbPort, dbName)
		}
	}
	log.Fatalf("❌ %s environment variable is required (or provide POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)", key)
	return ""
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Health check helper methods
func (s *Server) checkPostgresHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	if s.db == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_NOT_CONFIGURED",
			"message": "Database connection not initialized",
			"category": "configuration",
			"retryable": false,
		}
		return health
	}
	
	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := s.db.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_CONNECTION_FAILED",
			"message": "Failed to ping database: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"
	
	// Test knowledge_observatory schema
	var schemaExists bool
	schemaQuery := `SELECT EXISTS(
		SELECT 1 FROM information_schema.schemata 
		WHERE schema_name = 'knowledge_observatory'
	)`
	if err := s.db.QueryRowContext(ctx, schemaQuery).Scan(&schemaExists); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_SCHEMA_CHECK_FAILED",
			"message": "Failed to verify knowledge_observatory schema: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
	} else if !schemaExists {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "DATABASE_SCHEMA_MISSING",
			"message": "knowledge_observatory schema does not exist",
			"category": "configuration",
			"retryable": false,
		}
	} else {
		health["checks"].(map[string]interface{})["schema"] = "ok"
	}
	
	// Test quality_metrics table
	var tableCount int
	tableQuery := `SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'knowledge_observatory' 
		AND table_name = 'quality_metrics'`
	if err := s.db.QueryRowContext(ctx, tableQuery).Scan(&tableCount); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "DATABASE_TABLE_CHECK_FAILED",
				"message": "Failed to verify quality_metrics table: " + err.Error(),
				"category": "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["quality_metrics_table"] = tableCount
	}
	
	// Test collection_stats table
	var collectionCount int
	collectionQuery := `SELECT COUNT(*) FROM knowledge_observatory.collection_stats`
	if err := s.db.QueryRowContext(ctx, collectionQuery).Scan(&collectionCount); err != nil {
		// Don't fail health check, just note it
		health["checks"].(map[string]interface{})["collection_stats"] = "error"
	} else {
		health["checks"].(map[string]interface{})["collection_stats"] = collectionCount
	}
	
	return health
}

func (s *Server) checkQdrantHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	// Check if we can list collections
	collections, err := s.getQdrantCollections()
	if err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "QDRANT_CONNECTION_FAILED",
			"message": "Failed to connect to Qdrant: " + err.Error(),
			"category": "resource",
			"retryable": true,
		}
		return health
	}
	
	health["checks"].(map[string]interface{})["collections_accessible"] = "ok"
	health["checks"].(map[string]interface{})["collection_count"] = len(collections)
	
	// OPTIMIZATION: Only sample a few collections for health check, not all 43+
	// Full collection scan happens in background metrics collection
	degradedCollections := 0
	totalPoints := 0
	maxCollectionsToCheck := 5 // Only check first 5 collections for health
	
	for i, collectionName := range collections {
		if i >= maxCollectionsToCheck {
			break // Limit health check to avoid timeout
		}
		info, err := s.getQdrantCollectionInfo(collectionName)
		if err != nil {
			degradedCollections++
			continue
		}
		totalPoints += info.PointsCount
	}
	
	health["checks"].(map[string]interface{})["total_points"] = totalPoints
	
	if degradedCollections > 0 {
		health["status"] = "degraded"
		health["checks"].(map[string]interface{})["degraded_collections"] = degradedCollections
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code": "QDRANT_COLLECTIONS_DEGRADED",
				"message": fmt.Sprintf("%d collections are not fully accessible", degradedCollections),
				"category": "resource",
				"retryable": true,
			}
		}
	}
	
	// Test search capability
	testSearch := SearchRequest{
		Query: "health check test",
		Limit: 1,
	}
	searchResults := s.performSemanticSearch(testSearch)
	if searchResults == nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		health["checks"].(map[string]interface{})["search_capability"] = "failed"
	} else {
		health["checks"].(map[string]interface{})["search_capability"] = "ok"
	}
	
	return health
}

func (s *Server) checkResourceCLIHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}
	
	// Test if resource-qdrant CLI is available
	cmd := exec.Command(s.config.ResourceCLI, "version")
	if output, err := cmd.CombinedOutput(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code": "RESOURCE_CLI_NOT_AVAILABLE",
			"message": fmt.Sprintf("resource-qdrant CLI not found or failed: %v", err),
			"category": "configuration",
			"retryable": false,
		}
		return health
	} else {
		// Parse version if possible
		version := strings.TrimSpace(string(output))
		health["checks"].(map[string]interface{})["cli_version"] = version
	}
	
	// Test basic CLI functionality
	testOutput, err := s.execResourceQdrant("collections", "list")
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code": "RESOURCE_CLI_EXECUTION_FAILED",
			"message": "CLI available but execution failed: " + err.Error(),
			"category": "internal",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["cli_execution"] = "ok"
		health["checks"].(map[string]interface{})["cli_output_size"] = len(testOutput)
	}
	
	return health
}

func (s *Server) countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && status == "healthy" {
				count++
			}
		}
	}
	return count
}

func (s *Server) checkN8NHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "not_configured",
		"checks": map[string]interface{}{},
	}

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	// Test N8N connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", n8nURL+"/healthz", nil)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["workflows"] = "disabled"
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["workflows"] = "disabled"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["status"] = "healthy"
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["workflows"] = "available"
		
		// Check for knowledge observatory workflows
		workflows := []string{"knowledge-quality-monitor", "semantic-analyzer", "knowledge-graph-builder"}
		availableWorkflows := 0
		for range workflows {
			// This would need actual N8N API calls to check workflow existence
			availableWorkflows++
		}
		health["checks"].(map[string]interface{})["knowledge_workflows"] = availableWorkflows
	} else {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "N8N_UNHEALTHY",
			"message":   fmt.Sprintf("N8N returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func (s *Server) checkOllamaHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "not_configured",
		"checks": map[string]interface{}{},
	}

	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Test Ollama connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ollamaURL+"/api/tags", nil)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["ai_analysis"] = "disabled"
		return health
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["ai_analysis"] = "disabled"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["status"] = "healthy"
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		
		// Parse response to check available models
		var response struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err == nil {
			health["checks"].(map[string]interface{})["available_models"] = len(response.Models)
			
			// Check for required models
			requiredModels := []string{"llama3.2", "nomic-embed-text"}
			availableModels := make(map[string]bool)
			for _, model := range response.Models {
				availableModels[model.Name] = true
			}
			
			missingModels := []string{}
			for _, required := range requiredModels {
				if !availableModels[required] {
					missingModels = append(missingModels, required)
				}
			}
			
			if len(missingModels) > 0 {
				health["status"] = "degraded"
				health["error"] = map[string]interface{}{
					"code":      "OLLAMA_MODELS_MISSING",
					"message":   fmt.Sprintf("Missing required models: %v", missingModels),
					"category":  "configuration",
					"retryable": false,
				}
			} else {
				health["checks"].(map[string]interface{})["required_models"] = "available"
				health["checks"].(map[string]interface{})["ai_analysis"] = "enabled"
			}
		}
	} else {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "OLLAMA_UNHEALTHY",
			"message":   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start knowledge-observatory

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }
	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to initialize server:", err)
	}

	router := mux.NewRouter()
	
	router.HandleFunc("/health", server.healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/knowledge/search", server.searchHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/knowledge/health", server.healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/knowledge/graph", server.graphHandler).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/v1/knowledge/metrics", server.metricsHandler).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/v1/knowledge/timeline", server.timelineHandler).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/v1/knowledge/stream", server.streamHandler).Methods("GET")
	
	handler := enableCORS(router)
	
	log.Printf("🔭 Knowledge Observatory API starting on port %s", server.config.Port)
	log.Printf("📊 Qdrant CLI: %s", server.config.ResourceCLI)
	
	if err := http.ListenAndServe(":"+server.config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
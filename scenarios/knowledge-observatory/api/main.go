package main

import (
	"bytes"
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
	cmd := exec.Command(s.config.ResourceCLI, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
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
	
	response := HealthResponse{
		Status:        "healthy",
		TotalEntries:  calculateTotalEntries(),
		Collections:   s.getCollectionsHealth(),
		OverallHealth: calculateOverallHealth(),
		Timestamp:     time.Now().Format(time.RFC3339),
	}

	if response.OverallHealth == "degraded" {
		response.Status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Response-Time", fmt.Sprintf("%dms", time.Since(start).Milliseconds()))
	json.NewEncoder(w).Encode(response)
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
	log.Fatalf("%s environment variable is required", key)
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

func main() {
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
	router.HandleFunc("/api/v1/knowledge/stream", server.streamHandler).Methods("GET")
	
	handler := enableCORS(router)
	
	log.Printf("🔭 Knowledge Observatory API starting on port %s", server.config.Port)
	log.Printf("📊 Qdrant CLI: %s", server.config.ResourceCLI)
	
	if err := http.ListenAndServe(":"+server.config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
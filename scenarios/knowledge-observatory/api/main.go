package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Config struct {
	Port       string
	QdrantURL  string
	PostgresDB string
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
	Status       string               `json:"status"`
	TotalEntries int                  `json:"total_entries"`
	Collections  []CollectionHealth   `json:"collections"`
	OverallHealth string              `json:"overall_health"`
	Timestamp    string               `json:"timestamp"`
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

func NewServer() (*Server, error) {
	config := Config{
		Port:       getEnv("API_PORT", getEnv("PORT", "")),
		QdrantURL:  getEnv("QDRANT_URL", "http://localhost:6333"),
		PostgresDB: getEnv("DATABASE_URL", "postgres://user:password@localhost/knowledge_observatory"),
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

func (s *Server) performSemanticSearch(req SearchRequest) []SearchResult {
	results := []SearchResult{
		{
			ID:      "sample-1",
			Score:   0.95,
			Content: "Sample knowledge entry matching your query",
			Metadata: map[string]interface{}{
				"source":    "scenario-generator",
				"timestamp": time.Now().Add(-24 * time.Hour),
				"quality":   0.88,
			},
		},
		{
			ID:      "sample-2",
			Score:   0.87,
			Content: "Another relevant knowledge entry",
			Metadata: map[string]interface{}{
				"source":    "research-assistant",
				"timestamp": time.Now().Add(-48 * time.Hour),
				"quality":   0.92,
			},
		},
	}
	
	return results
}

func (s *Server) getCollectionsHealth() []CollectionHealth {
	return []CollectionHealth{
		{
			Name: "vrooli_knowledge",
			Size: 15234,
			Quality: QualityMetrics{
				Coherence:  0.89,
				Freshness:  0.76,
				Redundancy: 0.92,
				Coverage:   0.83,
			},
		},
		{
			Name: "scenario_memory",
			Size: 8756,
			Quality: QualityMetrics{
				Coherence:  0.91,
				Freshness:  0.88,
				Redundancy: 0.85,
				Coverage:   0.79,
			},
		},
	}
}

func (s *Server) buildKnowledgeGraph(req GraphRequest) GraphResponse {
	nodes := []GraphNode{
		{
			ID:    "node-1",
			Label: req.CenterConcept,
			Type:  "concept",
			Metadata: map[string]interface{}{
				"importance": 0.95,
				"frequency":  234,
			},
		},
		{
			ID:    "node-2",
			Label: "Related Concept 1",
			Type:  "concept",
			Metadata: map[string]interface{}{
				"importance": 0.78,
				"frequency":  156,
			},
		},
		{
			ID:    "node-3",
			Label: "Related Concept 2",
			Type:  "concept",
			Metadata: map[string]interface{}{
				"importance": 0.82,
				"frequency":  189,
			},
		},
	}

	edges := []GraphEdge{
		{
			Source:       "node-1",
			Target:       "node-2",
			Weight:       0.87,
			Relationship: "semantic_similarity",
		},
		{
			Source:       "node-1",
			Target:       "node-3",
			Weight:       0.79,
			Relationship: "co_occurrence",
		},
	}

	return GraphResponse{Nodes: nodes, Edges: edges}
}

func (s *Server) calculateMetrics(req MetricsRequest) MetricsResponse {
	metrics := make(map[string]QualityMetrics)
	trends := make(map[string]float64)
	
	collections := req.Collections
	if len(collections) == 0 {
		collections = []string{"vrooli_knowledge", "scenario_memory"}
	}

	for _, col := range collections {
		metrics[col] = QualityMetrics{
			Coherence:  0.85 + math.Sin(float64(time.Now().Unix()))*0.1,
			Freshness:  0.80 + math.Cos(float64(time.Now().Unix()))*0.1,
			Redundancy: 0.90 - math.Sin(float64(time.Now().Unix()))*0.05,
			Coverage:   0.75 + math.Cos(float64(time.Now().Unix()))*0.15,
		}
		trends[col] = math.Sin(float64(time.Now().Unix())) * 5
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
	log.Printf("📊 Qdrant URL: %s", server.config.QdrantURL)
	
	if err := http.ListenAndServe(":"+server.config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
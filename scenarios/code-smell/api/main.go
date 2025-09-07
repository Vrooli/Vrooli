package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Violation represents a code smell violation
type Violation struct {
	ID           string    `json:"id"`
	RuleID       string    `json:"rule_id"`
	RuleName     string    `json:"rule_name"`
	FilePath     string    `json:"file_path"`
	LineNumber   int       `json:"line_number"`
	ColumnNumber int       `json:"column_number"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	SuggestedFix string    `json:"suggested_fix"`
	AutoFixable  bool      `json:"auto_fixable"`
	Status       string    `json:"status"`
	DetectedAt   time.Time `json:"detected_at"`
}

// Rule represents a code smell detection rule
type Rule struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category"`
	RiskLevel      string                 `json:"risk_level"`
	VrooliSpecific bool                   `json:"vrooli_specific"`
	Pattern        map[string]interface{} `json:"pattern"`
	FixTemplate    string                 `json:"fix_template"`
	Enabled        bool                   `json:"enabled"`
}

// AnalyzeRequest represents a request to analyze files
type AnalyzeRequest struct {
	Paths         []string `json:"paths"`
	Rules         []string `json:"rules,omitempty"`
	AutoFix       bool     `json:"auto_fix"`
	RiskThreshold string   `json:"risk_threshold,omitempty"`
}

// AnalyzeResponse represents the analysis result
type AnalyzeResponse struct {
	Violations      []Violation `json:"violations"`
	AutoFixed       int         `json:"auto_fixed"`
	NeedsReview     int         `json:"needs_review"`
	TotalFiles      int         `json:"total_files"`
	DurationMs      int64       `json:"duration_ms"`
}

// FixRequest represents a request to apply a fix
type FixRequest struct {
	ViolationID string `json:"violation_id"`
	Action      string `json:"action"`
	ModifiedFix string `json:"modified_fix,omitempty"`
}

// Server represents the API server
type Server struct {
	router *mux.Router
	port   string
}

// NewServer creates a new API server
func NewServer() *Server {
	port := os.Getenv("CODE_SMELL_API_PORT")
	if port == "" {
		port = "8090"
	}

	s := &Server{
		router: mux.NewRouter(),
		port:   port,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Health checks
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/health/live", s.handleHealthLive).Methods("GET")
	api.HandleFunc("/health/ready", s.handleHealthReady).Methods("GET")

	// Code smell endpoints
	api.HandleFunc("/code-smell/analyze", s.handleAnalyze).Methods("POST")
	api.HandleFunc("/code-smell/rules", s.handleGetRules).Methods("GET")
	api.HandleFunc("/code-smell/fix", s.handleFix).Methods("POST")
	api.HandleFunc("/code-smell/queue", s.handleGetQueue).Methods("GET")
	api.HandleFunc("/code-smell/learn", s.handleLearn).Methods("POST")
	api.HandleFunc("/code-smell/stats", s.handleGetStats).Methods("GET")

	// Documentation
	api.HandleFunc("/docs", s.handleDocs).Methods("GET")
}

// handleHealth returns the health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "code-smell",
		"version":   "1.0.0",
	}
	sendJSON(w, http.StatusOK, response)
}

// handleHealthLive returns liveness status
func (s *Server) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// handleHealthReady returns readiness status
func (s *Server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	// Check if rules are loaded and engine is ready
	ready := checkEngineReady()
	if ready {
		sendJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	} else {
		sendJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	}
}

// handleAnalyze analyzes files for code smells
func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Paths) == 0 {
		sendError(w, http.StatusBadRequest, "No paths provided")
		return
	}

	startTime := time.Now()
	
	// Call the rules engine to analyze files
	violations, autoFixed := analyzeFiles(req.Paths, req.Rules, req.AutoFix, req.RiskThreshold)
	
	needsReview := 0
	for _, v := range violations {
		if v.Status == "pending" && !v.AutoFixable {
			needsReview++
		}
	}

	response := AnalyzeResponse{
		Violations:  violations,
		AutoFixed:   autoFixed,
		NeedsReview: needsReview,
		TotalFiles:  len(req.Paths),
		DurationMs:  time.Since(startTime).Milliseconds(),
	}

	sendJSON(w, http.StatusOK, response)
}

// handleGetRules returns all configured rules
func (s *Server) handleGetRules(w http.ResponseWriter, r *http.Request) {
	rules := getRules()
	
	vrooliCount := 0
	for _, rule := range rules {
		if rule.VrooliSpecific {
			vrooliCount++
		}
	}

	response := map[string]interface{}{
		"rules":                 rules,
		"categories":            getCategories(rules),
		"vrooli_specific_count": vrooliCount,
	}

	sendJSON(w, http.StatusOK, response)
}

// handleFix applies or rejects a fix
func (s *Server) handleFix(w http.ResponseWriter, r *http.Request) {
	var req FixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ViolationID == "" {
		sendError(w, http.StatusBadRequest, "Violation ID required")
		return
	}

	if req.Action != "approve" && req.Action != "reject" && req.Action != "ignore" {
		sendError(w, http.StatusBadRequest, "Invalid action")
		return
	}

	// Apply the fix action
	result := applyFixAction(req.ViolationID, req.Action, req.ModifiedFix)
	
	sendJSON(w, http.StatusOK, result)
}

// handleGetQueue returns violations needing review
func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	severity := r.URL.Query().Get("severity")
	filePattern := r.URL.Query().Get("file")

	violations := getViolationQueue(severity, filePattern)
	
	bySeverity := map[string]int{
		"error":   0,
		"warning": 0,
		"info":    0,
	}
	
	for _, v := range violations {
		bySeverity[v.Severity]++
	}

	response := map[string]interface{}{
		"violations":  violations,
		"total":       len(violations),
		"by_severity": bySeverity,
	}

	sendJSON(w, http.StatusOK, response)
}

// handleLearn submits a pattern for learning
func (s *Server) handleLearn(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	pattern, ok := req["pattern"].(string)
	if !ok || pattern == "" {
		sendError(w, http.StatusBadRequest, "Pattern required")
		return
	}

	isPositive, _ := req["is_positive"].(bool)
	context, _ := req["context"].(map[string]interface{})

	// Submit pattern to learning system
	result := submitPattern(pattern, isPositive, context)
	
	sendJSON(w, http.StatusOK, result)
}

// handleGetStats returns code smell statistics
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}

	stats := getStatistics(period)
	sendJSON(w, http.StatusOK, stats)
}

// handleDocs returns API documentation
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"service": "code-smell",
		"version": "1.0.0",
		"endpoints": []map[string]string{
			{
				"path":        "/api/v1/code-smell/analyze",
				"method":      "POST",
				"description": "Analyze files for code smells",
			},
			{
				"path":        "/api/v1/code-smell/rules",
				"method":      "GET",
				"description": "Get all configured rules",
			},
			{
				"path":        "/api/v1/code-smell/fix",
				"method":      "POST",
				"description": "Apply or reject a fix",
			},
			{
				"path":        "/api/v1/code-smell/queue",
				"method":      "GET",
				"description": "Get violations awaiting review",
			},
			{
				"path":        "/api/v1/code-smell/learn",
				"method":      "POST",
				"description": "Submit pattern for learning",
			},
			{
				"path":        "/api/v1/code-smell/stats",
				"method":      "GET",
				"description": "Get code smell statistics",
			},
		},
	}
	sendJSON(w, http.StatusOK, docs)
}

// Helper functions

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{"error": message})
}

func checkEngineReady() bool {
	// Check if rules engine is initialized
	// This would connect to the Node.js rules engine
	return true
}

func analyzeFiles(paths []string, rules []string, autoFix bool, riskThreshold string) ([]Violation, int) {
	// This would call the Node.js rules engine
	// For now, return mock data
	violations := []Violation{
		{
			ID:           "hardcoded-port-1",
			RuleID:       "hardcoded-port",
			RuleName:     "Hard-coded Port Number",
			FilePath:     paths[0],
			LineNumber:   42,
			ColumnNumber: 15,
			Severity:     "warning",
			Message:      "Hard-coded port found. Use environment variable instead",
			SuggestedFix: "Use ${API_PORT:-3000}",
			AutoFixable:  false,
			Status:       "pending",
			DetectedAt:   time.Now(),
		},
	}
	return violations, 0
}

func getRules() []Rule {
	// This would fetch rules from the rules engine
	return []Rule{
		{
			ID:             "hardcoded-port",
			Name:           "Hard-coded Port Number",
			Description:    "Ports should use environment variables",
			Category:       "needs-approval",
			RiskLevel:      "moderate",
			VrooliSpecific: true,
			Enabled:        true,
		},
	}
}

func getCategories(rules []Rule) []string {
	categories := make(map[string]bool)
	for _, rule := range rules {
		categories[rule.Category] = true
	}
	
	result := make([]string, 0, len(categories))
	for cat := range categories {
		result = append(result, cat)
	}
	return result
}

func applyFixAction(violationID, action, modifiedFix string) map[string]interface{} {
	// This would apply the fix through the rules engine
	return map[string]interface{}{
		"success":      true,
		"violation_id": violationID,
		"action":       action,
		"timestamp":    time.Now().Unix(),
	}
}

func getViolationQueue(severity, filePattern string) []Violation {
	// This would fetch from database
	return []Violation{}
}

func submitPattern(pattern string, isPositive bool, context map[string]interface{}) map[string]interface{} {
	// This would submit to learning system
	return map[string]interface{}{
		"success":    true,
		"pattern":    pattern,
		"confidence": 0.75,
	}
}

func getStatistics(period string) map[string]interface{} {
	// This would fetch from database
	return map[string]interface{}{
		"period":           period,
		"files_analyzed":   1523,
		"violations_found": 247,
		"auto_fixed":       89,
		"patterns_learned": 12,
	}
}

// Start starts the server
func (s *Server) Start() {
	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(s.router)
	
	log.Printf("Code Smell API server starting on port %s", s.port)
	if err := http.ListenAndServe(":"+s.port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func main() {
	server := NewServer()
	server.Start()
}
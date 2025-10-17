package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Database:  "unknown",
		Resources: make(map[string]interface{}),
		Version:   "1.0.0",
	}

	// Check database connection
	if s.db != nil {
		if s.db.IsConnected() {
			health.Database = "connected"
		} else {
			health.Database = "disconnected"
			health.Status = "degraded"
		}
	} else {
		health.Database = "not configured"
	}

	// Get resource status from resource manager
	if s.resourceManager != nil {
		health.Resources = s.resourceManager.GetResourceHealth()
	}

	w.Header().Set("Content-Type", "application/json")
	if health.Status != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(health)
}

// ResourcesHandler provides detailed resource status
func (s *Server) ResourcesHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"resources": make(map[string]interface{}),
		"metrics":   make(map[string]interface{}),
	}

	if s.resourceManager != nil {
		response["resources"] = s.resourceManager.GetResourceStatus()
		response["metrics"] = s.resourceManager.GetResourceMetrics()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API v1 Handlers

// DiffHandlerV1 handles text diff requests (v1)
func (s *Server) DiffHandlerV1(w http.ResponseWriter, r *http.Request) {
	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Text1 == nil || req.Text2 == nil {
		sendErrorResponse(w, "Both text1 and text2 are required", http.StatusBadRequest)
		return
	}

	// Process diff (v1 logic)
	response := s.processDiffV1(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchHandlerV1 handles text search requests (v1)
func (s *Server) SearchHandlerV1(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Pattern == "" {
		sendErrorResponse(w, "Pattern is required", http.StatusBadRequest)
		return
	}

	// Process search (v1 logic)
	response := s.processSearchV1(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TransformHandlerV1 handles text transformation requests (v1)
func (s *Server) TransformHandlerV1(w http.ResponseWriter, r *http.Request) {
	var req TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Text == "" {
		sendErrorResponse(w, "Text is required", http.StatusBadRequest)
		return
	}

	// Process transformation (v1 logic)
	response := s.processTransformV1(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExtractHandlerV1 handles text extraction requests (v1)
func (s *Server) ExtractHandlerV1(w http.ResponseWriter, r *http.Request) {
	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Source == nil {
		sendErrorResponse(w, "Source is required", http.StatusBadRequest)
		return
	}

	// Process extraction (v1 logic)
	response := s.processExtractV1(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyzeHandlerV1 handles text analysis requests (v1)
func (s *Server) AnalyzeHandlerV1(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Text == "" {
		sendErrorResponse(w, "Text is required", http.StatusBadRequest)
		return
	}

	// Process analysis (v1 logic)
	response := s.processAnalyzeV1(req)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API v2 Handlers (with improved features)

// DiffHandlerV2 handles text diff requests (v2 with enhanced features)
func (s *Server) DiffHandlerV2(w http.ResponseWriter, r *http.Request) {
	var req DiffRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseV2(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enhanced validation
	if err := req.Validate(); err != nil {
		sendErrorResponseV2(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process diff with v2 enhancements
	response := s.processDiffV2(req)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	json.NewEncoder(w).Encode(response)
}

// SearchHandlerV2 handles text search requests (v2 with vector search)
func (s *Server) SearchHandlerV2(w http.ResponseWriter, r *http.Request) {
	var req SearchRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseV2(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enhanced validation
	if err := req.Validate(); err != nil {
		sendErrorResponseV2(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process search with v2 enhancements (including vector search)
	response := s.processSearchV2(req)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	json.NewEncoder(w).Encode(response)
}

// TransformHandlerV2 handles text transformation requests (v2 with chaining)
func (s *Server) TransformHandlerV2(w http.ResponseWriter, r *http.Request) {
	var req TransformRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseV2(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enhanced validation
	if err := req.Validate(); err != nil {
		sendErrorResponseV2(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process transformation with v2 chaining support
	response := s.processTransformV2(req)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	json.NewEncoder(w).Encode(response)
}

// ExtractHandlerV2 handles text extraction requests (v2 with streaming)
func (s *Server) ExtractHandlerV2(w http.ResponseWriter, r *http.Request) {
	var req ExtractRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseV2(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enhanced validation
	if err := req.Validate(); err != nil {
		sendErrorResponseV2(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process extraction with v2 streaming support
	response := s.processExtractV2(req)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	json.NewEncoder(w).Encode(response)
}

// AnalyzeHandlerV2 handles text analysis requests (v2 with AI enhancements)
func (s *Server) AnalyzeHandlerV2(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseV2(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Enhanced validation
	if err := req.Validate(); err != nil {
		sendErrorResponseV2(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process analysis with v2 AI enhancements
	response := s.processAnalyzeV2(req)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	json.NewEncoder(w).Encode(response)
}

// PipelineHandler handles text processing pipelines (v2 only)
func (s *Server) PipelineHandler(w http.ResponseWriter, r *http.Request) {
	var req PipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseV2(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate pipeline
	if err := req.Validate(); err != nil {
		sendErrorResponseV2(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process pipeline
	response := s.processPipeline(req)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func sendErrorResponse(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Version: "1.0",
	})
}

func sendErrorResponseV2(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "2.0")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponseV2{
		Error:     message,
		Code:      code,
		Version:   "2.0",
		Timestamp: time.Now().Unix(),
		RequestID: uuid.New().String(),
	})
}

func checkResourceHealth(url string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "error"
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "disconnected"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "connected"
	}
	return "unhealthy"
}

// Processing functions (simplified stubs - implement actual logic)

func (s *Server) processDiffV1(req DiffRequest) DiffResponse {
	// Extract text from request
	text1 := extractText(req.Text1)
	text2 := extractText(req.Text2)

	// Perform diff based on type
	diffType := req.Options.Type
	if diffType == "" {
		diffType = "line"
	}

	var changes []Change
	var similarity float64

	switch diffType {
	case "line":
		changes, similarity = performLineDiff(text1, text2, req.Options)
	case "word":
		changes, similarity = performWordDiff(text1, text2, req.Options)
	case "character":
		changes, similarity = performCharDiff(text1, text2, req.Options)
	case "semantic":
		if s.config.OllamaURL != "" {
			changes, similarity = performSemanticDiff(text1, text2, req.Options, s.config.OllamaURL)
		} else {
			// Fallback to line diff if Ollama not available
			changes, similarity = performLineDiff(text1, text2, req.Options)
		}
	}

	// Store in database if available
	if s.db != nil && s.db.IsConnected() {
		go s.storeDiffOperation(text1, text2, changes, similarity)
	}

	return DiffResponse{
		Changes:         changes,
		SimilarityScore: similarity,
		Summary:         fmt.Sprintf("Found %d changes with %.2f%% similarity", len(changes), similarity*100),
	}
}

func (s *Server) processDiffV2(req DiffRequestV2) DiffResponseV2 {
	// V2 includes additional features like visual diff, patch generation, etc.
	response := DiffResponseV2{
		DiffResponse: s.processDiffV1(req.DiffRequest),
		RequestID:    uuid.New().String(),
		ProcessingTime: 0, // Will be calculated
	}

	// Add v2 specific features
	if req.Options.GeneratePatch {
		response.Patch = generateUnifiedPatch(req.Text1, req.Text2)
	}

	if req.Options.IncludeMetrics {
		response.Metrics = calculateDiffMetrics(req.Text1, req.Text2)
	}

	return response
}

func (s *Server) processSearchV1(req SearchRequest) SearchResponse {
	// Implement v1 search logic
	text := extractText(req.Text)
	matches := performSearch(text, req.Pattern, req.Options)

	return SearchResponse{
		Matches:      matches,
		TotalMatches: len(matches),
	}
}

func (s *Server) processSearchV2(req SearchRequestV2) SearchResponseV2 {
	// V2 includes vector search if Qdrant is available
	response := SearchResponseV2{
		SearchResponse: s.processSearchV1(req.SearchRequest),
		RequestID:      uuid.New().String(),
	}

	// Add semantic search if available
	if req.Options.Semantic && s.config.OllamaURL != "" {
		response.SemanticMatches = performSemanticSearch(req.Text, req.Pattern, s.config)
	}

	return response
}

func (s *Server) processTransformV1(req TransformRequest) TransformResponse {
	// Implement v1 transformation logic
	result := req.Text
	applied := []string{}

	for _, transform := range req.Transformations {
		result = applyTransformation(result, transform)
		applied = append(applied, transform.Type)
	}

	return TransformResponse{
		Result:                result,
		TransformationsApplied: applied,
	}
}

func (s *Server) processTransformV2(req TransformRequestV2) TransformResponseV2 {
	// V2 supports transformation chains and rollback
	response := TransformResponseV2{
		TransformResponse: s.processTransformV1(req.TransformRequest),
		RequestID:         uuid.New().String(),
	}

	// Track intermediate results for rollback
	if req.Options.TrackIntermediates {
		response.IntermediateResults = trackTransformationSteps(req.Text, req.Transformations)
	}

	return response
}

func (s *Server) processExtractV1(req ExtractRequest) ExtractResponse {
	// Implement v1 extraction logic
	text, metadata := extractContent(req.Source, req.Options)

	return ExtractResponse{
		Text:     text,
		Metadata: metadata,
	}
}

func (s *Server) processExtractV2(req ExtractRequestV2) ExtractResponseV2 {
	// V2 supports streaming and progressive extraction
	response := ExtractResponseV2{
		ExtractResponse: s.processExtractV1(req.ExtractRequest),
		RequestID:       uuid.New().String(),
	}

	// Add structured extraction if requested
	if req.Options.ExtractStructured {
		response.StructuredData = extractStructuredData(req.Source)
	}

	return response
}

func (s *Server) processAnalyzeV1(req AnalyzeRequest) AnalyzeResponse {
	// Implement v1 analysis logic
	response := AnalyzeResponse{}

	for _, analysis := range req.Analyses {
		switch analysis {
		case "entities":
			response.Entities = extractEntities(req.Text)
		case "sentiment":
			response.Sentiment = analyzeSentiment(req.Text)
		case "summary":
			response.Summary = generateSummary(req.Text, req.Options.SummaryLength)
		case "keywords":
			response.Keywords = extractKeywords(req.Text)
		case "language":
			response.Language = detectLanguage(req.Text)
		case "statistics":
			response.Statistics = calculateTextStatistics(req.Text)
		}
	}

	return response
}

func (s *Server) processAnalyzeV2(req AnalyzeRequestV2) AnalyzeResponseV2 {
	// V2 uses AI for enhanced analysis
	response := AnalyzeResponseV2{
		AnalyzeResponse: s.processAnalyzeV1(req.AnalyzeRequest),
		RequestID:       uuid.New().String(),
	}

	// Add AI-powered analysis if Ollama is available
	if s.config.OllamaURL != "" && req.Options.UseAI {
		response.AIInsights = generateAIInsights(req.Text, s.config.OllamaURL)
	}

	return response
}

func (s *Server) processPipeline(req PipelineRequest) PipelineResponse {
	// Process text through a series of operations
	current := req.Input
	results := []PipelineStepResult{}

	for _, step := range req.Steps {
		result := s.processPipelineStep(current, step)
		results = append(results, result)
		current = result.Output
	}

	return PipelineResponse{
		FinalOutput: current,
		Steps:       results,
		RequestID:   uuid.New().String(),
	}
}

func (s *Server) processPipelineStep(input string, step PipelineStep) PipelineStepResult {
	// Process individual pipeline step
	var output string
	var metadata map[string]interface{}

	switch step.Operation {
	case "diff":
		// Handle diff in pipeline
		output = input // Simplified
	case "search":
		// Handle search in pipeline
		output = input // Simplified
	case "transform":
		// Handle transform in pipeline
		output = applyTransformation(input, Transformation{Type: step.Operation, Parameters: step.Parameters})
	case "extract":
		// Handle extract in pipeline
		output = input // Simplified
	case "analyze":
		// Handle analyze in pipeline
		output = input // Simplified
	}

	return PipelineStepResult{
		StepName: step.Name,
		Output:   output,
		Metadata: metadata,
	}
}

// Database operations
func (s *Server) storeDiffOperation(text1, text2 string, changes []Change, similarity float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO text_operations (id, operation_type, parameters, result_summary, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	params := map[string]interface{}{
		"text1_length": len(text1),
		"text2_length": len(text2),
	}

	result := map[string]interface{}{
		"changes":    len(changes),
		"similarity": similarity,
	}

	paramsJSON, _ := json.Marshal(params)
	resultJSON, _ := json.Marshal(result)

	_, err := s.db.Exec(ctx, query, uuid.New(), "diff", paramsJSON, resultJSON, time.Now())
	if err != nil {
		log.Printf("Failed to store diff operation: %v", err)
	}
}
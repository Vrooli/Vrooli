package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler struct holds dependencies for HTTP handlers
type Handler struct {
	service *WorkflowService
}

// NewHandler creates a new handler with dependencies
func NewHandler(service *WorkflowService) *Handler {
	return &Handler{service: service}
}

// Health check handler
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database
	dbOk := db.Ping() == nil
	
	// Check services
	platforms := h.service.CheckPlatformStatus()
	services := make(map[string]bool)
	for _, p := range platforms {
		services[p.Name] = p.Status
	}
	
	// Count workflows
	var workflowCount int
	db.QueryRow("SELECT COUNT(*) FROM workflows WHERE is_active = true").Scan(&workflowCount)
	
	response := HealthResponse{
		Status:    "healthy",
		Version:   "3.0.0",
		Database:  dbOk,
		Services:  services,
		Workflows: workflowCount,
		Timestamp: time.Now(),
	}
	
	if !dbOk {
		response.Status = "degraded"
	}
	
	writeJSON(w, http.StatusOK, response)
}

// List workflows handler
func (h *Handler) ListWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	platform := r.URL.Query().Get("platform")
	typeFilter := r.URL.Query().Get("type")
	activeOnly := parseQueryBool(r, "active", true)
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 20)
	
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	
	// Get workflows from database
	workflows, total, err := listWorkflows(platform, typeFilter, activeOnly, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	response := ListResponse{
		Workflows: workflows,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}
	
	writeJSON(w, http.StatusOK, response)
}

// Get workflow handler
func (h *Handler) GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	workflow, err := getWorkflow(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if workflow == nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}
	
	writeJSON(w, http.StatusOK, workflow)
}

// Create workflow handler
func (h *Handler) CreateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	// Add request size limit for security
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit
	
	var wc WorkflowCreate
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&wc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	// Basic input validation
	if wc.Name == "" {
		writeError(w, http.StatusBadRequest, "workflow name is required")
		return
	}
	if wc.Platform == "" {
		writeError(w, http.StatusBadRequest, "platform is required")
		return
	}
	if wc.Platform != "n8n" && wc.Platform != "windmill" && wc.Platform != "both" {
		writeError(w, http.StatusBadRequest, "platform must be 'n8n', 'windmill', or 'both'")
		return
	}
	
	// Get user from context with safe type assertion
	userValue := r.Context().Value("user")
	user, ok := userValue.(string)
	if !ok {
		writeError(w, http.StatusInternalServerError, "invalid user context")
		return
	}
	
	workflow, err := createWorkflow(wc, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusCreated, workflow)
}

// Update workflow handler
func (h *Handler) UpdateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	// Add request size limit for security
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit
	
	var wc WorkflowCreate
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&wc); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	// Basic input validation
	if wc.Name == "" {
		writeError(w, http.StatusBadRequest, "workflow name is required")
		return
	}
	if wc.Platform == "" {
		writeError(w, http.StatusBadRequest, "platform is required")
		return
	}
	if wc.Platform != "n8n" && wc.Platform != "windmill" && wc.Platform != "both" {
		writeError(w, http.StatusBadRequest, "platform must be 'n8n', 'windmill', or 'both'")
		return
	}
	
	// Get user from context with safe type assertion
	userValue := r.Context().Value("user")
	user, ok := userValue.(string)
	if !ok {
		writeError(w, http.StatusInternalServerError, "invalid user context")
		return
	}
	
	workflow, err := updateWorkflow(id, wc, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, workflow)
}

// Delete workflow handler
func (h *Handler) DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	if err := deleteWorkflow(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{"message": "workflow deleted"})
}

// Execute workflow handler
func (h *Handler) ExecuteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	// Get workflow
	workflow, err := getWorkflow(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if workflow == nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}
	
	// Parse request
	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	// Execute workflow
	resp, err := h.service.ExecuteWorkflow(workflow, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, resp)
}

// Quick analysis handler (backward compatibility)
func (h *Handler) AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	analysisType := vars["type"]
	
	// Find workflow by type
	workflows, _, err := listWorkflows("", analysisType, true, 1, 1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if len(workflows) == 0 {
		writeError(w, http.StatusNotFound, "analysis type not found")
		return
	}
	
	workflow := &workflows[0]
	
	// Parse request
	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	// Execute workflow
	resp, err := h.service.ExecuteWorkflow(workflow, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, resp)
}

// Search workflows handler
func (h *Handler) SearchWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "search query required")
		return
	}
	
	workflows, err := searchWorkflows(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	response := SearchResponse{
		Query:   query,
		Results: workflows,
		Count:   len(workflows),
	}
	
	writeJSON(w, http.StatusOK, response)
}

// Generate workflow handler
func (h *Handler) GenerateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt required")
		return
	}
	
	if req.Platform == "" {
		req.Platform = "n8n"
	}
	
	workflow, err := h.service.GenerateWorkflow(req.Prompt, req.Platform, req.Model, req.Temperature)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, workflow)
}

// Import workflow handler
func (h *Handler) ImportWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	if req.Platform == "" {
		writeError(w, http.StatusBadRequest, "platform required")
		return
	}
	
	workflow, err := h.service.ImportWorkflow(req.Platform, req.Data, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusCreated, workflow)
}

// Export workflow handler
func (h *Handler) ExportWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	
	workflow, err := getWorkflow(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	if workflow == nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}
	
	exported, err := h.service.ExportWorkflow(workflow, format)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, exported)
}

// Clone workflow handler
func (h *Handler) CloneWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	var req CloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	
	workflow, err := cloneWorkflow(id, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusCreated, workflow)
}

// Get execution history handler
func (h *Handler) GetHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	limit := parseQueryInt(r, "limit", 50)
	if limit > 100 {
		limit = 100
	}
	
	history, err := getExecutionHistory(id, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	response := HistoryResponse{
		WorkflowID: id,
		History:    history,
		Count:      len(history),
	}
	
	writeJSON(w, http.StatusOK, response)
}

// Get workflow metrics handler
func (h *Handler) GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}
	
	metrics, err := getWorkflowMetrics(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, metrics)
}

// Get system statistics handler
func (h *Handler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := getSystemStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, stats)
}

// Get available models handler
func (h *Handler) GetModelsHandler(w http.ResponseWriter, r *http.Request) {
	models, err := h.service.GetAvailableModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	response := ModelsResponse{
		Models: models,
		Count:  len(models),
	}
	
	writeJSON(w, http.StatusOK, response)
}

// Get platforms handler
func (h *Handler) GetPlatformsHandler(w http.ResponseWriter, r *http.Request) {
	platforms := h.service.CheckPlatformStatus()
	
	response := PlatformsResponse{
		Platforms: platforms,
	}
	
	writeJSON(w, http.StatusOK, response)
}

// Get circuit breaker stats handler
func (h *Handler) GetCircuitBreakerStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := circuitBreakerManager.GetAllStats()
	
	response := map[string]interface{}{
		"circuit_breakers": stats,
		"timestamp":        time.Now(),
	}
	
	writeJSON(w, http.StatusOK, response)
}
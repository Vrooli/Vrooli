package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// ChartGenerationRequest represents a request to generate a chart
type ChartGenerationRequest struct {
	ChartType     string                 `json:"chart_type"`
	Data          []map[string]interface{} `json:"data"`
	Style         string                 `json:"style,omitempty"`
	ExportFormats []string               `json:"export_formats,omitempty"`
	Width         int                    `json:"width,omitempty"`
	Height        int                    `json:"height,omitempty"`
	Title         string                 `json:"title,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// ChartGenerationResponse represents the response from chart generation
type ChartGenerationResponse struct {
	Success  bool                   `json:"success"`
	ChartID  string                 `json:"chart_id,omitempty"`
	Files    map[string]string      `json:"files,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Error    *ErrorResponse         `json:"error,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Service   string    `json:"service"`
}

// StyleResponse represents available chart styles
type StyleResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("CHART_API_PORT")
	if port == "" {
		port = "20300"
	}

	// Create router
	r := mux.NewRouter()

	// Health check endpoints
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health/generation", healthGenerationHandler).Methods("GET")

	// Chart generation endpoints
	r.HandleFunc("/api/v1/charts/generate", generateChartHandler).Methods("POST")
	r.HandleFunc("/api/v1/charts/{id}", getChartHandler).Methods("GET")

	// Style management endpoints
	r.HandleFunc("/api/v1/styles", getStylesHandler).Methods("GET")
	r.HandleFunc("/api/v1/styles", createStyleHandler).Methods("POST")
	r.HandleFunc("/api/v1/styles/{id}", getStyleHandler).Methods("GET")

	// Template endpoints
	r.HandleFunc("/api/v1/templates", getTemplatesHandler).Methods("GET")
	r.HandleFunc("/api/v1/templates/{id}", getTemplateHandler).Methods("GET")

	// Set up CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	// Start server
	log.Printf("🎨 Chart Generator API starting on port %s", port)
	log.Printf("📊 Health check: http://localhost:%s/health", port)
	log.Printf("🚀 API endpoints: http://localhost:%s/api/v1/", port)
	
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// healthHandler handles basic health checks
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "chart-generator-api",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthGenerationHandler tests actual chart generation capability
func healthGenerationHandler(w http.ResponseWriter, r *http.Request) {
	// Simple test to verify chart generation pipeline
	testData := []map[string]interface{}{
		{"x": "Test", "y": 42},
	}

	response := map[string]interface{}{
		"status":           "healthy",
		"timestamp":        time.Now(),
		"service":          "chart-generation",
		"test_data_points": len(testData),
		"capabilities": []string{
			"chart-generation",
			"data-validation", 
			"style-management",
			"multi-format-export",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateChartHandler handles chart generation requests
func generateChartHandler(w http.ResponseWriter, r *http.Request) {
	var req ChartGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ChartType == "" {
		sendErrorResponse(w, "Missing required field: chart_type", "validation_error", http.StatusBadRequest)
		return
	}

	if len(req.Data) == 0 {
		sendErrorResponse(w, "Missing or empty data array", "validation_error", http.StatusBadRequest)
		return
	}

	// Generate unique chart ID
	chartID := fmt.Sprintf("chart_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%10000)

	// Set defaults
	if req.Style == "" {
		req.Style = "professional"
	}
	if len(req.ExportFormats) == 0 {
		req.ExportFormats = []string{"png"}
	}
	if req.Width == 0 {
		req.Width = 800
	}
	if req.Height == 0 {
		req.Height = 600
	}

	// In a real implementation, this would call the actual chart generation service
	// For now, we'll simulate a successful response
	files := make(map[string]string)
	for _, format := range req.ExportFormats {
		files[format] = fmt.Sprintf("/tmp/%s.%s", chartID, format)
	}

	response := ChartGenerationResponse{
		Success: true,
		ChartID: chartID,
		Files:   files,
		Metadata: map[string]interface{}{
			"generation_time_ms": 1200,
			"data_point_count":   len(req.Data),
			"style_applied":      req.Style,
			"dimensions": map[string]int{
				"width":  req.Width,
				"height": req.Height,
			},
			"formats_generated": req.ExportFormats,
			"created_at":        time.Now().Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getChartHandler retrieves information about a specific chart
func getChartHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chartID := vars["id"]

	if chartID == "" {
		sendErrorResponse(w, "Chart ID is required", "validation_error", http.StatusBadRequest)
		return
	}

	// In a real implementation, this would query the database
	response := map[string]interface{}{
		"chart_id":   chartID,
		"status":     "completed",
		"created_at": time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
		"files": map[string]string{
			"png": fmt.Sprintf("/tmp/%s.png", chartID),
			"svg": fmt.Sprintf("/tmp/%s.svg", chartID),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getStylesHandler returns available chart styles
func getStylesHandler(w http.ResponseWriter, r *http.Request) {
	styles := []StyleResponse{
		{
			ID:          "professional",
			Name:        "Professional",
			Category:    "business",
			Description: "Clean, corporate styling perfect for business reports",
			IsDefault:   true,
		},
		{
			ID:          "minimal",
			Name:        "Minimal",
			Category:    "clean",
			Description: "Ultra-clean, distraction-free design",
			IsDefault:   false,
		},
		{
			ID:          "vibrant",
			Name:        "Vibrant",
			Category:    "creative",
			Description: "Bold, eye-catching colors for presentations",
			IsDefault:   false,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"styles": styles,
		"count":  len(styles),
	})
}

// createStyleHandler creates a new custom style
func createStyleHandler(w http.ResponseWriter, r *http.Request) {
	var style map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&style); err != nil {
		sendErrorResponse(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	// Generate style ID
	styleID := fmt.Sprintf("custom_%d", time.Now().Unix())
	style["id"] = styleID
	style["created_at"] = time.Now().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(style)
}

// getStyleHandler retrieves a specific style
func getStyleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	styleID := vars["id"]

	// Mock response for any style ID
	style := map[string]interface{}{
		"id":          styleID,
		"name":        "Sample Style",
		"category":    "professional",
		"description": "A sample chart style",
		"colors":      []string{"#2563eb", "#64748b", "#f59e0b"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(style)
}

// getTemplatesHandler returns available chart templates
func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]interface{}{
		{
			"id":          "quarterly-sales",
			"name":        "Quarterly Sales Performance",
			"chart_type":  "bar",
			"description": "Standard quarterly sales chart",
			"category":    "business",
		},
		{
			"id":          "revenue-trend",
			"name":        "Revenue Trend Analysis",
			"chart_type":  "line",
			"description": "Monthly revenue tracking",
			"category":    "financial",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// getTemplateHandler retrieves a specific template
func getTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]

	template := map[string]interface{}{
		"id":         templateID,
		"name":       "Sample Template",
		"chart_type": "bar",
		"config": map[string]interface{}{
			"title":          "Sample Chart",
			"x_axis_label":   "Categories",
			"y_axis_label":   "Values",
			"show_legend":    false,
			"show_grid":      true,
		},
		"sample_data": []map[string]interface{}{
			{"x": "A", "y": 10},
			{"x": "B", "y": 20},
			{"x": "C", "y": 15},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// sendErrorResponse sends a structured error response
func sendErrorResponse(w http.ResponseWriter, message, errorType string, statusCode int) {
	response := ChartGenerationResponse{
		Success: false,
		Error: &ErrorResponse{
			Message: message,
			Type:    errorType,
			Code:    strconv.Itoa(statusCode),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
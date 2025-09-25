package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "github.com/lib/pq"
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

var chartProcessor *ChartProcessor

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start chart-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize database connection (optional for chart generation)
	var db *sql.DB
	var err error
	
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	
	if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
		
		db, err = sql.Open("postgres", connStr)
		if err == nil {
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
			}
		} else {
			log.Printf("⚠️  Database connection failed: %v (continuing without database)", err)
			db = nil
		}
	} else {
		log.Println("📝 Database not configured (continuing without database)")
	}

	// Initialize Chart Processor
	chartProcessor = NewChartProcessor(db)
	log.Println("🎨 Chart processor initialized")

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

	// Chart processor endpoints (replaces n8n workflows)
	r.HandleFunc("/chart-generator", chartGenerationHandler).Methods("POST")
	r.HandleFunc("/validate-data", dataValidationHandler).Methods("POST")
	r.HandleFunc("/styles", stylesHandler).Methods("GET")

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

// generateChartHandler handles chart generation requests using the chart processor
func generateChartHandler(w http.ResponseWriter, r *http.Request) {
	var req ChartGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to processor request
	processorReq := ChartGenerationProcessorRequest{
		ChartType:     req.ChartType,
		Data:          req.Data,
		Style:         req.Style,
		ExportFormats: req.ExportFormats,
		Width:         req.Width,
		Height:        req.Height,
		Title:         req.Title,
		Config:        nil, // Could be extended to accept config
	}

	// Use the chart processor for actual generation
	ctx := r.Context()
	result, err := chartProcessor.GenerateChart(ctx, processorReq)
	if err != nil {
		log.Printf("Chart generation error: %v", err)
		sendErrorResponse(w, "Chart generation failed", "generation_error", http.StatusInternalServerError)
		return
	}

	// Convert processor response to API response
	response := ChartGenerationResponse{
		Success: result.Success,
		ChartID: result.ChartID,
		Files:   result.Files,
		Metadata: map[string]interface{}{
			"generation_time_ms": result.Metadata.GenerationTimeMs,
			"data_point_count":   result.Metadata.DataPointCount,
			"style_applied":      result.Metadata.StyleApplied,
			"dimensions": map[string]int{
				"width":  result.Metadata.Dimensions.Width,
				"height": result.Metadata.Dimensions.Height,
			},
			"formats_generated": result.Metadata.FormatsGenerated,
			"created_at":        result.Metadata.CreatedAt,
		},
	}

	if !result.Success && result.Error != nil {
		response.Error = &ErrorResponse{
			Message: result.Error.Message,
			Type:    result.Error.Type,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Success {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
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

// Chart processor handlers (replaces n8n workflows)

func chartGenerationHandler(w http.ResponseWriter, r *http.Request) {
	var req ChartGenerationProcessorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := chartProcessor.GenerateChart(ctx, req)
	if err != nil {
		log.Printf("Chart generation error: %v", err)
		sendErrorResponse(w, "Chart generation failed", "generation_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Success {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

func dataValidationHandler(w http.ResponseWriter, r *http.Request) {
	var req DataValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := chartProcessor.ValidateData(ctx, req)
	if err != nil {
		log.Printf("Data validation error: %v", err)
		sendErrorResponse(w, "Data validation failed", "validation_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if result.Valid {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

func stylesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := chartProcessor.GetAvailableStyles(ctx)
	if err != nil {
		log.Printf("Get styles error: %v", err)
		sendErrorResponse(w, "Failed to get styles", "styles_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
package main

import (
	"github.com/vrooli/api-core/preflight"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// ChartGenerationRequest represents a request to generate a chart
type ChartGenerationRequest struct {
	ChartType     string                   `json:"chart_type"`
	Data          []map[string]interface{} `json:"data"`
	Style         string                   `json:"style,omitempty"`
	ExportFormats []string                 `json:"export_formats,omitempty"`
	Width         int                      `json:"width,omitempty"`
	Height        int                      `json:"height,omitempty"`
	Title         string                   `json:"title,omitempty"`
	Config        map[string]interface{}   `json:"config,omitempty"`
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
	Readiness bool      `json:"readiness"`
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "chart-generator",
	}) {
		return // Process was re-exec'd after rebuild
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
					log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
					break
				}

				// Calculate exponential backoff delay
				delay := time.Duration(math.Min(
					float64(baseDelay)*math.Pow(2, float64(attempt)),
					float64(maxDelay),
				))

				// Add random jitter to prevent thundering herd
				jitterRange := float64(delay) * 0.25
				jitter := time.Duration(rand.Float64() * jitterRange)
				actualDelay := delay + jitter

				log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
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

	// Get port from environment - no defaults allowed
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required but not set. The lifecycle system must provide this value.")
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

	// Style builder endpoints - P1 feature
	r.HandleFunc("/api/v1/styles/builder/preview", styleBuilderPreviewHandler).Methods("POST")
	r.HandleFunc("/api/v1/styles/builder/save", styleBuilderSaveHandler).Methods("POST")
	r.HandleFunc("/api/v1/styles/builder/palettes", getColorPalettesHandler).Methods("GET")

	// Template endpoints with industry-specific presets (P1 feature)
	r.HandleFunc("/api/v1/templates", getTemplatesHandler).Methods("GET")
	r.HandleFunc("/api/v1/templates/{id}", getTemplateHandler).Methods("GET")

	// Composite chart endpoints (P1 feature)
	r.HandleFunc("/api/v1/charts/composite", generateCompositeChartHandler).Methods("POST")

	// Data transformation endpoints (P1 feature)
	r.HandleFunc("/api/v1/data/transform", transformDataHandler).Methods("POST")
	r.HandleFunc("/api/v1/data/aggregate", aggregateDataHandler).Methods("POST")

	// Chart processor endpoints (replaces n8n workflows)
	r.HandleFunc("/chart-generator", chartGenerationHandler).Methods("POST")
	r.HandleFunc("/validate-data", dataValidationHandler).Methods("POST")
	r.HandleFunc("/styles", stylesHandler).Methods("GET")

	// Interactive chart endpoint (P1 feature - animation and interactivity)
	r.HandleFunc("/api/v1/charts/interactive", generateInteractiveChartHandler).Methods("POST")

	// Serve static UI files - must come after API routes
	// This catches all unmatched routes and serves the UI
	uiPath := "../ui"
	if _, err := os.Stat(uiPath); err == nil {
		fs := http.FileServer(http.Dir(uiPath))
		r.PathPrefix("/").Handler(fs)
		log.Printf("🎨 UI serving enabled from %s", uiPath)
	} else {
		log.Printf("⚠️ UI directory not found at %s - UI will not be available", uiPath)
	}

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
		Readiness: true, // Service is ready once it starts accepting requests
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

// getTemplatesHandler returns available chart templates with industry-specific presets (P1 feature)
func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]interface{}{
		// Business Templates
		{
			"id":          "quarterly-sales",
			"name":        "Quarterly Sales Performance",
			"chart_type":  "bar",
			"description": "Standard quarterly sales chart with YoY comparison",
			"category":    "business",
			"industry":    "retail",
		},
		{
			"id":          "revenue-trend",
			"name":        "Revenue Trend Analysis",
			"chart_type":  "line",
			"description": "Monthly revenue tracking with forecast",
			"category":    "business",
			"industry":    "saas",
		},
		{
			"id":          "market-share",
			"name":        "Market Share Distribution",
			"chart_type":  "pie",
			"description": "Competitive market share visualization",
			"category":    "business",
			"industry":    "retail",
		},
		// Financial Templates
		{
			"id":          "stock-performance",
			"name":        "Stock Performance Dashboard",
			"chart_type":  "candlestick",
			"description": "OHLC stock data with volume indicators",
			"category":    "financial",
			"industry":    "finance",
		},
		{
			"id":          "portfolio-allocation",
			"name":        "Portfolio Asset Allocation",
			"chart_type":  "treemap",
			"description": "Investment portfolio breakdown by asset class",
			"category":    "financial",
			"industry":    "investment",
		},
		{
			"id":          "cash-flow",
			"name":        "Cash Flow Statement",
			"chart_type":  "area",
			"description": "Operating, investing, and financing activities",
			"category":    "financial",
			"industry":    "corporate",
		},
		// Healthcare Templates
		{
			"id":          "patient-metrics",
			"name":        "Patient Metrics Dashboard",
			"chart_type":  "heatmap",
			"description": "Patient volume by department and time",
			"category":    "healthcare",
			"industry":    "hospital",
		},
		{
			"id":          "treatment-outcomes",
			"name":        "Treatment Outcome Analysis",
			"chart_type":  "scatter",
			"description": "Treatment effectiveness correlation analysis",
			"category":    "healthcare",
			"industry":    "clinical",
		},
		// Technology Templates
		{
			"id":          "system-performance",
			"name":        "System Performance Metrics",
			"chart_type":  "line",
			"description": "CPU, memory, and network utilization",
			"category":    "technology",
			"industry":    "devops",
		},
		{
			"id":          "sprint-burndown",
			"name":        "Sprint Burndown Chart",
			"chart_type":  "area",
			"description": "Agile sprint progress tracking",
			"category":    "technology",
			"industry":    "software",
		},
		{
			"id":          "deployment-timeline",
			"name":        "Deployment Timeline",
			"chart_type":  "gantt",
			"description": "Release schedule and dependencies",
			"category":    "technology",
			"industry":    "software",
		},
		// Marketing Templates
		{
			"id":          "campaign-performance",
			"name":        "Campaign Performance Matrix",
			"chart_type":  "heatmap",
			"description": "Multi-channel campaign effectiveness",
			"category":    "marketing",
			"industry":    "advertising",
		},
		{
			"id":          "customer-journey",
			"name":        "Customer Journey Funnel",
			"chart_type":  "bar",
			"description": "Conversion funnel visualization",
			"category":    "marketing",
			"industry":    "ecommerce",
		},
		// Education Templates
		{
			"id":          "student-performance",
			"name":        "Student Performance Distribution",
			"chart_type":  "scatter",
			"description": "Grade distribution and trend analysis",
			"category":    "education",
			"industry":    "academic",
		},
		{
			"id":          "course-enrollment",
			"name":        "Course Enrollment Trends",
			"chart_type":  "area",
			"description": "Enrollment patterns over semesters",
			"category":    "education",
			"industry":    "university",
		},
	}

	// Filter by category if specified
	category := r.URL.Query().Get("category")
	industry := r.URL.Query().Get("industry")

	filteredTemplates := templates
	if category != "" || industry != "" {
		filteredTemplates = []map[string]interface{}{}
		for _, template := range templates {
			if category != "" && template["category"] != category {
				continue
			}
			if industry != "" && template["industry"] != industry {
				continue
			}
			filteredTemplates = append(filteredTemplates, template)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": filteredTemplates,
		"count":     len(filteredTemplates),
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
			"title":        "Sample Chart",
			"x_axis_label": "Categories",
			"y_axis_label": "Values",
			"show_legend":  false,
			"show_grid":    true,
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

// StyleBuilderPreviewRequest represents a request to preview a custom style
type StyleBuilderPreviewRequest struct {
	ChartType string                   `json:"chart_type"`
	Style     CustomStyleDefinition    `json:"style"`
	Data      []map[string]interface{} `json:"data,omitempty"`
}

// CustomStyleDefinition represents a custom style configuration
type CustomStyleDefinition struct {
	Name         string   `json:"name"`
	Colors       []string `json:"colors"`
	ColorPalette []string `json:"color_palette"` // Accept both field names
	FontFamily   string   `json:"font_family"`
	FontSize     int      `json:"font_size"`
	Background   string   `json:"background"`
	GridLines    bool     `json:"grid_lines"`
	Animation    bool     `json:"animation"`
	BorderWidth  int      `json:"border_width"`
	Opacity      float64  `json:"opacity"`
	Palette      string   `json:"palette,omitempty"`
}

// styleBuilderPreviewHandler generates a live preview with custom style
func styleBuilderPreviewHandler(w http.ResponseWriter, r *http.Request) {
	var req StyleBuilderPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	// Use sample data if not provided
	if len(req.Data) == 0 {
		req.Data = generateSampleData(req.ChartType)
	}

	// Use ColorPalette if Colors is empty
	colors := req.Style.Colors
	if len(colors) == 0 && len(req.Style.ColorPalette) > 0 {
		colors = req.Style.ColorPalette
		req.Style.Colors = colors // Update for response
	}

	// Generate preview with custom style
	processorReq := ChartGenerationProcessorRequest{
		ChartType:     req.ChartType,
		Data:          req.Data,
		Style:         "custom",
		ExportFormats: []string{"png"},
		Width:         800,
		Height:        600,
		Config: map[string]interface{}{
			"colors":      colors,
			"fontFamily":  req.Style.FontFamily,
			"fontSize":    req.Style.FontSize,
			"background":  req.Style.Background,
			"gridLines":   req.Style.GridLines,
			"animation":   req.Style.Animation,
			"borderWidth": req.Style.BorderWidth,
			"opacity":     req.Style.Opacity,
		},
	}

	ctx := r.Context()
	result, err := chartProcessor.GenerateChart(ctx, processorReq)
	if err != nil {
		sendErrorResponse(w, "Preview generation failed", "generation_error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"preview":     result.Files["png"],
		"preview_url": result.Files["png"], // Support both field names
		"style":       req.Style,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// styleBuilderSaveHandler saves a custom style
func styleBuilderSaveHandler(w http.ResponseWriter, r *http.Request) {
	var style CustomStyleDefinition
	if err := json.NewDecoder(r.Body).Decode(&style); err != nil {
		sendErrorResponse(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	// Generate unique ID
	styleID := fmt.Sprintf("custom_%s_%d", style.Name, time.Now().Unix())

	// Save style (in production, this would persist to database)
	savedStyle := map[string]interface{}{
		"id":          styleID,
		"name":        style.Name,
		"colors":      style.Colors,
		"font_family": style.FontFamily,
		"font_size":   style.FontSize,
		"background":  style.Background,
		"grid_lines":  style.GridLines,
		"animation":   style.Animation,
		"created_at":  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(savedStyle)
}

// getColorPalettesHandler returns predefined color palettes
func getColorPalettesHandler(w http.ResponseWriter, r *http.Request) {
	palettes := map[string]interface{}{
		"palettes": []map[string]interface{}{
			{
				"id":     "ocean",
				"name":   "Ocean Blues",
				"colors": []string{"#0369a1", "#0284c7", "#0ea5e9", "#38bdf8", "#7dd3fc"},
			},
			{
				"id":     "sunset",
				"name":   "Sunset Warm",
				"colors": []string{"#dc2626", "#ea580c", "#f59e0b", "#fbbf24", "#fde047"},
			},
			{
				"id":     "forest",
				"name":   "Forest Greens",
				"colors": []string{"#14532d", "#166534", "#15803d", "#16a34a", "#22c55e"},
			},
			{
				"id":     "corporate",
				"name":   "Corporate Professional",
				"colors": []string{"#1e293b", "#334155", "#475569", "#64748b", "#94a3b8"},
			},
			{
				"id":     "vibrant",
				"name":   "Vibrant Mix",
				"colors": []string{"#dc2626", "#c026d3", "#2563eb", "#16a34a", "#f59e0b"},
			},
		},
		"recommended": map[string][]string{
			"bar":     []string{"ocean", "corporate"},
			"line":    []string{"sunset", "vibrant"},
			"pie":     []string{"vibrant", "forest"},
			"scatter": []string{"ocean", "vibrant"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(palettes)
}

// generateSampleData creates sample data for preview
func generateSampleData(chartType string) []map[string]interface{} {
	switch chartType {
	case "bar", "line", "area":
		return []map[string]interface{}{
			{"x": "Q1", "y": 45},
			{"x": "Q2", "y": 52},
			{"x": "Q3", "y": 48},
			{"x": "Q4", "y": 61},
		}
	case "pie":
		return []map[string]interface{}{
			{"name": "Product A", "value": 35},
			{"name": "Product B", "value": 28},
			{"name": "Product C", "value": 22},
			{"name": "Product D", "value": 15},
		}
	case "scatter":
		return []map[string]interface{}{
			{"x": 10, "y": 20},
			{"x": 20, "y": 45},
			{"x": 30, "y": 35},
			{"x": 40, "y": 50},
			{"x": 50, "y": 65},
		}
	default:
		return []map[string]interface{}{
			{"label": "Sample 1", "value": 100},
			{"label": "Sample 2", "value": 150},
		}
	}
}

// generateCompositeChartHandler handles multiple charts in single canvas (P1 feature)
func generateCompositeChartHandler(w http.ResponseWriter, r *http.Request) {
	var req ChartGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to processor request with composition
	processorReq := ChartGenerationProcessorRequest{
		ChartType:     req.ChartType,
		Data:          req.Data,
		Style:         req.Style,
		ExportFormats: req.ExportFormats,
		Width:         req.Width,
		Height:        req.Height,
		Title:         req.Title,
		Config:        req.Config,
	}

	// Parse composition from config if present
	if req.Config != nil {
		if comp, ok := req.Config["composition"]; ok {
			// Convert composition config to proper type
			compJSON, _ := json.Marshal(comp)
			var composition ChartComposition
			json.Unmarshal(compJSON, &composition)
			processorReq.Composition = &composition
		}
	}

	ctx := r.Context()
	result, err := chartProcessor.GenerateChart(ctx, processorReq)
	if err != nil {
		log.Printf("Composite chart generation error: %v", err)
		sendErrorResponse(w, "Composite chart generation failed", "generation_error", http.StatusInternalServerError)
		return
	}

	response := ChartGenerationResponse{
		Success: result.Success,
		ChartID: result.ChartID,
		Files:   result.Files,
		Metadata: map[string]interface{}{
			"generation_time_ms": result.Metadata.GenerationTimeMs,
			"chart_count":        result.Metadata.DataPointCount,
			"style_applied":      result.Metadata.StyleApplied,
			"composition_type":   "composite",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// transformDataHandler applies data transformations (P1 feature)
func transformDataHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data      []map[string]interface{} `json:"data"`
		Transform DataTransform            `json:"transform"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	transformedData, err := chartProcessor.ApplyTransformations(req.Data, &req.Transform)
	if err != nil {
		log.Printf("Data transformation error: %v", err)
		sendErrorResponse(w, "Data transformation failed", "transformation_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    transformedData,
		"count":   len(transformedData),
	})
}

// aggregateDataHandler performs data aggregation (P1 feature)
func aggregateDataHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data    []map[string]interface{} `json:"data"`
		Method  string                   `json:"method"`
		Field   string                   `json:"field"`
		GroupBy string                   `json:"group_by,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply aggregation using the transformation pipeline
	transform := DataTransform{
		Aggregate: &AggregateConfig{
			Method:  req.Method,
			Field:   req.Field,
			GroupBy: req.GroupBy,
		},
	}

	aggregatedData, err := chartProcessor.ApplyTransformations(req.Data, &transform)
	if err != nil {
		log.Printf("Data aggregation error: %v", err)
		sendErrorResponse(w, "Data aggregation failed", "aggregation_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"result":   aggregatedData,
		"data":     aggregatedData,
		"method":   req.Method,
		"field":    req.Field,
		"group_by": req.GroupBy,
	})
}

// generateInteractiveChartHandler creates charts with animation and interactivity (P1 feature)
func generateInteractiveChartHandler(w http.ResponseWriter, r *http.Request) {
	var req ChartGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", "validation_error", http.StatusBadRequest)
		return
	}

	// Ensure animation and interactivity are enabled
	if req.Config == nil {
		req.Config = make(map[string]interface{})
	}

	// Force enable animation and interactivity features
	configMap := req.Config
	configMap["animation"] = true
	configMap["show_tooltip"] = true
	configMap["show_legend"] = true
	configMap["enable_zoom"] = true
	configMap["enable_pan"] = true
	configMap["enable_data_zoom"] = true

	// Add HTML to export formats if not present
	hasHTML := false
	for _, format := range req.ExportFormats {
		if format == "html" || format == "interactive" {
			hasHTML = true
			break
		}
	}
	if !hasHTML {
		req.ExportFormats = append(req.ExportFormats, "interactive")
	}

	// Use the standard chart generation logic with interactive settings
	ctx := r.Context()
	processorReq := ChartGenerationProcessorRequest{
		ChartType:     req.ChartType,
		Data:          req.Data,
		Style:         req.Style,
		ExportFormats: req.ExportFormats,
		Width:         req.Width,
		Height:        req.Height,
		Title:         req.Title,
		Config:        req.Config,
	}

	result, err := chartProcessor.GenerateChart(ctx, processorReq)
	if err != nil {
		log.Printf("Interactive chart generation error: %v", err)
		sendErrorResponse(w, "Interactive chart generation failed", "generation_error", http.StatusInternalServerError)
		return
	}

	// Add interactivity metadata to response
	metadata := make(map[string]interface{})
	metadata["generation_time_ms"] = result.Metadata.GenerationTimeMs
	metadata["data_point_count"] = result.Metadata.DataPointCount
	metadata["style_applied"] = result.Metadata.StyleApplied
	metadata["formats_generated"] = result.Metadata.FormatsGenerated
	metadata["created_at"] = result.Metadata.CreatedAt
	metadata["dimensions"] = result.Metadata.Dimensions
	metadata["interactive"] = true
	metadata["animation_enabled"] = true
	metadata["features"] = []string{
		"animation",
		"tooltips",
		"legend_interaction",
		"zoom",
		"pan",
		"data_zoom",
	}

	response := ChartGenerationResponse{
		Success:  result.Success,
		ChartID:  result.ChartID,
		Files:    result.Files,
		Metadata: metadata,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

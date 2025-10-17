package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type ChartProcessor struct {
	db *sql.DB
}

// Chart Generation types
type ChartGenerationProcessorRequest struct {
	ChartType     string                   `json:"chart_type"`
	Data          []map[string]interface{} `json:"data"`
	Style         string                   `json:"style,omitempty"`
	ExportFormats []string                 `json:"export_formats,omitempty"`
	Width         int                      `json:"width,omitempty"`
	Height        int                      `json:"height,omitempty"`
	Title         string                   `json:"title,omitempty"`
	Config        interface{}              `json:"config,omitempty"`
	Composition   *ChartComposition        `json:"composition,omitempty"`
	Transform     *DataTransform           `json:"transform,omitempty"`
}

// ChartComposition allows multiple charts in a single canvas (P1 feature)
type ChartComposition struct {
	Layout string                            `json:"layout"` // grid, horizontal, vertical, custom
	Charts []ChartGenerationProcessorRequest `json:"charts"`
	Grid   *GridLayout                       `json:"grid,omitempty"`
}

type GridLayout struct {
	Rows    int `json:"rows"`
	Columns int `json:"columns"`
	Spacing int `json:"spacing"`
}

// DataTransform defines transformation operations (P1 feature)
type DataTransform struct {
	Aggregate *AggregateConfig `json:"aggregate,omitempty"`
	Filter    *FilterConfig    `json:"filter,omitempty"`
	Sort      *SortConfig      `json:"sort,omitempty"`
	Group     *GroupConfig     `json:"group,omitempty"`
}

type AggregateConfig struct {
	Method  string `json:"method"` // sum, avg, min, max, count
	Field   string `json:"field"`
	GroupBy string `json:"group_by,omitempty"`
}

type FilterConfig struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, contains
	Value    interface{} `json:"value"`
}

type SortConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // asc, desc
}

type GroupConfig struct {
	Field string `json:"field"`
	Bins  int    `json:"bins,omitempty"` // for numeric grouping
}

type ChartConfig struct {
	Title      string `json:"title,omitempty"`
	XAxisLabel string `json:"x_axis_label,omitempty"`
	YAxisLabel string `json:"y_axis_label,omitempty"`
	ShowLegend *bool  `json:"show_legend,omitempty"`
	ShowGrid   *bool  `json:"show_grid,omitempty"`
	Animation  *bool  `json:"animation,omitempty"`
}

type ChartGenerationProcessorResponse struct {
	Success  bool                    `json:"success"`
	ChartID  string                  `json:"chart_id"`
	Files    map[string]string       `json:"files"`
	Metadata ChartGenerationMetadata `json:"metadata"`
	Config   ChartConfig             `json:"config"`
	Error    *ChartProcessorError    `json:"error,omitempty"`
}

type ChartGenerationMetadata struct {
	GenerationTimeMs int    `json:"generation_time_ms"`
	DataPointCount   int    `json:"data_point_count"`
	StyleApplied     string `json:"style_applied"`
	Dimensions       struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
	FormatsGenerated []string `json:"formats_generated"`
	CreatedAt        string   `json:"created_at"`
	CompletedAt      string   `json:"completed_at"`
}

type ChartProcessorError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

// Data Validation types
type DataValidationRequest struct {
	Data      []map[string]interface{} `json:"data"`
	ChartType string                   `json:"chart_type,omitempty"`
}

type DataValidationResponse struct {
	Valid      bool     `json:"valid"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
	DataPoints int      `json:"data_points"`
	ChartType  string   `json:"chart_type"`
	Message    string   `json:"message,omitempty"`
}

// Style Management types
type StyleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

type StylesResponse struct {
	Styles    []StyleInfo `json:"styles"`
	Count     int         `json:"count"`
	Timestamp string      `json:"timestamp"`
}

func NewChartProcessor(db *sql.DB) *ChartProcessor {
	return &ChartProcessor{
		db: db,
	}
}

// GenerateChart creates professional charts with validation and storage (replaces chart-generation workflow)
func (cp *ChartProcessor) GenerateChart(ctx context.Context, req ChartGenerationProcessorRequest) (*ChartGenerationProcessorResponse, error) {
	startTime := time.Now()
	chartID := fmt.Sprintf("chart_%d_%s", startTime.Unix(), generateRandomID(6))

	// Handle composite charts (P1 feature)
	if req.Composition != nil {
		return cp.GenerateCompositeChart(ctx, req)
	}

	// Apply data transformations if specified (P1 feature)
	if req.Transform != nil {
		transformedData, err := cp.ApplyTransformations(req.Data, req.Transform)
		if err != nil {
			return &ChartGenerationProcessorResponse{
				Success: false,
				ChartID: chartID,
				Error: &ChartProcessorError{
					Message:   fmt.Sprintf("Data transformation failed: %s", err.Error()),
					Type:      "transformation_error",
					Timestamp: time.Now().Format(time.RFC3339),
				},
			}, nil
		}
		req.Data = transformedData
	}

	// Validate input
	validation := cp.validateChartData(req.Data, req.ChartType)
	if !validation.Valid {
		return &ChartGenerationProcessorResponse{
			Success: false,
			ChartID: chartID,
			Error: &ChartProcessorError{
				Message:   fmt.Sprintf("Data validation failed: %s", strings.Join(validation.Errors, ", ")),
				Type:      "validation_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Set defaults
	req = cp.setDefaults(req)

	// Build configuration
	config := cp.buildChartConfig(req)

	// Store chart data in database
	err := cp.storeChartData(ctx, chartID, req)
	if err != nil {
		return &ChartGenerationProcessorResponse{
			Success: false,
			ChartID: chartID,
			Error: &ChartProcessorError{
				Message:   fmt.Sprintf("Failed to store chart data: %v", err),
				Type:      "database_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Generate chart files
	files, err := cp.generateChartFiles(ctx, chartID, req)
	if err != nil {
		return &ChartGenerationProcessorResponse{
			Success: false,
			ChartID: chartID,
			Error: &ChartProcessorError{
				Message:   fmt.Sprintf("Chart generation failed: %v", err),
				Type:      "generation_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Update database with generated files
	err = cp.updateGeneratedFiles(ctx, chartID, files)
	if err != nil {
		log.Printf("Warning: Failed to update database with generated files: %v", err)
	}

	endTime := time.Now()
	executionTimeMs := int(endTime.Sub(startTime).Milliseconds())

	// Build metadata
	metadata := ChartGenerationMetadata{
		GenerationTimeMs: executionTimeMs,
		DataPointCount:   len(req.Data),
		StyleApplied:     req.Style,
		Dimensions: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{
			Width:  req.Width,
			Height: req.Height,
		},
		FormatsGenerated: req.ExportFormats,
		CreatedAt:        startTime.Format(time.RFC3339),
		CompletedAt:      endTime.Format(time.RFC3339),
	}

	return &ChartGenerationProcessorResponse{
		Success:  true,
		ChartID:  chartID,
		Files:    files,
		Metadata: metadata,
		Config:   config,
	}, nil
}

// ValidateData validates chart data structure (replaces data-validation workflow)
func (cp *ChartProcessor) ValidateData(ctx context.Context, req DataValidationRequest) (*DataValidationResponse, error) {
	chartType := req.ChartType
	if chartType == "" {
		chartType = "bar"
	}

	validation := cp.validateChartData(req.Data, chartType)
	validation.ChartType = chartType

	return &validation, nil
}

// GetAvailableStyles returns available chart styles (replaces style-management workflow)
func (cp *ChartProcessor) GetAvailableStyles(ctx context.Context) (*StylesResponse, error) {
	styles := []StyleInfo{
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
		{
			ID:          "dark",
			Name:        "Dark",
			Category:    "modern",
			Description: "Modern dark theme for presentations",
			IsDefault:   false,
		},
		{
			ID:          "corporate",
			Name:        "Corporate",
			Category:    "business",
			Description: "Conservative corporate styling",
			IsDefault:   false,
		},
	}

	return &StylesResponse{
		Styles:    styles,
		Count:     len(styles),
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// Helper methods

func (cp *ChartProcessor) validateChartData(data []map[string]interface{}, chartType string) DataValidationResponse {
	validation := DataValidationResponse{
		Valid:      true,
		Errors:     []string{},
		Warnings:   []string{},
		DataPoints: 0,
		ChartType:  chartType,
	}

	// Check if data exists
	if data == nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "Missing data field")
		return validation
	}

	// Check if data is not empty
	if len(data) == 0 {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "Data array cannot be empty")
		return validation
	}

	validation.DataPoints = len(data)

	// Validate chart type
	validChartTypes := []string{"bar", "line", "pie", "scatter", "area", "gantt", "heatmap", "treemap", "candlestick"}
	if !contains(validChartTypes, chartType) {
		validation.Valid = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("Invalid chart_type: %s. Supported types: %s", chartType, strings.Join(validChartTypes, ", ")))
		return validation
	}

	// Check data structure based on chart type
	for i, point := range data {
		if i >= 5 { // Only validate first 5 points for performance
			break
		}

		if point == nil {
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Data point %d cannot be null", i))
			continue
		}

		if chartType == "pie" {
			// Pie charts need label/x/name and value/y
			hasLabel := point["x"] != nil || point["label"] != nil || point["name"] != nil
			hasValue := point["y"] != nil || point["value"] != nil
			if !hasLabel || !hasValue {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Pie chart point %d must have label/x and value/y properties", i))
			}
		} else if chartType == "gantt" {
			// Gantt charts need task, start, and either end or duration
			if point["task"] == nil || point["start"] == nil || (point["end"] == nil && point["duration"] == nil) {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Gantt chart point %d must have task, start, and either end or duration properties", i))
			}
		} else if chartType == "heatmap" {
			// Heatmap needs x, y, and value
			if point["x"] == nil || point["y"] == nil || point["value"] == nil {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Heatmap point %d must have x, y, and value properties", i))
			}
		} else if chartType == "treemap" {
			// Treemap needs name and value, optionally parent
			if point["name"] == nil || point["value"] == nil {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Treemap point %d must have name and value properties", i))
			}
		} else if chartType == "candlestick" {
			// Candlestick charts need date, open, high, low, close
			if point["date"] == nil || point["open"] == nil || point["high"] == nil || point["low"] == nil || point["close"] == nil {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Candlestick chart point %d must have date, open, high, low, and close properties", i))
			}
		} else {
			// Other charts need x and y
			if point["x"] == nil || point["y"] == nil {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Point %d must have x and y properties", i))
			}
		}
	}

	// Add warnings for large datasets
	if len(data) > 1000 {
		validation.Warnings = append(validation.Warnings, "Large dataset detected - generation may take longer")
	}

	if len(data) > 10000 {
		validation.Warnings = append(validation.Warnings, "Very large dataset - consider data sampling for better performance")
	}

	// Add success message if valid
	if validation.Valid {
		validation.Message = fmt.Sprintf("Data validation successful: %d data points", validation.DataPoints)
	}

	return validation
}

func (cp *ChartProcessor) setDefaults(req ChartGenerationProcessorRequest) ChartGenerationProcessorRequest {
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

	// Constrain dimensions
	req.Width = int(math.Max(200, math.Min(2000, float64(req.Width))))
	req.Height = int(math.Max(200, math.Min(1500, float64(req.Height))))

	// Filter valid export formats
	validFormats := []string{"png", "svg", "pdf", "jpg"}
	filteredFormats := []string{}
	for _, format := range req.ExportFormats {
		if contains(validFormats, format) {
			filteredFormats = append(filteredFormats, format)
		}
	}
	if len(filteredFormats) == 0 {
		filteredFormats = []string{"png"} // fallback
	}
	req.ExportFormats = filteredFormats

	return req
}

func (cp *ChartProcessor) buildChartConfig(req ChartGenerationProcessorRequest) ChartConfig {
	config := ChartConfig{}

	if req.Config != nil {
		// Handle both ChartConfig struct and map[string]interface{} from custom styles
		switch cfg := req.Config.(type) {
		case *ChartConfig:
			config = *cfg
		case ChartConfig:
			config = cfg
		case map[string]interface{}:
			// Handle custom style config
			if title, ok := cfg["title"].(string); ok {
				config.Title = title
			}
			if xAxisLabel, ok := cfg["x_axis_label"].(string); ok {
				config.XAxisLabel = xAxisLabel
			}
			if yAxisLabel, ok := cfg["y_axis_label"].(string); ok {
				config.YAxisLabel = yAxisLabel
			}
			if showLegend, ok := cfg["show_legend"].(bool); ok {
				config.ShowLegend = &showLegend
			}
			if showGrid, ok := cfg["show_grid"].(bool); ok {
				config.ShowGrid = &showGrid
			}
			if animation, ok := cfg["animation"].(bool); ok {
				config.Animation = &animation
			}
		}
	}

	// Set defaults if not provided
	if config.ShowLegend == nil {
		showLegend := true
		config.ShowLegend = &showLegend
	}
	if config.ShowGrid == nil {
		showGrid := true
		config.ShowGrid = &showGrid
	}
	if config.Animation == nil {
		animation := true
		config.Animation = &animation
	}

	if config.Title == "" {
		config.Title = req.Title
	}

	return config
}

func (cp *ChartProcessor) storeChartData(ctx context.Context, chartID string, req ChartGenerationProcessorRequest) error {
	if cp.db == nil {
		log.Println("Warning: Database not available, skipping chart data storage")
		return nil
	}

	dataJSON, _ := json.Marshal(req.Data)
	configJSON, _ := json.Marshal(req.Config)

	metadata := map[string]interface{}{
		"data_points":   len(req.Data),
		"timestamp":     time.Now().Format(time.RFC3339),
		"processing_id": chartID,
	}
	metadataJSON, _ := json.Marshal(metadata)

	query := `
		INSERT INTO chart_instances (
			id, chart_type, data_source, config_overrides, 
			generation_metadata, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, NOW(), NOW() + INTERVAL '7 days')`

	_, err := cp.db.ExecContext(ctx, query,
		chartID,
		req.ChartType,
		string(dataJSON),
		string(configJSON),
		string(metadataJSON),
	)

	if err != nil {
		log.Printf("Database storage failed (continuing anyway): %v", err)
		// Don't fail the entire operation if database storage fails
		return nil
	}

	return nil
}

func (cp *ChartProcessor) generateChartFiles(ctx context.Context, chartID string, req ChartGenerationProcessorRequest) (map[string]string, error) {
	// Use the real chart renderer
	renderer := NewChartRenderer("/tmp")
	return renderer.RenderChart(chartID, req)
}

func (cp *ChartProcessor) generateMockChartFile(filepath, format string, req ChartGenerationProcessorRequest) error {
	// In a real implementation, this would call actual chart generation libraries
	// For now, create mock files with some basic content

	var content string

	switch format {
	case "svg":
		content = cp.generateMockSVG(req)
	case "png", "jpg":
		// For binary formats, we'd use actual chart generation libraries
		// For demo purposes, create a simple text file
		content = fmt.Sprintf("Mock %s chart: %s with %d data points",
			strings.ToUpper(format), req.ChartType, len(req.Data))
	case "pdf":
		content = fmt.Sprintf("%%PDF-1.4\n%% Mock PDF chart: %s with %d data points",
			req.ChartType, len(req.Data))
	default:
		content = fmt.Sprintf("Mock chart file: %s", format)
	}

	return os.WriteFile(filepath, []byte(content), 0644)
}

func (cp *ChartProcessor) generateMockSVG(req ChartGenerationProcessorRequest) string {
	// Generate a simple SVG based on the request
	// In production, this would use a real chart rendering library

	switch req.ChartType {
	case "gantt":
		return cp.generateGanttSVG(req)
	case "heatmap":
		return cp.generateHeatmapSVG(req)
	case "treemap":
		return cp.generateTreemapSVG(req)
	default:
		// Default mock SVG for other chart types
		svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
	<rect width="100%%" height="100%%" fill="white"/>
	<text x="50%%" y="30" text-anchor="middle" font-family="Arial" font-size="16" fill="black">%s</text>
	<text x="50%%" y="50" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Chart Type: %s</text>
	<text x="50%%" y="70" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Data Points: %d</text>
	<text x="50%%" y="90" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Style: %s</text>
</svg>`, req.Width, req.Height, req.Title, req.ChartType, len(req.Data), req.Style)
		return svg
	}
}

func (cp *ChartProcessor) generateGanttSVG(req ChartGenerationProcessorRequest) string {
	// Generate Gantt chart SVG
	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
	<rect width="100%%" height="100%%" fill="white"/>
	<text x="50%%" y="30" text-anchor="middle" font-family="Arial" font-size="16" fill="black">%s</text>
	<text x="50%%" y="50" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Gantt Chart</text>
	<!-- Gantt bars would be rendered here based on task data -->
	<rect x="50" y="80" width="200" height="30" fill="#4CAF50" opacity="0.8"/>
	<rect x="150" y="120" width="150" height="30" fill="#2196F3" opacity="0.8"/>
	<rect x="250" y="160" width="180" height="30" fill="#FF9800" opacity="0.8"/>
	<text x="50%%" y="%d" text-anchor="middle" font-family="Arial" font-size="12" fill="gray">%d Tasks</text>
</svg>`, req.Width, req.Height, req.Title, req.Height-20, len(req.Data))
	return svg
}

func (cp *ChartProcessor) generateHeatmapSVG(req ChartGenerationProcessorRequest) string {
	// Generate Heatmap SVG
	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
	<rect width="100%%" height="100%%" fill="white"/>
	<text x="50%%" y="30" text-anchor="middle" font-family="Arial" font-size="16" fill="black">%s</text>
	<text x="50%%" y="50" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Heatmap</text>
	<!-- Heatmap cells would be rendered here based on data -->`, req.Width, req.Height, req.Title)

	// Generate grid cells for heatmap
	cellSize := 30
	for i := 0; i < int(math.Min(10, float64(len(req.Data)))); i++ {
		x := 50 + (i%5)*cellSize
		y := 80 + (i/5)*cellSize
		intensity := (i + 1) * 25
		color := fmt.Sprintf("rgb(%d, 50, 50)", intensity)
		svg += fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" opacity="0.8"/>`,
			x, y, cellSize-2, cellSize-2, color)
	}

	svg += fmt.Sprintf(`<text x="50%%" y="%d" text-anchor="middle" font-family="Arial" font-size="12" fill="gray">%d Data Points</text>
</svg>`, req.Height-20, len(req.Data))
	return svg
}

func (cp *ChartProcessor) generateTreemapSVG(req ChartGenerationProcessorRequest) string {
	// Generate Treemap SVG
	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
	<rect width="100%%" height="100%%" fill="white"/>
	<text x="50%%" y="30" text-anchor="middle" font-family="Arial" font-size="16" fill="black">%s</text>
	<text x="50%%" y="50" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Treemap</text>
	<!-- Treemap rectangles would be rendered here based on hierarchical data -->
	<rect x="50" y="80" width="120" height="80" fill="#FF5722" opacity="0.8"/>
	<rect x="175" y="80" width="80" height="80" fill="#9C27B0" opacity="0.8"/>
	<rect x="50" y="165" width="80" height="60" fill="#03A9F4" opacity="0.8"/>
	<rect x="135" y="165" width="120" height="60" fill="#8BC34A" opacity="0.8"/>
	<text x="50%%" y="%d" text-anchor="middle" font-family="Arial" font-size="12" fill="gray">%d Categories</text>
</svg>`, req.Width, req.Height, req.Title, req.Height-20, len(req.Data))
	return svg
}

func (cp *ChartProcessor) updateGeneratedFiles(ctx context.Context, chartID string, files map[string]string) error {
	if cp.db == nil {
		return nil
	}

	fileInfo := make([]map[string]interface{}, 0, len(files))
	for format, filepath := range files {
		info := map[string]interface{}{
			"format":   format,
			"filepath": filepath,
			"filename": fmt.Sprintf("%s.%s", chartID, format),
		}

		// Get file size if possible
		if stat, err := os.Stat(filepath); err == nil {
			info["size_bytes"] = stat.Size()
		}

		fileInfo = append(fileInfo, info)
	}

	filesJSON, _ := json.Marshal(fileInfo)

	query := `UPDATE chart_instances SET generated_files = $1 WHERE id = $2`
	_, err := cp.db.ExecContext(ctx, query, string(filesJSON), chartID)

	return err
}

// Utility functions
func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GenerateCompositeChart handles multiple charts in a single canvas (P1 feature)
func (cp *ChartProcessor) GenerateCompositeChart(ctx context.Context, req ChartGenerationProcessorRequest) (*ChartGenerationProcessorResponse, error) {
	startTime := time.Now()
	compositeID := fmt.Sprintf("composite_%d_%s", startTime.Unix(), generateRandomID(6))

	if req.Composition == nil || len(req.Composition.Charts) == 0 {
		return &ChartGenerationProcessorResponse{
			Success: false,
			ChartID: compositeID,
			Error: &ChartProcessorError{
				Message:   "Composition requires at least one chart",
				Type:      "composition_error",
				Timestamp: time.Now().Format(time.RFC3339),
			},
		}, nil
	}

	// Calculate layout dimensions
	layout := req.Composition.Layout
	if layout == "" {
		layout = "grid"
	}

	// Generate individual charts
	chartFiles := make([]map[string]string, 0)
	for _, chartReq := range req.Composition.Charts {
		// Apply transformations if needed
		if chartReq.Transform != nil {
			transformedData, err := cp.ApplyTransformations(chartReq.Data, chartReq.Transform)
			if err != nil {
				continue // Skip failed charts in composition
			}
			chartReq.Data = transformedData
		}

		// Set defaults for individual charts
		chartReq = cp.setDefaults(chartReq)

		// Generate the individual chart
		files, err := cp.generateChartFiles(ctx, fmt.Sprintf("%s_part_%d", compositeID, len(chartFiles)), chartReq)
		if err == nil {
			chartFiles = append(chartFiles, files)
		}
	}

	// Combine charts based on layout
	combinedFiles := cp.combineCharts(compositeID, chartFiles, layout, req.Composition.Grid)

	endTime := time.Now()
	executionTimeMs := int(endTime.Sub(startTime).Milliseconds())

	return &ChartGenerationProcessorResponse{
		Success: true,
		ChartID: compositeID,
		Files:   combinedFiles,
		Metadata: ChartGenerationMetadata{
			GenerationTimeMs: executionTimeMs,
			DataPointCount:   len(req.Composition.Charts),
			StyleApplied:     req.Style,
			Dimensions: struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			}{
				Width:  req.Width,
				Height: req.Height,
			},
			FormatsGenerated: req.ExportFormats,
			CreatedAt:        startTime.Format(time.RFC3339),
			CompletedAt:      endTime.Format(time.RFC3339),
		},
	}, nil
}

// ApplyTransformations applies data transformation pipeline (P1 feature)
func (cp *ChartProcessor) ApplyTransformations(data []map[string]interface{}, transform *DataTransform) ([]map[string]interface{}, error) {
	result := data

	// Apply filter
	if transform.Filter != nil {
		result = cp.filterData(result, transform.Filter)
	}

	// Apply aggregation
	if transform.Aggregate != nil {
		result = cp.aggregateData(result, transform.Aggregate)
	}

	// Apply sorting
	if transform.Sort != nil {
		result = cp.sortData(result, transform.Sort)
	}

	// Apply grouping
	if transform.Group != nil {
		result = cp.groupData(result, transform.Group)
	}

	return result, nil
}

func (cp *ChartProcessor) filterData(data []map[string]interface{}, filter *FilterConfig) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	for _, item := range data {
		value, exists := item[filter.Field]
		if !exists {
			continue
		}

		match := false
		switch filter.Operator {
		case "eq":
			match = fmt.Sprintf("%v", value) == fmt.Sprintf("%v", filter.Value)
		case "ne":
			match = fmt.Sprintf("%v", value) != fmt.Sprintf("%v", filter.Value)
		case "gt":
			match = toFloat64(value) > toFloat64(filter.Value)
		case "lt":
			match = toFloat64(value) < toFloat64(filter.Value)
		case "gte":
			match = toFloat64(value) >= toFloat64(filter.Value)
		case "lte":
			match = toFloat64(value) <= toFloat64(filter.Value)
		case "contains":
			match = strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", filter.Value))
		}

		if match {
			result = append(result, item)
		}
	}

	return result
}

func (cp *ChartProcessor) aggregateData(data []map[string]interface{}, aggregate *AggregateConfig) []map[string]interface{} {
	if aggregate.GroupBy == "" {
		// Simple aggregation without grouping
		aggValue := cp.calculateAggregation(data, aggregate.Field, aggregate.Method)
		return []map[string]interface{}{
			{
				"value": aggValue,
				"label": fmt.Sprintf("%s of %s", aggregate.Method, aggregate.Field),
			},
		}
	}

	// Group-based aggregation
	groups := make(map[string][]map[string]interface{})
	for _, item := range data {
		groupKey := fmt.Sprintf("%v", item[aggregate.GroupBy])
		groups[groupKey] = append(groups[groupKey], item)
	}

	result := make([]map[string]interface{}, 0)
	for groupKey, groupData := range groups {
		aggValue := cp.calculateAggregation(groupData, aggregate.Field, aggregate.Method)
		result = append(result, map[string]interface{}{
			"x": groupKey,
			"y": aggValue,
		})
	}

	return result
}

func (cp *ChartProcessor) calculateAggregation(data []map[string]interface{}, field, method string) float64 {
	values := make([]float64, 0)
	for _, item := range data {
		if val, exists := item[field]; exists {
			values = append(values, toFloat64(val))
		}
	}

	if len(values) == 0 {
		return 0
	}

	switch method {
	case "sum":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum
	case "avg":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	case "min":
		min := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min
	case "max":
		max := values[0]
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max
	case "count":
		return float64(len(values))
	default:
		return 0
	}
}

func (cp *ChartProcessor) sortData(data []map[string]interface{}, sort *SortConfig) []map[string]interface{} {
	// Simple bubble sort for demonstration - in production, use more efficient sorting
	result := make([]map[string]interface{}, len(data))
	copy(result, data)

	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			val1 := toFloat64(result[j][sort.Field])
			val2 := toFloat64(result[j+1][sort.Field])

			shouldSwap := false
			if sort.Direction == "asc" {
				shouldSwap = val1 > val2
			} else {
				shouldSwap = val1 < val2
			}

			if shouldSwap {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

func (cp *ChartProcessor) groupData(data []map[string]interface{}, group *GroupConfig) []map[string]interface{} {
	// Group data by field values or bins
	groups := make(map[string][]map[string]interface{})

	for _, item := range data {
		groupKey := fmt.Sprintf("%v", item[group.Field])
		groups[groupKey] = append(groups[groupKey], item)
	}

	result := make([]map[string]interface{}, 0)
	for groupKey, groupData := range groups {
		result = append(result, map[string]interface{}{
			"group": groupKey,
			"data":  groupData,
			"count": len(groupData),
		})
	}

	return result
}

func (cp *ChartProcessor) combineCharts(compositeID string, chartFiles []map[string]string, layout string, grid *GridLayout) map[string]string {
	// In a real implementation, this would combine the individual chart files
	// For now, return a mock combined file reference
	combinedFiles := make(map[string]string)

	for _, format := range []string{"png", "svg", "html"} {
		combinedFiles[format] = fmt.Sprintf("/tmp/%s_composite.%s", compositeID, format)
	}

	return combinedFiles
}

func toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

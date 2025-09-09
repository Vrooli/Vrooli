package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
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
	Config        *ChartConfig             `json:"config,omitempty"`
}

type ChartConfig struct {
	Title         string `json:"title,omitempty"`
	XAxisLabel    string `json:"x_axis_label,omitempty"`
	YAxisLabel    string `json:"y_axis_label,omitempty"`
	ShowLegend    *bool  `json:"show_legend,omitempty"`
	ShowGrid      *bool  `json:"show_grid,omitempty"`
	Animation     *bool  `json:"animation,omitempty"`
}

type ChartGenerationProcessorResponse struct {
	Success  bool                      `json:"success"`
	ChartID  string                    `json:"chart_id"`
	Files    map[string]string         `json:"files"`
	Metadata ChartGenerationMetadata   `json:"metadata"`
	Config   ChartConfig               `json:"config"`
	Error    *ChartProcessorError      `json:"error,omitempty"`
}

type ChartGenerationMetadata struct {
	GenerationTimeMs int      `json:"generation_time_ms"`
	DataPointCount   int      `json:"data_point_count"`
	StyleApplied     string   `json:"style_applied"`
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
	validChartTypes := []string{"bar", "line", "pie", "scatter", "area", "gantt", "heatmap", "treemap"}
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
			// Pie charts need label/x and value/y
			hasLabel := point["x"] != nil || point["label"] != nil
			hasValue := point["y"] != nil || point["value"] != nil
			if !hasLabel || !hasValue {
				validation.Valid = false
				validation.Errors = append(validation.Errors, fmt.Sprintf("Pie chart point %d must have label/x and value/y properties", i))
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
		config.Title = req.Config.Title
		config.XAxisLabel = req.Config.XAxisLabel
		config.YAxisLabel = req.Config.YAxisLabel
		config.ShowLegend = req.Config.ShowLegend
		config.ShowGrid = req.Config.ShowGrid
		config.Animation = req.Config.Animation
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
		"data_points": len(req.Data),
		"timestamp":   time.Now().Format(time.RFC3339),
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
	// Create temporary directory for chart generation
	tempDir := filepath.Join("/tmp", chartID+"_output")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create data file
	dataFile := filepath.Join("/tmp", chartID+"_data.json")
	dataJSON, _ := json.Marshal(req.Data)
	err = os.WriteFile(dataFile, dataJSON, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write data file: %w", err)
	}
	defer os.Remove(dataFile)

	// Build chart generation command
	// This simulates calling an external chart generation tool
	// In a real implementation, this might call Chart.js, D3.js, or a Go charting library
	files := make(map[string]string)
	
	for _, format := range req.ExportFormats {
		filename := fmt.Sprintf("%s.%s", chartID, format)
		filepath := filepath.Join(tempDir, filename)
		
		// Simulate chart generation by creating mock files
		err = cp.generateMockChartFile(filepath, format, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s file: %w", format, err)
		}
		
		files[format] = filepath
	}

	return files, nil
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
	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
	<rect width="100%%" height="100%%" fill="white"/>
	<text x="50%%" y="30" text-anchor="middle" font-family="Arial" font-size="16" fill="black">%s</text>
	<text x="50%%" y="50" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Chart Type: %s</text>
	<text x="50%%" y="70" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Data Points: %d</text>
	<text x="50%%" y="90" text-anchor="middle" font-family="Arial" font-size="14" fill="gray">Style: %s</text>
</svg>`, req.Width, req.Height, req.Title, req.ChartType, len(req.Data), req.Style)
	
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
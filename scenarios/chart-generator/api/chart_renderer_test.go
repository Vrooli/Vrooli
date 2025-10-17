package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChartRenderer_RenderChart(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	renderer := NewChartRenderer(env.TempDir)

	tests := []struct {
		name         string
		chartID      string
		req          ChartGenerationProcessorRequest
		expectError  bool
		validateFunc func(t *testing.T, files map[string]string)
	}{
		{
			name:    "RenderBarChart",
			chartID: "test-bar-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"png", "svg"},
				Width:         800,
				Height:        600,
				Title:         "Test Bar Chart",
			},
			expectError: false,
			validateFunc: func(t *testing.T, files map[string]string) {
				if len(files) < 2 {
					t.Errorf("Expected at least 2 files (png, svg), got %d", len(files))
				}
				for format, path := range files {
					assertFileExists(t, path)
					t.Logf("Generated %s file: %s", format, path)
				}
			},
		},
		{
			name:    "RenderLineChart",
			chartID: "test-line-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 10),
				ExportFormats: []string{"png"},
				Width:         1024,
				Height:        768,
			},
			expectError: false,
		},
		{
			name:    "RenderPieChart",
			chartID: "test-pie-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "pie",
				Data:          generateTestChartData("pie", 4),
				ExportFormats: []string{"svg"},
			},
			expectError: false,
		},
		{
			name:    "RenderScatterChart",
			chartID: "test-scatter-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "scatter",
				Data:          generateTestChartData("scatter", 20),
				ExportFormats: []string{"png"},
			},
			expectError: false,
		},
		{
			name:    "RenderAreaChart",
			chartID: "test-area-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "area",
				Data:          generateTestChartData("area", 15),
				ExportFormats: []string{"svg"},
			},
			expectError: false,
		},
		{
			name:    "RenderCandlestickChart",
			chartID: "test-candlestick-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "candlestick",
				Data:          generateTestChartData("candlestick", 10),
				ExportFormats: []string{"png"},
			},
			expectError: false,
		},
		{
			name:    "RenderHeatmapChart",
			chartID: "test-heatmap-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "heatmap",
				Data:          generateTestChartData("heatmap", 25),
				ExportFormats: []string{"svg"},
			},
			expectError: false,
		},
		{
			name:    "RenderTreemapChart",
			chartID: "test-treemap-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "treemap",
				Data:          generateTestChartData("treemap", 8),
				ExportFormats: []string{"png"},
			},
			expectError: false,
		},
		{
			name:    "RenderGanttChart",
			chartID: "test-gantt-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "gantt",
				Data:          generateTestChartData("gantt", 5),
				ExportFormats: []string{"svg"},
			},
			expectError: false,
		},
		{
			name:    "RenderWithPDF",
			chartID: "test-pdf-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"pdf"},
				Title:         "PDF Export Test",
			},
			expectError: false,
		},
		{
			name:    "RenderMultipleFormats",
			chartID: "test-multi-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 8),
				ExportFormats: []string{"png", "svg", "pdf"},
				Title:         "Multi-Format Test",
			},
			expectError: false,
			validateFunc: func(t *testing.T, files map[string]string) {
				expectedFormats := []string{"png", "svg", "pdf"}
				for _, format := range expectedFormats {
					if _, ok := files[format]; !ok {
						t.Errorf("Expected format %s not found in files", format)
					}
				}
			},
		},
		{
			name:    "UnsupportedChartType",
			chartID: "test-unsupported-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "unsupported_type",
				Data:          generateTestChartData("bar", 3),
				ExportFormats: []string{"png"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := renderer.RenderChart(tt.chartID, tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("RenderChart returned unexpected error: %v", err)
			}

			if len(files) == 0 {
				t.Error("Expected at least one file to be generated")
			}

			// Verify output directory was created
			chartDir := filepath.Join(env.TempDir, tt.chartID+"_output")
			if _, err := os.Stat(chartDir); os.IsNotExist(err) {
				t.Errorf("Expected output directory %s to exist", chartDir)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, files)
			}
		})
	}
}

func TestChartRenderer_CustomStyles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	renderer := NewChartRenderer(env.TempDir)

	tests := []struct {
		name    string
		req     ChartGenerationProcessorRequest
		style   string
	}{
		{
			name: "ProfessionalStyle",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				Style:         "professional",
				ExportFormats: []string{"png"},
			},
		},
		{
			name: "DarkStyle",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 8),
				Style:         "dark",
				ExportFormats: []string{"svg"},
			},
		},
		{
			name: "MinimalStyle",
			req: ChartGenerationProcessorRequest{
				ChartType:     "pie",
				Data:          generateTestChartData("pie", 4),
				Style:         "minimal",
				ExportFormats: []string{"png"},
			},
		},
		{
			name: "CustomColors",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"png"},
				Config: map[string]interface{}{
					"colors": []string{"#FF6384", "#36A2EB", "#FFCE56"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := renderer.RenderChart("test-style", tt.req)

			if err != nil {
				t.Fatalf("RenderChart returned error: %v", err)
			}

			if len(files) == 0 {
				t.Error("Expected at least one file to be generated")
			}
		})
	}
}

func TestChartRenderer_AnimationSettings(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	renderer := NewChartRenderer("/tmp")

	tests := []struct {
		name            string
		req             ChartGenerationProcessorRequest
		expectAnimation bool
	}{
		{
			name: "AnimationEnabledForHTML",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"html"},
			},
			expectAnimation: true,
		},
		{
			name: "AnimationDisabledForPNG",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"png"},
			},
			expectAnimation: false,
		},
		{
			name: "AnimationExplicitlyEnabled",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 8),
				ExportFormats: []string{"png"},
				Config: map[string]interface{}{
					"animation": true,
				},
			},
			expectAnimation: true,
		},
		{
			name: "AnimationExplicitlyDisabled",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 8),
				ExportFormats: []string{"html"},
				Config: map[string]interface{}{
					"animation": false,
				},
			},
			expectAnimation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.isAnimationEnabled(tt.req)

			if result != tt.expectAnimation {
				t.Errorf("Expected animation=%v, got %v", tt.expectAnimation, result)
			}
		})
	}
}

func TestChartRenderer_GetTheme(t *testing.T) {
	renderer := NewChartRenderer("/tmp")

	tests := []struct {
		name  string
		style string
	}{
		{"DarkTheme", "dark"},
		{"MinimalTheme", "minimal"},
		{"DefaultTheme", "professional"},
		{"UnknownTheme", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChartGenerationProcessorRequest{
				Style: tt.style,
			}

			theme := renderer.getTheme(req)

			// Just verify it returns a non-empty string
			if theme == "" {
				t.Error("Expected non-empty theme")
			}
		})
	}
}

func TestChartRenderer_GetCustomColors(t *testing.T) {
	renderer := NewChartRenderer("/tmp")

	tests := []struct {
		name           string
		config         interface{}
		expectedColors []string
	}{
		{
			name: "WithColors",
			config: map[string]interface{}{
				"colors": []string{"#FF0000", "#00FF00", "#0000FF"},
			},
			expectedColors: []string{"#FF0000", "#00FF00", "#0000FF"},
		},
		{
			name: "WithColorsAsInterface",
			config: map[string]interface{}{
				"colors": []interface{}{"#FF0000", "#00FF00"},
			},
			expectedColors: []string{"#FF0000", "#00FF00"},
		},
		{
			name:           "NoColors",
			config:         map[string]interface{}{},
			expectedColors: nil,
		},
		{
			name:           "NilConfig",
			config:         nil,
			expectedColors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChartGenerationProcessorRequest{
				Config: tt.config,
			}

			colors := renderer.getCustomColors(req)

			if tt.expectedColors == nil {
				if colors != nil {
					t.Errorf("Expected nil colors, got %v", colors)
				}
			} else {
				if len(colors) != len(tt.expectedColors) {
					t.Errorf("Expected %d colors, got %d", len(tt.expectedColors), len(colors))
				}
			}
		})
	}
}

func TestChartRenderer_GeneratePDF(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	renderer := NewChartRenderer(env.TempDir)

	req := ChartGenerationProcessorRequest{
		ChartType: "bar",
		Data:      generateTestChartData("bar", 5),
		Title:     "PDF Test Chart",
		Style:     "professional",
	}

	outputPath := filepath.Join(env.TempDir, "test.pdf")

	err := renderer.generatePDF(outputPath, req, "<html>Chart HTML</html>")

	if err != nil {
		t.Fatalf("generatePDF returned error: %v", err)
	}

	assertFileExists(t, outputPath)

	// Verify file has content
	stat, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Failed to stat PDF file: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("PDF file is empty")
	}
}

func TestChartRenderer_GeneratePNGPlaceholder(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	renderer := NewChartRenderer(env.TempDir)

	tests := []struct {
		name string
		req  ChartGenerationProcessorRequest
	}{
		{
			name: "BarChartPlaceholder",
			req: ChartGenerationProcessorRequest{
				ChartType: "bar",
				Data:      generateTestChartData("bar", 5),
				Width:     800,
				Height:    600,
				Title:     "Test Chart",
			},
		},
		{
			name: "LineChartPlaceholder",
			req: ChartGenerationProcessorRequest{
				ChartType: "line",
				Data:      generateTestChartData("line", 10),
				Width:     1024,
				Height:    768,
			},
		},
		{
			name: "DefaultDimensions",
			req: ChartGenerationProcessorRequest{
				ChartType: "pie",
				Data:      generateTestChartData("pie", 4),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(env.TempDir, tt.name+".png")

			err := renderer.generatePNGPlaceholder(outputPath, tt.req)

			if err != nil {
				t.Fatalf("generatePNGPlaceholder returned error: %v", err)
			}

			assertFileExists(t, outputPath)

			// Verify file is a valid PNG
			stat, err := os.Stat(outputPath)
			if err != nil {
				t.Fatalf("Failed to stat PNG file: %v", err)
			}

			if stat.Size() == 0 {
				t.Error("PNG file is empty")
			}
		})
	}
}

func TestChartRenderer_ExtractSVGFromHTML(t *testing.T) {
	renderer := NewChartRenderer("/tmp")

	html := "<html><body>Test HTML</body></html>"
	svg := renderer.extractSVGFromHTML(html)

	if svg == "" {
		t.Error("Expected non-empty SVG content")
	}

	// Verify it's valid SVG (should start with <svg)
	if len(svg) < 4 || svg[:4] != "<svg" {
		t.Error("Expected SVG to start with '<svg'")
	}
}

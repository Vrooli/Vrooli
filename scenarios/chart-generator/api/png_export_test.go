package main

import (
	"testing"
)

// [REQ:CHART-P0-005-PNG] Export capabilities: PNG format
func TestPNGExport_RenderChart(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	renderer := NewChartRenderer(env.TempDir)

	tests := []struct {
		name        string
		chartID     string
		req         ChartGenerationProcessorRequest
		expectError bool
	}{
		{
			name:    "RenderBarChartPNG",
			chartID: "test-bar-png-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"png"},
				Width:         800,
				Height:        600,
				Title:         "Test Bar Chart PNG",
			},
			expectError: false,
		},
		{
			name:    "RenderLineChartPNG",
			chartID: "test-line-png-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 10),
				ExportFormats: []string{"png"},
				Width:         1024,
				Height:        768,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := renderer.RenderChart(tt.chartID, tt.req)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if len(files) == 0 {
					t.Error("Expected at least one file to be generated")
				}
				for format, path := range files {
					assertFileExists(t, path)
					t.Logf("Generated %s file: %s", format, path)
				}
			}
		})
	}
}

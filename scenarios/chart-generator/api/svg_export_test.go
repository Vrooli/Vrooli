package main

import (
	"testing"
)

// [REQ:CHART-P0-005-SVG] Export capabilities: SVG format
func TestSVGExport_RenderChart(t *testing.T) {
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
			name:    "RenderBarChartSVG",
			chartID: "test-bar-svg-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				ExportFormats: []string{"svg"},
				Width:         800,
				Height:        600,
			},
			expectError: false,
		},
		{
			name:    "RenderPieChartSVG",
			chartID: "test-pie-svg-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "pie",
				Data:          generateTestChartData("pie", 4),
				ExportFormats: []string{"svg"},
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

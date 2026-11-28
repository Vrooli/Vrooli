package main

import "testing"

// [REQ:CHART-P0-005-PNG] Export capabilities: PNG format
func TestPNGExport_RenderChart(t *testing.T) {
	tests := []chartExportCase{
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
		},
	}

	runChartExportCases(t, tests)
}

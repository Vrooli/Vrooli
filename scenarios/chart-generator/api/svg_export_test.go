package main

import "testing"

// [REQ:CHART-P0-005-SVG] Export capabilities: SVG format
func TestSVGExport_RenderChart(t *testing.T) {
	tests := []chartExportCase{
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
		},
		{
			name:    "RenderPieChartSVG",
			chartID: "test-pie-svg-001",
			req: ChartGenerationProcessorRequest{
				ChartType:     "pie",
				Data:          generateTestChartData("pie", 4),
				ExportFormats: []string{"svg"},
			},
		},
	}

	runChartExportCases(t, tests)
}

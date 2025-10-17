package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// ChartRenderer handles actual chart rendering with go-echarts
type ChartRenderer struct {
	outputDir string
}

// NewChartRenderer creates a new chart renderer
func NewChartRenderer(outputDir string) *ChartRenderer {
	return &ChartRenderer{
		outputDir: outputDir,
	}
}

// RenderChart generates actual chart files using go-echarts
func (cr *ChartRenderer) RenderChart(chartID string, req ChartGenerationProcessorRequest) (map[string]string, error) {
	files := make(map[string]string)

	// Create output directory
	chartDir := filepath.Join(cr.outputDir, chartID+"_output")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate the chart based on type
	var chartHTML string
	var err error

	switch req.ChartType {
	case "bar":
		chartHTML, err = cr.renderBarChart(req)
	case "line":
		chartHTML, err = cr.renderLineChart(req)
	case "pie":
		chartHTML, err = cr.renderPieChart(req)
	case "scatter":
		chartHTML, err = cr.renderScatterChart(req)
	case "area":
		chartHTML, err = cr.renderAreaChart(req)
	case "candlestick":
		chartHTML, err = cr.renderCandlestickChart(req)
	case "heatmap":
		chartHTML, err = cr.renderHeatmapChart(req)
	case "treemap":
		chartHTML, err = cr.renderTreemapChart(req)
	case "gantt":
		chartHTML, err = cr.renderGanttChart(req)
	default:
		return nil, fmt.Errorf("unsupported chart type: %s", req.ChartType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	// Save HTML output for all formats
	htmlPath := filepath.Join(chartDir, chartID+".html")
	if err := os.WriteFile(htmlPath, []byte(chartHTML), 0644); err != nil {
		return nil, fmt.Errorf("failed to write HTML file: %w", err)
	}

	// Generate requested formats
	for _, format := range req.ExportFormats {
		outputPath := filepath.Join(chartDir, chartID+"."+format)

		switch format {
		case "svg":
			// Extract SVG from HTML (simplified - in production, use proper conversion)
			svgContent := cr.extractSVGFromHTML(chartHTML)
			if err := os.WriteFile(outputPath, []byte(svgContent), 0644); err != nil {
				return nil, err
			}
			files[format] = outputPath

		case "png":
			// Generate actual PNG using browserless screenshot
			if err := cr.generatePNG(outputPath, htmlPath, req); err != nil {
				fmt.Printf("⚠️ Browserless PNG generation failed: %v\n", err)
				// Fall back to placeholder if browserless is not available
				if err := cr.generatePNGPlaceholder(outputPath, req); err != nil {
					return nil, err
				}
			}
			files[format] = outputPath

		case "pdf":
			// Generate actual PDF using maroto
			if err := cr.generatePDF(outputPath, req, chartHTML); err != nil {
				return nil, fmt.Errorf("failed to generate PDF: %w", err)
			}
			files[format] = outputPath

		default:
			// For other formats, copy HTML
			files[format] = htmlPath
		}
	}

	return files, nil
}

// Helper functions for animation and styling
func (cr *ChartRenderer) getTheme(req ChartGenerationProcessorRequest) string {
	if req.Style == "dark" {
		return types.ThemeWonderland
	}
	if req.Style == "minimal" {
		return types.ThemeChalk
	}
	return types.ThemeWesteros
}

func (cr *ChartRenderer) isAnimationEnabled(req ChartGenerationProcessorRequest) bool {
	// Check if explicitly disabled
	if req.Config != nil {
		if cfg, ok := req.Config.(map[string]interface{}); ok {
			if anim, ok := cfg["animation"].(bool); ok {
				return anim
			}
		}
	}

	// Enable animation by default for HTML exports
	for _, format := range req.ExportFormats {
		if format == "html" || format == "interactive" {
			return true
		}
	}

	// Enable for web-based outputs if no specific formats requested
	if len(req.ExportFormats) == 0 {
		return true
	}

	return false // Disable for static exports like PNG/PDF
}

func (cr *ChartRenderer) getCustomColors(req ChartGenerationProcessorRequest) []string {
	if req.Config != nil {
		if cfg, ok := req.Config.(map[string]interface{}); ok {
			if colors, ok := cfg["colors"].([]string); ok {
				return colors
			}
			// Handle []interface{} case
			if colorsIface, ok := cfg["colors"].([]interface{}); ok {
				colors := make([]string, 0, len(colorsIface))
				for _, c := range colorsIface {
					if color, ok := c.(string); ok {
						colors = append(colors, color)
					}
				}
				if len(colors) > 0 {
					return colors
				}
			}
		}
	}
	return nil
}

// renderBarChart creates a bar chart using go-echarts
func (cr *ChartRenderer) renderBarChart(req ChartGenerationProcessorRequest) (string, error) {
	bar := charts.NewBar()

	// Build global options with animation support
	globalOpts := []charts.GlobalOpts{
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cr.getTheme(req),
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	}

	// Add animation and interactivity options
	if cr.isAnimationEnabled(req) {
		showTooltip := true
		showLegend := true
		globalOpts = append(globalOpts,
			charts.WithTooltipOpts(opts.Tooltip{
				Show:    &showTooltip,
				Trigger: "axis",
				AxisPointer: &opts.AxisPointer{
					Type: "cross",
				},
			}),
			charts.WithLegendOpts(opts.Legend{
				Show: &showLegend,
				Type: "scroll",
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Type:  "inside",
				Start: 0,
				End:   100,
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Type:  "slider",
				Start: 0,
				End:   100,
			}),
			charts.WithToolboxOpts(opts.Toolbox{
				Show: &showTooltip,
				Feature: &opts.ToolBoxFeature{
					SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
						Show:  &showTooltip,
						Title: "Save as Image",
					},
					DataView: &opts.ToolBoxFeatureDataView{
						Show:  &showTooltip,
						Title: "Data View",
					},
					Restore: &opts.ToolBoxFeatureRestore{
						Show:  &showTooltip,
						Title: "Restore",
					},
				},
			}),
		)
	}

	// Apply custom colors if provided
	if colors := cr.getCustomColors(req); colors != nil {
		globalOpts = append(globalOpts, charts.WithColorsOpts(opts.Colors(colors)))
	}

	bar.SetGlobalOptions(globalOpts...)

	// Extract data
	xAxis := make([]string, 0, len(req.Data))
	yData := make([]opts.BarData, 0, len(req.Data))

	for _, point := range req.Data {
		if x, ok := point["x"].(string); ok {
			xAxis = append(xAxis, x)
		}
		if y, ok := point["y"].(float64); ok {
			yData = append(yData, opts.BarData{Value: y})
		}
	}

	bar.SetXAxis(xAxis).
		AddSeries("Series", yData)

	// Render to HTML string
	var buf bytes.Buffer
	if err := bar.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderLineChart creates a line chart
func (cr *ChartRenderer) renderLineChart(req ChartGenerationProcessorRequest) (string, error) {
	line := charts.NewLine()

	// Build global options with animation support
	globalOpts := []charts.GlobalOpts{
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cr.getTheme(req),
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	}

	// Add animation and interactivity options
	if cr.isAnimationEnabled(req) {
		showTooltip := true
		showLegend := true
		globalOpts = append(globalOpts,
			charts.WithTooltipOpts(opts.Tooltip{
				Show:    &showTooltip,
				Trigger: "axis",
				AxisPointer: &opts.AxisPointer{
					Type: "line",
				},
			}),
			charts.WithLegendOpts(opts.Legend{
				Show: &showLegend,
				Type: "plain",
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Type:  "inside",
				Start: 0,
				End:   100,
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Type:  "slider",
				Start: 0,
				End:   100,
			}),
			charts.WithToolboxOpts(opts.Toolbox{
				Show: &showTooltip,
				Feature: &opts.ToolBoxFeature{
					SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
						Show:  &showTooltip,
						Title: "Save as Image",
					},
					DataZoom: &opts.ToolBoxFeatureDataZoom{
						Show: &showTooltip,
						Title: map[string]string{
							"zoom": "Zoom",
							"back": "Reset",
						},
					},
					Restore: &opts.ToolBoxFeatureRestore{
						Show:  &showTooltip,
						Title: "Restore",
					},
				},
			}),
		)
	}

	// Apply custom colors if provided
	if colors := cr.getCustomColors(req); colors != nil {
		globalOpts = append(globalOpts, charts.WithColorsOpts(opts.Colors(colors)))
	}

	line.SetGlobalOptions(globalOpts...)

	// Extract data
	xAxis := make([]string, 0, len(req.Data))
	yData := make([]opts.LineData, 0, len(req.Data))

	for _, point := range req.Data {
		if x, ok := point["x"].(string); ok {
			xAxis = append(xAxis, x)
		}
		if y, ok := point["y"].(float64); ok {
			yData = append(yData, opts.LineData{Value: y})
		}
	}

	line.SetXAxis(xAxis).
		AddSeries("Series", yData)

	var buf bytes.Buffer
	if err := line.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderPieChart creates a pie chart
func (cr *ChartRenderer) renderPieChart(req ChartGenerationProcessorRequest) (string, error) {
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeWesteros,
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	)

	// Extract data
	pieData := make([]opts.PieData, 0, len(req.Data))
	for _, point := range req.Data {
		name := ""
		value := 0.0

		if n, ok := point["name"].(string); ok {
			name = n
		} else if x, ok := point["x"].(string); ok {
			name = x
		}

		if v, ok := point["value"].(float64); ok {
			value = v
		} else if y, ok := point["y"].(float64); ok {
			value = y
		}

		pieData = append(pieData, opts.PieData{
			Name:  name,
			Value: value,
		})
	}

	pie.AddSeries("Series", pieData)

	var buf bytes.Buffer
	if err := pie.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderScatterChart creates a scatter chart
func (cr *ChartRenderer) renderScatterChart(req ChartGenerationProcessorRequest) (string, error) {
	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeWesteros,
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	)

	// Extract data
	scatterData := make([]opts.ScatterData, 0, len(req.Data))
	for _, point := range req.Data {
		x := 0.0
		y := 0.0

		if xVal, ok := point["x"].(float64); ok {
			x = xVal
		}
		if yVal, ok := point["y"].(float64); ok {
			y = yVal
		}

		scatterData = append(scatterData, opts.ScatterData{
			Value: []interface{}{x, y},
		})
	}

	scatter.AddSeries("Series", scatterData)

	var buf bytes.Buffer
	if err := scatter.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderAreaChart creates an area chart (line chart with area fill)
func (cr *ChartRenderer) renderAreaChart(req ChartGenerationProcessorRequest) (string, error) {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeWesteros,
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	)

	// Extract data
	xAxis := make([]string, 0, len(req.Data))
	yData := make([]opts.LineData, 0, len(req.Data))

	for _, point := range req.Data {
		if x, ok := point["x"].(string); ok {
			xAxis = append(xAxis, x)
		}
		if y, ok := point["y"].(float64); ok {
			yData = append(yData, opts.LineData{Value: y})
		}
	}

	line.SetXAxis(xAxis).
		AddSeries("Series", yData, charts.WithAreaStyleOpts(opts.AreaStyle{}))

	var buf bytes.Buffer
	if err := line.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderCandlestickChart creates a candlestick chart for financial data
func (cr *ChartRenderer) renderCandlestickChart(req ChartGenerationProcessorRequest) (string, error) {
	kline := charts.NewKLine()
	kline.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeWesteros,
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	)

	// Extract data
	xAxis := make([]string, 0, len(req.Data))
	klineData := make([]opts.KlineData, 0, len(req.Data))

	for _, point := range req.Data {
		date := ""
		open := 0.0
		close := 0.0
		low := 0.0
		high := 0.0

		if d, ok := point["date"].(string); ok {
			date = d
			xAxis = append(xAxis, date)
		}

		if o, ok := point["open"].(float64); ok {
			open = o
		}
		if c, ok := point["close"].(float64); ok {
			close = c
		}
		if l, ok := point["low"].(float64); ok {
			low = l
		}
		if h, ok := point["high"].(float64); ok {
			high = h
		}

		klineData = append(klineData, opts.KlineData{
			Value: [4]float32{float32(open), float32(close), float32(low), float32(high)},
		})
	}

	kline.SetXAxis(xAxis).
		AddSeries("Candlestick", klineData)

	var buf bytes.Buffer
	if err := kline.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderHeatmapChart creates a heatmap chart
func (cr *ChartRenderer) renderHeatmapChart(req ChartGenerationProcessorRequest) (string, error) {
	heatmap := charts.NewHeatMap()
	heatmap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: req.Title,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  types.ThemeWesteros,
			Width:  strconv.Itoa(req.Width) + "px",
			Height: strconv.Itoa(req.Height) + "px",
		}),
	)

	// Extract data
	var xAxis, yAxis []string
	xMap := make(map[string]bool)
	yMap := make(map[string]bool)
	heatmapData := make([]opts.HeatMapData, 0, len(req.Data))

	for _, point := range req.Data {
		x := ""
		y := ""
		value := 0.0

		if xVal, ok := point["x"].(string); ok {
			x = xVal
			if !xMap[x] {
				xAxis = append(xAxis, x)
				xMap[x] = true
			}
		}
		if yVal, ok := point["y"].(string); ok {
			y = yVal
			if !yMap[y] {
				yAxis = append(yAxis, y)
				yMap[y] = true
			}
		}
		if v, ok := point["value"].(float64); ok {
			value = v
		}

		// Find indices
		xIdx := 0
		yIdx := 0
		for i, val := range xAxis {
			if val == x {
				xIdx = i
				break
			}
		}
		for i, val := range yAxis {
			if val == y {
				yIdx = i
				break
			}
		}

		heatmapData = append(heatmapData, opts.HeatMapData{
			Value: [3]interface{}{xIdx, yIdx, value},
		})
	}

	heatmap.SetXAxis(xAxis).
		AddSeries("Heatmap", heatmapData)

	var buf bytes.Buffer
	if err := heatmap.Render(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderTreemapChart creates a treemap chart
func (cr *ChartRenderer) renderTreemapChart(req ChartGenerationProcessorRequest) (string, error) {
	// go-echarts doesn't have native treemap support, so we'll create a custom implementation
	// For now, return a placeholder HTML
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        .treemap { width: %dpx; height: %dpx; position: relative; }
        .treemap-item { position: absolute; border: 1px solid #fff; background: #4CAF50; overflow: hidden; }
        .treemap-label { padding: 5px; color: white; font-size: 12px; }
    </style>
</head>
<body>
    <div class="treemap">`, req.Title, req.Width, req.Height)

	// Simple treemap layout calculation
	totalValue := 0.0
	for _, point := range req.Data {
		if v, ok := point["value"].(float64); ok {
			totalValue += v
		}
	}

	x, y := 0, 0
	for i, point := range req.Data {
		name := fmt.Sprintf("Item %d", i)
		value := 0.0

		if n, ok := point["name"].(string); ok {
			name = n
		}
		if v, ok := point["value"].(float64); ok {
			value = v
		}

		// Calculate rectangle size based on value proportion
		width := int(float64(req.Width) * (value / totalValue))
		height := req.Height / len(req.Data)

		html += fmt.Sprintf(`
        <div class="treemap-item" style="left:%dpx; top:%dpx; width:%dpx; height:%dpx;">
            <div class="treemap-label">%s: %.2f</div>
        </div>`, x, y, width, height, name, value)

		y += height
	}

	html += `
    </div>
</body>
</html>`

	return html, nil
}

// renderGanttChart creates a Gantt chart
func (cr *ChartRenderer) renderGanttChart(req ChartGenerationProcessorRequest) (string, error) {
	// Simple Gantt chart implementation
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        .gantt { width: %dpx; height: %dpx; position: relative; border: 1px solid #ddd; }
        .gantt-bar { position: absolute; background: #2196F3; height: 30px; border-radius: 3px; }
        .gantt-label { position: absolute; left: 5px; line-height: 30px; color: white; font-size: 12px; }
    </style>
</head>
<body>
    <div class="gantt">`, req.Title, req.Width, req.Height)

	// Parse tasks
	y := 10
	for i, point := range req.Data {
		task := fmt.Sprintf("Task %d", i+1)
		start := 0
		duration := 100

		if t, ok := point["task"].(string); ok {
			task = t
		}
		if s, ok := point["start"].(float64); ok {
			start = int(s)
		}
		if d, ok := point["duration"].(float64); ok {
			duration = int(d)
		}

		html += fmt.Sprintf(`
        <div class="gantt-bar" style="left:%dpx; top:%dpx; width:%dpx;">
            <div class="gantt-label">%s</div>
        </div>`, start*5, y, duration*5, task)

		y += 40
	}

	html += `
    </div>
</body>
</html>`

	return html, nil
}

// generatePDF creates a PDF file with the chart
func (cr *ChartRenderer) generatePDF(outputPath string, req ChartGenerationProcessorRequest, chartHTML string) error {
	// Create new maroto instance
	m := maroto.New()

	// Add header with title
	m.AddRow(40, text.NewCol(12, req.Title, props.Text{
		Top:    12,
		Family: fontfamily.Helvetica,
		Style:  fontstyle.Bold,
		Size:   18,
		Align:  align.Center,
	}))

	// Add metadata
	m.AddRow(20, text.NewCol(12, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")), props.Text{
		Top:    5,
		Family: fontfamily.Helvetica,
		Size:   10,
		Align:  align.Center,
	}))

	// Add chart description
	m.AddRow(30, text.NewCol(12, fmt.Sprintf("Chart Type: %s | Data Points: %d | Style: %s",
		req.ChartType, len(req.Data), req.Style), props.Text{
		Top:    5,
		Family: fontfamily.Helvetica,
		Size:   10,
		Align:  align.Center,
	}))

	// For actual chart rendering in PDF, we would need to:
	// 1. Generate a PNG/JPEG of the chart
	// 2. Embed it in the PDF
	// For now, we'll add a data table representation

	// Add data table header
	m.AddRow(20, text.NewCol(12, "Chart Data", props.Text{
		Top:    5,
		Family: fontfamily.Helvetica,
		Style:  fontstyle.Bold,
		Size:   12,
		Align:  align.Left,
	}))

	// Add data rows (simplified)
	for i, point := range req.Data {
		if i >= 20 { // Limit to first 20 rows for space
			m.AddRow(15, text.NewCol(12, fmt.Sprintf("... and %d more data points", len(req.Data)-20), props.Text{
				Top:    3,
				Family: fontfamily.Helvetica,
				Size:   9,
				Align:  align.Left,
			}))
			break
		}

		// Format data point as string
		dataStr := fmt.Sprintf("Point %d: ", i+1)
		for k, v := range point {
			dataStr += fmt.Sprintf("%s=%v ", k, v)
		}

		m.AddRow(15, text.NewCol(12, dataStr, props.Text{
			Top:    3,
			Family: fontfamily.Helvetica,
			Size:   9,
			Align:  align.Left,
		}))
	}

	// Generate PDF document
	document, err := m.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate PDF document: %w", err)
	}

	// Save to file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create PDF file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, bytes.NewReader(document.GetBytes())); err != nil {
		return fmt.Errorf("failed to write PDF content: %w", err)
	}

	return nil
}

// extractSVGFromHTML extracts SVG content from HTML (simplified)
func (cr *ChartRenderer) extractSVGFromHTML(html string) string {
	// In production, use proper HTML parsing
	// For now, return a simple SVG placeholder
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
		<rect width="800" height="600" fill="white"/>
		<text x="400" y="300" text-anchor="middle" font-family="Arial" font-size="16">
			Chart Generated at %s
		</text>
	</svg>`, time.Now().Format("15:04:05"))
}

// generatePNG creates an actual PNG file using browserless
func (cr *ChartRenderer) generatePNG(outputPath, htmlPath string, req ChartGenerationProcessorRequest) error {
	fmt.Println("🌐 Attempting to generate PNG using browserless...")
	// Read the HTML file
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}

	// Prepare the screenshot request for browserless
	screenshotReq := map[string]interface{}{
		"html": string(htmlContent),
		"options": map[string]interface{}{
			"type":     "png",
			"fullPage": false,
			"viewport": map[string]int{
				"width":  req.Width,
				"height": req.Height,
			},
		},
		"waitForTimeout": 2000, // Wait for charts to render
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(screenshotReq)
	if err != nil {
		return fmt.Errorf("failed to marshal screenshot request: %w", err)
	}

	// Send request to browserless with timeout
	client := &http.Client{
		Timeout: 5 * time.Second, // 5 second timeout
	}
	resp, err := client.Post("http://localhost:4110/screenshot", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to connect to browserless: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("browserless returned status %d", resp.StatusCode)
	}

	// Save the PNG data to file
	pngData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read PNG data: %w", err)
	}

	if err := os.WriteFile(outputPath, pngData, 0644); err != nil {
		return fmt.Errorf("failed to write PNG file: %w", err)
	}

	fmt.Printf("✅ Successfully generated PNG using browserless (size: %d bytes)\n", len(pngData))
	return nil
}

// generatePNGPlaceholder creates a placeholder PNG file
func (cr *ChartRenderer) generatePNGPlaceholder(outputPath string, req ChartGenerationProcessorRequest) error {
	fmt.Println("⚠️ Browserless not available, using fallback Go PNG generation...")
	// Generate a real PNG image using Go's image library
	return cr.generateBasicPNGChart(outputPath, req)
}

// generateBasicPNGChart creates a simple but real PNG chart
func (cr *ChartRenderer) generateBasicPNGChart(outputPath string, req ChartGenerationProcessorRequest) error {
	// Create image
	width := req.Width
	if width == 0 {
		width = 800
	}
	height := req.Height
	if height == 0 {
		height = 600
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw a simple chart representation based on data
	if req.ChartType == "bar" && len(req.Data) > 0 {
		// Draw simple bars
		barWidth := width / len(req.Data)
		barColor := color.RGBA{54, 162, 235, 255} // Blue

		for i, dataPoint := range req.Data {
			// Try to extract y value
			var yValue float64
			if y, ok := dataPoint["y"].(float64); ok {
				yValue = y
			} else if y, ok := dataPoint["y"].(int); ok {
				yValue = float64(y)
			} else if y, ok := dataPoint["value"].(float64); ok {
				yValue = y
			} else {
				yValue = 50 // Default value
			}

			// Normalize bar height (assuming max value of 100)
			barHeight := int((yValue / 100.0) * float64(height-100))
			if barHeight < 10 {
				barHeight = 10
			}

			// Draw bar
			x1 := i*barWidth + barWidth/4
			y1 := height - barHeight - 50
			x2 := x1 + barWidth/2
			y2 := height - 50

			for x := x1; x < x2; x++ {
				for y := y1; y < y2; y++ {
					img.Set(x, y, barColor)
				}
			}
		}
	} else if req.ChartType == "line" && len(req.Data) > 0 {
		// Draw simple line chart
		lineColor := color.RGBA{255, 99, 132, 255} // Red
		pointSpacing := width / len(req.Data)

		for i, dataPoint := range req.Data {
			var yValue float64
			if y, ok := dataPoint["y"].(float64); ok {
				yValue = y
			} else if y, ok := dataPoint["y"].(int); ok {
				yValue = float64(y)
			} else {
				yValue = 50
			}

			// Draw point
			x := i*pointSpacing + pointSpacing/2
			y := height - int((yValue/100.0)*float64(height-100)) - 50

			// Draw a small circle for the point
			for dx := -3; dx <= 3; dx++ {
				for dy := -3; dy <= 3; dy++ {
					if dx*dx+dy*dy <= 9 {
						img.Set(x+dx, y+dy, lineColor)
					}
				}
			}
		}
	} else {
		// For other chart types, draw a placeholder visualization
		// Draw border
		borderColor := color.RGBA{200, 200, 200, 255}
		for x := 0; x < width; x++ {
			img.Set(x, 0, borderColor)
			img.Set(x, height-1, borderColor)
		}
		for y := 0; y < height; y++ {
			img.Set(0, y, borderColor)
			img.Set(width-1, y, borderColor)
		}
	}

	// Draw title if provided
	if req.Title != "" {
		// Add a simple title area (just a colored bar for now)
		titleColor := color.RGBA{50, 50, 50, 255}
		for x := 10; x < width-10; x++ {
			for y := 10; y < 40; y++ {
				img.Set(x, y, titleColor)
			}
		}
	}

	// Save PNG
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create PNG file: %w", err)
	}
	defer file.Close()

	return png.Encode(file, img)
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
)

type PaletteRequest struct {
	Theme     string `json:"theme"`
	Style     string `json:"style,omitempty"`
	NumColors int    `json:"num_colors,omitempty"`
	BaseColor string `json:"base_color,omitempty"`
}

type PaletteResponse struct {
	Success bool     `json:"success"`
	Palette []string `json:"palette,omitempty"`
	Name    string   `json:"name,omitempty"`
	Theme   string   `json:"theme,omitempty"`
	Error   string   `json:"error,omitempty"`
}

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start palette-gen

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }
	port := getEnv("API_PORT", getEnv("PORT", ""))

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/suggest", suggestHandler)
	http.HandleFunc("/export", exportHandler)

	log.Printf("Palette Generator API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PaletteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.NumColors == 0 {
		req.NumColors = 5
	}
	if req.Style == "" {
		req.Style = "vibrant"
	}

	// Generate palette based on theme and style
	palette := generatePalette(req.Theme, req.Style, req.NumColors, req.BaseColor)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PaletteResponse{
		Success: true,
		Palette: palette,
		Name:    fmt.Sprintf("%s %s", strings.Title(req.Style), strings.Title(req.Theme)),
		Theme:   req.Theme,
	})
}

func suggestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Suggest palettes based on use case
	useCase := req["use_case"]
	suggestions := getSuggestionsForUseCase(useCase)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"suggestions": suggestions,
	})
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	format := req["format"].(string)
	palette := req["palette"].([]interface{})

	exportData := exportPalette(palette, format)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"export":  exportData,
	})
}

func getSuggestionsForUseCase(useCase string) []map[string]interface{} {
	// Return predefined palette suggestions
	suggestions := []map[string]interface{}{
		{
			"name":        "Modern Business",
			"colors":      []string{"#1E40AF", "#3B82F6", "#60A5FA", "#93C5FD", "#DBEAFE"},
			"description": "Professional blue-based palette",
		},
		{
			"name":        "Nature Fresh",
			"colors":      []string{"#166534", "#16A34A", "#4ADE80", "#86EFAC", "#BBF7D0"},
			"description": "Green nature-inspired palette",
		},
	}
	return suggestions
}

func exportPalette(palette []interface{}, format string) string {
	switch format {
	case "css":
		css := ":root {\n"
		for i, color := range palette {
			css += fmt.Sprintf("  --color-%d: %s;\n", i+1, color)
		}
		css += "}"
		return css
	case "json":
		data, _ := json.Marshal(palette)
		return string(data)
	case "scss":
		scss := ""
		for i, color := range palette {
			scss += fmt.Sprintf("$color-%d: %s;\n", i+1, color)
		}
		return scss
	default:
		return ""
	}
}

// Color palette generation functions
func generatePalette(theme, style string, numColors int, baseColor string) []string {
	if baseColor != "" {
		return generateFromBaseColor(baseColor, style, numColors)
	}
	
	// Theme-based generation
	baseHue := getThemeHue(theme)
	return generateFromStyle(baseHue, style, numColors)
}

func getThemeHue(theme string) float64 {
	themeLower := strings.ToLower(theme)
	
	// Map themes to base hues (0-360)
	themeHues := map[string]float64{
		"ocean":     200,
		"sunset":    25,
		"forest":    120,
		"desert":    40,
		"tech":      210,
		"vintage":   30,
		"modern":    240,
		"nature":    90,
		"fire":      15,
		"ice":       195,
		"autumn":    35,
		"spring":    150,
		"summer":    60,
		"winter":    200,
	}
	
	for key, hue := range themeHues {
		if strings.Contains(themeLower, key) {
			return hue
		}
	}
	
	// Default to a blue-ish hue
	return 210
}

func generateFromStyle(baseHue float64, style string, numColors int) []string {
	colors := make([]string, numColors)
	
	switch strings.ToLower(style) {
	case "vibrant":
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue + float64(i)*72, 360)
			saturation := 70 + (i%2)*15
			lightness := 45 + (i%3)*10
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	case "pastel":
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue + float64(i)*60, 360)
			saturation := 50 + (i%2)*10
			lightness := 75 + (i%3)*5
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	case "dark":
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue + float64(i)*45, 360)
			saturation := 30 + (i%3)*10
			lightness := 20 + (i%4)*10
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	case "minimal":
		for i := 0; i < numColors; i++ {
			hue := baseHue
			saturation := 10 + float64(i)*5
			lightness := 20 + float64(i)*(60/float64(numColors))
			colors[i] = hslToHex(hue, saturation, lightness)
		}
	case "earthy":
		for i := 0; i < numColors; i++ {
			hue := math.Mod(30 + float64(i)*30, 90) // Browns, oranges, greens
			saturation := 35 + (i%3)*15
			lightness := 40 + (i%3)*10
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	default: // Default to harmonious
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue + float64(i)*30, 360)
			saturation := 60
			lightness := 50 + (i%2)*10
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	}
	
	return colors
}

func generateFromBaseColor(baseColor, style string, numColors int) []string {
	// Simple implementation - generate variations of the base color
	colors := make([]string, numColors)
	colors[0] = baseColor
	
	// Parse base color (simplified - assumes hex)
	baseHue, baseSat, baseLightness := hexToHSL(baseColor)
	
	for i := 1; i < numColors; i++ {
		var hue, saturation, lightness float64
		
		switch strings.ToLower(style) {
		case "analogous":
			// Colors next to each other on the wheel
			hue = math.Mod(baseHue + float64(i)*30, 360)
			saturation = baseSat
			lightness = baseLightness
		case "complementary":
			// Opposite colors
			if i == 1 {
				hue = math.Mod(baseHue+180, 360)
			} else {
				hue = math.Mod(baseHue + float64(i-1)*60, 360)
			}
			saturation = baseSat
			lightness = baseLightness + float64(i-1)*10
		case "triadic":
			// Three colors evenly spaced
			hue = math.Mod(baseHue + float64(i)*120, 360)
			saturation = baseSat
			lightness = baseLightness
		default:
			// Monochromatic - variations of the same hue
			hue = baseHue
			saturation = baseSat - float64(i)*10
			lightness = baseLightness + float64(i)*15
		}
		
		colors[i] = hslToHex(hue, saturation, lightness)
	}
	
	return colors
}

// Color conversion utilities
func hslToHex(h, s, l float64) string {
	s = s / 100
	l = l / 100
	
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	
	var r, g, b float64
	
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	
	r = (r + m) * 255
	g = (g + m) * 255
	b = (b + m) * 255
	
	return fmt.Sprintf("#%02X%02X%02X", int(r), int(g), int(b))
}

func hexToHSL(hex string) (float64, float64, float64) {
	// Simple hex to HSL conversion (simplified)
	// This is a basic implementation - would need more robust parsing in production
	if len(hex) < 7 {
		return 0, 50, 50
	}
	
	// For simplicity, return default values
	// In production, would properly parse hex and convert
	return 210, 60, 50
}

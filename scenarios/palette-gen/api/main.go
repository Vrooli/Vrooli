package main

import (
	"github.com/vrooli/api-core/preflight"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"
)

type PaletteRequest struct {
	Theme          string `json:"theme"`
	Style          string `json:"style,omitempty"`
	NumColors      int    `json:"num_colors,omitempty"`
	BaseColor      string `json:"base_color,omitempty"`
	IncludeAIDebug bool   `json:"include_ai_debug,omitempty"`
}

type PaletteResponse struct {
	Success     bool       `json:"success"`
	Palette     []string   `json:"palette,omitempty"`
	Name        string     `json:"name,omitempty"`
	Theme       string     `json:"theme,omitempty"`
	Error       string     `json:"error,omitempty"`
	Style       string     `json:"style,omitempty"`
	Description string     `json:"description,omitempty"`
	Debug       *DebugInfo `json:"debug,omitempty"`
}

type DebugInfo struct {
	Strategy           string                   `json:"strategy"`
	BaseHue            float64                  `json:"base_hue,omitempty"`
	RequestedTheme     string                   `json:"requested_theme,omitempty"`
	RequestedStyle     string                   `json:"requested_style,omitempty"`
	ResolvedStyle      string                   `json:"resolved_style,omitempty"`
	RequestedBaseColor string                   `json:"requested_base_color,omitempty"`
	RequestedColors    int                      `json:"requested_colors,omitempty"`
	AIRequested        bool                     `json:"ai_requested"`
	AIAvailable        bool                     `json:"ai_available"`
	AIPrompt           string                   `json:"ai_prompt,omitempty"`
	AIModel            string                   `json:"ai_model,omitempty"`
	AIRawOutput        string                   `json:"ai_raw_output,omitempty"`
	AISuggestions      []map[string]interface{} `json:"ai_suggestions,omitempty"`
	AIError            string                   `json:"ai_error,omitempty"`
	AIDurationMs       int64                    `json:"ai_duration_ms,omitempty"`
}

type AISuggestionResult struct {
	Suggestions []map[string]interface{}
	RawResponse string
	Prompt      string
	Model       string
	DurationMs  int64
	Error       string
}

type AccessibilityRequest struct {
	Foreground string `json:"foreground"`
	Background string `json:"background"`
}

type AccessibilityResponse struct {
	Success        bool    `json:"success"`
	ContrastRatio  float64 `json:"contrast_ratio"`
	WCAGAA         bool    `json:"wcag_aa"`
	WCAGAAA        bool    `json:"wcag_aaa"`
	LargeTextAA    bool    `json:"large_text_aa"`
	LargeTextAAA   bool    `json:"large_text_aaa"`
	Recommendation string  `json:"recommendation,omitempty"`
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

type HarmonyRequest struct {
	Colors []string `json:"colors"`
}

type HarmonyResponse struct {
	Success      bool                   `json:"success"`
	Analysis     map[string]interface{} `json:"analysis"`
	IsHarmonious bool                   `json:"is_harmonious"`
	Score        float64                `json:"score"`
}

type ColorblindRequest struct {
	Colors []string `json:"colors"`
	Type   string   `json:"type"` // protanopia, deuteranopia, tritanopia
}

type ColorblindResponse struct {
	Success   bool     `json:"success"`
	Simulated []string `json:"simulated"`
	Type      string   `json:"type"`
}

var (
	redisClient *redis.Client
	ctx         = context.Background()
	logger      *slog.Logger
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "palette-gen",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize structured logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := getEnv("API_PORT", getEnv("PORT", ""))

	// Initialize Redis if available
	initRedis()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/suggest", suggestHandler)
	http.HandleFunc("/export", exportHandler)
	http.HandleFunc("/accessibility", accessibilityHandler)
	http.HandleFunc("/harmony", harmonyHandler)
	http.HandleFunc("/colorblind", colorblindHandler)
	http.HandleFunc("/history", historyHandler)

	logger.Info("Palette Generator API starting", "port", port, "service", "palette-gen-api")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func initRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		// Redis is optional, skip if not configured
		logger.Info("Redis not configured, caching disabled", "reason", "REDIS_PORT missing")
		redisClient = nil
		return
	}

	redisAddr := redisHost + ":" + redisPort

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Test connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		logger.Warn("Redis not available, caching disabled", "error", err, "address", redisAddr)
		redisClient = nil
	} else {
		logger.Info("Redis connection established", "address", redisAddr, "status", "connected")
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

	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "palette-gen-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": true,
	}

	// Check Redis connectivity if enabled
	if redisClient != nil {
		startTime := time.Now()
		_, err := redisClient.Ping(ctx).Result()
		latency := time.Since(startTime).Milliseconds()

		redisHealth := map[string]interface{}{
			"connected": err == nil,
		}

		if err == nil {
			redisHealth["latency_ms"] = latency
			redisHealth["error"] = nil
		} else {
			redisHealth["latency_ms"] = nil
			redisHealth["error"] = map[string]interface{}{
				"code":      "CONNECTION_FAILED",
				"message":   err.Error(),
				"category":  "network",
				"retryable": true,
			}
			health["status"] = "degraded"
		}

		health["dependencies"] = map[string]interface{}{
			"redis": redisHealth,
		}
	}

	json.NewEncoder(w).Encode(health)
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

	req.Theme = strings.TrimSpace(req.Theme)
	req.Style = strings.TrimSpace(req.Style)
	requestedStyle := req.Style
	if strings.EqualFold(req.Style, "auto") {
		req.Style = ""
		requestedStyle = "auto"
	}
	req.BaseColor = strings.TrimSpace(strings.ToUpper(req.BaseColor))
	if req.BaseColor != "" && !strings.HasPrefix(req.BaseColor, "#") {
		req.BaseColor = "#" + req.BaseColor
	}

	if req.NumColors <= 0 {
		req.NumColors = 5
	}
	if req.NumColors < 3 {
		req.NumColors = 3
	}
	if req.NumColors > 10 {
		req.NumColors = 10
	}

	if req.Style == "" {
		req.Style = "vibrant"
	}

	// Check cache first (skip when debug is requested)
	cacheKey := generateCacheKey(req)
	useCache := redisClient != nil && !req.IncludeAIDebug
	if useCache {
		cached, err := redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var response PaletteResponse
			if json.Unmarshal([]byte(cached), &response) == nil {
				logger.Debug("Cache hit", "cache_key", cacheKey, "theme", req.Theme, "style", req.Style)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				if req.IncludeAIDebug {
					// Augment cached response with debug metadata without mutating cache
					response.Debug = buildDebugInfo(req, requestedStyle)
				}
				json.NewEncoder(w).Encode(response)
				return
			}
		}
	}

	// Generate palette based on theme and style
	palette := generatePalette(req.Theme, req.Style, req.NumColors, req.BaseColor)

	debugInfo := buildDebugInfo(req, requestedStyle)
	if req.IncludeAIDebug {
		augmentWithAIDebug(req, debugInfo)
	}

	name := buildPaletteName(req.Style, req.Theme)
	description := buildPaletteDescription(req.Style, req.Theme, req.BaseColor, req.NumColors)

	response := PaletteResponse{
		Success:     true,
		Palette:     palette,
		Name:        name,
		Theme:       req.Theme,
		Style:       req.Style,
		Description: description,
		Debug:       nil,
	}

	if req.IncludeAIDebug {
		response.Debug = debugInfo
	}

	// Cache the result
	if useCache {
		cacheCopy := response
		cacheCopy.Debug = nil
		if data, err := json.Marshal(cacheCopy); err == nil {
			redisClient.Set(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	// Save to history
	savePaletteHistory(response)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(response)
}

func generateCacheKey(req PaletteRequest) string {
	data := fmt.Sprintf("%s:%s:%d:%s", req.Theme, req.Style, req.NumColors, req.BaseColor)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("palette:%x", hash[:8])
}

func buildDebugInfo(req PaletteRequest, requestedStyle string) *DebugInfo {
	info := &DebugInfo{
		Strategy:           resolveGenerationStrategy(req),
		RequestedTheme:     req.Theme,
		RequestedBaseColor: req.BaseColor,
		RequestedColors:    req.NumColors,
		RequestedStyle:     requestedStyle,
		ResolvedStyle:      req.Style,
		AIRequested:        req.IncludeAIDebug,
	}

	baseHue := resolveBaseHue(req.Theme, req.BaseColor)
	if baseHue >= 0 {
		info.BaseHue = math.Round(baseHue*100) / 100
	}

	return info
}

func augmentWithAIDebug(req PaletteRequest, info *DebugInfo) {
	if info == nil {
		return
	}

	info.AIRequested = true
	result := fetchAISuggestions(buildAISuggestionContext(req))
	if result == nil {
		info.AIError = "unable to reach AI suggestion service"
		return
	}

	info.AIPrompt = result.Prompt
	info.AIModel = result.Model
	info.AIDurationMs = result.DurationMs

	if result.Error != "" {
		info.AIError = result.Error
		return
	}

	info.AIAvailable = len(result.Suggestions) > 0
	info.AIRawOutput = result.RawResponse
	if len(result.Suggestions) > 0 {
		info.AISuggestions = result.Suggestions
	}
}

func resolveBaseHue(theme, baseColor string) float64 {
	if baseColor != "" {
		hue, _, _ := hexToHSL(baseColor)
		return hue
	}
	return getThemeHue(theme)
}

func resolveGenerationStrategy(req PaletteRequest) string {
	sanitizedTheme := strings.TrimSpace(req.Theme)
	switch {
	case req.BaseColor != "" && sanitizedTheme != "" && req.Style != "":
		return "base-color-and-theme"
	case req.BaseColor != "":
		return "base-color"
	case sanitizedTheme != "" && req.Style != "":
		return "theme-and-style"
	case sanitizedTheme != "":
		return "theme-only"
	default:
		return "procedural-default"
	}
}

func buildAISuggestionContext(req PaletteRequest) string {
	parts := make([]string, 0, 3)
	sanitizedTheme := strings.TrimSpace(req.Theme)
	if sanitizedTheme != "" {
		parts = append(parts, sanitizedTheme)
	}
	if req.Style != "" {
		parts = append(parts, fmt.Sprintf("%s style", req.Style))
	}
	if req.BaseColor != "" {
		parts = append(parts, fmt.Sprintf("anchored by %s", strings.ToUpper(req.BaseColor)))
	}
	if len(parts) == 0 {
		return "general purpose interface design"
	}
	return strings.Join(parts, " with ")
}

func buildPaletteName(style, theme string) string {
	styleName := titleCase(style)
	themeName := titleCase(theme)

	switch {
	case styleName != "" && themeName != "":
		return fmt.Sprintf("%s %s", styleName, themeName)
	case themeName != "":
		return themeName
	case styleName != "":
		return fmt.Sprintf("%s Palette", styleName)
	default:
		return "Custom Palette"
	}
}

func buildPaletteDescription(style, theme, baseColor string, numColors int) string {
	var fragments []string
	if strings.TrimSpace(theme) != "" {
		fragments = append(fragments, fmt.Sprintf("Inspired by %s", theme))
	}
	if strings.TrimSpace(style) != "" {
		fragments = append(fragments, fmt.Sprintf("using a %s style", style))
	}
	if baseColor != "" {
		fragments = append(fragments, fmt.Sprintf("anchored by %s", strings.ToUpper(baseColor)))
	}

	description := strings.Join(fragments, " ")
	if description == "" {
		description = "Procedurally generated palette"
	}

	return fmt.Sprintf("%s with %d colors", description, numColors)
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
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

	// Try to get AI-powered suggestions first
	suggestions := getAISuggestions(useCase)
	if suggestions == nil {
		// Fallback to predefined suggestions
		suggestions = getSuggestionsForUseCase(useCase)
	}

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

	// Validate format field
	formatRaw, ok := req["format"]
	if !ok {
		http.Error(w, "Missing 'format' field", http.StatusBadRequest)
		return
	}
	format, ok := formatRaw.(string)
	if !ok {
		http.Error(w, "Invalid 'format' field type (expected string)", http.StatusBadRequest)
		return
	}

	// Validate palette field
	paletteRaw, ok := req["palette"]
	if !ok {
		http.Error(w, "Missing 'palette' field", http.StatusBadRequest)
		return
	}
	palette, ok := paletteRaw.([]interface{})
	if !ok {
		http.Error(w, "Invalid 'palette' field type (expected array)", http.StatusBadRequest)
		return
	}

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
		"ocean":   200,
		"sunset":  25,
		"forest":  120,
		"desert":  40,
		"tech":    210,
		"vintage": 30,
		"modern":  240,
		"nature":  90,
		"fire":    15,
		"ice":     195,
		"autumn":  35,
		"spring":  150,
		"summer":  60,
		"winter":  200,
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
			hue := math.Mod(baseHue+float64(i)*72, 360)
			saturation := 70 + (i%2)*15
			lightness := 45 + (i%3)*10
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	case "pastel":
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue+float64(i)*60, 360)
			saturation := 50 + (i%2)*10
			lightness := 75 + (i%3)*5
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	case "dark":
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue+float64(i)*45, 360)
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
			hue := math.Mod(30+float64(i)*30, 90) // Browns, oranges, greens
			saturation := 35 + (i%3)*15
			lightness := 40 + (i%3)*10
			colors[i] = hslToHex(hue, float64(saturation), float64(lightness))
		}
	default: // Default to harmonious
		for i := 0; i < numColors; i++ {
			hue := math.Mod(baseHue+float64(i)*30, 360)
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
			hue = math.Mod(baseHue+float64(i)*30, 360)
			saturation = baseSat
			lightness = baseLightness
		case "complementary":
			// Opposite colors
			if i == 1 {
				hue = math.Mod(baseHue+180, 360)
			} else {
				hue = math.Mod(baseHue+float64(i-1)*60, 360)
			}
			saturation = baseSat
			lightness = math.Min(100, baseLightness+float64(i-1)*10)
		case "triadic":
			// Three colors evenly spaced
			hue = math.Mod(baseHue+float64(i)*120, 360)
			saturation = baseSat
			lightness = baseLightness
		default:
			// Monochromatic - variations of the same hue
			hue = baseHue
			saturation = math.Max(0, baseSat-float64(i)*10)
			lightness = math.Min(100, baseLightness+float64(i)*15)
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
	r, g, b := hexToRGB(hex)
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255

	max := math.Max(math.Max(rf, gf), bf)
	min := math.Min(math.Min(rf, gf), bf)

	l := (max + min) / 2

	if max == min {
		return 0, 0, l * 100 // achromatic
	}

	d := max - min
	var s float64
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	var h float64
	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	case bf:
		h = (rf-gf)/d + 4
	}
	h /= 6

	return h * 360, s * 100, l * 100
}

func hexToRGB(hex string) (int, int, int) {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	if len(hex) != 6 {
		return 0, 0, 0
	}

	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)

	return int(r), int(g), int(b)
}

func accessibilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AccessibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	contrast := calculateContrastRatio(req.Foreground, req.Background)

	response := AccessibilityResponse{
		Success:       true,
		ContrastRatio: contrast,
		WCAGAA:        contrast >= 4.5,
		WCAGAAA:       contrast >= 7,
		LargeTextAA:   contrast >= 3,
		LargeTextAAA:  contrast >= 4.5,
	}

	// Add recommendations
	if contrast < 3 {
		response.Recommendation = "Very poor contrast - not suitable for any text"
	} else if contrast < 4.5 {
		response.Recommendation = "Only suitable for large text (18pt+ or 14pt+ bold)"
	} else if contrast < 7 {
		response.Recommendation = "Meets AA standards for normal text"
	} else {
		response.Recommendation = "Excellent contrast - meets AAA standards"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func calculateContrastRatio(fg, bg string) float64 {
	fgL := getRelativeLuminance(fg)
	bgL := getRelativeLuminance(bg)

	// Ensure lighter color is L1
	if fgL > bgL {
		return (fgL + 0.05) / (bgL + 0.05)
	}
	return (bgL + 0.05) / (fgL + 0.05)
}

func getRelativeLuminance(hex string) float64 {
	r, g, b := hexToRGB(hex)

	// Convert to sRGB
	rs := float64(r) / 255
	gs := float64(g) / 255
	bs := float64(b) / 255

	// Apply gamma correction
	if rs <= 0.03928 {
		rs = rs / 12.92
	} else {
		rs = math.Pow((rs+0.055)/1.055, 2.4)
	}

	if gs <= 0.03928 {
		gs = gs / 12.92
	} else {
		gs = math.Pow((gs+0.055)/1.055, 2.4)
	}

	if bs <= 0.03928 {
		bs = bs / 12.92
	} else {
		bs = math.Pow((bs+0.055)/1.055, 2.4)
	}

	// Calculate relative luminance
	return 0.2126*rs + 0.7152*gs + 0.0722*bs
}

func getAISuggestions(useCase string) []map[string]interface{} {
	result := fetchAISuggestions(useCase)
	if result == nil || result.Error != "" || len(result.Suggestions) == 0 {
		return nil
	}
	return result.Suggestions
}

func fetchAISuggestions(useCase string) *AISuggestionResult {
	model := getEnv("OLLAMA_MODEL", "llama3.2")
	ollamaURL := getEnv("OLLAMA_API_GENERATE", "http://127.0.0.1:11434/api/generate")

	prompt := fmt.Sprintf(`Generate 2 color palettes for a %s use case.
For each palette provide:
1. A creative name
2. Exactly 5 hex colors
3. A brief description

Respond in this exact JSON format:
[
  {
    "name": "Palette Name",
    "colors": ["#HEX1", "#HEX2", "#HEX3", "#HEX4", "#HEX5"],
    "description": "Brief description"
  }
]`, useCase)

	result := &AISuggestionResult{
		Model:  model,
		Prompt: prompt,
	}

	reqBody := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		result.Error = fmt.Sprintf("marshal request: %v", err)
		logger.Error("Failed to marshal Ollama request", "error", err, "model", model)
		return result
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		logger.Error("Failed to contact Ollama", "error", err, "url", ollamaURL)
		return result
	}
	defer resp.Body.Close()
	result.DurationMs = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("status %d", resp.StatusCode)
		logger.Warn("Ollama returned non-OK status", "status_code", resp.StatusCode, "url", ollamaURL)
		return result
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("read response: %v", err)
		logger.Error("Failed to read Ollama response", "error", err)
		return result
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		result.Error = fmt.Sprintf("parse response: %v", err)
		logger.Error("Failed to parse Ollama response", "error", err, "body_length", len(body))
		return result
	}

	result.RawResponse = ollamaResp.Response

	raw := strings.TrimSpace(ollamaResp.Response)
	if raw == "" {
		result.Error = "empty AI response"
		return result
	}

	var suggestions []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &suggestions); err != nil {
		result.Error = fmt.Sprintf("parse AI JSON: %v", err)
		logger.Error("Failed to parse AI suggestions as JSON", "error", err, "raw_response_length", len(raw))
		return result
	}

	result.Suggestions = suggestions
	return result
}

// New handler functions

// New handler functions

func harmonyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HarmonyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	analysis := analyzeColorHarmony(req.Colors)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

func colorblindHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ColorblindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "protanopia"
	}

	simulated := simulateColorblindness(req.Colors, req.Type)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ColorblindResponse{
		Success:   true,
		Simulated: simulated,
		Type:      req.Type,
	})
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	history := getPaletteHistory(10)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"history": history,
	})
}

func savePaletteHistory(response PaletteResponse) {
	if redisClient == nil {
		return
	}

	cacheCopy := response
	cacheCopy.Debug = nil
	data, err := json.Marshal(cacheCopy)
	if err != nil {
		return
	}

	timestamp := time.Now().Unix()
	key := fmt.Sprintf("history:%d", timestamp)

	redisClient.Set(ctx, key, data, 7*24*time.Hour)
	redisClient.ZAdd(ctx, "palette:history", redis.Z{
		Score:  float64(timestamp),
		Member: key,
	})
}

func getPaletteHistory(limit int) []PaletteResponse {
	if redisClient == nil {
		return []PaletteResponse{}
	}

	keys, err := redisClient.ZRevRange(ctx, "palette:history", 0, int64(limit-1)).Result()
	if err != nil {
		return []PaletteResponse{}
	}

	history := make([]PaletteResponse, 0, len(keys))
	for _, key := range keys {
		data, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var response PaletteResponse
		if json.Unmarshal([]byte(data), &response) == nil {
			history = append(history, response)
		}
	}

	return history
}

func analyzeColorHarmony(colors []string) HarmonyResponse {
	if len(colors) < 2 {
		return HarmonyResponse{
			Success:      false,
			IsHarmonious: false,
			Score:        0,
			Analysis:     map[string]interface{}{"error": "Need at least 2 colors"},
		}
	}

	// Convert colors to HSL
	hues := make([]float64, len(colors))
	for i, color := range colors {
		hue, _, _ := hexToHSL(color)
		hues[i] = hue
	}

	// Analyze relationships
	analysis := make(map[string]interface{})
	score := 0.0
	relationships := []string{}

	// Check for complementary (opposite on color wheel ~180 degrees)
	for i := 0; i < len(hues); i++ {
		for j := i + 1; j < len(hues); j++ {
			diff := math.Abs(hues[i] - hues[j])
			if diff > 180 {
				diff = 360 - diff
			}

			if diff >= 170 && diff <= 190 {
				relationships = append(relationships, "complementary")
				score += 1.0
			} else if diff >= 25 && diff <= 35 {
				relationships = append(relationships, "analogous")
				score += 0.8
			} else if diff >= 115 && diff <= 125 {
				relationships = append(relationships, "triadic")
				score += 0.9
			}
		}
	}

	// Check for monochromatic (similar hues)
	isMonochromatic := true
	for i := 1; i < len(hues); i++ {
		if math.Abs(hues[i]-hues[0]) > 30 {
			isMonochromatic = false
			break
		}
	}

	if isMonochromatic {
		relationships = append(relationships, "monochromatic")
		score += 0.7
	}

	// Normalize score
	maxScore := float64(len(colors) * (len(colors) - 1) / 2)
	if maxScore > 0 {
		score = score / maxScore
	}

	analysis["relationships"] = relationships
	analysis["color_count"] = len(colors)
	analysis["hue_range"] = calculateHueRange(hues)

	return HarmonyResponse{
		Success:      true,
		Analysis:     analysis,
		IsHarmonious: score > 0.5 || len(relationships) > 0,
		Score:        score,
	}
}

func calculateHueRange(hues []float64) float64 {
	if len(hues) == 0 {
		return 0
	}

	min := hues[0]
	max := hues[0]

	for _, hue := range hues[1:] {
		if hue < min {
			min = hue
		}
		if hue > max {
			max = hue
		}
	}

	return max - min
}

func simulateColorblindness(colors []string, cbType string) []string {
	simulated := make([]string, len(colors))

	for i, color := range colors {
		r, g, b := hexToRGB(color)

		// Apply colorblind simulation matrix
		var nr, ng, nb int
		switch cbType {
		case "protanopia": // Red-blind
			nr = int(0.567*float64(r) + 0.433*float64(g))
			ng = int(0.558*float64(r) + 0.442*float64(g))
			nb = int(0.242 * float64(b))
		case "deuteranopia": // Green-blind
			nr = int(0.625*float64(r) + 0.375*float64(g))
			ng = int(0.700*float64(r) + 0.300*float64(g))
			nb = int(0.300 * float64(b))
		case "tritanopia": // Blue-blind
			nr = int(0.950*float64(r) + 0.050*float64(g))
			ng = int(0.433 * float64(g))
			nb = int(0.475*float64(r) + 0.525*float64(b))
		default:
			nr, ng, nb = r, g, b
		}

		// Clamp values
		nr = clamp(nr, 0, 255)
		ng = clamp(ng, 0, 255)
		nb = clamp(nb, 0, 255)

		simulated[i] = fmt.Sprintf("#%02X%02X%02X", nr, ng, nb)
	}

	return simulated
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

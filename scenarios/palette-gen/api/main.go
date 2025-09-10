package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	// Call n8n workflow for palette generation
	n8nURL := os.Getenv("N8N_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	webhookData := map[string]interface{}{
		"theme":      req.Theme,
		"style":      req.Style,
		"num_colors": req.NumColors,
		"base_color": req.BaseColor,
	}

	jsonData, _ := json.Marshal(webhookData)
	resp, err := http.Post(
		fmt.Sprintf("%s/webhook/palette-generator", n8nURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PaletteResponse{
			Success: false,
			Error:   "Failed to generate palette",
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
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

package main

import (
	"github.com/vrooli/api-core/preflight"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

type GenerateRequest struct {
	Text           string `json:"text"`
	Size           int    `json:"size"`
	Color          string `json:"color"`
	Background     string `json:"background"`
	ErrorCorrection string `json:"errorCorrection"`
	Format         string `json:"format"`
	Margin         int    `json:"margin"`
}

type GenerateResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data"`
	Format  string `json:"format"`
	Error   string `json:"error,omitempty"`
}

type BatchRequest struct {
	Items   []BatchItem    `json:"items"`
	Options GenerateRequest `json:"options"`
}

type BatchItem struct {
	Text  string `json:"text"`
	Label string `json:"label"`
}

type BatchResponse struct {
	Success bool              `json:"success"`
	Results []GenerateResponse `json:"results"`
	Error   string            `json:"error,omitempty"`
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "qr-code-generator",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Require explicit port configuration
	port := os.Getenv("API_PORT")
	if port == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: API_PORT environment variable is required\n")
		fmt.Fprintf(os.Stderr, "💡 This service must be run through the Vrooli lifecycle system.\n")
		os.Exit(1)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/batch", batchHandler)
	http.HandleFunc("/formats", formatsHandler)

	// Use structured logging format
	log.Printf("[INFO] service=qr-code-generator action=starting port=%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("[ERROR] service=qr-code-generator action=start_failed error=%v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "qr-code-generator",
		"timestamp": time.Now().Format(time.RFC3339),
		"features": map[string]bool{
			"generate": true,
			"batch": true,
			"formats": true,
		},
	})
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate and set defaults
	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}
	if req.Size == 0 {
		req.Size = 256
	}
	if req.Format == "" {
		req.Format = "png"
	}

	// Generate QR code
	response, err := generateQRCode(req)
	if err != nil {
		response = GenerateResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func batchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "Items array is required", http.StatusBadRequest)
		return
	}

	// Set default options
	if req.Options.Size == 0 {
		req.Options.Size = 256
	}
	if req.Options.Format == "" {
		req.Options.Format = "png"
	}

	// Generate QR codes for each item
	results := make([]GenerateResponse, 0, len(req.Items))
	for _, item := range req.Items {
		genReq := req.Options
		genReq.Text = item.Text
		
		response, err := generateQRCode(genReq)
		if err != nil {
			response = GenerateResponse{
				Success: false,
				Error:   err.Error(),
			}
		}
		results = append(results, response)
	}

	batchResp := BatchResponse{
		Success: true,
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(batchResp)
}

func formatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"formats": []string{"png", "base64"},
		"sizes": []int{128, 256, 512, 1024},
		"errorCorrections": []string{"Low", "Medium", "High", "Highest"},
	})
}

func generateQRCode(req GenerateRequest) (GenerateResponse, error) {
	// Map error correction level
	level := qrcode.Medium
	switch req.ErrorCorrection {
	case "Low", "L":
		level = qrcode.Low
	case "High", "H":
		level = qrcode.High
	case "Highest", "Q":
		level = qrcode.Highest
	default:
		level = qrcode.Medium
	}

	// Generate QR code
	qr, err := qrcode.New(req.Text, level)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("failed to create QR code: %v", err)
	}

	// Generate PNG bytes
	png, err := qr.PNG(req.Size)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("failed to generate PNG: %v", err)
	}

	// Encode to base64
	data := base64.StdEncoding.EncodeToString(png)

	return GenerateResponse{
		Success: true,
		Data:    data,
		Format:  "base64",
	}, nil
}
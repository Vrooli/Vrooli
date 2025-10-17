package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type GenerateRequest struct {
	BusinessName   string   `json:"business_name"`
	BusinessType   string   `json:"business_type"`
	Jurisdictions  []string `json:"jurisdictions"`
	DocumentType   string   `json:"document_type"`
	DataTypes      []string `json:"data_types,omitempty"`
	CustomClauses  []string `json:"custom_clauses,omitempty"`
	Email          string   `json:"email,omitempty"`
	Website        string   `json:"website,omitempty"`
	Format         string   `json:"format,omitempty"`
}

type GenerateResponse struct {
	DocumentID   string    `json:"document_id"`
	Content      string    `json:"content"`
	Format       string    `json:"format"`
	GeneratedAt  time.Time `json:"generated_at"`
	TemplateVersion string `json:"template_version"`
	PreviewURL   string    `json:"preview_url"`
}

type HealthResponse struct {
	Status     string `json:"status"`
	Timestamp  string `json:"timestamp"`
	Components map[string]string `json:"components"`
}

type TemplateFreshnessResponse struct {
	LastUpdate      time.Time     `json:"last_update"`
	StaleTemplates  []interface{} `json:"stale_templates"`
	UpdateAvailable bool          `json:"update_available"`
}

type DocumentHistoryResponse struct {
	History []HistoryEntry `json:"history"`
}

type HistoryEntry struct {
	ChangedAt     time.Time `json:"changed_at"`
	ChangeType    string    `json:"change_type"`
	ChangedBy     string    `json:"changed_by"`
	ChangeSummary string    `json:"change_summary"`
}

type SearchRequest struct {
	Query        string `json:"query"`
	Limit        int    `json:"limit,omitempty"`
	ClauseType   string `json:"clause_type,omitempty"`
	Jurisdiction string `json:"jurisdiction,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
}

type SearchResult struct {
	ClauseID     string `json:"clause_id"`
	ClauseType   string `json:"clause_type"`
	Jurisdiction string `json:"jurisdiction"`
	Content      string `json:"content"`
	Score        float64 `json:"score,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	components := make(map[string]string)
	
	// Check PostgreSQL
	cmd := exec.Command("resource-postgres", "status", "--json")
	if err := cmd.Run(); err == nil {
		components["postgres"] = "healthy"
	} else {
		components["postgres"] = "unhealthy"
	}
	
	// Check Ollama
	cmd = exec.Command("resource-ollama", "status", "--json")
	if err := cmd.Run(); err == nil {
		components["ollama"] = "healthy"
	} else {
		components["ollama"] = "unhealthy"
	}
	
	response := HealthResponse{
		Status:     "healthy",
		Timestamp:  time.Now().Format(time.RFC3339),
		Components: components,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	
	// Validate required fields
	if req.BusinessName == "" || req.DocumentType == "" || len(req.Jurisdictions) == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Determine format (default to markdown)
	format := req.Format
	if format == "" {
		format = "markdown"
	}
	
	// Generate document using CLI (for now, as a bridge)
	args := []string{
		"generate", req.DocumentType,
		"--business-name", req.BusinessName,
		"--jurisdiction", req.Jurisdictions[0],
		"--format", format,
	}
	
	if req.BusinessType != "" {
		args = append(args, "--business-type", req.BusinessType)
	}
	if req.Email != "" {
		args = append(args, "--email", req.Email)
	}
	if req.Website != "" {
		args = append(args, "--website", req.Website)
	}
	if len(req.DataTypes) > 0 {
		dataTypesStr := ""
		for i, dt := range req.DataTypes {
			if i > 0 {
				dataTypesStr += ","
			}
			dataTypesStr += dt
		}
		args = append(args, "--data-types", dataTypesStr)
	}
	
	cmd := exec.Command("/home/matthalloran8/Vrooli/scenarios/privacy-terms-generator/cli/privacy-terms-generator", args...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Document generation failed: %v", err)
		http.Error(w, "Document generation failed", http.StatusInternalServerError)
		return
	}
	
	response := GenerateResponse{
		DocumentID:      fmt.Sprintf("doc_%d", time.Now().Unix()),
		Content:         string(output),
		Format:          format,
		GeneratedAt:     time.Now(),
		TemplateVersion: "1.0.0",
		PreviewURL:      fmt.Sprintf("/preview/%d", time.Now().Unix()),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func templateFreshnessHandler(w http.ResponseWriter, r *http.Request) {
	response := TemplateFreshnessResponse{
		LastUpdate:      time.Now().AddDate(0, 0, -7), // Mock: 7 days ago
		StaleTemplates:  []interface{}{},
		UpdateAvailable: false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func documentHistoryHandler(w http.ResponseWriter, r *http.Request) {
	// Extract document ID from URL path
	docID := r.URL.Query().Get("id")
	if docID == "" {
		http.Error(w, "Document ID required", http.StatusBadRequest)
		return
	}

	// Call CLI to get history
	cmd := exec.Command("/home/matthalloran8/Vrooli/scenarios/privacy-terms-generator/cli/privacy-terms-generator",
		"history", docID, "--json")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to get document history: %v", err)
		http.Error(w, "Failed to retrieve document history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func searchClausesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	// Build CLI command
	args := []string{"search", req.Query, "--json"}
	if req.Limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.ClauseType != "" {
		args = append(args, "--type", req.ClauseType)
	}
	if req.Jurisdiction != "" {
		args = append(args, "--jurisdiction", req.Jurisdiction)
	}

	cmd := exec.Command("/home/matthalloran8/Vrooli/scenarios/privacy-terms-generator/cli/privacy-terms-generator", args...)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Search failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "15000"
	}

	// Health check at root level (required by orchestration)
	http.HandleFunc("/health", corsMiddleware(healthHandler))

	// API endpoints
	http.HandleFunc("/api/v1/legal/generate", corsMiddleware(generateHandler))
	http.HandleFunc("/api/v1/legal/templates/freshness", corsMiddleware(templateFreshnessHandler))
	http.HandleFunc("/api/v1/legal/documents/history", corsMiddleware(documentHistoryHandler))
	http.HandleFunc("/api/v1/legal/clauses/search", corsMiddleware(searchClausesHandler))
	
	log.Printf("Privacy & Terms Generator API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
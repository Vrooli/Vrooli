package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
)

type SEOAuditRequest struct {
	URL   string `json:"url"`
	Depth int    `json:"depth,omitempty"`
}

type ContentOptimizeRequest struct {
	Content        string `json:"content"`
	TargetKeywords string `json:"target_keywords"`
	ContentType    string `json:"content_type,omitempty"`
}

type CompetitorAnalysisRequest struct {
	CompetitorURL string `json:"competitor_url"`
	YourURL       string `json:"your_url"`
	AnalysisType  string `json:"analysis_type,omitempty"`
}

type KeywordResearchRequest struct {
	SeedKeyword    string `json:"seed_keyword"`
	TargetLocation string `json:"target_location,omitempty"`
	Language       string `json:"language,omitempty"`
}

// NOTE: HealthResponse replaced by api-core/health standardized response

var seoProcessor *SEOProcessor

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "seo-optimizer",
	}) {
		return // Process was re-exec'd after rebuild
	}

	port := getEnv("API_PORT", getEnv("PORT", ""))
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Initialize SEO Processor
	seoProcessor = NewSEOProcessor()

	// Enable CORS
	// Health check - use api-core/health for standardized response
	apiCoreHealthHandler := health.New().
		Version("1.0.0").
		Handler()
	http.HandleFunc("/health", corsMiddleware(apiCoreHealthHandler))
	http.HandleFunc("/api/seo-audit", corsMiddleware(seoAuditHandler))
	http.HandleFunc("/api/audit", corsMiddleware(seoAuditHandler))
	http.HandleFunc("/api/keyword-research", corsMiddleware(keywordResearchHandler))
	http.HandleFunc("/api/content-optimize", corsMiddleware(contentOptimizeHandler))
	http.HandleFunc("/api/competitor-analysis", corsMiddleware(competitorAnalysisHandler))

	log.Printf("SEO Optimizer API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func corsMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

// NOTE: healthHandler replaced by api-core/health for standardized responses

func seoAuditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SEOAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Use SEO Processor to perform audit
	ctx := context.Background()
	result, err := seoProcessor.PerformSEOAudit(ctx, req.URL, req.Depth)
	if err != nil {
		log.Printf("Error performing SEO audit: %v", err)
		http.Error(w, "Failed to perform SEO audit", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func contentOptimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ContentOptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Use SEO Processor to optimize content
	ctx := context.Background()
	result, err := seoProcessor.OptimizeContent(ctx, req.Content, req.TargetKeywords, req.ContentType)
	if err != nil {
		log.Printf("Error optimizing content: %v", err)
		http.Error(w, "Failed to optimize content", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func competitorAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompetitorAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.YourURL == "" || req.CompetitorURL == "" {
		http.Error(w, "Both your_url and competitor_url are required", http.StatusBadRequest)
		return
	}

	// Use SEO Processor to analyze competitor
	ctx := context.Background()
	result, err := seoProcessor.AnalyzeCompetitor(ctx, req.YourURL, req.CompetitorURL, req.AnalysisType)
	if err != nil {
		log.Printf("Error analyzing competitor: %v", err)
		http.Error(w, "Failed to analyze competitor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func keywordResearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req KeywordResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SeedKeyword == "" {
		http.Error(w, "Seed keyword is required", http.StatusBadRequest)
		return
	}

	// Use SEO Processor to research keywords
	ctx := context.Background()
	result, err := seoProcessor.ResearchKeywords(ctx, req.SeedKeyword, req.TargetLocation, req.Language)
	if err != nil {
		log.Printf("Error researching keywords: %v", err)
		http.Error(w, "Failed to research keywords", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

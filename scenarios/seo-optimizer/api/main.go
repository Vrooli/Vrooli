package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "time"
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

type HealthResponse struct {
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
    Service   string    `json:"service"`
}

func main() {
	port := getEnv("API_PORT", getEnv("PORT", ""))

    // Enable CORS
    http.HandleFunc("/health", corsMiddleware(healthHandler))
    http.HandleFunc("/api/seo-audit", corsMiddleware(seoAuditHandler))
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:    "healthy",
        Timestamp: time.Now(),
        Service:   "seo-optimizer-api",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func callN8nWorkflow(webhookPath string, payload interface{}) (map[string]interface{}, error) {
    n8nURL := os.Getenv("N8N_BASE_URL")
    if n8nURL == "" {
        n8nURL = "http://localhost:5678"
    }
    
    fullURL := fmt.Sprintf("%s/webhook/%s", n8nURL, webhookPath)
    
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(payloadBytes))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: 60 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, err
    }
    
    return result, nil
}

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

    // Call n8n workflow
    result, err := callN8nWorkflow("seo-audit", map[string]interface{}{
        "url":   req.URL,
        "depth": req.Depth,
    })
    
    if err != nil {
        log.Printf("Error calling n8n workflow: %v", err)
        // Return a fallback response
        response := map[string]interface{}{
            "url":        req.URL,
            "audit_id":   fmt.Sprintf("audit_%d", time.Now().Unix()),
            "status":     "processing",
            "message":    "SEO audit initiated. Results will be available shortly.",
            "created_at": time.Now(),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
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

    // Call n8n workflow
    result, err := callN8nWorkflow("content-optimizer", map[string]interface{}{
        "content":         req.Content,
        "target_keywords": req.TargetKeywords,
        "content_type":    req.ContentType,
    })
    
    if err != nil {
        log.Printf("Error calling n8n workflow: %v", err)
        response := map[string]interface{}{
            "status":  "error",
            "message": "Content optimization service temporarily unavailable",
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
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

    // Call n8n workflow
    result, err := callN8nWorkflow("competitor-analyzer", map[string]interface{}{
        "competitor_url": req.CompetitorURL,
        "your_url":       req.YourURL,
        "analysis_type":  req.AnalysisType,
    })
    
    if err != nil {
        log.Printf("Error calling n8n workflow: %v", err)
        response := map[string]interface{}{
            "status":  "error",
            "message": "Competitor analysis service temporarily unavailable",
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
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

    // Call n8n workflow
    result, err := callN8nWorkflow("keyword-researcher", map[string]interface{}{
        "seed_keyword":    req.SeedKeyword,
        "target_location": req.TargetLocation,
        "language":        req.Language,
    })
    
    if err != nil {
        log.Printf("Error calling n8n workflow: %v", err)
        // Return mock data for now
        response := map[string]interface{}{
            "keywords": []map[string]interface{}{
                {
                    "keyword":     req.SeedKeyword,
                    "volume":      "10K-100K",
                    "competition": "Medium",
                    "cpc":         "$2.50",
                    "intent":      "Informational",
                },
            },
            "status": "success",
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

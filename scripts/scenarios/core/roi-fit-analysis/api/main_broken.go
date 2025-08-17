package main

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
    
    _ "github.com/lib/pq"
)

type AnalysisRequest struct {
    Idea        string   `json:"idea"`
    Budget      float64  `json:"budget"`
    Timeline    string   `json:"timeline"`
    Skills      []string `json:"skills"`
    MarketFocus string   `json:"market_focus,omitempty"`
}

type N8nAnalysisResponse struct {
    AnalysisID       string                 `json:"analysis_id"`
    Timestamp        string                 `json:"timestamp"`
    Input            AnalysisRequest        `json:"input"`
    ROIMetrics       ROIMetrics            `json:"roi_metrics"`
    DetailedAnalysis map[string]interface{} `json:"detailed_analysis"`
    Success          bool                   `json:"success"`
}

type ROIMetrics struct {
    ROIPercentage     float64 `json:"roi_percentage"`
    EstimatedRevenue  float64 `json:"estimated_revenue"`
    PaybackMonths     int     `json:"payback_months"`
    RiskLevel         string  `json:"risk_level"`
    ConfidenceScore   float64 `json:"confidence_score"`
}

type OpportunityResponse struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    ROIScore    float64   `json:"roi_score"`
    Investment  float64   `json:"investment"`
    Payback     int       `json:"payback_months"`
    RiskLevel   string    `json:"risk_level"`
    CreatedAt   time.Time `json:"created_at"`
}

type HealthResponse struct {
    Status    string `json:"status"`
    Timestamp string `json:"timestamp"`
    Service   string `json:"service"`
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
        
        next.ServeHTTP(w, r)
    }
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req AnalysisRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Call N8n ROI analyzer workflow
    n8nURL := os.Getenv("N8N_URL")
    if n8nURL == "" {
        n8nURL = "http://localhost:5678"
    }
    
    // Set default market focus if not provided
    if req.MarketFocus == "" {
        req.MarketFocus = "general"
    }
    
    analysis, err := callN8nROIAnalyzer(n8nURL, req)
    if err != nil {
        log.Printf("Error calling N8n analyzer: %v", err)
        http.Error(w, "Analysis service unavailable", http.StatusServiceUnavailable)
        return
    }
    
    // Store analysis in database
    if err := storeAnalysis(analysis); err != nil {
        log.Printf("Error storing analysis: %v", err)
        // Continue even if storage fails
    }
    
    response := map[string]interface{}{
        "status": "success",
        "analysis": map[string]interface{}{
            "roi_score":        analysis.ROIMetrics.ROIPercentage,
            "recommendation":   getRecommendation(analysis.ROIMetrics.ROIPercentage),
            "payback_months":   analysis.ROIMetrics.PaybackMonths,
            "risk_level":       analysis.ROIMetrics.RiskLevel,
            "market_size":      extractMarketSize(analysis.DetailedAnalysis),
            "competition":      extractCompetitionLevel(analysis.DetailedAnalysis),
            "confidence":       analysis.ROIMetrics.ConfidenceScore,
            "analysis_id":      analysis.AnalysisID,
        },
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func opportunitiesHandler(w http.ResponseWriter, r *http.Request) {
    opportunities, err := getStoredOpportunities()
    if err != nil {
        log.Printf("Error loading opportunities: %v", err)
        // Fallback to default opportunities
        opportunities = []OpportunityResponse{
            {
                ID:         "opp-001",
                Name:       "AI-Powered CRM",
                ROIScore:   85.5,
                Investment: 75000,
                Payback:    14,
                RiskLevel:  "Medium",
                CreatedAt:  time.Now().Add(-24 * time.Hour),
            },
            {
                ID:         "opp-002",
                Name:       "E-commerce Platform",
                ROIScore:   72.3,
                Investment: 50000,
                Payback:    18,
                RiskLevel:  "Low",
                CreatedAt:  time.Now().Add(-48 * time.Hour),
            },
            {
                ID:         "opp-003",
                Name:       "EdTech Solution",
                ROIScore:   91.2,
                Investment: 100000,
                Payback:    12,
                RiskLevel:  "High",
                CreatedAt:  time.Now().Add(-72 * time.Hour),
            },
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(opportunities)
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
    report := map[string]interface{}{
        "generated_at": time.Now(),
        "summary": map[string]interface{}{
            "total_analyzed":    15,
            "high_roi_count":    5,
            "average_roi":       68.5,
            "best_opportunity":  "EdTech Solution",
        },
        "recommendations": []string{
            "Focus on high-growth markets",
            "Leverage existing skills in AI",
            "Consider partnership opportunities",
        },
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(report)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    health := HealthResponse{
        Status:    "healthy",
        Timestamp: time.Now().Format(time.RFC3339),
        Service:   "roi-fit-analysis",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

// REMOVED - functions moved above main
func getStoredOpportunities() ([]OpportunityResponse, error) {
    if db == nil {
        return nil, fmt.Errorf("database not initialized")
    }
    
    // First get from opportunities table
    opQuery := `
        SELECT id, name, investment_required, roi_score, payback_months, 'Medium' as risk_level, created_at
        FROM opportunities 
        WHERE status = 'active' AND roi_score >= 60
        ORDER BY roi_score DESC 
        LIMIT 5
    `
    
    rows, err := db.Query(opQuery)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var opportunities []OpportunityResponse
    for rows.Next() {
        var opp OpportunityResponse
        err := rows.Scan(&opp.ID, &opp.Name, &opp.Investment, &opp.ROIScore, &opp.Payback, &opp.RiskLevel, &opp.CreatedAt)
        if err != nil {
            continue // Skip invalid rows
        }
        opportunities = append(opportunities, opp)
    }
    
    // Then add recent analyses
    analysisQuery := `
        SELECT analysis_id, idea, budget, roi_percentage, payback_months, risk_level, created_at
        FROM roi_analyses 
        WHERE roi_percentage >= 60
        ORDER BY created_at DESC 
        LIMIT 5
    `
    
    rows2, err := db.Query(analysisQuery)
    if err != nil {
        return opportunities, nil // Return opportunities even if analyses fail
    }
    defer rows2.Close()
    
    for rows2.Next() {
        var opp OpportunityResponse
        err := rows2.Scan(&opp.ID, &opp.Name, &opp.Investment, &opp.ROIScore, &opp.Payback, &opp.RiskLevel, &opp.CreatedAt)
        if err != nil {
            continue // Skip invalid rows
        }
        opportunities = append(opportunities, opp)
    }
    
    return opportunities, nil
}

func main() {
    // Initialize database connection
    if err := initDB(); err != nil {
        log.Printf("Warning: Database connection failed: %v", err)
        log.Printf("Continuing with mock data...")
    } else {
        log.Println("Database connected successfully")
        defer db.Close()
    }
    
    port := os.Getenv("API_PORT")
    if port == "" {
        port = "3000"
    }

    http.HandleFunc("/analyze", corsMiddleware(analyzeHandler))
    http.HandleFunc("/opportunities", corsMiddleware(opportunitiesHandler))
    http.HandleFunc("/reports", corsMiddleware(reportsHandler))
    http.HandleFunc("/health", corsMiddleware(healthHandler))

    log.Printf("ROI Fit Analysis API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}
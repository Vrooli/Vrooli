package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "os"
    "strings"
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

// Global database connection and ROI engine
var db *sql.DB
var roiEngine *ROIAnalysisEngine

// Initialize database connection with exponential backoff
func initDB() error {
    // Database configuration - support both POSTGRES_URL and individual components
    postgresURL := os.Getenv("POSTGRES_URL")
    if postgresURL == "" {
        // Try to build from individual components - REQUIRED, no defaults
        dbHost := os.Getenv("POSTGRES_HOST")
        dbPort := os.Getenv("POSTGRES_PORT")
        dbUser := os.Getenv("POSTGRES_USER")
        dbPassword := os.Getenv("POSTGRES_PASSWORD")
        dbName := os.Getenv("POSTGRES_DB")
        
        if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
            return fmt.Errorf("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
        }
        
        postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
            dbUser, dbPassword, dbHost, dbPort, dbName)
    }
    
    var err error
    db, err = sql.Open("postgres", postgresURL)
    if err != nil {
        return err
    }
    
    // Set connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Implement exponential backoff for database connection
    maxRetries := 10
    baseDelay := 1 * time.Second
    maxDelay := 30 * time.Second
    
    log.Println("🔄 Attempting database connection with exponential backoff...")
    log.Printf("📆 Database URL configured")
    
    var pingErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        pingErr = db.Ping()
        if pingErr == nil {
            log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
            return nil
        }
        
        // Calculate exponential backoff delay
        delay := time.Duration(math.Min(
            float64(baseDelay) * math.Pow(2, float64(attempt)),
            float64(maxDelay),
        ))
        
        // Add progressive jitter to prevent thundering herd
        jitterRange := float64(delay) * 0.25
        jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
        actualDelay := delay + jitter
        
        log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
        log.Printf("⏳ Waiting %v before next attempt", actualDelay)
        
        // Provide detailed status every few attempts
        if attempt > 0 && attempt % 3 == 0 {
            log.Printf("📈 Retry progress:")
            log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
            log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
            log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
        }
        
        time.Sleep(actualDelay)
    }
    
    return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
}

// Helper functions for response formatting

// getRecommendationFromRating converts overall rating to recommendation
func getRecommendationFromRating(rating string) string {
    switch strings.ToLower(rating) {
    case "excellent":
        return "Excellent opportunity - High potential"
    case "good":
        return "Good opportunity - Moderate potential"  
    case "fair":
        return "Fair opportunity - Consider risks"
    case "poor":
        return "Poor opportunity - High risk"
    default:
        return "Assessment pending"
    }
}

// formatMarketSize formats market size for display
func formatMarketSize(size float64) string {
    if size >= 1000000000 {
        return fmt.Sprintf("$%.1fB", size/1000000000)
    } else if size >= 1000000 {
        return fmt.Sprintf("$%.1fM", size/1000000)
    } else if size >= 1000 {
        return fmt.Sprintf("$%.1fK", size/1000)
    }
    return fmt.Sprintf("$%.0f", size)
}

// analysisResultsHandler returns stored analysis results
func analysisResultsHandler(w http.ResponseWriter, r *http.Request) {
    if db == nil {
        http.Error(w, "Database not available", http.StatusServiceUnavailable)
        return
    }
    
    query := `
        SELECT id, idea, budget, timeline, market_focus, status, success,
               execution_time_ms, error_message, created_at
        FROM roi_analyses
        ORDER BY created_at DESC
        LIMIT 20
    `
    
    rows, err := db.Query(query)
    if err != nil {
        http.Error(w, "Failed to query results", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var results []map[string]interface{}
    for rows.Next() {
        var id, idea, budget, timeline, marketFocus, status, errorMsg string
        var success bool
        var executionTime int64
        var createdAt time.Time
        
        err := rows.Scan(&id, &idea, &budget, &timeline, &marketFocus, &status, 
                        &success, &executionTime, &errorMsg, &createdAt)
        if err != nil {
            continue
        }
        
        result := map[string]interface{}{
            "id":               id,
            "idea":             idea,
            "budget":           budget,
            "timeline":         timeline,
            "market_focus":     marketFocus,
            "status":           status,
            "success":          success,
            "execution_time_ms": executionTime,
            "created_at":       createdAt,
        }
        
        if errorMsg != "" {
            result["error"] = errorMsg
        }
        
        results = append(results, result)
    }
    
    response := map[string]interface{}{
        "results": results,
        "count":   len(results),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Get recommendation based on ROI score
func getRecommendation(roiScore float64) string {
    if roiScore >= 80 {
        return "Excellent opportunity - High potential"
    } else if roiScore >= 60 {
        return "Good opportunity - Moderate potential"
    } else if roiScore >= 40 {
        return "Fair opportunity - Consider risks"
    } else {
        return "Poor opportunity - High risk"
    }
}

// Extract market size from detailed analysis
func extractMarketSize(details map[string]interface{}) string {
    if size, ok := details["market_size"]; ok {
        if sizeStr, ok := size.(string); ok {
            return sizeStr
        }
    }
    return "Unknown"
}

// Extract competition level from detailed analysis
func extractCompetitionLevel(details map[string]interface{}) int {
    if comp, ok := details["competition_level"]; ok {
        if compFloat, ok := comp.(float64); ok {
            return int(compFloat)
        }
    }
    return 5 // Default moderate competition
}

// Get stored opportunities from database
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

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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

    // Convert to comprehensive analysis request
    comprehensiveReq := &ComprehensiveAnalysisRequest{
        Idea:          req.Idea,
        Budget:        req.Budget,
        Timeline:      req.Timeline,
        Skills:        req.Skills,
        MarketFocus:   req.MarketFocus,
        RiskTolerance: "medium", // Default
    }
    
    if comprehensiveReq.MarketFocus == "" {
        comprehensiveReq.MarketFocus = "general"
    }

    // Perform comprehensive analysis using ROI engine (replaces all n8n workflows)
    analysis, err := roiEngine.PerformComprehensiveAnalysis(comprehensiveReq)
    if err != nil {
        log.Printf("Error performing ROI analysis: %v", err)
        http.Error(w, "Analysis service unavailable", http.StatusServiceUnavailable)
        return
    }
    
    // Create simplified response for backward compatibility
    response := map[string]interface{}{
        "status":  "success",
        "analysis_id": analysis.AnalysisID,
        "execution_time_ms": analysis.ExecutionTime,
        "analysis": map[string]interface{}{
            "roi_score":        analysis.ROIAnalysis.ROIPercentage,
            "recommendation":   getRecommendationFromRating(analysis.Summary.OverallRating),
            "payback_months":   analysis.ROIAnalysis.PaybackMonths,
            "risk_level":       analysis.ROIAnalysis.RiskLevel,
            "market_size":      formatMarketSize(analysis.MarketResearch.MarketSize),
            "confidence":       analysis.ROIAnalysis.ConfidenceScore,
            "estimated_revenue": analysis.ROIAnalysis.EstimatedRevenue,
            "breakeven_month":  analysis.FinancialMetrics.BreakevenMonth,
            "competitive_score": analysis.Competitive.CompetitiveScore,
        },
        "comprehensive_analysis": analysis, // Full analysis results
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// New comprehensive analysis endpoint for full results
func comprehensiveAnalysisHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req ComprehensiveAnalysisRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    analysis, err := roiEngine.PerformComprehensiveAnalysis(&req)
    if err != nil {
        log.Printf("Error performing comprehensive analysis: %v", err)
        http.Error(w, "Analysis service unavailable", http.StatusServiceUnavailable)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(analysis)
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

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start roi-fit-analysis

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

    // Initialize database connection
    if err := initDB(); err != nil {
        log.Printf("Warning: Database connection failed: %v", err)
        log.Printf("Continuing with mock data...")
    } else {
        log.Println("🎉 Database connection pool established successfully!")
        defer db.Close()
        
        // Initialize ROI analysis engine
        roiEngine = NewROIAnalysisEngine(db)
        log.Println("ROI Analysis Engine initialized")
    }
    
    // Get port from environment - REQUIRED, no defaults
    port := os.Getenv("API_PORT")
    if port == "" {
        log.Fatal("❌ API_PORT environment variable is required")
    }

    // Original endpoints (backward compatibility)
    http.HandleFunc("/analyze", corsMiddleware(analyzeHandler))
    http.HandleFunc("/opportunities", corsMiddleware(opportunitiesHandler))
    http.HandleFunc("/reports", corsMiddleware(reportsHandler))
    http.HandleFunc("/health", corsMiddleware(healthHandler))
    
    // New comprehensive analysis endpoint
    http.HandleFunc("/comprehensive-analysis", corsMiddleware(comprehensiveAnalysisHandler))
    http.HandleFunc("/analysis/results", corsMiddleware(analysisResultsHandler))

    log.Printf("ROI Fit Analysis API starting on port %s", port)
    log.Println("Endpoints available:")
    log.Println("  POST /analyze (legacy)")
    log.Println("  POST /comprehensive-analysis (full analysis)")
    log.Println("  GET  /opportunities")
    log.Println("  GET  /reports")
    log.Println("  GET  /analysis/results")
    log.Println("  GET  /health")
    
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

// getEnv removed to prevent hardcoded defaults

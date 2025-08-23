package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
)

type Feature struct {
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Reach       int     `json:"reach"`
    Impact      int     `json:"impact"`
    Confidence  float64 `json:"confidence"`
    Effort      int     `json:"effort"`
    Priority    string  `json:"priority"`
    Score       float64 `json:"score"`
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "9200"
    }

    // Enable CORS
    http.HandleFunc("/health", corsMiddleware(healthHandler))
    http.HandleFunc("/api/features", corsMiddleware(featuresHandler))
    http.HandleFunc("/api/dashboard", corsMiddleware(dashboardHandler))
    http.HandleFunc("/api/prioritize", corsMiddleware(prioritizeHandler))

    fmt.Printf("Product Manager API server starting on port %s\n", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

func corsMiddleware(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        handler(w, r)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "service": "product-manager-api",
    })
}

func featuresHandler(w http.ResponseWriter, r *http.Request) {
    // Mock features for demonstration
    features := []Feature{
        {
            Name:        "Dark Mode Support",
            Description: "Implement system-wide dark mode",
            Reach:       5000,
            Impact:      4,
            Confidence:  0.9,
            Effort:      5,
            Priority:    "CRITICAL",
            Score:       7.2,
        },
        {
            Name:        "API Rate Limiting",
            Description: "Prevent API abuse",
            Reach:       3000,
            Impact:      5,
            Confidence:  0.95,
            Effort:      4,
            Priority:    "HIGH",
            Score:       8.9,
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(features)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    dashboard := map[string]interface{}{
        "metrics": map[string]interface{}{
            "active_features": 24,
            "sprint_progress": 87,
            "avg_priority_score": 4.2,
            "projected_roi": 125000,
        },
        "recent_decisions": []string{
            "Migrate to TypeScript",
            "Implement WebSocket support",
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(dashboard)
}

func prioritizeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var features []Feature
    if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Calculate RICE scores
    for i := range features {
        score := float64(features[i].Reach*features[i].Impact) * features[i].Confidence / float64(features[i].Effort)
        features[i].Score = score
        
        // Assign priority based on score
        if score > 1000 {
            features[i].Priority = "CRITICAL"
        } else if score > 500 {
            features[i].Priority = "HIGH"
        } else if score > 100 {
            features[i].Priority = "MEDIUM"
        } else {
            features[i].Priority = "LOW"
        }
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(features)
}
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
)

type HealthResponse struct {
    Status    string    `json:"status"`
    Service   string    `json:"service"`
    Timestamp time.Time `json:"timestamp"`
}

type SearchRequest struct {
    Query    string  `json:"query"`
    Lat      float64 `json:"lat"`
    Lon      float64 `json:"lon"`
    Radius   float64 `json:"radius"`
    Category string  `json:"category"`
}

type Place struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Address     string   `json:"address"`
    Category    string   `json:"category"`
    Distance    float64  `json:"distance"`
    Rating      float64  `json:"rating"`
    PriceLevel  int      `json:"price_level"`
    OpenNow     bool     `json:"open_now"`
    Photos      []string `json:"photos"`
    Description string   `json:"description"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:    "healthy",
        Service:   "local-info-scout",
        Timestamp: time.Now(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req SearchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Mock response for now
    places := []Place{
        {
            ID:          "1",
            Name:        "Green Garden Cafe",
            Address:     "123 Main St",
            Category:    "restaurant",
            Distance:    0.5,
            Rating:      4.5,
            PriceLevel:  2,
            OpenNow:     true,
            Description: "Cozy vegan restaurant with organic options",
        },
        {
            ID:          "2",
            Name:        "Nature's Bounty Market",
            Address:     "456 Oak Ave",
            Category:    "grocery",
            Distance:    0.8,
            Rating:      4.2,
            PriceLevel:  2,
            OpenNow:     true,
            Description: "Local organic grocery store",
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(places)
}

func categoriesHandler(w http.ResponseWriter, r *http.Request) {
    categories := []string{
        "restaurants",
        "grocery",
        "pharmacy",
        "parks",
        "shopping",
        "entertainment",
        "services",
        "fitness",
        "healthcare",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(categories)
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
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
	port := getEnv("API_PORT", getEnv("PORT", ""))
    
    http.HandleFunc("/health", enableCORS(healthHandler))
    http.HandleFunc("/api/search", enableCORS(searchHandler))
    http.HandleFunc("/api/categories", enableCORS(categoriesHandler))
    
    log.Printf("Local Info Scout API starting on port %s", port)
    if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
        log.Fatal(err)
    }
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

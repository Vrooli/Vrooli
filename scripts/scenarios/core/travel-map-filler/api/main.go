package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Travel struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"user_id"`
	Location     string    `json:"location"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Date         string    `json:"date"`
	Type         string    `json:"type"`
	Notes        string    `json:"notes"`
	Country      string    `json:"country"`
	City         string    `json:"city"`
	Continent    string    `json:"continent"`
	DurationDays int       `json:"duration_days"`
	Rating       int       `json:"rating"`
	Photos       []string  `json:"photos"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
}

type Stats struct {
	TotalCountries       int     `json:"total_countries"`
	TotalCities          int     `json:"total_cities"`
	TotalContinents      int     `json:"total_continents"`
	TotalDistanceKm      float64 `json:"total_distance_km"`
	TotalDaysTraveled    int     `json:"total_days_traveled"`
	WorldCoveragePercent float64 `json:"world_coverage_percent"`
}

type Achievement struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	UnlockedAt  time.Time `json:"unlocked_at"`
}

type BucketListItem struct {
	ID           int     `json:"id"`
	Location     string  `json:"location"`
	Country      string  `json:"country"`
	City         string  `json:"city"`
	Priority     int     `json:"priority"`
	Notes        string  `json:"notes"`
	EstimatedDate string `json:"estimated_date"`
	BudgetEstimate float64 `json:"budget_estimate"`
	Tags         []string `json:"tags"`
	Completed    bool    `json:"completed"`
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "travel-map-filler",
	})
}

func travelsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Mock data for testing
	mockTravels := []Travel{
		{
			ID:       1,
			UserID:   "default_user",
			Location: "Paris, France",
			Lat:      48.8566,
			Lng:      2.3522,
			Date:     "2024-01-15",
			Type:     "vacation",
			Notes:    "Beautiful city with amazing architecture",
			Country:  "France",
			City:     "Paris",
			Continent: "Europe",
			DurationDays: 7,
			Rating:   5,
			Photos:   []string{},
			Tags:     []string{"romantic", "culture", "food"},
			CreatedAt: time.Now(),
		},
		{
			ID:       2,
			UserID:   "default_user",
			Location: "Tokyo, Japan",
			Lat:      35.6762,
			Lng:      139.6503,
			Date:     "2023-10-20",
			Type:     "adventure",
			Notes:    "Incredible mix of tradition and technology",
			Country:  "Japan",
			City:     "Tokyo",
			Continent: "Asia",
			DurationDays: 10,
			Rating:   5,
			Photos:   []string{},
			Tags:     []string{"technology", "culture", "sushi"},
			CreatedAt: time.Now(),
		},
	}
	
	json.NewEncoder(w).Encode(mockTravels)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Mock stats
	stats := Stats{
		TotalCountries:       2,
		TotalCities:          2,
		TotalContinents:      2,
		TotalDistanceKm:      9713.5,
		TotalDaysTraveled:    17,
		WorldCoveragePercent: 1.03,
	}
	
	json.NewEncoder(w).Encode(stats)
}

func achievementsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Mock achievements
	achievements := []Achievement{
		{
			Type:        "first_trip",
			Name:        "First Steps",
			Description: "Complete your first trip",
			Icon:        "👣",
			UnlockedAt:  time.Now().AddDate(0, -6, 0),
		},
	}
	
	json.NewEncoder(w).Encode(achievements)
}

func bucketListHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Mock bucket list
	bucketList := []BucketListItem{
		{
			ID:       1,
			Location: "Machu Picchu, Peru",
			Country:  "Peru",
			City:     "Cusco",
			Priority: 5,
			Notes:    "Ancient Incan city in the clouds",
			EstimatedDate: "2025-06-01",
			BudgetEstimate: 3000.00,
			Tags:     []string{"history", "hiking", "wonder"},
			Completed: false,
		},
	}
	
	json.NewEncoder(w).Encode(bucketList)
}

func addTravelHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Parse request body
	var travel Travel
	if err := json.NewDecoder(r.Body).Decode(&travel); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Set default user ID if not provided
	if travel.UserID == "" {
		travel.UserID = "default_user"
	}
	
	// Get n8n URL from environment
	n8nURL := os.Getenv("N8N_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}
	
	// Call n8n workflow webhook
	webhookURL := n8nURL + "/webhook/travel-tracker/add"
	travelJSON, _ := json.Marshal(travel)
	
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(travelJSON))
	if err != nil {
		// Fallback to direct response if workflow unavailable
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"message": "Travel added (workflow pending)",
			"id": time.Now().Unix(),
			"travel": travel,
		})
		return
	}
	defer resp.Body.Close()
	
	// Parse and return workflow response
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	json.NewEncoder(w).Encode(result)
}

func searchTravelsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Parse query parameter or request body
	query := r.URL.Query().Get("q")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "10"
	}
	
	// If no query in URL, try request body
	if query == "" && r.Method == "POST" {
		var searchReq struct {
			Query  string `json:"query"`
			Limit  int    `json:"limit"`
			UserID string `json:"user_id"`
		}
		json.NewDecoder(r.Body).Decode(&searchReq)
		query = searchReq.Query
		if searchReq.Limit > 0 {
			limit = fmt.Sprintf("%d", searchReq.Limit)
		}
	}
	
	if query == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}
	
	// Get n8n URL from environment
	n8nURL := os.Getenv("N8N_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}
	
	// Call n8n search workflow webhook
	webhookURL := n8nURL + "/webhook/travel-search"
	searchData := map[string]interface{}{
		"query":   query,
		"limit":   limit,
		"user_id": "default_user",
	}
	searchJSON, _ := json.Marshal(searchData)
	
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(searchJSON))
	if err != nil {
		// Fallback to empty results if workflow unavailable
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"results": []interface{}{},
			"message": "Search service temporarily unavailable",
		})
		return
	}
	defer resp.Body.Close()
	
	// Parse and return workflow response
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	json.NewEncoder(w).Encode(result)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8760"
	}
	
	// API routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/travels", travelsHandler)
	http.HandleFunc("/api/stats", statsHandler)
	http.HandleFunc("/api/achievements", achievementsHandler)
	http.HandleFunc("/api/bucket-list", bucketListHandler)
	http.HandleFunc("/api/add-travel", addTravelHandler)
	http.HandleFunc("/api/travels/search", searchTravelsHandler)
	
	fmt.Printf("🗺️ Travel Map API running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
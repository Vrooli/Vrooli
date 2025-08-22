package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Status:  "healthy",
		Message: "Fall Foliage Explorer API is running",
	})
}

func regionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Mock regions data for now
	regions := []map[string]interface{}{
		{"id": 1, "name": "White Mountains", "state": "New Hampshire"},
		{"id": 2, "name": "Green Mountains", "state": "Vermont"},
		{"id": 3, "name": "Adirondacks", "state": "New York"},
	}
	
	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   regions,
	})
}

func foliageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	regionID := r.URL.Query().Get("region_id")
	if regionID == "" {
		regionID = "1"
	}
	
	// Mock foliage data
	foliageData := map[string]interface{}{
		"region_id":         regionID,
		"current_status":    "near_peak",
		"color_intensity":   7,
		"predicted_peak":    "2025-10-15",
		"confidence_score":  0.85,
	}
	
	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   foliageData,
	})
}

func predictHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: "Prediction triggered",
		Data:    request,
	})
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
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8920"
	}

	http.HandleFunc("/health", enableCORS(healthHandler))
	http.HandleFunc("/api/regions", enableCORS(regionsHandler))
	http.HandleFunc("/api/foliage", enableCORS(foliageHandler))
	http.HandleFunc("/api/predict", enableCORS(predictHandler))

	log.Printf("Fall Foliage Explorer API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
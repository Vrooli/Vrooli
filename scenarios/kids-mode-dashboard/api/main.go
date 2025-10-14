package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":    "healthy",
		"service":   "kids-mode-dashboard",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	html := `&lt;!DOCTYPE html&gt;
&lt;html&gt;
&lt;head&gt;&lt;title&gt;Kids Mode Dashboard&lt;/title&gt;&lt;/head&gt;
&lt;body&gt;&lt;h1&gt;Welcome to Kids Mode Dashboard&lt;/h1&gt;&lt;/body&gt;
&lt;/html&gt;`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func main() {
	// Health check at root level (required by orchestration)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", dashboardHandler)
	log.Println("Starting kids-mode-dashboard server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
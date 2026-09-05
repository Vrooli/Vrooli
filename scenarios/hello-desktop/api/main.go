package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "23100"
	}
	mux := http.NewServeMux()
	var lastGreeting struct {
		sync.RWMutex
		name    string
		message string
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The desktop runtime's /health contract accepts "healthy" or "degraded".
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	mux.HandleFunc("/api/greet", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Desktop User"
		}
		message := "Hello, " + name + "!"
		lastGreeting.Lock()
		lastGreeting.name, lastGreeting.message = name, message
		lastGreeting.Unlock()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
	})
	// Test-only bridge for the canonical fixture journey. The journey supplies
	// a run-specific name, so a stale response cannot satisfy the assertion.
	mux.HandleFunc("/api/test/last-greeting", func(w http.ResponseWriter, r *http.Request) {
		expected := r.URL.Query().Get("name")
		lastGreeting.RLock()
		name, message := lastGreeting.name, lastGreeting.message
		lastGreeting.RUnlock()
		if expected != "" && expected != name {
			http.Error(w, "greeting not observed", http.StatusNotFound)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"name": name, "message": message})
	})
	log.Printf("hello-desktop API listening on 127.0.0.1:%s", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, mux))
}

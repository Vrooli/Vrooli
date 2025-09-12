package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received: %s %s", r.Method, r.URL.Path)
		
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// Handle OPTIONS
		if r.Method == "OPTIONS" {
			log.Printf("Handling OPTIONS for %s", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Simple response
		if r.URL.Path == "/test" {
			fmt.Fprintf(w, "Test endpoint works!")
			return
		}
		
		http.Error(w, "Not found", 404)
	})
	
	log.Printf("Test CORS server starting on :8889")
	if err := http.ListenAndServe(":8889", handler); err != nil {
		log.Fatal(err)
	}
}
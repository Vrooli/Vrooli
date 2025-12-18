package main

import (
	"github.com/vrooli/api-core/preflight"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "kids-mode-dashboard",
	}) {
		return // Process was re-exec'd after rebuild
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", dashboardHandler)

	apiPort := strings.TrimSpace(os.Getenv("API_PORT"))
	if apiPort == "" {
		apiPort = "8080"
	}

	uiPort := strings.TrimSpace(os.Getenv("UI_PORT"))
	if uiPort == "" {
		uiPort = apiPort
	}

	if uiPort != apiPort {
		go func() {
			log.Printf("Starting kids-mode-dashboard UI server on :%s", uiPort)
			if err := http.ListenAndServe(":"+uiPort, mux); err != nil {
				log.Printf("Kids-mode-dashboard UI server stopped: %v", err)
			}
		}()
	}

	log.Printf("Starting kids-mode-dashboard API server on :%s", apiPort)
	log.Fatal(http.ListenAndServe(":"+apiPort, mux))
}

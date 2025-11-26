package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var (
	veganDB *VeganDatabase
	cache   *CacheClient
	logger  *slog.Logger
)

func init() {
	// Initialize structured logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Initialize the vegan database
	veganDB = InitVeganDatabase()

	// Initialize cache client
	cache = NewCacheClient()
}

// Handlers are defined in handlers.go to maintain modularity

func main() {
	// NOTE: VROOLI_LIFECYCLE_MANAGED validation ensures proper environment setup
	// The lifecycle system provides critical variables: API_PORT, UI_PORT, etc.
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start make-it-vegan

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/check", checkIngredients).Methods("POST")
	router.HandleFunc("/api/substitute", findSubstitute).Methods("POST")
	router.HandleFunc("/api/veganize", veganizeRecipe).Methods("POST")
	router.HandleFunc("/api/products", getCommonProducts).Methods("GET")
	router.HandleFunc("/api/nutrition", getNutrition).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	port := getEnv("API_PORT", getEnv("PORT", ""))

	slog.Info("Make It Vegan API starting",
		"service", "make-it-vegan",
		"action", "start",
		"port", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("Server failed to start", "error", err, "port", port)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

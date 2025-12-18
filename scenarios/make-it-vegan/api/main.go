package main

import (
	"github.com/vrooli/api-core/preflight"
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "make-it-vegan",
	}) {
		return // Process was re-exec'd after rebuild
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

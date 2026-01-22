// Package main is the entry point for the prompt-manager API server.
// This file is intentionally thin - it only handles server bootstrap and wiring.
// All business logic lives in domain packages: prompts/, metrics/, tags/, testing/.
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"prompt-manager/avatars"
	"prompt-manager/metrics"
	"prompt-manager/prompts"
	"prompt-manager/tags"
	"prompt-manager/testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	// Preflight checks
	if preflight.Run(preflight.Config{
		ScenarioName: "prompt-manager",
	}) {
		return
	}

	// Configuration from environment
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Println("OLLAMA_URL not provided - prompt testing will be disabled")
	}

	promptsDir := filepath.Join("..", "prompts")
	if envDir := os.Getenv("PROMPTS_DIR"); envDir != "" {
		promptsDir = envDir
	}

	// Connect to database
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Set search path
	if _, err = db.Exec("SET search_path TO public"); err != nil {
		log.Fatal("Failed to set search_path:", err)
	}

	// Initialize domain components (seams for testing)
	promptStore := prompts.NewStore(promptsDir)
	metricsRepo := metrics.NewRepository(db)
	tagsRepo := tags.NewRepository(db)
	testingRepo := testing.NewRepository(db)
	ollamaClient := testing.NewOllamaClient(ollamaURL)

	// Initialize handlers with interface adapters
	metricsAdapter := prompts.NewMetricsAdapter(metricsRepo)
	promptHandlers := prompts.NewHandlers(promptStore, metricsAdapter)
	tagsHandlers := tags.NewHandlers(tagsRepo)
	testingHandlers := testing.NewHandlers(testingRepo, ollamaClient, promptStore)

	// Avatar store and handlers
	avatarDataDir := filepath.Join(promptsDir, "data")
	avatarStore := avatars.NewStore(avatarDataDir)
	avatarHandlers := avatars.NewHandlers(avatarStore)

	// Setup routes
	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health check
	healthHandler := health.New().Version("2.0.0").Check(health.DB(db), health.Critical).Handler()
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/health", healthHandler).Methods("GET")

	// Prompt routes
	v1.HandleFunc("/prompts", promptHandlers.List).Methods("GET")
	v1.HandleFunc("/prompts/sync", promptHandlers.Sync).Methods("GET")
	v1.HandleFunc("/prompts/combine", promptHandlers.Combine).Methods("POST")
	v1.HandleFunc("/prompts", promptHandlers.Create).Methods("POST")
	v1.HandleFunc("/prompts/{id}", promptHandlers.Get).Methods("GET")
	v1.HandleFunc("/prompts/{id}", promptHandlers.Update).Methods("PUT")
	v1.HandleFunc("/prompts/{id}", promptHandlers.Delete).Methods("DELETE")

	// Usage tracking routes (part of prompts domain)
	v1.HandleFunc("/prompts/{id}/use", promptHandlers.RecordUsage).Methods("POST")
	v1.HandleFunc("/prompts/{id}/rating", promptHandlers.SetRating).Methods("PUT")

	// Tags routes
	v1.HandleFunc("/tags", tagsHandlers.List).Methods("GET")
	v1.HandleFunc("/tags", tagsHandlers.Create).Methods("POST")

	// Testing routes
	v1.HandleFunc("/prompts/{id}/test", testingHandlers.Test).Methods("POST")
	v1.HandleFunc("/prompts/{id}/test-history", testingHandlers.GetHistory).Methods("GET")

	// Avatar routes
	v1.HandleFunc("/avatars", avatarHandlers.List).Methods("GET")
	v1.HandleFunc("/avatars", avatarHandlers.Create).Methods("POST")
	v1.HandleFunc("/avatars/{id}", avatarHandlers.Get).Methods("GET")
	v1.HandleFunc("/avatars/{id}", avatarHandlers.Update).Methods("PUT")
	v1.HandleFunc("/avatars/{id}", avatarHandlers.Delete).Methods("DELETE")

	log.Printf("Prompt Manager API v2.0 starting")
	log.Printf("Prompts directory: %s", promptsDir)
	if ollamaURL != "" {
		log.Printf("Ollama: %s", ollamaURL)
	}

	handler := corsHandler(router)
	if err := server.Run(server.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			if db != nil {
				return db.Close()
			}
			return nil
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

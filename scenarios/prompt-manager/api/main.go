// Package main is the entry point for the prompt-manager API server.
// This file is intentionally thin - it only handles server bootstrap and wiring.
// All business logic lives in domain packages: skills/, metrics/, tags/, testing/.
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"prompt-manager/members"
	"prompt-manager/metrics"
	"prompt-manager/ogmeta"
	"prompt-manager/search"
	"prompt-manager/skills"
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
		log.Println("OLLAMA_URL not provided - skill testing will be disabled")
	}

	skillsDir := filepath.Join("..", "skills")
	if envDir := os.Getenv("SKILLS_DIR"); envDir != "" {
		skillsDir = envDir
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
	skillStore := skills.NewStore(skillsDir)
	metricsRepo := metrics.NewRepository(db)
	tagsRepo := tags.NewRepository(db)
	testingRepo := testing.NewRepository(db)
	ollamaClient := testing.NewOllamaClient(ollamaURL)

	// Initialize handlers with interface adapters
	metricsAdapter := skills.NewMetricsAdapter(metricsRepo)
	skillHandlers := skills.NewHandlers(skillStore, metricsAdapter)
	tagsHandlers := tags.NewHandlers(tagsRepo)
	testingHandlers := testing.NewHandlers(testingRepo, ollamaClient, skillStore)

	// Member store and handlers
	memberDataDir := filepath.Join(skillsDir, "data")
	memberStore := members.NewStore(memberDataDir)
	memberHandlers := members.NewHandlers(memberStore)

	// OG metadata handlers
	ogmetaHandlers := ogmeta.NewHandlers()

	// Search service and handlers
	searchService := search.NewService(skillStore)
	searchHandlers := search.NewHandlers(searchService)

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

	// Skill routes
	v1.HandleFunc("/skills", skillHandlers.List).Methods("GET")
	v1.HandleFunc("/skills/sync", skillHandlers.Sync).Methods("GET")
	v1.HandleFunc("/skills/combine", skillHandlers.Combine).Methods("POST")
	v1.HandleFunc("/skills", skillHandlers.Create).Methods("POST")
	v1.HandleFunc("/skills/{id}", skillHandlers.Get).Methods("GET")
	v1.HandleFunc("/skills/{id}", skillHandlers.Update).Methods("PUT")
	v1.HandleFunc("/skills/{id}", skillHandlers.Delete).Methods("DELETE")

	// Version history routes (part of skills domain)
	v1.HandleFunc("/skills/{id}/versions", skillHandlers.GetVersions).Methods("GET")
	v1.HandleFunc("/skills/{id}/revert/{version}", skillHandlers.RevertToVersion).Methods("POST")

	// Usage tracking routes (part of skills domain)
	v1.HandleFunc("/skills/{id}/use", skillHandlers.RecordUsage).Methods("POST")
	v1.HandleFunc("/skills/{id}/rating", skillHandlers.SetRating).Methods("PUT")

	// Search routes
	v1.HandleFunc("/search/skills", searchHandlers.Search).Methods("GET")

	// Tags routes
	v1.HandleFunc("/tags", tagsHandlers.List).Methods("GET")
	v1.HandleFunc("/tags", tagsHandlers.Create).Methods("POST")

	// Testing routes
	v1.HandleFunc("/skills/{id}/test", testingHandlers.Test).Methods("POST")
	v1.HandleFunc("/skills/{id}/test-history", testingHandlers.GetHistory).Methods("GET")

	// Member routes
	v1.HandleFunc("/members", memberHandlers.List).Methods("GET")
	v1.HandleFunc("/members", memberHandlers.Create).Methods("POST")
	v1.HandleFunc("/members/{id}", memberHandlers.Get).Methods("GET")
	v1.HandleFunc("/members/{id}", memberHandlers.Update).Methods("PUT")
	v1.HandleFunc("/members/{id}", memberHandlers.Delete).Methods("DELETE")

	// OG metadata routes (for link previews)
	v1.HandleFunc("/og-metadata", ogmetaHandlers.Get).Methods("GET")

	log.Printf("Prompt Manager API v2.0 starting")
	log.Printf("Skills directory: %s", skillsDir)
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

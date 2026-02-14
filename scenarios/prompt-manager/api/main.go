// Package main is the entry point for the prompt-manager API server.
// This file is intentionally thin - it only handles server bootstrap and wiring.
// All business logic lives in domain packages: skills/, metrics/, tags/, testing/.
//
// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/reference/api-endpoints.md
// DOC: docs/internal/SEAMS.md#dependency-wiring-in-maingo
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"prompt-manager/agents"
	"prompt-manager/aisearch"
	"prompt-manager/graph"
	"prompt-manager/heartbeat"
	"prompt-manager/metrics"
	"prompt-manager/ogmeta"
	"prompt-manager/search"
	"prompt-manager/skills"
	"prompt-manager/store"
	"prompt-manager/tags"
	"prompt-manager/teams"
	"prompt-manager/templates"
	"prompt-manager/testing"
	"prompt-manager/worldscale"
	"prompt-manager/worldseats"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// discoverScenarioNames returns the names of all scenario directories.
// storeDir is expected to be an absolute path like ".../scenarios/prompt-manager/store";
// we walk up to the "scenarios/" parent and list its subdirectories.
func discoverScenarioNames(storeDir string) []string {
	scenariosDir := filepath.Join(storeDir, "..", "..")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		log.Printf("Warning: could not read scenarios dir %s: %v", scenariosDir, err)
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names
}

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

	// Storage configuration
	// The new storage uses store/ directory with per-entity files
	storeDir := filepath.Join("..", "store")
	if envDir := os.Getenv("STORE_DIR"); envDir != "" {
		storeDir = envDir
	}

	// Resolve to absolute path for consistent file path reporting
	absStoreDir, err := filepath.Abs(storeDir)
	if err != nil {
		log.Printf("Warning: Could not resolve absolute path for store dir: %v", err)
		absStoreDir = storeDir
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

	// Initialize the new file-based store
	fileStore := store.NewFileStore(storeDir)

	// Initialize domain components (seams for testing)
	// Use the store adapter to bridge new storage to existing handlers
	skillStoreAdapter := skills.NewStoreAdapter(fileStore.FileSkills(), store.NewFileContentIO())
	metricsRepo := metrics.NewRepository(db)
	tagsRepo := tags.NewRepository(db)
	testingRepo := testing.NewRepository(db)
	ollamaClient := testing.NewOllamaClient(ollamaURL)

	// Initialize handlers with interface adapters
	metricsAdapter := skills.NewMetricsAdapter(metricsRepo)
	skillHandlers := skills.NewHandlers(skillStoreAdapter, metricsAdapter, absStoreDir)
	tagsHandlers := tags.NewHandlers(tagsRepo)
	testingHandlers := testing.NewHandlers(testingRepo, ollamaClient, skillStoreAdapter)
	templateHandlers := templates.NewHandlers(templates.NewStore(absStoreDir))

	// Agent handlers (new storage-backed, replaces member handlers)
	agentHandlers := agents.NewHandlers(fileStore.Agents(), fileStore.Indexes(), absStoreDir, fileStore.Relations(), fileStore.Teams())

	// OG metadata handlers
	ogmetaHandlers := ogmeta.NewHandlers()

	// Search service and handlers
	searchService := search.NewService(skillStoreAdapter)
	searchHandlers := search.NewHandlers(searchService)

	// AI Search service (graceful degradation when unavailable)
	qdrantURL := os.Getenv("QDRANT_URL")
	qdrantAPIKey := os.Getenv("QDRANT_API_KEY")

	aiSearchCollection := os.Getenv("AI_SEARCH_COLLECTION")
	if aiSearchCollection == "" {
		aiSearchCollection = "prompt-manager-skills"
	}

	aiSearchThreshold := 0.5
	if thresholdStr := os.Getenv("AI_SEARCH_THRESHOLD"); thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			aiSearchThreshold = parsed
		}
	}

	// Initialize AI search components
	embedder := aisearch.NewEmbedder(ollamaURL, "nomic-embed-text")
	vectorStore := aisearch.NewVectorStore(qdrantURL, qdrantAPIKey, aiSearchCollection, 768)
	aiSearchService := aisearch.NewService(embedder, vectorStore, skillStoreAdapter, searchService, aiSearchThreshold)
	aiSearchHandlers := aisearch.NewHandlers(aiSearchService)

	// Set AI indexer on skill handlers for CRUD hook integration
	skillHandlers.SetAIIndexer(aiSearchService)

	// Graph detection
	scenarioNames := discoverScenarioNames(absStoreDir)
	cliDetector := graph.NewCLIDetector(scenarioNames)
	graphScanner := graph.NewScanner(
		fileStore.Agents().(*store.FileAgentStore),
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.FileSkills(),
		fileStore.Relations(),
		cliDetector,
	)
	graphBuilder := graph.NewBuilder(
		fileStore.Agents().(*store.FileAgentStore),
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.FileSkills(),
		graphScanner,
		graph.DefaultScoreFns(),
	)
	graphHealthConfigStore := graph.NewHealthConfigStore(absStoreDir)
	graphBuilder.SetHealthConfigProvider(graphHealthConfigStore)
	graphBuilder.SetScenarioHealthProvider(graph.NewScenarioCompletenessCLIProvider(15 * time.Second))
	graphIndex := graph.NewIndexStore(absStoreDir, graphBuilder)
	// Always regenerate on startup so the index reflects the current detection code.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := graphIndex.Regenerate(ctx); err != nil {
			log.Printf("graph: startup rebuild failed: %v", err)
		}
	}()
	graphHandlers := graph.NewHandlers(graphIndex, graphHealthConfigStore)

	// Inject graph invalidator into mutation handlers
	skillHandlers.SetGraphInvalidator(graphIndex)
	agentHandlers.SetGraphInvalidator(graphIndex)

	// Log AI search status and trigger startup indexing if available
	if ollamaURL != "" && qdrantURL != "" {
		log.Printf("AI Search: Ollama=%s, Qdrant=%s, Collection=%s", ollamaURL, qdrantURL, aiSearchCollection)
		// Check availability and index if needed (async to not block startup)
		go func() {
			ctx := context.Background()
			if !aiSearchService.Available(ctx) {
				log.Println("AI Search: Resources not reachable at startup, skipping initial index")
				return
			}
			// Ensure collection exists
			if err := vectorStore.EnsureCollection(ctx); err != nil {
				log.Printf("AI Search: Failed to ensure collection: %v", err)
				return
			}
			// Check if index is empty
			count, err := vectorStore.CountPoints(ctx)
			if err != nil {
				log.Printf("AI Search: Failed to count points: %v", err)
				return
			}
			if count == 0 {
				log.Println("AI Search: Index empty, starting initial indexing...")
				status, started := aiSearchService.StartReindex()
				if started {
					log.Printf("AI Search: Initial indexing started at %s", status.StartedAt)
				} else {
					log.Printf("AI Search: Reindex already running (started at %s)", status.StartedAt)
				}
			} else {
				log.Printf("AI Search: Index contains %d skills", count)
			}
		}()
	} else {
		log.Println("AI Search: Resources not fully configured (will gracefully degrade to text search)")
	}

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
	v1.HandleFunc("/skills", skillHandlers.Create).Methods("POST")
	v1.HandleFunc("/skills/read", skillHandlers.Read).Methods("POST")
	v1.HandleFunc("/skills/{id}", skillHandlers.Get).Methods("GET")
	v1.HandleFunc("/skills/{id}", skillHandlers.Update).Methods("PUT")
	v1.HandleFunc("/skills/{id}", skillHandlers.Delete).Methods("DELETE")

	// Version history routes (part of skills domain)
	v1.HandleFunc("/skills/{id}/versions", skillHandlers.GetVersions).Methods("GET")
	v1.HandleFunc("/skills/{id}/revert/{version}", skillHandlers.RevertToVersion).Methods("POST")

	// Graph routes
	v1.HandleFunc("/graph", graphHandlers.GetGraph).Methods("GET")
	v1.HandleFunc("/graph/regenerate", graphHandlers.Regenerate).Methods("POST")
	v1.HandleFunc("/graph/orphans", graphHandlers.GetOrphans).Methods("GET")
	v1.HandleFunc("/graph/skillless", graphHandlers.GetSkillless).Methods("GET")
	v1.HandleFunc("/graph/empty-teams", graphHandlers.GetEmptyTeams).Methods("GET")
	v1.HandleFunc("/graph/unaffiliated", graphHandlers.GetUnaffiliated).Methods("GET")
	v1.HandleFunc("/graph/popular", graphHandlers.GetPopular).Methods("GET")
	v1.HandleFunc("/graph/cycles", graphHandlers.GetCycles).Methods("GET")
	v1.HandleFunc("/graph/health", graphHandlers.GetHealthScores).Methods("GET")
	v1.HandleFunc("/graph/health-config", graphHandlers.GetHealthConfig).Methods("GET")
	v1.HandleFunc("/graph/health-config", graphHandlers.PutHealthConfig).Methods("PUT")
	v1.HandleFunc("/graph/nodes/{id}", graphHandlers.GetNode).Methods("GET")
	v1.HandleFunc("/graph/nodes/{id}/edges", graphHandlers.GetNodeEdges).Methods("GET")

	// Usage tracking routes (part of skills domain)
	v1.HandleFunc("/skills/{id}/use", skillHandlers.RecordUsage).Methods("POST")
	v1.HandleFunc("/skills/{id}/rating", skillHandlers.SetRating).Methods("PUT")

	// Search routes
	v1.HandleFunc("/search/skills", searchHandlers.Search).Methods("GET")
	v1.HandleFunc("/search/skills/content", searchHandlers.ContentSearch).Methods("GET")

	// AI Search routes
	v1.HandleFunc("/search/ai", aiSearchHandlers.Search).Methods("POST")
	v1.HandleFunc("/search/ai/status", aiSearchHandlers.Status).Methods("GET")
	v1.HandleFunc("/search/ai/reindex", aiSearchHandlers.Reindex).Methods("POST")
	v1.HandleFunc("/search/ai/reindex/status", aiSearchHandlers.ReindexStatus).Methods("GET")
	v1.HandleFunc("/search/ai/reindex/cancel", aiSearchHandlers.CancelReindex).Methods("POST")

	// Tags routes
	v1.HandleFunc("/tags", tagsHandlers.List).Methods("GET")
	v1.HandleFunc("/tags", tagsHandlers.Create).Methods("POST")

	// Testing routes
	v1.HandleFunc("/skills/{id}/test", testingHandlers.Test).Methods("POST")
	v1.HandleFunc("/skills/{id}/test-history", testingHandlers.GetHistory).Methods("GET")

	// Agent routes
	v1.HandleFunc("/agents", agentHandlers.List).Methods("GET")
	v1.HandleFunc("/agents", agentHandlers.Create).Methods("POST")
	v1.HandleFunc("/agents/{id}", agentHandlers.Get).Methods("GET")
	v1.HandleFunc("/agents/{id}", agentHandlers.Update).Methods("PUT")
	v1.HandleFunc("/agents/{id}", agentHandlers.Delete).Methods("DELETE")
	v1.HandleFunc("/agents/{id}/soul", agentHandlers.GetSoul).Methods("GET")
	v1.HandleFunc("/agents/{id}/soul", agentHandlers.SetSoul).Methods("PUT")
	v1.HandleFunc("/agents/{id}/files", agentHandlers.ListFiles).Methods("GET")
	v1.HandleFunc("/agents/{id}/files/content", agentHandlers.GetFile).Methods("GET")
	v1.HandleFunc("/agents/{id}/files/content", agentHandlers.SetFile).Methods("PUT")
	v1.HandleFunc("/agents/{id}/files", agentHandlers.CreateFile).Methods("POST")
	v1.HandleFunc("/agents/{id}/files/rename", agentHandlers.RenameFile).Methods("POST")
	v1.HandleFunc("/agents/{id}/teams", agentHandlers.ListTeams).Methods("GET")
	v1.HandleFunc("/agents/{id}/files", agentHandlers.DeleteFile).Methods("DELETE")

	// Agent file templates
	v1.HandleFunc("/agent-file-templates", templateHandlers.ListAgentFileTemplates).Methods("GET")

	// Team routes
	teamHandlers := teams.NewHandlers(fileStore.Teams(), fileStore.Agents(), fileStore.Relations(), fileStore.Indexes(), nil)
	teamHandlers.SetGraphInvalidator(graphIndex)
	// Import routes must come before /teams/{id} to avoid mux treating "import" as an ID
	v1.HandleFunc("/teams/import/claude-code/available", teamHandlers.ListAvailableCCTeams).Methods("GET")
	v1.HandleFunc("/teams/import/claude-code", teamHandlers.ImportClaudeCode).Methods("POST")
	v1.HandleFunc("/teams", teamHandlers.List).Methods("GET")
	v1.HandleFunc("/teams", teamHandlers.Create).Methods("POST")
	v1.HandleFunc("/teams/{id}", teamHandlers.Get).Methods("GET")
	v1.HandleFunc("/teams/{id}", teamHandlers.Update).Methods("PUT")
	v1.HandleFunc("/teams/{id}", teamHandlers.Delete).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/exclusive-members", teamHandlers.GetExclusiveMembers).Methods("GET")
	v1.HandleFunc("/teams/{id}/members", teamHandlers.AddMember).Methods("POST")
	v1.HandleFunc("/teams/{id}/members/{agentId}", teamHandlers.UpdateMember).Methods("PUT")
	v1.HandleFunc("/teams/{id}/members/{agentId}", teamHandlers.RemoveMember).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/roles", teamHandlers.GetRoles).Methods("GET")
	v1.HandleFunc("/teams/{id}/roles", teamHandlers.SetRoles).Methods("PUT")
	v1.HandleFunc("/teams/{id}/shared/files", teamHandlers.ListSharedFiles).Methods("GET")
	v1.HandleFunc("/teams/{id}/shared/files/content", teamHandlers.GetSharedFile).Methods("GET")
	v1.HandleFunc("/teams/{id}/shared/files/content", teamHandlers.SetSharedFile).Methods("PUT")
	v1.HandleFunc("/teams/{id}/shared/files", teamHandlers.CreateSharedFile).Methods("POST")
	v1.HandleFunc("/teams/{id}/shared/files/rename", teamHandlers.RenameSharedFile).Methods("POST")
	v1.HandleFunc("/teams/{id}/shared/files", teamHandlers.DeleteSharedFile).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/org", teamHandlers.GetOrgChart).Methods("GET")
	v1.HandleFunc("/teams/{id}/org", teamHandlers.SetOrgChart).Methods("PUT")
	v1.HandleFunc("/teams/{id}/org/edges/{reportId}", teamHandlers.UpdateOrgChartEdge).Methods("PUT")
	v1.HandleFunc("/teams/{id}/org/edges/{reportId}", teamHandlers.DeleteOrgChartEdge).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/members/{agentId}/messages", teamHandlers.ListTeamMessages).Methods("GET")
	v1.HandleFunc("/teams/{id}/members/{agentId}/messages", teamHandlers.SendTeamMessage).Methods("POST")
	v1.HandleFunc("/teams/{id}/members/{agentId}/messages", teamHandlers.ClearTeamMessages).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/members/{agentId}/messages/{messageId}", teamHandlers.DeleteTeamMessage).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/export/claude-code", teamHandlers.ExportClaudeCode).Methods("GET")

	// Heartbeat system
	// Get Vrooli root for working directory
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		// Default to parent of store dir
		vrooliRoot, _ = filepath.Abs(filepath.Join(storeDir, ".."))
	}

	// Initialize heartbeat components
	agentManagerClient := heartbeat.NewAgentManagerClient(30 * time.Second)
	runRegistry := heartbeat.NewRunRegistry(absStoreDir)
	heartbeatExecutor := heartbeat.NewExecutor(
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.Agents().(*store.FileAgentStore),
		agentManagerClient,
		vrooliRoot,
		runRegistry,
	)
	heartbeatScheduler := heartbeat.NewScheduler(
		heartbeatExecutor,
		agentManagerClient,
		fileStore.Teams().(*store.FileTeamStore),
	)
	heartbeatHandlers := heartbeat.NewHandlers(
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.Agents().(*store.FileAgentStore),
		fileStore.Relations(),
		heartbeatScheduler,
		heartbeatExecutor,
		runRegistry,
		agentManagerClient,
	)
	teamHandlers.SetHeartbeatScheduler(heartbeatScheduler)

	// Recover any active runs from a previous process
	runRegistry.Recover(context.Background(), agentManagerClient)

	// Start scheduler (doesn't auto-start heartbeats - they must be explicitly enabled)
	go func() {
		if err := heartbeatScheduler.Start(context.Background()); err != nil {
			log.Printf("Warning: Failed to start heartbeat scheduler: %v", err)
		}

		// Load enabled heartbeats from all teams
		teams, _ := fileStore.Teams().List(context.Background())
		for _, team := range teams {
			if !team.Enabled {
				continue
			}
			configs, _ := fileStore.Teams().(*store.FileTeamStore).ListHeartbeatConfigs(context.Background(), team.ID)
			for _, config := range configs {
				if config.Enabled {
					if err := heartbeatScheduler.Schedule(config.TeamID, config.AgentID, config.Schedule); err != nil {
						log.Printf("Warning: Failed to schedule heartbeat for %s/%s: %v", config.TeamID, config.AgentID, err)
					}
				}
			}
		}
	}()

	// Heartbeat routes
	v1.HandleFunc("/heartbeats/running", heartbeatHandlers.ListRunning).Methods("GET")
	v1.HandleFunc("/heartbeats/running/{teamId}/{agentId}/stop", heartbeatHandlers.StopRunning).Methods("POST")
	v1.HandleFunc("/prompt-preview", heartbeatHandlers.PreviewPrompt).Methods("POST")
	v1.HandleFunc("/teams/{id}/heartbeats", heartbeatHandlers.ListHeartbeats).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.GetHeartbeat).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.CreateHeartbeat).Methods("POST")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.UpdateHeartbeat).Methods("PUT")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.DeleteHeartbeat).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}/trigger", heartbeatHandlers.TriggerHeartbeat).Methods("POST")
	v1.HandleFunc("/teams/{id}/trigger", heartbeatHandlers.TriggerTeam).Methods("POST")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}/logs", heartbeatHandlers.ListLogs).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}/logs/{logId}", heartbeatHandlers.GetLog).Methods("GET")

	// Member document routes (RESPONSIBILITIES.md and HEARTBEAT.md)
	v1.HandleFunc("/teams/{id}/members/{agentId}/responsibilities", heartbeatHandlers.GetResponsibilities).Methods("GET")
	v1.HandleFunc("/teams/{id}/members/{agentId}/responsibilities", heartbeatHandlers.SetResponsibilities).Methods("PUT")
	v1.HandleFunc("/teams/{id}/members/{agentId}/heartbeat-instructions", heartbeatHandlers.GetHeartbeatInstructions).Methods("GET")
	v1.HandleFunc("/teams/{id}/members/{agentId}/heartbeat-instructions", heartbeatHandlers.SetHeartbeatInstructions).Methods("PUT")

	// World scale routes
	v1.HandleFunc("/world-scale", worldscale.HandleGet(absStoreDir)).Methods("GET")
	v1.HandleFunc("/world-scale", worldscale.HandlePut(absStoreDir)).Methods("PUT")

	// World seats routes
	v1.HandleFunc("/world-seats", worldseats.HandleGet(absStoreDir)).Methods("GET")
	v1.HandleFunc("/world-seats", worldseats.HandlePut(absStoreDir)).Methods("PUT")

	// OG metadata routes (for link previews)
	v1.HandleFunc("/og-metadata", ogmetaHandlers.Get).Methods("GET")

	log.Printf("Prompt Manager API v2.0 starting")
	log.Printf("Store directory: %s", storeDir)
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

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
	"prompt-manager/topics"
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

	// Agent and team search services
	agentSearchService := search.NewAgentSearchService(fileStore.Agents(), fileStore.Agents().(*store.FileAgentStore))
	teamSearchService := search.NewTeamSearchService(fileStore.Teams(), fileStore.Relations(), fileStore.Teams().(*store.FileTeamStore))
	searchHandlers.SetAgentService(agentSearchService)
	searchHandlers.SetTeamService(teamSearchService)

	// AI Search service (graceful degradation when unavailable)
	qdrantURL := resolveQdrantURL()
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

	// Agent and team AI search vector stores
	agentAICollection := os.Getenv("AI_SEARCH_AGENT_COLLECTION")
	if agentAICollection == "" {
		agentAICollection = "prompt-manager-agents"
	}
	agentVectorStore := aisearch.NewVectorStore(qdrantURL, qdrantAPIKey, agentAICollection, 768)

	teamAICollection := os.Getenv("AI_SEARCH_TEAM_COLLECTION")
	if teamAICollection == "" {
		teamAICollection = "prompt-manager-teams"
	}
	teamVectorStore := aisearch.NewVectorStore(qdrantURL, qdrantAPIKey, teamAICollection, 768)

	topicAICollection := os.Getenv("AI_SEARCH_TOPIC_COLLECTION")
	if topicAICollection == "" {
		topicAICollection = "prompt-manager-topics"
	}
	topicVectorStore := aisearch.NewVectorStore(qdrantURL, qdrantAPIKey, topicAICollection, 768)

	// Wire agent/team/topic AI search into the service
	aiSearchService.SetAgentSearch(agentVectorStore, fileStore.Agents().(*store.FileAgentStore), agentSearchService)
	aiSearchService.SetTeamSearch(teamVectorStore, fileStore.Teams().(*store.FileTeamStore), fileStore.Relations(), teamSearchService)
	aiSearchService.SetTopicSearch(topicVectorStore, fileStore.FileTopics())

	// Budget config store
	budgetConfigStore := aisearch.NewBudgetConfigStore(absStoreDir)
	aiSearchService.SetBudgetConfig(budgetConfigStore)
	aiSearchHandlers.SetBudgetConfigStore(budgetConfigStore)

	// Set AI indexer on agent and team handlers for CRUD hook integration
	agentHandlers.SetAIIndexer(aiSearchService)

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
			// Check if index is stale (count mismatch between Qdrant and disk)
			needs, indexed, disk, err := aiSearchService.NeedsReindex(ctx)
			if err != nil {
				log.Printf("AI Search: Failed staleness check: %v", err)
				return
			}
			if needs {
				log.Printf("AI Search: Index out of sync (indexed=%d, on-disk=%d), reindexing...", indexed, disk)
				status, started := aiSearchService.StartReindex()
				if started {
					log.Printf("AI Search: Reindexing started at %s", status.StartedAt)
				} else {
					log.Printf("AI Search: Reindex already running (started at %s)", status.StartedAt)
				}
			} else {
				log.Printf("AI Search: Index up-to-date (%d skills)", indexed)
			}

			// Start periodic sync to catch external file changes and service recovery
			aiSearchService.StartPeriodicSync(ctx, 5*time.Minute)
		}()
	} else {
		log.Println("AI Search: Resources not fully configured (will gracefully degrade to text search)")
	}

	// Setup routes
	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
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
	v1.HandleFunc("/search/agents", searchHandlers.SearchAgents).Methods("GET")
	v1.HandleFunc("/search/agents/content", searchHandlers.AgentContentSearch).Methods("GET")
	v1.HandleFunc("/search/teams", searchHandlers.SearchTeams).Methods("GET")
	v1.HandleFunc("/search/teams/content", searchHandlers.TeamContentSearch).Methods("GET")

	// AI Search routes
	v1.HandleFunc("/search/ai", aiSearchHandlers.Search).Methods("POST")
	v1.HandleFunc("/search/agents/ai", aiSearchHandlers.SearchAgents).Methods("POST")
	v1.HandleFunc("/search/teams/ai", aiSearchHandlers.SearchTeams).Methods("POST")
	v1.HandleFunc("/search/ai/status", aiSearchHandlers.Status).Methods("GET")
	v1.HandleFunc("/search/ai/reindex", aiSearchHandlers.Reindex).Methods("POST")
	v1.HandleFunc("/search/ai/reindex/status", aiSearchHandlers.ReindexStatus).Methods("GET")
	v1.HandleFunc("/search/ai/reindex/cancel", aiSearchHandlers.CancelReindex).Methods("POST")

	// Discovery route (unified topic + skill search)
	v1.HandleFunc("/discover", aiSearchHandlers.Discover).Methods("POST")

	// Budget config routes
	v1.HandleFunc("/config/budgets", aiSearchHandlers.GetBudgetConfig).Methods("GET")
	v1.HandleFunc("/config/budgets", aiSearchHandlers.PutBudgetConfig).Methods("PUT")

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
	teamHandlers.SetAIIndexer(aiSearchService)
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

	// Topic routes
	topicHandlers := topics.NewHandlers(fileStore.Topics(), fileStore.Indexes())
	topicHandlers.SetGraphInvalidator(graphIndex)
	topicHandlers.SetAIIndexer(aiSearchService)
	topicHandlers.SetTopicMatchFn(buildTopicMatchFn(aiSearchService, fileStore.Topics()))
	// Match route must come before /topics/{id} to avoid mux treating "match" as an ID
	v1.HandleFunc("/topics/match", topicHandlers.Match).Methods("POST")
	v1.HandleFunc("/topics", topicHandlers.List).Methods("GET")
	v1.HandleFunc("/topics", topicHandlers.Create).Methods("POST")
	v1.HandleFunc("/topics/{id}", topicHandlers.Get).Methods("GET")
	v1.HandleFunc("/topics/{id}", topicHandlers.Update).Methods("PUT")
	v1.HandleFunc("/topics/{id}", topicHandlers.Delete).Methods("DELETE")
	v1.HandleFunc("/topics/{id}/skills", topicHandlers.AccumulatedSkills).Methods("GET")

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
		nil, // uses default SentinelExtractor
	)
	teamExecStore := heartbeat.NewTeamExecutionStore(heartbeatExecutor, absStoreDir)
	heartbeatExecutor.OnComplete = teamExecStore.OnComplete
	heartbeatScheduler := heartbeat.NewScheduler(
		heartbeatExecutor,
		agentManagerClient,
		fileStore.Teams().(*store.FileTeamStore),
		teamExecStore,
	)
	heartbeatHandlers := heartbeat.NewHandlers(
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.Agents().(*store.FileAgentStore),
		fileStore.Relations(),
		heartbeatScheduler,
		heartbeatExecutor,
		runRegistry,
		agentManagerClient,
		teamExecStore,
	)
	teamHandlers.SetHeartbeatScheduler(heartbeatScheduler)

	// Recover any active runs from a previous process
	runRegistry.Recover(context.Background(), agentManagerClient)
	teamExecStore.Recover(context.Background())

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

	// Heartbeat routes - static paths before parameterized
	v1.HandleFunc("/tasks", heartbeatHandlers.CreateTask).Methods("POST")
	v1.HandleFunc("/runs", heartbeatHandlers.CreateRun).Methods("POST")
	v1.HandleFunc("/runs", heartbeatHandlers.ListRuns).Methods("GET")
	v1.HandleFunc("/runs/investigate", heartbeatHandlers.CreateInvestigationRun).Methods("POST")
	v1.HandleFunc("/runs/investigation-apply", heartbeatHandlers.CreateInvestigationApplyRun).Methods("POST")
	v1.HandleFunc("/runs/{runId}", heartbeatHandlers.GetRun).Methods("GET")
	v1.HandleFunc("/runs/{runId}/retry", heartbeatHandlers.RetryRun).Methods("POST")
	v1.HandleFunc("/runs/{runId}/events", heartbeatHandlers.GetRunEvents).Methods("GET")
	v1.HandleFunc("/runs/{runId}/continue", heartbeatHandlers.ContinueRun).Methods("POST")
	v1.HandleFunc("/heartbeats/running", heartbeatHandlers.ListRunning).Methods("GET")
	v1.HandleFunc("/heartbeats/running/{teamId}/{agentId}/stop", heartbeatHandlers.StopRunning).Methods("POST")
	v1.HandleFunc("/prompt-preview", heartbeatHandlers.PreviewPrompt).Methods("POST")
	v1.HandleFunc("/prompt-preview-structured", heartbeatHandlers.PreviewPromptStructured).Methods("POST")
	v1.HandleFunc("/teams/{id}/prompt-matrix", heartbeatHandlers.PreviewPromptMatrix).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats", heartbeatHandlers.ListHeartbeats).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.GetHeartbeat).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.CreateHeartbeat).Methods("POST")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.UpdateHeartbeat).Methods("PUT")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}", heartbeatHandlers.DeleteHeartbeat).Methods("DELETE")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}/trigger", heartbeatHandlers.TriggerHeartbeat).Methods("POST")
	v1.HandleFunc("/teams/{id}/trigger", heartbeatHandlers.TriggerTeam).Methods("POST")
	v1.HandleFunc("/teams/{id}/execution-status", heartbeatHandlers.GetTeamExecutionStatus).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/logs", heartbeatHandlers.ListTeamLogs).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}/logs", heartbeatHandlers.ListLogs).Methods("GET")
	v1.HandleFunc("/teams/{id}/heartbeats/{agentId}/logs/{logId}", heartbeatHandlers.GetLog).Methods("GET")

	// Member document routes (RESPONSIBILITIES.md and HEARTBEAT.md)
	v1.HandleFunc("/teams/{id}/members/{agentId}/responsibilities", heartbeatHandlers.GetResponsibilities).Methods("GET")
	v1.HandleFunc("/teams/{id}/members/{agentId}/responsibilities", heartbeatHandlers.SetResponsibilities).Methods("PUT")
	v1.HandleFunc("/teams/{id}/members/{agentId}/heartbeat-instructions", heartbeatHandlers.GetHeartbeatInstructions).Methods("GET")
	v1.HandleFunc("/teams/{id}/members/{agentId}/heartbeat-instructions", heartbeatHandlers.SetHeartbeatInstructions).Methods("PUT")
	v1.HandleFunc("/teams/{id}/members/{agentId}/context", heartbeatHandlers.GetMemberContext).Methods("GET")

	// Team state routes (handoff, task board, decisions)
	v1.HandleFunc("/teams/{id}/members/{agentId}/handoff", heartbeatHandlers.GetLastHandoff).Methods("GET")
	v1.HandleFunc("/teams/{id}/handoff-history", heartbeatHandlers.GetHandoffHistory).Methods("GET")
	v1.HandleFunc("/teams/{id}/tasks", heartbeatHandlers.GetTaskBoard).Methods("GET")
	v1.HandleFunc("/teams/{id}/tasks", heartbeatHandlers.AddTask).Methods("POST")
	v1.HandleFunc("/teams/{id}/tasks/{taskId}", heartbeatHandlers.UpdateTaskHandler).Methods("PATCH", "PUT")
	v1.HandleFunc("/teams/{id}/tasks/{taskId}", heartbeatHandlers.DeleteTaskHandler).Methods("DELETE")
	v1.HandleFunc("/decisions/pending", heartbeatHandlers.GetAllPendingDecisions).Methods("GET")
	v1.HandleFunc("/teams/{id}/decisions", heartbeatHandlers.AddDecision).Methods("POST")
	v1.HandleFunc("/teams/{id}/decisions", heartbeatHandlers.GetDecisions).Methods("GET")
	v1.HandleFunc("/teams/{id}/decisions/{decisionId}", heartbeatHandlers.UpdateDecisionHandler).Methods("PATCH", "PUT")
	v1.HandleFunc("/teams/{id}/decisions/{decisionId}", heartbeatHandlers.DeleteDecisionHandler).Methods("DELETE")

	// Knowledge log routes
	v1.HandleFunc("/teams/{id}/knowledge", heartbeatHandlers.AddKnowledge).Methods("POST")
	v1.HandleFunc("/teams/{id}/knowledge", heartbeatHandlers.GetKnowledge).Methods("GET")
	v1.HandleFunc("/teams/{id}/knowledge/{knowledgeId}", heartbeatHandlers.UpdateKnowledgeHandler).Methods("PATCH", "PUT")
	v1.HandleFunc("/teams/{id}/knowledge/{knowledgeId}", heartbeatHandlers.DeleteKnowledgeHandler).Methods("DELETE")

	// Retention / prune routes
	v1.HandleFunc("/teams/{id}/retention", heartbeatHandlers.GetRetention).Methods("GET")
	v1.HandleFunc("/teams/{id}/prune", heartbeatHandlers.PruneSharedState).Methods("POST")

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

// buildTopicMatchFn creates a TopicMatchFunc that uses the AI search service
// to perform topic matching and skill accumulation.
func buildTopicMatchFn(aiSvc *aisearch.Service, topicStore store.TopicStore) topics.TopicMatchFunc {
	return func(ctx context.Context, queries []string, limit int) ([]topics.MatchedTopic, []string, string, error) {
		type topicEntry struct {
			topic topics.MatchedTopic
			score float64
		}
		seen := make(map[string]*topicEntry)
		allSkillIDs := make(map[string]bool)
		method := "ai"

		for _, query := range queries {
			topicResults, topicMethod, err := aiSvc.SearchTopics(ctx, query, limit)
			if err != nil {
				continue
			}
			if topicMethod != "ai" {
				method = topicMethod
			}

			for _, tr := range topicResults {
				topicID, _ := tr.Payload["topic_id"].(string)
				if topicID == "" {
					continue
				}
				name, _ := tr.Payload["name"].(string)
				description, _ := tr.Payload["description"].(string)
				parentID, _ := tr.Payload["parent_topic_id"].(string)

				scorePercent := int(tr.Score * 100)
				if scorePercent > 100 {
					scorePercent = 100
				}

				if existing, ok := seen[topicID]; !ok || tr.Score > existing.score {
					var parentPtr *string
					if parentID != "" {
						parentPtr = &parentID
					}
					seen[topicID] = &topicEntry{
						topic: topics.MatchedTopic{
							ID:            topicID,
							Name:          name,
							Description:   description,
							ParentTopicID: parentPtr,
							Score:         tr.Score,
							ScorePercent:  scorePercent,
						},
						score: tr.Score,
					}
				}

				skillIDs, err := topicStore.AccumulateSkills(ctx, topicID)
				if err == nil {
					for _, sid := range skillIDs {
						allSkillIDs[sid] = true
					}
				}
			}
		}

		matched := make([]topics.MatchedTopic, 0, len(seen))
		for _, e := range seen {
			matched = append(matched, e.topic)
		}
		// Sort by score descending
		for i := 1; i < len(matched); i++ {
			for j := i; j > 0 && matched[j].Score > matched[j-1].Score; j-- {
				matched[j], matched[j-1] = matched[j-1], matched[j]
			}
		}

		skillsList := make([]string, 0, len(allSkillIDs))
		for sid := range allSkillIDs {
			skillsList = append(skillsList, sid)
		}

		return matched, skillsList, method, nil
	}
}

// resolveQdrantURL returns the Qdrant base URL from environment variables.
// It checks QDRANT_URL, QDRANT_BASE_URL (Vrooli resource export), and
// falls back to constructing from QDRANT_PORT if available.
func resolveQdrantURL() string {
	if url := os.Getenv("QDRANT_URL"); url != "" {
		return url
	}
	if url := os.Getenv("QDRANT_BASE_URL"); url != "" {
		return url
	}
	if port := os.Getenv("QDRANT_PORT"); port != "" {
		return "http://localhost:" + port
	}
	return ""
}

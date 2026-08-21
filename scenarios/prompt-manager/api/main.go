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
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"prompt-manager/handlers/actions"
	"prompt-manager/handlers/agents"
	"prompt-manager/handlers/aisearch"
	discoveryhandlers "prompt-manager/handlers/discovery"
	"prompt-manager/handlers/experiments"
	"prompt-manager/handlers/graph"
	"prompt-manager/handlers/heartbeat"
	"prompt-manager/handlers/memberflow"
	"prompt-manager/handlers/metadata"
	"prompt-manager/handlers/ogmeta"
	"prompt-manager/handlers/search"
	"prompt-manager/handlers/skills"
	"prompt-manager/handlers/tags"
	"prompt-manager/handlers/teams"
	"prompt-manager/handlers/templates"
	"prompt-manager/handlers/testing"
	"prompt-manager/handlers/topics"
	"prompt-manager/handlers/worldscale"
	"prompt-manager/handlers/worldseats"
	promptmeasures "prompt-manager/internal/measures"
	"prompt-manager/internal/metrics"
	localmodules "prompt-manager/internal/modules"
	"prompt-manager/internal/paths"
	"prompt-manager/internal/sourceledger"
	"prompt-manager/internal/store"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/receiptsigning"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	measurelib "github.com/vrooli/measures-go"
	searchregister "github.com/vrooli/searchregister-go"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// gorillaMuxAdapter gives api-core's development-only Connect registration a
// minimal, router-agnostic mount surface without coupling that package to
// gorilla/mux's fluent Handle return value.
type gorillaMuxAdapter struct{ router *mux.Router }

func (m gorillaMuxAdapter) Handle(pattern string, handler http.Handler) {
	if strings.HasSuffix(pattern, "/") {
		m.router.PathPrefix(pattern).Handler(handler)
		return
	}
	m.router.Handle(pattern, handler)
}

func tableMeasure(rows []map[string]string, query string) measurelib.MeasureResult {
	return measurelib.MeasureResult{
		Fields:     rows,
		Provenance: measurelib.Provenance{ExecutedQuery: query},
	}
}

func scalarMeasure(value int, query string) measurelib.MeasureResult {
	return measurelib.MeasureResult{
		Value:      strconv.Itoa(value),
		Provenance: measurelib.Provenance{ExecutedQuery: query},
	}
}

// securityHeaders applies the API-wide baseline before any route handler runs.
// Keeping this at the router boundary prevents individual Connect and legacy
// compatibility handlers from drifting on response hardening.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

// parseMeasureWindow accepts the same Go-duration syntax as the existing
// discovery endpoints and defaults to the bounded 30-day operational window.
func parseMeasureWindow(raw string) time.Duration {
	if window, err := time.ParseDuration(strings.TrimSpace(raw)); err == nil && window > 0 {
		return window
	}
	return 30 * 24 * time.Hour
}

func matchesActionMeasure(pack, status, owner, ownerID string, tags []string, params map[string]string) bool {
	if value := params["pack"]; value != "" && value != pack {
		return false
	}
	if value := params["status"]; value != "" && value != status {
		return false
	}
	if value := params["owner"]; value != "" && value != owner && value != ownerID {
		return false
	}
	if value := params["tag"]; value != "" {
		for _, tag := range tags {
			if tag == value {
				return true
			}
		}
		return false
	}
	return true
}

func receiptSignerFromRuntimeConfig() (receiptsigning.ReceiptSigner, bool, error) {
	type manifestTrustSigning struct {
		Provider           string `json:"provider"`
		Identity           string `json:"identity"`
		Field              string `json:"field"`
		Address            string `json:"address"`
		KeyName            string `json:"key_name"`
		CredentialFile     string `json:"credential_file"`
		LegacyVaultTransit *struct {
			Address        string `json:"address"`
			KeyName        string `json:"key_name"`
			CredentialFile string `json:"credential_file"`
		} `json:"legacy_vault_transit"`
	}
	type manifest struct {
		TrustSigning *manifestTrustSigning `json:"trust_signing"`
	}
	scenarioDir := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR"))
	if scenarioDir == "" {
		scenarioDir = filepath.Clean("..")
	}
	manifestBytes, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err == nil {
		var service manifest
		if err := json.Unmarshal(manifestBytes, &service); err != nil {
			return nil, false, fmt.Errorf("parse trust signing lifecycle declaration: %w", err)
		}
		if service.TrustSigning != nil {
			if service.TrustSigning.Provider == "development" {
				return receiptsigning.NewDevelopmentSigner(), false, nil
			}
			if service.TrustSigning.Provider == receiptsigning.ModeCredentialAuthorityEd25519 {
				config := receiptsigning.RuntimeConfig{
					Version: receiptsigning.RuntimeConfigVersion,
					Mode:    receiptsigning.ModeCredentialAuthorityEd25519,
					CredentialAuthority: &receiptsigning.CredentialAuthorityRuntimeConfig{
						Identity: service.TrustSigning.Identity,
						Field:    service.TrustSigning.Field,
					},
				}
				if legacy := service.TrustSigning.LegacyVaultTransit; legacy != nil {
					config.LegacyVaultTransit = &receiptsigning.VaultTransitRuntimeConfig{Address: legacy.Address, KeyName: legacy.KeyName, CredentialFile: legacy.CredentialFile}
				}
				return credentialauthoritysigning.NewSignerFromRuntimeConfig(config)
			}
			if service.TrustSigning.Provider != "vault-transit" {
				return nil, false, fmt.Errorf("unsupported trust signing lifecycle provider %q", service.TrustSigning.Provider)
			}
			return receiptsigning.NewSignerFromRuntimeConfig(receiptsigning.RuntimeConfig{Version: receiptsigning.RuntimeConfigVersion, Mode: "vault-transit", VaultTransit: &receiptsigning.VaultTransitRuntimeConfig{Address: service.TrustSigning.Address, KeyName: service.TrustSigning.KeyName, CredentialFile: service.TrustSigning.CredentialFile}})
		}
	}
	// A missing declaration is retained only for old developer checkouts. New
	// lifecycle manifests must declare trust_signing explicitly.
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, false, fmt.Errorf("create receipt signing storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("prompt-manager")
	if err != nil {
		return nil, false, fmt.Errorf("resolve prompt-manager receipt signing namespace: %w", err)
	}
	path, err := resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassConfig, "receipt-signing.json")
	if err != nil {
		return nil, false, fmt.Errorf("resolve receipt signing runtime config: %w", err)
	}
	config, err := receiptsigning.LoadRuntimeConfig(path)
	if os.IsNotExist(err) {
		return receiptsigning.NewDevelopmentSigner(), false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("load receipt signing runtime config: %w", err)
	}
	var signer receiptsigning.ReceiptSigner
	var production bool
	if config.Mode == receiptsigning.ModeCredentialAuthorityEd25519 {
		signer, production, err = credentialauthoritysigning.NewSignerFromRuntimeConfig(config)
	} else {
		signer, production, err = receiptsigning.NewSignerFromRuntimeConfig(config)
	}
	if err != nil {
		return nil, false, fmt.Errorf("build receipt signer: %w", err)
	}
	return signer, production, nil
}

// discoverScenarioNames returns the names of every scenario directory
// under the repo's scenarios/ folder. The scenarios path comes from the
// repo contract (paths.Roots.ScenariosDir), not from climbing a storage
// path — the latter conflated source-code location with where mutable
// data lives, which is precisely the smell this scenario was built to fix.
func discoverScenarioNames(scenariosDir string) []string {
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

type heartbeatPromptSectionProvider struct {
	executor *heartbeat.Executor
}

func (p heartbeatPromptSectionProvider) SectionsForMember(ctx context.Context, team, member string) ([]memberflow.OperatingGraphPromptSection, error) {
	sections, err := p.executor.BuildPromptStructured(ctx, team, member)
	if err != nil {
		return nil, err
	}
	out := make([]memberflow.OperatingGraphPromptSection, 0, len(sections))
	for _, section := range sections {
		out = append(out, memberflow.OperatingGraphPromptSection{
			Team:       team,
			Member:     member,
			Kind:       section.Kind,
			SourcePath: section.SourcePath,
			Content:    section.Content,
			SourceKind: memberflow.OperatingGraphPromptSectionSourceLive,
		})
	}
	return out, nil
}

func main() {
	// Preflight checks
	if preflight.Run(preflight.Config{
		ScenarioName: "prompt-manager",
	}) {
		return
	}

	// Configuration from environment
	ollamaEnabled, ollamaConfigErr := resolveOllamaEnabled(os.Getenv)
	if ollamaConfigErr != nil {
		log.Printf("%v - skill testing will be disabled", ollamaConfigErr)
	}
	ollamaGatewayBin := strings.TrimSpace(os.Getenv("OLLAMA_GATEWAY_BIN"))
	if ollamaGatewayBin == "" {
		ollamaGatewayBin = "resource-ollama"
	}
	if !ollamaEnabled {
		log.Println("OLLAMA_ENABLED not set to true - skill testing will be disabled")
	}

	// Storage configuration.
	//
	// Config root: the authored, git-tracked store/ tree. Defaults to
	// ../store relative to the binary's working directory; overridable via
	// STORE_DIR for development workflows.
	//
	// Runtime data + cache roots: resolved by paths.Resolve via
	// api-core/storage (ProfileAuto). The three roots live on the Roots
	// struct and are threaded through the FileStore constructor and any
	// handler that needs a class-aware path.
	configDir := filepath.Join("..", "store")
	if envDir := os.Getenv("STORE_DIR"); envDir != "" {
		configDir = envDir
	}
	roots, err := paths.Resolve(configDir)
	if err != nil {
		log.Fatalf("Storage path resolution failed: %v", err)
	}

	// Open SQLite. Bound startup storage work so the lifecycle health gate gets
	// a clear failure instead of killing a silent retry loop.
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer dbCancel()
	db, err := database.Open(dbCtx, database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "prompt-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		Logger:       log.Printf,
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	if err := database.EnsureSchemas(dbCtx, db.Primary(), localmodules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	// A leased test pool starts empty. Initialize its schema before devrouting
	// makes it visible, so test-mode requests cannot race into an unprepared DB.
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, localmodules.AllSchemas()...)
	})
	log.Println("SQLite database connected and schemas initialized")

	// Initialize the new file-based store
	fileRoots := filerouting.New(storage.Paths{
		ConfigDir: roots.Config,
		DataDir:   roots.RuntimeData,
		CacheDir:  roots.RuntimeCache,
	})
	fileStore, err := store.NewFileStore(roots, fileRoots)
	if err != nil {
		log.Fatalf("initialize file store: %v", err)
	}
	ledger, err := sourceledger.New(dbCtx)
	if err != nil {
		log.Fatalf("connect to source-ledger: %v", err)
	}
	eventsBase, err := discovery.ResolveScenarioURLDefault(dbCtx, "vrooli-events")
	if err != nil {
		log.Fatalf("connect to vrooli-events: %v", err)
	}
	teamScopes := []string{"director-swarm", "infra-health", "marketing-crew", "meta-optimization", "monetization", "scenario-qa"}
	for _, teamID := range teamScopes {
		if err := ledger.EnsureTeamScope(dbCtx, teamID); err != nil {
			log.Fatalf("register source-ledger scope %q: %v", teamID, err)
		}
	}
	fileStore.SetSourceLedger(ledger)
	fileStore.SetEventsEndpoint(eventsBase)
	fileStore.SetExperimentStoreDatabase(db)

	// Initialize domain components (seams for testing)
	// Use the store adapter to bridge new storage to existing handlers
	skillStoreAdapter := skills.NewStoreAdapter(fileStore.FileSkills(), store.NewFileContentIO())
	metricsRepo := metrics.NewRepository(db)
	tagsRepo := tags.NewRepository(db)
	testingRepo := testing.NewRepository(db)
	ollamaClient := testing.NewOllamaClient(ollamaEnabled, ollamaGatewayBin)

	// Initialize handlers with interface adapters
	metricsAdapter := skills.NewMetricsAdapter(metricsRepo)
	skillHandlers := skills.NewHandlers(skillStoreAdapter, metricsAdapter, roots.Config)
	tagsHandlers := tags.NewHandlers(tagsRepo)
	testingHandlers := testing.NewHandlers(testingRepo, ollamaClient, skillStoreAdapter)
	templateHandlers := templates.NewHandlers(templates.NewStore(roots.Config))
	actionService := actions.NewService(fileStore.Actions(), actions.NewCLIHealthCommandResolver())
	actionHandlers := actions.NewHandlers(actionService)
	actionsConnectPath, actionsConnectHandler := actions.NewConnectMount(actionHandlers)
	tagsConnectPath, tagsConnectHandler := tags.NewConnectMount(tagsHandlers)

	// Agent handlers (new storage-backed, replaces member handlers)
	agentHandlers := agents.NewHandlers(fileStore.Agents(), fileStore.Indexes(), roots.Config, fileStore.Relations(), fileStore.Teams())

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

	aiSearchCollection := collectionForDomain("AI_SEARCH_COLLECTION", "skills")

	aiSearchThreshold := 0.5
	if thresholdStr := os.Getenv("AI_SEARCH_THRESHOLD"); thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			aiSearchThreshold = parsed
		}
	}

	// Initialize AI search components. Qdrant collection dimensions resolve from
	// the embedding role when each collection is ensured.
	embeddingRole := "embedding.default"
	embedder := aisearch.NewEmbedder(embeddingRole)
	vectorStore := aisearch.NewVectorStoreForRole(qdrantURL, qdrantAPIKey, aiSearchCollection, embeddingRole)
	aiSearchService := aisearch.NewService(embedder, vectorStore, skillStoreAdapter, searchService, aiSearchThreshold)
	aiSearchHandlers := aisearch.NewHandlers(aiSearchService)

	// Set AI indexer on skill handlers for CRUD hook integration
	skillHandlers.SetAIIndexer(aiSearchService)

	// Set experiment stores on skill handlers for variant-aware read
	skillHandlers.SetExperimentStores(fileStore.Experiments(), fileStore.Variants(), fileStore.Skills())
	skillHandlers.SetIdentityVerifier(skills.NewAgentManagerIdentityVerifier(nil))

	// Variant and experiment handlers
	variantHandlers := skills.NewVariantHandlers(fileStore.Variants(), fileStore.Skills())
	skillsConnectPath, skillsConnectHandler := skills.NewConnectMount(skillHandlers, variantHandlers)
	experimentHandlers := skills.NewExperimentHandlers(fileStore.Experiments(), fileStore.Variants(), fileStore.Skills())
	experimentHandlers.SetWorkPublisher(skills.NewHTTPWorkPublisherFromEnv())
	// Lifecycle/Secrets Manager writes a standard runtime config into the
	// scenario config directory. Absent config is explicitly development-only;
	// production config selects the credential-authority signer without an
	// environment key.
	receiptSigner, productionReceiptSigning, err := receiptSignerFromRuntimeConfig()
	if err != nil {
		panic(err)
	}
	experimentHandlers.SetReceiptSigner(receiptSigner)
	experimentHandlers.SetProductionReceiptSigningRequired(productionReceiptSigning)
	experimentsConnectPath, experimentsConnectHandler := experiments.NewConnectMount(experimentHandlers)

	// Agent and team AI search vector stores
	agentAICollection := collectionForDomain("AI_SEARCH_AGENT_COLLECTION", "agents")
	agentVectorStore := aisearch.NewVectorStoreForRole(qdrantURL, qdrantAPIKey, agentAICollection, embeddingRole)

	teamAICollection := collectionForDomain("AI_SEARCH_TEAM_COLLECTION", "teams")
	teamVectorStore := aisearch.NewVectorStoreForRole(qdrantURL, qdrantAPIKey, teamAICollection, embeddingRole)

	topicAICollection := collectionForDomain("AI_SEARCH_TOPIC_COLLECTION", "topics")
	topicVectorStore := aisearch.NewVectorStoreForRole(qdrantURL, qdrantAPIKey, topicAICollection, embeddingRole)

	actionAICollection := collectionForDomain("AI_SEARCH_ACTION_COLLECTION", "actions")
	actionVectorStore := aisearch.NewVectorStoreForRole(qdrantURL, qdrantAPIKey, actionAICollection, embeddingRole)

	// Wire agent/team/topic AI search into the service
	aiSearchService.SetAgentSearch(agentVectorStore, fileStore.Agents().(*store.FileAgentStore), agentSearchService)
	aiSearchService.SetTeamSearch(teamVectorStore, fileStore.Teams().(*store.FileTeamStore), fileStore.Relations(), teamSearchService)
	aiSearchService.SetTopicSearch(topicVectorStore, fileStore.FileTopics())
	aiSearchService.SetActionSearch(actionVectorStore, fileStore.Actions())

	// Wire the semantic-similarity seam used by `action create` previews to
	// surface near-duplicate actions (structural + semantic dedup guard).
	actionService.SetSemanticSearcher(actionSemanticAdapter{svc: aiSearchService})

	// Budget config store
	budgetConfigStore := aisearch.NewBudgetConfigStore(roots.Config)
	aiSearchService.SetBudgetConfig(budgetConfigStore)
	aiSearchHandlers.SetBudgetConfigStore(budgetConfigStore)

	// Discover filter config store
	discoverFilterConfigStore := aisearch.NewDiscoverFilterConfigStore(roots.Config)
	aiSearchService.SetDiscoverFilterConfig(discoverFilterConfigStore)
	aiSearchHandlers.SetDiscoverFilterConfigStore(discoverFilterConfigStore)

	// Discover ranking config store (topic gate, high-confidence bar, caps).
	// The topic gate must exceed the skill similarity threshold, so the store is
	// constructed with it for validation. Hot-loadable like budgets.json.
	discoverRankingConfigStore := aisearch.NewDiscoverRankingConfigStore(roots.Config, aiSearchThreshold)
	aiSearchService.SetDiscoverRankingConfig(discoverRankingConfigStore)

	// Discovery-miss telemetry store. Lives under the runtime-data root (not
	// the git-tracked store tree) so misses are durable runtime signal, not
	// source. Resolved via the storage path layer — no hard-coded ~/.vrooli.
	discoveryMissStore := store.NewDiscoveryMissStore(roots.RuntimeData)
	aiSearchService.SetDiscoveryMissStore(discoveryMissStore)

	// Per-call discovery telemetry store (sibling of the miss store) records
	// every discover call so threshold/budget/clipping behavior is measurable.
	discoveryCallStore := store.NewDiscoveryCallStore(roots.RuntimeData)
	aiSearchService.SetDiscoveryCallStore(discoveryCallStore)

	// Skill-read telemetry. Recording happens at the read handler, which is the
	// only point every consumer passes through; the reporter joins those reads
	// against the discovery calls above so a skill returned often and read
	// rarely reads as a search-precision defect rather than as demand.
	skillReadStore := store.NewSkillReadStore(roots.RuntimeData)
	skillHandlers.SetReadRecorder(skills.NewReadRecorder(skillReadStore, discoveryCallStore, metricsAdapter))
	usageReporter := skills.NewUsageReporter(skillReadStore, discoveryCallStore)
	skillHandlers.SetUsageReporter(usageReporter)
	// Opt-in threshold-clipping probe (default off). DISCOVERY_PROBE_SAMPLE=N
	// samples 1-in-N calls to re-search at threshold 0 and count clipped hits.
	if sampleStr := os.Getenv("DISCOVERY_PROBE_SAMPLE"); sampleStr != "" {
		if parsed, err := strconv.Atoi(sampleStr); err == nil {
			aiSearchService.SetDiscoveryProbeSample(parsed)
		}
	}

	// Set AI indexer on agent and team handlers for CRUD hook integration
	agentHandlers.SetAIIndexer(aiSearchService)
	actionHandlers.SetAIIndexer(aiSearchService)

	// Graph detection
	scenarioNames := discoverScenarioNames(roots.ScenariosDir)
	cliDetector := graph.NewCLIDetector(scenarioNames)
	graphScanner := graph.NewScanner(
		fileStore.Agents().(*store.FileAgentStore),
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.FileSkills(),
		fileStore.Relations(),
		cliDetector,
		fileStore.Actions(),
	)
	graphScanner.SetRepositoryRoot(roots.RepoRoot)
	generatedPromptBuilder := heartbeat.NewPromptBuilder(
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.Agents().(*store.FileAgentStore),
	)
	graphScanner.SetGeneratedPromptProvider(func(ctx context.Context, teamID, agentID string) (string, error) {
		return generatedPromptBuilder.BuildContext(ctx, heartbeat.PromptBuildRequest{TeamID: teamID, AgentID: agentID})
	})
	graphBuilder := graph.NewBuilder(
		fileStore.Agents().(*store.FileAgentStore),
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.FileSkills(),
		graphScanner,
		graph.DefaultScoreFns(),
		fileStore.Actions(),
	)
	graphHealthConfigStore := graph.NewHealthConfigStore(roots.Config)
	graphBuilder.SetHealthConfigProvider(graphHealthConfigStore)
	graphBuilder.SetScenarioHealthProvider(graph.NewScenarioCompletenessCLIProvider(15 * time.Second))
	graphBuilder.SetCommandReferenceValidator(graph.NewCLIHealthCommandValidator())
	graphIndex := graph.NewIndexStore(roots.RuntimeCache, graphBuilder)
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
	actionHandlers.SetGraphInvalidator(graphIndex)
	agentHandlers.SetGraphInvalidator(graphIndex)

	// AI Search: build the Reconciler + SyncLoop and wire them through the
	// handlers. The reconciler replaces the old count-based reindex loop with
	// a hash-driven Plan/Apply pipeline (see scenarios/prompt-manager/docs/...).
	if qdrantURL != "" {
		log.Printf("AI Search: Ollama gateway=%s, Qdrant=%s, Collection=%s", ollamaGatewayBin, qdrantURL, aiSearchCollection)
		go func() {
			ctx := context.Background()
			if !aiSearchService.Available(ctx) {
				log.Println("AI Search: Resources not reachable at startup, skipping initial reconcile")
				return
			}

			// Ensure every collection exists before the reconciler scrolls them.
			for _, vs := range []aisearch.VectorStore{vectorStore, agentVectorStore, teamVectorStore, topicVectorStore, actionVectorStore} {
				if err := vs.EnsureCollection(ctx); err != nil {
					log.Printf("AI Search: Failed to ensure collection: %v", err)
					return
				}
			}

			descriptors := []aisearch.CollectionDescriptor{
				aisearch.NewSkillDescriptor(vectorStore, skillStoreAdapter),
				aisearch.NewAgentDescriptor(agentVectorStore, fileStore.Agents().(*store.FileAgentStore)),
				aisearch.NewTeamDescriptor(teamVectorStore, fileStore.Teams().(*store.FileTeamStore), fileStore.Relations()),
				aisearch.NewTopicDescriptor(topicVectorStore, fileStore.FileTopics()),
				aisearch.NewActionDescriptor(actionVectorStore, fileStore.Actions()),
			}
			cfg := aisearch.LoadConfigFromEnv()
			reconciler := aisearch.NewReconciler(embedder, descriptors, cfg.ReconcileParallelism)
			aiSearchHandlers.SetReconciler(reconciler)

			bootCtx, bootCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if _, _, err := reconciler.RunOnce(bootCtx); err != nil && err != aisearch.ErrReconcileBusy {
				log.Printf("AI Search: boot-time reconcile failed: %v", err)
			}
			bootCancel()

			syncLoop := aisearch.NewSyncLoop(reconciler)
			go syncLoop.Start(ctx)
		}()
	} else {
		log.Println("AI Search: Resources not fully configured (will gracefully degrade to text search)")
	}

	// Search Hub registration is a best-effort mirror of this scenario's
	// .vrooli/search.json SSOT. It registers both high-traffic leaves (skills and
	// actions) and their starter corpora through the shared bridge, so registry
	// and eval governance see exactly the same boot-owned source of truth.
	go searchregister.Register(context.Background(), searchregister.Config{
		ScenarioID:     "prompt-manager",
		SearchFilePath: filepath.Join(roots.RepoRoot, "scenarios", "prompt-manager", ".vrooli", "search.json"),
		Logger:         log.Default(),
	})

	// Setup routes
	searchConnectPath, searchConnectHandler := search.NewConnectMount(searchHandlers)
	aiSearchConnectPath, aiSearchConnectHandler := aisearch.NewConnectMount(aiSearchHandlers)
	discoveryConnectPath, discoveryConnectHandler := discoveryhandlers.NewConnectMount(aiSearchHandlers, skillHandlers)
	agentsConnectPath, agentsConnectHandler := agents.NewConnectMount(agentHandlers)
	templatesConnectPath, templatesConnectHandler := templates.NewConnectMount(templateHandlers)
	testingConnectPath, testingConnectHandler := testing.NewConnectMount(testingHandlers)
	metadataConnectPath, metadataConnectHandler := metadata.NewConnectMount(ogmetaHandlers)
	worldScaleConnectPath, worldScaleConnectHandler := worldscale.NewConnectMount(roots.Config)
	worldSeatsConnectPath, worldSeatsConnectHandler := worldseats.NewConnectMount(roots.Config)
	router := mux.NewRouter()
	if !devrouting.RegisterWithFileRoots(gorillaMuxAdapter{router: router}, db, fileRoots) {
		log.Println("test-mode RoutingService disabled: scenario is not in development mode")
	}

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-Vrooli-Attribution", "X-Caller-ID", "X-Agent-Identity-Token", "X-Experiment-Read-Receipt-ID"}),
	)

	// Health check
	healthHandler := health.New().Version("2.0.0").Check(health.DB(db.Primary()), health.Critical).Handler()
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/health", healthHandler).Methods("GET")
	connectx.RegisterServices(
		router,
		connectx.ServiceMount{Path: skillsConnectPath, Handler: skillsConnectHandler},
		connectx.ServiceMount{Path: actionsConnectPath, Handler: actionsConnectHandler},
		connectx.ServiceMount{Path: tagsConnectPath, Handler: tagsConnectHandler},
		connectx.ServiceMount{Path: searchConnectPath, Handler: searchConnectHandler},
		connectx.ServiceMount{Path: aiSearchConnectPath, Handler: aiSearchConnectHandler},
		connectx.ServiceMount{Path: discoveryConnectPath, Handler: discoveryConnectHandler},
		connectx.ServiceMount{Path: agentsConnectPath, Handler: agentsConnectHandler},
		connectx.ServiceMount{Path: templatesConnectPath, Handler: templatesConnectHandler},
		connectx.ServiceMount{Path: testingConnectPath, Handler: testingConnectHandler},
		connectx.ServiceMount{Path: metadataConnectPath, Handler: metadataConnectHandler},
		connectx.ServiceMount{Path: worldScaleConnectPath, Handler: worldScaleConnectHandler},
		connectx.ServiceMount{Path: worldSeatsConnectPath, Handler: worldSeatsConnectHandler},
		connectx.ServiceMount{Path: experimentsConnectPath, Handler: experimentsConnectHandler},
	)

	// Graph Connect-RPC surface. This owns the complete graph contract, including
	// the health projection consumed by meta-optimization-manager's Guide metric.
	// See docs/internal/SEAMS.md#graph-connect-handler.
	graphConnectPath, graphConnectHandler := graph.NewConnectMount(graphIndex, graphHealthConfigStore)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: graphConnectPath, Handler: graphConnectHandler})

	// Budget config routes
	v1.HandleFunc("/config/budgets", aiSearchHandlers.GetBudgetConfig).Methods("GET")
	v1.HandleFunc("/config/budgets", aiSearchHandlers.PutBudgetConfig).Methods("PUT")

	// Discover filter config routes
	v1.HandleFunc("/config/discover-filters", aiSearchHandlers.GetDiscoverFilterConfig).Methods("GET")
	v1.HandleFunc("/config/discover-filters", aiSearchHandlers.PutDiscoverFilterConfig).Methods("PUT")

	// Team services
	teamHandlers := teams.NewHandlers(fileStore.Teams(), fileStore.Agents(), fileStore.Relations(), fileStore.Indexes(), nil)
	teamHandlers.SetGraphInvalidator(graphIndex)
	teamHandlers.SetAIIndexer(aiSearchService)
	teamsConnectPath, teamsConnectHandler := teams.NewConnectMount(teamHandlers)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: teamsConnectPath, Handler: teamsConnectHandler})
	// Member-flow (per-member topics.json) routes — declares each member's
	// intake/output topic prefixes and feeds the team graph view.
	// DOC: docs/agent-system/TOPICS_SCHEMA.md
	memberFlowHandlers := memberflow.NewHandlers(roots.Config, roots.RuntimeData)
	memberFlowHandlers.SetKnowledgeQuery(
		newTeamKnowledgeQuery(fileStore.Teams().(*store.FileTeamStore)),
		memberflow.InboxAgingOptions{},
	)
	memberflowConnectPath, memberflowConnectHandler := memberflow.NewConnectMount(memberFlowHandlers, graphHandlers)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: memberflowConnectPath, Handler: memberflowConnectHandler})

	// Topic routes
	topicHandlers := topics.NewHandlers(fileStore.Topics(), fileStore.Indexes())
	topicHandlers.SetGraphInvalidator(graphIndex)
	topicHandlers.SetAIIndexer(aiSearchService)
	topicHandlers.SetTopicMatchFn(buildTopicMatchFn(aiSearchService, fileStore.Topics()))
	topicsConnectPath, topicsConnectHandler := topics.NewConnectMount(topicHandlers)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: topicsConnectPath, Handler: topicsConnectHandler})

	// The uniform measures substrate executes the same domain reads as the
	// generated RPCs. It is mounted outside /api/v1 because measures-go defines
	// the fleet-wide /measures/declarations and /measures/execute contract.
	measureHandler, err := promptmeasures.Handler(map[string]promptmeasures.Provider{
		"actions.list": func(ctx context.Context, req measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			items, err := fileStore.Actions().List(ctx)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(items))
			for _, item := range items {
				if !matchesActionMeasure(item.Pack, item.Status, item.Owner.Type+":"+item.Owner.ID, item.Owner.ID, item.Tags, req.Params) {
					continue
				}
				rows = append(rows, map[string]string{"id": item.ID, "name": item.Name, "status": item.Status})
			}
			return tableMeasure(rows, "action store list with manifest filters"), nil
		},
		"agents.list": func(ctx context.Context, _ measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			items, err := fileStore.Agents().List(ctx)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, map[string]string{"id": item.ID, "name": item.DisplayName, "status": item.Status})
			}
			return tableMeasure(rows, "agent store list"), nil
		},
		"aisearch.discovery-metrics": func(_ context.Context, req measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			window := parseMeasureWindow(req.Params["since"])
			report, err := aiSearchService.DiscoveryMetrics(window, req.Params["type"])
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			return scalarMeasure(report.CallCount, fmt.Sprintf("discovery call telemetry over %s", window)), nil
		},
		"graph.health": func(ctx context.Context, _ measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			index, err := graphIndex.Get(ctx)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(index.Graph.HealthScores))
			for _, score := range index.Graph.HealthScores {
				rows = append(rows, map[string]string{"node_id": score.NodeID, "score": strconv.FormatFloat(score.Score, 'f', -1, 64)})
			}
			return tableMeasure(rows, "materialized graph health scores"), nil
		},
		"metrics.skill-usage": func(_ context.Context, req measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			window := parseMeasureWindow(req.Params["since"])
			report, err := usageReporter.Report(window)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(report.Rows))
			for _, item := range report.Rows {
				rows = append(rows, map[string]string{"skill_id": item.SkillID, "reads": strconv.Itoa(item.Reads), "returned": strconv.Itoa(item.Returned)})
			}
			return tableMeasure(rows, fmt.Sprintf("skill read and discovery telemetry over %s", window)), nil
		},
		"skills.list": func(ctx context.Context, _ measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			items, err := fileStore.FileSkills().List(ctx)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, map[string]string{"id": item.ID, "name": item.Name, "status": item.Status})
			}
			return tableMeasure(rows, "skill store list"), nil
		},
		"tags.list": func(ctx context.Context, _ measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			items, err := tagsRepo.WithContext(ctx).GetAll()
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, map[string]string{"id": item.ID, "name": item.Name})
			}
			return tableMeasure(rows, "tag repository list"), nil
		},
		"teams.list": func(ctx context.Context, _ measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			items, err := fileStore.Teams().List(ctx)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, map[string]string{"id": item.ID, "name": item.DisplayName, "enabled": strconv.FormatBool(item.Enabled)})
			}
			return tableMeasure(rows, "team store list"), nil
		},
		"topics.list": func(ctx context.Context, _ measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			items, err := fileStore.Topics().List(ctx)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			rows := make([]map[string]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, map[string]string{"id": item.ID, "name": item.Name, "status": item.Status})
			}
			return tableMeasure(rows, "topic store list"), nil
		},
	})
	if err != nil {
		log.Fatalf("initialize measure registry: %v", err)
	}
	router.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", measureHandler))
	// Heartbeat system: repo root flows from the repo contract via
	// paths.Roots (resolved at startup). Empty when paths.Resolve already
	// failed, but main.go would have log.Fatal'd long before here.
	vrooliRoot := roots.RepoRoot

	// Initialize heartbeat components
	agentManagerClient := heartbeat.NewAgentManagerClient(30 * time.Second)
	runRegistry := heartbeat.NewRunRegistry(roots.RuntimeData)
	heartbeatExecutor := heartbeat.NewExecutor(
		fileStore.Teams().(*store.FileTeamStore),
		fileStore.Agents().(*store.FileAgentStore),
		agentManagerClient,
		vrooliRoot,
		runRegistry,
		nil, // uses default SentinelExtractor
	)
	// Routes each member's own open declaration findings into its heartbeat
	// prompt. Wired on the executor because that builder serves both the real
	// spawn path and `team prompt-preview`, so the operator previews exactly
	// what the agent receives.
	heartbeatExecutor.SetContractFindingsProvider(&heartbeat.MemberflowContractFindings{
		StoreDir:       roots.Config,
		RepoRoot:       roots.RepoRoot,
		RuntimeDataDir: roots.RuntimeData,
	})
	memberFlowHandlers.SetPromptSectionProvider(heartbeatPromptSectionProvider{executor: heartbeatExecutor})
	operatingMapStore, err := graph.NewOperatingMapStore(memberflow.OperatingModelService{
		RepoRoot:       roots.RepoRoot,
		StoreDir:       roots.Config,
		PromptSections: heartbeatPromptSectionProvider{executor: heartbeatExecutor},
	}, roots.RepoRoot)
	if err != nil {
		log.Fatalf("operating map: %v", err)
	}
	graphIndex.SetDependentInvalidators(operatingMapStore)
	graphHandlers.SetOperatingMapStore(operatingMapStore)
	teamExecStore := heartbeat.NewTeamExecutionStore(
		fileStore.Teams().(*store.FileTeamStore),
		heartbeatExecutor,
		roots.RuntimeData,
		agentManagerClient,
	)
	heartbeatControlStore := heartbeat.NewHeartbeatControlStore(roots.RuntimeData)
	heartbeatExecutor.OnComplete = teamExecStore.OnComplete
	heartbeatExecutor.SetTeamExecStore(teamExecStore)
	heartbeatScheduler := heartbeat.NewScheduler(
		heartbeatExecutor,
		agentManagerClient,
		fileStore.Teams().(*store.FileTeamStore),
		teamExecStore,
	)
	heartbeatScheduler.SetControlStore(heartbeatControlStore)
	heartbeatHandlers := heartbeat.NewHandlers(heartbeat.HandlersDeps{
		TeamStore:     fileStore.Teams().(*store.FileTeamStore),
		AgentStore:    fileStore.Agents().(*store.FileAgentStore),
		RelationStore: fileStore.Relations(),
		Scheduler:     heartbeatScheduler,
		Executor:      heartbeatExecutor,
		RunRegistry:   runRegistry,
		AgentClient:   agentManagerClient,
		TeamExecStore: teamExecStore,
	})
	heartbeatHandlers.SetControlStore(heartbeatControlStore)
	heartbeatConnectPath, heartbeatConnectHandler := heartbeat.NewConnectMount(heartbeatHandlers)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: heartbeatConnectPath, Handler: heartbeatConnectHandler})
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

	log.Printf("Prompt Manager API v2.0 starting")
	log.Printf("Config root: %s", roots.Config)
	log.Printf("Runtime data root: %s", roots.RuntimeData)
	log.Printf("Runtime cache root: %s", roots.RuntimeCache)
	if ollamaEnabled {
		log.Printf("Ollama gateway: %s", ollamaGatewayBin)
	}

	handler := securityHeaders(apihttp.TestModeMiddleware(corsHandler(router)))
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

// resolveOllamaEnabled honors an explicit feature override and otherwise uses
// the resource environment injected by the control plane. This keeps optional
// Ollama dependencies disabled when absent without requiring scenarios to
// duplicate resource state in a second environment flag.
func resolveOllamaEnabled(getenv func(string) string) (bool, error) {
	if raw := strings.TrimSpace(getenv("OLLAMA_ENABLED")); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return false, fmt.Errorf("invalid OLLAMA_ENABLED=%q", raw)
		}
		return enabled, nil
	}
	for _, key := range []string{"OLLAMA_BASE_URL", "OLLAMA_URL", "OLLAMA_PORT"} {
		if strings.TrimSpace(getenv(key)) != "" {
			return true, nil
		}
	}
	return false, nil
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

// collectionForDomain preserves explicit operator overrides while deriving the
// default from the active storage namespace. Shadow/test-mode processes must
// never fall back to a live prompt-manager collection name.
func collectionForDomain(envName, domain string) string {
	if collection := strings.TrimSpace(os.Getenv(envName)); collection != "" {
		return collection
	}
	collection, err := storage.Collection(domain)
	if err != nil {
		log.Fatalf("resolve %s collection: %v", domain, err)
	}
	return collection
}

// actionSemanticAdapter adapts the aisearch service to the actions package's
// SemanticActionSearcher seam, so create previews can surface semantically
// similar existing actions without the actions package depending on aisearch.
type actionSemanticAdapter struct {
	svc *aisearch.Service
}

func (a actionSemanticAdapter) SearchSimilarActions(ctx context.Context, query string, limit int) ([]actions.SemanticActionHit, error) {
	resp, err := a.svc.SearchActions(ctx, query, limit)
	if err != nil || resp == nil {
		return nil, err
	}
	hits := make([]actions.SemanticActionHit, 0, len(resp.Results))
	for _, result := range resp.Results {
		hits = append(hits, actions.SemanticActionHit{ID: result.ID, Name: result.Name, Score: result.Score})
	}
	return hits, nil
}

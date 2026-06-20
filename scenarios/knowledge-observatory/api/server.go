package main

// DOC: docs/concepts/ARCHITECTURE.md#api-runtime
// DOC: docs/reference/configuration.md#api-runtime-configuration
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	pkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/maturity-go/assessment"
	searchregister "github.com/vrooli/searchregister-go"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

	knowledgeobservatoryv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1/knowledgeobservatoryv1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	dochealthhandler "knowledge-observatory/handlers/dochealth"
	"knowledge-observatory/internal/adapters/agentmanager"
	"knowledge-observatory/internal/adapters/deepsearchstore"
	"knowledge-observatory/internal/adapters/docaccessstore"
	"knowledge-observatory/internal/adapters/dochealingstore"
	"knowledge-observatory/internal/adapters/embedder"
	"knowledge-observatory/internal/adapters/metadatastore"
	"knowledge-observatory/internal/adapters/promptmanager"
	"knowledge-observatory/internal/adapters/vectorstore"
	"knowledge-observatory/internal/aisearch"
	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/deepsearch"
	"knowledge-observatory/internal/services/dochealing"
	"knowledge-observatory/internal/services/dochealth"
	"knowledge-observatory/internal/services/docsearch"
	"knowledge-observatory/internal/services/explorer"
	"knowledge-observatory/internal/services/graph"
	"knowledge-observatory/internal/services/viewer"
)

// Config holds minimal runtime configuration
type Config struct {
	Port                   string
	DatabaseURL            string
	QdrantURL              string
	QdrantAPIKey           string
	OllamaEmbeddingRole    string
	OllamaStructuredRole   string
	ResourceQdrantCLI      string
	ResourceCommandTimeout time.Duration
	ScenariosRoot          string
	AgentManagerTimeout    time.Duration
	PromptManagerTimeout   time.Duration
}

// Server wires the HTTP router and database connection
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router

	vectorStore ports.VectorStore
	embedder    ports.Embedder

	graphService         *graph.Service
	docHealthService     *dochealth.Service
	docHealthMaturity    *assessment.Spec
	docHealthEnvironment *commonv1.CaptureEnvironment
	docSearchService     *docsearch.Service
	docExplorerService   *explorer.Service
	docViewerService     *viewer.Service
	docDeepSearchService deepsearch.API
	docHealingService    dochealing.API

	docAccessLogger ports.DocAccessLogger

	materializer *Materializer

	// Documentation hybrid search (Phase 6 cutover): the indexer owns the
	// vrooli-docs collection + reconciler, docSearch is the read path (hybrid
	// dense+sparse RRF + rerank), and docSyncLoop drives periodic reconcile.
	docIndexer  *aisearch.Indexer
	docSearch   docSearchEngine
	docSyncLoop *pkg.SyncLoop

	// searchToken caches the control token search-hub mints for the
	// knowledge-observatory.docs provider at self-registration; the reindex
	// handler validates token-gated requests against it. See searchTokenHolder.
	searchToken *searchTokenHolder
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port:                   requireEnv("API_PORT"),
		QdrantURL:              strings.TrimSpace(os.Getenv("QDRANT_URL")),
		QdrantAPIKey:           strings.TrimSpace(os.Getenv("QDRANT_API_KEY")),
		OllamaEmbeddingRole:    strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_ROLE")),
		OllamaStructuredRole:   strings.TrimSpace(os.Getenv("OLLAMA_STRUCTURED_OUTPUT_ROLE")),
		ResourceQdrantCLI:      strings.TrimSpace(os.Getenv("RESOURCE_QDRANT_CLI")),
		ResourceCommandTimeout: 5 * time.Second,
		ScenariosRoot:          resolveScenariosRoot(),
		AgentManagerTimeout:    30 * time.Second,
		PromptManagerTimeout:   15 * time.Second,
	}

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	var db *sql.DB
	if !shouldSkipDBConnect() {
		var err error
		db, err = database.Connect(context.Background(), database.Config{
			Driver: "postgres",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	srv := &Server{
		config:      cfg,
		db:          db,
		router:      mux.NewRouter(),
		searchToken: &searchTokenHolder{},
	}

	srv.setupServices()
	srv.setupRoutes()
	return srv, nil
}

func shouldSkipDBConnect() bool {
	return isTruthy(os.Getenv("SKIP_DB_TESTS")) || isTruthy(os.Getenv("SKIP_DB_CONNECT"))
}

func isTruthy(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func (s *Server) setupServices() {
	vs := &vectorstore.Qdrant{
		BaseURL: s.qdrantURL(),
		APIKey:  s.qdrantAPIKey(),
	}
	emb := &embedder.Ollama{
		Role: s.ollamaEmbeddingRole(),
	}

	var meta *metadatastore.Postgres
	if s.db != nil {
		meta = &metadatastore.Postgres{DB: s.db}
	}

	s.vectorStore = vs
	s.embedder = emb

	s.graphService = &graph.Service{
		VectorStore: vs,
		Embedder:    emb,
	}

	if s.config != nil && s.config.ScenariosRoot != "" {
		service, err := dochealth.NewService(s.config.ScenariosRoot)
		if err != nil {
			s.log("doc health service disabled", map[string]interface{}{"error": err.Error()})
		} else {
			s.docHealthService = service
		}
		repoRoot := filepath.Dir(s.config.ScenariosRoot)
		spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "knowledge-observatory"))
		if err != nil {
			s.log("doc health maturity assessment unavailable", map[string]interface{}{"error": err.Error()})
		} else {
			s.docHealthMaturity = spec
		}
		// Capture host facts once; they do not change during the process lifetime.
		// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
		// os/arch/num_cpu from the stdlib, leaving richer facts unset.
		environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
		if envErr != nil {
			s.log("doc health: host inventory unavailable, metrics environment limited to stdlib baseline", map[string]interface{}{"error": envErr.Error()})
			environment = nil
		}
		s.docHealthEnvironment = environment
	}
	if s.config != nil && s.config.ScenariosRoot != "" {
		service, err := explorer.NewService(s.config.ScenariosRoot, s.docHealthService)
		if err != nil {
			s.log("doc explorer service disabled", map[string]interface{}{"error": err.Error()})
		} else {
			s.docExplorerService = service
		}
	}
	if s.config != nil && s.config.ScenariosRoot != "" {
		service, err := docsearch.NewService(s.config.ScenariosRoot)
		if err != nil {
			s.log("doc search service disabled", map[string]interface{}{"error": err.Error()})
		} else {
			s.docSearchService = service
		}
	}

	// Documentation hybrid search (Phase 6 cutover). The shared engine
	// (packages/ai-go/search) supplies the embedder, vector store, reconciler,
	// env config, and sync loop; the KO-local internal/aisearch package supplies
	// the manifest-driven doc source, markdown chunker, hybrid read path, and
	// reranker chain. The grep fallback reuses docSearchService; the unified
	// endpoint's semantic leg is re-pointed here (replacing the deleted
	// internal/services/search read path).
	if s.config != nil && s.config.ScenariosRoot != "" {
		docCfg := pkg.LoadConfig("KO_DOCS")
		maxEmbeds := docCfg.MaxEmbedsPerTick
		if maxEmbeds <= 0 {
			// The documentation corpus is large (~6k chunks); cap embeds per
			// reconcile tick so the first full index never starves Ollama
			// (plan §4.2). A one-shot `reindex run` still applies uncapped.
			maxEmbeds = defaultDocEmbedsPerTick
		}
		// The scenario-owned `.vrooli/search.json` is the SSOT for the docs search
		// tuning (engine shape, embed recipe, rerank policy). LoadConfig above
		// supplies only operational wiring (sync cadence, parallelism). The
		// KO_DOCS_{RERANK_*,EMBED_TASK_PREFIX} tuning env vars are no longer
		// consulted. KO's docs corpus is hybrid by construction; the SSOT records
		// that plus the symmetric-embedder / rerank-off baseline that preserves the
		// guarded recall@5=0.818.
		docsTuning := s.loadDocsTuning()
		policyCtx, cancelPolicy := context.WithTimeout(context.Background(), s.config.ResourceCommandTimeout)
		docSearchCfg, err := pkg.ResolveEmbeddingConfig(policyCtx, pkg.Config{
			EmbedRole:       docCfg.EmbedRole,
			EmbedTaskPrefix: docsTuning.EmbedTaskPrefix,
		})
		cancelPolicy()
		if err != nil {
			s.log("documentation search disabled; embedding policy resolution failed", map[string]interface{}{"error": err.Error()})
		}
		if err == nil {
			embeddingPolicy := pkg.EmbeddingPolicy{
				Role:       docSearchCfg.EmbedRole,
				Model:      docSearchCfg.EmbedModel,
				Dimensions: docSearchCfg.EmbedDimensions,
			}
			if embeddingPolicy.Model == "" || embeddingPolicy.Dimensions <= 0 {
				s.log("documentation search disabled; embedding policy was invalid", map[string]interface{}{"model": embeddingPolicy.Model, "dimensions": embeddingPolicy.Dimensions})
			} else {
				docEmbedder := pkg.NewEmbedderForConfig(docSearchCfg)
				docStore := pkg.NewVectorStore(s.qdrantURL(), s.qdrantAPIKey(), aisearch.DefaultCollection)
				indexer, err := aisearch.NewIndexer(aisearch.Options{
					Embedder:         docEmbedder,
					VectorStore:      docStore,
					ScenariosRoot:    s.config.ScenariosRoot,
					Embedding:        embeddingPolicy,
					Parallelism:      docCfg.ReconcileParallelism,
					MaxEmbedsPerTick: maxEmbeds,
				})
				if err != nil {
					s.log("documentation indexer disabled", map[string]interface{}{"error": err.Error()})
				} else {
					s.docIndexer = indexer
					// EnsureCollection is best-effort: if qdrant is down at boot the
					// scenario still serves grep-fallback search and a degraded status.
					if cerr := indexer.EnsureCollection(context.Background()); cerr != nil {
						s.log("vrooli-docs collection ensure failed (degraded search)", map[string]interface{}{"error": cerr.Error()})
					}
					var fallback aisearch.TextFallback
					if s.docSearchService != nil {
						fallback = aisearch.NewDocsearchFallback(s.docSearchService)
					}
					// Rerank on docs is a tuning decision, now owned by search.json (plan §5).
					// Default OFF: the validated finding is that hybrid RRF + authority boost
					// ties the cross-encoder and beats the LLM reranker on recall for this
					// corpus — reranking buys ordering parity, not recall. Set
					// tuning.rerank_enabled=true in search.json (and re-run the search-hub
					// eval A/B) to enable. The flag is passed explicitly (shared convention);
					// the chain is only built when enabled.
					var docReranker *pkg.RerankerChain
					if docsTuning.RerankEnabled {
						docReranker = aisearch.NewDefaultReranker()
					}
					searchSvc := aisearch.NewSearchService(aisearch.ServiceOptions{
						Embedder:        docEmbedder,
						VectorStore:     docStore,
						RerankEnabled:   docsTuning.RerankEnabled,
						RerankBlend:     docsTuning.RerankBlend,
						RerankShortlist: docsTuning.RerankShortlist,
						Floor:           docsTuning.Floor.Config(),
						Reranker:        docReranker,
						TextFallback:    fallback,
						Reconciler:      indexer.Reconciler(),
					})
					s.docSearch = searchSvc
					if s.docSearchService != nil {
						s.docSearchService.Semantic = docSemanticAdapter{engine: searchSvc}
					}
					s.docSyncLoop = pkg.NewSyncLoop("knowledge-observatory", indexer.Reconciler(), docCfg)
					// Self-register the docs provider with search-hub from the same
					// `.vrooli/search.json` SSOT (descriptor mapped to the registry
					// contract). search-hub is an OPTIONAL dependency: this runs in the
					// background with bounded retry and degrades gracefully, so KO serves
					// docs search whether or not the hub is up. The upsert is idempotent.
					go s.selfRegisterSearch()
				}
			}
		}
	}
	if s.config != nil && s.config.ScenariosRoot != "" {
		service, err := viewer.NewService(s.config.ScenariosRoot)
		if err != nil {
			s.log("doc viewer service disabled", map[string]interface{}{"error": err.Error()})
		} else {
			s.docViewerService = service
		}
	}
	if s.config != nil && s.config.ScenariosRoot != "" && s.db != nil {
		jobStore := &deepsearchstore.Postgres{DB: s.db}
		agentCfg := agentmanager.DefaultDeepSearchProfileConfig()
		agentClient := agentmanager.NewDeepSearchClient(s.config.AgentManagerTimeout, agentCfg)
		promptClient := promptmanager.NewClient(s.config.PromptManagerTimeout)
		skillPath := filepath.Join(s.config.ScenariosRoot, "prompt-manager", "skills", "core", "documentation-search.md")
		skillProvider := deepsearch.CompositeSkillProvider{
			Primary:  promptClient,
			Fallback: deepsearch.FileSkillProvider{Path: skillPath},
		}
		var parser deepsearch.ResultParser = &deepsearch.JSONParser{}
		if s.config.OllamaStructuredRole != "" {
			parser = &deepsearch.JSONParser{
				Fallback: &deepsearch.OllamaParser{
					Role: s.config.OllamaStructuredRole,
				},
			}
		}
		service, err := deepsearch.NewService(s.config.ScenariosRoot, agentClient, jobStore, skillProvider, parser)
		if err != nil {
			s.log("deep search service disabled", map[string]interface{}{"error": err.Error()})
		} else {
			s.docDeepSearchService = service
		}
	}
	if s.config != nil && s.config.ScenariosRoot != "" && s.db != nil {
		jobStore := &dochealingstore.Postgres{DB: s.db}
		agentCfg := agentmanager.DefaultDocHealingProfileConfig()
		agentClient := agentmanager.NewDocHealingClient(s.config.AgentManagerTimeout, agentCfg)
		promptClient := promptmanager.NewClient(s.config.PromptManagerTimeout)
		skillPath := filepath.Join(s.config.ScenariosRoot, "prompt-manager", "skills", "core", "documentation-health.md")
		skillProvider := dochealing.CompositeSkillProvider{
			Primary:  promptClient,
			Fallback: dochealing.FileSkillProvider{Path: skillPath},
		}
		service, err := dochealing.NewService(s.config.ScenariosRoot, s.docHealthService, agentClient, jobStore, skillProvider)
		if err != nil {
			s.log("doc healing service disabled", map[string]interface{}{"error": err.Error()})
		} else {
			s.docHealingService = service
		}
	}

	if s.db != nil {
		s.docAccessLogger = &docaccessstore.Postgres{DB: s.db}
	}

	if meta != nil {
		s.materializer = &Materializer{
			VectorStore:           vs,
			Metadata:              meta,
			Now:                   time.Now,
			Sleep:                 time.Sleep,
			Interval:              5 * time.Minute,
			SampleLimit:           200,
			RelationshipThreshold: 0.85,
			MaxEdges:              500,
			MaxPairsPerVector:     25,
		}
	}
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint using api-core/health for standardized response format
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Semantic search endpoint [REQ:KO-SS-001] (re-pointed to the hybrid engine)
	s.router.HandleFunc("/api/v1/knowledge/search", s.handleSearch).Methods("POST")

	// Documentation hybrid search surface (Phase 6 cutover): converged
	// search query/status + reindex run/status/cancel verbs over the
	// vrooli-docs collection. Mirrors cli-health's verb semantics.
	s.router.HandleFunc("/api/v1/search/query", s.handleSearchQuery).Methods("POST")
	s.router.HandleFunc("/api/v1/search/status", s.handleSearchStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/reindex/run", s.handleReindexRun).Methods("POST")
	s.router.HandleFunc("/api/v1/reindex/status", s.handleReindexStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/reindex/cancel", s.handleReindexCancel).Methods("POST")

	// Knowledge health metrics endpoint [REQ:KO-QM-004]
	s.router.HandleFunc("/api/v1/knowledge/health", s.handleHealthEndpoint).Methods("GET")

	// Knowledge graph endpoint
	s.router.HandleFunc("/api/v1/knowledge/graph", s.handleGraph).Methods("GET", "POST")

	// Documentation health endpoints
	s.router.HandleFunc("/api/v1/scenarios", s.handleListScenarios).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs", s.handleDocsTree).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/reset", s.handleDocsReset).Methods("POST")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/heal", s.handleDocsHeal).Methods("POST")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/audit", s.handleDocsAudit).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/autofix", s.handleDocsAutoFix).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/search/files", s.handleDocsSearchFiles).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/search/text", s.handleDocsSearchText).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/search/unified", s.handleDocsSearchUnified).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/search/deep", s.handleDocsSearchDeep).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/search/deep/{job_id}", s.handleDocsSearchDeepStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/{doc_type}/content", s.handleDocsReadByType).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/{doc_type}/entries", s.handleDocsAppendEntry).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/stats", s.handleDocsAccessStats).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/content", s.handleDocsContent).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/reset", s.handleDocsViewerReset).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/heal/{job_id}", s.handleDocsHealStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/heal/{job_id}/approve", s.handleDocsHealApprove).Methods("POST")
	s.router.HandleFunc("/api/v1/docs/heal/{job_id}/reject", s.handleDocsHealReject).Methods("POST")

	// Documentation templates
	s.router.HandleFunc("/api/v1/docs/templates", s.handleDocsTemplateList).Methods("GET")
	s.router.HandleFunc("/api/v1/docs/templates/{doc_type}", s.handleDocsTemplateGet).Methods("GET")

	// Connect-RPC: KnowledgeObservatoryService (DocHealth + future RPCs).
	if s.docHealthService != nil {
		handler := dochealthhandler.NewWithDeps(dochealthhandler.Deps{
			Service:      s.docHealthService,
			MaturitySpec: s.docHealthMaturity,
			Environment:  s.docHealthEnvironment,
		})
		path, h := knowledgeobservatoryv1connect.NewKnowledgeObservatoryServiceHandler(handler)
		connectx.RegisterServices(s.router, connectx.ServiceMount{Path: path, Handler: h})
		validationPath, validationHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(handler)
		connectx.RegisterServices(s.router, connectx.ServiceMount{Path: validationPath, Handler: validationHandler})
	}

	// Knowledge collection inventory + maintenance (qdrant-direct utilities).
	s.router.HandleFunc("/api/v1/knowledge/collections", s.handleCollectionInventory).Methods("GET")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}", s.handleDeleteCollection).Methods("DELETE")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/diagnostics", s.handleCollectionDiagnostics).Methods("GET")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/records", s.handleCollectionRecords).Methods("GET")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/maintenance/prune-stale-chunks", s.handlePruneStaleChunks).Methods("POST")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/maintenance/dedupe-content", s.handleDedupeContent).Methods("POST")
}

func (s *Server) handler() http.Handler {
	handler := handlers.RecoveryHandler()(s.router)
	return handlers.CORS(s.corsOptions()...)(handler)
}

func (s *Server) corsOptions() []handlers.CORSOption {
	allowed := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(allowed) > 0 {
		allowedSet := map[string]struct{}{}
		for _, origin := range allowed {
			allowedSet[origin] = struct{}{}
		}
		return []handlers.CORSOption{
			handlers.AllowedOriginValidator(func(origin string) bool {
				_, ok := allowedSet[origin]
				return ok
			}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		}
	}

	return []handlers.CORSOption{
		handlers.AllowedOriginValidator(isLocalOrigin),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	}
}

func parseAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func isLocalOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()(w, r)
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "knowledge-observatory-api",
		"port":    s.config.Port,
	})

	runnerCtx, runnerCancel := context.WithCancel(context.Background())
	if s.materializer != nil && s.db != nil {
		go s.materializer.Run(runnerCtx)
	}
	if s.docSyncLoop != nil {
		go s.docSyncLoop.Start(runnerCtx)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      s.handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	runnerCancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	log.Printf("%s | %v", msg, fields)
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario start <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func (s *Server) qdrantURL() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.QdrantURL); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("QDRANT_URL")); value != "" {
		return value
	}
	return "http://localhost:6333"
}

func (s *Server) qdrantAPIKey() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.QdrantAPIKey); value != "" {
			return value
		}
	}
	return strings.TrimSpace(os.Getenv("QDRANT_API_KEY"))
}

// loadDocsTuning reads the docs search tuning from the scenario-owned
// `.vrooli/search.json` SSOT (provider knowledge-observatory.docs). On a
// missing/malformed file it logs and falls back to the typed DocCorpusTuning
// preset — whose values are exactly what the committed file holds — so docs
// search degrades to its measured-best baseline rather than failing to boot.
func (s *Server) loadDocsTuning() pkg.TuningConfig {
	path := filepath.Join(s.config.ScenariosRoot, "knowledge-observatory", ".vrooli", "search.json")
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		s.log("docs search tuning: falling back to DocCorpus preset", map[string]interface{}{"error": err.Error(), "path": path})
		return pkg.DocCorpusTuning()
	}
	provider, ok := file.Provider("knowledge-observatory.docs")
	if !ok {
		s.log("docs search tuning: provider missing, falling back to DocCorpus preset", map[string]interface{}{"path": path})
		return pkg.DocCorpusTuning()
	}
	tuning := provider.ResolvedTuning()
	// KO's docs read+index path (internal/aisearch) is HYBRID BY CONSTRUCTION: the
	// indexer always builds a NewHybridBinding (dense + BM25 sparse) and the read
	// service always carries a sparse leg — there is no dense code path. So
	// tuning.engine is a fixed invariant for this provider, not a tunable axis
	// (KO also exposes no config_endpoint, so search-hub cannot sweep it). Enforce
	// the invariant loudly rather than silently honoring a value the engine can't
	// satisfy: an SSOT that says anything but "hybrid" (including the WithDefaults
	// "dense" fallback when the field is omitted) is corrected to hybrid with a
	// warning, so the running config can never diverge from what is actually built.
	if tuning.Engine != pkg.EngineHybrid {
		s.log("docs search tuning: engine pinned to hybrid (KO is hybrid by construction; SSOT value ignored)",
			map[string]interface{}{"ssot_engine": tuning.Engine, "path": path})
		tuning.Engine = pkg.EngineHybrid
	}
	return tuning
}

// selfRegisterSearch pushes KO's `knowledge-observatory.docs` provider to
// search-hub from the `.vrooli/search.json` SSOT. It is best-effort and blocks
// only its own goroutine; failures are logged inside searchregister.Register and
// returned as error-bearing Results (search-hub is optional).
func (s *Server) selfRegisterSearch() {
	path := filepath.Join(s.config.ScenariosRoot, "knowledge-observatory", ".vrooli", "search.json")
	searchregister.Register(context.Background(), searchregister.Config{
		ScenarioID:     "knowledge-observatory",
		SearchFilePath: path,
		Logger:         log.Default(),
		// Cache the control token search-hub mints so the reindex handler can
		// validate token-gated requests, and echo it on re-registration so a
		// different actor can't hijack the provider_id (mirrors cli-health).
		OnControlToken: func(_ string, token string) { s.searchToken.Set(token) },
		ControlToken:   func(string) string { return s.searchToken.Get() },
	})
}

func (s *Server) ollamaEmbeddingRole() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.OllamaEmbeddingRole); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_ROLE")); value != "" {
		return value
	}
	return "embedding.default"
}

func (s *Server) resourceQdrantCLI() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.ResourceQdrantCLI); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("RESOURCE_QDRANT_CLI")); value != "" {
		return value
	}
	return "resource-qdrant"
}

func (s *Server) resourceCommandTimeout() time.Duration {
	if s == nil || s.config == nil || s.config.ResourceCommandTimeout <= 0 {
		return 5 * time.Second
	}
	return s.config.ResourceCommandTimeout
}

func (s *Server) execResourceQdrant(ctx context.Context, args ...string) ([]byte, error) {
	timeout := s.resourceCommandTimeout()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, s.resourceQdrantCLI(), args...)
	return cmd.Output()
}

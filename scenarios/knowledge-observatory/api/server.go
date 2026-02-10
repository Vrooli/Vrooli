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
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"

	"knowledge-observatory/internal/adapters/agentmanager"
	"knowledge-observatory/internal/adapters/deepsearchstore"
	"knowledge-observatory/internal/adapters/docaccessstore"
	"knowledge-observatory/internal/adapters/dochealingstore"
	"knowledge-observatory/internal/adapters/embedder"
	"knowledge-observatory/internal/adapters/jobstore"
	"knowledge-observatory/internal/adapters/metadatastore"
	"knowledge-observatory/internal/adapters/promptmanager"
	"knowledge-observatory/internal/adapters/vectorstore"
	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/deepsearch"
	"knowledge-observatory/internal/services/dochealing"
	"knowledge-observatory/internal/services/dochealth"
	"knowledge-observatory/internal/services/docsearch"
	"knowledge-observatory/internal/services/explorer"
	"knowledge-observatory/internal/services/graph"
	"knowledge-observatory/internal/services/ingest"
	"knowledge-observatory/internal/services/ingestjobs"
	"knowledge-observatory/internal/services/search"
	"knowledge-observatory/internal/services/viewer"
)

// Config holds minimal runtime configuration
type Config struct {
	Port                   string
	DatabaseURL            string
	QdrantURL              string
	QdrantAPIKey           string
	OllamaURL              string
	OllamaEmbeddingModel   string
	OllamaStructuredModel  string
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
	metadata    ports.MetadataStore
	jobStore    ports.JobStore

	ingestService        *ingest.Service
	searchService        *search.Service
	graphService         *graph.Service
	docHealthService     *dochealth.Service
	docSearchService     *docsearch.Service
	docExplorerService   *explorer.Service
	docViewerService     *viewer.Service
	docDeepSearchService deepsearch.API
	docHealingService    dochealing.API

	docAccessLogger ports.DocAccessLogger

	ingestJobRunner *ingestjobs.Runner
	materializer    *Materializer
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port:                   requireEnv("API_PORT"),
		QdrantURL:              strings.TrimSpace(os.Getenv("QDRANT_URL")),
		QdrantAPIKey:           strings.TrimSpace(os.Getenv("QDRANT_API_KEY")),
		OllamaURL:              strings.TrimSpace(os.Getenv("OLLAMA_URL")),
		OllamaEmbeddingModel:   strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_MODEL")),
		OllamaStructuredModel:  strings.TrimSpace(os.Getenv("OLLAMA_STRUCTURED_OUTPUT_MODEL")),
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
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
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
		BaseURL: s.ollamaURL(),
		Model:   s.ollamaEmbeddingModel(),
	}

	var meta *metadatastore.Postgres
	if s.db != nil {
		meta = &metadatastore.Postgres{DB: s.db}
	}

	var js ports.JobStore
	if s.db != nil {
		js = &jobstore.Postgres{DB: s.db}
	}

	s.vectorStore = vs
	s.embedder = emb
	s.metadata = meta
	s.jobStore = js

	s.ingestService = &ingest.Service{
		VectorStore: vs,
		Embedder:    emb,
		Metadata:    meta,
	}

	s.searchService = &search.Service{
		VectorStore: vs,
		Embedder:    emb,
		Metadata:    meta,
	}

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
			service.Semantic = s.searchService
			s.docSearchService = service
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
		if s.config.OllamaURL != "" && s.config.OllamaStructuredModel != "" {
			parser = &deepsearch.JSONParser{
				Fallback: &deepsearch.OllamaParser{
					BaseURL: s.config.OllamaURL,
					Model:   s.config.OllamaStructuredModel,
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

	if js != nil {
		s.ingestJobRunner = &ingestjobs.Runner{
			Jobs:      js,
			Ingest:    s.ingestService,
			Now:       time.Now,
			Sleep:     time.Sleep,
			MaxChunks: maxChunksPerDoc,
		}
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

	// Semantic search endpoint [REQ:KO-SS-001]
	s.router.HandleFunc("/api/v1/knowledge/search", s.handleSearch).Methods("POST")

	// Knowledge health metrics endpoint [REQ:KO-QM-004]
	s.router.HandleFunc("/api/v1/knowledge/health", s.handleHealthEndpoint).Methods("GET")

	// Knowledge graph endpoint
	s.router.HandleFunc("/api/v1/knowledge/graph", s.handleGraph).Methods("GET", "POST")

	// Documentation health endpoints
	s.router.HandleFunc("/api/v1/scenarios", s.handleListScenarios).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs", s.handleDocsTree).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{name}/docs/health", s.handleDocsHealth).Methods("GET")
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

	// Canonical knowledge write path (records) - sync upsert
	s.router.HandleFunc("/api/v1/knowledge/records/upsert", s.handleUpsertRecord).Methods("POST")
	s.router.HandleFunc("/api/v1/knowledge/records/{record_id}", s.handleDeleteRecord).Methods("DELETE")
	s.router.HandleFunc("/api/v1/knowledge/documents/ingest", s.handleIngestDocument).Methods("POST")
	s.router.HandleFunc("/api/v1/knowledge/documents/delete", s.handleDeleteDocument).Methods("POST")
	s.router.HandleFunc("/api/v1/knowledge/collections", s.handleCollectionInventory).Methods("GET")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}", s.handleDeleteCollection).Methods("DELETE")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/diagnostics", s.handleCollectionDiagnostics).Methods("GET")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/records", s.handleCollectionRecords).Methods("GET")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/maintenance/prune-stale-chunks", s.handlePruneStaleChunks).Methods("POST")
	s.router.HandleFunc("/api/v1/knowledge/collections/{collection}/maintenance/dedupe-content", s.handleDedupeContent).Methods("POST")

	// Async ingest jobs
	s.router.HandleFunc("/api/v1/ingest/jobs", s.handleCreateIngestJob).Methods("POST")
	s.router.HandleFunc("/api/v1/ingest/jobs/{job_id}", s.handleGetIngestJob).Methods("GET")
	s.router.HandleFunc("/api/v1/ingest/health", s.handleIngestHealth).Methods("GET")
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
	if s.ingestJobRunner != nil && s.db != nil {
		go s.ingestJobRunner.Run(runnerCtx)
	}
	if s.materializer != nil && s.db != nil {
		go s.materializer.Run(runnerCtx)
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

func (s *Server) ollamaURL() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.OllamaURL); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("OLLAMA_URL")); value != "" {
		return value
	}
	return "http://localhost:11434"
}

func (s *Server) ollamaEmbeddingModel() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.OllamaEmbeddingModel); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_MODEL")); value != "" {
		return value
	}
	return "nomic-embed-text"
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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"web-search/internal/clock"
	"web-search/internal/findingindex"
	"web-search/internal/modules"
	"web-search/internal/server"

	aisearchpkg "github.com/vrooli/aisearch-go"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	findingsH "web-search/handlers/findings"
	healthH "web-search/handlers/health"
	livesearchH "web-search/handlers/livesearch"
	researchH "web-search/handlers/research"
	internalfindings "web-search/internal/findings"
	internallivesearch "web-search/internal/livesearch"
	internalresearch "web-search/internal/research"
	researchagent "web-search/internal/research/agentmanager"
	researchfetch "web-search/internal/research/fetch"
)

// findingsProviderID is the search-hub provider id whose tuning block in
// .vrooli/search.json drives the findings semantic index.
const findingsProviderID = "web-search.learnings"

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("web-search")
	if err != nil {
		return "", fmt.Errorf("resolve web-search storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"web-search.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve web-search db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "web-search"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	logger := log.Default()

	// Boot-time control surface (docs/reference/configuration.md):
	// WEB_SEARCH_-prefixed levers, each zero when unset so the compiled
	// defaults in the owning packages stay the SSOT. The decay half-life is
	// applied before any service is constructed so the GC min-age default
	// derives from the effective value.
	tuning := tuningFromEnv()
	if tuning.DecayHalfLife > 0 {
		internalfindings.SetDecayHalfLife(tuning.DecayHalfLife)
	}

	// AI search wiring for the findings knowledge store. The scenario-owned
	// .vrooli/search.json is the SSOT for the search tuning (engine shape, embed
	// recipe, rerank policy, floor band); the shared aisearch-go engine provides
	// the embedder + vector store + reconciler + sync loop. LoadConfig supplies
	// the operational wiring (qdrant address, sync cadence, parallelism, reranker
	// resource endpoints) from env, not the tuning. The index is best-effort:
	// when qdrant/ollama are unreachable the rest of the API still serves.
	searchCfg := aisearchpkg.LoadConfig("WEB_SEARCH")
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "web-search", ".vrooli", "search.json")
	findingsTuning := loadFindingsTuning(searchJSONPath, findingsProviderID)

	indexFindings := internalfindings.NewService(internalfindings.NewSQLiteRepository(db, clock.System{}))
	searcher := findingindex.New(findingsTuning, findingindex.Options{
		Loader:           indexFindings.LoadIndexable,
		Parallelism:      searchCfg.ReconcileParallelism,
		MaxEmbedsPerTick: searchCfg.MaxEmbedsPerTick,
		EngineDeps: aisearchpkg.EngineDeps{
			QdrantURL:     searchCfg.QdrantURL,
			QdrantAPIKey:  searchCfg.QdrantAPIKey,
			Collection:    findingindex.DefaultCollection,
			RerankerURL:   searchCfg.RerankerURL,
			RerankerModel: searchCfg.RerankerModel,
			RerankRole:    searchCfg.RerankRole,
		},
	})

	if err := searcher.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[web-search] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}

	// Sync loop drives periodic reconcile of the findings index against qdrant.
	// Cancelled on shutdown via the Cleanup hook below.
	syncCtx, cancelSync := context.WithCancel(context.Background())
	syncLoop := aisearchpkg.NewSyncLoopFunc("web-search", searcher.Reconciler, searchCfg)
	go syncLoop.Start(syncCtx)

	// Self-register both search providers (web-search.learnings SCOPE_PROJECT and
	// web-search.live SCOPE_EXTERNAL) with search-hub from the same
	// .vrooli/search.json SSOT. search-hub is an OPTIONAL dependency, so this runs
	// in the background with bounded retry and degrades gracefully; the upserts
	// are idempotent, so re-registering on every boot is safe. The returned
	// control tokens are cached so a re-registration can echo them as the
	// ownership proof.
	tokens := newTokenHolder()
	go searchregister.Register(syncCtx, searchregister.Config{
		ScenarioID:     "web-search",
		SearchFilePath: searchJSONPath,
		Logger:         logger,
		OnControlToken: func(providerID, token string) { tokens.set(providerID, token) },
		ControlToken:   func(providerID string) string { return tokens.get(providerID) },
	})

	// Live web search (L0) + optional snippet synthesis (L1). The SearXNG
	// client, in-memory TTL cache, budget governor, and synthesizer are all
	// seams constructed here from env (with localhost defaults for SearXNG and
	// Ollama); the service owns the orchestration. Synthesis is off unless a
	// request opts in, and the raw L0 results are never blocked by it.
	liveClock := clock.System{}
	liveService := internallivesearch.NewService(internallivesearch.Deps{
		Client:      internallivesearch.NewHTTPSearxngClient(os.Getenv("SEARXNG_URL"), nil),
		Cache:       internallivesearch.NewCache(tuning.CacheTTL, liveClock),
		Governor:    internallivesearch.NewGovernor(tuning.GovernorCapacity, internallivesearch.DefaultGovernorWindow, liveClock),
		Synthesizer: internallivesearch.NewOllamaSynthesizer(os.Getenv("OLLAMA_SYNTHESIS_ROLE")),
		Logger:      logger,
	})

	// L2/L3 deep research. L2 reuses the live-search service for candidate URLs,
	// the HTTP-first fetch stack (with per-URL escalation to a
	// browser-automation-studio capture for JS-shell pages) for full-page
	// readable text, and an Ollama chat model for the single-pass cited
	// synthesis. L3 hands the iterative research-and-reconcile loop to an
	// agent-manager run. Every seam is constructed from env with graceful
	// defaults; all are best-effort, so the API still serves when
	// browser-automation-studio/agent-manager are down (L2 degrades to
	// HTTP-only fetch and abstains only if every page fails, L3 surfaces
	// Unavailable). Capture (L3 always, L2 opt-in) writes to the same findings
	// store, attributed to the "agent" actor.
	// Wrap with the index kick: an L2 --capture or L3 distillation write
	// becomes searchable within the kick debounce (back-to-back L3 runs can
	// GATHER each other's fresh findings) instead of waiting out the sync
	// interval, which stays on as the repair cadence.
	researchFindings := internalfindings.WithMutationNotify(
		internalfindings.NewServiceWithActor(internalfindings.NewSQLiteRepository(db, clock.System{}), "agent"),
		syncLoop.Kick)
	// L2 excerpting: relevance-aware by default (reusing the findings index's
	// tuned embedder so chunk scoring lives in the same vector space), with
	// the escape-hatch lever reverting to positional truncation. The
	// relevance excerpter itself degrades to positional when the embedder is
	// unreachable, so this wiring never makes L2 less available.
	var excerpter internalresearch.Excerpter = internalresearch.RelevantExcerpter{
		Embedder: searcher.Embedder(),
		Budget:   tuning.SynthExcerptChars,
		Logger:   logger,
	}
	if tuning.RelevantExcerptsOff {
		excerpter = internalresearch.PositionalExcerpter{Budget: tuning.SynthExcerptChars}
	}
	researchService := internalresearch.NewService(internalresearch.Deps{
		Searcher:    internalresearch.LiveSearcher{Service: liveService},
		Fetcher:     newL2Fetcher(tuning, logger),
		Synthesizer: internalresearch.NewOllamaSynthesizer(os.Getenv("OLLAMA_SYNTHESIS_ROLE")),
		Excerpter:   excerpter,
		Findings:    researchFindings,
		// Bounded GATHER (OT-P1-003): the semantic findings index supplies nearby
		// ids, the findings store hydrates them. The hard cap is enforced inside
		// the research service, not here.
		Gatherer: internalresearch.IndexGatherer{
			Index: findingIndexAdapter{idx: searcher},
			Store: researchFindings,
		},
		AgentManager: researchagent.NewService(researchagent.NewHTTPClient()),
		Logger:       logger,

		ConfidenceGate:   tuning.ConfidenceGate,
		GatherCap:        tuning.GatherCap,
		MaxResearchLoops: tuning.MaxResearchLoops,
	})

	// Usage telemetry (OT-P2-001): the async surfacing recorder counts which
	// findings a search returned, entirely off the hot path. It drains on the
	// same sync context so it stops on shutdown.
	usageRecorder := internalfindings.NewUsageRecorder(
		internalfindings.NewSQLiteRepository(db, clock.System{}),
		internalfindings.DefaultUsageBuffer, logger)
	go usageRecorder.Run(syncCtx)

	// Periodic store-consistency GC (OT-P2-003): a background sweep, separate from
	// per-query reconcile, that soft-retires never-surfaced fully-decayed findings
	// (confidence-gated) and reports cold-archive/stale-dispute/orphan drift. The
	// interval is configurable via WEB_SEARCH_GC_INTERVAL (0/unset ⇒ disabled, so
	// the on-demand `findings gc` CLI is the only path until an operator opts in).
	if gcInterval := gcIntervalFromEnv(); gcInterval > 0 {
		gcRunner := internalfindings.NewGCService(
			internalfindings.WithMutationNotify(
				internalfindings.NewService(internalfindings.NewSQLiteRepository(db, clock.System{})),
				syncLoop.Kick),
			clock.System{}, internalfindings.GCConfig{})
		go runGCLoop(syncCtx, gcRunner, gcInterval, logger)
	}

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "web-search-api", "1.0.0"),
		findingsH.Module(db, clock.System{}, searcher, usageRecorder, syncLoop.Kick, logger),
		livesearchH.Module(liveService, logger),
		researchH.Module(researchService, logger),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	// /measures is the measures-go serve substrate: the central measures index
	// harvests <prefix>/declarations and the auto-execution path POSTs
	// <prefix>/execute. The findings domain owns the canonical measure
	// (findings.count).
	findingsMeasures, err := findingsH.MeasuresHandler(db, clock.System{})
	if err != nil {
		log.Fatalf("measures registry: %v", err)
	}
	rootMux.Handle("/measures/", http.StripPrefix("/measures", findingsMeasures))

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// RunL2 is synchronous and legitimately slow: sequential page fetches
		// (15s budget each) plus an LLM synthesis (60s budget, longer when the
		// model cold-loads on a CPU-resident ollama). The api-core default
		// WriteTimeout of 30s would sever the connection mid-synthesis.
		WriteTimeout: 5 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			cancelSync()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// gcIntervalFromEnv reads the OT-P2-003 GC cadence from WEB_SEARCH_GC_INTERVAL
// (a Go duration like "24h"). A zero/unset/invalid value disables the background
// loop — the on-demand `findings gc` CLI still works.
func gcIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("WEB_SEARCH_GC_INTERVAL"))
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

// runGCLoop runs the store-consistency GC on a ticker until ctx is cancelled.
// Each run is best-effort: a failure is logged, never fatal.
func runGCLoop(ctx context.Context, gc *internalfindings.GCService, interval time.Duration, logger *log.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report, err := gc.Run(ctx, false)
			if err != nil {
				logger.Printf("[web-search] periodic GC failed: %v", err)
				continue
			}
			logger.Printf("[web-search] periodic GC: superseded=%d cold-archive=%d stale-disputes=%d orphans=%d",
				len(report.SupersededDecayed), len(report.ColdArchiveCandidates), len(report.StaleDisputes), len(report.Orphans))
		}
	}
}

// newL2Fetcher assembles the L2 fetch stack: a plain HTTP leg first, with
// per-URL escalation to a browser-automation-studio capture (the greenfield
// replacement for the deleted browserless coupling — browser rendering goes
// through the BAS scenario only). Escalation is default-ON and degrades
// gracefully to HTTP-only when BAS is unreachable;
// WEB_SEARCH_BROWSER_ESCALATION=off removes the browser leg entirely.
func newL2Fetcher(tuning Tuning, logger *log.Logger) internalresearch.Fetcher {
	var browserLeg researchfetch.Fetcher
	if !tuning.BrowserEscalationOff {
		browserLeg = researchfetch.NewBASFetcher()
	}
	return &researchfetch.EscalatingFetcher{
		HTTP:             researchfetch.NewHTTPFetcher(tuning.FetchTimeout, int64(tuning.FetchMaxBytes)),
		Browser:          browserLeg,
		MinReadableChars: tuning.MinReadableChars,
		Logger:           logger,
	}
}

// findingIndexAdapter bridges the concrete findingindex.Service to the research
// package's SemanticIndex seam, projecting findingindex.Hit -> research.GatherHit
// (id + score) so the research package stays decoupled from the index package.
type findingIndexAdapter struct {
	idx *findingindex.Service
}

func (a findingIndexAdapter) Search(ctx context.Context, query string, limit int) ([]internalresearch.GatherHit, error) {
	hits, _, err := a.idx.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]internalresearch.GatherHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, internalresearch.GatherHit{FindingID: h.FindingID, Score: h.Score})
	}
	return out, nil
}

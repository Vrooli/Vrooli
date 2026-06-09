package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cli-health/internal/aisearch"
	"cli-health/internal/clock"
	"cli-health/internal/modules"
	"cli-health/internal/server"

	aisearchpkg "github.com/vrooli/aisearch-go"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	healthH "cli-health/handlers/health"
	searchH "cli-health/handlers/search"
	searchcontrolH "cli-health/handlers/searchcontrol"
	validationH "cli-health/handlers/validation"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
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
	path, err := resolver.Path(
		storage.Options{ScenarioID: "cli-health"},
		storage.ClassData,
		"cli-health.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve cli-health db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "cli-health"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	logger := log.Default()
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}

	// The scenario-owned search.json is the single source of truth for the search
	// descriptor + tuning + tests. Both the boot tuning read and the self-register
	// push read it, and the config-write control RPC rewrites it, so the path is
	// resolved once here.
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "cli-health", ".vrooli", "search.json")

	// AI search wiring: the scenario-owned `.vrooli/search.json` is the SSOT for
	// the search tuning factors (engine shape, embed recipe, rerank policy, floor
	// band) — read here at boot. The shared engine (packages/aisearch-go) provides
	// the embedder + vector store + reconciler + sync loop; LoadConfig supplies
	// only the OPERATIONAL wiring (Qdrant address, sync cadence, parallelism,
	// reranker resource endpoints), not the tuning. NewServiceForTuning picks dense
	// vs hybrid from the tuning DATA, so the engine shape is no longer a code
	// literal. The CLI_HEALTH_{RERANK_*,EMBED_TASK_PREFIX,RELEVANCE_*} tuning env
	// vars are no longer consulted — search.json replaces them.
	searchCfg := aisearchpkg.LoadConfig("CLI_HEALTH")
	tuning := loadSearchTuning(searchJSONPath, commandsProviderID)
	discovery := aisearch.NewFilesystemDiscoverySource(repoRoot)
	// Index the top-level vrooli CLI alongside scenario CLIs. The
	// records carry Origin="vrooli"; the validation handler rejects
	// "vrooli" because no proto contract exists for it.
	discovery.ExternalCLIs = []aisearch.ExternalCLI{{Name: "vrooli", Binary: "vrooli"}}
	// NewTunedService assembles the engine FROM the tuning (engine shape, embed
	// recipe, rerank policy, floor band) and retains the builder so the search
	// control plane's WriteConfig can re-assemble + re-embed in place when a sweep
	// changes an index-time factor — no restart. The wiring (Qdrant, reranker
	// resource endpoints, reconcile cadence) stays in EngineDeps/LoadConfig.
	aiService := aisearch.NewTunedService(tuning, aisearch.TunedOptions{
		Discovery:        discovery,
		Parallelism:      searchCfg.ReconcileParallelism,
		MaxEmbedsPerTick: searchCfg.MaxEmbedsPerTick,
		EngineDeps: aisearchpkg.EngineDeps{
			QdrantURL:     searchCfg.QdrantURL,
			QdrantAPIKey:  searchCfg.QdrantAPIKey,
			Collection:    aisearch.DefaultCollection,
			RerankerURL:   searchCfg.RerankerURL,
			RerankerModel: searchCfg.RerankerModel,
			RerankModel:   searchCfg.RerankModel,
		},
	})

	// EnsureCollection is best-effort: if qdrant is unreachable at boot, the
	// scenario still serves text-fallback search and a degraded status.
	if err := aiService.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[cli-health] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}

	// Sync loop drives periodic reconcile against qdrant. Cancelled by the
	// api-core server's shutdown context.
	syncCtx, cancelSync := context.WithCancel(context.Background())
	// Resolve the reconciler each tick (not bound once) so a live ApplyTuning swap
	// re-points the loop at the new engine — otherwise the loop would keep
	// reconciling with the old recipe and the drift hash would undo the apply.
	syncLoop := aisearchpkg.NewSyncLoopFunc("cli-health", aiService.Reconciler, searchCfg)
	go syncLoop.Start(syncCtx)

	// The override gate is the OUTER security layer of the query-time override
	// channel: search-hub's sweep/A-B can vary rerank/floor/shortlist per request,
	// but only when the request carries the control token search-hub minted for
	// this provider. The token is cached in memory from the self-registration echo
	// below; until then Get() returns "" and the gate stays closed. Since
	// search-hub is the only holder of that token, only its sweep can apply
	// overrides — no env flag is needed (a public request carries no token and
	// gets ordinary search).
	controlToken := searchH.NewTokenHolder()
	overrideGate := &searchH.OverrideGate{Token: controlToken.Get}

	// The control gate guards the SHARED reindex + config-write plane
	// (search-hub.v1.control.SearchControlService). It shares the same minted
	// control token; the token alone gates the mutating verbs. A provider that does
	// not want to be tuned at all omits its control endpoints in search.json (the
	// control client then gets ErrNoControlPlane) — tunability is declared in the
	// SSOT, not toggled by an env var.
	controlGate := &searchcontrolH.Gate{Token: controlToken.Get}

	// Self-register this scenario's search provider(s) AND their evaluation corpus
	// with search-hub from the same `.vrooli/search.json` SSOT: the descriptor goes
	// to the registry, the tests block is mirrored into the eval store as
	// "<provider_id>.primary" (corpusStoreMirrorsFile — the store is a cache of the
	// file, healed on every boot). search-hub is an OPTIONAL dependency, so this
	// runs in the background with bounded retry and degrades gracefully — the
	// scenario serves search whether or not the hub is up. Both upserts are
	// idempotent, so re-registering on every boot is safe. The returned control
	// token is cached so the override gate above can validate it.
	go searchregister.Register(syncCtx, searchregister.Config{
		ScenarioID:     "cli-health",
		SearchFilePath: searchJSONPath,
		Logger:         logger,
		OnControlToken: func(_ string, token string) { controlToken.Set(token) },
		// Echo the cached token on re-registration so a different actor can't hijack
		// the provider_id. Empty until the first registration completes (in-memory
		// holder); the hub then treats it as first-contact. Single provider, so the
		// provider id is irrelevant here.
		ControlToken: func(string) string { return controlToken.Get() },
	})

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "cli-health-api", "1.0.0"),
		validationH.Module(logger, repoRoot, externalCLINames(discovery.ListExternalCLIs())),
		searchH.Module(logger, aiService, overrideGate),
		searchcontrolH.Module(logger, searchcontrolH.Deps{
			Logger:       logger,
			Reindexer:    searchcontrolH.ServiceAdapter{Service: aiService},
			ConfigWriter: searchcontrolH.FileConfigWriter{Path: searchJSONPath},
			CorpusWriter: searchcontrolH.FileCorpusWriter{Path: searchJSONPath},
			Gate:         controlGate,
		}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			cancelSync()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// commandsProviderID is the provider id cli-health owns in its search.json SSOT.
const commandsProviderID = "cli-health.commands"

// loadSearchTuning reads the search tuning for a provider from the scenario-owned
// `.vrooli/search.json` (the SSOT). The file is committed and authoritative, so a
// missing/malformed file or an absent provider is a fatal boot error — there is
// no env/code fallback by design (greenfield §0).
func loadSearchTuning(path, providerID string) aisearchpkg.TuningConfig {
	file, err := aisearchpkg.LoadSearchFile(path)
	if err != nil {
		log.Fatalf("load search tuning: %v", err)
	}
	provider, ok := file.Provider(providerID)
	if !ok {
		log.Fatalf("load search tuning: provider %q not found in %s", providerID, path)
	}
	return provider.ResolvedTuning()
}

func externalCLINames(clis []aisearch.ExternalCLI) []string {
	out := make([]string, 0, len(clis))
	for _, c := range clis {
		out = append(out, c.Name)
	}
	return out
}

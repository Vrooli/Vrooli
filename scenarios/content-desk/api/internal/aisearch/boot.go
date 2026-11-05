package aisearch

import (
	"context"
	"log"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

// envPrefix scopes the shared engine's operational env under CONTENT_DESK_*
// (<prefix>_QDRANT_URL, _SYNC_INTERVAL, _RECONCILE_PARALLELISM, …). The search
// tuning factors themselves live in the scenario-owned .vrooli/search.json.
const envPrefix = "CONTENT_DESK"

// Start assembles the engine-backed read path over the editorial source from the
// scenario-owned search.json tuning plus the operational env. It never fails
// boot: an unreadable provider, an unresolved embedding policy, or an
// unreachable vector store degrades to the lexical Service, which stays
// authoritative offline. The returned LiveSearch is safe to serve immediately.
func Start(ctx context.Context, source *StoreSource, searchJSONPath string, logger *log.Logger) *LiveSearch {
	if logger == nil {
		logger = log.Default()
	}
	lexical := NewService(source)

	provider, ok := loadProvider(searchJSONPath)
	if !ok {
		logger.Printf("[content-desk/aisearch] provider %q unavailable in %s — serving lexical search", ProviderID, searchJSONPath)
		return NewLexicalSearch(lexical)
	}
	tuning := provider.ResolvedTuning()

	cfg := pkg.LoadConfig(envPrefix)
	deps := pkg.EngineDeps{
		QdrantURL:     cfg.QdrantURL,
		QdrantAPIKey:  cfg.QdrantAPIKey,
		Collection:    Collection,
		EmbedRole:     cfg.EmbedRole,
		RerankerURL:   cfg.RerankerURL,
		RerankerModel: cfg.RerankerModel,
		RerankRole:    cfg.RerankRole,
	}
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	resolved, err := pkg.ResolveEngineDepsEmbedding(resolveCtx, deps)
	cancel()
	if err != nil {
		logger.Printf("[content-desk/aisearch] embedding policy unresolved (degraded lexical search): %v", err)
		return NewLexicalSearch(lexical)
	}
	deps = resolved

	engine := NewTunedEngine(tuning, deps, NewEditorialSource(source), textFallbackOf(lexical), cfg.ReconcileParallelism, cfg.MaxEmbedsPerTick)
	if err := engine.EnsureCollection(ctx); err != nil {
		logger.Printf("[content-desk/aisearch] collection ensure failed (degraded lexical search): %v", err)
	} else if !cfg.SyncDisabled {
		loop := pkg.NewSyncLoopFunc(envPrefix, engine.Reconciler, cfg)
		go loop.Start(ctx)
		go func() {
			if _, _, err := loop.RunOnce(ctx); err != nil {
				logger.Printf("[content-desk/aisearch] initial reconcile failed (continuing degraded): %v", err)
			}
		}()
	}
	return NewLiveSearch(engine, lexical)
}

// loadProvider reads the single content-desk provider block from search.json. A
// missing/malformed file or an absent provider is a degraded lexical boot, never
// a fatal one: search is an enhancement over the authoritative corpus.
func loadProvider(path string) (pkg.ProviderConfig, bool) {
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		log.Printf("[content-desk/aisearch] load search.json (%s): %v", path, err)
		return pkg.ProviderConfig{}, false
	}
	return file.Provider(ProviderID)
}

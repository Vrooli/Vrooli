package aisearch

import (
	"context"
	"log"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

// DefaultTuning is the v1 engine recipe for the governance corpus: dense,
// nomic task prefixes on, rerank off. (T2 federation moves this into
// .vrooli/search.json so it becomes a sweep-tunable lever.)
func DefaultTuning() pkg.TuningConfig {
	return pkg.TuningConfig{
		Engine:          pkg.EngineDense,
		EmbedModel:      pkg.DefaultEmbedModel,
		EmbedTaskPrefix: true,
		RerankEnabled:   false,
	}.WithDefaults()
}

// Start builds the governance-records search service from env config, ensures
// the Qdrant collection (best-effort — a down backend degrades to text search,
// never fails boot), and launches the background reconcile loop bound to ctx.
// It never returns an error: search is an enhancement, not a boot dependency.
func Start(ctx context.Context, provider RecordProvider) *Service {
	cfg := pkg.LoadConfig(envPrefix)
	deps := pkg.EngineDeps{
		QdrantURL:     cfg.QdrantURL,
		QdrantAPIKey:  cfg.QdrantAPIKey,
		Collection:    DefaultCollection,
		EmbedRole:     cfg.EmbedRole,
		EmbedModel:    cfg.EmbedModel,
		RerankerURL:   cfg.RerankerURL,
		RerankerModel: cfg.RerankerModel,
		RerankRole:    cfg.RerankRole,
	}
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	resolved, err := pkg.ResolveEngineDepsEmbedding(resolveCtx, deps)
	cancel()
	if err != nil {
		log.Printf("[scenario-dependency-analyzer/aisearch] embedding policy unresolved (continuing degraded): %v", err)
	} else {
		deps = resolved
	}

	svc := New(provider, DefaultTuning(), deps, cfg.ReconcileParallelism, cfg.MaxEmbedsPerTick)

	if err := svc.EnsureCollection(ctx); err != nil {
		log.Printf("[scenario-dependency-analyzer/aisearch] qdrant collection ensure failed (degraded text search): %v", err)
	}

	if !cfg.SyncDisabled {
		loop := pkg.NewSyncLoopFunc(envPrefix, svc.Reconciler, cfg)
		go loop.Start(ctx)
	}

	return svc
}

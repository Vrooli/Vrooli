package aisearch

import (
	"context"
	"log"
	"strings"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

// Start builds SDA's multi-corpus search service from env config + the wired
// data sources, ensures every corpus's Qdrant collection (best-effort — a down
// backend degrades to text search, never fails boot), and launches ONE
// background reconcile loop that converges all corpora. It never returns an
// error: search is an enhancement, not a boot dependency.
//
// The embedding policy is resolved ONCE and shared across corpora (they share a
// single embedder/Reconciler); per-corpus Collection is set inside New.
func Start(ctx context.Context, sources Sources) *Service {
	cfg := pkg.LoadConfig(envPrefix)
	deps := pkg.EngineDeps{
		QdrantURL:     cfg.QdrantURL,
		QdrantAPIKey:  cfg.QdrantAPIKey,
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

	specs := buildCorpusSpecs(sources)
	applyTuningFromSearchJSON(specs, sources.SearchJSONPath)
	svc := New(specs, deps, cfg.ReconcileParallelism, cfg.MaxEmbedsPerTick)

	if err := svc.EnsureCollections(ctx); err != nil {
		log.Printf("[scenario-dependency-analyzer/aisearch] qdrant collection ensure failed (degraded text search): %v", err)
	}

	if !cfg.SyncDisabled {
		loop := pkg.NewSyncLoopFunc(envPrefix, svc.Reconciler, cfg)
		go loop.Start(ctx)
		go func() {
			if _, _, err := loop.RunOnce(ctx); err != nil {
				log.Printf("[scenario-dependency-analyzer/aisearch] initial search reconcile failed (continuing degraded): %v", err)
			}
		}()
	}

	return svc
}

// applyTuningFromSearchJSON overrides each corpus's hardcoded tuning with the
// resolved tuning from its provider entry in search.json (the SSOT), in place. A
// missing file or provider leaves the hardcoded default (logged), never failing
// boot — search is an added capability. The embed recipe (model/task_prefix) is
// NOT overridden divergently: if a provider declares a different embed recipe
// than the first corpus, it is kept as-authored but the New() shared-embedder
// invariant still uses the first corpus's embedder, so authors must keep the
// recipe uniform (enforced by docs + the searchjson test).
func applyTuningFromSearchJSON(specs []corpusSpec, path string) {
	if strings.TrimSpace(path) == "" || len(specs) == 0 {
		return
	}
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		log.Printf("[scenario-dependency-analyzer/aisearch] load search.json tuning (%s): %v — using hardcoded defaults", path, err)
		return
	}
	for i := range specs {
		provider, ok := file.Provider(ProviderID(specs[i].id))
		if !ok {
			log.Printf("[scenario-dependency-analyzer/aisearch] provider %q not in search.json — using hardcoded default tuning", ProviderID(specs[i].id))
			continue
		}
		specs[i].tuning = provider.ResolvedTuning()
	}
}

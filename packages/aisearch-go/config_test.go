package aisearch

import (
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// No env set for this prefix → every field is the engine default.
	cfg := LoadConfig("CFG_TEST_UNSET")
	if cfg.SyncInterval != DefaultSyncInterval {
		t.Errorf("SyncInterval = %s, want %s", cfg.SyncInterval, DefaultSyncInterval)
	}
	if cfg.SyncDisabled {
		t.Error("SyncDisabled = true, want false")
	}
	if cfg.ReconcileParallelism != DefaultReconcileParallelism {
		t.Errorf("ReconcileParallelism = %d, want %d", cfg.ReconcileParallelism, DefaultReconcileParallelism)
	}
	if cfg.MaxEmbedsPerTick != 0 {
		t.Errorf("MaxEmbedsPerTick = %d, want 0 (unlimited)", cfg.MaxEmbedsPerTick)
	}
	if cfg.QdrantURL != DefaultQdrantURL {
		t.Errorf("QdrantURL = %q, want %q", cfg.QdrantURL, DefaultQdrantURL)
	}
	if cfg.EmbedModel != DefaultEmbedModel {
		t.Errorf("EmbedModel = %q, want %q", cfg.EmbedModel, DefaultEmbedModel)
	}
	if cfg.RerankShortlist != DefaultRerankShortlist {
		t.Errorf("RerankShortlist = %d, want %d", cfg.RerankShortlist, DefaultRerankShortlist)
	}
	// Floor overrides default to 0 ("unset") so FloorForMethodLeg supplies the regime
	// default rather than a hard-coded cosine value seeded by config.
	if cfg.RelevanceMaxGap != 0 || cfg.RelevanceHardFloor != 0 {
		t.Errorf("floor overrides = (%g, %g), want (0, 0) unset", cfg.RelevanceMaxGap, cfg.RelevanceHardFloor)
	}
}

func TestLoadConfigRerankShortlist(t *testing.T) {
	t.Run("parses set value", func(t *testing.T) {
		t.Setenv("SL_RERANK_SHORTLIST", "120")
		if got := LoadConfig("SL").RerankShortlist; got != 120 {
			t.Errorf("RerankShortlist = %d, want 120", got)
		}
	})
	t.Run("clamps above max", func(t *testing.T) {
		t.Setenv("SLMAX_RERANK_SHORTLIST", "99999")
		if got := LoadConfig("SLMAX").RerankShortlist; got != MaxRerankShortlist {
			t.Errorf("RerankShortlist = %d, want clamp to %d", got, MaxRerankShortlist)
		}
	})
	t.Run("clamps below min", func(t *testing.T) {
		t.Setenv("SLMIN_RERANK_SHORTLIST", "0")
		if got := LoadConfig("SLMIN").RerankShortlist; got != MinRerankShortlist {
			t.Errorf("RerankShortlist = %d, want clamp to %d", got, MinRerankShortlist)
		}
	})
	t.Run("malformed falls back to default", func(t *testing.T) {
		t.Setenv("SLBAD_RERANK_SHORTLIST", "not-a-number")
		if got := LoadConfig("SLBAD").RerankShortlist; got != DefaultRerankShortlist {
			t.Errorf("RerankShortlist = %d, want default %d", got, DefaultRerankShortlist)
		}
	})
}

func TestLoadConfigReadsPrefixedEnv(t *testing.T) {
	t.Setenv("DEMO_SYNC_INTERVAL", "90s")
	t.Setenv("DEMO_SYNC_DISABLED", "true")
	t.Setenv("DEMO_RECONCILE_PARALLELISM", "8")
	t.Setenv("DEMO_MAX_EMBEDS_PER_TICK", "250")
	t.Setenv("DEMO_QDRANT_URL", "http://example:6333")
	t.Setenv("DEMO_EMBED_MODEL", "custom-model")

	cfg := LoadConfig("DEMO")
	if cfg.SyncInterval != 90*time.Second {
		t.Errorf("SyncInterval = %s, want 90s", cfg.SyncInterval)
	}
	if !cfg.SyncDisabled {
		t.Error("SyncDisabled = false, want true")
	}
	if cfg.ReconcileParallelism != 8 {
		t.Errorf("ReconcileParallelism = %d, want 8", cfg.ReconcileParallelism)
	}
	if cfg.MaxEmbedsPerTick != 250 {
		t.Errorf("MaxEmbedsPerTick = %d, want 250", cfg.MaxEmbedsPerTick)
	}
	if cfg.QdrantURL != "http://example:6333" {
		t.Errorf("QdrantURL = %q", cfg.QdrantURL)
	}
	if cfg.EmbedModel != "custom-model" {
		t.Errorf("EmbedModel = %q", cfg.EmbedModel)
	}
}

func TestLoadConfigClampsAndFallsBack(t *testing.T) {
	// Parallelism above the max clamps; a malformed duration falls back.
	t.Setenv("CLMP_RECONCILE_PARALLELISM", "9999")
	t.Setenv("CLMP_SYNC_INTERVAL", "not-a-duration")
	cfg := LoadConfig("CLMP")
	if cfg.ReconcileParallelism != MaxReconcileParallelism {
		t.Errorf("ReconcileParallelism = %d, want clamp to %d", cfg.ReconcileParallelism, MaxReconcileParallelism)
	}
	if cfg.SyncInterval != DefaultSyncInterval {
		t.Errorf("SyncInterval = %s, want default after parse failure", cfg.SyncInterval)
	}
}

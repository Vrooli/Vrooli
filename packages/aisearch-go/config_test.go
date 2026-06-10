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
	// The search TUNING factors (rerank_*, floor band, embed_task_prefix) are no
	// longer env-read here — they are owned by `.vrooli/search.json` / TuningConfig
	// (config.go is the operational/wiring layer). Their coverage lives in
	// tuning_test.go (the factor taxonomy + NewServiceForTuning), not here.
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

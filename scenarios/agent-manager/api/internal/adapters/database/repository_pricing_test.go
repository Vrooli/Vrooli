package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func floatPtr(v float64) *float64 { return &v }

func TestModelPricingDomainRowRoundTripPreservesOptionalComponents(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	model := &pricing.ModelPricing{
		ID: uuid.New(), CanonicalModelName: "test-model", Provider: "provider", FetchedAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		InputTokenPrice: floatPtr(1), OutputTokenPrice: floatPtr(2), CacheReadPrice: floatPtr(3), CacheCreationPrice: floatPtr(4), WebSearchPrice: floatPtr(5), ServerToolUsePrice: floatPtr(6),
		InputTokenSource: pricing.SourceProviderAPI, OutputTokenSource: pricing.SourceManualOverride, CacheReadSource: pricing.SourceManualOverride, CacheCreationSource: pricing.SourceProviderAPI, WebSearchSource: pricing.SourceManualOverride, ServerToolUseSource: pricing.SourceProviderAPI, PricingVersion: "v1",
	}
	got := modelPricingFromDomain(model).toDomain()
	if got.ID != model.ID || got.InputTokenPrice == nil || *got.InputTokenPrice != 1 || got.ServerToolUsePrice == nil || *got.ServerToolUsePrice != 6 || got.PricingVersion != "v1" {
		t.Fatalf("pricing round trip = %+v", got)
	}
}

func TestPricingRepositoryPersistsAndQueriesModelPricing(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewPricingRepository(db, logrus.New())
	now := time.Now().UTC()
	model := &pricing.ModelPricing{ID: uuid.New(), CanonicalModelName: "model", Provider: "provider", InputTokenPrice: floatPtr(1.5), FetchedAt: now, ExpiresAt: now.Add(-time.Hour), CreatedAt: now, UpdatedAt: now}
	if err := repo.UpsertPricing(t.Context(), model); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetPricing(t.Context(), "model", "provider")
	if err != nil || got == nil || got.InputTokenPrice == nil || *got.InputTokenPrice != 1.5 {
		t.Fatalf("get=%+v err=%v", got, err)
	}
	expired, err := repo.GetExpiredPricing(t.Context(), now)
	if err != nil || len(expired) != 1 {
		t.Fatalf("expired=%+v err=%v", expired, err)
	}
	if err := repo.DeletePricing(t.Context(), "model", "provider"); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetPricing(t.Context(), "model", "provider"); err != nil || got != nil {
		t.Fatalf("deleted get=%+v err=%v", got, err)
	}
}

func TestPricingRepositoryPersistsQuotaObservationWithoutInventingAmounts(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewPricingRepository(db, logrus.New())
	when := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	observation, err := pricing.FromRateLimitEvent("openai", "codex", "codex-native-rate-limit", when, &domain.RateLimitEventData{LimitType: "5h", Message: "limit reached"})
	if err != nil {
		t.Fatal(err)
	}
	store := repo.(pricing.QuotaObservationRepository)
	if err := store.RecordQuotaObservation(t.Context(), observation); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordQuotaObservation(t.Context(), observation); err != nil {
		t.Fatalf("replaying the same observation: %v", err)
	}
	rows, err := store.ListQuotaObservations(t.Context(), "openai", "codex", "5h", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[0].Standing != pricing.QuotaStandingExhausted || rows[0].Freshness != pricing.ObservationUnknown || rows[0].Used != nil || rows[0].Limit != nil || rows[0].Remaining != nil {
		t.Fatalf("persisted observation invented quota amounts: %+v", rows[0])
	}
}

func TestPricingRepositoryBulkAliasesOverridesAndSettings(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repo := NewPricingRepository(db, logrus.New())
	ctx := t.Context()
	now := time.Now().UTC()
	models := []*pricing.ModelPricing{
		{CanonicalModelName: "alpha", Provider: "one", FetchedAt: now, ExpiresAt: now.Add(time.Hour)},
		{CanonicalModelName: "beta", Provider: "one", FetchedAt: now, ExpiresAt: now.Add(time.Hour)},
	}
	if err := repo.BulkUpsertPricing(ctx, models); err != nil {
		t.Fatal(err)
	}
	if all, err := repo.GetAllPricing(ctx); err != nil || len(all) != 2 {
		t.Fatalf("all=%+v err=%v", all, err)
	}
	if byProvider, err := repo.GetPricingByProvider(ctx, "one"); err != nil || len(byProvider) != 2 {
		t.Fatalf("byProvider=%+v err=%v", byProvider, err)
	}

	alias := &pricing.ModelAlias{RunnerModel: "runner", RunnerType: "codex", CanonicalModel: "alpha", Provider: "one"}
	if err := repo.UpsertAlias(ctx, alias); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetAlias(ctx, "runner", "codex"); err != nil || got == nil || got.CanonicalModel != "alpha" {
		t.Fatalf("alias=%+v err=%v", got, err)
	}
	if aliases, err := repo.ListAliases(ctx, "codex"); err != nil || len(aliases) != 1 {
		t.Fatalf("aliases=%+v err=%v", aliases, err)
	}
	if aliases, err := repo.GetAllAliases(ctx); err != nil || len(aliases) != 1 {
		t.Fatalf("all aliases=%+v err=%v", aliases, err)
	}
	if err := repo.DeleteAlias(ctx, "runner", "codex"); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetAlias(ctx, "runner", "codex"); err != nil || got != nil {
		t.Fatalf("deleted alias=%+v err=%v", got, err)
	}

	expires := now.Add(-time.Hour)
	override := &pricing.ManualPriceOverride{CanonicalModelName: "alpha", Component: pricing.ComponentInputTokens, PriceUSD: 1.25, Note: "operator", CreatedBy: "test", ExpiresAt: &expires}
	if err := repo.UpsertOverride(ctx, override); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetOverride(ctx, "alpha", pricing.ComponentInputTokens); err != nil || got == nil || got.PriceUSD != 1.25 {
		t.Fatalf("override=%+v err=%v", got, err)
	}
	if got, err := repo.GetOverridesForModel(ctx, "alpha"); err != nil || len(got) != 1 {
		t.Fatalf("model overrides=%+v err=%v", got, err)
	}
	if got, err := repo.GetAllOverrides(ctx); err != nil || len(got) != 1 {
		t.Fatalf("all overrides=%+v err=%v", got, err)
	}
	if removed, err := repo.CleanupExpiredOverrides(ctx); err != nil || removed != 1 {
		t.Fatalf("removed=%d err=%v", removed, err)
	}
	if err := repo.UpsertOverride(ctx, &pricing.ManualPriceOverride{CanonicalModelName: "alpha", Component: pricing.ComponentOutputTokens, PriceUSD: 2}); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteOverride(ctx, "alpha", pricing.ComponentOutputTokens); err != nil {
		t.Fatal(err)
	}

	defaults, err := repo.GetSettings(ctx)
	if err != nil || defaults == nil {
		t.Fatalf("defaults=%+v err=%v", defaults, err)
	}
	wantSettings := &pricing.PricingSettings{ProviderCacheTTL: 15 * time.Minute}
	if err := repo.UpdateSettings(ctx, wantSettings); err != nil {
		t.Fatal(err)
	}
	gotSettings, err := repo.GetSettings(ctx)
	if err != nil || gotSettings.ProviderCacheTTL != 15*time.Minute {
		t.Fatalf("settings=%+v err=%v", gotSettings, err)
	}
}

func TestPricingAliasAndOverrideRowsRoundTripOptionalFields(t *testing.T) {
	now := time.Now().UTC()
	alias := &pricing.ModelAlias{ID: uuid.New(), RunnerModel: "runner", RunnerType: "codex", CanonicalModel: "canonical", Provider: "provider", CreatedAt: now, UpdatedAt: now}
	if got := modelAliasFromDomain(alias).toDomain(); got.ID != alias.ID || got.CanonicalModel != alias.CanonicalModel {
		t.Fatalf("alias round trip = %+v", got)
	}
	expires := now.Add(time.Hour)
	override := &pricing.ManualPriceOverride{ID: uuid.New(), CanonicalModelName: "canonical", Component: pricing.ComponentInputTokens, PriceUSD: 2.5, Note: "operator", CreatedBy: "test", CreatedAt: now, ExpiresAt: &expires}
	if got := manualOverrideFromDomain(override).toDomain(); got.ID != override.ID || got.Note != "operator" || got.CreatedBy != "test" || got.ExpiresAt == nil {
		t.Fatalf("override round trip = %+v", got)
	}
}

func TestNewConnectionInitializesSchemaAndHealthCheck(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	db, err := NewConnection(logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.HealthCheck(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("SELECT 1 FROM pricing_settings WHERE id = 1"); err != nil {
		t.Fatalf("schema was not initialized: %v", err)
	}
}

func TestInitSchemaCreatesRunIndexesAndUpgradesLegacyColumnsIdempotently(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	db, err := NewConnection(logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	// An install that predates a column keeps its rows and gains the column.
	if _, err := db.ExecContext(ctx, "ALTER TABLE agent_profiles DROP COLUMN max_weighted_tokens"); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		if err := db.InitializeSchema(); err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
	}
	columns, err := db.tableColumns(ctx, "agent_profiles")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := columns["max_weighted_tokens"]; !ok {
		t.Fatal("max_weighted_tokens was not restored")
	}
	for _, index := range []string{"idx_runs_import_provenance", "idx_runs_status_created_at_id", "idx_runs_web_console_session", "idx_runs_identity_renewal_anchor"} {
		var count int
		if err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index); err != nil || count != 1 {
			t.Fatalf("index %s count=%d err=%v", index, count, err)
		}
	}
}

package manifest_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"architecture-cartographer/internal/clock"
	localdb "architecture-cartographer/internal/database"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(manifest.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return d
}

func sampleManifest() manifest.ManifestDefinition {
	return manifest.ManifestDefinition{
		Scenario:    "demo",
		Version:     manifest.ManifestVersionV1,
		ContentHash: "hash-1",
		Domains: []manifest.DomainSpec{
			{Name: "graph", Paths: []string{"api/internal/graph/**"}, Glossary: []string{"GraphSnapshot"}},
			{Name: "manifest", Paths: []string{"api/internal/manifest/**"}, AllowedDependencies: []string{"graph"}},
		},
		SharedSubstrate: []string{"api/internal/clock/**"},
		SignalWeights:   manifest.SignalWeights{Weights: map[string]float64{"path-token": 1.7}},
		Thresholds:      []manifest.Threshold{{Tier: "auto_place", MinValue: 0.85}},
		Transitional:    []manifest.TransitionalDeclaration{{ID: "tmp", Kind: "allow_cycle", Locator: "x->y", Rationale: "WIP", ExpiresWhen: "after:2026-09-01"}},
	}
}

func TestSQLiteRepository_SaveAndGet(t *testing.T) {
	d := newSchemaDB(t)
	repo := manifest.NewSQLiteRepository(d, clock.System{})

	saved, err := repo.SaveManifest(context.Background(), sampleManifest())
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.ParsedAt.IsZero() {
		t.Fatalf("ParsedAt should be set on save")
	}

	got, err := repo.GetManifest(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Scenario != "demo" {
		t.Fatalf("Scenario mismatch: %q", got.Scenario)
	}
	if len(got.Domains) != 2 || got.Domains[0].Name != "graph" {
		t.Fatalf("Domains mismatch: %+v", got.Domains)
	}
	if got.SignalWeights.Weights["path-token"] != 1.7 {
		t.Fatalf("SignalWeights overlay not persisted: %+v", got.SignalWeights)
	}
	if got.ContentHash != "hash-1" {
		t.Fatalf("ContentHash mismatch: %q", got.ContentHash)
	}
	if len(got.Transitional) != 1 || got.Transitional[0].Kind != "allow_cycle" {
		t.Fatalf("Transitional not persisted: %+v", got.Transitional)
	}
}

func TestSQLiteRepository_UpsertReplaces(t *testing.T) {
	d := newSchemaDB(t)
	repo := manifest.NewSQLiteRepository(d, clock.System{})

	first := sampleManifest()
	if _, err := repo.SaveManifest(context.Background(), first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	updated := first
	updated.ContentHash = "hash-2"
	updated.Domains = updated.Domains[:1]
	if _, err := repo.SaveManifest(context.Background(), updated); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	got, err := repo.GetManifest(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ContentHash != "hash-2" {
		t.Fatalf("upsert did not replace ContentHash: %q", got.ContentHash)
	}
	if len(got.Domains) != 1 {
		t.Fatalf("upsert did not replace Domains: %+v", got.Domains)
	}
}

func TestSQLiteRepository_NotFound(t *testing.T) {
	d := newSchemaDB(t)
	repo := manifest.NewSQLiteRepository(d, clock.System{})

	_, err := repo.GetManifest(context.Background(), "missing")
	var typed manifest.ErrManifestNotFound
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrManifestNotFound, got %v", err)
	}
}

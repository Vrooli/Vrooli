package manifest_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/manifest/mocks"
)

func validManifest() manifest.ManifestDefinition {
	return manifest.ManifestDefinition{
		Scenario: "demo",
		Domains: []manifest.DomainSpec{
			{Name: "graph", Paths: []string{"api/internal/graph/**"}},
			{Name: "manifest", Paths: []string{"api/internal/manifest/**"}, AllowedDependencies: []string{"graph"}},
		},
		Thresholds: []manifest.Threshold{{Tier: "auto_place", MinValue: 0.85}},
	}
}

func TestService_ValidateManifest_HappyPath(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := manifest.NewService(repo)
	m, diags, err := svc.ValidateManifest(context.Background(), validManifest())
	if err != nil {
		t.Fatalf("Validate: %v (diags=%v)", err, diags)
	}
	if m.Version != manifest.ManifestVersionV1 {
		t.Fatalf("expected version V1 defaulted, got %q", m.Version)
	}
	if repo.SaveCalls.Load() != 1 {
		t.Fatalf("Save should be called once, got %d", repo.SaveCalls.Load())
	}
}

func TestService_ValidateManifest_MissingScenarioErrors(t *testing.T) {
	svc := manifest.NewService(&mocks.FakeRepository{})
	m := validManifest()
	m.Scenario = ""
	_, _, err := svc.ValidateManifest(context.Background(), m)
	var typed manifest.ErrInvalidManifest
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrInvalidManifest, got %v", err)
	}
}

func TestService_ValidateManifest_UnknownDependencyErrors(t *testing.T) {
	svc := manifest.NewService(&mocks.FakeRepository{})
	m := validManifest()
	m.Domains[1].AllowedDependencies = []string{"nope"}
	_, _, err := svc.ValidateManifest(context.Background(), m)
	var typed manifest.ErrInvalidManifest
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrInvalidManifest, got %v", err)
	}
}

func TestService_ListDomains_PassesThrough(t *testing.T) {
	repo := &mocks.FakeRepository{ByScenario: map[string]manifest.ManifestDefinition{
		"demo": validManifest(),
	}}
	svc := manifest.NewService(repo)
	doms, err := svc.ListDomains(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(doms) != 2 {
		t.Fatalf("want 2 domains, got %d", len(doms))
	}
}

func TestService_GetManifest_NotFound(t *testing.T) {
	svc := manifest.NewService(&mocks.FakeRepository{})
	_, err := svc.GetManifest(context.Background(), "missing")
	var typed manifest.ErrManifestNotFound
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrManifestNotFound, got %v", err)
	}
}

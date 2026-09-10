package deployment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
)

type fakeClosureSource struct {
	closure domain.Closure
	err     error
	calls   []string
}

func (f *fakeClosureSource) Resolve(_ context.Context, scenarioID, environment string) (domain.Closure, error) {
	f.calls = append(f.calls, scenarioID+"@"+environment)
	if f.err != nil {
		return domain.Closure{}, f.err
	}
	return f.closure, nil
}

type fakePortsFetcher struct{ ports map[string]int }

func (f fakePortsFetcher) FetchPorts(context.Context, string) (map[string]int, error) {
	return f.ports, nil
}

func fixtureClosure() domain.Closure {
	return domain.Closure{
		SchemaVersion: domain.ClosureSchemaVersion,
		ScenarioID:    "app",
		Environment:   "production",
		Components: []domain.ClosureComponent{
			{ID: "app", Kind: domain.ClosureKindScenario, Required: true},
			{ID: "records-service", Kind: domain.ClosureKindScenario, Required: true},
			{ID: "store", Kind: domain.ClosureKindResource, Required: true},
			{ID: "cache", Kind: domain.ClosureKindResource, OptionalSelected: true},
			{ID: "curl", Kind: domain.ClosureKindTool, Required: true},
		},
		Sources: domain.ClosureSources{AnalyzerTool: "scenario-dependency-analyzer", AnalyzerUsed: true},
		Digest:  "sha256:fixture",
	}
}

// TestRefreshManifestFromClosureBindsDigest proves the manifest's dependency
// and bundle sections are the closure projection and carry the closure
// digest for plans to bind to. [REQ:STC-P0-021]
func TestRefreshManifestFromClosureBindsDigest(t *testing.T) {
	source := &fakeClosureSource{closure: fixtureClosure()}
	refresher := NewManifestRefresher(ManifestRefresherConfig{
		Closure:      source,
		PortsFetcher: fakePortsFetcher{ports: map[string]int{"api": 8080}},
	})
	base := domain.CloudManifest{
		Version:     "1",
		Environment: "staging",
		Scenario:    domain.ManifestScenario{ID: "app"},
		Dependencies: domain.ManifestDependencies{
			Scenarios:           []string{"app", "stale-dependency"},
			ProgramBindingPeers: []string{"peer"},
		},
		Bundle:  domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true, Scenarios: []string{"stale-dependency"}},
		Secrets: &domain.ManifestSecrets{},
	}
	refreshed, err := refresher.RefreshManifest(context.Background(), base)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if len(source.calls) != 1 || source.calls[0] != "app@staging" {
		t.Fatalf("closure must be resolved for the manifest environment: %v", source.calls)
	}
	if refreshed.Dependencies.ClosureDigest != "sha256:fixture" {
		t.Fatalf("closure_digest = %q", refreshed.Dependencies.ClosureDigest)
	}
	if strings.Join(refreshed.Dependencies.Scenarios, ",") != "app,records-service" || strings.Join(refreshed.Bundle.Scenarios, ",") != "app,records-service" {
		t.Fatalf("scenarios = %v / %v", refreshed.Dependencies.Scenarios, refreshed.Bundle.Scenarios)
	}
	if strings.Join(refreshed.Dependencies.Resources, ",") != "cache,store" || strings.Join(refreshed.Bundle.Resources, ",") != "cache,store" {
		t.Fatalf("resources = %v / %v", refreshed.Dependencies.Resources, refreshed.Bundle.Resources)
	}
	if refreshed.Dependencies.Analyzer.Tool != "scenario-dependency-analyzer" || refreshed.Dependencies.Analyzer.GeneratedAt == "" {
		t.Fatalf("analyzer identity: %+v", refreshed.Dependencies.Analyzer)
	}
	if len(refreshed.Dependencies.ProgramBindingPeers) != 1 || !refreshed.Bundle.IncludePackages || !refreshed.Bundle.IncludeAutoheal {
		t.Fatalf("manifest fields outside the closure projection must be preserved: %+v", refreshed)
	}
	if refreshed.Ports["api"] != 8080 || refreshed.Secrets != nil {
		t.Fatalf("ports must refresh and secrets must clear: %+v", refreshed)
	}
}

// TestRefreshManifestRefusesWhenClosureUnavailable proves a typed closure
// failure stops the refresh instead of rebuilding from stale dependencies.
// [REQ:STC-P0-021]
func TestRefreshManifestRefusesWhenClosureUnavailable(t *testing.T) {
	cause := &closure.Error{Code: closure.CodeUnavailable, Message: "analyzer down"}
	refresher := NewManifestRefresher(ManifestRefresherConfig{
		Closure:      &fakeClosureSource{err: cause},
		PortsFetcher: fakePortsFetcher{},
	})
	base := domain.CloudManifest{Scenario: domain.ManifestScenario{ID: "app"}, Dependencies: domain.ManifestDependencies{Scenarios: []string{"app"}}}
	refreshed, err := refresher.RefreshManifest(context.Background(), base)
	if !closure.Is(err, closure.CodeUnavailable) {
		t.Fatalf("expected typed closure_unavailable, got %v", err)
	}
	var typed *closure.Error
	if !errors.As(err, &typed) {
		t.Fatalf("typed error must be wrapped, got %T", err)
	}
	if refreshed.Dependencies.ClosureDigest != "" || strings.Join(refreshed.Dependencies.Scenarios, ",") != "app" {
		t.Fatalf("a refused refresh must return the base manifest unchanged: %+v", refreshed.Dependencies)
	}
}

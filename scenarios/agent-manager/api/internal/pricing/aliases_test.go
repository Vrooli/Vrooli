package pricing

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestResolveModelAliasNeverInfersProvider(t *testing.T) {
	if _, _, found := ResolveModelAlias("vendor/phoenix-9000"); found {
		t.Fatal("qualified future model was resolved without resource metadata")
	}
	if _, _, found := ResolveModelAlias("phoenix-9000"); found {
		t.Fatal("bare future model was resolved by Agent Manager heuristics")
	}
}

type fakeModelResolver struct{}

func (fakeModelResolver) Resolve(_ context.Context, runner, model string) (string, string, error) {
	if runner != "future-runner" || model != "phoenix-9000" {
		return "", "", fmt.Errorf("unexpected resolver request runner=%q model=%q", runner, model)
	}
	return "vendor/phoenix-9000", "future-pricing", nil
}

func TestPricingUsesResourceResolutionForFutureModelVocabulary(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithModelResolver(repo, nil, logrus.New(), fakeModelResolver{})
	canonical, provider, err := service.ResolveCanonicalModel(context.Background(), "phoenix-9000", "future-runner")
	if err != nil {
		t.Fatalf("ResolveCanonicalModel: %v", err)
	}
	if canonical != "vendor/phoenix-9000" || provider != "future-pricing" {
		t.Fatalf("resolution=%q/%q", canonical, provider)
	}
}

func TestCLIModelResolverUsesGenericResourceContract(t *testing.T) {
	resolver := CLIModelResolver{run: func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"schema_version":"v1","runner":"future-runner","model":"phoenix-9000","canonical_model":"vendor/phoenix-9000","provider":"future-pricing"}`), nil
	}}
	canonical, provider, err := resolver.Resolve(context.Background(), "future-runner", "phoenix-9000")
	if err != nil || canonical != "vendor/phoenix-9000" || provider != "future-pricing" {
		t.Fatalf("resolution=%q/%q err=%v", canonical, provider, err)
	}
}

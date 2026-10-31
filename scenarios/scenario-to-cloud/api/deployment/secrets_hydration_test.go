package deployment

import (
	"context"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/secrets"
)

func TestEnsureSecretsAvailableHydratesExplicitlyEmptyManifest(t *testing.T) {
	fetcher := &secretsFetcherForHydrationTest{
		response: &secrets.ManagerResponse{BundleSecrets: []domain.BundleSecretPlan{{
			ID:         "service-secret",
			Class:      domain.SecretClassPerInstallGenerated,
			Required:   true,
			Target:     domain.BundleSecretTarget{Type: "env", Name: "LPBS_SERVICE_SECRET"},
			Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/app", Field: "service-secret"},
		}}},
	}
	o := &Orchestrator{secretsFetcher: fetcher}
	manifest := domain.CloudManifest{
		Scenario: domain.ManifestScenario{ID: "app"},
		Secrets:  &domain.ManifestSecrets{},
	}

	if err := o.ensureSecretsAvailable(context.Background(), &manifest, nil, "dep-1", func(string, string, string) {}); err != nil {
		t.Fatalf("ensure secrets: %v", err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("secrets fetch calls = %d, want 1", fetcher.calls)
	}
	if manifest.Secrets == nil || len(manifest.Secrets.BundleSecrets) != 1 {
		t.Fatalf("manifest was not hydrated: %+v", manifest.Secrets)
	}
}

type secretsFetcherForHydrationTest struct {
	response *secrets.ManagerResponse
	calls    int
}

func (f *secretsFetcherForHydrationTest) FetchBundleSecrets(context.Context, string, string, []string) (*secrets.ManagerResponse, error) {
	f.calls++
	return f.response, nil
}

func (*secretsFetcherForHydrationTest) HealthCheck(context.Context) error { return nil }

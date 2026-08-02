package cleanup

import (
	"context"
	"testing"
)

type stubProvider struct {
	meta ProviderMetadata
}

func (p stubProvider) Metadata() ProviderMetadata { return p.meta }
func (p stubProvider) Estimate(context.Context, EstimateRequest) (Estimate, error) {
	return Estimate{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}, nil
}

func (p stubProvider) Preview(context.Context, PreviewRequest) (Preview, error) {
	return Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}, nil
}

func (p stubProvider) Apply(context.Context, ApplyRequest) (ApplyResult, error) {
	return ApplyResult{ProviderID: p.meta.ID}, nil
}

func (p stubProvider) Verify(context.Context, VerifyRequest) (VerifyResult, error) {
	return VerifyResult{Verified: true}, nil
}

func TestValidateProviderRequiresPreviewFirstMetadata(t *testing.T) {
	t.Parallel()

	err := ValidateProvider(stubProvider{meta: ProviderMetadata{
		ID:                  "tmp",
		Name:                "Temporary files",
		Version:             "v1",
		OwnerScenario:       "storage-manager",
		SafetyTier:          SafetyTierSafe,
		DefaultMode:         ProviderModeDisabled,
		DefaultApproval:     ApprovalModeOperator,
		SupportedPlatforms:  []string{"linux"},
		IrreversibleEffects: []string{"removes previewed files"},
		TestSubstitute:      "fake-filesystem",
	}})
	if err != nil {
		t.Fatalf("ValidateProvider() unexpected error: %v", err)
	}
}

func TestValidateProviderRejectsConditionalWithoutOperatorGate(t *testing.T) {
	t.Parallel()

	err := ValidateProvider(stubProvider{meta: ProviderMetadata{
		ID:                  "docker-volumes",
		Name:                "Docker volumes",
		Version:             "v1",
		OwnerScenario:       "storage-manager",
		SafetyTier:          SafetyTierConditional,
		DefaultMode:         ProviderModeEnabled,
		DefaultApproval:     ApprovalModeNone,
		SupportedPlatforms:  []string{"linux"},
		IrreversibleEffects: []string{"prunes docker data"},
		TestSubstitute:      "fake-docker",
	}})
	if err == nil {
		t.Fatal("ValidateProvider() expected conditional provider gate error")
	}
}

func TestValidateProviderRejectsForbiddenEnabledProvider(t *testing.T) {
	t.Parallel()

	err := ValidateProvider(stubProvider{meta: ProviderMetadata{
		ID:                  "live-database",
		Name:                "Live database",
		Version:             "v1",
		OwnerScenario:       "database",
		SafetyTier:          SafetyTierForbidden,
		DefaultMode:         ProviderModeEnabled,
		DefaultApproval:     ApprovalModeOperator,
		SupportedPlatforms:  []string{"linux"},
		IrreversibleEffects: []string{"deletes live database"},
		TestSubstitute:      "fake-owner-provider",
	}})
	if err == nil {
		t.Fatal("ValidateProvider() expected forbidden provider error")
	}
}

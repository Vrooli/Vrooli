package settings

import (
	"context"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/store"
)

// seam: ProviderConfigRepository is the settings provider-config seam
// (SEAMS.md row "store.ProviderConfigRepository"). Production wires
// *store.ProviderConfigStore; tests wire handlers/settings/mocks.
type ProviderConfigRepository interface {
	Get(ctx context.Context) (store.ProviderConfig, error)
	Update(ctx context.Context, p store.ProviderConfigPatch) (store.ProviderConfig, error)
}

// seam: BYOKRepository is the settings BYOK persistence seam (SEAMS.md
// row "store.BYOKRepository"). Production wires *byokstore.Store; tests
// wire handlers/settings/mocks.
type BYOKRepository interface {
	List(ctx context.Context) ([]byokstore.Credential, error)
	Upsert(ctx context.Context, providerID, capability, secret string) (byokstore.Credential, error)
	Delete(ctx context.Context, providerID, capability string) (bool, error)
}

// seam: VoiceOverridesRepository is the settings voice-overrides
// persistence seam.
type VoiceOverridesRepository interface {
	List(ctx context.Context) ([]store.VoiceOverride, error)
	Set(ctx context.Context, v store.VoiceOverride) error
}

// Compile-time guarantees.
var (
	_ ProviderConfigRepository = (*store.ProviderConfigStore)(nil)
	_ BYOKRepository           = (*byokstore.Store)(nil)
	_ VoiceOverridesRepository = (*store.VoiceOverrideStore)(nil)
)

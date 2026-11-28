package capabilitycatalog

import (
	"testing"

	"ai-gateway/internal/gatewaycatalog"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
	"github.com/stretchr/testify/require"
)

func TestBuildProjectsNativeCatalogsAsDeterministicInventory(t *testing.T) {
	profiles := &gatewaycatalog.ProfileCatalog{SchemaVersion: 1, Profiles: []gatewaycatalog.Profile{
		{Proto: "PROFILE_REMOTE_ONLY", Key: "remote-only", LabelKey: "profiles.remoteOnly", DescriptionKey: "profiles.remoteOnlyDesc", Order: 2},
		{Proto: "PROFILE_LOCAL_ONLY", Key: "local-only", LabelKey: "profiles.localOnly", DescriptionKey: "profiles.localOnlyDesc", Order: 1},
	}}
	roles := rolecatalog.RoleCatalog{
		Roles: map[string]rolecatalog.InferenceRole{
			"chat.default": {Description: "Typed chat", Candidates: []rolecatalog.Candidate{
				{Provider: "zeta", ResourceRole: "chat.default", Reason: "later declared"},
				{Provider: "alpha", ResourceRole: "chat.default", Reason: "earlier alphabetically"},
			}},
		},
		CodeRoles: map[string]rolecatalog.CodeRoleContract{
			"policy.check": {HandlerID: "policy-check/v1"},
		},
	}
	specs := []providers.ProviderSpec{
		{ID: "zeta", Family: providers.FamilyOpenAICompatible, Locality: "remote", Text: true, SpendCapable: true},
		{ID: "alpha", Family: providers.FamilyOllama, Locality: "local", Text: true},
	}

	got := Build(profiles, roles, specs)
	require.Equal(t, ContractVersion, got.ContractVersion)
	require.Equal(t, Scenario, got.Scenario)
	require.Equal(t, []string{"PROFILE_LOCAL_ONLY", "PROFILE_REMOTE_ONLY"}, []string{got.Profiles[0].ProtoName, got.Profiles[1].ProtoName})
	require.Equal(t, "local-only", got.Profiles[0].Key)
	require.Equal(t, "profiles.localOnly", got.Profiles[0].LabelKey)
	require.Equal(t, "profiles.localOnlyDesc", got.Profiles[0].DescriptionKey)
	require.NotEmpty(t, got.RequestKinds)
	require.Equal(t, "text", got.RequestKinds[0].RequiredInputModalities[0])
	require.Equal(t, []string{"text"}, got.RequestKinds[0].RequiredInputModalities)
	require.Equal(t, "text", got.RequestKinds[0].OutputModality)
	require.Equal(t, "vector", got.RequestKinds[1].OutputModality)
	require.Equal(t, "json", got.RequestKinds[2].OutputKind)
	require.Empty(t, got.RequestKinds[2].OutputModality)
	require.NotEmpty(t, got.RequestKinds[0].RequiredProviderCapabilities)

	require.Equal(t, "chat.default", got.Roles[0].Id)
	require.Equal(t, "Typed chat", got.Roles[0].Description)
	require.Equal(t, []string{"alpha", "zeta"}, []string{got.Roles[0].Candidates[0].Provider, got.Roles[0].Candidates[1].Provider})
	require.Equal(t, "chat.default", got.Roles[0].Candidates[0].ResourceRole)
	require.Equal(t, "earlier alphabetically", got.Roles[0].Candidates[0].Reason)
	require.Equal(t, "policy.check", got.Roles[1].Id)
	require.Equal(t, "code", got.Roles[1].Kind)
	require.Equal(t, "policy-check/v1", got.Roles[1].HandlerId)
	require.Equal(t, "alpha", got.Providers[0].Id)
	require.Equal(t, []string{providers.FamilyOllama}, got.Providers[0].SupportedFamilies)
	require.Equal(t, "local", got.Providers[0].Locality)
	require.True(t, got.Providers[0].SupportsText)
	require.Equal(t, "zeta", got.Providers[1].Id)
	require.Equal(t, "remote", got.Providers[1].Locality)
	require.True(t, got.Providers[1].SupportsText)
	require.True(t, got.Providers[1].SpendCapable)

	// Rebuilding the projection is stable; dispatch declaration order is not
	// exposed as candidate ordering.
	require.Equal(t, got, Build(profiles, roles, specs))
}

func TestBuildProjectsNilAndEmptyInventories(t *testing.T) {
	got := Build(nil, rolecatalog.RoleCatalog{}, nil)
	require.Equal(t, ContractVersion, got.ContractVersion)
	require.Equal(t, Scenario, got.Scenario)
	require.Empty(t, got.Profiles)
	require.NotEmpty(t, got.RequestKinds)
	require.Empty(t, got.Roles)
	require.Empty(t, got.Providers)
}

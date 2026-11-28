// Package capabilitycatalog projects AI Gateway's native catalogs onto the
// shared, read-only capability discovery contract.
package capabilitycatalog

import (
	"sort"
	"strings"

	"ai-gateway/internal/gatewaycatalog"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/requestvocab"
	"ai-gateway/internal/rolecatalog"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const (
	ContractVersion = "1"
	Scenario        = "ai-gateway"
)

// Build returns a deterministic inventory projection. The projection is for
// discovery only and is never consulted by authorization or dispatch.
func Build(profiles *gatewaycatalog.ProfileCatalog, roles rolecatalog.RoleCatalog, specs []providers.ProviderSpec) *commonv1.AIGatewayCapabilityCatalog {
	out := &commonv1.AIGatewayCapabilityCatalog{ContractVersion: ContractVersion, Scenario: Scenario}
	out.Profiles = projectProfiles(profiles)
	out.RequestKinds = projectRequestKinds()
	out.Roles = projectRoles(roles)
	out.Providers = projectProviders(specs)
	return out
}

func projectProfiles(profiles *gatewaycatalog.ProfileCatalog) []*commonv1.AIGatewayProfile {
	if profiles == nil {
		return nil
	}
	projected := make([]*commonv1.AIGatewayProfile, 0, len(profiles.Profiles))
	for _, profile := range profiles.Sorted() {
		projected = append(projected, &commonv1.AIGatewayProfile{
			ProtoName: profile.Proto, Key: profile.Key, LabelKey: profile.LabelKey, DescriptionKey: profile.DescriptionKey,
		})
	}
	return projected
}

func projectRequestKinds() []*commonv1.AIGatewayRequestKind {
	definitions := requestvocab.RequestKindCatalog()
	projected := make([]*commonv1.AIGatewayRequestKind, 0, len(definitions))
	for _, definition := range definitions {
		projected = append(projected, projectRequestKind(definition))
	}
	return projected
}

func projectRequestKind(definition requestvocab.RequestKindDefinition) *commonv1.AIGatewayRequestKind {
	kind := &commonv1.AIGatewayRequestKind{
		ProtoName: definition.Kind.String(), OutputKind: definition.OutputKind,
		RequiredProviderCapabilities: append([]string(nil), definition.RequiredCapabilities...),
	}
	for _, input := range definition.RequiredInputs {
		kind.RequiredInputModalities = append(kind.RequiredInputModalities, modalityName(input.String()))
	}
	if isModality(definition.OutputKind) {
		kind.OutputModality = definition.OutputKind
	}
	return kind
}

func projectRoles(roles rolecatalog.RoleCatalog) []*commonv1.AIGatewayRole {
	names := roleNames(roles)
	projected := make([]*commonv1.AIGatewayRole, 0, len(names))
	for _, name := range names {
		projected = append(projected, projectRole(name, roles))
	}
	return projected
}

func roleNames(roles rolecatalog.RoleCatalog) []string {
	names := make([]string, 0, len(roles.Roles)+len(roles.CodeRoles))
	for name := range roles.Roles {
		names = append(names, name)
	}
	for name := range roles.CodeRoles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func projectRole(name string, roles rolecatalog.RoleCatalog) *commonv1.AIGatewayRole {
	if definition, ok := roles.Roles[name]; ok {
		return projectInferenceRole(name, definition)
	}
	contract := roles.CodeRoles[name]
	return &commonv1.AIGatewayRole{Id: name, Kind: "code", HandlerId: contract.HandlerID}
}

func projectInferenceRole(name string, definition rolecatalog.InferenceRole) *commonv1.AIGatewayRole {
	kind, _ := requestvocab.NormalizeRoleKind(definition.Kind)
	role := &commonv1.AIGatewayRole{Id: name, Description: definition.Description, Kind: kind}
	candidates := sortedCandidates(definition.Candidates)
	for _, candidate := range candidates {
		role.Candidates = append(role.Candidates, &commonv1.AIGatewayRoleCandidate{
			Provider: candidate.Provider, ResourceRole: candidate.ResourceRole, Reason: candidate.Reason,
		})
	}
	return role
}

func sortedCandidates(candidates []rolecatalog.Candidate) []rolecatalog.Candidate {
	sorted := append([]rolecatalog.Candidate(nil), candidates...)
	// Sort candidates as inventory; declaration order is routing policy and
	// must not become an implicit consumer dispatch sequence.
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Provider != sorted[j].Provider {
			return sorted[i].Provider < sorted[j].Provider
		}
		return sorted[i].ResourceRole < sorted[j].ResourceRole
	})
	return sorted
}

func projectProviders(specs []providers.ProviderSpec) []*commonv1.AIGatewayProvider {
	projected := make([]*commonv1.AIGatewayProvider, 0, len(specs))
	for _, spec := range specs {
		projected = append(projected, &commonv1.AIGatewayProvider{
			Id: spec.ID, SupportedFamilies: []string{spec.Family}, Locality: spec.Locality,
			SupportsText: spec.Text, SpendCapable: spec.SpendCapable,
		})
	}
	sort.Slice(projected, func(i, j int) bool { return projected[i].Id < projected[j].Id })
	return projected
}

func isModality(kind string) bool {
	switch kind {
	case "text", "vector", "image", "video", "audio":
		return true
	default:
		return false
	}
}

func modalityName(protoName string) string {
	return strings.ToLower(strings.TrimPrefix(protoName, "MODALITY_"))
}

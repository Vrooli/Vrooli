package routing

import (
	"context"
	"sort"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/requestvocab"
	profilepolicy "github.com/vrooli/ai-provider-go/routing"
)

func findRole(roles []providers.Role, role string) (providers.Role, bool) {
	for _, candidate := range roles {
		if candidate.Role == role {
			return candidate, true
		}
	}
	return providers.Role{}, false
}

func roleSupportsRequest(role providers.Role, req *sharedv1.GatewayRequest) bool {
	if !roleSupportsKind(role, req.GetKind()) {
		return false
	}
	declared := map[sharedv1.Modality]struct{}{}
	for _, modality := range role.InputModalities {
		declared[modality] = struct{}{}
	}
	// Older fixtures omitted modalities. Treat them as text-only so existing
	// text callers remain compatible while image requests fail closed.
	if len(declared) == 0 {
		declared[sharedv1.Modality_MODALITY_TEXT] = struct{}{}
	}
	for _, required := range requiredInputModalities(req) {
		if _, ok := declared[required]; !ok {
			return false
		}
	}
	return true
}

func requiredInputModalities(req *sharedv1.GatewayRequest) []sharedv1.Modality {
	seen := map[sharedv1.Modality]struct{}{}
	add := func(modality sharedv1.Modality) {
		if modality != sharedv1.Modality_MODALITY_UNSPECIFIED {
			seen[modality] = struct{}{}
		}
	}
	if definition, ok := requestvocab.LookupRequestKind(req.GetKind()); ok {
		for _, modality := range definition.RequiredInputs {
			add(modality)
		}
	}
	for _, attachment := range req.GetAttachments() {
		if attachment != nil {
			add(attachment.GetModality())
		}
	}
	out := make([]sharedv1.Modality, 0, len(seen))
	for modality := range seen {
		out = append(out, modality)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func roleSupportsKind(role providers.Role, kind sharedv1.RequestKind) bool {
	definition, ok := requestvocab.LookupRequestKind(kind)
	if !ok {
		return false
	}
	caps := map[string]struct{}{}
	for _, cap := range role.Capabilities {
		caps[strings.ToLower(strings.TrimSpace(cap))] = struct{}{}
	}
	for _, capability := range definition.RequiredCapabilities {
		if _, ok := caps[capability]; ok {
			return true
		}
	}
	return false
}

func localityRejection(req *sharedv1.GatewayRequest, locality string, allowConsent bool) string {
	privacyClass := req.GetPrivacyClass().String()
	if allowConsent && req.GetPrivacyClass() == sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL && strings.EqualFold(locality, "remote") {
		privacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL.String()
	}
	_, reason := profilepolicy.Admits(req.GetProfile().String(), privacyClass, locality)
	return reason
}

func (s *Service) remoteConsentAllowed(ctx context.Context, req *sharedv1.GatewayRequest, provider, locality string) bool {
	if req.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || !strings.EqualFold(locality, "remote") || s.consent == nil {
		return false
	}
	source := strings.ToLower(strings.TrimSpace(providers.Source(ctx)))
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch source {
	case "own-key":
		if provider == providers.ProviderMetered || provider == providers.ProviderOllama || provider == providers.ProviderOllaya {
			return false
		}
	case "vrooli-credits":
		if provider != providers.ProviderMetered {
			return false
		}
	default:
		return false
	}
	return s.consent.Verify(ctx, providers.PrivacyConsent(ctx), source) == nil
}

func (s *Service) admitsRoute(ctx context.Context, req *sharedv1.GatewayRequest, provider, locality string) bool {
	if _, reason := profilepolicy.Admits(req.GetProfile().String(), req.GetPrivacyClass().String(), locality); reason == "" {
		return true
	}
	if !s.remoteConsentAllowed(ctx, req, provider, locality) || req.GetProfile() == sharedv1.Profile_PROFILE_LOCAL_ONLY || req.GetProfile() == sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE {
		return false
	}
	allowed, _ := profilepolicy.Admits(req.GetProfile().String(), sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL.String(), locality)
	return allowed
}

func rank(profile sharedv1.Profile, locality, provider string) int {
	return profilepolicy.Rank(profile.String(), locality, provider)
}

func profileReason(profile sharedv1.Profile) string {
	switch profile {
	case sharedv1.Profile_PROFILE_LOCAL_ONLY:
		return "local-only profile admits only local providers"
	case sharedv1.Profile_PROFILE_LOCAL_FIRST:
		return "local-first profile ranks local providers before remote providers"
	case sharedv1.Profile_PROFILE_REMOTE_ONLY:
		return "remote-only profile admits only remote providers"
	case sharedv1.Profile_PROFILE_QUALITY_FIRST:
		return "quality-first profile ranks remote providers before local fallbacks"
	case sharedv1.Profile_PROFILE_CHEAP_FIRST:
		return "cheap-first profile ranks local providers before remote providers"
	case sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE:
		return "privacy-sensitive profile ranks local providers and blocks remote routing for confidential data"
	default:
		return "unspecified profile"
	}
}

func profileEnum(v int32) sharedv1.Profile {
	return sharedv1.Profile(v)
}

func privacyEnum(v int32) sharedv1.PrivacyClass {
	return sharedv1.PrivacyClass(v)
}

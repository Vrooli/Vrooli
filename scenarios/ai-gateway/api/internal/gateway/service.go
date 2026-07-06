package gateway

import (
	"sort"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

var forbiddenMetadataKeys = map[string]string{
	"api_key":              "Provider credentials are resource-owned and must not enter AI Gateway requests.",
	"base_url":             "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"context_window":       "Concrete context windows are provider catalog details, not caller authority.",
	"embedding_dimensions": "Embedding dimensions are resolved through provider role policy.",
	"model":                "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
	"model_slug":           "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
	"openrouter_api_key":   "Provider credentials are resource-owned and must not enter AI Gateway requests.",
	"ollama_url":           "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"provider":             "Provider selection belongs to routing policy, not callers.",
	"provider_base_url":    "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"provider_model":       "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
	"provider_url":         "Provider base URLs are resource-owned and must not enter AI Gateway requests.",
	"remote_model":         "Concrete model slugs are resource-owned and must not enter AI Gateway requests.",
}

// Service owns provider-neutral request validation. It deliberately has no
// provider adapter dependency; provider execution starts in later phases after
// this boundary accepts the request shape.
type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) Validate(req *sharedv1.GatewayRequest) []*sharedv1.ValidationIssue {
	if req == nil {
		return []*sharedv1.ValidationIssue{issue("request", "required", "request is required")}
	}
	var issues []*sharedv1.ValidationIssue
	if req.GetKind() == sharedv1.RequestKind_REQUEST_KIND_UNSPECIFIED {
		issues = append(issues, issue("kind", "required", "request kind is required"))
	}
	if strings.TrimSpace(req.GetRole()) == "" {
		issues = append(issues, issue("role", "required", "role is required"))
	}
	if req.GetProfile() == sharedv1.Profile_PROFILE_UNSPECIFIED {
		issues = append(issues, issue("profile", "required", "routing profile is required"))
	}
	if req.GetPrivacyClass() == sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		issues = append(issues, issue("privacy_class", "required", "privacy class is required"))
	}
	if req.GetTimeoutMs() < 0 {
		issues = append(issues, issue("timeout_ms", "invalid", "timeout_ms cannot be negative"))
	}
	if req.GetMaxCostUsd() < 0 {
		issues = append(issues, issue("max_cost_usd", "invalid", "max_cost_usd cannot be negative"))
	}
	if req.GetMaxOutputTokens() < 0 {
		issues = append(issues, issue("max_output_tokens", "invalid", "max_output_tokens cannot be negative"))
	}
	issues = append(issues, validateProviderSpecificMetadata(req.GetMetadata())...)
	issues = append(issues, validatePrivacyProfile(req)...)
	return issues
}

func AcceptedProfiles() []string {
	return []string{
		"local-only",
		"local-first",
		"remote-only",
		"quality-first",
		"cheap-first",
		"privacy-sensitive",
	}
}

func validateProviderSpecificMetadata(metadata map[string]string) []*sharedv1.ValidationIssue {
	if len(metadata) == 0 {
		return nil
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var issues []*sharedv1.ValidationIssue
	for _, rawKey := range keys {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		key = strings.ReplaceAll(key, "-", "_")
		if msg, ok := forbiddenMetadataKeys[key]; ok {
			issues = append(issues, issue("metadata."+rawKey, "provider_specific_field", msg))
		}
	}
	return issues
}

func validatePrivacyProfile(req *sharedv1.GatewayRequest) []*sharedv1.ValidationIssue {
	privacy := req.GetPrivacyClass()
	profile := req.GetProfile()
	if privacy != sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET {
		return nil
	}
	switch profile {
	case sharedv1.Profile_PROFILE_REMOTE_ONLY,
		sharedv1.Profile_PROFILE_QUALITY_FIRST,
		sharedv1.Profile_PROFILE_CHEAP_FIRST:
		return []*sharedv1.ValidationIssue{
			issue("profile", "privacy_profile_conflict", "secret requests cannot use profiles that may require remote providers"),
		}
	default:
		return nil
	}
}

func issue(field, code, message string) *sharedv1.ValidationIssue {
	return &sharedv1.ValidationIssue{Field: field, Code: code, Message: message}
}

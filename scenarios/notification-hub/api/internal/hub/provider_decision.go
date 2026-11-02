package hub

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	ProviderDecisionScopeRule      = "provider_rule"
	ProviderDecisionScopeAgreement = "provider_agreement"
	AnswerIdentityKindUnknown      = "unknown"
	AnswerIdentityKindHuman        = "verified_human"
	AnswerIdentitySourceAccess     = "cloudflare_access"
	AnswerIdentitySourceLocal      = "personal_local"
	providerDecisionCorrelationKey = "notification_hub.provider_decision_context.v1"
)

var providerTermsSHA256 = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ProviderDecisionContext is the immutable binding between a Hub Ask and the
// canonical terms being considered. It carries no approval by itself.
type ProviderDecisionContext struct {
	SchemaVersion        uint32 `json:"schema_version"`
	CanonicalTermsSHA256 string `json:"canonical_terms_sha256"`
	Scope                string `json:"scope"`
}

func normalizeProviderDecisionContext(context *ProviderDecisionContext) (*ProviderDecisionContext, error) {
	if context == nil {
		return nil, nil
	}
	normalized := *context
	normalized.CanonicalTermsSHA256 = strings.TrimSpace(normalized.CanonicalTermsSHA256)
	normalized.Scope = strings.TrimSpace(normalized.Scope)
	if normalized.SchemaVersion != 1 {
		return nil, fmt.Errorf("%w: provider decision context schema version must be 1", ErrInvalidArgument)
	}
	if !providerTermsSHA256.MatchString(normalized.CanonicalTermsSHA256) {
		return nil, fmt.Errorf("%w: provider decision context requires a lowercase SHA-256 digest", ErrInvalidArgument)
	}
	if normalized.Scope != ProviderDecisionScopeRule && normalized.Scope != ProviderDecisionScopeAgreement {
		return nil, fmt.Errorf("%w: provider decision context scope is unsupported", ErrInvalidArgument)
	}
	return &normalized, nil
}

func encodeProviderDecisionContext(context *ProviderDecisionContext) (string, error) {
	normalized, err := normalizeProviderDecisionContext(context)
	if err != nil || normalized == nil {
		return "", err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode provider decision context: %w", err)
	}
	return string(encoded), nil
}

func decodeProviderDecisionContext(encoded string) *ProviderDecisionContext {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var context ProviderDecisionContext
	if err := json.Unmarshal([]byte(encoded), &context); err != nil {
		return nil
	}
	normalized, err := normalizeProviderDecisionContext(&context)
	if err != nil {
		return nil
	}
	return normalized
}

func sameProviderDecisionContext(a, b *ProviderDecisionContext) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

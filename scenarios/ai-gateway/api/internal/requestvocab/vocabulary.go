package requestvocab

import (
	"fmt"
	"sort"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// Role kinds are gateway execution contracts. The empty value in existing
// role documents is the canonical typed-value kind for backwards-compatible
// JSON; decision roles use the explicit "decision" value.
const (
	RoleKindTypedValue = "typed_value"
	RoleKindDecision   = "decision"
)

// RequestKindDefinition is the gateway's canonical interpretation of a proto
// request kind. Capability aliases are resource-policy vocabulary: a resource
// may expose any one of them to serve the request.
type RequestKindDefinition struct {
	// RequiredInputs names the provider-neutral modalities a caller must supply.
	RequiredInputs []sharedv1.Modality
	// OutputKind describes the result shape; JSON is a typed contract rather
	// than an input/output Modality in the shared wire vocabulary.
	OutputKind           string
	Kind                 sharedv1.RequestKind
	RequiredCapabilities []string
}

var requestKinds = map[sharedv1.RequestKind]RequestKindDefinition{
	sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION:       {Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "text", RequiredCapabilities: []string{"generate", "chat", "completion"}},
	sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING:        {Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "vector", RequiredCapabilities: []string{"embedding", "embeddings"}},
	sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION: {Kind: sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "json", RequiredCapabilities: []string{"extract", "structured", "json", "tools", "generate", "chat"}},
	sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION:      {Kind: sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "image", RequiredCapabilities: []string{"image_generation", "image_output"}},
	sharedv1.RequestKind_REQUEST_KIND_VIDEO_GENERATION:      {Kind: sharedv1.RequestKind_REQUEST_KIND_VIDEO_GENERATION, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "video", RequiredCapabilities: []string{"video_generation", "video_output"}},
	sharedv1.RequestKind_REQUEST_KIND_DETECT_PII:            {Kind: sharedv1.RequestKind_REQUEST_KIND_DETECT_PII, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "json", RequiredCapabilities: []string{"detect_pii", "detect", "pii"}},
	sharedv1.RequestKind_REQUEST_KIND_DECISION:              {Kind: sharedv1.RequestKind_REQUEST_KIND_DECISION, RequiredInputs: []sharedv1.Modality{sharedv1.Modality_MODALITY_TEXT}, OutputKind: "json", RequiredCapabilities: []string{"decide"}},
}

// RequestKindCatalog returns request-kind definitions in proto numeric order.
func RequestKindCatalog() []RequestKindDefinition {
	out := make([]RequestKindDefinition, 0, len(requestKinds))
	values := make([]int, 0, len(sharedv1.RequestKind_name))
	for value := range sharedv1.RequestKind_name {
		values = append(values, int(value))
	}
	sort.Ints(values)
	for _, value := range values {
		kind := sharedv1.RequestKind(value)
		if definition, ok := requestKinds[kind]; ok {
			definition.RequiredInputs = append([]sharedv1.Modality(nil), definition.RequiredInputs...)
			definition.RequiredCapabilities = append([]string(nil), definition.RequiredCapabilities...)
			out = append(out, definition)
		}
	}
	return out
}

// ValidateRequestKinds proves the runtime catalog covers every selectable
// proto value exactly once and gives each kind an unambiguous capability set.
func ValidateRequestKinds() error {
	seen := map[sharedv1.RequestKind]struct{}{}
	for _, definition := range RequestKindCatalog() {
		if err := validateRequestKindDefinition(definition); err != nil {
			return err
		}
		seen[definition.Kind] = struct{}{}
	}
	return validateRequestKindCoverage(seen)
}

func validateRequestKindDefinition(definition RequestKindDefinition) error {
	if definition.Kind == sharedv1.RequestKind_REQUEST_KIND_UNSPECIFIED || len(definition.RequiredInputs) == 0 || definition.OutputKind == "" || len(definition.RequiredCapabilities) == 0 {
		return fmt.Errorf("request kind %s must declare required inputs, an output kind, and capabilities", definition.Kind)
	}
	for _, input := range definition.RequiredInputs {
		if input == sharedv1.Modality_MODALITY_UNSPECIFIED {
			return fmt.Errorf("request kind %s has an unspecified required input", definition.Kind)
		}
	}
	return validateRequestKindCapabilities(definition)
}

func validateRequestKindCapabilities(definition RequestKindDefinition) error {
	aliases := map[string]struct{}{}
	for _, alias := range definition.RequiredCapabilities {
		alias = strings.ToLower(strings.TrimSpace(alias))
		if alias == "" {
			return fmt.Errorf("request kind %s has an empty resource capability", definition.Kind)
		}
		if _, duplicate := aliases[alias]; duplicate {
			return fmt.Errorf("request kind %s repeats resource capability %q", definition.Kind, alias)
		}
		aliases[alias] = struct{}{}
	}
	return nil
}

func validateRequestKindCoverage(seen map[sharedv1.RequestKind]struct{}) error {
	for value := range sharedv1.RequestKind_name {
		kind := sharedv1.RequestKind(value)
		if kind == sharedv1.RequestKind_REQUEST_KIND_UNSPECIFIED {
			continue
		}
		if _, ok := seen[kind]; !ok {
			return fmt.Errorf("proto request kind %s has no gateway catalog entry", kind)
		}
	}
	return nil
}

// LookupRequestKind resolves an executable request kind and its capability
// aliases. Unspecified and unknown wire values fail closed.
func LookupRequestKind(kind sharedv1.RequestKind) (RequestKindDefinition, bool) {
	definition, ok := requestKinds[kind]
	if !ok {
		return RequestKindDefinition{}, false
	}
	definition.RequiredInputs = append([]sharedv1.Modality(nil), definition.RequiredInputs...)
	definition.RequiredCapabilities = append([]string(nil), definition.RequiredCapabilities...)
	return definition, true
}

// NormalizeRoleKind makes the implicit JSON default explicit for validation
// and lookup while preserving existing empty-string catalog data.
func NormalizeRoleKind(kind string) (string, error) {
	switch strings.TrimSpace(kind) {
	case "", RoleKindTypedValue:
		return RoleKindTypedValue, nil
	case RoleKindDecision:
		return RoleKindDecision, nil
	default:
		return "", fmt.Errorf("unsupported role kind %q", kind)
	}
}

// KnownResourceCapability reports whether a capability participates in the
// request-kind routing contract. Resource policies may declare additional
// resource-specific capabilities; those remain resource-owned.
func KnownResourceCapability(capability string) bool {
	capability = strings.ToLower(strings.TrimSpace(capability))
	for _, definition := range requestKinds {
		for _, accepted := range definition.RequiredCapabilities {
			if capability == accepted {
				return true
			}
		}
	}
	return false
}

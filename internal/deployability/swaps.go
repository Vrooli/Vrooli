package deployability

import (
	"encoding/json"
	"sort"
	"strings"
)

// SwapSource is the manifest-owned alternative set for one dependency. The
// deployability control plane consumes this shape so callers do not need to
// import a scenario's internal deployment model.
type SwapSource struct {
	Original     string
	Alternatives []SwapAlternative
}

type SwapAlternative struct {
	Name         string
	Relationship string
	Reason       string
}

type ResourceSwapSuggestion struct {
	OriginalResource    string   `json:"original_resource"`
	AlternativeResource string   `json:"alternative_resource"`
	Reason              string   `json:"reason"`
	ApplicableTiers     []string `json:"applicable_tiers,omitempty"`
	Relationship        string   `json:"relationship"`
	ImpactDescription   string   `json:"impact_description,omitempty"`
}

// SuggestResourceSwaps is the single swap derivation used by SDA reports and
// the generated fleet readout. It considers only alternatives declared by
// dependency manifests; it never falls back to a name-based catalog.
func SuggestResourceSwaps(sources []SwapSource) []ResourceSwapSuggestion {
	result := make([]ResourceSwapSuggestion, 0)
	seen := make(map[string]struct{})
	for _, source := range sources {
		original := strings.TrimSpace(source.Original)
		if original == "" {
			continue
		}
		for _, alternative := range source.Alternatives {
			name := strings.TrimSpace(alternative.Name)
			if name == "" {
				continue
			}
			key := original + "\x00" + name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			relationship := strings.TrimSpace(alternative.Relationship)
			if relationship == "" {
				relationship = "declared_alternative"
			}
			reason := strings.TrimSpace(alternative.Reason)
			if reason == "" {
				reason = "declared by the dependency manifest"
			}
			result = append(result, ResourceSwapSuggestion{
				OriginalResource:    original,
				AlternativeResource: name,
				Reason:              reason,
				Relationship:        relationship,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].OriginalResource != result[j].OriginalResource {
			return result[i].OriginalResource < result[j].OriginalResource
		}
		return result[i].AlternativeResource < result[j].AlternativeResource
	})
	return result
}

// ExtractDeclaredAlternatives accepts the deployment-dependency object as
// authored in service.json. This keeps the fleet reader schema-tolerant while
// preserving the same fields SDA already understands.
func ExtractDeclaredAlternatives(raw json.RawMessage) []SwapAlternative {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var value struct {
		SwappableWith []struct {
			ID           string `json:"id"`
			Relationship string `json:"relationship"`
			Notes        string `json:"notes"`
		} `json:"swappable_with"`
		PlatformSupport map[string]struct {
			Alternatives []string `json:"alternatives"`
		} `json:"platform_support"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	result := make([]SwapAlternative, 0)
	for _, item := range value.SwappableWith {
		result = append(result, SwapAlternative{Name: item.ID, Relationship: item.Relationship, Reason: item.Notes})
	}
	for _, support := range value.PlatformSupport {
		for _, name := range support.Alternatives {
			result = append(result, SwapAlternative{Name: name, Relationship: "declared_alternative"})
		}
	}
	return result
}

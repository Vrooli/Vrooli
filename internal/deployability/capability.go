package deployability

import (
	"fmt"
	"sort"
	"strings"
)

// CapabilityImplementation describes one tool or safeguard at the resolver
// boundary. The manifest loader supplies these declarations; the resolver
// never knows the fleet's object names.
type CapabilityImplementation struct {
	Name       string
	Capability string
	Role       string
	Platforms  map[HostOS]PlatformDeclaration
}

type CapabilityResolutionStatus string

const (
	CapabilityImplemented CapabilityResolutionStatus = "implemented"
	CapabilityUnwired     CapabilityResolutionStatus = "unwired"
	CapabilityPeerless    CapabilityResolutionStatus = "peerless"
)

type CapabilityResolution struct {
	Capability  string                     `json:"capability"`
	OS          HostOS                     `json:"host_os"`
	Status      CapabilityResolutionStatus `json:"status"`
	Implementer string                     `json:"implementer,omitempty"`
	Mechanism   string                     `json:"mechanism,omitempty"`
	Reason      string                     `json:"reason"`
}

// ResolveCapability finds a platform implementation without maintaining a
// second catalog. A declared mechanism with no implementation is unwired;
// absence of both is peerless. Unknown input is represented by peerless with
// an explicit reason rather than a permissive fallback.
func ResolveCapability(implementations []CapabilityImplementation, capability string, os HostOS) CapabilityResolution {
	capability = strings.TrimSpace(capability)
	result := CapabilityResolution{Capability: capability, OS: os}
	if capability == "" {
		result.Status = CapabilityPeerless
		result.Reason = "capability is empty"
		return result
	}
	if !validOS(os) {
		result.Status = CapabilityPeerless
		result.Reason = fmt.Sprintf("host OS %q is not in the resolver vocabulary", os)
		return result
	}

	var unwired []string
	var candidates []CapabilityImplementation
	for _, implementation := range implementations {
		if strings.TrimSpace(implementation.Capability) != capability {
			continue
		}
		platform, declared := implementation.Platforms[os]
		if !declared {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(platform.Status)) {
		case platformSupported, platformPartial:
			if strings.TrimSpace(implementation.Name) != "" {
				candidates = append(candidates, implementation)
			}
		case platformUnsupported:
			if mechanism := strings.TrimSpace(platform.Mechanism); mechanism != "" {
				unwired = append(unwired, mechanism)
			}
		default:
			if mechanism := strings.TrimSpace(platform.Mechanism); mechanism != "" {
				unwired = append(unwired, mechanism)
			}
		}
	}
	if len(candidates) > 0 {
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Role != candidates[j].Role {
				return candidates[i].Role == "primary"
			}
			return candidates[i].Name < candidates[j].Name
		})
		result.Status = CapabilityImplemented
		result.Implementer = candidates[0].Name
		result.Reason = "declared implementation is available on this host OS"
		return result
	}
	if len(unwired) > 0 {
		sort.Strings(unwired)
		result.Status = CapabilityUnwired
		result.Mechanism = unwired[0]
		result.Reason = "a mechanism is named but no implementation is wired for this host OS"
		return result
	}
	result.Status = CapabilityPeerless
	result.Reason = "no implementation or mechanism is declared for this capability on this host OS"
	return result
}

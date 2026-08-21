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
	// CapabilityImplemented means an implementation is available on this host
	// OS. How much that implementation has been proven is carried separately
	// by CapabilityResolution.Qualification.
	CapabilityImplemented CapabilityResolutionStatus = "implemented"
	// CapabilityDegraded means an implementation is available with known
	// functional limits on this host OS.
	CapabilityDegraded CapabilityResolutionStatus = "degraded"
	// CapabilityIneligible means every declaration deliberately marks this
	// host OS out of scope and none names a mechanism to wire.
	CapabilityIneligible CapabilityResolutionStatus = "ineligible"
	// CapabilityUnwired means a mechanism is named for this host OS but no
	// implementation is declared for it.
	CapabilityUnwired CapabilityResolutionStatus = "unwired"
	// CapabilityPeerless means nothing at all is declared for this host OS.
	CapabilityPeerless CapabilityResolutionStatus = "peerless"
	// CapabilityStatusInvalid is a terminal verdict, not a resolution: a
	// declaration authored a platform status outside the vocabulary.
	CapabilityStatusInvalid CapabilityResolutionStatus = "status_invalid"
)

// HasImplementation reports whether the resolution found code that runs on
// this host OS, whatever its proof level. Callers that only ask "is this
// implemented?" should use this rather than comparing against a single status.
func (s CapabilityResolutionStatus) HasImplementation() bool {
	return s == CapabilityImplemented || s == CapabilityDegraded
}

type CapabilityResolution struct {
	Capability string                     `json:"capability"`
	OS         HostOS                     `json:"host_os"`
	Status     CapabilityResolutionStatus `json:"status"`
	// Qualification is the honesty rung of the winning declaration: how much
	// real-world proof it carries, independent of whether it resolved.
	Qualification Qualification `json:"qualification"`
	Implementer   string        `json:"implementer,omitempty"`
	Mechanism     string        `json:"mechanism,omitempty"`
	Reason        string        `json:"reason"`
}

type capabilityCandidate struct {
	implementation CapabilityImplementation
	qualification  Qualification
}

// ResolveCapability finds a platform implementation without maintaining a
// second catalog. Every member of the platform status vocabulary gets an
// explicit outcome; an unrecognised token is a terminal invalid verdict rather
// than a silent downgrade.
func ResolveCapability(implementations []CapabilityImplementation, capability string, os HostOS) CapabilityResolution {
	capability = strings.TrimSpace(capability)
	result := CapabilityResolution{Capability: capability, OS: os, Qualification: QualificationUndeclared}
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
	var ineligible bool
	var candidates []capabilityCandidate
	for _, implementation := range implementations {
		if strings.TrimSpace(implementation.Capability) != capability {
			continue
		}
		platform, declared := implementation.Platforms[os]
		if !declared {
			continue
		}
		status, err := ParsePlatformStatus(platform.Status)
		if err != nil {
			result.Status = CapabilityStatusInvalid
			result.Implementer = strings.TrimSpace(implementation.Name)
			result.Mechanism = strings.TrimSpace(platform.Mechanism)
			result.Reason = err.Error()
			return result
		}
		named := strings.TrimSpace(implementation.Name) != ""
		switch status {
		case StatusSupported, StatusBuildVerified, StatusExperimental, StatusUnqualified, StatusPartial:
			if named {
				candidates = append(candidates, capabilityCandidate{implementation: implementation, qualification: status.Qualification()})
				continue
			}
			// A declaration that claims an implementation without naming one is
			// exactly the unwired case: the mechanism is described, the code is not.
			if mechanism := strings.TrimSpace(platform.Mechanism); mechanism != "" {
				unwired = append(unwired, mechanism)
			}
		case StatusUnsupported:
			if mechanism := strings.TrimSpace(platform.Mechanism); mechanism != "" {
				unwired = append(unwired, mechanism)
				continue
			}
			ineligible = true
		}
	}
	if len(candidates) > 0 {
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].qualification != candidates[j].qualification {
				return candidates[i].qualification.Rank() > candidates[j].qualification.Rank()
			}
			if candidates[i].implementation.Role != candidates[j].implementation.Role {
				return candidates[i].implementation.Role == "primary"
			}
			return candidates[i].implementation.Name < candidates[j].implementation.Name
		})
		winner := candidates[0]
		result.Qualification = winner.qualification
		result.Status = CapabilityImplemented
		if winner.qualification == QualificationDegraded {
			result.Status = CapabilityDegraded
		}
		result.Implementer = winner.implementation.Name
		result.Mechanism = strings.TrimSpace(winner.implementation.Platforms[os].Mechanism)
		result.Reason = winner.qualification.Reason()
		return result
	}
	if len(unwired) > 0 {
		sort.Strings(unwired)
		result.Status = CapabilityUnwired
		result.Mechanism = unwired[0]
		result.Qualification = QualificationUndeclared
		result.Reason = "a mechanism is named but no implementation is declared for this host OS"
		return result
	}
	if ineligible {
		result.Status = CapabilityIneligible
		result.Qualification = QualificationIneligible
		result.Reason = QualificationIneligible.Reason()
		return result
	}
	result.Status = CapabilityPeerless
	result.Reason = "no implementation or mechanism is declared for this capability on this host OS"
	return result
}

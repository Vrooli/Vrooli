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

type CapabilityDeclarer struct {
	Name           string `json:"name"`
	Role           string `json:"role"`
	DeclaredStatus string `json:"declared_status"`
	Resolved       bool   `json:"resolved"`
	Reason         string `json:"reason"`
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
	CapabilityStatusInvalid      CapabilityResolutionStatus = "status_invalid"
	CapabilityControlsIncomplete CapabilityResolutionStatus = "controls_incomplete"
)

// HasImplementation reports whether the resolution found code that runs on
// this host OS, whatever its proof level. Callers that only ask "is this
// implemented?" should use this rather than comparing against a single status.
func (s CapabilityResolutionStatus) HasImplementation() bool {
	return s == CapabilityImplemented || s == CapabilityDegraded || s == CapabilityControlsIncomplete
}

type CapabilityResolution struct {
	Capability string                     `json:"capability"`
	OS         HostOS                     `json:"host_os"`
	Status     CapabilityResolutionStatus `json:"status"`
	// Qualification is the honesty rung of the winning declaration: how much
	// real-world proof it carries, independent of whether it resolved.
	Qualification   Qualification        `json:"qualification"`
	Implementer     string               `json:"implementer,omitempty"`
	Mechanism       string               `json:"mechanism,omitempty"`
	Reason          string               `json:"reason"`
	Controls        []string             `json:"controls,omitempty"`
	Absent          []string             `json:"absent,omitempty"`
	AbsentControls  []string             `json:"absent_controls,omitempty"`
	AbsentProviders []string             `json:"absent_providers,omitempty"`
	Declarers       []CapabilityDeclarer `json:"declarers,omitempty"`
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
	var controls []capabilityCandidate
	var declarers []string
	resolved := make(map[string]bool)
	for _, implementation := range implementations {
		if strings.TrimSpace(implementation.Capability) != capability {
			continue
		}
		name := strings.TrimSpace(implementation.Name)
		if name != "" {
			declarers = append(declarers, name)
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
			result.Declarers = declarerDetails(implementations, capability, os, resolved)
			return result
		}
		if implementation.Role == "control" && name != "" && status != StatusUnsupported {
			controls = append(controls, capabilityCandidate{implementation: implementation, qualification: status.Qualification()})
			resolved[name] = true
		}
		if implementation.Role == "control" {
			continue
		}
		named := strings.TrimSpace(implementation.Name) != ""
		switch status {
		case StatusSupported, StatusBuildVerified, StatusExperimental, StatusUnqualified, StatusPartial:
			if named {
				candidates = append(candidates, capabilityCandidate{implementation: implementation, qualification: status.Qualification()})
				resolved[name] = true
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
		result.Controls = sortedNames(controls)
		result.Absent = absentNames(declarers, resolved)
		result.AbsentControls, result.AbsentProviders = absentByRole(implementations, capability, os, resolved)
		result.Declarers = declarerDetails(implementations, capability, os, resolved)
		if len(controls) < countControls(implementations, capability, os) {
			result.Status = CapabilityControlsIncomplete
			result.Reason = fmt.Sprintf("provider %q resolves, but required controls are absent: %s", result.Implementer, strings.Join(result.AbsentControls, ", "))
		}
		return result
	}
	if len(unwired) > 0 {
		sort.Strings(unwired)
		result.Status = CapabilityUnwired
		result.Mechanism = unwired[0]
		result.Qualification = QualificationUndeclared
		result.Reason = "a mechanism is named but no implementation is declared for this host OS"
		result.Controls = sortedNames(controls)
		result.Absent = absentNames(declarers, resolved)
		result.AbsentControls, result.AbsentProviders = absentByRole(implementations, capability, os, resolved)
		result.Declarers = declarerDetails(implementations, capability, os, resolved)
		return result
	}
	if ineligible {
		result.Status = CapabilityIneligible
		result.Qualification = QualificationIneligible
		result.Reason = QualificationIneligible.Reason()
		result.Absent = absentNames(declarers, resolved)
		result.AbsentControls, result.AbsentProviders = absentByRole(implementations, capability, os, resolved)
		result.Declarers = declarerDetails(implementations, capability, os, resolved)
		return result
	}
	result.Status = CapabilityPeerless
	result.Reason = "no implementation or mechanism is declared for this capability on this host OS"
	result.Controls = sortedNames(controls)
	result.Absent = absentNames(declarers, resolved)
	result.AbsentControls, result.AbsentProviders = absentByRole(implementations, capability, os, resolved)
	result.Declarers = declarerDetails(implementations, capability, os, resolved)
	return result
}

func declarerDetails(implementations []CapabilityImplementation, capability string, os HostOS, resolved map[string]bool) []CapabilityDeclarer {
	details := make([]CapabilityDeclarer, 0)
	for _, implementation := range implementations {
		if strings.TrimSpace(implementation.Capability) != capability || strings.TrimSpace(implementation.Name) == "" {
			continue
		}
		name := strings.TrimSpace(implementation.Name)
		platform, declared := implementation.Platforms[os]
		detail := CapabilityDeclarer{Name: name, Role: implementation.Role, Resolved: resolved[name]}
		if !declared {
			detail.DeclaredStatus = string(StatusUnsupported)
			detail.Reason = "no declaration for this host OS"
		} else {
			detail.DeclaredStatus = strings.TrimSpace(platform.Status)
			if !detail.Resolved {
				detail.Reason = "declaration does not resolve on this host OS"
			}
		}
		details = append(details, detail)
	}
	sort.Slice(details, func(i, j int) bool { return details[i].Name < details[j].Name })
	return details
}

func countControls(implementations []CapabilityImplementation, capability string, os HostOS) int {
	count := 0
	for _, implementation := range implementations {
		if strings.TrimSpace(implementation.Capability) == capability && implementation.Role == "control" && strings.TrimSpace(implementation.Name) != "" {
			if _, declared := implementation.Platforms[os]; !declared {
				continue
			}
			count++
		}
	}
	return count
}

func absentByRole(implementations []CapabilityImplementation, capability string, os HostOS, resolved map[string]bool) ([]string, []string) {
	var controls, providers []string
	seen := make(map[string]struct{})
	for _, implementation := range implementations {
		name := strings.TrimSpace(implementation.Name)
		if strings.TrimSpace(implementation.Capability) != capability || name == "" || resolved[name] {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if implementation.Role == "control" {
			controls = append(controls, name)
		} else {
			providers = append(providers, name)
		}
	}
	sort.Strings(controls)
	sort.Strings(providers)
	return controls, providers
}

func sortedNames(candidates []capabilityCandidate) []string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, strings.TrimSpace(candidate.implementation.Name))
	}
	sort.Strings(names)
	return names
}

func absentNames(declarers []string, resolved map[string]bool) []string {
	seen := make(map[string]struct{}, len(declarers))
	absent := make([]string, 0, len(declarers))
	for _, name := range declarers {
		if resolved[name] {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		absent = append(absent, name)
	}
	sort.Strings(absent)
	return absent
}

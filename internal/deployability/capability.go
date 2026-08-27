package deployability

import (
	"fmt"
	"sort"
	"strings"
)

const (
	capabilityControl = "control"
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
	Evidence        *Evidence            `json:"evidence,omitempty"`
	Implementer     string               `json:"implementer,omitempty"`
	Mechanism       string               `json:"mechanism,omitempty"`
	Since           string               `json:"since,omitempty"`
	ReviewBy        string               `json:"review_by,omitempty"`
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

	scan, invalid := scanCapabilityDeclarations(implementations, capability, os)
	if invalid != nil {
		return *invalid
	}
	if len(scan.candidates) > 0 {
		return resolveProviderCandidate(result, implementations, scan, capability, os)
	}
	if len(scan.controls) > 0 && countControls(implementations, capability, os) == len(scan.controls) {
		return resolveControlOnlyCapability(result, implementations, scan, capability, os)
	}
	return resolveAbsentCapability(result, implementations, scan, capability, os)
}

type capabilityScan struct {
	unwired                     []string
	unwiredSince, unwiredReview string
	ineligibleReview            string
	ineligible                  bool
	candidates                  []capabilityCandidate
	controls                    []capabilityCandidate
	declarers                   []string
	resolved                    map[string]bool
}

func scanCapabilityDeclarations(implementations []CapabilityImplementation, capability string, os HostOS) (capabilityScan, *CapabilityResolution) {
	scan := capabilityScan{resolved: make(map[string]bool)}
	for _, implementation := range implementations {
		if invalid := scan.addImplementation(implementation, capability, os); invalid != nil {
			invalid.Declarers = declarerDetails(implementations, capability, os, scan.resolved)
			return scan, invalid
		}
	}
	return scan, nil
}

func (s *capabilityScan) addImplementation(implementation CapabilityImplementation, capability string, os HostOS) *CapabilityResolution {
	if strings.TrimSpace(implementation.Capability) != capability {
		return nil
	}
	name := strings.TrimSpace(implementation.Name)
	if name != "" {
		s.declarers = append(s.declarers, name)
	}
	platform, declared := implementation.Platforms[os]
	if !declared {
		return nil
	}
	status, err := ParsePlatformStatus(platform.Status)
	if err != nil {
		return &CapabilityResolution{Capability: capability, OS: os, Status: CapabilityStatusInvalid, Qualification: QualificationUndeclared, Implementer: name, Mechanism: strings.TrimSpace(platform.Mechanism), Reason: err.Error()}
	}
	if implementation.Role == capabilityControl {
		s.addControl(implementation, name, status)
		return nil
	}
	switch status {
	case StatusSupported, StatusBuildVerified, StatusExperimental, StatusUnqualified, StatusPartial:
		if name != "" {
			s.candidates = append(s.candidates, capabilityCandidate{implementation: implementation, qualification: status.Qualification()})
			s.resolved[name] = true
			return nil
		}
	}
	s.addUnresolvedPlatform(platform, status)
	return nil
}

func (s *capabilityScan) addControl(implementation CapabilityImplementation, name string, status PlatformStatus) {
	if name == "" || status == StatusUnsupported || status == StatusNotImplemented || status == StatusNotApplicable {
		return
	}
	s.controls = append(s.controls, capabilityCandidate{implementation: implementation, qualification: status.Qualification()})
	s.resolved[name] = true
}

func (s *capabilityScan) addUnresolvedPlatform(platform PlatformDeclaration, status PlatformStatus) {
	mechanism := strings.TrimSpace(platform.Mechanism)
	if mechanism != "" && status != StatusNotApplicable {
		s.unwired = append(s.unwired, mechanism)
		if s.unwiredSince == "" {
			s.unwiredSince = strings.TrimSpace(platform.Since)
			s.unwiredReview = strings.TrimSpace(platform.ReviewBy)
		}
		return
	}
	s.ineligible = true
	if status == StatusNotApplicable && s.ineligibleReview == "" {
		s.ineligibleReview = strings.TrimSpace(platform.ReviewBy)
	}
}

func resolveProviderCandidate(result CapabilityResolution, implementations []CapabilityImplementation, scan capabilityScan, capability string, os HostOS) CapabilityResolution {
	sort.SliceStable(scan.candidates, func(i, j int) bool {
		if scan.candidates[i].qualification != scan.candidates[j].qualification {
			return scan.candidates[i].qualification.Rank() > scan.candidates[j].qualification.Rank()
		}
		if scan.candidates[i].implementation.Role != scan.candidates[j].implementation.Role {
			return scan.candidates[i].implementation.Role == "primary"
		}
		return scan.candidates[i].implementation.Name < scan.candidates[j].implementation.Name
	})
	winner := scan.candidates[0]
	result.Qualification = winner.qualification
	result.Status = CapabilityImplemented
	if winner.qualification == QualificationDegraded {
		result.Status = CapabilityDegraded
	}
	result.Implementer = winner.implementation.Name
	result.Mechanism = strings.TrimSpace(winner.implementation.Platforms[os].Mechanism)
	result.Since = strings.TrimSpace(winner.implementation.Platforms[os].Since)
	result.ReviewBy = strings.TrimSpace(winner.implementation.Platforms[os].ReviewBy)
	result.Evidence = winner.implementation.Platforms[os].Evidence
	result.Reason = winner.qualification.Reason()
	populateResolutionDetails(&result, implementations, scan, capability, os)
	if len(scan.controls) < countControls(implementations, capability, os) {
		result.Status = CapabilityControlsIncomplete
		result.Reason = fmt.Sprintf("provider %q resolves, but required controls are absent: %s", result.Implementer, strings.Join(result.AbsentControls, ", "))
	}
	return result
}

func resolveControlOnlyCapability(result CapabilityResolution, implementations []CapabilityImplementation, scan capabilityScan, capability string, os HostOS) CapabilityResolution {
	// A controls-only capability is still an implementation when every
	// required control resolves. Controls are deliberately not promoted to a
	// provider role just to satisfy this branch: the capability remains owned
	// by its controls and reports the weakest control qualification.
	qualification := QualificationQualified
	for _, control := range scan.controls {
		if control.qualification.Rank() < qualification.Rank() {
			qualification = control.qualification
		}
	}
	result.Status = CapabilityImplemented
	if qualification == QualificationDegraded {
		result.Status = CapabilityDegraded
	}
	result.Qualification = qualification
	result.Reason = fmt.Sprintf("all %d control declarers resolve on %s", len(scan.controls), os)
	populateResolutionDetails(&result, implementations, scan, capability, os)
	return result
}

func resolveAbsentCapability(result CapabilityResolution, implementations []CapabilityImplementation, scan capabilityScan, capability string, os HostOS) CapabilityResolution {
	if len(scan.controls) > 0 && len(scan.controls) < countControls(implementations, capability, os) {
		result.Status = CapabilityControlsIncomplete
		result.Qualification = QualificationUndeclared
		result.Reason = fmt.Sprintf("%d of %d required control declarers resolve on %s", len(scan.controls), countControls(implementations, capability, os), os)
		populateResolutionDetails(&result, implementations, scan, capability, os)
		return result
	}
	if len(scan.unwired) > 0 {
		sort.Strings(scan.unwired)
		result.Status = CapabilityUnwired
		result.Mechanism = scan.unwired[0]
		result.Since = scan.unwiredSince
		result.ReviewBy = scan.unwiredReview
		result.Qualification = QualificationUndeclared
		result.Reason = "a mechanism is named but no implementation is declared for this host OS"
		populateResolutionDetails(&result, implementations, scan, capability, os)
		return result
	}
	if scan.ineligible {
		result.Status = CapabilityIneligible
		result.Qualification = QualificationIneligible
		result.ReviewBy = scan.ineligibleReview
		result.Reason = QualificationIneligible.Reason()
		populateResolutionDetails(&result, implementations, scan, capability, os)
		return result
	}
	result.Status = CapabilityPeerless
	result.Reason = "no implementation or mechanism is declared for this capability on this host OS"
	populateResolutionDetails(&result, implementations, scan, capability, os)
	return result
}

func populateResolutionDetails(result *CapabilityResolution, implementations []CapabilityImplementation, scan capabilityScan, capability string, os HostOS) {
	result.Controls = sortedNames(scan.controls)
	result.Absent = absentNames(scan.declarers, scan.resolved)
	result.AbsentControls, result.AbsentProviders = absentByRole(implementations, capability, os, scan.resolved)
	result.Declarers = declarerDetails(implementations, capability, os, scan.resolved)
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
		if strings.TrimSpace(implementation.Capability) == capability && implementation.Role == capabilityControl && strings.TrimSpace(implementation.Name) != "" {
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
		if implementation.Role == capabilityControl {
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

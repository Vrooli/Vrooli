// Package deployability derives deployment truth from declared inputs.
//
// The package is deliberately pure: callers load manifests and operator state
// at their boundary, then pass immutable declarations to Resolve. It never
// knows a resource, scenario, tool, or safeguard by name.
package deployability

import (
	"fmt"
	"sort"
	"strings"
)

type HostOS string

const (
	HostOSLinux   HostOS = "linux"
	HostOSMacOS   HostOS = "macos"
	HostOSWindows HostOS = "windows"
)

type Bundling string

const (
	BundlingVendorable   Bundling = "vendorable"
	BundlingHostRequired Bundling = "host-required"
	BundlingProhibited   Bundling = "prohibited"
)

type DeliveryTier string

const (
	TierLocal      DeliveryTier = "tier-1-local"
	TierDesktop    DeliveryTier = "tier-2-desktop"
	TierMobile     DeliveryTier = "tier-3-mobile"
	TierSaaS       DeliveryTier = "tier-4-saas"
	TierEnterprise DeliveryTier = "tier-5-enterprise"
)

type Verdict string

const (
	VerdictEligible   Verdict = "eligible"
	VerdictDegraded   Verdict = "degraded"
	VerdictIneligible Verdict = "ineligible"
	VerdictUnknown    Verdict = "unknown"
)

type ResourceRequirements struct {
	Class      string
	Weight     float64
	RAMMB      float64
	DiskMB     float64
	CPUCores   float64
	GPU        bool
	Network    string
	Source     string
	Confidence string
}

type PlatformDeclaration struct {
	Status    string
	Mechanism string
}

type CapabilityDeclaration struct {
	Name      string
	Mechanism string
	Platforms map[HostOS]PlatformDeclaration
}

// DependencyDeclaration is the normalized boundary representation of any
// resource, scenario, tool, or safeguard. The loader owns translating its
// manifest into this shape; the resolver only reasons over it.
type DependencyDeclaration struct {
	Kind            string
	Name            string
	Required        bool
	Bundling        Bundling
	Present         bool
	Artifact        bool
	PlatformSupport map[HostOS]PlatformDeclaration
	// HostRequirements are authored by a delivery profile and keyed by host
	// OS because a profile may require different mechanisms per platform.
	HostRequirements map[HostOS][]string
	Requirements     *ResourceRequirements
	Capabilities     []CapabilityDeclaration
	Children         []TargetDeclaration
}

type TargetDeclaration struct {
	Name         string
	Dependencies []DependencyDeclaration
}

type ResolutionInput struct {
	Target TargetDeclaration
	Tier   DeliveryTier
	OS     HostOS
}

type Reason struct {
	Code        string `json:"code"`
	Dependency  string `json:"dependency,omitempty"`
	Requirement string `json:"requirement,omitempty"`
	Message     string `json:"message"`
}

type DependencyResult struct {
	Kind     string   `json:"kind"`
	Name     string   `json:"name"`
	Required bool     `json:"required"`
	Verdict  Verdict  `json:"verdict"`
	Reasons  []Reason `json:"reasons,omitempty"`
}

type Resolution struct {
	Target       string             `json:"target"`
	Tier         DeliveryTier       `json:"tier"`
	OS           HostOS             `json:"host_os"`
	Verdict      Verdict            `json:"verdict"`
	Reasons      []Reason           `json:"reasons,omitempty"`
	Dependencies []DependencyResult `json:"dependencies,omitempty"`
}

const (
	platformSupported   = "supported"
	platformPartial     = "partial"
	platformUnsupported = "unsupported"
	minimumMobileWeight = 3
	minimumMobileRAMMB  = 4096
	minimumMobileCPU    = 2
)

func Resolve(input ResolutionInput) Resolution {
	result := Resolution{Target: strings.TrimSpace(input.Target.Name), Tier: input.Tier, OS: input.OS, Verdict: VerdictEligible}
	if result.Target == "" {
		return unknownResolution(result, Reason{Code: "target_missing", Message: "target name is required"})
	}
	if !validTier(input.Tier) {
		return unknownResolution(result, Reason{Code: "tier_unknown", Message: fmt.Sprintf("delivery tier %q is not in the declared tier vocabulary", input.Tier)})
	}
	if !validOS(input.OS) {
		return unknownResolution(result, Reason{Code: "host_os_unknown", Message: fmt.Sprintf("host OS %q is not in the declared host OS vocabulary", input.OS)})
	}

	for _, dependency := range input.Target.Dependencies {
		resolved := resolveDependency(dependency, input.Tier, input.OS, map[string]struct{}{})
		result.Dependencies = append(result.Dependencies, resolved)
		result.Reasons = append(result.Reasons, resolved.Reasons...)
	}
	result.Verdict = aggregateVerdict(result.Dependencies)
	return result
}

func resolveDependency(dependency DependencyDeclaration, tier DeliveryTier, os HostOS, stack map[string]struct{}) DependencyResult {
	result := DependencyResult{Kind: strings.TrimSpace(dependency.Kind), Name: strings.TrimSpace(dependency.Name), Required: dependency.Required, Verdict: VerdictEligible}
	if result.Kind == "" || result.Name == "" {
		return unknownDependency(result, Reason{Code: "declaration_incomplete", Message: "dependency kind and name are required"})
	}
	if _, exists := stack[result.Kind+":"+result.Name]; exists {
		return unknownDependency(result, Reason{Code: "dependency_cycle", Dependency: result.Name, Message: "dependency closure contains a cycle"})
	}
	if len(dependency.PlatformSupport) == 0 {
		return unknownDependency(result, Reason{Code: "platform_declaration_missing", Dependency: result.Name, Message: "dependency does not declare platform support"})
	}
	platform, exists := dependency.PlatformSupport[os]
	if !exists {
		return unknownDependency(result, Reason{Code: "platform_declaration_missing", Dependency: result.Name, Message: fmt.Sprintf("dependency has no declaration for %s", os)})
	}
	switch strings.ToLower(strings.TrimSpace(platform.Status)) {
	case platformUnsupported:
		result.Verdict = VerdictIneligible
		result.Reasons = append(result.Reasons, Reason{Code: "platform_unsupported", Dependency: result.Name, Message: fmt.Sprintf("platform declaration marks %s unsupported on %s", result.Name, os)})
	case platformPartial:
		result.Verdict = VerdictDegraded
		result.Reasons = append(result.Reasons, Reason{Code: "platform_partial", Dependency: result.Name, Message: fmt.Sprintf("platform declaration is partial on %s", os)})
	case platformSupported:
	default:
		return unknownDependency(result, Reason{Code: "platform_status_unknown", Dependency: result.Name, Message: fmt.Sprintf("platform status %q is not supported by the resolver", platform.Status)})
	}

	if dependency.Requirements == nil {
		return unknownDependency(result, Reason{Code: "requirements_missing", Dependency: result.Name, Message: "dependency does not declare requirements"})
	}
	if err := validateRequirements(*dependency.Requirements); err != nil {
		return unknownDependency(result, Reason{Code: "requirements_invalid", Dependency: result.Name, Message: err.Error()})
	}
	if desktop := desktopEligibility(dependency, tier, os); desktop != nil {
		result.Verdict = combineVerdict(result.Verdict, Verdict(desktop.Verdict))
		result.Reasons = append(result.Reasons, Reason{Code: "bundling_" + string(desktop.Verdict), Dependency: result.Name, Message: desktop.Reason})
	}
	if tier == TierDesktop {
		for _, requirement := range dependency.HostRequirements[os] {
			requirement = strings.TrimSpace(requirement)
			if requirement == "" {
				continue
			}
			verdict := VerdictIneligible
			if !dependency.Required {
				verdict = VerdictDegraded
			}
			result.Verdict = combineVerdict(result.Verdict, verdict)
			result.Reasons = append(result.Reasons, Reason{
				Code:        "host_requirement",
				Dependency:  result.Name,
				Requirement: requirement,
				Message:     fmt.Sprintf("desktop profile requires host prerequisite %q on %s", requirement, os),
			})
		}
	}
	if footprintVerdict, footprintReason := evaluateFootprint(*dependency.Requirements, tier); footprintVerdict != VerdictEligible {
		result.Verdict = combineVerdict(result.Verdict, footprintVerdict)
		result.Reasons = append(result.Reasons, Reason{Code: "footprint_" + string(footprintVerdict), Dependency: result.Name, Message: footprintReason})
	}

	childStack := cloneStack(stack)
	childStack[result.Kind+":"+result.Name] = struct{}{}
	for _, child := range dependency.Children {
		childResult := resolveTarget(child, tier, os, childStack)
		result.Verdict = combineVerdict(result.Verdict, childResult.Verdict)
		result.Reasons = append(result.Reasons, childResult.Reasons...)
	}
	return result
}

func resolveTarget(target TargetDeclaration, tier DeliveryTier, os HostOS, stack map[string]struct{}) DependencyResult {
	result := DependencyResult{Kind: "scenario", Name: strings.TrimSpace(target.Name), Required: true, Verdict: VerdictEligible}
	if result.Name == "" {
		return unknownDependency(result, Reason{Code: "target_missing", Message: "child target name is required"})
	}
	key := result.Kind + ":" + result.Name
	if _, exists := stack[key]; exists {
		return unknownDependency(result, Reason{Code: "dependency_cycle", Dependency: result.Name, Message: "dependency closure contains a cycle"})
	}
	stack = cloneStack(stack)
	stack[key] = struct{}{}
	for _, child := range target.Dependencies {
		childResult := resolveDependency(child, tier, os, stack)
		result.Verdict = combineVerdict(result.Verdict, childResult.Verdict)
		result.Reasons = append(result.Reasons, childResult.Reasons...)
	}
	return result
}

type bundlingResult struct {
	Verdict Verdict
	Reason  string
}

func desktopEligibility(dependency DependencyDeclaration, tier DeliveryTier, os HostOS) *bundlingResult {
	if tier != TierDesktop {
		return nil
	}
	present := dependency.Present || dependency.Artifact
	result := &bundlingResult{Verdict: VerdictEligible}
	switch dependency.Bundling {
	case BundlingProhibited:
		result.Verdict = VerdictIneligible
		result.Reason = fmt.Sprintf("%s %q is prohibited from desktop bundles", dependency.Kind, dependency.Name)
	case BundlingHostRequired:
		if present {
			result.Reason = fmt.Sprintf("host-required %s %q is present", dependency.Kind, dependency.Name)
		} else if dependency.Required {
			result.Verdict = VerdictIneligible
			result.Reason = fmt.Sprintf("required host %s %q is absent on %s", dependency.Kind, dependency.Name, os)
		} else {
			result.Verdict = VerdictDegraded
			result.Reason = fmt.Sprintf("optional host %s %q is absent on %s", dependency.Kind, dependency.Name, os)
		}
	case BundlingVendorable:
		if !present {
			result.Verdict = VerdictIneligible
			result.Reason = fmt.Sprintf("vendorable %s %q has no artifact for %s", dependency.Kind, dependency.Name, os)
		} else {
			result.Reason = fmt.Sprintf("vendorable %s %q has an artifact for %s", dependency.Kind, dependency.Name, os)
		}
	default:
		result.Verdict = VerdictUnknown
		result.Reason = fmt.Sprintf("%s %q has no bundling declaration", dependency.Kind, dependency.Name)
	}
	return result
}

func evaluateFootprint(requirements ResourceRequirements, tier DeliveryTier) (Verdict, string) {
	if tier != TierMobile {
		return VerdictEligible, ""
	}
	if requirements.GPU || requirements.Weight >= minimumMobileWeight || requirements.RAMMB >= minimumMobileRAMMB || requirements.CPUCores >= minimumMobileCPU {
		return VerdictIneligible, fmt.Sprintf("declared footprint class=%q weight=%.1f ram_mb=%.0f cpu_cores=%.1f is not suitable for mobile delivery", requirements.Class, requirements.Weight, requirements.RAMMB, requirements.CPUCores)
	}
	return VerdictEligible, ""
}

func validateRequirements(requirements ResourceRequirements) error {
	if strings.TrimSpace(requirements.Class) == "" {
		return fmt.Errorf("requirements.class is required")
	}
	if requirements.Weight <= 0 {
		return fmt.Errorf("requirements.weight must be greater than zero")
	}
	if requirements.RAMMB < 0 || requirements.DiskMB < 0 || requirements.CPUCores < 0 {
		return fmt.Errorf("resource footprint values cannot be negative")
	}
	return nil
}

func validOS(os HostOS) bool { return os == HostOSLinux || os == HostOSMacOS || os == HostOSWindows }

func validTier(tier DeliveryTier) bool {
	return tier == TierLocal || tier == TierDesktop || tier == TierMobile || tier == TierSaaS || tier == TierEnterprise
}

func aggregateVerdict(results []DependencyResult) Verdict {
	verdict := VerdictEligible
	for _, result := range results {
		verdict = combineVerdict(verdict, result.Verdict)
	}
	return verdict
}

func combineVerdict(left, right Verdict) Verdict {
	if left == VerdictIneligible || right == VerdictIneligible {
		return VerdictIneligible
	}
	if left == VerdictUnknown || right == VerdictUnknown {
		return VerdictUnknown
	}
	if left == VerdictDegraded || right == VerdictDegraded {
		return VerdictDegraded
	}
	return VerdictEligible
}

func unknownResolution(result Resolution, reason Reason) Resolution {
	result.Verdict = VerdictUnknown
	result.Reasons = append(result.Reasons, reason)
	return result
}

func unknownDependency(result DependencyResult, reason Reason) DependencyResult {
	result.Verdict = VerdictUnknown
	result.Reasons = append(result.Reasons, reason)
	return result
}

func cloneStack(stack map[string]struct{}) map[string]struct{} {
	clone := make(map[string]struct{}, len(stack)+1)
	for key := range stack {
		clone[key] = struct{}{}
	}
	return clone
}

func SortReasons(reasons []Reason) {
	sort.SliceStable(reasons, func(i, j int) bool {
		if reasons[i].Dependency != reasons[j].Dependency {
			return reasons[i].Dependency < reasons[j].Dependency
		}
		return reasons[i].Code < reasons[j].Code
	})
}

package deployment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	deployability "github.com/vrooli/vrooli/packages/deployability"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type resourceManifestInput struct {
	Name         string                     `json:"name"`
	Bundling     deployability.Bundling     `json:"bundling"`
	Requirements *resourceRequirementsInput `json:"requirements"`
	Platforms    map[string]string          `json:"platforms"`
	Deployment   resourceDeploymentInput    `json:"deployment"`
}

type resourceRequirementsInput struct {
	Class      string                        `json:"class"`
	Weight     float64                       `json:"weight"`
	RAMMB      float64                       `json:"ram_mb"`
	DiskMB     float64                       `json:"disk_mb"`
	CPUCores   float64                       `json:"cpu_cores"`
	GPU        *deployability.GPURequirement `json:"gpu"`
	Network    string                        `json:"network"`
	Source     string                        `json:"source"`
	Confidence string                        `json:"confidence"`
}

type resourceProfileInput struct {
	Requires []string `json:"requires"`
}

type resourceDeploymentInput struct {
	Profiles map[string]map[string]resourceProfileInput `json:"profiles"`
}

func loadResourceDeclaration(repoRoot, name string, required bool) (deployability.DependencyDeclaration, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return deployability.DependencyDeclaration{}, fmt.Errorf("resource name is required")
	}
	path := filepath.Join(repoRoot, "resources", name, "resource.json")
	raw, err := os.ReadFile(path) // #nosec G304 -- path is rooted at the configured repository and name is a manifest key.
	if err != nil {
		return deployability.DependencyDeclaration{}, fmt.Errorf("read resource manifest %q: %w", name, err)
	}
	var manifest resourceManifestInput
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return deployability.DependencyDeclaration{}, fmt.Errorf("parse resource manifest %q: %w", name, err)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		manifest.Name = name
	}
	var requirements *deployability.ResourceRequirements
	if manifest.Requirements != nil {
		requirements = &deployability.ResourceRequirements{
			Class: manifest.Requirements.Class, Weight: manifest.Requirements.Weight,
			RAMMB: manifest.Requirements.RAMMB, DiskMB: manifest.Requirements.DiskMB,
			CPUCores:       manifest.Requirements.CPUCores,
			GPURequirement: manifest.Requirements.GPU,
			Network:        manifest.Requirements.Network, Source: manifest.Requirements.Source, Confidence: manifest.Requirements.Confidence,
		}
	}
	platforms := make(map[deployability.HostOS]deployability.PlatformDeclaration, len(manifest.Platforms))
	for platform, status := range manifest.Platforms {
		osName, ok := normalizeHostOS(platform)
		if !ok {
			continue
		}
		platforms[osName] = deployability.PlatformDeclaration{Status: status}
	}
	return deployability.DependencyDeclaration{
		Kind:             "resource",
		Name:             manifest.Name,
		Required:         required,
		Present:          true,
		Bundling:         manifest.Bundling,
		PlatformSupport:  platforms,
		HostRequirements: desktopHostRequirements(manifest.Deployment.Profiles),
		Requirements:     requirements,
	}, nil
}

func desktopHostRequirements(profiles map[string]map[string]resourceProfileInput) map[deployability.HostOS][]string {
	result := make(map[deployability.HostOS][]string)
	for rawOS, profile := range profiles["desktop"] {
		hostOS, ok := normalizeHostOS(rawOS)
		if !ok {
			continue
		}
		seen := make(map[string]struct{})
		for _, requirement := range profile.Requires {
			requirement = strings.TrimSpace(requirement)
			if requirement == "" {
				continue
			}
			if _, exists := seen[requirement]; exists {
				continue
			}
			seen[requirement] = struct{}{}
			result[hostOS] = append(result[hostOS], requirement)
		}
		sort.Strings(result[hostOS])
	}
	return result
}

func normalizeHostOS(value string) (deployability.HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return deployability.HostOSLinux, true
	case "macos", "darwin":
		return deployability.HostOSMacOS, true
	case "windows", "win32":
		return deployability.HostOSWindows, true
	default:
		return "", false
	}
}

func resolveResourceTierSupport(repoRoot, name string, required bool) map[string]types.TierSupportSummary {
	declaration, err := loadResourceDeclaration(repoRoot, name, required)
	if err != nil {
		return unknownTierSupport(err.Error())
	}
	return resolveResourceTierSupportFromDeclaration(declaration)
}

func resolveResourceTierSupportFromDeclaration(declaration deployability.DependencyDeclaration) map[string]types.TierSupportSummary {
	tiers := []deployability.DeliveryTier{
		deployability.TierLocal,
		deployability.TierDesktop,
		deployability.TierMobile,
		deployability.TierSaaS,
		deployability.TierEnterprise,
	}
	result := make(map[string]types.TierSupportSummary, len(tiers))
	for _, tier := range tiers {
		verdict := deployability.VerdictEligible
		var reasons []string
		for _, osName := range []deployability.HostOS{deployability.HostOSLinux, deployability.HostOSMacOS, deployability.HostOSWindows} {
			resolved := deployability.Resolve(deployability.ResolutionInput{
				Target: deployability.TargetDeclaration{Name: declaration.Name, Dependencies: []deployability.DependencyDeclaration{declaration}},
				Tier:   tier,
				OS:     osName,
			})
			verdict = combineResolverVerdict(verdict, resolved.Verdict)
			for _, reason := range resolved.Reasons {
				if reason.Message != "" {
					reasons = append(reasons, string(osName)+": "+reason.Message)
				}
			}
		}
		result[string(tier)] = resolverTierSummary(verdict, reasons, declaration.Requirements)
	}
	return result
}

func unknownTierSupport(reason string) map[string]types.TierSupportSummary {
	result := make(map[string]types.TierSupportSummary)
	for _, tier := range []deployability.DeliveryTier{deployability.TierLocal, deployability.TierDesktop, deployability.TierMobile, deployability.TierSaaS, deployability.TierEnterprise} {
		result[string(tier)] = types.TierSupportSummary{Reason: reason}
	}
	return result
}

func resolverTierSummary(verdict deployability.Verdict, reasons []string, requirements *deployability.ResourceRequirements) types.TierSupportSummary {
	summary := types.TierSupportSummary{Reason: strings.Join(dedupeStrings(reasons), "; ")}
	switch verdict {
	case deployability.VerdictEligible, deployability.VerdictDegraded:
		supported := true
		summary.Supported = &supported
	case deployability.VerdictIneligible:
		supported := false
		summary.Supported = &supported
	}
	if requirements != nil {
		gpu := requirements.GPURequirement != nil
		summary.Requirements = &types.DeploymentRequirements{
			Class: requirements.Class, Weight: ptr(requirements.Weight),
			RAMMB: ptr(requirements.RAMMB), DiskMB: ptr(requirements.DiskMB), CPUCores: ptr(requirements.CPUCores),
			GPU: ptr(gpu), Network: requirements.Network, Source: requirements.Source, Confidence: requirements.Confidence,
		}
	}
	return summary
}

func combineResolverVerdict(left, right deployability.Verdict) deployability.Verdict {
	if left == deployability.VerdictIneligible || right == deployability.VerdictIneligible {
		return deployability.VerdictIneligible
	}
	if left == deployability.VerdictUnknown || right == deployability.VerdictUnknown {
		return deployability.VerdictUnknown
	}
	if left == deployability.VerdictDegraded || right == deployability.VerdictDegraded {
		return deployability.VerdictDegraded
	}
	return deployability.VerdictEligible
}

func ptr[T any](value T) *T { return &value }

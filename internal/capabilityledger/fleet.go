package capabilityledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/deployability"
)

// FleetReadout is the computed fleet view. It deliberately contains evidence
// names and resolver verdicts rather than a second authored deployment table.
type FleetReadout struct {
	BlockedByOS     []ScenarioBlock        `json:"blocked_by_os"`
	DockerBlocked   []ScenarioBlock        `json:"docker_blocked"`
	Peerless        []ScenarioPeerless     `json:"peerless"`
	TierUpgrades    []TierUpgrade          `json:"tier_upgrades"`
	DesktopBundling DesktopBundlingVerdict `json:"desktop_bundling"`
}

type ScenarioBlock struct {
	Scenario     string                           `json:"scenario"`
	HostOS       deployability.HostOS             `json:"host_os"`
	Dependencies []deployability.DependencyResult `json:"dependencies"`
}

type ScenarioPeerless struct {
	Scenario     string               `json:"scenario"`
	HostOS       deployability.HostOS `json:"host_os"`
	Capabilities []string             `json:"capabilities"`
}

type TierUpgrade struct {
	Scenario           string                     `json:"scenario"`
	HostOS             deployability.HostOS       `json:"host_os"`
	CurrentTier        deployability.DeliveryTier `json:"current_tier"`
	NextTier           deployability.DeliveryTier `json:"next_tier"`
	Change             string                     `json:"single_change"`
	BlockingDependency string                     `json:"blocking_dependency"`
}

type DesktopBundlingVerdict struct {
	Resources       int    `json:"resources"`
	HostRequired    int    `json:"host_required"`
	Vendorable      int    `json:"vendorable"`
	Prohibited      int    `json:"prohibited"`
	Unknown         int    `json:"unknown"`
	DatabaseBlocked bool   `json:"database_blocked"`
	Reason          string `json:"reason"`
}

type scenarioInput struct {
	Name         string
	Capabilities []string
	Resources    map[string]json.RawMessage
	Swaps        []deployability.SwapSource
}

type dependencyInput struct {
	Enabled  *bool `json:"enabled"`
	Required bool  `json:"required"`
}

type resourceInput struct {
	Name         string                              `json:"name"`
	Bundling     deployability.Bundling              `json:"bundling"`
	Platforms    map[string]string                   `json:"platforms"`
	Requirements *deployability.ResourceRequirements `json:"requirements"`
	Deployment   resourceDeploymentInput             `json:"deployment"`
}

type resourceProfileInput struct {
	Requires []string `json:"requires"`
}

type resourceDeploymentInput struct {
	Profiles map[string]map[string]resourceProfileInput `json:"profiles"`
}

func GenerateFleet(root string) (FleetReadout, error) {
	ledger, err := Generate(root)
	if err != nil {
		return FleetReadout{}, err
	}
	capabilityStatus := make(map[string]map[deployability.HostOS]deployability.CapabilityResolutionStatus)
	for _, entry := range ledger.Capabilities {
		capabilityStatus[entry.Capability] = map[deployability.HostOS]deployability.CapabilityResolutionStatus{}
		for _, hostOS := range operatingSystems {
			capabilityStatus[entry.Capability][hostOS] = entry.Platforms[string(hostOS)].Status
		}
	}

	resources, err := readResourceInputs(root)
	if err != nil {
		return FleetReadout{}, err
	}
	scenarios, err := readScenarioInputs(root)
	if err != nil {
		return FleetReadout{}, err
	}
	readout := FleetReadout{BlockedByOS: []ScenarioBlock{}, DockerBlocked: []ScenarioBlock{}, Peerless: []ScenarioPeerless{}, TierUpgrades: []TierUpgrade{}}
	for _, scenario := range scenarios {
		declarations := make([]deployability.DependencyDeclaration, 0, len(scenario.Resources))
		for name, raw := range scenario.Resources {
			dep, ok, err := resourceDeclaration(resources, name, raw)
			if err != nil {
				return FleetReadout{}, fmt.Errorf("scenario %s resource %s: %w", scenario.Name, name, err)
			}
			if ok {
				declarations = append(declarations, dep)
			}
		}
		sort.Slice(declarations, func(i, j int) bool { return declarations[i].Name < declarations[j].Name })
		for _, hostOS := range operatingSystems {
			resolution := deployability.Resolve(deployability.ResolutionInput{
				Target: deployability.TargetDeclaration{Name: scenario.Name, Dependencies: declarations},
				Tier:   deployability.TierLocal, OS: hostOS,
			})
			blocked := nonEligibleDependencies(resolution.Dependencies)
			if len(blocked) > 0 {
				readout.BlockedByOS = append(readout.BlockedByOS, ScenarioBlock{Scenario: scenario.Name, HostOS: hostOS, Dependencies: blocked})
			}
			desktopResolution := deployability.Resolve(deployability.ResolutionInput{
				Target: deployability.TargetDeclaration{Name: scenario.Name, Dependencies: declarations},
				Tier:   deployability.TierDesktop, OS: hostOS,
			})
			docker := dockerRequirements(desktopResolution.Dependencies)
			if len(docker) > 0 {
				readout.DockerBlocked = append(readout.DockerBlocked, ScenarioBlock{Scenario: scenario.Name, HostOS: hostOS, Dependencies: docker})
			}
			peerless := make([]string, 0)
			for _, capability := range scenario.Capabilities {
				if capabilityStatus[capability][hostOS] == "peerless" {
					peerless = append(peerless, capability)
				}
			}
			if len(peerless) > 0 {
				readout.Peerless = append(readout.Peerless, ScenarioPeerless{Scenario: scenario.Name, HostOS: hostOS, Capabilities: peerless})
			}
			readout.TierUpgrades = append(readout.TierUpgrades, tierUpgradeCandidates(scenario.Name, declarations, scenario.Swaps, hostOS)...)
		}
	}
	readout.DesktopBundling = desktopBundling(resources)
	return readout, nil
}

func readResourceInputs(root string) (map[string]resourceInput, error) {
	paths, err := filepath.Glob(filepath.Join(root, "resources", "*", "resource.json"))
	if err != nil {
		return nil, err
	}
	result := make(map[string]resourceInput, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var item resourceInput
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, fmt.Errorf("decode resource %s: %w", path, err)
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = filepath.Base(filepath.Dir(path))
			item.Name = name
		}
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("resource manifests declare duplicate name %q", name)
		}
		result[name] = item
	}
	return result, nil
}

func readScenarioInputs(root string) ([]scenarioInput, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	result := make([]scenarioInput, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var raw struct {
			Service struct {
				Name         string   `json:"name"`
				Capabilities []string `json:"capabilities"`
			} `json:"service"`
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
			Deployment struct {
				Dependencies struct {
					Resources map[string]json.RawMessage `json:"resources"`
				} `json:"dependencies"`
			} `json:"deployment"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decode scenario %s: %w", path, err)
		}
		resources := map[string]json.RawMessage{}
		swaps := make([]deployability.SwapSource, 0)
		for name, dep := range raw.Dependencies.Resources {
			resources[name] = dep
			swaps = append(swaps, deployability.SwapSource{Original: name, Alternatives: deployability.ExtractDeclaredAlternatives(dep)})
		}
		for name, dep := range raw.Deployment.Dependencies.Resources {
			resources[name] = dep
			swaps = append(swaps, deployability.SwapSource{Original: name, Alternatives: deployability.ExtractDeclaredAlternatives(dep)})
		}
		name := raw.Service.Name
		if name == "" {
			name = filepath.Base(filepath.Dir(filepath.Dir(path)))
		}
		result = append(result, scenarioInput{Name: name, Capabilities: raw.Service.Capabilities, Resources: resources, Swaps: swaps})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func resourceDeclaration(resources map[string]resourceInput, name string, raw json.RawMessage) (deployability.DependencyDeclaration, bool, error) {
	var dep dependencyInput
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &dep); err != nil {
			return deployability.DependencyDeclaration{}, false, err
		}
	}
	if dep.Enabled != nil && !*dep.Enabled {
		return deployability.DependencyDeclaration{}, false, nil
	}
	item, ok := resources[name]
	if !ok {
		// Preserve the declared dependency at the resolver boundary. Dropping it
		// would make a missing manifest look like a successful query; the pure
		// resolver must instead return an explicit unknown verdict.
		return deployability.DependencyDeclaration{
			Kind:     "resource",
			Name:     name,
			Required: dep.Required,
		}, true, nil
	}
	platforms := map[deployability.HostOS]deployability.PlatformDeclaration{}
	for osName, status := range item.Platforms {
		if hostOS, ok := normalizeHostOS(osName); ok {
			platforms[hostOS] = deployability.PlatformDeclaration{Status: status}
		}
	}
	return deployability.DependencyDeclaration{
		Kind:             "resource",
		Name:             item.Name,
		Required:         dep.Required,
		Bundling:         item.Bundling,
		Present:          true,
		PlatformSupport:  platforms,
		HostRequirements: desktopHostRequirements(item.Deployment.Profiles),
		Requirements:     item.Requirements,
	}, true, nil
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

func nonEligibleDependencies(dependencies []deployability.DependencyResult) []deployability.DependencyResult {
	result := make([]deployability.DependencyResult, 0)
	for _, dependency := range dependencies {
		if dependency.Verdict == deployability.VerdictIneligible || dependency.Verdict == deployability.VerdictUnknown {
			result = append(result, dependency)
		}
	}
	return result
}

func dockerRequirements(dependencies []deployability.DependencyResult) []deployability.DependencyResult {
	result := make([]deployability.DependencyResult, 0)
	for _, dependency := range dependencies {
		reasons := make([]deployability.Reason, 0)
		for _, reason := range dependency.Reasons {
			if reason.Code == "host_requirement" && isDockerRequirement(reason.Requirement) {
				reasons = append(reasons, reason)
			}
		}
		if len(reasons) == 0 {
			continue
		}
		dependency.Reasons = reasons
		result = append(result, dependency)
	}
	return result
}

func isDockerRequirement(requirement string) bool {
	switch strings.ToLower(strings.TrimSpace(requirement)) {
	case "docker", "docker-engine", "docker-desktop":
		return true
	default:
		return false
	}
}

func tierUpgradeCandidates(name string, deps []deployability.DependencyDeclaration, swaps []deployability.SwapSource, hostOS deployability.HostOS) []TierUpgrade {
	result := []TierUpgrade{}
	byOriginal := make(map[string][]deployability.ResourceSwapSuggestion)
	for _, suggestion := range deployability.SuggestResourceSwaps(swaps) {
		byOriginal[suggestion.OriginalResource] = append(byOriginal[suggestion.OriginalResource], suggestion)
	}
	for _, tier := range []deployability.DeliveryTier{deployability.TierDesktop, deployability.TierMobile} {
		resolution := deployability.Resolve(deployability.ResolutionInput{Target: deployability.TargetDeclaration{Name: name, Dependencies: deps}, Tier: tier, OS: hostOS})
		if resolution.Verdict == deployability.VerdictEligible {
			continue
		}
		for _, dependency := range resolution.Dependencies {
			if dependency.Verdict == deployability.VerdictIneligible || dependency.Verdict == deployability.VerdictUnknown {
				candidates := byOriginal[dependency.Name]
				if len(candidates) == 0 {
					result = append(result, TierUpgrade{Scenario: name, HostOS: hostOS, CurrentTier: deployability.TierLocal, NextTier: tier, Change: "resolve or replace dependency " + dependency.Name, BlockingDependency: dependency.Name})
				} else {
					result = append(result, TierUpgrade{Scenario: name, HostOS: hostOS, CurrentTier: deployability.TierLocal, NextTier: tier, Change: "replace dependency " + dependency.Name + " with " + candidates[0].AlternativeResource, BlockingDependency: dependency.Name})
				}
				break
			}
		}
	}
	return result
}

func desktopBundling(resources map[string]resourceInput) DesktopBundlingVerdict {
	result := DesktopBundlingVerdict{Resources: len(resources)}
	for _, item := range resources {
		switch item.Bundling {
		case deployability.BundlingHostRequired:
			result.HostRequired++
		case deployability.BundlingVendorable:
			result.Vendorable++
		case deployability.BundlingProhibited:
			result.Prohibited++
		default:
			result.Unknown++
		}
	}
	result.DatabaseBlocked = result.HostRequired > 0
	result.Reason = fmt.Sprintf("%d of %d resources are host-required; desktop scenarios using one cannot be self-contained without a host supply", result.HostRequired, result.Resources)
	return result
}

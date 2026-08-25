package deployment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
	appconfig "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

var platformVerdictOSes = []deployability.HostOS{
	deployability.HostOSLinux,
	deployability.HostOSMacOS,
	deployability.HostOSWindows,
}

// PlatformVerdict is SDA's derived answer for one scenario and one host OS.
// The blocking dependency is deliberately retained: a bare boolean is not
// actionable and invites consumers to rebuild the closure themselves.
type PlatformVerdict struct {
	HostOS             deployability.HostOS
	Status             string
	Reason             string
	BlockingDependency string
	Derived            bool
	Overridden         bool
}

type ScenarioPlatformVerdict struct {
	Scenario       string
	Platforms      []PlatformVerdict
	Overridden     bool
	OverrideReason string
}

type FleetDependencyBlock struct {
	Scenario   string
	HostOS     deployability.HostOS
	Dependency string
	Reason     string
}

type FleetTierUpgrade struct {
	Scenario           string
	HostOS             deployability.HostOS
	CurrentTier        deployability.DeliveryTier
	NextTier           deployability.DeliveryTier
	Change             string
	BlockingDependency string
}

type PlatformFleetReport struct {
	Scenarios     []ScenarioPlatformVerdict
	DockerBlocked []FleetDependencyBlock
	TierUpgrades  []FleetTierUpgrade
}

// ListPlatformVerdicts derives every scenario from its declared dependency
// closure. It is intentionally the only SDA-owned fleet platform computation;
// consumers receive this result over the typed Connect surface.
func ListPlatformVerdicts(scenariosDir, filter string, now time.Time) ([]ScenarioPlatformVerdict, error) {
	report, err := BuildPlatformFleet(scenariosDir, filter, now)
	if err != nil {
		return nil, err
	}
	return report.Scenarios, nil
}

func BuildPlatformFleet(scenariosDir, filter string, now time.Time) (PlatformFleetReport, error) {
	scenarios, err := listScenarioNames(scenariosDir)
	if err != nil {
		return PlatformFleetReport{}, err
	}
	filter = strings.TrimSpace(filter)
	if filter != "" {
		scenarios = []string{filter}
	}
	repoRoot := filepath.Dir(scenariosDir)
	result := PlatformFleetReport{Scenarios: make([]ScenarioPlatformVerdict, 0, len(scenarios)), DockerBlocked: []FleetDependencyBlock{}, TierUpgrades: []FleetTierUpgrade{}}
	for _, name := range scenarios {
		cfg, err := appconfig.LoadServiceConfig(filepath.Join(scenariosDir, name))
		if err != nil {
			return PlatformFleetReport{}, fmt.Errorf("load scenario %q: %w", name, err)
		}
		declarations, swaps, err := collectScenarioDependencies(scenariosDir, repoRoot, name, cfg, map[string]bool{})
		if err != nil {
			return PlatformFleetReport{}, err
		}
		overrides, err := authoredPlatformOverrides(filepath.Join(scenariosDir, name, ".vrooli", "service.json"))
		if err != nil {
			return PlatformFleetReport{}, fmt.Errorf("scenario %s platform overrides: %w", name, err)
		}
		verdict := ScenarioPlatformVerdict{Scenario: name, Platforms: make([]PlatformVerdict, 0, len(platformVerdictOSes))}
		for _, hostOS := range platformVerdictOSes {
			if override, ok := overrides[hostOS]; ok {
				verdict.Overridden = true
				verdict.OverrideReason = "service.platform_capabilities overrides dependency derivation for this scenario"
				verdict.Platforms = append(verdict.Platforms, override)
			} else {
				resolution := deployability.Resolve(deployability.ResolutionInput{
					Target: deployability.TargetDeclaration{Name: name, Dependencies: declarations},
					Tier:   deployability.TierLocal,
					OS:     hostOS,
				})
				blocking := ""
				for _, dependency := range resolution.Dependencies {
					if dependency.Verdict == deployability.VerdictIneligible || dependency.Verdict == deployability.VerdictUnknown {
						blocking = dependency.Name
						break
					}
				}
				status := "eligible"
				if blocking != "" {
					status = "blocked"
				}
				reason := "all declared dependencies resolve on " + string(hostOS)
				if blocking != "" {
					reason = "dependency " + blocking + " does not resolve on " + string(hostOS)
				}
				verdict.Platforms = append(verdict.Platforms, PlatformVerdict{HostOS: hostOS, Status: status, Reason: reason, BlockingDependency: blocking, Derived: true})
			}
			desktop := deployability.Resolve(deployability.ResolutionInput{Target: deployability.TargetDeclaration{Name: name, Dependencies: declarations}, Tier: deployability.TierDesktop, OS: hostOS})
			for _, dependency := range desktop.Dependencies {
				for _, dependencyReason := range dependency.Reasons {
					if dependencyReason.Code == "host_requirement" && isDockerRequirement(dependencyReason.Requirement) {
						result.DockerBlocked = append(result.DockerBlocked, FleetDependencyBlock{Scenario: name, HostOS: hostOS, Dependency: dependency.Name, Reason: dependencyReason.Message})
					}
				}
			}
			result.TierUpgrades = append(result.TierUpgrades, tierUpgradeCandidates(name, declarations, swaps, hostOS)...)
		}
		result.Scenarios = append(result.Scenarios, verdict)
	}
	_ = now // computed_at is owned by the transport boundary.
	return result, nil
}

type authoredPlatformCapability struct {
	Status    string `json:"status"`
	Mechanism string `json:"mechanism"`
	Evidence  string `json:"evidence"`
	Essential bool   `json:"essential"`
}

// authoredPlatformOverrides reads the two intentionally hand-authored
// scenario capability blocks. They describe capability-level truth that a
// resource closure cannot see, so they win for the corresponding host OS and
// remain explicitly marked as overrides in the transport result.
func authoredPlatformOverrides(path string) (map[deployability.HostOS]PlatformVerdict, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Service struct {
			PlatformCapabilities map[string]map[string]authoredPlatformCapability `json:"platform_capabilities"`
		} `json:"service"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	byOS := make(map[deployability.HostOS][]struct {
		name  string
		claim authoredPlatformCapability
	})
	for capability, platforms := range manifest.Service.PlatformCapabilities {
		for osName, claim := range platforms {
			hostOS, ok := parsePlatformVerdictOS(osName)
			if !ok {
				return nil, fmt.Errorf("unknown host OS %q for capability %q", osName, capability)
			}
			byOS[hostOS] = append(byOS[hostOS], struct {
				name  string
				claim authoredPlatformCapability
			}{name: capability, claim: claim})
		}
	}
	result := make(map[deployability.HostOS]PlatformVerdict, len(byOS))
	for hostOS, claims := range byOS {
		sort.Slice(claims, func(i, j int) bool { return claims[i].name < claims[j].name })
		status := "supported"
		reason := "authored platform_capabilities claims resolve on " + string(hostOS)
		blocking := ""
		var unsupported []string
		for _, item := range claims {
			switch strings.ToLower(strings.TrimSpace(item.claim.Status)) {
			case "unsupported":
				unsupported = append(unsupported, item.name)
				if item.claim.Essential {
					status = "blocked"
					blocking = "capability:" + item.name
				}
			case "partial", "experimental", "unqualified", "build-verified":
				if status == "supported" {
					status = "degraded"
					reason = "authored platform_capabilities include an unqualified claim on " + string(hostOS)
				}
			}
		}
		if len(unsupported) > 0 {
			if status == "blocked" {
				reason = "essential authored capabilities unsupported on " + string(hostOS) + ": " + strings.Join(unsupported, ", ")
			} else {
				status = "degraded"
				reason = "non-essential authored capabilities unsupported on " + string(hostOS) + ": " + strings.Join(unsupported, ", ")
			}
		}
		result[hostOS] = PlatformVerdict{HostOS: hostOS, Status: status, Reason: reason, BlockingDependency: blocking, Overridden: true}
	}
	return result, nil
}

func parsePlatformVerdictOS(value string) (deployability.HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return deployability.HostOSLinux, true
	case "macos", "darwin":
		return deployability.HostOSMacOS, true
	case "windows":
		return deployability.HostOSWindows, true
	default:
		return "", false
	}
}

func isDockerRequirement(requirement string) bool {
	switch strings.ToLower(strings.TrimSpace(requirement)) {
	case "docker", "docker-engine", "docker-desktop":
		return true
	default:
		return false
	}
}

func listScenarioNames(scenariosDir string) ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(scenariosDir, "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(entries))
	for _, path := range entries {
		result = append(result, filepath.Base(filepath.Dir(filepath.Dir(path))))
	}
	sort.Strings(result)
	return result, nil
}

func collectScenarioDependencies(scenariosDir, repoRoot, name string, cfg *types.Manifest, visiting map[string]bool) ([]deployability.DependencyDeclaration, []deployability.SwapSource, error) {
	if visiting[name] {
		return nil, nil, nil
	}
	visiting[name] = true
	declarations := make([]deployability.DependencyDeclaration, 0)
	resources, err := rawScenarioResources(filepath.Join(scenariosDir, name, ".vrooli", "service.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("scenario %s resources: %w", name, err)
	}
	for resourceName, dependency := range resources {
		if dependency.Enabled != nil && !*dependency.Enabled {
			continue
		}
		declaration, err := loadResourceDeclaration(repoRoot, resourceName, dependency.Required)
		if err != nil {
			return nil, nil, fmt.Errorf("scenario %s resource %s: %w", name, resourceName, err)
		}
		declarations = append(declarations, declaration)
	}
	delete(visiting, name)
	sort.SliceStable(declarations, func(i, j int) bool { return declarations[i].Name < declarations[j].Name })
	sources := make([]deployability.SwapSource, 0, len(resources))
	for resourceName, dependency := range resources {
		sources = append(sources, deployability.SwapSource{Original: resourceName, Alternatives: dependency.Alternatives})
	}
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].Original < sources[j].Original })
	return declarations, sources, nil
}

type rawDependencyInput struct {
	Enabled      *bool                           `json:"enabled"`
	Required     bool                            `json:"required"`
	Alternatives []deployability.SwapAlternative `json:"-"`
}

func rawScenarioResources(path string) (map[string]rawDependencyInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Dependencies struct {
			Resources map[string]json.RawMessage `json:"resources"`
		} `json:"dependencies"`
		Deployment struct {
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
		} `json:"deployment"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	result := make(map[string]rawDependencyInput)
	for name, raw := range manifest.Dependencies.Resources {
		var dependency rawDependencyInput
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &dependency); err != nil {
				return nil, fmt.Errorf("decode resource dependency %q: %w", name, err)
			}
		}
		dependency.Alternatives = deployability.ExtractDeclaredAlternatives(raw)
		result[name] = dependency
	}
	for name, raw := range manifest.Deployment.Dependencies.Resources {
		var dependency rawDependencyInput
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &dependency); err != nil {
				return nil, fmt.Errorf("decode deployment resource dependency %q: %w", name, err)
			}
		}
		dependency.Alternatives = deployability.ExtractDeclaredAlternatives(raw)
		result[name] = dependency
	}
	return result, nil
}

func tierUpgradeCandidates(name string, declarations []deployability.DependencyDeclaration, swaps []deployability.SwapSource, hostOS deployability.HostOS) []FleetTierUpgrade {
	byOriginal := make(map[string][]deployability.ResourceSwapSuggestion)
	for _, suggestion := range deployability.SuggestResourceSwaps(swaps) {
		byOriginal[suggestion.OriginalResource] = append(byOriginal[suggestion.OriginalResource], suggestion)
	}
	result := make([]FleetTierUpgrade, 0)
	for _, tier := range []deployability.DeliveryTier{deployability.TierDesktop, deployability.TierMobile} {
		resolution := deployability.Resolve(deployability.ResolutionInput{Target: deployability.TargetDeclaration{Name: name, Dependencies: declarations}, Tier: tier, OS: hostOS})
		if resolution.Verdict == deployability.VerdictEligible {
			continue
		}
		for _, dependency := range resolution.Dependencies {
			if dependency.Verdict != deployability.VerdictIneligible && dependency.Verdict != deployability.VerdictUnknown {
				continue
			}
			change := "resolve or replace dependency " + dependency.Name
			if candidates := byOriginal[dependency.Name]; len(candidates) > 0 {
				change = "replace dependency " + dependency.Name + " with " + candidates[0].AlternativeResource
			}
			result = append(result, FleetTierUpgrade{Scenario: name, HostOS: hostOS, CurrentTier: deployability.TierLocal, NextTier: tier, Change: change, BlockingDependency: dependency.Name})
			break
		}
	}
	return result
}

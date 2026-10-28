package pipeline

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	hostreq "github.com/vrooli/vrooli/packages/hostreq"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	desktoptoolprofiles "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/toolprofiles"
)

// ResourceDeploymentPlan is an immutable per-target deployment decision. It
// is resolved before bundle packaging, staged verbatim, and consumed by the
// desktop runtime. A resource is never silently omitted from a target.
type ResourceDeploymentPlan struct {
	SchemaVersion     string                               `json:"schema_version"`
	ArtifactTrustMode resourcedeployment.ArtifactTrustMode `json:"artifact_trust_mode"`
	Promotable        bool                                 `json:"promotable"`
	Resources         []ResourceDeploymentPlanItem         `json:"resources"`
	HostRequirements  []HostRequirementPlanItem            `json:"host_requirements,omitempty"`
	RendererPacks     []RendererPackPlanItem               `json:"renderer_packs,omitempty"`
	DeferredTargets   []DeferredResourceTarget             `json:"deferred_targets,omitempty"`
}

// RendererPackPlanItem is an explicit, owner-pinned optional renderer tree.
type RendererPackPlanItem struct {
	ToolID       string `json:"tool_id"`
	ProfileID    string `json:"profile_id"`
	Artifact     string `json:"artifact"`
	Role         string `json:"role"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	TreeSHA256   string `json:"tree_sha256"`
}

// DeferredResourceTarget records an acquisition candidate that could not be
// selected while packaging because it requires runtime hardware facts. The
// installed control plane re-evaluates the original resource manifest with
// those facts on first run and can offer the accelerated artifact explicitly.
type DeferredResourceTarget struct {
	TargetIndex    int               `json:"target_index"`
	Resource       string            `json:"resource"`
	OS             string            `json:"os"`
	Architecture   string            `json:"architecture"`
	When           map[string]string `json:"when"`
	Kind           string            `json:"kind,omitempty"`
	URL            string            `json:"url,omitempty"`
	Image          string            `json:"image,omitempty"`
	SHA256         string            `json:"sha256,omitempty"`
	ArtifactSHA256 string            `json:"artifact_sha256,omitempty"`
	Archive        string            `json:"archive,omitempty"`
	Layout         string            `json:"layout,omitempty"`
	BinPath        string            `json:"bin_path,omitempty"`
	Mode           string            `json:"mode,omitempty"`
	SizeBytes      int64             `json:"size_bytes,omitempty"`
	AbsentFacts    []string          `json:"absent_facts,omitempty"`
}

// HostRequirementPlanItem records the selected scenario/resource host
// requirement independently from the resource service artifact decision.
// It preserves the resolver's provenance and terminal eligibility reason in
// the bundle plan instead of making runtime discovery a hidden prerequisite.
type HostRequirementPlanItem struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	OS           string   `json:"os"`
	Architecture string   `json:"architecture"`
	Privilege    string   `json:"privilege"`
	Bundling     string   `json:"bundling"`
	Required     bool     `json:"required"`
	Verdict      string   `json:"verdict"`
	Reason       string   `json:"reason"`
	Artifact     string   `json:"artifact,omitempty"`
	Provenance   []string `json:"provenance,omitempty"`
}

type ResourceDeploymentPlanItem struct {
	RequestedResource string                       `json:"requested_resource"`
	Resource          string                       `json:"resource"`
	OS                string                       `json:"os"`
	Architecture      string                       `json:"architecture"`
	Mode              string                       `json:"mode"`
	Support           string                       `json:"support"`
	Privilege         string                       `json:"privilege"`
	Bundling          string                       `json:"bundling"`
	Eligibility       string                       `json:"eligibility"`
	EligibilityReason string                       `json:"eligibility_reason,omitempty"`
	Requires          []string                     `json:"requires,omitempty"`
	Limitations       []string                     `json:"limitations,omitempty"`
	Evidence          []string                     `json:"evidence,omitempty"`
	SelectedFallback  *ResourceDeploymentFallback  `json:"selected_fallback,omitempty"`
	Artifact          string                       `json:"artifact,omitempty"`
	Files             []ResourceDeploymentArtifact `json:"files,omitempty"`
	Service           *ResourceDeploymentService   `json:"service,omitempty"`
}

// ResourceDeploymentFallback records the declared replacement selected for a
// target. Keeping this explicit makes a compatible fallback auditable instead
// of looking like an accidental resource substitution.
type ResourceDeploymentFallback struct {
	Resource string `json:"resource"`
	Reason   string `json:"reason"`
}

type ResourceDeploymentArtifact struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// ResourceDeploymentService is the independently pinned server portion of a
// bundled-service resource. It is separate from the resource controller so a
// desktop runtime cannot accidentally launch a controller as a service.
type ResourceDeploymentService struct {
	ProviderPolicy resourcedeployment.ProviderPolicy    `json:"provider_policy"`
	Artifact       string                               `json:"artifact"`
	Layout         string                               `json:"layout"`
	EntryPath      string                               `json:"entry_path,omitempty"`
	Version        string                               `json:"version"`
	SHA256         string                               `json:"sha256"`
	Arguments      []string                             `json:"arguments,omitempty"`
	Environment    map[string]string                    `json:"environment,omitempty"`
	Bootstrap      *resourcedeployment.ServiceBootstrap `json:"bootstrap,omitempty"`
	Config         *resourcedeployment.ServiceConfig    `json:"config,omitempty"`
	Ports          []ResourceDeploymentServicePort      `json:"ports,omitempty"`
	HealthChecks   []ResourceDeploymentHealthCheck      `json:"health_checks,omitempty"`
	Files          []ResourceDeploymentArtifact         `json:"files"`
}

// ResourceDeploymentHealthCheck is the resolved readiness contract for a
// bundled server. It is copied from the resource manifest during packaging;
// the runtime must not infer a health endpoint from an open port.
type ResourceDeploymentHealthCheck struct {
	Type           string `json:"type"`
	Target         string `json:"target"`
	ExpectedStatus []int  `json:"expected_status,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type ResourceDeploymentServicePort struct {
	Name string `json:"name"`
	Host int    `json:"host"`
}

type resourceArtifactManifest struct {
	Privilege string `json:"privilege"`
	Bundling  string `json:"bundling"`
	CLI       struct {
		Distribution *struct {
			Kind         string `json:"kind"`
			ArtifactName string `json:"artifact_name"`
		} `json:"distribution"`
	} `json:"cli"`
	Deployment     resourcedeployment.Deployment      `json:"deployment"`
	ManagedService *resourcedeployment.ManagedService `json:"managed_service,omitempty"`
	HealthChecks   []ResourceDeploymentHealthCheck    `json:"health_checks,omitempty"`
	Ports          []ResourceDeploymentServicePort    `json:"ports,omitempty"`
}

// resolveResourceDeploymentPlanWithTrust makes every resource selection before
// any bundle output is written. The artifact hashes are resolved here as well
// so staging cannot change the deployment choice after the gate has passed.
func resolveResourceDeploymentPlanWithTrust(scenarioPath, artifactRoot, toolArtifactRoot string, platformInputs []string, trustMode resourcedeployment.ArtifactTrustMode, rendererSelections ...[]string) (*ResourceDeploymentPlan, error) {
	required, fallbacks, err := requiredScenarioResources(filepath.Join(scenarioPath, ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(filepath.Dir(scenarioPath))
	platforms := make([]resourcedeployment.Platform, 0, len(platformInputs))
	if len(platformInputs) == 0 {
		return nil, fmt.Errorf("desktop target matrix is required")
	}
	seenPlatforms := make(map[string]bool, len(platformInputs))
	for _, raw := range platformInputs {
		platform, err := resourcedeployment.ParsePlatform(raw)
		if err != nil {
			return nil, err
		}
		if seenPlatforms[platform.String()] {
			return nil, fmt.Errorf("desktop target matrix contains duplicate target %s", platform.String())
		}
		seenPlatforms[platform.String()] = true
		platforms = append(platforms, platform)
	}
	checksums := map[string]string(nil)
	plan := &ResourceDeploymentPlan{SchemaVersion: "v6", ArtifactTrustMode: trustMode, Promotable: trustMode == resourcedeployment.ArtifactTrustProduction}
	for _, requested := range required {
		for _, platform := range platforms {
			item, bundled, err := resolveResourceForTarget(root, requested, requested, fallbacks[requested], platform, map[string]bool{})
			if err != nil {
				return nil, err
			}
			if item.Eligibility == "ineligible" {
				// An inspectable artifact may still be built for target validation,
				// but it must never be promoted as a deployable release.
				plan.Promotable = false
			}
			if bundled {
				if artifactRoot == "" {
					return nil, fmt.Errorf("resource %s uses %s on %s but resource_artifact_root is not set", item.Resource, item.Mode, platform.String())
				}
				if checksums == nil {
					checksums, err = loadReleaseManifestArtifacts(artifactRoot, trustMode, filepath.Join(root, "install", "vrooli-release.pub"))
					if err != nil {
						return nil, err
					}
				}
				if err := resolveArtifactFiles(artifactRoot, checksums, &item); err != nil {
					return nil, err
				}
			}
			if deferred := deferredResourceTargets(item.Resource, root, platform); len(deferred) > 0 {
				plan.DeferredTargets = append(plan.DeferredTargets, deferred...)
			}
			plan.Resources = append(plan.Resources, item)
		}
	}
	for _, platform := range platforms {
		hostItems, err := resolveDesktopHostRequirements(root, scenarioPath, required, platform, toolArtifactRoot, trustMode)
		if err != nil {
			return nil, err
		}
		for _, item := range hostItems {
			if item.Verdict == string(hostreq.EligibilityIneligible) {
				plan.Promotable = false
			}
			plan.HostRequirements = append(plan.HostRequirements, item)
		}
	}
	if len(rendererSelections) > 1 {
		return nil, fmt.Errorf("renderer profile selection was provided more than once")
	}
	if len(rendererSelections) == 1 {
		packs, err := resolveRendererPacks(root, toolArtifactRoot, platforms, trustMode, rendererSelections[0])
		if err != nil {
			return nil, err
		}
		plan.RendererPacks = packs
	}
	sort.Slice(plan.Resources, func(i, j int) bool {
		if plan.Resources[i].RequestedResource == plan.Resources[j].RequestedResource {
			return plan.Resources[i].OS+plan.Resources[i].Architecture < plan.Resources[j].OS+plan.Resources[j].Architecture
		}
		return plan.Resources[i].RequestedResource < plan.Resources[j].RequestedResource
	})
	return plan, nil
}

func resolveRendererPacks(root, artifactRoot string, platforms []resourcedeployment.Platform, trustMode resourcedeployment.ArtifactTrustMode, selections []string) ([]RendererPackPlanItem, error) {
	return resolveRendererPacksWithPins(root, artifactRoot, platforms, trustMode, selections, rendererPin)
}

func resolveRendererPacksWithPins(root, artifactRoot string, platforms []resourcedeployment.Platform, trustMode resourcedeployment.ArtifactTrustMode, selections []string, pinFor func(string) (rendererPackPin, bool)) ([]RendererPackPlanItem, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if artifactRoot == "" {
		return nil, fmt.Errorf("renderer_tool_profiles requires tool_artifact_root")
	}
	if trustMode != resourcedeployment.ArtifactTrustDevelopmentLocal && trustMode != resourcedeployment.ArtifactTrustProduction {
		return nil, fmt.Errorf("renderer_tool_profiles requires an explicit artifact_trust_mode")
	}
	selections, err := normalizeRendererToolProfiles(selections)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(selections))
	var result []RendererPackPlanItem
	for _, id := range selections {
		if seen[id] {
			return nil, fmt.Errorf("renderer_tool_profiles repeats %q", id)
		}
		seen[id] = true
		pin, ok := pinFor(id)
		if !ok {
			return nil, fmt.Errorf("unsupported renderer tool profile %q", id)
		}
		item := RendererPackPlanItem{ToolID: id, ProfileID: pin.profileID, TreeSHA256: pin.treeSHA256, Role: pin.role, OS: "linux", Architecture: "amd64"}
		item.Artifact = toolArtifactName(pin.registryName, resourcedeployment.Platform{OS: "linux", Arch: "amd64"})
		for _, target := range platforms {
			if target.OS != item.OS || target.Arch != item.Architecture {
				return nil, fmt.Errorf("renderer profile %q supports linux-amd64 only; target %s is unsupported", id, target.String())
			}
		}
		result = append(result, item)
	}
	manifest, _, err := resourcedeployment.VerifyReleaseDirectory(artifactRoot, trustMode, filepath.Join(root, "install", "vrooli-release.pub"))
	if err != nil {
		return nil, fmt.Errorf("verify renderer tool release: %w", err)
	}
	byName := make(map[string]resourcedeployment.ReleaseArtifact, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		byName[artifact.Name] = artifact
	}
	for index := range result {
		item := &result[index]
		releaseArtifact, ok := byName[item.Artifact]
		if !ok || releaseArtifact.Role != item.Role || releaseArtifact.OS != item.OS || releaseArtifact.Arch != item.Architecture || !strings.EqualFold(releaseArtifact.SHA256, item.TreeSHA256) {
			return nil, fmt.Errorf("renderer release artifact %q does not match pinned profile metadata", item.Artifact)
		}
		path := filepath.Join(artifactRoot, item.Artifact)
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("renderer artifact %q must be a real directory", item.Artifact)
		}
		actual, err := binaryfetch.TreeDigest(path)
		if err != nil || !strings.EqualFold(actual, item.TreeSHA256) {
			return nil, fmt.Errorf("renderer artifact %q does not match its pinned tree", item.Artifact)
		}
	}
	return result, nil
}

type rendererPackPin struct {
	profileID, treeSHA256, role, registryName string
}

func rendererPin(id string) (rendererPackPin, bool) {
	declaration, registryName, ok := desktoptoolprofiles.PinnedDeclaration(id)
	if !ok {
		return rendererPackPin{}, false
	}
	return rendererPackPin{declaration.ProfileID, declaration.TreeSHA256, "renderer-profile:" + declaration.ToolID, registryName}, true
}

func normalizeRendererToolProfiles(profiles []string) ([]string, error) {
	seen := make(map[string]bool, len(profiles))
	result := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		if profile != "blender" && profile != "ffmpeg-video" {
			return nil, fmt.Errorf("unsupported renderer tool profile %q", profile)
		}
		if seen[profile] {
			return nil, fmt.Errorf("renderer_tool_profiles repeats %q", profile)
		}
		seen[profile] = true
		result = append(result, profile)
	}
	sort.Strings(result)
	return result, nil
}

func deferredResourceTargets(resource, root string, platform resourcedeployment.Platform) []DeferredResourceTarget {
	manifest, err := loadResourceArtifactManifest(filepath.Join(root, "resources", resource, "resource.json"))
	if err != nil || manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return nil
	}
	factsOS := platform.OS
	if factsOS == "macos" {
		factsOS = "darwin"
	}
	deferred := make([]DeferredResourceTarget, 0)
	for targetIndex, target := range manifest.ManagedService.Acquisition.Targets {
		if binaryfetch.UsesOnlyBuildTimeFacts(target) {
			continue
		}
		whenOS := strings.TrimSpace(target.When["os"])
		whenArch := strings.TrimSpace(target.When["arch"])
		if whenOS != "" && whenOS != factsOS || whenArch != "" && whenArch != platform.Arch {
			continue
		}
		absent := make([]string, 0)
		for fact := range target.When {
			if fact != "os" && fact != "arch" {
				absent = append(absent, fact)
			}
		}
		sort.Strings(absent)
		deferred = append(deferred, DeferredResourceTarget{
			TargetIndex: targetIndex,
			Resource:    resource, OS: platform.OS, Architecture: platform.Arch, When: cloneStringMap(target.When),
			Kind: manifest.ManagedService.Acquisition.EffectiveKind(target), URL: target.URL, Image: target.Image,
			SHA256: target.SHA256, ArtifactSHA256: target.ArtifactSHA256, Archive: target.Archive,
			Layout: target.Layout, BinPath: target.BinPath, Mode: target.Mode,
			AbsentFacts: absent,
		})
	}
	return deferred
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copy := make(map[string]string, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return copy
}

func resolveDesktopHostRequirements(root, scenarioPath string, resources []string, platform resourcedeployment.Platform, toolArtifactRoot string, trustMode resourcedeployment.ArtifactTrustMode) ([]HostRequirementPlanItem, error) {
	// Unit-level artifact fixtures deliberately contain only the resource and
	// scenario contracts. A real pipeline always has the repository service
	// manifest; do not make those isolated fixture plans invent host state.
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "service.json")); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat repository service manifest: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve desktop host home: %w", err)
	}
	resolution, err := hostreq.Resolve(root, home, hostreq.ResolveOptions{
		ExcludeRoot:   true,
		Environment:   "production",
		When:          "develop",
		Resources:     strings.Join(resources, ","),
		ScenarioPaths: []string{scenarioPath},
		Platform:      platform.OS,
		Architecture:  platform.Arch,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve desktop host requirements: %w", err)
	}
	items := make([]HostRequirementPlanItem, 0, len(resolution.Tools)+len(resolution.Safeguards))
	toolChecksums := map[string]string(nil)
	if toolArtifactRoot != "" {
		manifest, _, err := resourcedeployment.VerifyReleaseDirectory(toolArtifactRoot, trustMode, filepath.Join(root, "install", "vrooli-release.pub"))
		if err != nil {
			return nil, fmt.Errorf("verify tool artifact release: %w", err)
		}
		toolChecksums = make(map[string]string, len(manifest.Artifacts))
		for _, artifact := range manifest.Artifacts {
			toolChecksums[artifact.Name] = artifact.SHA256
		}
	}
	appendRequirement := func(requirement hostreq.ResolvedRequirement) {
		present := false
		artifactName := ""
		if requirement.Bundling == "vendorable" && toolArtifactRoot != "" {
			if requirement.Acquisition != nil {
				factsOS := platform.OS
				if factsOS == "macos" {
					factsOS = "darwin"
				}
				_, resolveErr := requirement.Acquisition.Resolve(binaryfetch.Facts{"os": factsOS, "arch": platform.Arch})
				if resolveErr == nil {
					artifactName = toolArtifactName(requirement.Name, platform)
					_, statErr := os.Stat(filepath.Join(toolArtifactRoot, artifactName))
					present = statErr == nil
					if present && toolChecksums != nil {
						expected, ok := toolChecksums[artifactName]
						actual, digestErr := artifactDigest(filepath.Join(toolArtifactRoot, artifactName))
						present = ok && digestErr == nil && strings.EqualFold(expected, actual)
					}
				}
			}
		}
		probeKnown := true
		currentOS := hostreq.CurrentPlatform()
		if currentOS == "darwin" {
			currentOS = "macos"
		}
		if requirement.Bundling == "host-required" && platform.OS == currentOS {
			_, lookupErr := exec.LookPath(requirement.Name)
			present = lookupErr == nil
		} else if requirement.Bundling == "host-required" {
			probeKnown = false
		}
		eligibility := hostreq.EvaluateEligibility(requirement, hostreq.TierDesktop, platform.OS, platform.Arch, present, probeKnown)
		provenance := make([]string, 0, len(requirement.Provenance))
		for _, source := range requirement.Provenance {
			provenance = append(provenance, source.Kind+":"+source.Name+":"+source.Source)
		}
		items = append(items, HostRequirementPlanItem{Name: requirement.Name, Kind: string(requirement.Kind), OS: platform.OS, Architecture: platform.Arch, Privilege: string(requirement.Privilege), Bundling: string(requirement.Bundling), Required: requirement.Required, Verdict: string(eligibility.Verdict), Reason: eligibility.Reason, Artifact: artifactName, Provenance: provenance})
	}
	for _, requirement := range resolution.Tools {
		appendRequirement(requirement)
	}
	for _, requirement := range resolution.Safeguards {
		appendRequirement(requirement)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Kind+items[i].Name+items[i].OS < items[j].Kind+items[j].Name+items[j].OS
	})
	return items, nil
}

func loadReleaseManifestArtifacts(root string, mode resourcedeployment.ArtifactTrustMode, publicKeyPath string) (map[string]string, error) {
	m, _, err := resourcedeployment.VerifyReleaseDirectory(root, mode, publicKeyPath)
	if err != nil {
		return nil, err
	}
	checksums := make(map[string]string, len(m.Artifacts))
	for _, a := range m.Artifacts {
		checksums[a.Name] = a.SHA256
	}
	return checksums, nil
}

func resolveResourceForTarget(root, requested, candidate string, alternatives []string, platform resourcedeployment.Platform, seen map[string]bool) (ResourceDeploymentPlanItem, bool, error) {
	if seen[candidate] {
		return ResourceDeploymentPlanItem{}, false, fmt.Errorf("resource fallback cycle while resolving %s", requested)
	}
	seen[candidate] = true
	manifest, err := loadResourceArtifactManifest(filepath.Join(root, "resources", candidate, "resource.json"))
	if err != nil {
		return ResourceDeploymentPlanItem{}, false, fmt.Errorf("load deployment contract for resource %q: %w", candidate, err)
	}
	item, bundled, applicable, err := resourcePlanItem(requested, candidate, &manifest, platform)
	if err != nil {
		return ResourceDeploymentPlanItem{}, false, err
	}
	if applicable {
		return item, bundled, nil
	}
	reason := unsupportedResourceReason(&manifest, platform)
	for _, fallback := range alternatives {
		if fallback == candidate {
			continue
		}
		item, bundled, err := resolveResourceForTarget(root, requested, fallback, nil, platform, seen)
		if err == nil {
			item.SelectedFallback = &ResourceDeploymentFallback{Resource: fallback, Reason: reason}
			return item, bundled, nil
		}
	}
	// Unsupported targets are a packaging limitation, not an admission failure.
	// The bundle records the reason so the runtime can refuse the missing
	// required service with an explanation instead of leaving the operator with
	// an artifact that appears universally deployable.
	return unsupportedResourcePlanItem(requested, candidate, &manifest, platform, reason), false, nil
}

func resourcePlanItem(requested, candidate string, manifest *resourceArtifactManifest, platform resourcedeployment.Platform) (ResourceDeploymentPlanItem, bool, bool, error) {
	target, found := manifest.Deployment.ResolveTarget("desktop", platform)
	if !found || target.Support == "unsupported" {
		return ResourceDeploymentPlanItem{}, false, false, nil
	}
	item := ResourceDeploymentPlanItem{RequestedResource: requested, Resource: candidate, OS: platform.OS, Architecture: platform.Arch, Mode: target.Mode, Support: target.Support, Privilege: manifest.Privilege, Bundling: manifest.Bundling, Eligibility: "eligible", Requires: target.Requires, Limitations: target.Limitations, Evidence: target.Evidence}
	if !strings.HasPrefix(target.Mode, "bundled-") {
		return item, false, true, nil
	}
	if err := addBundledResourceArtifacts(&item, candidate, target.Mode, manifest, platform); err != nil {
		return ResourceDeploymentPlanItem{}, false, false, err
	}
	return item, true, true, nil
}

func unsupportedResourcePlanItem(requested, candidate string, manifest *resourceArtifactManifest, platform resourcedeployment.Platform, reason string) ResourceDeploymentPlanItem {
	target, found := manifest.Deployment.ResolveTarget("desktop", platform)
	limitations := []string{reason}
	if found {
		limitations = append(limitations, target.Limitations...)
	}
	return ResourceDeploymentPlanItem{
		RequestedResource: requested,
		Resource:          candidate,
		OS:                platform.OS,
		Architecture:      platform.Arch,
		Mode:              targetMode(target, found),
		Support:           "unsupported",
		Privilege:         manifest.Privilege,
		Bundling:          manifest.Bundling,
		Eligibility:       "ineligible",
		EligibilityReason: reason,
		Limitations:       sortedUniqueStrings(limitations),
	}
}

func targetMode(target resourcedeployment.Target, found bool) string {
	if !found {
		return ""
	}
	return target.Mode
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func unsupportedResourceReason(manifest *resourceArtifactManifest, platform resourcedeployment.Platform) string {
	target, found := manifest.Deployment.ResolveTarget("desktop", platform)
	if found && target.Reason != "" {
		return target.Reason
	}
	return "no desktop deployment profile covers this OS/architecture"
}

func addBundledResourceArtifacts(item *ResourceDeploymentPlanItem, candidate, mode string, manifest *resourceArtifactManifest, platform resourcedeployment.Platform) error {
	if manifest.CLI.Distribution == nil || manifest.CLI.Distribution.Kind != "prebuilt_artifact" {
		return fmt.Errorf("resource %s uses %s but has no prebuilt artifact contract", candidate, mode)
	}
	artifact, err := resourcedeployment.ArtifactName(manifest.CLI.Distribution.ArtifactName, platform.OS, platform.Arch)
	if err != nil {
		return fmt.Errorf("resolve artifact for %s: %w", candidate, err)
	}
	item.Artifact = artifact
	if mode != "bundled-service" {
		return nil
	}
	service, err := bundledServicePlan(manifest, candidate, platform)
	if err != nil {
		return err
	}
	item.Service = service
	return nil
}

func bundledServicePlan(manifest *resourceArtifactManifest, candidate string, platform resourcedeployment.Platform) (*ResourceDeploymentService, error) {
	if manifest.ManagedService == nil {
		return nil, fmt.Errorf("resource %s uses bundled-service but has no managed_service contract", candidate)
	}
	if err := manifest.ManagedService.ProviderPolicy.ValidateManagedServiceTargets(); err != nil {
		return nil, fmt.Errorf("resource %s has an invalid managed-service provider policy: %w", candidate, err)
	}
	artifact, err := manifest.ManagedService.Artifact.ForPlatform(platform.OS, platform.Arch)
	if err != nil {
		return nil, fmt.Errorf("resolve managed-service artifact for %s: %w", candidate, err)
	}
	name, err := artifact.BundleArtifactForPlatform(platform.OS, platform.Arch)
	if err != nil {
		return nil, fmt.Errorf("resolve bundled service artifact for %s: %w", candidate, err)
	}
	environment := cloneServiceEnvironment(manifest.ManagedService.Environment)
	if acquisition := manifest.ManagedService.Acquisition; acquisition != nil {
		target, err := acquisition.Resolve(binaryfetch.Facts{"os": platform.OS, "arch": platform.Arch})
		if err != nil {
			return nil, fmt.Errorf("resolve managed-service runtime environment for %s: %w", candidate, err)
		}
		if len(target.RuntimeEnv) > 0 && environment == nil {
			environment = make(map[string]string, len(target.RuntimeEnv))
		}
		for key, value := range target.RuntimeEnv {
			environment[key] = value
		}
	}
	return &ResourceDeploymentService{ProviderPolicy: manifest.ManagedService.ProviderPolicy, Artifact: name, Layout: artifact.Layout, EntryPath: artifact.EntryPath, Version: artifact.Version, SHA256: artifact.SHA256, Arguments: append([]string(nil), manifest.ManagedService.Arguments...), Environment: environment, Bootstrap: cloneServiceBootstrap(manifest.ManagedService.Bootstrap), Config: cloneServiceConfig(manifest.ManagedService.Config), Ports: append([]ResourceDeploymentServicePort(nil), manifest.Ports...), HealthChecks: append([]ResourceDeploymentHealthCheck(nil), manifest.HealthChecks...)}, nil
}

func cloneServiceBootstrap(bootstrap *resourcedeployment.ServiceBootstrap) *resourcedeployment.ServiceBootstrap {
	if bootstrap == nil {
		return nil
	}
	clone := *bootstrap
	clone.Arguments = append([]string(nil), bootstrap.Arguments...)
	return &clone
}

func cloneServiceConfig(config *resourcedeployment.ServiceConfig) *resourcedeployment.ServiceConfig {
	if config == nil {
		return nil
	}
	copy := *config
	return &copy
}

func cloneServiceEnvironment(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func resolveArtifactFiles(root string, checksums map[string]string, item *ResourceDeploymentPlanItem) error {
	files, err := resourcedeployment.ArtifactFiles(item.Artifact)
	if err != nil {
		return err
	}
	for _, name := range files {
		expected, ok := checksums[name]
		if !ok {
			return fmt.Errorf("signed release checksum is missing resource artifact %s", name)
		}
		actual, err := artifactDigest(filepath.Join(root, name))
		if err != nil {
			return fmt.Errorf("hash resource artifact %s: %w", name, err)
		}
		if !strings.EqualFold(expected, actual) {
			return fmt.Errorf("resource artifact checksum mismatch for %s", name)
		}
		item.Files = append(item.Files, ResourceDeploymentArtifact{Name: name, SHA256: actual})
	}
	if item.Service == nil {
		return nil
	}
	if !resourcedeployment.IsSafeArtifactName(item.Service.Artifact) {
		return fmt.Errorf("resource %s has unsafe bundled service artifact", item.Resource)
	}
	expected, ok := checksums[item.Service.Artifact]
	if !ok {
		return fmt.Errorf("signed release checksum is missing bundled service artifact %s", item.Service.Artifact)
	}
	actual, err := artifactDigest(filepath.Join(root, item.Service.Artifact))
	if err != nil {
		return fmt.Errorf("hash bundled service artifact %s: %w", item.Service.Artifact, err)
	}
	if !strings.EqualFold(expected, actual) || !strings.EqualFold(item.Service.SHA256, actual) {
		return fmt.Errorf("bundled service artifact checksum mismatch for %s", item.Service.Artifact)
	}
	item.Service.Files = append(item.Service.Files, ResourceDeploymentArtifact{Name: item.Service.Artifact, SHA256: actual})
	return nil
}

// stageBundledResourceArtifacts copies exactly the artifacts selected by the
// resolved plan. It never re-reads manifests, chooses another architecture, or
// makes a fallback decision after the gate.
func stageBundledResourceArtifacts(bundleDir, artifactRoot string, plan *ResourceDeploymentPlan) ([]string, error) {
	if plan == nil || len(plan.Resources) == 0 {
		return nil, nil
	}
	var copied []string
	for _, item := range plan.Resources {
		for _, file := range item.Files {
			if !resourcedeployment.IsSafeArtifactName(file.Name) {
				return nil, fmt.Errorf("unsafe resource artifact %q", file.Name)
			}
			destination := filepath.Join(bundleDir, "resources", item.Resource, file.Name)
			if err := copyArtifact(filepath.Join(artifactRoot, file.Name), destination); err != nil {
				return nil, err
			}
			copied = append(copied, destination)
		}
		if item.Service != nil {
			for _, file := range item.Service.Files {
				if !resourcedeployment.IsSafeArtifactName(file.Name) {
					return nil, fmt.Errorf("unsafe bundled service artifact %q", file.Name)
				}
				destination := filepath.Join(bundleDir, "resources", item.Resource, file.Name)
				if err := copyArtifact(filepath.Join(artifactRoot, file.Name), destination); err != nil {
					return nil, err
				}
				copied = append(copied, destination)
			}
		}
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serialize resource deployment plan: %w", err)
	}
	path := filepath.Join(bundleDir, "resource-deployment-plan.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write resource deployment plan: %w", err)
	}
	return append(copied, path), nil
}

// stageBundledToolArtifacts copies only vendorable tools admitted by the
// target-specific plan. Safeguards and host-required tools never enter the
// bundle, even when a similarly named file exists in the release directory.
func stageBundledToolArtifacts(bundleDir, artifactRoot string, plan *ResourceDeploymentPlan) ([]string, error) {
	if plan == nil || artifactRoot == "" {
		return nil, nil
	}
	var copied []string
	for _, requirement := range plan.HostRequirements {
		if requirement.Bundling != "vendorable" || requirement.Artifact == "" || requirement.Verdict != string(hostreq.EligibilityEligible) {
			continue
		}
		if !resourcedeployment.IsSafeArtifactName(requirement.Artifact) {
			return nil, fmt.Errorf("unsafe tool artifact %q", requirement.Artifact)
		}
		destination := filepath.Join(bundleDir, "tools", requirement.Artifact)
		if err := copyArtifact(filepath.Join(artifactRoot, requirement.Artifact), destination); err != nil {
			return nil, err
		}
		copied = append(copied, destination)
	}
	return copied, nil
}

func stageSelectedRendererPacks(bundleDir, manifestPath, artifactRoot, publicKeyPath string, plan *ResourceDeploymentPlan) ([]string, error) {
	return stageSelectedRendererPacksWithPins(bundleDir, manifestPath, artifactRoot, publicKeyPath, plan, rendererPin)
}

func stageSelectedRendererPacksWithPins(bundleDir, manifestPath, artifactRoot, publicKeyPath string, plan *ResourceDeploymentPlan, pinFor func(string) (rendererPackPin, bool)) ([]string, error) {
	if plan == nil || len(plan.RendererPacks) == 0 {
		return nil, nil
	}
	if artifactRoot == "" || manifestPath == "" {
		return nil, fmt.Errorf("selected renderer packs require a tool artifact root and packaged bundle manifest")
	}
	release, _, err := resourcedeployment.VerifyReleaseDirectory(artifactRoot, plan.ArtifactTrustMode, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reverify renderer release before staging: %w", err)
	}
	byName := make(map[string]resourcedeployment.ReleaseArtifact, len(release.Artifacts))
	for _, artifact := range release.Artifacts {
		byName[artifact.Name] = artifact
	}
	for _, item := range plan.RendererPacks {
		pin, ok := pinFor(item.ToolID)
		if !ok || item.ProfileID != pin.profileID || item.TreeSHA256 != pin.treeSHA256 || item.Role != pin.role || item.OS != "linux" || item.Architecture != "amd64" || item.Artifact != toolArtifactName(pin.registryName, resourcedeployment.Platform{OS: "linux", Arch: "amd64"}) || !resourcedeployment.IsSafeArtifactName(item.Artifact) {
			return nil, fmt.Errorf("renderer plan contains an unsupported package")
		}
		artifact, ok := byName[item.Artifact]
		if !ok || artifact.Role != item.Role || artifact.OS != item.OS || artifact.Arch != item.Architecture || !strings.EqualFold(artifact.SHA256, item.TreeSHA256) {
			return nil, fmt.Errorf("renderer release artifact %q changed from the resolved plan", item.Artifact)
		}
	}
	toolDir := filepath.Join(bundleDir, "tools")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return nil, err
	}
	var copied []string
	for _, item := range plan.RendererPacks {
		source := filepath.Join(artifactRoot, item.Artifact)
		info, err := os.Lstat(source)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("renderer source %q is not a real directory", item.Artifact)
		}
		destination := filepath.Join(toolDir, item.ToolID)
		if err := stageRendererDirectory(source, destination, item.TreeSHA256); err != nil {
			return nil, fmt.Errorf("stage renderer %q without replacing existing output: %w", item.ToolID, err)
		}
		copied = append(copied, destination)
	}
	if err := addRendererDeclarations(manifestPath, plan.RendererPacks); err != nil {
		return nil, err
	}
	return copied, nil
}

func stageRendererDirectory(source, destination, expectedTree string) error {
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if info, err := os.Lstat(parent); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("renderer destination parent must be a real directory")
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("destination already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("renderer source must be a real directory")
	}
	actual, err := binaryfetch.TreeDigest(source)
	if err != nil || !strings.EqualFold(actual, expectedTree) {
		return fmt.Errorf("renderer source tree does not match expected digest")
	}
	temp, err := os.MkdirTemp(parent, ".renderer-stage-")
	if err != nil {
		return err
	}
	if err := os.Remove(temp); err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	if err := copyRendererTree(source, temp); err != nil {
		return err
	}
	actual, err = binaryfetch.TreeDigest(temp)
	if err != nil || !strings.EqualFold(actual, expectedTree) {
		return fmt.Errorf("staged renderer tree failed verification")
	}
	return renameDirectoryNoReplace(temp, destination)
}

func copyRendererTree(source, destination string) error {
	rootInfo, err := os.Lstat(source)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("renderer source must be a real directory")
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(destination, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case info.IsDir():
			return os.Mkdir(target, 0o755)
		case info.Mode().IsRegular():
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
			if err != nil {
				in.Close()
				return err
			}
			_, copyErr := io.Copy(out, in)
			closeInErr, closeOutErr := in.Close(), out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeInErr != nil {
				return closeInErr
			}
			return closeOutErr
		default:
			return fmt.Errorf("renderer tree contains unsupported file type at %q", rel)
		}
	})
}

func addRendererDeclarations(path string, packs []RendererPackPlanItem) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("packaged bundle manifest is not a regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse packaged bundle manifest: %w", err)
	}
	decls := make([]bundlemanifest.ToolProfilePackage, 0, len(packs))
	if existing := document["tool_profiles"]; len(existing) > 0 {
		if err := json.Unmarshal(existing, &decls); err != nil {
			return fmt.Errorf("parse existing bundle renderer declarations: %w", err)
		}
	}
	if err := bundlemanifest.ValidateToolProfiles(decls); err != nil {
		return fmt.Errorf("validate existing bundle tool declarations: %w", err)
	}
	byID := make(map[string]bundlemanifest.ToolProfilePackage, len(decls)+len(packs))
	for _, declared := range decls {
		byID[declared.ToolID] = declared
	}
	for _, item := range packs {
		wanted := bundlemanifest.ToolProfilePackage{ToolID: item.ToolID, ProfileID: item.ProfileID, OS: item.OS, Architecture: item.Architecture, TreeSHA256: item.TreeSHA256}
		if existing, ok := byID[item.ToolID]; ok && existing != wanted {
			return fmt.Errorf("bundle manifest renderer declaration %q conflicts with selected profile", item.ToolID)
		}
		byID[item.ToolID] = wanted
	}
	decls = decls[:0]
	for _, declared := range byID {
		decls = append(decls, declared)
	}
	if err := bundlemanifest.ValidateToolProfiles(decls); err != nil {
		return fmt.Errorf("validate packaged bundle tool declarations: %w", err)
	}
	sort.Slice(decls, func(i, j int) bool { return decls[i].ToolID < decls[j].ToolID })
	raw, err := json.Marshal(decls)
	if err != nil {
		return err
	}
	document["tool_profiles"] = raw
	updated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".bundle-manifest-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(append(updated, '\n')); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func requiredScenarioResources(path string) ([]string, map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read scenario service manifest: %w", err)
	}
	var service struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled  bool `json:"enabled"`
				Required bool `json:"required"`
			} `json:"resources"`
		} `json:"dependencies"`
		Deployment struct {
			Dependencies struct {
				Resources map[string]struct {
					PlatformSupport map[string]struct {
						Alternatives []string `json:"alternatives"`
					} `json:"platform_support"`
					SwappableWith []struct {
						ID string `json:"id"`
					} `json:"swappable_with"`
				} `json:"resources"`
			} `json:"dependencies"`
		} `json:"deployment"`
	}
	if err := json.Unmarshal(data, &service); err != nil {
		return nil, nil, fmt.Errorf("parse scenario service manifest: %w", err)
	}
	var resources []string
	fallbacks := map[string][]string{}
	for name, dep := range service.Dependencies.Resources {
		if !dep.Enabled || !dep.Required {
			continue
		}
		resources = append(resources, name)
		meta := service.Deployment.Dependencies.Resources[name]
		fallbacks[name] = append(fallbacks[name], meta.PlatformSupport["tier-2-desktop"].Alternatives...)
		for _, swap := range meta.SwappableWith {
			fallbacks[name] = append(fallbacks[name], swap.ID)
		}
	}
	sort.Strings(resources)
	return resources, fallbacks, nil
}

func loadResourceArtifactManifest(path string) (resourceArtifactManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return resourceArtifactManifest{}, err
	}
	var manifest resourceArtifactManifest
	return manifest, json.Unmarshal(data, &manifest)
}

func artifactDigest(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return binaryfetch.TreeDigest(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

func copyArtifact(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("replace artifact destination %q: %w", dst, err)
		}
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			target := filepath.Join(dst, rel)
			if entry.Type()&os.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				if existing, err := os.Lstat(target); err == nil {
					if existing.Mode()&os.ModeSymlink == 0 {
						return fmt.Errorf("artifact destination %q exists and is not a symlink", target)
					}
					existingLink, err := os.Readlink(target)
					if err != nil {
						return err
					}
					if existingLink != link {
						return fmt.Errorf("artifact destination %q links to %q, want %q", target, existingLink, link)
					}
					return nil
				} else if !os.IsNotExist(err) {
					return err
				}
				return os.Symlink(link, target)
			}
			if entry.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fileInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if existing, err := os.Lstat(target); err == nil {
				if !existing.Mode().IsRegular() {
					return fmt.Errorf("artifact destination %q is not a regular file", target)
				}
				if err := os.Chmod(target, 0o600); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}
			if err := os.WriteFile(target, data, fileInfo.Mode().Perm()); err != nil {
				return err
			}
			if err := os.Chmod(target, fileInfo.Mode().Perm()); err != nil {
				return err
			}
			return nil
		})
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(link, dst)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

func toolArtifactName(name string, platform resourcedeployment.Platform) string {
	osName := platform.OS
	if osName == "macos" {
		osName = "darwin"
	}
	suffix := ""
	if osName == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("tool_%s_%s_%s%s", name, osName, platform.Arch, suffix)
}

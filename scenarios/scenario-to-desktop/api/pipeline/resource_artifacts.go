package pipeline

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ResourceDeploymentPlan is an immutable per-target deployment decision. It
// is resolved before bundle packaging, staged verbatim, and consumed by the
// desktop runtime. A resource is never silently omitted from a target.
type ResourceDeploymentPlan struct {
	SchemaVersion string                       `json:"schema_version"`
	Resources     []ResourceDeploymentPlanItem `json:"resources"`
}

type ResourceDeploymentPlanItem struct {
	RequestedResource string                       `json:"requested_resource"`
	Resource          string                       `json:"resource"`
	OS                string                       `json:"os"`
	Architecture      string                       `json:"architecture"`
	Mode              string                       `json:"mode"`
	Support           string                       `json:"support"`
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
	ProviderPolicy resourcedeployment.ProviderPolicy `json:"provider_policy"`
	Artifact       string                            `json:"artifact"`
	Version        string                            `json:"version"`
	SHA256         string                            `json:"sha256"`
	Arguments      []string                          `json:"arguments,omitempty"`
	Environment    map[string]string                 `json:"environment,omitempty"`
	Config         *resourcedeployment.ServiceConfig `json:"config,omitempty"`
	Ports          []ResourceDeploymentServicePort   `json:"ports,omitempty"`
	HealthChecks   []ResourceDeploymentHealthCheck   `json:"health_checks,omitempty"`
	Files          []ResourceDeploymentArtifact      `json:"files"`
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
	CLI struct {
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

// resolveResourceDeploymentPlan makes every resource selection before any
// bundle output is written. The artifact hashes are resolved here as well so
// staging cannot change the deployment choice after the gate has passed.
func resolveResourceDeploymentPlan(scenarioPath, artifactRoot string, platformInputs []string) (*ResourceDeploymentPlan, error) {
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
	plan := &ResourceDeploymentPlan{SchemaVersion: "v2"}
	for _, requested := range required {
		for _, platform := range platforms {
			item, bundled, err := resolveResourceForTarget(root, requested, requested, fallbacks[requested], platform, map[string]bool{})
			if err != nil {
				return nil, err
			}
			if bundled {
				if artifactRoot == "" {
					return nil, fmt.Errorf("resource %s uses %s on %s but resource_artifact_root is not set", item.Resource, item.Mode, platform.String())
				}
				if checksums == nil {
					checksums, err = loadReleaseChecksums(artifactRoot, filepath.Join(root, "install", "vrooli-release.pub"))
					if err != nil {
						return nil, err
					}
				}
				if err := resolveArtifactFiles(artifactRoot, checksums, &item); err != nil {
					return nil, err
				}
			}
			plan.Resources = append(plan.Resources, item)
		}
	}
	sort.Slice(plan.Resources, func(i, j int) bool {
		if plan.Resources[i].RequestedResource == plan.Resources[j].RequestedResource {
			return plan.Resources[i].OS+plan.Resources[i].Architecture < plan.Resources[j].OS+plan.Resources[j].Architecture
		}
		return plan.Resources[i].RequestedResource < plan.Resources[j].RequestedResource
	})
	return plan, nil
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
	return ResourceDeploymentPlanItem{}, false, fmt.Errorf("resource %s cannot deploy on %s: %s", requested, platform.String(), reason)
}

func resourcePlanItem(requested, candidate string, manifest *resourceArtifactManifest, platform resourcedeployment.Platform) (ResourceDeploymentPlanItem, bool, bool, error) {
	target, found := manifest.Deployment.ResolveTarget("desktop", platform)
	if !found || target.Support == "unsupported" {
		return ResourceDeploymentPlanItem{}, false, false, nil
	}
	item := ResourceDeploymentPlanItem{RequestedResource: requested, Resource: candidate, OS: platform.OS, Architecture: platform.Arch, Mode: target.Mode, Support: target.Support, Requires: target.Requires, Limitations: target.Limitations, Evidence: target.Evidence}
	if !strings.HasPrefix(target.Mode, "bundled-") {
		return item, false, true, nil
	}
	if err := addBundledResourceArtifacts(&item, candidate, target.Mode, manifest, platform); err != nil {
		return ResourceDeploymentPlanItem{}, false, false, err
	}
	return item, true, true, nil
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
	return &ResourceDeploymentService{ProviderPolicy: manifest.ManagedService.ProviderPolicy, Artifact: name, Version: artifact.Version, SHA256: artifact.SHA256, Arguments: append([]string(nil), manifest.ManagedService.Arguments...), Environment: cloneServiceEnvironment(manifest.ManagedService.Environment), Config: cloneServiceConfig(manifest.ManagedService.Config), Ports: append([]ResourceDeploymentServicePort(nil), manifest.Ports...), HealthChecks: append([]ResourceDeploymentHealthCheck(nil), manifest.HealthChecks...)}, nil
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
		actual, err := sha256File(filepath.Join(root, name))
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
	actual, err := sha256File(filepath.Join(root, item.Service.Artifact))
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

func loadReleaseChecksums(root, trustedPublicKeyPath string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		return nil, fmt.Errorf("read signed release checksums: %w", err)
	}
	if err := verifyReleaseChecksumSignature(data, filepath.Join(root, "SHA256SUMS.sig"), trustedPublicKeyPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("release artifact root is unsigned: Vrooli release signing authority must provide SHA256SUMS.sig for this exact SHA256SUMS manifest")
		}
		return nil, err
	}
	checksums := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && len(fields[0]) == 64 && resourcedeployment.IsSafeArtifactName(fields[1]) {
			checksums[fields[1]] = fields[0]
		}
	}
	if len(checksums) == 0 {
		return nil, fmt.Errorf("release checksum manifest has no entries")
	}
	return checksums, nil
}

func verifyReleaseChecksumSignature(checksums []byte, signaturePath, publicKeyPath string) error {
	signatureText, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("read release checksum signature: %w", err)
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signatureText)))
	if err != nil {
		return fmt.Errorf("decode release checksum signature: %w", err)
	}
	pemData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("read trusted release public key: %w", err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("parse trusted release public key: invalid PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse trusted release public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("trusted release public key is not RSA")
	}
	digest := sha256.Sum256(checksums)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("verify release checksum signature: %w", err)
	}
	return nil
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func copyArtifact(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

// Package resources loads and validates the resolved resource deployment plan
// included in a desktop bundle. It deliberately does not re-resolve manifests:
// runtime executes the exact, signed selection admitted by the pipeline.
package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type Plan struct {
	SchemaVersion     string                   `json:"schema_version"`
	ArtifactTrustMode string                   `json:"artifact_trust_mode,omitempty"`
	Promotable        bool                     `json:"promotable"`
	Resources         []Item                   `json:"resources"`
	DeferredTargets   []DeferredResourceTarget `json:"deferred_targets,omitempty"`
}

// DeferredResourceTarget is a hardware-dependent candidate retained by the
// packager. The staged resource remains the fallback until an explicit
// first-run upgrade outcome is recorded.
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

type UpgradeOffer struct {
	Resource  string `json:"resource"`
	Kind      string `json:"kind"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Reason    string `json:"reason"`
}

type UpgradeOutcome struct {
	Resource   string `json:"resource"`
	Decision   string `json:"decision"` // accepted, declined, failed
	Reason     string `json:"reason,omitempty"`
	ObservedAt string `json:"observed_at"`
}

// ApplyDeferredUpgrade acquires a matching candidate into a temporary sibling
// and atomically replaces the selected service tree only after verification.
// The caller records the outcome after this returns; any error leaves the
// shipped artifact untouched.
func ApplyDeferredUpgrade(ctx context.Context, bundleRoot string, plan *Plan, resource string, facts binaryfetch.Facts) error {
	if plan == nil {
		return fmt.Errorf("resource upgrade plan is unavailable")
	}
	var candidate *DeferredResourceTarget
	var resolved binaryfetch.AcquisitionTarget
	var resolveErr error
	for i := range plan.DeferredTargets {
		current := &plan.DeferredTargets[i]
		if current.Resource != resource || current.OS != runtimeOS() || current.Architecture != runtime.GOARCH {
			continue
		}
		candidateAcquisition := binaryfetch.Acquisition{Kind: current.Kind, Targets: []binaryfetch.AcquisitionTarget{{When: current.When, Kind: current.Kind, URL: current.URL, Image: current.Image, SHA256: current.SHA256, ArtifactSHA256: current.ArtifactSHA256, Archive: current.Archive, Layout: current.Layout, BinPath: current.BinPath, Mode: current.Mode}}}
		resolved, resolveErr = candidateAcquisition.Resolve(facts)
		if resolveErr == nil {
			candidate = current
			break
		}
	}
	if candidate == nil {
		if resolveErr != nil {
			return fmt.Errorf("deferred candidate no longer matches: %w", resolveErr)
		}
		return fmt.Errorf("no deferred candidate for resource %q", resource)
	}
	var item *Item
	for i := range plan.Resources {
		if plan.Resources[i].Resource == resource {
			item = &plan.Resources[i]
			break
		}
	}
	if item == nil || item.Service == nil {
		return fmt.Errorf("resource %q has no bundled service fallback", resource)
	}
	resourceDir := filepath.Join(bundleRoot, "resources", resource)
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(resourceDir, ".upgrade-staging")
	_ = os.RemoveAll(tmp)
	defer os.RemoveAll(tmp)
	layout := strings.ToLower(strings.TrimSpace(resolved.Layout))
	kind := strings.ToLower(strings.TrimSpace(resolved.Kind))
	if kind == "oci-image" {
		if layout == "dir" {
			if _, err := binaryfetch.FetchOCI(ctx, resolved, tmp, nil); err != nil {
				return fmt.Errorf("acquire deferred resource %q: %w", resource, err)
			}
		} else {
			if _, err := binaryfetch.FetchOCIFile(ctx, resolved, tmp, nil); err != nil {
				return fmt.Errorf("acquire deferred resource %q: %w", resource, err)
			}
		}
	} else if kind == "url" {
		spec := binaryfetch.Target{Name: "upgrade", URL: resolved.URL, SHA256: resolved.SHA256, Archive: resolved.Archive, Layout: layout, BinPath: resolved.BinPath, Mode: resolved.Mode}
		if layout == "dir" {
			if _, err := binaryfetch.FetchDir(ctx, spec, tmp, nil); err != nil {
				return fmt.Errorf("acquire deferred resource %q: %w", resource, err)
			}
		} else {
			if _, err := binaryfetch.Fetch(ctx, spec, resourceDir, nil); err != nil {
				return fmt.Errorf("acquire deferred resource %q: %w", resource, err)
			}
			if err := os.Rename(filepath.Join(resourceDir, "upgrade"), tmp); err != nil {
				return fmt.Errorf("stage deferred resource %q: %w", resource, err)
			}
		}
	} else {
		return fmt.Errorf("resource upgrade %q uses unsupported acquisition kind %q", resource, kind)
	}
	actual, err := serviceArtifactDigest(tmp, layout)
	if err != nil {
		return fmt.Errorf("digest deferred resource %q: %w", resource, err)
	}
	if candidate.ArtifactSHA256 != "" && !strings.EqualFold(actual, candidate.ArtifactSHA256) {
		return fmt.Errorf("deferred resource %q artifact digest mismatch: got %s want %s", resource, actual, candidate.ArtifactSHA256)
	}
	finalPath := filepath.Join(resourceDir, item.Service.Artifact)
	backup := filepath.Join(resourceDir, ".upgrade-previous")
	_ = os.RemoveAll(backup)
	if err := os.Rename(finalPath, backup); err != nil {
		return fmt.Errorf("stage fallback for resource %q: %w", resource, err)
	}
	if err := os.Rename(tmp, finalPath); err != nil {
		_ = os.Rename(backup, finalPath)
		return fmt.Errorf("install deferred resource %q: %w", resource, err)
	}
	item.Service.Layout = "dir"
	item.Service.EntryPath = resolved.BinPath
	item.Service.SHA256 = actual
	item.Service.Files = []Artifact{{Name: item.Service.Artifact, SHA256: actual}}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		_ = os.RemoveAll(finalPath)
		_ = os.Rename(backup, finalPath)
		return err
	}
	planPath := filepath.Join(bundleRoot, "resource-deployment-plan.json")
	planTmp := planPath + ".upgrade-tmp"
	if err := os.WriteFile(planTmp, append(data, '\n'), 0o644); err != nil {
		_ = os.RemoveAll(finalPath)
		_ = os.Rename(backup, finalPath)
		return err
	}
	if err := os.Rename(planTmp, planPath); err != nil {
		_ = os.RemoveAll(planTmp)
		_ = os.RemoveAll(finalPath)
		_ = os.Rename(backup, finalPath)
		return err
	}
	_ = os.RemoveAll(backup)
	return nil
}

func ResolveDeferredOffers(plan *Plan, facts binaryfetch.Facts, outcomes map[string]UpgradeOutcome) []UpgradeOffer {
	if plan == nil {
		return nil
	}
	offers := make([]UpgradeOffer, 0)
	for _, candidate := range plan.DeferredTargets {
		if candidate.OS != runtimeOS() || candidate.Architecture != runtime.GOARCH {
			continue
		}
		if outcome, ok := outcomes[candidate.Resource]; ok && strings.TrimSpace(outcome.Decision) != "" {
			continue
		}
		acq := binaryfetch.Acquisition{Kind: candidate.Kind, Targets: []binaryfetch.AcquisitionTarget{{When: candidate.When, Kind: candidate.Kind, URL: candidate.URL, Image: candidate.Image, SHA256: candidate.SHA256, ArtifactSHA256: candidate.ArtifactSHA256, Archive: candidate.Archive, Layout: candidate.Layout, BinPath: candidate.BinPath, Mode: candidate.Mode}}}
		if _, err := acq.Resolve(facts); err != nil {
			continue
		}
		offers = append(offers, UpgradeOffer{Resource: candidate.Resource, Kind: candidate.Kind, SizeBytes: candidate.SizeBytes, Reason: "accelerated resource candidate matches detected host capabilities"})
	}
	return offers
}

func LoadUpgradeOutcomes(appData string) (map[string]UpgradeOutcome, error) {
	path := filepath.Join(appData, "resource-upgrade-outcomes.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]UpgradeOutcome{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read resource upgrade outcomes: %w", err)
	}
	var outcomes map[string]UpgradeOutcome
	if err := json.Unmarshal(data, &outcomes); err != nil {
		return nil, fmt.Errorf("parse resource upgrade outcomes: %w", err)
	}
	return outcomes, nil
}

func RecordUpgradeOutcome(appData string, outcome UpgradeOutcome) error {
	if strings.TrimSpace(outcome.Resource) == "" || strings.TrimSpace(outcome.Decision) == "" {
		return fmt.Errorf("resource upgrade outcome requires resource and decision")
	}
	switch outcome.Decision {
	case "accepted", "declined", "failed":
	default:
		return fmt.Errorf("resource upgrade outcome has unknown decision %q", outcome.Decision)
	}
	outcomes, err := LoadUpgradeOutcomes(appData)
	if err != nil {
		return err
	}
	if outcome.ObservedAt == "" {
		outcome.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	outcomes[outcome.Resource] = outcome
	data, err := json.MarshalIndent(outcomes, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(appData, 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(appData, ".resource-upgrade-outcomes.tmp")
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(appData, "resource-upgrade-outcomes.json"))
}

type Item struct {
	RequestedResource string     `json:"requested_resource"`
	Resource          string     `json:"resource"`
	OS                string     `json:"os"`
	Architecture      string     `json:"architecture"`
	Mode              string     `json:"mode"`
	Support           string     `json:"support"`
	Privilege         string     `json:"privilege"`
	Bundling          string     `json:"bundling"`
	Requires          []string   `json:"requires,omitempty"`
	Limitations       []string   `json:"limitations,omitempty"`
	Evidence          []string   `json:"evidence,omitempty"`
	SelectedFallback  *Fallback  `json:"selected_fallback,omitempty"`
	Artifact          string     `json:"artifact,omitempty"`
	Files             []Artifact `json:"files,omitempty"`
	Service           *Service   `json:"service,omitempty"`
}
type Fallback struct {
	Resource string `json:"resource"`
	Reason   string `json:"reason"`
}
type Artifact struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Service is the independently signed executable launched as the hidden
// server component of a bundled-service resource. It is deliberately not a
// controller artifact and must be verified separately before any supervisor
// receives it.
type Service struct {
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
	Ports          []ServicePort                        `json:"ports,omitempty"`
	HealthChecks   []HealthCheck                        `json:"health_checks,omitempty"`
	Files          []Artifact                           `json:"files"`
}

type HealthCheck struct {
	Type           string `json:"type"`
	Target         string `json:"target"`
	ExpectedStatus []int  `json:"expected_status,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type ServicePort struct {
	Name string `json:"name"`
	Host int    `json:"host"`
}

var findDocker = exec.LookPath

// Load validates all items selected for this runtime host. Bundled clients are
// ready for scenario services to invoke from the bundle; bundled services are
// verified here and later launched only by ServiceSupervisor from their
// explicit plan arguments and configuration contract.
func Load(bundleRoot string) (*Plan, error) {
	path := filepath.Join(bundleRoot, "resource-deployment-plan.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Plan{SchemaVersion: "v4"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read resource deployment plan: %w", err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse resource deployment plan: %w", err)
	}
	if plan.SchemaVersion != "v3" && plan.SchemaVersion != "v4" && plan.SchemaVersion != "v6" {
		return nil, fmt.Errorf("unsupported resource deployment plan version %q", plan.SchemaVersion)
	}
	for _, item := range plan.Resources {
		if err := verifyPlanItem(bundleRoot, item); err != nil {
			return nil, err
		}
	}
	return &plan, nil
}

func verifyPlanItem(bundleRoot string, item Item) error {
	if err := validateItem(item); err != nil {
		return err
	}
	if item.OS != runtimeOS() || item.Architecture != runtime.GOARCH {
		return nil
	}
	return verifyHostResource(bundleRoot, item)
}

func verifyHostResource(bundleRoot string, item Item) error {
	switch item.Mode {
	case "bundled-client":
		return verifyClient(bundleRoot, item)
	case "bundled-service":
		if err := verifyClient(bundleRoot, item); err != nil {
			return err
		}
		return verifyService(bundleRoot, item)
	case "docker-desktop":
		if _, err := findDocker("docker"); err != nil {
			return fmt.Errorf("resource %s requires Docker Desktop or Docker Engine before it can run: install and start Docker, then retry", item.Resource)
		}
	}
	return nil
}

func validateItem(item Item) error {
	if item.RequestedResource == "" || item.Resource == "" || item.OS == "" || item.Architecture == "" {
		return fmt.Errorf("resource deployment plan contains an incomplete resource selection")
	}
	if item.Support == "unsupported" {
		return fmt.Errorf("resolved resource %s is unsupported", item.RequestedResource)
	}
	if item.Privilege != "none" && item.Privilege != "user" && item.Privilege != "elevated" {
		return fmt.Errorf("resource %s has unknown privilege %q", item.Resource, item.Privilege)
	}
	if item.Bundling != "vendorable" && item.Bundling != "host-required" && item.Bundling != "prohibited" {
		return fmt.Errorf("resource %s has unknown bundling policy %q", item.Resource, item.Bundling)
	}
	switch item.Mode {
	case "bundled-client", "bundled-service", "docker-desktop", "native-host-tool", "remote-service", "manual":
	default:
		return fmt.Errorf("resource %s uses unknown deployment mode %q", item.Resource, item.Mode)
	}
	if item.SelectedFallback != nil && (item.SelectedFallback.Resource != item.Resource || item.SelectedFallback.Reason == "") {
		return fmt.Errorf("resource %s has an invalid selected fallback record", item.RequestedResource)
	}
	return nil
}

func verifyService(bundleRoot string, item Item) error {
	if item.Service == nil {
		return fmt.Errorf("bundled-service resource %s is missing its server artifact", item.Resource)
	}
	service := item.Service
	if err := validateServiceDeclaration(item.Resource, service); err != nil {
		return err
	}
	artifactPath := filepath.Join(bundleRoot, "resources", item.Resource, service.Artifact)
	actual, err := serviceArtifactDigest(artifactPath, service.Layout)
	if err != nil {
		return fmt.Errorf("read bundled service artifact %s: %w", service.Artifact, err)
	}
	if !strings.EqualFold(actual, service.SHA256) {
		return fmt.Errorf("bundled service artifact hash mismatch for %s", service.Artifact)
	}
	return nil
}

func validateServiceDeclaration(resource string, service *Service) error {
	if err := validateServicePolicy(resource, service); err != nil {
		return err
	}
	if err := validateServiceIdentity(resource, service); err != nil {
		return err
	}
	if err := validateServiceRuntime(resource, service); err != nil {
		return err
	}
	return validateServicePorts(resource, service.Ports)
}

func validateServicePolicy(resource string, service *Service) error {
	if err := service.ProviderPolicy.ValidateManagedServiceTargets(); err != nil {
		return fmt.Errorf("resource %s has invalid bundled service provider policy: %w", resource, err)
	}
	if _, err := service.ProviderPolicy.ResolveProvider(resourcedeployment.ProviderRequest{Target: resourcedeployment.ProviderTargetDesktopBundle}); err != nil {
		return fmt.Errorf("resource %s cannot resolve the desktop bundled provider: %w", resource, err)
	}
	return nil
}

func validateServiceIdentity(resource string, service *Service) error {
	if strings.TrimSpace(service.Version) == "" || !resourcedeployment.IsSafeArtifactName(service.Artifact) || len(service.SHA256) != sha256.Size*2 {
		return fmt.Errorf("resource %s has an invalid bundled service identity", resource)
	}
	switch service.Layout {
	case "file", "":
		if service.EntryPath != "" {
			return fmt.Errorf("resource %s has entry_path for file-layout service", resource)
		}
	case "dir":
		if !safeRelativePath(service.EntryPath) {
			return fmt.Errorf("resource %s has invalid directory service entry_path", resource)
		}
	default:
		return fmt.Errorf("resource %s has unknown bundled service layout %q", resource, service.Layout)
	}
	if len(service.Files) != 1 || service.Files[0].Name != service.Artifact || !strings.EqualFold(service.Files[0].SHA256, service.SHA256) {
		return fmt.Errorf("resource %s has an invalid bundled service file record", resource)
	}
	return nil
}

func serviceArtifactDigest(path, layout string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(strings.TrimSpace(layout), "dir") || info.IsDir() {
		return binaryfetch.TreeDigest(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func safeRelativePath(path string) bool {
	path = filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	return path != "." && path != "" && !filepath.IsAbs(path) && path != ".." && !strings.HasPrefix(path, ".."+string(filepath.Separator))
}

func validateServiceRuntime(resource string, service *Service) error {
	if service.Bootstrap != nil {
		if err := service.Bootstrap.Validate(); err != nil {
			return fmt.Errorf("resource %s has invalid bundled service bootstrap: %w", resource, err)
		}
	}
	if service.Config != nil {
		if err := service.Config.Validate(); err != nil {
			return fmt.Errorf("resource %s has invalid bundled service config: %w", resource, err)
		}
	}
	for _, check := range service.HealthChecks {
		if (check.Type != "http" && check.Type != "tcp") || strings.TrimSpace(check.Target) == "" {
			return fmt.Errorf("resource %s has an unsupported bundled service health check", resource)
		}
		for _, status := range check.ExpectedStatus {
			if status < 100 || status > 599 {
				return fmt.Errorf("resource %s has invalid bundled service health status", resource)
			}
		}
	}
	return nil
}

func validateServicePorts(resource string, ports []ServicePort) error {
	seen := map[string]bool{}
	for _, port := range ports {
		if strings.TrimSpace(port.Name) == "" || port.Host <= 0 || port.Host > 65535 || seen[port.Name] {
			return fmt.Errorf("resource %s has an invalid bundled service port", resource)
		}
		seen[port.Name] = true
	}
	return nil
}

func runtimeOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func verifyClient(bundleRoot string, item Item) error {
	if !resourcedeployment.IsSafeArtifactName(item.Artifact) {
		return fmt.Errorf("resource %s has unsafe artifact identity", item.Resource)
	}
	if len(item.Files) != 3 {
		return fmt.Errorf("resource %s must include binary, manifest, and build metadata", item.Resource)
	}
	if err := verifyClientFiles(bundleRoot, item); err != nil {
		return err
	}
	if err := verifyClientContract(bundleRoot, item); err != nil {
		return err
	}
	return verifyClientMetadata(bundleRoot, item)
}

func verifyClientFiles(bundleRoot string, item Item) error {
	expectedFiles, err := resourcedeployment.ArtifactFiles(item.Artifact)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(item.Files))
	for _, artifact := range item.Files {
		if !resourcedeployment.IsSafeArtifactName(artifact.Name) {
			return fmt.Errorf("resource %s has unsafe artifact file", item.Resource)
		}
		data, err := os.ReadFile(filepath.Join(bundleRoot, "resources", item.Resource, artifact.Name))
		if err != nil {
			return fmt.Errorf("read resource artifact %s: %w", artifact.Name, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != artifact.SHA256 {
			return fmt.Errorf("resource artifact hash mismatch for %s", artifact.Name)
		}
		seen[artifact.Name] = true
	}
	for _, expected := range expectedFiles {
		if !seen[expected] {
			return fmt.Errorf("resource %s is missing required artifact %s", item.Resource, expected)
		}
	}
	return nil
}

func verifyClientContract(bundleRoot string, item Item) error {
	var contract struct {
		Name string `json:"name"`
	}
	manifestData, err := os.ReadFile(filepath.Join(bundleRoot, "resources", item.Resource, item.Artifact+".manifest.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(manifestData, &contract); err != nil {
		return fmt.Errorf("parse resource contract for %s: %w", item.Resource, err)
	}
	if contract.Name != item.Resource {
		return fmt.Errorf("resource artifact contract mismatch: plan selects %s, contract names %s", item.Resource, contract.Name)
	}
	return nil
}

func verifyClientMetadata(bundleRoot string, item Item) error {
	var metadata struct {
		Resource string `json:"resource"`
		Artifact string `json:"artifact"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
	}
	metadataData, err := os.ReadFile(filepath.Join(bundleRoot, "resources", item.Resource, item.Artifact+".build.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return fmt.Errorf("parse resource build metadata for %s: %w", item.Resource, err)
	}
	if metadata.Resource != item.Resource || metadata.Artifact != item.Artifact || metadata.OS != artifactOS(item.OS) || metadata.Arch != item.Architecture {
		return fmt.Errorf("resource artifact build metadata mismatch for %s", item.Resource)
	}
	return nil
}

func artifactOS(os string) string {
	if os == "macos" {
		return "darwin"
	}
	return os
}

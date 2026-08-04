// Package resources loads and validates the resolved resource deployment plan
// included in a desktop bundle. It deliberately does not re-resolve manifests:
// runtime executes the exact, signed selection admitted by the pipeline.
package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type Plan struct {
	SchemaVersion     string `json:"schema_version"`
	ArtifactTrustMode string `json:"artifact_trust_mode,omitempty"`
	Promotable        bool   `json:"promotable"`
	Resources         []Item `json:"resources"`
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
	ProviderPolicy resourcedeployment.ProviderPolicy `json:"provider_policy"`
	Artifact       string                            `json:"artifact"`
	Version        string                            `json:"version"`
	SHA256         string                            `json:"sha256"`
	Arguments      []string                          `json:"arguments,omitempty"`
	Environment    map[string]string                 `json:"environment,omitempty"`
	Config         *resourcedeployment.ServiceConfig `json:"config,omitempty"`
	Ports          []ServicePort                     `json:"ports,omitempty"`
	HealthChecks   []HealthCheck                     `json:"health_checks,omitempty"`
	Files          []Artifact                        `json:"files"`
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
	data, err := os.ReadFile(filepath.Join(bundleRoot, "resources", item.Resource, service.Artifact))
	if err != nil {
		return fmt.Errorf("read bundled service artifact %s: %w", service.Artifact, err)
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), service.SHA256) {
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
	if len(service.Files) != 1 || service.Files[0].Name != service.Artifact || !strings.EqualFold(service.Files[0].SHA256, service.SHA256) {
		return fmt.Errorf("resource %s has an invalid bundled service file record", resource)
	}
	return nil
}

func validateServiceRuntime(resource string, service *Service) error {
	if service.Config != nil {
		if err := service.Config.Validate(); err != nil {
			return fmt.Errorf("resource %s has invalid bundled service config: %w", resource, err)
		}
	}
	for _, check := range service.HealthChecks {
		if check.Type != "http" || strings.TrimSpace(check.Target) == "" {
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

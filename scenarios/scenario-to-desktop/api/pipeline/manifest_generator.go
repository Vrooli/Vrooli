package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/packages/resource-deployment"
	sharedpath "scenario-to-desktop-api/shared/path"
)

// DeploymentManagerGenerator implements ManifestGenerator by calling
// the deployment-manager scenario to create bundle manifests.
type DeploymentManagerGenerator struct {
	client     *http.Client
	logger     Logger
	resolveURL DeploymentManagerURLResolver
}

// DeploymentManagerURLResolver resolves the deployment-manager base URL.
// It is injectable so callers can use an explicit local endpoint in tests
// while production continues to use service discovery.
type DeploymentManagerURLResolver func(context.Context) (string, error)

// DeploymentManagerGeneratorOption configures a DeploymentManagerGenerator.
type DeploymentManagerGeneratorOption func(*DeploymentManagerGenerator)

// WithGeneratorLogger sets the logger for the generator.
func WithGeneratorLogger(l Logger) DeploymentManagerGeneratorOption {
	return func(g *DeploymentManagerGenerator) {
		g.logger = l
	}
}

// WithGeneratorHTTPClient sets a custom HTTP client.
func WithGeneratorHTTPClient(c *http.Client) DeploymentManagerGeneratorOption {
	return func(g *DeploymentManagerGenerator) {
		g.client = c
	}
}

// WithDeploymentManagerURLResolver overrides deployment-manager discovery.
func WithDeploymentManagerURLResolver(resolver DeploymentManagerURLResolver) DeploymentManagerGeneratorOption {
	return func(g *DeploymentManagerGenerator) {
		if resolver != nil {
			g.resolveURL = resolver
		}
	}
}

// NewDeploymentManagerGenerator creates a new manifest generator.
func NewDeploymentManagerGenerator(opts ...DeploymentManagerGeneratorOption) *DeploymentManagerGenerator {
	g := &DeploymentManagerGenerator{
		client: &http.Client{Timeout: 5 * time.Minute},
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
		},
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// bundleExportRequest is the request payload for deployment-manager export.
type bundleExportRequest struct {
	Scenario          string `json:"scenario"`
	Tier              string `json:"tier"`
	ArtifactTrustMode string `json:"artifact_trust_mode,omitempty"`
	IncludeSecrets    *bool  `json:"include_secrets,omitempty"`
	OutputDir         string `json:"output_dir,omitempty"`
	StageBundle       bool   `json:"stage_bundle"`
}

// bundleExportResponse is the response from deployment-manager export.
type bundleExportResponse struct {
	Status       string      `json:"status"`
	Schema       string      `json:"schema"`
	Manifest     interface{} `json:"manifest"`
	Checksum     string      `json:"checksum,omitempty"`
	GeneratedAt  string      `json:"generated_at,omitempty"`
	ManifestPath string      `json:"manifest_path,omitempty"`
}

// GenerateManifest creates a bundle manifest by calling deployment-manager.
func (g *DeploymentManagerGenerator) GenerateManifest(ctx context.Context, scenarioName, outputDir string, trustMode resourcedeployment.ArtifactTrustMode) (string, error) {
	if g.logger != nil {
		g.logger.Info("generating bundle manifest", "scenario", scenarioName, "output_dir", outputDir)
	}

	// Resolve deployment-manager URL
	deploymentManagerURL, err := g.resolveURL(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve deployment-manager: %w", err)
	}

	// Build request
	// Note: StageBundle is false because binaries may not exist yet (especially for TypeScript services).
	// The packager will handle staging after services are built.
	includeSecrets, err := scenarioRequiresSecretPlans(scenarioName)
	if err != nil {
		return "", err
	}
	req := bundleExportRequest{
		Scenario:          scenarioName,
		Tier:              "tier-2-desktop",
		ArtifactTrustMode: string(trustMode),
		IncludeSecrets:    &includeSecrets,
		OutputDir:         outputDir,
		StageBundle:       false,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1/bundles/export", deploymentManagerURL),
		bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("deployment-manager export failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("deployment-manager returned status %d: %s", resp.StatusCode, string(body))
	}

	var exportResp bundleExportResponse
	if err := json.Unmarshal(body, &exportResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Use manifest path from response if provided
	if exportResp.ManifestPath != "" {
		if err := g.requireScenarioServiceManifest(exportResp.ManifestPath, scenarioName); err != nil {
			return "", err
		}
		if g.logger != nil {
			g.logger.Info("manifest generated", "path", exportResp.ManifestPath)
		}
		return exportResp.ManifestPath, nil
	}

	// Otherwise, write the manifest ourselves
	if exportResp.Manifest == nil {
		return "", fmt.Errorf("deployment-manager returned no manifest")
	}

	manifestPath, err := g.writeManifest(scenarioName, outputDir, exportResp.Manifest)
	if err != nil {
		return "", err
	}
	if err := g.requireScenarioServiceManifest(manifestPath, scenarioName); err != nil {
		return "", err
	}

	if g.logger != nil {
		g.logger.Info("manifest written", "path", manifestPath)
	}
	return manifestPath, nil
}

// requireScenarioServiceManifest stages the app's shared dependency source in
// the generated desktop package. The runtime can then project the same
// declarations used by the app integrations API when it joins dependencies to
// signed component releases.
func (g *DeploymentManagerGenerator) requireScenarioServiceManifest(manifestPath, scenarioName string) error {
	if scenarioName == "" || filepath.Base(scenarioName) != scenarioName || scenarioName == "." || scenarioName == ".." {
		return fmt.Errorf("invalid scenario name for dependency catalog")
	}
	source := filepath.Join(sharedpath.DetectScenariosRoot(), scenarioName, ".vrooli", "service.json")
	if info, err := os.Stat(source); os.IsNotExist(err) {
		// Synthetic or remote scenario exports have no repository service
		// manifest to stage. Their optional dependencies remain unavailable.
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect scenario service manifest: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("scenario service manifest is not a regular file")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read generated desktop manifest: %w", err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decode generated desktop manifest: %w", err)
	}
	if document == nil {
		return fmt.Errorf("generated desktop manifest must be a JSON object")
	}
	var requirements []string
	if raw := document["catalog_requirements"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &requirements); err != nil {
			return fmt.Errorf("decode generated catalog requirements: %w", err)
		}
	}
	requiredPath := filepath.ToSlash(filepath.Join("catalog", "scenarios", scenarioName))
	for _, existing := range requirements {
		if existing == requiredPath {
			return nil
		}
	}
	requirements = append(requirements, requiredPath)
	document["catalog_requirements"], err = json.Marshal(requirements)
	if err != nil {
		return fmt.Errorf("encode generated catalog requirements: %w", err)
	}
	updated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode generated desktop manifest: %w", err)
	}
	info, err := os.Stat(manifestPath)
	if err != nil {
		return fmt.Errorf("inspect generated desktop manifest: %w", err)
	}
	tmpPath := manifestPath + ".tmp"
	if err := os.WriteFile(tmpPath, updated, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write generated desktop manifest: %w", err)
	}
	if err := os.Rename(tmpPath, manifestPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace generated desktop manifest: %w", err)
	}
	return nil
}

// scenarioRequiresSecretPlans avoids making an otherwise self-contained
// desktop build depend on Secrets Manager. Resource-backed scenarios still
// obtain their bundle secret plans; an empty dependency set deliberately
// produces an empty secret plan without a cross-scenario runtime call.
func scenarioRequiresSecretPlans(scenarioName string) (bool, error) {
	root := filepath.Join(sharedpath.DetectScenariosRoot(), scenarioName, ".vrooli", "service.json")
	data, err := os.ReadFile(root)
	if err != nil {
		// The generator can be used against a remote or synthetic scenario in
		// tests. Preserve the safe historic behavior in that case: ask Secrets
		// Manager for plans rather than asserting that no plans exist.
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("read scenario service manifest for secret planning: %w", err)
	}
	var service struct {
		Dependencies struct {
			Resources map[string]struct {
				Enabled  bool `json:"enabled"`
				Required bool `json:"required"`
			} `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &service); err != nil {
		return false, fmt.Errorf("parse scenario service manifest for secret planning: %w", err)
	}
	for _, dependency := range service.Dependencies.Resources {
		if dependency.Enabled && dependency.Required {
			return true, nil
		}
	}
	return false, nil
}

// writeManifest writes the manifest to the output directory.
func (g *DeploymentManagerGenerator) writeManifest(scenarioName, outputDir string, manifest interface{}) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize manifest: %w", err)
	}

	// Write to file
	manifestPath := filepath.Join(outputDir, "bundle.json")
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write manifest: %w", err)
	}

	return manifestPath, nil
}

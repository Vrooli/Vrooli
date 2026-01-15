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
)

// DeploymentManagerGenerator implements ManifestGenerator by calling
// the deployment-manager scenario to create bundle manifests.
type DeploymentManagerGenerator struct {
	client *http.Client
	logger Logger
}

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

// NewDeploymentManagerGenerator creates a new manifest generator.
func NewDeploymentManagerGenerator(opts ...DeploymentManagerGeneratorOption) *DeploymentManagerGenerator {
	g := &DeploymentManagerGenerator{
		client: &http.Client{Timeout: 5 * time.Minute},
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// bundleExportRequest is the request payload for deployment-manager export.
type bundleExportRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
	OutputDir      string `json:"output_dir,omitempty"`
	StageBundle    bool   `json:"stage_bundle"`
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
func (g *DeploymentManagerGenerator) GenerateManifest(ctx context.Context, scenarioName, outputDir string) (string, error) {
	if g.logger != nil {
		g.logger.Info("generating bundle manifest", "scenario", scenarioName, "output_dir", outputDir)
	}

	// Resolve deployment-manager URL
	deploymentManagerURL, err := discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
	if err != nil {
		return "", fmt.Errorf("failed to resolve deployment-manager: %w", err)
	}

	// Build request
	includeSecrets := true
	req := bundleExportRequest{
		Scenario:       scenarioName,
		Tier:           "tier-2-desktop",
		IncludeSecrets: &includeSecrets,
		OutputDir:      outputDir,
		StageBundle:    true,
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

	if g.logger != nil {
		g.logger.Info("manifest written", "path", manifestPath)
	}
	return manifestPath, nil
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

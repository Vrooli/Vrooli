package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scenario-to-desktop-api/shared/path"
)

// ServiceJSON represents the structure of .vrooli/service.json.
type ServiceJSON struct {
	Service struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Category    string   `json:"category"`
		Tags        []string `json:"tags"`
		License     string   `json:"license"`
		Maintainers []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"maintainers"`
	} `json:"service"`
	Ports struct {
		API struct {
			EnvVar string `json:"env_var"`
			Range  string `json:"range"`
		} `json:"api"`
		UI struct {
			EnvVar string `json:"env_var"`
			Range  string `json:"range"`
		} `json:"ui"`
	} `json:"ports"`
	Dependencies struct {
		Resources map[string]ResourceDependency `json:"resources"`
	} `json:"dependencies"`
}

// ResourceDependency describes a scenario's dependency on an external resource.
type ResourceDependency struct {
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// DeploymentResourceDependency extends ResourceDependency with platform support metadata.
// This is used when parsing the deployment.dependencies.resources section of service.json.
type DeploymentResourceDependency struct {
	ResourceType    string                         `json:"resource_type"`
	PlatformSupport map[string]PlatformTierSupport `json:"platform_support"`
	SwappableWith   []SwapOption                   `json:"swappable_with"`
	PackagingHints  []string                       `json:"packaging_hints"`
}

// PlatformTierSupport describes a resource's support status for a deployment tier.
type PlatformTierSupport struct {
	Supported    bool     `json:"supported"`
	FitnessScore float64  `json:"fitness_score"`
	Reason       string   `json:"reason"`
	Alternatives []string `json:"alternatives"`
	Notes        string   `json:"notes"`
}

// SwapOption describes an alternative resource that can replace the current one.
type SwapOption struct {
	ID           string `json:"id"`
	Relationship string `json:"relationship"`
	Notes        string `json:"notes"`
}

// DeploymentSection represents the deployment metadata in service.json.
type DeploymentSection struct {
	Dependencies struct {
		Resources map[string]DeploymentResourceDependency `json:"resources"`
	} `json:"dependencies"`
}

// FullServiceJSON extends ServiceJSON with deployment metadata for platform support.
type FullServiceJSON struct {
	ServiceJSON
	Deployment DeploymentSection `json:"deployment"`
}

// FallbackUnbundleableResources is used ONLY when a resource has no
// platform_support metadata in the scenario's service.json.
// Prefer using the scenario's declared metadata over this list.
var FallbackUnbundleableResources = map[string]bool{
	"postgres":      true,
	"redis":         true,
	"qdrant":        true,
	"mysql":         true,
	"mongodb":       true,
	"elasticsearch": true,
	"rabbitmq":      true,
	"kafka":         true,
}

// UIPackageJSON represents relevant fields from ui/package.json.
type UIPackageJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	License     string `json:"license"`
}

// DefaultAnalyzer is the default implementation of ScenarioAnalyzer.
type DefaultAnalyzer struct {
	vrooliRoot string
}

// NewAnalyzer creates a new scenario analyzer.
// If vrooliRoot is empty, attempts to detect it using the canonical DetectVrooliRoot function.
func NewAnalyzer(vrooliRoot string) *DefaultAnalyzer {
	if vrooliRoot == "" {
		// Use canonical VROOLI_ROOT detection from shared/path package
		vrooliRoot = path.DetectVrooliRoot()
	}
	return &DefaultAnalyzer{
		vrooliRoot: vrooliRoot,
	}
}

// AnalyzeScenario analyzes a scenario and extracts all relevant metadata.
func (a *DefaultAnalyzer) AnalyzeScenario(scenarioName string) (*ScenarioMetadata, error) {
	scenarioPath := filepath.Join(a.vrooliRoot, "scenarios", scenarioName)

	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario '%s' not found at %s", scenarioName, scenarioPath)
	}

	metadata := &ScenarioMetadata{
		Name:         scenarioName,
		ScenarioPath: scenarioPath,
		License:      "MIT", // Default
	}

	_ = a.readServiceJSON(metadata)
	_ = a.readUIPackageJSON(metadata)
	a.checkUIBuild(metadata)
	a.setDefaults(metadata)

	return metadata, nil
}

// ValidateScenarioForDesktop checks if a scenario is ready for desktop generation.
func (a *DefaultAnalyzer) ValidateScenarioForDesktop(scenarioName string) error {
	metadata, err := a.AnalyzeScenario(scenarioName)
	if err != nil {
		return err
	}

	if !metadata.HasUI {
		return fmt.Errorf("scenario '%s' does not have a built UI - build the UI first with: cd scenarios/%s/ui && npm run build",
			scenarioName, scenarioName)
	}

	return nil
}

// BundleabilityResult contains the result of checking if a scenario can be bundled.
type BundleabilityResult struct {
	Bundleable           bool
	UnbundleableReason   string
	RequiredResources    []string
	UnbundleableResource string
	Alternatives         []string      // Available swap alternatives for the unbundleable resource
	Warnings             []SwapWarning // Warnings about resources that require swaps
}

// SwapWarning indicates a resource requires a swap for desktop bundling.
type SwapWarning struct {
	Resource     string   // The resource that requires a swap
	Alternatives []string // Available alternatives (e.g., ["sqlite"] for postgres)
	Message      string   // Human-readable warning message
}

// CheckBundleability checks if a scenario can be packaged in bundled mode.
// Returns a BundleabilityResult indicating whether bundled mode is possible
// and which resources prevent it if not.
//
// This method uses a data-driven approach:
// 1. Primary: Check scenario's deployment.dependencies.resources.<name>.platform_support.tier-2-desktop
// 2. Fallback: Use FallbackUnbundleableResources for resources without metadata
//
// If a resource is not supported but has alternatives/swaps declared, the method
// returns a warning but allows the build to proceed.
func (a *DefaultAnalyzer) CheckBundleability(scenarioName string) (*BundleabilityResult, error) {
	metadata, err := a.AnalyzeScenario(scenarioName)
	if err != nil {
		return nil, err
	}

	result := &BundleabilityResult{
		Bundleable:        true,
		RequiredResources: metadata.RequiredResources,
		Warnings:          []SwapWarning{},
	}

	// Read full service.json with deployment metadata
	fullServiceJSON, err := a.readFullServiceJSON(metadata.ScenarioPath)
	if err != nil {
		// Fall back to the old approach if we can't read deployment metadata
		return a.checkBundleabilityFallback(result)
	}

	// Check each required resource
	for _, resourceName := range metadata.RequiredResources {
		// Look for deployment metadata for this resource
		deployDep, hasDeploymentMetadata := fullServiceJSON.Deployment.Dependencies.Resources[resourceName]

		if hasDeploymentMetadata {
			// Check tier-2-desktop support
			desktopSupport, hasTierSupport := deployDep.PlatformSupport["tier-2-desktop"]

			if hasTierSupport && !desktopSupport.Supported {
				// Resource explicitly not supported for desktop
				alternatives := a.getAlternatives(deployDep, desktopSupport)

				if len(alternatives) == 0 {
					// No swap available - fail
					result.Bundleable = false
					result.UnbundleableResource = resourceName
					result.UnbundleableReason = desktopSupport.Reason
					if result.UnbundleableReason == "" {
						result.UnbundleableReason = fmt.Sprintf(
							"Resource '%s' is not supported for desktop bundling", resourceName)
					}
					return result, nil
				}

				// Has swap declared - warn but allow
				result.Warnings = append(result.Warnings, SwapWarning{
					Resource:     resourceName,
					Alternatives: alternatives,
					Message: fmt.Sprintf(
						"Scenario requires '%s' which is not supported for desktop bundling. "+
							"A swap to '%s' is declared. "+
							"NOTE: Dependency swaps require supporting code in the scenario's API. "+
							"If smoke test fails at runtime_readyz, verify the scenario implements the swapped backend.",
						resourceName, strings.Join(alternatives, " or ")),
				})
			}
		} else {
			// No deployment metadata - use fallback detection
			if FallbackUnbundleableResources[resourceName] {
				result.Bundleable = false
				result.UnbundleableResource = resourceName
				result.UnbundleableReason = fmt.Sprintf(
					"Scenario requires '%s' which cannot be bundled into a desktop application. "+
						"No tier-2-desktop platform support is declared in the scenario's service.json.",
					resourceName)
				return result, nil
			}
		}
	}

	return result, nil
}

// getAlternatives extracts alternative resources from deployment metadata.
func (a *DefaultAnalyzer) getAlternatives(dep DeploymentResourceDependency, tierSupport PlatformTierSupport) []string {
	alternatives := make([]string, 0)

	// First check tier-specific alternatives
	alternatives = append(alternatives, tierSupport.Alternatives...)

	// Then check swappable_with from the resource itself
	for _, swap := range dep.SwappableWith {
		// Avoid duplicates
		found := false
		for _, alt := range alternatives {
			if alt == swap.ID {
				found = true
				break
			}
		}
		if !found {
			alternatives = append(alternatives, swap.ID)
		}
	}

	return alternatives
}

// checkBundleabilityFallback uses the legacy hard-coded approach when deployment metadata is unavailable.
func (a *DefaultAnalyzer) checkBundleabilityFallback(result *BundleabilityResult) (*BundleabilityResult, error) {
	for _, resource := range result.RequiredResources {
		if FallbackUnbundleableResources[resource] {
			result.Bundleable = false
			result.UnbundleableResource = resource
			result.UnbundleableReason = fmt.Sprintf(
				"Scenario requires '%s' which cannot be bundled into a desktop application. "+
					"Use --deployment-mode external-server to connect to a running server instead.",
				resource)
			break
		}
	}

	return result, nil
}

// readFullServiceJSON reads and parses .vrooli/service.json with deployment metadata.
func (a *DefaultAnalyzer) readFullServiceJSON(scenarioPath string) (*FullServiceJSON, error) {
	servicePath := filepath.Join(scenarioPath, ".vrooli", "service.json")

	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service.json: %w", err)
	}

	var fullServiceJSON FullServiceJSON
	if err := json.Unmarshal(data, &fullServiceJSON); err != nil {
		return nil, fmt.Errorf("failed to parse service.json: %w", err)
	}

	return &fullServiceJSON, nil
}

// CreateDesktopConfigFromMetadata generates a DesktopConfig from analyzed metadata.
func (a *DefaultAnalyzer) CreateDesktopConfigFromMetadata(metadata *ScenarioMetadata, templateType string) (*DesktopConfig, error) {
	if !metadata.HasUI {
		return nil, fmt.Errorf("scenario '%s' does not have a built UI at %s/ui/dist - run 'cd %s/ui && npm run build' first",
			metadata.Name, metadata.ScenarioPath, metadata.ScenarioPath)
	}

	serverType := "static"
	apiEndpoint := "http://localhost:3000"
	serverPath := metadata.UIDistPath
	externalServerURL := ""
	externalAPIURL := ""

	// Check if scenario has an API
	apiPath := filepath.Join(metadata.ScenarioPath, "api")
	if info, err := os.Stat(apiPath); err == nil && info.IsDir() {
		serverType = "external"
		externalServerURL = fmt.Sprintf("http://localhost:%d", metadata.UIPort)
		if metadata.UIPort == 0 {
			externalServerURL = "http://localhost:3000"
		}
		externalAPIURL = fmt.Sprintf("http://localhost:%d", metadata.APIPort)
		if metadata.APIPort == 0 {
			externalAPIURL = "http://localhost:4000"
		}
		serverPath = externalServerURL
		apiEndpoint = externalAPIURL
	}

	config := &DesktopConfig{
		AppName:        metadata.Name,
		AppDisplayName: metadata.DisplayName,
		AppDescription: metadata.Description,
		Version:        metadata.Version,
		Author:         metadata.Author,
		License:        metadata.License,
		AppID:          metadata.AppID,

		ServerType:        serverType,
		ServerPort:        metadata.UIPort,
		ServerPath:        serverPath,
		APIEndpoint:       apiEndpoint,
		ScenarioPath:      metadata.UIDistPath,
		ScenarioName:      metadata.Name,
		AutoManageVrooli:  false,
		VrooliBinaryPath:  "",
		DeploymentMode:    "external-server",
		ProxyURL:          externalServerURL,
		ExternalServerURL: externalServerURL,
		ExternalAPIURL:    externalAPIURL,

		Framework:    "electron",
		TemplateType: templateType,

		Features: map[string]interface{}{
			"splash":         true,
			"autoUpdater":    true,
			"devTools":       true,
			"singleInstance": true,
			"systemTray":     templateType == "advanced" || templateType == "multi_window",
		},

		Window: map[string]interface{}{
			"width":      1200,
			"height":     800,
			"background": "#1e1e2e",
		},

		Platforms: []string{"win", "mac", "linux"},

		OutputPath: "",

		Styling: map[string]interface{}{
			"splashBackgroundStart": "#1e1e2e",
			"splashBackgroundEnd":   "#89b4fa",
			"splashTextColor":       "#cdd6f4",
			"splashAccentColor":     "#89b4fa",
		},
	}

	return config, nil
}

// readServiceJSON reads and parses .vrooli/service.json.
func (a *DefaultAnalyzer) readServiceJSON(metadata *ScenarioMetadata) error {
	servicePath := filepath.Join(metadata.ScenarioPath, ".vrooli", "service.json")
	metadata.ServiceJSONPath = servicePath

	data, err := os.ReadFile(servicePath)
	if err != nil {
		return fmt.Errorf("failed to read service.json: %w", err)
	}

	var serviceJSON ServiceJSON
	if err := json.Unmarshal(data, &serviceJSON); err != nil {
		return fmt.Errorf("failed to parse service.json: %w", err)
	}

	if serviceJSON.Service.DisplayName != "" {
		metadata.DisplayName = serviceJSON.Service.DisplayName
	}
	if serviceJSON.Service.Description != "" {
		metadata.Description = serviceJSON.Service.Description
	}
	if serviceJSON.Service.Version != "" {
		metadata.Version = serviceJSON.Service.Version
	}
	if serviceJSON.Service.Category != "" {
		metadata.Category = serviceJSON.Service.Category
	}
	if len(serviceJSON.Service.Tags) > 0 {
		metadata.Tags = serviceJSON.Service.Tags
	}
	if serviceJSON.Service.License != "" {
		metadata.License = serviceJSON.Service.License
	}
	if len(serviceJSON.Service.Maintainers) > 0 {
		metadata.Author = serviceJSON.Service.Maintainers[0].Name
	}

	if serviceJSON.Ports.API.Range != "" {
		parts := strings.Split(serviceJSON.Ports.API.Range, "-")
		if len(parts) == 2 {
			var port int
			if _, err := fmt.Sscanf(parts[0], "%d", &port); err == nil {
				metadata.APIPort = port
			}
		}
	}
	if serviceJSON.Ports.UI.Range != "" {
		parts := strings.Split(serviceJSON.Ports.UI.Range, "-")
		if len(parts) == 2 {
			var port int
			if _, err := fmt.Sscanf(parts[0], "%d", &port); err == nil {
				metadata.UIPort = port
			}
		}
	}

	// Extract required resources from dependencies
	for name, dep := range serviceJSON.Dependencies.Resources {
		if dep.Required && dep.Enabled {
			metadata.RequiredResources = append(metadata.RequiredResources, name)
		}
	}

	return nil
}

// readUIPackageJSON reads and parses ui/package.json.
func (a *DefaultAnalyzer) readUIPackageJSON(metadata *ScenarioMetadata) error {
	packagePath := filepath.Join(metadata.ScenarioPath, "ui", "package.json")
	metadata.PackageJSONPath = packagePath

	data, err := os.ReadFile(packagePath)
	if err != nil {
		return fmt.Errorf("failed to read ui/package.json: %w", err)
	}

	var pkgJSON UIPackageJSON
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return fmt.Errorf("failed to parse ui/package.json: %w", err)
	}

	if pkgJSON.Description != "" && metadata.Description == "" {
		metadata.Description = pkgJSON.Description
	}
	if pkgJSON.Version != "" && metadata.Version == "" {
		metadata.Version = pkgJSON.Version
	}
	if pkgJSON.Author != "" && metadata.Author == "" {
		metadata.Author = pkgJSON.Author
	}
	if pkgJSON.License != "" {
		metadata.License = pkgJSON.License
	}

	return nil
}

// checkUIBuild checks if the UI has been built.
func (a *DefaultAnalyzer) checkUIBuild(metadata *ScenarioMetadata) {
	uiDistPath := filepath.Join(metadata.ScenarioPath, "ui", "dist")

	if info, err := os.Stat(uiDistPath); err == nil && info.IsDir() {
		indexPath := filepath.Join(uiDistPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			metadata.HasUI = true
			metadata.UIDistPath = uiDistPath
		}
	}
}

// setDefaults sets smart defaults for missing fields.
func (a *DefaultAnalyzer) setDefaults(metadata *ScenarioMetadata) {
	if metadata.DisplayName == "" {
		metadata.DisplayName = FormatDisplayName(metadata.Name)
	}
	if metadata.Description == "" {
		metadata.Description = fmt.Sprintf("%s desktop application", metadata.DisplayName)
	}
	if metadata.Version == "" {
		metadata.Version = "1.0.0"
	}
	if metadata.Author == "" {
		metadata.Author = "Vrooli Team"
	}
	if metadata.AppID == "" {
		metadata.AppID = fmt.Sprintf("com.vrooli.%s", metadata.Name)
	}
	if metadata.APIPort == 0 {
		metadata.APIPort = 3000
	}
	if metadata.UIPort == 0 {
		metadata.UIPort = 5173
	}
}

// FormatDisplayName converts kebab-case to Title Case.
func FormatDisplayName(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			switch strings.ToLower(part) {
			case "qr", "api", "ui", "cli", "ai", "ml", "nlp":
				parts[i] = strings.ToUpper(part)
			case "prd":
				parts[i] = "PRD"
			default:
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
	}
	return strings.Join(parts, " ")
}

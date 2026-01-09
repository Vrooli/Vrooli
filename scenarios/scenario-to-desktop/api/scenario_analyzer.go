package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScenarioMetadata contains extracted information about a scenario
type ScenarioMetadata struct {
	Name            string
	DisplayName     string
	Description     string
	Version         string
	Author          string
	License         string
	AppID           string
	HasUI           bool
	UIDistPath      string
	UIPort          int
	APIPort         int
	ScenarioPath    string
	Category        string
	Tags            []string
	ServiceJSONPath string
	PackageJSONPath string
}

// ServiceJSON represents the structure of .vrooli/service.json
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
}

// UIPackageJSON represents relevant fields from ui/package.json
type UIPackageJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	License     string `json:"license"`
}

// ScenarioAnalyzer analyzes scenarios and extracts metadata
type ScenarioAnalyzer struct {
	vrooliRoot string
}

// NewScenarioAnalyzer creates a new analyzer instance
func NewScenarioAnalyzer(vrooliRoot string) *ScenarioAnalyzer {
	if vrooliRoot == "" {
		// Fallback to calculating from current directory
		currentDir, _ := os.Getwd()
		vrooliRoot = filepath.Join(currentDir, "../../..")
	}
	return &ScenarioAnalyzer{
		vrooliRoot: vrooliRoot,
	}
}

// AnalyzeScenario analyzes a scenario and extracts all relevant metadata
func (sa *ScenarioAnalyzer) AnalyzeScenario(scenarioName string) (*ScenarioMetadata, error) {
	scenarioPath := filepath.Join(sa.vrooliRoot, "scenarios", scenarioName)

	// Check if scenario exists
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("scenario '%s' not found at %s", scenarioName, scenarioPath)
	}

	metadata := &ScenarioMetadata{
		Name:         scenarioName,
		ScenarioPath: scenarioPath,
		License:      "MIT", // Default
	}

	// Read .vrooli/service.json
	if err := sa.readServiceJSON(metadata); err != nil {
		// Not critical - service.json might not exist for all scenarios
		// Continue with defaults
	}

	// Read ui/package.json if it exists
	if err := sa.readUIPackageJSON(metadata); err != nil {
		// Not critical - some scenarios don't have UI
	}

	// Check for built UI
	sa.checkUIBuild(metadata)

	// Set smart defaults if fields are still empty
	sa.setDefaults(metadata)

	return metadata, nil
}

// readServiceJSON reads and parses .vrooli/service.json
func (sa *ScenarioAnalyzer) readServiceJSON(metadata *ScenarioMetadata) error {
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

	// Extract metadata
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

	// Extract port ranges and pick default ports
	// For desktop apps, we'll use external mode, so we need to know the API port
	if serviceJSON.Ports.API.Range != "" {
		// Parse range like "15000-19999" and pick the start
		parts := strings.Split(serviceJSON.Ports.API.Range, "-")
		if len(parts) == 2 {
			var port int
			fmt.Sscanf(parts[0], "%d", &port)
			metadata.APIPort = port
		}
	}
	if serviceJSON.Ports.UI.Range != "" {
		parts := strings.Split(serviceJSON.Ports.UI.Range, "-")
		if len(parts) == 2 {
			var port int
			fmt.Sscanf(parts[0], "%d", &port)
			metadata.UIPort = port
		}
	}

	return nil
}

// readUIPackageJSON reads and parses ui/package.json
func (sa *ScenarioAnalyzer) readUIPackageJSON(metadata *ScenarioMetadata) error {
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

	// UI package.json can override some fields
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

// checkUIBuild checks if the UI has been built
func (sa *ScenarioAnalyzer) checkUIBuild(metadata *ScenarioMetadata) {
	uiDistPath := filepath.Join(metadata.ScenarioPath, "ui", "dist")

	// Check if dist directory exists
	if info, err := os.Stat(uiDistPath); err == nil && info.IsDir() {
		// Check if index.html exists (required for built UI)
		indexPath := filepath.Join(uiDistPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			metadata.HasUI = true
			metadata.UIDistPath = uiDistPath
		}
	}
}

// setDefaults sets smart defaults for missing fields
func (sa *ScenarioAnalyzer) setDefaults(metadata *ScenarioMetadata) {
	// Display name defaults to capitalized scenario name
	if metadata.DisplayName == "" {
		metadata.DisplayName = formatDisplayName(metadata.Name)
	}

	// Description defaults to generic description
	if metadata.Description == "" {
		metadata.Description = fmt.Sprintf("%s desktop application", metadata.DisplayName)
	}

	// Version defaults to 1.0.0
	if metadata.Version == "" {
		metadata.Version = "1.0.0"
	}

	// Author defaults to scenario maintainer or Vrooli Team
	if metadata.Author == "" {
		metadata.Author = "Vrooli Team"
	}

	// App ID defaults to com.vrooli.<scenario-name>
	if metadata.AppID == "" {
		metadata.AppID = fmt.Sprintf("com.vrooli.%s", metadata.Name)
	}

	// Default ports if not found in service.json
	if metadata.APIPort == 0 {
		metadata.APIPort = 3000
	}
	if metadata.UIPort == 0 {
		metadata.UIPort = 5173 // Default Vite port
	}
}

// formatDisplayName converts kebab-case to Title Case
// Example: "qr-code-generator" → "QR Code Generator"
func formatDisplayName(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			// Special case common abbreviations
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

// CreateDesktopConfigFromMetadata generates a DesktopConfig from analyzed metadata
func (sa *ScenarioAnalyzer) CreateDesktopConfigFromMetadata(metadata *ScenarioMetadata, templateType string) (*DesktopConfig, error) {
	// Validate that UI is built
	if !metadata.HasUI {
		return nil, fmt.Errorf("scenario '%s' does not have a built UI at %s/ui/dist - run 'cd %s/ui && npm run build' first",
			metadata.Name, metadata.ScenarioPath, metadata.ScenarioPath)
	}

	// Determine server type based on whether scenario has API
	serverType := "static"
	apiEndpoint := "http://localhost:3000"
	serverPath := metadata.UIDistPath
	externalServerURL := ""
	externalAPIURL := ""

	// Check if scenario has an API
	apiPath := filepath.Join(metadata.ScenarioPath, "api")
	if info, err := os.Stat(apiPath); err == nil && info.IsDir() {
		// Has API - use external mode (desktop app connects to running API)
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
		ScenarioPath:      metadata.UIDistPath, // UI dist path for copying
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

		// OutputPath will be auto-set by generator to platforms/electron/
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

// ValidateScenarioForDesktop checks if a scenario is ready for desktop generation
func (sa *ScenarioAnalyzer) ValidateScenarioForDesktop(scenarioName string) error {
	metadata, err := sa.AnalyzeScenario(scenarioName)
	if err != nil {
		return err
	}

	if !metadata.HasUI {
		return fmt.Errorf("scenario '%s' does not have a built UI - build the UI first with: cd scenarios/%s/ui && npm run build",
			scenarioName, scenarioName)
	}

	return nil
}

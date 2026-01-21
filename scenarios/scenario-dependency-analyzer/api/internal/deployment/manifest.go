package deployment

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// BuildBundleManifest generates a manifest of all files and dependencies needed
// to package the scenario for deployment.
func BuildBundleManifest(scenarioName, scenarioPath string, generatedAt time.Time, nodes []types.DeploymentDependencyNode, cfg *types.ServiceConfig) types.BundleManifest {
	manifest := types.BundleManifest{
		Scenario:     scenarioName,
		GeneratedAt:  generatedAt,
		Files:        discoverBundleFiles(scenarioName, scenarioPath),
		Dependencies: flattenBundleDependencies(nodes),
		Skeleton:     buildDesktopBundleSkeleton(scenarioName, scenarioPath, cfg, nodes),
	}
	manifest.Dependencies = includeDeclaredResources(manifest.Dependencies, cfg)
	return manifest
}

// discoverBundleFiles scans scenario folders and builds a list of files needed for deployment.
// Uses marker-based detection instead of hardcoded folder names.
func discoverBundleFiles(scenarioName, scenarioPath string) []types.BundleFileEntry {
	entries := make([]types.BundleFileEntry, 0)

	// Always include service config
	serviceConfigPath := filepath.Join(".vrooli", "service.json")
	configAbsolute := filepath.Join(scenarioPath, serviceConfigPath)
	_, configErr := os.Stat(configAbsolute)
	entries = append(entries, types.BundleFileEntry{
		Path:   filepath.ToSlash(serviceConfigPath),
		Type:   "service-config",
		Exists: configErr == nil,
	})

	// Scan for buildable folders using marker-based detection
	folders, err := ScanScenarioFolders(scenarioPath)
	if err != nil {
		return entries
	}

	for _, folder := range folders {
		switch folder.Role {
		case RoleAPI:
			// Add API source directory
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(folder.Path),
				Type:   fmt.Sprintf("%s-source", folder.Name),
				Exists: true, // We know it exists because scanner found it
			})
			// Add API binary path (may not exist yet - to be compiled)
			binaryName := fmt.Sprintf("%s-%s", scenarioName, folder.Name)
			binaryPath := filepath.Join(folder.Path, binaryName)
			binaryAbsolute := filepath.Join(scenarioPath, binaryPath)
			_, binErr := os.Stat(binaryAbsolute)
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(binaryPath),
				Type:   fmt.Sprintf("%s-binary", folder.Name),
				Exists: binErr == nil,
				Notes:  fmt.Sprintf("Detected %s component (%s)", folder.Role, folder.Language),
			})

		case RoleCLI:
			// Add CLI source directory
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(folder.Path),
				Type:   fmt.Sprintf("%s-source", folder.Name),
				Exists: true,
			})
			// Add CLI binary path
			binaryName := scenarioName
			if folder.Name != "cli" {
				binaryName = fmt.Sprintf("%s-%s", scenarioName, folder.Name)
			}
			binaryPath := filepath.Join(folder.Path, binaryName)
			binaryAbsolute := filepath.Join(scenarioPath, binaryPath)
			_, binErr := os.Stat(binaryAbsolute)
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(binaryPath),
				Type:   fmt.Sprintf("%s-binary", folder.Name),
				Exists: binErr == nil,
				Notes:  fmt.Sprintf("Detected %s component (%s)", folder.Role, folder.Language),
			})

		case RoleUI:
			// Add UI dist bundle
			distPath := filepath.Join(folder.Path, "dist")
			distAbsolute := filepath.Join(scenarioPath, distPath)
			_, distErr := os.Stat(distAbsolute)
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(distPath),
				Type:   "ui-bundle",
				Exists: distErr == nil,
			})
			// Add UI entry point
			entryPath := filepath.Join(folder.Path, "dist", "index.html")
			entryAbsolute := filepath.Join(scenarioPath, entryPath)
			_, entryErr := os.Stat(entryAbsolute)
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(entryPath),
				Type:   "ui-entry",
				Exists: entryErr == nil,
				Notes:  fmt.Sprintf("Detected UI component (%s)", folder.Language),
			})

		case RoleWorker:
			// Add worker source directory
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(folder.Path),
				Type:   fmt.Sprintf("%s-source", folder.Name),
				Exists: true,
			})
			// Add worker binary path
			binaryName := fmt.Sprintf("%s-%s", scenarioName, folder.Name)
			binaryPath := filepath.Join(folder.Path, binaryName)
			binaryAbsolute := filepath.Join(scenarioPath, binaryPath)
			_, binErr := os.Stat(binaryAbsolute)
			entries = append(entries, types.BundleFileEntry{
				Path:   filepath.ToSlash(binaryPath),
				Type:   fmt.Sprintf("%s-binary", folder.Name),
				Exists: binErr == nil,
				Notes:  fmt.Sprintf("Detected %s component (%s)", folder.Role, folder.Language),
			})
		}
	}

	return entries
}

// flattenBundleDependencies walks the entire dependency tree and returns a flat,
// deduplicated list of all resource and scenario dependencies.
func flattenBundleDependencies(nodes []types.DeploymentDependencyNode) []types.BundleDependencyEntry {
	seen := map[string]types.BundleDependencyEntry{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		key := fmt.Sprintf("%s:%s", node.Type, node.Name)
		if _, exists := seen[key]; !exists {
			seen[key] = types.BundleDependencyEntry{
				Name:         node.Name,
				Type:         node.Type,
				ResourceType: node.ResourceType,
				TierSupport:  node.TierSupport,
				Alternatives: dedupeStrings(node.Alternatives),
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	entries := make([]types.BundleDependencyEntry, 0, len(seen))
	for _, entry := range seen {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type < entries[j].Type
	})
	return entries
}

func includeDeclaredResources(entries []types.BundleDependencyEntry, cfg *types.ServiceConfig) []types.BundleDependencyEntry {
	if cfg == nil {
		return entries
	}
	resources := config.ResolvedResourceMap(cfg)
	if len(resources) == 0 {
		return entries
	}
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if entry.Type == "resource" {
			seen[fmt.Sprintf("resource:%s", entry.Name)] = struct{}{}
		}
	}
	for name, resource := range resources {
		if !(resource.Required || resource.Enabled) {
			continue
		}
		key := fmt.Sprintf("resource:%s", name)
		if _, exists := seen[key]; exists {
			continue
		}
		entries = append(entries, types.BundleDependencyEntry{
			Name:         name,
			Type:         "resource",
			ResourceType: resource.Type,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type < entries[j].Type
	})
	return entries
}

func buildDesktopBundleSkeleton(scenarioName, scenarioPath string, cfg *types.ServiceConfig, nodes []types.DeploymentDependencyNode) *types.DesktopBundleSkeleton {
	if cfg == nil {
		return nil
	}

	// Resolve app name from v2.0 flat fields first, then v1.x nested fields
	appName := cfg.DisplayName
	if appName == "" {
		appName = cfg.Name
	}
	if appName == "" {
		appName = cfg.Service.DisplayName
	}
	if appName == "" {
		appName = cfg.Service.Name
	}
	if appName == "" {
		appName = scenarioName
	}

	// Resolve version similarly
	version := cfg.Service.Version
	if version == "" {
		version = "0.1.0"
	}

	// Resolve description
	description := cfg.Description
	if description == "" {
		description = cfg.Service.Description
	}

	skeleton := &types.DesktopBundleSkeleton{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App: types.BundleSkeletonApp{
			Name:        appName,
			Version:     version,
			Description: description,
		},
		IPC: types.BundleSkeletonIPC{
			Mode:          "loopback-http",
			Host:          "127.0.0.1",
			Port:          39200,
			AuthTokenPath: filepath.ToSlash(filepath.Join("runtime", "auth_token")),
		},
		Telemetry: types.BundleSkeletonTelemetry{
			File: filepath.ToSlash(filepath.Join("telemetry", "deployment-telemetry.jsonl")),
		},
		Ports: types.BundleSkeletonPorts{
			DefaultRange: types.BundleSkeletonPortRange{Min: 20000, Max: 24000},
		},
	}

	skeleton.Swaps = deriveSwaps(nodes)
	skeleton.Services = buildSkeletonServices(scenarioName, scenarioPath, cfg)

	return skeleton
}

func deriveSwaps(nodes []types.DeploymentDependencyNode) []types.BundleSkeletonSwap {
	swaps := make([]types.BundleSkeletonSwap, 0)
	seen := map[string]struct{}{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		if len(node.Alternatives) > 0 && node.Type == "resource" {
			if _, exists := seen[node.Name]; !exists {
				seen[node.Name] = struct{}{}
				swaps = append(swaps, types.BundleSkeletonSwap{
					Original:    node.Name,
					Replacement: node.Alternatives[0],
					Reason:      "Recommended bundle-safe alternative from dependency metadata",
				})
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	sort.Slice(swaps, func(i, j int) bool {
		return swaps[i].Original < swaps[j].Original
	})
	return swaps
}

func buildSkeletonServices(scenarioName, scenarioPath string, cfg *types.ServiceConfig) []types.BundleSkeletonService {
	// Use marker-based folder scanning to detect buildable components
	folders, err := ScanScenarioFolders(scenarioPath)
	if err != nil {
		// If scanning fails, return empty slice - don't fall back to hardcoded paths
		return []types.BundleSkeletonService{}
	}

	var services []types.BundleSkeletonService
	for _, folder := range folders {
		svc := buildServiceForFolder(scenarioName, scenarioPath, folder, cfg)
		if svc != nil {
			services = append(services, *svc)
		}
	}

	return services
}

// buildServiceForFolder creates a skeleton service based on the detected folder role and language.
func buildServiceForFolder(scenarioName, scenarioPath string, folder DetectedFolder, cfg *types.ServiceConfig) *types.BundleSkeletonService {
	switch folder.Role {
	case RoleAPI:
		return buildAPIServiceFromFolder(scenarioName, scenarioPath, folder, cfg)
	case RoleCLI:
		return buildCLIServiceFromFolder(scenarioName, folder, cfg)
	case RoleUI:
		return buildUIServiceFromFolder(scenarioName, scenarioPath, folder, cfg)
	case RoleWorker:
		return buildWorkerServiceFromFolder(scenarioName, folder, cfg)
	default:
		// Skip libraries and unknown roles - they don't produce standalone services
		return nil
	}
}

// buildAPIServiceFromFolder creates an API service skeleton from a detected folder.
func buildAPIServiceFromFolder(scenarioName, scenarioPath string, folder DetectedFolder, cfg *types.ServiceConfig) *types.BundleSkeletonService {
	binaryBase := fmt.Sprintf("%s-%s", scenarioName, folder.Name)

	service := &types.BundleSkeletonService{
		ID:          fmt.Sprintf("%s-%s", scenarioName, folder.Name),
		Type:        "api-binary",
		Description: fmt.Sprintf("Bundled %s process for %s", folder.Name, scenarioName),
		Binaries: map[string]types.BundleSkeletonServiceBinary{
			"darwin-x64": {Path: platformBinaryPath(folder.Name, "darwin-x64", binaryBase)},
			"linux-x64":  {Path: platformBinaryPath(folder.Name, "linux-x64", binaryBase)},
			"win-x64":    {Path: platformBinaryPath(folder.Name, "win-x64", binaryBase)},
		},
		Env: map[string]string{},
		DataDirs: []string{
			filepath.ToSlash(filepath.Join("data", folder.Name)),
		},
		LogDir: filepath.ToSlash(filepath.Join("logs", folder.Name)),
		Ports: &types.BundleSkeletonServicePorts{
			Requested: []types.BundleSkeletonRequestedPort{
				{Name: folder.Name, Range: types.BundleSkeletonPortRange{Min: 23100, Max: 23200}},
			},
		},
		Health: types.BundleSkeletonHealth{
			Type:     "http",
			Path:     "/health",
			PortName: folder.Name,
			Interval: 2000,
			Timeout:  15000,
			Retries:  5,
		},
		Readiness: types.BundleSkeletonReadiness{
			Type:     "health_success",
			PortName: folder.Name,
			Timeout:  30000,
		},
		Migrations: []types.BundleSkeletonMigration{},
		Assets:     []types.BundleSkeletonAsset{},
	}

	// Use build config from service.json if available
	if cfg != nil && cfg.Deployment != nil && cfg.Deployment.BuildConfigs != nil {
		if buildCfg, ok := cfg.Deployment.BuildConfigs[folder.Name]; ok {
			service.Build = &types.BundleSkeletonBuildConfig{
				Type:          buildCfg.Type,
				SourceDir:     buildCfg.SourceDir,
				EntryPoint:    buildCfg.EntryPoint,
				OutputPattern: buildCfg.OutputPattern,
			}
		}
	}

	// Generate build config from detected language if not provided
	if service.Build == nil {
		service.Build = buildConfigForLanguage(scenarioName, folder)
	}

	service.Env = deriveServiceEnv(cfg, service.ID, service.Ports, scenarioName, service.Env)

	return service
}

// buildCLIServiceFromFolder creates a CLI service skeleton from a detected folder.
func buildCLIServiceFromFolder(scenarioName string, folder DetectedFolder, cfg *types.ServiceConfig) *types.BundleSkeletonService {
	binaryBase := scenarioName
	// For CLI folders not named "cli", append folder name
	if folder.Name != "cli" {
		binaryBase = fmt.Sprintf("%s-%s", scenarioName, folder.Name)
	}

	service := &types.BundleSkeletonService{
		ID:          fmt.Sprintf("%s-%s", scenarioName, folder.Name),
		Type:        "resource",
		Description: fmt.Sprintf("%s component for %s", folder.Name, scenarioName),
		Binaries: map[string]types.BundleSkeletonServiceBinary{
			"darwin-x64": {Path: platformBinaryPath(folder.Name, "darwin-x64", binaryBase)},
			"linux-x64":  {Path: platformBinaryPath(folder.Name, "linux-x64", binaryBase)},
			"win-x64":    {Path: platformBinaryPath(folder.Name, "win-x64", binaryBase)},
		},
		Health: types.BundleSkeletonHealth{
			Type:     "command",
			Command:  []string{"echo", "healthy"},
			Interval: 10000,
			Timeout:  5000,
			Retries:  1,
		},
		Readiness: types.BundleSkeletonReadiness{
			Type:     "health_success",
			PortName: "http",
			Timeout:  5000,
		},
	}

	// Use build config from service.json if available
	if cfg != nil && cfg.Deployment != nil && cfg.Deployment.BuildConfigs != nil {
		if buildCfg, ok := cfg.Deployment.BuildConfigs[folder.Name]; ok {
			service.Build = &types.BundleSkeletonBuildConfig{
				Type:          buildCfg.Type,
				SourceDir:     buildCfg.SourceDir,
				EntryPoint:    buildCfg.EntryPoint,
				OutputPattern: buildCfg.OutputPattern,
			}
		}
	}

	// Generate build config from detected language if not provided
	if service.Build == nil {
		service.Build = buildConfigForLanguage(scenarioName, folder)
	}

	service.Env = map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": "true",
	}

	return service
}

// buildUIServiceFromFolder creates a UI service skeleton from a detected folder.
func buildUIServiceFromFolder(scenarioName, scenarioPath string, folder DetectedFolder, cfg *types.ServiceConfig) *types.BundleSkeletonService {
	// UI services serve pre-built static assets from dist/
	entry := filepath.ToSlash(filepath.Join(folder.Name, "dist", "index.html"))

	// Check if the dist folder actually exists
	distPath := filepath.Join(scenarioPath, folder.Name, "dist")
	if _, err := os.Stat(distPath); err != nil {
		// UI dist doesn't exist - skip this folder
		return nil
	}

	service := &types.BundleSkeletonService{
		ID:          "ui",
		Type:        "ui-bundle",
		Description: fmt.Sprintf("Production UI bundle for %s", scenarioName),
		Binaries: map[string]types.BundleSkeletonServiceBinary{
			"darwin-x64": {Path: entry},
			"linux-x64":  {Path: entry},
			"win-x64":    {Path: entry},
		},
		LogDir: filepath.ToSlash(filepath.Join("logs", "ui")),
		Ports: &types.BundleSkeletonServicePorts{
			Requested: []types.BundleSkeletonRequestedPort{
				{Name: "ui", Range: types.BundleSkeletonPortRange{Min: 24100, Max: 24200}},
			},
		},
		Health: types.BundleSkeletonHealth{
			Type:     "http",
			Path:     "/health",
			PortName: "ui",
			Interval: 2000,
			Timeout:  10000,
			Retries:  3,
		},
		Readiness: types.BundleSkeletonReadiness{
			Type:     "health_success",
			PortName: "ui",
			Timeout:  20000,
		},
		Assets: []types.BundleSkeletonAsset{
			{Path: entry, SHA256: "pending"},
		},
		Critical: ptrBool(true),
	}

	service.Env = deriveServiceEnv(cfg, service.ID, service.Ports, scenarioName, service.Env)

	return service
}

// buildWorkerServiceFromFolder creates a worker service skeleton from a detected folder.
func buildWorkerServiceFromFolder(scenarioName string, folder DetectedFolder, cfg *types.ServiceConfig) *types.BundleSkeletonService {
	binaryBase := fmt.Sprintf("%s-%s", scenarioName, folder.Name)

	service := &types.BundleSkeletonService{
		ID:          fmt.Sprintf("%s-%s", scenarioName, folder.Name),
		Type:        "worker",
		Description: fmt.Sprintf("Worker process for %s", scenarioName),
		Binaries: map[string]types.BundleSkeletonServiceBinary{
			"darwin-x64": {Path: platformBinaryPath(folder.Name, "darwin-x64", binaryBase)},
			"linux-x64":  {Path: platformBinaryPath(folder.Name, "linux-x64", binaryBase)},
			"win-x64":    {Path: platformBinaryPath(folder.Name, "win-x64", binaryBase)},
		},
		Health: types.BundleSkeletonHealth{
			Type:     "command",
			Command:  []string{"echo", "healthy"},
			Interval: 10000,
			Timeout:  5000,
			Retries:  1,
		},
		Readiness: types.BundleSkeletonReadiness{
			Type:    "health_success",
			Timeout: 10000,
		},
	}

	// Use build config from service.json if available
	if cfg != nil && cfg.Deployment != nil && cfg.Deployment.BuildConfigs != nil {
		if buildCfg, ok := cfg.Deployment.BuildConfigs[folder.Name]; ok {
			service.Build = &types.BundleSkeletonBuildConfig{
				Type:          buildCfg.Type,
				SourceDir:     buildCfg.SourceDir,
				EntryPoint:    buildCfg.EntryPoint,
				OutputPattern: buildCfg.OutputPattern,
			}
		}
	}

	// Generate build config from detected language if not provided
	if service.Build == nil {
		service.Build = buildConfigForLanguage(scenarioName, folder)
	}

	service.Env = map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": "true",
	}

	return service
}

// buildConfigForLanguage generates a build config based on the detected language.
func buildConfigForLanguage(scenarioName string, folder DetectedFolder) *types.BundleSkeletonBuildConfig {
	switch folder.Language {
	case LangGo:
		binaryBase := scenarioName
		if folder.Name != "cli" {
			binaryBase = fmt.Sprintf("%s-%s", scenarioName, folder.Name)
		}
		return &types.BundleSkeletonBuildConfig{
			Type:          "go",
			SourceDir:     folder.Path,
			EntryPoint:    ".",
			OutputPattern: filepath.ToSlash(filepath.Join("bin", folder.Name, "{{platform}}", binaryBase+"{{ext}}")),
			Env: map[string]string{
				"CGO_ENABLED": "0",
			},
		}
	case LangRust:
		return &types.BundleSkeletonBuildConfig{
			Type:          "rust",
			SourceDir:     folder.Path,
			EntryPoint:    "src/main.rs",
			OutputPattern: filepath.ToSlash(filepath.Join("bin", folder.Name, "{{platform}}", fmt.Sprintf("%s-%s{{ext}}", scenarioName, folder.Name))),
		}
	case LangTypeScript, LangJavaScript:
		return &types.BundleSkeletonBuildConfig{
			Type:          "npm",
			SourceDir:     folder.Path,
			EntryPoint:    ".",
			OutputPattern: filepath.ToSlash(filepath.Join("bin", folder.Name, "{{platform}}", fmt.Sprintf("%s-%s{{ext}}", scenarioName, folder.Name))),
		}
	case LangPython:
		return &types.BundleSkeletonBuildConfig{
			Type:      "custom",
			SourceDir: folder.Path,
			// Python bundling to be defined by scenario-specific config
		}
	default:
		return nil
	}
}

func platformBinaryPath(folder, platform, base string) string {
	ext := ""
	if strings.HasPrefix(platform, "win") {
		ext = ".exe"
	}
	return filepath.ToSlash(filepath.Join("bin", folder, platform, base+ext))
}

// deriveServiceEnv injects lifecycle protection, port bindings, and swap-aware overrides.
func deriveServiceEnv(cfg *types.ServiceConfig, serviceID string, ports *types.BundleSkeletonServicePorts, scenarioName string, existing map[string]string) map[string]string {
	env := map[string]string{}
	for k, v := range existing {
		env[k] = v
	}

	// Lifecycle protection is required by all APIs we bundle.
	env["VROOLI_LIFECYCLE_MANAGED"] = "true"

	// Wire port env vars from service.json where present.
	if cfg != nil {
		portEnv := extractPortEnv(cfg)
		if ports != nil {
			for _, p := range ports.Requested {
				if envVar, ok := portEnv[p.Name]; ok && envVar != "" {
					env[envVar] = fmt.Sprintf("${%s.%s}", serviceID, p.Name)
				}
			}
		}
	}

	// Swap-aware defaults: if Postgres -> SQLite is suggested, prefer SQLite backend for the API service.
	if serviceID == fmt.Sprintf("%s-api", scenarioName) && prefersSQLite(cfg) {
		env["BAS_DB_BACKEND"] = "sqlite"
		if _, ok := env["BAS_SQLITE_PATH"]; !ok {
			env["BAS_SQLITE_PATH"] = filepath.ToSlash(filepath.Join("${data}", "data", "api", fmt.Sprintf("%s.sqlite", scenarioName)))
		}
	}

	return env
}

// extractPortEnv maps port names to their env_var from service.json.
func extractPortEnv(cfg *types.ServiceConfig) map[string]string {
	result := map[string]string{}
	if cfg == nil || cfg.Ports == nil {
		return result
	}
	for name, raw := range cfg.Ports {
		if rawMap, ok := raw.(map[string]interface{}); ok {
			if envVar, ok := rawMap["env_var"].(string); ok && envVar != "" {
				result[name] = envVar
			}
		}
	}
	return result
}

// prefersSQLite checks deployment metadata for a Postgres->SQLite swap hint.
func prefersSQLite(cfg *types.ServiceConfig) bool {
	if cfg == nil || cfg.Deployment == nil || cfg.Deployment.Dependencies.Resources == nil {
		return false
	}
	res, ok := cfg.Deployment.Dependencies.Resources["postgres"]
	if !ok {
		return false
	}
	for _, sw := range res.SwappableWith {
		if strings.EqualFold(sw.ID, "sqlite") {
			return true
		}
	}
	return false
}

func ptrBool(v bool) *bool {
	return &v
}

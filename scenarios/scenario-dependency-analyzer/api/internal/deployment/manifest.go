package deployment

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	types "scenario-dependency-analyzer/internal/types"
)

// BuildBundleManifest generates a manifest of all files and dependencies needed
// to package the scenario for deployment.
func BuildBundleManifest(scenarioName, scenarioPath string, generatedAt time.Time, nodes []types.DeploymentDependencyNode, cfg *types.ServiceConfig) types.BundleManifest {
	return types.BundleManifest{
		Scenario:     scenarioName,
		GeneratedAt:  generatedAt,
		Files:        discoverBundleFiles(scenarioName, scenarioPath),
		Dependencies: flattenBundleDependencies(nodes),
		Skeleton:     buildDesktopBundleSkeleton(scenarioName, scenarioPath, cfg, nodes),
	}
}

// discoverBundleFiles scans for standard scenario files needed in a deployment bundle
func discoverBundleFiles(scenarioName, scenarioPath string) []types.BundleFileEntry {
	candidates := []struct {
		Path  string
		Type  string
		Notes string
	}{
		{Path: filepath.Join(".vrooli", "service.json"), Type: "service-config"},
		{Path: "api", Type: "api-source"},
		{Path: filepath.Join("api", fmt.Sprintf("%s-api", scenarioName)), Type: "api-binary"},
		{Path: filepath.Join("ui", "dist"), Type: "ui-bundle"},
		{Path: filepath.Join("ui", "dist", "index.html"), Type: "ui-entry"},
		{Path: filepath.Join("cli", scenarioName), Type: "cli-binary"},
	}
	entries := make([]types.BundleFileEntry, 0, len(candidates))
	for _, candidate := range candidates {
		absolute := filepath.Join(scenarioPath, candidate.Path)
		_, err := os.Stat(absolute)
		entries = append(entries, types.BundleFileEntry{
			Path:   filepath.ToSlash(candidate.Path),
			Type:   candidate.Type,
			Exists: err == nil,
			Notes:  candidate.Notes,
		})
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
	services := []types.BundleSkeletonService{
		buildAPISkeletonService(scenarioName, scenarioPath, cfg),
	}

	if uiExists(scenarioPath) {
		services = append(services, buildUISkeletonService(scenarioName, scenarioPath, cfg))
	}

	if cliSvc := buildCLISkeletonService(scenarioName, scenarioPath, cfg); cliSvc != nil {
		services = append(services, *cliSvc)
	}

	return services
}

func buildAPISkeletonService(scenarioName, scenarioPath string, cfg *types.ServiceConfig) types.BundleSkeletonService {
	defaultPath := filepath.ToSlash(filepath.Join("api", fmt.Sprintf("%s-api", scenarioName)))
	apiBinary := defaultPath
	if _, err := os.Stat(filepath.Join(scenarioPath, defaultPath)); err != nil {
		apiBinary = filepath.ToSlash(filepath.Join("api", "server"))
	}

	service := types.BundleSkeletonService{
		ID:          fmt.Sprintf("%s-api", scenarioName),
		Type:        "api-binary",
		Description: fmt.Sprintf("Bundled API process for %s", scenarioName),
		Binaries: map[string]types.BundleSkeletonServiceBinary{
			"darwin-x64": {Path: apiBinary},
			"linux-x64":  {Path: apiBinary},
			"win-x64":    {Path: apiBinary + ".exe"},
		},
		Env: map[string]string{},
		DataDirs: []string{
			filepath.ToSlash(filepath.Join("data", "api")),
		},
		LogDir: filepath.ToSlash(filepath.Join("logs", "api")),
		Ports: &types.BundleSkeletonServicePorts{
			Requested: []types.BundleSkeletonRequestedPort{
				{Name: "api", Range: types.BundleSkeletonPortRange{Min: 23100, Max: 23200}},
			},
		},
		Health: types.BundleSkeletonHealth{
			Type:     "http",
			Path:     "/health",
			PortName: "api",
			Interval: 2000,
			Timeout:  15000,
			Retries:  5,
		},
		Readiness: types.BundleSkeletonReadiness{
			Type:     "port_open",
			PortName: "api",
			Timeout:  30000,
		},
		Migrations: []types.BundleSkeletonMigration{},
		Assets:     []types.BundleSkeletonAsset{},
	}

	// Add build config from service.json if available
	if cfg != nil && cfg.Deployment != nil && cfg.Deployment.BuildConfigs != nil {
		if buildCfg, ok := cfg.Deployment.BuildConfigs["api"]; ok {
			service.Build = &types.BundleSkeletonBuildConfig{
				Type:          buildCfg.Type,
				SourceDir:     buildCfg.SourceDir,
				EntryPoint:    buildCfg.EntryPoint,
				OutputPattern: buildCfg.OutputPattern,
			}
		}
	}

	// If no build config was provided, infer a cross-platform Go build.
	// This unlocks deployment-manager + scenario-to-desktop to compile missing binaries
	// for win/mac/linux instead of failing when only a single platform binary exists.
	if service.Build == nil {
		service.Build = &types.BundleSkeletonBuildConfig{
			Type:          "go",
			SourceDir:     "api",
			EntryPoint:    ".",
			OutputPattern: filepath.ToSlash(filepath.Join("bin", "{{platform}}", fmt.Sprintf("%s-api{{ext}}", scenarioName))),
			Env: map[string]string{
				"CGO_ENABLED": "0",
			},
		}
	}

	service.Env = deriveServiceEnv(cfg, service.ID, service.Ports, scenarioName, service.Env)

	return service
}

func buildCLISkeletonService(scenarioName, scenarioPath string, cfg *types.ServiceConfig) *types.BundleSkeletonService {
	cliDir := filepath.Join(scenarioPath, "cli")
	if _, err := os.Stat(cliDir); err != nil {
		return nil
	}

	cliBinary := filepath.ToSlash(filepath.Join("cli", scenarioName))
	service := types.BundleSkeletonService{
		ID:          fmt.Sprintf("%s-cli", scenarioName),
		Type:        "resource",
		Description: fmt.Sprintf("CLI for %s scenario", scenarioName),
		Binaries: map[string]types.BundleSkeletonServiceBinary{
			"darwin-x64": {Path: cliBinary},
			"linux-x64":  {Path: cliBinary},
			"win-x64":    {Path: cliBinary + ".exe"},
		},
		Assets: []types.BundleSkeletonAsset{
			{Path: cliBinary, SHA256: "pending"},
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

	// Use build config if present in service.json
	if cfg != nil && cfg.Deployment != nil && cfg.Deployment.BuildConfigs != nil {
		if buildCfg, ok := cfg.Deployment.BuildConfigs["cli"]; ok {
			service.Build = &types.BundleSkeletonBuildConfig{
				Type:          buildCfg.Type,
				SourceDir:     buildCfg.SourceDir,
				EntryPoint:    buildCfg.EntryPoint,
				OutputPattern: buildCfg.OutputPattern,
			}
		}
	}

	// Fallback build config for Go CLIs if none provided
	if service.Build == nil {
		service.Build = &types.BundleSkeletonBuildConfig{
			Type:          "go",
			SourceDir:     "cli",
			EntryPoint:    ".",
			OutputPattern: filepath.ToSlash(filepath.Join("bin", "{{platform}}", fmt.Sprintf("%s{{ext}}", scenarioName))),
			Env: map[string]string{
				"CGO_ENABLED": "0",
			},
		}
	}

	// Minimal env; PATH will be enriched by runtime, but ensure lifecycle flag.
	service.Env = map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": "true",
	}

	return &service
}

func buildUISkeletonService(scenarioName, scenarioPath string, cfg *types.ServiceConfig) types.BundleSkeletonService {
	entry := filepath.ToSlash(filepath.Join("ui", "dist", "index.html"))
	service := types.BundleSkeletonService{
		ID:          "ui",
		Type:        "ui-bundle",
		Description: "Production UI bundle served locally",
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
			Path:     "/",
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

func uiExists(scenarioPath string) bool {
	paths := []string{
		filepath.Join(scenarioPath, "ui", "dist", "index.html"),
		filepath.Join(scenarioPath, "ui", "dist"),
	}
	for _, candidate := range paths {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

func ptrBool(v bool) *bool {
	return &v
}

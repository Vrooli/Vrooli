package deployment

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	appName := cfg.Service.DisplayName
	if appName == "" {
		appName = cfg.Service.Name
	}
	if appName == "" {
		appName = scenarioName
	}
	version := cfg.Service.Version
	if version == "" {
		version = "0.1.0"
	}

	skeleton := &types.DesktopBundleSkeleton{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App: types.BundleSkeletonApp{
			Name:        appName,
			Version:     version,
			Description: cfg.Service.Description,
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
	skeleton.Services = buildSkeletonServices(scenarioName, scenarioPath)

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

func buildSkeletonServices(scenarioName, scenarioPath string) []types.BundleSkeletonService {
	services := []types.BundleSkeletonService{
		buildAPISkeletonService(scenarioName, scenarioPath),
	}

	if uiExists(scenarioPath) {
		services = append(services, buildUISkeletonService(scenarioPath))
	}

	return services
}

func buildAPISkeletonService(scenarioName, scenarioPath string) types.BundleSkeletonService {
	defaultPath := filepath.ToSlash(filepath.Join("api", fmt.Sprintf("%s-api", scenarioName)))
	apiBinary := defaultPath
	if _, err := os.Stat(filepath.Join(scenarioPath, defaultPath)); err != nil {
		apiBinary = filepath.ToSlash(filepath.Join("api", "server"))
	}

	return types.BundleSkeletonService{
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
}

func buildUISkeletonService(scenarioPath string) types.BundleSkeletonService {
	entry := filepath.ToSlash(filepath.Join("ui", "dist", "index.html"))
	uiDir := filepath.ToSlash(filepath.Join("ui", "dist"))
	return types.BundleSkeletonService{
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
			{Path: uiDir, SHA256: "pending"},
			{Path: entry, SHA256: "pending"},
		},
		Critical: ptrBool(true),
	}
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

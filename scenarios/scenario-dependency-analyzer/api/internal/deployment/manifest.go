package deployment

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// BuildBundleManifest generates a manifest of all files and dependencies needed
// to package the scenario for deployment.
func BuildBundleManifest(scenarioName, scenarioPath string, generatedAt time.Time, nodes []types.DeploymentDependencyNode, cfg *types.Manifest) types.BundleManifest {
	skeleton := buildDesktopBundleSkeleton(scenarioName, scenarioPath, cfg, nodes)
	manifest := types.BundleManifest{
		Scenario:     scenarioName,
		GeneratedAt:  generatedAt,
		Files:        discoverBundleFiles(scenarioName, scenarioPath, cfg),
		Dependencies: flattenBundleDependencies(nodes),
		Skeleton:     skeleton,
	}
	manifest.Dependencies = includeDeclaredResources(manifest.Dependencies, cfg)
	return manifest
}

// discoverBundleFiles derives package inputs from the declared component graph.
func discoverBundleFiles(scenarioName, scenarioPath string, cfg *types.Manifest) []types.BundleFileEntry {
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

	if cfg == nil {
		return entries
	}
	names := make([]string, 0, len(cfg.Components))
	for name := range cfg.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	seenSources := map[string]struct{}{}
	for _, name := range names {
		component := cfg.Components[name]
		if dir := filepath.ToSlash(strings.TrimSpace(component.Build.Dir)); dir != "" {
			if _, seen := seenSources[dir]; !seen {
				_, statErr := os.Stat(filepath.Join(scenarioPath, filepath.FromSlash(dir)))
				entries = append(entries, types.BundleFileEntry{Path: dir, Type: name + "-source", Exists: statErr == nil})
				seenSources[dir] = struct{}{}
			}
		}
		if component.Build.Reuse != "" {
			continue
		}
		output := strings.TrimSpace(component.Build.Output)
		if output == "" {
			switch component.Build.Kind {
			case "go_module":
				output = filepath.Join(component.Build.Dir, scenarioName+"-api")
			case "pnpm_vite":
				output = filepath.Join(component.Build.Dir, "dist", "index.html")
			}
		}
		if output == "" {
			continue
		}
		output = filepath.ToSlash(output)
		_, statErr := os.Stat(filepath.Join(scenarioPath, filepath.FromSlash(output)))
		entryType := name + "-artifact"
		if component.Role == "ui" {
			entryType = "ui-bundle"
		}
		entries = append(entries, types.BundleFileEntry{Path: output, Type: entryType, Exists: statErr == nil, Notes: "Declared component output"})
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

func includeDeclaredResources(entries []types.BundleDependencyEntry, cfg *types.Manifest) []types.BundleDependencyEntry {
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

func buildDesktopBundleSkeleton(scenarioName, scenarioPath string, cfg *types.Manifest, nodes []types.DeploymentDependencyNode) *types.DesktopBundleSkeleton {
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

	// Resolve version similarly
	version := cfg.Service.Version
	if version == "" {
		version = "0.1.0"
	}

	description := cfg.Service.Description

	skeleton := &types.DesktopBundleSkeleton{
		SchemaVersion: "v0.1",
		Target:        "desktop",
		App: types.BundleSkeletonApp{
			Name:        appName,
			Version:     version,
			Description: description,
			Scenario:    scenarioName,
		},
		IPC: types.BundleSkeletonIPC{
			Mode:          "loopback-http",
			Host:          "127.0.0.1",
			Port:          0,
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
	skeleton.Peers = derivePeers(cfg)
	skeleton.Services = buildSkeletonServices(cfg)
	skeleton.Services = embedPeerServices(skeleton.Services, cfg, nodes)

	return skeleton
}

func embedPeerServices(root []types.BundleSkeletonService, cfg *types.Manifest, nodes []types.DeploymentDependencyNode) []types.BundleSkeletonService {
	peerNames := make([]string, 0)
	for name, dependency := range cfg.Dependencies.Scenarios {
		if dependency.BundlePolicy == "embed" {
			peerNames = append(peerNames, name)
		}
	}
	sort.Strings(peerNames)
	for _, peerName := range peerNames {
		node, ok := findScenarioNode(nodes, peerName)
		if !ok || node.Path == "" {
			log.Printf("cannot embed declared peer %s: dependency DAG has no scenario path", peerName)
			continue
		}
		peerConfig, err := config.LoadServiceConfig(node.Path)
		if err != nil {
			log.Printf("cannot embed declared peer %s from %s: %v", peerName, node.Path, err)
			continue
		}
		peerServices := buildSkeletonServices(peerConfig)
		prefix := peerName + "--"
		prefixedIDs := map[string]string{}
		for _, service := range peerServices {
			prefixedIDs[service.ID] = prefix + service.ID
		}
		for index := range peerServices {
			originalID := peerServices[index].ID
			peerServices[index].ID = prefixedIDs[originalID]
			for dependencyIndex, dependency := range peerServices[index].Dependencies {
				peerServices[index].Dependencies[dependencyIndex] = prefixedIDs[dependency]
			}
			for key, value := range peerServices[index].Env {
				for source, target := range prefixedIDs {
					value = strings.ReplaceAll(value, "${"+source+".", "${"+target+".")
				}
				peerServices[index].Env[key] = value
			}
			if peerServices[index].Build != nil {
				peerServices[index].Build.SourceDir = filepath.ToSlash(filepath.Join("peers", peerName, peerServices[index].Build.SourceDir))
				peerServices[index].Build.OutputPattern = strings.ReplaceAll(peerServices[index].Build.OutputPattern, "/"+originalID+"{{ext}}", "/"+prefix+originalID+"{{ext}}")
			}
			for platform, binary := range peerServices[index].Binaries {
				binary.Path = strings.ReplaceAll(binary.Path, "/"+originalID, "/"+prefix+originalID)
				peerServices[index].Binaries[platform] = binary
			}
		}
		root = append(root, peerServices...)
	}
	return root
}

func findScenarioNode(nodes []types.DeploymentDependencyNode, name string) (types.DeploymentDependencyNode, bool) {
	for _, node := range nodes {
		if node.Type == "scenario" && node.Name == name {
			return node, true
		}
		if found, ok := findScenarioNode(node.Children, name); ok {
			return found, true
		}
	}
	return types.DeploymentDependencyNode{}, false
}

func derivePeers(cfg *types.Manifest) []types.BundleSkeletonPeer {
	names := make([]string, 0, len(cfg.Dependencies.Scenarios))
	for name, dependency := range cfg.Dependencies.Scenarios {
		if dependency.BundlePolicy != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	peers := make([]types.BundleSkeletonPeer, 0, len(names))
	for _, name := range names {
		dependency := cfg.Dependencies.Scenarios[name]
		peer := types.BundleSkeletonPeer{
			Scenario:         name,
			BundlePolicy:     dependency.BundlePolicy,
			StartupPolicy:    dependency.StartupPolicy,
			DegradedBehavior: dependency.DegradedBehavior,
		}
		peers = append(peers, peer)
	}
	return peers
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

func buildSkeletonServices(cfg *types.Manifest) []types.BundleSkeletonService {
	if cfg == nil {
		return []types.BundleSkeletonService{}
	}
	return buildDeclaredComponentServices(cfg)
}

var desktopPlatforms = []string{"darwin-x64", "linux-x64", "win-x64"}

func buildDeclaredComponentServices(cfg *types.Manifest) []types.BundleSkeletonService {
	names := make([]string, 0, len(cfg.Components))
	for name := range cfg.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	portOwners := map[string]string{}
	for componentName, component := range cfg.Components {
		if component.Run.Port != "" {
			portOwners[component.Run.Port] = componentName
		}
	}

	services := make([]types.BundleSkeletonService, 0, len(names))
	for _, name := range names {
		component := cfg.Components[name]
		service := types.BundleSkeletonService{
			ID:          name,
			Type:        desktopServiceType(component.Role),
			Description: fmt.Sprintf("Declared %s component", component.Role),
			Binaries:    declaredComponentBinaries(name, component, cfg.Components),
			Build:       declaredComponentBuild(name, component),
			Env:         projectedComponentEnv(component.Run.Env, cfg.Ports, portOwners),
			DataDirs:    append([]string(nil), component.Run.DataDirs...),
			LogDir:      component.Run.LogDir,
			Critical:    ptrBool(true),
		}

		if component.Run.Port != "" {
			service.Ports = &types.BundleSkeletonServicePorts{Requested: []types.BundleSkeletonRequestedPort{{
				Name:  component.Run.Port,
				Range: desktopPortRange(cfg.Ports[component.Run.Port]),
			}}}
		}
		if component.Role == "ui" {
			service.DistRoot = filepath.ToSlash(filepath.Join(component.Build.Dir, "dist"))
			index := filepath.ToSlash(filepath.Join(service.DistRoot, "index.html"))
			service.Assets = []types.BundleSkeletonAsset{{Path: index, SHA256: "pending"}}
		}
		service.Env["VROOLI_LIFECYCLE_MANAGED"] = "true"
		service.Health = declaredComponentHealth(name, component, cfg)
		service.Readiness = declaredComponentReadiness(component)
		for _, dependency := range component.Run.DependsOn {
			service.Dependencies = append(service.Dependencies, dependency.Component)
		}
		if component.Run.SupervisedBy != "" {
			service.Dependencies = append(service.Dependencies, component.Run.SupervisedBy)
		}
		service.Dependencies = dedupeStrings(service.Dependencies)
		services = append(services, service)
	}
	return services
}

func desktopServiceType(role string) string {
	switch role {
	case "api":
		return "api-binary"
	case "ui":
		return "ui-bundle"
	case "worker", "sidecar":
		return "worker"
	default:
		return "resource"
	}
}

func declaredComponentBuild(name string, component scenariomodel.Component) *types.BundleSkeletonBuildConfig {
	if component.Build.Reuse != "" {
		return nil
	}
	kind := ""
	switch component.Build.Kind {
	case "go_module":
		kind = "go"
	case "pnpm_vite", "node_bundle":
		kind = "npm"
	default:
		kind = component.Build.Kind
	}
	if kind == "" {
		return nil
	}
	output := filepath.ToSlash(filepath.Join("bin", component.Role, "{{platform}}", name+"{{ext}}"))
	if component.Role == "ui" {
		output = strings.TrimSpace(component.Build.Output)
		if output == "" {
			output = filepath.ToSlash(filepath.Join(component.Build.Dir, "dist", "index.html"))
		}
	}
	return &types.BundleSkeletonBuildConfig{
		Type:          kind,
		SourceDir:     component.Build.Dir,
		EntryPoint:    component.Build.Entry,
		OutputPattern: output,
	}
}

func projectedComponentEnv(input map[string]string, ports map[string]scenariomodel.Port, owners map[string]string) map[string]string {
	result := copyStringMap(input)
	for portName, port := range ports {
		owner, ok := owners[portName]
		if !ok || port.EnvVar == "" {
			continue
		}
		placeholder := fmt.Sprintf("${%s.%s}", owner, portName)
		result[port.EnvVar] = placeholder
		for key, value := range result {
			result[key] = strings.ReplaceAll(value, "${"+port.EnvVar+"}", placeholder)
		}
	}
	return result
}

func declaredComponentBinaries(name string, component scenariomodel.Component, components map[string]scenariomodel.Component) map[string]types.BundleSkeletonServiceBinary {
	result := make(map[string]types.BundleSkeletonServiceBinary, len(desktopPlatforms))
	for _, platform := range desktopPlatforms {
		argv := expandDesktopArgv(component.Run.Argv, platform, components)
		path := desktopComponentBinaryPath(name, component.Role, platform)
		args := []string(nil)
		if len(argv) > 0 {
			path = argv[0]
			args = append(args, argv[1:]...)
		}
		result[platform] = types.BundleSkeletonServiceBinary{
			Path: path,
			Args: args,
			Cwd:  component.Run.CWD,
		}
	}
	return result
}

func expandDesktopArgv(argv []string, platform string, components map[string]scenariomodel.Component) []string {
	ext := ""
	if strings.HasPrefix(platform, "win-") {
		ext = ".exe"
	}
	result := make([]string, len(argv))
	for index, value := range argv {
		value = strings.ReplaceAll(value, "{{ext}}", ext)
		for {
			start := strings.Index(value, "{{bin.")
			if start < 0 {
				break
			}
			end := strings.Index(value[start:], "}}")
			if end < 0 {
				break
			}
			componentName := value[start+len("{{bin.") : start+end]
			target, ok := components[componentName]
			if !ok {
				break
			}
			replacement := desktopComponentBinaryPath(componentName, target.Role, platform)
			value = value[:start] + replacement + value[start+end+2:]
		}
		result[index] = value
	}
	return result
}

func desktopComponentBinaryPath(name, role, platform string) string {
	ext := ""
	if strings.HasPrefix(platform, "win-") {
		ext = ".exe"
	}
	return filepath.ToSlash(filepath.Join("bin", role, platform, name+ext))
}

func desktopPortRange(port scenariomodel.Port) types.BundleSkeletonPortRange {
	if port.Port != nil && *port.Port > 0 {
		return types.BundleSkeletonPortRange{Min: *port.Port, Max: *port.Port}
	}
	parts := strings.SplitN(strings.TrimSpace(port.Range), "-", 2)
	if len(parts) == 2 {
		minimum, minimumErr := strconv.Atoi(strings.TrimSpace(parts[0]))
		maximum, maximumErr := strconv.Atoi(strings.TrimSpace(parts[1]))
		if minimumErr == nil && maximumErr == nil && minimum > 0 && maximum >= minimum {
			return types.BundleSkeletonPortRange{Min: minimum, Max: maximum}
		}
	}
	return types.BundleSkeletonPortRange{Min: 20000, Max: 24000}
}

func declaredComponentHealth(_ string, component scenariomodel.Component, cfg *types.Manifest) types.BundleSkeletonHealth {
	health := types.BundleSkeletonHealth{Type: "command", Command: []string{"true"}, Interval: 2000, Timeout: 2000, Retries: 5}
	if component.Run.Port == "" {
		return health
	}
	health = types.BundleSkeletonHealth{Type: "tcp", PortName: component.Run.Port, Interval: 2000, Timeout: 2000, Retries: 5}
	if cfg.Lifecycle.Health == nil {
		return health
	}
	envVar := cfg.Ports[component.Run.Port].EnvVar
	for _, check := range cfg.Lifecycle.Health.Checks {
		if check.Type != "http" || (envVar != "" && !strings.Contains(check.Target, "${"+envVar+"}")) {
			continue
		}
		path := desktopHealthPath(check.Target)
		health = types.BundleSkeletonHealth{Type: "http", Path: path, PortName: component.Run.Port, Interval: check.Interval, Timeout: check.Timeout, Retries: 5}
		if health.Interval == 0 {
			health.Interval = 2000
		}
		if health.Timeout == 0 {
			health.Timeout = 2000
		}
		return health
	}
	return health
}

func desktopHealthPath(target string) string {
	if parsed, err := url.Parse(target); err == nil && parsed.Path != "" {
		return parsed.Path
	}
	if marker := strings.Index(target, "}"); marker >= 0 {
		if suffix := target[marker+1:]; strings.HasPrefix(suffix, "/") {
			return suffix
		}
	}
	return "/"
}

func declaredComponentReadiness(component scenariomodel.Component) types.BundleSkeletonReadiness {
	readiness := types.BundleSkeletonReadiness{Type: "health_success", Timeout: 30000}
	if component.Run.Readiness != nil {
		readiness.Timeout = component.Run.Readiness.TimeoutMS
		if readiness.Timeout == 0 {
			readiness.Timeout = 30000
		}
		switch component.Run.Readiness.Type {
		case "port_open":
			readiness.Type = "port_open"
		case "http":
			readiness.Type = "health_success"
		}
	}
	readiness.PortName = component.Run.Port
	return readiness
}

func copyStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input)+1)
	for key, value := range input {
		result[key] = value
	}
	return result
}

func ptrBool(v bool) *bool {
	return &v
}

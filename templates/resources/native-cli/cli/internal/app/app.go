package app

import (
	"{{RESOURCE_CLI_COMMAND}}/cli/internal/discovery"
	"{{RESOURCE_CLI_COMMAND}}/cli/internal/domain"
	"{{RESOURCE_CLI_COMMAND}}/cli/internal/env"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// BuildInfo captures build-time metadata for the resource CLI entrypoint.
type BuildInfo struct {
	Name        string
	Version     string
	Description string
	Fingerprint string
	Timestamp   string
	SourceRoot  string
}

// BuildCommandApp wires the native resource command surface behind the shared
// cli/ entrypoint so command registration and resource logic stay in
// cli/internal/.
func BuildCommandApp(info BuildInfo) (*cliapp.App, error) {
	cfg := env.Load()
	runtime := discovery.DiscoverRuntime(info.SourceRoot)
	service := domain.NewService(cfg, runtime)

	stale := cliutil.NewStaleChecker(
		info.Name,
		info.Fingerprint,
		info.Timestamp,
		info.SourceRoot,
		"VROOLI_CLI_SOURCE_ROOT",
	)
	stale.SourceContextPath = ".."
	stale.ManifestSourcePath = "resource.json"
	stale.FreshnessInputs = []string{
		"cli/**",
		"cli/internal/**",
		"docs/**",
		"README.md",
		"resource.json",
	}

	return cliapp.NewApp(cliapp.AppOptions{
		Name:        info.Name,
		Version:     info.Version,
		Description: info.Description,
		Commands: []cliapp.CommandGroup{
			{
				Title: "Resource",
				Commands: []cliapp.Command{
					{Name: "info", Description: "Show build/runtime information", Run: func(args []string) error { return service.PrintInfo(info.Name, info.Version, info.Description) }},
					{Name: "status", Description: "Show placeholder resource status", Run: func(args []string) error { return service.PrintStatus() }},
				},
			},
			{
				Title: "Domain",
				Commands: []cliapp.Command{
					{Name: "content", Description: "Placeholder resource-specific command group", Run: func(args []string) error { return service.PrintDomainHelp() }},
				},
			},
		},
		StaleChecker: stale,
	}), nil
}

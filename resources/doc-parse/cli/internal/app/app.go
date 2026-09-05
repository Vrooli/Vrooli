package app

import (
	"fmt"
	"strings"

	"resource-doc-parse/cli/internal/discovery"
	"resource-doc-parse/cli/internal/domain"
	"resource-doc-parse/cli/internal/env"

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
					{Name: "status", Description: "Show resource readiness status", Run: func(args []string) error { return service.PrintStatus() }},
					{Name: "health", Description: "Verify the checked artifact and parser readiness", Run: func(args []string) error { return service.Health() }},
					{Name: "capabilities", Description: "List parser capabilities", Run: func(args []string) error { return service.Capabilities() }},
					{Name: "version", Description: "Show CLI and parser artifact versions", Run: func(args []string) error { return service.Version(info.Name, info.Version) }},
				},
			},
			{
				Title: "Domain",
				Commands: []cliapp.Command{
					{Name: "content", Description: "List parser capabilities", Run: func(args []string) error { return service.PrintDomainHelp() }},
					{Name: "classify", Usage: "classify <pdf>", Description: "Classify a PDF and report page routing", Run: func(args []string) error {
						if len(args) != 1 {
							return fmt.Errorf("usage: classify <pdf>")
						}
						return service.Classify(args[0])
					}},
					{Name: "parse", Usage: "parse <file> [--capabilities content,tables,geometry]", Description: "Parse a document through the selected capabilities", Run: func(args []string) error {
						input, capabilities, err := parseArguments(args)
						if err != nil {
							return err
						}
						if input == "" {
							return fmt.Errorf("usage: parse <file> [--capabilities content,tables,geometry]")
						}
						var selected []string
						for _, value := range strings.Split(capabilities, ",") {
							if value = strings.TrimSpace(value); value != "" {
								selected = append(selected, value)
							}
						}
						return service.Parse(input, selected)
					}},
				},
			},
		},
		StaleChecker: stale,
	}), nil
}

func parseArguments(args []string) (string, string, error) {
	capabilities := "content,tables,geometry"
	input := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--capabilities":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--capabilities requires a value")
			}
			i++
			capabilities = args[i]
		case strings.HasPrefix(arg, "--capabilities="):
			capabilities = strings.TrimPrefix(arg, "--capabilities=")
		case strings.HasPrefix(arg, "-"):
			return "", "", fmt.Errorf("unknown parse option %q", arg)
		case input == "":
			input = arg
		default:
			return "", "", fmt.Errorf("parse accepts one input path")
		}
	}
	return input, capabilities, nil
}

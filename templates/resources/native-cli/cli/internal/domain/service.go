package domain

import (
	"fmt"
	"{{RESOURCE_CLI_COMMAND}}/cli/internal/discovery"
	"{{RESOURCE_CLI_COMMAND}}/cli/internal/env"
	"{{RESOURCE_CLI_COMMAND}}/cli/internal/version"
)

// Service is the default home for resource-specific Go logic in a native-cli
// resource.
type Service struct {
	Config   env.Config
	Runtime  discovery.Runtime
	Manifest version.Manifest
}

// NewService wires the default resource-local implementation surface.
func NewService(cfg env.Config, runtime discovery.Runtime) Service {
	return Service{
		Config:  cfg,
		Runtime: runtime,
		Manifest: version.Manifest{
			InstalledPath: runtime.InstalledManifest,
			SourcePath:    runtime.SourceManifestPath,
		},
	}
}

// PrintInfo prints placeholder runtime metadata.
func (s Service) PrintInfo(name, version, description string) error {
	fmt.Printf("%s %s\n", name, version)
	fmt.Printf("%s\n", description)
	if s.Runtime.InstalledManifest != "" {
		fmt.Printf("manifest: %s\n", s.Runtime.InstalledManifest)
	}
	return nil
}

// PrintStatus prints placeholder status output.
func (s Service) PrintStatus() error {
	fmt.Println("Status is resource-specific. Implement it in cli/internal/domain.")
	return nil
}

// PrintDomainHelp prints placeholder guidance for the resource-specific command
// surface.
func (s Service) PrintDomainHelp() error {
	fmt.Println("Add the real operator command surface in cli/internal/app and cli/internal/domain.")
	return nil
}

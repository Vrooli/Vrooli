// Package capabilities exposes the generated scenario capability registry.
package capabilities

import (
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	capabilitydefaults "github.com/vrooli/vrooli/packages/capability-registry-go/defaults"
)

type Registry = capabilityregistry.Registry

var Known = capabilitydefaults.AudioToolsDefinitions()

func NewRegistry() *Registry { return capabilitydefaults.NewAudioToolsRegistry() }

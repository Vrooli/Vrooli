// Package config is the CLI's config-domain command surface. Mirrors the
// API's Connect-RPC ConfigService and the tunnel's ingress reconciler.
//
// Follows the canonical domain shape: a Register(core, manifest) returning
// a cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest is the single source of truth for the
// command-line shape.
package config

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "config"

// Register builds the config subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ConfigService.GetConfig":                  h.get,
		"ConfigService.BootstrapCloudflare":        h.bootstrap,
		"ConfigService.GetCredentialStatus":        h.credentialsStatus,
		"ConfigService.SetCloudflareCredentials":   h.credentialsSet,
		"ConfigService.ClearCloudflareCredentials": h.credentialsClear,
		"ConfigService.Sync":                       h.sync,
		"ConfigService.SwitchMode":                 h.mode,
		"ConfigService.SetPublicExposure":          h.publicExposure,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("config: load from manifest: %w", err)
	}
	return group, nil
}

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
	bindings := map[string]cliapp.PrimitiveHandler{
		"ConfigService.GetConfig":                  cliapp.ProtoList(h.getCall, h.getReport),
		"ConfigService.BootstrapCloudflare":        cliapp.ProtoMutation(h.bootstrapCall, h.bootstrapReport),
		"ConfigService.GetCredentialStatus":        cliapp.ProtoList(h.credentialsStatusCall, h.credentialsStatusReport),
		"ConfigService.SetCloudflareCredentials":   cliapp.ProtoMutation(h.credentialsSetCall, h.credentialsSetReport),
		"ConfigService.ClearCloudflareCredentials": cliapp.ProtoMutation(h.credentialsClearCall, h.credentialsClearReport),
		"ConfigService.Sync":                       cliapp.ProtoMutation(h.syncCall, h.syncReport),
		"ConfigService.SwitchMode":                 cliapp.ProtoMutation(h.modeCall, h.modeReport),
		"ConfigService.SetPublicExposure":          cliapp.ProtoMutation(h.publicExposureCall, h.publicExposureReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("config: load from manifest: %w", err)
	}
	return group, nil
}

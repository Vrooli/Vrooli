// Package agentapp builds the CLI shell shared by coding-agent resources: the
// lifecycle commands, install and update, model discovery and the coding policy.
// Each resource supplies only what differs. The binary comes from the resource's
// harness descriptor.
package agentapp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/cli-core/upstreamcheck/upstreamverb"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

// Version is the version of every coding-agent resource CLI.
const Version = "0.1.0"

// Build is what cli-installer links into a resource's main package.
type Build struct {
	Fingerprint string
	Timestamp   string
	SourceRoot  string
}

// Spec is what one coding-agent resource declares.
type Spec struct {
	// Name is the resource name, which its harness descriptor is found by.
	Name        string
	Description string
	Build       Build
	// DataDir is the agent's own data directory, as path segments under $HOME.
	DataDir []string
	// UpstreamSourceKind and UpstreamSourceID select an acquisition source for
	// install and update; empty means the resource's manifest decides.
	UpstreamSourceKind string
	UpstreamSourceID   string
	Posture            agentharness.EnforcementPosture
	// NoModelDiscovery leaves out `models` for a runner with no way to list them.
	NoModelDiscovery bool
	// WrapInstall extends the install-direct command.
	WrapInstall func(cliapp.Command) cliapp.Command
	// InstallCommands follow install-direct and update in the Installation group.
	InstallCommands []cliapp.Command
	// CommandGroups follow the Installation group.
	CommandGroups []cliapp.CommandGroup
	// Groups returns the resource's own subcommand groups, which follow `models`
	// and `policy` in the order given.
	Groups func(install agentinstall.Spec) ([]cliapp.SubcommandGroup, error)
	// NativeSettings, when set, adds the `settings` group and makes `ensure`
	// apply resources/<name>/native-settings.json.
	NativeSettings *NativeSettings
	// EnsureSteps are resource-specific actions `ensure` runs after native
	// settings. Every agent resource gets `ensure`, because the manifests
	// advertise supports_ensure and `vrooli setup` calls it.
	EnsureSteps []EnsureStep
}

// Main runs newApp's CLI as the process and exits non-zero on failure.
func Main(newApp func() (*cliapp.ResourceApp, error)) {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// New builds the resource's CLI from its spec.
func New(spec Spec) (*cliapp.ResourceApp, error) {
	descriptor, ok := harness.Builtin().Lookup(spec.Name)
	if !ok {
		return nil, fmt.Errorf("resource %q declares no harness", spec.Name)
	}
	env := cliapp.StandardResourceEnv(spec.Name, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                spec.Name,
		Version:             Version,
		Description:         spec.Description,
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    spec.Build.Fingerprint,
		BuildTimestamp:      spec.Build.Timestamp,
		BuildSourceRoot:     spec.Build.SourceRoot,
	})
	if err != nil {
		return nil, err
	}
	home := os.Getenv("HOME")
	install := agentinstall.Spec{
		ResourceName:       spec.Name,
		Binary:             descriptor.Binary,
		BinDir:             filepath.Join(home, ".local", "bin"),
		DataDir:            filepath.Join(append([]string{home}, spec.DataDir...)...),
		UpstreamSourceKind: spec.UpstreamSourceKind,
		UpstreamSourceID:   spec.UpstreamSourceID,
	}
	direct := agentinstall.DirectInstallCommand(install)
	if spec.WrapInstall != nil {
		direct = spec.WrapInstall(direct)
	}
	installation := append([]cliapp.Command{direct, agentinstall.UpdateCommand(install)}, spec.InstallCommands...)
	commands := append(app.StandardLifecycleCommands(),
		cliapp.CommandGroup{Title: "Installation", Commands: installation},
		cliapp.CommandGroup{Title: "Configuration", Commands: []cliapp.Command{ensureCommand(spec, descriptor.Binary)}})
	commands = append(commands, spec.CommandGroups...)

	catalog := agentharness.ResourceCatalogPath(spec.Name)
	var groups []cliapp.SubcommandGroup
	if !spec.NoModelDiscovery {
		groups = append(groups, agentharness.ModelDiscoveryCommands(agentharness.ModelDiscoveryConfig{Runner: spec.Name, CatalogPath: catalog}))
	}
	groups = append(groups, agentharness.CodingPolicyCommands(agentharness.CodingPolicyConfig{Runner: spec.Name, CatalogPath: catalog, Posture: spec.Posture}))
	if spec.NativeSettings != nil {
		groups = append(groups, agentharness.NativeSettingsCommands(nativeSettingsConfig(spec, descriptor.Binary, os.Stdout, os.Stderr)))
	}
	own, err := spec.Groups(install)
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(commands, append(groups, own...))
	return app, nil
}

// NPMUpstream is the upstream-check group for a CLI published to npm.
func NPMUpstream(install agentinstall.Spec, pinnedVersion, npmPackage string) cliapp.SubcommandGroup {
	return upstreamverb.Commands(upstreamcheck.Default(upstreamcheck.Config{
		DisplayName:   install.ResourceName,
		InstalledCmd:  []string{install.Binary, "--version"},
		PinnedVersion: pinnedVersion,
		SourceKind:    upstreamcheck.SourceNPM,
		SourceID:      npmPackage,
	}))
}

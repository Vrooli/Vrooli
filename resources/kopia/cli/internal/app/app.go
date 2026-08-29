// Package app wires the resource-kopia CLI: standard resource lifecycle
// commands plus the wrapped kopia command surface (repo / snapshot / policy /
// maintenance) and an engine `version` command that reports the pinned kopia
// version. Production dependencies (the kexec binary runner, the credential
// authority, and the on-disk registry) are constructed here; unit tests build
// the same Service values with fakes.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/discovery"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/env"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/install"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/maintenance"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repo"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/snapshot"

	"github.com/vrooli/cli-core/cliapp"
	agentinstall "github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "kopia"
	appVersion = "0.1.0"
)

var errReexeced = errors.New("kopia cli reexeced after rebuild")

// New builds the resource-kopia CLI with production dependencies.
func New(buildFingerprint, buildTimestamp, buildSourceRoot string) (*cliapp.ResourceApp, error) {
	resourceEnv := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Kopia backup engine resource CLI",
		SourceRootEnvVars:   resourceEnv.SourceRootEnvVars,
		ControlPlaneEnvVars: resourceEnv.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}

	runtime := env.Load()
	runner := kexec.NewBinaryRunner("kopia")
	credentialStore, err := credentials.Default()
	if err != nil {
		return nil, fmt.Errorf("initialize credential authority: %w", err)
	}
	reg := registry.New(runtime.RegistryFile)
	resolver := repoctx.Resolver{Registry: reg, Credentials: credentialStore}

	repoSvc := repo.Service{Runner: runner, Credentials: credentialStore, Registry: reg, Resolver: resolver, Env: runtime, Out: os.Stdout}
	snapSvc := snapshot.Service{Runner: runner, Resolver: resolver, Out: os.Stdout}
	policySvc := policy.Service{Runner: runner, Resolver: resolver, Out: os.Stdout}
	maintSvc := maintenance.Service{Runner: runner, Resolver: resolver, Out: os.Stdout}

	lifecycle := app.StandardLifecycleCommands()
	engine := engineGroup(app)
	subgroups := subcommandGroups(app, repoSvc, snapSvc, policySvc, maintSvc)

	install := agentinstall.DirectInstallCommand(agentinstall.Spec{
		Binary:  "kopia",
		BinDir:  filepath.Join(os.Getenv("HOME"), ".vrooli", "bin"),
		Version: "0.23.0",
		URLTemplates: map[string]string{
			"linux":   "https://github.com/kopia/kopia/releases/download/v${version}/kopia-${version}-linux-x64.tar.gz",
			"darwin":  "https://github.com/kopia/kopia/releases/download/v${version}/kopia-${version}-macOS-x64.tar.gz",
			"windows": "https://github.com/kopia/kopia/releases/download/v${version}/kopia-${version}-windows-x64.zip",
		},
		ArchiveEntry: "kopia",
	})
	app.SetCommandsWithSubgroups(append(lifecycle, engine, cliapp.CommandGroup{Title: "Installation", Commands: []cliapp.Command{install}}), subgroups)
	return app, nil
}

// engineGroup returns the engine-level commands (version with pin reporting).
func engineGroup(app *cliapp.ResourceApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Engine",
		Commands: []cliapp.Command{
			{
				Name:        "version",
				Aliases:     []string{"--version", "-v"},
				Description: "Show wrapper + pinned/installed kopia version",
				Run: func(args []string) error {
					return runVersion(app)
				},
			},
		},
	}
}

func runVersion(app *cliapp.ResourceApp) error {
	if err := checkStale(app); err != nil {
		if errors.Is(err, errReexeced) {
			return nil
		}
		return err
	}
	fmt.Printf("resource-kopia (wrapper) version %s\n", appVersion)
	report := install.Inspect(context.Background(), discovery.NewLocator(), nil)
	fmt.Printf("kopia pinned version       %s\n", report.Pinned)
	if report.Present {
		fmt.Printf("kopia installed version    %s\n", report.Installed)
	} else {
		fmt.Println("kopia installed version    (not installed)")
	}
	if warn := report.DriftWarning(); warn != "" {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warn)
	}
	return nil
}

// subcommandGroups builds the repo/snapshot/policy/maintenance command tree.
func subcommandGroups(app *cliapp.ResourceApp, repoSvc repo.Service, snapSvc snapshot.Service, policySvc policy.Service, maintSvc maintenance.Service) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		{
			Name:        "repo",
			Description: "Repository lifecycle (== Vrooli destination)",
			Subcommands: []cliapp.Command{
				{Name: "create", Description: "Create a repository (encryption always on)", Run: wrap(app, repoSvc.Create)},
				{Name: "connect", Description: "Connect to a repository", Run: wrap(app, repoSvc.Connect)},
				{Name: "status", Description: "Show repository status", Run: wrap(app, repoSvc.Status)},
				{Name: "stats", Description: "Show repository size/dedup statistics", Run: wrap(app, repoSvc.Stats)},
				{Name: "list", Description: "List registered repositories", Run: wrap(app, repoSvc.List)},
				{Name: "disconnect", Description: "Disconnect from a repository", Run: wrap(app, repoSvc.Disconnect)},
				{Name: "delete", Description: "Delete local repository metadata and credential-authority refs", Run: wrap(app, repoSvc.Delete)},
				{Name: "validate", Description: "Validate repository connectivity/integrity", Run: wrap(app, repoSvc.Validate)},
			},
		},
		{
			Name:        "snapshot",
			Description: "Snapshot lifecycle",
			Subcommands: []cliapp.Command{
				{Name: "create", Description: "Create a snapshot of a source path", Run: wrap(app, snapSvc.Create)},
				{Name: "list", Description: "List snapshots", Run: wrap(app, snapSvc.List)},
				{Name: "browse", Description: "Browse a snapshot directory as JSON", Run: wrap(app, snapSvc.Browse)},
				{Name: "restore", Description: "Restore a snapshot to a target dir", Run: wrap(app, snapSvc.Restore)},
				{Name: "verify", Description: "Verify snapshot/content integrity", Run: wrap(app, snapSvc.Verify)},
				{Name: "delete", Description: "Delete a snapshot", Run: wrap(app, snapSvc.Delete)},
			},
		},
		{
			Name:        "policy",
			Description: "Per-source retention + compression policies",
			Subcommands: []cliapp.Command{
				{Name: "set", Description: "Set GFS retention + compression for a source", Run: wrap(app, policySvc.Set)},
				{Name: "show", Description: "Show a source's resolved policy", Run: wrap(app, policySvc.Show)},
				{Name: "list", Description: "List policies in a repository", Run: wrap(app, policySvc.List)},
			},
		},
		{
			Name:        "maintenance",
			Description: "Repository maintenance (prune/retention enforcement)",
			Subcommands: []cliapp.Command{
				{Name: "run", Description: "Run a maintenance cycle", Run: wrap(app, maintSvc.Run)},
				{Name: "set", Description: "Configure automatic maintenance", Run: wrap(app, maintSvc.Set)},
			},
		},
	}
}

// wrap adapts a context-taking Service method into the CLI's Run signature,
// running the stale-rebuild check first (resource App-level dispatch does not
// auto-trigger it for custom subcommands).
func wrap(app *cliapp.ResourceApp, fn func(ctx context.Context, args []string) error) func([]string) error {
	return func(args []string) error {
		if err := checkStale(app); err != nil {
			if errors.Is(err, errReexeced) {
				return nil
			}
			return err
		}
		return fn(context.Background(), args)
	}
}

func checkStale(app *cliapp.ResourceApp) error {
	if app == nil || app.StaleChecker == nil {
		return nil
	}
	app.StaleChecker.ReexecArgs = append([]string(nil), os.Args[1:]...)
	if restarted := app.StaleChecker.CheckAndMaybeRebuild(); restarted {
		return errReexeced
	}
	return nil
}
